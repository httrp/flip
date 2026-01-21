package protocols

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSingleProtocol(t *testing.T) {
	// Create test meeting
	content := `---
title: Project Kickoff
date: 2026-01-15
time: 14:00
duration: 60min
organization: TestCorp
participants: []
---

# Project Kickoff

## Participants
- Alice
- Bob

## Notes
We decided to use React for the project.

## Action Items
- [ ] Setup repository (@alice, due: 2026-01-20)
- [x] Book meeting room (@bob)
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "meeting.md")

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse meeting
	meeting, err := ParseMeetingFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to parse meeting: %v", err)
	}

	// Generate protocol
	protocol := GenerateSingleProtocol(meeting)

	if protocol.Type != ProtocolTypeSingle {
		t.Errorf("Type = %v, want %v", protocol.Type, ProtocolTypeSingle)
	}

	if len(protocol.Meetings) != 1 {
		t.Errorf("Meetings count = %d, want 1", len(protocol.Meetings))
	}

	// Render markdown
	markdown := protocol.RenderMarkdown()

	// Verify content
	if !strings.Contains(markdown, "# Meeting Protocol: Project Kickoff") {
		t.Error("Markdown should contain protocol title")
	}

	if !strings.Contains(markdown, "**Date**: 2026-01-15") {
		t.Error("Markdown should contain date")
	}

	if !strings.Contains(markdown, "**Organization**: TestCorp") {
		t.Error("Markdown should contain organization")
	}

	if !strings.Contains(markdown, "Alice, Bob") {
		t.Error("Markdown should contain participants")
	}

	if !strings.Contains(markdown, "## Action Items") {
		t.Error("Markdown should contain action items section")
	}

	if !strings.Contains(markdown, "Setup repository") {
		t.Error("Markdown should contain action items")
	}

	if !strings.Contains(markdown, "[x] Book meeting room") {
		t.Error("Markdown should show completed items")
	}
}

func TestGenerateRollingProtocol(t *testing.T) {
	// Create multiple test meetings
	meetings := []*MeetingContent{
		{
			Title:        "Sprint Planning Week 1",
			Date:         "2026-01-08",
			Organization: "TestCorp",
			Series:       "Sprint Planning",
			Participants: []string{"Alice", "Bob"},
			ActionItems: []ActionItem{
				{Text: "Define sprint goals", Assignee: "alice", Completed: true},
				{Text: "Setup Jira board", Assignee: "bob", Completed: false},
			},
			Decisions: []Decision{
				{Text: "We decided to use 2-week sprints"},
			},
		},
		{
			Title:        "Sprint Planning Week 2",
			Date:         "2026-01-15",
			Organization: "TestCorp",
			Series:       "Sprint Planning",
			Participants: []string{"Alice", "Bob", "Charlie"},
			ActionItems: []ActionItem{
				{Text: "Review backlog", Assignee: "charlie", Completed: false},
				{Text: "Setup Jira board", Assignee: "bob", Completed: true}, // Carried over, now completed
			},
			Decisions: []Decision{
				{Text: "The team agreed to adopt TypeScript"},
			},
		},
	}

	// Generate rolling protocol
	protocol := GenerateRollingProtocol(meetings, "Sprint Planning")

	if protocol == nil {
		t.Fatal("GenerateRollingProtocol returned nil")
	}

	if protocol.Type != ProtocolTypeRolling {
		t.Errorf("Type = %v, want %v", protocol.Type, ProtocolTypeRolling)
	}

	if protocol.SeriesName != "Sprint Planning" {
		t.Errorf("SeriesName = %q, want %q", protocol.SeriesName, "Sprint Planning")
	}

	if len(protocol.Meetings) != 2 {
		t.Errorf("Meetings count = %d, want 2", len(protocol.Meetings))
	}

	// Render markdown
	markdown := protocol.RenderMarkdown()

	// Verify content
	if !strings.Contains(markdown, "# Rolling Protocol: Sprint Planning") {
		t.Error("Markdown should contain rolling protocol title")
	}

	if !strings.Contains(markdown, "**Meetings**: 2") {
		t.Error("Markdown should show meeting count")
	}

	if !strings.Contains(markdown, "## Timeline") {
		t.Error("Markdown should contain timeline section")
	}

	if !strings.Contains(markdown, "2026-01-08: Sprint Planning Week 1") {
		t.Error("Markdown should contain first meeting")
	}

	if !strings.Contains(markdown, "2026-01-15: Sprint Planning Week 2") {
		t.Error("Markdown should contain second meeting")
	}

	if !strings.Contains(markdown, "## Summary") {
		t.Error("Markdown should contain summary section")
	}

	if !strings.Contains(markdown, "### Action Items Overview") {
		t.Error("Markdown should aggregate action items")
	}

	if !strings.Contains(markdown, "### Decisions Timeline") {
		t.Error("Markdown should aggregate decisions")
	}

	// Verify decisions are shown
	if !strings.Contains(markdown, "2-week sprints") {
		t.Error("Markdown should contain first decision")
	}

	if !strings.Contains(markdown, "TypeScript") {
		t.Error("Markdown should contain second decision")
	}
}

func TestAggregateActionItems(t *testing.T) {
	meetings := []*MeetingContent{
		{
			ActionItems: []ActionItem{
				{Text: "Task 1", Completed: true},
				{Text: "Task 2", Completed: false},
			},
		},
		{
			ActionItems: []ActionItem{
				{Text: "Task 3", Completed: false},
			},
		},
	}

	protocol := &Protocol{Meetings: meetings}
	items := protocol.aggregateActionItems()

	if len(items) != 3 {
		t.Errorf("aggregateActionItems() = %d items, want 3", len(items))
	}
}

func TestAggregateDecisions(t *testing.T) {
	meetings := []*MeetingContent{
		{
			Decisions: []Decision{
				{Text: "Decision 1"},
			},
		},
		{
			Decisions: []Decision{
				{Text: "Decision 2"},
				{Text: "Decision 3"},
			},
		},
	}

	protocol := &Protocol{Meetings: meetings}
	decisions := protocol.aggregateDecisions()

	if len(decisions) != 3 {
		t.Errorf("aggregateDecisions() = %d decisions, want 3", len(decisions))
	}
}
