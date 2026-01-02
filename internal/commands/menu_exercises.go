package commands

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
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

// runExercisesSubmenu shows exercise-specific actions (called from Browse & Search menu)
func runExercisesSubmenu() error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	showBreadcrumb("Main", "Browse & Search", "Exercises")

	menuItems := []MenuItem{
		{
			Label:       "🏃 Track Session",
			Description: "Log a new exercise session",
			Command:     "flip exercise track",
			Action: func() error {
				ExerciseTrackCmd.Run(nil, []string{})
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesSubmenu()
			},
		},
		{
			Label:       "📋 List Exercises",
			Description: "Show all exercises",
			Command:     "flip exercise list",
			Action: func() error {
				// Build selectable list of known contexts from existing exercises
				ctx := ""
				var tags []string

				// Detect active brain and scan exercises to collect contexts
				activeBrain, err := getActiveBrain()
				if err == nil {
					detector := brain.NewDetector()
					if result, derr := detector.DetectBrainType(activeBrain.Path); derr == nil && result.Compatible {
						scanner := exercises.NewScanner()
						if exs, serr := scanner.ScanExercises(activeBrain.Path, string(result.Type)); serr == nil {
							ctxSet := map[string]struct{}{}
							for _, ex := range exs {
								if ex.Context != "" {
									ctxSet[ex.Context] = struct{}{}
								}
							}
							var ctxList []string
							for c := range ctxSet {
								ctxList = append(ctxList, c)
							}
							sort.Strings(ctxList)
							if len(ctxList) > 0 {
								options := append([]string{"<no context filter>"}, ctxList...)
								options = append(options, "<custom>")
								prompt := promptui.Select{
									Label:     "Filter by context",
									Items:     options,
									Templates: createSimpleSelectTemplates(),
									Size:      calculateMenuSize(len(options)),
									HideHelp:  true,
								}
								idx, _, perr := prompt.Run()
								if perr == nil {
									if idx == 0 {
										ctx = ""
									} else if options[idx] == "<custom>" {
										fmt.Print("Enter custom context: ")
										var input string
										fmt.Scanln(&input)
										ctx = strings.TrimSpace(input)
									} else {
										ctx = options[idx]
									}
								}
							}
						}
					}
				}

				// Optional tags filter (manual input)
				fmt.Print("Filter by tags (comma-separated, optional): ")
				reader := bufio.NewReader(os.Stdin)
				tagLine, _ := reader.ReadString('\n')
				tagLine = strings.TrimSpace(tagLine)
				if tagLine != "" {
					parts := strings.Split(tagLine, ",")
					for _, p := range parts {
						trim := strings.TrimSpace(p)
						if trim != "" {
							tags = append(tags, trim)
						}
					}
				}

				// Build a temporary cobra.Command carrying the flags for filtering
				tmp := &cobra.Command{}
				tmp.Flags().String("context", ctx, "")
				tmp.Flags().StringSlice("tags", tags, "")

				runExerciseList(tmp, []string{})
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesSubmenu()
			},
		},
		{
			Label:       "📊 Show Exercise",
			Description: "View exercise details and history",
			Command:     "flip exercise show [id]",
			Action: func() error {
				fmt.Print("\n📝 Exercise ID: ")
				var id string
				fmt.Scanln(&id)
				if id != "" {
					ExerciseShowCmd.Run(nil, []string{id})
				}
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesSubmenu()
			},
		},
		{
			Label:       lang.GetText("menu.exercises.edit_label"),
			Description: lang.GetText("menu.exercises.edit_desc"),
			Command:     lang.GetText("menu.exercises.edit_cmd"),
			Action: func() error {
				ExerciseEditCmd.Run(nil, []string{})
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesSubmenu()
			},
		},
		{
			Label:       "🗂️  Plans",
			Description: "Manage exercise plans",
			Command:     "flip exercise plan",
			Action: func() error {
				// For now, just list plans
				ExercisePlanListCmd.Run(nil, []string{})
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesSubmenu()
			},
		},
		{
			Label:       lang.GetText("menu.exercises.plan_add_label"),
			Description: lang.GetText("menu.exercises.plan_add_desc"),
			Command:     lang.GetText("menu.exercises.plan_add_cmd"),
			Action: func() error {
				ExercisePlanEditCmd.Run(nil, []string{})
				fmt.Println(lang.GetText("prompts.continue"))
				fmt.Scanln()
				return runExercisesSubmenu()
			},
		},
		{
			Label:       "◀️  Back",
			Description: "Return to Browse & Search menu",
			Command:     "",
			Action:      nil, // Will return to parent menu
		},
	}

	templates := createMenuItemWithCommandTemplates()

	selectMenu := promptui.Select{
		Label:     "Exercise Actions",
		Items:     menuItems,
		Templates: templates,
		Size:      calculateMenuSize(len(menuItems)),
		HideHelp:  true,
	}

	idx, _, err := selectMenu.Run()
	if err != nil {
		return nil // Return to parent menu
	}

	fmt.Println()
	if menuItems[idx].Action == nil {
		return nil // Back button
	}
	return menuItems[idx].Action()
}
