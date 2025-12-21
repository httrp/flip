package migration

import (
	"os"
	"path/filepath"
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
