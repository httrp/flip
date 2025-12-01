package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// ExercisePlanNewCmd creates a new exercise plan
var ExercisePlanNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new exercise plan",
	Long:  "Create a new exercise plan that groups multiple exercises together for structured training.",
	Run:   runExercisePlanNew,
}

func runExercisePlanNew(cmd *cobra.Command, args []string) {
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

	plan := &exercises.ExercisePlan{
		Created: time.Now(),
		Items:   []exercises.PlanItem{},
	}

	// Name
	namePrompt := promptui.Prompt{
		Label: "Plan Name",
	}
	plan.Name, err = namePrompt.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Description
	descPrompt := promptui.Prompt{
		Label: "Description (optional)",
	}
	plan.Description, _ = descPrompt.Run()

	// Schedule
	schedulePrompt := promptui.Prompt{
		Label: "Schedule (e.g., 'monday, wednesday, friday' or 'daily')",
	}
	scheduleInput, _ := schedulePrompt.Run()
	if scheduleInput != "" {
		schedule := strings.Split(scheduleInput, ",")
		for _, day := range schedule {
			plan.Schedule = append(plan.Schedule, strings.TrimSpace(day))
		}
	}

	// Add exercises
	scanner := exercises.NewScanner()
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		fmt.Printf("Error scanning exercises: %v\n", err)
		os.Exit(1)
	}

	if len(allExercises) == 0 {
		fmt.Println("No exercises found. Create exercises first with 'flip exercise new'")
		os.Exit(1)
	}

	fmt.Println("\nAdd exercises to plan (leave empty to finish):")
	order := 1
	for {
		exerciseNames := make([]string, len(allExercises)+1)
		exerciseMap := make(map[string]*exercises.Exercise)
		for i, ex := range allExercises {
			exerciseNames[i] = fmt.Sprintf("%s (%s)", ex.Name, string(ex.Type))
			exerciseMap[exerciseNames[i]] = ex
		}
		exerciseNames[len(allExercises)] = "Done adding exercises"

		selectPrompt := promptui.Select{
			Label: fmt.Sprintf("Exercise #%d", order),
			Items: exerciseNames,
		}
		_, selected, err := selectPrompt.Run()
		if err != nil || selected == "Done adding exercises" {
			break
		}

		exercise := exerciseMap[selected]

		// Optional target
		targetPrompt := promptui.Prompt{
			Label: "Target (e.g., '3x12 reps', '30 min') - optional",
		}
		target, _ := targetPrompt.Run()

		plan.Items = append(plan.Items, exercises.PlanItem{
			ExerciseID: exercise.ID,
			Exercise:   fmt.Sprintf("[[%s]]", exercise.Name),
			Order:      order,
			Target:     target,
		})

		order++
	}

	if len(plan.Items) == 0 {
		fmt.Println("No exercises added. Plan not created.")
		os.Exit(1)
	}

	// Generate ID
	plan.ID = generateExerciseID(plan.Name)

	// Write plan file
	parser := exercises.NewParser()
	planPath := parser.GetPlanFilePath(brainPath, plan.ID, string(detection.Type))
	if err := parser.WritePlan(plan, planPath); err != nil {
		fmt.Printf("Error creating plan: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Exercise plan created: %s\n", planPath)
}
