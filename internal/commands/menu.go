package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/lang"
	"github.com/httrp/flip/internal/tasks"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// getTerminalSize returns width and height of terminal, with safe defaults
func getTerminalSize() (width, height int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Safe defaults if we can't detect terminal size
		return 80, 24
	}
	return w, h
}

// truncateString truncates a string to fit in terminal width, leaving room for UI elements
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// createSimpleSelectTemplates creates templates without multi-line details
func createSimpleSelectTemplates() *promptui.SelectTemplates {
	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ . | cyan | bold }}",
		Inactive: "  {{ . }}",
		Selected: "{{ . | green | bold }}",
	}
}

// createMenuItemSelectTemplates creates templates for menu items with inline description
func createMenuItemSelectTemplates() *promptui.SelectTemplates {
	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ .Label | cyan | bold }}",
		Inactive: "  {{ .Label }}",
		Selected: "{{ .Label | green | bold }}",
	}
}

// calculateMenuSize determines optimal menu size based on terminal height
func calculateMenuSize(itemCount int) int {
	_, height := getTerminalSize()
	// Leave room for: title (2 lines), help text (2 lines), prompt (1 line), padding (3 lines)
	availableLines := height - 8
	if availableLines < 5 {
		availableLines = 5 // Minimum
	}
	if availableLines > 15 {
		availableLines = 15 // Maximum for usability
	}
	if itemCount < availableLines {
		return itemCount
	}
	return availableLines
}

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
	// Load banner from embedded files
	banner := lang.GetBanner()

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
			Label:       lang.GetText("menu.main.quickstart_label"),
			Description: lang.GetText("menu.main.quickstart_desc"),
			Action: func() error {
				if err := runQuickstart(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.main.browse_label"),
			Description: lang.GetText("menu.main.browse_desc"),
			Action:      runBrowseSearchMenu,
		},
		{
			Label:       lang.GetText("menu.main.create_label"),
			Description: lang.GetText("menu.main.create_desc"),
			Action:      runCreateNewMenu,
		},
		{
			Label:       lang.GetText("menu.main.manage_label"),
			Description: lang.GetText("menu.main.manage_desc"),
			Action:      runManageResourcesMenu,
		},
		{
			Label:       lang.GetText("menu.main.switch_label"),
			Description: lang.GetText("menu.main.switch_desc"),
			Action:      runSwitchContextMenu,
		},
		{
			Label:       lang.GetText("menu.main.status_label"),
			Description: lang.GetText("menu.main.status_desc"),
			Action: func() error {
				if err := runStatus(); err != nil {
					return err
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.main.help_label"),
			Description: lang.GetText("menu.main.help_desc"),
			Action: func() error {
				fmt.Println()
				fmt.Println(lang.GetTemplate("commands"))
				fmt.Println()
				fmt.Println("For detailed help: flip <command> --help")
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.main.exit_label"),
			Description: lang.GetText("menu.main.exit_desc"),
			Action: func() error {
				fmt.Println("\n👋 See you later!")
				return nil
			},
		},
	}

	// Create interactive select with improved rendering
	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     "What would you like to do?",
		Items:     menuItems,
		Templates: templates,
		Size:      calculateMenuSize(len(menuItems)),
		HideHelp:  true, // Hide "Use arrow keys" message
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
			Label:       lang.GetText("menu.create.workspace_label"),
			Description: lang.GetText("menu.create.workspace_desc"),
			Action: func() error {
				if err := runNewWorkspace(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.create.brain_label"),
			Description: lang.GetText("menu.create.brain_desc"),
			Action:      runBrainMenu,
		},
		{
			Label:       lang.GetText("menu.create.note_label"),
			Description: lang.GetText("menu.create.note_desc"),
			Action: func() error {
				if err := runCreateNote(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.create.meeting_label"),
			Description: lang.GetText("menu.create.meeting_desc"),
			Action: func() error {
				if err := runCreateMeeting(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.create.journal_label"),
			Description: lang.GetText("menu.create.journal_desc"),
			Action: func() error {
				if err := runCreateJournal(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.create.task_label"),
			Description: lang.GetText("menu.create.task_desc"),
			Action: func() error {
				// TODO: Implement task creation
				fmt.Println("\n🚧 Task creation coming soon!")
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.create.back_label"),
			Description: lang.GetText("menu.create.back_desc"),
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     "What would you like to create or add?",
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

// runBrowseSearchMenu shows submenu for browsing and searching notes
func runBrowseSearchMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       lang.GetText("menu.browse.recent_label"),
			Description: lang.GetText("menu.browse.recent_desc"),
			Action: func() error {
				if err := showRecentNotes(20); err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
				}
				return runBrowseSearchMenu()
			},
		},
		{
			Label:       lang.GetText("menu.browse.search_label"),
			Description: lang.GetText("menu.browse.search_desc"),
			Action: func() error {
				fmt.Print("\n🔍 Enter search query: ")
				var query string
				fmt.Scanln(&query)

				if query == "" {
					fmt.Println("No query entered")
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runBrowseSearchMenu()
				}

				// Ask if content search is needed
				contentPrompt := promptui.Select{
					Label:     lang.GetText("prompts.search_in"),
					Items:     []string{lang.GetText("prompts.search_filename_only"), lang.GetText("prompts.search_filename_content")},
					Templates: createSimpleSelectTemplates(),
					HideHelp:  true,
				}

				contentIdx, _, err := contentPrompt.Run()
				if err != nil {
					return runBrowseSearchMenu()
				}

				searchContent := contentIdx == 1

				if err := searchNotes(query, searchContent); err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
				}
				return runBrowseSearchMenu()
			},
		},
		{
			Label:       lang.GetText("menu.browse.tasks_label"),
			Description: lang.GetText("menu.browse.tasks_desc"),
			Action: func() error {
				if err := runTaskBrowseMenu(); err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
				}
				return runBrowseSearchMenu()
			},
		},
		{
			Label:       lang.GetText("menu.browse.back_label"),
			Description: lang.GetText("menu.browse.back_desc"),
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     "Browse & Search Notes",
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

// runCreateNewMenu shows submenu for creating new content
func runCreateNewMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       lang.GetText("menu.create.note_label"),
			Description: lang.GetText("menu.create.note_desc"),
			Action: func() error {
				if err := runCreateNote(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.create.meeting_label"),
			Description: lang.GetText("menu.create.meeting_desc"),
			Action: func() error {
				if err := runCreateMeeting(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.create.journal_label"),
			Description: lang.GetText("menu.create.journal_desc"),
			Action: func() error {
				if err := runCreateJournal(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.create.task_label"),
			Description: lang.GetText("menu.create.task_desc"),
			Action: func() error {
				if err := runCreateTask(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.create.back_label"),
			Description: lang.GetText("menu.create.back_desc"),
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     "Create New Content",
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

// runManageResourcesMenu shows submenu for managing workspaces, brains, and templates
func runManageResourcesMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       lang.GetText("menu.manage.workspaces_label"),
			Description: lang.GetText("menu.manage.workspaces_desc"),
			Action:      runEditWorkspaceMenu,
		},
		{
			Label:       lang.GetText("menu.manage.brains_label"),
			Description: lang.GetText("menu.manage.brains_desc"),
			Action:      runManageBrainsMenu,
		},
		{
			Label:       "View/Edit workspace config",
			Description: "View or edit the global workspace configuration",
			Action:      runViewWorkspaceConfig,
		},
		{
			Label:       lang.GetText("menu.manage.templates_label"),
			Description: lang.GetText("menu.manage.templates_desc"),
			Action:      runEditManageMenu, // Reuse existing template management
		},
		{
			Label:       lang.GetText("menu.manage.back_label"),
			Description: lang.GetText("menu.manage.back_desc"),
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     "Manage Resources",
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
				prompt := promptui.Prompt{
					Label: "Path to existing brain directory",
				}
				path, err := prompt.Run()
				if err != nil {
					return runInteractiveMenu()
				}
				if err := runDirectoryInit(path, "", false); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.manage_brains.scan_label"),
			Description: lang.GetText("menu.manage_brains.scan_desc"),
			Action: func() error {
				prompt := promptui.Prompt{
					Label:   "Path to scan (leave empty for home directory)",
					Default: "",
				}
				scanPath, err := prompt.Run()
				if err != nil {
					return runInteractiveMenu()
				}
				if scanPath == "" {
					scanPath = os.Getenv("HOME")
				}
				if err := runScan(scanPath); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
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
		Label:     "Manage Brains",
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
		Label:     "Switch Context",
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
			Label:       lang.GetText("menu.manage_brains.create_label"),
			Description: lang.GetText("menu.manage_brains.create_desc"),
			Action: func() error {
				if err := runNewBrain(); err != nil {
					fmt.Printf("\nError: %v\n", err)
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
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.manage_brains.scan_label"),
			Description: lang.GetText("menu.manage_brains.scan_desc"),
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
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.manage_brains.back_label"),
			Description: "Return to Create/Add menu",
			Action:      runCreateAddMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     "Brain Options",
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
		return runInteractiveMenu()
	}

	if len(config.Workspaces) == 0 {
		fmt.Println(lang.GetText("errors.no_workspaces"))
		fmt.Println(lang.GetText("prompts.continue"))
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
		Label:     "Select workspace to switch to",
		Items:     items,
		Size:      calculateMenuSize(len(items)),
		Templates: createSimpleSelectTemplates(),
		HideHelp:  true,
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

	fmt.Println(lang.GetText("prompts.continue"))
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
				return runInteractiveMenu()
			},
		},
		{
			Label:       lang.GetText("menu.edit_manage.back_label"),
			Description: lang.GetText("menu.edit_manage.back_desc"),
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     "What would you like to edit or manage?",
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
					Label: "New workspace name",
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
		Label:     "Workspace Management",
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
			Label:       "View brain details",
			Description: "Show configuration and details of a brain",
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

				brain := ws.Brains[idx]

				// Display brain details
				fmt.Println("\n" + strings.Repeat("━", 60))
				fmt.Printf("Brain: %s\n", brain.Name)
				fmt.Println(strings.Repeat("━", 60))
				fmt.Printf("Type:        %s\n", brain.Type)
				fmt.Printf("Description: %s\n", brain.Description)
				fmt.Printf("Path:        %s\n", brain.Path)

				// Check if path exists
				if stat, err := os.Stat(brain.Path); os.IsNotExist(err) {
					fmt.Printf("Status:      %s Path does not exist\n", IconError)
				} else if err != nil {
					fmt.Printf("Status:      %s Error accessing path: %v\n", IconError, err)
				} else if !stat.IsDir() {
					fmt.Printf("Status:      %s Path is not a directory\n", IconError)
				} else {
					fmt.Printf("Status:      %s Available\n", IconCheck)
				}

				if brain.Name == ws.DefaultBrain {
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
					configPath := filepath.Join(brain.Path, ".flip.yaml")
					if _, err := os.Stat(configPath); os.IsNotExist(err) {
						fmt.Printf("\n%s Brain config not found: %s\n", IconError, configPath)
					} else {
						content, err := os.ReadFile(configPath)
						if err != nil {
							fmt.Printf("\n%s Error reading config: %v\n", IconError, err)
						} else {
							fmt.Println("\n" + strings.Repeat("━", 60))
							fmt.Printf("Brain Config: %s\n", brain.Name)
							fmt.Printf("File: %s\n", configPath)
							fmt.Println(strings.Repeat("━", 60))
							fmt.Println(string(content))
							fmt.Println(strings.Repeat("━", 60))
						}
					}
				case "c":
					// Edit config file with warning
					configPath := filepath.Join(brain.Path, ".flip.yaml")
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
					fmt.Printf("\nCurrent path: %s\n", brain.Path)
					fmt.Printf("Enter new path (or press Enter to cancel): ")
					newPath, _ := reader.ReadString('\n')
					newPath = strings.TrimSpace(newPath)

					if newPath != "" && newPath != brain.Path {
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
											if config.Workspaces[i].Brains[j].Name == brain.Name {
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
					// Open folder
					if _, err := os.Stat(brain.Path); err == nil {
						exec.Command("open", brain.Path).Start()
						fmt.Printf("\n%s Opening folder...\n", IconCheck)
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

// runTaskBrowseMenu shows submenu for browsing and managing tasks
func runTaskBrowseMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       lang.GetText("menu.tasks.all_open_label"),
			Description: lang.GetText("menu.tasks.all_open_desc"),
			Action: func() error {
				filter := tasks.TaskFilter{
					Status: tasks.StatusOpen,
				}
				if err := runListTasks(filter, true); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runTaskBrowseMenu()
			},
		},
		{
			Label:       lang.GetText("menu.tasks.due_today_label"),
			Description: lang.GetText("menu.tasks.due_today_desc"),
			Action: func() error {
				filter := tasks.TaskFilter{
					DueToday: true,
				}
				if err := runListTasks(filter, true); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runTaskBrowseMenu()
			},
		},
		{
			Label:       lang.GetText("menu.tasks.due_week_label"),
			Description: lang.GetText("menu.tasks.due_week_desc"),
			Action: func() error {
				filter := tasks.TaskFilter{
					DueThisWeek: true,
				}
				if err := runListTasks(filter, true); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runTaskBrowseMenu()
			},
		},
		{
			Label:       lang.GetText("menu.tasks.overdue_label"),
			Description: lang.GetText("menu.tasks.overdue_desc"),
			Action: func() error {
				filter := tasks.TaskFilter{
					Overdue: true,
				}
				if err := runListTasks(filter, true); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runTaskBrowseMenu()
			},
		},
		{
			Label:       lang.GetText("menu.tasks.high_priority_label"),
			Description: lang.GetText("menu.tasks.high_priority_desc"),
			Action: func() error {
				filter := tasks.TaskFilter{
					Priority: tasks.PriorityHigh,
					Status:   tasks.StatusOpen,
				}
				if err := runListTasks(filter, true); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runTaskBrowseMenu()
			},
		},
		{
			Label:       lang.GetText("menu.tasks.stats_label"),
			Description: lang.GetText("menu.tasks.stats_desc"),
			Action: func() error {
				if err := runTaskStats(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runTaskBrowseMenu()
			},
		},
		{
			Label:       lang.GetText("menu.tasks.back_label"),
			Description: lang.GetText("menu.tasks.back_desc"),
			Action:      runBrowseSearchMenu,
		},
	}

	templates := createMenuItemSelectTemplates()

	selectMenu := promptui.Select{
		Label:     "Task Management",
		Items:     menuItems,
		Templates: templates,
		Size:      calculateMenuSize(len(menuItems)),
		HideHelp:  true,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runBrowseSearchMenu()
	}

	fmt.Println()
	return menuItems[idx].Action()
}

// runViewWorkspaceConfig shows and optionally edits the workspace configuration
func runViewWorkspaceConfig() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()

	configPath, err := getConfigPath()
	if err != nil {
		fmt.Printf("\n%s Error: Failed to get config path: %v\n", IconError, err)
		fmt.Println(lang.GetText("prompts.continue"))
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("%s Workspace configuration not found\n", IconWarning)
		fmt.Printf("   Expected at: %s\n", configPath)
		fmt.Printf("\n%s Configuration will be created when you add your first workspace or brain.\n", IconInfo)
		fmt.Println(lang.GetText("prompts.continue"))
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	// Read and display config
	content, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("\n%s Error reading config: %v\n", IconError, err)
		fmt.Println(lang.GetText("prompts.continue"))
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	fmt.Println(strings.Repeat("━", 60))
	fmt.Println("📋 Workspace Configuration")
	fmt.Printf("File: %s\n", configPath)
	fmt.Println(strings.Repeat("━", 60))
	fmt.Println(string(content))
	fmt.Println(strings.Repeat("━", 60))

	fmt.Println("\nOptions:")
	fmt.Println("  e) Edit configuration (ADVANCED - BE CAREFUL!)")
	fmt.Println("  b) Back to menu")
	fmt.Printf("\nChoose (e/b) [default: b]: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(strings.ToLower(choice))

	if choice == "e" {
		fmt.Println("\n" + strings.Repeat("━", 60))
		fmt.Println("⚠️  DANGER: Manual Configuration Editing")
		fmt.Println(strings.Repeat("━", 60))
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
		fmt.Println(strings.Repeat("━", 60))
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

	fmt.Println(lang.GetText("prompts.continue"))
	fmt.Scanln()
	return runManageResourcesMenu()
}
