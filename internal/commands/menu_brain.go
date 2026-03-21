package commands

// menu_brain.go - Core Brain & Workspace Menu Navigation
//
// Contains main menu navigation functions:
//   - runSwitchWorkspaceMenu() Switch between workspaces
//   - runEditManageMenu()      Edit/manage resources submenu
//
// Edit operations are in menu_brain_edit.go
// View/details functions are in menu_brain_view.go

import (
	"fmt"

	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
)


// runSwitchWorkspaceMenu shows menu to switch workspace
func runSwitchWorkspaceMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	config, err := loadWorkspaceConfig()
	if err != nil {
		fmt.Printf("Error loading workspaces: %v\n", err)
		fmt.Println(lang.GetText("prompts.continue"))
		fmt.Scanln()
		return nil
	}

	if len(config.Workspaces) == 0 {
		fmt.Println(lang.GetText("errors.no_workspaces"))
		fmt.Println(lang.GetText("prompts.continue"))
		fmt.Scanln()
		return nil
	}

	var items []string
	for _, ws := range config.Workspaces {
		label := ws.Name
		if ws.Name == config.ActiveWorkspace {
			label += " (active)"
		}
		items = append(items, label)
	}
	items = append(items, "◀️  Back to Main Menu")

	selectMenu := promptui.Select{
		Label:     "Select workspace to switch to",
		Items:     items,
		Size:      calculateMenuSize(len(items)),
		Templates: createSimpleSelectTemplates(),
		HideHelp:  true,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return nil
	}

	// Check if "Back" was selected
	if idx == len(items)-1 {
		return nil
	}

	selectedWS := config.Workspaces[idx]
	if selectedWS.Name == config.ActiveWorkspace {
		fmt.Printf("\n✓ Already in workspace '%s'\n", selectedWS.Name)
	} else {
		config.ActiveWorkspace = selectedWS.Name
		if err := saveWorkspaceConfig(config); err != nil {
			fmt.Printf("\nError switching workspace: %v\n", err)
		} else {
			fmt.Printf("\n✓ Switched to workspace '%s'\n", selectedWS.Name)
		}
	}

	fmt.Println(lang.GetText("prompts.continue"))
	fmt.Scanln()
	return nil
}

// runEditManageMenu shows submenu for editing and managing resources
func runEditManageMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       lang.GetText("menu.edit_manage.workspace_label"),
			Description: lang.GetText("menu.edit_manage.workspace_desc"),
			Action:      runEditWorkspaceMenu,
		},
		{
			Label:       lang.GetText("menu.edit_manage.brain_label"),
			Description: lang.GetText("menu.edit_manage.brain_desc"),
			Action:      runEditBrainMenu,
		},
		{
			Label:       lang.GetText("menu.edit_manage.templates_label"),
			Description: lang.GetText("menu.edit_manage.templates_desc"),
			Action: func() error {
				if err := runTemplateMenu(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return nil
			},
		},
		{
			Label:       lang.GetText("menu.edit_manage.back_label"),
			Description: lang.GetText("menu.edit_manage.back_desc"),
			Action: func() error { return nil },
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     lang.GetText("menu.titles.edit_manage"),
		Items:     menuItems,
		Templates: templates,
		Size:      calculateMenuSize(len(menuItems)),
		HideHelp:  true,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return nil
	}

	fmt.Println()
	return menuItems[idx].Action()
}
