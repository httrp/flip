package tasks

import (
	"testing"
	"time"
)

func TestUpdateTaskStatusInLine(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		newStatus Status
		expected  string
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
			// Uses '/' as defined in CheckboxFromStatus
			expected:  "- [/] Task to start",
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
			// Uses '>' as defined in CheckboxFromStatus
			expected:  "- [>] Task to defer",
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

func TestStatusFromCheckbox(t *testing.T) {
	tests := []struct {
		marker   string
		expected Status
	}{
		{" ", StatusOpen},
		{"", StatusOpen},
		{"x", StatusDone},
		{"X", StatusDone},
		{"/", StatusInProgress},
		{"~", StatusInProgress},
		{">", StatusDeferred},
		{"-", StatusCancelled},
		{"?", StatusOpen}, // Unknown defaults to open
	}

	for _, tt := range tests {
		t.Run(tt.marker, func(t *testing.T) {
			result := StatusFromCheckbox(tt.marker)
			if result != tt.expected {
				t.Errorf("StatusFromCheckbox(%q) = %v, want %v", tt.marker, result, tt.expected)
			}
		})
	}
}

func TestCheckboxFromStatus(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusOpen, " "},
		{StatusDone, "x"},
		{StatusInProgress, "/"},
		{StatusDeferred, ">"},
		{StatusCancelled, "-"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := CheckboxFromStatus(tt.status)
			if result != tt.expected {
				t.Errorf("CheckboxFromStatus(%v) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	// All statuses should have an icon
	statuses := []Status{StatusOpen, StatusDone, StatusInProgress, StatusDeferred, StatusCancelled}
	
	for _, s := range statuses {
		icon := StatusIcon(s)
		if icon == "" {
			t.Errorf("StatusIcon(%v) returned empty string", s)
		}
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusOpen, "Open"},
		{StatusDone, "Done"},
		{StatusInProgress, "In Progress"},
		{StatusDeferred, "Deferred"},
		{StatusCancelled, "Cancelled"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := StatusLabel(tt.status)
			if result != tt.expected {
				t.Errorf("StatusLabel(%v) = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	ctx := TaskContext{
		FilePath:   "/path/to/file.md",
		LineNumber: 10,
	}

	id1 := GenerateID("Test task", ctx, 10)
	id2 := GenerateID("Test task", ctx, 10)
	id3 := GenerateID("Different task", ctx, 10)
	id4 := GenerateID("Test task", ctx, 20)

	// Same input should produce same ID
	if id1 != id2 {
		t.Errorf("Same input should produce same ID: %s != %s", id1, id2)
	}

	// Different description should produce different ID
	if id1 == id3 {
		t.Errorf("Different description should produce different ID")
	}

	// Different line number should produce different ID
	if id1 == id4 {
		t.Errorf("Different line number should produce different ID")
	}

	// ID should be 16 characters (8 bytes hex)
	if len(id1) != 16 {
		t.Errorf("ID should be 16 characters, got %d", len(id1))
	}
}

func TestTaskIsOverdue(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name:     "No due date",
			task:     Task{Status: StatusOpen},
			expected: false,
		},
		{
			name:     "Due yesterday, still open",
			task:     Task{Status: StatusOpen, Due: &yesterday},
			expected: true,
		},
		{
			name:     "Due tomorrow",
			task:     Task{Status: StatusOpen, Due: &tomorrow},
			expected: false,
		},
		{
			name:     "Due yesterday but done",
			task:     Task{Status: StatusDone, Due: &yesterday},
			expected: false,
		},
		{
			name:     "Due yesterday but cancelled",
			task:     Task{Status: StatusCancelled, Due: &yesterday},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.task.IsOverdue(); result != tt.expected {
				t.Errorf("IsOverdue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTaskIsDueToday(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		name     string
		due      *time.Time
		expected bool
	}{
		{"No due date", nil, false},
		{"Due today", &today, true},
		{"Due yesterday", &yesterday, false},
		{"Due tomorrow", &tomorrow, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{Due: tt.due}
			if result := task.IsDueToday(); result != tt.expected {
				t.Errorf("IsDueToday() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTaskDuration(t *testing.T) {
	start := time.Now()
	end := start.Add(2 * time.Hour)

	tests := []struct {
		name     string
		task     Task
		expected time.Duration
	}{
		{
			name:     "Not started",
			task:     Task{},
			expected: 0,
		},
		{
			name:     "Started but not completed",
			task:     Task{Started: &start},
			expected: 0,
		},
		{
			name:     "Completed",
			task:     Task{Started: &start, Completed: &end},
			expected: 2 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := tt.task.Duration(); result != tt.expected {
				t.Errorf("Duration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPriorityConstants(t *testing.T) {
	// Test priority values
	if PriorityHigh != "high" {
		t.Errorf("PriorityHigh should be 'high', got %s", PriorityHigh)
	}
	if PriorityMedium != "medium" {
		t.Errorf("PriorityMedium should be 'medium', got %s", PriorityMedium)
	}
	if PriorityLow != "low" {
		t.Errorf("PriorityLow should be 'low', got %s", PriorityLow)
	}
	if PriorityNone != "" {
		t.Errorf("PriorityNone should be empty, got %s", PriorityNone)
	}
}

func TestTaskContext(t *testing.T) {
	ctx := TaskContext{
		BrainName:  "main",
		BrainPath:  "/path/to/brain",
		FilePath:   "/path/to/brain/tasks/todo.md",
		FileName:   "todo.md",
		LineNumber: 42,
		Section:    "Today",
		ParentNote: "Todo List",
	}

	if ctx.BrainName != "main" {
		t.Errorf("BrainName mismatch")
	}
	if ctx.LineNumber != 42 {
		t.Errorf("LineNumber mismatch")
	}
}

func TestRecurringPattern(t *testing.T) {
	until := time.Now().AddDate(1, 0, 0)
	pattern := RecurringPattern{
		Interval: "weekly",
		Count:    2,
		Until:    &until,
	}

	if pattern.Interval != "weekly" {
		t.Errorf("Interval should be 'weekly', got %s", pattern.Interval)
	}
	if pattern.Count != 2 {
		t.Errorf("Count should be 2, got %d", pattern.Count)
	}
}
