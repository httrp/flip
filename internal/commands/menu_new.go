package commands

// menu_new.go - New bubbletea-based menu system
//
// This file provides integration between the new bubbletea UI and
// the existing flip commands. Enable with FLIP_FEATURE_NEWMENU=1

import (
	"fmt"
	"strings"

	"github.com/httrp/flip/internal/git"
	"github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// NewMenuV2Command creates the new menu command (bubbletea-based)
func NewMenuV2Command() *cobra.Command {
	return &cobra.Command{
		Use:   "menu2",
		Short: "Interactive main menu (new UI)",
		Long:  "Start flip with the new bubbletea-based interactive menu",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveMenuV2()
		},
	}
}

// runInteractiveMenuV2 runs the new bubbletea-based menu
func runInteractiveMenuV2() error {
	// Build status header
	status := buildStatusStringV2()

	// Run the bubbletea app
	execCmd, err := ui.RunApp(status)
	if err != nil {
		return err
	}

	// Handle the command that was selected
	if execCmd != "" {
		return executeCommandV2(execCmd)
	}

	// User quit without selecting a command
	fmt.Println("👋 See you later!")
	return nil
}

// buildStatusStringV2 creates the status line for the menu header
func buildStatusStringV2() string {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return "⚠️  No workspace configured"
	}

	if config.ActiveWorkspace == "" {
		return "⚠️  No active workspace"
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
		return "⚠️  Workspace not found"
	}

	// Build status string
	parts := []string{fmt.Sprintf("📂 %s", activeWs.Name)}

	// Default brain
	var defaultBrain *Brain
	if activeWs.DefaultBrain != "" {
		parts = append(parts, fmt.Sprintf("🧠 %s", activeWs.DefaultBrain))
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == activeWs.DefaultBrain {
				defaultBrain = &activeWs.Brains[i]
				break
			}
		}
	}

	// Git status
	if defaultBrain != nil {
		status, err := git.GetStatus(defaultBrain.Path)
		if err == nil && status.IsRepo {
			parts = append(parts, fmt.Sprintf("⎇ %s", status.Branch))
			if status.HasChanges {
				parts = append(parts, "●")
			} else {
				parts = append(parts, "✓")
			}
		}
	}

	return strings.Join(parts, " · ")
}

// executeCommandV2 executes the command selected from the menu
func executeCommandV2(cmd string) error {
	switch cmd {
	case "note":
		return runCreateNote()
	case "quicknote":
		return runCreateQuicknote()
	case "meeting":
		return runCreateMeeting()
	case "journal":
		return runCreateJournal()
	case "task":
		return runCreateTask()
	case "recent":
		return showRecentNotes(20)
	case "search":
		return runBrowseSearchMenu() // Fall back to old menu for now
	case "task-browse":
		return runTaskBrowser("")
	case "task-frog":
		return runTaskBrowser("frog")
	case "brain":
		return runSwitchWorkspaceMenu() // Fall back to old brain menu
	case "status":
		return runStatus() // Show status summary
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}
