package commands

// menu.go - Interactive Menu Entry Point
//
// This is the main entry point for flip's interactive menu system.
// The actual menu implementations are split across multiple files:
//
//   - menu.go          This file - entry point, runInteractiveMenu()
//   - menu_browse.go   Browse & Search menus
//   - menu_create.go   Create/Add menus
//   - menu_manage.go   Manage resources, brains, contexts
//   - menu_brain.go    Brain & workspace management
//   - menu_status.go   Status, Git, VS Code, Help menus
//   - menu_helpers.go  MenuItem type, templates, terminal helpers
//   - menu_git.go      Git sync operations
//   - menu_exercises.go Exercise tracking submenu
//
// Pattern: Each menu function follows the same structure:
//   1. Display header + breadcrumb
//   2. Define []MenuItem with Label, Description, Command, Action
//   3. Show promptui.Select
//   4. Execute selected Action (which returns to self or parent)

import (
	"fmt"

	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// NewMenuCommand creates the menu command
func NewMenuCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "menu-legacy",
		Short: "Interactive main menu (legacy promptui)",
		Long:  "Start flip with an interactive menu",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveMenu()
		},
	}
}

// runInteractiveMenu shows the main menu
func runInteractiveMenu() error {
	fmt.Println()
	// Compact status line
	displayStatusHeader()
	fmt.Println()

	// Main menu options with commands
	menuItems := []MenuItem{
		{
			Label:       lang.GetText("menu.main.create_label"),
			Description: lang.GetText("menu.main.create_desc"),
			Command:     lang.GetText("menu.main.create_cmd"),
			Action:      runCreateNewMenu,
		},
		{
			Label:       lang.GetText("menu.main.browse_label"),
			Description: lang.GetText("menu.main.browse_desc"),
			Command:     lang.GetText("menu.main.browse_cmd"),
			Action:      runBrowseSearchMenu,
		},
		{
			Label:       lang.GetText("menu.main.manage_label"),
			Description: lang.GetText("menu.main.manage_desc"),
			Command:     lang.GetText("menu.main.manage_cmd"),
			Action:      runManageResourcesMenu,
		},
		{
			Label:       lang.GetText("menu.main.status_label"),
			Description: lang.GetText("menu.main.status_desc"),
			Command:     lang.GetText("menu.main.status_cmd"),
			Action:      runStatusMenu,
		},
		{
			Label:       lang.GetText("menu.main.help_label"),
			Description: lang.GetText("menu.main.help_desc"),
			Command:     lang.GetText("menu.main.help_cmd"),
			Action:      runHelpMenu,
		},
		{
			Label:       "ℹ️  About",
			Description: "About flip, version info, and credits",
			Command:     "flip about",
			Action:      runAboutMenu,
		},
		{
			Label:       lang.GetText("menu.main.exit_label"),
			Description: lang.GetText("menu.main.exit_desc"),
			Command:     lang.GetText("menu.main.exit_cmd"),
			Action: func() error {
				checkUncommittedChangesOnExit()
				fmt.Println("👋 See you later!")
				return nil
			},
		},
	}

	// Create interactive select with command display
	templates := createMenuItemWithCommandTemplates()

	selectMenu := promptui.Select{
		Label:     lang.GetText("menu.titles.main"),
		Items:     menuItems,
		Templates: templates,
		Size:      calculateMenuSize(len(menuItems)),
		HideHelp:  true,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		// User pressed Ctrl+C
		checkUncommittedChangesOnExit()
		fmt.Println("👋 See you later!")
		return nil
	}

	// Execute selected action
	fmt.Println()
	return menuItems[idx].Action()
}
