package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// ExerciseEditCmd edits an existing exercise
var ExerciseEditCmd = &cobra.Command{
	Use:   "edit [exercise-id]",
	Short: lang.GetText("exercise.edit.short"),
	Long:  lang.GetText("exercise.edit.long"),
	Run:   runExerciseEdit,
}

func runExerciseEdit(cmd *cobra.Command, args []string) {
	// Allow user to select the brain before editing
	activeBrain, err := selectBrainForOperation(lang.GetText("prompts.select_brain_for_edit_exercise"))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	brainPath := activeBrain.Path
	detection, err := brain.NewDetector().DetectBrainType(brainPath)
	if err != nil {
		fmt.Printf("Error detecting brain: %v\n", err)
		os.Exit(1)
	}
	if !detection.Compatible {
		fmt.Println("Error: Not in a compatible brain directory")
		os.Exit(1)
	}

	scanner := exercises.NewScanner()
	parser := exercises.NewParser()

	// Get exercise ID
	var exerciseID string
	if len(args) > 0 {
		exerciseID = args[0]
	} else {
		// List exercises and let user choose
		allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
		if err != nil {
			fmt.Printf("Error scanning exercises: %v\n", err)
			os.Exit(1)
		}

		if len(allExercises) == 0 {
			fmt.Println("No exercises found. Create one with 'flip exercise new'")
			os.Exit(1)
		}

		exerciseNames := make([]string, len(allExercises))
		exerciseMap := make(map[string]*exercises.Exercise)
		for i, ex := range allExercises {
			contextStr := "no context"
			if ex.Context != "" {
				contextStr = ex.Context
			}
			exerciseNames[i] = fmt.Sprintf("%s (%s)", ex.Name, contextStr)
			exerciseMap[exerciseNames[i]] = ex
		}

		selectPrompt := promptui.Select{
			Label: "Select Exercise to Edit",
			Items: exerciseNames,
		}
		_, selected, err := selectPrompt.Run()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		exerciseID = exerciseMap[selected].ID
	}

	// Load exercise
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	var exercise *exercises.Exercise
	for _, ex := range allExercises {
		if ex.ID == exerciseID {
			exercise = ex
			break
		}
	}

	if exercise == nil {
		fmt.Printf("Exercise not found: %s\n", exerciseID)
		os.Exit(1)
	}

	fmt.Printf("\nEditing: %s\n\n", exercise.Name)

	// What to edit
	editOptions := []string{
		"Name",
		"Context",
		"Description",
		"Goal",
		"Variants",
		"Status",
		"Cancel",
	}

	editPrompt := promptui.Select{
		Label: "What would you like to edit?",
		Items: editOptions,
	}

	editIdx, _, err := editPrompt.Run()
	if err != nil || editIdx == len(editOptions)-1 {
		fmt.Println("Edit cancelled")
		return
	}

	switch editIdx {
	case 0: // Name
		namePrompt := promptui.Prompt{
			Label:   "Name",
			Default: exercise.Name,
		}
		name, err := namePrompt.Run()
		if err == nil && name != "" {
			exercise.Name = name
		}

	case 1: // Context
		contextPrompt := promptui.Prompt{
			Label:   "Context",
			Default: exercise.Context,
		}
		context, err := contextPrompt.Run()
		if err == nil {
			exercise.Context = context
		}

	case 2: // Description
		descPrompt := promptui.Prompt{
			Label:   "Description",
			Default: exercise.Description,
		}
		desc, err := descPrompt.Run()
		if err == nil {
			exercise.Description = desc
		}

	case 3: // Goal
		goalPrompt := promptui.Prompt{
			Label:   "Goal",
			Default: exercise.Goal,
		}
		goal, err := goalPrompt.Run()
		if err == nil {
			exercise.Goal = goal
		}

	case 4: // Variants
		for {
			fmt.Printf("\nCurrent variants: %d\n", len(exercise.Variants))
			for i, v := range exercise.Variants {
				if v.Name != "" {
					fmt.Printf("  %d. %s\n", i+1, v.Name)
				} else {
					fmt.Printf("  %d. (unnamed)\n", i+1)
				}
			}

			actionPrompt := promptui.Select{
				Label: lang.GetText("prompts.variants_title"),
				Items: []string{lang.GetText("prompts.variant_add"), lang.GetText("prompts.open_in_editor"), lang.GetText("prompts.done")},
			}
			idx, _, err := actionPrompt.Run()
			if err != nil || idx == 2 {
				break
			}

			switch idx {
			case 0: // Add variant
				// Configure new variant similar to creation flow
				variant := exercises.ExerciseVariant{TrackingProperties: make(map[string]string)}

				varNamePrompt := promptui.Prompt{Label: "Variant Name (optional)"}
				variant.Name, _ = varNamePrompt.Run()

				varDescPrompt := promptui.Prompt{Label: "Description (optional)"}
				variant.Description, _ = varDescPrompt.Run()

				propMgr, err := exercises.NewPropertyManager(brainPath)
				if err != nil {
					fmt.Printf("Warning: Could not load property manager: %v\n", err)
				}

				for {
					propOptions := propMgr.GetPropertyNames()
					propOptions = append(propOptions, lang.GetText("prompts.property_add_new"), lang.GetText("prompts.property_done"))

					selectPrompt := promptui.Select{
						Label: lang.GetText("prompts.property_select"),
						Items: propOptions,
						Size:  15,
					}
					_, selected, err := selectPrompt.Run()
					if err != nil || selected == lang.GetText("prompts.property_done") {
						break
					}

					if selected == lang.GetText("prompts.property_add_new") {
						namePrompt := promptui.Prompt{Label: lang.GetText("prompts.property_name")}
						propName, err := namePrompt.Run()
						if err != nil || strings.TrimSpace(propName) == "" {
							continue
						}
						propName = strings.TrimSpace(propName)

						unitPrompt := promptui.Prompt{Label: lang.GetText("prompts.property_unit"), Default: "text"}
						propUnit, err := unitPrompt.Run()
						if err != nil {
							continue
						}
						propUnit = strings.TrimSpace(propUnit)

						if err := propMgr.Add(propName, propUnit); err != nil {
							fmt.Printf("Warning: Could not save property: %v\n", err)
						}
						variant.TrackingProperties[propName] = propUnit
						fmt.Printf(lang.GetText("prompts.added")+"\n", propName, propUnit)
					} else {
						// Parse "name (unit)"
						propName := strings.Split(selected, " (")[0]
						prop := propMgr.GetByName(propName)
						if prop != nil {
							variant.TrackingProperties[prop.Name] = prop.Unit
							fmt.Printf(lang.GetText("prompts.added")+"\n", prop.Name, prop.Unit)
						}
					}
				}

				exercise.Variants = append(exercise.Variants, variant)
				fmt.Println(lang.GetText("prompts.variant_added"))

			case 1: // Open in editor
				if err := promptAndOpenEditor(exercise.FilePath); err != nil {
					fmt.Printf("⚠️  Could not open editor: %v\n", err)
				}
			}
		}

	case 5: // Status
		statusOptions := []string{"active", "inactive", "paused"}
		statusPrompt := promptui.Select{
			Label: "Status",
			Items: statusOptions,
		}
		_, status, err := statusPrompt.Run()
		if err == nil {
			exercise.Status = status
		}
	}

	// Save changes
	filePath := parser.GetExerciseFilePath(brainPath, exerciseID, string(detection.Type))
	if err := parser.WriteExercise(exercise, filePath); err != nil {
		fmt.Printf("Error saving exercise: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Exercise updated: %s\n", filePath)
}
