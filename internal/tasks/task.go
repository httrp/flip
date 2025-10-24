package tasks

import (
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Status represents the current state of a task
type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in-progress"
	StatusDone       Status = "done"
	StatusDeferred   Status = "deferred"
	StatusCancelled  Status = "cancelled"
)

// Priority represents task priority level
type Priority string

const (
	PriorityHigh   Priority = "high"   // ⏫
	PriorityMedium Priority = "medium" // 🔼
	PriorityLow    Priority = "low"    // 🔽
	PriorityNone   Priority = ""
)

// Task represents a single task with all metadata
type Task struct {
	ID          string // Unique identifier (hash-based)
	Description string // Task text
	Status      Status // Current status

	// Timestamps
	Created   time.Time
	Started   *time.Time
	Due       *time.Time
	Completed *time.Time

	// Priority & Effort
	Priority Priority
	Effort   string // "2h", "1d", "3w"

	// Categorization
	Tags         []string
	Project      string // [[Project Name]] or proj:NAME
	Organization string // [ORG] or org:NAME
	ContextTag   string // ctx:VALUE or [ORG:CTX] - business context/area
	Assigned     string

	// Location Context - where this task lives (file location)
	Context TaskContext

	// Relationships
	DependsOn []string // Task IDs this task depends on
	Blocks    []string // Task IDs this task blocks
	RelatedTo []string // Related note links

	// Additional metadata
	Recurring *RecurringPattern
	Notes     string // Additional notes below task
}

// TaskContext stores information about where a task is located
type TaskContext struct {
	BrainName  string // Which brain contains this task
	BrainPath  string // Brain root path
	FilePath   string // Full file path
	FileName   string // Base filename
	LineNumber int    // Line number in file
	Section    string // Heading context (if under a heading)
	ParentNote string // Note title/name
}

// RecurringPattern defines how a task repeats
type RecurringPattern struct {
	Interval string     // "daily", "weekly", "monthly", "yearly"
	Count    int        // Every N intervals
	Until    *time.Time // Stop recurring after this date
}

// GenerateID creates a unique ID for a task based on its context and content
func GenerateID(description string, context TaskContext, lineNum int) string {
	// Use file path + line number + description for uniqueness
	data := fmt.Sprintf("%s:%d:%s", context.FilePath, lineNum, description)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes for shorter ID
}

// IsOverdue returns true if the task has a due date in the past
func (t *Task) IsOverdue() bool {
	if t.Due == nil || t.Status == StatusDone || t.Status == StatusCancelled {
		return false
	}
	return t.Due.Before(time.Now())
}

// IsDueToday returns true if the task is due today
func (t *Task) IsDueToday() bool {
	if t.Due == nil {
		return false
	}
	now := time.Now()
	dueDate := t.Due
	return dueDate.Year() == now.Year() &&
		dueDate.Month() == now.Month() &&
		dueDate.Day() == now.Day()
}

// IsDueThisWeek returns true if the task is due within the next 7 days
func (t *Task) IsDueThisWeek() bool {
	if t.Due == nil {
		return false
	}
	weekFromNow := time.Now().AddDate(0, 0, 7)
	return t.Due.Before(weekFromNow) && !t.IsOverdue()
}

// Duration returns the time spent on the task (if completed)
func (t *Task) Duration() time.Duration {
	if t.Started == nil || t.Completed == nil {
		return 0
	}
	return t.Completed.Sub(*t.Started)
}

// StatusFromCheckbox converts a checkbox marker to Status
func StatusFromCheckbox(marker string) Status {
	switch marker {
	case " ", "":
		return StatusOpen
	case "x", "X":
		return StatusDone
	case "/", "~":
		return StatusInProgress
	case ">":
		return StatusDeferred
	case "-":
		return StatusCancelled
	default:
		return StatusOpen
	}
}

// CheckboxFromStatus converts a Status to checkbox marker
func CheckboxFromStatus(status Status) string {
	switch status {
	case StatusOpen:
		return " "
	case StatusDone:
		return "x"
	case StatusInProgress:
		return "/"
	case StatusDeferred:
		return ">"
	case StatusCancelled:
		return "-"
	default:
		return " "
	}
}

// PriorityIcon returns the emoji icon for a priority
func PriorityIcon(priority Priority) string {
	switch priority {
	case PriorityHigh:
		return "⏫"
	case PriorityMedium:
		return "🔼"
	case PriorityLow:
		return "🔽"
	default:
		return ""
	}
}

// PriorityFromIcon converts an emoji icon to Priority
func PriorityFromIcon(icon string) Priority {
	switch icon {
	case "⏫":
		return PriorityHigh
	case "🔼":
		return PriorityMedium
	case "🔽":
		return PriorityLow
	default:
		return PriorityNone
	}
}

// SaveToFile updates the task's status in its markdown file
func (t *Task) SaveToFile() error {
	// Read file
	content, err := os.ReadFile(t.Context.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")

	// Validate line number
	if t.Context.LineNumber < 1 || t.Context.LineNumber > len(lines) {
		return fmt.Errorf("invalid line number: %d (file has %d lines)", t.Context.LineNumber, len(lines))
	}

	// Update the task line with new status
	oldLine := lines[t.Context.LineNumber-1]
	newLine := updateTaskStatusInLine(oldLine, t.Status)
	lines[t.Context.LineNumber-1] = newLine

	// Write back to file
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(t.Context.FilePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// updateTaskStatusInLine replaces the status marker in a task line
func updateTaskStatusInLine(line string, newStatus Status) string {
	// Determine new marker
	var newMarker string
	switch newStatus {
	case StatusOpen:
		newMarker = "[ ]"
	case StatusDone:
		newMarker = "[x]"
	case StatusInProgress:
		newMarker = "[>]"
	case StatusDeferred:
		newMarker = "[~]"
	case StatusCancelled:
		newMarker = "[-]"
	default:
		newMarker = "[ ]"
	}

	// Find and replace checkbox marker
	// Pattern: - [ ], - [x], - [>], etc. (also supports * and + list markers)
	re := regexp.MustCompile(`^(\s*[-*+]\s+)\[.\]`)
	return re.ReplaceAllString(line, "${1}"+newMarker)
}
