package commands

import (
	"fmt"
	"os"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/templates"
	"github.com/httrp/flip/internal/ui"
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
	fmt.Println("\n🔧 Template Management")

	menuItems := []struct {
		Label       string
		Description string
		Action      func() error
	}{
		{
			Label:       "📋 List Templates",
			Description: "Show all available templates",
			Action:      listTemplatesFlow,
		},
		{
			Label:       "👁️  View Template",
			Description: "View template content",
			Action:      viewTemplateFlow,
		},
		{
			Label:       "✏️  Edit Template",
			Description: "Modify an existing template",
			Action:      editTemplateFlow,
		},
		{
			Label:       "🔄 Reset Template",
			Description: "Restore template to default",
			Action:      resetTemplateFlow,
		},
		{
			Label:       "📂 Open Template Directory",
			Description: "Open templates folder in file manager",
			Action:      showTemplateDirectoryFlow,
		},
		{
			Label:       "◀️  Back",
			Description: "Return to manage menu",
			Action: func() error {
				return nil
			},
		},
	}

	selectItems := make([]ui.SelectItem, len(menuItems))
	for i, item := range menuItems {
		selectItems[i] = ui.SelectItem{Label: fmt.Sprintf("%s - %s", item.Label, item.Description), Value: fmt.Sprintf("%d", i)}
	}

	idx, _, err := ui.RunSelect("What would you like to do?", selectItems, 10)
	if err != nil {
		return err
	}

	if idx == len(menuItems)-1 { // Back
		return nil
	}

	return menuItems[idx].Action()
}

func editTemplateFlow() error {
	// Select brain type
	_, brainStr, err := ui.RunSelect("Select brain type", []ui.SelectItem{
		{Label: "flip", Value: "flip"},
		{Label: "obsidian", Value: "obsidian"},
		{Label: "logseq", Value: "logseq"},
		{Label: "dendron", Value: "dendron"},
	}, 0)
	if err != nil {
		return runTemplateMenu()
	}

	brainType := parseBrainType(brainStr)

	// Select template type
	_, templateStr, err := ui.RunSelect(fmt.Sprintf("Select template type for %s", brainStr), []ui.SelectItem{
		{Label: "note", Value: "note"},
		{Label: "meeting", Value: "meeting"},
		{Label: "journal", Value: "journal"},
		{Label: "task", Value: "task"},
		{Label: "prompt", Value: "prompt"},
	}, 0)
	if err != nil {
		return runTemplateMenu()
	}

	templateType := parseTemplateType(templateStr)

	// Get template path
	templatePath, err := templates.GetTemplatePath(brainType, templateType)
	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		return runTemplateMenu()
	}

	// Check if template exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		fmt.Printf("\n❌ Template not found: %s\n", templatePath)
		return runTemplateMenu()
	}

	fmt.Printf("\n✏️  Opening template: %s\n", templatePath)
	fmt.Println("\nAvailable placeholders:")
	fmt.Println("  {{title}}        - Note title")
	fmt.Println("  {{date}}         - Current date (YYYY-MM-DD)")
	fmt.Println("  {{time}}         - Current time (HH:MM)")
	fmt.Println("  {{tags}}         - Tags placeholder")
	fmt.Println("  {{participants}} - Meeting participants")
	fmt.Println("  {{id}}           - Unique ID")
	fmt.Println("  {{created}}      - Creation timestamp")
	fmt.Println("  {{updated}}      - Last update timestamp")
	fmt.Println()

	// Open in editor
	if err := openInEditor(templatePath); err != nil {
		fmt.Printf("❌ Error opening editor: %v\n", err)
	}

	return runTemplateMenu()
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
		{"Foam", brain.BrainTypeFoam},
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
			// Check if file exists
			if _, err := os.Stat(path); err == nil {
				fmt.Printf("  ✅ %s\n", t)
			} else {
				fmt.Printf("  ❌ %s (not found)\n", t)
			}
		}
	}
	return runTemplateMenu()
}

func viewTemplateFlow() error {
	// Select brain type
	_, brainStr, err := ui.RunSelect("Select brain type", []ui.SelectItem{
		{Label: "flip", Value: "flip"},
		{Label: "obsidian", Value: "obsidian"},
		{Label: "logseq", Value: "logseq"},
		{Label: "dendron", Value: "dendron"},
		{Label: "foam", Value: "foam"},
	}, 0)
	if err != nil {
		return runTemplateMenu()
	}

	brainType := parseBrainType(brainStr)

	// Select template type
	_, templateStr, err := ui.RunSelect(fmt.Sprintf("Select template type for %s", brainStr), []ui.SelectItem{
		{Label: "note", Value: "note"},
		{Label: "meeting", Value: "meeting"},
		{Label: "journal", Value: "journal"},
		{Label: "task", Value: "task"},
		{Label: "prompt", Value: "prompt"},
	}, 0)
	if err != nil {
		return runTemplateMenu()
	}

	templateType := parseTemplateType(templateStr)

	// Load and display template
	content, err := templates.Load(brainType, templateType)
	if err != nil {
		fmt.Printf("\n❌ Error loading template: %v\n", err)
		return runTemplateMenu()
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Template: %s - %s\n", brainStr, templateType)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(content)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("\nAvailable placeholders: {{title}}, {{date}}, {{time}}, {{tags}}, {{participants}}, {{id}}, {{created}}, {{updated}}")
	return runTemplateMenu()
}

func resetTemplateFlow() error {
	fmt.Println("\n🔄 Reset Template")

	// Option to reset all or single template
	_, scopeChoice, err := ui.RunSelect("What would you like to reset?", []ui.SelectItem{
		{Label: "Single template", Value: "single"},
		{Label: "All templates for a brain type", Value: "all"},
		{Label: "◀️  Back", Value: "back"},
	}, 0)
	if err != nil || scopeChoice == "back" {
		return runTemplateMenu()
	}

	// Select brain type
	_, brainStr, err := ui.RunSelect("Select brain type", []ui.SelectItem{
		{Label: "flip", Value: "flip"},
		{Label: "obsidian", Value: "obsidian"},
		{Label: "logseq", Value: "logseq"},
		{Label: "dendron", Value: "dendron"},
		{Label: "foam", Value: "foam"},
	}, 0)
	if err != nil {
		return runTemplateMenu()
	}

	brainType := parseBrainType(brainStr)

	if scopeChoice == "all" {
		// Reset all templates for this brain type
		yes, err := ui.RunConfirm(fmt.Sprintf("Reset ALL templates for %s to defaults", brainStr), false)
		if err != nil || !yes {
			fmt.Println("\n❌ Cancelled")
			return runTemplateMenu()
		}

		if err := templates.ResetBrainTemplates(brainType); err != nil {
			fmt.Printf("\n❌ Error resetting templates: %v\n", err)
		} else {
			fmt.Printf("\n✅ All %s templates reset to defaults\n", brainStr)
		}
		return runTemplateMenu()
	}

	// Reset single template
	_, templateStr, err := ui.RunSelect(fmt.Sprintf("Select template type for %s", brainStr), []ui.SelectItem{
		{Label: "note", Value: "note"},
		{Label: "meeting", Value: "meeting"},
		{Label: "journal", Value: "journal"},
		{Label: "task", Value: "task"},
		{Label: "prompt", Value: "prompt"},
	}, 0)
	if err != nil {
		return runTemplateMenu()
	}

	templateType := parseTemplateType(templateStr)

	// Confirm reset
	yes, err := ui.RunConfirm(fmt.Sprintf("Reset %s template for %s to default", templateStr, brainStr), false)
	if err != nil || !yes {
		fmt.Println("\n❌ Cancelled")
		return runTemplateMenu()
	}

	if err := templates.ResetTemplate(brainType, templateType); err != nil {
		fmt.Printf("\n❌ Error resetting template: %v\n", err)
	} else {
		fmt.Printf("\n✅ %s template for %s reset to default\n", templateStr, brainStr)
	}
	return runTemplateMenu()
}

func showTemplateDirectoryFlow() error {
	templateDir, err := templates.GetTemplateDir()
	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		return runTemplateMenu()
	}

	fmt.Printf("\n📂 Template Directory\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\nLocation: %s\n\n", templateDir)
	fmt.Println("Directory structure:")
	fmt.Println("  templates/")
	fmt.Println("    ├── flip/      (Flip brain templates)")
	fmt.Println("    ├── obsidian/  (Obsidian vault templates)")
	fmt.Println("    ├── logseq/    (Logseq graph templates)")
	fmt.Println("    ├── dendron/   (Dendron workspace templates)")
	fmt.Println("    └── foam/      (Foam brain templates)")
	fmt.Println()
	fmt.Println("Each directory contains:")
	fmt.Println("  - note.md      (Regular notes)")
	fmt.Println("  - meeting.md   (Meeting notes)")
	fmt.Println("  - journal.md   (Daily journal/notes)")
	fmt.Println("  - task.md      (Task notes)")
	fmt.Println("  - prompt.md    (Prompt notes)")
	fmt.Println()

	// Offer to open in file browser
	yes, err := ui.RunConfirm("Open template directory in file manager?", false)
	if err != nil {
		return runTemplateMenu()
	}

	if yes {
		if err := openInFileManager(templateDir); err != nil {
			fmt.Printf("\n❌ Error opening directory: %v\n", err)
		}
	}

	return runTemplateMenu()
}
