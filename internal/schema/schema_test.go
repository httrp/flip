package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractSchemaVersion(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *SchemaVersion
	}{
		{
			name: "YAML frontmatter",
			content: `---
title: Test Note
schema: journal
schema_version: 1.0.0
---

Content here`,
			expected: &SchemaVersion{
				SchemaName:    "journal",
				SchemaVersion: "1.0.0",
			},
		},
		{
			name: "Logseq properties",
			content: `schema:: meeting
schema_version:: 2.0.0

# Meeting Notes`,
			expected: &SchemaVersion{
				SchemaName:    "meeting",
				SchemaVersion: "2.0.0",
			},
		},
		{
			name: "No schema",
			content: `---
title: Simple Note
---

Just content`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSchemaVersion(tt.content)
			if tt.expected == nil {
				if result != nil {
					t.Error("Expected nil, got result")
				}
				return
			}
			if result == nil {
				t.Fatal("Expected result, got nil")
			}
			if result.SchemaName != tt.expected.SchemaName {
				t.Errorf("SchemaName = %q, want %q", result.SchemaName, tt.expected.SchemaName)
			}
			if result.SchemaVersion != tt.expected.SchemaVersion {
				t.Errorf("SchemaVersion = %q, want %q", result.SchemaVersion, tt.expected.SchemaVersion)
			}
		})
	}
}

func TestExtractFrontmatter(t *testing.T) {
	content := `---
title: Test
date: 2025-12-20
tags: work, project
---

Content`

	fm := extractFrontmatter(content)

	if fm["title"] != "Test" {
		t.Errorf("title = %q, want 'Test'", fm["title"])
	}
	if fm["date"] != "2025-12-20" {
		t.Errorf("date = %q, want '2025-12-20'", fm["date"])
	}
}

func TestHasSection(t *testing.T) {
	content := `# Main Title

## Summary

Some summary text.

## Notes

More notes here.

### Subsection

Details.`

	if !hasSection(content, "## Summary") {
		t.Error("Should find '## Summary' section")
	}
	if !hasSection(content, "## Notes") {
		t.Error("Should find '## Notes' section")
	}
	if hasSection(content, "## Missing") {
		t.Error("Should not find '## Missing' section")
	}
}

func TestRenameField(t *testing.T) {
	tests := []struct {
		name    string
		content string
		oldName string
		newName string
		want    string
		changed bool
	}{
		{
			name:    "YAML field rename",
			content: "---\nold_field: value\n---\n",
			oldName: "old_field",
			newName: "new_field",
			want:    "---\nnew_field: value\n---\n",
			changed: true,
		},
		{
			name:    "Logseq property rename",
			content: "old_field:: value\n",
			oldName: "old_field",
			newName: "new_field",
			want:    "new_field:: value\n",
			changed: true,
		},
		{
			name:    "Field not found",
			content: "other_field: value\n",
			oldName: "missing_field",
			newName: "new_field",
			want:    "other_field: value\n",
			changed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, changed := renameField(tt.content, tt.oldName, tt.newName)
			if result != tt.want {
				t.Errorf("renameField() = %q, want %q", result, tt.want)
			}
			if changed != tt.changed {
				t.Errorf("changed = %v, want %v", changed, tt.changed)
			}
		})
	}
}

func TestSchemaManager_LoadSchemas(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()
	schemasDir := filepath.Join(tempDir, "definitions", "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test schema file
	schemaContent := `
version: "1.0.0"
name: journal
description: Daily journal entry schema
fields:
  - name: date
    type: date
    required: true
  - name: mood
    type: string
    required: false
    default: neutral
sections:
  - name: summary
    heading: "## Summary"
    required: true
`
	schemaPath := filepath.Join(schemasDir, "journal.yaml")
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test loading
	sm := NewSchemaManager(tempDir)
	if err := sm.LoadSchemas(); err != nil {
		t.Fatalf("LoadSchemas failed: %v", err)
	}

	// Verify schema was loaded
	schema, ok := sm.GetSchema("journal")
	if !ok {
		t.Fatal("journal schema not found")
	}

	if schema.Version != "1.0.0" {
		t.Errorf("Version = %q, want '1.0.0'", schema.Version)
	}

	if len(schema.Fields) != 2 {
		t.Errorf("Fields count = %d, want 2", len(schema.Fields))
	}

	if len(schema.Sections) != 1 {
		t.Errorf("Sections count = %d, want 1", len(schema.Sections))
	}
}

func TestSchemaManager_ValidateNote(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()
	schemasDir := filepath.Join(tempDir, "definitions", "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create schema
	schemaContent := `
version: "2.0.0"
name: journal
fields:
  - name: date
    type: date
    required: true
  - name: title
    type: string
    required: true
sections:
  - name: summary
    heading: "## Summary"
    required: true
`
	if err := os.WriteFile(filepath.Join(schemasDir, "journal.yaml"), []byte(schemaContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test notes
	validNote := `---
schema: journal
schema_version: 2.0.0
date: 2025-12-20
title: Test Entry
---

## Summary

A valid entry.
`
	invalidNote := `---
schema: journal
schema_version: 2.0.0
date: 2025-12-20
---

Missing title and summary section.
`
	needsMigrationNote := `---
schema: journal
schema_version: 1.0.0
date: 2025-12-20
title: Old Entry
---

## Summary

Old version.
`

	notesDir := filepath.Join(tempDir, "journal")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notesDir, "valid.md"), []byte(validNote), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notesDir, "invalid.md"), []byte(invalidNote), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(notesDir, "old.md"), []byte(needsMigrationNote), 0644); err != nil {
		t.Fatal(err)
	}

	// Test validation
	sm := NewSchemaManager(tempDir)
	if err := sm.LoadSchemas(); err != nil {
		t.Fatal(err)
	}

	// Valid note
	result, err := sm.ValidateNote(filepath.Join(notesDir, "valid.md"))
	if err != nil {
		t.Fatalf("ValidateNote failed: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid note should be valid. Errors: %v", result.Errors)
	}
	if result.NeedsMigration {
		t.Error("Valid note should not need migration")
	}

	// Invalid note
	result, err = sm.ValidateNote(filepath.Join(notesDir, "invalid.md"))
	if err != nil {
		t.Fatalf("ValidateNote failed: %v", err)
	}
	if result.Valid {
		t.Error("Invalid note should not be valid")
	}
	if len(result.Errors) == 0 {
		t.Error("Invalid note should have errors")
	}

	// Note needing migration
	result, err = sm.ValidateNote(filepath.Join(notesDir, "old.md"))
	if err != nil {
		t.Fatalf("ValidateNote failed: %v", err)
	}
	if !result.NeedsMigration {
		t.Error("Old note should need migration")
	}
	if result.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %q, want '1.0.0'", result.CurrentVersion)
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("LatestVersion = %q, want '2.0.0'", result.LatestVersion)
	}
}

func TestInferFieldType(t *testing.T) {
	tests := []struct {
		value    string
		expected string
	}{
		{"{{date}}", "date"},
		{"date placeholder", "date"},
		{"[[Project]]", "tags"},
		{"#tag1 #tag2", "tags"},
		{"true", "boolean"},
		{"false", "boolean"},
		{"some text", "string"},
		{"2025-12-20", "string"}, // Actual dates are strings, we detect placeholders
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := inferFieldType(tt.value)
			if result != tt.expected {
				t.Errorf("inferFieldType(%q) = %q, want %q", tt.value, result, tt.expected)
			}
		})
	}
}
