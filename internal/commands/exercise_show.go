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
	ui "github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// ExerciseShowCmd shows details and progress for an exercise
var ExerciseShowCmd = &cobra.Command{
	Use:   "show [exercise-id]",
	Short: lang.GetText("exercise.show.short"),
	Long:  lang.GetText("exercise.show.long"),
	Args:  cobra.MaximumNArgs(1),
	RunE:  runExerciseShow,
}

func init() {
	ExerciseShowCmd.Flags().IntP("limit", "n", 10, "Number of recent sessions to show")
}

func runExerciseShow(cmd *cobra.Command, args []string) error {
	var exerciseID string
	if len(args) > 0 {
		exerciseID = args[0]
	}

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

	scanner := exercises.NewScanner()

	// Load all exercises
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		return fmt.Errorf("scanning exercises: %w", err)
	}

	if len(allExercises) == 0 {
		return fmt.Errorf("no exercises found - create one with 'flip exercise new'")
	}

	// If no ID provided, let user select
	if exerciseID == "" {
		exerciseNames := make([]string, len(allExercises))
		exerciseMap := make(map[string]*exercises.Exercise)
		for i, ex := range allExercises {
			contextStr := "no context"
			if ex.Context != "" {
				contextStr = ex.Context
			}
			exerciseNames[i] = fmt.Sprintf("%s (%s)", ex.Name, contextStr)
			exerciseMap[exerciseNames[i]] = ex
		}

		selectItems := make([]ui.SelectItem, len(exerciseNames))
		for i, name := range exerciseNames {
			selectItems[i] = ui.SelectItem{Label: name, Value: name}
		}

		_, selected, err := ui.RunSelect("Select Exercise", selectItems, 10)
		if err != nil {
			return fmt.Errorf("selecting exercise: %w", err)
		}

		exerciseID = exerciseMap[selected].ID
	}

	// Find the exercise
	var exercise *exercises.Exercise
	for _, ex := range allExercises {
		if ex.ID == exerciseID {
			exercise = ex
			break
		}
	}

	if exercise == nil {
		return fmt.Errorf("exercise not found: %s", exerciseID)
	}

	// Load sessions
	sessions, err := scanner.ScanSessions(brainPath, exerciseID, string(detection.Type))
	if err != nil {
		return fmt.Errorf("loading sessions: %w", err)
	}

	// Sort sessions by date (newest first)
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Date.After(sessions[j].Date)
	})

	// Display exercise details
	fmt.Printf("Exercise: %s\n", exercise.Name)
	if exercise.Context != "" {
		fmt.Printf("Context: %s\n", exercise.Context)
	}
	if exercise.Description != "" {
		fmt.Printf("Description: %s\n", exercise.Description)
	}
	if exercise.Goal != "" {
		fmt.Printf("Goal: %s\n", exercise.Goal)
	}
	if len(exercise.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(exercise.Tags, ", "))
	}

	// Display variants
	if len(exercise.Variants) > 0 {
		fmt.Printf("\nVariants: %d\n", len(exercise.Variants))
		for i, v := range exercise.Variants {
			if v.Name != "" {
				fmt.Printf("  %d. %s\n", i+1, v.Name)
			} else {
				fmt.Printf("  %d. (unnamed)\n", i+1)
			}
		}
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
		fmt.Fprintln(w, "DATE\tVARIANT\tDURATION\tPROPERTIES")

		for i := 0; i < min(limit, len(sessions)); i++ {
			s := sessions[i]

			// Variant name
			variantStr := "-"
			if s.VariantName != "" {
				variantStr = s.VariantName
			}

			// Build properties string
			propsStr := "-"
			if len(s.Properties) > 0 {
				var props []string
				for k, v := range s.Properties {
					props = append(props, fmt.Sprintf("%s: %v", k, v))
				}
				propsStr = strings.Join(props, ", ")
			}

			fmt.Fprintf(w, "%s\t%s\t%dm\t%s\n",
				s.Date.Format("2006-01-02"),
				variantStr,
				s.Duration,
				propsStr,
			)

			if s.Notes != "" {
				fmt.Fprintf(w, "\t\t\t%s\n", s.Notes)
			}
		}

		w.Flush()
	} else {
		fmt.Println("No sessions tracked yet")
	}

	return nil
}

// Use min from new.go; avoid duplicate definition
