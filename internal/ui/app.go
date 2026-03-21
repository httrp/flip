package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// MenuDef defines a single menu (title, breadcrumb, items).
// Menus are identified by name (e.g. "main", "create", "browse").
type MenuDef struct {
	Title      string
	Breadcrumb []string
	Items      []MenuItem
}

// MenuConfig holds all menu definitions, keyed by name.
// Built by the caller (commands package) with feature-flag awareness.
type MenuConfig struct {
	Menus map[string]MenuDef
}

// AppModel is the root model that manages navigation between named menus.
type AppModel struct {
	currentMenu string
	menuStack   []string // For back navigation
	menu        MenuModel
	config      MenuConfig
	status      string // Status header content
	quitting    bool
	width       int
	height      int
	execCommand string // Command to execute after quitting
}

func (m *AppModel) loadMenu(name string) {
	def, ok := m.config.Menus[name]
	if !ok {
		// Fallback to main
		def = m.config.Menus["main"]
		name = "main"
	}
	m.currentMenu = name
	m.menu = NewMenuModel(def.Title, def.Items, def.Breadcrumb)
	m.menu.SetStatus(m.status)
}

// ExecCommandMsg requests executing a command (exits bubbletea).
type ExecCommandMsg struct {
	Command string
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case NavigateMsg:
		// Push current menu to stack and navigate
		m.menuStack = append(m.menuStack, m.currentMenu)
		m.loadMenu(msg.Menu)
		return m, nil

	case BackMsg:
		// Pop from stack and go back
		if len(m.menuStack) > 0 {
			prev := m.menuStack[len(m.menuStack)-1]
			m.menuStack = m.menuStack[:len(m.menuStack)-1]
			m.loadMenu(prev)
		}
		return m, nil

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit

	case ExecCommandMsg:
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

func (m AppModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	if m.status != "" {
		b.WriteString(StatusStyle.Render(m.status))
		b.WriteString("\n\n")
	}
	b.WriteString(m.menu.View())
	return b.String()
}

// CurrentCommand returns the command that should be executed after quitting.
func (m AppModel) CurrentCommand() string {
	return m.execCommand
}

// RunApp runs the interactive menu application with the given menu configuration.
func RunApp(status string, config MenuConfig) (string, error) {
	app := AppModel{
		config: config,
		status: status,
	}
	app.loadMenu("main")

	p := tea.NewProgram(app, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	if model, ok := finalModel.(AppModel); ok {
		return model.CurrentCommand(), nil
	}

	return "", nil
}
