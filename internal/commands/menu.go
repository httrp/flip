package commands

import (
	"fmt"
	"os"
	"path/filepath"

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

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(banner)
	fmt.Println(getText("welcome_banner"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Show status header
	displayStatusHeader()
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
			Action: func() error {
				if err := runQuickstart(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "✨ Create New / Add",
			Description: "Create or add workspaces, brains, notes, tasks",
			Action:      runCreateAddMenu,
		},
		{
			Label:       "🔄 Switch Workspace",
			Description: "Switch to a different workspace",
			Action:      runSwitchWorkspaceMenu,
		},
		{
			Label:       "✏️  Edit / Manage",
			Description: "Edit, rename, repair workspaces and brains",
			Action:      runEditManageMenu,
		},
		{
			Label:       "📊 View Status",
			Description: "Show all workspaces and brains",
			Action: func() error {
				if err := runStatus(); err != nil {
					return err
				}
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
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
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
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

// runCreateAddMenu shows submenu for creating/adding resources
func runCreateAddMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       "📁 Workspace",
			Description: "Create a new workspace",
			Action: func() error {
				if err := runNewWorkspace(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "🧠 Brain",
			Description: "Create new or add existing brain",
			Action:      runBrainMenu,
		},
		{
			Label:       "📝 Note",
			Description: "Create a new note",
			Action: func() error {
				// TODO: Implement note creation
				fmt.Println("\n🚧 Note creation coming soon!")
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "📅 Meeting Note",
			Description: "Create a meeting note",
			Action: func() error {
				// TODO: Implement meeting note creation
				fmt.Println("\n🚧 Meeting note creation coming soon!")
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "✅ Task",
			Description: "Create a new task",
			Action: func() error {
				// TODO: Implement task creation
				fmt.Println("\n🚧 Task creation coming soon!")
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "◀️  Back to Main Menu",
			Description: "Return to main menu",
			Action:      runInteractiveMenu,
		},
	}

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
		Label:     "What would you like to create or add?",
		Items:     menuItems,
		Templates: templates,
		Size:      8,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runInteractiveMenu()
	}

	fmt.Println()
	return menuItems[idx].Action()
}

// runBrainMenu shows submenu for brain operations
func runBrainMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       "✨ Create New Brain",
			Description: "Create a brand new knowledge base",
			Action: func() error {
				if err := runNewBrain(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "📂 Init/Add Existing Brain",
			Description: "Initialize or add an existing brain to workspace",
			Action: func() error {
				home, _ := os.UserHomeDir()
				examples := []string{
					filepath.Join(home, "Documents", "Obsidian"),
					filepath.Join(home, "Notes"),
					filepath.Join(home, "Logseq"),
				}
				fmt.Println("\n💡 Example paths:")
				for _, ex := range examples {
					fmt.Printf("   %s\n", ex)
				}
				fmt.Println()

				promptPath := promptui.Prompt{
					Label: "Path to existing brain directory",
				}
				path, err := promptPath.Run()
				if err != nil {
					return runInteractiveMenu()
				}
				if err := runDirectoryInit(path, "", false); err != nil {
					fmt.Printf("\nError: %v\n", err)
				} else {
					fmt.Printf("\n✓ Brain initialized successfully at: %s\n", path)
				}
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "🔍 Scan for Brains",
			Description: "Scan directories for existing brains",
			Action: func() error {
				promptPath := promptui.Prompt{
					Label:   "Path to scan (leave empty for home directory)",
					Default: "",
				}
				path, err := promptPath.Run()
				if err != nil {
					return runInteractiveMenu()
				}
				if path == "" {
					home, _ := os.UserHomeDir()
					path = home
				}
				if err := runScan(path); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "◀️  Back",
			Description: "Return to Create/Add menu",
			Action:      runCreateAddMenu,
		},
	}

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
		Label:     "Brain Options",
		Items:     menuItems,
		Templates: templates,
		Size:      6,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runInteractiveMenu()
	}

	fmt.Println()
	return menuItems[idx].Action()
}

// runSwitchWorkspaceMenu shows menu to switch workspace
func runSwitchWorkspaceMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	config, err := loadWorkspaceConfig()
	if err != nil {
		fmt.Printf("Error loading workspaces: %v\n", err)
		fmt.Println("\nPress Enter to return to menu...")
		fmt.Scanln()
		return runInteractiveMenu()
	}

	if len(config.Workspaces) == 0 {
		fmt.Println("📭 No workspaces found. Create one first!")
		fmt.Println("\nPress Enter to return to menu...")
		fmt.Scanln()
		return runInteractiveMenu()
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
		Label: "Select workspace to switch to",
		Items: items,
		Size:  10,
		Templates: &promptui.SelectTemplates{
			Active:   "▸ {{ . | cyan | bold }}",
			Inactive: "  {{ . }}",
			Selected: "{{ . | green | bold }}",
		},
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runInteractiveMenu()
	}

	// Check if "Back" was selected
	if idx == len(items)-1 {
		return runInteractiveMenu()
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

	fmt.Println("\nPress Enter to return to menu...")
	fmt.Scanln()
	return runInteractiveMenu()
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
			Label:       "📁 Edit Workspace",
			Description: "Rename, repair, or remove workspaces",
			Action:      runEditWorkspaceMenu,
		},
		{
			Label:       "🧠 Edit Brain",
			Description: "Rename, repair, or remove brains",
			Action:      runEditBrainMenu,
		},
		{
			Label:       "◀️  Back to Main Menu",
			Description: "Return to main menu",
			Action:      runInteractiveMenu,
		},
	}

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
		Label:     "What would you like to edit or manage?",
		Items:     menuItems,
		Templates: templates,
		Size:      5,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runInteractiveMenu()
	}

	fmt.Println()
	return menuItems[idx].Action()
}

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
			Label:       "✏️  Rename Workspace",
			Description: "Change the name of a workspace",
			Action: func() error {
				config, err := loadWorkspaceConfig()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(config.Workspaces) == 0 {
					fmt.Println("\n📭 No workspaces found.")
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Select workspace to rename
				var wsNames []string
				for _, ws := range config.Workspaces {
					wsNames = append(wsNames, ws.Name)
				}

				selectWS := promptui.Select{
					Label: "Select workspace to rename",
					Items: wsNames,
					Size:  10,
				}

				idx, _, err := selectWS.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				oldName := wsNames[idx]

				// Don't allow renaming default
				if oldName == "default" {
					fmt.Println("\n❌ Cannot rename 'default' workspace - it's reserved")
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Get new name
				promptName := promptui.Prompt{
					Label: "New workspace name",
				}

				newName, err := promptName.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				// Check if there's a workspace directory that should be renamed
				// Look for common workspace directory patterns
				home, _ := os.UserHomeDir()
				possiblePaths := []string{
					filepath.Join(home, "flap", "workspaces", oldName),
					filepath.Join(home, "flip", "workspaces", oldName),
					filepath.Join(home, "Documents", "flip", "workspaces", oldName),
				}

				var workspaceDirPath string
				for _, path := range possiblePaths {
					if info, err := os.Stat(path); err == nil && info.IsDir() {
						workspaceDirPath = path
						break
					}
				}

				if workspaceDirPath != "" {
					suggestedPath := normalizePathName(newName)
					currentDirName := filepath.Base(workspaceDirPath)

					if currentDirName != suggestedPath {
						promptRenameDir := promptui.Prompt{
							Label:     fmt.Sprintf("Also rename workspace directory '%s' to '%s'? (yes/no)", currentDirName, suggestedPath),
							IsConfirm: true,
						}

						_, errConfirm := promptRenameDir.Run()
						renameDir := errConfirm == nil

						if renameDir {
							parentDir := filepath.Dir(workspaceDirPath)
							newPath := filepath.Join(parentDir, suggestedPath)

							if err := os.Rename(workspaceDirPath, newPath); err != nil {
								fmt.Printf("\n%s Warning: Could not rename workspace directory: %v\n", IconWarning, err)
								fmt.Println("   Continuing with name change in config only...")
							} else {
								fmt.Printf("\n%s Workspace directory renamed: %s -> %s\n", IconCheck, currentDirName, suggestedPath)
							}
						}
					}
				}

				if err := runWorkspaceRename(oldName, newName); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}

				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "🔧 Repair Workspaces",
			Description: "Validate paths and re-detect brain types in all workspaces",
			Action: func() error {
				if err := runWorkspaceRepair(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "🗑️  Remove Workspace",
			Description: "Remove a workspace (brains stay intact)",
			Action: func() error {
				config, err := loadWorkspaceConfig()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(config.Workspaces) == 0 {
					fmt.Println("\n📭 No workspaces found.")
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Select workspace to remove
				var wsNames []string
				for _, ws := range config.Workspaces {
					wsNames = append(wsNames, ws.Name)
				}

				selectWS := promptui.Select{
					Label: "Select workspace to remove",
					Items: wsNames,
					Size:  10,
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
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if err := runWorkspaceRemove(wsName); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}

				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "◀️  Back",
			Description: "Return to Edit/Manage menu",
			Action:      runEditManageMenu,
		},
	}

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
		Label:     "Workspace Management",
		Items:     menuItems,
		Templates: templates,
		Size:      6,
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
			Label:       "✏️  Rename Brain",
			Description: "Change the name of a brain in active workspace",
			Action: func() error {
				ws, err := getActiveWorkspace()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(ws.Brains) == 0 {
					fmt.Println("\n📭 No brains in active workspace.")
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Select brain to rename
				var brainNames []string
				for _, b := range ws.Brains {
					brainNames = append(brainNames, b.Name)
				}

				selectBrain := promptui.Select{
					Label: "Select brain to rename",
					Items: brainNames,
					Size:  10,
				}

				idx, _, err := selectBrain.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				oldName := brainNames[idx]
				oldBrain := ws.Brains[idx]

				// Get new name
				promptName := promptui.Prompt{
					Label: "New brain name",
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

				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "⭐ Set Default Brain",
			Description: "Set the default brain for active workspace",
			Action: func() error {
				ws, err := getActiveWorkspace()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(ws.Brains) == 0 {
					fmt.Println("\n📭 No brains in active workspace.")
					fmt.Println("\nPress Enter to return to menu...")
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
					Label: "Select brain to set as default",
					Items: items,
					Size:  10,
				}

				idx, _, err := selectBrain.Run()
				if err != nil {
					return runInteractiveMenu()
				}

				brainName := ws.Brains[idx].Name

				if err := runBrainSetDefault(brainName); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}

				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "🔧 Repair Brains",
			Description: "Validate paths and re-detect types in active workspace",
			Action: func() error {
				if err := runBrainRepair(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "🗑️  Remove Brain",
			Description: "Remove a brain from active workspace (files stay intact)",
			Action: func() error {
				ws, err := getActiveWorkspace()
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if len(ws.Brains) == 0 {
					fmt.Println("\n📭 No brains in active workspace.")
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				// Select brain to remove
				var brainNames []string
				for _, b := range ws.Brains {
					brainNames = append(brainNames, b.Name)
				}

				selectBrain := promptui.Select{
					Label: "Select brain to remove",
					Items: brainNames,
					Size:  10,
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
					fmt.Println("\nPress Enter to return to menu...")
					fmt.Scanln()
					return runInteractiveMenu()
				}

				if err := runBrainRemove(brainName); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}

				fmt.Println("\nPress Enter to return to menu...")
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       "◀️  Back",
			Description: "Return to Edit/Manage menu",
			Action:      runEditManageMenu,
		},
	}

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
		Label:     "Brain Management",
		Items:     menuItems,
		Templates: templates,
		Size:      7,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runInteractiveMenu()
	}

	fmt.Println()
	return menuItems[idx].Action()
}
