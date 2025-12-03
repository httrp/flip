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
		Created:  time.Now(),
		Variants: []exercises.ExerciseVariant{},
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

	// Context (optional, user-defined)
	contextPrompt := promptui.Prompt{
		Label: "Context (e.g., Sport, Music, Basketball, Drums) - optional",
	}
	exercise.Context, _ = contextPrompt.Run()

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

	// Variants - always create at least one
	fmt.Println("\n━━━ Variant Configuration ━━━")
	fmt.Println("Every exercise needs at least one variant.")
	fmt.Println("Variants define different ways to practice this exercise.")
	fmt.Println()
	
	variantNum := 1
	for {
		fmt.Printf("━━ Variant %d ━━\n", variantNum)
		
		variant := exercises.ExerciseVariant{
			TrackingProperties: make(map[string]string),
		}
		
		// Optional variant name
		varNamePrompt := promptui.Prompt{
			Label: fmt.Sprintf("Variant Name (optional, e.g., 'Balance & Control')"),
		}
		variant.Name, _ = varNamePrompt.Run()
		
		// Variant description
		varDescPrompt := promptui.Prompt{
			Label: "Description (optional, explain what makes this variant unique)",
		}
		variant.Description, _ = varDescPrompt.Run()
		
		// Tracking properties for this variant
		fmt.Println("\nTracking properties for this variant:")
		fmt.Println("Examples: tempo=bpm, focus=text, reps=count, level=1-10")
		for {
			propPrompt := promptui.Prompt{
				Label: "Property (name=unit or empty to finish)",
			}
			propInput, _ := propPrompt.Run()
			if strings.TrimSpace(propInput) == "" {
				break
			}

			parts := strings.SplitN(propInput, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				variant.TrackingProperties[key] = value
			}
		}
		
		exercise.Variants = append(exercise.Variants, variant)
		
		// Ask if user wants to add another variant
		if variantNum == 1 {
			fmt.Println()
		}
		
		addMorePrompt := promptui.Select{
			Label: "Add another variant?",
			Items: []string{"Yes", "No"},
		}
		addIdx, _, err := addMorePrompt.Run()
		if err != nil || addIdx == 1 {
			break
		}
		
		fmt.Println()
		variantNum++
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

	fmt.Printf("✓ Exercise created: %s\n\n", exPath)
	
	// Ask what to do next
	nextPrompt := promptui.Select{
		Label: "What would you like to do next?",
		Items: []string{"View/Edit in editor", "Done - Track later with 'flip exercise track'"},
	}
	
	nextIdx, _, err := nextPrompt.Run()
	if err != nil {
		return
	}
	
	switch nextIdx {
	case 0: // View/Edit
		if err := promptAndOpenEditor(exPath); err != nil {
			fmt.Printf("⚠️  Could not open editor: %v\n", err)
		}
	case 1: // Done
		fmt.Printf("💡 To track a session, run: flip exercise track %s\n", exercise.ID)
	}
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
