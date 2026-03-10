package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/httrp/flip/internal/brain"
)

func TestFindMeetingSeriesSkipsProtocolsDir(t *testing.T) {
	dir := t.TempDir()
	meetingsDir := filepath.Join(dir, "meetings")
	protocolsDir := filepath.Join(meetingsDir, "protocols")

	if err := os.MkdirAll(protocolsDir, 0755); err != nil {
		t.Fatal(err)
	}

	meetingContent := "---\nseries: Weekly Sync\n---\n"
	protocolContent := "---\nseries: Should Not Count\n---\n"

	if err := os.WriteFile(filepath.Join(meetingsDir, "2026-03-10-weekly-sync.md"), []byte(meetingContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(protocolsDir, "weekly-sync-rolling-protocol.md"), []byte(protocolContent), 0644); err != nil {
		t.Fatal(err)
	}

	series, err := findMeetingSeries(dir, brain.BrainTypeFlip)
	if err != nil {
		t.Fatalf("findMeetingSeries() error = %v", err)
	}

	if len(series) != 1 {
		t.Fatalf("findMeetingSeries() count = %d, want 1", len(series))
	}
	if series[0].Name != "Weekly Sync" {
		t.Fatalf("findMeetingSeries() name = %q, want %q", series[0].Name, "Weekly Sync")
	}
	if series[0].Count != 1 {
		t.Fatalf("findMeetingSeries() meeting count = %d, want 1", series[0].Count)
	}
}
