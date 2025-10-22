package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewStatusCommand() *cobra.Command {
	var theme string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show overview of workspaces and brains",
		Long:  "Display all configured workspaces, active workspace, and their brains",
		RunE: func(cmd *cobra.Command, args []string) error {
			if theme != "" {
				SetIconTheme(theme)
			}
			return runStatus()
		},
	}
	cmd.Flags().StringVar(&theme, "theme", "", "icon theme: ascii|emoji|mixed")
	return cmd
}

func runStatus() error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	// Optional banner (load from file)
	banner := ""
	data, err := os.ReadFile("lang/banner.txt")
	if err == nil {
		banner = string(data)
	} else {
		banner = "Flip - Your Digital 2nd Brain Assistant"
	}
	fmt.Println(banner)

	if len(config.Workspaces) == 0 {
		fmt.Println("[ ] No workspaces configured yet.")
		fmt.Println("\n💡 Tip: Use the main menu to get started with Quickstart or create a new brain.")
		return nil
	}

	fmt.Println("=== Flip Status ===")
	fmt.Println("===============================================================")
	fmt.Println()

	// Check for issues across all brains and show warning banner
	var allIssues []string
	var allWarnings []string
	for _, ws := range config.Workspaces {
		for _, b := range ws.Brains {
			health := checkBrainHealth(b.Path)
			if health.HasIssues {
				for _, issue := range health.Issues {
					allIssues = append(allIssues, fmt.Sprintf("%s/%s: %s", ws.Name, b.Name, issue))
				}
			}
			if len(health.Warnings) > 0 {
				for _, warning := range health.Warnings {
					allWarnings = append(allWarnings, fmt.Sprintf("%s/%s: %s", ws.Name, b.Name, warning))
				}
			}
		}
	}

	if len(allIssues) > 0 {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  ATTENTION: Issues detected with your brains!")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for _, issue := range allIssues {
			fmt.Printf("  ❌ %s\n", issue)
		}
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
	}

	if len(allWarnings) > 0 && len(allIssues) == 0 {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚡ Warnings: Some brains need attention")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		for i, warning := range allWarnings {
			if i < 5 { // Show max 5 warnings in summary
				fmt.Printf("  ⚠️  %s\n", warning)
			}
		}
		if len(allWarnings) > 5 {
			fmt.Printf("  ... and %d more warnings\n", len(allWarnings)-5)
		}
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
	}

	// Active workspace + default brain line
	if config.ActiveWorkspace != "" {
		fmt.Printf("%s Active Workspace: %s\n", IconActive, config.ActiveWorkspace)
		if ws, err := getActiveWorkspace(); err == nil && ws.DefaultBrain != "" {
			fmt.Printf("%s Default Brain: %s\n", IconDefault, ws.DefaultBrain)
		}
		fmt.Println()
	}

	// Table: Workspaces overview
	fmt.Printf("%s All Workspaces\n\n", IconWorkspace)
	wsHeaders := []string{"Active", "Workspace", "Brains", "Default Brain", "Description"}
	wsRows := [][]string{}
	for _, ws := range config.Workspaces {
		active := ""
		if ws.Name == config.ActiveWorkspace {
			active = IconActive
		}
		def := ws.DefaultBrain
		wsRows = append(wsRows, []string{active, ws.Name, fmt.Sprintf("%d", len(ws.Brains)), def, ws.Description})
	}
	renderTableStdout(wsHeaders, wsRows)
	fmt.Println()

	// Table: Brains in active workspace
	if ws, err := getActiveWorkspace(); err == nil {
		fmt.Printf("%s Brains in '%s'\n\n", IconBrain, ws.Name)
		bHeaders := []string{"Default", "Name", "Type", "Health", "Sync/Git", "Details"}
		bRows := [][]string{}

		hasIssues := false
		for _, b := range ws.Brains {
			def := ""
			if b.Name == ws.DefaultBrain {
				def = IconDefault
			}

			// Check brain health
			health := checkBrainHealth(b.Path)
			healthIcon := formatHealthStatus(health)

			// Track if any brain has issues
			if health.HasIssues {
				hasIssues = true
			}

			// Build sync/git column
			syncGitInfo := ""
			if health.SyncService != "" {
				syncGitInfo = "☁️ " + health.SyncService
			}
			if health.IsGitRepo {
				if syncGitInfo != "" {
					syncGitInfo += " "
				}
				syncGitInfo += "📦"
				if health.GitStatus.Branch != "" {
					syncGitInfo += " " + health.GitStatus.Branch
				}
			}
			if syncGitInfo == "" {
				syncGitInfo = "Local"
			}

			// Get health details
			details := formatHealthDetails(health)
			if _, err := os.Stat(b.Path); os.IsNotExist(err) {
				healthIcon = IconError
				details = "Path does not exist"
			}

			bRows = append(bRows, []string{def, b.Name, b.Type, healthIcon, syncGitInfo, details})
		}
		renderTableStdout(bHeaders, bRows)
		fmt.Println()

		// Show warning if issues detected
		if hasIssues {
			fmt.Println("⚠️  Issues detected! Review the details above.")
			fmt.Println("   Use 'flip brain repair' to fix path issues.")
			fmt.Println("   Use 'git pull' to sync behind commits.")
			fmt.Println("   Resolve conflict files in your sync service.")
			fmt.Println()
		}
	}

	// Quick menu
	fmt.Println("Menu:")
	fmt.Println("  intro          Short introduction to Flip")
	fmt.Println("  quickstart     Guided onboarding")
	fmt.Println("  workspace      Manage workspaces (create/list/switch/remove/rename/repair)")
	fmt.Println("  brain          Manage brains (add/list/remove/set-default/rename/repair)")
	fmt.Println("  brain init     Initialize a brain in a directory")
	fmt.Println("  brain new      Create a new brain with a chosen dialect")
	fmt.Println("  brain scan     Find existing brain folders")
	fmt.Println()

	// Stats
	totalBrains := 0
	for _, ws := range config.Workspaces {
		totalBrains += len(ws.Brains)
	}
	fmt.Println("---------------------------------------------------------------")
	fmt.Printf("Total: %d workspaces, %d brains\n", len(config.Workspaces), totalBrains)

	return nil
}
