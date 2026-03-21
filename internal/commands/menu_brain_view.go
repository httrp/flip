package commands

// menu_brain_view.go - View and Details Functions for Brain/Workspace Management
//
// Contains:
//   - runViewWorkspaceConfig()  View/edit workspace configuration file
//   - runWorkspaceDetails()     Workspace detail view with actions
//   - runBrainDetails()         Brain detail view with actions

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/git"
	"github.com/httrp/flip/internal/lang"
)

// runViewWorkspaceConfig shows and optionally edits the workspace configuration
func runViewWorkspaceConfig() error {
	displayStatusHeader()

	configPath, err := getConfigPath()
	if err != nil {
		fmt.Printf("\n%s Error: Failed to get config path: %v\n", IconError, err)
		return runManageResourcesMenu()
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("%s Workspace configuration not found\n", IconWarning)
		fmt.Printf("   Expected at: %s\n", configPath)
		fmt.Printf("\n%s Configuration will be created when you add your first workspace or brain.\n", IconInfo)
		return runManageResourcesMenu()
	}

	// Read and display config
	content, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("\n%s Error reading config: %v\n", IconError, err)
		return runManageResourcesMenu()
	}

	fmt.Println(strings.Repeat("━", 40))
	fmt.Println("📋 Workspace Configuration")
	fmt.Printf("File: %s\n", configPath)
	fmt.Println(strings.Repeat("━", 40))
	fmt.Println(string(content))
	fmt.Println(strings.Repeat("━", 40))

	fmt.Println("\nOptions:")
	fmt.Println("  e) Edit configuration (ADVANCED - BE CAREFUL!)")
	fmt.Println("  b) Back to menu")
	fmt.Printf("\nChoose (e/b) [default: b]: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(strings.ToLower(choice))

	if choice == "e" {
		fmt.Println("\n" + strings.Repeat("━", 40))
		fmt.Println("⚠️  DANGER: Manual Configuration Editing")
		fmt.Println(strings.Repeat("━", 40))
		fmt.Println("⚠️  Editing this file incorrectly can BREAK flip completely!")
		fmt.Println()
		fmt.Println("This file contains:")
		fmt.Println("  • All workspace definitions")
		fmt.Println("  • Brain paths and configurations")
		fmt.Println("  • Active workspace settings")
		fmt.Println()
		fmt.Println("Mistakes can cause:")
		fmt.Println("  ✗ Loss of access to all your brains")
		fmt.Println("  ✗ Broken workspace switching")
		fmt.Println("  ✗ Data inconsistency")
		fmt.Println()
		fmt.Println("Recommended: Use safe commands instead:")
		fmt.Println("  • flip workspace add/remove/switch")
		fmt.Println("  • flip brain add/remove/rename")
		fmt.Println("  • Menu options for all operations")
		fmt.Println(strings.Repeat("━", 40))
		fmt.Printf("\n⚠️  Are you ABSOLUTELY SURE you want to edit this? (type 'YES' to confirm): ")
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(confirm)

		if confirm == "YES" {
			// Create backup first
			backupPath := configPath + ".backup"
			if err := os.WriteFile(backupPath, content, 0644); err != nil {
				fmt.Printf("\n%s Warning: Could not create backup: %v\n", IconWarning, err)
			} else {
				fmt.Printf("\n%s Backup created: %s\n", IconCheck, backupPath)
			}

			// Try different editors
			editors := []string{"code", "nano", "vim", "vi"}
			opened := false
			for _, editor := range editors {
				cmd := exec.Command(editor, configPath)
				if editor == "code" {
					cmd.Start()
					fmt.Printf("\n%s Opening in VS Code...\n", IconCheck)
					fmt.Printf("   Backup: %s\n", backupPath)
					opened = true
					break
				} else {
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
			fmt.Println("\n✓ Cancelled. Good decision!")
		}
	}
	return runManageResourcesMenu()
}

// runWorkspaceDetails shows detailed information and actions for a specific workspace
func runWorkspaceDetails(workspace Workspace) func() error {
	return func() error {
		showBreadcrumb("Main › Manage › Workspaces › " + workspace.Name)
		fmt.Println()

		// Get current config to check if this workspace is active
		config, err := loadWorkspaceConfig()
		isActive := err == nil && config.ActiveWorkspace == workspace.Name

		// Display workspace details
		fmt.Printf("📁 Workspace: %s\n", workspace.Name)
		if workspace.Description != "" {
			fmt.Printf("   Description: %s\n", workspace.Description)
		}
		fmt.Printf("   Active: %t\n", isActive)
		fmt.Printf("   Brains: %d\n", len(workspace.Brains))
		if workspace.DefaultBrain != "" {
			fmt.Printf("   Default Brain: %s\n", workspace.DefaultBrain)
		}

		fmt.Println()

		// Build action menu
		menuItems := []MenuItem{
			{
				Label:       lang.GetText("menu.workspace_details.rename_label"),
				Description: lang.GetText("menu.workspace_details.rename_desc"),
				Command:     lang.GetText("menu.workspace_details.rename_command"),
				Action: func() error {
					fmt.Println("⚠️  Rename workspace not yet implemented")
					time.Sleep(2 * time.Second)
					return runWorkspaceDetails(workspace)()
				},
			},
			{
				Label:       lang.GetText("menu.workspace_details.remove_label"),
				Description: lang.GetText("menu.workspace_details.remove_desc"),
				Command:     lang.GetText("menu.workspace_details.remove_command"),
				Action: func() error {
					fmt.Println("⚠️  Remove workspace not yet implemented")
					time.Sleep(2 * time.Second)
					return runWorkspaceDetails(workspace)()
				},
			},
			{
				Label:       lang.GetText("menu.workspace_details.switch_label"),
				Description: lang.GetText("menu.workspace_details.switch_desc"),
				Command:     lang.GetText("menu.workspace_details.switch_command"),
				Action: func() error {
					// Switch to this workspace
					config, err := loadWorkspaceConfig()
					if err != nil {
						fmt.Printf("❌ Error loading config: %v\n", err)
						time.Sleep(2 * time.Second)
						return runWorkspaceDetails(workspace)()
					}
					config.ActiveWorkspace = workspace.Name
					if err := saveWorkspaceConfig(config); err != nil {
						fmt.Printf("❌ Error switching workspace: %v\n", err)
						time.Sleep(2 * time.Second)
						return runWorkspaceDetails(workspace)()
					}
					fmt.Printf("✅ Switched to workspace: %s\n", workspace.Name)
					time.Sleep(1 * time.Second)
					return nil
				},
			},
			{
				Label:       lang.GetText("menu.workspace_details.back_label"),
				Description: lang.GetText("menu.workspace_details.back_desc"),
				Command:     "",
				Action:      runManageResourcesMenu,
			},
		}

		idx, err := runMenuItemSelect("Workspace Actions", menuItems)
		if err != nil {
			return runManageResourcesMenu()
		}

		fmt.Println()
		return menuItems[idx].Action()
	}
}

// runBrainDetails shows detailed information and actions for a specific brain
func runBrainDetails(brainInfo Brain) func() error {
	return func() error {
		showBreadcrumb("Main › Manage › Brains › " + brainInfo.Name)
		fmt.Println()

		// Get workspace to check if this is the default brain
		workspace, _ := getActiveWorkspace()
		isDefault := workspace != nil && workspace.DefaultBrain == brainInfo.Name

		// Display brain details
		fmt.Printf("🧠 Brain: %s\n", brainInfo.Name)
		fmt.Printf("   Type: %s\n", brainInfo.Type)
		fmt.Printf("   Path: %s\n", brainInfo.Path)
		if brainInfo.Description != "" {
			fmt.Printf("   Description: %s\n", brainInfo.Description)
		}
		if isDefault {
			fmt.Printf("   Default: ⭐ Yes\n")
		} else {
			fmt.Printf("   Default: ○ No\n")
		}

		// Show git status
		if git.IsGitRepo(brainInfo.Path) {
			if git.HasUncommittedChanges(brainInfo.Path) {
				fmt.Printf("   Git: ● Has changes\n")
			} else {
				fmt.Printf("   Git: ✅ Clean\n")
			}
		}

		// Check brain type
		detector := brain.NewDetector()
		if result, err := detector.DetectBrainType(brainInfo.Path); err == nil {
			if result.Compatible {
				fmt.Printf("   Compatible: ✅ Yes\n")
			} else {
				fmt.Printf("   Compatible: ⚠️  %s\n", result.Description)
			}
		}

		fmt.Println()

		// Build action menu
		menuItems := []MenuItem{
			{
				Label:       lang.GetText("menu.brain_details.rename_label"),
				Description: lang.GetText("menu.brain_details.rename_desc"),
				Command:     lang.GetText("menu.brain_details.rename_command"),
				Action: func() error {
					fmt.Println("⚠️  Rename brain not yet implemented")
					time.Sleep(2 * time.Second)
					return runBrainDetails(brainInfo)()
				},
			},
			{
				Label:       lang.GetText("menu.edit_brain.set_default_label"),
				Description: lang.GetText("menu.edit_brain.set_default_desc"),
				Command:     "flip brain set-default",
				Action: func() error {
					// Set as default brain in workspace
					config, err := loadWorkspaceConfig()
					if err != nil {
						fmt.Printf("❌ Error loading config: %v\n", err)
						time.Sleep(2 * time.Second)
						return runBrainDetails(brainInfo)()
					}

					// Find active workspace and set default brain
					for i := range config.Workspaces {
						if config.Workspaces[i].Name == config.ActiveWorkspace {
							config.Workspaces[i].DefaultBrain = brainInfo.Name
							break
						}
					}

					if err := saveWorkspaceConfig(config); err != nil {
						fmt.Printf("❌ Error saving config: %v\n", err)
						time.Sleep(2 * time.Second)
						return runBrainDetails(brainInfo)()
					}

					fmt.Printf("✅ Set '%s' as default brain\n", brainInfo.Name)
					time.Sleep(1 * time.Second)
					return runBrainDetails(brainInfo)()
				},
			},
			{
				Label:       lang.GetText("menu.brain_details.remove_label"),
				Description: lang.GetText("menu.brain_details.remove_desc"),
				Command:     lang.GetText("menu.brain_details.remove_command"),
				Action: func() error {
					fmt.Println("⚠️  Remove brain not yet implemented")
					time.Sleep(2 * time.Second)
					return runBrainDetails(brainInfo)()
				},
			},
			{
				Label:       lang.GetText("menu.brain_details.repair_label"),
				Description: lang.GetText("menu.brain_details.repair_desc"),
				Command:     lang.GetText("menu.brain_details.repair_command"),
				Action: func() error {
					fmt.Println("⚠️  Brain repair not yet implemented in detail view")
					time.Sleep(2 * time.Second)
					return runBrainDetails(brainInfo)()
				},
			},
			{
				Label:       lang.GetText("menu.brain_details.health_label"),
				Description: lang.GetText("menu.brain_details.health_desc"),
				Command:     lang.GetText("menu.brain_details.health_command"),
				Action: func() error {
					// Show brain detection details
					detector := brain.NewDetector()
					if result, err := detector.DetectBrainType(brainInfo.Path); err == nil {
						fmt.Println("\nBrain Detection Results:")
						fmt.Printf("  Type: %s\n", result.Type)
						fmt.Printf("  Compatible: %t\n", result.Compatible)
						fmt.Printf("  Description: %s\n", result.Description)
						if len(result.Indicators) > 0 {
							fmt.Println("  Indicators:")
							for _, ind := range result.Indicators {
								fmt.Printf("    - %s\n", ind)
							}
						}
					} else {
						fmt.Printf("❌ Error checking brain: %v\n", err)
					}
					return runBrainDetails(brainInfo)()
				},
			},
			{
				Label:       lang.GetText("menu.brain_details.git_status_label"),
				Description: lang.GetText("menu.brain_details.git_status_desc"),
				Command:     lang.GetText("menu.brain_details.git_status_command"),
				Action: func() error {
					// Show git status
					if !git.IsGitRepo(brainInfo.Path) {
						fmt.Println("\n⚠️  Not a git repository")
						return runBrainDetails(brainInfo)()
					}

					fmt.Println("\nGit Status:")
					if git.HasUncommittedChanges(brainInfo.Path) {
						changes, err := git.GetChangedFiles(brainInfo.Path)
						if err == nil {
							fmt.Printf("  Modified files: %d\n", len(changes))
							for _, file := range changes {
								fmt.Printf("    - %s\n", file)
							}
						}
					} else {
						fmt.Println("  ✅ No uncommitted changes")
					}
					return runBrainDetails(brainInfo)()
				},
			},
			{
				Label:       lang.GetText("menu.brain_details.open_folder_label"),
				Description: lang.GetText("menu.brain_details.open_folder_desc"),
				Command:     lang.GetText("menu.brain_details.open_folder_command"),
				Action: func() error {
					// Open in file manager (cross-platform)
					if err := openInFileManager(brainInfo.Path); err != nil {
						fmt.Printf("❌ Error opening folder: %v\n", err)
						time.Sleep(2 * time.Second)
					} else {
						fmt.Println("✅ Opened in file manager")
						time.Sleep(1 * time.Second)
					}
					return runBrainDetails(brainInfo)()
				},
			},
			{
				Label:       lang.GetText("menu.brain_details.back_label"),
				Description: lang.GetText("menu.brain_details.back_desc"),
				Command:     "",
				Action:      runManageBrainsMenu,
			},
		}

		idx, err := runMenuItemSelect("Brain Actions", menuItems)
		if err != nil {
			return runManageBrainsMenu()
		}

		fmt.Println()
		return menuItems[idx].Action()
	}
}
