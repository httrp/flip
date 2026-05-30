package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/httrp/flip/internal/protocols"
)

func TestDefaultRollingProtocolPath(t *testing.T) {
	got, err := defaultRollingProtocolPath("/brain", "flip", "Copilot Jour fixe")
	if err != nil {
		t.Fatalf("defaultRollingProtocolPath() error = %v", err)
	}

	want := filepath.Join("/brain", "meetings", "protocols", "copilot-jour-fixe-rolling-protocol.md")
	if got != want {
		t.Fatalf("defaultRollingProtocolPath() = %q, want %q", got, want)
	}
}

func TestDefaultRollingProtocolPathLogseq(t *testing.T) {
	got, err := defaultRollingProtocolPath("/graph", "logseq", "Weekly Sync")
	if err != nil {
		t.Fatalf("defaultRollingProtocolPath() error = %v", err)
	}

	want := filepath.Join("/graph", "pages", "protocols", "weekly-sync-rolling-protocol.md")
	if got != want {
		t.Fatalf("defaultRollingProtocolPath() = %q, want %q", got, want)
	}
}

func TestSlugifySeriesName(t *testing.T) {
	got := slugifySeriesName("(Project Alpha) Sync: Smith / Jones")
	want := "project-alpha-sync-smith-jones"
	if got != want {
		t.Fatalf("slugifySeriesName() = %q, want %q", got, want)
	}
}

func TestMergeRollingProtocolPreservedSectionsKeepsManualNotes(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "rolling.md")
	existing := strings.Join([]string{
		"# Rolling Protocol: Weekly Sync",
		"",
		"## Manual Notes",
		"",
		protocols.RollingProtocolManualNotesStart,
		"Customer asked for explicit budget update.",
		protocols.RollingProtocolManualNotesEnd,
		"",
		"## Open Questions",
		"",
		protocols.RollingProtocolOpenQuestionsStart,
		"Should we split rollout into two phases?",
		protocols.RollingProtocolOpenQuestionsEnd,
		"",
	}, "\n")
	if err := os.WriteFile(filePath, []byte(existing), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	generated := strings.Join([]string{
		"# Rolling Protocol: Weekly Sync",
		"",
		"## Manual Notes",
		"",
		protocols.RollingProtocolManualNotesStart,
		"_Add manual notes here. This section is preserved by flip meeting protocol --sync._",
		protocols.RollingProtocolManualNotesEnd,
		"",
		"## Open Questions",
		"",
		protocols.RollingProtocolOpenQuestionsStart,
		"_Track unresolved questions here. This section is preserved by flip meeting protocol --sync._",
		protocols.RollingProtocolOpenQuestionsEnd,
		"",
	}, "\n")

	merged, err := mergeRollingProtocolPreservedSections(filePath, generated)
	if err != nil {
		t.Fatalf("mergeRollingProtocolPreservedSections() error = %v", err)
	}

	if !strings.Contains(merged, "Customer asked for explicit budget update.") {
		t.Fatal("Expected merged protocol to keep existing manual notes")
	}
	if strings.Contains(merged, "_Add manual notes here") {
		t.Fatal("Expected merged protocol to replace placeholder manual notes")
	}
	if !strings.Contains(merged, "Should we split rollout into two phases?") {
		t.Fatal("Expected merged protocol to keep existing open questions")
	}
}

func TestMergeRollingProtocolPreservedSectionsMissingFileUsesGenerated(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "missing.md")
	generated := strings.Join([]string{
		"# Rolling Protocol: Weekly Sync",
		protocols.RollingProtocolManualNotesStart,
		"placeholder",
		protocols.RollingProtocolManualNotesEnd,
	}, "\n")

	merged, err := mergeRollingProtocolPreservedSections(filePath, generated)
	if err != nil {
		t.Fatalf("mergeRollingProtocolPreservedSections() error = %v", err)
	}
	if merged != generated {
		t.Fatal("Expected generated content to be returned unchanged when no existing file is present")
	}
}

func TestAddRelatedSectionBulletAppendsAndAvoidsDuplicate(t *testing.T) {
	content := strings.Join([]string{
		"# Weekly Sync",
		"",
		"## Related",
		"- [[related-note]]",
		"",
	}, "\n")
	bullet := "- [Rolling Protocol: Weekly Sync](protocols/weekly-sync-rolling-protocol.md)"

	updated := addRelatedSectionBullet(content, bullet)
	if !strings.Contains(updated, bullet) {
		t.Fatal("Expected backlink to be added to Related section")
	}

	updatedAgain := addRelatedSectionBullet(updated, bullet)
	if strings.Count(updatedAgain, bullet) != 1 {
		t.Fatal("Expected backlink to be added only once")
	}
}

func TestEnsureMeetingRollingProtocolBacklinkCreatesRelatedSection(t *testing.T) {
	meetingPath := filepath.Join(t.TempDir(), "meeting.md")
	protocolPath := filepath.Join(filepath.Dir(meetingPath), "protocols", "weekly-sync-rolling-protocol.md")
	content := strings.Join([]string{
		"---",
		"title: Weekly Sync",
		"date: 2026-03-10",
		"---",
		"",
		"# Weekly Sync",
		"",
		"## Notes",
		"Budget risk was raised.",
	}, "\n")

	if err := os.WriteFile(meetingPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(protocolPath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := ensureMeetingRollingProtocolBacklink(meetingPath, protocolPath, "Weekly Sync"); err != nil {
		t.Fatalf("ensureMeetingRollingProtocolBacklink() error = %v", err)
	}

	updated, err := os.ReadFile(meetingPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(updated), "## Related") {
		t.Fatal("Expected Related section to be created")
	}
	if !strings.Contains(string(updated), "[Rolling Protocol: Weekly Sync](protocols/weekly-sync-rolling-protocol.md)") {
		t.Fatal("Expected rolling protocol backlink to be created with relative path")
	}
}

func TestFindMeetingsInSeriesSkipsProtocolsDirectory(t *testing.T) {
	meetingsDir := filepath.Join(t.TempDir(), "meetings")
	protocolsDir := filepath.Join(meetingsDir, "protocols")
	if err := os.MkdirAll(protocolsDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	meetingContent := strings.Join([]string{
		"---",
		"title: Weekly Sync 1",
		"date: 2026-03-10",
		"series: Weekly Sync",
		"---",
		"",
		"# Weekly Sync 1",
	}, "\n")
	protocolContent := strings.Join([]string{
		"---",
		"title: Weekly Sync Rolling Protocol",
		"date: 2026-03-10",
		"series: Weekly Sync",
		"---",
		"",
		"# Rolling Protocol",
	}, "\n")

	if err := os.WriteFile(filepath.Join(meetingsDir, "2026-03-10-weekly-sync.md"), []byte(meetingContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(protocolsDir, "weekly-sync-rolling-protocol.md"), []byte(protocolContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	meetings, err := findMeetingsInSeries(meetingsDir, "Weekly Sync", "", "")
	if err != nil {
		t.Fatalf("findMeetingsInSeries() error = %v", err)
	}
	if len(meetings) != 1 {
		t.Fatalf("findMeetingsInSeries() count = %d, want 1", len(meetings))
	}
}
