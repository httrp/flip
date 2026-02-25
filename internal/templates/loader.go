package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/brain"
)

// GetTemplateDir returns the path to the templates directory
func GetTemplateDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	templateDir := filepath.Join(homeDir, "Library", "Application Support", "flip", "templates")
	return templateDir, nil
}

// TemplateType represents the type of template
type TemplateType string

const (
	TemplateTypeNote         TemplateType = "note"
	TemplateTypeMeeting      TemplateType = "meeting"
	TemplateTypeJournal      TemplateType = "journal"
	TemplateTypeTask         TemplateType = "task"
	TemplateTypeExercise     TemplateType = "exercise"
	TemplateTypeExercisePlan TemplateType = "exercise-plan"
)

// Load reads a template file for the given brain type and template type
// Falls back to embedded defaults if file not found
func Load(brainType brain.BrainType, templateType TemplateType) (string, error) {
	templateDir, err := GetTemplateDir()
	if err != nil {
		return GetDefault(brainType, templateType), nil
	}

	brainDir := getBrainDir(brainType)
	templatePath := filepath.Join(templateDir, brainDir, string(templateType)+".md")

	// Check if template exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return GetDefault(brainType, templateType), nil
	}

	// Read template file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return GetDefault(brainType, templateType), nil
	}

	return string(content), nil
}

// GetDefault returns the embedded default template
func GetDefault(brainType brain.BrainType, templateType TemplateType) string {
	defaults := getDefaultTemplates(brainType)
	if tmpl, ok := defaults[templateType]; ok {
		return tmpl
	}
	// Ultimate fallback
	return "# {{title}}\n"
}

// getBrainDir maps brain type to directory name
func getBrainDir(brainType brain.BrainType) string {
	switch brainType {
	case brain.BrainTypeLogseq:
		return "logseq"
	case brain.BrainTypeObsidian:
		return "obsidian"
	case brain.BrainTypeDendron:
		return "dendron"
	case brain.BrainTypeFoam:
		return "foam"
	case brain.BrainTypeFlip:
		return "flip"
	default:
		return "flip"
	}
}

// getDefaultTemplates returns embedded default templates for a brain type
func getDefaultTemplates(brainType brain.BrainType) map[TemplateType]string {
	switch brainType {
	case brain.BrainTypeLogseq:
		return logseqDefaults
	case brain.BrainTypeObsidian:
		return obsidianDefaults
	case brain.BrainTypeDendron:
		return dendronDefaults
	case brain.BrainTypeFoam:
		return foamDefaults
	default:
		return flipDefaults
	}
}

// ============================================================================
// Embedded Default Templates - Clean & Minimal
// ============================================================================

var flipDefaults = map[TemplateType]string{
	TemplateTypeNote: `---
title: {{title}}
created: {{date}}
tags: []
---

# {{title}}

`,

	TemplateTypeMeeting: `---
title: {{title}}
date: {{date}}
duration: {{duration}}
type: meeting
tags: [meeting]
---

# {{title}}

## Participants

{{participants}}

## Notes

## Action Items

`,

	TemplateTypeJournal: `---
date: {{date}}
day: {{weekday}}
brain: {{brain_name}}
type: journal
---

# {{date}} - {{weekday}}

## Activities

## Meeting-Notes

## New Tasks

## Exercises

`,

	TemplateTypeTask: `---
title: {{title}}
created: {{date}}
status: todo
priority: normal
tags: [task]
---

# {{title}}

## Description

## Notes

`,
}

var obsidianDefaults = map[TemplateType]string{
	TemplateTypeNote: `---
title: {{title}}
created: {{date}}
tags: []
---

# {{title}}

`,

	TemplateTypeMeeting: `---
title: {{title}}
date: {{date}}
duration: {{duration}}
type: meeting
tags:
  - meeting
---

# {{title}}

## Participants

{{participants}}

## Notes

## Action Items

`,

	TemplateTypeJournal: `---
date: {{date}}
day: {{weekday}}
brain: {{brain_name}}
type: journal
---

# {{date}} - {{weekday}}

## Activities

## Meeting-Notes

## New Tasks

## Exercises

`,

	TemplateTypeTask: `---
title: {{title}}
created: {{date}}
status: todo
priority: normal
tags:
  - task
---

# {{title}}

## Description

## Notes

`,
}

var logseqDefaults = map[TemplateType]string{
	TemplateTypeNote: `- title:: {{title}}
- created:: {{date}}
- tags:: 

- # {{title}}
	- 
`,

	TemplateTypeMeeting: `- title:: {{title}}
- date:: {{date}}
- duration:: {{duration}}
- type:: meeting
- tags:: meeting

- # {{title}}
	- ## Participants
		- {{participants}}
	- ## Notes
		- 
	- ## Action Items
		- TODO 
`,

	TemplateTypeJournal: `- brain:: {{brain_name}}
- # {{date}}
	- ## Activities
		- 
	- ## Meeting-Notes
		- 
	- ## New Tasks
		- 
	- ## Exercises
		- 
`,

	TemplateTypeTask: `- title:: {{title}}
- created:: {{date}}
- status:: TODO
- priority:: normal
- tags:: task

- # {{title}}
	- ## Description
		- 
	- ## Notes
		- 
`,
}

var dendronDefaults = map[TemplateType]string{
	TemplateTypeNote: `---
id: {{id}}
title: {{title}}
created: {{created}}
updated: {{updated}}
---

# {{title}}

`,

	TemplateTypeMeeting: `---
id: {{id}}
title: {{title}}
created: {{created}}
updated: {{updated}}
duration: {{duration}}
type: meeting
---

# {{title}}

## Participants

{{participants}}

## Notes

## Action Items

`,

	TemplateTypeJournal: `---
id: {{id}}
title: {{date}}
created: {{created}}
updated: {{updated}}
brain: {{brain_name}}
---

# {{date}} - {{weekday}}

## Activities

## Meeting-Notes

## New Tasks

## Exercises

`,

	TemplateTypeTask: `---
id: {{id}}
title: {{title}}
created: {{created}}
updated: {{updated}}
status: todo
priority: normal
---

# {{title}}

## Description

## Notes

`,
}

var foamDefaults = map[TemplateType]string{
	TemplateTypeNote: `---
title: {{title}}
date: {{date}}
tags: []
---

# {{title}}

`,

	TemplateTypeMeeting: `---
title: {{title}}
date: {{date}}
duration: {{duration}}
type: meeting
tags:
  - meeting
---

# {{title}}

## Participants

{{participants}}

## Notes

## Action Items

`,

	TemplateTypeJournal: `---
date: {{date}}
day: {{weekday}}
brain: {{brain_name}}
type: journal
---

# {{date}} - {{weekday}}

## Activities

## Meeting-Notes

## New Tasks

## Exercises

`,

	TemplateTypeTask: `---
title: {{title}}
date: {{date}}
status: todo
priority: normal
tags:
  - task
---

# {{title}}

## Description

## Notes

`,
}

// ============================================================================
// Template Management Functions
// ============================================================================

// InitTemplates creates default template files if they don't exist
func InitTemplates() error {
	templateDir, err := GetTemplateDir()
	if err != nil {
		return err
	}

	brainTypes := []brain.BrainType{
		brain.BrainTypeFlip,
		brain.BrainTypeObsidian,
		brain.BrainTypeLogseq,
		brain.BrainTypeDendron,
		brain.BrainTypeFoam,
	}

	templateTypes := []TemplateType{
		TemplateTypeNote,
		TemplateTypeMeeting,
		TemplateTypeJournal,
		TemplateTypeTask,
	}

	for _, bt := range brainTypes {
		brainDir := filepath.Join(templateDir, getBrainDir(bt))
		if err := os.MkdirAll(brainDir, 0755); err != nil {
			return fmt.Errorf("failed to create template directory: %w", err)
		}

		defaults := getDefaultTemplates(bt)
		for _, tt := range templateTypes {
			templatePath := filepath.Join(brainDir, string(tt)+".md")

			// Only create if doesn't exist
			if _, err := os.Stat(templatePath); os.IsNotExist(err) {
				if content, ok := defaults[tt]; ok {
					if err := os.WriteFile(templatePath, []byte(content), 0644); err != nil {
						return fmt.Errorf("failed to write template: %w", err)
					}
				}
			}
		}
	}

	return nil
}

// ResetTemplate resets a specific template to default
func ResetTemplate(brainType brain.BrainType, templateType TemplateType) error {
	templateDir, err := GetTemplateDir()
	if err != nil {
		return err
	}

	brainDir := filepath.Join(templateDir, getBrainDir(brainType))
	templatePath := filepath.Join(brainDir, string(templateType)+".md")

	// Ensure directory exists
	if err := os.MkdirAll(brainDir, 0755); err != nil {
		return err
	}

	// Write default content
	content := GetDefault(brainType, templateType)
	return os.WriteFile(templatePath, []byte(content), 0644)
}

// ResetBrainTemplates resets all templates for a specific brain type to defaults
func ResetBrainTemplates(brainType brain.BrainType) error {
	templateTypes := []TemplateType{
		TemplateTypeNote,
		TemplateTypeMeeting,
		TemplateTypeJournal,
		TemplateTypeTask,
	}

	for _, tt := range templateTypes {
		if err := ResetTemplate(brainType, tt); err != nil {
			return err
		}
	}
	return nil
}

// ResetAllTemplates resets all templates to defaults
func ResetAllTemplates() error {
	templateDir, err := GetTemplateDir()
	if err != nil {
		return err
	}

	// Remove entire templates directory
	if err := os.RemoveAll(templateDir); err != nil {
		return err
	}

	// Recreate with defaults
	return InitTemplates()
}

// Render replaces placeholders in the template with actual values
func Render(template string, vars map[string]string) string {
	result := template
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// GetTemplatePath returns the full path to a specific template file
func GetTemplatePath(brainType brain.BrainType, templateType TemplateType) (string, error) {
	templateDir, err := GetTemplateDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(templateDir, getBrainDir(brainType), string(templateType)+".md"), nil
}

// ListTemplates returns all available templates for a brain type
func ListTemplates(brainType brain.BrainType) ([]TemplateType, error) {
	templateDir, err := GetTemplateDir()
	if err != nil {
		return nil, err
	}

	brainTemplateDir := filepath.Join(templateDir, getBrainDir(brainType))
	entries, err := os.ReadDir(brainTemplateDir)
	if err != nil {
		// Return default template types if directory doesn't exist
		return []TemplateType{
			TemplateTypeNote,
			TemplateTypeMeeting,
			TemplateTypeJournal,
			TemplateTypeTask,
		}, nil
	}

	var templates []TemplateType
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			templateName := strings.TrimSuffix(entry.Name(), ".md")
			templates = append(templates, TemplateType(templateName))
		}
	}

	return templates, nil
}
