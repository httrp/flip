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

// ExerciseListCmd lists all exercises
var ExerciseListCmd = &cobra.Command{
	Use:   "list",
	Short: lang.GetText("exercise.list.short"),
	Long:  lang.GetText("exercise.list.long"),
	Run:   runExerciseList,
}

func init() {
	ExerciseListCmd.Flags().StringP("context", "c", "", "Filter by context (e.g., Sport, Music)")
	ExerciseListCmd.Flags().StringSliceP("tags", "g", []string{}, "Filter by tags")
}

func runExerciseList(cmd *cobra.Command, args []string) {
	// When invoked from the interactive menu, cmd may be nil.
	// In that case, default filters are empty.
	var filterContext string
	var filterTags []string

	activeBrain, err := getActiveBrain()
	if err != nil {
		fmt.Println("⚠️  No active brain selected.")
		fmt.Println("Use 'flip brain set-default <name>' or switch workspace.")
		return
	}
	
	brainPath := activeBrain.Path
	detection, err := brain.NewDetector().DetectBrainType(brainPath)
	if err != nil {
		fmt.Printf("⚠️  Could not detect brain type: %v\n", err)
		return
	}
	if !detection.Compatible {
		fmt.Println("⚠️  Active brain is not a compatible directory.")
		return
	}

	// Get filters if cmd is available (CLI path); otherwise keep defaults (menu path)
	if cmd != nil && cmd.Flags() != nil {
		if v, err := cmd.Flags().GetString("context"); err == nil {
			filterContext = v
		}
		if v, err := cmd.Flags().GetStringSlice("tags"); err == nil {
			filterTags = v
		}
	}

	// Scan exercises
	scanner := exercises.NewScanner()
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		fmt.Printf("⚠️  Error scanning exercises: %v\n", err)
		return
	}

	// Apply filters
	var filteredExercises []*exercises.Exercise
	for _, ex := range allExercises {
		// Filter by context if specified
		if filterContext != "" && !strings.EqualFold(ex.Context, filterContext) {
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
		fmt.Println("ℹ️  No exercises found in the active brain.")
		fmt.Println("Tip: Create one via 'flip exercise new'.")
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
