package commands

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// ExerciseTrackCmd tracks a session for an exercise
var ExerciseTrackCmd = &cobra.Command{
	Use:   "track [exercise-id]",
	Short: lang.GetText("exercise.track.short"),
	Long:  lang.GetText("exercise.track.long"),
	Run:   runExerciseTrack,
}

func runExerciseTrack(cmd *cobra.Command, args []string) {
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
			exerciseNames[i] = fmt.Sprintf("%s (%s)", ex.Name, string(ex.Type))
			exerciseMap[exerciseNames[i]] = ex
		}

		selectPrompt := promptui.Select{
			Label: "Select Exercise",
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

	// Create session
	now := time.Now()
	session := &exercises.ExerciseSession{
		ExerciseID: exerciseID,
		Date:       now,
		Unit:       exercise.TargetUnit,
	}

	// Prompt for session details based on exercise type
	switch exercise.Type {
	case exercises.TypeRepetition, exercises.TypeTarget:
		// Ask for value
		unit := "reps"
		if exercise.TargetUnit != "" {
			unit = exercise.TargetUnit
		}
		valuePrompt := promptui.Prompt{
			Label: fmt.Sprintf("Value (%s)", unit),
		}
		valueStr, err := valuePrompt.Run()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		value, _ := strconv.ParseFloat(valueStr, 64)
		session.Value = int(value)

	case exercises.TypeVariations:
		// Ask for variant
		if len(exercise.Variants) > 0 {
			variantPrompt := promptui.Select{
				Label: "Select Variant",
				Items: exercise.Variants,
			}
			_, session.Variant, err = variantPrompt.Run()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		}
	}

	// Duration
	durationPrompt := promptui.Prompt{
		Label:   "Duration (minutes)",
		Default: "30",
	}
	durationStr, err := durationPrompt.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	duration, _ := strconv.Atoi(durationStr)
	session.Duration = duration

	// Notes (optional)
	notesPrompt := promptui.Prompt{
		Label: "Notes (optional)",
	}
	session.Notes, _ = notesPrompt.Run()

	// Rating omitted in MVP

	// Write session file
	filePath := parser.GetSessionFilePath(brainPath, exerciseID, now, string(detection.Type))
	if err := parser.WriteSession(session, filePath); err != nil {
		fmt.Printf("Error creating session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Session tracked: %s\n", filePath)
}
