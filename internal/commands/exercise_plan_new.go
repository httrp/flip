package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	ui "github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// ExercisePlanNewCmd creates a new exercise plan
var ExercisePlanNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new exercise plan",
	Long:  "Create a new exercise plan that groups multiple exercises together for structured training.",
	RunE:  runExercisePlanNew,
}

func runExercisePlanNew(cmd *cobra.Command, args []string) error {
	activeBrain, err := getActiveBrain()
	if err != nil {
		return fmt.Errorf("getting active brain: %w", err)
	}

	brainPath := activeBrain.Path
	detection, err := brain.NewDetector().DetectBrainType(brainPath)
	if err != nil {
		return fmt.Errorf("detecting brain: %w", err)
	}

	if !detection.Compatible {
		return fmt.Errorf("not in a compatible brain directory")
	}

	plan := &exercises.ExercisePlan{
		Created: time.Now(),
		Items:   []exercises.PlanItem{},
	}

	// Name
	plan.Name, err = ui.RunInput("Plan Name", "", "", nil)
	if err != nil {
		return fmt.Errorf("reading plan name: %w", err)
	}

	// Description
	plan.Description, _ = ui.RunInput("Description (optional)", "", "", nil)

	// Schedule
	scheduleInput, _ := ui.RunInput("Schedule (e.g., 'monday, wednesday, friday' or 'daily')", "", "", nil)
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
		return fmt.Errorf("scanning exercises: %w", err)
	}

	if len(allExercises) == 0 {
		return fmt.Errorf("no exercises found - create exercises first with 'flip exercise new'")
	}

	fmt.Println("\nAdd exercises to plan (leave empty to finish):")
	order := 1
	for {
		exerciseNames := make([]string, len(allExercises)+1)
		exerciseMap := make(map[string]*exercises.Exercise)
		for i, ex := range allExercises {
			contextStr := "no context"
			if ex.Context != "" {
				contextStr = ex.Context
			}
			exerciseNames[i] = fmt.Sprintf("%s (%s)", ex.Name, contextStr)
			exerciseMap[exerciseNames[i]] = ex
		}
		exerciseNames[len(allExercises)] = "Done adding exercises"

		exerciseItems := make([]ui.SelectItem, len(exerciseNames))
		for i, name := range exerciseNames {
			exerciseItems[i] = ui.SelectItem{Label: name, Value: name}
		}

		_, selected, err := ui.RunSelect(fmt.Sprintf("Exercise #%d", order), exerciseItems, 10)
		if err != nil || selected == "Done adding exercises" {
			break
		}

		exercise := exerciseMap[selected]

		// Optional target
		target, _ := ui.RunInput("Target (e.g., '3x12 reps', '30 min') - optional", "", "", nil)

		plan.Items = append(plan.Items, exercises.PlanItem{
			ExerciseID: exercise.ID,
			Exercise:   fmt.Sprintf("[[%s]]", exercise.Name),
			Order:      order,
			Target:     target,
		})

		order++
	}

	if len(plan.Items) == 0 {
		return fmt.Errorf("no exercises added - plan not created")
	}

	// Generate ID
	plan.ID = generateExerciseID(plan.Name)

	// Write plan file
	parser := exercises.NewParser()
	planPath := parser.GetPlanFilePath(brainPath, plan.ID, string(detection.Type))
	if err := parser.WritePlan(plan, planPath); err != nil {
		return fmt.Errorf("creating plan: %w", err)
	}

	fmt.Printf("✓ Exercise plan created: %s\n", planPath)
	return nil
}
