package commands

import (
	"fmt"
	"os"

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
	// Detect brain
	brainPath, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(brainPath)
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
		fmt.Printf("\nCurrent variants: %d\n", len(exercise.Variants))
		for i, v := range exercise.Variants {
			if v.Name != "" {
				fmt.Printf("  %d. %s\n", i+1, v.Name)
			} else {
				fmt.Printf("  %d. (unnamed)\n", i+1)
			}
		}
		fmt.Println("\nNote: To modify variants and tracking properties, please edit the file directly")
		fmt.Println(lang.GetText("prompts.continue"))
		fmt.Scanln()
		return

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
