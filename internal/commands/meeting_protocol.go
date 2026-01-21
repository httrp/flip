package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/httrp/flip/internal/protocols"
	"github.com/spf13/cobra"
)

func NewMeetingProtocolCommand() *cobra.Command {
	var seriesName string
	var fromDate string
	var toDate string
	var outputPath string
	var format string

	cmd := &cobra.Command{
		Use:   "protocol [meeting-file]",
		Short: "Generate meeting protocol (single or rolling)",
		Long: `Generate a structured meeting protocol from meeting notes.

Examples:
  # Single meeting protocol
  flip meeting protocol meetings/2026-01-15-meeting-kickoff.md
  
  # Rolling protocol for a series
  flip meeting protocol --series "Copilot Jour fixe"
  
  # Rolling protocol with date range
  flip meeting protocol --series "Sprint Planning" --from 2026-01-01 --to 2026-01-31
  
  # Save to file
  flip meeting protocol meetings/2026-01-15-meeting-kickoff.md --output ~/Desktop/protocol.md
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if we're generating single or rolling protocol
			if seriesName != "" {
				return generateRollingProtocol(seriesName, fromDate, toDate, outputPath, format)
			}

			if len(args) == 0 {
				return fmt.Errorf("please provide a meeting file or use --series for rolling protocol")
			}

			meetingFile := args[0]
			return generateSingleProtocol(meetingFile, outputPath, format)
		},
	}

	cmd.Flags().StringVar(&seriesName, "series", "", "Generate rolling protocol for series")
	cmd.Flags().StringVar(&fromDate, "from", "", "Start date for rolling protocol (YYYY-MM-DD)")
	cmd.Flags().StringVar(&toDate, "to", "", "End date for rolling protocol (YYYY-MM-DD)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: print to stdout)")
	cmd.Flags().StringVarP(&format, "format", "f", "markdown", "Output format (markdown)")

	return cmd
}

// generateSingleProtocol creates a protocol for a single meeting
func generateSingleProtocol(meetingFile, outputPath, format string) error {
	// Resolve file path
	absPath, err := resolveFilePath(meetingFile)
	if err != nil {
		return fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Parse meeting file
	meeting, err := protocols.ParseMeetingFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to parse meeting file: %w", err)
	}

	// Generate protocol
	protocol := protocols.GenerateSingleProtocol(meeting)

	// Render output
	var output string
	switch format {
	case "markdown", "md":
		output = protocol.RenderMarkdown()
	default:
		return fmt.Errorf("unsupported format: %s (currently only 'markdown' is supported)", format)
	}

	// Write output
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("✓ Protocol saved to: %s\n", outputPath)
	} else {
		fmt.Println(output)
	}

	return nil
}

// generateRollingProtocol creates a protocol for a series of meetings
func generateRollingProtocol(seriesName, fromDate, toDate, outputPath, format string) error {
	// Get active brain
	activeBrain, err := getActiveBrain()
	if err != nil {
		return fmt.Errorf("failed to get active brain: %w", err)
	}

	brainPath := activeBrain.Path
	brainType := activeBrain.Type

	// Find meetings directory
	meetingsDir := filepath.Join(brainPath, "meetings")
	if brainType == "logseq" {
		meetingsDir = brainPath // Logseq uses flat structure
	}

	if _, err := os.Stat(meetingsDir); os.IsNotExist(err) {
		return fmt.Errorf("meetings directory not found: %s", meetingsDir)
	}

	// Scan for meetings in the series
	meetings, err := findMeetingsInSeries(meetingsDir, seriesName, fromDate, toDate)
	if err != nil {
		return fmt.Errorf("failed to find meetings: %w", err)
	}

	if len(meetings) == 0 {
		return fmt.Errorf("no meetings found for series: %s", seriesName)
	}

	// Generate rolling protocol
	protocol := protocols.GenerateRollingProtocol(meetings, seriesName)
	if protocol == nil {
		return fmt.Errorf("failed to generate rolling protocol")
	}

	// Render output
	var output string
	switch format {
	case "markdown", "md":
		output = protocol.RenderMarkdown()
	default:
		return fmt.Errorf("unsupported format: %s (currently only 'markdown' is supported)", format)
	}

	// Write output
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("✓ Rolling protocol saved to: %s\n", outputPath)
		fmt.Printf("  Meetings included: %d\n", len(meetings))
	} else {
		fmt.Println(output)
	}

	return nil
}

// findMeetingsInSeries scans for meetings belonging to a series
func findMeetingsInSeries(meetingsDir, seriesName, fromDate, toDate string) ([]*protocols.MeetingContent, error) {
	var meetings []*protocols.MeetingContent

	// Walk through meetings directory
	err := filepath.Walk(meetingsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-markdown files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Parse meeting file
		meeting, parseErr := protocols.ParseMeetingFile(path)
		if parseErr != nil {
			// Skip files that can't be parsed
			return nil
		}

		// Check if meeting belongs to the series
		if meeting.Series != seriesName {
			return nil
		}

		// Check date range if specified
		if fromDate != "" || toDate != "" {
			meetingDate, dateErr := meeting.GetDateParsed()
			if dateErr != nil {
				return nil
			}

			if fromDate != "" {
				from, _ := parseDate(fromDate)
				if meetingDate.Before(from) {
					return nil
				}
			}

			if toDate != "" {
				to, _ := parseDate(toDate)
				if meetingDate.After(to) {
					return nil
				}
			}
		}

		meetings = append(meetings, meeting)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort meetings by date
	sort.Slice(meetings, func(i, j int) bool {
		dateI, _ := meetings[i].GetDateParsed()
		dateJ, _ := meetings[j].GetDateParsed()
		return dateI.Before(dateJ)
	})

	return meetings, nil
}

// resolveFilePath resolves a file path (relative to cwd or brain)
func resolveFilePath(path string) (string, error) {
	// If absolute path, use it
	if filepath.IsAbs(path) {
		return path, nil
	}

	// Try relative to current directory
	absPath, err := filepath.Abs(path)
	if err == nil {
		if _, err := os.Stat(absPath); err == nil {
			return absPath, nil
		}
	}

	// Try relative to active brain
	activeBrain, err := getActiveBrain()
	if err != nil {
		return "", err
	}

	brainRelPath := filepath.Join(activeBrain.Path, path)
	if _, err := os.Stat(brainRelPath); err == nil {
		return brainRelPath, nil
	}

	return "", fmt.Errorf("file not found: %s", path)
}

// parseDate parses a date string in YYYY-MM-DD format
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}
