package commands

import (
	"fmt"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/httrp/flip/internal/tasks"
	"github.com/spf13/cobra"
)

// Note: TaskInfo and TasksResult are defined in vscode.go and reused here

// NewTaskBrowseCommand creates the interactive task browse command
func NewTaskBrowseCommand() *cobra.Command {
	var (
		filterMode string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse tasks interactively",
		Long:  "Launch an interactive terminal UI to browse, filter, and manage tasks. Use --json to output JSON for VS Code extension.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				return runTaskBrowseJSON(filterMode)
			}
			return runTaskBrowser(filterMode)
		},
	}

	cmd.Flags().StringVarP(&filterMode, "filter", "f", "open", "Initial filter (open, all, today, week, overdue, high)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output tasks as JSON (for VS Code integration)")

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

	// Map old filter parameter to new two-axis system
	// For backward compatibility with command-line flags
	switch initialFilter {
	case "open", "inprogress", "done", "deferred", "cancelled":
		model.statusFilter = initialFilter
		model.scopeFilter = "all"
	case "today", "week", "overdue", "high":
		model.scopeFilter = initialFilter
		model.statusFilter = "open"
	case "all":
		model.scopeFilter = "all"
		model.statusFilter = "all"
	default:
		// Keep defaults from NewTaskBrowser (all scope, open status)
	}

	model.applyFilter()

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running browser: %w", err)
	}

	return nil
}

// runTaskBrowseJSON returns tasks as JSON (for VS Code extension)
func runTaskBrowseJSON(initialFilter string) error {
	// Load workspace config
	config, err := loadWorkspaceConfig()
	if err != nil {
		OutputJSONError("task-browse", err)
		return nil
	}

	if config.ActiveWorkspace == "" {
		OutputJSONError("task-browse", fmt.Errorf("no active workspace"))
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
		OutputJSONError("task-browse", fmt.Errorf("active workspace not found"))
		return nil
	}

	if len(activeWs.Brains) == 0 {
		OutputJSONError("task-browse", fmt.Errorf("no brains in workspace"))
		return nil
	}

	result := TasksResult{
		Tasks: []TaskInfo{},
	}

	// Scan each brain individually so we retain brain type information
	for _, brain := range activeWs.Brains {
		scanner := tasks.NewScanner(brain.Path)
		brainTasks, err := scanner.ScanBrain()
		if err != nil {
			continue // Skip brain on error
		}

		// Convert tasks to TaskInfo format
		for _, t := range brainTasks {
			relPath, _ := filepath.Rel(brain.Path, t.Context.FilePath)
			
			taskInfo := TaskInfo{
				Description:  t.Description,
				Status:       statusString(t.Status),      // Use vscode.go function
				Priority:     priorityString(t.Priority),  // Use vscode.go function
				Tags:         t.Tags,
				Frog:         t.Frog,
				Path:         t.Context.FilePath,
				RelPath:      relPath,
				Line:         t.Context.LineNumber,
				BrainName:    brain.Name,    // From Brain struct
				BrainType:    brain.Type,    // From Brain struct
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
	OutputJSONSuccess("task-browse", result)
	return nil
}
