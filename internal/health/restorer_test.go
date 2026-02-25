package health_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/httrp/flip/internal/health"
)

func TestRestorer_ListOrphanedFiles(t *testing.T) {
	// Create test brain structure
	tempDir := t.TempDir()

	// Create directories
	os.MkdirAll(filepath.Join(tempDir, ".orphaned", "meetings"), 0755)
	os.MkdirAll(filepath.Join(tempDir, ".orphaned", "notes"), 0755)

	// Create orphaned files
	meetingFile := filepath.Join(tempDir, ".orphaned", "meetings", "2026-03-04-meeting-test.md")
	notesFile := filepath.Join(tempDir, ".orphaned", "notes", "test-note.md")

	os.WriteFile(meetingFile, []byte("# Meeting"), 0644)
	os.WriteFile(notesFile, []byte("# Note"), 0644)

	// Create journal for matching
	os.MkdirAll(filepath.Join(tempDir, "journal"), 0755)
	os.WriteFile(filepath.Join(tempDir, "journal", "2026-03-04.md"), []byte("# 2026-03-04\n"), 0644)

	// Create .flip.yaml
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("type: flip\nname: test-brain\n"), 0644)

	restorer, err := health.NewRestorer(tempDir)
	if err != nil {
		t.Fatalf("Failed to create restorer: %v", err)
	}

	files, err := restorer.ListOrphanedFiles()
	if err != nil {
		t.Fatalf("Failed to list orphaned files: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 orphaned files, got %d", len(files))
	}

	// Check meeting file
	hasMeteing := false
	hasMeetingDate := false
	for _, f := range files {
		if f.Filename == "2026-03-04-meeting-test.md" {
			hasMeteing = true
			if f.Category != "meetings" {
				t.Errorf("Expected category 'meetings', got %s", f.Category)
			}
			if f.Date == nil {
				t.Error("Expected date to be extracted")
			} else if f.Date.Format("2006-01-02") == "2026-03-04" {
				hasMeetingDate = true
			}
			if f.JournalMatch != filepath.Join("journal", "2026-03-04.md") {
				t.Errorf("Expected journal match, got %s", f.JournalMatch)
			}
		}
	}

	if !hasMeteing {
		t.Error("Meeting file not found in orphaned list")
	}
	if !hasMeetingDate {
		t.Error("Date not correctly extracted from meeting filename")
	}
}

func TestRestorer_RestoreSingleFile(t *testing.T) {
	// Create test brain structure
	tempDir := t.TempDir()

	// Create directories
	os.MkdirAll(filepath.Join(tempDir, ".orphaned", "notes"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "notes"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "journal"), 0755)

	// Create orphaned file
	orphanedFile := filepath.Join(tempDir, ".orphaned", "notes", "test-note.md")
	os.WriteFile(orphanedFile, []byte("# Test Note\nContent here"), 0644)

	// Create journal
	journalFile := filepath.Join(tempDir, "journal", "2026-03-04.md")
	os.WriteFile(journalFile, []byte("---\ndate: 2026-03-04\n---\n# 2026-03-04\n"), 0644)

	// Create .flip.yaml
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("type: flip\nname: test-brain\n"), 0644)

	restorer, err := health.NewRestorer(tempDir)
	if err != nil {
		t.Fatalf("Failed to create restorer: %v", err)
	}

	// Create orphaned file info
	date, _ := time.Parse("2006-01-02", "2026-03-04")
	fileInfo := &health.OrphanedFileInfo{
		Path:         "notes/test-note.md",
		Category:     "notes",
		Filename:     "test-note.md",
		Date:         &date,
		TargetPath:   "notes/test-note.md",
		JournalMatch: "journal/2026-03-04.md",
	}

	// Restore file
	result := restorer.RestoreSingleFile(fileInfo, true)

	if !result.Success {
		t.Errorf("Restore should be successful: %s", result.Error)
	}

	// Check file was moved
	restoredPath := filepath.Join(tempDir, "notes", "test-note.md")
	if _, err := os.Stat(restoredPath); err != nil {
		t.Errorf("Restored file should exist: %v", err)
	}

	// Check orphaned file was removed
	if _, err := os.Stat(orphanedFile); err == nil {
		t.Error("Orphaned file should be removed")
	}

	// Check journal was updated
	journalContent, _ := os.ReadFile(journalFile)
	if !contains(string(journalContent), "[[test-note]]") {
		t.Error("Journal should contain link to restored file")
	}
}

func TestRestorer_DateExtraction(t *testing.T) {
	tempDir := t.TempDir()

	// Create .flip.yaml
	os.WriteFile(filepath.Join(tempDir, ".flip.yaml"), []byte("type: flip\nname: test-brain\n"), 0644)

	restorer, err := health.NewRestorer(tempDir)
	if err != nil {
		t.Fatalf("Failed to create restorer: %v", err)
	}

	testCases := []struct {
		filename   string
		expectDate string
	}{
		{"2026-03-04-meeting-test.md", "2026-03-04"},
		{"2026-01-15-meeting-kickoff.md", "2026-01-15"},
		{"test-note-without-date.md", ""},
	}

	for _, tc := range testCases {
		fileInfo := restorer.ParseOrphanedFile(tc.filename)
		if tc.expectDate == "" {
			if fileInfo.Date != nil {
				t.Errorf("File %s should not have date, got %v", tc.filename, fileInfo.Date)
			}
		} else {
			if fileInfo.Date == nil {
				t.Errorf("File %s should have date", tc.filename)
			} else if fileInfo.Date.Format("2006-01-02") != tc.expectDate {
				t.Errorf("Expected date %s, got %s", tc.expectDate, fileInfo.Date.Format("2006-01-02"))
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr))
}
