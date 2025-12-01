package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// ExerciseNewCmd creates a new exercise definition
var ExerciseNewCmd = &cobra.Command{
	Use:   "new",
	Short: lang.GetText("exercise.new.short"),
	Long:  lang.GetText("exercise.new.long"),
	Run:   runExerciseNew,
}

func runExerciseNew(cmd *cobra.Command, args []string) {
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
		fmt.Println("Error: Not in a brain directory")
		os.Exit(1)
	}

	exercise := &exercises.Exercise{
		Created: time.Now(),
	}

	// Name
	namePrompt := promptui.Prompt{
		Label: "Exercise Name",
	}
	exercise.Name, err = namePrompt.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Type
	typePrompt := promptui.Select{
		Label: "Exercise Type",
		Items: []string{
			"Repetition",
			"Variations",
			"Target",
			"Skill",
			"Project",
		},
	}
	_, typeStr, err := typePrompt.Run()

		switch typeStr {
		case "Repetition":
			exercise.Type = exercises.TypeRepetition
		case "Variations":
			exercise.Type = exercises.TypeVariations
		case "Target":
			exercise.Type = exercises.TypeTarget
		case "Skill":
			exercise.Type = exercises.TypeSkill
		case "Project":
			exercise.Type = exercises.TypeProject
		}
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Description
	descPrompt := promptui.Prompt{
		Label: "Description (optional)",
	}
	exercise.Description, _ = descPrompt.Run()

	// Goal
	goalPrompt := promptui.Prompt{
		Label: "Goal (optional)",
	}
	exercise.Goal, _ = goalPrompt.Run()

	// Unit (for measurable exercises)
	if exercise.Type == exercises.TypeTarget || exercise.Type == exercises.TypeRepetition {
		unitPrompt := promptui.Prompt{
			Label:   "Unit",
			Default: "reps",
		}
		exercise.TargetUnit, _ = unitPrompt.Run()
	}

	// Tags
	tagsPrompt := promptui.Prompt{
		Label: "Tags (comma-separated, optional)",
	}
	tagsInput, _ := tagsPrompt.Run()
	if tagsInput != "" {
		tags := strings.Split(tagsInput, ",")
		for _, tag := range tags {
			exercise.Tags = append(exercise.Tags, strings.TrimSpace(tag))
		}
	}

	// Generate ID
	exercise.ID = generateExerciseID(exercise.Name)

	// Write exercise file
	parser := exercises.NewParser()
	exPath := parser.GetExerciseFilePath(brainPath, exercise.ID, string(detection.Type))
	if err := parser.WriteExercise(exercise, exPath); err != nil {
		fmt.Printf("Error creating exercise: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Exercise created: %s\n", exPath)
}

func generateExerciseID(name string) string {
	// Convert name to lowercase, replace spaces with hyphens
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	// Remove special characters
	id = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, id)
	return id
}
