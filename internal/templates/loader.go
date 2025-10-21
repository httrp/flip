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
	TemplateTypeNote    TemplateType = "note"
	TemplateTypeMeeting TemplateType = "meeting"
	TemplateTypeJournal TemplateType = "journal"
	TemplateTypeTask    TemplateType = "task"
)

// Load reads a template file for the given brain type and template type
func Load(brainType brain.BrainType, templateType TemplateType) (string, error) {
	templateDir, err := GetTemplateDir()
	if err != nil {
		return "", err
	}

	// Map brain type to subdirectory
	brainDir := ""
	switch brainType {
	case brain.BrainTypeLogseq:
		brainDir = "logseq"
	case brain.BrainTypeObsidian:
		brainDir = "obsidian"
	case brain.BrainTypeDendron:
		brainDir = "dendron"
	case brain.BrainTypeFlip:
		brainDir = "flip"
	default:
		brainDir = "flip" // Use flip as default
	}

	templatePath := filepath.Join(templateDir, brainDir, string(templateType)+".md")

	// Check if template exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return "", fmt.Errorf("template not found: %s", templatePath)
	}

	// Read template file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	return string(content), nil
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

	brainDir := ""
	switch brainType {
	case brain.BrainTypeLogseq:
		brainDir = "logseq"
	case brain.BrainTypeObsidian:
		brainDir = "obsidian"
	case brain.BrainTypeDendron:
		brainDir = "dendron"
	case brain.BrainTypeFlip:
		brainDir = "flip"
	default:
		brainDir = "flip"
	}

	return filepath.Join(templateDir, brainDir, string(templateType)+".md"), nil
}

// ListTemplates returns all available templates for a brain type
func ListTemplates(brainType brain.BrainType) ([]TemplateType, error) {
	templateDir, err := GetTemplateDir()
	if err != nil {
		return nil, err
	}

	brainDir := ""
	switch brainType {
	case brain.BrainTypeLogseq:
		brainDir = "logseq"
	case brain.BrainTypeObsidian:
		brainDir = "obsidian"
	case brain.BrainTypeDendron:
		brainDir = "dendron"
	case brain.BrainTypeFlip:
		brainDir = "flip"
	default:
		brainDir = "flip"
	}

	brainTemplateDir := filepath.Join(templateDir, brainDir)
	entries, err := os.ReadDir(brainTemplateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read template directory: %w", err)
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
