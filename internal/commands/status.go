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
		bHeaders := []string{"Default", "Name", "Type", "Status", "Path"}
		bRows := [][]string{}
		for _, b := range ws.Brains {
			def := ""
			if b.Name == ws.DefaultBrain {
				def = IconDefault
			}
			status := IconCheck
			if _, err := os.Stat(b.Path); os.IsNotExist(err) {
				status = IconError
			}
			bRows = append(bRows, []string{def, b.Name, b.Type, status, b.Path})
		}
		renderTableStdout(bHeaders, bRows)
		fmt.Println()
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
