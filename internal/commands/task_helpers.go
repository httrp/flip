package commands

import (
	"fmt"

	"github.com/httrp/flip/internal/tasks"
	"github.com/spf13/cobra"
)

// NewTaskDoneCommand creates the task done command
func NewTaskDoneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "done [task-id]",
		Short: "Mark a task as done",
		Long:  "Mark a task as completed.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("✅ Task completion functionality coming soon!")
			fmt.Println("For now, you can manually edit the task file and change [ ] to [x]")
			return nil
		},
	}
	return cmd
}

// NewTaskStartCommand creates the task start command
func NewTaskStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [task-id]",
		Short: "Mark a task as in-progress",
		Long:  "Mark a task as started/in-progress.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("🔄 Task start functionality coming soon!")
			fmt.Println("For now, you can manually edit the task file and change [ ] to [/]")
			return nil
		},
	}
	return cmd
}

// NewTaskUpdateCommand creates the task update command
func NewTaskUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [task-id]",
		Short: "Update a task",
		Long:  "Update task properties interactively.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("📝 Task update functionality coming soon!")
			return nil
		},
	}
	return cmd
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
			return runListTasks(filter)
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

	fmt.Println()
	displayStatusHeader()
	fmt.Println()

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
