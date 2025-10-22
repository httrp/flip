package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/httrp/flip/internal/tasks"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// NewTaskNewCommand creates the task new command
func NewTaskNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new task",
		Long:  "Create a new task interactively.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateTask()
		},
	}
	return cmd
}

// runCreateTask creates a new task interactively
func runCreateTask() error {
	fmt.Println("\n📝 Create New Task")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Get task description
	descPrompt := promptui.Prompt{
		Label: "Task Description",
	}
	description, err := descPrompt.Run()
	if err != nil {
		return err
	}

	// Get due date
	duePrompt := promptui.Prompt{
		Label:   "Due Date (YYYY-MM-DD, 'today', 'tomorrow', or leave empty)",
		Default: "",
	}
	dueStr, _ := duePrompt.Run()
	dueDate := parseDueDateString(dueStr)

	// Get priority
	prioritySelect := promptui.Select{
		Label: "Priority",
		Items: []string{"High ⏫", "Medium 🔼", "Low 🔽", "None"},
	}
	priorityIdx, _, _ := prioritySelect.Run()
	priority := indexToPriority(priorityIdx)

	// Get tags
	tagsPrompt := promptui.Prompt{
		Label:   "Tags (comma-separated, e.g., work,urgent)",
		Default: "",
	}
	tagsStr, _ := tagsPrompt.Run()
	tagList := parseTagsString(tagsStr)

	// Select where to save
	locationSelect := promptui.Select{
		Label: "Where to save the task?",
		Items: []string{
			"📋 tasks/backlog.md",
			"📅 tasks/today.md",
			"📆 tasks/this-week.md",
			"📁 Specify file...",
		},
	}
	locIdx, _, _ := locationSelect.Run()

	// Get active brain
	activeBrain, err := getActiveBrain()
	if err != nil {
		return fmt.Errorf("failed to get active brain: %w", err)
	}

	var filePath string
	switch locIdx {
	case 0:
		filePath = filepath.Join(activeBrain.Path, "tasks", "backlog.md")
	case 1:
		filePath = filepath.Join(activeBrain.Path, "tasks", "today.md")
	case 2:
		filePath = filepath.Join(activeBrain.Path, "tasks", "this-week.md")
	case 3:
		filePrompt := promptui.Prompt{
			Label:   "File path (relative to brain root)",
			Default: "tasks/backlog.md",
		}
		relPath, _ := filePrompt.Run()
		filePath = filepath.Join(activeBrain.Path, relPath)
	}

	// Ensure tasks directory exists
	taskDir := filepath.Dir(filePath)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Create task object
	task := &tasks.Task{
		ID:          uuid.New().String(),
		Description: description,
		Status:      tasks.StatusOpen,
		Created:     time.Now(),
		Due:         dueDate,
		Priority:    priority,
		Tags:        tagList,
	}

	// Append to file
	err = appendTaskToFile(filePath, task)
	if err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	fmt.Printf("\n✅ Task created in %s\n", relativePathFromBrain(filePath, activeBrain.Path))
	fmt.Println()
	fmt.Println(tasks.FormatTask(task))
	fmt.Println()

	return nil
}

// appendTaskToFile appends a task to a markdown file
func appendTaskToFile(filePath string, task *tasks.Task) error {
	// Create file if it doesn't exist
	var content string
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Create new file with header
		content = fmt.Sprintf("# Tasks\n\nCreated: %s\n\n", time.Now().Format("2006-01-02"))
	} else {
		// Read existing content
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		content = string(data)
	}

	// Append task
	content += "\n" + tasks.FormatTask(task) + "\n"

	// Write back
	return os.WriteFile(filePath, []byte(content), 0644)
}

// Helper functions

func parseDueDateString(s string) *time.Time {
	if s == "" {
		return nil
	}

	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "today":
		t := time.Now()
		return &t
	case "tomorrow":
		t := time.Now().AddDate(0, 0, 1)
		return &t
	case "next week":
		t := time.Now().AddDate(0, 0, 7)
		return &t
	}

	// Try to parse as date
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}

	return nil
}

func indexToPriority(idx int) tasks.Priority {
	switch idx {
	case 0:
		return tasks.PriorityHigh
	case 1:
		return tasks.PriorityMedium
	case 2:
		return tasks.PriorityLow
	default:
		return tasks.PriorityNone
	}
}

func parseTagsString(s string) []string {
	if s == "" {
		return []string{}
	}

	tags := strings.Split(s, ",")
	var result []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		tag = strings.TrimPrefix(tag, "#")
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

func relativePathFromBrain(fullPath, brainPath string) string {
	rel, err := filepath.Rel(brainPath, fullPath)
	if err != nil {
		return fullPath
	}
	return rel
}
