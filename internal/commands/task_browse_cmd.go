package commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/httrp/flip/internal/tasks"
	"github.com/spf13/cobra"
)

// NewTaskBrowseCommand creates the interactive task browse command
func NewTaskBrowseCommand() *cobra.Command {
	var (
		filterMode string
	)

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse tasks interactively",
		Long:  "Launch an interactive terminal UI to browse, filter, and manage tasks.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskBrowser(filterMode)
		},
	}

	cmd.Flags().StringVarP(&filterMode, "filter", "f", "all", "Initial filter (all, today, week, overdue, high)")

	return cmd
}

// runTaskBrowser launches the interactive task browser
func runTaskBrowser(initialFilter string) error {
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
	fmt.Println("\n🔍 Loading tasks...")
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

	// Convert to pointers for the browser
	taskPointers := make([]*tasks.Task, len(allTasks))
	for i := range allTasks {
		taskPointers[i] = &allTasks[i]
	}

	// Create and run the browser
	model := NewTaskBrowser(taskPointers)
	model.filterMode = initialFilter
	model.applyFilter()

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running browser: %w", err)
	}

	return nil
}
