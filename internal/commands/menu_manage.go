package commands

// menu_manage.go - Resource Management Menu Functions
//
// Contains:
//   - runManageResourcesMenu()  Main manage submenu
//   - runManageBrainsMenu()     Brain management options

import (
	"fmt"

	"github.com/httrp/flip/internal/lang"
	"github.com/httrp/flip/internal/platform"
	"github.com/manifoldco/promptui"
)

// runManageResourcesMenu shows submenu for managing workspaces, brains, and templates
func runManageResourcesMenu() error {
	fmt.Println()
	displayStatusHeader()
	showBreadcrumb("Main › Manage")
	fmt.Println()

	// Build menu items dynamically based on enabled features
	var menuItems []MenuItem

	// Core management items (always available)
	menuItems = append(menuItems,
		MenuItem{
			Label:       lang.GetText("menu.manage.workspaces_label"),
			Description: lang.GetText("menu.manage.workspaces_desc"),
			Command:     lang.GetText("menu.manage.workspaces_cmd"),
			Action:      runEditWorkspaceMenu,
		},
		MenuItem{
			Label:       lang.GetText("menu.manage.brains_label"),
			Description: lang.GetText("menu.manage.brains_desc"),
			Command:     lang.GetText("menu.manage.brains_cmd"),
			Action:      runManageBrainsMenu,
		},
		MenuItem{
			Label:       lang.GetText("menu.manage.config_label"),
			Description: lang.GetText("menu.manage.config_desc"),
			Command:     "",
			Action:      runViewWorkspaceConfig,
		},
	)

	// Templates feature
	if IsEnabled(FeatureTemplates) {
		menuItems = append(menuItems, MenuItem{
			Label:       lang.GetText("menu.manage.templates_label"),
			Description: lang.GetText("menu.manage.templates_desc"),
			Command:     lang.GetText("menu.manage.templates_cmd"),
			Action:      runEditManageMenu, // Reuse existing template management
		})
	}

	// Git feature (always available - part of core)
	menuItems = append(menuItems,
		MenuItem{
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
		MenuItem{
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
		MenuItem{
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
	)

	// Migration feature
	if IsEnabled(FeatureMigration) {
		menuItems = append(menuItems, MenuItem{
			Label:       lang.GetText("menu.manage.migrate_label"),
			Description: lang.GetText("menu.manage.migrate_desc"),
			Command:     lang.GetText("menu.manage.migrate_cmd"),
			Action:      runBrainMigrationMenu,
		})
	}

	// Back button (always available)
	menuItems = append(menuItems, MenuItem{
		Label:       lang.GetText("menu.manage.back_label"),
		Description: lang.GetText("menu.manage.back_desc"),
		Command:     "",
		Action:      runInteractiveMenu,
	})

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
				// Get platform-specific common paths
				commonPaths := platform.GetCloudStoragePaths()

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
				// Get platform-specific common paths
				commonPaths := platform.GetCloudStoragePaths()

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

