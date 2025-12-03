package commands

import (
	"github.com/httrp/flip/internal/lang"
	"github.com/spf13/cobra"
)

// NewExerciseCommand creates the exercise command with subcommands
func NewExerciseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exercise",
		Short: lang.GetText("exercise.short"),
		Long:  lang.GetText("exercise.long"),
	}

	// Add subcommands
	cmd.AddCommand(ExerciseNewCmd)
	cmd.AddCommand(ExerciseListCmd)
	cmd.AddCommand(ExerciseTrackCmd)
	cmd.AddCommand(ExerciseShowCmd)
	cmd.AddCommand(ExerciseEditCmd)
	cmd.AddCommand(NewExercisePlanCommand())

	return cmd
}
