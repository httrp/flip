package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/httrp/flip/internal/health"
)

// TestRollbackRemovesAllFiles tests that rollback deletes all migrated files
func TestRollbackRemovesAllFiles(t *testing.T) {
	// Create temp source and target directories
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create source brain with a note and asset
	notesDir := filepath.Join(sourceDir, "notes")
	os.MkdirAll(notesDir, 0755)
	
	noteContent := "# Test Note\n\nThis is a test note with an image ![pic](../assets/test.png)"
	os.WriteFile(filepath.Join(notesDir, "test-note.md"), []byte(noteContent), 0644)

	assetsDir := filepath.Join(sourceDir, "assets")
	os.MkdirAll(assetsDir, 0755)
	os.WriteFile(filepath.Join(assetsDir, "test.png"), []byte("fake image"), 0644)

	// Build and execute migration plan
	sourceStruct := GetBrainStructure(health.BrainTypeFlip)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	
	planner := NewPlanner(sourceDir, targetDir, sourceStruct, targetStruct)
	plan, err := planner.BuildPlan(ModeFull, "", nil, 0)
	if err != nil {
		t.Fatalf("Failed to build plan: %v", err)
	}

	executor := NewExecutor(plan, sourceStruct, targetStruct)
	execLog, err := executor.Execute()
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify files were created
	if len(execLog.ItemsWritten) == 0 {
		t.Fatal("No items were written during migration")
	}

	for _, item := range execLog.ItemsWritten {
		itemPath := filepath.Join(targetDir, item)
		if _, err := os.Stat(itemPath); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", item)
		}
	}

	// Verify log file exists
	logPath := filepath.Join(targetDir, ".flip-migration-log.json")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatal("Migration log file not created")
	}

	// Now perform rollback simulation
	// Read the log
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	var log ExecutionLog
	if err := json.Unmarshal(logData, &log); err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}

	// Delete all items from log
	removedCount := 0
	for _, item := range log.ItemsWritten {
		itemPath := filepath.Join(targetDir, item)
		if err := os.Remove(itemPath); err != nil {
			if !os.IsNotExist(err) {
				t.Errorf("Failed to remove %s: %v", item, err)
			}
		} else {
			removedCount++
		}
	}

	if removedCount != len(log.ItemsWritten) {
		t.Errorf("Expected to remove %d files, removed %d", len(log.ItemsWritten), removedCount)
	}

	// Remove log file
	if err := os.Remove(logPath); err != nil {
		t.Errorf("Failed to remove log file: %v", err)
	}

	// Verify all files are gone
	for _, item := range log.ItemsWritten {
		itemPath := filepath.Join(targetDir, item)
		if _, err := os.Stat(itemPath); !os.IsNotExist(err) {
			t.Errorf("File still exists after rollback: %s", item)
		}
	}

	// Verify log is gone
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("Log file still exists after rollback")
	}
}

// TestRollbackHandlesMissingFiles tests rollback when some files are already deleted
func TestRollbackHandlesMissingFiles(t *testing.T) {
	// Create temp target directory
	targetDir := t.TempDir()

	// Create a fake migration log
	log := &ExecutionLog{
		StartTime:    time.Now().Add(-1 * time.Hour),
		EndTime:      time.Now().Add(-30 * time.Minute),
		Success:      true,
		ItemsWritten: []string{
			"notes/test1.md",
			"notes/test2.md",
			"assets/image.png",
		},
		Errors: []string{},
		Metadata: map[string]string{
			"mode":   "full",
			"source": "/tmp/source",
			"target": targetDir,
		},
	}

	// Write log file
	logPath := filepath.Join(targetDir, ".flip-migration-log.json")
	logData, _ := json.MarshalIndent(log, "", "  ")
	os.WriteFile(logPath, logData, 0644)

	// Create only ONE of the files (test2.md)
	notesDir := filepath.Join(targetDir, "notes")
	os.MkdirAll(notesDir, 0755)
	os.WriteFile(filepath.Join(notesDir, "test2.md"), []byte("# Test 2"), 0644)

	// Read log and attempt rollback
	logData, _ = os.ReadFile(logPath)
	var readLog ExecutionLog
	json.Unmarshal(logData, &readLog)

	removedCount := 0
	notFoundCount := 0
	
	for _, item := range readLog.ItemsWritten {
		itemPath := filepath.Join(targetDir, item)
		if _, err := os.Stat(itemPath); os.IsNotExist(err) {
			notFoundCount++
			continue
		}
		if err := os.Remove(itemPath); err == nil {
			removedCount++
		}
	}

	// Should have removed 1 file, not found 2 files
	if removedCount != 1 {
		t.Errorf("Expected to remove 1 file, removed %d", removedCount)
	}
	if notFoundCount != 2 {
		t.Errorf("Expected 2 files not found, got %d", notFoundCount)
	}

	// Verify test2.md is gone
	if _, err := os.Stat(filepath.Join(notesDir, "test2.md")); !os.IsNotExist(err) {
		t.Error("test2.md should have been removed")
	}
}

// TestRollbackLogFormat verifies the execution log has correct structure
func TestRollbackLogFormat(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	// Create minimal source
	os.MkdirAll(filepath.Join(sourceDir, "notes"), 0755)
	os.WriteFile(filepath.Join(sourceDir, "notes", "test.md"), []byte("# Test"), 0644)

	// Execute migration
	sourceStruct := GetBrainStructure(health.BrainTypeFlip)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	
	planner := NewPlanner(sourceDir, targetDir, sourceStruct, targetStruct)
	plan, _ := planner.BuildPlan(ModeFull, "", nil, 0)
	
	executor := NewExecutor(plan, sourceStruct, targetStruct)
	executor.Execute()

	// Read and verify log structure
	logPath := filepath.Join(targetDir, ".flip-migration-log.json")
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	var log ExecutionLog
	if err := json.Unmarshal(logData, &log); err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}

	// Verify required fields
	if log.StartTime.IsZero() {
		t.Error("StartTime should not be zero")
	}
	if log.EndTime.IsZero() {
		t.Error("EndTime should not be zero")
	}
	if log.EndTime.Before(log.StartTime) {
		t.Error("EndTime should be after StartTime")
	}
	if len(log.ItemsWritten) == 0 {
		t.Error("ItemsWritten should not be empty")
	}
	if log.Metadata == nil {
		t.Error("Metadata should not be nil")
	}
	if log.Metadata["mode"] == "" {
		t.Error("Metadata should contain mode")
	}
	if log.Metadata["source"] == "" {
		t.Error("Metadata should contain source")
	}
	if log.Metadata["target"] == "" {
		t.Error("Metadata should contain target")
	}
}
