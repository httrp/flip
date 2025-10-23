package tasks

import (
	"bufio"
	"os"
	"strings"
)

// NoteFrontmatter contains metadata from the note's frontmatter
type NoteFrontmatter struct {
	Organization string
	Project      string
	Context      string
	Tags         []string
	// Can be extended with more fields as needed
}

// ParseFrontmatter extracts YAML frontmatter from a file
// Frontmatter must be at the start of the file between --- delimiters
func ParseFrontmatter(filePath string) (*NoteFrontmatter, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Check if file starts with ---
	if !scanner.Scan() || scanner.Text() != "---" {
		// No frontmatter
		return &NoteFrontmatter{}, nil
	}

	fm := &NoteFrontmatter{
		Tags: []string{},
	}

	// Read until closing ---
	for scanner.Scan() {
		line := scanner.Text()

		// End of frontmatter
		if line == "---" {
			break
		}

		// Parse key: value
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, `"'`)

		switch strings.ToLower(key) {
		case "organization", "org":
			fm.Organization = value
		case "project", "proj":
			fm.Project = value
		case "context", "ctx":
			fm.Context = value
		case "tags":
			// Handle both single tag and array of tags
			// Simple parsing: split by comma or space
			if strings.Contains(value, ",") {
				tags := strings.Split(value, ",")
				for _, tag := range tags {
					tag = strings.TrimSpace(tag)
					tag = strings.TrimPrefix(tag, "#")
					if tag != "" {
						fm.Tags = append(fm.Tags, tag)
					}
				}
			} else {
				// Space-separated or single tag
				tags := strings.Fields(value)
				for _, tag := range tags {
					tag = strings.TrimPrefix(tag, "#")
					if tag != "" {
						fm.Tags = append(fm.Tags, tag)
					}
				}
			}
		}
	}

	return fm, scanner.Err()
}

// ApplyFrontmatterToTask applies note-level metadata to a task
// Task-level metadata takes precedence (doesn't override if already set)
func ApplyFrontmatterToTask(task *Task, fm *NoteFrontmatter) {
	if task.Organization == "" && fm.Organization != "" {
		task.Organization = fm.Organization
	}

	if task.Project == "" && fm.Project != "" {
		task.Project = fm.Project
	}

	if task.ContextTag == "" && fm.Context != "" {
		task.ContextTag = fm.Context
	}

	// Merge tags (frontmatter tags + task tags)
	if len(fm.Tags) > 0 {
		tagMap := make(map[string]bool)

		// Add existing task tags
		for _, tag := range task.Tags {
			tagMap[tag] = true
		}

		// Add frontmatter tags
		for _, tag := range fm.Tags {
			tagMap[tag] = true
		}

		// Convert back to slice
		task.Tags = make([]string, 0, len(tagMap))
		for tag := range tagMap {
			task.Tags = append(task.Tags, tag)
		}
	}
}
