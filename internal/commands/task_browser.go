package commands

import (
	"fmt"
	"os/exec"
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
	filterMode    string // "all", "today", "week", "overdue", "high"
	searchQuery   string
	width         int
	height        int
	showHelp      bool
	groupByFile   bool
}

// NewTaskBrowser creates a new interactive task browser
func NewTaskBrowser(taskList []*tasks.Task) taskBrowserModel {
	return taskBrowserModel{
		tasks:         taskList,
		filteredTasks: taskList,
		selected:      make(map[int]struct{}),
		filterMode:    "all",
		groupByFile:   true,
		showHelp:      true,
		width:         80, // Default width
		height:        24, // Default height
	}
}

// Init implements bubbletea.Model
func (m taskBrowserModel) Init() tea.Cmd {
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
			// Toggle done
			if m.cursor < len(m.filteredTasks) {
				task := m.filteredTasks[m.cursor]
				if task.Status == tasks.StatusDone {
					task.Status = tasks.StatusOpen
				} else {
					task.Status = tasks.StatusDone
				}
				// TODO: Save back to file
			}

		case "f":
			// Cycle through filters
			m.filterMode = m.nextFilterMode()
			m.applyFilter()

		case "?":
			// Toggle help
			m.showHelp = !m.showHelp

		case "/":
			// Search mode (TODO: implement)
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

	// Header
	header := titleStyle.Render("📋 Task Browser")
	filterInfo := dimmedStyle.Render(fmt.Sprintf(" [Filter: %s | %d tasks | cursor: %d]", m.filterMode, len(m.filteredTasks), m.cursor))
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

	// Footer / Help
	if m.showHelp {
		width := m.width
		if width == 0 {
			width = 80
		}
		b.WriteString("\n" + strings.Repeat("─", width) + "\n")
		help := []string{
			"↑/k: up",
			"↓/j: down",
			"g/G: top/bottom",
			"Enter: open",
			"d: toggle done",
			"f: filter",
			"q: quit",
		}
		b.WriteString(helpStyle.Render(strings.Join(help, " • ")))
	}

	return b.String()
}

// renderList renders tasks as a simple list
func (m taskBrowserModel) renderList() string {
	var b strings.Builder

	start := maxInt(0, m.cursor-10)
	end := minInt(len(m.filteredTasks), m.cursor+10)

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

	// Render each file group
	taskIndex := 0
	for _, filePath := range fileOrder {
		fileTasks := fileGroups[filePath]
		relPath := getRelativePathFromBrain(filePath, fileTasks[0].Context.BrainPath)

		// File header
		fileHeader := fmt.Sprintf("📁 %s (%d)", relPath, len(fileTasks))
		if filePath == currentFile {
			fileHeader = headerStyle.Render(fileHeader)
		} else {
			fileHeader = dimmedStyle.Render(fileHeader)
		}
		b.WriteString(fileHeader + "\n")

		// Tasks in this file
		for _, task := range fileTasks {
			line := "  " + m.formatTaskLine(task, taskIndex == m.cursor)
			b.WriteString(line + "\n")
			taskIndex++
		}
		b.WriteString("\n")
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
	line := fmt.Sprintf("%s %s %s%s%s",
		priorityIcon,
		statusIcon,
		truncate(task.Description, 60),
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

	for _, task := range m.tasks {
		include := false

		switch m.filterMode {
		case "all":
			include = task.Status == tasks.StatusOpen
		case "today":
			include = task.IsDueToday()
		case "week":
			include = task.IsDueThisWeek()
		case "overdue":
			include = task.IsOverdue()
		case "high":
			include = task.Priority == tasks.PriorityHigh && task.Status == tasks.StatusOpen
		case "done":
			include = task.Status == tasks.StatusDone
		}

		if include {
			m.filteredTasks = append(m.filteredTasks, task)
		}
	}

	// Reset cursor
	if m.cursor >= len(m.filteredTasks) {
		m.cursor = maxInt(0, len(m.filteredTasks)-1)
	}
}

// nextFilterMode cycles to next filter
func (m *taskBrowserModel) nextFilterMode() string {
	modes := []string{"all", "today", "week", "overdue", "high", "done"}
	for i, mode := range modes {
		if mode == m.filterMode {
			return modes[(i+1)%len(modes)]
		}
	}
	return "all"
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
