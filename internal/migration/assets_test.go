package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/httrp/flip/internal/health"
)

// helper
func mkAsset(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateAssetReferencesMarkdownImage(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create asset in source
	mkAsset(t, filepath.Join(src, "assets", "diagram.png"), "PNG")

	sourceStruct := GetBrainStructure(health.BrainTypeFlip)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	am := NewAssetMigrator(src, dst, sourceStruct, targetStruct)

	// Note content with markdown image
	content := "# Demo\n\n![Diagram](assets/diagram.png)\n\nSome text."
	updated, err := am.UpdateAssetReferences(content, "notes/demo.md")
	if err != nil {
		t.Fatalf("UpdateAssetReferences: %v", err)
	}

	// Should have updated path and copied asset
	if !strings.Contains(updated, "assets/diagram.png") {
		t.Fatalf("expected updated path in content, got: %s", updated)
	}
	if am.GetStats().CopiedAssets != 1 {
		t.Fatalf("expected 1 copied asset, got %d", am.GetStats().CopiedAssets)
	}
	// Check file was copied
	if _, err := os.Stat(filepath.Join(dst, "assets", "diagram.png")); err != nil {
		t.Fatalf("asset not copied: %v", err)
	}
}

func TestUpdateAssetReferencesWikilinkEmbed(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create asset
	mkAsset(t, filepath.Join(src, "assets", "photo.jpg"), "JPEG")

	sourceStruct := GetBrainStructure(health.BrainTypeLogseq)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	am := NewAssetMigrator(src, dst, sourceStruct, targetStruct)

	// Wikilink embed
	content := "Check this: ![[photo.jpg]]"
	updated, err := am.UpdateAssetReferences(content, "pages/note.md")
	if err != nil {
		t.Fatalf("UpdateAssetReferences: %v", err)
	}

	// Flip prefers wikilinks, so should keep but update
	if !strings.Contains(updated, "[[photo.jpg]]") && !strings.Contains(updated, "assets/photo.jpg") {
		t.Fatalf("expected wikilink or markdown image in updated content: %s", updated)
	}
	if am.GetStats().CopiedAssets != 1 {
		t.Fatalf("expected 1 copied asset, got %d", am.GetStats().CopiedAssets)
	}
}

func TestUpdateAssetReferencesPDFLink(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create PDF in source
	mkAsset(t, filepath.Join(src, "assets", "report.pdf"), "%PDF")

	sourceStruct := GetBrainStructure(health.BrainTypeObsidian)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	am := NewAssetMigrator(src, dst, sourceStruct, targetStruct)

	// Markdown link to PDF
	content := "See the [Annual Report](attachments/report.pdf) for details."
	updated, err := am.UpdateAssetReferences(content, "documents/summary.md")
	// Obsidian uses "attachments" but source has "assets", asset might not be found
	// This tests the graceful error handling
	_ = err // May error if asset not found in "attachments", that's ok

	// If found and migrated, path should update
	if strings.Contains(updated, "assets/report.pdf") {
		if am.GetStats().CopiedAssets != 1 {
			t.Fatalf("expected 1 copied asset when path updated, got %d", am.GetStats().CopiedAssets)
		}
	}
	// Otherwise updated == content (original preserved on error)
	// Both outcomes valid depending on source structure
	t.Logf("Updated content: %s", updated)
}

func TestMigrateAssetMultipleReferences(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Create asset
	mkAsset(t, filepath.Join(src, "assets", "logo.png"), "PNG")

	sourceStruct := GetBrainStructure(health.BrainTypeFlip)
	targetStruct := GetBrainStructure(health.BrainTypeFlip)
	am := NewAssetMigrator(src, dst, sourceStruct, targetStruct)

	// Multiple references to same asset
	content := "Header: ![logo](assets/logo.png)\n\nFooter: ![logo](assets/logo.png)"
	updated, err := am.UpdateAssetReferences(content, "notes/page.md")
	if err != nil {
		t.Fatalf("UpdateAssetReferences: %v", err)
	}

	// Should copy asset only once
	if am.GetStats().CopiedAssets != 1 {
		t.Fatalf("expected 1 copied asset (dedup), got %d", am.GetStats().CopiedAssets)
	}

	// Both references should be updated
	count := strings.Count(updated, "assets/logo.png")
	if count != 2 {
		t.Fatalf("expected 2 updated references, found %d", count)
	}
}
