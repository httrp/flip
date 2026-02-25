package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/httrp/flip/internal/health"
)

// Integration test: full migration with notes, assets, and links
func TestFullMigrationWithLinks(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create source brain (Logseq-style)
	// pages/project-alpha.md references project-beta
	mkNote(t, filepath.Join(src, "pages", "project-alpha.md"),
		"# Project Alpha\n\nSee [[Project Beta]] for related work.\n\n![diagram](../assets/diagram.png)")
	mkNote(t, filepath.Join(src, "pages", "project-beta.md"),
		"# Project Beta\n\nReferences back to [[Project Alpha]].")
	mkAsset(t, filepath.Join(src, "assets", "diagram.png"), "PNG_DATA")

	// Plan migration
	sourceStruct := GetBrainStructure(health.BrainTypeLogseq)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	planner := NewPlanner(src, dst, sourceStruct, targetStruct)
	plan, err := planner.BuildPlan(ModeFull, "", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	// Execute
	executor := NewExecutor(plan, sourceStruct, targetStruct)
	execLog, err := executor.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Verify
	if !execLog.Success {
		t.Fatalf("execution not successful")
	}
	if len(execLog.ItemsWritten) != 3 {
		t.Fatalf("expected 3 items written (2 notes + 1 asset), got %d", len(execLog.ItemsWritten))
	}

	// Check notes exist
	alphaMigrated := filepath.Join(dst, "notes", "project-alpha.md")
	betaMigrated := filepath.Join(dst, "notes", "project-beta.md")
	assetMigrated := filepath.Join(dst, "assets", "diagram.png")

	if _, err := os.Stat(alphaMigrated); err != nil {
		t.Fatalf("alpha note not migrated: %v", err)
	}
	if _, err := os.Stat(betaMigrated); err != nil {
		t.Fatalf("beta note not migrated: %v", err)
	}
	if _, err := os.Stat(assetMigrated); err != nil {
		t.Fatalf("asset not migrated: %v", err)
	}

	// Check link rewriting
	alphaContent, _ := os.ReadFile(alphaMigrated)
	if !strings.Contains(string(alphaContent), "project-beta") && !strings.Contains(string(alphaContent), "Project Beta") {
		t.Errorf("expected link to project-beta in alpha, got: %s", alphaContent)
	}

	// Check asset reference updated
	if !strings.Contains(string(alphaContent), "assets/diagram.png") {
		t.Errorf("expected asset reference updated, got: %s", alphaContent)
	}

	// Check execution log exists
	logPath := filepath.Join(dst, ".flip-migration-log.json")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("execution log not written: %v", err)
	}
}

// Test partial migration (selective folders)
func TestPartialMigrationFolders(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Source with multiple folders
	mkNote(t, filepath.Join(src, "work", "task1.md"), "# Task 1")
	mkNote(t, filepath.Join(src, "work", "task2.md"), "# Task 2")
	mkNote(t, filepath.Join(src, "personal", "note.md"), "# Personal Note")
	mkNote(t, filepath.Join(src, "archive", "old.md"), "# Old")

	sourceStruct := GetBrainStructure(health.BrainTypeObsidian)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	planner := NewPlanner(src, dst, sourceStruct, targetStruct)

	// Migrate only work folder
	plan, err := planner.BuildPlan(ModePartial, "", []string{"work"}, 0)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	if plan.NotesCount != 2 {
		t.Fatalf("expected 2 notes in plan (work folder only), got %d", plan.NotesCount)
	}

	executor := NewExecutor(plan, sourceStruct, targetStruct)
	execLog, err := executor.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Verify only work notes migrated
	if len(execLog.ItemsWritten) != 2 {
		t.Fatalf("expected 2 items, got %d", len(execLog.ItemsWritten))
	}

	// Personal should NOT be migrated
	personalPath := filepath.Join(dst, "notes", "note.md")
	if _, err := os.Stat(personalPath); err == nil {
		t.Errorf("personal note should not be migrated")
	}
}

// Test single note migration
func TestSingleNoteMigration(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	mkNote(t, filepath.Join(src, "notes", "target.md"), "# Target Note\n\nContent here.")
	mkNote(t, filepath.Join(src, "notes", "other.md"), "# Other Note")

	sourceStruct := GetBrainStructure(health.BrainTypeFlip)
	targetStruct := GetBrainStructure(health.BrainTypeObsidian)
	planner := NewPlanner(src, dst, sourceStruct, targetStruct)

	plan, err := planner.BuildPlan(ModeSingle, "target", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	if plan.NotesCount != 1 {
		t.Fatalf("expected 1 note, got %d", plan.NotesCount)
	}

	executor := NewExecutor(plan, sourceStruct, targetStruct)
	_, err = executor.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Check target migrated - Obsidian has no enforced structure, so file goes to root
	// The filename is preserved since both Flip and Obsidian use same format
	targetMigrated := filepath.Join(dst, "notes", "target.md")
	if _, err := os.Stat(targetMigrated); err != nil {
		t.Fatalf("target note not migrated: %v", err)
	}

	// Other should NOT be migrated
	otherMigrated := filepath.Join(dst, "notes", "other.md")
	if _, err := os.Stat(otherMigrated); err == nil {
		t.Errorf("other note should not be migrated in single mode")
	}
}

// Test conflict detection prevents execution
func TestConflictDetection(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Two files in SAME directory that slug to same target name
	// "Test Note.md" and "test-note.md" both become "test-note.md" in Flip
	mkNote(t, filepath.Join(src, "notes", "Test Note.md"), "# Test")
	mkNote(t, filepath.Join(src, "notes", "test-note.md"), "# Test")

	sourceStruct := GetBrainStructure(health.BrainTypeObsidian)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	planner := NewPlanner(src, dst, sourceStruct, targetStruct)

	plan, err := planner.BuildPlan(ModeFull, "", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	// Should detect conflict
	if len(plan.Conflicts) == 0 {
		t.Fatalf("expected conflicts for duplicate target names")
	}
}

func TestFullMigrationIncludesDefinitions(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	mkNote(t, filepath.Join(src, "pages", "alpha.md"), "# Alpha")
	defsContent := "organizations:\n  WORK:\n    name: Work\n"
	mkNote(t, filepath.Join(src, "definitions", "organizations.yaml"), defsContent)

	sourceStruct := GetBrainStructure(health.BrainTypeLogseq)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	planner := NewPlanner(src, dst, sourceStruct, targetStruct)

	plan, err := planner.BuildPlan(ModeFull, "", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	foundDefinition := false
	for _, item := range plan.Items {
		if item.Type == "definition" && item.SourcePath == filepath.Join("definitions", "organizations.yaml") {
			foundDefinition = true
			break
		}
	}
	if !foundDefinition {
		t.Fatalf("expected organizations.yaml to be included as definition item")
	}

	executor := NewExecutor(plan, sourceStruct, targetStruct)
	_, err = executor.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	migratedPath := filepath.Join(dst, "definitions", "organizations.yaml")
	migrated, err := os.ReadFile(migratedPath)
	if err != nil {
		t.Fatalf("definitions file not migrated: %v", err)
	}

	if string(migrated) != defsContent {
		t.Fatalf("definitions content changed during migration")
	}
}

// Helper to create note with directories
func mkNote(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
