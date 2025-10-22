package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewTaskCommand creates the main task command with subcommands
func NewTaskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks across your brains",
		Long:  "Create, list, update, and manage tasks in your second brain system.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no subcommand, show interactive menu
			return runTaskMenu()
		},
	}

	// Add subcommands
	cmd.AddCommand(NewTaskListCommand())
	cmd.AddCommand(NewTaskNewCommand())
	cmd.AddCommand(NewTaskDoneCommand())
	cmd.AddCommand(NewTaskStartCommand())
	cmd.AddCommand(NewTaskUpdateCommand())
	cmd.AddCommand(NewTaskSearchCommand())
	cmd.AddCommand(NewTaskStatsCommand())

	return cmd
}

// runTaskMenu shows an interactive task management menu
func runTaskMenu() error {
	fmt.Println("\n📋 Task Management")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Use 'flip task --help' to see all available commands")
	fmt.Println()
	fmt.Println("Quick commands:")
	fmt.Println("  flip task list           - List all open tasks")
	fmt.Println("  flip task new            - Create a new task")
	fmt.Println("  flip task list --today   - Show tasks due today")
	fmt.Println("  flip task stats          - Show task statistics")
	fmt.Println()

	return nil
}
