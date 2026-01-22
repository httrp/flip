package commands

// menu_brain_edit.go - Edit Operations for Brain/Workspace Management
//
// Contains:
//   - runEditWorkspaceMenu()    Workspace editing operations
//   - runEditBrainMenu()        Brain editing operations

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
)

// runEditWorkspaceMenu shows menu for editing workspaces
func runEditWorkspaceMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       lang.GetText("menu.edit_workspace.details_label"),
			Description: lang.GetText("menu.edit_workspace.details_desc"),
			Action: func() error {
				config, err := loadWorkspaceConfig()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runManageResourcesMenu()
				}

				if len(config.Workspaces) == 0 {
					fmt.Println("\n📭 No workspaces found.")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runManageResourcesMenu()
				}

				// Select workspace to view with status indicators
				var wsNames []string
				for _, ws := range config.Workspaces {
					indicator := "  "
					if config.ActiveWorkspace == ws.Name {
						indicator = "⭐"
					}
					wsNames = append(wsNames, fmt.Sprintf("%s %s (%d brains)", indicator, ws.Name, len(ws.Brains)))
				}

				selectWS := promptui.Select{
					Label:     "Select workspace to view",
					Items:     wsNames,
					Size:      calculateMenuSize(len(wsNames)),
					Templates: createSimpleSelectTemplates(),
					HideHelp:  true,
				}

				idx, _, err := selectWS.Run()
				if err != nil {
					return runManageResourcesMenu()
				}

				// Show detail view
				return runWorkspaceDetails(config.Workspaces[idx])()
			},
		},
		{
			Label:       lang.GetText("menu.edit_workspace.rename_label"),
			Description: lang.GetText("menu.edit_workspace.rename_desc"),
			Action: func() error {
				config, err := loadWorkspaceConfig()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(config.Workspaces) == 0 {
					fmt.Println("\n📭 No workspaces found.")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Select workspace to rename
				var wsNames []string
				for _, ws := range config.Workspaces {
					wsNames = append(wsNames, ws.Name)
				}

				selectWS := promptui.Select{
					Label:     lang.GetText("prompts.select_workspace_rename"),
					Items:     wsNames,
					Size:      calculateMenuSize(len(wsNames)),
					Templates: createSimpleSelectTemplates(),
					HideHelp:  true,
				}

				idx, _, err := selectWS.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				oldName := wsNames[idx]

				// Don't allow renaming default
				if oldName == "default" {
					fmt.Println("\n❌ Cannot rename 'default' workspace - it's reserved")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Get new name
				promptName := promptui.Prompt{
					Label: lang.GetText("prompts.workspace_name"),
				}

				newName, err := promptName.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				if err := runWorkspaceRename(oldName, newName); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}

				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.edit_workspace.repair_label"),
			Description: lang.GetText("menu.edit_workspace.repair_desc"),
			Action: func() error {
				if err := runWorkspaceRepair(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.edit_workspace.remove_label"),
			Description: lang.GetText("menu.edit_workspace.remove_desc"),
			Action: func() error {
				config, err := loadWorkspaceConfig()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(config.Workspaces) == 0 {
					fmt.Println("\n📭 No workspaces found.")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Select workspace to remove
				var wsNames []string
				for _, ws := range config.Workspaces {
					wsNames = append(wsNames, ws.Name)
				}

				selectWS := promptui.Select{
					Label:     lang.GetText("prompts.select_workspace_remove"),
					Items:     wsNames,
					Size:      calculateMenuSize(len(wsNames)),
					Templates: createSimpleSelectTemplates(),
					HideHelp:  true,
				}

				idx, _, err := selectWS.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				wsName := wsNames[idx]

				// Confirm removal
				promptConfirm := promptui.Prompt{
					Label:     fmt.Sprintf("Remove workspace '%s'? (yes/no)", wsName),
					IsConfirm: true,
				}

				_, err = promptConfirm.Run()
				if err != nil {
					fmt.Println("\n❌ Cancelled")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if err := runWorkspaceRemove(wsName); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}

				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.edit_workspace.back_label"),
			Description: lang.GetText("menu.edit_workspace.back_desc"),
			Action:      runEditManageMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     lang.GetText("menu.titles.workspace_management"),
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

// runEditBrainMenu shows menu for editing brains
func runEditBrainMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       lang.GetText("menu.edit_brain.details_label"),
			Description: lang.GetText("menu.edit_brain.details_desc"),
			Action: func() error {
				ws, err := getActiveWorkspace()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runEditBrainMenu()
				}

				if len(ws.Brains) == 0 {
					fmt.Println(lang.GetText("errors.no_brains"))
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runEditBrainMenu()
				}

				// Select brain to view
				var brainNames []string
				for _, b := range ws.Brains {
					label := b.Name
					if b.Name == ws.DefaultBrain {
						label += " (default)"
					}
					brainNames = append(brainNames, label)
				}

				selectBrain := promptui.Select{
					Label:     "Select brain to view details",
					Items:     brainNames,
					Size:      calculateMenuSize(len(brainNames)),
					Templates: createSimpleSelectTemplates(),
					HideHelp:  true,
				}

				idx, _, err := selectBrain.Run()
				if err != nil {
					return runEditBrainMenu()
				}

				brainItem := ws.Brains[idx]

				// Display brain details
				fmt.Println("\n" + strings.Repeat("━", 60))
				fmt.Printf("Brain: %s\n", brainItem.Name)
				fmt.Println(strings.Repeat("━", 60))
				fmt.Printf("Type:        %s\n", brainItem.Type)
				fmt.Printf("Description: %s\n", brainItem.Description)
				fmt.Printf("Path:        %s\n", brainItem.Path)

				// Check if path exists
				if stat, err := os.Stat(brainItem.Path); os.IsNotExist(err) {
					fmt.Printf("Status:      %s Path does not exist\n", IconError)
				} else if err != nil {
					fmt.Printf("Status:      %s Error accessing path: %v\n", IconError, err)
				} else if !stat.IsDir() {
					fmt.Printf("Status:      %s Path is not a directory\n", IconError)
				} else {
					fmt.Printf("Status:      %s Available\n", IconCheck)
				}

				if brainItem.Name == ws.DefaultBrain {
					fmt.Printf("Default:     %s Yes\n", IconDefault)
				} else {
					fmt.Printf("Default:     No\n")
				}

				fmt.Println(strings.Repeat("━", 60))
				fmt.Println("\nOptions:")
				fmt.Println("  v) View brain config (.flip.yaml)")
				fmt.Println("  e) Edit path")
				fmt.Println("  o) Open folder")
				fmt.Println("  c) Edit config file (ADVANCED)")
				fmt.Println("  b) Back to menu")
				fmt.Printf("\nChoose (v/e/o/c/b) [default: b]: ")

				reader := bufio.NewReader(os.Stdin)
				choice, _ := reader.ReadString('\n')
				choice = strings.TrimSpace(strings.ToLower(choice))

				switch choice {
				case "v":
					// View brain config
					configPath := filepath.Join(brainItem.Path, ".flip.yaml")
					if _, err := os.Stat(configPath); os.IsNotExist(err) {
						fmt.Printf("\n%s Brain config not found: %s\n", IconError, configPath)
					} else {
						content, err := os.ReadFile(configPath)
						if err != nil {
							fmt.Printf("\n%s Error reading config: %v\n", IconError, err)
						} else {
							fmt.Println("\n" + strings.Repeat("━", 60))
							fmt.Printf("Brain Config: %s\n", brainItem.Name)
							fmt.Printf("File: %s\n", configPath)
							fmt.Println(strings.Repeat("━", 60))
							fmt.Println(string(content))
							fmt.Println(strings.Repeat("━", 60))
						}
					}
				case "c":
					// Edit config file with warning
					configPath := filepath.Join(brainItem.Path, ".flip.yaml")
					if _, err := os.Stat(configPath); os.IsNotExist(err) {
						fmt.Printf("\n%s Brain config not found: %s\n", IconError, configPath)
					} else {
						fmt.Println("\n" + strings.Repeat("━", 60))
						fmt.Println("⚠️  WARNING: Manual Config Editing")
						fmt.Println(strings.Repeat("━", 60))
						fmt.Println("Editing config files manually can break your brain setup!")
						fmt.Println()
						fmt.Println("Recommended: Use 'flip' commands instead:")
						fmt.Println("  • flip brain rename <name>")
						fmt.Println("  • flip brain repair")
						fmt.Println("  • Use menu options for safe changes")
						fmt.Println()
						fmt.Println("Only proceed if you know what you're doing.")
						fmt.Println(strings.Repeat("━", 60))
						fmt.Printf("\n? Open config in editor anyway? (yes/no) [default: no]: ")
						confirm, _ := reader.ReadString('\n')
						confirm = strings.TrimSpace(strings.ToLower(confirm))

						if confirm == "yes" || confirm == "y" {
							// Try different editors
							editors := []string{"code", "nano", "vim", "vi"}
							opened := false
							for _, editor := range editors {
								cmd := exec.Command(editor, configPath)
								if editor == "code" {
									// VS Code: just launch and return
									cmd.Start()
									fmt.Printf("\n%s Opening in VS Code...\n", IconCheck)
									opened = true
									break
								} else {
									// Terminal editors: run interactively
									cmd.Stdin = os.Stdin
									cmd.Stdout = os.Stdout
									cmd.Stderr = os.Stderr
									if err := cmd.Run(); err == nil {
										opened = true
										break
									}
								}
							}
							if !opened {
								fmt.Printf("\n%s Could not find a suitable editor\n", IconError)
								fmt.Printf("   Config file: %s\n", configPath)
							}
						} else {
							fmt.Println("\n✓ Cancelled. Smart choice!")
						}
					}
				case "e":
					// Edit path
					fmt.Printf("\nCurrent path: %s\n", brainItem.Path)
					fmt.Printf("Enter new path (or press Enter to cancel): ")
					newPath, _ := reader.ReadString('\n')
					newPath = strings.TrimSpace(newPath)

					if newPath != "" && newPath != brainItem.Path {
						// Validate new path
						absPath, err := filepath.Abs(newPath)
						if err != nil {
							fmt.Printf("\n%s Error: Invalid path: %v\n", IconError, err)
						} else if _, err := os.Stat(absPath); os.IsNotExist(err) {
							fmt.Printf("\n%s Error: Path does not exist: %s\n", IconError, absPath)
						} else {
							// Update config
							config, err := loadWorkspaceConfig()
							if err == nil {
								for i := range config.Workspaces {
									if config.Workspaces[i].Name == ws.Name {
										for j := range config.Workspaces[i].Brains {
											if config.Workspaces[i].Brains[j].Name == brainItem.Name {
												config.Workspaces[i].Brains[j].Path = absPath
												if err := saveWorkspaceConfig(config); err != nil {
													fmt.Printf("\n%s Error saving config: %v\n", IconError, err)
												} else {
													fmt.Printf("\n%s Brain path updated successfully\n", IconCheck)
												}
												break
											}
										}
										break
									}
								}
							}
						}
					}
				case "o":
					// Open folder (cross-platform)
					if _, err := os.Stat(brainItem.Path); err == nil {
						if err := openInFileManager(brainItem.Path); err != nil {
							fmt.Printf("\n%s Error opening folder: %v\n", IconError, err)
						} else {
							fmt.Printf("\n%s Opening folder...\n", IconCheck)
						}
					} else {
						fmt.Printf("\n%s Error: Cannot open folder - path does not exist\n", IconError)
					}
				}

				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runEditBrainMenu()
			},
		},
		{
			Label:       lang.GetText("menu.edit_brain.rename_label"),
			Description: lang.GetText("menu.edit_brain.rename_desc"),
			Action: func() error {
				ws, err := getActiveWorkspace()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(ws.Brains) == 0 {
					fmt.Println(lang.GetText("errors.no_brains"))
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Select brain to rename
				var brainNames []string
				for _, b := range ws.Brains {
					brainNames = append(brainNames, b.Name)
				}

				selectBrain := promptui.Select{
					Label:     lang.GetText("prompts.select_brain_rename"),
					Items:     brainNames,
					Size:      calculateMenuSize(len(brainNames)),
					Templates: createSimpleSelectTemplates(),
					HideHelp:  true,
				}

				idx, _, err := selectBrain.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				oldName := brainNames[idx]
				oldBrain := ws.Brains[idx]

				// Get new name
				promptName := promptui.Prompt{
					Label: lang.GetText("prompts.brain_name"),
				}

				newName, err := promptName.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				// Ask if directory should also be renamed
				suggestedPath := normalizePathName(newName)
				currentDirName := filepath.Base(oldBrain.Path)

				if currentDirName != suggestedPath {
					promptRenameDir := promptui.Prompt{
						Label:     fmt.Sprintf("Also rename directory '%s' to '%s'? (yes/no)", currentDirName, suggestedPath),
						IsConfirm: true,
					}

					_, errConfirm := promptRenameDir.Run()
					renameDir := errConfirm == nil

					if renameDir {
						// Rename the directory
						parentDir := filepath.Dir(oldBrain.Path)
						newPath := filepath.Join(parentDir, suggestedPath)

						if err := os.Rename(oldBrain.Path, newPath); err != nil {
							fmt.Printf("\n%s Warning: Could not rename directory: %v\n", IconWarning, err)
							fmt.Println("   Continuing with name change in config only...")
						} else {
							fmt.Printf("\n%s Directory renamed: %s -> %s\n", IconCheck, currentDirName, suggestedPath)

							// Update path in config before calling runBrainRename
							config, err := loadWorkspaceConfig()
							if err == nil {
								for i := range config.Workspaces {
									for j := range config.Workspaces[i].Brains {
										if config.Workspaces[i].Brains[j].Path == oldBrain.Path {
											config.Workspaces[i].Brains[j].Path = newPath
										}
									}
								}
								saveWorkspaceConfig(config)
							}
						}
					}
				}

				if err := runBrainRename(oldName, newName); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}

				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.edit_brain.default_label"),
			Description: lang.GetText("menu.edit_brain.default_desc"),
			Action: func() error {
				ws, err := getActiveWorkspace()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(ws.Brains) == 0 {
					fmt.Println(lang.GetText("errors.no_brains"))
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Select brain to set as default
				var items []string
				for _, b := range ws.Brains {
					label := b.Name
					if b.Name == ws.DefaultBrain {
						label += " (current default)"
					}
					items = append(items, label)
				}

				selectBrain := promptui.Select{
					Label:     lang.GetText("prompts.select_brain_default_set"),
					Items:     items,
					Size:      calculateMenuSize(len(items)),
					Templates: createSimpleSelectTemplates(),
					HideHelp:  true,
				}

				idx, _, err := selectBrain.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				brainName := ws.Brains[idx].Name

				if err := runBrainSetDefault(brainName); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}

				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.edit_brain.repair_label"),
			Description: lang.GetText("menu.edit_brain.repair_desc"),
			Action: func() error {
				if err := runBrainRepair(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.edit_brain.remove_label"),
			Description: lang.GetText("menu.edit_brain.remove_desc"),
			Action: func() error {
				ws, err := getActiveWorkspace()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(ws.Brains) == 0 {
					fmt.Println(lang.GetText("errors.no_brains"))
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Select brain to remove
				var brainNames []string
				for _, b := range ws.Brains {
					brainNames = append(brainNames, b.Name)
				}

				selectBrain := promptui.Select{
					Label:     lang.GetText("prompts.select_brain_remove"),
					Items:     brainNames,
					Size:      calculateMenuSize(len(brainNames)),
					Templates: createSimpleSelectTemplates(),
					HideHelp:  true,
				}

				idx, _, err := selectBrain.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				brainName := brainNames[idx]

				// Confirm removal
				promptConfirm := promptui.Prompt{
					Label:     fmt.Sprintf("Remove brain '%s' from workspace? (yes/no)", brainName),
					IsConfirm: true,
				}

				_, err = promptConfirm.Run()
				if err != nil {
					fmt.Println("\n❌ Cancelled")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if err := runBrainRemove(brainName); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}

				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.status.git_status_label"),
			Description: lang.GetText("menu.status.git_status_desc"),
			Action: func() error {
				if err := runBrainGitStatus(false); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runEditBrainMenu()
			},
		},
		{
			Label:       lang.GetText("menu.status.git_log_label"),
			Description: lang.GetText("menu.status.git_log_desc"),
			Action: func() error {
				if err := runBrainGitLog(10, false); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runEditBrainMenu()
			},
		},
		{
			Label:       lang.GetText("menu.status.git_commit_label"),
			Description: lang.GetText("menu.status.git_commit_desc"),
			Action: func() error {
				checkUncommittedChangesOnExit()
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runEditBrainMenu()
			},
		},
		{
			Label:       lang.GetText("menu.status.git_pull_label"),
			Description: lang.GetText("menu.status.git_pull_desc"),
			Action: func() error {
				checkRemoteUpdatesOnStart()
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runEditBrainMenu()
			},
		},
		{
			Label:       lang.GetText("menu.edit_brain.back_label"),
			Description: lang.GetText("menu.edit_brain.back_desc"),
			Action:      runEditManageMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     "Brain Management",
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
