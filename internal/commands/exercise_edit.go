package commands

import (
	"fmt"
	"strings"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	ui "github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// ExerciseEditCmd edits an existing exercise
var ExerciseEditCmd = &cobra.Command{
	Use:   "edit [exercise-id]",
	Short: lang.GetText("exercise.edit.short"),
	Long:  lang.GetText("exercise.edit.long"),
	RunE:  runExerciseEdit,
}

func runExerciseEdit(cmd *cobra.Command, args []string) error {
	// Allow user to select the brain before editing
	activeBrain, err := selectBrainForOperation(lang.GetText("prompts.select_brain_for_edit_exercise"))
	if err != nil {
		return fmt.Errorf("selecting brain: %w", err)
	}

	brainPath := activeBrain.Path
	detection, err := brain.NewDetector().DetectBrainType(brainPath)
	if err != nil {
		return fmt.Errorf("detecting brain: %w", err)
	}
	if !detection.Compatible {
		return fmt.Errorf("not in a compatible brain directory")
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
			return fmt.Errorf("scanning exercises: %w", err)
		}

		if len(allExercises) == 0 {
			return fmt.Errorf("no exercises found - create one with 'flip exercise new'")
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

		selectItems := make([]ui.SelectItem, len(exerciseNames))
		for i, name := range exerciseNames {
			selectItems[i] = ui.SelectItem{Label: name, Value: name}
		}

		_, selected, err := ui.RunSelect("Select Exercise to Edit", selectItems, 10)
		if err != nil {
			return fmt.Errorf("selecting exercise: %w", err)
		}

		exerciseID = exerciseMap[selected].ID
	}

	// Load exercise
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		return fmt.Errorf("scanning exercises: %w", err)
	}

	var exercise *exercises.Exercise
	for _, ex := range allExercises {
		if ex.ID == exerciseID {
			exercise = ex
			break
		}
	}

	if exercise == nil {
		return fmt.Errorf("exercise not found: %s", exerciseID)
	}

	fmt.Printf("\nEditing: %s\n\n", exercise.Name)

	// What to edit
	editItems := []ui.SelectItem{
		{Label: "Name", Value: "name"},
		{Label: "Context", Value: "context"},
		{Label: "Description", Value: "description"},
		{Label: "Goal", Value: "goal"},
		{Label: "Variants", Value: "variants"},
		{Label: "Status", Value: "status"},
		{Label: "Cancel", Value: "cancel"},
	}

	_, editVal, err := ui.RunSelect("What would you like to edit?", editItems, 8)
	if err != nil || editVal == "cancel" {
		fmt.Println("Edit cancelled")
		return nil
	}

	switch editVal {
	case "name":
		name, err := ui.RunInput("Name", "", exercise.Name, nil)
		if err == nil && name != "" {
			exercise.Name = name
		}

	case "context":
		context, err := ui.RunInput("Context", "", exercise.Context, nil)
		if err == nil {
			exercise.Context = context
		}

	case "description":
		desc, err := ui.RunInput("Description", "", exercise.Description, nil)
		if err == nil {
			exercise.Description = desc
		}

	case "goal":
		goal, err := ui.RunInput("Goal", "", exercise.Goal, nil)
		if err == nil {
			exercise.Goal = goal
		}

	case "variants":
		for {
			fmt.Printf("\nCurrent variants: %d\n", len(exercise.Variants))
			for i, v := range exercise.Variants {
				if v.Name != "" {
					fmt.Printf("  %d. %s\n", i+1, v.Name)
				} else {
					fmt.Printf("  %d. (unnamed)\n", i+1)
				}
			}

			variantActionItems := []ui.SelectItem{
				{Label: lang.GetText("prompts.variant_add"), Value: "add"},
				{Label: lang.GetText("prompts.open_in_editor"), Value: "editor"},
				{Label: lang.GetText("prompts.done"), Value: "done"},
			}
			_, actionVal, err := ui.RunSelect(lang.GetText("prompts.variants_title"), variantActionItems, 5)
			if err != nil || actionVal == "done" {
				break
			}

			switch actionVal {
			case "add":
				// Configure new variant similar to creation flow
				variant := exercises.ExerciseVariant{TrackingProperties: make(map[string]string)}

				variant.Name, _ = ui.RunInput("Variant Name (optional)", "", "", nil)

				variant.Description, _ = ui.RunInput("Description (optional)", "", "", nil)

				propMgr, err := exercises.NewPropertyManager(brainPath)
				if err != nil {
					fmt.Printf("Warning: Could not load property manager: %v\n", err)
				}

				for {
					propOptions := propMgr.GetPropertyNames()
					propOptions = append(propOptions, lang.GetText("prompts.property_add_new"), lang.GetText("prompts.property_done"))

					propSelectItems := make([]ui.SelectItem, len(propOptions))
					for i, opt := range propOptions {
						propSelectItems[i] = ui.SelectItem{Label: opt, Value: opt}
					}

					_, selected, err := ui.RunSelect(lang.GetText("prompts.property_select"), propSelectItems, 15)
					if err != nil || selected == lang.GetText("prompts.property_done") {
						break
					}

					if selected == lang.GetText("prompts.property_add_new") {
						propName, err := ui.RunInput(lang.GetText("prompts.property_name"), "", "", nil)
						if err != nil || strings.TrimSpace(propName) == "" {
							continue
						}
						propName = strings.TrimSpace(propName)

						propUnit, err := ui.RunInput(lang.GetText("prompts.property_unit"), "", "text", nil)
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

			case "editor":
				if err := promptAndOpenEditor(exercise.FilePath); err != nil {
					fmt.Printf("⚠️  Could not open editor: %v\n", err)
				}
			}
		}

	case "status":
		statusItems := []ui.SelectItem{
			{Label: "active", Value: "active"},
			{Label: "inactive", Value: "inactive"},
			{Label: "paused", Value: "paused"},
		}
		_, status, err := ui.RunSelect("Status", statusItems, 4)
		if err == nil {
			exercise.Status = status
		}
	}

	// Save changes
	filePath := parser.GetExerciseFilePath(brainPath, exerciseID, string(detection.Type))
	if err := parser.WriteExercise(exercise, filePath); err != nil {
		return fmt.Errorf("saving exercise: %w", err)
	}

	fmt.Printf("✓ Exercise updated: %s\n", filePath)
	return nil
}
