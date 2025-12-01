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
	Run:   runExercisePlanList,
}

func runExercisePlanList(cmd *cobra.Command, args []string) {
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
	plans, err := scanner.ScanPlans(brainPath, string(detection.Type))
	if err != nil {
		fmt.Printf("Error scanning plans: %v\n", err)
		os.Exit(1)
	}

	if len(plans) == 0 {
		fmt.Println("No exercise plans found")
		return
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
}
