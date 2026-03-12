package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// StreamingText Model - For AI response streaming
// =============================================================================

// StreamChunkMsg contains a chunk of streamed text
type StreamChunkMsg struct {
	Text string
}

// StreamDoneMsg signals streaming is complete
type StreamDoneMsg struct {
	Err error
}

// StreamingTextModel displays text as it streams in (like ChatGPT)
type StreamingTextModel struct {
	viewport viewport.Model
	content  strings.Builder
	spinner  spinner.Model
	title    string
	thinking bool
	done     bool
	err      error
	width    int
	height   int
}

// NewStreamingTextModel creates a new streaming text display
func NewStreamingTextModel(title string) StreamingTextModel {
	s := spinner.New()
	s.Spinner = spinner.Pulse
	s.Style = AIThinkingStyle

	vp := viewport.New(80, 20)
	vp.Style = AIResponseStyle

	return StreamingTextModel{
		viewport: vp,
		spinner:  s,
		title:    title,
		thinking: true,
		width:    80,
		height:   24,
	}
}

func (m StreamingTextModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m StreamingTextModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.done {
				return m, tea.Quit
			}
		}
		// Pass other keys to viewport for scrolling
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 4 // Title + thinking indicator + padding
		footerHeight := 2 // Help text
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - footerHeight

	case StreamChunkMsg:
		m.thinking = false
		m.content.WriteString(msg.Text)
		m.viewport.SetContent(m.content.String())
		// Auto-scroll to bottom
		m.viewport.GotoBottom()

	case StreamDoneMsg:
		m.done = true
		m.thinking = false
		m.err = msg.Err

	case spinner.TickMsg:
		if m.thinking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m StreamingTextModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(TitleStyle.Render(m.title))
	b.WriteString("\n")

	// Thinking indicator or status
	if m.thinking {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(AIThinkingStyle.Render("Thinking..."))
		b.WriteString("\n")
	} else if m.err != nil {
		b.WriteString(ErrorStyle.Render("✗ Error: " + m.err.Error()))
		b.WriteString("\n")
	} else if m.done {
		b.WriteString(SuccessStyle.Render("✓ Complete"))
		b.WriteString("\n")
	} else {
		b.WriteString("\n") // Empty line for spacing
	}

	// Main content viewport
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Help
	if m.done {
		b.WriteString(HelpStyle.Render("↑/↓: scroll • q/esc: close"))
	} else {
		b.WriteString(HelpStyle.Render("ctrl+c: cancel"))
	}

	return b.String()
}

// Content returns the full streamed content
func (m StreamingTextModel) Content() string {
	return m.content.String()
}

// IsDone returns whether streaming is complete
func (m StreamingTextModel) IsDone() bool {
	return m.done
}

// =============================================================================
// Diff View Model - For showing before/after changes
// =============================================================================

// DiffLine represents a line in a diff
type DiffLine struct {
	Type    string // "add", "remove", "context"
	Content string
}

// DiffViewModel displays a diff with accept/reject options
type DiffViewModel struct {
	viewport viewport.Model
	lines    []DiffLine
	title    string
	accepted bool
	done     bool
	width    int
	height   int
}

var (
	diffAddStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Background(lipgloss.Color("22"))

	diffRemoveStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Background(lipgloss.Color("52"))

	diffContextStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)
)

// NewDiffViewModel creates a new diff view
func NewDiffViewModel(title string, lines []DiffLine) DiffViewModel {
	vp := viewport.New(80, 20)

	// Render diff content
	var content strings.Builder
	for _, line := range lines {
		switch line.Type {
		case "add":
			content.WriteString(diffAddStyle.Render("+ " + line.Content))
		case "remove":
			content.WriteString(diffRemoveStyle.Render("- " + line.Content))
		default:
			content.WriteString(diffContextStyle.Render("  " + line.Content))
		}
		content.WriteString("\n")
	}
	vp.SetContent(content.String())

	return DiffViewModel{
		viewport: vp,
		lines:    lines,
		title:    title,
	}
}

func (m DiffViewModel) Init() tea.Cmd {
	return nil
}

func (m DiffViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "enter":
			m.accepted = true
			m.done = true
			return m, tea.Quit
		case "n", "esc", "q":
			m.accepted = false
			m.done = true
			return m, tea.Quit
		}

		// Scrolling
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 6
	}

	return m, nil
}

func (m DiffViewModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(m.title))
	b.WriteString("\n\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("y/enter: accept • n/esc: reject • ↑/↓: scroll"))

	return b.String()
}

// Accepted returns whether the diff was accepted
func (m DiffViewModel) Accepted() bool {
	return m.accepted
}
