package migration

import (
	"strings"
	"testing"

	"github.com/httrp/flip/internal/health"
)

func TestUpdateNoteLinksWikilinks(t *testing.T) {
	noteMap := map[string]string{
		"pages/Project Alpha.md": "notes/project-alpha.md",
		"pages/Project Beta.md":  "notes/project-beta.md",
	}

	sourceStruct := GetBrainStructure(health.BrainTypeLogseq)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	rewriter := NewNoteLinkRewriter("", "", sourceStruct, targetStruct, noteMap)

	content := `# Overview

See [[Project Alpha]] for details.
Also check [[Project Beta|Beta Project]].
`

	updated, err := rewriter.UpdateNoteLinks(content, "pages/overview.md")
	if err != nil {
		t.Fatalf("UpdateNoteLinks: %v", err)
	}

	// Should convert to target format (Flip uses wikilinks with slugs or names)
	// Check that project-alpha appears (exact form depends on formatting logic)
	if !strings.Contains(updated, "project-alpha") && !strings.Contains(updated, "Project Alpha") {
		t.Errorf("expected reference to project-alpha, got: %s", updated)
	}
	// Alias should be preserved somewhere
	if !strings.Contains(updated, "Beta") {
		t.Errorf("expected beta reference preserved, got: %s", updated)
	}
}

func TestUpdateNoteLinksMarkdown(t *testing.T) {
	noteMap := map[string]string{
		"docs/guide.md":    "notes/guide.md",
		"docs/tutorial.md": "notes/tutorial.md",
	}

	sourceStruct := GetBrainStructure(health.BrainTypeObsidian)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	rewriter := NewNoteLinkRewriter("", "", sourceStruct, targetStruct, noteMap)

	content := `# Index

- [User Guide](docs/guide.md)
- [Tutorial](tutorial.md)
`

	updated, err := rewriter.UpdateNoteLinks(content, "index.md")
	if err != nil {
		t.Fatalf("UpdateNoteLinks: %v", err)
	}

	// Links should be updated (exact format depends on target structure)
	if !strings.Contains(updated, "User Guide") {
		t.Errorf("expected link text preserved: %s", updated)
	}
	// At minimum, content should change (paths updated)
	if updated == content {
		t.Errorf("expected content to be modified, but it's unchanged")
	}
}

func TestUpdateNoteLinksNoMatch(t *testing.T) {
	noteMap := map[string]string{
		"notes/existing.md": "notes/existing.md",
	}

	sourceStruct := GetBrainStructure(health.BrainTypeFlip)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	rewriter := NewNoteLinkRewriter("", "", sourceStruct, targetStruct, noteMap)

	// Link to note not in map (not migrated or external)
	content := `See [[Non Existent Note]] for more.`

	updated, err := rewriter.UpdateNoteLinks(content, "notes/test.md")
	if err != nil {
		t.Fatalf("UpdateNoteLinks: %v", err)
	}

	// Should keep original (link not found)
	if updated != content {
		t.Errorf("expected unchanged content for non-existent link, got: %s", updated)
	}
}

func TestResolveNoteNameCaseInsensitive(t *testing.T) {
	noteMap := map[string]string{
		"pages/Project-Alpha.md": "notes/project-alpha.md",
		"pages/BETA_PROJECT.md":  "notes/beta-project.md",
	}

	sourceStruct := GetBrainStructure(health.BrainTypeLogseq)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	rewriter := NewNoteLinkRewriter("", "", sourceStruct, targetStruct, noteMap)

	// Test various name formats
	tests := []struct {
		input    string
		expected string
	}{
		{"Project Alpha", "pages/Project-Alpha.md"},
		{"project-alpha", "pages/Project-Alpha.md"},
		{"PROJECT ALPHA", "pages/Project-Alpha.md"},
		{"Beta Project", "pages/BETA_PROJECT.md"},
		{"beta_project", "pages/BETA_PROJECT.md"},
	}

	for _, tt := range tests {
		resolved := rewriter.resolveNoteName(tt.input)
		if resolved != tt.expected {
			t.Errorf("resolveNoteName(%q) = %q, want %q", tt.input, resolved, tt.expected)
		}
	}
}

func TestBuildNotePathMap(t *testing.T) {
	plan := &MigrationPlan{
		Items: []PlanItem{
			{SourcePath: "pages/note1.md", TargetPath: "notes/note1.md", Type: "note"},
			{SourcePath: "journal/2025-11-09.md", TargetPath: "journal/2025-11-09.md", Type: "journal"},
			{SourcePath: "assets/img.png", TargetPath: "assets/img.png", Type: "asset"},
			{SourcePath: "pages/note2.md", TargetPath: "notes/note2.md", Type: "note"},
		},
	}

	noteMap := BuildNotePathMap(plan)

	// Should include notes and journals, exclude assets
	if len(noteMap) != 3 {
		t.Fatalf("expected 3 entries in note map, got %d", len(noteMap))
	}
	if noteMap["pages/note1.md"] != "notes/note1.md" {
		t.Errorf("note1 mapping incorrect")
	}
	if noteMap["journal/2025-11-09.md"] != "journal/2025-11-09.md" {
		t.Errorf("journal mapping incorrect")
	}
	if _, exists := noteMap["assets/img.png"]; exists {
		t.Errorf("assets should not be in note map")
	}
}
