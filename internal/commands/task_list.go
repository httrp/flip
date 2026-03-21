package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/httrp/flip/internal/tasks"
	ui "github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// NewTaskListCommand creates the task list command
func NewTaskListCommand() *cobra.Command {
	var (
		statusFilter   string
		priorityFilter string
		tagFilter      string
		projectFilter  string
		orgFilter      string
		contextFilter  string
		dueToday       bool
		dueThisWeek    bool
		overdue        bool
		groupByFile    bool
		frogOnly       bool
		jsonOutput     bool // Add JSON output flag
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Long:  "List and filter tasks from all brains in the active workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := tasks.TaskFilter{
				DueToday:    dueToday,
				DueThisWeek: dueThisWeek,
				Overdue:     overdue,
				FrogOnly:    frogOnly,
			}

			if statusFilter != "" {
				filter.Status = tasks.Status(statusFilter)
			}
			if priorityFilter != "" {
				filter.Priority = tasks.Priority(priorityFilter)
			}
			if tagFilter != "" {
				filter.Tags = []string{tagFilter}
			}
			if projectFilter != "" {
				filter.Project = projectFilter
			}
			if orgFilter != "" {
				filter.Organization = orgFilter
			}
			if contextFilter != "" {
				filter.Context = contextFilter
			}

			// Use JSON output if requested
			if jsonOutput {
				return runListTasksJSON(filter)
			}

			return runListTasks(filter, groupByFile)
		},
	}

	cmd.Flags().StringVar(&statusFilter, "status", "", "Filter by status (open, in-progress, done)")
	cmd.Flags().StringVar(&priorityFilter, "priority", "", "Filter by priority (high, medium, low)")
	cmd.Flags().StringVar(&tagFilter, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&projectFilter, "project", "", "Filter by project")
	cmd.Flags().StringVar(&orgFilter, "org", "", "Filter by organization")
	cmd.Flags().StringVar(&contextFilter, "context", "", "Filter by context")
	cmd.Flags().BoolVar(&dueToday, "today", false, "Show tasks due today")
	cmd.Flags().BoolVar(&dueThisWeek, "week", false, "Show tasks due this week")
	cmd.Flags().BoolVar(&overdue, "overdue", false, "Show overdue tasks")
	cmd.Flags().BoolVarP(&groupByFile, "group", "g", false, "Group tasks by file")
	cmd.Flags().BoolVar(&frogOnly, "frog", false, "Show only 🐸 eat-the-frog tasks")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output tasks as JSON")

	return cmd
}

// runListTasks lists and filters tasks
func runListTasks(filter tasks.TaskFilter, groupByFile bool) error {
	fmt.Println("\n📋 Task List")
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

	if len(activeWs.Brains) == 0 {
		return fmt.Errorf("no brains in workspace")
	}

	// Scan all brains for tasks
	fmt.Println("\n🔍 Scanning for tasks...")
	brainPaths := make(map[string]string)
	for _, brain := range activeWs.Brains {
		brainPaths[brain.Name] = brain.Path
	}

	allTasks, err := tasks.ScanMultipleBrains(brainPaths)
	if err != nil {
		return fmt.Errorf("failed to scan tasks: %w", err)
	}

	if len(allTasks) == 0 {
		fmt.Println("\n✨ No tasks found!")
		return nil
	}

	// Build index and query
	index := tasks.NewTaskIndex()
	index.Build(allTasks)

	// Apply default filter if none specified
	if filter.Status == "" && !filter.DueToday && !filter.DueThisWeek && !filter.Overdue {
		filter.Status = tasks.StatusOpen
	}

	results := index.Query(filter)

	if len(results) == 0 {
		fmt.Println("\n✨ No tasks matching filter!")
		return nil
	}

	// Group tasks for display
	overdueTasks := filterOverdue(results)
	todayTasks := filterDueToday(results)
	weekTasks := filterDueThisWeek(results)
	laterTasks := filterLater(results)

	// Display grouped tasks
	displayStatusHeader()

	totalShown := 0

	// Choose display mode based on flag
	if groupByFile {
		// Group by file display
		fmt.Printf("📋 Tasks grouped by file (%d total)\n", len(results))
		displayTasksGroupedByFile(results)
		totalShown = len(results)
	} else {
		// Original time-based grouping
		if len(overdueTasks) > 0 {
			fmt.Printf("🚨 Overdue (%d)\n", len(overdueTasks))
			displayTaskList(overdueTasks)
			fmt.Println()
			totalShown += len(overdueTasks)
		}

		if len(todayTasks) > 0 {
			fmt.Printf("📅 Due Today (%d)\n", len(todayTasks))
			displayTaskList(todayTasks)
			fmt.Println()
			totalShown += len(todayTasks)
		}

		if len(weekTasks) > 0 {
			fmt.Printf("📆 This Week (%d)\n", len(weekTasks))
			displayTaskList(weekTasks)
			fmt.Println()
			totalShown += len(weekTasks)
		}

		if len(laterTasks) > 0 {
			limit := 10
			if len(laterTasks) > limit {
				fmt.Printf("📌 Later (showing %d of %d)\n", limit, len(laterTasks))
				displayTaskList(laterTasks[:limit])
			} else {
				fmt.Printf("📌 Later (%d)\n", len(laterTasks))
				displayTaskList(laterTasks)
			}
			fmt.Println()
			totalShown += len(laterTasks)
		}
	}

	fmt.Printf("\nTotal: %d tasks\n\n", totalShown)

	// Interactive selection
	if totalShown > 0 {
		return selectAndActOnTask(results)
	}

	return nil
}

// displayTaskList displays a list of tasks
func displayTaskList(taskList []*tasks.Task) {
	for _, task := range taskList {
		icon := tasks.PriorityIcon(task.Priority)
		if icon == "" {
			icon = "  "
		}

		// Organization/Project/Context metadata
		metaStr := ""
		if task.Organization != "" {
			metaStr = fmt.Sprintf("[%s]", task.Organization)
		}
		if task.Project != "" {
			if metaStr != "" {
				metaStr += " "
			}
			metaStr += fmt.Sprintf("proj:%s", task.Project)
		}
		if task.ContextTag != "" {
			if metaStr != "" {
				metaStr += " "
			}
			metaStr += fmt.Sprintf("ctx:%s", task.ContextTag)
		}

		dueStr := ""
		if task.Due != nil {
			dueStr = formatTaskDueDate(task.Due)
		}

		tags := ""
		if len(task.Tags) > 0 {
			tags = fmt.Sprintf("#%s", task.Tags[0])
			if len(task.Tags) > 1 {
				tags += fmt.Sprintf(" +%d", len(task.Tags)-1)
			}
		}

		brain := ""
		if task.Context.BrainName != "" {
			brain = fmt.Sprintf("[%s]", task.Context.BrainName)
		}

		// Build output line
		parts := []string{icon, truncate(task.Description, 50)}
		if metaStr != "" {
			parts = append(parts, metaStr)
		}
		if dueStr != "" {
			parts = append(parts, dueStr)
		}
		if tags != "" {
			parts = append(parts, tags)
		}
		if brain != "" {
			parts = append(parts, brain)
		}

		fmt.Printf("  %s\n", strings.Join(parts, " "))
	}
}

// selectAndActOnTask shows interactive menu for task actions
func selectAndActOnTask(taskList []*tasks.Task) error {
	if len(taskList) == 0 {
		return nil
	}

	// First, let user select a task
	items := make([]string, len(taskList))
	for i, task := range taskList {
		statusIcon := tasks.StatusIcon(task.Status)
		priorityIcon := tasks.PriorityIcon(task.Priority)
		frogIcon := ""
		if task.Frog {
			frogIcon = "🐸 "
		}
		dueInfo := ""
		if task.Due != nil {
			dueInfo = fmt.Sprintf(" %s", formatTaskDueDate(task.Due))
		}
		items[i] = fmt.Sprintf("%s %s %s%s%s | %s", statusIcon, priorityIcon, frogIcon, task.Description, dueInfo, task.Context.FileName)
	}

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
	}

	taskIdx, _, err := ui.RunSelect("Select a task", selectItems, 10)
	if err != nil {
		return nil // User cancelled
	}

	selectedTask := taskList[taskIdx]

	// Then show action menu
	actionItems := []ui.SelectItem{
		{Label: "📝 Open task file in editor", Value: "open"},
		{Label: "✅ Mark task as done", Value: "done"},
		{Label: "🔄 Mark as in-progress", Value: "in-progress"},
		{Label: "⭕ Mark as open", Value: "mark-open"},
		{Label: "📋 Update task properties", Value: "update"},
		{Label: "◀️  Back", Value: "back"},
	}

	_, actionVal, err := ui.RunSelect("What would you like to do?", actionItems, 7)
	if err != nil {
		return nil
	}

	switch actionVal {
	case "open":
		return openInEditor(selectedTask.Context.FilePath)

	case "done":
		if err := tasks.SetTaskStatus(selectedTask, tasks.StatusDone); err != nil {
			return fmt.Errorf("failed to mark task as done: %w", err)
		}
		fmt.Printf("\n✅ Task marked as done: %s\n", selectedTask.Description)
		fmt.Printf("   📄 File: %s (line %d)\n\n", selectedTask.Context.FilePath, selectedTask.Context.LineNumber)

	case "in-progress":
		if err := tasks.SetTaskStatus(selectedTask, tasks.StatusInProgress); err != nil {
			return fmt.Errorf("failed to mark task as in-progress: %w", err)
		}
		fmt.Printf("\n🔄 Task started: %s\n", selectedTask.Description)
		fmt.Printf("   📄 File: %s (line %d)\n\n", selectedTask.Context.FilePath, selectedTask.Context.LineNumber)

	case "mark-open":
		if err := tasks.SetTaskStatus(selectedTask, tasks.StatusOpen); err != nil {
			return fmt.Errorf("failed to mark task as open: %w", err)
		}
		fmt.Printf("\n⭕ Task marked as open: %s\n", selectedTask.Description)
		fmt.Printf("   📄 File: %s (line %d)\n\n", selectedTask.Context.FilePath, selectedTask.Context.LineNumber)

	case "update":
		return runTaskUpdateInteractive(selectedTask)

	case "back":
		return nil
	}

	return nil
}

// runTaskUpdateInteractive updates a task interactively
func runTaskUpdateInteractive(task *tasks.Task) error {
	propertyItems := []ui.SelectItem{
		{Label: "📝 Description", Value: "description"},
		{Label: "📅 Due Date", Value: "due"},
		{Label: "⏫ Priority", Value: "priority"},
		{Label: "🏷️  Tags", Value: "tags"},
		{Label: "📂 Project", Value: "project"},
		{Label: "❌ Cancel", Value: "cancel"},
	}

	_, propVal, err := ui.RunSelect("What would you like to update?", propertyItems, 7)
	if err != nil || propVal == "cancel" {
		return nil // User cancelled
	}

	switch propVal {
	case "description":
		newDesc, err := ui.RunInput("New description", "", task.Description, nil)
		if err != nil {
			return nil
		}
		task.Description = strings.TrimSpace(newDesc)

	case "due":
		dueDateStr, err := ui.RunInput("Due date (YYYY-MM-DD, 'today', 'tomorrow', 'next week', or leave empty to remove)", "", "", nil)
		if err != nil {
			return nil
		}
		dueDateStr = strings.TrimSpace(dueDateStr)

		if dueDateStr == "" {
			task.Due = nil
		} else {
			dueDate, err := parseDueDateFromString(dueDateStr)
			if err != nil {
				return fmt.Errorf("invalid due date: %w", err)
			}
			task.Due = &dueDate
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
		task.Priority = parsePriorityString(prioVal)

	case "tags":
		currentTags := ""
		if len(task.Tags) > 0 {
			currentTags = "#" + strings.Join(task.Tags, " #")
		}
		tagsStr, err := ui.RunInput("Tags (space-separated, with #)", "", currentTags, nil)
		if err != nil {
			return nil
		}
		tagsStr = strings.TrimSpace(tagsStr)
		if tagsStr == "" {
			task.Tags = nil
		} else {
			// Parse #tag1 #tag2 format
			task.Tags = nil
			for _, tag := range strings.Fields(tagsStr) {
				tag = strings.TrimPrefix(tag, "#")
				if tag != "" {
					task.Tags = append(task.Tags, tag)
				}
			}
		}

	case "project":
		project, err := ui.RunInput("Project name", "", task.Project, nil)
		if err != nil {
			return nil
		}
		task.Project = strings.TrimSpace(project)
	}

	// Update the task in file
	if err := tasks.UpdateTaskInFile(task.Context.FilePath, task.Context.LineNumber, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	fmt.Printf("\n✅ Task updated: %s\n", task.Description)
	fmt.Printf("   📄 File: %s (line %d)\n\n", task.Context.FilePath, task.Context.LineNumber)

	return nil
}

// parseDueDateFromString parses various date formats
func parseDueDateFromString(input string) (time.Time, error) {
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

// Helper functions for filtering

func filterOverdue(taskList []*tasks.Task) []*tasks.Task {
	var result []*tasks.Task
	for _, task := range taskList {
		if task.IsOverdue() && task.Status != tasks.StatusDone {
			result = append(result, task)
		}
	}
	return result
}

func filterDueToday(taskList []*tasks.Task) []*tasks.Task {
	var result []*tasks.Task
	for _, task := range taskList {
		if task.IsDueToday() && !task.IsOverdue() {
			result = append(result, task)
		}
	}
	return result
}

func filterDueThisWeek(taskList []*tasks.Task) []*tasks.Task {
	var result []*tasks.Task
	for _, task := range taskList {
		if task.IsDueThisWeek() && !task.IsDueToday() && !task.IsOverdue() {
			result = append(result, task)
		}
	}
	return result
}

func filterLater(taskList []*tasks.Task) []*tasks.Task {
	var result []*tasks.Task
	for _, task := range taskList {
		if !task.IsOverdue() && !task.IsDueToday() && !task.IsDueThisWeek() {
			result = append(result, task)
		}
	}
	return result
}

// formatTaskDueDate formats a due date for display
func formatTaskDueDate(t *time.Time) string {
	if t == nil {
		return ""
	}

	now := time.Now()
	diff := t.Sub(now)

	if diff < 0 {
		days := int(-diff.Hours() / 24)
		if days == 0 {
			return "⚠️  today"
		}
		return fmt.Sprintf("⚠️  %dd ago", days)
	}

	days := int(diff.Hours() / 24)
	if days == 0 {
		return "📅 today"
	} else if days == 1 {
		return "📅 tomorrow"
	} else if days < 7 {
		return fmt.Sprintf("📅 in %dd", days)
	}

	return fmt.Sprintf("📅 %s", t.Format("Jan 02"))
}

// displayTasksGroupedByFile displays tasks grouped by their source file
func displayTasksGroupedByFile(taskList []*tasks.Task) {
	// Group tasks by file
	fileGroups := make(map[string][]*tasks.Task)
	for _, task := range taskList {
		filePath := task.Context.FilePath
		fileGroups[filePath] = append(fileGroups[filePath], task)
	}

	// Display each file group
	for filePath, fileTasks := range fileGroups {
		// Get relative path for better readability
		relPath := getRelativePathFromBrain(filePath, fileTasks[0].Context.BrainPath)

		// Show file header with task count
		fmt.Printf("\n📁 %s (%d)\n", relPath, len(fileTasks))

		// Display tasks in this file
		for _, task := range fileTasks {
			displayTaskLine(task, true) // true = show context like section
		}
	}
}

// displayTaskLine displays a single task line with all relevant info
func displayTaskLine(task *tasks.Task, showContext bool) {
	// Priority icon
	icon := tasks.PriorityIcon(task.Priority)
	if icon == "" {
		icon = "  "
	}

	// Status indicator
	statusIcon := ""
	switch task.Status {
	case tasks.StatusOpen:
		statusIcon = "[ ]"
	case tasks.StatusInProgress:
		statusIcon = "[~]"
	case tasks.StatusDone:
		statusIcon = "[x]"
	case tasks.StatusDeferred:
		statusIcon = "[>]"
	case tasks.StatusCancelled:
		statusIcon = "[-]"
	}

	// Organization/Project/Context metadata
	metaStr := ""
	if task.Organization != "" {
		metaStr = fmt.Sprintf(" [%s]", task.Organization)
	}
	if task.Project != "" {
		metaStr += fmt.Sprintf(" proj:%s", task.Project)
	}
	if task.ContextTag != "" {
		metaStr += fmt.Sprintf(" ctx:%s", task.ContextTag)
	}

	// Due date
	dueStr := ""
	if task.Due != nil {
		dueStr = " " + formatTaskDueDate(task.Due)
	}

	// Tags
	tagsStr := ""
	if len(task.Tags) > 0 {
		tagList := make([]string, 0, len(task.Tags))
		for _, tag := range task.Tags {
			tagList = append(tagList, "#"+tag)
		}
		tagsStr = " " + strings.Join(tagList, " ")
	}

	// Context (section heading)
	contextStr := ""
	if showContext && task.Context.Section != "" {
		contextStr = fmt.Sprintf(" › %s", task.Context.Section)
	}

	fmt.Printf("  %s %s %s%s%s%s%s\n",
		icon,
		statusIcon,
		truncate(task.Description, 60),
		metaStr,
		dueStr,
		tagsStr,
		contextStr,
	)
}

// getRelativePathFromBrain returns a relative path from the brain root
func getRelativePathFromBrain(fullPath, brainPath string) string {
	if strings.HasPrefix(fullPath, brainPath) {
		rel := strings.TrimPrefix(fullPath, brainPath)
		rel = strings.TrimPrefix(rel, "/")
		return rel
	}
	return fullPath
}

// runListTasksJSON outputs filtered tasks as JSON (for VS Code extension)
func runListTasksJSON(filter tasks.TaskFilter) error {
	// Load workspace config
	config, err := loadWorkspaceConfig()
	if err != nil {
		OutputJSONError("task-list", err)
		return nil
	}

	if config.ActiveWorkspace == "" {
		OutputJSONError("task-list", fmt.Errorf("no active workspace"))
		return nil
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
		OutputJSONError("task-list", fmt.Errorf("active workspace not found"))
		return nil
	}

	if len(activeWs.Brains) == 0 {
		OutputJSONError("task-list", fmt.Errorf("no brains in workspace"))
		return nil
	}

	result := TasksResult{
		Tasks: []TaskInfo{},
	}

	// Scan each brain
	for _, brain := range activeWs.Brains {
		scanner := tasks.NewScanner(brain.Path)
		brainTasks, err := scanner.ScanBrain()
		if err != nil {
			continue
		}

		// Build index and apply filter
		index := tasks.NewTaskIndex()
		index.Build(brainTasks)

		filteredTasks := index.Query(filter)

		// Convert to TaskInfo
		for _, t := range filteredTasks {
			relPath := getRelativePathFromBrain(t.Context.FilePath, brain.Path)

			taskInfo := TaskInfo{
				Description:  t.Description,
				Status:       statusString(t.Status),
				Priority:     priorityString(t.Priority),
				Tags:         t.Tags,
				Frog:         t.Frog,
				Path:         t.Context.FilePath,
				RelPath:      relPath,
				Line:         t.Context.LineNumber,
				BrainName:    brain.Name,
				BrainType:    brain.Type,
				Organization: t.Organization,
				Project:      t.Project,
				ContextTag:   t.ContextTag,
			}

			if t.Due != nil {
				taskInfo.Due = t.Due.Format("2006-01-02")
			}

			result.Tasks = append(result.Tasks, taskInfo)
		}
	}

	result.TotalCount = len(result.Tasks)
	OutputJSONSuccess("task-list", result)
	return nil
}
