package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/httrp/flip/internal/tasks"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// NewTaskNewCommand creates the task new command
func NewTaskNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "new",
		Aliases: []string{"n"},
		Short:   "Create a new task",
		Long:    "Create a new task interactively.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateTask()
		},
	}
	return cmd
}

// runCreateTask creates a new task interactively
func runCreateTask() error {
	fmt.Println("\n📝 Create New Task")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Get task description
	descPrompt := promptui.Prompt{
		Label: "Task Description",
	}
	description, err := descPrompt.Run()
	if err != nil {
		return err
	}

	// Get due date
	duePrompt := promptui.Prompt{
		Label:   "Due Date (YYYY-MM-DD, 'today', 'tomorrow', or leave empty)",
		Default: "",
	}
	dueStr, _ := duePrompt.Run()
	dueDate := parseDueDateString(dueStr)

	// Get priority
	prioritySelect := promptui.Select{
		Label:     "Priority",
		Items:     []string{"High ⏫", "Medium 🔼", "Low 🔽", "None"},
		CursorPos: 1, // Default to Medium (index 1)
	}
	priorityIdx, _, _ := prioritySelect.Run()
	priority := indexToPriority(priorityIdx)

	// Get organization
	organization, err := promptForOrganization()
	if err != nil {
		return err
	}

	// Get project
	project, err := promptForProject()
	if err != nil {
		return err
	}

	// Get context
	context, err := promptForContext()
	if err != nil {
		return err
	}

	// Get tags
	tagsPrompt := promptui.Prompt{
		Label:   "Tags (comma-separated, e.g., work,urgent)",
		Default: "",
	}
	tagsStr, _ := tagsPrompt.Run()
	tagList := parseTagsString(tagsStr)

	// Get active brain and allow selection if multiple brains exist
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get active workspace: %w", err)
	}

	if len(activeWs.Brains) == 0 {
		return fmt.Errorf("no brains configured")
	}

	// Use confirmOrSelectBrain to allow selection if multiple brains
	activeBrain, err := confirmOrSelectBrain(activeWs)
	if err != nil {
		return err
	}

	// Select where to save with improved options
	filePath, err := promptForTaskFile(activeBrain)
	if err != nil {
		return err
	}

	// Ensure tasks directory exists
	taskDir := filepath.Dir(filePath)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Create task object
	task := &tasks.Task{
		ID:           uuid.New().String(),
		Description:  description,
		Status:       tasks.StatusOpen,
		Created:      time.Now(),
		Due:          dueDate,
		Priority:     priority,
		Tags:         tagList,
		Organization: organization,
		Project:      project,
		ContextTag:   context,
	}

	// Append to file
	err = appendTaskToFile(filePath, task)
	if err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// Update task history
	relPath := relativePathFromBrain(filePath, activeBrain.Path)
	if err := updateTaskHistory(organization, project, context, relPath); err != nil {
		// Don't fail on history update error, just log
		fmt.Printf("⚠️  Warning: Could not update task history: %v\n", err)
	}

	fmt.Printf("\n✅ Task created in %s\n", relPath)
	fmt.Println()
	fmt.Println(tasks.FormatTask(task))
	fmt.Println()

	return nil
}

// appendTaskToFile appends a task to a markdown file
func appendTaskToFile(filePath string, task *tasks.Task) error {
	var content string
	var isNewFile bool

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		isNewFile = true
		// Check if this is the new unified tasks.md file
		if strings.HasSuffix(filePath, "tasks/tasks.md") {
			content = createUnifiedTasksTemplate()
		} else {
			// Legacy format for other files
			content = fmt.Sprintf("# Tasks\n\nCreated: %s\n\n", time.Now().Format("2006-01-02"))
		}
	} else {
		// Read existing content
		data, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		content = string(data)
	}

	// For unified tasks.md, append to appropriate section
	if strings.HasSuffix(filePath, "tasks/tasks.md") {
		content = appendToUnifiedTasks(content, task)
	} else {
		// Legacy: just append to end
		if !isNewFile {
			content += "\n"
		}
		content += tasks.FormatTask(task) + "\n"
	}

	// Write back
	return os.WriteFile(filePath, []byte(content), 0644)
}

// createUnifiedTasksTemplate creates the template for the new unified tasks.md
func createUnifiedTasksTemplate() string {
	today := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`# Tasks

Created: %s

## 🎯 Today

<!-- Tasks for today -->

## 📅 This Week

<!-- Tasks for this week -->

## 📋 Backlog

<!-- All other tasks -->

## ✅ Completed

<!-- Completed tasks (archive) -->

`, today)
}

// appendToUnifiedTasks appends a task to the appropriate section in unified tasks.md
func appendToUnifiedTasks(content string, task *tasks.Task) string {
	// Determine which section based on due date
	var section string
	if task.Due != nil {
		today := time.Now().Truncate(24 * time.Hour)
		dueDate := task.Due.Truncate(24 * time.Hour)

		if dueDate.Equal(today) {
			section = "## 🎯 Today"
		} else if dueDate.Before(today.Add(7 * 24 * time.Hour)) {
			section = "## 📅 This Week"
		} else {
			section = "## 📋 Backlog"
		}
	} else {
		section = "## 📋 Backlog"
	}

	// Find the section and insert after it
	lines := strings.Split(content, "\n")
	sectionIdx := -1

	for i, line := range lines {
		if strings.TrimSpace(line) == section {
			sectionIdx = i
			break
		}
	}

	if sectionIdx == -1 {
		// Section not found, append to end
		return content + "\n" + tasks.FormatTask(task) + "\n"
	}

	// Find insertion point (after section header and comment)
	insertIdx := sectionIdx + 1
	// Skip comments (but not empty lines yet)
	for insertIdx < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[insertIdx]), "<!--") {
		insertIdx++
	}

	// Now skip ALL empty lines after the comment to clean up spacing
	for insertIdx < len(lines) && strings.TrimSpace(lines[insertIdx]) == "" {
		insertIdx++
	}

	// Insert task with proper spacing: 1 empty line before, 1 empty line after
	taskLine := tasks.FormatTask(task)
	newLines := append(lines[:insertIdx], append([]string{"", taskLine, ""}, lines[insertIdx:]...)...)

	return strings.Join(newLines, "\n")
}

// Helper functions

func parseDueDateString(s string) *time.Time {
	if s == "" {
		return nil
	}

	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "today":
		t := time.Now()
		return &t
	case "tomorrow":
		t := time.Now().AddDate(0, 0, 1)
		return &t
	case "next week":
		t := time.Now().AddDate(0, 0, 7)
		return &t
	}

	// Try to parse as date
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return &t
	}

	return nil
}

func indexToPriority(idx int) tasks.Priority {
	switch idx {
	case 0:
		return tasks.PriorityHigh
	case 1:
		return tasks.PriorityMedium
	case 2:
		return tasks.PriorityLow
	default:
		return tasks.PriorityNone
	}
}

func parseTagsString(s string) []string {
	if s == "" {
		return []string{}
	}

	tags := strings.Split(s, ",")
	var result []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		tag = strings.TrimPrefix(tag, "#")
		if tag != "" {
			result = append(result, tag)
		}
	}
	return result
}

func relativePathFromBrain(fullPath, brainPath string) string {
	rel, err := filepath.Rel(brainPath, fullPath)
	if err != nil {
		return fullPath
	}
	return rel
}

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

	selector := promptui.Select{
		Label: "Organization",
		Items: items,
		Size:  10,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return "", err
	}

	selected := items[idx]

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

	selector := promptui.Select{
		Label: "Organization",
		Items: items,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return "", err
	}

	// Skip
	if idx == len(items)-1 {
		return "", nil
	}

	// New organization
	if idx == 0 {
		prompt := promptui.Prompt{
			Label:   "Enter Organization (or leave empty to skip)",
			Default: "",
		}
		org, err := prompt.Run()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(org), nil
	}

	// Selected from history
	return items[idx], nil
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

	selector := promptui.Select{
		Label: "Project",
		Items: items,
		Size:  10,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return "", err
	}

	selected := items[idx]

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

	selector := promptui.Select{
		Label: "Project",
		Items: items,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return "", err
	}

	// Skip
	if idx == len(items)-1 {
		return "", nil
	}

	// New project
	if idx == 0 {
		prompt := promptui.Prompt{
			Label:   "Enter Project (or leave empty to skip)",
			Default: "",
		}
		proj, err := prompt.Run()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(proj), nil
	}

	// Selected from history
	return items[idx], nil
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

	selector := promptui.Select{
		Label: "Context",
		Items: items,
		Size:  10,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return "", err
	}

	selected := items[idx]

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

	selector := promptui.Select{
		Label: "Context",
		Items: items,
	}

	idx, _, err := selector.Run()
	if err != nil {
		return "", err
	}

	// Skip
	if idx == len(items)-1 {
		return "", nil
	}

	// New context
	if idx == 0 {
		prompt := promptui.Prompt{
			Label:   "Enter Context (or leave empty to skip)",
			Default: "",
		}
		ctx, err := prompt.Run()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(ctx), nil
	}

	// Selected from history
	return items[idx], nil
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

	// 3. Recently used files
	if len(history.RecentFiles) > 0 {
		for _, file := range history.RecentFiles {
			// Skip if it's the same as defaults
			if file != journalPath && file != "tasks/tasks.md" {
				items = append(items, fmt.Sprintf("🕒 %s", file))
			}
		}
	}

	// 4. Legacy options (for backward compatibility)
	items = append(items, "📂 tasks/backlog.md")
	items = append(items, "📆 tasks/today.md")
	items = append(items, "📆 tasks/this-week.md")

	// 5. Custom file
	items = append(items, "📁 Specify custom file...")

	selector := promptui.Select{
		Label:     "Where to save the task?",
		Items:     items,
		Size:      10,
		CursorPos: 1, // Default to tasks/tasks.md (index 1)
	}

	idx, _, err := selector.Run()
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
	case idx == len(items)-1:
		// Custom file
		filePrompt := promptui.Prompt{
			Label:   "File path (relative to brain root)",
			Default: "tasks/tasks.md",
		}
		relPath, err := filePrompt.Run()
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
			// Recent file
			relPath = strings.TrimPrefix(selected, "🕒 ")
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
	abbrPrompt := promptui.Prompt{
		Label:   "Abbreviation (e.g., WORK, PERS)",
		Default: "",
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		return "", err
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Allow skip with empty abbreviation
	if abbr == "" {
		return "", nil
	}

	// Prompt for full name
	namePrompt := promptui.Prompt{
		Label:   "Full Name (e.g., Work, Personal)",
		Default: "",
	}
	name, err := namePrompt.Run()
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
	abbrPrompt := promptui.Prompt{
		Label:   "Abbreviation (e.g., FLIP, BRAIN)",
		Default: "",
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		return "", err
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Allow skip with empty abbreviation
	if abbr == "" {
		return "", nil
	}

	// Prompt for full name
	namePrompt := promptui.Prompt{
		Label:   "Full Name (e.g., Flip CLI, Brain System)",
		Default: "",
	}
	name, err := namePrompt.Run()
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
	abbrPrompt := promptui.Prompt{
		Label:   "Abbreviation (e.g., MTG, EMAIL)",
		Default: "",
	}
	abbr, err := abbrPrompt.Run()
	if err != nil {
		return "", err
	}
	abbr = strings.TrimSpace(strings.ToUpper(abbr))

	// Allow skip with empty abbreviation
	if abbr == "" {
		return "", nil
	}

	// Prompt for full name
	namePrompt := promptui.Prompt{
		Label:   "Full Name (e.g., Meeting, Email)",
		Default: "",
	}
	name, err := namePrompt.Run()
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
