package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIssueTypeConstants(t *testing.T) {
	// Verify issue type constants
	types := []IssueType{
		IssueTypeBrokenLink,
		IssueTypeMissingAsset,
		IssueTypeOrphanedFile,
		IssueTypeDuplicate,
		IssueTypeFormat,
		IssueTypeWrongLinkFormat,
		IssueTypeWrongFilename,
		IssueTypeWrongMediaFilename,
		IssueTypeWrongMediaLocation,
		IssueTypeMissingMetadata,
	}

	for _, it := range types {
		if it == "" {
			t.Error("IssueType should not be empty")
		}
	}
}

func TestSeverityConstants(t *testing.T) {
	if SeverityError != "error" {
		t.Errorf("SeverityError should be 'error', got %s", SeverityError)
	}
	if SeverityWarning != "warning" {
		t.Errorf("SeverityWarning should be 'warning', got %s", SeverityWarning)
	}
	if SeverityInfo != "info" {
		t.Errorf("SeverityInfo should be 'info', got %s", SeverityInfo)
	}
}

func TestIssueStruct(t *testing.T) {
	issue := Issue{
		Type:     IssueTypeBrokenLink,
		Severity: SeverityError,
		File:     "notes/test.md",
		Line:     42,
		Message:  "Link not found",
		Details:  "[[missing-note]]",
	}

	if issue.Type != IssueTypeBrokenLink {
		t.Errorf("Issue type mismatch")
	}
	if issue.Severity != SeverityError {
		t.Errorf("Severity mismatch")
	}
	if issue.Line != 42 {
		t.Errorf("Line number mismatch")
	}
}

func TestCheckStats(t *testing.T) {
	stats := CheckStats{
		FilesScanned:  100,
		LinksChecked:  500,
		AssetsChecked: 50,
		ErrorCount:    5,
		WarningCount:  10,
		InfoCount:     20,
	}

	if stats.FilesScanned != 100 {
		t.Errorf("FilesScanned mismatch")
	}

	total := stats.ErrorCount + stats.WarningCount + stats.InfoCount
	if total != 35 {
		t.Errorf("Total issue count should be 35, got %d", total)
	}
}

func TestCheckResult(t *testing.T) {
	result := CheckResult{
		BrainInfo: &BrainInfo{
			Type: BrainTypeFlip,
			Path: "/test/brain",
		},
		Issues: []Issue{
			{Type: IssueTypeBrokenLink, Severity: SeverityError},
			{Type: IssueTypeOrphanedFile, Severity: SeverityWarning},
		},
		Stats: CheckStats{
			FilesScanned: 10,
			ErrorCount:   1,
			WarningCount: 1,
		},
	}

	if len(result.Issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(result.Issues))
	}
	if result.BrainInfo.Type != BrainTypeFlip {
		t.Errorf("BrainInfo type mismatch")
	}
}

func TestCheckMissingMetadataJournalBrain(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "journals"), 0755); err != nil {
		t.Fatal(err)
	}

	journalFile := "journals/2025-01-01.md"
	content := "---\ntitle: 2025-01-01\ncreated: 2025-01-01\n---\n\n# Journal\n"
	if err := os.WriteFile(filepath.Join(tempDir, journalFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	checker := &Checker{brainPath: tempDir, brainType: BrainTypeFlip}
	issues := checker.checkMissingMetadata(journalFile)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	if issues[0].Type != IssueTypeMissingMetadata {
		t.Fatalf("Expected issue type %s, got %s", IssueTypeMissingMetadata, issues[0].Type)
	}

	// Should report all missing fields: brain, date, day, type
	msg := issues[0].Message
	for _, field := range []string{"brain", "date", "day", "type"} {
		if !strings.Contains(msg, field) {
			t.Errorf("Expected missing field %q in message, got: %s", field, msg)
		}
	}
}

func TestCheckJournalMetadataAllFieldsPresent(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "journal"), 0755); err != nil {
		t.Fatal(err)
	}

	journalFile := "journal/2025-01-01.md"
	content := "---\ndate: 2025-01-01\nday: Wednesday\nbrain: testbrain\ntype: journal\n---\n\n# 2025-01-01\n"
	if err := os.WriteFile(filepath.Join(tempDir, journalFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	checker := &Checker{brainPath: tempDir, brainType: BrainTypeFlip}
	issues := checker.checkMissingMetadata(journalFile)

	if len(issues) != 0 {
		t.Fatalf("Expected 0 issues for complete journal metadata, got %d: %v", len(issues), issues)
	}
}

func TestCheckJournalMetadataNoFrontmatter(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "journal"), 0755); err != nil {
		t.Fatal(err)
	}

	journalFile := "journal/2025-01-01.md"
	content := "# 2025-01-01\n\nSome content\n"
	if err := os.WriteFile(filepath.Join(tempDir, journalFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	checker := &Checker{brainPath: tempDir, brainType: BrainTypeFlip}
	issues := checker.checkMissingMetadata(journalFile)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue for missing frontmatter, got %d", len(issues))
	}

	if issues[0].Severity != SeverityWarning {
		t.Errorf("Expected severity warning for missing frontmatter, got %s", issues[0].Severity)
	}
}

func TestCheckNoteMetadataMissingBrainFlip(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "notes"), 0755); err != nil {
		t.Fatal(err)
	}

	noteFile := "notes/test-note.md"
	content := "---\ntitle: Test Note\ncreated: 2025-01-01\n---\n\n# Test Note\n"
	if err := os.WriteFile(filepath.Join(tempDir, noteFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	checker := &Checker{brainPath: tempDir, brainType: BrainTypeFlip}
	issues := checker.checkMissingMetadata(noteFile)

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue for missing brain field, got %d", len(issues))
	}

	if !strings.Contains(issues[0].Message, "brain") {
		t.Errorf("Expected brain in missing fields message, got: %s", issues[0].Message)
	}
}

// ============================================================================
// LOGSEQ ARTIFACT DETECTION TESTS
// ============================================================================

func TestCheckLogseqArtifacts(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "notes"), 0755)

	tests := []struct {
		name       string
		content    string
		wantIssues int
	}{
		{
			name: "Collapsed property in body",
			content: `---
title: Test
---

Some text
- collapsed:: true
More text
`,
			wantIssues: 1,
		},
		{
			name: "Video embed",
			content: `---
title: Test
---

{{video https://www.youtube.com/watch?v=abc}}
`,
			wantIssues: 1,
		},
		{
			name: "Query block",
			content: `---
title: Test
---

{{query (and (task TODO))}}
`,
			wantIssues: 1,
		},
		{
			name: "Task marker",
			content: `---
title: Test
---

- TODO Buy groceries
- DONE Write tests
`,
			wantIssues: 1,
		},
		{
			name: "Clean file - no issues",
			content: `---
title: Test
---

- [ ] Buy groceries
- [x] Write tests
Regular content
`,
			wantIssues: 0,
		},
		{
			name: "Multiple artifact types",
			content: `---
title: Test
---

- collapsed:: true
- TODO Buy groceries
{{video https://example.com/video}}
`,
			wantIssues: 3,
		},
	}

	checker := &Checker{brainPath: tempDir, brainType: BrainTypeFlip}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noteFile := "notes/" + strings.ReplaceAll(tt.name, " ", "-") + ".md"
			os.WriteFile(filepath.Join(tempDir, noteFile), []byte(tt.content), 0644)

			issues := checker.checkLogseqArtifacts(noteFile)
			if len(issues) != tt.wantIssues {
				t.Errorf("checkLogseqArtifacts(%s): got %d issues, want %d.\nIssues: %+v",
					tt.name, len(issues), tt.wantIssues, issues)
			}
		})
	}
}

func TestIssueTypeLogseqArtifactConstant(t *testing.T) {
	if IssueTypeLogseqArtifact != "logseq-artifact" {
		t.Errorf("IssueTypeLogseqArtifact = %q, want %q", IssueTypeLogseqArtifact, "logseq-artifact")
	}
}
