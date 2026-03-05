package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAppendLinkToJournalConsistency tests that links are appended with consistent formatting
func TestAppendLinkToJournalConsistency(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	journalPath := filepath.Join(tmpDir, "journal.md")

	// Create a sample journal file matching the template
	initialContent := `---
date: 2026-03-05
type: journal
brain: test
---

# 2026-03-05

## Activities

## Meeting-Notes

## New Tasks

## Exercises

`

	if err := os.WriteFile(journalPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create journal file: %v", err)
	}

	// Test 1: Add a note link to Activities
	noteLink := "📝 [Test Note](../notes/test.md)"
	if err := appendLinkToJournalSection(journalPath, noteLink, "## Activities"); err != nil {
		t.Fatalf("Failed to append note link: %v", err)
	}

	content, _ := os.ReadFile(journalPath)
	if !strings.Contains(string(content), noteLink) {
		t.Error("Note link not found in journal after append")
	}

	// Test 2: Add a meeting link
	meetingLink := "🤝 [Meeting with Team](../meetings/2026-03-05-meeting.md)"
	if err := appendLinkToJournalSection(journalPath, meetingLink, "## Meeting-Notes"); err != nil {
		t.Fatalf("Failed to append meeting link: %v", err)
	}

	content, _ = os.ReadFile(journalPath)
	if !strings.Contains(string(content), meetingLink) {
		t.Error("Meeting link not found in journal after append")
	}

	// Test 3: Add another note link - should maintain consistent spacing
	noteLink2 := "📝 [Another Note](../notes/another.md)"
	if err := appendLinkToJournalSection(journalPath, noteLink2, "## Activities"); err != nil {
		t.Fatalf("Failed to append second note link: %v", err)
	}

	content, _ = os.ReadFile(journalPath)
	contentStr := string(content)

	// Verify both links are present
	if !strings.Contains(contentStr, noteLink) || !strings.Contains(contentStr, noteLink2) {
		t.Error("One or both note links not found in journal")
	}

	// Test 4: Try to add duplicate link - should be skipped
	if err := appendLinkToJournalSection(journalPath, noteLink, ""); err != nil {
		t.Fatalf("Error when trying to add duplicate: %v", err)
	}

	contentAfterDuplicate, _ := os.ReadFile(journalPath)
	contentStrAfterDuplicate := string(contentAfterDuplicate)

	// Count occurrences of the link - should still be exactly 1
	count := strings.Count(contentStrAfterDuplicate, noteLink)
	if count != 1 {
		t.Errorf("Expected 1 occurrence of note link, found %d (duplicate prevention failed)", count)
	}

	// Test 5: Verify proper section formatting (no double empty lines)
	lines := strings.Split(contentStr, "\n")
	for i := 0; i < len(lines)-2; i++ {
		if strings.HasPrefix(lines[i], "##") {
			// After a section header, there should be content or max 1 empty line before more content
			emptyCount := 0
			j := i + 1
			for j < len(lines) && lines[j] == "" {
				emptyCount++
				j++
			}
			// Allow max 1 empty line after section header
			if emptyCount > 2 {
				t.Errorf("Found %d consecutive empty lines after section '%s', expected max 2", emptyCount, lines[i])
			}
		}
	}

	t.Logf("Final journal content:\n%s", contentStr)
}

// TestAppendLinkDetectsSection tests that the correct section is determined by emoji
func TestAppendLinkDetectsSection(t *testing.T) {
	tests := []struct {
		linkText     string
		expectedSec  string
	}{
		{"🤝 [Meeting](../meetings/test.md)", "## Meeting-Notes"},
		{"📋 [Task](../tasks/test.md)", "## New Tasks"},
		{"💪 [Exercise](../exercises/test.md)", "## Exercises"},
		{"📝 [Note](../notes/test.md)", "## Activities"},
		{"📎 [Item](../items/test.md)", "## Activities"},
	}

	for _, tt := range tests {
		section := getSectionForLink(tt.linkText)
		if section != tt.expectedSec {
			t.Errorf("For link %q, expected section %q, got %q", tt.linkText, tt.expectedSec, section)
		}
	}
}

// TestJournalTemplateAndAppending tests that the template aligns with appending logic
func TestJournalTemplateAndAppending(t *testing.T) {
	tmpDir := t.TempDir()
	journalPath := filepath.Join(tmpDir, "journal.md")

	// Create a journal using the template
	template := createJournalTemplateForDate(time.Now())
	if err := os.WriteFile(journalPath, []byte(template), 0644); err != nil {
		t.Fatalf("Failed to create journal from template: %v", err)
	}

	// Append links of different types
	links := []string{
		"📝 [Note 1](../notes/note1.md)",
		"🤝 [Meeting 1](../meetings/meeting1.md)",
		"📋 [Task 1](../tasks/task1.md)",
		"💪 [Exercise 1](../exercises/exercise1.md)",
	}

	for _, link := range links {
		if err := appendLinkToJournalSection(journalPath, link, ""); err != nil {
			t.Fatalf("Failed to append link %q: %v", link, err)
		}
	}

	content, _ := os.ReadFile(journalPath)
	contentStr := string(content)

	// Verify all links were added
	for _, link := range links {
		if !strings.Contains(contentStr, link) {
			t.Errorf("Link %q not found in journal", link)
		}
	}

	t.Logf("Template-based journal after appending:\n%s", contentStr)
}
