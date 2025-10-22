package commands

import (
	"fmt"
	"time"

	"github.com/httrp/flip/internal/tasks"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// NewTaskListCommand creates the task list command
func NewTaskListCommand() *cobra.Command {
	var (
		statusFilter   string
		priorityFilter string
		tagFilter      string
		projectFilter  string
		dueToday       bool
		dueThisWeek    bool
		overdue        bool
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

			return runListTasks(filter)
		},
	}

	cmd.Flags().StringVar(&statusFilter, "status", "", "Filter by status (open, in-progress, done)")
	cmd.Flags().StringVar(&priorityFilter, "priority", "", "Filter by priority (high, medium, low)")
	cmd.Flags().StringVar(&tagFilter, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&projectFilter, "project", "", "Filter by project")
	cmd.Flags().BoolVar(&dueToday, "today", false, "Show tasks due today")
	cmd.Flags().BoolVar(&dueThisWeek, "week", false, "Show tasks due this week")
	cmd.Flags().BoolVar(&overdue, "overdue", false, "Show overdue tasks")

	return cmd
}

// runListTasks lists and filters tasks
func runListTasks(filter tasks.TaskFilter) error {
	fmt.Println("\n📋 Task List")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

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
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	totalShown := 0

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

	fmt.Printf("Total: %d tasks\n\n", totalShown)

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

		fmt.Printf("  %s %s %s %s %s\n",
			icon,
			truncate(task.Description, 50),
			dueStr,
			tags,
			brain,
		)
	}
}

// selectAndActOnTask shows interactive menu for task actions
func selectAndActOnTask(taskList []*tasks.Task) error {
	items := []string{
		"📝 Open task file in editor",
		"✅ Mark task as done",
		"🔄 Mark as in-progress",
		"◀️  Back",
	}

	prompt := promptui.Select{
		Label: "What would you like to do?",
		Items: items,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return nil
	}

	switch idx {
	case 0: // Open file
		if len(taskList) > 0 {
			return openInEditor(taskList[0].Context.FilePath)
		}
	case 1: // Mark done
		fmt.Println("Task completion coming soon!")
	case 2: // Mark in-progress
		fmt.Println("Task status update coming soon!")
	case 3: // Back
		return nil
	}

	return nil
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
