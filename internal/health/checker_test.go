package health

import (
	"os"
	"path/filepath"
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

	if issues[0].File != journalFile {
		t.Fatalf("Expected issue file %s, got %s", journalFile, issues[0].File)
	}
}
