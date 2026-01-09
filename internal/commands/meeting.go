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
	var jsonOutput bool
	// Non-interactive options
	var title string
	var participants string
	var organization string
	var project string
	var context string
	var tags string
	var duration string
	var seriesName string
	var brainName string
	var noEdit bool
	var noLink bool

	cmd := &cobra.Command{
		Use:     "meeting-note [new]",
		Aliases: []string{"meeting"}, // Backward compatibility
		Short:   "Create a meeting note",
		Long:    "Create a meeting note with date, participants, agenda, and action items.\n\nExamples:\n  flip meeting-note       # Create new meeting note (default action)\n  flip meeting-note new   # Create new meeting note (explicit)\n  flip meeting-note n     # Create new meeting note (shortcut)\n  flip new meeting-note   # Create new meeting note (alternative syntax)",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			// Non-interactive mode if any field is provided or JSON requested
			if JSONOutput || title != "" || participants != "" || organization != "" || project != "" || context != "" || tags != "" || duration != "" || seriesName != "" || brainName != "" || noEdit || noLink {
				return runCreateMeetingNonInteractive(meetingCreateOptions{
					Title:        title,
					Participants: participants,
					Organization: organization,
					Project:      project,
					Context:      context,
					Tags:         tags,
					Duration:     duration,
					Series:       seriesName,
					Brain:        brainName,
					NoEdit:       noEdit,
					NoLink:       noLink,
				})
			}
			// Default action: interactive
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

	// Non-interactive flags (for VS Code integration)
	cmd.Flags().StringVar(&title, "title", "", "Meeting title")
	cmd.Flags().StringVar(&participants, "participants", "", "Comma-separated participants")
	cmd.Flags().StringVar(&organization, "organization", "", "Organization")
	cmd.Flags().StringVar(&project, "project", "", "Project")
	cmd.Flags().StringVar(&context, "context", "", "Context tag")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&duration, "duration", "", "Duration (e.g., 60 min)")
	cmd.Flags().StringVar(&seriesName, "series", "", "Series name (for recurring meetings)")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain to use (default: active brain)")
	cmd.Flags().BoolVar(&noEdit, "no-edit", false, "Don't open editor after creation")
	cmd.Flags().BoolVar(&noLink, "no-link", false, "Don't add link to journal")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")

	return cmd
}

// meetingCreateOptions holds non-interactive creation inputs
type meetingCreateOptions struct {
	Title        string
	Participants string
	Organization string
	Project      string
	Context      string
	Tags         string
	Duration     string
	Series       string
	Brain        string
	NoEdit       bool
	NoLink       bool
}

// runCreateMeetingNonInteractive creates meeting note without prompts (for VS Code integration)
func runCreateMeetingNonInteractive(opts meetingCreateOptions) error {
	if strings.TrimSpace(opts.Title) == "" {
		err := fmt.Errorf("--title is required for non-interactive mode")
		if JSONOutput { OutputJSONError("meeting-note", err); return nil }
		return err
	}

	activeWs, err := getActiveWorkspace()
	if err != nil { if JSONOutput { OutputJSONError("meeting-note", err); return nil }; return err }

	// Resolve brain
	var activeBrain *Brain
	if opts.Brain != "" {
		for i := range activeWs.Brains { if activeWs.Brains[i].Name == opts.Brain { activeBrain = &activeWs.Brains[i]; break } }
		if activeBrain == nil { err := fmt.Errorf("brain not found: %s", opts.Brain); if JSONOutput { OutputJSONError("meeting-note", err); return nil }; return err }
	} else {
		for i := range activeWs.Brains { if activeWs.Brains[i].Name == activeWs.DefaultBrain { activeBrain = &activeWs.Brains[i]; break } }
		if activeBrain == nil && len(activeWs.Brains) > 0 { activeBrain = &activeWs.Brains[0] }
	}
	if activeBrain == nil { err := fmt.Errorf("no brain available"); if JSONOutput { OutputJSONError("meeting-note", err); return nil }; return err }

	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(activeBrain.Path)
	if err != nil { if JSONOutput { OutputJSONError("meeting-note", err); return nil }; return err }

	// Filename and directory
	filename := generateMeetingFilename(opts.Title, opts.Series, detection.Type, activeBrain.Path)
	baseDir := getMeetingsDirectory(activeBrain.Path, detection.Type)
	if err := os.MkdirAll(baseDir, 0755); err != nil { if JSONOutput { OutputJSONError("meeting-note", err); return nil }; return err }
	// Use baseDir for non-interactive to keep simple
	filePath := filepath.Join(baseDir, filename)
	if _, err := os.Stat(filePath); err == nil { err := fmt.Errorf("file already exists: %s", filePath); if JSONOutput { OutputJSONError("meeting-note", err); return nil }; return err }

	// Content
	participantsList := strings.TrimSpace(opts.Participants)
	content := generateMeetingContent(opts.Title, participantsList, opts.Organization, opts.Project, opts.Context, opts.Tags, coalesce(opts.Duration, "60 min"), opts.Series, detection.Type, activeBrain.Path)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil { if JSONOutput { OutputJSONError("meeting-note", err); return nil }; return err }

	// Auto-commit
	_ = autoCommitFile(activeBrain.Path, filePath, "meeting note")

	// Optional link to journal
	if !opts.NoLink {
		rel := relativePathFromBrain(filePath, activeBrain.Path)
		_ = AddLinkToJournal(JournalLinkOptions{ ItemType: "meeting", ItemName: opts.Title, ItemPath: rel, Brain: activeBrain, Interactive: false })
	}

	if JSONOutput {
		OutputJSONSuccess("meeting-note", NoteResult{ // reuse NoteResult layout for path/title
			Action:    "created",
			Path:      filePath,
			Title:     opts.Title,
			BrainName: activeBrain.Name,
			BrainPath: activeBrain.Path,
			BrainType: string(detection.Type),
		})
		return nil
	}

	fmt.Printf("✓ Meeting note created: %s\n", filePath)
	if !opts.NoEdit { _ = openInEditor(filePath) }
	return nil
}

func coalesce(a, b string) string { if strings.TrimSpace(a) != "" { return a } ; return b }

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

	// Step 1: Prompt for organization (optional)
	promptOrg := promptui.Prompt{
		Label:   "Organization (optional, press Enter to skip)",
		Default: "",
	}

	var organization string
	organization, _ = promptOrg.Run()
	organization = strings.TrimSpace(organization)
	if organization != "" {
		fmt.Println()
	}

	// Step 2: Ask if part of a series
	promptIsSeries := promptui.Select{
		Label: "Part of a meeting series?",
		Items: []string{"No, single meeting", "Yes, add to series"},
	}

	_, isSeries, err := promptIsSeries.Run()
	if err != nil {
		return fmt.Errorf("series selection cancelled: %w", err)
	}

	var seriesName string
	var title string
	var participants string
	var project string
	var context string
	var tags string

	if isSeries == "Yes, add to series" {
		// Step 2a: New or existing series?
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
					// Copy metadata from series
					participants = seriesMeta.Participants
					// Use organization from series if not already set
					if organization == "" {
						organization = seriesMeta.Organization
					}
					project = seriesMeta.Project
					context = seriesMeta.Context
					tags = seriesMeta.Tags

					fmt.Printf("\n✅ Loaded metadata from series: %s\n", seriesName)
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
		}

		// Auto-generate title from series name with today's date
		now := time.Now()
		title = fmt.Sprintf("%s - %s", seriesName, now.Format("02.01.2006"))
	} else {
		// Single meeting: ask for title
		promptTitle := promptui.Prompt{
			Label: "Meeting title",
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

	// Prompt for duration
	promptDuration := promptui.Select{
		Label: "Meeting duration",
		Items: []string{"30 min", "45 min", "60 min", "90 min", "120 min", "Custom..."},
	}

	_, duration, err := promptDuration.Run()
	if err != nil {
		duration = "60 min" // Default if cancelled
	}

	if duration == "Custom..." {
		promptCustomDuration := promptui.Prompt{
			Label:   "Duration (e.g., 45 min, 2h)",
			Default: "60 min",
		}
		duration, err = promptCustomDuration.Run()
		if err != nil {
			duration = "60 min"
		}
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
	content := generateMeetingContent(title, participants, organization, project, context, tags, duration, seriesName, detection.Type, activeBrain.Path)

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

	// Ask if user wants to add link to journal
	relPath := relativePathFromBrain(filePath, activeBrain.Path)
	if err := AddLinkToJournal(JournalLinkOptions{
		ItemType:    "meeting",
		ItemName:    title,
		ItemPath:    relPath,
		Brain:       activeBrain,
		Interactive: true,
	}); err != nil {
		fmt.Printf("⚠️  Could not add journal link: %v\n", err)
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
func generateMeetingContent(title, participants, organization, project, context, tags, duration, seriesName string, brainType brain.BrainType, brainPath string) string {
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
		return generateDefaultMeetingContent(displayTitle, participantList, organization, project, context, tagList, duration, seriesName, meetingType, brainType, now, dateStr, timeStr, author)
	}

	// Prepare template variables
	vars := map[string]string{
		"title":        displayTitle,
		"date":         dateStr,
		"time":         timeStr,
		"duration":     duration,
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
func generateDefaultMeetingContent(title, participantList, organization, project, context, tagList, duration, seriesName, meetingType string, brainType brain.BrainType, now time.Time, dateStr, timeStr, author string) string {
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
- duration:: %s
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

`, title, meetingType, dateStr, timeStr, duration, author, tagList, orgField, projField, ctxField, seriesField, title, participantList)

	case brain.BrainTypeObsidian:
		return fmt.Sprintf(`---
title: %s
type: %s
date: %s
time: %s
duration: %s
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

`, title, meetingType, dateStr, timeStr, duration, author, tagList, frontmatterOrg, frontmatterProj, frontmatterCtx, frontmatterSeries, title, participantList)

	case brain.BrainTypeDendron:
		return fmt.Sprintf(`---
id: %s
title: %s
desc: 'Meeting note'
type: %s
date: %s
duration: %s
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

`, uuid.New().String(), title, meetingType, dateStr, duration, author, now.Unix(), now.Unix(), tagList, frontmatterOrg, frontmatterProj, frontmatterCtx, frontmatterSeries, title, participantList)

	case brain.BrainTypeFlip:
		return fmt.Sprintf(`---
title: %s
created: %s
updated: %s
type: %s
date: %s
time: %s
duration: %s
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

`, title, dateStr, dateStr, meetingType, dateStr, timeStr, duration, author, tagList, frontmatterOrg, frontmatterProj, frontmatterCtx, frontmatterSeries, title, participantList)

	default:
		return fmt.Sprintf(`---
title: %s
type: %s
date: %s
time: %s
duration: %s
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

`, title, meetingType, dateStr, timeStr, duration, tagList, frontmatterOrg, frontmatterProj, frontmatterCtx, frontmatterSeries, title, participantList)
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
