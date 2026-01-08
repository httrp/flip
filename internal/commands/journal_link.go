package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/manifoldco/promptui"
)

// JournalLinkOptions holds options for adding a link to journal
type JournalLinkOptions struct {
	ItemType    string // "note", "task", "exercise", etc.
	ItemName    string // display name of the item
	ItemPath    string // relative path to the item from brain root
	Brain       *Brain // brain where the item was created (required)
	Interactive bool   // true = ask user, false = always add link
}

// AddLinkToJournal asks user if they want to add a link to today's journal entry.
// Uses the same brain where the item was created.
func AddLinkToJournal(opts JournalLinkOptions) error {
	if opts.Brain == nil {
		return fmt.Errorf("brain is required for journal link")
	}

	// Only ask in interactive mode
	if opts.Interactive {
		// Ask if user wants to add link
		promptLink := promptui.Select{
			Label: "Add link to today's journal?",
			Items: []string{"Yes", "No"},
		}

		idx, _, err := promptLink.Run()
		if err != nil {
			return nil // User cancelled
		}

		if idx == 1 { // No
			return nil
		}
	}

	// Get today's date
	today := time.Now()
	dateStr := today.Format("2006-01-02")

	// Build link format based on brain type
	linkText := buildJournalLink(opts, dateStr)

	// Get journal path for this brain
	journalPath := getJournalFilePathForToday(opts.Brain)

	// Ensure journal exists
	if err := ensureJournalExists(journalPath); err != nil {
		return fmt.Errorf("failed to ensure journal exists: %w", err)
	}

	// Add link to journal
	if err := appendLinkToJournal(journalPath, linkText); err != nil {
		return fmt.Errorf("failed to add link to journal: %w", err)
	}

	PrintOrJSON("✓ Link added to journal in brain '%s'\n", opts.Brain.Name)
	return nil
}

// buildJournalLink creates the link text for adding to journal
func buildJournalLink(opts JournalLinkOptions, dateStr string) string {
	// Emoji based on item type
	emoji := "📎" // default
	switch opts.ItemType {
	case "note":
		emoji = "📝"
	case "task":
		emoji = "✅"
	case "exercise":
		emoji = "💪"
	case "meeting":
		emoji = "🤝"
	}

	// Build markdown link [title](path)
	// Path is relative to brain, so we use it as-is
	relPath := opts.ItemPath

	return fmt.Sprintf("%s [%s](%s)", emoji, opts.ItemName, relPath)
}

// getJournalFilePathForToday gets the journal file path for today in a brain
// Uses brain type detection to determine correct path and filename format
func getJournalFilePathForToday(b *Brain) string {
	today := time.Now()

	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(b.Path)
	if err != nil {
		// Fallback to flip default
		return filepath.Join(b.Path, "journal", today.Format("2006-01-02")+".md")
	}

	// Get journal directory based on brain type
	journalDir := getJournalDirectory(b.Path, detection.Type)

	// Get filename format based on brain type
	var filename string
	switch detection.Type {
	case brain.BrainTypeLogseq:
		// Logseq: YYYY_MM_DD.md (underscores)
		filename = today.Format("2006_01_02") + ".md"
	case brain.BrainTypeDendron:
		// Dendron: daily.journal.YYYY-MM-DD.md (hierarchy notation)
		filename = "daily.journal." + today.Format("2006-01-02") + ".md"
	default:
		// Obsidian, Foam, Flip, others: YYYY-MM-DD.md (dashes)
		filename = today.Format("2006-01-02") + ".md"
	}

	return filepath.Join(journalDir, filename)
}

// ensureJournalExists creates journal entry if it doesn't exist
func ensureJournalExists(journalPath string) error {
	// Check if file exists
	if _, err := os.Stat(journalPath); err == nil {
		// File exists
		return nil
	}

	// Create journal directory
	journalDir := filepath.Dir(journalPath)
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return err
	}

	// Create empty journal file with header
	content := createSimpleJournalTemplate()

	return os.WriteFile(journalPath, []byte(content), 0644)
}

// createSimpleJournalTemplate creates a simple journal template
func createSimpleJournalTemplate() string {
	today := time.Now()
	dateStr := today.Format("2006-01-02")
	weekday := today.Format("Monday")

	return fmt.Sprintf(`# %s - %s

## Activities

`, dateStr, weekday)
}

// appendLinkToJournal appends a link to the journal file
func appendLinkToJournal(journalPath, linkText string) error {
	// Read existing content
	content, err := os.ReadFile(journalPath)
	if err != nil {
		return err
	}

	fileContent := string(content)

	// Find "## Activities" section and add link after it
	// or just append to end if section not found
	activitySection := "## Activities"
	if idx := strings.Index(fileContent, activitySection); idx != -1 {
		// Found Activities section, insert after it
		insertPos := idx + len(activitySection)

		// Skip to end of line
		for insertPos < len(fileContent) && fileContent[insertPos] != '\n' {
			insertPos++
		}
		// Move past newline(s)
		for insertPos < len(fileContent) && fileContent[insertPos] == '\n' {
			insertPos++
		}

		// Build new content with link
		newContent := fileContent[:insertPos] + linkText + "\n" + fileContent[insertPos:]

		return os.WriteFile(journalPath, []byte(newContent), 0644)
	}

	// Fallback: append to end
	fileContent += "\n" + linkText + "\n"
	return os.WriteFile(journalPath, []byte(fileContent), 0644)
}
