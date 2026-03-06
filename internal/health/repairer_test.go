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
	expected := filepath.Join("notes", "my-great-note.md")
	if result != expected {
		t.Errorf("resolveTarget exact match: expected %q, got %q", expected, result)
	}
}

func TestResolveTargetJournalDate(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "journal"), 0755)
	os.WriteFile(filepath.Join(dir, "journal", "2024-05-13.md"), []byte("# 2024-05-13\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}

	// Logseq-style underscore date should resolve to flip-style dash date
	result := r.resolveTarget(dir, "2024_05_13")
	expected := filepath.Join("journal", "2024-05-13.md")
	if result != expected {
		t.Errorf("resolveTarget journal date: expected %q, got %q", expected, result)
	}
}

func TestResolveTargetFuzzy(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "notes"), 0755)
	os.WriteFile(filepath.Join(dir, "notes", "project-alpha.md"), []byte("# Project Alpha\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}
	result := r.resolveTarget(dir, "project-alfa")
	expected := filepath.Join("notes", "project-alpha.md")
	if result != expected {
		t.Errorf("resolveTarget fuzzy: expected %q, got %q", expected, result)
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

func TestResolveTargetPrioritizesBrainDirs(t *testing.T) {
	dir := t.TempDir()
	// Create the same file in notes/ and at root — should prefer notes/
	os.MkdirAll(filepath.Join(dir, "notes"), 0755)
	os.MkdirAll(filepath.Join(dir, "meetings"), 0755)
	os.WriteFile(filepath.Join(dir, "notes", "my-note.md"), []byte("# Note\n"), 0644)
	os.WriteFile(filepath.Join(dir, "my-note.md"), []byte("# Root note\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}
	result := r.resolveTarget(dir, "my-note")
	expected := filepath.Join("notes", "my-note.md")
	if result != expected {
		t.Errorf("resolveTarget should prefer notes/ dir: expected %q, got %q", expected, result)
	}
}

func TestResolveTargetSlugNormalized(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "notes"), 0755)
	os.WriteFile(filepath.Join(dir, "notes", "visual-studio-code.md"), []byte("# VS Code\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}
	// Capitalized with hyphens should match slug
	result := r.resolveTarget(dir, "Visual-Studio-Code")
	expected := filepath.Join("notes", "visual-studio-code.md")
	if result != expected {
		t.Errorf("resolveTarget slug: expected %q, got %q", expected, result)
	}
}

func TestResolveTargetOrphanedDir(t *testing.T) {
	dir := t.TempDir()
	// File exists only in .orphaned/notes/
	os.MkdirAll(filepath.Join(dir, ".orphaned", "notes"), 0755)
	os.WriteFile(filepath.Join(dir, ".orphaned", "notes", "flip.md"), []byte("# Flip\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}
	result := r.resolveTarget(dir, "flip")
	expected := filepath.Join(".orphaned", "notes", "flip.md")
	if result != expected {
		t.Errorf("resolveTarget .orphaned: expected %q, got %q", expected, result)
	}
}

func TestResolveTargetPrefersNonOrphaned(t *testing.T) {
	dir := t.TempDir()
	// File exists in both notes/ and .orphaned/notes/
	os.MkdirAll(filepath.Join(dir, "notes"), 0755)
	os.MkdirAll(filepath.Join(dir, ".orphaned", "notes"), 0755)
	os.WriteFile(filepath.Join(dir, "notes", "flip.md"), []byte("# Flip\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".orphaned", "notes", "flip.md"), []byte("# Flip orphaned\n"), 0644)

	r := &Repairer{brainPath: dir, brainType: BrainTypeFlip}
	result := r.resolveTarget(dir, "flip")
	expected := filepath.Join("notes", "flip.md")
	if result != expected {
		t.Errorf("resolveTarget should prefer non-orphaned: expected %q, got %q", expected, result)
	}
}

func TestComputeRelativeLink(t *testing.T) {
	tests := []struct {
		name       string
		sourceFile string
		targetFile string
		expected   string
	}{
		{
			name:       "journal to notes",
			sourceFile: "journal/2025-11-05.md",
			targetFile: "notes/flip.md",
			expected:   "../notes/flip.md",
		},
		{
			name:       "same directory",
			sourceFile: "notes/a.md",
			targetFile: "notes/b.md",
			expected:   "b.md",
		},
		{
			name:       "journal to meetings",
			sourceFile: "journal/2025-11-19.md",
			targetFile: "meetings/2025-11-19-meeting-standup.md",
			expected:   "../meetings/2025-11-19-meeting-standup.md",
		},
		{
			name:       "root to notes",
			sourceFile: "README.md",
			targetFile: "notes/something.md",
			expected:   "notes/something.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeRelativeLink(tt.sourceFile, tt.targetFile)
			if result != tt.expected {
				t.Errorf("computeRelativeLink(%q, %q) = %q, want %q",
					tt.sourceFile, tt.targetFile, result, tt.expected)
			}
		})
	}
}

func TestExtractLinkTarget(t *testing.T) {
	tests := []struct {
		message  string
		expected string
	}{
		{"Broken link: flip.md", "flip.md"},
		{"Broken link: github-copilot.md", "github-copilot.md"},
		{"Broken wikilink: [[my-note]]", "my-note"},
		{"Link to 'old-note' not found", "old-note"},
		{"Something unrecognized", ""},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := extractLinkTarget(tt.message)
			if result != tt.expected {
				t.Errorf("extractLinkTarget(%q) = %q, want %q", tt.message, result, tt.expected)
			}
		})
	}
}

func TestRepairBrokenLinkSmartResolution(t *testing.T) {
	// Test the full repair flow: journal file has a link to flip.md,
	// which actually exists in notes/flip.md
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "journal"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "notes"), 0755)

	// Create the target file
	os.WriteFile(filepath.Join(tempDir, "notes", "flip.md"), []byte("---\ntitle: Flip\n---\n# Flip\n"), 0644)

	// Create journal file with broken link
	content := "---\ntitle: 2025-11-05\n---\n\n# 2025-11-05\n\n- [flip](flip.md)\n"
	os.WriteFile(filepath.Join(tempDir, "journal", "2025-11-05.md"), []byte(content), 0644)

	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	issue := Issue{
		Type:     IssueTypeBrokenLink,
		Severity: SeverityError,
		File:     "journal/2025-11-05.md",
		Line:     7,
		Message:  "Broken link: flip.md",
	}

	results := repairer.RepairIssues([]Issue{issue})
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("Repair failed: %+v", results)
	}

	newContent, _ := os.ReadFile(filepath.Join(tempDir, "journal", "2025-11-05.md"))
	s := string(newContent)

	// Should have been resolved to ../notes/flip.md
	if !strings.Contains(s, "../notes/flip.md") {
		t.Errorf("Expected link to be resolved to ../notes/flip.md, got:\n%s", s)
	}

	// Should NOT contain BROKEN marker
	if strings.Contains(s, "BROKEN") {
		t.Errorf("Resolved link should not have BROKEN marker, got:\n%s", s)
	}
}

func TestRepairBrokenLinkIdempotent(t *testing.T) {
	// Running repair twice on same file shouldn't nest BROKEN comments
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "notes"), 0755)

	// File with an actually broken link (no resolution possible)
	content := "---\ntitle: Test\n---\n\n- [gone](gone-forever.md)\n"
	os.WriteFile(filepath.Join(tempDir, "notes", "test.md"), []byte(content), 0644)

	repairer, _ := NewRepairer(tempDir, false)

	issue := Issue{
		Type:     IssueTypeBrokenLink,
		Severity: SeverityError,
		File:     "notes/test.md",
		Line:     5,
		Message:  "Broken link: gone-forever.md",
	}

	// First repair
	repairer.RepairIssues([]Issue{issue})

	content1, _ := os.ReadFile(filepath.Join(tempDir, "notes", "test.md"))

	// Second repair on same line
	repairer.RepairIssues([]Issue{issue})

	content2, _ := os.ReadFile(filepath.Join(tempDir, "notes", "test.md"))

	// Content should be identical after second repair (idempotent)
	if string(content1) != string(content2) {
		t.Errorf("Repair is not idempotent.\nAfter 1st:\n%s\nAfter 2nd:\n%s",
			string(content1), string(content2))
	}

	// Should contain only ONE BROKEN marker
	brokenCount := strings.Count(string(content2), "BROKEN")
	if brokenCount > 1 {
		t.Errorf("Expected at most 1 BROKEN marker, got %d in:\n%s", brokenCount, string(content2))
	}
}

func TestRepairBrokenLinkUncomment(t *testing.T) {
	// When a previously commented-out broken link can now be resolved,
	// it should be un-commented and fixed
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "journal"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "notes"), 0755)

	// Journal file with a previously commented-out broken link
	content := "---\ntitle: 2025-11-05\n---\n\n<!-- BROKEN LINK: - [flip](flip.md) -->\n"
	os.WriteFile(filepath.Join(tempDir, "journal", "2025-11-05.md"), []byte(content), 0644)

	// The target file now exists in notes/
	os.WriteFile(filepath.Join(tempDir, "notes", "flip.md"), []byte("# Flip\n"), 0644)

	repairer, _ := NewRepairer(tempDir, false)

	issue := Issue{
		Type:     IssueTypeBrokenLink,
		Severity: SeverityError,
		File:     "journal/2025-11-05.md",
		Line:     5,
		Message:  "Broken link: flip.md",
	}

	repairer.RepairIssues([]Issue{issue})

	result, _ := os.ReadFile(filepath.Join(tempDir, "journal", "2025-11-05.md"))
	resultStr := string(result)

	// Should NOT contain BROKEN comment anymore
	if strings.Contains(resultStr, "<!-- BROKEN") {
		t.Errorf("Should have un-commented the line, got:\n%s", resultStr)
	}

	// Should contain the fixed link pointing to ../notes/flip.md
	if !strings.Contains(resultStr, "../notes/flip.md") {
		t.Errorf("Should contain resolved link ../notes/flip.md, got:\n%s", resultStr)
	}
}

func TestRepairBrokenLinkUncommentNested(t *testing.T) {
	// Deeply nested BROKEN comments should be unwrapped and fixed
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "journal"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "notes"), 0755)

	// Deeply nested BROKEN comment (from repeated prior repairs)
	content := "---\ntitle: Test\n---\n\n<!-- BROKEN LINK: <!-- BROKEN LINK: - [flip](flip.md) --> -->\n"
	os.WriteFile(filepath.Join(tempDir, "journal", "test.md"), []byte(content), 0644)
	os.WriteFile(filepath.Join(tempDir, "notes", "flip.md"), []byte("# Flip\n"), 0644)

	repairer, _ := NewRepairer(tempDir, false)

	issue := Issue{
		Type:     IssueTypeBrokenLink,
		Severity: SeverityError,
		File:     "journal/test.md",
		Line:     5,
		Message:  "Broken link: flip.md",
	}

	repairer.RepairIssues([]Issue{issue})

	result, _ := os.ReadFile(filepath.Join(tempDir, "journal", "test.md"))
	resultStr := string(result)

	if strings.Contains(resultStr, "<!-- BROKEN") {
		t.Errorf("Should have un-commented nested line, got:\n%s", resultStr)
	}
	if !strings.Contains(resultStr, "../notes/flip.md") {
		t.Errorf("Should contain resolved link, got:\n%s", resultStr)
	}
}

func TestRepairBrokenLinkUncommentOrphaned(t *testing.T) {
	// When target is in .orphaned/notes/, should resolve and un-comment
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "journal"), 0755)
	os.MkdirAll(filepath.Join(tempDir, ".orphaned", "notes"), 0755)

	content := "---\ntitle: 2025-11-25\n---\n\n<!-- BROKEN LINK: - [flip](flip.md) -->\n"
	os.WriteFile(filepath.Join(tempDir, "journal", "2025-11-25.md"), []byte(content), 0644)
	os.WriteFile(filepath.Join(tempDir, ".orphaned", "notes", "flip.md"), []byte("# Flip\n"), 0644)

	repairer, _ := NewRepairer(tempDir, false)

	issue := Issue{
		Type:     IssueTypeBrokenLink,
		Severity: SeverityError,
		File:     "journal/2025-11-25.md",
		Line:     5,
		Message:  "Broken link: flip.md",
	}

	results := repairer.RepairIssues([]Issue{issue})
	t.Logf("Repair result: success=%v message=%s", results[0].Success, results[0].Message)

	result, _ := os.ReadFile(filepath.Join(tempDir, "journal", "2025-11-25.md"))
	resultStr := string(result)
	t.Logf("File content after repair:\n%s", resultStr)

	if strings.Contains(resultStr, "<!-- BROKEN") {
		t.Errorf("Should have un-commented the line, got:\n%s", resultStr)
	}
	if !strings.Contains(resultStr, ".orphaned/notes/flip.md") {
		t.Errorf("Should contain resolved link to .orphaned/notes/flip.md, got:\n%s", resultStr)
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

// ============================================================================
// LOGSEQ ARTIFACT REPAIR TESTS
// ============================================================================

func TestRepairLogseqArtifact(t *testing.T) {
	tempDir := t.TempDir()

	// Create .flip.yaml to make it a Flip brain
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "notes"), 0755)

	content := `---
title: Test Note
---

Some content
- collapsed:: true
- background-color:: red
- TODO Buy groceries
- DONE Write tests
{{video https://www.youtube.com/watch?v=abc123}}
- {{query (and (task TODO) (page "project"))}}
More content
`
	testFile := "notes/test-note.md"
	os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644)

	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to create repairer: %v", err)
	}

	issue := Issue{
		Type:     IssueTypeLogseqArtifact,
		Severity: SeverityWarning,
		File:     testFile,
		Message:  "Logseq artifact found",
	}

	results := repairer.RepairIssues([]Issue{issue})
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("Repair failed: %+v", results)
	}

	newContent, err := os.ReadFile(filepath.Join(tempDir, testFile))
	if err != nil {
		t.Fatal(err)
	}
	s := string(newContent)

	// collapsed and background-color should be removed
	if strings.Contains(s, "collapsed::") {
		t.Error("collapsed:: should be removed")
	}
	if strings.Contains(s, "background-color::") {
		t.Error("background-color:: should be removed")
	}

	// Task markers should be converted
	if strings.Contains(s, "TODO Buy") {
		t.Error("TODO should be converted to checkbox")
	}
	if !strings.Contains(s, "- [ ] Buy groceries") {
		t.Error("Expected '- [ ] Buy groceries' checkbox")
	}
	if !strings.Contains(s, "- [x] Write tests") {
		t.Error("Expected '- [x] Write tests' checkbox")
	}

	// Video embed should be converted
	if strings.Contains(s, "{{video") {
		t.Error("{{video}} should be converted")
	}
	if !strings.Contains(s, "[YouTube Video]") {
		t.Error("Expected YouTube Video markdown link")
	}

	// Query should be removed
	if strings.Contains(s, "{{query") {
		t.Error("{{query}} should be removed")
	}
}

func TestRepairLogseqArtifact_ExtractsMeaningfulProps(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "notes"), 0755)

	content := `---
title: Existing Title
---

- tags:: important, work
- author:: John Doe
More content
`
	testFile := "notes/props-note.md"
	os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644)

	repairer, err := NewRepairer(tempDir, false)
	if err != nil {
		t.Fatal(err)
	}

	issue := Issue{
		Type:     IssueTypeLogseqArtifact,
		Severity: SeverityWarning,
		File:     testFile,
		Message:  "Logseq artifact found",
	}

	repairer.RepairIssues([]Issue{issue})

	newContent, _ := os.ReadFile(filepath.Join(tempDir, testFile))
	s := string(newContent)

	// tags and author should be merged into frontmatter
	if strings.Contains(s, "- tags::") {
		t.Error("tags:: should be removed from body")
	}
	if !strings.Contains(s, "tags: important, work") {
		t.Error("tags should be in frontmatter")
	}
	if !strings.Contains(s, "author: John Doe") {
		t.Error("author should be in frontmatter")
	}
}

func TestRepairLogseqArtifact_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("version: 1\n"), 0644)
	os.MkdirAll(filepath.Join(tempDir, "notes"), 0755)

	content := `---
title: Test
---

- collapsed:: true
`
	testFile := "notes/dry-run.md"
	os.WriteFile(filepath.Join(tempDir, testFile), []byte(content), 0644)

	repairer, err := NewRepairer(tempDir, true) // dry-run mode
	if err != nil {
		t.Fatal(err)
	}

	issue := Issue{
		Type:     IssueTypeLogseqArtifact,
		Severity: SeverityWarning,
		File:     testFile,
		Message:  "Logseq artifact found",
	}

	results := repairer.RepairIssues([]Issue{issue})
	if len(results) != 1 || !results[0].Success {
		t.Fatalf("Dry-run should succeed: %+v", results)
	}
	if !strings.Contains(results[0].Message, "DRY-RUN") {
		t.Error("Expected DRY-RUN message")
	}

	// File should be unchanged
	newContent, _ := os.ReadFile(filepath.Join(tempDir, testFile))
	if string(newContent) != content {
		t.Error("File should not be modified in dry-run mode")
	}
}

func TestCleanLogseqTaskMarkers(t *testing.T) {
	r := &Repairer{}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "TODO to checkbox",
			input:    "- TODO Buy groceries",
			expected: "- [ ] Buy groceries",
		},
		{
			name:     "DONE to checked",
			input:    "- DONE Write tests",
			expected: "- [x] Write tests",
		},
		{
			name:     "CANCELLED to cancelled",
			input:    "- CANCELLED Old task",
			expected: "- [~] Old task",
		},
		{
			name:     "LATER to unchecked",
			input:    "- LATER Read book",
			expected: "- [ ] Read book",
		},
		{
			name:     "Indented marker",
			input:    "  - TODO Subtask",
			expected: "  - [ ] Subtask",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.cleanLogseqTaskMarkers(tt.input)
			if result != tt.expected {
				t.Errorf("cleanLogseqTaskMarkers(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCleanLogseqVideoEmbeds(t *testing.T) {
	r := &Repairer{}
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "YouTube embed",
			input:    "{{video https://www.youtube.com/watch?v=abc}}",
			expected: "[YouTube Video](https://www.youtube.com/watch?v=abc)",
		},
		{
			name:     "youtu.be embed",
			input:    "{{video https://youtu.be/abc}}",
			expected: "[YouTube Video](https://youtu.be/abc)",
		},
		{
			name:     "Non-youtube embed",
			input:    "{{video https://vimeo.com/12345}}",
			expected: "[Video](https://vimeo.com/12345)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.cleanLogseqVideoEmbeds(tt.input)
			if result != tt.expected {
				t.Errorf("cleanLogseqVideoEmbeds(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
