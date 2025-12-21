package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/templates"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func NewMeetingCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "meeting-note [new]",
		Aliases: []string{"meeting"}, // Backward compatibility
		Short:   "Create a meeting note",
		Long:    "Create a meeting note with date, participants, agenda, and action items.\n\nExamples:\n  flip meeting-note       # Create new meeting note (default action)\n  flip meeting-note new   # Create new meeting note (explicit)\n  flip meeting-note n     # Create new meeting note (shortcut)\n  flip new meeting-note   # Create new meeting note (alternative syntax)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action: create new meeting note
			// Support: flip meeting-note, flip meeting-note new, flip meeting-note n
			if len(args) == 0 || args[0] == "new" || args[0] == "n" {
				return runCreateMeeting()
			}
			return fmt.Errorf("unknown subcommand: %s", args[0])
		},
	}

	// Add explicit 'new' subcommand for clarity
	newCmd := &cobra.Command{
		Use:     "new",
		Aliases: []string{"n"},
		Short:   "Create a new meeting note (explicit)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateMeeting()
		},
	}
	cmd.AddCommand(newCmd)

	return cmd
}

// runCreateMeeting creates a new meeting note
func runCreateMeeting() error {
	fmt.Println("\n📅 Create Meeting Note")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Get active workspace
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get active workspace: %w", err)
	}

	if len(activeWs.Brains) == 0 {
		fmt.Println("⚠️  No brains found in active workspace. Please add a brain first.")
		return nil
	}

	// Select brain
	activeBrain, err := confirmOrSelectBrain(activeWs)
	if err != nil {
		return err
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(activeBrain.Path)
	if err != nil {
		return fmt.Errorf("failed to detect brain type: %w", err)
	}

	fmt.Printf("Brain type: %s\n\n", detection.Type)

	// Step 1: Ask if single meeting or series
	promptMeetingType := promptui.Select{
		Label: "Meeting type",
		Items: []string{"Single meeting", "Part of a series"},
	}

	_, meetingType, err := promptMeetingType.Run()
	if err != nil {
		return fmt.Errorf("meeting type selection cancelled: %w", err)
	}

	var seriesName string
	var title string
	var participants string
	var organization string
	var project string
	var context string
	var tags string

	if meetingType == "Part of a series" {
		// Step 2: New or existing series?
		promptSeriesChoice := promptui.Select{
			Label: "Series",
			Items: []string{"Create new series", "Add to existing series"},
		}

		_, seriesChoice, err := promptSeriesChoice.Run()
		if err != nil {
			return fmt.Errorf("series choice cancelled: %w", err)
		}

		if seriesChoice == "Add to existing series" {
			// Find existing series
			series, err := findMeetingSeries(activeBrain.Path, detection.Type)
			if err != nil {
				return fmt.Errorf("failed to find series: %w", err)
			}

			if len(series) == 0 {
				fmt.Println("\n⚠️  No existing series found. Creating new series instead.")
				seriesChoice = "Create new series"
			} else {
				// Select from existing series
				seriesNames := make([]string, len(series))
				for i, s := range series {
					seriesNames[i] = fmt.Sprintf("%s (%d meetings)", s.Name, s.Count)
				}

				promptSelectSeries := promptui.Select{
					Label: "Select series",
					Items: seriesNames,
					Size:  10,
				}

				idx, _, err := promptSelectSeries.Run()
				if err != nil {
					return fmt.Errorf("series selection cancelled: %w", err)
				}

				selectedSeries := series[idx]
				seriesName = selectedSeries.Name

				// Load metadata from most recent meeting in series
				seriesMeta, err := loadSeriesMetadata(selectedSeries.LatestFile)
				if err != nil {
					fmt.Printf("⚠️  Could not load series metadata: %v\n", err)
				} else {
					// Copy metadata from series (but use series name as title, not the dated title)
					title = seriesName // Use series name directly, not the dated title from metadata
					participants = seriesMeta.Participants
					organization = seriesMeta.Organization
					project = seriesMeta.Project
					context = seriesMeta.Context
					tags = seriesMeta.Tags

					fmt.Printf("\n✅ Loaded metadata from series: %s\n", seriesName)
					fmt.Printf("   Title: %s\n", seriesMeta.Title) // Show the old title for reference
					if organization != "" {
						fmt.Printf("   Organization: %s\n", organization)
					}
					if project != "" {
						fmt.Printf("   Project: %s\n", project)
					}
					if context != "" {
						fmt.Printf("   Context: %s\n", context)
					}
					fmt.Println()
				}
			}
		}

		if seriesChoice == "Create new series" {
			// Prompt for series name
			promptSeriesName := promptui.Prompt{
				Label: "Series name (e.g., 'Weekly Standup', 'Sprint Planning')",
				Validate: func(input string) error {
					if strings.TrimSpace(input) == "" {
						return fmt.Errorf("series name cannot be empty")
					}
					return nil
				},
			}

			seriesName, err = promptSeriesName.Run()
			if err != nil {
				return fmt.Errorf("series name prompt cancelled: %w", err)
			}
			seriesName = strings.TrimSpace(seriesName)
			// For series: series name IS the title
			title = seriesName
		}
	}

	// Prompt for meeting title (only for single meetings, not for series)
	if seriesName == "" {
		promptTitle := promptui.Prompt{
			Label:   "Meeting title",
			Default: title,
			Validate: func(input string) error {
				if strings.TrimSpace(input) == "" {
					return fmt.Errorf("title cannot be empty")
				}
				return nil
			},
		}

		title, err = promptTitle.Run()
		if err != nil {
			return fmt.Errorf("title prompt cancelled: %w", err)
		}
		title = strings.TrimSpace(title)
	}

	// Prompt for participants - optional, can be filled in later (use Default if from series)
	promptParticipants := promptui.Prompt{
		Label:   "Participants (optional, press Enter to skip)",
		Default: participants,
	}

	participants, err = promptParticipants.Run()
	if err != nil {
		// If cancelled, keep existing value
	} else {
		participants = strings.TrimSpace(participants)
	}

	// Prompt for organization
	promptOrganization := promptui.Prompt{
		Label:   "Organization (e.g., P1174, DANORAMA, press Enter to skip)",
		Default: organization,
	}

	organization, err = promptOrganization.Run()
	if err != nil {
		// If cancelled, keep existing value
	} else {
		organization = strings.TrimSpace(organization)
	}

	// Prompt for project
	promptProject := promptui.Prompt{
		Label:   "Project (press Enter to skip)",
		Default: project,
	}

	project, err = promptProject.Run()
	if err != nil {
		// If cancelled, keep existing value
	} else {
		project = strings.TrimSpace(project)
	}

	// Prompt for context
	promptContext := promptui.Prompt{
		Label:   "Context (e.g., BACKEND, FINANCE, press Enter to skip)",
		Default: context,
	}

	context, err = promptContext.Run()
	if err != nil {
		// If cancelled, keep existing value
	} else {
		context = strings.TrimSpace(context)
	}

	// Prompt for tags
	promptTags := promptui.Prompt{
		Label:   "Tags (comma-separated, optional)",
		Default: tags,
	}

	tags, err = promptTags.Run()
	if err != nil {
		if tags == "" {
			tags = "meeting" // Keep default if cancelled and no value yet
		}
	} else {
		tags = strings.TrimSpace(tags)
	}

	// Generate filename
	filename := generateMeetingFilename(title, seriesName, detection.Type, activeBrain.Path)

	// Determine meetings directory (dedicated directory for meeting notes)
	baseDir := getMeetingsDirectory(activeBrain.Path, detection.Type)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prompt for subfolder
	targetDir, err := promptForSubfolder(baseDir, detection.Type)
	if err != nil {
		return err
	}

	// Full file path
	filePath := filepath.Join(targetDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		fmt.Printf("\n⚠️  File already exists: %s\n", filePath)
		return nil
	}

	// Show preview
	fmt.Printf("\n📄 Will create: %s\n", filename)
	fmt.Printf("   Location: %s\n\n", targetDir)

	// Generate content
	content := generateMeetingContent(title, participants, organization, project, context, tags, seriesName, detection.Type, activeBrain.Path)

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create meeting note: %w", err)
	}

	fmt.Printf("✓ Meeting note created successfully!\n")
	fmt.Printf("  Path: %s\n", filePath)
	fmt.Printf("  Brain: %s (%s)\n\n", activeBrain.Name, detection.Type)

	// Optional git commit
	if err := autoCommitFile(activeBrain.Path, filePath, "meeting note"); err != nil {
		fmt.Printf("⚠️  Git commit failed: %v\n", err)
	}

	// Ask if user wants to edit
	if err := promptAndOpenEditor(filePath); err != nil {
		fmt.Printf("⚠️  Could not open editor: %v\n", err)
	}

	return nil
}

// generateMeetingFilename creates a filename for meeting notes
func generateMeetingFilename(title, seriesName string, brainType brain.BrainType, brainPath string) string {
	safeName := sanitizeFilename(title)
	now := time.Now()

	// If this is part of a series, append a counter to make the filename unique
	if seriesName != "" {
		// Get meetings directory
		meetingsPath := getMeetingsDirectory(brainPath, brainType)

		// Count existing meetings in this series
		counter := 1
		files, err := os.ReadDir(meetingsPath)
		if err == nil {
			for _, file := range files {
				if file.IsDir() {
					continue
				}

				filePath := filepath.Join(meetingsPath, file.Name())
				existingSeries, err := extractSeriesNameFromFile(filePath)
				if err == nil && existingSeries == seriesName {
					counter++
				}
			}
		}

		// Append counter to the filename
		safeName = fmt.Sprintf("%s-%02d", safeName, counter)
	}

	switch brainType {
	case brain.BrainTypeLogseq:
		return fmt.Sprintf("%s___meeting-%s.md", now.Format("2006_01_02"), safeName)
	case brain.BrainTypeObsidian:
		return fmt.Sprintf("meeting-%s-%s.md", now.Format("2006-01-02"), safeName)
	case brain.BrainTypeDendron:
		return fmt.Sprintf("meetings.%s-%s.md", now.Format("2006-01-02"), safeName)
	case brain.BrainTypeFoam:
		return fmt.Sprintf("meeting-%s-%s.md", now.Format("2006-01-02"), safeName)
	case brain.BrainTypeFlip:
		return fmt.Sprintf("%s-meeting-%s.md", now.Format("2006-01-02"), safeName)
	default:
		return fmt.Sprintf("%s-meeting-%s.md", now.Format("2006-01-02"), safeName)
	}
}

// generateMeetingContent creates meeting note content
func generateMeetingContent(title, participants, organization, project, context, tags, seriesName string, brainType brain.BrainType, brainPath string) string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04")

	// Get author from brain config
	author := getBrainAuthor(brainPath)
	if author == "" {
		author = "Unknown"
	}

	// For series meetings, append date to title for uniqueness
	displayTitle := title
	if seriesName != "" {
		displayTitle = fmt.Sprintf("%s - %s", title, dateStr)
	}

	// Determine meeting type
	meetingType := "meeting"
	if seriesName != "" {
		meetingType = "meeting-series"
	}

	// Parse participants into list
	participantList := ""
	if participants != "" {
		parts := strings.Split(participants, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				participantList += fmt.Sprintf("- %s\n", p)
			}
		}
	} else {
		// Empty placeholder that user can fill in
		participantList = "- \n"
	}

	// Parse tags
	tagList := "meeting"
	if tags != "" {
		tagList = tags
	}

	// Try to load template from file
	tmpl, err := templates.Load(brainType, templates.TemplateTypeMeeting)
	if err != nil {
		// Fallback to hardcoded template if file not found
		fmt.Printf("Warning: Could not load template, using default (%v)\n", err)
		return generateDefaultMeetingContent(displayTitle, participantList, organization, project, context, tagList, seriesName, meetingType, brainType, now, dateStr, timeStr, author)
	}

	// Prepare template variables
	vars := map[string]string{
		"title":        displayTitle,
		"date":         dateStr,
		"time":         timeStr,
		"participants": participantList,
		"organization": organization,
		"project":      project,
		"contextTag":   context,
		"series":       seriesName,
		"type":         meetingType,
		"tags":         tagList,
		"author":       author,
		"id":           uuid.New().String(), // Proper UUID for Dendron compatibility
		"updated":      fmt.Sprintf("%d", now.Unix()),
		"created":      fmt.Sprintf("%d", now.Unix()),
	}

	return templates.Render(tmpl, vars)
}

// generateDefaultMeetingContent provides fallback templates when template files don't exist
func generateDefaultMeetingContent(title, participantList, organization, project, context, tagList, seriesName, meetingType string, brainType brain.BrainType, now time.Time, dateStr, timeStr, author string) string {
	// Build frontmatter fields
	frontmatterOrg := ""
	if organization != "" {
		frontmatterOrg = fmt.Sprintf("\norganization: %s", organization)
	}
	frontmatterProj := ""
	if project != "" {
		frontmatterProj = fmt.Sprintf("\nproject: %s", project)
	}
	frontmatterCtx := ""
	if context != "" {
		frontmatterCtx = fmt.Sprintf("\ncontext: %s", context)
	}
	frontmatterSeries := ""
	if seriesName != "" {
		frontmatterSeries = fmt.Sprintf("\nseries: %s", seriesName)
	}

	switch brainType {
	case brain.BrainTypeLogseq:
		orgField := ""
		if organization != "" {
			orgField = fmt.Sprintf("- organization:: %s\n", organization)
		}
		projField := ""
		if project != "" {
			projField = fmt.Sprintf("- project:: %s\n", project)
		}
		ctxField := ""
		if context != "" {
			ctxField = fmt.Sprintf("- context:: %s\n", context)
		}
		seriesField := ""
		if seriesName != "" {
			seriesField = fmt.Sprintf("- series:: %s\n", seriesName)
		}

		return fmt.Sprintf(`- title:: %s
- type:: %s
- date:: %s
- time:: %s
- author:: %s
- tags:: %s
%s%s%s%s
## %s

### Participants
%s
### Agenda

### Notes

### Action Items
- TODO Action item [priority: high] [deadline: YYYY-MM-DD] [assignee: name] [status: open]

### Related
- [[related-note]]

`, title, meetingType, dateStr, timeStr, author, tagList, orgField, projField, ctxField, seriesField, title, participantList)

	case brain.BrainTypeObsidian:
		return fmt.Sprintf(`---
title: %s
type: %s
date: %s
time: %s
author: %s
tags: [%s]%s%s%s%s
---

# %s

## Participants
%s
## Agenda

## Notes

## Action Items
- [ ] Action item - Priority: High, Deadline: YYYY-MM-DD, Assignee: name, Status: Open

## Related
- [[related-note]]

`, title, meetingType, dateStr, timeStr, author, tagList, frontmatterOrg, frontmatterProj, frontmatterCtx, frontmatterSeries, title, participantList)

	case brain.BrainTypeDendron:
		return fmt.Sprintf(`---
id: %s
title: %s
desc: 'Meeting note'
type: %s
date: %s
author: %s
updated: %d
created: %d
tags: [%s]%s%s%s%s
---

# %s

## Participants
%s
## Agenda

## Notes

## Action Items
- [ ] Action item - Priority: High, Deadline: YYYY-MM-DD, Assignee: name, Status: Open

## Related
- [[related-note]]

`, uuid.New().String(), title, meetingType, dateStr, author, now.Unix(), now.Unix(), tagList, frontmatterOrg, frontmatterProj, frontmatterCtx, frontmatterSeries, title, participantList)

	case brain.BrainTypeFlip:
		return fmt.Sprintf(`---
title: %s
created: %s
updated: %s
type: %s
date: %s
time: %s
author: %s
tags: [%s]%s%s%s%s
---

# %s

## Participants
%s
## Agenda

## Notes

## Action Items
- [ ] Action item - Priority: High, Deadline: YYYY-MM-DD, Assignee: name, Status: Open

## Related
- [[related-note]]

`, title, dateStr, dateStr, meetingType, dateStr, timeStr, author, tagList, frontmatterOrg, frontmatterProj, frontmatterCtx, frontmatterSeries, title, participantList)

	default:
		return fmt.Sprintf(`---
title: %s
type: %s
date: %s
tags: [%s]%s%s%s%s
---

# %s

## Participants
%s
## Agenda

## Notes

## Action Items
- [ ] Action item - Priority: High, Deadline: YYYY-MM-DD, Assignee: name, Status: Open

## Related
- [[related-note]]

`, title, meetingType, dateStr, tagList, frontmatterOrg, frontmatterProj, frontmatterCtx, frontmatterSeries, title, participantList)
	}
}

// MeetingSeries represents a meeting series
type MeetingSeries struct {
	Name       string
	Count      int
	LatestFile string
}

// SeriesMetadata holds metadata from a series meeting
type SeriesMetadata struct {
	Title        string
	Participants string
	Organization string
	Project      string
	Context      string
	Tags         string
}

// findMeetingSeries scans for existing meeting series
func findMeetingSeries(brainPath string, brainType brain.BrainType) ([]MeetingSeries, error) {
	seriesMap := make(map[string]*MeetingSeries)

	// Get meetings directory
	meetingsDir := getMeetingsDirectory(brainPath, brainType)

	// If directory doesn't exist, return empty list
	if _, err := os.Stat(meetingsDir); os.IsNotExist(err) {
		return []MeetingSeries{}, nil
	}

	// Walk through all meetings to find those with series field
	err := filepath.WalkDir(meetingsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Only process markdown files
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		// Try to extract series name from frontmatter
		seriesName, err := extractSeriesNameFromFile(path)
		if err != nil || seriesName == "" {
			return nil
		}

		// Get file info for mod time
		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Add or update series
		if series, exists := seriesMap[seriesName]; exists {
			series.Count++
			// Update latest file if this one is newer
			if info.ModTime().After(getFileModTime(series.LatestFile)) {
				series.LatestFile = path
			}
		} else {
			seriesMap[seriesName] = &MeetingSeries{
				Name:       seriesName,
				Count:      1,
				LatestFile: path,
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice
	result := make([]MeetingSeries, 0, len(seriesMap))
	for _, series := range seriesMap {
		result = append(result, *series)
	}

	return result, nil
}

// extractSeriesNameFromFile reads the series field from frontmatter
func extractSeriesNameFromFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	inFrontmatter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				break // End of frontmatter
			}
		}

		if !inFrontmatter {
			continue
		}

		// Look for series: field
		if strings.HasPrefix(trimmed, "series:") {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "series:"))
			return strings.Trim(value, "\"'"), nil
		}
	}

	return "", nil
}

// loadSeriesMetadata loads metadata from a series file
func loadSeriesMetadata(filePath string) (*SeriesMetadata, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	inFrontmatter := false

	meta := &SeriesMetadata{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				break // End of frontmatter
			}
		}

		if !inFrontmatter {
			continue
		}

		// Parse frontmatter fields
		if strings.HasPrefix(trimmed, "title:") {
			meta.Title = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "title:")), "\"'")
		} else if strings.HasPrefix(trimmed, "participants:") {
			meta.Participants = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "participants:")), "\"'")
		} else if strings.HasPrefix(trimmed, "organization:") {
			meta.Organization = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "organization:")), "\"'")
		} else if strings.HasPrefix(trimmed, "project:") {
			meta.Project = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "project:")), "\"'")
		} else if strings.HasPrefix(trimmed, "context:") {
			meta.Context = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "context:")), "\"'")
		} else if strings.HasPrefix(trimmed, "tags:") {
			meta.Tags = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "tags:")), "\"'[]")
		}
	}

	return meta, nil
}

// getFileModTime returns the modification time of a file
func getFileModTime(filePath string) time.Time {
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
