package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// NewDefinitionsCommand creates the definitions command
func NewDefinitionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "definitions",
		Aliases: []string{"def", "defs"},
		Short:   "Manage task definitions (organizations, projects, contexts)",
		Long:    "View and manage definitions for organizations, projects, and contexts used in tasks.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsMenu()
		},
	}

	cmd.AddCommand(newDefinitionsListCommand())
	cmd.AddCommand(newDefinitionsAddCommand())
	cmd.AddCommand(newDefinitionsRemoveCommand())
	cmd.AddCommand(newDefinitionsScanCommand())
	cmd.AddCommand(newDefinitionsPathCommand())

	return cmd
}

// runDefinitionsMenu shows an interactive menu for managing definitions
func runDefinitionsMenu() error {
	for {
		fmt.Println("\n📚 Task Definitions")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		items := []string{
			lang.GetText("menu.definitions.organizations_label"),
			lang.GetText("menu.definitions.projects_label"),
			lang.GetText("menu.definitions.contexts_label"),
			lang.GetText("menu.definitions.people_label"),
			lang.GetText("menu.definitions.scan_label"),
			lang.GetText("menu.definitions.export_label"),
			lang.GetText("menu.definitions.file_label"),
			lang.GetText("menu.definitions.back_label"),
		}

		selector := promptui.Select{
			Label: "What would you like to do?",
			Items: items,
			Size:  10,
		}

		idx, _, err := selector.Run()
		if err != nil {
			return err
		}

		switch idx {
		case 0:
			if err := runDefinitionsBrowseOrganizations(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 1:
			if err := runDefinitionsBrowseProjects(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 2:
			if err := runDefinitionsBrowseContexts(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 3:
			if err := runDefinitionsBrowsePeople(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 4:
			if err := runDefinitionsScan(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			fmt.Println(lang.GetText("prompts.continue"))
			fmt.Scanln()
		case 5:
			if err := runDefinitionsExportToNote(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			fmt.Println(lang.GetText("prompts.continue"))
			fmt.Scanln()
		case 6:
			if err := runDefinitionsOpenFile(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			fmt.Println(lang.GetText("prompts.continue"))
			fmt.Scanln()
		case 7:
			return nil
		}
	}
}

// runDefinitionsBrowseOrganizations shows and manages organizations
func runDefinitionsBrowseOrganizations() error {
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	if len(defs.Organizations) == 0 {
		fmt.Println("\n🏢 No organizations defined yet.")
		fmt.Println("\nWould you like to add one?")

		prompt := promptui.Select{
			Label: "Add Organization",
			Items: []string{"Yes", "No"},
		}

		idx, _, err := prompt.Run()
		if err != nil || idx == 1 {
			return nil
		}

		return runDefinitionsAddOrg()
	}

	// Build list of organizations with "Add New" option
	items := make([]string, len(defs.Organizations)+1)
	for i, org := range defs.Organizations {
		items[i] = fmt.Sprintf("🏢 [%s] %s", org.Abbreviation, org.Name)
	}
	items[len(items)-1] = "➕ Add New Organization"

	selector := promptui.Select{
		Label: fmt.Sprintf("Organizations (%d total)", len(defs.Organizations)),
		Items: items,
		Size:  15,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return nil
	}

	// Check if "Add New" was selected
	if idx == len(items)-1 {
		return runDefinitionsAddOrg()
	}

	// Show details of selected organization
	org := defs.Organizations[idx]
	fmt.Println("\n" + strings.Repeat("━", 60))
	fmt.Printf("Organization: [%s] %s\n", org.Abbreviation, org.Name)
	fmt.Println(strings.Repeat("━", 60))
	if org.Description != "" {
		fmt.Printf("Description: %s\n", org.Description)
	}
	fmt.Println(strings.Repeat("━", 60))

	fmt.Println(lang.GetText("prompts.continue"))
	fmt.Scanln()
	return nil
}

// runDefinitionsBrowseProjects shows and manages projects
func runDefinitionsBrowseProjects() error {
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	if len(defs.Projects) == 0 {
		fmt.Println("\n📁 No projects defined yet.")
		fmt.Println("\nWould you like to add one?")

		prompt := promptui.Select{
			Label: "Add Project",
			Items: []string{"Yes", "No"},
		}

		idx, _, err := prompt.Run()
		if err != nil || idx == 1 {
			return nil
		}

		return runDefinitionsAddProject()
	}

	// Build list of projects with "Add New" option
	items := make([]string, len(defs.Projects)+1)
	for i, proj := range defs.Projects {
		if proj.Organization != "" {
			items[i] = fmt.Sprintf("📁 [%s] %s (%s)", proj.Abbreviation, proj.Name, proj.Organization)
		} else {
			items[i] = fmt.Sprintf("📁 [%s] %s", proj.Abbreviation, proj.Name)
		}
	}
	items[len(items)-1] = "➕ Add New Project"

	selector := promptui.Select{
		Label: fmt.Sprintf("Projects (%d total)", len(defs.Projects)),
		Items: items,
		Size:  15,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return nil
	}

	// Check if "Add New" was selected
	if idx == len(items)-1 {
		return runDefinitionsAddProject()
	}

	// Show details of selected project
	proj := defs.Projects[idx]
	fmt.Println("\n" + strings.Repeat("━", 60))
	fmt.Printf("Project: [%s] %s\n", proj.Abbreviation, proj.Name)
	fmt.Println(strings.Repeat("━", 60))
	if proj.Organization != "" {
		fmt.Printf("Organization: %s\n", proj.Organization)
	}
	if proj.Description != "" {
		fmt.Printf("Description: %s\n", proj.Description)
	}
	fmt.Println(strings.Repeat("━", 60))

	fmt.Println(lang.GetText("prompts.continue"))
	fmt.Scanln()
	return nil
}

// runDefinitionsBrowseContexts shows and manages contexts
func runDefinitionsBrowseContexts() error {
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	if len(defs.Contexts) == 0 {
		fmt.Println("\n📂 No contexts defined yet.")
		fmt.Println("\nWould you like to add one?")

		prompt := promptui.Select{
			Label: "Add Context",
			Items: []string{"Yes", "No"},
		}

		idx, _, err := prompt.Run()
		if err != nil || idx == 1 {
			return nil
		}

		return runDefinitionsAddContext()
	}

	// Build list of contexts with "Add New" option
	items := make([]string, len(defs.Contexts)+1)
	for i, ctx := range defs.Contexts {
		items[i] = fmt.Sprintf("📂 [%s] %s", ctx.Abbreviation, ctx.Name)
	}
	items[len(items)-1] = "➕ Add New Context"

	selector := promptui.Select{
		Label: fmt.Sprintf("Contexts (%d total)", len(defs.Contexts)),
		Items: items,
		Size:  15,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return nil
	}

	// Check if "Add New" was selected
	if idx == len(items)-1 {
		return runDefinitionsAddContext()
	}

	// Show details of selected context
	ctx := defs.Contexts[idx]
	fmt.Println("\n" + strings.Repeat("━", 60))
	fmt.Printf("Context: [%s] %s\n", ctx.Abbreviation, ctx.Name)
	fmt.Println(strings.Repeat("━", 60))
	if ctx.Description != "" {
		fmt.Printf("Description: %s\n", ctx.Description)
	}
	fmt.Println(strings.Repeat("━", 60))

	fmt.Println(lang.GetText("prompts.continue"))
	fmt.Scanln()
	return nil
}

// runDefinitionsBrowsePeople shows and manages people
func runDefinitionsBrowsePeople() error {
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	if len(defs.People) == 0 {
		fmt.Println("\n👥 No people defined yet.")
		fmt.Println("\nWould you like to add someone?")

		prompt := promptui.Select{
			Label: "Add Person",
			Items: []string{"Yes", "No"},
		}

		idx, _, err := prompt.Run()
		if err != nil || idx == 1 {
			return nil
		}

		return runDefinitionsAddPerson()
	}

	// Build list of people with "Add New" option
	items := make([]string, len(defs.People)+1)
	for i, person := range defs.People {
		if person.Organization != "" {
			items[i] = fmt.Sprintf("👤 [%s] %s (%s)", person.Abbreviation, person.Name, person.Organization)
		} else {
			items[i] = fmt.Sprintf("👤 [%s] %s", person.Abbreviation, person.Name)
		}
	}
	items[len(items)-1] = "➕ Add New Person"

	selector := promptui.Select{
		Label: fmt.Sprintf("People (%d total)", len(defs.People)),
		Items: items,
		Size:  15,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return nil
	}

	// Check if "Add New" was selected
	if idx == len(items)-1 {
		return runDefinitionsAddPerson()
	}

	// Show details of selected person
	person := defs.People[idx]
	fmt.Println("\n" + strings.Repeat("━", 60))
	fmt.Printf("Person: [%s] %s\n", person.Abbreviation, person.Name)
	fmt.Println(strings.Repeat("━", 60))
	if person.Organization != "" {
		fmt.Printf("Organization: %s\n", person.Organization)
	}
	if person.OrgCode != "" {
		fmt.Printf("Org Code: %s\n", person.OrgCode)
	}
	if person.Role != "" {
		fmt.Printf("Role: %s\n", person.Role)
	}
	if person.Email != "" {
		fmt.Printf("Email: %s\n", person.Email)
	}
	fmt.Println(strings.Repeat("━", 60))

	fmt.Println(lang.GetText("prompts.continue"))
	fmt.Scanln()
	return nil
}

// runDefinitionsExportToNote exports definitions to a new note
func runDefinitionsExportToNote() error {
	fmt.Println("\n📝 Export Definitions to Note")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	// Choose what to export
	exportOptions := []string{
		"All definitions",
		"Definitions by organization",
		"All projects",
		"All people",
		"All contexts",
		"Cancel",
	}

	selector := promptui.Select{
		Label: "What would you like to export?",
		Items: exportOptions,
		Size:  10,
	}

	idx, _, err := selector.Run()
	if err != nil || idx == len(exportOptions)-1 {
		return nil
	}

	var content strings.Builder
	var title string

	switch idx {
	case 0: // All definitions
		title = "All Definitions"
		content.WriteString("# All Definitions\n\n")
		content.WriteString(fmt.Sprintf("*Exported: %s*\n\n", time.Now().Format("2006-01-02 15:04")))

		// Organizations
		content.WriteString("## Organizations\n\n")
		for _, org := range defs.Organizations {
			content.WriteString(fmt.Sprintf("### [%s] %s\n\n", org.Abbreviation, org.Name))
			if org.Description != "" {
				content.WriteString(fmt.Sprintf("%s\n\n", org.Description))
			}
		}

		// Projects
		content.WriteString("## Projects\n\n")
		for _, proj := range defs.Projects {
			content.WriteString(fmt.Sprintf("- **[%s] %s**", proj.Abbreviation, proj.Name))
			if proj.Organization != "" {
				content.WriteString(fmt.Sprintf(" (Organization: %s)", proj.Organization))
			}
			if proj.Description != "" {
				content.WriteString(fmt.Sprintf("\n  - %s", proj.Description))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")

		// Contexts
		content.WriteString("## Contexts\n\n")
		for _, ctx := range defs.Contexts {
			content.WriteString(fmt.Sprintf("- **[%s] %s**", ctx.Abbreviation, ctx.Name))
			if ctx.Description != "" {
				content.WriteString(fmt.Sprintf(": %s", ctx.Description))
			}
			content.WriteString("\n")
		}
		content.WriteString("\n")

		// People
		content.WriteString("## People\n\n")
		for _, person := range defs.People {
			content.WriteString(fmt.Sprintf("### [%s] %s\n\n", person.Abbreviation, person.Name))
			if person.Organization != "" {
				content.WriteString(fmt.Sprintf("- Organization: %s\n", person.Organization))
			}
			if person.OrgCode != "" {
				content.WriteString(fmt.Sprintf("- Org Code: %s\n", person.OrgCode))
			}
			if person.Role != "" {
				content.WriteString(fmt.Sprintf("- Role: %s\n", person.Role))
			}
			if person.Email != "" {
				content.WriteString(fmt.Sprintf("- Email: %s\n", person.Email))
			}
			content.WriteString("\n")
		}

	case 1: // By organization
		// Select organization
		if len(defs.Organizations) == 0 {
			fmt.Println("⚠️  No organizations found.")
			return nil
		}

		orgItems := make([]string, len(defs.Organizations))
		for i, org := range defs.Organizations {
			orgItems[i] = fmt.Sprintf("[%s] %s", org.Abbreviation, org.Name)
		}

		orgSelector := promptui.Select{
			Label: "Select Organization",
			Items: orgItems,
			Size:  10,
		}

		orgIdx, _, err := orgSelector.Run()
		if err != nil {
			return nil
		}

		org := defs.Organizations[orgIdx]
		title = fmt.Sprintf("Definitions - %s", org.Name)

		content.WriteString(fmt.Sprintf("# Definitions: %s [%s]\n\n", org.Name, org.Abbreviation))
		content.WriteString(fmt.Sprintf("*Exported: %s*\n\n", time.Now().Format("2006-01-02 15:04")))

		if org.Description != "" {
			content.WriteString(fmt.Sprintf("**Description:** %s\n\n", org.Description))
		}

		// Projects for this organization
		content.WriteString("## Projects\n\n")
		foundProjects := false
		for _, proj := range defs.Projects {
			if proj.Organization == org.Abbreviation {
				content.WriteString(fmt.Sprintf("- **[%s] %s**", proj.Abbreviation, proj.Name))
				if proj.Description != "" {
					content.WriteString(fmt.Sprintf(": %s", proj.Description))
				}
				content.WriteString("\n")
				foundProjects = true
			}
		}
		if !foundProjects {
			content.WriteString("*No projects defined*\n")
		}
		content.WriteString("\n")

		// People for this organization
		content.WriteString("## People\n\n")
		foundPeople := false
		for _, person := range defs.People {
			if person.Organization == org.Abbreviation {
				content.WriteString(fmt.Sprintf("### [%s] %s\n\n", person.Abbreviation, person.Name))
				if person.OrgCode != "" {
					content.WriteString(fmt.Sprintf("- Org Code: %s\n", person.OrgCode))
				}
				if person.Role != "" {
					content.WriteString(fmt.Sprintf("- Role: %s\n", person.Role))
				}
				if person.Email != "" {
					content.WriteString(fmt.Sprintf("- Email: %s\n", person.Email))
				}
				content.WriteString("\n")
				foundPeople = true
			}
		}
		if !foundPeople {
			content.WriteString("*No people defined*\n")
		}

	case 2: // All projects
		title = "All Projects"
		content.WriteString("# All Projects\n\n")
		content.WriteString(fmt.Sprintf("*Exported: %s*\n\n", time.Now().Format("2006-01-02 15:04")))

		for _, proj := range defs.Projects {
			content.WriteString(fmt.Sprintf("## [%s] %s\n\n", proj.Abbreviation, proj.Name))
			if proj.Organization != "" {
				content.WriteString(fmt.Sprintf("**Organization:** %s\n\n", proj.Organization))
			}
			if proj.Description != "" {
				content.WriteString(fmt.Sprintf("%s\n\n", proj.Description))
			}
		}

	case 3: // All people
		title = "All People"
		content.WriteString("# All People\n\n")
		content.WriteString(fmt.Sprintf("*Exported: %s*\n\n", time.Now().Format("2006-01-02 15:04")))

		for _, person := range defs.People {
			content.WriteString(fmt.Sprintf("## [%s] %s\n\n", person.Abbreviation, person.Name))
			if person.Organization != "" {
				content.WriteString(fmt.Sprintf("- **Organization:** %s\n", person.Organization))
			}
			if person.OrgCode != "" {
				content.WriteString(fmt.Sprintf("- **Org Code:** %s\n", person.OrgCode))
			}
			if person.Role != "" {
				content.WriteString(fmt.Sprintf("- **Role:** %s\n", person.Role))
			}
			if person.Email != "" {
				content.WriteString(fmt.Sprintf("- **Email:** %s\n", person.Email))
			}
			content.WriteString("\n")
		}

	case 4: // All contexts
		title = "All Contexts"
		content.WriteString("# All Contexts\n\n")
		content.WriteString(fmt.Sprintf("*Exported: %s*\n\n", time.Now().Format("2006-01-02 15:04")))

		for _, ctx := range defs.Contexts {
			content.WriteString(fmt.Sprintf("## [%s] %s\n\n", ctx.Abbreviation, ctx.Name))
			if ctx.Description != "" {
				content.WriteString(fmt.Sprintf("%s\n\n", ctx.Description))
			}
		}
	}

	// Get active brain for creating the note
	brain, err := getActiveBrain()
	if err != nil {
		return fmt.Errorf("no active brain: %w", err)
	}

	// Create filename
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-definitions-%s.md", timestamp, strings.ToLower(strings.ReplaceAll(title, " ", "-")))
	notePath := filepath.Join(brain.Path, "notes", filename)

	// Ensure notes directory exists
	notesDir := filepath.Join(brain.Path, "notes")
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		return fmt.Errorf("failed to create notes directory: %w", err)
	}

	// Write note
	if err := os.WriteFile(notePath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write note: %w", err)
	}

	fmt.Printf("\n✅ Note created: %s\n", notePath)

	// Ask if user wants to open it
	openPrompt := promptui.Select{
		Label: "Open note in editor?",
		Items: []string{"Yes", "No"},
	}

	openIdx, _, err := openPrompt.Run()
	if err == nil && openIdx == 0 {
		// Open in editor (cross-platform)
		if err := openInEditor(notePath); err != nil {
			fmt.Printf("\n⚠️  Could not open editor: %v\n", err)
			fmt.Printf("   File path: %s\n", notePath)
		}
	}

	return nil
}

// runDefinitionsOpenFile opens the definitions file in an editor
func runDefinitionsOpenFile() error {
	path, err := getTaskDefinitionsPath()
	if err != nil {
		return fmt.Errorf("failed to get definitions path: %w", err)
	}

	fmt.Printf("\n📂 Definitions file: %s\n", path)

	// Open in editor (cross-platform)
	if err := openInEditor(path); err != nil {
		fmt.Printf("\n⚠️  Could not open editor: %v\n", err)
		fmt.Printf("   File path: %s\n", path)
		return nil
	}

	fmt.Printf("✅ Opening in editor...\n")
	return nil
}

// newDefinitionsListCommand creates the list subcommand
func newDefinitionsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all definitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsList()
		},
	}
}

// newDefinitionsAddCommand creates the add subcommand
func newDefinitionsAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [type]",
		Short: "Add a new definition",
		Long:  "Add a new organization, project, or context definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("please specify type: org, project, or context")
			}

			switch args[0] {
			case "org", "organization":
				return runDefinitionsAddOrg()
			case "proj", "project":
				return runDefinitionsAddProject()
			case "ctx", "context":
				return runDefinitionsAddContext()
			default:
				return fmt.Errorf("unknown type: %s (use: org, project, or context)", args[0])
			}
		},
	}
	return cmd
}

// newDefinitionsRemoveCommand creates the remove subcommand
func newDefinitionsRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove a definition",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsRemove()
		},
	}
}

// newDefinitionsScanCommand creates the scan subcommand
func newDefinitionsScanCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Scan tasks for organizations, projects, and contexts",
		Long:  "Scan all tasks in the active brain and suggest adding found organizations, projects, and contexts to definitions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsScan()
		},
	}
}

// newDefinitionsPathCommand creates the path subcommand
func newDefinitionsPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show definitions file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDefinitionsPath()
		},
	}
}

// runDefinitionsList lists all definitions
func runDefinitionsList() error {
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	// Show source information
	sourceInfo := ""
	switch defs.Source {
	case "brain":
		brain, _ := getActiveBrain()
		if brain != nil {
			sourceInfo = fmt.Sprintf(" 📂 Source: Brain definitions (%s/definitions/)", brain.Name)
		}
	case "simple":
		path, _ := getTaskDefinitionsPath()
		sourceInfo = fmt.Sprintf(" 📂 Source: %s", path)
	case "default":
		sourceInfo = " 📂 Source: Default values (no definitions file found)"
	}

	if sourceInfo != "" {
		fmt.Println(sourceInfo)
	}

	fmt.Println("\n🏢 Organizations")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if len(defs.Organizations) == 0 {
		fmt.Println("  (none)")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		// Show extra columns if brain format
		if defs.Source == "brain" {
			fmt.Fprintln(w, "  ABBR\tNAME\tTYPE\tSTATUS\tDESCRIPTION")
			fmt.Fprintln(w, "  ────\t────\t────\t──────\t───────────")
			for _, org := range defs.Organizations {
				typ := org.Type
				if typ == "" {
					typ = "-"
				}
				status := org.Status
				if status == "" {
					status = "-"
				}
				desc := org.Description
				if desc == "" {
					desc = "-"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n", org.Abbreviation, org.Name, typ, status, desc)
			}
		} else {
			fmt.Fprintln(w, "  ABBR\tNAME\tDESCRIPTION")
			fmt.Fprintln(w, "  ────\t────\t───────────")
			for _, org := range defs.Organizations {
				desc := org.Description
				if desc == "" {
					desc = "-"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\n", org.Abbreviation, org.Name, desc)
			}
		}
		w.Flush()
	}

	fmt.Println("\n📁 Projects")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if len(defs.Projects) == 0 {
		fmt.Println("  (none)")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		// Show extra columns if brain format
		if defs.Source == "brain" {
			fmt.Fprintln(w, "  ABBR\tNAME\tORG\tSTATUS\tDESCRIPTION")
			fmt.Fprintln(w, "  ────\t────\t───\t──────\t───────────")
			for _, proj := range defs.Projects {
				org := proj.Organization
				if org == "" {
					org = "-"
				}
				status := proj.Status
				if status == "" {
					status = "-"
				}
				desc := proj.Description
				if desc == "" {
					desc = "-"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n", proj.Abbreviation, proj.Name, org, status, desc)
			}
		} else {
			fmt.Fprintln(w, "  ABBR\tNAME\tORG\tDESCRIPTION")
			fmt.Fprintln(w, "  ────\t────\t───\t───────────")
			for _, proj := range defs.Projects {
				org := proj.Organization
				if org == "" {
					org = "-"
				}
				desc := proj.Description
				if desc == "" {
					desc = "-"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", proj.Abbreviation, proj.Name, org, desc)
			}
		}
		w.Flush()
	}

	fmt.Println("\n🏷️  Contexts")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if len(defs.Contexts) == 0 {
		fmt.Println("  (none)")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		// Show extra columns if brain format
		if defs.Source == "brain" {
			fmt.Fprintln(w, "  ABBR\tNAME\tORG\tSTATUS\tDESCRIPTION")
			fmt.Fprintln(w, "  ────\t────\t───\t──────\t───────────")
			for _, ctx := range defs.Contexts {
				org := ctx.Organization
				if org == "" {
					org = "-"
				}
				status := ctx.Status
				if status == "" {
					status = "-"
				}
				desc := ctx.Description
				if desc == "" {
					desc = "-"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\n", ctx.Abbreviation, ctx.Name, org, status, desc)
			}
		} else {
			fmt.Fprintln(w, "  ABBR\tNAME\tDESCRIPTION")
			fmt.Fprintln(w, "  ────\t────\t───────────")
			for _, ctx := range defs.Contexts {
				desc := ctx.Description
				if desc == "" {
					desc = "-"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\n", ctx.Abbreviation, ctx.Name, desc)
			}
		}
		w.Flush()
	}

	// Add People section
	fmt.Println("\n👥 People")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if len(defs.People) == 0 {
		fmt.Println("  (none)")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		// Show extra columns if brain format
		if defs.Source == "brain" {
			fmt.Fprintln(w, "  ABBR\tNAME\tORG\tORG-CODE\tROLE\tEMAIL")
			fmt.Fprintln(w, "  ────\t────\t───\t────────\t────\t─────")
			for _, person := range defs.People {
				org := person.Organization
				if org == "" {
					org = "-"
				}
				orgCode := person.OrgCode
				if orgCode == "" {
					orgCode = "-"
				}
				role := person.Role
				if role == "" {
					role = "-"
				}
				email := person.Email
				if email == "" {
					email = "-"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n", person.Abbreviation, person.Name, org, orgCode, role, email)
			}
		} else {
			fmt.Fprintln(w, "  ABBR\tNAME\tORG\tROLE")
			fmt.Fprintln(w, "  ────\t────\t───\t────")
			for _, person := range defs.People {
				org := person.Organization
				if org == "" {
					org = "-"
				}
				role := person.Role
				if role == "" {
					role = "-"
				}
				fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", person.Abbreviation, person.Name, org, role)
			}
		}
		w.Flush()
	}
	fmt.Println()

	return nil
}

// runDefinitionsAddOrg adds a new organization
func runDefinitionsAddOrg() error {
	fmt.Println("\n➕ Add Organization")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("💡 Press Ctrl+C to cancel at any time")

	// Prompt for abbreviation
	abbrPrompt := promptui.Prompt{
		Label: "Abbreviation (e.g., WORK, PERS)",
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Prompt for name
	namePrompt := promptui.Prompt{
		Label: "Full Name",
	}
	name, err := namePrompt.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}
	name = strings.TrimSpace(name)

	// Prompt for description (optional)
	descPrompt := promptui.Prompt{
		Label:   "Description (optional)",
		Default: "",
	}
	desc, _ := descPrompt.Run()
	desc = strings.TrimSpace(desc)

	// Load, add, save
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	if err := defs.addOrganization(name, abbr, desc); err != nil {
		return err
	}

	if err := saveTaskDefinitions(defs); err != nil {
		return fmt.Errorf("failed to save definitions: %w", err)
	}

	fmt.Printf("\n✅ Organization '[%s] %s' added successfully\n", abbr, name)

	// Ask if user wants to open the file
	if err := promptOpenDefinitionsFile(); err != nil {
		return err
	}

	return nil
}

// runDefinitionsAddProject adds a new project
func runDefinitionsAddProject() error {
	fmt.Println("\n➕ Add Project")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("💡 Press Ctrl+C to cancel at any time")

	// Load definitions first to get organizations
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	// Select organization (required)
	if len(defs.Organizations) == 0 {
		fmt.Println("⚠️  No organizations found. Please create an organization first.")
		fmt.Println("   Use: flip definitions add org")
		return nil
	}

	orgItems := make([]string, len(defs.Organizations))
	for i, org := range defs.Organizations {
		orgItems[i] = fmt.Sprintf("[%s] %s", org.Abbreviation, org.Name)
	}

	orgSelector := promptui.Select{
		Label: "Select Organization",
		Items: orgItems,
		Size:  10,
	}

	orgIdx, _, err := orgSelector.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}

	selectedOrg := defs.Organizations[orgIdx].Abbreviation

	// Prompt for abbreviation
	abbrPrompt := promptui.Prompt{
		Label: "Abbreviation (e.g., FLIP, BRAIN)",
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Prompt for name
	namePrompt := promptui.Prompt{
		Label: "Full Name",
	}
	name, err := namePrompt.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}
	name = strings.TrimSpace(name)

	// Prompt for description (optional)
	descPrompt := promptui.Prompt{
		Label:   "Description (optional)",
		Default: "",
	}
	desc, _ := descPrompt.Run()
	desc = strings.TrimSpace(desc)

	// Add project
	if err := defs.addProject(name, abbr, selectedOrg, desc); err != nil {
		return err
	}

	if err := saveTaskDefinitions(defs); err != nil {
		return fmt.Errorf("failed to save definitions: %w", err)
	}

	fmt.Printf("\n✅ Project '[%s] %s' added to organization '%s'\n", abbr, name, selectedOrg)

	// Ask if user wants to open the file
	if err := promptOpenDefinitionsFile(); err != nil {
		return err
	}

	return nil
}

// runDefinitionsAddContext adds a new context
func runDefinitionsAddContext() error {
	fmt.Println("\n➕ Add Context")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("💡 Press Ctrl+C to cancel at any time")

	// Load definitions first
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	// Prompt for name first to check for similar contexts
	namePrompt := promptui.Prompt{
		Label: "Context Name",
	}
	name, err := namePrompt.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}
	name = strings.TrimSpace(name)

	// Check for similar contexts
	similar := findSimilarContexts(name, defs)
	if len(similar) > 0 {
		fmt.Println("\n⚠️  Found similar contexts:")
		for _, s := range similar {
			fmt.Printf("   • %s\n", s)
		}
		fmt.Println()

		actions := []string{
			"Create new context anyway",
			"View/edit existing context",
			"Cancel",
		}

		actionPrompt := promptui.Select{
			Label: "What would you like to do?",
			Items: actions,
			Size:  5,
		}

		idx, _, err := actionPrompt.Run()
		if err != nil {
			fmt.Println("\n⚠️  Cancelled")
			return nil
		}

		switch idx {
		case 1: // View/edit
			// TODO: Implement context editing
			fmt.Println("\n💡 Context editing not yet implemented. Please use the definitions file directly.")
			fmt.Println("   Use: flip definitions file")
			return nil
		case 2: // Cancel
			fmt.Println("\n⚠️  Cancelled")
			return nil
		}
		// case 0: Continue with creation
	}

	// Prompt for abbreviation
	abbrPrompt := promptui.Prompt{
		Label: "Abbreviation (e.g., MTG, EMAIL)",
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Prompt for description (optional)
	descPrompt := promptui.Prompt{
		Label:   "Description (optional)",
		Default: "",
	}
	desc, _ := descPrompt.Run()
	desc = strings.TrimSpace(desc)

	// Add context
	if err := defs.addContext(name, abbr, desc); err != nil {
		return err
	}

	if err := saveTaskDefinitions(defs); err != nil {
		return fmt.Errorf("failed to save definitions: %w", err)
	}

	fmt.Printf("\n✅ Context '[%s] %s' added successfully\n", abbr, name)

	// Ask if user wants to open the file
	if err := promptOpenDefinitionsFile(); err != nil {
		return err
	}

	return nil
}

// runDefinitionsAddPerson adds a new person
func runDefinitionsAddPerson() error {
	fmt.Println("\n➕ Add Person")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("💡 Press Ctrl+C to cancel at any time")

	// Load definitions first
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	// Select organization (required)
	if len(defs.Organizations) == 0 {
		fmt.Println("⚠️  No organizations found. Please create an organization first.")
		fmt.Println("   Use: flip definitions add org")
		return nil
	}

	orgItems := make([]string, len(defs.Organizations))
	for i, org := range defs.Organizations {
		orgItems[i] = fmt.Sprintf("[%s] %s", org.Abbreviation, org.Name)
	}

	orgSelector := promptui.Select{
		Label: "Select Organization",
		Items: orgItems,
		Size:  10,
	}

	orgIdx, _, err := orgSelector.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}

	selectedOrg := defs.Organizations[orgIdx].Abbreviation

	// Prompt for last name first
	lastNamePrompt := promptui.Prompt{
		Label: "Last Name",
	}
	lastName, err := lastNamePrompt.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}
	lastName = strings.TrimSpace(lastName)

	// Check for similar people by last name
	similar := findSimilarPeople(lastName, defs)
	if len(similar) > 0 {
		fmt.Println("\n⚠️  Found people with similar last names:")
		for _, s := range similar {
			fmt.Printf("   • %s\n", s)
		}
		fmt.Println()

		actions := []string{
			"Create new person anyway",
			"View/edit existing person",
			"Cancel",
		}

		actionPrompt := promptui.Select{
			Label: "What would you like to do?",
			Items: actions,
			Size:  5,
		}

		idx, _, err := actionPrompt.Run()
		if err != nil {
			fmt.Println("\n⚠️  Cancelled")
			return nil
		}

		switch idx {
		case 1: // View/edit
			// TODO: Implement person editing
			fmt.Println("\n💡 Person editing not yet implemented. Please use the definitions file directly.")
			fmt.Println("   Use: flip definitions file")
			return nil
		case 2: // Cancel
			fmt.Println("\n⚠️  Cancelled")
			return nil
		}
		// case 0: Continue with creation
	}

	// Prompt for first name
	firstNamePrompt := promptui.Prompt{
		Label: "First Name",
	}
	firstName, err := firstNamePrompt.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}
	firstName = strings.TrimSpace(firstName)

	fullName := firstName + " " + lastName

	// Prompt for abbreviation
	abbrPrompt := promptui.Prompt{
		Label:   "Abbreviation (e.g., JD, SELF)",
		Default: strings.ToUpper(string(firstName[0]) + string(lastName[0])),
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		fmt.Println("\n⚠️  Cancelled")
		return nil
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Prompt for org code (optional)
	orgCodePrompt := promptui.Prompt{
		Label:   "Organization Code (optional)",
		Default: "",
	}
	orgCode, _ := orgCodePrompt.Run()
	orgCode = strings.TrimSpace(orgCode)

	// Prompt for role (optional)
	rolePrompt := promptui.Prompt{
		Label:   "Role (optional)",
		Default: "",
	}
	role, _ := rolePrompt.Run()
	role = strings.TrimSpace(role)

	// Prompt for email (optional)
	emailPrompt := promptui.Prompt{
		Label:   "Email (optional)",
		Default: "",
	}
	email, _ := emailPrompt.Run()
	email = strings.TrimSpace(email)

	// Add person
	if err := defs.addPerson(fullName, abbr, selectedOrg, orgCode, role, email); err != nil {
		return err
	}

	if err := saveTaskDefinitions(defs); err != nil {
		return fmt.Errorf("failed to save definitions: %w", err)
	}

	fmt.Printf("\n✅ Person '[%s] %s' added to organization '%s'\n", abbr, fullName, selectedOrg)

	// Ask if user wants to open the file
	if err := promptOpenDefinitionsFile(); err != nil {
		return err
	}

	return nil
}

// runDefinitionsRemove removes a definition
func runDefinitionsRemove() error {
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	// Build list of all definitions
	items := []string{}

	for _, org := range defs.Organizations {
		items = append(items, fmt.Sprintf("🏢 [%s] %s", org.Abbreviation, org.Name))
	}
	for _, proj := range defs.Projects {
		items = append(items, fmt.Sprintf("📁 [%s] %s", proj.Abbreviation, proj.Name))
	}
	for _, ctx := range defs.Contexts {
		items = append(items, fmt.Sprintf("🏷️  [%s] %s", ctx.Abbreviation, ctx.Name))
	}

	if len(items) == 0 {
		fmt.Println("\nNo definitions to remove.")
		return nil
	}

	items = append(items, "← Cancel")

	selector := promptui.Select{
		Label: "Select definition to remove",
		Items: items,
		Size:  15,
	}

	idx, selected, err := selector.Run()
	if err != nil {
		return err
	}

	// Cancel
	if idx == len(items)-1 {
		return nil
	}

	// Confirm removal
	confirmPrompt := promptui.Prompt{
		Label:     fmt.Sprintf("Remove '%s'? (y/N)", selected),
		Default:   "N",
		IsConfirm: true,
	}

	result, err := confirmPrompt.Run()
	if err != nil || (result != "y" && result != "Y") {
		fmt.Println("Cancelled.")
		return nil
	}

	// Remove the selected item
	orgCount := len(defs.Organizations)
	projCount := len(defs.Projects)

	if idx < orgCount {
		// Remove organization
		defs.Organizations = append(defs.Organizations[:idx], defs.Organizations[idx+1:]...)
	} else if idx < orgCount+projCount {
		// Remove project
		projIdx := idx - orgCount
		defs.Projects = append(defs.Projects[:projIdx], defs.Projects[projIdx+1:]...)
	} else {
		// Remove context
		ctxIdx := idx - orgCount - projCount
		defs.Contexts = append(defs.Contexts[:ctxIdx], defs.Contexts[ctxIdx+1:]...)
	}

	// Save
	if err := saveTaskDefinitions(defs); err != nil {
		return fmt.Errorf("failed to save definitions: %w", err)
	}

	fmt.Println("\n✅ Definition removed successfully")
	return nil
}

// runDefinitionsScan scans tasks for organizations, projects, and contexts
func runDefinitionsScan() error {
	fmt.Println("\n🔍 Scanning Tasks & Checking Consistency")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Get active brain
	brain, err := getActiveBrain()
	if err != nil {
		return fmt.Errorf("no active brain: %w", err)
	}

	// Load current definitions
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	// First: Check for duplicates in definitions
	fmt.Println("\n📋 Checking definitions for duplicates...")
	duplicates := FindDuplicates(defs)
	if len(duplicates) > 0 {
		fmt.Println("\n⚠️  Found duplicates:")
		for _, dup := range duplicates {
			fmt.Printf("  • %s abbreviation '%s' has multiple names: %v\n", dup.Type, dup.Abbreviation, dup.Names)
		}
		fmt.Println()
	} else {
		fmt.Println("  ✅ No duplicates found")
	}

	// Scan tasks directory
	tasksDir := brain.Path + "/tasks"

	foundOrgs := make(map[string]bool)
	foundProjects := make(map[string]bool)
	foundContexts := make(map[string]bool)

	// Scan all .md files in tasks directory
	err = scanTaskFiles(tasksDir, foundOrgs, foundProjects, foundContexts)
	if err != nil {
		return fmt.Errorf("failed to scan tasks: %w", err)
	}

	// Filter out existing definitions
	newOrgs := []string{}
	for org := range foundOrgs {
		exists := false
		for _, existing := range defs.Organizations {
			if strings.EqualFold(existing.Abbreviation, org) || strings.EqualFold(existing.Name, org) {
				exists = true
				break
			}
		}
		if !exists && org != "" {
			newOrgs = append(newOrgs, org)
		}
	}

	newProjects := []string{}
	for proj := range foundProjects {
		exists := false
		for _, existing := range defs.Projects {
			if strings.EqualFold(existing.Abbreviation, proj) || strings.EqualFold(existing.Name, proj) {
				exists = true
				break
			}
		}
		if !exists && proj != "" {
			newProjects = append(newProjects, proj)
		}
	}

	newContexts := []string{}
	for ctx := range foundContexts {
		exists := false
		for _, existing := range defs.Contexts {
			if strings.EqualFold(existing.Abbreviation, ctx) || strings.EqualFold(existing.Name, ctx) {
				exists = true
				break
			}
		}
		if !exists && ctx != "" {
			newContexts = append(newContexts, ctx)
		}
	}

	// Display findings
	totalNew := len(newOrgs) + len(newProjects) + len(newContexts)

	if totalNew == 0 {
		fmt.Println("\n✅ No new definitions found. All organizations, projects, and contexts are already defined.")
		return nil
	}

	fmt.Printf("\nFound %d new items:\n\n", totalNew)

	if len(newOrgs) > 0 {
		fmt.Println("🏢 Organizations:")
		for _, org := range newOrgs {
			fmt.Printf("  - %s\n", org)
		}
		fmt.Println()
	}

	if len(newProjects) > 0 {
		fmt.Println("📁 Projects:")
		for _, proj := range newProjects {
			fmt.Printf("  - %s\n", proj)
		}
		fmt.Println()
	}

	if len(newContexts) > 0 {
		fmt.Println("🏷️  Contexts:")
		for _, ctx := range newContexts {
			fmt.Printf("  - %s\n", ctx)
		}
		fmt.Println()
	}

	// Ask if user wants to add them
	confirmPrompt := promptui.Prompt{
		Label:     "Add these to definitions? (y/N)",
		Default:   "N",
		IsConfirm: true,
	}

	result, err := confirmPrompt.Run()
	if err != nil || (result != "y" && result != "Y") {
		fmt.Println("Scan completed. No changes made.")
		return nil
	}

	// Add new definitions
	added := 0

	for _, org := range newOrgs {
		abbr := strings.ToUpper(org)
		if len(abbr) > 10 {
			abbr = abbr[:10]
		}
		if err := defs.addOrganization(org, abbr, "Discovered from task scan"); err == nil {
			added++
		}
	}

	for _, proj := range newProjects {
		abbr := strings.ToUpper(proj)
		if len(abbr) > 10 {
			abbr = abbr[:10]
		}
		if err := defs.addProject(proj, abbr, "", "Discovered from task scan"); err == nil {
			added++
		}
	}

	for _, ctx := range newContexts {
		abbr := strings.ToUpper(ctx)
		if len(abbr) > 10 {
			abbr = abbr[:10]
		}
		if err := defs.addContext(ctx, abbr, "Discovered from task scan"); err == nil {
			added++
		}
	}

	// Save
	if err := saveTaskDefinitions(defs); err != nil {
		return fmt.Errorf("failed to save definitions: %w", err)
	}

	fmt.Printf("\n✅ Added %d new definitions\n", added)
	fmt.Println("💡 Tip: Use 'flip definitions list' to review and 'flip definitions remove' to clean up")

	return nil
}

// scanTaskFiles recursively scans markdown files for org/project/context metadata
func scanTaskFiles(dir string, orgs, projects, contexts map[string]bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory might not exist yet
		return nil
	}

	for _, entry := range entries {
		fullPath := dir + "/" + entry.Name()

		if entry.IsDir() {
			// Recurse into subdirectories
			if err := scanTaskFiles(fullPath, orgs, projects, contexts); err != nil {
				return err
			}
			continue
		}

		// Only process markdown files
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Read file and extract metadata
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			// Look for metadata lines: organization::, project::, context::
			line = strings.TrimSpace(line)

			if strings.HasPrefix(line, "organization::") {
				org := strings.TrimSpace(strings.TrimPrefix(line, "organization::"))
				if org != "" {
					orgs[org] = true
				}
			} else if strings.HasPrefix(line, "project::") {
				proj := strings.TrimSpace(strings.TrimPrefix(line, "project::"))
				// Remove [[]] if present
				proj = strings.Trim(proj, "[]")
				if proj != "" {
					projects[proj] = true
				}
			} else if strings.HasPrefix(line, "context::") {
				ctx := strings.TrimSpace(strings.TrimPrefix(line, "context::"))
				if ctx != "" {
					contexts[ctx] = true
				}
			}
		}
	}

	return nil
}

// runDefinitionsPath shows the path to the definitions file
func runDefinitionsPath() error {
	path, err := getTaskDefinitionsPath()
	if err != nil {
		return fmt.Errorf("failed to get definitions path: %w", err)
	}

	fmt.Printf("\n📂 Definitions file: %s\n\n", path)
	return nil
}

// promptOpenDefinitionsFile asks if user wants to open the definitions file
func promptOpenDefinitionsFile() error {
	prompt := promptui.Select{
		Label: "Open definitions file in editor?",
		Items: []string{"Yes", "No"},
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return nil // Just continue if user cancels
	}

	if idx == 0 { // Yes
		path, err := getTaskDefinitionsPath()
		if err != nil {
			fmt.Printf("⚠️  Could not get definitions path: %v\n", err)
			return nil
		}

		// Open in editor (cross-platform)
		if err := openInEditor(path); err != nil {
			fmt.Printf("⚠️  Could not open editor: %v\n", err)
			fmt.Printf("   File path: %s\n", path)
		}

		fmt.Printf("✅ Opening %s\n", path)
	}

	return nil
}
