package commands

// menu_new.go - Bubbletea-based menu system
//
// Builds all menu definitions with feature-flag awareness and passes
// them to the generic ui.RunApp(). After the user selects an action,
// executeCommandV2() dispatches to the appropriate command.

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/httrp/flip/internal/git"
	"github.com/httrp/flip/internal/lang"
	"github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// NewMenuV2Command creates the menu command (bubbletea-based)
func NewMenuV2Command() *cobra.Command {
	return &cobra.Command{
		Use:   "menu",
		Short: "Interactive main menu",
		Long:  "Start flip with the interactive bubbletea-based menu",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInteractiveMenuV2()
		},
	}
}

// ── Helpers ──────────────────────────────────────────────────────────

// menuNav returns a MenuItem that navigates to a named submenu.
func menuNav(label, desc, cmd, target string) ui.MenuItem {
	return ui.NewMenuItem(label, desc, cmd, func() tea.Cmd {
		return func() tea.Msg { return ui.NavigateMsg{Menu: target} }
	})
}

// menuExec returns a MenuItem that exits the TUI and executes a command.
func menuExec(label, desc, cmd, command string) ui.MenuItem {
	return ui.NewMenuItem(label, desc, cmd, func() tea.Cmd {
		return func() tea.Msg { return ui.ExecCommandMsg{Command: command} }
	})
}

// menuBack returns a "← Back" menu item.
func menuBack() ui.MenuItem {
	return ui.NewMenuItem("← Back", "Return to previous menu", "", func() tea.Cmd {
		return func() tea.Msg { return ui.BackMsg{} }
	})
}

// menuQuit returns an exit menu item.
func menuQuit() ui.MenuItem {
	return ui.NewMenuItem(
		lang.GetText("menu.main.exit_label"),
		lang.GetText("menu.main.exit_desc"),
		lang.GetText("menu.main.exit_cmd"),
		func() tea.Cmd {
			return func() tea.Msg { return ui.QuitMsg{} }
		},
	)
}

// ── Menu Builders ───────────────────────────────────────────────────

func buildMenuConfig() ui.MenuConfig {
	menus := make(map[string]ui.MenuDef)

	menus["main"] = buildMainMenuDef()
	menus["create"] = buildCreateMenuDef()
	menus["browse"] = buildBrowseMenuDef()
	menus["manage"] = buildManageMenuDef()
	menus["status"] = buildStatusMenuDef()
	menus["help"] = buildHelpMenuDef()

	if IsEnabled(FeatureVSCode) {
		menus["vscode"] = buildVSCodeMenuDef()
	}
	if IsEnabled(FeatureExercises) {
		menus["exercises"] = buildExercisesMenuDef()
	}

	return ui.MenuConfig{Menus: menus}
}

func buildMainMenuDef() ui.MenuDef {
	items := []ui.MenuItem{
		menuNav(lang.GetText("menu.main.create_label"),
			lang.GetText("menu.main.create_desc"),
			lang.GetText("menu.main.create_cmd"), "create"),
		menuNav(lang.GetText("menu.main.browse_label"),
			lang.GetText("menu.main.browse_desc"),
			lang.GetText("menu.main.browse_cmd"), "browse"),
		menuNav(lang.GetText("menu.main.manage_label"),
			lang.GetText("menu.main.manage_desc"),
			lang.GetText("menu.main.manage_cmd"), "manage"),
		menuNav(lang.GetText("menu.main.status_label"),
			lang.GetText("menu.main.status_desc"),
			lang.GetText("menu.main.status_cmd"), "status"),
		menuNav(lang.GetText("menu.main.help_label"),
			lang.GetText("menu.main.help_desc"),
			lang.GetText("menu.main.help_cmd"), "help"),
		menuExec("ℹ️  About", "About flip, version info, and credits", "flip about", "about"),
		menuQuit(),
	}
	return ui.MenuDef{Title: "flip", Breadcrumb: []string{"Main Menu"}, Items: items, Parent: ""}
}

func buildCreateMenuDef() ui.MenuDef {
	var items []ui.MenuItem

	if IsEnabled(FeatureMeetings) {
		items = append(items,
			menuExec(lang.GetText("menu.create.note_label"),
				lang.GetText("menu.create.note_desc"),
				lang.GetText("menu.create.note_cmd"), "note"),
			menuExec("⚡ Quick Note", "Quick note with minimal prompts", "flip quicknote", "quicknote"),
			menuExec(lang.GetText("menu.create.meeting_label"),
				lang.GetText("menu.create.meeting_desc"),
				lang.GetText("menu.create.meeting_cmd"), "meeting"),
			menuExec(lang.GetText("menu.create.journal_label"),
				lang.GetText("menu.create.journal_desc"),
				lang.GetText("menu.create.journal_cmd"), "journal"),
		)
	}

	if IsEnabled(FeatureTasks) {
		items = append(items,
			menuExec(lang.GetText("menu.create.task_label"),
				lang.GetText("menu.create.task_desc"),
				lang.GetText("menu.create.task_cmd"), "task"),
		)
	}

	if IsEnabled(FeatureExercises) {
		items = append(items,
			menuExec("🏋️  New Exercise", "Create a new repeatable exercise", "flip exercise new", "exercise-new"),
			menuExec("📋 New Exercise Plan", "Create a structured training plan", "flip exercise plan new", "exercise-plan-new"),
		)
	}

	if IsEnabled(FeatureDefinitions) {
		items = append(items,
			menuExec(lang.GetText("menu.create.organization_label"),
				lang.GetText("menu.create.organization_desc"),
				lang.GetText("menu.create.organization_cmd"), "def-org"),
			menuExec(lang.GetText("menu.create.project_label"),
				lang.GetText("menu.create.project_desc"),
				lang.GetText("menu.create.project_cmd"), "def-project"),
			menuExec(lang.GetText("menu.create.context_label"),
				lang.GetText("menu.create.context_desc"),
				lang.GetText("menu.create.context_cmd"), "def-context"),
			menuExec(lang.GetText("menu.create.person_label"),
				lang.GetText("menu.create.person_desc"),
				lang.GetText("menu.create.person_cmd"), "def-person"),
		)
	}

	items = append(items,
		menuExec(lang.GetText("menu.create.brain_label"),
			lang.GetText("menu.create.brain_desc"),
			lang.GetText("menu.create.brain_cmd"), "new-brain"),
		menuExec(lang.GetText("menu.create.workspace_label"),
			lang.GetText("menu.create.workspace_desc"),
			lang.GetText("menu.create.workspace_cmd"), "new-workspace"),
		menuBack(),
	)

	return ui.MenuDef{Title: "Create New", Breadcrumb: []string{"Main Menu", "Create"}, Items: items, Parent: "main"}
}

func buildBrowseMenuDef() ui.MenuDef {
	var items []ui.MenuItem

	items = append(items,
		menuExec(lang.GetText("menu.browse.recent_label"),
			lang.GetText("menu.browse.recent_desc"),
			lang.GetText("menu.browse.recent_cmd"), "recent"),
		menuExec(lang.GetText("menu.browse.search_label"),
			lang.GetText("menu.browse.search_desc"),
			lang.GetText("menu.browse.search_cmd"), "search"),
	)

	if IsEnabled(FeatureTasks) {
		items = append(items,
			menuExec(lang.GetText("menu.browse.task_browser_label"),
				lang.GetText("menu.browse.task_browser_desc"),
				lang.GetText("menu.browse.task_browser_cmd"), "task-browse"),
			menuExec("🐸 Eat the Frog", "Important tasks to tackle first", "flip task frog", "task-frog"),
		)
	}

	if IsEnabled(FeatureExercises) {
		items = append(items,
			menuNav("🏋️  Exercises", "Track session or browse exercise history", "flip exercise", "exercises"),
		)
	}

	items = append(items, menuBack())
	return ui.MenuDef{Title: "Browse & Search", Breadcrumb: []string{"Main Menu", "Browse"}, Items: items, Parent: "main"}
}

func buildManageMenuDef() ui.MenuDef {
	var items []ui.MenuItem

	items = append(items,
		menuExec(lang.GetText("menu.manage.workspaces_label"),
			lang.GetText("menu.manage.workspaces_desc"),
			lang.GetText("menu.manage.workspaces_cmd"), "manage-workspaces"),
		menuExec(lang.GetText("menu.manage.brains_label"),
			lang.GetText("menu.manage.brains_desc"),
			lang.GetText("menu.manage.brains_cmd"), "manage-brains"),
		menuExec(lang.GetText("menu.manage.config_label"),
			lang.GetText("menu.manage.config_desc"), "", "view-config"),
	)

	if IsEnabled(FeatureTemplates) {
		items = append(items,
			menuExec(lang.GetText("menu.manage.templates_label"),
				lang.GetText("menu.manage.templates_desc"),
				lang.GetText("menu.manage.templates_cmd"), "manage-templates"),
		)
	}

	items = append(items,
		menuExec(lang.GetText("menu.manage.git_status_label"),
			lang.GetText("menu.manage.git_status_desc"), "", "git-status"),
		menuExec(lang.GetText("menu.manage.git_pull_label"),
			lang.GetText("menu.manage.git_pull_desc"), "", "git-pull"),
		menuExec(lang.GetText("menu.manage.git_commit_label"),
			lang.GetText("menu.manage.git_commit_desc"), "", "git-commit"),
	)

	if IsEnabled(FeatureMigration) {
		items = append(items,
			menuExec(lang.GetText("menu.manage.migrate_label"),
				lang.GetText("menu.manage.migrate_desc"),
				lang.GetText("menu.manage.migrate_cmd"), "migrate"),
		)
	}

	items = append(items, menuBack())
	return ui.MenuDef{Title: "Manage", Breadcrumb: []string{"Main Menu", "Manage"}, Items: items, Parent: "main"}
}

func buildStatusMenuDef() ui.MenuDef {
	var items []ui.MenuItem

	items = append(items,
		menuExec(lang.GetText("menu.status.overview_label"),
			lang.GetText("menu.status.overview_desc"),
			lang.GetText("menu.status.overview_cmd"), "status-overview"),
		menuExec(lang.GetText("menu.status.git_status_label"),
			lang.GetText("menu.status.git_status_desc"),
			lang.GetText("menu.status.git_status_cmd"), "git-status-all"),
		menuExec(lang.GetText("menu.status.git_log_label"),
			lang.GetText("menu.status.git_log_desc"),
			lang.GetText("menu.status.git_log_cmd"), "git-log"),
		menuExec(lang.GetText("menu.status.git_commit_label"),
			lang.GetText("menu.status.git_commit_desc"),
			lang.GetText("menu.status.git_commit_cmd"), "git-commit"),
		menuExec(lang.GetText("menu.status.git_pull_label"),
			lang.GetText("menu.status.git_pull_desc"),
			lang.GetText("menu.status.git_pull_cmd"), "git-pull"),
	)

	if IsEnabled(FeatureVSCode) {
		items = append(items,
			menuNav(lang.GetText("menu.status.vscode_label"),
				lang.GetText("menu.status.vscode_desc"),
				lang.GetText("menu.status.vscode_cmd"), "vscode"),
		)
	}

	items = append(items, menuBack())
	return ui.MenuDef{Title: "Status & Git", Breadcrumb: []string{"Main Menu", "Status"}, Items: items, Parent: "main"}
}

func buildVSCodeMenuDef() ui.MenuDef {
	items := []ui.MenuItem{
		menuExec("📦 Install Extension", "Install/update the Flip VS Code extension", "flip vscode install-extension", "vscode-install-ext"),
		menuExec("📊 Extension Status", "Show extension installation status", "flip vscode extension-status", "vscode-ext-status"),
		menuExec(lang.GetText("menu.vscode.install_label"),
			lang.GetText("menu.vscode.install_desc"),
			lang.GetText("menu.vscode.install_cmd"), "vscode-install-tasks"),
		menuExec(lang.GetText("menu.vscode.install_local_label"),
			lang.GetText("menu.vscode.install_local_desc"),
			lang.GetText("menu.vscode.install_local_cmd"), "vscode-install-tasks-local"),
		menuExec(lang.GetText("menu.vscode.status_label"),
			lang.GetText("menu.vscode.status_desc"),
			lang.GetText("menu.vscode.status_cmd"), "vscode-status"),
		menuExec("🗑️  Uninstall Extension", "Remove the Flip VS Code extension", "flip vscode uninstall-extension", "vscode-uninstall-ext"),
		menuExec(lang.GetText("menu.vscode.uninstall_label"),
			lang.GetText("menu.vscode.uninstall_desc"),
			lang.GetText("menu.vscode.uninstall_cmd"), "vscode-uninstall-tasks"),
		menuBack(),
	}
	return ui.MenuDef{Title: "VS Code Integration", Breadcrumb: []string{"Main Menu", "Status", "VS Code"}, Items: items, Parent: "status"}
}

func buildHelpMenuDef() ui.MenuDef {
	items := []ui.MenuItem{
		menuExec(lang.GetText("menu.help.quickstart_label"),
			lang.GetText("menu.help.quickstart_desc"),
			lang.GetText("menu.help.quickstart_cmd"), "help-quickstart"),
		menuExec(lang.GetText("menu.help.commands_label"),
			lang.GetText("menu.help.commands_desc"),
			lang.GetText("menu.help.commands_cmd"), "help-commands"),
		menuExec(lang.GetText("menu.help.intro_label"),
			lang.GetText("menu.help.intro_desc"),
			lang.GetText("menu.help.intro_cmd"), "help-intro"),
		menuExec(lang.GetText("menu.help.config_label"),
			lang.GetText("menu.help.config_desc"),
			lang.GetText("menu.help.config_cmd"), "help-config"),
		menuBack(),
	}
	return ui.MenuDef{Title: "Help & Documentation", Breadcrumb: []string{"Main Menu", "Help"}, Items: items, Parent: "main"}
}

func buildExercisesMenuDef() ui.MenuDef {
	items := []ui.MenuItem{
		menuExec("🏃 Track Session", "Log a new exercise session", "flip exercise track", "exercise-track"),
		menuExec("📋 List Exercises", "Show all exercises", "flip exercise list", "exercise-list"),
		menuExec("📊 Show Exercise", "View exercise details and history", "flip exercise show", "exercise-show"),
		menuExec("✏️  Edit Exercise", "Add or modify exercise variants", "flip exercise edit", "exercise-edit"),
		menuExec("📋 List Plans", "Show all exercise plans", "flip exercise plan list", "exercise-plan-list"),
		menuExec("📋 Show Plan", "View plan details", "flip exercise plan show", "exercise-plan-show"),
		menuBack(),
	}
	return ui.MenuDef{Title: "Exercises", Breadcrumb: []string{"Main Menu", "Browse", "Exercises"}, Items: items, Parent: "browse"}
}

// ── Status Builder ──────────────────────────────────────────────────

func buildStatusStringV2() string {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return "⚠️  No workspace configured"
	}

	if config.ActiveWorkspace == "" {
		return "⚠️  No active workspace"
	}

	var activeWs *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWs = &config.Workspaces[i]
			break
		}
	}

	if activeWs == nil {
		return "⚠️  Workspace not found"
	}

	parts := []string{fmt.Sprintf("📂 %s", activeWs.Name)}

	var defaultBrain *Brain
	if activeWs.DefaultBrain != "" {
		parts = append(parts, fmt.Sprintf("🧠 %s", activeWs.DefaultBrain))
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == activeWs.DefaultBrain {
				defaultBrain = &activeWs.Brains[i]
				break
			}
		}
	}

	if defaultBrain != nil {
		status, err := git.GetStatus(defaultBrain.Path)
		if err == nil && status.IsRepo {
			parts = append(parts, fmt.Sprintf("⎇ %s", status.Branch))
			if status.HasChanges {
				parts = append(parts, "●")
			} else {
				parts = append(parts, "✓")
			}
		}
	}

	return strings.Join(parts, " · ")
}

// ── Entry Point & Dispatcher ────────────────────────────────────────

func runInteractiveMenuV2() error {
	startMenu := "" // empty = "main"

	for {
		// Rebuild status & config each iteration (may change after git ops etc.)
		status := buildStatusStringV2()
		config := buildMenuConfig()

		execCmd, lastMenu, err := ui.RunApp(status, config, startMenu)
		if err != nil {
			return err
		}

		// User quit (Exit or Ctrl-C)
		if execCmd == "" {
			fmt.Println("👋 See you later!")
			return nil
		}

		// Execute the selected command
		if err := executeCommandV2(execCmd); err != nil {
			fmt.Printf("\n⚠️  %v\n", err)
		}

		// Wait for user to read command output before returning to menu
		fmt.Print("\n↩  Press Enter to return to menu...")
		fmt.Scanln()

		// Return to the menu the user was in when they selected the command
		startMenu = lastMenu
	}
}

// executeCommandV2 dispatches the command selected from the menu
func executeCommandV2(cmd string) error {
	switch cmd {
	// ── Create ──────────────────────────────────────────────
	case "note":
		return runCreateNote()
	case "quicknote":
		return runCreateQuicknote()
	case "meeting":
		return runCreateMeeting()
	case "journal":
		return runCreateJournal()
	case "task":
		return runCreateTask()
	case "exercise-new":
		runExerciseNew(nil, []string{})
		return nil
	case "exercise-plan-new":
		ExercisePlanNewCmd.Run(nil, []string{})
		return nil
	case "def-org":
		return runDefinitionsAddOrg()
	case "def-project":
		return runDefinitionsAddProject()
	case "def-context":
		return runDefinitionsAddContext()
	case "def-person":
		return runDefinitionsAddPerson()
	case "new-brain":
		return runNewBrain()
	case "new-workspace":
		return runNewWorkspace()

	// ── Browse ──────────────────────────────────────────────
	case "recent":
		return showRecentNotes(20)
	case "search":
		return searchNotesInteractive()
	case "task-browse":
		return runTaskBrowser("")
	case "task-frog":
		return runTaskBrowser("frog")
	case "exercise-track":
		ExerciseTrackCmd.Run(nil, []string{})
		return nil
	case "exercise-list":
		runExerciseList(&cobra.Command{}, []string{})
		return nil
	case "exercise-show":
		ExerciseShowCmd.Run(nil, []string{})
		return nil
	case "exercise-edit":
		ExerciseEditCmd.Run(nil, []string{})
		return nil
	case "exercise-plan-list":
		ExercisePlanListCmd.Run(nil, []string{})
		return nil
	case "exercise-plan-show":
		ExercisePlanShowCmd.Run(nil, []string{})
		return nil

	// ── Manage ──────────────────────────────────────────────
	case "manage-workspaces":
		return runEditWorkspaceMenu()
	case "manage-brains":
		return runManageBrainsMenu()
	case "view-config":
		return runViewWorkspaceConfig()
	case "manage-templates":
		return runEditManageMenu()
	case "git-status":
		return runBrainGitStatus(false)
	case "git-status-all":
		return runBrainGitStatus(true)
	case "git-log":
		return runBrainGitLog(10, true)
	case "git-commit":
		checkUncommittedChangesOnExit()
		return nil
	case "git-pull":
		checkRemoteUpdatesOnStart()
		return nil
	case "migrate":
		return runBrainMigrationMenu()

	// ── Status ──────────────────────────────────────────────
	case "status-overview":
		return runStatus()
	case "about":
		return runAboutMenuV2()

	// ── VS Code ─────────────────────────────────────────────
	case "vscode-install-ext":
		return installExtension()
	case "vscode-ext-status":
		showExtensionStatus()
		return nil
	case "vscode-install-tasks":
		return installTasksGlobal()
	case "vscode-install-tasks-local":
		return installTasksLocal()
	case "vscode-status":
		showVSCodeStatus()
		return nil
	case "vscode-uninstall-ext":
		return uninstallExtension()
	case "vscode-uninstall-tasks":
		return uninstallTasksGlobal()

	// ── Help ────────────────────────────────────────────────
	case "help-quickstart":
		return runQuickstart()
	case "help-commands":
		fmt.Println()
		fmt.Println(lang.GetTemplate("commands"))
		fmt.Println()
		fmt.Println("For detailed help: flip <command> --help")
		return nil
	case "help-intro":
		return runIntro()
	case "help-config":
		return showConfigLocation()

	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// runAboutMenuV2 shows about info (non-interactive, just prints)
func runAboutMenuV2() error {
	fmt.Println()
	banner := lang.GetBanner()
	fmt.Println(banner)
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  flip %s — Personal Knowledge Management\n", FlipVersion)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("  📝 Standard Markdown — no proprietary formats")
	fmt.Println("  🧠 Multi-brain — Obsidian, Logseq, Dendron, Foam, …")
	fmt.Println("  📂 Workspaces — group related brains together")
	fmt.Println("  🔄 Git integration — sync, commit, pull built in")
	fmt.Println("  🤖 AI features — summarize, research, improve notes")
	fmt.Println("  💻 VS Code — extension & task runner integration")
	fmt.Println()

	// Show active features
	features := GetFeatureInfo()
	var enabled, disabled []string
	for _, f := range features {
		if f.IsCore {
			continue
		}
		if f.Enabled {
			enabled = append(enabled, string(f.Name))
		} else {
			disabled = append(disabled, string(f.Name))
		}
	}
	if len(enabled) > 0 {
		fmt.Printf("  ✅ Active modules:   %s\n", strings.Join(enabled, ", "))
	}
	if len(disabled) > 0 {
		fmt.Printf("  ⬚  Disabled modules: %s\n", strings.Join(disabled, ", "))
	}
	fmt.Println()

	// Show workspace stats
	config, err := loadWorkspaceConfig()
	if err == nil && len(config.Workspaces) > 0 {
		totalBrains := 0
		for _, ws := range config.Workspaces {
			totalBrains += len(ws.Brains)
		}
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("  📂 %d workspace(s), 🧠 %d brain(s)\n", len(config.Workspaces), totalBrains)
		if config.ActiveWorkspace != "" {
			fmt.Printf("  ⭐ Active: %s\n", config.ActiveWorkspace)
		}
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
	}

	fmt.Println("  📖 flip help · flip intro · flip quickstart")
	fmt.Println()
	return nil
}

// showConfigLocation prints the config file path
func showConfigLocation() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	fmt.Printf("\n📁 Configuration Location:\n")
	fmt.Printf("   %s\n", configPath)
	return nil
}

// searchNotesInteractive runs an interactive note search from the terminal
func searchNotesInteractive() error {
	fmt.Print("\n🔍 Enter search query: ")
	var query string
	fmt.Scanln(&query)
	if query == "" {
		fmt.Println("No query entered")
		return nil
	}
	return searchNotes(query, false)
}
