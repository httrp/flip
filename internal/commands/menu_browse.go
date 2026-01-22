package commands

// menu_browse.go - Browse & Search Menu Functions
//
// Contains:
//   - runBrowseSearchMenu()  Main browse submenu
//   - runTaskBrowseMenu()    Task-specific browsing options

import (
	"fmt"

	"github.com/httrp/flip/internal/lang"
	"github.com/httrp/flip/internal/tasks"
	"github.com/manifoldco/promptui"
)

// runBrowseSearchMenu shows submenu for browsing and searching notes
func runBrowseSearchMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	showBreadcrumb("Main", "Browse & Search")

	// Build menu items dynamically based on enabled features
	var menuItems []MenuItem

	// Core browse functions (always available)
	menuItems = append(menuItems,
		MenuItem{
			Label:       lang.GetText("menu.browse.recent_label"),
			Description: lang.GetText("menu.browse.recent_desc"),
			Command:     lang.GetText("menu.browse.recent_cmd"),
			Action: func() error {
				if err := showRecentNotes(20); err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
				}
				return runBrowseSearchMenu()
			},
		},
		MenuItem{
			Label:       lang.GetText("menu.browse.search_label"),
			Description: lang.GetText("menu.browse.search_desc"),
			Command:     lang.GetText("menu.browse.search_cmd"),
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
	)

	// Tasks feature
	if IsEnabled(FeatureTasks) {
		menuItems = append(menuItems, MenuItem{
			Label:       lang.GetText("menu.browse.task_browser_label"),
			Description: lang.GetText("menu.browse.task_browser_desc"),
			Command:     lang.GetText("menu.browse.task_browser_cmd"),
			Action: func() error {
				// Call task browser directly with default filter
				if err := runTaskBrowser(""); err != nil {
					fmt.Printf("\nError: %v\n", err)
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
				}
				return runBrowseSearchMenu()
			},
		})
	}

	// Exercises feature
	if IsEnabled(FeatureExercises) {
		menuItems = append(menuItems, MenuItem{
			Label:       "🏋️  Exercises",
			Description: "Track session or browse exercise history",
			Command:     "flip exercise",
			Action: func() error {
				if err := runExercisesSubmenu(); err != nil {
					return err
				}
				return runBrowseSearchMenu()
			},
		})
	}

	// Back button (always available)
	menuItems = append(menuItems, MenuItem{
		Label:       lang.GetText("menu.browse.back_label"),
		Description: lang.GetText("menu.browse.back_desc"),
		Command:     "",
		Action:      runInteractiveMenu,
	})

	templates := createMenuItemWithCommandTemplates()

	selectMenu := promptui.Select{
		Label:     lang.GetText("menu.titles.browse"),
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

// runTaskBrowseMenu shows task-specific browsing options
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
