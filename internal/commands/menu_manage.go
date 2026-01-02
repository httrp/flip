package commands

// menu_manage.go - Resource Management Menu Functions
//
// Contains:
//   - runManageResourcesMenu()  Main manage submenu
//   - runManageBrainsMenu()     Brain management options
//   - runSwitchContextMenu()    Switch workspace/brain context

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
)

// runManageResourcesMenu shows submenu for managing workspaces, brains, and templates
func runManageResourcesMenu() error {
	fmt.Println()
	displayStatusHeader()
	showBreadcrumb("Main › Manage")
	fmt.Println()

	menuItems := []MenuItem{
		{
			Label:       lang.GetText("menu.manage.workspaces_label"),
			Description: lang.GetText("menu.manage.workspaces_desc"),
			Command:     lang.GetText("menu.manage.workspaces_cmd"),
			Action:      runEditWorkspaceMenu,
		},
		{
			Label:       lang.GetText("menu.manage.brains_label"),
			Description: lang.GetText("menu.manage.brains_desc"),
			Command:     lang.GetText("menu.manage.brains_cmd"),
			Action:      runManageBrainsMenu,
		},
		{
			Label:       lang.GetText("menu.manage.config_label"),
			Description: lang.GetText("menu.manage.config_desc"),
			Command:     "",
			Action:      runViewWorkspaceConfig,
		},
		{
			Label:       lang.GetText("menu.manage.templates_label"),
			Description: lang.GetText("menu.manage.templates_desc"),
			Command:     lang.GetText("menu.manage.templates_cmd"),
			Action:      runEditManageMenu, // Reuse existing template management
		},
		{
			Label:       lang.GetText("menu.manage.git_status_label"),
			Description: lang.GetText("menu.manage.git_status_desc"),
			Command:     "",
			Action: func() error {
				if err := runBrainGitStatus(false); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runManageResourcesMenu()
			},
		},
		{
			Label:       lang.GetText("menu.manage.git_pull_label"),
			Description: lang.GetText("menu.manage.git_pull_desc"),
			Command:     "",
			Action: func() error {
				checkRemoteUpdatesOnStart()
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runManageResourcesMenu()
			},
		},
		{
			Label:       lang.GetText("menu.manage.git_commit_label"),
			Description: lang.GetText("menu.manage.git_commit_desc"),
			Command:     "",
			Action: func() error {
				checkUncommittedChangesOnExit()
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runManageResourcesMenu()
			},
		},
		{
			Label:       lang.GetText("menu.manage.migrate_label"),
			Description: lang.GetText("menu.manage.migrate_desc"),
			Command:     lang.GetText("menu.manage.migrate_cmd"),
			Action:      runBrainMigrationMenu,
		},
		{
			Label:       lang.GetText("menu.manage.back_label"),
			Description: lang.GetText("menu.manage.back_desc"),
			Command:     "",
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemWithCommandTemplates()

	selectMenu := promptui.Select{
		Label:     lang.GetText("menu.titles.manage"),
		Items:     menuItems,
		Templates: templates,
		Size:      calculateMenuSize(len(menuItems)),
		HideHelp:  true,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runInteractiveMenu()
	}

	fmt.Println()
	return menuItems[idx].Action()
}

// runManageBrainsMenu shows submenu for brain management
func runManageBrainsMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       lang.GetText("menu.brains.details_label"),
			Description: lang.GetText("menu.brains.details_desc"),
			Action: func() error {
				// Get active workspace
				workspace, err := getActiveWorkspace()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runManageResourcesMenu()
				}

				if len(workspace.Brains) == 0 {
					fmt.Println("\n🧠 No brains in active workspace.")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runManageResourcesMenu()
				}

				// Select brain to view with status indicators
				// Note: Git status checks are expensive, so we skip them here for performance
				// Full status will be shown in detail view
				var brainNames []string
				for _, b := range workspace.Brains {
					indicator := "  "
					if workspace.DefaultBrain == b.Name {
						indicator = "⭐"
					}
					brainNames = append(brainNames, fmt.Sprintf("%s %s (%s)", indicator, b.Name, b.Type))
				}

				selectBrain := promptui.Select{
					Label:     "Select brain to view",
					Items:     brainNames,
					Size:      calculateMenuSize(len(brainNames)),
					Templates: createSimpleSelectTemplates(),
					HideHelp:  true,
				}

				idx, _, err := selectBrain.Run()
				if err != nil {
					return runManageResourcesMenu()
				}

				// Show detail view
				return runBrainDetails(workspace.Brains[idx])()
			},
		},
		{
			Label:       lang.GetText("menu.manage_brains.create_label"),
			Description: lang.GetText("menu.manage_brains.create_desc"),
			Action: func() error {
				if err := runNewBrain(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.manage_brains.add_label"),
			Description: lang.GetText("menu.manage_brains.add_desc"),
			Action: func() error {
				// Show common paths for brain selection
				home, _ := os.UserHomeDir()
				commonPaths := map[string]string{
					"Home Directory":              home,
					"Documents":                   filepath.Join(home, "Documents"),
					"Documents/Obsidian":          filepath.Join(home, "Documents", "Obsidian"),
					"Documents/Logseq":            filepath.Join(home, "Documents", "Logseq"),
					"Dropbox (if available)":      filepath.Join(home, "Dropbox"),
					"iCloud Drive (if available)": filepath.Join(home, "Library", "Mobile Documents"),
				}

				path, err := BrowseDirectoryWithCommonPaths(true, commonPaths)
				if err != nil {
					fmt.Printf("\n❌ Selection cancelled\n")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runManageBrainsMenu()
				}

				if err := runDirectoryInit(path, "", false); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runManageBrainsMenu()
			},
		},
		{
			Label:       lang.GetText("menu.manage_brains.scan_label"),
			Description: lang.GetText("menu.manage_brains.scan_desc"),
			Action: func() error {
				// Show common paths for scanning
				home, _ := os.UserHomeDir()
				commonPaths := map[string]string{
					"Home Directory (~)":          home,
					"Documents":                   filepath.Join(home, "Documents"),
					"Dropbox (if available)":      filepath.Join(home, "Dropbox"),
					"iCloud Drive (if available)": filepath.Join(home, "Library", "Mobile Documents"),
				}

				scanPath, err := BrowseDirectoryWithCommonPaths(true, commonPaths)
				if err != nil {
					fmt.Printf("\n❌ Selection cancelled\n")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runManageBrainsMenu()
				}

				if err := runScan(scanPath); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runManageBrainsMenu()
			},
		},
		{
			Label:       lang.GetText("menu.manage_brains.edit_label"),
			Description: lang.GetText("menu.manage_brains.edit_desc"),
			Action:      runEditBrainMenu,
		},
		{
			Label:       lang.GetText("menu.manage_brains.back_label"),
			Description: lang.GetText("menu.manage_brains.back_desc"),
			Action:      runManageResourcesMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     lang.GetText("menu.titles.manage_brains"),
		Items:     menuItems,
		Templates: templates,
		Size:      calculateMenuSize(len(menuItems)),
		HideHelp:  true,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runManageResourcesMenu()
	}

	fmt.Println()
	return menuItems[idx].Action()
}

// runSwitchContextMenu shows submenu for switching context
func runSwitchContextMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       lang.GetText("menu.switch.workspace_label"),
			Description: lang.GetText("menu.switch.workspace_desc"),
			Action:      runSwitchWorkspaceMenu,
		},
		{
			Label:       lang.GetText("menu.switch.brain_label"),
			Description: lang.GetText("menu.switch.brain_desc"),
			Action: func() error {
				// Load config
				config, err := loadWorkspaceConfig()
				if err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if config.ActiveWorkspace == "" {
					fmt.Println("\n❌ No active workspace")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Find active workspace
				var activeWs *Workspace
				for i := range config.Workspaces {
					if config.Workspaces[i].Name == config.ActiveWorkspace {
						activeWs = &config.Workspaces[i]
						break
					}
				}

				if activeWs == nil || len(activeWs.Brains) == 0 {
					fmt.Println(lang.GetText("errors.no_brains"))
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Create menu items
				var brainNames []string
				for _, brain := range activeWs.Brains {
					brainNames = append(brainNames, brain.Name)
				}

				prompt := promptui.Select{
					Label:     lang.GetText("prompts.select_brain_default"),
					Items:     brainNames,
					Templates: createSimpleSelectTemplates(),
					Size:      calculateMenuSize(len(brainNames)),
					HideHelp:  true,
				}

				idx, _, err := prompt.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				if err := runBrainSetDefault(brainNames[idx]); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				} else {
					fmt.Printf("\n✓ Set '%s' as default brain\n", brainNames[idx])
				}

				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.switch.back_label"),
			Description: lang.GetText("menu.switch.back_desc"),
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     lang.GetText("menu.titles.switch"),
		Items:     menuItems,
		Templates: templates,
		Size:      calculateMenuSize(len(menuItems)),
		HideHelp:  true,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runInteractiveMenu()
	}

	fmt.Println()
	return menuItems[idx].Action()
}
