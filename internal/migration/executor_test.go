package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/httrp/flip/internal/health"
)

func TestExecutorTransformsContentBetweenBrainTypes(t *testing.T) {
	sourceDir := t.TempDir()
	targetDir := t.TempDir()

	noteDir := filepath.Join(sourceDir, "pages")
	if err := os.MkdirAll(noteDir, 0o755); err != nil {
		t.Fatalf("failed to create source notes dir: %v", err)
	}

	sourcePath := filepath.Join(noteDir, "project-alpha.md")
	sourceContent := "title:: Project Alpha\ntags:: [work, important]\n\n# Project Alpha\n"
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0o644); err != nil {
		t.Fatalf("failed to write source note: %v", err)
	}

	plan := &MigrationPlan{
		SourceBrainPath: sourceDir,
		TargetBrainPath: targetDir,
		Mode:            ModeSingle,
		Items: []PlanItem{
			{
				SourcePath: "pages/project-alpha.md",
				TargetPath: "notes/project-alpha.md",
				Type:       "note",
			},
		},
	}

	sourceStruct := GetBrainStructure(health.BrainTypeLogseq)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	executor := NewExecutor(plan, sourceStruct, targetStruct)

	if _, err := executor.Execute(); err != nil {
		t.Fatalf("migration execute failed: %v", err)
	}

	targetPath := filepath.Join(targetDir, "notes", "project-alpha.md")
	migrated, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read migrated note: %v", err)
	}

	content := string(migrated)
	if !strings.Contains(content, "---\n") {
		t.Fatalf("expected YAML frontmatter in migrated note, got:\n%s", content)
	}
	if !strings.Contains(content, "title: Project Alpha") {
		t.Fatalf("expected transformed title property, got:\n%s", content)
	}
	if !strings.Contains(content, "tags: [work, important]") {
		t.Fatalf("expected transformed tags property, got:\n%s", content)
	}
}
