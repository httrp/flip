package commands

import (
"fmt"

"github.com/httrp/flip/internal/lang"
	ui "github.com/httrp/flip/internal/ui"
"github.com/spf13/cobra"
)

// NewDefinitionsCommand creates the definitions command
func NewDefinitionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "definitions",
		Aliases: []string{"def", "defs"},
		Short:   "Manage task definitions (organizations, projects, contexts)",
		Long:    "View and manage definitions for organizations, projects, and contexts used in tasks.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsMenu()
		},
	}

	cmd.AddCommand(newDefinitionsListCommand())
	cmd.AddCommand(newDefinitionsAddCommand())
	cmd.AddCommand(newDefinitionsRemoveCommand())
	cmd.AddCommand(newDefinitionsScanCommand())
	cmd.AddCommand(newDefinitionsPathCommand())

	return cmd
}

// runDefinitionsMenu shows an interactive menu for managing definitions
func runDefinitionsMenu() error {
	for {
		fmt.Println("\n📚 Task Definitions")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		items := []string{
			lang.GetText("menu.definitions.organizations_label"),
			lang.GetText("menu.definitions.projects_label"),
			lang.GetText("menu.definitions.contexts_label"),
			lang.GetText("menu.definitions.people_label"),
			lang.GetText("menu.definitions.scan_label"),
			lang.GetText("menu.definitions.export_label"),
			lang.GetText("menu.definitions.file_label"),
			lang.GetText("menu.definitions.back_label"),
		}

		selectItems := make([]ui.SelectItem, len(items))
		for i, item := range items {
			selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
		}

		idx, _, err := ui.RunSelect("What would you like to do?", selectItems, 10)
		if err != nil {
			return err
		}

		switch idx {
		case 0:
			if err := runDefinitionsBrowseOrganizations(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 1:
			if err := runDefinitionsBrowseProjects(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 2:
			if err := runDefinitionsBrowseContexts(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 3:
			if err := runDefinitionsBrowsePeople(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 4:
			if err := runDefinitionsScan(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			fmt.Println(lang.GetText("prompts.continue"))
			fmt.Scanln()
		case 5:
			if err := runDefinitionsExportToNote(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			fmt.Println(lang.GetText("prompts.continue"))
			fmt.Scanln()
		case 6:
			if err := runDefinitionsOpenFile(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			fmt.Println(lang.GetText("prompts.continue"))
			fmt.Scanln()
		case 7:
			return nil
		}
	}
}

// newDefinitionsListCommand creates the list subcommand
func newDefinitionsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all definitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsList()
		},
	}
}

// newDefinitionsAddCommand creates the add subcommand
func newDefinitionsAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [type]",
		Short: "Add a new definition",
		Long:  "Add a new organization, project, or context definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("please specify type: org, project, or context")
			}

			switch args[0] {
			case "org", "organization":
				return runDefinitionsAddOrg()
			case "proj", "project":
				return runDefinitionsAddProject()
			case "ctx", "context":
				return runDefinitionsAddContext()
			default:
				return fmt.Errorf("unknown type: %s (use: org, project, or context)", args[0])
			}
		},
	}
	return cmd
}

// newDefinitionsRemoveCommand creates the remove subcommand
func newDefinitionsRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove a definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsRemove()
		},
	}
}

// newDefinitionsScanCommand creates the scan subcommand
func newDefinitionsScanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Scan tasks for organizations, projects, and contexts",
		Long:  "Scan all tasks in the active brain and suggest adding found organizations, projects, and contexts to definitions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsScan()
		},
	}
}

// newDefinitionsPathCommand creates the path subcommand
func newDefinitionsPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show definitions file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsPath()
		},
	}
}
