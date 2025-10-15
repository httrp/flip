package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show overview of workspaces and brains",
		Long:  "Display all configured workspaces, active workspace, and their brains",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}
}

func runStatus() error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	if len(config.Workspaces) == 0 {
		fmt.Println("[ ] No workspaces configured yet.")
		fmt.Println()
		fmt.Println("Get started:")
		fmt.Println("  flip quickstart       # Guided onboarding")
		fmt.Println("  flip new              # Create a new brain")
		fmt.Println("  flip workspace create personal    # Create empty workspace")
		fmt.Println("  flip brain add <path> # Add existing brain to workspace")
		return nil
	}

	fmt.Println("=== Flip Status ===")
	fmt.Println("===============================================================")
	fmt.Println()

	// Show active workspace prominently
	if config.ActiveWorkspace != "" {
		fmt.Printf("%s Active Workspace: %s\n", IconActive, config.ActiveWorkspace)

		// Show default brain
		ws, err := getActiveWorkspace()
		if err == nil && ws.DefaultBrain != "" {
			fmt.Printf("%s Default Brain: %s\n", IconDefault, ws.DefaultBrain)
		}
		fmt.Println()
	}

	// List all workspaces
	fmt.Printf("~ Workspaces (%d):\n\n", len(config.Workspaces))

	for _, ws := range config.Workspaces {
		activeMarker := "  "
		if ws.Name == config.ActiveWorkspace {
			activeMarker = IconActive + " "
		}

		fmt.Printf("%s%s", activeMarker, ws.Name)
		if ws.Description != "" {
			fmt.Printf(" - %s", ws.Description)
		}
		fmt.Println()

		if len(ws.Brains) == 0 {
			fmt.Println("   `- (empty)")
		} else {
			for i, brain := range ws.Brains {
				isLast := i == len(ws.Brains)-1
				connector := "+-"
				if isLast {
					connector = "`-"
				}

				brainMarker := ""
				if brain.Name == ws.DefaultBrain {
					brainMarker = " " + IconDefault
				}

				// Check if path exists
				status := IconCheck
				if _, err := os.Stat(brain.Path); os.IsNotExist(err) {
					status = IconError
				}

				fmt.Printf("   %s %s %s (%s)%s\n", connector, status, brain.Name, brain.Type, brainMarker)

				// Show path indented
				indent := "   "
				if !isLast {
					indent = "   |  "
				} else {
					indent = "      "
				}
				fmt.Printf("%s%s\n", indent, brain.Path)
			}
		}
		fmt.Println()
	}

	// Quick stats
	totalBrains := 0
	for _, ws := range config.Workspaces {
		totalBrains += len(ws.Brains)
	}

	fmt.Println("---------------------------------------------------------------")
	fmt.Printf("Total: %d workspaces, %d brains\n", len(config.Workspaces), totalBrains)

	return nil
}
