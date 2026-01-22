package commands

// menu_create.go - Create/Add Menu Functions
//
// Contains:
//   - runCreateNewMenu()  Main create submenu for notes, meetings, tasks, etc.

import (
	"fmt"

	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
)

// runCreateNewMenu shows submenu for creating new content
func runCreateNewMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	showBreadcrumb("Main", "Create")

	// Build menu items dynamically based on enabled features
	var menuItems []MenuItem

	// Meetings feature: note, meeting, journal
	if IsEnabled(FeatureMeetings) {
		menuItems = append(menuItems,
			MenuItem{
				Label:       lang.GetText("menu.create.note_label"),
				Description: lang.GetText("menu.create.note_desc"),
				Command:     lang.GetText("menu.create.note_cmd"),
				Action: func() error {
					if err := runCreateNote(); err != nil {
						fmt.Printf("\n❌ Error: %v\n", err)
					}
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runCreateNewMenu()
				},
			},
			MenuItem{
				Label:       lang.GetText("menu.create.meeting_label"),
				Description: lang.GetText("menu.create.meeting_desc"),
				Command:     lang.GetText("menu.create.meeting_cmd"),
				Action: func() error {
					if err := runCreateMeeting(); err != nil {
						fmt.Printf("\n❌ Error: %v\n", err)
					}
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runCreateNewMenu()
				},
			},
			MenuItem{
				Label:       lang.GetText("menu.create.journal_label"),
				Description: lang.GetText("menu.create.journal_desc"),
				Command:     lang.GetText("menu.create.journal_cmd"),
				Action: func() error {
					if err := runCreateJournal(); err != nil {
						fmt.Printf("\n❌ Error: %v\n", err)
					}
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runCreateNewMenu()
				},
			},
		)
	}

	// Tasks feature
	if IsEnabled(FeatureTasks) {
		menuItems = append(menuItems, MenuItem{
			Label:       lang.GetText("menu.create.task_label"),
			Description: lang.GetText("menu.create.task_desc"),
			Command:     lang.GetText("menu.create.task_cmd"),
			Action: func() error {
				if err := runCreateTask(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runCreateNewMenu()
			},
		})
	}

	// Exercises feature
	if IsEnabled(FeatureExercises) {
		menuItems = append(menuItems,
			MenuItem{
				Label:       "🏋️  New Exercise",
				Description: "Create a new repeatable exercise for practice and tracking",
				Command:     "flip exercise new",
				Action: func() error {
					runExerciseNew(nil, []string{})
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runCreateNewMenu()
				},
			},
			MenuItem{
				Label:       "📋 New Exercise Plan",
				Description: "Create a structured training/learning plan with multiple exercises",
				Command:     "flip exercise plan new",
				Action: func() error {
					ExercisePlanNewCmd.Run(nil, []string{})
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runCreateNewMenu()
				},
			},
		)
	}

	// Definitions feature
	if IsEnabled(FeatureDefinitions) {
		menuItems = append(menuItems,
			MenuItem{
				Label:       lang.GetText("menu.create.organization_label"),
				Description: lang.GetText("menu.create.organization_desc"),
				Command:     lang.GetText("menu.create.organization_cmd"),
				Action: func() error {
					if err := runDefinitionsAddOrg(); err != nil {
						fmt.Printf("\n❌ Error: %v\n", err)
					}
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runCreateNewMenu()
				},
			},
			MenuItem{
				Label:       lang.GetText("menu.create.project_label"),
				Description: lang.GetText("menu.create.project_desc"),
				Command:     lang.GetText("menu.create.project_cmd"),
				Action: func() error {
					if err := runDefinitionsAddProject(); err != nil {
						fmt.Printf("\n❌ Error: %v\n", err)
					}
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runCreateNewMenu()
				},
			},
			MenuItem{
				Label:       lang.GetText("menu.create.context_label"),
				Description: lang.GetText("menu.create.context_desc"),
				Command:     lang.GetText("menu.create.context_cmd"),
				Action: func() error {
					if err := runDefinitionsAddContext(); err != nil {
						fmt.Printf("\n❌ Error: %v\n", err)
					}
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runCreateNewMenu()
				},
			},
			MenuItem{
				Label:       lang.GetText("menu.create.person_label"),
				Description: lang.GetText("menu.create.person_desc"),
				Command:     lang.GetText("menu.create.person_cmd"),
				Action: func() error {
					if err := runDefinitionsAddPerson(); err != nil {
						fmt.Printf("\n❌ Error: %v\n", err)
					}
					fmt.Println(lang.GetText("prompts.continue"))
					fmt.Scanln()
					return runCreateNewMenu()
				},
			},
			MenuItem{
				Label:       lang.GetText("menu.create.definitions_label"),
				Description: lang.GetText("menu.create.definitions_desc"),
				Command:     lang.GetText("menu.create.definitions_cmd"),
				Action: func() error {
					if err := runDefinitionsMenu(); err != nil {
						return err
					}
					return runCreateNewMenu()
				},
			},
		)
	}

	// Core features: brain and workspace (always available)
	menuItems = append(menuItems,
		MenuItem{
			Label:       lang.GetText("menu.create.brain_label"),
			Description: lang.GetText("menu.create.brain_desc"),
			Command:     lang.GetText("menu.create.brain_cmd"),
			Action: func() error {
				if err := runNewBrain(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runCreateNewMenu()
			},
		},
		MenuItem{
			Label:       lang.GetText("menu.create.workspace_label"),
			Description: lang.GetText("menu.create.workspace_desc"),
			Command:     lang.GetText("menu.create.workspace_cmd"),
			Action: func() error {
				if err := runNewWorkspace(); err != nil {
					fmt.Printf("\n❌ Error: %v\n", err)
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runCreateNewMenu()
			},
		},
		MenuItem{
			Label:       lang.GetText("menu.create.back_label"),
			Description: lang.GetText("menu.create.back_desc"),
			Command:     "",
			Action:      runInteractiveMenu,
		},
	)

	templates := createMenuItemWithCommandTemplates()

	selectMenu := promptui.Select{
		Label:     lang.GetText("menu.titles.create"),
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
