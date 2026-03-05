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
		name        string
		input       string
		contains    []string
		notContains []string
	}{
		{
			name:        "CRLF to LF",
			input:       "line1\r\nline2\r\n",
			notContains: []string{"\r"},
		},
		{
			name:        "Trailing whitespace",
			input:       "line1   \nline2\t\n",
			notContains: []string{"   \n", "\t\n"},
		},
		{
			name:     "Ends with newline",
			input:    "content",
			contains: []string{"content\n"},
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

func TestRepairMissingMetadataLogseq(t *testing.T) {
	// Create temp Logseq brain
	tempDir := t.TempDir()

	// Create logseq directory structure (makes it a Logseq brain)
	pagesDir := filepath.Join(tempDir, "pages")
	journalsDir := filepath.Join(tempDir, "journals")
	logseqDir := filepath.Join(tempDir, "logseq")
	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(journalsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logseqDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test page without metadata (Logseq outline format)
	testFile := "pages/my-test-page.md"
	content := "- This is a test page\n  - With some content\n"
	if err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create repairer
	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	// Verify brain type
	if repairer.brainType != BrainTypeLogseq {
		t.Fatalf("Expected Logseq brain type, got %s", repairer.brainType)
	}

	// Create issue for missing metadata
	issue := Issue{
		Type:     IssueTypeMissingMetadata,
		Severity: SeverityInfo,
		File:     testFile,
		Message:  "Missing Logseq properties: title::, created-at::",
	}

	// Repair the issue
	results := repairer.RepairIssues([]Issue{issue})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("Repair failed: %s", results[0].Message)
	}

	// Read the repaired file
	repairedContent, err := os.ReadFile(filepath.Join(tempDir, testFile))
	if err != nil {
		t.Fatal(err)
	}

	// Verify metadata was added
	repairedStr := string(repairedContent)
	if !strings.Contains(repairedStr, "title:: My Test Page") {
		t.Errorf("Expected title property, got: %s", repairedStr)
	}
	if !strings.Contains(repairedStr, "created-at::") {
		t.Errorf("Expected created-at property, got: %s", repairedStr)
	}
	// Original content should still be present
	if !strings.Contains(repairedStr, "This is a test page") {
		t.Errorf("Original content should be preserved, got: %s", repairedStr)
	}
}

func TestRepairMissingMetadataYAML(t *testing.T) {
	// Create temp Flip brain
	tempDir := t.TempDir()

	// Create .flip.yaml to make it a Flip brain
	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create notes directory
	notesDir := filepath.Join(tempDir, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test note without frontmatter
	testFile := "notes/my-test-note.md"
	content := "# My Test Note\n\nThis is some content.\n"
	if err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create repairer
	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	// Create issue for missing metadata
	issue := Issue{
		Type:     IssueTypeMissingMetadata,
		Severity: SeverityInfo,
		File:     testFile,
		Message:  "Missing YAML frontmatter",
	}

	// Repair the issue
	results := repairer.RepairIssues([]Issue{issue})

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Errorf("Repair failed: %s", results[0].Message)
	}

	// Read the repaired file
	repairedContent, err := os.ReadFile(filepath.Join(tempDir, testFile))
	if err != nil {
		t.Fatal(err)
	}

	// Verify frontmatter was added
	repairedStr := string(repairedContent)
	if !strings.HasPrefix(repairedStr, "---") {
		t.Errorf("Expected YAML frontmatter, got: %s", repairedStr)
	}
	if !strings.Contains(repairedStr, "title: My Test Note") {
		t.Errorf("Expected title in frontmatter, got: %s", repairedStr)
	}
	if !strings.Contains(repairedStr, "created:") {
		t.Errorf("Expected created in frontmatter, got: %s", repairedStr)
	}
	// Original content should still be present
	if !strings.Contains(repairedStr, "This is some content.") {
		t.Errorf("Original content should be preserved, got: %s", repairedStr)
	}
}

func TestRepairMissingMetadataJournalYAMLBackfillsBrain(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "journals"), 0755); err != nil {
		t.Fatal(err)
	}

	testFile := "journals/2025-01-01.md"
	content := "---\ntitle: 2025-01-01\n---\n\nJournal content\n"
	if err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	issue := Issue{
		Type:     IssueTypeMissingMetadata,
		Severity: SeverityInfo,
		File:     testFile,
		Message:  "Journal frontmatter missing fields: date, day, brain, type",
	}

	results := repairer.RepairIssues([]Issue{issue})
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("Repair failed: %+v", results)
	}

	repairedContent, err := os.ReadFile(filepath.Join(tempDir, testFile))
	if err != nil {
		t.Fatal(err)
	}

	repairedStr := string(repairedContent)
	// Should derive date from filename
	if !strings.Contains(repairedStr, "date: 2025-01-01") {
		t.Errorf("Expected date: 2025-01-01 in repaired journal, got: %s", repairedStr)
	}
	// Should derive weekday from date
	if !strings.Contains(repairedStr, "day: Wednesday") {
		t.Errorf("Expected day: Wednesday in repaired journal, got: %s", repairedStr)
	}
	// Should add brain name (from directory basename)
	if !strings.Contains(repairedStr, "brain: ") {
		t.Errorf("Expected brain field in repaired journal, got: %s", repairedStr)
	}
	// Should add type
	if !strings.Contains(repairedStr, "type: journal") {
		t.Errorf("Expected type: journal in repaired journal, got: %s", repairedStr)
	}
}

func TestRepairJournalMetadataNoFrontmatter(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "journal"), 0755); err != nil {
		t.Fatal(err)
	}

	testFile := "journal/2025-06-15.md"
	content := "# 2025-06-15\n\nSome notes\n"
	if err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	issue := Issue{
		Type:     IssueTypeMissingMetadata,
		Severity: SeverityWarning,
		File:     testFile,
		Message:  "Journal missing YAML frontmatter",
	}

	results := repairer.RepairIssues([]Issue{issue})
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("Repair failed: %+v", results)
	}

	repairedContent, err := os.ReadFile(filepath.Join(tempDir, testFile))
	if err != nil {
		t.Fatal(err)
	}

	repairedStr := string(repairedContent)

	// Should create complete frontmatter from scratch
	if !strings.HasPrefix(repairedStr, "---\n") {
		t.Errorf("Expected frontmatter to start with ---, got: %s", repairedStr[:20])
	}
	if !strings.Contains(repairedStr, "date: 2025-06-15") {
		t.Errorf("Expected date from filename, got: %s", repairedStr)
	}
	if !strings.Contains(repairedStr, "day: Sunday") {
		t.Errorf("Expected day: Sunday (2025-06-15), got: %s", repairedStr)
	}
	if !strings.Contains(repairedStr, "type: journal") {
		t.Errorf("Expected type: journal, got: %s", repairedStr)
	}
	// Original content should be preserved
	if !strings.Contains(repairedStr, "# 2025-06-15") {
		t.Errorf("Expected original content preserved, got: %s", repairedStr)
	}
}

func TestRepairNoteMetadataAddsBrainForFlip(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(tempDir, "notes"), 0755); err != nil {
		t.Fatal(err)
	}

	testFile := "notes/test-note.md"
	content := "---\ntitle: Test Note\ncreated: 2025-01-01\n---\n\n# Test Note\n"
	if err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	issue := Issue{
		Type:     IssueTypeMissingMetadata,
		Severity: SeverityInfo,
		File:     testFile,
		Message:  "Frontmatter missing fields: brain",
	}

	results := repairer.RepairIssues([]Issue{issue})
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("Repair failed: %+v", results)
	}

	repairedContent, err := os.ReadFile(filepath.Join(tempDir, testFile))
	if err != nil {
		t.Fatal(err)
	}

	repairedStr := string(repairedContent)
	if !strings.Contains(repairedStr, "brain: ") {
		t.Errorf("Expected brain field in repaired note, got: %s", repairedStr)
	}
	// Should still have existing fields
	if !strings.Contains(repairedStr, "title: Test Note") {
		t.Errorf("Expected title preserved, got: %s", repairedStr)
	}
}

func TestRepairMissingMetadataJournalLogseqBackfillsBrain(t *testing.T) {
	tempDir := t.TempDir()

	pagesDir := filepath.Join(tempDir, "pages")
	journalsDir := filepath.Join(tempDir, "journals")
	logseqDir := filepath.Join(tempDir, "logseq")
	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(journalsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logseqDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := "journals/2025_01_01.md"
	content := "title:: 2025-01-01\ncreated-at:: 1735689600000\n\n- journal content\n"
	if err := os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	issue := Issue{
		Type:     IssueTypeMissingMetadata,
		Severity: SeverityInfo,
		File:     testFile,
		Message:  "Journal missing metadata field: brain::",
	}

	results := repairer.RepairIssues([]Issue{issue})
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("Repair failed: %+v", results)
	}

	repairedContent, err := os.ReadFile(filepath.Join(tempDir, testFile))
	if err != nil {
		t.Fatal(err)
	}

	repairedStr := string(repairedContent)
	if !strings.Contains(repairedStr, "brain:: ") {
		t.Fatalf("Expected brain:: property in repaired Logseq journal, got: %s", repairedStr)
	}
}

func TestMarkBrokenLinkIdempotent(t *testing.T) {
	// markBrokenLink should not double-wrap already-marked links
	line := `Some text with ~~[[old-target]]~~ <!-- BROKEN --> here`
	result := markBrokenLink(line, "old-target")
	if result != line {
		t.Errorf("markBrokenLink should be idempotent for already-marked links\ngot:      %s\nexpected: %s", result, line)
	}
}

func TestMarkBrokenLinkWikilink(t *testing.T) {
	line := `Check [[my-note]] for details`
	result := markBrokenLink(line, "my-note")
	expected := `Check ~~[[my-note]]~~ <!-- BROKEN --> for details`
	if result != expected {
		t.Errorf("markBrokenLink wikilink\ngot:      %s\nexpected: %s", result, expected)
	}
}

func TestMarkBrokenLinkMarkdownLink(t *testing.T) {
	line := `Check [my note](my-note.md) for details`
	result := markBrokenLink(line, "my-note")
	expected := `Check ~~[my note](my-note.md)~~ <!-- BROKEN --> for details`
	if result != expected {
		t.Errorf("markBrokenLink markdown\ngot:      %s\nexpected: %s", result, expected)
	}
}

func TestResolveTargetExactMatch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".flip-brain.yaml"), 0755)
	os.MkdirAll(filepath.Join(dir, "notes"), 0755)
	os.WriteFile(filepath.Join(dir, "notes", "my-great-note.md"), []byte("# My Great Note\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}
	result := r.resolveTarget(dir, "My-Great-Note")
	if result != "my-great-note" {
		t.Errorf("resolveTarget exact match: expected 'my-great-note', got '%s'", result)
	}
}

func TestResolveTargetJournalDate(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "journal"), 0755)
	os.WriteFile(filepath.Join(dir, "journal", "2024-05-13.md"), []byte("# 2024-05-13\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}

	// Logseq-style underscore date should resolve to flip-style dash date
	result := r.resolveTarget(dir, "2024_05_13")
	if result != "2024-05-13" {
		t.Errorf("resolveTarget journal date: expected '2024-05-13', got '%s'", result)
	}
}

func TestResolveTargetFuzzy(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "notes"), 0755)
	os.WriteFile(filepath.Join(dir, "notes", "project-alpha.md"), []byte("# Project Alpha\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}
	result := r.resolveTarget(dir, "project-alfa")
	if result != "project-alpha" {
		t.Errorf("resolveTarget fuzzy: expected 'project-alpha', got '%s'", result)
	}
}

func TestResolveTargetNoMatch(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "notes"), 0755)
	os.WriteFile(filepath.Join(dir, "notes", "something.md"), []byte("hi\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}
	result := r.resolveTarget(dir, "completely-unrelated-name")
	if result != "" {
		t.Errorf("resolveTarget should return empty for no match, got '%s'", result)
	}
}

func TestReplaceLinkTargetWikilink(t *testing.T) {
	line := `See [[old-name]] for more`
	result := replaceLinkTarget(line, "old-name", "new-name")
	expected := `See [[new-name]] for more`
	if result != expected {
		t.Errorf("replaceLinkTarget wikilink\ngot:      %s\nexpected: %s", result, expected)
	}
}

func TestReplaceLinkTargetMarkdown(t *testing.T) {
	line := `See [old name](old-name.md) for more`
	result := replaceLinkTarget(line, "old-name", "new-name")
	if !strings.Contains(result, "new-name.md") {
		t.Errorf("replaceLinkTarget markdown should update target\ngot: %s", result)
	}
}
