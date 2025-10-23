package tasks

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// UpdateTaskInFile updates a task in its source file
func UpdateTaskInFile(filePath string, lineNum int, newTask *Task) error {
	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Validate line number
	if lineNum < 1 || lineNum > len(lines) {
		return fmt.Errorf("invalid line number: %d (file has %d lines)", lineNum, len(lines))
	}

	// Format the task back to markdown
	taskLine := FormatTask(newTask)

	// Check if we have multi-line task (with metadata)
	// We need to update both the task line and its metadata lines
	startLine := lineNum - 1
	endLine := startLine

	// Find the extent of this task (including indented metadata)
	for i := startLine + 1; i < len(lines); i++ {
		line := lines[i]
		// If line is empty or starts a new task/heading, stop
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "- [ ]") ||
			strings.HasPrefix(strings.TrimSpace(line), "- [x]") ||
			strings.HasPrefix(strings.TrimSpace(line), "- [/]") ||
			strings.HasPrefix(strings.TrimSpace(line), "- [-]") ||
			strings.HasPrefix(strings.TrimSpace(line), "#") {
			break
		}
		// If line is indented (metadata), include it
		if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
			endLine = i
		} else {
			break
		}
	}

	// Build new task block (task line + metadata lines)
	var newLines []string
	newLines = append(newLines, taskLine)

	// Add metadata lines if task has them
	if !newTask.Created.IsZero() {
		newLines = append(newLines, fmt.Sprintf("  created:: %s", newTask.Created.Format("2006-01-02 15:04")))
	}
	if newTask.Due != nil && !newTask.Due.IsZero() {
		newLines = append(newLines, fmt.Sprintf("  due:: %s", newTask.Due.Format("2006-01-02")))
	}
	if newTask.Priority != "" && newTask.Priority != "medium" {
		newLines = append(newLines, fmt.Sprintf("  priority:: %s", newTask.Priority))
	}
	if newTask.Status == StatusInProgress && newTask.Started != nil && !newTask.Started.IsZero() {
		newLines = append(newLines, fmt.Sprintf("  started:: %s", newTask.Started.Format("2006-01-02 15:04")))
	}
	if newTask.Status == StatusDone && newTask.Completed != nil && !newTask.Completed.IsZero() {
		newLines = append(newLines, fmt.Sprintf("  completed:: %s", newTask.Completed.Format("2006-01-02 15:04")))
	}
	if len(newTask.Tags) > 0 {
		newLines = append(newLines, fmt.Sprintf("  tags:: %s", strings.Join(newTask.Tags, ", ")))
	}
	if newTask.Project != "" {
		newLines = append(newLines, fmt.Sprintf("  project:: %s", newTask.Project))
	}

	// Replace the old task block with the new one
	result := append([]string{}, lines[:startLine]...)
	result = append(result, newLines...)
	result = append(result, lines[endLine+1:]...)

	// Write back to file
	output := strings.Join(result, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return os.WriteFile(filePath, []byte(output), 0644)
}

// ToggleTaskStatus toggles a task between open and done
func ToggleTaskStatus(task *Task) error {
	if task.Status == StatusDone {
		task.Status = StatusOpen
		task.Completed = nil
	} else {
		task.Status = StatusDone
		now := time.Now()
		task.Completed = &now
	}

	return UpdateTaskInFile(task.Context.FilePath, task.Context.LineNumber, task)
}

// SetTaskStatus sets a task to a specific status
func SetTaskStatus(task *Task, status Status) error {
	now := time.Now()

	switch status {
	case StatusOpen:
		task.Status = StatusOpen
		task.Started = nil
		task.Completed = nil
	case StatusInProgress:
		task.Status = StatusInProgress
		if task.Started == nil {
			task.Started = &now
		}
		task.Completed = nil
	case StatusDone:
		task.Status = StatusDone
		if task.Completed == nil {
			task.Completed = &now
		}
	case StatusDeferred:
		task.Status = StatusDeferred
	case StatusCancelled:
		task.Status = StatusCancelled
		task.Completed = &now
	}

	return UpdateTaskInFile(task.Context.FilePath, task.Context.LineNumber, task)
}

// UpdateTaskProperty updates a specific property of a task
func UpdateTaskProperty(task *Task, property string, value interface{}) error {
	switch property {
	case "description":
		if desc, ok := value.(string); ok {
			task.Description = desc
		}
	case "due":
		if due, ok := value.(*time.Time); ok {
			task.Due = due
		}
	case "priority":
		if priority, ok := value.(Priority); ok {
			task.Priority = priority
		}
	case "tags":
		if tags, ok := value.([]string); ok {
			task.Tags = tags
		}
	case "project":
		if project, ok := value.(string); ok {
			task.Project = project
		}
	default:
		return fmt.Errorf("unknown property: %s", property)
	}

	return UpdateTaskInFile(task.Context.FilePath, task.Context.LineNumber, task)
}

// FindTaskByID finds a task by its ID across all brains
func FindTaskByID(brainPaths map[string]string, taskID string) (*Task, error) {
	allTasks, err := ScanMultipleBrains(brainPaths)
	if err != nil {
		return nil, err
	}

	for i := range allTasks {
		task := &allTasks[i]
		if task.ID == taskID {
			return task, nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

// FindTasksByDescription finds tasks matching a description pattern
func FindTasksByDescription(brainPaths map[string]string, pattern string) ([]*Task, error) {
	allTasks, err := ScanMultipleBrains(brainPaths)
	if err != nil {
		return nil, err
	}

	pattern = strings.ToLower(pattern)
	var matches []*Task

	for i := range allTasks {
		task := &allTasks[i]
		if strings.Contains(strings.ToLower(task.Description), pattern) {
			matches = append(matches, task)
		}
	}

	return matches, nil
}

// BatchUpdateTasks updates multiple tasks at once
func BatchUpdateTasks(tasks []*Task, updateFn func(*Task) error) []error {
	var errors []error

	for _, task := range tasks {
		if err := updateFn(task); err != nil {
			errors = append(errors, fmt.Errorf("failed to update task %s: %w", task.ID, err))
		}
	}

	return errors
}

// ArchiveCompletedTasks moves completed tasks to an archive section
func ArchiveCompletedTasks(filePath string) error {
	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Separate completed and active tasks
	var activeLines []string
	var completedLines []string
	var inArchive bool

	for _, line := range lines {
		// Check for archive section
		if strings.Contains(line, "## ✅ Completed") || strings.Contains(line, "## Completed") {
			inArchive = true
			completedLines = append(completedLines, line)
			continue
		}

		// Check if we're leaving archive section
		if inArchive && strings.HasPrefix(strings.TrimSpace(line), "##") {
			inArchive = false
		}

		// Check if line is a completed task
		if strings.Contains(line, "- [x]") {
			completedLines = append(completedLines, line)
		} else {
			if inArchive {
				completedLines = append(completedLines, line)
			} else {
				activeLines = append(activeLines, line)
			}
		}
	}

	// Rebuild file with archive section at the end
	var result []string
	result = append(result, activeLines...)

	if len(completedLines) > 0 {
		// Add archive section if not already present
		hasArchiveHeader := false
		for _, line := range completedLines {
			if strings.Contains(line, "## ✅ Completed") || strings.Contains(line, "## Completed") {
				hasArchiveHeader = true
				break
			}
		}

		if !hasArchiveHeader {
			result = append(result, "", "## ✅ Completed", "")
		}

		result = append(result, completedLines...)
	}

	// Write back
	output := strings.Join(result, "\n")
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	return os.WriteFile(filePath, []byte(output), 0644)
}
