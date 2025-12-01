package commands

import (
	"fmt"
	"strings"

	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
)

// Exercises submenu
func runExercisesMenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	showBreadcrumb("Main", "Exercises")

	menuItems := []MenuItem{
		{
			Label:       lang.GetText("menu.exercises.track_label"),
			Description: lang.GetText("menu.exercises.track_desc"),
			Command:     lang.GetText("menu.exercises.track_cmd"),
			Action: func() error {
				ExerciseTrackCmd.Run(ExerciseTrackCmd, []string{})
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesMenu()
			},
		},
		{
			Label:       lang.GetText("menu.exercises.list_label"),
			Description: lang.GetText("menu.exercises.list_desc"),
			Command:     lang.GetText("menu.exercises.list_cmd"),
			Action: func() error {
				ExerciseListCmd.Run(ExerciseListCmd, []string{})
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesMenu()
			},
		},
		{
			Label:       lang.GetText("menu.exercises.new_label"),
			Description: lang.GetText("menu.exercises.new_desc"),
			Command:     lang.GetText("menu.exercises.new_cmd"),
			Action: func() error {
				ExerciseNewCmd.Run(ExerciseNewCmd, []string{})
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesMenu()
			},
		},
		{
			Label:       lang.GetText("menu.exercises.show_label"),
			Description: lang.GetText("menu.exercises.show_desc"),
			Command:     lang.GetText("menu.exercises.show_cmd"),
			Action: func() error {
				prompt := promptui.Prompt{Label: "Exercise ID"}
				id, err := prompt.Run()
				if err == nil && strings.TrimSpace(id) != "" {
					ExerciseShowCmd.Run(ExerciseShowCmd, []string{id})
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesMenu()
			},
		},
		{
			Label:       lang.GetText("menu.browse.back_label"),
			Description: lang.GetText("menu.browse.back_desc"),
			Command:     "",
			Action:      runInteractiveMenu,
		},
	}

	templates := createMenuItemWithCommandTemplates()

	selectMenu := promptui.Select{
		Label:     "Exercises",
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
