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
	var write bool
	var sync bool

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
			if write && outputPath != "" {
				return fmt.Errorf("use either --write or --output, not both")
			}
			if sync && seriesName == "" {
				return fmt.Errorf("--sync requires --series")
			}
			if sync {
				write = true
			}

			// Check if we're generating single or rolling protocol
			if seriesName != "" {
				return generateRollingProtocol(seriesName, fromDate, toDate, outputPath, format, write, sync)
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
	cmd.Flags().BoolVar(&write, "write", false, "Write rolling protocol to the default protocol location")
	cmd.Flags().BoolVar(&sync, "sync", false, "Update the rolling protocol file for a series in place")

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
func generateRollingProtocol(seriesName, fromDate, toDate, outputPath, format string, write, sync bool) error {
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

	if write && outputPath == "" {
		outputPath, err = defaultRollingProtocolPath(brainPath, brainType, seriesName)
		if err != nil {
			return err
		}
	}

	// Write output
	if outputPath != "" {
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create protocol directory: %w", err)
		}
		if sync {
			output, err = mergeRollingProtocolPreservedSections(outputPath, output)
			if err != nil {
				return err
			}
		}
		if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		if err := ensureRollingProtocolBacklinks(meetings, outputPath, seriesName); err != nil {
			return err
		}
		if sync {
			fmt.Printf("✓ Rolling protocol updated: %s\n", outputPath)
		} else {
			fmt.Printf("✓ Rolling protocol saved to: %s\n", outputPath)
		}
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

		if info.IsDir() {
			if info.Name() == "protocols" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories and non-markdown files
		if !strings.HasSuffix(info.Name(), ".md") {
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

func defaultRollingProtocolPath(brainPath, brainType, seriesName string) (string, error) {
	meetingsDir := filepath.Join(brainPath, "meetings")
	if brainType == "logseq" {
		meetingsDir = filepath.Join(brainPath, "pages")
	}

	slug := slugifySeriesName(seriesName)
	if slug == "" {
		return "", fmt.Errorf("series name cannot be empty")
	}

	return filepath.Join(meetingsDir, "protocols", slug+"-rolling-protocol.md"), nil
}

func slugifySeriesName(seriesName string) string {
	slug := strings.ToLower(strings.TrimSpace(seriesName))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ":", "", ".", "", ",", "", "(", "", ")", "", "\"", "", "'", "")
	slug = replacer.Replace(slug)
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return strings.Trim(slug, "-")
}

func mergeRollingProtocolPreservedSections(path, generated string) (string, error) {
	existingBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return generated, nil
		}
		return "", fmt.Errorf("failed to read existing rolling protocol: %w", err)
	}

	existing := string(existingBytes)
	merged := generated
	for _, section := range protocols.RollingProtocolPreservedSections {
		merged = preserveMarkedSection(
			merged,
			existing,
			section.StartMarker,
			section.EndMarker,
		)
	}

	return merged, nil
}

func preserveMarkedSection(generated, existing, startMarker, endMarker string) string {
	existingBlock, ok := extractMarkedSection(existing, startMarker, endMarker)
	if !ok {
		return generated
	}

	generatedStart := strings.Index(generated, startMarker)
	if generatedStart == -1 {
		return generated
	}
	generatedEnd := strings.Index(generated[generatedStart+len(startMarker):], endMarker)
	if generatedEnd == -1 {
		return generated
	}
	generatedEnd += generatedStart + len(startMarker)

	return generated[:generatedStart] + existingBlock + generated[generatedEnd+len(endMarker):]
}

func extractMarkedSection(content, startMarker, endMarker string) (string, bool) {
	start := strings.Index(content, startMarker)
	if start == -1 {
		return "", false
	}

	endOffset := strings.Index(content[start+len(startMarker):], endMarker)
	if endOffset == -1 {
		return "", false
	}
	end := start + len(startMarker) + endOffset

	return content[start : end+len(endMarker)], true
}

func ensureRollingProtocolBacklinks(meetings []*protocols.MeetingContent, protocolPath, seriesName string) error {
	for _, meeting := range meetings {
		if meeting == nil || meeting.FilePath == "" {
			continue
		}
		if err := ensureMeetingRollingProtocolBacklink(meeting.FilePath, protocolPath, seriesName); err != nil {
			return err
		}
	}
	return nil
}

func ensureMeetingRollingProtocolBacklink(meetingPath, protocolPath, seriesName string) error {
	if meetingPath == "" || protocolPath == "" || meetingPath == protocolPath {
		return nil
	}

	data, err := os.ReadFile(meetingPath)
	if err != nil {
		return fmt.Errorf("failed to read meeting for protocol backlink: %w", err)
	}

	relPath, err := filepath.Rel(filepath.Dir(meetingPath), protocolPath)
	if err != nil {
		return fmt.Errorf("failed to compute protocol backlink for %s: %w", meetingPath, err)
	}

	backlink := fmt.Sprintf("- [Rolling Protocol: %s](%s)", seriesName, filepath.ToSlash(relPath))
	updated := addRelatedSectionBullet(string(data), backlink)
	if updated == string(data) {
		return nil
	}

	if err := os.WriteFile(meetingPath, []byte(updated), 0644); err != nil {
		return fmt.Errorf("failed to update meeting backlink: %w", err)
	}

	return nil
}

func addRelatedSectionBullet(content, bullet string) string {
	if strings.Contains(content, bullet) {
		return content
	}

	lines := strings.Split(content, "\n")
	sectionIndex, heading := findRelatedSection(lines)
	if sectionIndex == -1 {
		heading = detectRelatedHeading(content)
		trimmed := strings.TrimRight(content, "\n")
		return trimmed + "\n\n" + heading + "\n" + bullet + "\n"
	}

	end := len(lines)
	for i := sectionIndex + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if heading == "### Related" {
			if strings.HasPrefix(trimmed, "### ") || strings.HasPrefix(trimmed, "## ") {
				end = i
				break
			}
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			end = i
			break
		}
	}

	insertAt := end
	for insertAt > sectionIndex+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
		insertAt--
	}

	updated := append([]string{}, lines[:insertAt]...)
	updated = append(updated, bullet)
	updated = append(updated, lines[insertAt:]...)
	return strings.Join(updated, "\n")
}

func findRelatedSection(lines []string) (int, string) {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case "## Related":
			return i, trimmed
		case "### Related":
			return i, trimmed
		}
	}
	return -1, ""
}

func detectRelatedHeading(content string) string {
	if strings.Contains(content, "\n### Participants") || strings.Contains(content, "\n### Notes") || strings.Contains(content, "\n### Action Items") {
		return "### Related"
	}
	return "## Related"
}
