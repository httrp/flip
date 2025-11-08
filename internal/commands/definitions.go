package commands

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

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
			"📋 List All Definitions",
			"➕ Add Organization",
			"➕ Add Project",
			"➕ Add Context",
			"🗑️  Remove Definition",
			"📂 Show Definitions File Path",
			"← Back",
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
			if err := runDefinitionsList(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 1:
			if err := runDefinitionsAddOrg(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 2:
			if err := runDefinitionsAddProject(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 3:
			if err := runDefinitionsAddContext(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 4:
			if err := runDefinitionsRemove(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 5:
			if err := runDefinitionsPath(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case 6:
			return nil
		}
	}
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

	// Prompt for abbreviation
	abbrPrompt := promptui.Prompt{
		Label: "Abbreviation (e.g., WORK, PERS)",
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		return err
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Prompt for name
	namePrompt := promptui.Prompt{
		Label: "Full Name",
	}
	name, err := namePrompt.Run()
	if err != nil {
		return err
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
	return nil
}

// runDefinitionsAddProject adds a new project
func runDefinitionsAddProject() error {
	fmt.Println("\n➕ Add Project")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Prompt for abbreviation
	abbrPrompt := promptui.Prompt{
		Label: "Abbreviation (e.g., FLIP, BRAIN)",
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		return err
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Prompt for name
	namePrompt := promptui.Prompt{
		Label: "Full Name",
	}
	name, err := namePrompt.Run()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)

	// Prompt for organization (optional)
	orgPrompt := promptui.Prompt{
		Label:   "Organization (abbreviation, optional)",
		Default: "",
	}
	org, _ := orgPrompt.Run()
	org = strings.TrimSpace(strings.ToUpper(org))

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

	if err := defs.addProject(name, abbr, org, desc); err != nil {
		return err
	}

	if err := saveTaskDefinitions(defs); err != nil {
		return fmt.Errorf("failed to save definitions: %w", err)
	}

	fmt.Printf("\n✅ Project '[%s] %s' added successfully\n", abbr, name)
	return nil
}

// runDefinitionsAddContext adds a new context
func runDefinitionsAddContext() error {
	fmt.Println("\n➕ Add Context")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Prompt for abbreviation
	abbrPrompt := promptui.Prompt{
		Label: "Abbreviation (e.g., MTG, EMAIL)",
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		return err
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Prompt for name
	namePrompt := promptui.Prompt{
		Label: "Full Name",
	}
	name, err := namePrompt.Run()
	if err != nil {
		return err
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

	if err := defs.addContext(name, abbr, desc); err != nil {
		return err
	}

	if err := saveTaskDefinitions(defs); err != nil {
		return fmt.Errorf("failed to save definitions: %w", err)
	}

	fmt.Printf("\n✅ Context '[%s] %s' added successfully\n", abbr, name)
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
