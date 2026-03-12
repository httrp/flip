package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuItem represents an item in a menu
type MenuItem struct {
	label       string
	description string
	command     string // CLI command hint
	action      func() tea.Cmd
}

// Implement list.Item interface
func (m MenuItem) Title() string       { return m.label }
func (m MenuItem) Description() string { return m.description }
func (m MenuItem) FilterValue() string { return m.label }

// NewMenuItem creates a new menu item
func NewMenuItem(label, description, command string, action func() tea.Cmd) MenuItem {
	return MenuItem{
		label:       label,
		description: description,
		command:     command,
		action:      action,
	}
}

// NewMenuItemSimple creates a menu item that executes a simple function
func NewMenuItemSimple(label, description, command string, fn func() error) MenuItem {
	return MenuItem{
		label:       label,
		description: description,
		command:     command,
		action: func() tea.Cmd {
			return func() tea.Msg {
				return ActionResultMsg{Err: fn()}
			}
		},
	}
}

// ActionResultMsg is sent when an action completes
type ActionResultMsg struct {
	Err error
}

// NavigateMsg requests navigation to a submenu
type NavigateMsg struct {
	Menu string
}

// BackMsg requests going back to parent menu
type BackMsg struct{}

// QuitMsg requests quitting the application
type QuitMsg struct{}

// MenuModel is a bubbletea model for interactive menus
type MenuModel struct {
	list       list.Model
	title      string
	breadcrumb []string
	status     string // Status header content
	width      int
	height     int
	quitting   bool
	err        error
}

// Custom delegate for rendering menu items
type menuItemDelegate struct {
	showCommand bool
	styles      menuItemStyles
}

type menuItemStyles struct {
	normal      lipgloss.Style
	selected    lipgloss.Style
	description lipgloss.Style
	command     lipgloss.Style
}

func newMenuItemDelegate() menuItemDelegate {
	return menuItemDelegate{
		showCommand: true,
		styles: menuItemStyles{
			normal:      NormalItemStyle,
			selected:    SelectedItemStyle,
			description: ItemDescStyle,
			command:     CommandStyle,
		},
	}
}

func (d menuItemDelegate) Height() int                             { return 2 }
func (d menuItemDelegate) Spacing() int                            { return 0 }
func (d menuItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d menuItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(MenuItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	// Build the line
	var line strings.Builder

	if isSelected {
		line.WriteString("▸ ")
		line.WriteString(d.styles.selected.Render(item.label))
	} else {
		line.WriteString("  ")
		line.WriteString(d.styles.normal.Render(item.label))
	}

	// Add command hint if available and terminal is wide enough
	if d.showCommand && item.command != "" && m.Width() > 60 {
		cmdHint := d.styles.command.Render(" · " + item.command)
		line.WriteString(cmdHint)
	}

	fmt.Fprintln(w, line.String())

	// Second line: description (only if selected or enough space)
	if item.description != "" && (isSelected || m.Height() > 15) {
		desc := d.styles.description.Render(item.description)
		fmt.Fprintln(w, desc)
	} else {
		fmt.Fprintln(w, "") // Empty line for spacing
	}
}

// NewMenuModel creates a new menu with the given items
func NewMenuModel(title string, items []MenuItem, breadcrumb []string) MenuModel {
	// Convert to list.Item slice
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	// Create list with custom delegate
	delegate := newMenuItemDelegate()
	l := list.New(listItems, delegate, 80, 20)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()

	// Custom key bindings
	l.KeyMap.Quit = key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	)

	// Style the list
	l.Styles.Title = TitleStyle
	l.Styles.FilterPrompt = MutedStyle
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ColorPrimary)

	return MenuModel{
		list:       l,
		title:      title,
		breadcrumb: breadcrumb,
	}
}

// SetStatus sets the status header content
func (m *MenuModel) SetStatus(status string) {
	m.status = status
}

// Init implements tea.Model
func (m MenuModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys if filtering
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "enter":
			item, ok := m.list.SelectedItem().(MenuItem)
			if ok && item.action != nil {
				return m, item.action()
			}

		case "backspace", "esc":
			if len(m.breadcrumb) > 1 {
				return m, func() tea.Msg { return BackMsg{} }
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height-6) // Reserve space for header

	case ActionResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
		}

	case QuitMsg:
		m.quitting = true
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m MenuModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Status header
	if m.status != "" {
		b.WriteString(StatusStyle.Render(m.status))
		b.WriteString("\n")
	}

	// Breadcrumb
	if len(m.breadcrumb) > 0 {
		crumb := BreadcrumbStyle.Render(strings.Join(m.breadcrumb, " > "))
		b.WriteString(crumb)
		b.WriteString("\n")
	}

	// Error display
	if m.err != nil {
		b.WriteString(ErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n")
	}

	// Main list
	b.WriteString(m.list.View())

	return b.String()
}

// SelectedItem returns the currently selected item
func (m MenuModel) SelectedItem() (MenuItem, bool) {
	item, ok := m.list.SelectedItem().(MenuItem)
	return item, ok
}
