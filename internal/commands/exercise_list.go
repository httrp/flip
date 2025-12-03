package commands

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	"github.com/spf13/cobra"
)

// ExerciseListCmd lists all exercises
var ExerciseListCmd = &cobra.Command{
	Use:   "list",
	Short: lang.GetText("exercise.list.short"),
	Long:  lang.GetText("exercise.list.long"),
	Run:   runExerciseList,
}

func init() {
	ExerciseListCmd.Flags().StringP("type", "t", "", "Filter by exercise type")
	ExerciseListCmd.Flags().StringSliceP("tags", "g", []string{}, "Filter by tags")
}

func runExerciseList(cmd *cobra.Command, args []string) {
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

	// Get filters
	filterType, _ := cmd.Flags().GetString("type")
	filterTags, _ := cmd.Flags().GetStringSlice("tags")

	// Scan exercises
	scanner := exercises.NewScanner()
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		fmt.Printf("Error scanning exercises: %v\n", err)
		os.Exit(1)
	}

	// Apply filters
	var filteredExercises []*exercises.Exercise
	for _, ex := range allExercises {
		// Filter by context if specified
		if filterType != "" && ex.Context != filterType {
			continue
		}

		if len(filterTags) > 0 && !hasAnyTag(ex.Tags, filterTags) {
			continue
		}

		filteredExercises = append(filteredExercises, ex)
	}

	// Sort by name
	sort.Slice(filteredExercises, func(i, j int) bool {
		return filteredExercises[i].Name < filteredExercises[j].Name
	})

	// Display
	if len(filteredExercises) == 0 {
		fmt.Println("No exercises found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCONTEXT\tVARIANTS\tSESSIONS\tLAST TRACKED")

	for _, ex := range filteredExercises {
		lastSession := "-"
		if ex.LastSession != nil {
			lastSession = ex.LastSession.Format("2006-01-02")
		}

		context := "-"
		if ex.Context != "" {
			context = ex.Context
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\n",
			ex.Name,
			context,
			len(ex.Variants),
			ex.SessionCount,
			lastSession,
		)
	}

	w.Flush()
}

func hasAnyTag(exerciseTags []string, filterTags []string) bool {
	for _, filterTag := range filterTags {
		for _, exerciseTag := range exerciseTags {
			if exerciseTag == filterTag {
				return true
			}
		}
	}
	return false
}
