package commands

import (
	"fmt"
	"strings"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// ExercisePlanEditCmd edits an existing exercise plan (add exercises)
var ExercisePlanEditCmd = &cobra.Command{
	Use:   "edit [plan-id]",
	Short: lang.GetText("exercise.plan.edit.short"),
	Long:  lang.GetText("exercise.plan.edit.long"),
	RunE:  runExercisePlanEdit,
}

func runExercisePlanEdit(cmd *cobra.Command, args []string) error {
	// Select brain
	selectedBrain, err := selectBrainForOperation(lang.GetText("prompts.select_brain_for_plan"))
	if err != nil {
		return fmt.Errorf("selecting brain: %w", err)
	}
	brainPath := selectedBrain.Path

	detection, err := brain.NewDetector().DetectBrainType(brainPath)
	if err != nil {
		return fmt.Errorf("detecting brain: %w", err)
	}
	if !detection.Compatible {
		return fmt.Errorf("not in a compatible brain directory")
	}

	scanner := exercises.NewScanner()
	parser := exercises.NewParser()

	// Load all plans
	plans, err := scanner.ScanPlans(brainPath, string(detection.Type))
	if err != nil {
		return fmt.Errorf("scanning plans: %w", err)
	}
	if len(plans) == 0 {
		return fmt.Errorf("no exercise plans found - create one with 'flip exercise plan new'")
	}

	// Pick plan
	var plan *exercises.ExercisePlan
	if len(args) > 0 {
		pid := args[0]
		for _, p := range plans {
			if p.ID == pid {
				plan = p
				break
			}
		}
		if plan == nil {
			return fmt.Errorf("plan not found: %s", pid)
		}
	} else {
		items := make([]string, len(plans))
		idxMap := make(map[int]*exercises.ExercisePlan)
		for i, p := range plans {
			items[i] = fmt.Sprintf("%s — %d items", p.Name, len(p.Items))
			idxMap[i] = p
		}
		sel := promptui.Select{Label: lang.GetText("prompts.select_plan"), Items: items}
		i, _, err := sel.Run()
		if err != nil {
			fmt.Println("Cancelled")
			return nil
		}
		plan = idxMap[i]
	}

	// Load exercises to add
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		return fmt.Errorf("scanning exercises: %w", err)
	}
	if len(allExercises) == 0 {
		return fmt.Errorf("no exercises found - create exercises first with 'flip exercise new'")
	}

	// Build a set of existing items to avoid duplicate orders
	maxOrder := 0
	for _, it := range plan.Items {
		if it.Order > maxOrder {
			maxOrder = it.Order
		}
	}

	// Append exercises
	fmt.Printf("\nEditing plan: %s\n", plan.Name)
	for {
		names := make([]string, len(allExercises)+1)
		byName := make(map[string]*exercises.Exercise)
		for i, ex := range allExercises {
			c := "no context"
			if ex.Context != "" {
				c = ex.Context
			}
			names[i] = fmt.Sprintf("%s (%s)", ex.Name, c)
			byName[names[i]] = ex
		}
		names[len(allExercises)] = lang.GetText("prompts.done")

		sel := promptui.Select{Label: lang.GetText("prompts.add_exercise"), Items: names}
		_, choice, err := sel.Run()
		if err != nil || choice == lang.GetText("prompts.done") {
			break
		}
		ex := byName[choice]

		tPrompt := promptui.Prompt{Label: lang.GetText("prompts.target_optional")}
		target, _ := tPrompt.Run()

		maxOrder++
		plan.Items = append(plan.Items, exercises.PlanItem{
			ExerciseID: ex.ID,
			Exercise:   fmt.Sprintf("[[%s]]", ex.Name),
			Order:      maxOrder,
			Target:     strings.TrimSpace(target),
		})
		fmt.Println("✓ Added")
	}

	// Save if changed
	planPath := parser.GetPlanFilePath(brainPath, plan.ID, string(detection.Type))
	if err := parser.WritePlan(plan, planPath); err != nil {
		return fmt.Errorf("saving plan: %w", err)
	}
	fmt.Printf("✓ Plan updated: %s\n", planPath)
	return nil
}
