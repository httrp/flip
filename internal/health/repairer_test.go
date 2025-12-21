package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepairBrokenLink(t *testing.T) {
	// Create temp test brain
	tempDir := t.TempDir()
	
	// Create .flip.yaml to make it a Flip brain
	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file with a broken link
	testFile := "notes/test-note.md"
	testDir := filepath.Join(tempDir, "notes")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `# Test Note

This has a [[broken-link]] that doesn't exist.

And a [markdown link](missing.md) too.

Good content here.
`
	if err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create repairer
	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	// Create issue for the broken wikilink
	issue := Issue{
		Type:     IssueTypeBrokenLink,
		Severity: SeverityError,
		File:     testFile,
		Line:     3,
		Message:  "Link to 'broken-link' not found",
	}

	// Repair it
	results := repairer.RepairIssues([]Issue{issue})
	
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("Repair failed: %s", results[0].Message)
	}

	// Verify the file was modified
	newContent, err := os.ReadFile(filepath.Join(tempDir, testFile))
	if err != nil {
		t.Fatal(err)
	}

	// Should contain some indication of broken link
	if !strings.Contains(string(newContent), "BROKEN") && !strings.Contains(string(newContent), "~~") {
		t.Error("File should be modified to mark broken link")
	}
}

func TestRepairOrphanedFile(t *testing.T) {
	// Create temp test brain
	tempDir := t.TempDir()
	
	// Create .flip.yaml
	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an "orphaned" file
	orphanFile := "notes/orphan.md"
	testDir := filepath.Join(tempDir, "notes")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, orphanFile), []byte("# Orphan\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create repairer
	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	// Create issue
	issue := Issue{
		Type:     IssueTypeOrphanedFile,
		Severity: SeverityInfo,
		File:     orphanFile,
		Message:  "File is not linked from anywhere",
	}

	// Repair
	results := repairer.RepairIssues([]Issue{issue})

	if len(results) != 1 || !results[0].Success {
		t.Fatalf("Repair failed: %v", results)
	}

	// Original file should be gone
	if _, err := os.Stat(filepath.Join(tempDir, orphanFile)); !os.IsNotExist(err) {
		t.Error("Original file should be moved")
	}

	// Should be in .orphaned folder
	orphanedPath := filepath.Join(tempDir, ".orphaned", orphanFile)
	if _, err := os.Stat(orphanedPath); err != nil {
		t.Errorf("File should exist in .orphaned: %v", err)
	}
}

func TestRepairDryRun(t *testing.T) {
	// Create temp test brain
	tempDir := t.TempDir()
	
	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	testFile := "test.md"
	originalContent := "# Test\n\nContent [[broken]]\n"
	if err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create repairer in dry-run mode
	repairer, err := NewRepairer(tempDir, true)
	if err != nil {
		t.Fatal(err)
	}

	issue := Issue{
		Type: IssueTypeBrokenLink,
		File: testFile,
		Line: 3,
	}

	results := repairer.RepairIssues([]Issue{issue})

	if !results[0].Success {
		t.Error("Dry run should report success")
	}

	if !strings.Contains(results[0].Message, "DRY-RUN") {
		t.Error("Result should indicate dry run")
	}

	// File should NOT be modified
	content, _ := os.ReadFile(filepath.Join(tempDir, testFile))
	if string(content) != originalContent {
		t.Error("Dry run should not modify files")
	}
}

func TestRepairFormatIssue(t *testing.T) {
	tempDir := t.TempDir()
	
	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	testFile := "bad-format.md"
	// Content with formatting issues
	badContent := "# Title  \r\n\r\n\r\nContent\r\n\r\n\r\n\r\n\r\nMore content   "
	
	if err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatal(err)
	}

	issue := Issue{
		Type: IssueTypeFormat,
		File: testFile,
	}

	results := repairer.RepairIssues([]Issue{issue})

	if !results[0].Success {
		t.Errorf("Format repair failed: %s", results[0].Message)
	}

	// Check normalized content
	content, _ := os.ReadFile(filepath.Join(tempDir, testFile))
	contentStr := string(content)

	// Should end with single newline
	if !strings.HasSuffix(contentStr, "\n") || strings.HasSuffix(contentStr, "\n\n") {
		t.Error("Content should end with exactly one newline")
	}

	// Should not have CRLF
	if strings.Contains(contentStr, "\r") {
		t.Error("Content should not contain CR")
	}

	// Should not have trailing whitespace
	for _, line := range strings.Split(contentStr, "\n") {
		if strings.HasSuffix(line, " ") || strings.HasSuffix(line, "\t") {
			t.Error("Lines should not have trailing whitespace")
		}
	}
}

func TestCanRepair(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	repairer, _ := NewRepairer(tempDir, false)

	// Can repair
	if !repairer.CanRepair(IssueTypeBrokenLink) {
		t.Error("Should be able to repair broken links")
	}
	if !repairer.CanRepair(IssueTypeOrphanedFile) {
		t.Error("Should be able to repair orphaned files")
	}
	if !repairer.CanRepair(IssueTypeFormat) {
		t.Error("Should be able to repair format issues")
	}

	// Cannot repair (yet)
	if repairer.CanRepair(IssueTypeMissingAsset) {
		t.Error("Should not be able to repair missing assets")
	}
	if repairer.CanRepair(IssueTypeDuplicate) {
		t.Error("Should not be able to repair duplicates")
	}
}

func TestCalculateStats(t *testing.T) {
	results := []RepairResult{
		{Success: true, Message: "Repaired"},
		{Success: true, Message: "Repaired"},
		{Success: false, SkipReason: "No repair action available"},
		{Success: false, Message: "Failed to repair"},
		{Success: true, Message: "[DRY-RUN] Would repair"},
	}

	stats := CalculateStats(results)

	if stats.TotalIssues != 5 {
		t.Errorf("TotalIssues = %d, want 5", stats.TotalIssues)
	}

	// 3 successful (including dry-run)
	if stats.Repaired != 3 {
		t.Errorf("Repaired = %d, want 3", stats.Repaired)
	}

	if stats.NotRepairable != 1 {
		t.Errorf("NotRepairable = %d, want 1", stats.NotRepairable)
	}

	if stats.Failed != 1 {
		t.Errorf("Failed = %d, want 1", stats.Failed)
	}
}

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		notContains []string
	}{
		{
			name:     "CRLF to LF",
			input:    "line1\r\nline2\r\n",
			notContains: []string{"\r"},
		},
		{
			name:     "Trailing whitespace",
			input:    "line1   \nline2\t\n",
			notContains: []string{"   \n", "\t\n"},
		},
		{
			name:        "Ends with newline",
			input:       "content",
			contains:    []string{"content\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeContent(tt.input)
			
			for _, c := range tt.contains {
				if !strings.Contains(result, c) {
					t.Errorf("Result should contain %q", c)
				}
			}
			
			for _, nc := range tt.notContains {
				if strings.Contains(result, nc) {
					t.Errorf("Result should not contain %q", nc)
				}
			}
		})
	}
}
