package commands

// menu_status.go - Status, Help & VS Code Menu Functions
//
// Contains:
//   - runStatusMenu()   Status & Git operations submenu
//   - runVSCodeMenu()   VS Code integration submenu
//   - runAboutMenu()    About flip information
//   - runHelpMenu()     Help & documentation submenu

import (
	"fmt"
	"os"

	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
)

// runStatusMenu shows status and git operations menu
func runStatusMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	showBreadcrumb("Main", "Status & Git")

	menuItems := []MenuItem{
		{
			Label:       lang.GetText("menu.status.overview_label"),
			Description: lang.GetText("menu.status.overview_desc"),
			Command:     lang.GetText("menu.status.overview_cmd"),
			Action: func() error {
				if err := runStatus(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runStatusMenu()
			},
		},
		{
			Label:       lang.GetText("menu.status.git_status_label"),
			Description: lang.GetText("menu.status.git_status_desc"),
			Command:     lang.GetText("menu.status.git_status_cmd"),
			Action: func() error {
				if err := runBrainGitStatus(true); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runStatusMenu()
			},
		},
		{
			Label:       lang.GetText("menu.status.git_log_label"),
			Description: lang.GetText("menu.status.git_log_desc"),
			Command:     lang.GetText("menu.status.git_log_cmd"),
			Action: func() error {
				if err := runBrainGitLog(10, true); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runStatusMenu()
			},
		},
		{
			Label:       lang.GetText("menu.status.git_commit_label"),
			Description: lang.GetText("menu.status.git_commit_desc"),
			Command:     lang.GetText("menu.status.git_commit_cmd"),
			Action: func() error {
				checkUncommittedChangesOnExit()
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runStatusMenu()
			},
		},
		{
			Label:       lang.GetText("menu.status.git_pull_label"),
			Description: lang.GetText("menu.status.git_pull_desc"),
			Command:     lang.GetText("menu.status.git_pull_cmd"),
			Action: func() error {
				checkRemoteUpdatesOnStart()
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runStatusMenu()
			},
		},
		{
			Label:       lang.GetText("menu.status.vscode_label"),
			Description: lang.GetText("menu.status.vscode_desc"),
			Command:     lang.GetText("menu.status.vscode_cmd"),
			Action:      runVSCodeMenu,
		},
		{
			Label:       lang.GetText("menu.status.back_label"),
			Description: lang.GetText("menu.status.back_desc"),
			Command:     "",
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemWithCommandTemplates()

	selectMenu := promptui.Select{
		Label:     "Status & Git",
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

// runVSCodeMenu shows VS Code integration submenu
func runVSCodeMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	showBreadcrumb("Main", "Status & Git", "VS Code Integration")

	menuItems := []MenuItem{
		{
			Label:       lang.GetText("menu.vscode.install_label"),
			Description: lang.GetText("menu.vscode.install_desc"),
			Command:     lang.GetText("menu.vscode.install_cmd"),
			Action: func() error {
				if err := installTasksGlobal(); err != nil {
					fmt.Printf("\n%s Error: %v\n", IconError, err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runVSCodeMenu()
			},
		},
		{
			Label:       lang.GetText("menu.vscode.install_local_label"),
			Description: lang.GetText("menu.vscode.install_local_desc"),
			Command:     lang.GetText("menu.vscode.install_local_cmd"),
			Action: func() error {
				if err := installTasksLocal(); err != nil {
					fmt.Printf("\n%s Error: %v\n", IconError, err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runVSCodeMenu()
			},
		},
		{
			Label:       lang.GetText("menu.vscode.status_label"),
			Description: lang.GetText("menu.vscode.status_desc"),
			Command:     lang.GetText("menu.vscode.status_cmd"),
			Action: func() error {
				showVSCodeStatus()
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runVSCodeMenu()
			},
		},
		{
			Label:       lang.GetText("menu.vscode.uninstall_label"),
			Description: lang.GetText("menu.vscode.uninstall_desc"),
			Command:     lang.GetText("menu.vscode.uninstall_cmd"),
			Action: func() error {
				if err := uninstallTasksGlobal(); err != nil {
					fmt.Printf("\n%s Error: %v\n", IconError, err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runVSCodeMenu()
			},
		},
		{
			Label:       lang.GetText("menu.vscode.back_label"),
			Description: lang.GetText("menu.vscode.back_desc"),
			Command:     "",
			Action:      runStatusMenu,
		},
	}

	templates := createMenuItemWithCommandTemplates()

	selectMenu := promptui.Select{
		Label:     "VS Code Integration",
		Items:     menuItems,
		Templates: templates,
		Size:      calculateMenuSize(len(menuItems)),
		HideHelp:  true,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return runStatusMenu()
	}

	fmt.Println()
	return menuItems[idx].Action()
}

// runAboutMenu shows information about flip
func runAboutMenu() error {
	fmt.Println()

	// ASCII Art Banner
	banner := lang.GetBanner()
	fmt.Println(banner)
	fmt.Println()

	// Version and description
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Personal Knowledge Management Tool")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Key features
	fmt.Println("  📝 Standard Markdown - no proprietary formats")
	fmt.Println("  🧠 Multi-brain support - Obsidian, Logseq, Dendron, Foam")
	fmt.Println("  📂 Workspace organization - group related brains")
	fmt.Println("  🔄 Built-in Git integration")
	fmt.Println("  📊 Task & exercise tracking")
	fmt.Println("  🔀 Brain migration tools")
	fmt.Println()

	// Links
	fmt.Println("  📖 Docs:   flip help / flip intro / flip quickstart")
	fmt.Println("  🌐 GitHub: github.com/danorama-dh/flip")
	fmt.Println()

	// Stats
	config, err := loadWorkspaceConfig()
	if err == nil && len(config.Workspaces) > 0 {
		totalBrains := 0
		for _, ws := range config.Workspaces {
			totalBrains += len(ws.Brains)
		}
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("  Your setup: %d workspace(s), %d brain(s)\n", len(config.Workspaces), totalBrains)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
	}

	fmt.Println(lang.GetText("prompts.continue"))
	fmt.Scanln()
	return runInteractiveMenu()
}

// runHelpMenu shows help and documentation menu
func runHelpMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	showBreadcrumb("Main", "Help & Documentation")

	menuItems := []MenuItem{
		{
			Label:       lang.GetText("menu.help.quickstart_label"),
			Description: lang.GetText("menu.help.quickstart_desc"),
			Command:     lang.GetText("menu.help.quickstart_cmd"),
			Action: func() error {
				if err := runQuickstart(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runHelpMenu()
			},
		},
		{
			Label:       lang.GetText("menu.help.commands_label"),
			Description: lang.GetText("menu.help.commands_desc"),
			Command:     lang.GetText("menu.help.commands_cmd"),
			Action: func() error {
				fmt.Println()
				fmt.Println(lang.GetTemplate("commands"))
				fmt.Println()
				fmt.Println("For detailed help: flip <command> --help")
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runHelpMenu()
			},
		},
		{
			Label:       lang.GetText("menu.help.intro_label"),
			Description: lang.GetText("menu.help.intro_desc"),
			Command:     lang.GetText("menu.help.intro_cmd"),
			Action: func() error {
				if err := runIntro(); err != nil {
					fmt.Printf("\nError: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runHelpMenu()
			},
		},
		{
			Label:       lang.GetText("menu.help.config_label"),
			Description: lang.GetText("menu.help.config_desc"),
			Command:     lang.GetText("menu.help.config_cmd"),
			Action: func() error {
				configPath, err := getConfigPath()
				if err != nil {
					fmt.Printf("\n%s Error: %v\n", IconError, err)
				} else {
					fmt.Printf("\n📁 Configuration Location:\n")
					fmt.Printf("   %s\n", configPath)
					if _, err := os.Stat(configPath); os.IsNotExist(err) {
						fmt.Printf("\n%s File does not exist yet (will be created on first use)\n", IconWarning)
					}
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runHelpMenu()
			},
		},
		{
			Label:       lang.GetText("menu.help.back_label"),
			Description: lang.GetText("menu.help.back_desc"),
			Command:     "",
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemWithCommandTemplates()

	selectMenu := promptui.Select{
		Label:     "Help & Documentation",
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
