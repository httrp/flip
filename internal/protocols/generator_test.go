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
			Notes:        "Backend dependencies were reviewed. Budget risk was raised.",
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
			Notes:        "Budget risk was raised. Release plan was aligned.",
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
	if !strings.Contains(markdown, "protocol_type: rolling") {
		t.Error("Markdown should contain rolling protocol frontmatter")
	}

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

	if !strings.Contains(markdown, "## Changes Since Last Meeting") {
		t.Error("Markdown should contain a changes since last meeting section")
	}

	if !strings.Contains(markdown, "### Actions") {
		t.Error("Markdown should contain the actions delta section")
	}

	if !strings.Contains(markdown, "### Information Overview") {
		t.Error("Markdown should contain information overview")
	}

	if !strings.Contains(markdown, "Budget risk was raised (last updated: 2026-01-15, seen 2 times, Sprint Planning Week 2)") {
		t.Error("Markdown should collapse recurring information into a single summary entry")
	}

	if !strings.Contains(markdown, "Release plan was aligned") {
		t.Error("Markdown should contain latest information statements")
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

	if !strings.Contains(markdown, "DONE: Setup Jira board") {
		t.Error("Markdown should contain completed action changes from the latest meeting")
	}

	if !strings.Contains(markdown, "last seen: 2026-01-15") {
		t.Error("Markdown should contain action progress metadata")
	}

	if !strings.Contains(markdown, "## Manual Notes") {
		t.Error("Markdown should contain a preserved manual notes section")
	}

	if !strings.Contains(markdown, RollingProtocolManualNotesStart) || !strings.Contains(markdown, RollingProtocolManualNotesEnd) {
		t.Error("Markdown should contain manual notes preserve markers")
	}

	if !strings.Contains(markdown, "## Open Questions") || !strings.Contains(markdown, RollingProtocolOpenQuestionsStart) || !strings.Contains(markdown, RollingProtocolOpenQuestionsEnd) {
		t.Error("Markdown should contain preserved open questions section")
	}

	if !strings.Contains(markdown, "## Follow-Ups") || !strings.Contains(markdown, RollingProtocolFollowUpsStart) || !strings.Contains(markdown, RollingProtocolFollowUpsEnd) {
		t.Error("Markdown should contain preserved follow-ups section")
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

func TestAggregateActionSummariesMergesRecurringTasks(t *testing.T) {
	meetings := []*MeetingContent{
		{
			Date:  "2026-01-08",
			Title: "Week 1",
			ActionItems: []ActionItem{
				{Text: "Setup Jira board", Assignee: "bob", Completed: false},
			},
		},
		{
			Date:  "2026-01-15",
			Title: "Week 2",
			ActionItems: []ActionItem{
				{Text: "Setup Jira board", Assignee: "bob", Completed: true},
			},
		},
	}

	protocol := GenerateRollingProtocol(meetings, "Sprint Planning")
	if protocol == nil {
		t.Fatal("GenerateRollingProtocol returned nil")
	}

	if len(protocol.Actions) != 1 {
		t.Fatalf("Actions count = %d, want 1", len(protocol.Actions))
	}

	action := protocol.Actions[0]
	if !action.Completed {
		t.Fatal("Expected merged action to be completed")
	}
	if action.Occurrences != 2 {
		t.Fatalf("Occurrences = %d, want 2", action.Occurrences)
	}
	if action.LastMeetingDate != "2026-01-15" {
		t.Fatalf("LastMeetingDate = %q, want %q", action.LastMeetingDate, "2026-01-15")
	}
}

func TestLatestChangesDetectsActionTransitions(t *testing.T) {
	meetings := []*MeetingContent{
		{
			Date:  "2026-01-08",
			Title: "Week 1",
			ActionItems: []ActionItem{
				{Text: "Setup Jira board", Assignee: "bob", Completed: false},
			},
		},
		{
			Date:  "2026-01-15",
			Title: "Week 2",
			Notes: "Budget risk was raised. Release plan was aligned.",
			ActionItems: []ActionItem{
				{Text: "Setup Jira board", Assignee: "bob", Completed: true},
				{Text: "Review backlog", Assignee: "alice", Completed: false},
			},
			Decisions: []Decision{
				{Text: "Decision 2"},
			},
		},
	}

	protocol := GenerateRollingProtocol(meetings, "Sprint Planning")
	changes := protocol.latestChanges()
	if changes == nil {
		t.Fatal("latestChanges() returned nil")
	}

	if len(changes.Information) != 2 {
		t.Fatalf("Information count = %d, want 2", len(changes.Information))
	}
	if len(changes.Decisions) != 1 {
		t.Fatalf("Decision count = %d, want 1", len(changes.Decisions))
	}
	if len(changes.CompletedActions) != 1 {
		t.Fatalf("Completed action count = %d, want 1", len(changes.CompletedActions))
	}
	if len(changes.NewActions) != 1 {
		t.Fatalf("New action count = %d, want 1", len(changes.NewActions))
	}
}

func TestAggregateInformationCondensesRecurringStatements(t *testing.T) {
	meetings := []*MeetingContent{
		{
			Date:  "2026-01-08",
			Title: "Week 1",
			Notes: "Budget risk was raised. Backend dependencies were reviewed.",
		},
		{
			Date:  "2026-01-15",
			Title: "Week 2",
			Notes: "Budget risk was raised\n- Release plan was aligned",
		},
	}

	protocol := GenerateRollingProtocol(meetings, "Sprint Planning")
	if protocol == nil {
		t.Fatal("GenerateRollingProtocol returned nil")
	}

	if len(protocol.Information) != 3 {
		t.Fatalf("Information count = %d, want 3", len(protocol.Information))
	}

	budget := protocol.Information[0]
	if budget.Text != "Budget risk was raised" {
		t.Fatalf("First information text = %q, want %q", budget.Text, "Budget risk was raised")
	}
	if budget.Occurrences != 2 {
		t.Fatalf("Occurrences = %d, want 2", budget.Occurrences)
	}
	if budget.LastMeetingDate != "2026-01-15" {
		t.Fatalf("LastMeetingDate = %q, want %q", budget.LastMeetingDate, "2026-01-15")
	}
	if budget.LastMeetingTitle != "Week 2" {
		t.Fatalf("LastMeetingTitle = %q, want %q", budget.LastMeetingTitle, "Week 2")
	}
}

func TestExtractInformationStatementsSplitsBulletsAndSentences(t *testing.T) {
	notes := "Budget risk was raised. Release plan was aligned.\n- Backend dependencies were reviewed"

	got := extractInformationStatements(notes)
	if len(got) != 3 {
		t.Fatalf("extractInformationStatements() count = %d, want 3", len(got))
	}

	want := []string{"Budget risk was raised", "Release plan was aligned", "Backend dependencies were reviewed"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("extractInformationStatements()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
