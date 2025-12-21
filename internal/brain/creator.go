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
	return c.CreateCompatible(path, config, BrainTypeEmpty)
}

// CreateCompatible creates a flip brain structure that's compatible with existing brain systems
func (c *Creator) CreateCompatible(path string, config Config, existingType BrainType) error {
	fmt.Printf("> Creating flip structure compatible with %s...\n", existingType)

	// Create directory structure (respectfully)
	if err := c.createDirectoriesCompatible(path, existingType); err != nil {
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

	// Create templates compatible with existing system
	if err := c.createTemplatesCompatible(path, existingType); err != nil {
		return fmt.Errorf("failed to create templates: %w", err)
	}

	// Create configuration
	if err := c.createConfiguration(path, config); err != nil {
		return fmt.Errorf("failed to create configuration: %w", err)
	}

	// Create initial files (if appropriate)
	if existingType == BrainTypeEmpty || existingType == BrainTypeFlip || existingType == BrainTypeUnknown {
		if err := c.createInitialFiles(path, config); err != nil {
			return fmt.Errorf("failed to create initial files: %w", err)
		}
	}

	// Create example files tailored to the existing/dialect type
	if err := c.createExamplesCompatible(path, config, existingType); err != nil {
		return fmt.Errorf("failed to create example files: %w", err)
	}

	fmt.Printf("[OK] Flip brain initialized successfully!\n")
	return nil
}

// createExamplesCompatible adds a few example markdown files to help onboarding
func (c *Creator) createExamplesCompatible(basePath string, config Config, brainType BrainType) error {
	today := time.Now().Format("2006-01-02")
	// Welcome note explaining structure
	welcomePath := filepath.Join(basePath, "WELCOME.md")
	if _, err := os.Stat(welcomePath); os.IsNotExist(err) {
		content := fmt.Sprintf(`# Welcome to Flip

This folder is a 2nd brain workspace managed by Flip.

Key concepts:
- Workspace: a registered 2nd brain folder (this one)
- Dialect: the note system style (flip/obsidian/logseq/dendron)
- Structure: folders and templates matching your chosen dialect

Author: %s
Date: %s
`, config.Author, today)
		if err := c.writeFile(welcomePath, content); err != nil {
			return err
		}
	}

	switch brainType {
	case BrainTypeObsidian:
		// Daily note
		dn := filepath.Join(basePath, "Daily Notes", today+".md")
		if _, err := os.Stat(dn); os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(dn), 0755)
			_ = c.writeFile(dn, `# {{date:YYYY-MM-DD}}

## Notes
- Welcome to your Obsidian-compatible Flip workspace.
`)
		}
		// Example note
		ex := filepath.Join(basePath, "Projects", "Example Note.md")
		if _, err := os.Stat(ex); os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(ex), 0755)
			_ = c.writeFile(ex, `---
title: "Example Note"
date: {{date:YYYY-MM-DD}}
tags: [example]
---

# Example Note

This note demonstrates Obsidian frontmatter and headings.`)
		}
	case BrainTypeLogseq:
		// Journal
		jname := time.Now().Format("2006_01_02") + ".md"
		jpath := filepath.Join(basePath, "journals", jname)
		if _, err := os.Stat(jpath); os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(jpath), 0755)
			_ = c.writeFile(jpath, `- # Journal
- ## Notes
  - Welcome to your Logseq-compatible Flip workspace.`)
		}
		// Page
		p := filepath.Join(basePath, "pages", "example.md")
		if _, err := os.Stat(p); os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(p), 0755)
			_ = c.writeFile(p, `title:: Example Page
tags:: example

- # Example
  - Demonstrates Logseq properties and bullets.`)
		}
	case BrainTypeDendron:
		// Welcome note in notes/
		n := filepath.Join(basePath, "notes", "welcome.md")
		if _, err := os.Stat(n); os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(n), 0755)
			_ = c.writeFile(n, `---
id: welcome
title: Welcome
---

# Welcome

This is a Dendron-compatible workspace scaffolded by Flip.`)
		}
	default:
		// Flip default: journal + example note
		j := filepath.Join(basePath, "journal", today+".md")
		if _, err := os.Stat(j); os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(j), 0755)
			_ = c.writeFile(j, `# `+today+`

## Notes
- First journal entry created by Flip.`)
		}
		note := filepath.Join(basePath, "notes", "example.md")
		if _, err := os.Stat(note); os.IsNotExist(err) {
			_ = os.MkdirAll(filepath.Dir(note), 0755)
			_ = c.writeFile(note, `---
title: "Example"
date: `+today+`
tags: [example]
type: note
status: active
---

# Example

This is an example note to get you started.`)
		}
	}
	return nil
}

func (c *Creator) createDirectories(basePath string) error {
	return c.createDirectoriesCompatible(basePath, BrainTypeEmpty)
}

func (c *Creator) createDirectoriesCompatible(basePath string, existingType BrainType) error {
	directories := c.getDirectoriesForType(basePath, existingType)

	for _, dir := range directories {
		fullPath := filepath.Join(basePath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
		}
	}

	return nil
}

func (c *Creator) getDirectoriesForType(basePath string, brainType BrainType) []string {
	baseDirectories := []string{
		"definitions",
		"templates",
		"assets/images",
		"assets/documents",
	}

	switch brainType {
	case BrainTypeObsidian:
		// Obsidian compatible structure
		return append(baseDirectories, []string{
			"Daily Notes", // Obsidian's default daily notes folder
			"Templates",   // Obsidian templates
			"Meetings",
			"Projects",
			"Tasks",
		}...)

	case BrainTypeLogseq:
		// Logseq compatible structure
		existingJournals := filepath.Join(basePath, "journals")
		existingPages := filepath.Join(basePath, "pages")

		dirs := baseDirectories
		if _, err := os.Stat(existingJournals); os.IsNotExist(err) {
			dirs = append(dirs, "journals")
		}
		if _, err := os.Stat(existingPages); os.IsNotExist(err) {
			dirs = append(dirs, "pages")
		}

		return append(dirs, []string{
			"meetings",
			"tasks",
		}...)

	case BrainTypeDendron:
		// Dendron compatible structure - respect existing vault structure
		return append(baseDirectories, []string{
			"notes",
			"meetings",
			"tasks",
		}...)

	case BrainTypeFoam:
		// Foam compatible structure - flexible like Obsidian but with common conventions
		return append(baseDirectories, []string{
			"journal",     // Daily notes
			"notes",       // General notes
			"docs",        // Documentation (common in Foam for GitHub Pages)
			"attachments", // Assets
		}...)

	default:
		// Default flip structure
		return append(baseDirectories, []string{
			"journal",
			"meetings",
			"notes",
			"tasks",
		}...)
	}
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
	return c.createTemplatesCompatible(basePath, BrainTypeEmpty)
}

func (c *Creator) createTemplatesCompatible(basePath string, existingType BrainType) error {
	templateDir := "templates"

	// Adjust template location based on brain type
	if existingType == BrainTypeObsidian {
		templateDir = "Templates" // Obsidian's default template folder
	}

	// Journal template - adapted for different systems
	journalTemplate := c.getJournalTemplate(existingType)
	if err := c.writeFile(filepath.Join(basePath, templateDir, "journal-template.md"), journalTemplate); err != nil {
		return err
	}

	// Meeting template - adapted for different systems
	meetingTemplate := c.getMeetingTemplate(existingType)
	if err := c.writeFile(filepath.Join(basePath, templateDir, "meeting-template.md"), meetingTemplate); err != nil {
		return err
	}

	// Note template - adapted for different systems
	noteTemplate := c.getNoteTemplate(existingType)
	return c.writeFile(filepath.Join(basePath, templateDir, "note-template.md"), noteTemplate)
}

func (c *Creator) getJournalTemplate(brainType BrainType) string {
	switch brainType {
	case BrainTypeLogseq:
		// Logseq uses block structure and different date format
		return `- ## Morning Reflection
  - 
- ## Tasks
  - TODO 
- ## Notes
  - 
- ## Evening Reflection
  - What went well:
  - What could be improved:
  - Tomorrow's priority:`

	case BrainTypeObsidian:
		// Obsidian template with template variables
		return `---
title: "Journal {{date:YYYY-MM-DD}}"
date: {{date:YYYY-MM-DD}}
type: journal
mood: 
energy: 
weather: 
tags: []
---

# {{date:YYYY-MM-DD}}

`

	default:
		// Default flip template - Metadata-rich, Content-minimal
		return `---
title: "Journal {{.Date}}"
date: {{.Date}}
type: journal
mood: 
energy: 
weather: 
tags: []
highlights: []
gratitude: []
---

# {{.Date}}

`
	}
}

func (c *Creator) getMeetingTemplate(brainType BrainType) string {
	switch brainType {
	case BrainTypeLogseq:
		return `- # {{.Title}}
- **Date:** {{.Date}}
- **Organization:** {{.Organization}}
- **Participants:** 
- **Type:** 
- ## Notes
  - 
- ## Action Items
  - TODO `

	case BrainTypeObsidian:
		return `---
title: "{{title}}"
date: {{date:YYYY-MM-DD}}
type: meeting
attendees: []
organization: 
project: 
location: 
tags: []
decisions: []
action_items: []
follow_up: 
---

# {{title}}

`

	default:
		// Default flip template - Metadata-rich, Content-minimal
		return `---
title: "{{.Title}}"
date: {{.Date}}
type: meeting
attendees: []
organization: {{.Organization}}
project: 
location: 
tags: []
decisions: []
action_items: []
follow_up: 
---

# {{.Title}}

`
	}
}

func (c *Creator) getNoteTemplate(brainType BrainType) string {
	switch brainType {
	case BrainTypeLogseq:
		// Logseq uses properties and block structure
		return `title:: {{.Title}}
date:: {{.Date}}
tags:: {{.Tags}}
type:: note
status:: active

- # {{.Title}}
  - `

	case BrainTypeObsidian:
		// Obsidian uses YAML frontmatter - Metadata-rich, Content-minimal
		return `---
title: "{{title}}"
date: {{date:YYYY-MM-DD}}
type: note
status: active
tags: []
related: []
---

# {{title}}

`

	default:
		// Default flip template - Metadata-rich, Content-minimal
		return `---
title: "{{.Title}}"
date: {{.Date}}
type: note
status: active
tags: []
related: []
---

# {{.Title}}

`
	}
}

func (c *Creator) createConfiguration(basePath string, config Config) error {
	configContent := fmt.Sprintf(`# ==================================================================
# FLIP BRAIN CONFIGURATION
# ==================================================================
# ⚠️  WARNING: Manual editing can break your brain setup!
# 
# This file configures the behavior of your flip brain.
# 
# Recommended: Use 'flip' commands for safe changes:
#   flip brain rename <name>    - Rename your brain
#   flip brain repair           - Fix configuration issues
#   Use the flip menu           - Interactive, safe operations
#
# Only edit manually if you understand the format and implications.
# Invalid YAML or incorrect paths can prevent flip from working!
# ==================================================================

repositories:
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
  # Note: LLM integration is planned for future releases
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

Happy note-taking!
`
	return c.writeFile(filepath.Join(basePath, "README.md"), readmeContent)
}

func (c *Creator) writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
