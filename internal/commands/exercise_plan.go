package commands

import (
	"github.com/httrp/flip/internal/lang"
	"github.com/spf13/cobra"
)

// NewExercisePlanCommand creates the exercise plan command with subcommands
func NewExercisePlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: lang.GetText("exercise.plan.short"),
		Long:  lang.GetText("exercise.plan.long"),
	}

	cmd.AddCommand(ExercisePlanNewCmd)
	cmd.AddCommand(ExercisePlanListCmd)
	cmd.AddCommand(ExercisePlanShowCmd)
	cmd.AddCommand(ExercisePlanEditCmd)

	return cmd
}
