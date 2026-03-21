package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/httrp/flip/internal/tasks"
	ui "github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// NewTaskDoneCommand creates the task done command
func NewTaskDoneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "done [search-pattern]",
		Short: "Mark a task as done",
		Long:  "Mark a task as completed. If no pattern provided, shows interactive selection.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskDone(args)
		},
	}
	return cmd
}

// runTaskDone handles the task done command
func runTaskDone(args []string) error {
	// Load workspace config
	config, err := loadWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load workspace: %w", err)
	}

	if config.ActiveWorkspace == "" {
		return fmt.Errorf("no active workspace")
	}

	// Find active workspace
	var activeWs *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWs = &config.Workspaces[i]
			break
		}
	}

	if activeWs == nil {
		return fmt.Errorf("active workspace not found")
	}

	// Scan all brains for open tasks
	brainPaths := make(map[string]string)
	for _, brain := range activeWs.Brains {
		brainPaths[brain.Name] = brain.Path
	}

	allTasks, err := tasks.ScanMultipleBrains(brainPaths)
	if err != nil {
		return fmt.Errorf("failed to scan tasks: %w", err)
	}

	// Filter to open tasks only
	var openTasks []*tasks.Task
	for i := range allTasks {
		if allTasks[i].Status == tasks.StatusOpen || allTasks[i].Status == tasks.StatusInProgress {
			openTasks = append(openTasks, &allTasks[i])
		}
	}

	if len(openTasks) == 0 {
		fmt.Println("✨ No open tasks found!")
		return nil
	}

	// Filter by search pattern if provided
	if len(args) > 0 {
		pattern := strings.ToLower(args[0])
		var filtered []*tasks.Task
		for _, task := range openTasks {
			if strings.Contains(strings.ToLower(task.Description), pattern) {
				filtered = append(filtered, task)
			}
		}
		openTasks = filtered

		if len(openTasks) == 0 {
			fmt.Printf("❌ No tasks found matching: %s\n", args[0])
			return nil
		}
	}

	// Interactive selection
	items := make([]string, len(openTasks))
	for i, task := range openTasks {
		priorityIcon := tasks.PriorityIcon(task.Priority)
		dueInfo := ""
		if task.Due != nil {
			dueInfo = fmt.Sprintf(" 📅 %s", task.Due.Format("2006-01-02"))
		}
		items[i] = fmt.Sprintf("%s %s%s | %s", priorityIcon, task.Description, dueInfo, task.Context.FileName)
	}

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
	}

	idx, _, err := ui.RunSelect("Select task to mark as done", selectItems, 10)
	if err != nil {
		return nil // User cancelled
	}

	selectedTask := openTasks[idx]

	// Mark as done
	if err := tasks.SetTaskStatus(selectedTask, tasks.StatusDone); err != nil {
		return fmt.Errorf("failed to mark task as done: %w", err)
	}

	fmt.Printf("\n✅ Task marked as done: %s\n", selectedTask.Description)
	fmt.Printf("   📄 File: %s (line %d)\n\n", selectedTask.Context.FilePath, selectedTask.Context.LineNumber)

	return nil
}

// NewTaskStartCommand creates the task start command
func NewTaskStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [search-pattern]",
		Short: "Mark a task as in-progress",
		Long:  "Mark a task as started/in-progress. If no pattern provided, shows interactive selection.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskStart(args)
		},
	}
	return cmd
}

// runTaskStart handles the task start command
func runTaskStart(args []string) error {
	// Load workspace config
	config, err := loadWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load workspace: %w", err)
	}

	if config.ActiveWorkspace == "" {
		return fmt.Errorf("no active workspace")
	}

	// Find active workspace
	var activeWs *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWs = &config.Workspaces[i]
			break
		}
	}

	if activeWs == nil {
		return fmt.Errorf("active workspace not found")
	}

	// Scan all brains for open tasks
	brainPaths := make(map[string]string)
	for _, brain := range activeWs.Brains {
		brainPaths[brain.Name] = brain.Path
	}

	allTasks, err := tasks.ScanMultipleBrains(brainPaths)
	if err != nil {
		return fmt.Errorf("failed to scan tasks: %w", err)
	}

	// Filter to open tasks only
	var openTasks []*tasks.Task
	for i := range allTasks {
		if allTasks[i].Status == tasks.StatusOpen {
			openTasks = append(openTasks, &allTasks[i])
		}
	}

	if len(openTasks) == 0 {
		fmt.Println("✨ No open tasks found!")
		return nil
	}

	// Filter by search pattern if provided
	if len(args) > 0 {
		pattern := strings.ToLower(args[0])
		var filtered []*tasks.Task
		for _, task := range openTasks {
			if strings.Contains(strings.ToLower(task.Description), pattern) {
				filtered = append(filtered, task)
			}
		}
		openTasks = filtered

		if len(openTasks) == 0 {
			fmt.Printf("❌ No tasks found matching: %s\n", args[0])
			return nil
		}
	}

	// Interactive selection
	items := make([]string, len(openTasks))
	for i, task := range openTasks {
		priorityIcon := tasks.PriorityIcon(task.Priority)
		dueInfo := ""
		if task.Due != nil {
			dueInfo = fmt.Sprintf(" 📅 %s", task.Due.Format("2006-01-02"))
		}
		items[i] = fmt.Sprintf("%s %s%s | %s", priorityIcon, task.Description, dueInfo, task.Context.FileName)
	}

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
	}

	idx, _, err := ui.RunSelect("Select task to start", selectItems, 10)
	if err != nil {
		return nil // User cancelled
	}

	selectedTask := openTasks[idx]

	// Mark as in-progress
	if err := tasks.SetTaskStatus(selectedTask, tasks.StatusInProgress); err != nil {
		return fmt.Errorf("failed to mark task as in-progress: %w", err)
	}

	fmt.Printf("\n🔄 Task started: %s\n", selectedTask.Description)
	fmt.Printf("   📄 File: %s (line %d)\n\n", selectedTask.Context.FilePath, selectedTask.Context.LineNumber)

	return nil
}

// NewTaskUpdateCommand creates the task update command
func NewTaskUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [search-pattern]",
		Short: "Update a task",
		Long:  "Update task properties interactively. If no pattern provided, shows interactive selection.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskUpdate(args)
		},
	}
	return cmd
}

// runTaskUpdate handles the task update command
func runTaskUpdate(args []string) error {
	// Load workspace config
	config, err := loadWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load workspace: %w", err)
	}

	if config.ActiveWorkspace == "" {
		return fmt.Errorf("no active workspace")
	}

	// Find active workspace
	var activeWs *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWs = &config.Workspaces[i]
			break
		}
	}

	if activeWs == nil {
		return fmt.Errorf("active workspace not found")
	}

	// Scan all brains
	brainPaths := make(map[string]string)
	for _, brain := range activeWs.Brains {
		brainPaths[brain.Name] = brain.Path
	}

	allTasks, err := tasks.ScanMultipleBrains(brainPaths)
	if err != nil {
		return fmt.Errorf("failed to scan tasks: %w", err)
	}

	// Convert to pointer slice
	var taskList []*tasks.Task
	for i := range allTasks {
		taskList = append(taskList, &allTasks[i])
	}

	if len(taskList) == 0 {
		fmt.Println("✨ No tasks found!")
		return nil
	}

	// Filter by search pattern if provided
	if len(args) > 0 {
		pattern := strings.ToLower(args[0])
		var filtered []*tasks.Task
		for _, task := range taskList {
			if strings.Contains(strings.ToLower(task.Description), pattern) {
				filtered = append(filtered, task)
			}
		}
		taskList = filtered

		if len(taskList) == 0 {
			fmt.Printf("❌ No tasks found matching: %s\n", args[0])
			return nil
		}
	}

	// Task selection
	items := make([]string, len(taskList))
	for i, task := range taskList {
		statusIcon := tasks.StatusIcon(task.Status)
		priorityIcon := tasks.PriorityIcon(task.Priority)
		dueInfo := ""
		if task.Due != nil {
			dueInfo = fmt.Sprintf(" � %s", task.Due.Format("2006-01-02"))
		}
		items[i] = fmt.Sprintf("%s %s %s%s | %s", statusIcon, priorityIcon, task.Description, dueInfo, task.Context.FileName)
	}

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
	}

	idx, _, err := ui.RunSelect("Select task to update", selectItems, 10)
	if err != nil {
		return nil // User cancelled
	}

	selectedTask := taskList[idx]

	// Property selection
	propertyItems := []ui.SelectItem{
		{Label: "📝 Description", Value: "description"},
		{Label: "📅 Due Date", Value: "due"},
		{Label: "⏫ Priority", Value: "priority"},
		{Label: "🏷️  Tags", Value: "tags"},
		{Label: "📂 Project", Value: "project"},
		{Label: "🔄 Status", Value: "status"},
		{Label: "❌ Cancel", Value: "cancel"},
	}

	_, propVal, err := ui.RunSelect("What would you like to update?", propertyItems, 10)
	if err != nil || propVal == "cancel" {
		return nil // User cancelled
	}

	switch propVal {
	case "description":
		newDesc, err := ui.RunInput("New description", "", selectedTask.Description, nil)
		if err != nil {
			return nil
		}
		selectedTask.Description = strings.TrimSpace(newDesc)

	case "due":
		dueDateStr, err := ui.RunInput("Due date (YYYY-MM-DD, 'today', 'tomorrow', 'next week', or leave empty to remove)", "", "", nil)
		if err != nil {
			return nil
		}
		dueDateStr = strings.TrimSpace(dueDateStr)

		if dueDateStr == "" {
			selectedTask.Due = nil
		} else {
			dueDate, err := parseDueDate(dueDateStr)
			if err != nil {
				return fmt.Errorf("invalid due date: %w", err)
			}
			selectedTask.Due = &dueDate
		}

	case "priority":
		prioItems := []ui.SelectItem{
			{Label: "⏫ High", Value: "high"},
			{Label: "🔼 Medium", Value: "medium"},
			{Label: "🔽 Low", Value: "low"},
			{Label: "   None", Value: "none"},
		}
		_, prioVal, err := ui.RunSelect("Select priority", prioItems, 5)
		if err != nil {
			return nil
		}
		selectedTask.Priority = parsePriorityString(prioVal)

	case "tags":
		currentTags := strings.Join(selectedTask.Tags, ", ")
		tagsStr, err := ui.RunInput("Tags (comma-separated)", "", currentTags, nil)
		if err != nil {
			return nil
		}
		tagsStr = strings.TrimSpace(tagsStr)
		if tagsStr == "" {
			selectedTask.Tags = nil
		} else {
			selectedTask.Tags = strings.Split(tagsStr, ",")
			for i := range selectedTask.Tags {
				selectedTask.Tags[i] = strings.TrimSpace(selectedTask.Tags[i])
			}
		}

	case "project":
		project, err := ui.RunInput("Project name", "", selectedTask.Project, nil)
		if err != nil {
			return nil
		}
		selectedTask.Project = strings.TrimSpace(project)

	case "status":
		statusItems := []ui.SelectItem{
			{Label: "⭕ Open", Value: "open"},
			{Label: "🔄 In Progress", Value: "in-progress"},
			{Label: "✅ Done", Value: "done"},
			{Label: "⏸️  Deferred", Value: "deferred"},
			{Label: "❌ Cancelled", Value: "cancelled"},
		}
		_, statusVal, err := ui.RunSelect("Select status", statusItems, 6)
		if err != nil {
			return nil
		}
		var newStatus tasks.Status
		switch statusVal {
		case "open":
			newStatus = tasks.StatusOpen
		case "in-progress":
			newStatus = tasks.StatusInProgress
		case "done":
			newStatus = tasks.StatusDone
		case "deferred":
			newStatus = tasks.StatusDeferred
		case "cancelled":
			newStatus = tasks.StatusCancelled
		}
		if err := tasks.SetTaskStatus(selectedTask, newStatus); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
		fmt.Printf("\n✅ Task status updated: %s\n", selectedTask.Description)
		fmt.Printf("   📄 File: %s (line %d)\n\n", selectedTask.Context.FilePath, selectedTask.Context.LineNumber)
		return nil
	}

	// Update the task in file
	if err := tasks.UpdateTaskInFile(selectedTask.Context.FilePath, selectedTask.Context.LineNumber, selectedTask); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	fmt.Printf("\n✅ Task updated: %s\n", selectedTask.Description)
	fmt.Printf("   📄 File: %s (line %d)\n\n", selectedTask.Context.FilePath, selectedTask.Context.LineNumber)

	return nil
}

// parseDueDate parses various date formats
func parseDueDate(input string) (time.Time, error) {
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "today":
		return time.Now().Truncate(24 * time.Hour), nil
	case "tomorrow":
		return time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour), nil
	case "next week":
		return time.Now().Add(7 * 24 * time.Hour).Truncate(24 * time.Hour), nil
	default:
		// Try parsing as YYYY-MM-DD
		parsed, err := time.Parse("2006-01-02", input)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid date format (use YYYY-MM-DD, 'today', 'tomorrow', or 'next week')")
		}
		return parsed, nil
	}
}

// NewTaskSearchCommand creates the task search command
func NewTaskSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [keywords...]",
		Short: "Search for tasks",
		Long:  "Search for tasks by description.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			filter := tasks.TaskFilter{
				SearchText: query,
				Status:     tasks.StatusOpen,
			}
			return runListTasks(filter, false)
		},
	}
	return cmd
}

// NewTaskStatsCommand creates the task stats command
func NewTaskStatsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show task statistics",
		Long:  "Display statistics about tasks across all brains.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskStats()
		},
	}
	return cmd
}

// runTaskStats shows task statistics
func runTaskStats() error {
	fmt.Println("\n📊 Task Statistics")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Load workspace config
	config, err := loadWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load workspace: %w", err)
	}

	if config.ActiveWorkspace == "" {
		return fmt.Errorf("no active workspace")
	}

	// Find active workspace
	var activeWs *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWs = &config.Workspaces[i]
			break
		}
	}

	if activeWs == nil {
		return fmt.Errorf("active workspace not found")
	}

	// Scan all brains
	brainPaths := make(map[string]string)
	for _, brain := range activeWs.Brains {
		brainPaths[brain.Name] = brain.Path
	}

	fmt.Println("\n🔍 Scanning tasks...")
	allTasks, err := tasks.ScanMultipleBrains(brainPaths)
	if err != nil {
		return fmt.Errorf("failed to scan tasks: %w", err)
	}

	// Build index and get stats
	index := tasks.NewTaskIndex()
	index.Build(allTasks)
	stats := index.Stats()

	displayStatusHeader()

	// Display statistics
	fmt.Printf("📋 Total Tasks:        %d\n\n", stats.Total)

	fmt.Println("Status Breakdown:")
	fmt.Printf("  ⭕ Open:             %d\n", stats.Open)
	fmt.Printf("  🔄 In Progress:      %d\n", stats.InProgress)
	fmt.Printf("  ✅ Done:             %d\n", stats.Done)
	fmt.Printf("  ⏸️  Deferred:         %d\n", stats.Deferred)
	fmt.Printf("  ❌ Cancelled:        %d\n\n", stats.Cancelled)

	fmt.Println("Due Dates:")
	fmt.Printf("  🚨 Overdue:          %d\n", stats.Overdue)
	fmt.Printf("  📅 Due Today:        %d\n", stats.DueToday)
	fmt.Printf("  📆 Due This Week:    %d\n\n", stats.DueThisWeek)

	// Completion rate
	if stats.Total > 0 {
		completionRate := float64(stats.Done) / float64(stats.Total) * 100
		fmt.Printf("✨ Completion Rate:   %.1f%%\n\n", completionRate)
	}

	return nil
}
