package protocols

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMeetingFile(t *testing.T) {
	// Create a test meeting file
	content := `---
title: Test Meeting
created: 2026-01-15
updated: 2026-01-15
type: meeting
date: 2026-01-15
time: 14:30
duration: 60min
author: John Doe
tags: [meeting, test]
organization: TestOrg
project: TestProject
context: TestContext
series: Test Series
---

# Test Meeting

## Participants
- Alice Smith
- Bob Johnson

## Agenda
- Discuss project timeline
- Review technical approach

## Notes
We decided to use React for the frontend. The team agreed to start with TypeScript.

Some additional discussion points here.

## Action Items
- [ ] Setup repository (@alice, due: 2026-01-20, priority: high)
- [x] Create project plan (@bob)
- [ ] Review architecture

## Related
- [[related-note-1]]
- [[related-note-2]]
`

	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-meeting.md")

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the file
	meeting, err := ParseMeetingFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseMeetingFile failed: %v", err)
	}

	// Verify frontmatter
	if meeting.Title != "Test Meeting" {
		t.Errorf("Title = %q, want %q", meeting.Title, "Test Meeting")
	}

	if meeting.Date != "2026-01-15" {
		t.Errorf("Date = %q, want %q", meeting.Date, "2026-01-15")
	}

	if meeting.Organization != "TestOrg" {
		t.Errorf("Organization = %q, want %q", meeting.Organization, "TestOrg")
	}

	if meeting.Series != "Test Series" {
		t.Errorf("Series = %q, want %q", meeting.Series, "Test Series")
	}

	// Verify participants
	if len(meeting.Participants) != 2 {
		t.Errorf("Participants count = %d, want 2", len(meeting.Participants))
	} else {
		if meeting.Participants[0] != "Alice Smith" {
			t.Errorf("Participant[0] = %q, want %q", meeting.Participants[0], "Alice Smith")
		}
	}

	// Verify action items
	if len(meeting.ActionItems) != 3 {
		t.Errorf("ActionItems count = %d, want 3", len(meeting.ActionItems))
	} else {
		// Check first action item
		item := meeting.ActionItems[0]
		if item.Completed {
			t.Error("First action item should not be completed")
		}
		if item.Assignee != "alice" {
			t.Errorf("Assignee = %q, want %q", item.Assignee, "alice")
		}
		if item.DueDate != "2026-01-20" {
			t.Errorf("DueDate = %q, want %q", item.DueDate, "2026-01-20")
		}
		if item.Priority != "high" {
			t.Errorf("Priority = %q, want %q", item.Priority, "high")
		}

		// Check second action item (completed)
		item2 := meeting.ActionItems[1]
		if !item2.Completed {
			t.Error("Second action item should be completed")
		}
	}

	// Verify decisions were extracted
	if len(meeting.Decisions) == 0 {
		t.Error("Expected decisions to be extracted from notes")
	}

	// Verify related links
	if len(meeting.Related) != 2 {
		t.Errorf("Related count = %d, want 2", len(meeting.Related))
	}

	// Verify series detection
	if !meeting.IsSeriesMeeting() {
		t.Error("Meeting should be part of a series")
	}

	if meeting.GetSeriesName() != "Test Series" {
		t.Errorf("SeriesName = %q, want %q", meeting.GetSeriesName(), "Test Series")
	}
}

func TestParseActionItems(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
		wantText string
	}{
		{
			name:     "Simple task",
			content:  "- [ ] Task 1",
			expected: 1,
			wantText: "Task 1",
		},
		{
			name:     "Completed task",
			content:  "- [x] Done task",
			expected: 1,
			wantText: "Done task",
		},
		{
			name: "Multiple tasks",
			content: `- [ ] Task 1
- [x] Task 2
- [ ] Task 3`,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := parseActionItems(tt.content)
			if len(items) != tt.expected {
				t.Errorf("got %d items, want %d", len(items), tt.expected)
			}
			if tt.wantText != "" && len(items) > 0 {
				if items[0].Text != tt.wantText {
					t.Errorf("got text %q, want %q", items[0].Text, tt.wantText)
				}
			}
		})
	}
}

func TestExtractAssignee(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"Task @john", "john"},
		{"Task (@alice)", "alice"},
		{"Task without assignee", ""},
		{"Multiple @john @bob", "john"}, // Takes first match
	}

	for _, tt := range tests {
		result := extractAssignee(tt.text)
		if result != tt.expected {
			t.Errorf("extractAssignee(%q) = %q, want %q", tt.text, result, tt.expected)
		}
	}
}

func TestExtractDueDate(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"Task due: 2026-01-20", "2026-01-20"},
		{"Task Deadline: 2026-02-15", "2026-02-15"},
		{"Task without date", ""},
	}

	for _, tt := range tests {
		result := extractDueDate(tt.text)
		if result != tt.expected {
			t.Errorf("extractDueDate(%q) = %q, want %q", tt.text, result, tt.expected)
		}
	}
}

func TestExtractPriority(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"Task Priority: High", "high"},
		{"Task priority: low", "low"},
		{"Task Priority: MEDIUM", "medium"},
		{"Task without priority", ""},
	}

	for _, tt := range tests {
		result := extractPriority(tt.text)
		if result != tt.expected {
			t.Errorf("extractPriority(%q) = %q, want %q", tt.text, result, tt.expected)
		}
	}
}
