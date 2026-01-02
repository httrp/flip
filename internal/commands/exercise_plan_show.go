package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/spf13/cobra"
)

// ExercisePlanShowCmd shows details for an exercise plan
var ExercisePlanShowCmd = &cobra.Command{
	Use:   "show [plan-id]",
	Short: "Show exercise plan details",
	Long:  "Show details for a specific exercise plan including exercises and recent sessions.",
	Args:  cobra.ExactArgs(1),
	RunE:  runExercisePlanShow,
}

func runExercisePlanShow(cmd *cobra.Command, args []string) error {
	planID := args[0]

	brainPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(brainPath)
	if err != nil {
		return fmt.Errorf("detecting brain: %w", err)
	}

	if !detection.Compatible {
		return fmt.Errorf("not in a compatible brain directory")
	}

	scanner := exercises.NewScanner()

	// Load plan
	plans, err := scanner.ScanPlans(brainPath, string(detection.Type))
	if err != nil {
		return fmt.Errorf("scanning plans: %w", err)
	}

	var plan *exercises.ExercisePlan
	for _, p := range plans {
		if p.ID == planID {
			plan = p
			break
		}
	}

	if plan == nil {
		return fmt.Errorf("plan not found: %s", planID)
	}

	// Display plan details
	fmt.Printf("Plan: %s\n", plan.Name)
	if plan.Description != "" {
		fmt.Printf("Description: %s\n", plan.Description)
	}
	if len(plan.Schedule) > 0 {
		fmt.Printf("Schedule: %s\n", strings.Join(plan.Schedule, ", "))
	}
	fmt.Printf("Status: %s\n", plan.Status)
	fmt.Println()

	// Display exercises in plan
	fmt.Printf("Exercises (%d):\n", len(plan.Items))
	for _, item := range plan.Items {
		target := ""
		if item.Target != "" {
			target = fmt.Sprintf(" - %s", item.Target)
		}
		fmt.Printf("  %d. %s%s\n", item.Order, item.ExerciseID, target)
	}
	fmt.Println()

	// Display recent sessions
	planSessions, err := scanner.ScanPlanSessions(brainPath, planID, string(detection.Type))
	if err != nil {
		return fmt.Errorf("loading sessions: %w", err)
	}

	if len(planSessions) > 0 {
		fmt.Printf("Recent Sessions (%d total):\n", len(planSessions))
		limit := 5
		if len(planSessions) < limit {
			limit = len(planSessions)
		}
		for i := 0; i < limit; i++ {
			ps := planSessions[i]
			completed := 0
			for _, item := range ps.Items {
				if item.Completed {
					completed++
				}
			}
			fmt.Printf("  %s - %d/%d completed\n", ps.Date.Format("2006-01-02"), completed, len(ps.Items))
		}
	} else {
		fmt.Println("No sessions tracked yet")
	}
	return nil
}
