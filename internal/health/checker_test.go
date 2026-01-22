package health

import (
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
