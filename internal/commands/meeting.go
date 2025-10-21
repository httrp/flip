package commands

import (
	"fmt"
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
		Use:   "meeting",
		Short: "Create a meeting note",
		Long:  "Create a meeting note with date, participants, agenda, and action items",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateMeeting()
		},
	}
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

	// Prompt for meeting title
	promptTitle := promptui.Prompt{
		Label: "Meeting title",
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("title cannot be empty")
			}
			return nil
		},
	}

	title, err := promptTitle.Run()
	if err != nil {
		return fmt.Errorf("title prompt cancelled: %w", err)
	}
	title = strings.TrimSpace(title)

	// Prompt for participants
	promptParticipants := promptui.Prompt{
		Label:   "Participants (comma-separated, optional)",
		Default: "",
	}

	participants, err := promptParticipants.Run()
	if err != nil {
		return fmt.Errorf("participants prompt cancelled: %w", err)
	}
	participants = strings.TrimSpace(participants)

	// Prompt for tags
	promptTags := promptui.Prompt{
		Label:   "Tags (comma-separated, optional)",
		Default: "meeting",
	}

	tags, err := promptTags.Run()
	if err != nil {
		return fmt.Errorf("tags prompt cancelled: %w", err)
	}
	tags = strings.TrimSpace(tags)

	// Generate filename
	filename := generateMeetingFilename(title, detection.Type)

	// Determine base notes directory
	baseDir := getNotesDirectory(activeBrain.Path, detection.Type)
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
	content := generateMeetingContent(title, participants, tags, detection.Type, activeBrain.Path)

	// Write file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create meeting note: %w", err)
	}

	fmt.Printf("✓ Meeting note created successfully!\n")
	fmt.Printf("  Path: %s\n", filePath)
	fmt.Printf("  Brain: %s (%s)\n\n", activeBrain.Name, detection.Type)

	// Ask if user wants to edit
	if err := promptAndOpenEditor(filePath); err != nil {
		fmt.Printf("⚠️  Could not open editor: %v\n", err)
	}

	return nil
}

// generateMeetingFilename creates a filename for meeting notes
func generateMeetingFilename(title string, brainType brain.BrainType) string {
	safeName := sanitizeFilename(title)
	now := time.Now()

	switch brainType {
	case brain.BrainTypeLogseq:
		return fmt.Sprintf("%s___meeting-%s.md", now.Format("2006_01_02"), safeName)
	case brain.BrainTypeObsidian:
		return fmt.Sprintf("meeting-%s-%s.md", now.Format("2006-01-02"), safeName)
	case brain.BrainTypeDendron:
		return fmt.Sprintf("meetings.%s-%s.md", now.Format("2006-01-02"), safeName)
	case brain.BrainTypeFlip:
		return fmt.Sprintf("%s-meeting-%s.md", now.Format("2006-01-02"), safeName)
	default:
		return fmt.Sprintf("%s-meeting-%s.md", now.Format("2006-01-02"), safeName)
	}
}

// generateMeetingContent creates meeting note content
func generateMeetingContent(title, participants, tags string, brainType brain.BrainType, brainPath string) string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04")

	// Get author from brain config
	author := getBrainAuthor(brainPath)
	if author == "" {
		author = "Unknown"
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
		return generateDefaultMeetingContent(title, participantList, tagList, brainType, now, dateStr, timeStr, author)
	}

	// Prepare template variables
	vars := map[string]string{
		"title":        title,
		"date":         dateStr,
		"time":         timeStr,
		"participants": participantList,
		"tags":         tagList,
		"author":       author,
		"id":           uuid.New().String(), // Proper UUID for Dendron compatibility
		"updated":      fmt.Sprintf("%d", now.Unix()),
		"created":      fmt.Sprintf("%d", now.Unix()),
	}

	return templates.Render(tmpl, vars)
}

// generateDefaultMeetingContent provides fallback templates when template files don't exist
func generateDefaultMeetingContent(title, participantList, tagList string, brainType brain.BrainType, now time.Time, dateStr, timeStr, author string) string {
	switch brainType {
	case brain.BrainTypeLogseq:
		return fmt.Sprintf(`- title:: %s
- type:: meeting
- date:: %s
- time:: %s
- author:: %s
- tags:: %s

## %s

### Participants
%s
### Agenda

### Notes

### Action Items
- TODO Action item [priority: high] [deadline: YYYY-MM-DD] [assignee: name] [status: open]

### Related
- [[related-note]]

`, title, dateStr, timeStr, author, tagList, title, participantList)

	case brain.BrainTypeObsidian:
		return fmt.Sprintf(`---
title: %s
type: meeting
date: %s
time: %s
author: %s
tags: [%s]
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

`, title, dateStr, timeStr, author, tagList, title, participantList)

	case brain.BrainTypeDendron:
		return fmt.Sprintf(`---
id: %s
title: %s
desc: 'Meeting note'
type: meeting
date: %s
author: %s
updated: %d
created: %d
tags: [%s]
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

`, uuid.New().String(), title, dateStr, author, now.Unix(), now.Unix(), tagList, title, participantList)

	case brain.BrainTypeFlip:
		return fmt.Sprintf(`---
title: %s
created: %s
updated: %s
type: meeting
date: %s
time: %s
author: %s
tags: [%s]
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

`, title, dateStr, dateStr, dateStr, timeStr, author, tagList, title, participantList)

	default:
		return fmt.Sprintf(`---
title: %s
type: meeting
date: %s
tags: [%s]
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

`, title, dateStr, tagList, title, participantList)
	}
}
