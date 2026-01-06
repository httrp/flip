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

	// Regex to match frog emoji: 🐸 (eat-the-frog)
	frogEmojiPattern = regexp.MustCompile(`🐸`)

	// Regex to match tags: #tag
	tagPattern = regexp.MustCompile(`#([\w\-/]+)`)

	// Regex to match wiki links: [[Link]]
	wikiLinkPattern = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

	// Regex to match organization/context patterns:
	// [ORG], [ORG:CTX], [ORG:PROJ:CTX]
	orgContextPattern = regexp.MustCompile(`\[([A-Z0-9\-]+)(?::([^\]:]+))?(?::([^\]]+))?\]`)

	// Regex to match key:value metadata (org:VALUE, ctx:VALUE, proj:VALUE)
	keyValuePattern = regexp.MustCompile(`\b(org|ctx|proj|context|organization|project):([A-Z0-9\-]+)\b`)
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

	// Extract organization/context patterns [ORG] or [ORG:CTX] or [ORG:PROJ:CTX]
	if matches := orgContextPattern.FindStringSubmatch(text); matches != nil {
		// matches[1] = ORG (always present)
		// matches[2] = second part (optional - could be CTX or PROJ)
		// matches[3] = third part (optional - CTX if PROJ exists)

		metadata["organization"] = matches[1]

		if matches[3] != "" {
			// Three parts: [ORG:PROJ:CTX]
			metadata["project"] = matches[2]
			metadata["context"] = matches[3]
		} else if matches[2] != "" {
			// Two parts: [ORG:CTX]
			metadata["context"] = matches[2]
		}
		// One part: [ORG] - only organization set

		cleanText = strings.Replace(cleanText, matches[0], "", 1)
	}

	// Extract key:value patterns (org:VALUE, ctx:VALUE, proj:VALUE)
	kvMatches := keyValuePattern.FindAllStringSubmatch(text, -1)
	for _, match := range kvMatches {
		key := strings.ToLower(match[1])
		value := match[2]

		switch key {
		case "org", "organization":
			metadata["organization"] = value
		case "ctx", "context":
			metadata["context"] = value
		case "proj", "project":
			metadata["project"] = value
		}

		cleanText = strings.Replace(cleanText, match[0], "", 1)
	}

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

	// Extract frog emoji (🐸 eat-the-frog)
	if frogEmojiPattern.MatchString(text) {
		metadata["frog"] = "true"
		cleanText = frogEmojiPattern.ReplaceAllString(cleanText, "")
	}

	// Extract tags (#tag) - also check for #frog tag
	tags := tagPattern.FindAllStringSubmatch(text, -1)
	if len(tags) > 0 {
		var tagList []string
		for _, tag := range tags {
			// Check for #frog tag
			if strings.ToLower(tag[1]) == "frog" {
				metadata["frog"] = "true"
			} else {
				tagList = append(tagList, tag[1])
			}
			cleanText = strings.Replace(cleanText, tag[0], "", 1)
		}
		if len(tagList) > 0 {
			metadata["tags"] = strings.Join(tagList, ",")
		}
	}

	// Extract wiki links ([[Project]])
	links := wikiLinkPattern.FindAllStringSubmatch(text, -1)
	if len(links) > 0 {
		// First link could be project (if not already set by other patterns)
		if _, exists := metadata["project"]; !exists {
			metadata["project"] = links[0][1]
		}
		// Don't remove from description - links are part of the description
	}

	// Clean up extra whitespace
	cleanDescription = strings.TrimSpace(cleanText)
	cleanDescription = regexp.MustCompile(`\s+`).ReplaceAllString(cleanDescription, " ")

	return cleanDescription, metadata
}

// applyInlineMetadata applies extracted metadata to a task
func applyInlineMetadata(task *Task, metadata map[string]string) {
	// Apply organization
	if org, ok := metadata["organization"]; ok {
		task.Organization = org
	}

	// Apply context
	if ctx, ok := metadata["context"]; ok {
		task.ContextTag = ctx
	}

	// Apply project
	if project, ok := metadata["project"]; ok {
		task.Project = project
	}

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

	// Apply frog (eat-the-frog)
	if frog, ok := metadata["frog"]; ok && frog == "true" {
		task.Frog = true
	}

	// Apply tags
	if tagsStr, ok := metadata["tags"]; ok {
		task.Tags = strings.Split(tagsStr, ",")
		// Clean tags
		for i, tag := range task.Tags {
			task.Tags[i] = strings.TrimSpace(tag)
		}
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
		case "project", "proj":
			// Remove [[ ]] if present
			task.Project = strings.Trim(value, "[]")
		case "organization", "org":
			task.Organization = value
		case "context", "ctx":
			task.ContextTag = value
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
	if task.Frog {
		description += " 🐸"
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
	if task.Organization != "" {
		lines = append(lines, fmt.Sprintf("  organization:: %s", task.Organization))
	}
	if task.Project != "" {
		lines = append(lines, fmt.Sprintf("  project:: [[%s]]", task.Project))
	}
	if task.ContextTag != "" {
		lines = append(lines, fmt.Sprintf("  context:: %s", task.ContextTag))
	}
	if task.Assigned != "" {
		lines = append(lines, fmt.Sprintf("  assigned:: @%s", task.Assigned))
	}

	return strings.Join(lines, "\n")
}
