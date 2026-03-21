package commands

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/ui"
)

// promptForOrganization prompts for organization with history and definitions
func promptForOrganization() (string, error) {
	// Load definitions
	defs, err := loadTaskDefinitions()
	if err != nil {
		// If definitions fail to load, fall back to simple prompt
		return promptForSimpleOrganization()
	}

	items := []string{}

	// Add defined organizations
	if len(defs.Organizations) > 0 {
		items = append(items, defs.getOrganizationChoices()...)
	}

	// Add recent organizations from history (that aren't in definitions)
	history, _ := loadTaskHistory()
	if history != nil && len(history.RecentOrganizations) > 0 {
		for _, recent := range history.RecentOrganizations {
			// Check if already in definitions
			isInDefs := false
			for _, org := range defs.Organizations {
				if org.Abbreviation == recent || org.Name == recent {
					isInDefs = true
					break
				}
			}
			if !isInDefs {
				items = append(items, fmt.Sprintf("🕒 %s", recent))
			}
		}
	}

	items = append(items, "[New Organization]", "[Skip]")

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: item}
	}

	_, selected, err := ui.RunSelect("Organization", selectItems, 10)
	if err != nil {
		return "", err
	}

	// Skip
	if selected == "[Skip]" {
		return "", nil
	}

	// New organization
	if selected == "[New Organization]" {
		return promptForNewOrganization()
	}

	// Extract abbreviation from selected choice
	if strings.HasPrefix(selected, "🕒") {
		// Recent item
		return strings.TrimPrefix(selected, "🕒 "), nil
	}

	// From definitions - extract abbreviation
	return parseAbbreviationFromChoice(selected), nil
}

// promptForSimpleOrganization is a fallback when definitions can't be loaded
func promptForSimpleOrganization() (string, error) {
	history, err := loadTaskHistory()
	if err != nil {
		history = &TaskHistory{}
	}

	items := []string{"[New Organization]"}
	if len(history.RecentOrganizations) > 0 {
		items = append(items, history.RecentOrganizations...)
	}
	items = append(items, "[Skip]")

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: item}
	}

	_, selected, err := ui.RunSelect("Organization", selectItems, 0)
	if err != nil {
		return "", err
	}

	// Skip
	if selected == "[Skip]" {
		return "", nil
	}

	// New organization
	if selected == "[New Organization]" {
		org, err := ui.RunInput("Enter Organization (or leave empty to skip)", "", "", nil)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(org), nil
	}

	// Selected from history
	return selected, nil
}

// promptForProject prompts for project with history and definitions
func promptForProject() (string, error) {
	// Load definitions
	defs, err := loadTaskDefinitions()
	if err != nil {
		// If definitions fail to load, fall back to simple prompt
		return promptForSimpleProject()
	}

	items := []string{}

	// Add defined projects
	if len(defs.Projects) > 0 {
		items = append(items, defs.getProjectChoices()...)
	}

	// Add recent projects from history
	history, _ := loadTaskHistory()
	if history != nil && len(history.RecentProjects) > 0 {
		for _, recent := range history.RecentProjects {
			// Check if already in definitions
			isInDefs := false
			for _, proj := range defs.Projects {
				if proj.Abbreviation == recent || proj.Name == recent {
					isInDefs = true
					break
				}
			}
			if !isInDefs {
				items = append(items, fmt.Sprintf("🕒 %s", recent))
			}
		}
	}

	items = append(items, "[New Project]", "[Skip]")

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: item}
	}

	_, selected, err := ui.RunSelect("Project", selectItems, 10)
	if err != nil {
		return "", err
	}

	// Skip
	if selected == "[Skip]" {
		return "", nil
	}

	// New project
	if selected == "[New Project]" {
		return promptForNewProject()
	}

	// Extract abbreviation from selected choice
	if strings.HasPrefix(selected, "🕒") {
		return strings.TrimPrefix(selected, "🕒 "), nil
	}

	return parseAbbreviationFromChoice(selected), nil
}

// promptForSimpleProject is a fallback when definitions can't be loaded
func promptForSimpleProject() (string, error) {
	history, err := loadTaskHistory()
	if err != nil {
		history = &TaskHistory{}
	}

	items := []string{"[New Project]"}
	if len(history.RecentProjects) > 0 {
		items = append(items, history.RecentProjects...)
	}
	items = append(items, "[Skip]")

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: item}
	}

	_, selected, err := ui.RunSelect("Project", selectItems, 0)
	if err != nil {
		return "", err
	}

	// Skip
	if selected == "[Skip]" {
		return "", nil
	}

	// New project
	if selected == "[New Project]" {
		proj, err := ui.RunInput("Enter Project (or leave empty to skip)", "", "", nil)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(proj), nil
	}

	// Selected from history
	return selected, nil
}

// promptForContext prompts for context with history and definitions
func promptForContext() (string, error) {
	// Load definitions
	defs, err := loadTaskDefinitions()
	if err != nil {
		// If definitions fail to load, fall back to simple prompt
		return promptForSimpleContext()
	}

	items := []string{}

	// Add defined contexts
	if len(defs.Contexts) > 0 {
		items = append(items, defs.getContextChoices()...)
	}

	// Add recent contexts from history
	history, _ := loadTaskHistory()
	if history != nil && len(history.RecentContexts) > 0 {
		for _, recent := range history.RecentContexts {
			// Check if already in definitions
			isInDefs := false
			for _, ctx := range defs.Contexts {
				if ctx.Abbreviation == recent || ctx.Name == recent {
					isInDefs = true
					break
				}
			}
			if !isInDefs {
				items = append(items, fmt.Sprintf("🕒 %s", recent))
			}
		}
	}

	items = append(items, "[New Context]", "[Skip]")

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: item}
	}

	_, selected, err := ui.RunSelect("Context", selectItems, 10)
	if err != nil {
		return "", err
	}

	// Skip
	if selected == "[Skip]" {
		return "", nil
	}

	// New context
	if selected == "[New Context]" {
		return promptForNewContext()
	}

	// Extract abbreviation from selected choice
	if strings.HasPrefix(selected, "🕒") {
		return strings.TrimPrefix(selected, "🕒 "), nil
	}

	return parseAbbreviationFromChoice(selected), nil
}

// promptForSimpleContext is a fallback when definitions can't be loaded
func promptForSimpleContext() (string, error) {
	history, err := loadTaskHistory()
	if err != nil {
		history = &TaskHistory{}
	}

	items := []string{"[New Context]"}
	if len(history.RecentContexts) > 0 {
		items = append(items, history.RecentContexts...)
	}
	items = append(items, "[Skip]")

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: item}
	}

	_, selected, err := ui.RunSelect("Context", selectItems, 10)
	if err != nil {
		return "", err
	}

	// Skip
	if selected == "[Skip]" {
		return "", nil
	}

	// New context
	if selected == "[New Context]" {
		ctx, err := ui.RunInput("Enter Context (or leave empty to skip)", "", "", nil)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(ctx), nil
	}

	// Selected from history
	return selected, nil
}

// promptForTaskFile prompts for task file with smart suggestions
func promptForTaskFile(brain *Brain) (string, error) {
	history, err := loadTaskHistory()
	if err != nil {
		history = &TaskHistory{}
	}

	// Build file options
	items := []string{}

	// 1. Today's journal file
	today := time.Now().Format("2006-01-02")
	journalPath := filepath.Join("journal", today+".md")
	items = append(items, fmt.Sprintf("📅 Today's Journal (%s)", journalPath))

	// 2. Default task file
	items = append(items, "📋 tasks/tasks.md (recommended)")

	// 3. Recently modified markdown files in brain (last 5)
	recentMdFiles := getRecentlyModifiedMdFiles(brain.Path, 5)
	for _, file := range recentMdFiles {
		// Skip if it's the same as defaults or journals
		if file != journalPath && file != "tasks/tasks.md" && !strings.HasPrefix(file, "journal/") {
			items = append(items, fmt.Sprintf("📄 %s (recent)", file))
		}
	}

	// 4. Recently used task files from history
	if len(history.RecentFiles) > 0 {
		for _, file := range history.RecentFiles {
			// Skip if already shown or same as defaults
			alreadyShown := false
			for _, item := range items {
				if strings.Contains(item, file) {
					alreadyShown = true
					break
				}
			}
			if !alreadyShown && file != journalPath && file != "tasks/tasks.md" {
				items = append(items, fmt.Sprintf("🕒 %s", file))
			}
		}
	}

	// 5. Legacy options (for backward compatibility)
	items = append(items, "📂 tasks/backlog.md")
	items = append(items, "📆 tasks/today.md")
	items = append(items, "📆 tasks/this-week.md")

	// 6. Custom file
	items = append(items, "📁 Specify custom file...")

	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: item}
	}

	idx, selected, err := ui.RunSelect("Where to save the task?", selectItems, 10)
	if err != nil {
		return "", err
	}

	// Handle selection
	switch {
	case idx == 0:
		// Today's journal
		return filepath.Join(brain.Path, journalPath), nil
	case idx == 1:
		// tasks/tasks.md
		return filepath.Join(brain.Path, "tasks", "tasks.md"), nil
	case selected == "📁 Specify custom file...":
		// Custom file
		relPath, err := ui.RunInput("File path (relative to brain root)", "", "tasks/tasks.md", nil)
		if err != nil {
			return "", err
		}
		return filepath.Join(brain.Path, relPath), nil
	default:
		// Either recent file or legacy option
		// Extract path from display string
		selected := items[idx]
		var relPath string

		if strings.HasPrefix(selected, "🕒 ") {
			// Recent task file from history
			relPath = strings.TrimPrefix(selected, "🕒 ")
		} else if strings.HasPrefix(selected, "📄 ") {
			// Recent markdown file
			relPath = strings.TrimPrefix(selected, "📄 ")
			relPath = strings.TrimSuffix(relPath, " (recent)")
		} else if strings.HasPrefix(selected, "📂 ") {
			relPath = strings.TrimPrefix(selected, "📂 ")
		} else if strings.HasPrefix(selected, "📆 ") {
			relPath = strings.TrimPrefix(selected, "📆 ")
		}

		return filepath.Join(brain.Path, relPath), nil
	}
}

// promptForNewOrganization prompts for creating a new organization with both name and abbreviation
func promptForNewOrganization() (string, error) {
	fmt.Println("\n📝 Create New Organization")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Prompt for abbreviation
	abbr, err := ui.RunInput("Abbreviation (e.g., WORK, PERS)", "", "", nil)
	if err != nil {
		return "", err
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Allow skip with empty abbreviation
	if abbr == "" {
		return "", nil
	}

	// Prompt for full name
	name, err := ui.RunInput("Full Name (e.g., Work, Personal)", "", "", nil)
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)

	// If only abbreviation provided, use it as name too
	if name == "" {
		name = abbr
	}

	// Load definitions and add new organization
	defs, err := loadTaskDefinitions()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not load definitions: %v\n", err)
		return abbr, nil
	}

	// Check for duplicates
	if defs.hasOrganization(name, abbr) {
		fmt.Printf("⚠️  Warning: Organization '%s' or abbreviation '%s' already exists\n", name, abbr)
		return abbr, nil
	}

	// Add to definitions
	if err := defs.addOrganization(name, abbr, ""); err != nil {
		fmt.Printf("⚠️  Warning: Could not add organization: %v\n", err)
		return abbr, nil
	}

	// Save definitions
	if err := saveTaskDefinitions(defs); err != nil {
		fmt.Printf("⚠️  Warning: Could not save definitions: %v\n", err)
	} else {
		fmt.Printf("✅ Organization '[%s] %s' added to definitions\n", abbr, name)
	}

	return abbr, nil
}

// promptForNewProject prompts for creating a new project with both name and abbreviation
func promptForNewProject() (string, error) {
	fmt.Println("\n📝 Create New Project")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Prompt for abbreviation
	abbr, err := ui.RunInput("Abbreviation (e.g., FLIP, BRAIN)", "", "", nil)
	if err != nil {
		return "", err
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Allow skip with empty abbreviation
	if abbr == "" {
		return "", nil
	}

	// Prompt for full name
	name, err := ui.RunInput("Full Name (e.g., Flip CLI, Brain System)", "", "", nil)
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)

	// If only abbreviation provided, use it as name too
	if name == "" {
		name = abbr
	}

	// Load definitions and add new project
	defs, err := loadTaskDefinitions()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not load definitions: %v\n", err)
		return abbr, nil
	}

	// Check for duplicates
	if defs.hasProject(name, abbr) {
		fmt.Printf("⚠️  Warning: Project '%s' or abbreviation '%s' already exists\n", name, abbr)
		return abbr, nil
	}

	// Add to definitions
	if err := defs.addProject(name, abbr, "", ""); err != nil {
		fmt.Printf("⚠️  Warning: Could not add project: %v\n", err)
		return abbr, nil
	}

	// Save definitions
	if err := saveTaskDefinitions(defs); err != nil {
		fmt.Printf("⚠️  Warning: Could not save definitions: %v\n", err)
	} else {
		fmt.Printf("✅ Project '[%s] %s' added to definitions\n", abbr, name)
	}

	return abbr, nil
}

// promptForNewContext prompts for creating a new context with both name and abbreviation
func promptForNewContext() (string, error) {
	fmt.Println("\n📝 Create New Context")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Prompt for abbreviation
	abbr, err := ui.RunInput("Abbreviation (e.g., MTG, EMAIL)", "", "", nil)
	if err != nil {
		return "", err
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Allow skip with empty abbreviation
	if abbr == "" {
		return "", nil
	}

	// Prompt for full name
	name, err := ui.RunInput("Full Name (e.g., Meeting, Email)", "", "", nil)
	if err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)

	// If only abbreviation provided, use it as name too
	if name == "" {
		name = abbr
	}

	// Load definitions and add new context
	defs, err := loadTaskDefinitions()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not load definitions: %v\n", err)
		return abbr, nil
	}

	// Check for duplicates
	if defs.hasContext(name, abbr) {
		fmt.Printf("⚠️  Warning: Context '%s' or abbreviation '%s' already exists\n", name, abbr)
		return abbr, nil
	}

	// Add to definitions
	if err := defs.addContext(name, abbr, ""); err != nil {
		fmt.Printf("⚠️  Warning: Could not add context: %v\n", err)
		return abbr, nil
	}

	// Save definitions
	if err := saveTaskDefinitions(defs); err != nil {
		fmt.Printf("⚠️  Warning: Could not save definitions: %v\n", err)
	} else {
		fmt.Printf("✅ Context '[%s] %s' added to definitions\n", abbr, name)
	}

	return abbr, nil
}
