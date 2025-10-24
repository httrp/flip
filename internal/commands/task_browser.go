package commands

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/httrp/flip/internal/tasks"
)

// Styles for the task browser
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	selectedItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("57")).
				Foreground(lipgloss.Color("230")).
				Bold(true)

	dimmedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	priorityHighStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

	priorityMedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))

	overdueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
)

// taskBrowserModel is the bubbletea model for interactive task browsing
type taskBrowserModel struct {
	tasks         []*tasks.Task
	filteredTasks []*tasks.Task
	cursor        int
	selected      map[int]struct{}

	// Two-axis filter system
	scopeFilter  string // "all", "org", "project", "context", "today", "week", "overdue", "high"
	scopeValue   string // e.g. "ECE" when scopeFilter is "org"
	statusFilter string // "all", "open", "inprogress", "done", "deferred", "cancelled"

	sortMode    string // "default", "priority", "due", "alpha"
	searchQuery string // Search/filter by keyword (future feature)

	width       int
	height      int
	showHelp    bool
	groupByFile bool
	message     string // Status message to display to user
}

// NewTaskBrowser creates a new interactive task browser
func NewTaskBrowser(taskList []*tasks.Task) taskBrowserModel {
	m := taskBrowserModel{
		tasks:         taskList,
		filteredTasks: []*tasks.Task{}, // Empty initially, will be set by applyFilter
		selected:      make(map[int]struct{}),
		scopeFilter:   "all",     // Start with all scopes
		statusFilter:  "open",    // Start with open tasks
		scopeValue:    "",        // No specific scope value initially
		sortMode:      "default", // Default file order
		groupByFile:   true,
		showHelp:      false, // Start with help hidden for cleaner view
		width:         80,    // Default width
		height:        30,    // Default height
	}
	// Apply initial filter
	m.applyFilter()
	return m
}

// Init implements bubbletea.Model
func (m taskBrowserModel) Init() tea.Cmd {
	// No initial command needed - WindowSizeMsg will arrive automatically
	return nil
}

// Update implements bubbletea.Model
func (m taskBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.filteredTasks)-1 {
				m.cursor++
			}

		case "g":
			// Go to top
			m.cursor = 0

		case "G":
			// Go to bottom
			if len(m.filteredTasks) > 0 {
				m.cursor = len(m.filteredTasks) - 1
			}

		case "enter", "o":
			// Open task in editor
			if m.cursor < len(m.filteredTasks) {
				task := m.filteredTasks[m.cursor]
				// Try to open in VS Code
				cmd := exec.Command("code", task.Context.FilePath)
				cmd.Start() // Fire and forget
				return m, tea.Quit
			}

		case "d":
			// Cycle through status: Open -> Done -> In Progress -> Deferred -> Cancelled -> Open
			if m.cursor < len(m.filteredTasks) {
				task := m.filteredTasks[m.cursor]
				oldStatus := task.Status

				// Cycle to next status
				switch task.Status {
				case tasks.StatusOpen:
					task.Status = tasks.StatusDone
				case tasks.StatusDone:
					task.Status = tasks.StatusInProgress
				case tasks.StatusInProgress:
					task.Status = tasks.StatusDeferred
				case tasks.StatusDeferred:
					task.Status = tasks.StatusCancelled
				case tasks.StatusCancelled:
					task.Status = tasks.StatusOpen
				default:
					task.Status = tasks.StatusDone
				}

				// Save to file
				if err := task.SaveToFile(); err != nil {
					m.message = fmt.Sprintf("❌ Save failed: %v | File: %s Line: %d", err, task.Context.FilePath, task.Context.LineNumber)
					// Revert status change
					task.Status = oldStatus
				} else {
					statusName := map[tasks.Status]string{
						tasks.StatusOpen:       "Open",
						tasks.StatusDone:       "Done",
						tasks.StatusInProgress: "In Progress",
						tasks.StatusDeferred:   "Deferred",
						tasks.StatusCancelled:  "Cancelled",
					}
					m.message = fmt.Sprintf("✅ Saved: %s → %s", statusName[oldStatus], statusName[task.Status])
				}
			}

		case "f":
			// Cycle through scope filters
			m.cycleScopeFilter()
			m.applyFilter()

		case "F":
			// Cycle through scope values (e.g. different orgs, projects)
			m.cycleScopeValue()
			m.applyFilter()

		case "t":
			// Cycle through status filters
			m.cycleStatusFilter()
			m.applyFilter()

		case "s":
			// Cycle through sort modes
			m.cycleSortMode()
			m.applyFilter() // Re-apply to sort

		case "/":
			// Toggle search/filter by keyword
			if m.searchQuery == "" {
				// For now, simple implementation - in future could add input field
				m.message = "💡 Search: Type keywords in description to filter"
			} else {
				m.searchQuery = ""
				m.message = "🔍 Search cleared"
				m.applyFilter()
			}

		case "?":
			// Toggle help
			m.showHelp = !m.showHelp
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View implements bubbletea.Model
func (m taskBrowserModel) View() string {
	var b strings.Builder

	// Move cursor to top-left to prevent first line cutoff
	b.WriteString("\033[H")

	// Header
	header := titleStyle.Render("📋 Task Browser")

	// Build scope text
	scopeText := m.scopeFilter
	if m.scopeValue != "" {
		scopeText = fmt.Sprintf("%s:%s", m.scopeFilter, m.scopeValue)
	}

	// Build info string with scope, status, and sort
	infoText := fmt.Sprintf(" [Scope: %s | Status: %s | Sort: %s | %d tasks]", scopeText, m.statusFilter, m.sortMode, len(m.filteredTasks))
	filterInfo := dimmedStyle.Render(infoText)
	b.WriteString(header + filterInfo + "\n")

	// Use max of actual width or default
	width := m.width
	if width == 0 {
		width = 80
	}
	b.WriteString(strings.Repeat("─", width) + "\n\n")

	// Tasks
	if len(m.filteredTasks) == 0 {
		b.WriteString(dimmedStyle.Render("  No tasks found\n"))
	} else {
		if m.groupByFile {
			b.WriteString(m.renderGroupedByFile())
		} else {
			b.WriteString(m.renderList())
		}
	}

	// Footer - Always show command hints
	b.WriteString("\n" + strings.Repeat("─", width) + "\n")

	// Compact command hints (always visible)
	commands := "j/k: ↑↓ • g/G: top/bottom • d: toggle status • f: scope • F: scope value • t: status • s: sort • Enter: open • q: quit"
	b.WriteString(helpStyle.Render(commands))

	// Extended help (only when toggled)
	if m.showHelp {
		scopeHelp := "\nScope (f): all → org → project → context → today → week → overdue → high"
		statusHelp := "\nStatus (t): open → inprogress → all → done → deferred → cancelled"
		b.WriteString(dimmedStyle.Render(scopeHelp))
		b.WriteString(dimmedStyle.Render(statusHelp))
	} else {
		b.WriteString(dimmedStyle.Render("\n? for help"))
	}

	// Status message
	if m.message != "" {
		b.WriteString("\n" + m.message)
	}

	// No padding needed - viewport calculation in render functions ensures correct size
	return b.String()
}

// renderList renders tasks as a simple list
func (m taskBrowserModel) renderList() string {
	var b strings.Builder

	// Calculate viewport
	headerFooterLines := 5
	if m.showHelp {
		headerFooterLines = 9
	}

	viewportSize := 20 // Default - limit to prevent rendering issues
	if m.height > 10 {
		viewportSize = m.height - headerFooterLines
		if viewportSize < 10 {
			viewportSize = 10
		}
		if viewportSize > 20 {
			viewportSize = 20 // Max 20 tasks visible at once
		}
	}

	// Start from beginning if cursor is near top
	start := 0
	if m.cursor > viewportSize/2 {
		start = m.cursor - viewportSize/2
	}
	end := minInt(len(m.filteredTasks), start+viewportSize)

	for i := start; i < end; i++ {
		task := m.filteredTasks[i]
		line := m.formatTaskLine(task, i == m.cursor)
		b.WriteString(line + "\n")
	}

	return b.String()
}

// renderGroupedByFile renders tasks grouped by file
func (m taskBrowserModel) renderGroupedByFile() string {
	var b strings.Builder

	// Calculate viewport - show tasks around cursor (based on terminal height)
	// Reserve space for: header(3) + footer(0 or 4 if help) + message(1-2)
	headerFooterLines := 5
	if m.showHelp {
		headerFooterLines = 9 // More space when help is shown
	}

	viewportSize := m.height - headerFooterLines
	if m.height == 0 {
		// Terminal height not yet set - use safe default
		viewportSize = 20
	} else if viewportSize < 10 {
		viewportSize = 10 // Minimum viewport
	}
	if viewportSize > 20 {
		viewportSize = 20 // Maximum 20 tasks to prevent header rendering issues
	}

	// Calculate start position - if cursor is near the top, show from beginning
	start := 0
	if m.cursor > viewportSize/2 {
		start = m.cursor - viewportSize/2
	}
	end := minInt(len(m.filteredTasks), start+viewportSize)

	// Adjust start if we're near the end
	if end-start < viewportSize && len(m.filteredTasks) > viewportSize {
		start = maxInt(0, end-viewportSize)
	}

	// Group tasks by file
	fileGroups := make(map[string][]*tasks.Task)
	fileOrder := []string{}
	for _, task := range m.filteredTasks {
		filePath := task.Context.FilePath
		if _, exists := fileGroups[filePath]; !exists {
			fileOrder = append(fileOrder, filePath)
		}
		fileGroups[filePath] = append(fileGroups[filePath], task)
	}

	// Find current task's file for highlighting
	currentFile := ""
	if m.cursor < len(m.filteredTasks) {
		currentFile = m.filteredTasks[m.cursor].Context.FilePath
	}

	// Render file groups (only show tasks in viewport)
	taskIndex := 0
	for _, filePath := range fileOrder {
		fileTasks := fileGroups[filePath]
		relPath := getRelativePathFromBrain(filePath, fileTasks[0].Context.BrainPath)

		// Check if this file has any visible tasks
		fileStartIdx := taskIndex
		fileEndIdx := taskIndex + len(fileTasks)

		hasVisibleTasks := !(fileEndIdx <= start || fileStartIdx >= end)

		if hasVisibleTasks {
			// Only render file header if we're actually going to render tasks from it
			renderedAnyTask := false

			// Render tasks in this file (only if in viewport)
			for _, task := range fileTasks {
				if taskIndex >= start && taskIndex < end {
					// Render file header before first task
					if !renderedAnyTask {
						fileHeader := fmt.Sprintf("📁 %s (%d)", relPath, len(fileTasks))
						if filePath == currentFile {
							fileHeader = headerStyle.Render(fileHeader)
						} else {
							fileHeader = dimmedStyle.Render(fileHeader)
						}
						b.WriteString(fileHeader + "\n")
						renderedAnyTask = true
					}

					line := "  " + m.formatTaskLine(task, taskIndex == m.cursor)
					b.WriteString(line + "\n")
				}
				taskIndex++
			}

			// Add spacing after file group (only if we rendered tasks)
			if renderedAnyTask {
				b.WriteString("\n")
			}
		} else {
			// Skip this file, but still increment index
			taskIndex += len(fileTasks)
		}
	}

	// Show scroll indicator
	if len(m.filteredTasks) > viewportSize {
		scrollInfo := fmt.Sprintf("  [%d-%d of %d tasks]", start+1, minInt(end, len(m.filteredTasks)), len(m.filteredTasks))
		b.WriteString(dimmedStyle.Render(scrollInfo) + "\n")
	}

	return b.String()
}

// formatTaskLine formats a single task line
func (m taskBrowserModel) formatTaskLine(task *tasks.Task, isSelected bool) string {
	// Priority icon
	priorityIcon := tasks.PriorityIcon(task.Priority)
	if priorityIcon == "" {
		priorityIcon = "  "
	}

	// Status
	statusIcon := "[ ]"
	switch task.Status {
	case tasks.StatusDone:
		statusIcon = "[✓]"
	case tasks.StatusInProgress:
		statusIcon = "[~]"
	case tasks.StatusDeferred:
		statusIcon = "[>]"
	case tasks.StatusCancelled:
		statusIcon = "[-]"
	}

	// Organization/Context metadata
	metaStr := ""
	if task.Organization != "" {
		metaStr = fmt.Sprintf(" [%s]", task.Organization)
	}
	if task.ContextTag != "" {
		metaStr += fmt.Sprintf(" ctx:%s", task.ContextTag)
	}

	// Due date
	dueStr := ""
	if task.Due != nil {
		dueStr = " " + formatTaskDueDate(task.Due)
	}

	// Tags (first 2)
	tagsStr := ""
	if len(task.Tags) > 0 {
		tagList := task.Tags
		if len(tagList) > 2 {
			tagList = tagList[:2]
		}
		tags := []string{}
		for _, tag := range tagList {
			tags = append(tags, "#"+tag)
		}
		tagsStr = " " + strings.Join(tags, " ")
		if len(task.Tags) > 2 {
			tagsStr += fmt.Sprintf(" +%d", len(task.Tags)-2)
		}
	}

	// Build line
	line := fmt.Sprintf("%s %s %s%s%s%s",
		priorityIcon,
		statusIcon,
		truncate(task.Description, 60),
		metaStr,
		dueStr,
		tagsStr,
	)

	// Apply styling
	if isSelected {
		return selectedItemStyle.Render("→ " + line)
	}

	// Color based on priority/status
	if task.IsOverdue() {
		return overdueStyle.Render("  " + line)
	}
	if task.Priority == tasks.PriorityHigh {
		return priorityHighStyle.Render("  " + line)
	}
	if task.Priority == tasks.PriorityMedium {
		return priorityMedStyle.Render("  " + line)
	}

	return "  " + line
}

// applyFilter filters tasks based on current filter mode
func (m *taskBrowserModel) applyFilter() {
	m.filteredTasks = []*tasks.Task{}

	// Auto-select first value for scope filters if not set
	if m.scopeFilter == "org" && m.scopeValue == "" {
		m.scopeValue = m.findFirstOrganization()
	}
	if m.scopeFilter == "project" && m.scopeValue == "" {
		m.scopeValue = m.findFirstProject()
	}
	if m.scopeFilter == "context" && m.scopeValue == "" {
		m.scopeValue = m.findFirstContext()
	}

	for _, task := range m.tasks {
		// Apply scope filter first
		scopeMatch := false
		switch m.scopeFilter {
		case "all":
			scopeMatch = true
		case "org":
			scopeMatch = task.Organization == m.scopeValue
		case "project":
			scopeMatch = task.Project == m.scopeValue
		case "context":
			scopeMatch = task.ContextTag == m.scopeValue
		case "today":
			scopeMatch = task.IsDueToday()
		case "week":
			scopeMatch = task.IsDueThisWeek()
		case "overdue":
			scopeMatch = task.IsOverdue()
		case "high":
			scopeMatch = task.Priority == tasks.PriorityHigh
		}

		// Apply status filter second
		statusMatch := false
		switch m.statusFilter {
		case "all":
			statusMatch = true
		case "open":
			statusMatch = task.Status == tasks.StatusOpen
		case "inprogress":
			statusMatch = task.Status == tasks.StatusInProgress
		case "done":
			statusMatch = task.Status == tasks.StatusDone
		case "deferred":
			statusMatch = task.Status == tasks.StatusDeferred
		case "cancelled":
			statusMatch = task.Status == tasks.StatusCancelled
		}

		// Include if both filters match
		if scopeMatch && statusMatch {
			m.filteredTasks = append(m.filteredTasks, task)
		}
	}

	// Apply sorting
	m.sortTasks()

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filteredTasks) {
		m.cursor = maxInt(0, len(m.filteredTasks)-1)
	}
}

// sortTasks sorts filtered tasks based on current sort mode
func (m *taskBrowserModel) sortTasks() {
	switch m.sortMode {
	case "priority":
		// Sort by priority (high to low), then by description
		sort.Slice(m.filteredTasks, func(i, j int) bool {
			if m.filteredTasks[i].Priority != m.filteredTasks[j].Priority {
				return m.filteredTasks[i].Priority > m.filteredTasks[j].Priority
			}
			return m.filteredTasks[i].Description < m.filteredTasks[j].Description
		})
	case "due":
		// Sort by due date (earliest first), tasks without due date go to end
		sort.Slice(m.filteredTasks, func(i, j int) bool {
			ti := m.filteredTasks[i].Due
			tj := m.filteredTasks[j].Due
			// Both nil - sort by description
			if ti == nil && tj == nil {
				return m.filteredTasks[i].Description < m.filteredTasks[j].Description
			}
			// i has no due date - put it after j
			if ti == nil {
				return false
			}
			// j has no due date - put it after i
			if tj == nil {
				return true
			}
			// Both have due dates - sort by date
			return ti.Before(*tj)
		})
	case "alpha":
		// Sort alphabetically by description
		sort.Slice(m.filteredTasks, func(i, j int) bool {
			return m.filteredTasks[i].Description < m.filteredTasks[j].Description
		})
	case "default":
		// No sorting - keep file order
	}
}

// cycleSortMode cycles through sort modes
func (m *taskBrowserModel) cycleSortMode() {
	modes := []string{"default", "priority", "due", "alpha"}
	for i, mode := range modes {
		if mode == m.sortMode {
			m.sortMode = modes[(i+1)%len(modes)]
			m.message = fmt.Sprintf("📊 Sort: %s", m.sortMode)
			return
		}
	}
	m.sortMode = "default"
	m.message = "📊 Sort: default"
}

// Helper functions to find first available filter values
func (m *taskBrowserModel) findFirstOrganization() string {
	for _, task := range m.tasks {
		if task.Organization != "" {
			return task.Organization
		}
	}
	return ""
}

func (m *taskBrowserModel) findFirstProject() string {
	for _, task := range m.tasks {
		if task.Project != "" {
			return task.Project
		}
	}
	return ""
}

func (m *taskBrowserModel) findFirstContext() string {
	for _, task := range m.tasks {
		if task.ContextTag != "" {
			return task.ContextTag
		}
	}
	return ""
}

// nextFilterMode cycles to next filter
// cycleScopeFilter cycles through scope filter options
func (m *taskBrowserModel) cycleScopeFilter() {
	scopes := []string{
		"all",     // All tasks
		"org",     // By organization
		"project", // By project
		"context", // By context
		"today",   // Due today
		"week",    // Due this week
		"overdue", // Overdue
		"high",    // High priority
	}
	for i, scope := range scopes {
		if scope == m.scopeFilter {
			m.scopeFilter = scopes[(i+1)%len(scopes)]
			// Reset scope value when changing scope type
			if m.scopeFilter == "org" || m.scopeFilter == "project" || m.scopeFilter == "context" {
				m.scopeValue = "" // Will be auto-selected in applyFilter
			}
			m.message = fmt.Sprintf("🔍 Scope: %s", m.scopeFilter)
			return
		}
	}
	m.scopeFilter = "all"
}

// cycleStatusFilter cycles through status filter options
func (m *taskBrowserModel) cycleStatusFilter() {
	statuses := []string{
		"open",       // Open tasks
		"inprogress", // In progress
		"all",        // All statuses
		"done",       // Completed
		"deferred",   // Deferred
		"cancelled",  // Cancelled
	}
	for i, status := range statuses {
		if status == m.statusFilter {
			m.statusFilter = statuses[(i+1)%len(statuses)]
			m.message = fmt.Sprintf("📊 Status: %s", m.statusFilter)
			return
		}
	}
	m.statusFilter = "open"
}

// cycleScopeValue cycles through available values for the current scope filter
func (m *taskBrowserModel) cycleScopeValue() {
	switch m.scopeFilter {
	case "org":
		orgs := m.getUniqueOrganizations()
		if len(orgs) == 0 {
			m.message = "❌ No organizations found"
			return
		}
		m.scopeValue = m.cycleToNext(orgs, m.scopeValue)
		m.message = fmt.Sprintf("📊 Organization: %s", m.scopeValue)

	case "project":
		projects := m.getUniqueProjects()
		if len(projects) == 0 {
			m.message = "❌ No projects found"
			return
		}
		m.scopeValue = m.cycleToNext(projects, m.scopeValue)
		m.message = fmt.Sprintf("📁 Project: %s", m.scopeValue)

	case "context":
		contexts := m.getUniqueContexts()
		if len(contexts) == 0 {
			m.message = "❌ No contexts found"
			return
		}
		m.scopeValue = m.cycleToNext(contexts, m.scopeValue)
		m.message = fmt.Sprintf("🏷️  Context: %s", m.scopeValue)

	default:
		m.message = "⚠️  Shift+F only works with org/project/context scope filters"
	}
}

// cycleToNext finds the next item in a list, cycling back to start if needed
func (m *taskBrowserModel) cycleToNext(items []string, current string) string {
	for i, item := range items {
		if item == current {
			return items[(i+1)%len(items)]
		}
	}
	// If current not found, return first
	return items[0]
}

// getUniqueOrganizations returns all unique organizations from tasks
func (m *taskBrowserModel) getUniqueOrganizations() []string {
	return m.getUniqueValues(func(t *tasks.Task) string { return t.Organization })
}

// getUniqueProjects returns all unique projects from tasks
func (m *taskBrowserModel) getUniqueProjects() []string {
	return m.getUniqueValues(func(t *tasks.Task) string { return t.Project })
}

// getUniqueContexts returns all unique context tags from tasks
func (m *taskBrowserModel) getUniqueContexts() []string {
	return m.getUniqueValues(func(t *tasks.Task) string { return t.ContextTag })
}

// getUniqueValues is a generic helper to extract unique non-empty values from tasks
func (m *taskBrowserModel) getUniqueValues(getter func(*tasks.Task) string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, task := range m.tasks {
		value := getter(task)
		if value != "" && !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}
	return result
}

// Helper functions
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
