package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Creator struct{}

type Config struct {
	Name                string
	Type                string
	DefaultOrganization string
	Author              string
}

func NewCreator() *Creator {
	return &Creator{}
}

func (c *Creator) CreateWithConfig(path string, config Config) error {
	// Create directory structure
	if err := c.createDirectories(path); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create marker file
	if err := c.createMarkerFile(path, config); err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}

	// Create definitions files
	if err := c.createDefinitions(path, config); err != nil {
		return fmt.Errorf("failed to create definitions: %w", err)
	}

	// Create templates
	if err := c.createTemplates(path); err != nil {
		return fmt.Errorf("failed to create templates: %w", err)
	}

	// Create configuration
	if err := c.createConfiguration(path, config); err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}

	// Create initial files
	if err := c.createInitialFiles(path, config); err != nil {
		return fmt.Errorf("failed to create initial files: %w", err)
	}

	return nil
}

func (c *Creator) createDirectories(basePath string) error {
	directories := []string{
		"definitions",
		"journal",
		"meetings",
		"notes",
		"tasks",
		"templates",
		"assets/images",
		"assets/documents",
	}

	for _, dir := range directories {
		fullPath := filepath.Join(basePath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}

	return nil
}

func (c *Creator) createMarkerFile(basePath string, config Config) error {
	markerContent := fmt.Sprintf(`brain:
  name: "%s"
  type: "%s"
  created: "%s"
  version: "1.0"
config:
  default_organization: "%s"
  author: "%s"
`, config.Name, config.Type, time.Now().Format(time.RFC3339), config.DefaultOrganization, config.Author)

	return c.writeFile(filepath.Join(basePath, ".flip-brain.yaml"), markerContent)
}

func (c *Creator) createDefinitions(basePath string, config Config) error {
	// Organizations
	orgContent := fmt.Sprintf(`organizations:
  %s:
    name: "%s"
    type: "%s"
    description: "Primary organization"
    color: "#4ECDC4"
  PERSONAL:
    name: "Personal"
    type: "personal"
    description: "Personal activities"
    color: "#45B7D1"
`, config.DefaultOrganization, config.DefaultOrganization, config.Type)

	if err := c.writeFile(filepath.Join(basePath, "definitions", "organizations.yaml"), orgContent); err != nil {
		return err
	}

	// Contexts
	contextContent := fmt.Sprintf(`contexts:
  %s:
    GENERAL:
      name: "General"
      status: "active"
      description: "General activities"
  PERSONAL:
    HEALTH:
      name: "Health & Fitness"
      status: "ongoing"
      description: "Health related activities"
`, config.DefaultOrganization)

	if err := c.writeFile(filepath.Join(basePath, "definitions", "contexts.yaml"), contextContent); err != nil {
		return err
	}

	// People
	peopleContent := fmt.Sprintf(`people:
  SELF:
    name: "%s"
    organization: "%s"
    role: "Your Role"
    email: "your.email@example.com"
`, config.Author, config.DefaultOrganization)

	return c.writeFile(filepath.Join(basePath, "definitions", "people.yaml"), peopleContent)
}

func (c *Creator) createTemplates(basePath string) error {
	// Journal template
	journalTemplate := `# {{.Date}}

## Morning Reflection
- 

## Tasks
- [ ] 

## Notes
- 

## Evening Reflection
- What went well:
- What could be improved:
- Tomorrow's priority:
`
	if err := c.writeFile(filepath.Join(basePath, "templates", "journal-template.md"), journalTemplate); err != nil {
		return err
	}

	// Meeting template
	meetingTemplate := `# Meeting: {{.Title}} - {{.Date}}

**Date:** {{.Date}}
**Time:** [HH:MM - HH:MM]
**Organization:** [{{.Organization}}]
**Participants:** {{.Participants}}
**Type:** [standup|planning|review|retrospective|other]

## Agenda
- 

## Discussion
- 

## Decisions
- 

## Action Items
- [ ] [Task] - @[PER] - [{{.Organization}}] - due: YYYY-MM-DD

## Next Steps
- 
`
	if err := c.writeFile(filepath.Join(basePath, "templates", "meeting-template.md"), meetingTemplate); err != nil {
		return err
	}

	// Note template
	noteTemplate := `---
title: "{{.Title}}"
date: {{.Date}}
tags: [{{.Tags}}]
type: note
status: active
---

# {{.Title}}

## Summary
Brief summary of the note content.

## Content
Main note content here.

## Links
- [[Related Note 1]]
- [[Related Note 2]]

## References
- 
`
	return c.writeFile(filepath.Join(basePath, "templates", "note-template.md"), noteTemplate)
}

func (c *Creator) createConfiguration(basePath string, config Config) error {
	configContent := fmt.Sprintf(`repositories:
  - name: "primary"
    path: "."
    type: "primary"

definitions:
  auto_resolve: true
  validate_on_create: true
  suggest_missing: true

templates:
  journal: "templates/journal-template.md"
  meeting: "templates/meeting-template.md"
  note: "templates/note-template.md"

defaults:
  author: "%s"
  timezone: "Europe/Berlin"
  date_format: "2006-01-02"
  default_organization: "%s"

llm:
  provider: "ollama"
  model: "llama2"
  endpoint: "http://localhost:11434"
`, config.Author, config.DefaultOrganization)
	return c.writeFile(filepath.Join(basePath, ".flip.yaml"), configContent)
}

func (c *Creator) createInitialFiles(basePath string, config Config) error {
	// Task inbox
	inboxContent := fmt.Sprintf(`# Task Inbox

## New Tasks
- [ ] Setup your 2nd brain [%s] @SELF due:today priority:high

## Completed
- [x] Initialize 2nd brain [%s] @SELF ✅ today
`, config.DefaultOrganization, config.DefaultOrganization)

	if err := c.writeFile(filepath.Join(basePath, "tasks", "inbox.md"), inboxContent); err != nil {
		return err
	}

	// README
	readmeContent := `# My Second Brain

Welcome to your personal knowledge management system powered by Flip!

## Getting Started

1. **Journal Notes**: Create your daily journal
2. **Meeting Notes**: Create structured meeting notes  
3. **General Notes**: Create knowledge notes

## Folder Structure

- journal/ - Daily journal entries
- meetings/ - Meeting notes and minutes
- notes/ - General knowledge notes
- tasks/ - Task lists and tracking
- definitions/ - Organization, context, and people definitions
- templates/ - Note templates
- assets/ - Images and documents

Happy note-taking! 🚀
`
	return c.writeFile(filepath.Join(basePath, "README.md"), readmeContent)
}

func (c *Creator) writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
