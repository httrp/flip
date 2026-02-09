package commands

// menu_browse.go - Browse & Search Menu Functions
//
// Contains:
//   - runBrowseSearchMenu()  Main browse submenu

import (
	"fmt"

	"github.com/httrp/flip/internal/lang"
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

