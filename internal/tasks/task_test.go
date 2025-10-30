package tasks

import (
	"testing"
)

func TestUpdateTaskStatusInLine(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		newStatus  Status
		expected   string
	}{
		{
			name:      "Open to Done",
			input:     "- [ ] Task 1: Test toggling to done",
			newStatus: StatusDone,
			expected:  "- [x] Task 1: Test toggling to done",
		},
		{
			name:      "Done to Open",
			input:     "- [x] Task already done",
			newStatus: StatusOpen,
			expected:  "- [ ] Task already done",
		},
		{
			name:      "Open to In Progress",
			input:     "- [ ] Task to start",
			newStatus: StatusInProgress,
			expected:  "- [>] Task to start",
		},
		{
			name:      "Indented task",
			input:     "  - [ ] Indented task",
			newStatus: StatusDone,
			expected:  "  - [x] Indented task",
		},
		{
			name:      "Asterisk marker",
			input:     "* [ ] Task with asterisk",
			newStatus: StatusDone,
			expected:  "* [x] Task with asterisk",
		},
		{
			name:      "Plus marker",
			input:     "+ [ ] Task with plus",
			newStatus: StatusDone,
			expected:  "+ [x] Task with plus",
		},
		{
			name:      "To Deferred",
			input:     "- [ ] Task to defer",
			newStatus: StatusDeferred,
			expected:  "- [~] Task to defer",
		},
		{
			name:      "To Cancelled",
			input:     "- [ ] Task to cancel",
			newStatus: StatusCancelled,
			expected:  "- [-] Task to cancel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updateTaskStatusInLine(tt.input, tt.newStatus)
			if result != tt.expected {
				t.Errorf("updateTaskStatusInLine() = %q, want %q", result, tt.expected)
			}
		})
	}
}
