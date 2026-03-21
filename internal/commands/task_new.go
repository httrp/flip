package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/httrp/flip/internal/tasks"
	ui "github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// TaskNewOptions holds options for task creation
type TaskNewOptions struct {
	File        string // Target file for task (for VS Code integration)
	Line        int    // Line number to insert at (for VS Code integration)
	Description string // Task description (for non-interactive mode)
	Brain       string // Brain name
	Due         string // Due date (YYYY-MM-DD, today, tomorrow)
	Priority    string // Priority (high, medium, low)
	Frog        bool   // 🐸 Eat-the-frog task
	NoEdit      bool   // Don't open editor
	NoLink      bool   // Don't add link to journal
}

// TaskResult is the JSON response for task creation
type TaskResult struct {
	Action      string `json:"action"`
	Path        string `json:"path"`
	Line        int    `json:"line,omitempty"`
	Description string `json:"description"`
	Due         string `json:"due,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Frog        bool   `json:"frog,omitempty"`
	Status      string `json:"status,omitempty"`
	BrainName   string `json:"brain_name"`
	BrainPath   string `json:"brain_path"`
}

// NewTaskNewCommand creates the task new command
func NewTaskNewCommand() *cobra.Command {
	var opts TaskNewOptions
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "new",
		Aliases: []string{"n"},
		Short:   "Create a new task",
		Long: `Create a new task interactively or non-interactively.

Examples:
  flip task new                                    # Interactive mode
  flip task new --file /path/to/note.md --line 42 # Insert at cursor position
  flip task new --description "My task" --json    # Non-interactive with JSON output
  flip task new --description "Do X" --due today --priority high --json
  flip task new --description "Important task" --frog --json  # Eat-the-frog task`,
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			// Non-interactive mode if description is provided
			if opts.Description != "" || JSONOutput {
				return runCreateTaskNonInteractive(opts)
			}
			// Position-based mode
			if opts.File != "" {
				return runCreateTaskAtPosition(opts)
			}
			return runCreateTask()
		},
	}

	cmd.Flags().StringVar(&opts.File, "file", "", "Target file for task (VS Code integration)")
	cmd.Flags().IntVar(&opts.Line, "line", 0, "Line number to insert at (VS Code integration)")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Task description (for non-interactive mode)")
	cmd.Flags().StringVar(&opts.Brain, "brain", "", "Brain to use (default: active brain)")
	cmd.Flags().StringVar(&opts.Due, "due", "", "Due date (YYYY-MM-DD, today, tomorrow)")
	cmd.Flags().StringVar(&opts.Priority, "priority", "", "Priority (high, medium, low)")
	cmd.Flags().BoolVar(&opts.Frog, "frog", false, "🐸 Mark as eat-the-frog task")
	cmd.Flags().BoolVar(&opts.NoEdit, "no-edit", false, "Don't open editor after creation")
	cmd.Flags().BoolVar(&opts.NoLink, "no-link", false, "Don't add link to journal")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")

	return cmd
}

// runCreateTaskNonInteractive creates a task without prompts (for VS Code integration)
func runCreateTaskNonInteractive(opts TaskNewOptions) error {
	// Validate required fields
	if opts.Description == "" {
		err := fmt.Errorf("--description is required for non-interactive mode")
		if JSONOutput {
			OutputJSONError("task", err)
			return nil
		}
		return err
	}

	// Get workspace
	activeWs, err := getActiveWorkspace()
	if err != nil {
		if JSONOutput {
			OutputJSONError("task", err)
			return nil
		}
		return err
	}

	// Get brain
	var activeBrain *Brain
	if opts.Brain != "" {
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == opts.Brain {
				activeBrain = &activeWs.Brains[i]
				break
			}
		}
		if activeBrain == nil {
			err := fmt.Errorf("brain not found: %s", opts.Brain)
			if JSONOutput {
				OutputJSONError("task", err)
				return nil
			}
			return err
		}
	} else {
		// Use default brain
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == activeWs.DefaultBrain {
				activeBrain = &activeWs.Brains[i]
				break
			}
		}
		if activeBrain == nil && len(activeWs.Brains) > 0 {
			activeBrain = &activeWs.Brains[0]
		}
	}

	if activeBrain == nil {
		err := fmt.Errorf("no brain available")
		if JSONOutput {
			OutputJSONError("task", err)
			return nil
		}
		return err
	}

	// Parse due date
	var dueDate *time.Time
	if opts.Due != "" {
		dueDate = parseDueDateString(opts.Due)
	}

	// Create task
	task := &tasks.Task{
		ID:          uuid.New().String()[:8],
		Description: opts.Description,
		Status:      tasks.StatusOpen,
		Priority:    parsePriorityString(opts.Priority),
		Due:         dueDate,
		Created:     time.Now(),
		Frog:        opts.Frog,
	}

	// Get task directory
	taskDir := filepath.Join(activeBrain.Path, "tasks")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		if JSONOutput {
			OutputJSONError("task", err)
			return nil
		}
		return err
	}

	// Target file
	var filePath string
	if opts.File != "" {
		filePath = opts.File
	} else {
		// Default tasks file
		filePath = filepath.Join(taskDir, "todo.md")
	}

	// Append task to file
	if err := appendTaskToFile(filePath, task); err != nil {
		if JSONOutput {
			OutputJSONError("task", err)
			return nil
		}
		return err
	}

	// Add link to journal (unless disabled)
	if !opts.NoLink {
		relPath := relativePathFromBrain(filePath, activeBrain.Path)
		if err := AddLinkToJournal(JournalLinkOptions{
			ItemType:    "task",
			ItemName:    opts.Description,
			ItemPath:    relPath,
			Brain:       activeBrain,
			Interactive: false,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "\u26a0\ufe0f  journal link failed: %v\n", err)
		}
	}

	// Output result
	if JSONOutput {
		result := TaskResult{
			Action:      "created",
			Path:        filePath,
			Description: opts.Description,
			BrainName:   activeBrain.Name,
			BrainPath:   activeBrain.Path,
		}
		if dueDate != nil {
			result.Due = dueDate.Format("2006-01-02")
		}
		if opts.Priority != "" {
			result.Priority = opts.Priority
		}
		OutputJSONSuccess("task", result)
		return nil
	}

	fmt.Printf("✅ Task created: %s\n", filePath)

	// Open editor if not disabled
	if !opts.NoEdit {
		_ = openInEditor(filePath)
	}

	return nil
}

// parsePriorityString converts a priority string to tasks.Priority
func parsePriorityString(s string) tasks.Priority {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "high", "h", "1":
		return tasks.PriorityHigh
	case "medium", "med", "m", "2":
		return tasks.PriorityMedium
	case "low", "l", "3":
		return tasks.PriorityLow
	default:
		return tasks.PriorityNone
	}
}

// runCreateTaskAtPosition creates a task at a specific file/line (for VS Code integration)
func runCreateTaskAtPosition(opts TaskNewOptions) error {
	fmt.Println("\n📝 Create Task at Current Position")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📄 File: %s\n", opts.File)
	if opts.Line > 0 {
		fmt.Printf("📍 Line: %d\n", opts.Line)
	}
	fmt.Println()

	// Verify file exists
	if _, err := os.Stat(opts.File); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", opts.File)
	}

	// Get task description
	description, err := ui.RunInput("Task Description", "", "", nil)
	if err != nil {
		return err
	}

	// Get due date (simplified for cursor-based insertion)
	dueStr, _ := ui.RunInput("Due Date (YYYY-MM-DD, 'today', 'tomorrow', or leave empty)", "", "", nil)
	dueDate := parseDueDateString(dueStr)

	// Get priority
	priorityItems := []ui.SelectItem{
		{Label: "High ⏫", Value: "high"},
		{Label: "Medium 🔼", Value: "medium"},
		{Label: "Low 🔽", Value: "low"},
		{Label: "None", Value: "none"},
	}
	_, priorityVal, _ := ui.RunSelect("Priority", priorityItems, 5)
	priority := parsePriorityString(priorityVal)

	// Create task object
	task := &tasks.Task{
		ID:          uuid.New().String(),
		Description: description,
		Status:      tasks.StatusOpen,
		Created:     time.Now(),
		Due:         dueDate,
		Priority:    priority,
	}

	// Insert at position or append
	if opts.Line > 0 {
		err = insertTaskAtLine(opts.File, opts.Line, task)
	} else {
		err = appendTaskToFile(opts.File, task)
	}
	if err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	fmt.Printf("\n✅ Task inserted at %s", opts.File)
	if opts.Line > 0 {
		fmt.Printf(":%d", opts.Line)
	}
	fmt.Println()
	fmt.Println(tasks.FormatTask(task))
	fmt.Println()

	return nil
}

// insertTaskAtLine inserts a task at a specific line in a file
func insertTaskAtLine(filePath string, lineNum int, task *tasks.Task) error {
	// Read existing content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")

	// Format task
	taskLine := tasks.FormatTask(task)

	// Ensure line number is valid
	if lineNum < 1 {
		lineNum = 1
	}
	if lineNum > len(lines) {
		// Append to end
		lines = append(lines, "", taskLine)
	} else {
		// Insert at line (1-indexed)
		idx := lineNum - 1
		newLines := make([]string, 0, len(lines)+2)
		newLines = append(newLines, lines[:idx]...)
		newLines = append(newLines, taskLine, "")
		newLines = append(newLines, lines[idx:]...)
		lines = newLines
	}

	// Write back
	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
}

// runCreateTask creates a new task interactively
func runCreateTask() error {
	fmt.Println("\n📝 Create New Task")

	// Get task description
	description, err := ui.RunInput("Task Description", "", "", nil)
	if err != nil {
		return err
	}

	// Get due date
	dueStr, _ := ui.RunInput("Due Date (YYYY-MM-DD, 'today', 'tomorrow', or leave empty)", "", "", nil)
	dueDate := parseDueDateString(dueStr)

	// Get priority
	priorityItems := []ui.SelectItem{
		{Label: "High ⏫", Value: "high"},
		{Label: "Medium 🔼", Value: "medium"},
		{Label: "Low 🔽", Value: "low"},
		{Label: "None", Value: "none"},
	}
	_, priorityVal, _ := ui.RunSelect("Priority", priorityItems, 5)
	priority := parsePriorityString(priorityVal)

	// Get organization
	organization, err := promptForOrganization()
	if err != nil {
		return err
	}

	// Get project
	project, err := promptForProject()
	if err != nil {
		return err
	}

	// Get context
	context, err := promptForContext()
	if err != nil {
		return err
	}

	// Get tags
	tagsStr, _ := ui.RunInput("Tags (comma-separated, e.g., work,urgent)", "", "", nil)
	tagList := parseTagsString(tagsStr)

	// Get active brain and allow selection if multiple brains exist
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get active workspace: %w", err)
	}

	if len(activeWs.Brains) == 0 {
		return fmt.Errorf("no brains configured")
	}

	// Use confirmOrSelectBrain to allow selection if multiple brains
	activeBrain, err := confirmOrSelectBrain(activeWs)
	if err != nil {
		return err
	}

	// Select where to save with improved options
	filePath, err := promptForTaskFile(activeBrain)
	if err != nil {
		return err
	}

	// Ensure tasks directory exists
	taskDir := filepath.Dir(filePath)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Create task object
	task := &tasks.Task{
		ID:           uuid.New().String(),
		Description:  description,
		Status:       tasks.StatusOpen,
		Created:      time.Now(),
		Due:          dueDate,
		Priority:     priority,
		Tags:         tagList,
		Organization: organization,
		Project:      project,
		ContextTag:   context,
	}

	// Append to file
	err = appendTaskToFile(filePath, task)
	if err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// Update task history
	relPath := relativePathFromBrain(filePath, activeBrain.Path)
	if err := updateTaskHistory(organization, project, context, relPath); err != nil {
		// Don't fail on history update error, just log
		fmt.Printf("⚠️  Warning: Could not update task history: %v\n", err)
	}

	fmt.Printf("\n✅ Task created in %s\n", relPath)
	fmt.Println()
	fmt.Println(tasks.FormatTask(task))
	fmt.Println()

	// Ask if user wants to add link to journal
	if err := AddLinkToJournal(JournalLinkOptions{
		ItemType:    "task",
		ItemName:    description,
		ItemPath:    relPath,
		Brain:       activeBrain,
		Interactive: true,
	}); err != nil {
		fmt.Printf("⚠️  Could not add journal link: %v\n", err)
	}

	return nil
}

// appendTaskToFile appends a task to a markdown file
func appendTaskToFile(filePath string, task *tasks.Task) error {
	var content string
	var isNewFile bool

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		isNewFile = true
		// Check if this is the new unified tasks.md file
		if strings.HasSuffix(filePath, "tasks/tasks.md") {
			content = createUnifiedTasksTemplate()
		} else {
			// Legacy format for other files
			content = fmt.Sprintf("# Tasks\n\nCreated: %s\n\n", time.Now().Format("2006-01-02"))
		}
	} else {
		// Read existing content
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		content = string(data)
	}

	// For unified tasks.md, append to appropriate section
	if strings.HasSuffix(filePath, "tasks/tasks.md") {
		content = appendToUnifiedTasks(content, task)
	} else {
		// Legacy: just append to end
		if !isNewFile {
			content += "\n"
		}
		content += tasks.FormatTask(task) + "\n"
	}

	// Write back
	return os.WriteFile(filePath, []byte(content), 0644)
}

// createUnifiedTasksTemplate creates the template for the new unified tasks.md
func createUnifiedTasksTemplate() string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`# Tasks

Created: %s

## 🎯 Today

<!-- Tasks for today -->

## 📅 This Week

<!-- Tasks for this week -->

## 📋 Backlog

<!-- All other tasks -->

## ✅ Completed

<!-- Completed tasks (archive) -->

`, today)
}

// appendToUnifiedTasks appends a task to the appropriate section in unified tasks.md
func appendToUnifiedTasks(content string, task *tasks.Task) string {
	// Determine which section based on due date
	var section string
	if task.Due != nil {
		today := time.Now().Truncate(24 * time.Hour)
		dueDate := task.Due.Truncate(24 * time.Hour)

		if dueDate.Equal(today) {
			section = "## 🎯 Today"
		} else if dueDate.Before(today.Add(7 * 24 * time.Hour)) {
			section = "## 📅 This Week"
		} else {
			section = "## 📋 Backlog"
		}
	} else {
		section = "## 📋 Backlog"
	}

	// Find the section and insert after it
	lines := strings.Split(content, "\n")
	sectionIdx := -1

	for i, line := range lines {
		if strings.TrimSpace(line) == section {
			sectionIdx = i
			break
		}
	}

	if sectionIdx == -1 {
		// Section not found, append to end
		return content + "\n" + tasks.FormatTask(task) + "\n"
	}

	// Find insertion point (after section header and comment)
	insertIdx := sectionIdx + 1
	// Skip comments (but not empty lines yet)
	for insertIdx < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[insertIdx]), "<!--") {
		insertIdx++
	}

	// Now skip ALL empty lines after the comment to clean up spacing
	for insertIdx < len(lines) && strings.TrimSpace(lines[insertIdx]) == "" {
		insertIdx++
	}

	// Insert task with proper spacing: 1 empty line before, 1 empty line after
	taskLine := tasks.FormatTask(task)
	newLines := append(lines[:insertIdx], append([]string{"", taskLine, ""}, lines[insertIdx:]...)...)

	return strings.Join(newLines, "\n")
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

// getRecentlyModifiedMdFiles returns the most recently modified .md files in a brain
func getRecentlyModifiedMdFiles(brainPath string, limit int) []string {
	type fileInfo struct {
		relPath string
		modTime time.Time
	}

	var files []fileInfo

	// Walk the brain directory
	filepath.Walk(brainPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories and hidden files/folders
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only .md files
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(brainPath, path)
		if err != nil {
			return nil
		}

		// Skip .git and node_modules
		if strings.Contains(relPath, ".git") || strings.Contains(relPath, "node_modules") {
			return nil
		}

		files = append(files, fileInfo{
			relPath: relPath,
			modTime: info.ModTime(),
		})

		return nil
	})

	// Sort by modification time (newest first)
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].modTime.After(files[i].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// Return top N
	result := make([]string, 0, limit)
	for i := 0; i < len(files) && i < limit; i++ {
		result = append(result, files[i].relPath)
	}

	return result
}
