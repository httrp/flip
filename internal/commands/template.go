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
	fmt.Println("\n🔧 Template Management")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

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

	prompt := promptui.Select{
		Label: "What would you like to do?",
		Items: menuItems,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "▸ {{ .Label | cyan }} - {{ .Description }}",
			Inactive: "  {{ .Label }} - {{ .Description }}",
			Selected: "{{ .Label | green }}",
		},
		Size:     10,
		HideHelp: true,
	}

	idx, _, err := prompt.Run()
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
	brainTypePrompt := promptui.Select{
		Label: "Select brain type",
		Items: []string{"flip", "obsidian", "logseq", "dendron"},
	}

	brainIdx, brainStr, err := brainTypePrompt.Run()
	if err != nil {
		return runTemplateMenu()
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
		Items: []string{"note", "meeting", "journal", "task"},
	}

	templateIdx, _, err := templatePrompt.Run()
	if err != nil {
		return runTemplateMenu()
	}

	var templateType templates.TemplateType
	switch templateIdx {
	case 0:
		templateType = templates.TemplateTypeNote
	case 1:
		templateType = templates.TemplateTypeMeeting
	case 2:
		templateType = templates.TemplateTypeJournal
	case 3:
		templateType = templates.TemplateTypeTask
	}

	// Get template path
	templatePath, err := templates.GetTemplatePath(brainType, templateType)
	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		return runTemplateMenu()
	}

	// Check if template exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		fmt.Printf("\n❌ Template not found: %s\n", templatePath)
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
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
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
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

	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
	return runTemplateMenu()
}

func viewTemplateFlow() error {
	// Select brain type
	brainTypePrompt := promptui.Select{
		Label: "Select brain type",
		Items: []string{"flip", "obsidian", "logseq", "dendron", "foam"},
	}

	brainIdx, brainStr, err := brainTypePrompt.Run()
	if err != nil {
		return runTemplateMenu()
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
	case 4:
		brainType = brain.BrainTypeFoam
	}

	// Select template type
	templatePrompt := promptui.Select{
		Label: fmt.Sprintf("Select template type for %s", brainStr),
		Items: []string{"note", "meeting", "journal", "task"},
	}

	templateIdx, _, err := templatePrompt.Run()
	if err != nil {
		return runTemplateMenu()
	}

	var templateType templates.TemplateType
	switch templateIdx {
	case 0:
		templateType = templates.TemplateTypeNote
	case 1:
		templateType = templates.TemplateTypeMeeting
	case 2:
		templateType = templates.TemplateTypeJournal
	case 3:
		templateType = templates.TemplateTypeTask
	}

	// Load and display template
	content, err := templates.Load(brainType, templateType)
	if err != nil {
		fmt.Printf("\n❌ Error loading template: %v\n", err)
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		return runTemplateMenu()
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Template: %s - %s\n", brainStr, templateType)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(content)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("\nAvailable placeholders: {{title}}, {{date}}, {{time}}, {{tags}}, {{participants}}, {{id}}, {{created}}, {{updated}}")
	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
	return runTemplateMenu()
}

func resetTemplateFlow() error {
	fmt.Println("\n🔄 Reset Template")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Option to reset all or single template
	scopePrompt := promptui.Select{
		Label: "What would you like to reset?",
		Items: []string{"Single template", "All templates for a brain type", "◀️  Back"},
	}

	scopeIdx, _, err := scopePrompt.Run()
	if err != nil || scopeIdx == 2 {
		return runTemplateMenu()
	}

	// Select brain type
	brainTypePrompt := promptui.Select{
		Label: "Select brain type",
		Items: []string{"flip", "obsidian", "logseq", "dendron", "foam"},
	}

	brainIdx, brainStr, err := brainTypePrompt.Run()
	if err != nil {
		return runTemplateMenu()
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
	case 4:
		brainType = brain.BrainTypeFoam
	}

	if scopeIdx == 1 {
		// Reset all templates for this brain type
		confirmPrompt := promptui.Prompt{
			Label:     fmt.Sprintf("Reset ALL templates for %s to defaults", brainStr),
			IsConfirm: true,
		}

		_, err = confirmPrompt.Run()
		if err != nil {
			fmt.Println("\n❌ Cancelled")
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
			return runTemplateMenu()
		}

		if err := templates.ResetBrainTemplates(brainType); err != nil {
			fmt.Printf("\n❌ Error resetting templates: %v\n", err)
		} else {
			fmt.Printf("\n✅ All %s templates reset to defaults\n", brainStr)
		}
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		return runTemplateMenu()
	}

	// Reset single template
	templatePrompt := promptui.Select{
		Label: fmt.Sprintf("Select template type for %s", brainStr),
		Items: []string{"note", "meeting", "journal", "task"},
	}

	templateIdx, templateStr, err := templatePrompt.Run()
	if err != nil {
		return runTemplateMenu()
	}

	var templateType templates.TemplateType
	switch templateIdx {
	case 0:
		templateType = templates.TemplateTypeNote
	case 1:
		templateType = templates.TemplateTypeMeeting
	case 2:
		templateType = templates.TemplateTypeJournal
	case 3:
		templateType = templates.TemplateTypeTask
	}

	// Confirm reset
	confirmPrompt := promptui.Prompt{
		Label:     fmt.Sprintf("Reset %s template for %s to default", templateStr, brainStr),
		IsConfirm: true,
	}

	_, err = confirmPrompt.Run()
	if err != nil {
		fmt.Println("\n❌ Cancelled")
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		return runTemplateMenu()
	}

	if err := templates.ResetTemplate(brainType, templateType); err != nil {
		fmt.Printf("\n❌ Error resetting template: %v\n", err)
	} else {
		fmt.Printf("\n✅ %s template for %s reset to default\n", templateStr, brainStr)
	}
	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
	return runTemplateMenu()
}

func showTemplateDirectoryFlow() error {
	templateDir, err := templates.GetTemplateDir()
	if err != nil {
		fmt.Printf("\n❌ Error: %v\n", err)
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
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
	fmt.Println()

	// Offer to open in file browser
	openPrompt := promptui.Select{
		Label: "Open template directory in file manager?",
		Items: []string{"Yes", "No"},
	}

	idx, _, err := openPrompt.Run()
	if err != nil {
		return runTemplateMenu()
	}

	if idx == 0 {
		if err := openInFileManager(templateDir); err != nil {
			fmt.Printf("\n❌ Error opening directory: %v\n", err)
			fmt.Println("\nPress Enter to continue...")
			fmt.Scanln()
		}
	}

	return runTemplateMenu()
}
