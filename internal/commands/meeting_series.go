package commands

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/httrp/flip/internal/brain"
	"github.com/spf13/cobra"
)

// NewMeetingSeriesCommand manages meeting series operations.
func NewMeetingSeriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "series",
		Short: "Work with meeting series",
	}

	cmd.AddCommand(NewMeetingSeriesListCommand())

	return cmd
}

// NewMeetingSeriesListCommand lists known meeting series in the active brain.
func NewMeetingSeriesListCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List meeting series in the active brain",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeBrain, err := getActiveBrain()
			if err != nil {
				return err
			}

			detector := brain.NewDetector()
			detection, err := detector.DetectBrainType(activeBrain.Path)
			if err != nil {
				return fmt.Errorf("failed to detect brain type: %w", err)
			}

			series, err := findMeetingSeries(activeBrain.Path, detection.Type)
			if err != nil {
				return fmt.Errorf("failed to find meeting series: %w", err)
			}

			sort.Slice(series, func(i, j int) bool {
				if series[i].Count == series[j].Count {
					return series[i].Name < series[j].Name
				}
				return series[i].Count > series[j].Count
			})

			if jsonOutput {
				seriesResults := make([]MeetingSeriesResult, len(series))
				for i, s := range series {
					seriesResults[i] = MeetingSeriesResult{Name: s.Name, Count: s.Count}
				}
				OutputJSONSuccess("meeting-series-list", MeetingSeriesListResult{
					Series:    seriesResults,
					BrainName: activeBrain.Name,
					BrainPath: activeBrain.Path,
				})
				return nil
			}

			if len(series) == 0 {
				fmt.Printf("No meeting series found in %s\n", activeBrain.Name)
				return nil
			}

			fmt.Printf("Meeting series in %s\n\n", activeBrain.Name)
			for _, s := range series {
				fmt.Printf("- %s (%d meetings)\n", s.Name, s.Count)
				fmt.Printf("  latest: %s\n", filepath.Base(s.LatestFile))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON")

	return cmd
}
