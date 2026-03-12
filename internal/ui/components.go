package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// Input Model - For prompting user input
// =============================================================================

// InputModel handles text input with a prompt
type InputModel struct {
	textInput textinput.Model
	prompt    string
	err       error
	done      bool
}

// InputResultMsg is sent when input is complete
type InputResultMsg struct {
	Value string
	Err   error
}

// NewInputModel creates a new input prompt
func NewInputModel(prompt, placeholder string) InputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return InputModel{
		textInput: ti,
		prompt:    prompt,
	}
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.done = true
			return m, func() tea.Msg {
				return InputResultMsg{Value: m.textInput.Value()}
			}
		case "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render(m.prompt))
	b.WriteString("\n\n")
	b.WriteString(m.textInput.View())
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("enter: confirm • esc: cancel"))
	return b.String()
}

// Value returns the current input value
func (m InputModel) Value() string {
	return m.textInput.Value()
}

// =============================================================================
// Loading Model - For showing progress
// =============================================================================

// LoadingModel shows a spinner with a message
type LoadingModel struct {
	spinner spinner.Model
	message string
	done    bool
	result  string
	err     error
}

// LoadingDoneMsg signals loading is complete
type LoadingDoneMsg struct {
	Result string
	Err    error
}

// NewLoadingModel creates a new loading spinner
func NewLoadingModel(message string) LoadingModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return LoadingModel{
		spinner: s,
		message: message,
	}
}

func (m LoadingModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m LoadingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case LoadingDoneMsg:
		m.done = true
		m.result = msg.Result
		m.err = msg.Err
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m LoadingModel) View() string {
	if m.done {
		if m.err != nil {
			return ErrorStyle.Render("✗ " + m.err.Error())
		}
		return SuccessStyle.Render("✓ " + m.result)
	}

	return m.spinner.View() + " " + m.message
}

// =============================================================================
// Confirm Model - Yes/No confirmation
// =============================================================================

// ConfirmModel handles yes/no confirmation
type ConfirmModel struct {
	question string
	selected bool // true = yes, false = no
	done     bool
	answer   bool
}

// ConfirmResultMsg is sent when confirmation is complete
type ConfirmResultMsg struct {
	Confirmed bool
}

// NewConfirmModel creates a new confirmation prompt
func NewConfirmModel(question string) ConfirmModel {
	return ConfirmModel{
		question: question,
		selected: true, // Default to "Yes"
	}
}

func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "y":
			m.selected = true
		case "right", "l", "n":
			m.selected = false
		case "enter":
			m.done = true
			m.answer = m.selected
			return m, func() tea.Msg {
				return ConfirmResultMsg{Confirmed: m.selected}
			}
		case "ctrl+c", "esc":
			m.done = true
			m.answer = false
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m ConfirmModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render(m.question))
	b.WriteString("\n\n")

	yes := "  Yes  "
	no := "  No  "

	if m.selected {
		yes = SelectedItemStyle.Render(" Yes ")
		no = NormalItemStyle.Render(" No ")
	} else {
		yes = NormalItemStyle.Render(" Yes ")
		no = SelectedItemStyle.Render(" No ")
	}

	b.WriteString("    ")
	b.WriteString(yes)
	b.WriteString("    ")
	b.WriteString(no)
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("←/→: select • enter: confirm • esc: cancel"))

	return b.String()
}

// Answer returns the confirmation result
func (m ConfirmModel) Answer() bool {
	return m.answer
}
