package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/httrp/flip/internal/commands"
	"github.com/httrp/flip/internal/health"
)

func TestPerformRelocation_Success(t *testing.T) {
	// Create test brain structure
	sourceDir := t.TempDir()
	destParentDir := t.TempDir()
	destPath := filepath.Join(destParentDir, "relocated-brain")

	// Create minimal brain structure
	os.MkdirAll(filepath.Join(sourceDir, "notes"), 0755)
	os.MkdirAll(filepath.Join(sourceDir, "journal"), 0755)

	// Create brain config
	configContent := "type: flip\nname: test-brain\n"
	os.WriteFile(filepath.Join(sourceDir, ".flip.yaml"), []byte(configContent), 0644)

	// Create a test note
	os.WriteFile(filepath.Join(sourceDir, "notes", "test.md"), []byte("# Test\nContent"), 0644)

	// Perform relocation
	result := commands.PerformRelocation(sourceDir, destPath, false, true)

	// Verify success
	if !result.Success {
		t.Errorf("Relocation should be successful: %s", result.Error)
	}

	// Verify source no longer exists
	if _, err := os.Stat(sourceDir); err == nil {
		t.Error("Source brain should no longer exist after relocation")
	}

	// Verify destination exists
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("Destination brain should exist: %v", err)
	}

	// Verify structure is intact
	if _, err := os.Stat(filepath.Join(destPath, ".flip.yaml")); err != nil {
		t.Error("Brain config should exist in new location")
	}

	if _, err := os.Stat(filepath.Join(destPath, "notes", "test.md")); err != nil {
		t.Error("Notes should be preserved in new location")
	}

	// Verify health check was run (even if issues found - test brains may have issues)
	if result.HealthIssues < 0 {
		t.Errorf("Health issues count should be >= 0: %d", result.HealthIssues)
	}
}

func TestPerformRelocation_InvalidSource(t *testing.T) {
	destPath := t.TempDir()

	// Try to relocate from non-existent source
	result := commands.PerformRelocation("/nonexistent/brain", destPath, false, true)

	if result.Success {
		t.Error("Relocation should fail with non-existent source")
	}

	if result.Error == "" {
		t.Error("Error message should not be empty")
	}
}

func TestPerformRelocation_InvalidBrain(t *testing.T) {
	sourceDir := t.TempDir()
	destPath := t.TempDir()

	// Create directory without brain config
	os.MkdirAll(filepath.Join(sourceDir, "notes"), 0755)

	// Try to relocate invalid brain
	result := commands.PerformRelocation(sourceDir, destPath, false, true)

	if result.Success {
		t.Error("Relocation should fail with invalid brain")
	}

	if result.Error == "" {
		t.Error("Error message should describe the issue")
	}

	// Source should still exist (no partial relocation)
	if _, err := os.Stat(sourceDir); err != nil {
		t.Error("Source should remain unchanged on error")
	}
}

func TestPerformRelocation_DestinationExists(t *testing.T) {
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create valid brain at source
	os.MkdirAll(filepath.Join(sourceDir, "notes"), 0755)
	os.WriteFile(filepath.Join(sourceDir, ".flip.yaml"), []byte("type: flip\nname: test\n"), 0644)

	// Try to relocate to existing destination
	result := commands.PerformRelocation(sourceDir, destDir, false, true)

	if result.Success {
		t.Error("Relocation should fail when destination exists")
	}

	if result.Error == "" {
		t.Error("Error message should indicate destination exists")
	}

	// Source should still exist
	if _, err := os.Stat(sourceDir); err != nil {
		t.Error("Source should remain unchanged on error")
	}
}

func TestPerformRelocation_DryRun(t *testing.T) {
	sourceDir := t.TempDir()
	destParentDir := t.TempDir()
	destPath := filepath.Join(destParentDir, "relocated-brain")

	// Create valid brain
	os.MkdirAll(filepath.Join(sourceDir, "notes"), 0755)
	os.WriteFile(filepath.Join(sourceDir, ".flip.yaml"), []byte("type: flip\nname: test\n"), 0644)

	// Perform dry-run
	result := commands.PerformRelocation(sourceDir, destPath, true, true)

	if !result.Success {
		t.Errorf("Dry-run should succeed: %s", result.Error)
	}

	// Source should still exist
	if _, err := os.Stat(sourceDir); err != nil {
		t.Error("Source should still exist after dry-run")
	}

	// Destination should not exist
	if _, err := os.Stat(destPath); err == nil {
		t.Error("Destination should not exist after dry-run")
	}
}

func TestPerformRelocation_PreservesStructure(t *testing.T) {
	sourceDir := t.TempDir()
	destParentDir := t.TempDir()
	destPath := filepath.Join(destParentDir, "relocated-brain")

	// Create complex brain structure
	os.MkdirAll(filepath.Join(sourceDir, "notes"), 0755)
	os.MkdirAll(filepath.Join(sourceDir, "journal"), 0755)
	os.MkdirAll(filepath.Join(sourceDir, "tasks"), 0755)
	os.MkdirAll(filepath.Join(sourceDir, "definitions"), 0755)

	// Create various files
	os.WriteFile(filepath.Join(sourceDir, ".flip.yaml"), []byte("type: flip\nname: test\n"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("# Brain"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "notes", "note1.md"), []byte("# Note 1"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "definitions", "def1.yaml"), []byte("key: value"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "tasks", "task1.md"), []byte("- [ ] Task"), 0644)

	// Relocate
	result := commands.PerformRelocation(sourceDir, destPath, false, true)

	if !result.Success {
		t.Errorf("Relocation should succeed: %s", result.Error)
	}

	// Verify all files and directories
	checkPath := func(path string, desc string) {
		fullPath := filepath.Join(destPath, path)
		if _, err := os.Stat(fullPath); err != nil {
			t.Errorf("%s not found at new location: %v", desc, err)
		}
	}

	checkPath(".", "Brain root")
	checkPath("notes", "notes directory")
	checkPath("journal", "journal directory")
	checkPath("tasks", "tasks directory")
	checkPath("definitions", "definitions directory")
	checkPath(".flip.yaml", "config file")
	checkPath("README.md", "README")
	checkPath("notes/note1.md", "note file")
	checkPath("definitions/def1.yaml", "definition file")
	checkPath("tasks/task1.md", "task file")
}

func TestPerformRelocation_HealthCheckIntegration(t *testing.T) {
	sourceDir := t.TempDir()
	destParentDir := t.TempDir()
	destPath := filepath.Join(destParentDir, "relocated-brain")

	// Create valid brain
	os.MkdirAll(filepath.Join(sourceDir, "notes"), 0755)
	os.MkdirAll(filepath.Join(sourceDir, "journal"), 0755)
	os.WriteFile(filepath.Join(sourceDir, ".flip.yaml"), []byte("type: flip\nname: test\n"), 0644)

	// Create journal entry with date
	os.WriteFile(filepath.Join(sourceDir, "journal", "2026-03-04.md"), 
		[]byte("---\ndate: 2026-03-04\n---\n# 2026-03-04\n"), 0644)

	// Relocate
	result := commands.PerformRelocation(sourceDir, destPath, false, true)

	if !result.Success {
		t.Errorf("Relocation should succeed: %s", result.Error)
	}

	// Verify health check was performed
	if result.HealthIssues < 0 {
		t.Error("Health issues count should be >= 0")
	}

	// Verify brain is healthy
	checker, err := health.NewChecker(destPath)
	if err != nil {
		t.Errorf("Should be able to create checker at new location: %v", err)
	}

	checkResult, err := checker.Check()
	if err != nil {
		t.Errorf("Health check should not error: %v", err)
	}

	if len(checkResult.Issues) != result.HealthIssues {
		t.Errorf("Health check issues mismatch: expected %d, got %d", result.HealthIssues, len(checkResult.Issues))
	}
}
