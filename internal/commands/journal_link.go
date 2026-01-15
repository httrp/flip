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
	ItemType    string          // "note", "task", "exercise", etc.
	ItemName    string          // display name of the item
	ItemPath    string          // relative path to the item from brain root
	Brain       *Brain          // brain where the item was created (required)
	Interactive bool            // true = ask user, false = always add link
	Date        *time.Time      // optional: target date (nil = today)
	BrainType   brain.BrainType // brain type for correct link format
}

// AddLinkToJournal asks user if they want to add a link to today's journal entry.
// Uses the same brain where the item was created.
func AddLinkToJournal(opts JournalLinkOptions) error {
	if opts.Brain == nil {
		return fmt.Errorf("brain is required for journal link")
	}

	// Only ask in interactive mode
	if opts.Interactive {
		// Ask if user wants to skip adding link to journal (default: add it)
		promptLink := promptui.Select{
			Label: "Add link to today's journal?",
			Items: []string{"Add link (default)", "Skip"},
		}

		idx, _, err := promptLink.Run()
		if err != nil {
			// User cancelled: default to adding link
		}

		if idx == 1 { // Skip
			return nil
		}
		// idx == 0 or cancelled: add the link (default behavior)
	}

	// Detect brain type if not provided
	if opts.BrainType == "" {
		detector := brain.NewDetector()
		detection, err := detector.DetectBrainType(opts.Brain.Path)
		if err == nil {
			opts.BrainType = detection.Type
		}
	}

	// Get target date (default to today)
	targetDate := time.Now()
	if opts.Date != nil {
		targetDate = *opts.Date
	}
	dateStr := targetDate.Format("2006-01-02")

	// Build link format based on brain type
	linkText := buildJournalLink(opts, dateStr)

	// Get journal path for this brain and date
	journalPath := getJournalFilePathForDate(opts.Brain, targetDate)

	// Ensure journal exists
	if err := ensureJournalExistsForDate(journalPath, targetDate); err != nil {
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
		emoji = "📋"
	case "exercise":
		emoji = "💪"
	case "meeting":
		emoji = "🤝"
	}

	// Build link based on brain type
	switch opts.BrainType {
	case brain.BrainTypeLogseq:
		// Logseq uses [[wiki-links]] with page name (filename without extension)
		// Extract just the filename without path and extension
		filename := filepath.Base(opts.ItemPath)
		pageName := strings.TrimSuffix(filename, ".md")
		return fmt.Sprintf("%s [[%s]]", emoji, pageName)

	case brain.BrainTypeObsidian:
		// Obsidian also uses [[wiki-links]] but can include path
		filename := filepath.Base(opts.ItemPath)
		pageName := strings.TrimSuffix(filename, ".md")
		return fmt.Sprintf("%s [[%s]]", emoji, pageName)

	default:
		// Flip and others: use markdown links with relative path
		return fmt.Sprintf("%s [%s](%s)", emoji, opts.ItemName, opts.ItemPath)
	}
}

// getJournalFilePathForToday gets the journal file path for today in a brain
// Uses brain type detection to determine correct path and filename format
func getJournalFilePathForToday(b *Brain) string {
	return getJournalFilePathForDate(b, time.Now())
}

// getJournalFilePathForDate gets the journal file path for a specific date in a brain
func getJournalFilePathForDate(b *Brain, date time.Time) string {
	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(b.Path)
	if err != nil {
		// Fallback to flip default
		return filepath.Join(b.Path, "journal", date.Format("2006-01-02")+".md")
	}

	// Get journal directory based on brain type
	journalDir := getJournalDirectory(b.Path, detection.Type)

	// Get filename format based on brain type
	var filename string
	switch detection.Type {
	case brain.BrainTypeLogseq:
		// Logseq: YYYY_MM_DD.md (underscores)
		filename = date.Format("2006_01_02") + ".md"
	case brain.BrainTypeDendron:
		// Dendron: daily.journal.YYYY-MM-DD.md (hierarchy notation)
		filename = "daily.journal." + date.Format("2006-01-02") + ".md"
	default:
		// Obsidian, Foam, Flip, others: YYYY-MM-DD.md (dashes)
		filename = date.Format("2006-01-02") + ".md"
	}

	return filepath.Join(journalDir, filename)
}

// ensureJournalExists creates journal entry if it doesn't exist (for today)
func ensureJournalExists(journalPath string) error {
	return ensureJournalExistsForDate(journalPath, time.Now())
}

// ensureJournalExistsForDate creates journal entry if it doesn't exist for a specific date
func ensureJournalExistsForDate(journalPath string, date time.Time) error {
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
	content := createJournalTemplateForDate(date)

	return os.WriteFile(journalPath, []byte(content), 0644)
}

// createSimpleJournalTemplate creates a simple journal template for today
func createSimpleJournalTemplate() string {
	return createJournalTemplateForDate(time.Now())
}

// createJournalTemplateForDate creates a simple journal template for a specific date
func createJournalTemplateForDate(date time.Time) string {
	dateStr := date.Format("2006-01-02")
	weekday := date.Format("Monday")

	return fmt.Sprintf(`# %s - %s

## Activities

## Meeting-Notes

## New Tasks

## Exercises

`, dateStr, weekday)
}

// appendLinkToJournal appends a link to the journal file in the appropriate section
func appendLinkToJournal(journalPath, linkText string) error {
	return appendLinkToJournalSection(journalPath, linkText, "")
}

// appendLinkToJournalSection appends a link to a specific section in the journal file
// If section is empty, it determines the section from the link emoji
func appendLinkToJournalSection(journalPath, linkText, section string) error {
	// Read existing content
	content, err := os.ReadFile(journalPath)
	if err != nil {
		return err
	}

	fileContent := string(content)

	// Determine section from link text if not specified
	if section == "" {
		section = getSectionForLink(linkText)
	}

	// Try to find the target section
	if idx := strings.Index(fileContent, section); idx != -1 {
		// Found target section, insert after it
		insertPos := idx + len(section)

		// Skip to end of line
		for insertPos < len(fileContent) && fileContent[insertPos] != '\n' {
			insertPos++
		}
		// Move past one newline
		if insertPos < len(fileContent) && fileContent[insertPos] == '\n' {
			insertPos++
		}
		// Skip empty line if present
		if insertPos < len(fileContent) && fileContent[insertPos] == '\n' {
			insertPos++
		}

		// Build new content with link
		newContent := fileContent[:insertPos] + linkText + "\n" + fileContent[insertPos:]

		return os.WriteFile(journalPath, []byte(newContent), 0644)
	}

	// Fallback to Activities section
	if section != "## Activities" {
		if idx := strings.Index(fileContent, "## Activities"); idx != -1 {
			insertPos := idx + len("## Activities")
			for insertPos < len(fileContent) && fileContent[insertPos] != '\n' {
				insertPos++
			}
			if insertPos < len(fileContent) && fileContent[insertPos] == '\n' {
				insertPos++
			}
			if insertPos < len(fileContent) && fileContent[insertPos] == '\n' {
				insertPos++
			}
			newContent := fileContent[:insertPos] + linkText + "\n" + fileContent[insertPos:]
			return os.WriteFile(journalPath, []byte(newContent), 0644)
		}
	}

	// Ultimate fallback: append to end
	fileContent += "\n" + linkText + "\n"
	return os.WriteFile(journalPath, []byte(fileContent), 0644)
}

// getSectionForLink determines the journal section based on the link emoji
func getSectionForLink(linkText string) string {
	if strings.HasPrefix(linkText, "🤝") {
		return "## Meeting-Notes"
	}
	if strings.HasPrefix(linkText, "📋") {
		return "## New Tasks"
	}
	if strings.HasPrefix(linkText, "💪") {
		return "## Exercises"
	}
	// Default: Activities for notes and other items
	return "## Activities"
}
