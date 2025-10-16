package commands

import (
	"fmt"
	"os"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func NewMenuCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "menu",
		Short: "Interactive main menu",
		Long:  "Start flip with an interactive menu",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveMenu()
		},
	}
}

func runInteractiveMenu() error {
	// Load banner
	banner := ""
	data, err := os.ReadFile("lang/banner.txt")
	if err == nil {
		banner = string(data)
	} else {
		banner = "Flip - Your Intelligent Assistant"
	}
	fmt.Println("\n" + banner)
	fmt.Println("\n" + getText("welcome_banner"))
	fmt.Println()

	// Main menu options
	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       "🚀 Quickstart (Guided Setup)",
			Description: "First time? Walk through creating your first workspace and brain",
			Action:      runQuickstart,
		},
		{
			Label:       "✨ Create New Brain",
			Description: "Quickly create a new knowledge base",
			Action: func() error {
				return runNewBrain()
			},
		},
		{
			Label:       "📊 View Status",
			Description: "Show all workspaces and brains",
			Action:      runStatus,
		},
		{
			Label:       "❓ Help & Documentation",
			Description: "View available commands and options",
			Action: func() error {
				fmt.Println("\n=== Flip Commands ===\n")
				fmt.Println("Core Commands:")
				fmt.Println("  flip quickstart          # Guided onboarding")
				fmt.Println("  flip new                 # Create new brain")
				fmt.Println("  flip init <path>         # Initialize existing directory")
				fmt.Println("  flip status              # Show overview")
				fmt.Println()
				fmt.Println("Workspace Management:")
				fmt.Println("  flip workspace create <name>   # Create workspace")
				fmt.Println("  flip workspace list            # List all workspaces")
				fmt.Println("  flip workspace switch <name>   # Switch active workspace")
				fmt.Println("  flip workspace remove <name>   # Remove workspace")
				fmt.Println()
				fmt.Println("Brain Management:")
				fmt.Println("  flip brain add <path>          # Add brain to workspace")
				fmt.Println("  flip brain list                # List brains in workspace")
				fmt.Println("  flip brain set-default <name>  # Set default brain")
				fmt.Println("  flip brain remove <name>       # Remove brain")
				fmt.Println()
				fmt.Println("For detailed help on any command, use: flip <command> --help")
				fmt.Println()
				return nil
			},
		},
		{
			Label:       "🚪 Exit",
			Description: "Exit flip",
			Action: func() error {
				fmt.Println("\n👋 See you later!")
				return nil
			},
		},
	}

	// Create interactive select
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ .Label | cyan | bold }}",
		Inactive: "  {{ .Label }}",
		Selected: "{{ .Label | green | bold }}",
		Details: `
--------- Details ----------
{{ "Description:" | faint }}  {{ .Description }}`,
	}

	selectMenu := promptui.Select{
		Label:     "What would you like to do?",
		Items:     menuItems,
		Templates: templates,
		Size:      6,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		// User pressed Ctrl+C
		fmt.Println("\n👋 See you later!")
		return nil
	}

	// Execute selected action
	fmt.Println()
	return menuItems[idx].Action()
}
