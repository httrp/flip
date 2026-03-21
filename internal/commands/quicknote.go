package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// QuicknoteOptions holds options for quicknote creation (VS Code integration)
type QuicknoteOptions struct {
	Title  string // Note title (required for non-interactive)
	Brain  string // Brain name (empty = active brain)
	NoEdit bool   // Don't open editor after creation
	NoLink bool   // Don't add link to journal
}

func NewQuicknoteCommand() *cobra.Command {
	var opts QuicknoteOptions
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "quicknote [new]",
		Aliases: []string{"qn", "quick"},
		Short:   "Create a quick note (minimal prompts)",
		Long: `Create a quick note with minimal prompts. Organization, project, and context can be filled in later in the file frontmatter.

Examples:
  flip quicknote                          # Interactive: Create quick note
  flip quicknote --title "My Note" --json # Non-interactive with JSON output
  flip qn --title "Quick idea" --brain log`,
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			// Default action: create quick note
			if len(args) == 0 || args[0] == "new" || args[0] == "n" {
				if JSONOutput || opts.Title != "" {
					return runCreateQuicknoteNonInteractive(opts)
				}
				return runCreateQuicknote()
			}
			return fmt.Errorf("unknown subcommand: %s", args[0])
		},
	}

	// Flags for non-interactive mode (VS Code integration)
	cmd.Flags().StringVar(&opts.Title, "title", "", "Note title (required for non-interactive mode)")
	cmd.Flags().StringVar(&opts.Brain, "brain", "", "Brain to use (default: active brain)")
	cmd.Flags().BoolVar(&opts.NoEdit, "no-edit", false, "Don't open editor after creation")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")
	cmd.Flags().BoolVar(&opts.NoLink, "no-link", false, "Don't add link to journal")

	// Add explicit 'new' subcommand for clarity
	newCmd := &cobra.Command{
		Use:     "new",
		Aliases: []string{"n"},
		Short:   "Create a new quick note (explicit)",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			if JSONOutput || opts.Title != "" {
				return runCreateQuicknoteNonInteractive(opts)
			}
			return runCreateQuicknote()
		},
	}
	cmd.AddCommand(newCmd)

	return cmd
}

// runCreateQuicknoteNonInteractive creates quick note without prompts (for VS Code integration)
func runCreateQuicknoteNonInteractive(opts QuicknoteOptions) error {
	// Validate required fields
	if opts.Title == "" {
		err := fmt.Errorf("--title is required for non-interactive mode")
		if JSONOutput {
			OutputJSONError("quicknote", err)
			return nil
		}
		return err
	}

	// Get workspace
	activeWs, err := getActiveWorkspace()
	if err != nil {
		if JSONOutput {
			OutputJSONError("quicknote", err)
			return nil
		}
		return err
	}

	// Get brain
	var activeBrain *Brain
	if opts.Brain != "" {
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == opts.Brain {
				activeBrain = &activeWs.Brains[i]
				break
			}
		}
		if activeBrain == nil {
			err := fmt.Errorf("brain not found: %s", opts.Brain)
			if JSONOutput {
				OutputJSONError("quicknote", err)
				return nil
			}
			return err
		}
	} else {
		// Use default brain
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == activeWs.DefaultBrain {
				activeBrain = &activeWs.Brains[i]
				break
			}
		}
		if activeBrain == nil && len(activeWs.Brains) > 0 {
			activeBrain = &activeWs.Brains[0]
		}
	}

	if activeBrain == nil {
		err := fmt.Errorf("no brain available")
		if JSONOutput {
			OutputJSONError("quicknote", err)
			return nil
		}
		return err
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(activeBrain.Path)
	if err != nil {
		if JSONOutput {
			OutputJSONError("quicknote", err)
			return nil
		}
		return err
	}

	// Generate filename
	filename := generateQuicknoteFilename(opts.Title, detection.Type)

	// Target directory (notes root, no subfolder for quicknotes)
	targetDir := getNotesDirectory(activeBrain.Path, detection.Type)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		if JSONOutput {
			OutputJSONError("quicknote", err)
			return nil
		}
		return err
	}

	// Full file path
	filePath := filepath.Join(targetDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		err := fmt.Errorf("file already exists: %s", filePath)
		if JSONOutput {
			OutputJSONError("quicknote", err)
			return nil
		}
		return err
	}

	// Generate content
	content := generateQuicknoteContent(opts.Title, "", detection.Type, activeBrain.Path)

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		if JSONOutput {
			OutputJSONError("quicknote", err)
			return nil
		}
		return err
	}

	// Add link to journal (unless disabled)
	if !opts.NoLink {
		relPath := relativePathFromBrain(filePath, activeBrain.Path)
		if err := AddLinkToJournal(JournalLinkOptions{
			ItemType:    "note",
			ItemName:    opts.Title,
			ItemPath:    relPath,
			Brain:       activeBrain,
			Interactive: false,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "\u26a0\ufe0f  journal link failed: %v\n", err)
		}
	}

	// Output result
	if JSONOutput {
		OutputJSONSuccess("quicknote", QuicknoteResult{
			Action:    "created",
			Path:      filePath,
			BrainName: activeBrain.Name,
			BrainPath: activeBrain.Path,
		})
		return nil
	}

	fmt.Printf("✅ Quick note created: %s\n", filePath)

	// Open editor if not disabled
	if !opts.NoEdit {
		_ = openInEditor(filePath)
	}

	return nil
}

// runCreateQuicknote creates a quick note with minimal prompts
func runCreateQuicknote() error {
	fmt.Println("\n⚡ Create Quick Note")

	// Get active workspace
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get active workspace: %w", err)
	}

	if len(activeWs.Brains) == 0 {
		fmt.Println("⚠️  No brains found in active workspace. Please add a brain first.")
		return nil
	}

	// Select brain
	activeBrain, err := confirmOrSelectBrain(activeWs)
	if err != nil {
		return err
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(activeBrain.Path)
	if err != nil {
		return fmt.Errorf("failed to detect brain type: %w", err)
	}

	// Prompt for title (ONLY required field)
	title, err := ui.RunInput("Note title", "", "", func(input string) error {
		if strings.TrimSpace(input) == "" {
			return fmt.Errorf("title cannot be empty")
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("title prompt cancelled: %w", err)
	}
	title = strings.TrimSpace(title)

	// Optional tags (quick input)
	tags, err := ui.RunInput("Tags (comma-separated, optional, press Enter to skip)", "", "", nil)
	if err != nil {
		return fmt.Errorf("tags prompt cancelled: %w", err)
	}
	tags = strings.TrimSpace(tags)

	// Generate filename
	filename := generateQuicknoteFilename(title, detection.Type)

	// Determine base notes directory
	baseDir := getNotesDirectory(activeBrain.Path, detection.Type)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prompt for subfolder
	targetDir, err := promptForSubfolder(baseDir, detection.Type)
	if err != nil {
		return err
	}

	// Full file path
	filePath := filepath.Join(targetDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		fmt.Printf("\n⚠️  File already exists: %s\n", filePath)
		return nil
	}

	// Show preview
	fmt.Printf("\n📄 Will create: %s\n", filename)
	fmt.Printf("   Location: %s\n\n", targetDir)

	// Generate content with empty frontmatter fields
	content := generateQuicknoteContent(title, tags, detection.Type, activeBrain.Path)

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("✅ Quick note created: %s\n", filePath)

	// Ask if user wants to add link to journal
	relPath := relativePathFromBrain(filePath, activeBrain.Path)
	if err := AddLinkToJournal(JournalLinkOptions{
		ItemType:    "note",
		ItemName:    title,
		ItemPath:    relPath,
		Brain:       activeBrain,
		Interactive: true,
	}); err != nil {
		fmt.Printf("⚠️  Could not add journal link: %v\n", err)
	}

	// Prompt to open in editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "code" // Default to VS Code
	}

	yes, err := ui.RunConfirm("Open in editor", true)
	if err == nil && yes {
		cmd := exec.Command(editor, filePath)
		if err := cmd.Start(); err != nil {
			fmt.Printf("⚠️  Could not open editor: %v\n", err)
		} else {
			fmt.Printf("📝 Opening in %s...\n", editor)
		}
	}

	return nil
}

// generateQuicknoteFilename creates a filename for quick notes
func generateQuicknoteFilename(title string, brainType brain.BrainType) string {
	now := time.Now()

	// Sanitize title for filename
	cleanTitle := strings.ToLower(title)
	cleanTitle = strings.ReplaceAll(cleanTitle, " ", "-")
	// Remove special characters
	cleanTitle = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return -1
	}, cleanTitle)

	// Limit length
	if len(cleanTitle) > 50 {
		cleanTitle = cleanTitle[:50]
	}

	switch brainType {
	case brain.BrainTypeDendron:
		// Dendron uses dot-separated hierarchical naming
		return fmt.Sprintf("note.%s.md", cleanTitle)
	default:
		// Most systems use date prefix
		dateStr := now.Format("2006-01-02")
		return fmt.Sprintf("%s-%s.md", dateStr, cleanTitle)
	}
}

// generateQuicknoteContent creates quick note content with empty metadata fields
func generateQuicknoteContent(title, tags string, brainType brain.BrainType, brainPath string) string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04")

	// Get author from brain config
	author := getBrainAuthor(brainPath)
	if author == "" {
		author = "Unknown"
	}

	// Parse tags
	tagList := ""
	if tags != "" {
		tagList = tags
	}

	switch brainType {
	case brain.BrainTypeLogseq:
		return fmt.Sprintf(`- title:: %s
- created:: %s %s
- author:: %s
- tags:: %s
- organization:: 
- project:: 
- context:: 

## %s

`, title, dateStr, timeStr, author, tagList, title)

	case brain.BrainTypeObsidian:
		return fmt.Sprintf(`---
title: %s
created: %s
author: %s
tags: [%s]
organization: 
project: 
context: 
---

# %s

`, title, dateStr, author, tagList, title)

	case brain.BrainTypeDendron:
		return fmt.Sprintf(`---
id: %s
title: %s
desc: ''
created: %d
updated: %d
author: %s
tags: [%s]
organization: 
project: 
context: 
---

# %s

`, uuid.New().String(), title, now.Unix(), now.Unix(), author, tagList, title)

	case brain.BrainTypeFlip:
		return fmt.Sprintf(`---
title: %s
created: %s
updated: %s
author: %s
tags: [%s]
organization: 
project: 
context: 
---

# %s

`, title, dateStr, dateStr, author, tagList, title)

	default:
		return fmt.Sprintf(`---
title: %s
created: %s
tags: [%s]
organization: 
project: 
context: 
---

# %s

`, title, dateStr, tagList, title)
	}
}
