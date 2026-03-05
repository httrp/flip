package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/httrp/flip/internal/health"
)

// helper to create files with dirs
func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestBuildPlanSingleNoteFound(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Simulate Logseq source: pages/hello-world.md
	writeFile(t, filepath.Join(src, "pages", "hello-world.md"), "# Hello World\n")

	p := NewPlanner(src, dst, GetBrainStructure(health.BrainTypeLogseq), GetBrainStructure(health.BrainTypeFlip))
	plan, err := p.BuildPlan(ModeSingle, "Hello World", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan single: %v", err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(plan.Items))
	}
	if plan.Items[0].TargetPath != filepath.Join("notes", "hello-world.md") {
		t.Fatalf("unexpected target path: %s", plan.Items[0].TargetPath)
	}
}

func TestBuildPlanPartialFolders(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	writeFile(t, filepath.Join(src, "projects", "Project A.md"), "# A\n")
	writeFile(t, filepath.Join(src, "projects", "notes.txt"), "ignore")
	writeFile(t, filepath.Join(src, "projects", "Project B.md"), "# B\n")

	p := NewPlanner(src, dst, GetBrainStructure(health.BrainTypeObsidian), GetBrainStructure(health.BrainTypeFlip))
	plan, err := p.BuildPlan(ModePartial, "", []string{"projects"}, 0)
	if err != nil {
		t.Fatalf("BuildPlan partial: %v", err)
	}
	if plan.NotesCount != 2 {
		t.Fatalf("expected 2 notes, got %d", plan.NotesCount)
	}
	// Source folder is preserved, filenames are transformed to Flip slug format
	expected := map[string]bool{
		filepath.Join("projects", "project-a.md"): true,
		filepath.Join("projects", "project-b.md"): true,
	}
	for _, it := range plan.Items {
		if it.Type != "note" {
			continue
		}
		if !expected[it.TargetPath] {
			t.Fatalf("unexpected target: %s (expected one of %v)", it.TargetPath, expected)
		}
	}
}

func TestBuildPlanFullCountsAndAssets(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	writeFile(t, filepath.Join(src, "pages", "Alpha.md"), "# Alpha\n![](assets/img.png)")
	writeFile(t, filepath.Join(src, "pages", "Beta.md"), "# Beta\n")
	// assets
	writeFile(t, filepath.Join(src, "assets", "img.png"), "PNG")
	writeFile(t, filepath.Join(src, "attachments", "doc.pdf"), "%PDF")

	p := NewPlanner(src, dst, GetBrainStructure(health.BrainTypeLogseq), GetBrainStructure(health.BrainTypeFlip))
	plan, err := p.BuildPlan(ModeFull, "", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan full: %v", err)
	}
	if plan.NotesCount != 2 {
		t.Fatalf("expected 2 notes, got %d", plan.NotesCount)
	}
	if plan.AssetsCount == 0 {
		t.Fatalf("expected some assets, got 0")
	}
}

func TestDetectConflictsOnTarget(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Two different source files in SAME directory that slug to the same target name
	writeFile(t, filepath.Join(src, "notes", "Alpha Note.md"), "# Alpha\n")
	writeFile(t, filepath.Join(src, "notes", "alpha-note.md"), "# alpha\n")

	p := NewPlanner(src, dst, GetBrainStructure(health.BrainTypeObsidian), GetBrainStructure(health.BrainTypeFlip))
	plan, err := p.BuildPlan(ModeFull, "", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan full for conflict: %v", err)
	}
	if len(plan.Conflicts) == 0 {
		t.Fatalf("expected conflicts due to same target path, found none")
	}
}

func TestAddFullBrainIncludesOrphaned(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create a .orphaned directory with content
	writeFile(t, filepath.Join(src, "notes", "visible.md"), "# Visible\n")
	writeFile(t, filepath.Join(src, ".orphaned", "pages", "recovered.md"), "# Recovered\n")
	// Other hidden dirs should still be skipped
	writeFile(t, filepath.Join(src, ".git", "config"), "gitconfig")

	p := NewPlanner(src, dst, GetBrainStructure(health.BrainTypeFlip), GetBrainStructure(health.BrainTypeFlip))
	plan, err := p.BuildPlan(ModeFull, "", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	foundOrphaned := false
	for _, item := range plan.Items {
		if item.SourcePath == filepath.Join(".orphaned", "pages", "recovered.md") {
			foundOrphaned = true
		}
		// Ensure .git content is NOT in the plan
		if item.SourcePath == filepath.Join(".git", "config") {
			t.Fatalf(".git content should not be included in plan")
		}
	}
	if !foundOrphaned {
		t.Errorf("expected .orphaned/pages/recovered.md to be in plan, but it was not")
	}
}

func TestBoilerplateFilesSkipped(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	writeFile(t, filepath.Join(src, "README.md"), "# Source README\n")
	writeFile(t, filepath.Join(src, "WELCOME.md"), "# Welcome\n")
	writeFile(t, filepath.Join(src, "notes", "real-note.md"), "# Real content\n")

	p := NewPlanner(src, dst, GetBrainStructure(health.BrainTypeFlip), GetBrainStructure(health.BrainTypeFlip))
	plan, err := p.BuildPlan(ModeFull, "", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	skippedCount := 0
	for _, item := range plan.Items {
		if item.Skipped && item.SourcePath == "README.md" {
			skippedCount++
		}
		if item.Skipped && item.SourcePath == "WELCOME.md" {
			skippedCount++
		}
	}
	if skippedCount != 2 {
		t.Errorf("expected README.md and WELCOME.md to be skipped, got %d skipped", skippedCount)
	}

	// Real note should NOT be skipped
	for _, item := range plan.Items {
		if item.SourcePath == filepath.Join("notes", "real-note.md") && item.Skipped {
			t.Errorf("real-note.md should not be skipped")
		}
	}
}

func TestDetectTargetConflicts(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Source has a note
	writeFile(t, filepath.Join(src, "notes", "existing.md"), "# From Source\n")
	// Target already has the same file
	writeFile(t, filepath.Join(dst, "notes", "existing.md"), "# Already Here\n")

	p := NewPlanner(src, dst, GetBrainStructure(health.BrainTypeFlip), GetBrainStructure(health.BrainTypeFlip))
	plan, err := p.BuildPlan(ModeFull, "", nil, 0)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}

	foundTargetConflict := false
	for _, c := range plan.Conflicts {
		if strings.Contains(c, "target already exists") {
			foundTargetConflict = true
			break
		}
	}
	if !foundTargetConflict {
		t.Errorf("expected 'target already exists' conflict, got: %v", plan.Conflicts)
	}
}

func TestApplyConflictStrategySkip(t *testing.T) {
	plan := &MigrationPlan{
		Items: []PlanItem{
			{SourcePath: "a.md", TargetPath: "notes/a.md", Type: "note", Reason: "target already exists"},
			{SourcePath: "b.md", TargetPath: "notes/b.md", Type: "note"},
		},
		Conflicts: []string{"target already exists: notes/a.md (from a.md)"},
	}

	ApplyConflictStrategy(plan, ConflictSkip)

	if !plan.Items[0].Skipped {
		t.Error("item with 'target already exists' should be skipped")
	}
	if plan.Items[1].Skipped {
		t.Error("item without conflict should not be skipped")
	}
	if len(plan.Conflicts) != 0 {
		t.Errorf("conflicts for target-exists should be cleared with skip strategy, got %d", len(plan.Conflicts))
	}
}

func TestApplyConflictStrategyOverwrite(t *testing.T) {
	plan := &MigrationPlan{
		Items: []PlanItem{
			{SourcePath: "a.md", TargetPath: "notes/a.md", Type: "note", Reason: "target already exists"},
		},
		Conflicts: []string{"target already exists: notes/a.md (from a.md)"},
	}

	ApplyConflictStrategy(plan, ConflictOverwrite)

	if plan.Items[0].Skipped {
		t.Error("overwrite strategy should not skip items")
	}
	if plan.Items[0].Reason != "will overwrite existing target" {
		t.Errorf("reason should be updated, got: %s", plan.Items[0].Reason)
	}
	if len(plan.Conflicts) != 0 {
		t.Errorf("target-exists conflicts should be cleared with overwrite strategy, got %d", len(plan.Conflicts))
	}
}
