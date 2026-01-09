package commands

import (
	"fmt"

	"github.com/httrp/flip/internal/brain"
	"github.com/spf13/cobra"
)

// MeetingSeriesResult represents a meeting series for JSON output
type MeetingSeriesResult struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// MeetingSeriesListResult is the response from vscode meetings list-series
type MeetingSeriesListResult struct {
	Series    []MeetingSeriesResult `json:"series"`
	BrainName string                `json:"brain_name"`
	BrainPath string                `json:"brain_path"`
}

// MeetingOrganizationResult represents an organization for JSON output
type MeetingOrganizationResult struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// MeetingOrganizationsListResult is the response from vscode meetings list-organizations
type MeetingOrganizationsListResult struct {
	Organizations []MeetingOrganizationResult `json:"organizations"`
	BrainName     string                      `json:"brain_name"`
	BrainPath     string                      `json:"brain_path"`
}

// NewVSCodeMeetingsCommand creates the vscode meetings subcommand
func NewVSCodeMeetingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "meetings",
		Short: "Meeting-related VS Code integration commands",
	}

	cmd.AddCommand(NewMeetingsListSeriesCommand())
	cmd.AddCommand(NewMeetingsListOrganizationsCommand())

	return cmd
}

// NewMeetingsListSeriesCommand lists all meeting series in the active brain
func NewMeetingsListSeriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-series",
		Short: "List all meeting series",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeBrain, err := getActiveBrain()
			if err != nil {
				return err
			}

			// Detect brain type
			detector := brain.NewDetector()
			detection, err := detector.DetectBrainType(activeBrain.Path)
			if err != nil {
				return fmt.Errorf("failed to detect brain type: %w", err)
			}

			// Find series
			series, err := findMeetingSeries(activeBrain.Path, detection.Type)
			if err != nil {
				return fmt.Errorf("failed to find series: %w", err)
			}

			// Convert to result format
			seriesResults := make([]MeetingSeriesResult, len(series))
			for i, s := range series {
				seriesResults[i] = MeetingSeriesResult{
					Name:  s.Name,
					Count: s.Count,
				}
			}

			result := MeetingSeriesListResult{
				Series:    seriesResults,
				BrainName: activeBrain.Name,
				BrainPath: activeBrain.Path,
			}

			OutputJSONSuccess("meetings-list-series", result)
			return nil
		},
	}

	return cmd
}

// NewMeetingsListOrganizationsCommand lists all organizations found in meeting notes
func NewMeetingsListOrganizationsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-organizations",
		Short: "List all meeting organizations",
		RunE: func(cmd *cobra.Command, args []string) error {
			activeBrain, err := getActiveBrain()
			if err != nil {
				return err
			}

			// Detect brain type
			detector := brain.NewDetector()
			detection, err := detector.DetectBrainType(activeBrain.Path)
			if err != nil {
				return fmt.Errorf("failed to detect brain type: %w", err)
			}

			orgs, err := findMeetingOrganizations(activeBrain.Path, detection.Type)
			if err != nil {
				return fmt.Errorf("failed to find organizations: %w", err)
			}

			orgResults := make([]MeetingOrganizationResult, len(orgs))
			for i, o := range orgs {
				orgResults[i] = MeetingOrganizationResult{Name: o.Name, Count: o.Count}
			}

			result := MeetingOrganizationsListResult{
				Organizations: orgResults,
				BrainName:     activeBrain.Name,
				BrainPath:     activeBrain.Path,
			}

			OutputJSONSuccess("meetings-list-organizations", result)
			return nil
		},
	}

	return cmd
}
