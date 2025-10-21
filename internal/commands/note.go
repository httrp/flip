package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func NewNoteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "note",
		Short: "Create a new note",
		Long:  "Create a new note in your active brain with appropriate template and naming conventions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateNote()
		},
	}
	return cmd
}

// runCreateNote creates a new note in the active brain
func runCreateNote() error {
	fmt.Println("\n📝 Create New Note")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Load config
	config, err := loadWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if we have any workspaces
	if len(config.Workspaces) == 0 {
		fmt.Println("⚠️  No workspaces found. Please run 'flip quickstart' first.")
		return nil
	}

	// Get active workspace
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get active workspace: %w", err)
	}

	// Check if workspace has any brains
	if len(activeWs.Brains) == 0 {
		fmt.Println("⚠️  No brains found in active workspace. Please add a brain first.")
		return nil
	}

	// Get default brain or let user select
	activeBrain, err := confirmOrSelectBrain(activeWs)
	if err != nil {
		return err
	}

	// Prompt for note title
	promptTitle := promptui.Prompt{
		Label: "Note title",
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("title cannot be empty")
			}
			return nil
		},
	}

	title, err := promptTitle.Run()
	if err != nil {
		return fmt.Errorf("title prompt cancelled: %w", err)
	}
	title = strings.TrimSpace(title)

	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(activeBrain.Path)
	if err != nil {
		return fmt.Errorf("failed to detect brain type: %w", err)
	}

	// Generate filename based on brain type
	filename := generateNoteFilename(title, detection.Type)

	// Determine target directory
	targetDir := getNotesDirectory(activeBrain.Path, detection.Type)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create notes directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(targetDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		fmt.Printf("\n⚠️  File already exists: %s\n", filePath)
		return nil
	}

	// Generate content based on brain type
	content := generateNoteContent(title, detection.Type)

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	fmt.Printf("\n✓ Note created: %s\n", filePath)
	fmt.Printf("  Brain: %s (%s)\n", activeBrain.Name, detection.Type)
	fmt.Println()

	return nil
}

// confirmOrSelectBrain asks user to confirm default brain or select another
func confirmOrSelectBrain(ws *Workspace) (*Brain, error) {
	// If there's only one brain, use it
	if len(ws.Brains) == 1 {
		brain := &ws.Brains[0]
		fmt.Printf("📍 Using brain: %s\n", brain.Name)
		fmt.Printf("   Path: %s\n\n", brain.Path)
		return brain, nil
	}

	// Find default brain
	var defaultBrain *Brain
	for i := range ws.Brains {
		if ws.Brains[i].Name == ws.DefaultBrain {
			defaultBrain = &ws.Brains[i]
			break
		}
	}

	// If no default, use first brain
	if defaultBrain == nil {
		defaultBrain = &ws.Brains[0]
	}

	// Ask for confirmation
	fmt.Printf("📍 Default brain: %s\n", defaultBrain.Name)
	fmt.Printf("   Path: %s\n\n", defaultBrain.Path)

	promptConfirm := promptui.Select{
		Label: "Use this brain?",
		Items: []string{"Yes, use default", "No, select different brain"},
	}

	idx, _, err := promptConfirm.Run()
	if err != nil {
		return nil, fmt.Errorf("confirmation prompt cancelled: %w", err)
	}

	if idx == 0 {
		return defaultBrain, nil
	}

	// Let user select brain
	brainNames := make([]string, len(ws.Brains))
	for i, b := range ws.Brains {
		brainNames[i] = fmt.Sprintf("%s (%s)", b.Name, b.Type)
	}

	promptBrain := promptui.Select{
		Label: "Select brain",
		Items: brainNames,
	}

	brainIdx, _, err := promptBrain.Run()
	if err != nil {
		return nil, fmt.Errorf("brain selection cancelled: %w", err)
	}

	return &ws.Brains[brainIdx], nil
}

// generateNoteFilename creates a filename based on brain type conventions
func generateNoteFilename(title string, brainType brain.BrainType) string {
	// Sanitize title for filename
	safeName := strings.ToLower(title)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	safeName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, safeName)

	now := time.Now()

	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: YYYY_MM_DD___title.md
		return fmt.Sprintf("%s___%s.md", now.Format("2006_01_02"), safeName)

	case brain.BrainTypeObsidian:
		// Obsidian: simple title.md (no date prefix usually)
		return fmt.Sprintf("%s.md", safeName)

	case brain.BrainTypeDendron:
		// Dendron: notes.YYYY-MM-DD-title.md
		return fmt.Sprintf("notes.%s-%s.md", now.Format("2006-01-02"), safeName)

	case brain.BrainTypeFlip:
		// Flip: YYYY-MM-DD-title.md
		return fmt.Sprintf("%s-%s.md", now.Format("2006-01-02"), safeName)

	default:
		// Generic: YYYY-MM-DD-title.md
		return fmt.Sprintf("%s-%s.md", now.Format("2006-01-02"), safeName)
	}
}

// getNotesDirectory returns the appropriate directory for notes based on brain type
func getNotesDirectory(brainPath string, brainType brain.BrainType) string {
	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: pages/ directory
		return filepath.Join(brainPath, "pages")

	case brain.BrainTypeObsidian:
		// Obsidian: root or Notes/ if it exists
		notesDir := filepath.Join(brainPath, "Notes")
		if _, err := os.Stat(notesDir); err == nil {
			return notesDir
		}
		return brainPath

	case brain.BrainTypeDendron:
		// Dendron: root directory
		return brainPath

	case brain.BrainTypeFlip:
		// Flip: notes/ directory
		return filepath.Join(brainPath, "notes")

	default:
		// Generic: root directory
		return brainPath
	}
}

// generateNoteContent creates note content with appropriate frontmatter/metadata
func generateNoteContent(title string, brainType brain.BrainType) string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04")

	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: simple markdown with properties
		return fmt.Sprintf(`- title:: %s
- created:: %s %s
- tags:: 

## %s

`, title, dateStr, timeStr, title)

	case brain.BrainTypeObsidian:
		// Obsidian: YAML frontmatter
		return fmt.Sprintf(`---
title: %s
created: %s
tags: []
---

# %s

`, title, dateStr, title)

	case brain.BrainTypeDendron:
		// Dendron: YAML frontmatter
		return fmt.Sprintf(`---
id: %s
title: %s
desc: ''
updated: %d
created: %d
---

# %s

`, generateID(), title, now.Unix(), now.Unix(), title)

	case brain.BrainTypeFlip:
		// Flip: YAML frontmatter
		return fmt.Sprintf(`---
title: %s
created: %s
updated: %s
type: note
tags: []
---

# %s

`, title, dateStr, dateStr, title)

	default:
		// Generic: simple markdown with frontmatter
		return fmt.Sprintf(`---
title: %s
date: %s
---

# %s

`, title, dateStr, title)
	}
}

// generateID creates a unique ID for notes (used by Dendron)
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
