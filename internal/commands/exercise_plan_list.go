package commands

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/spf13/cobra"
)

// ExercisePlanListCmd lists all exercise plans
var ExercisePlanListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all exercise plans",
	Long:  "List all exercise plans in your brain with basic stats.",
	RunE:  runExercisePlanList,
}

func runExercisePlanList(cmd *cobra.Command, args []string) error {
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
	plans, err := scanner.ScanPlans(brainPath, string(detection.Type))
	if err != nil {
		return fmt.Errorf("scanning plans: %w", err)
	}

	if len(plans) == 0 {
		fmt.Println("No exercise plans found")
		return nil
	}

	// Sort by name
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Name < plans[j].Name
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tEXERCISES\tSCHEDULE\tSTATUS")

	for _, plan := range plans {
		schedule := "-"
		if len(plan.Schedule) > 0 {
			schedule = plan.Schedule[0]
			if len(plan.Schedule) > 1 {
				schedule += fmt.Sprintf(" +%d", len(plan.Schedule)-1)
			}
		}

		status := "active"
		if plan.Status != "" {
			status = plan.Status
		}

		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			plan.Name,
			len(plan.Items),
			schedule,
			status,
		)
	}

	w.Flush()
	return nil
}
