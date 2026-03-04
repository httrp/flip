package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/templates"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// JournalOptions holds options for journal creation
type JournalOptions struct {
	Date   string // Date in YYYY-MM-DD format (empty = today)
	Brain  string // Brain name (empty = active brain)
	NoEdit bool   // Don't open editor after creation
}

func NewJournalCommand() *cobra.Command {
	var opts JournalOptions
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "journal [new]",
		Short: "Create or open a daily journal note",
		Long: `Create a new daily journal/daily note or open existing one. Prevents duplicate entries for the same date.

Examples:
  flip journal                    # Interactive: Create/open journal
  flip journal --date 2025-12-22  # Create journal for specific date
  flip journal --json             # JSON output for VS Code integration
  flip journal --brain log        # Use specific brain`,
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			if len(args) == 0 || args[0] == "new" || args[0] == "n" {
				if JSONOutput || opts.Date != "" || opts.Brain != "" {
					return runCreateJournalNonInteractive(opts)
				}
				return runCreateJournal()
			}
			return fmt.Errorf("unknown subcommand: %s", args[0])
		},
	}

	// Flags for non-interactive mode
	cmd.Flags().StringVar(&opts.Date, "date", "", "Date for journal (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&opts.Brain, "brain", "", "Brain to use (default: active brain)")
	cmd.Flags().BoolVar(&opts.NoEdit, "no-edit", false, "Don't open editor after creation")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")

	// Add explicit 'new' subcommand for clarity
	newCmd := &cobra.Command{
		Use:     "new",
		Aliases: []string{"n"},
		Short:   "Create or open a daily journal note (explicit)",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			if JSONOutput || opts.Date != "" || opts.Brain != "" {
				return runCreateJournalNonInteractive(opts)
			}
			return runCreateJournal()
		},
	}
	cmd.AddCommand(newCmd)

	// Add 'link' subcommand: add a link to today's journal for a given file
	linkCmd := &cobra.Command{
		Use:   "link",
		Short: "Add a link to today's journal for a file",
		Long:  "Add a markdown link to today's journal entry, pointing to the specified file. Automatically detects brain and file type.",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			filePath, _ := cmd.Flags().GetString("file")
			itemTitle, _ := cmd.Flags().GetString("title")
			itemType, _ := cmd.Flags().GetString("type")
			brainName, _ := cmd.Flags().GetString("brain")
			dateStr, _ := cmd.Flags().GetString("date")
			if strings.TrimSpace(filePath) == "" {
				err := fmt.Errorf("--file is required")
				if JSONOutput {
					OutputJSONError("journal link", err)
					return nil
				}
				return err
			}

			// Parse date if provided
			var targetDate *time.Time
			if dateStr != "" {
				parsed, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					err := fmt.Errorf("invalid date format, use YYYY-MM-DD: %w", err)
					if JSONOutput {
						OutputJSONError("journal link", err)
						return nil
					}
					return err
				}
				targetDate = &parsed
			}

			// Resolve brain and type via file-info if not provided
			info := getFileInfo(filePath)
			if info.Error != "" && info.Error != "file does not exist" {
				// Continue, but note type unknown
			}

			// Determine brain to use
			var activeBrain *Brain
			if brainName != "" {
				ws, err := getActiveWorkspace()
				if err == nil {
					for i := range ws.Brains {
						if ws.Brains[i].Name == brainName {
							activeBrain = &ws.Brains[i]
							break
						}
					}
				}
			}
			if activeBrain == nil && info.BrainPath != "" {
				// Use detected brain from file
				ws, err := getActiveWorkspace()
				if err == nil {
					for i := range ws.Brains {
						if ws.Brains[i].Path == info.BrainPath {
							activeBrain = &ws.Brains[i]
							break
						}
					}
				}
				if activeBrain == nil {
					// Fallback construct Brain with minimal fields
					activeBrain = &Brain{Name: filepath.Base(info.BrainPath), Path: info.BrainPath, Type: ""}
				}
			}
			if activeBrain == nil {
				err := fmt.Errorf("could not determine brain for file")
				if JSONOutput {
					OutputJSONError("journal link", err)
					return nil
				}
				return err
			}

			// Determine item type and title
			typ := itemType
			if strings.TrimSpace(typ) == "" {
				typ = info.Type
			}
			if strings.TrimSpace(typ) == "" {
				typ = "note"
			}
			name := itemTitle
			if strings.TrimSpace(name) == "" {
				name = info.Name
			}
			if strings.TrimSpace(name) == "" {
				name = info.FileName
			}

			rel := relativePathFromBrain(filePath, activeBrain.Path)
			if err := AddLinkToJournal(JournalLinkOptions{
				ItemType:    typ,
				ItemName:    name,
				ItemPath:    rel,
				Brain:       activeBrain,
				Interactive: false,
				Date:        targetDate,
			}); err != nil {
				if JSONOutput {
					OutputJSONError("journal link", err)
					return nil
				}
				return err
			}

			if JSONOutput {
				OutputJSONSuccess("journal link", map[string]string{
					"path":       filePath,
					"rel_path":   rel,
					"brain_name": activeBrain.Name,
					"item_type":  typ,
					"item_name":  name,
				})
				return nil
			}

			fmt.Printf("✓ Added link to journal for '%s' (%s)\n", name, typ)
			return nil
		},
	}
	linkCmd.Flags().String("file", "", "File to link in the journal")
	linkCmd.Flags().String("title", "", "Override display title for the link")
	linkCmd.Flags().String("type", "", "Override item type (note, task, exercise, meeting)")
	linkCmd.Flags().String("brain", "", "Brain name (defaults to detected brain from file)")
	linkCmd.Flags().String("date", "", "Target journal date (YYYY-MM-DD, default: today)")
	linkCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")
	cmd.AddCommand(linkCmd)

	return cmd
}

// runCreateJournalNonInteractive creates journal without prompts (for VS Code integration)
func runCreateJournalNonInteractive(opts JournalOptions) error {
	// Parse date
	var targetDate time.Time
	if opts.Date == "" {
		targetDate = time.Now()
	} else {
		var err error
		targetDate, err = time.Parse("2006-01-02", opts.Date)
		if err != nil {
			if JSONOutput {
				OutputJSONError("journal", fmt.Errorf("invalid date format: %s (use YYYY-MM-DD)", opts.Date))
				return nil
			}
			return fmt.Errorf("invalid date format: %s (use YYYY-MM-DD)", opts.Date)
		}
	}

	// Get workspace and brain
	activeWs, err := getActiveWorkspace()
	if err != nil {
		if JSONOutput {
			OutputJSONError("journal", err)
			return nil
		}
		return err
	}

	var activeBrain *Brain
	if opts.Brain != "" {
		// Find specified brain
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == opts.Brain {
				activeBrain = &activeWs.Brains[i]
				break
			}
		}
		if activeBrain == nil {
			err := fmt.Errorf("brain not found: %s", opts.Brain)
			if JSONOutput {
				OutputJSONError("journal", err)
				return nil
			}
			return err
		}
	} else {
		// Use default brain
		if activeWs.DefaultBrain != "" {
			for i := range activeWs.Brains {
				if activeWs.Brains[i].Name == activeWs.DefaultBrain {
					activeBrain = &activeWs.Brains[i]
					break
				}
			}
		}
		if activeBrain == nil && len(activeWs.Brains) > 0 {
			activeBrain = &activeWs.Brains[0]
		}
	}

	if activeBrain == nil {
		err := fmt.Errorf("no brains found in workspace")
		if JSONOutput {
			OutputJSONError("journal", err)
			return nil
		}
		return err
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(activeBrain.Path)
	if err != nil {
		if JSONOutput {
			OutputJSONError("journal", err)
			return nil
		}
		return err
	}

	// Generate filename and path
	filename := generateJournalFilename(targetDate, detection.Type)
	journalDir := getJournalDirectory(activeBrain.Path, detection.Type)
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		if JSONOutput {
			OutputJSONError("journal", err)
			return nil
		}
		return err
	}

	filePath := filepath.Join(journalDir, filename)

	// Check if exists
	action := "created"
	if _, err := os.Stat(filePath); err == nil {
		action = "opened"
	} else {
		// Create new journal
		content := generateJournalContent(targetDate, detection.Type, activeBrain.Name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			if JSONOutput {
				OutputJSONError("journal", err)
				return nil
			}
			return err
		}
	}

	// JSON output
	if JSONOutput {
		OutputJSONSuccess("journal", JournalResult{
			Action:    action,
			Path:      filePath,
			Date:      targetDate.Format("2006-01-02"),
			BrainName: activeBrain.Name,
			BrainPath: activeBrain.Path,
			BrainType: string(detection.Type),
		})
		return nil
	}

	// Human-readable output
	if action == "created" {
		fmt.Printf("✓ Journal created: %s\n", filePath)
	} else {
		fmt.Printf("📖 Journal exists: %s\n", filePath)
	}

	// Open editor unless --no-edit
	if !opts.NoEdit {
		if err := openInEditor(filePath); err != nil {
			fmt.Printf("⚠️  Could not open editor: %v\n", err)
		}
	}

	return nil
}

// runCreateJournal creates or opens a daily journal note (interactive mode)
func runCreateJournal() error {
	fmt.Println("\n📔 Daily Journal / Daily Note")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

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

	fmt.Printf("Brain type: %s\n\n", detection.Type)

	// Ask for date (today or other)
	promptDate := promptui.Select{
		Label: "Journal date",
		Items: []string{"Today", "Other date"},
	}

	dateIdx, _, err := promptDate.Run()
	if err != nil {
		return fmt.Errorf("date selection cancelled: %w", err)
	}

	var targetDate time.Time
	if dateIdx == 0 {
		// Today
		targetDate = time.Now()
	} else {
		// Other date - prompt for input
		promptCustomDate := promptui.Prompt{
			Label:   "Date (YYYY-MM-DD)",
			Default: time.Now().Format("2006-01-02"),
			Validate: func(input string) error {
				_, err := time.Parse("2006-01-02", input)
				if err != nil {
					return fmt.Errorf("invalid date format, use YYYY-MM-DD")
				}
				return nil
			},
		}

		dateStr, err := promptCustomDate.Run()
		if err != nil {
			return fmt.Errorf("date input cancelled: %w", err)
		}

		targetDate, err = time.Parse("2006-01-02", strings.TrimSpace(dateStr))
		if err != nil {
			return fmt.Errorf("failed to parse date: %w", err)
		}
	}

	// Generate filename based on brain type
	filename := generateJournalFilename(targetDate, detection.Type)

	// Determine journal directory (different from notes!)
	journalDir := getJournalDirectory(activeBrain.Path, detection.Type)
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return fmt.Errorf("failed to create journal directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(journalDir, filename)

	// Check if journal already exists
	if _, err := os.Stat(filePath); err == nil {
		fmt.Printf("\n📖 Journal already exists for %s\n", targetDate.Format("2006-01-02"))
		fmt.Printf("   Path: %s\n\n", filePath)

		// Just open it
		if err := promptAndOpenEditor(filePath); err != nil {
			fmt.Printf("⚠️  Could not open editor: %v\n", err)
		}
		return nil
	}

	// Show preview
	fmt.Printf("\n📄 Will create: %s\n", filename)
	fmt.Printf("   Location: %s\n\n", journalDir)

	// Generate content
	content := generateJournalContent(targetDate, detection.Type, activeBrain.Name)

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create journal: %w", err)
	}

	fmt.Printf("✓ Journal created successfully!\n")
	fmt.Printf("  Path: %s\n", filePath)
	fmt.Printf("  Brain: %s (%s)\n\n", activeBrain.Name, detection.Type)

	// Optional git commit
	if err := autoCommitFile(activeBrain.Path, filePath, "journal entry"); err != nil {
		fmt.Printf("⚠️  Git commit failed: %v\n", err)
	}

	// Ask if user wants to edit
	if err := promptAndOpenEditor(filePath); err != nil {
		fmt.Printf("⚠️  Could not open editor: %v\n", err)
	}

	return nil
}

// generateJournalFilename creates a filename for journal entries
func generateJournalFilename(date time.Time, brainType brain.BrainType) string {
	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: YYYY_MM_DD.md in journals/
		return fmt.Sprintf("%s.md", date.Format("2006_01_02"))

	case brain.BrainTypeObsidian:
		// Obsidian: YYYY-MM-DD.md (daily notes pattern)
		return fmt.Sprintf("%s.md", date.Format("2006-01-02"))

	case brain.BrainTypeDendron:
		// Dendron: journal.YYYY-MM-DD.md
		return fmt.Sprintf("journal.%s.md", date.Format("2006-01-02"))

	case brain.BrainTypeFoam:
		// Foam: YYYY-MM-DD.md (standard daily note pattern)
		return fmt.Sprintf("%s.md", date.Format("2006-01-02"))

	case brain.BrainTypeFlip:
		// Flip: YYYY-MM-DD.md in journal/
		return fmt.Sprintf("%s.md", date.Format("2006-01-02"))

	default:
		// Generic: YYYY-MM-DD.md
		return fmt.Sprintf("%s.md", date.Format("2006-01-02"))
	}
}

// getJournalDirectory is now in content_common.go

// generateJournalContent creates journal entry content
func generateJournalContent(date time.Time, brainType brain.BrainType, brainName string) string {
	dateStr := date.Format("2006-01-02")
	weekday := date.Format("Monday")

	// Try to load template from file
	tmpl, err := templates.Load(brainType, templates.TemplateTypeJournal)
	if err != nil {
		// Fallback to hardcoded template if file not found
		if !JSONOutput {
			fmt.Printf("Warning: Could not load template, using default (%v)\n", err)
		}
		return generateDefaultJournalContent(date, brainType, brainName)
	}

	// Prepare template variables
	vars := map[string]string{
		"date":      dateStr,
		"weekday":   weekday,
		"brain":     brainName,
		"brain_name": brainName,
		"id":        uuid.New().String(), // Proper UUID for Dendron compatibility
		"updated":   fmt.Sprintf("%d", time.Now().Unix()),
		"created":   fmt.Sprintf("%d", time.Now().Unix()),
	}

	rendered := templates.Render(tmpl, vars)
	return ensureJournalMetadata(rendered, brainType, brainName)
}

// generateDefaultJournalContent provides fallback templates when template files don't exist
func generateDefaultJournalContent(date time.Time, brainType brain.BrainType, brainName string) string {
	dateStr := date.Format("2006-01-02")
	weekday := date.Format("Monday")
	timeStr := time.Now().Format("15:04")

	switch brainType {
	case brain.BrainTypeLogseq:
		return fmt.Sprintf(`- brain:: %s
- %s, %s

## Morning

## Work

## Evening

## Tasks
- TODO Daily task example

## Notes

## Grateful For
- 

`, brainName, weekday, dateStr)

	case brain.BrainTypeObsidian:
		return fmt.Sprintf(`---
date: %s
day: %s
brain: %s
tags: [daily-note, journal]
---

# %s, %s

## Morning

## Work

## Evening

## Tasks
- [ ] Daily task example

## Notes

## Grateful For
- 

`, dateStr, weekday, brainName, weekday, dateStr)

	case brain.BrainTypeDendron:
		return fmt.Sprintf(`---
id: %s
title: Journal %s
desc: 'Daily journal entry'
updated: %d
created: %d
date: %s
brain: %s
---

# Journal - %s (%s)

## Morning

## Work

## Evening

## Tasks
- [ ] Daily task example

## Notes

## Grateful For
- 

`, uuid.New().String(), dateStr, time.Now().Unix(), time.Now().Unix(), dateStr, brainName, dateStr, weekday)

	case brain.BrainTypeFoam:
		return fmt.Sprintf(`---
date: %s
day: %s
brain: %s
tags: [daily]
---

# %s, %s

## Morning

## Work

## Evening

## Tasks
- [ ] Daily task example

## Notes

## Grateful For
- 

`, dateStr, weekday, brainName, weekday, dateStr)

	case brain.BrainTypeFlip:
		return fmt.Sprintf(`---
date: %s
day: %s
brain: %s
type: journal
tags: [daily, journal]
created: %s
---

# %s, %s

## Morning

## Work

## Evening

## Tasks
- [ ] Daily task example

## Notes

## Grateful For
- 

`, dateStr, weekday, brainName, timeStr, weekday, dateStr)

	default:
		return fmt.Sprintf(`---
date: %s
day: %s
brain: %s
type: journal
---

# %s, %s

## Morning

## Work

## Evening

## Tasks
- [ ] Daily task example

## Notes

## Grateful For
- 

`, dateStr, weekday, brainName, weekday, dateStr)
	}
}

// normalizeJournalTitle fixes malformed journal titles (e.g., "2026 01 13" -> "2026-01-13")
func normalizeJournalTitle(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	
	for _, line := range lines {
		// Check if this is a title line with space-separated date (e.g., "title: 2026 01 13")
		if strings.HasPrefix(line, "title:") {
			// Try to match "YYYY MM DD" pattern (4 digits, space, 2 digits, space, 2 digits)
			titleValue := strings.TrimPrefix(line, "title:")
			titleValue = strings.TrimSpace(titleValue)
			
			// Match space-separated date pattern like "2026 01 13"
			re := regexp.MustCompile(`^(\d{4})\s+(\d{2})\s+(\d{2})$`)
			if matches := re.FindStringSubmatch(titleValue); matches != nil {
				// Replace with hyphenated format: "2026-01-13"
				normalizedTitle := fmt.Sprintf("%s-%s-%s", matches[1], matches[2], matches[3])
				line = fmt.Sprintf("title: %s", normalizedTitle)
			}
		}
		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

func ensureJournalMetadata(content string, brainType brain.BrainType, brainName string) string {
	// First normalize any malformed titles
	content = normalizeJournalTitle(content)
	
	switch brainType {
	case brain.BrainTypeLogseq:
		return ensureLogseqProperty(content, "brain", brainName)
	default:
		return ensureYAMLFrontmatterField(content, "brain", brainName)
	}
}
