package commands

import (
	"fmt"
	"os"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/templates"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func NewTemplateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage templates",
		Long:  "List, edit, and manage templates for different brain types and content types",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplateMenu()
		},
	}

	return cmd
}

func runTemplateMenu() error {
	// Select action
	actionPrompt := promptui.Select{
		Label: "What would you like to do?",
		Items: []string{"Edit a template", "List templates", "Show template directory", "Back"},
	}

	actionIdx, _, err := actionPrompt.Run()
	if err != nil {
		return err
	}

	switch actionIdx {
	case 0: // Edit template
		return editTemplateFlow()
	case 1: // List templates
		return listTemplatesFlow()
	case 2: // Show template directory
		return showTemplateDirectoryFlow()
	case 3: // Back
		return nil
	}

	return nil
}

func editTemplateFlow() error {
	// Select brain type
	brainTypePrompt := promptui.Select{
		Label: "Select brain type",
		Items: []string{"flip", "obsidian", "logseq", "dendron"},
	}

	brainIdx, brainStr, err := brainTypePrompt.Run()
	if err != nil {
		return err
	}

	var brainType brain.BrainType
	switch brainIdx {
	case 0:
		brainType = brain.BrainTypeFlip
	case 1:
		brainType = brain.BrainTypeObsidian
	case 2:
		brainType = brain.BrainTypeLogseq
	case 3:
		brainType = brain.BrainTypeDendron
	}

	// Select template type
	templatePrompt := promptui.Select{
		Label: fmt.Sprintf("Select template type for %s", brainStr),
		Items: []string{"note", "meeting", "journal"},
	}

	templateIdx, _, err := templatePrompt.Run()
	if err != nil {
		return err
	}

	var templateType templates.TemplateType
	switch templateIdx {
	case 0:
		templateType = templates.TemplateTypeNote
	case 1:
		templateType = templates.TemplateTypeMeeting
	case 2:
		templateType = templates.TemplateTypeJournal
	}

	// Get template path
	templatePath, err := templates.GetTemplatePath(brainType, templateType)
	if err != nil {
		return fmt.Errorf("failed to get template path: %w", err)
	}

	// Check if template exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		fmt.Printf("Template not found: %s\n", templatePath)
		return nil
	}

	fmt.Printf("\nOpening template: %s\n", templatePath)
	fmt.Printf("Available placeholders: {{title}}, {{date}}, {{time}}, {{tags}}, {{participants}}, {{id}}, {{created}}, {{updated}}\n\n")

	// Open in editor
	return openInEditor(templatePath)
}

func listTemplatesFlow() error {
	fmt.Println("\n=== Available Templates ===")
	fmt.Println()

	brainTypes := []struct {
		name      string
		brainType brain.BrainType
	}{
		{"Flip", brain.BrainTypeFlip},
		{"Obsidian", brain.BrainTypeObsidian},
		{"Logseq", brain.BrainTypeLogseq},
		{"Dendron", brain.BrainTypeDendron},
	}

	for _, bt := range brainTypes {
		fmt.Printf("\n%s:\n", bt.name)
		tmpl, err := templates.ListTemplates(bt.brainType)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		if len(tmpl) == 0 {
			fmt.Printf("  No templates found\n")
			continue
		}

		for _, t := range tmpl {
			path, _ := templates.GetTemplatePath(bt.brainType, t)
			fmt.Printf("  - %s (%s)\n", t, path)
		}
	}

	fmt.Println()
	return nil
}

func showTemplateDirectoryFlow() error {
	templateDir, err := templates.GetTemplateDir()
	if err != nil {
		return fmt.Errorf("failed to get template directory: %w", err)
	}

	fmt.Printf("\nTemplate directory: %s\n\n", templateDir)
	fmt.Println("Directory structure:")
	fmt.Println("  templates/")
	fmt.Println("    ├── flip/      (Flip brain templates)")
	fmt.Println("    ├── obsidian/  (Obsidian vault templates)")
	fmt.Println("    ├── logseq/    (Logseq graph templates)")
	fmt.Println("    └── dendron/   (Dendron workspace templates)")
	fmt.Println()
	fmt.Println("Each directory contains:")
	fmt.Println("  - note.md      (Regular notes)")
	fmt.Println("  - meeting.md   (Meeting notes)")
	fmt.Println("  - journal.md   (Daily journal/notes)")
	fmt.Println()

	// Offer to open in file browser
	openPrompt := promptui.Select{
		Label: "Open template directory in Finder?",
		Items: []string{"Yes", "No"},
	}

	idx, _, err := openPrompt.Run()
	if err != nil {
		return err
	}

	if idx == 0 {
		return openInFinder(templateDir)
	}

	return nil
}

// openInFinder opens a directory in macOS Finder
func openInFinder(dirPath string) error {
	return trySystemOpen(dirPath)
}
