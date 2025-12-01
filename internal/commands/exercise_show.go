package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	"github.com/spf13/cobra"
)

// ExerciseShowCmd shows details and progress for an exercise
var ExerciseShowCmd = &cobra.Command{
	Use:   "show [exercise-id]",
	Short: lang.GetText("exercise.show.short"),
	Long:  lang.GetText("exercise.show.long"),
	Args:  cobra.ExactArgs(1),
	Run:   runExerciseShow,
}

func init() {
	ExerciseShowCmd.Flags().IntP("limit", "n", 10, "Number of recent sessions to show")
}

func runExerciseShow(cmd *cobra.Command, args []string) {
	exerciseID := args[0]

	// Detect brain
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

	// Load exercise
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	var exercise *exercises.Exercise
	for _, ex := range allExercises {
		if ex.ID == exerciseID {
			exercise = ex
			break
		}
	}

	if exercise == nil {
		fmt.Printf("Exercise not found: %s\n", exerciseID)
		os.Exit(1)
	}

	// Load sessions
	sessions, err := scanner.ScanSessions(brainPath, exerciseID, string(detection.Type))
	if err != nil {
		fmt.Printf("Error loading sessions: %v\n", err)
		os.Exit(1)
	}

	// Sort sessions by date (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Date.After(sessions[j].Date)
	})

	// Display exercise details
	fmt.Printf("Exercise: %s\n", exercise.Name)
	fmt.Printf("Type: %s\n", string(exercise.Type))
	if exercise.Description != "" {
		fmt.Printf("Description: %s\n", exercise.Description)
	}
	if exercise.Goal != "" {
		fmt.Printf("Goal: %s\n", exercise.Goal)
	}
	if len(exercise.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(exercise.Tags, ", "))
	}
	fmt.Println()

	// Display statistics
	fmt.Printf("Statistics:\n")
	fmt.Printf("  Total Sessions: %d\n", len(sessions))
	if len(sessions) > 0 {
		totalDuration := 0
		for _, s := range sessions {
			totalDuration += s.Duration
		}
		avgDuration := totalDuration / len(sessions)
		fmt.Printf("  Total Time: %d minutes\n", totalDuration)
		fmt.Printf("  Average Duration: %d minutes\n", avgDuration)
		fmt.Printf("  First Tracked: %s\n", sessions[len(sessions)-1].Date)
		fmt.Printf("  Last Tracked: %s\n", sessions[0].Date)
	}
	fmt.Println()

	// Display recent sessions
	limit, _ := cmd.Flags().GetInt("limit")
	if len(sessions) > 0 {
		fmt.Printf("Recent Sessions (showing %d):\n", min(limit, len(sessions)))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "DATE\tDURATION\tVALUE\tVARIANT")

		for i := 0; i < min(limit, len(sessions)); i++ {
			s := sessions[i]
			
			valueStr := "-"
			if s.Value > 0 {
				unit := "reps"
				if exercise.TargetUnit != "" {
					unit = exercise.TargetUnit
				}
				valueStr = fmt.Sprintf("%.1f %s", s.Value, unit)
			}

			variantStr := "-"
			if s.Variant != "" {
				variantStr = s.Variant
			}

            // No rating in MVP

			fmt.Fprintf(w, "%s\t%dm\t%s\t%s\n",
				s.Date.Format("2006-01-02"),
				s.Duration,
				valueStr,
				variantStr,
			)

			if s.Notes != "" {
				fmt.Fprintf(w, "\t\t\t%s\n", s.Notes)
			}
		}

		w.Flush()
	} else {
		fmt.Println("No sessions tracked yet")
	}
}

// Use min from new.go; avoid duplicate definition
