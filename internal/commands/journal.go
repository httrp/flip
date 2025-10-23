package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/templates"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func NewJournalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journal [new]",
		Short: "Create or open a daily journal note",
		Long:  "Create a new daily journal/daily note or open existing one. Prevents duplicate entries for the same date.\n\nExamples:\n  flip journal         # Create/open journal (default action)\n  flip journal new     # Create/open journal (explicit)\n  flip journal n       # Create/open journal (shortcut)\n  flip new journal     # Create/open journal (alternative syntax)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action: create/open journal
			// Support: flip journal, flip journal new, flip journal n
			if len(args) == 0 || args[0] == "new" || args[0] == "n" {
				return runCreateJournal()
			}
			return fmt.Errorf("unknown subcommand: %s", args[0])
		},
	}

	// Add explicit 'new' subcommand for clarity
	newCmd := &cobra.Command{
		Use:     "new",
		Aliases: []string{"n"},
		Short:   "Create or open a daily journal note (explicit)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateJournal()
		},
	}
	cmd.AddCommand(newCmd)

	return cmd
}

// runCreateJournal creates or opens a daily journal note
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
	content := generateJournalContent(targetDate, detection.Type)

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create journal: %w", err)
	}

	fmt.Printf("✓ Journal created successfully!\n")
	fmt.Printf("  Path: %s\n", filePath)
	fmt.Printf("  Brain: %s (%s)\n\n", activeBrain.Name, detection.Type)

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
func generateJournalContent(date time.Time, brainType brain.BrainType) string {
	dateStr := date.Format("2006-01-02")

	// Try to load template from file
	tmpl, err := templates.Load(brainType, templates.TemplateTypeJournal)
	if err != nil {
		// Fallback to hardcoded template if file not found
		fmt.Printf("Warning: Could not load template, using default (%v)\n", err)
		return generateDefaultJournalContent(date, brainType)
	}

	// Prepare template variables
	vars := map[string]string{
		"date":    dateStr,
		"id":      uuid.New().String(), // Proper UUID for Dendron compatibility
		"updated": fmt.Sprintf("%d", time.Now().Unix()),
		"created": fmt.Sprintf("%d", time.Now().Unix()),
	}

	return templates.Render(tmpl, vars)
}

// generateDefaultJournalContent provides fallback templates when template files don't exist
func generateDefaultJournalContent(date time.Time, brainType brain.BrainType) string {
	dateStr := date.Format("2006-01-02")
	weekday := date.Format("Monday")
	timeStr := time.Now().Format("15:04")

	switch brainType {
	case brain.BrainTypeLogseq:
		return fmt.Sprintf(`- %s, %s

## Morning

## Work

## Evening

## Tasks
- TODO Daily task example

## Notes

## Grateful For
- 

`, weekday, dateStr)

	case brain.BrainTypeObsidian:
		return fmt.Sprintf(`---
date: %s
day: %s
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

`, dateStr, weekday, weekday, dateStr)

	case brain.BrainTypeDendron:
		return fmt.Sprintf(`---
id: %s
title: Journal %s
desc: 'Daily journal entry'
updated: %d
created: %d
date: %s
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

`, uuid.New().String(), dateStr, time.Now().Unix(), time.Now().Unix(), dateStr, dateStr, weekday)

	case brain.BrainTypeFlip:
		return fmt.Sprintf(`---
date: %s
day: %s
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

`, dateStr, weekday, timeStr, weekday, dateStr)

	default:
		return fmt.Sprintf(`---
date: %s
day: %s
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

`, dateStr, weekday, weekday, dateStr)
	}
}
