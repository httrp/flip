package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ViewType represents the current view
type ViewType int

const (
	ViewMainMenu ViewType = iota
	ViewCreateMenu
	ViewBrowseMenu
	ViewManageMenu
	ViewStatusMenu
	ViewHelpMenu
	ViewAboutMenu
)

// AppModel is the root model that manages navigation between views
type AppModel struct {
	currentView    ViewType
	viewStack      []ViewType // For back navigation
	menu           MenuModel
	status         string // Status header content
	quitting       bool
	width          int
	height         int
	execCommand    string // Command to execute after quitting
	pendingAction  func() error
}

// NewAppModel creates a new application model
func NewAppModel() AppModel {
	app := AppModel{
		currentView: ViewMainMenu,
		viewStack:   []ViewType{},
	}
	app.menu = app.buildMainMenu()
	return app
}

// SetStatus sets the status header content
func (m *AppModel) SetStatus(status string) {
	m.status = status
}

// buildMainMenu creates the main menu items
func (m *AppModel) buildMainMenu() MenuModel {
	items := []MenuItem{
		NewMenuItem(
			"✨ Create New",
			"Create notes, meetings, tasks...",
			"flip new",
			func() tea.Cmd {
				return func() tea.Msg { return NavigateMsg{Menu: "create"} }
			},
		),
		NewMenuItem(
			"🔍 Browse & Search",
			"Find and explore your content",
			"flip search",
			func() tea.Cmd {
				return func() tea.Msg { return NavigateMsg{Menu: "browse"} }
			},
		),
		NewMenuItem(
			"⚙️  Manage Brains",
			"Switch brains, manage contexts",
			"flip brain",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "brain"} }
			},
		),
		NewMenuItem(
			"📊 Status",
			"View status overview",
			"flip status",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "status"} }
			},
		),
		NewMenuItem(
			"👋 Exit",
			"Exit flip",
			"",
			func() tea.Cmd {
				return func() tea.Msg { return QuitMsg{} }
			},
		),
	}

	menu := NewMenuModel("flip", items, []string{"Main Menu"})
	menu.SetStatus(m.status)
	return menu
}

// buildCreateMenu creates the create submenu
func (m *AppModel) buildCreateMenu() MenuModel {
	items := []MenuItem{
		NewMenuItem(
			"📝 Note",
			"Create a new note",
			"flip note new",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "note"} }
			},
		),
		NewMenuItem(
			"⚡ Quick Note",
			"Quick note with minimal prompts",
			"flip quicknote",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "quicknote"} }
			},
		),
		NewMenuItem(
			"📅 Meeting Note",
			"Create a meeting note",
			"flip meeting",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "meeting"} }
			},
		),
		NewMenuItem(
			"📓 Journal",
			"Open or create today's journal",
			"flip journal",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "journal"} }
			},
		),
		NewMenuItem(
			"✅ Task",
			"Create a new task",
			"flip task new",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "task"} }
			},
		),
		NewMenuItem(
			"← Back",
			"Return to main menu",
			"",
			func() tea.Cmd {
				return func() tea.Msg { return BackMsg{} }
			},
		),
	}

	return NewMenuModel("Create New", items, []string{"Main Menu", "Create"})
}

// buildBrowseMenu creates the browse submenu
func (m *AppModel) buildBrowseMenu() MenuModel {
	items := []MenuItem{
		NewMenuItem(
			"📄 Recent Notes",
			"View recently modified notes",
			"flip recent",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "recent"} }
			},
		),
		NewMenuItem(
			"🔍 Search Notes",
			"Search across all brains",
			"flip search",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "search"} }
			},
		),
		NewMenuItem(
			"✅ Task Browser",
			"Browse and manage tasks",
			"flip task browse",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "task-browse"} }
			},
		),
		NewMenuItem(
			"🐸 Eat the Frog",
			"Important tasks to tackle",
			"flip task frog",
			func() tea.Cmd {
				return func() tea.Msg { return ExecCommandMsg{Command: "task-frog"} }
			},
		),
		NewMenuItem(
			"← Back",
			"Return to main menu",
			"",
			func() tea.Cmd {
				return func() tea.Msg { return BackMsg{} }
			},
		),
	}

	return NewMenuModel("Browse & Search", items, []string{"Main Menu", "Browse"})
}

// ExecCommandMsg requests executing a command (exits bubbletea)
type ExecCommandMsg struct {
	Command string
}

func (m AppModel) Init() tea.Cmd {
	// Menu is built in NewAppModel
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case NavigateMsg:
		// Push current view to stack and navigate
		m.viewStack = append(m.viewStack, m.currentView)

		switch msg.Menu {
		case "create":
			m.currentView = ViewCreateMenu
			m.menu = m.buildCreateMenu()
		case "browse":
			m.currentView = ViewBrowseMenu
			m.menu = m.buildBrowseMenu()
		// Add more menus as needed
		}
		return m, nil

	case BackMsg:
		// Pop from stack and go back
		if len(m.viewStack) > 0 {
			m.currentView = m.viewStack[len(m.viewStack)-1]
			m.viewStack = m.viewStack[:len(m.viewStack)-1]
			m.menu = m.buildMenuForView(m.currentView)
		}
		return m, nil

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit

	case ExecCommandMsg:
		// Store the command and quit - caller will handle execution
		m.execCommand = msg.Command
		m.quitting = true
		return m, tea.Quit
	}

	// Update the current menu
	var cmd tea.Cmd
	newMenu, cmd := m.menu.Update(msg)
	m.menu = newMenu.(MenuModel)
	return m, cmd
}

func (m AppModel) buildMenuForView(view ViewType) MenuModel {
	switch view {
	case ViewCreateMenu:
		return m.buildCreateMenu()
	case ViewBrowseMenu:
		return m.buildBrowseMenu()
	default:
		return m.buildMainMenu()
	}
}

func (m AppModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Status header
	if m.status != "" {
		b.WriteString(StatusStyle.Render(m.status))
		b.WriteString("\n\n")
	}

	// Current menu
	b.WriteString(m.menu.View())

	return b.String()
}

// CurrentCommand returns the command that should be executed after quitting
func (m AppModel) CurrentCommand() string {
	return m.execCommand
}

// RunApp runs the interactive menu application
func RunApp(status string) (string, error) {
	app := AppModel{
		currentView: ViewMainMenu,
		viewStack:   []ViewType{},
		status:      status,
	}
	app.menu = app.buildMainMenu()

	p := tea.NewProgram(app, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	// Check if we need to execute a command
	if model, ok := finalModel.(AppModel); ok {
		return model.CurrentCommand(), nil
	}

	return "", nil
}
