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
	
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(banner)
	fmt.Println(getText("welcome_banner"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Show current status
	config, err := loadWorkspaceConfig()
	if err == nil && len(config.Workspaces) > 0 {
		if config.ActiveWorkspace != "" {
			fmt.Printf("📂 Active Workspace: %s\n\n", config.ActiveWorkspace)
		}
	}

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
	fmt.Println("━━━ Create New / Add ━━━")
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
	fmt.Println("━━━ Brain Options ━━━")
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
	fmt.Println("━━━ Switch Workspace ━━━")
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
