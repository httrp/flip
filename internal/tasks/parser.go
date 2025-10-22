package tasks

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// Regex to match task checkbox lines: - [ ] or - [x] etc.
	checkboxPattern = regexp.MustCompile(`^(\s*)- \[([ xX\/~>\-?])\] (.+)$`)

	// Regex to match metadata lines: key:: value
	metadataPattern = regexp.MustCompile(`^\s+(\w+)::\s*(.+)$`)

	// Regex to match inline date: 📅 YYYY-MM-DD
	inlineDatePattern = regexp.MustCompile(`📅\s*(\d{4}-\d{2}-\d{2})`)

	// Regex to match inline priority: ⏫ 🔼 🔽
	inlinePriorityPattern = regexp.MustCompile(`(⏫|🔼|🔽)`)

	// Regex to match tags: #tag
	tagPattern = regexp.MustCompile(`#([\w\-/]+)`)

	// Regex to match wiki links: [[Link]]
	wikiLinkPattern = regexp.MustCompile(`\[\[([^\]]+)\]\]`)
)

// ParseTask extracts a task from a markdown line
func ParseTask(line string, lineNum int, context TaskContext) (*Task, error) {
	matches := checkboxPattern.FindStringSubmatch(line)
	if matches == nil {
		return nil, nil // Not a task line
	}

	// Extract parts
	statusMarker := matches[2]
	rawDescription := matches[3]

	// Parse status
	status := StatusFromCheckbox(statusMarker)

	// Parse inline metadata and clean description
	description, metadata := parseInlineMetadata(rawDescription)

	// Create task
	task := &Task{
		Description: description,
		Status:      status,
		Created:     time.Now(), // Will be overwritten if metadata exists
		Context:     context,
		Tags:        []string{},
	}

	// Apply inline metadata
	applyInlineMetadata(task, metadata)

	// Generate ID based on context
	task.ID = GenerateID(description, context, lineNum)

	return task, nil
}

// parseInlineMetadata extracts emoji indicators and tags from description
func parseInlineMetadata(text string) (cleanDescription string, metadata map[string]string) {
	metadata = make(map[string]string)
	cleanText := text

	// Extract due date (📅 2025-10-30)
	if matches := inlineDatePattern.FindStringSubmatch(text); matches != nil {
		metadata["due"] = matches[1]
		cleanText = strings.Replace(cleanText, matches[0], "", 1)
	}

	// Extract priority (⏫ 🔼 🔽)
	if matches := inlinePriorityPattern.FindStringSubmatch(text); matches != nil {
		metadata["priority"] = matches[1]
		cleanText = strings.Replace(cleanText, matches[0], "", 1)
	}

	// Extract tags (#tag)
	tags := tagPattern.FindAllStringSubmatch(text, -1)
	if len(tags) > 0 {
		var tagList []string
		for _, tag := range tags {
			tagList = append(tagList, tag[1])
			cleanText = strings.Replace(cleanText, tag[0], "", 1)
		}
		metadata["tags"] = strings.Join(tagList, ",")
	}

	// Extract wiki links ([[Project]])
	links := wikiLinkPattern.FindAllStringSubmatch(text, -1)
	if len(links) > 0 {
		// First link could be project
		metadata["project"] = links[0][1]
		// Don't remove from description - links are part of the description
	}

	// Clean up extra whitespace
	cleanDescription = strings.TrimSpace(cleanText)
	cleanDescription = regexp.MustCompile(`\s+`).ReplaceAllString(cleanDescription, " ")

	return cleanDescription, metadata
}

// applyInlineMetadata applies extracted metadata to a task
func applyInlineMetadata(task *Task, metadata map[string]string) {
	// Apply due date
	if dueStr, ok := metadata["due"]; ok {
		if dueDate, err := time.Parse("2006-01-02", dueStr); err == nil {
			task.Due = &dueDate
		}
	}

	// Apply priority
	if priorityIcon, ok := metadata["priority"]; ok {
		task.Priority = PriorityFromIcon(priorityIcon)
	}

	// Apply tags
	if tagsStr, ok := metadata["tags"]; ok {
		task.Tags = strings.Split(tagsStr, ",")
		// Clean tags
		for i, tag := range task.Tags {
			task.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// Apply project
	if project, ok := metadata["project"]; ok {
		task.Project = project
	}
}

// ParseTaskMetadata reads indented metadata lines below a task
func ParseTaskMetadata(task *Task, lines []string) error {
	for _, line := range lines {
		matches := metadataPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		key := strings.ToLower(matches[1])
		value := strings.TrimSpace(matches[2])

		switch key {
		case "created":
			if t, err := parseDateTime(value); err == nil {
				task.Created = t
			}
		case "started":
			if t, err := parseDateTime(value); err == nil {
				task.Started = &t
			}
		case "due":
			if t, err := parseDate(value); err == nil {
				task.Due = &t
			}
		case "completed":
			if t, err := parseDateTime(value); err == nil {
				task.Completed = &t
			}
		case "priority":
			task.Priority = Priority(value)
		case "effort":
			task.Effort = value
		case "tags":
			tags := strings.Split(value, ",")
			for _, tag := range tags {
				tag = strings.TrimSpace(tag)
				tag = strings.TrimPrefix(tag, "#")
				if tag != "" {
					task.Tags = append(task.Tags, tag)
				}
			}
		case "project":
			// Remove [[ ]] if present
			task.Project = strings.Trim(value, "[]")
		case "assigned":
			task.Assigned = strings.TrimPrefix(value, "@")
		case "notes":
			task.Notes = value
		}
	}

	return nil
}

// parseDate parses a date string (YYYY-MM-DD)
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// parseDateTime parses a datetime string (YYYY-MM-DD HH:MM or YYYY-MM-DD)
func parseDateTime(s string) (time.Time, error) {
	// Try with time first
	if t, err := time.Parse("2006-01-02 15:04", s); err == nil {
		return t, nil
	}
	// Try date only
	return time.Parse("2006-01-02", s)
}

// IsTaskLine checks if a line is a task checkbox
func IsTaskLine(line string) bool {
	return checkboxPattern.MatchString(line)
}

// IsMetadataLine checks if a line is task metadata
func IsMetadataLine(line string) bool {
	return metadataPattern.MatchString(line)
}

// FormatTask converts a task back to markdown format
func FormatTask(task *Task) string {
	var lines []string

	// Main task line
	checkbox := CheckboxFromStatus(task.Status)
	description := task.Description

	// Add inline metadata
	if task.Due != nil {
		description += fmt.Sprintf(" 📅 %s", task.Due.Format("2006-01-02"))
	}
	if task.Priority != PriorityNone {
		description += " " + PriorityIcon(task.Priority)
	}
	if len(task.Tags) > 0 {
		for _, tag := range task.Tags {
			description += " #" + tag
		}
	}

	lines = append(lines, fmt.Sprintf("- [%s] %s", checkbox, description))

	// Add metadata lines
	if !task.Created.IsZero() {
		lines = append(lines, fmt.Sprintf("  created:: %s", task.Created.Format("2006-01-02 15:04")))
	}
	if task.Started != nil {
		lines = append(lines, fmt.Sprintf("  started:: %s", task.Started.Format("2006-01-02 15:04")))
	}
	if task.Completed != nil {
		lines = append(lines, fmt.Sprintf("  completed:: %s", task.Completed.Format("2006-01-02 15:04")))
	}
	if task.Effort != "" {
		lines = append(lines, fmt.Sprintf("  effort:: %s", task.Effort))
	}
	if task.Project != "" {
		lines = append(lines, fmt.Sprintf("  project:: [[%s]]", task.Project))
	}
	if task.Assigned != "" {
		lines = append(lines, fmt.Sprintf("  assigned:: @%s", task.Assigned))
	}

	return strings.Join(lines, "\n")
}
