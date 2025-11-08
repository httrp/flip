package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// TaskDefinitions stores definitions for organizations, projects, and contexts
type TaskDefinitions struct {
	Organizations []OrganizationDef `yaml:"organizations"`
	Projects      []ProjectDef      `yaml:"projects"`
	Contexts      []ContextDef      `yaml:"contexts"`
}

// OrganizationDef defines an organization with its abbreviation
type OrganizationDef struct {
	Name         string `yaml:"name"`
	Abbreviation string `yaml:"abbreviation"`
	Description  string `yaml:"description,omitempty"`
}

// ProjectDef defines a project with its abbreviation and organization
type ProjectDef struct {
	Name         string `yaml:"name"`
	Abbreviation string `yaml:"abbreviation"`
	Organization string `yaml:"organization,omitempty"`
	Description  string `yaml:"description,omitempty"`
}

// ContextDef defines a context with its abbreviation
type ContextDef struct {
	Name         string `yaml:"name"`
	Abbreviation string `yaml:"abbreviation"`
	Description  string `yaml:"description,omitempty"`
}

// getTaskDefinitionsPath returns the path to the task definitions file
// First checks brain's definitions, then falls back to user config
func getTaskDefinitionsPath() (string, error) {
	// Try to get active brain's definitions first
	brain, err := getActiveBrain()
	if err == nil && brain != nil {
		brainDefPath := filepath.Join(brain.Path, "definitions", "task-definitions.yaml")
		if _, err := os.Stat(brainDefPath); err == nil {
			return brainDefPath, nil
		}
	}

	// Fall back to user config directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	flipDir := filepath.Join(configDir, "flip")
	if err := os.MkdirAll(flipDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(flipDir, "task-definitions.yaml"), nil
}

// loadTaskDefinitions loads the task definitions from disk
func loadTaskDefinitions() (*TaskDefinitions, error) {
	path, err := getTaskDefinitionsPath()
	if err != nil {
		return createDefaultDefinitions(), err
	}

	// If file doesn't exist, create default
	if _, err := os.Stat(path); os.IsNotExist(err) {
		defs := createDefaultDefinitions()
		if saveErr := saveTaskDefinitions(defs); saveErr != nil {
			return defs, saveErr
		}
		return defs, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return createDefaultDefinitions(), err
	}

	var defs TaskDefinitions
	if err := yaml.Unmarshal(data, &defs); err != nil {
		return createDefaultDefinitions(), err
	}

	return &defs, nil
}

// saveTaskDefinitions saves the task definitions to disk
func saveTaskDefinitions(defs *TaskDefinitions) error {
	path, err := getTaskDefinitionsPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(defs)
	if err != nil {
		return err
	}

	// Add header comment
	header := `# Task Definitions
# Define your organizations, projects, and contexts here with abbreviations
# This file can be in your brain's definitions folder or in ~/.config/flip/

`
	content := header + string(data)

	return os.WriteFile(path, []byte(content), 0644)
}

// createDefaultDefinitions creates default definitions
func createDefaultDefinitions() *TaskDefinitions {
	return &TaskDefinitions{
		Organizations: []OrganizationDef{
			{
				Name:         "Personal",
				Abbreviation: "PERS",
				Description:  "Personal tasks and projects",
			},
			{
				Name:         "Work",
				Abbreviation: "WORK",
				Description:  "Work-related tasks",
			},
		},
		Projects: []ProjectDef{
			{
				Name:         "Example Project",
				Abbreviation: "EX",
				Organization: "WORK",
				Description:  "Example project",
			},
		},
		Contexts: []ContextDef{
			{
				Name:         "Meeting",
				Abbreviation: "MTG",
				Description:  "Tasks from meetings",
			},
			{
				Name:         "Email",
				Abbreviation: "EMAIL",
				Description:  "Tasks from emails",
			},
		},
	}
}

// findOrganizationByAbbr finds an organization by abbreviation
func (d *TaskDefinitions) findOrganizationByAbbr(abbr string) *OrganizationDef {
	for i := range d.Organizations {
		if d.Organizations[i].Abbreviation == abbr {
			return &d.Organizations[i]
		}
	}
	return nil
}

// findProjectByAbbr finds a project by abbreviation
func (d *TaskDefinitions) findProjectByAbbr(abbr string) *ProjectDef {
	for i := range d.Projects {
		if d.Projects[i].Abbreviation == abbr {
			return &d.Projects[i]
		}
	}
	return nil
}

// findContextByAbbr finds a context by abbreviation
func (d *TaskDefinitions) findContextByAbbr(abbr string) *ContextDef {
	for i := range d.Contexts {
		if d.Contexts[i].Abbreviation == abbr {
			return &d.Contexts[i]
		}
	}
	return nil
}

// getOrganizationChoices returns formatted choices for organization selection
func (d *TaskDefinitions) getOrganizationChoices() []string {
	choices := make([]string, len(d.Organizations))
	for i, org := range d.Organizations {
		choices[i] = fmt.Sprintf("[%s] %s", org.Abbreviation, org.Name)
	}
	sort.Strings(choices)
	return choices
}

// getProjectChoices returns formatted choices for project selection
func (d *TaskDefinitions) getProjectChoices() []string {
	choices := make([]string, len(d.Projects))
	for i, proj := range d.Projects {
		if proj.Organization != "" {
			choices[i] = fmt.Sprintf("[%s] %s (%s)", proj.Abbreviation, proj.Name, proj.Organization)
		} else {
			choices[i] = fmt.Sprintf("[%s] %s", proj.Abbreviation, proj.Name)
		}
	}
	sort.Strings(choices)
	return choices
}

// getContextChoices returns formatted choices for context selection
func (d *TaskDefinitions) getContextChoices() []string {
	choices := make([]string, len(d.Contexts))
	for i, ctx := range d.Contexts {
		choices[i] = fmt.Sprintf("[%s] %s", ctx.Abbreviation, ctx.Name)
	}
	sort.Strings(choices)
	return choices
}

// parseAbbreviationFromChoice extracts abbreviation from a choice string like "[ABBR] Name"
func parseAbbreviationFromChoice(choice string) string {
	if len(choice) < 2 {
		return ""
	}
	// Extract text between [ and ]
	start := 0
	end := len(choice)
	for i, c := range choice {
		if c == '[' {
			start = i + 1
		} else if c == ']' {
			end = i
			break
		}
	}
	if start < end {
		return choice[start:end]
	}
	return ""
}
