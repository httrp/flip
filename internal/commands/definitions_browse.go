package commands

import (
	"fmt"
	"strings"

	"github.com/httrp/flip/internal/lang"
	ui "github.com/httrp/flip/internal/ui"
)

// runDefinitionsBrowseOrganizations shows and manages organizations
func runDefinitionsBrowseOrganizations() error {
	defs, err := loadTaskDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	if len(defs.Organizations) == 0 {
		fmt.Println("\n🏢 No organizations defined yet.")
		fmt.Println("\nWould you like to add one?")

		addItems := []ui.SelectItem{
			{Label: "Yes", Value: "yes"},
			{Label: "No", Value: "no"},
		}

		_, val, err := ui.RunSelect("Add Organization", addItems, 3)
		if err != nil || val == "no" {
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

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
	}

	idx, _, err := ui.RunSelect(fmt.Sprintf("Organizations (%d total)", len(defs.Organizations)), selectItems, 15)
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

		addItems := []ui.SelectItem{
			{Label: "Yes", Value: "yes"},
			{Label: "No", Value: "no"},
		}

		_, val, err := ui.RunSelect("Add Project", addItems, 3)
		if err != nil || val == "no" {
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

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
	}

	idx, _, err := ui.RunSelect(fmt.Sprintf("Projects (%d total)", len(defs.Projects)), selectItems, 15)
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

		addItems := []ui.SelectItem{
			{Label: "Yes", Value: "yes"},
			{Label: "No", Value: "no"},
		}

		_, val, err := ui.RunSelect("Add Context", addItems, 3)
		if err != nil || val == "no" {
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

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
	}

	idx, _, err := ui.RunSelect(fmt.Sprintf("Contexts (%d total)", len(defs.Contexts)), selectItems, 15)
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

		addItems := []ui.SelectItem{
			{Label: "Yes", Value: "yes"},
			{Label: "No", Value: "no"},
		}

		_, val, err := ui.RunSelect("Add Person", addItems, 3)
		if err != nil || val == "no" {
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

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
	}

	idx, _, err := ui.RunSelect(fmt.Sprintf("People (%d total)", len(defs.People)), selectItems, 15)
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
