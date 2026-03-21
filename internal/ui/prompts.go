package ui

// prompts.go — Standalone prompt runners for inline use.
//
// These wrap bubbletea models in tea.NewProgram() so callers can use
// simple function calls instead of managing the TUI lifecycle.
//
//   RunInput(...)   — text input with validation
//   RunSelect(...)  — selection list
//   RunConfirm(...) — yes/no confirmation
//
// All runners are inline (no AltScreen) and return ErrCancelled on Ctrl-C/Esc.

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrCancelled is returned when the user cancels a prompt (Ctrl-C / Esc).
var ErrCancelled = errors.New("cancelled")

// =============================================================================
// RunInput — Text input with optional default and validation
// =============================================================================

// RunInput shows an inline text prompt and returns the entered value.
// - prompt: the label shown above the input
// - placeholder: ghost text inside the input field
// - defaultVal: pre-filled value (user can edit)
// - validate: optional callback; return non-nil to show inline error
//
// Returns ("", ErrCancelled) on Ctrl-C/Esc.
func RunInput(prompt, placeholder, defaultVal string, validate func(string) error) (string, error) {
	m := newInputPromptModel(prompt, placeholder, defaultVal, validate)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}
	result := finalModel.(inputPromptModel)
	if result.cancelled {
		return "", ErrCancelled
	}
	return result.value, nil
}

// inputPromptModel is the bubbletea model backing RunInput.
type inputPromptModel struct {
	textInput textinput.Model
	prompt    string
	validate  func(string) error
	errMsg    string // inline validation error
	value     string
	cancelled bool
	done      bool
}

func newInputPromptModel(prompt, placeholder, defaultVal string, validate func(string) error) inputPromptModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50
	if defaultVal != "" {
		ti.SetValue(defaultVal)
	}
	return inputPromptModel{
		textInput: ti,
		prompt:    prompt,
		validate:  validate,
	}
}

func (m inputPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.textInput.Value()
			if m.validate != nil {
				if err := m.validate(val); err != nil {
					m.errMsg = err.Error()
					return m, nil
				}
			}
			m.value = val
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		default:
			// Clear validation error on new input
			m.errMsg = ""
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputPromptModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString(TitleStyle.Render(m.prompt))
	b.WriteString("\n")
	b.WriteString(m.textInput.View())
	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Render("  ✗ " + m.errMsg))
	}
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("enter: confirm • esc: cancel"))
	return b.String()
}

// =============================================================================
// RunSelect — Lightweight selection list
// =============================================================================

// SelectItem represents a single option in a select list.
type SelectItem struct {
	Label string // Display text
	Value string // Return value (can equal Label)
}

// RunSelect shows an inline selection list and returns the chosen item.
// - label: heading shown above the list
// - items: options to choose from
// - size: max visible items (0 = auto)
//
// Returns (-1, "", ErrCancelled) on Ctrl-C/Esc.
func RunSelect(label string, items []SelectItem, size int) (int, string, error) {
	if len(items) == 0 {
		return -1, "", fmt.Errorf("no items to select from")
	}
	m := newSelectModel(label, items, size)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return -1, "", err
	}
	result := finalModel.(selectModel)
	if result.cancelled {
		return -1, "", ErrCancelled
	}
	return result.selectedIdx, result.selectedVal, nil
}

// selectModel is the bubbletea model backing RunSelect.
type selectModel struct {
	list        list.Model
	label       string
	items       []SelectItem
	selectedIdx int
	selectedVal string
	cancelled   bool
	done        bool
}

// selectListItem adapts SelectItem for the bubbles/list.
type selectListItem struct {
	label string
	value string
}

func (i selectListItem) Title() string       { return i.label }
func (i selectListItem) Description() string { return "" }
func (i selectListItem) FilterValue() string { return i.label }

func newSelectModel(label string, items []SelectItem, size int) selectModel {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = selectListItem{label: item.Label, value: item.Value}
	}

	delegate := newSelectDelegate()

	// Determine visible height
	height := len(items) + 4 // items + header/footer padding
	if size > 0 && size+4 < height {
		height = size + 4
	}
	if height > 20 {
		height = 20
	}

	l := list.New(listItems, delegate, 60, height)
	l.Title = label
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.SetFilteringEnabled(len(items) > 6)
	l.DisableQuitKeybindings()

	// Minimal styling
	l.Styles.Title = TitleStyle
	l.Styles.FilterPrompt = MutedStyle
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ColorPrimary)

	return selectModel{
		list:  l,
		label: label,
		items: items,
	}
}

// selectDelegate renders items in the select list (single-line, compact).
type selectDelegate struct{}

func newSelectDelegate() selectDelegate { return selectDelegate{} }

func (d selectDelegate) Height() int                             { return 1 }
func (d selectDelegate) Spacing() int                            { return 0 }
func (d selectDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d selectDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(selectListItem)
	if !ok {
		return
	}
	isSelected := index == m.Index()
	if isSelected {
		fmt.Fprintln(w, SelectedItemStyle.Render("▸ "+item.label))
	} else {
		fmt.Fprintln(w, NormalItemStyle.Render("  "+item.label))
	}
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't intercept keys while filtering
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "enter":
			item, ok := m.list.SelectedItem().(selectListItem)
			if ok {
				m.selectedIdx = m.list.Index()
				m.selectedVal = item.value
				m.done = true
				return m, tea.Quit
			}
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectModel) View() string {
	if m.done {
		return ""
	}
	return m.list.View() + "\n" + HelpStyle.Render("enter: select • esc: cancel")
}

// =============================================================================
// RunConfirm — Yes/No confirmation
// =============================================================================

// RunConfirm shows an inline yes/no prompt.
// - question: the text shown
// - defaultYes: whether "Yes" is pre-selected
//
// Returns (false, ErrCancelled) on Ctrl-C/Esc.
func RunConfirm(question string, defaultYes bool) (bool, error) {
	m := newConfirmPromptModel(question, defaultYes)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}
	result := finalModel.(confirmPromptModel)
	if result.cancelled {
		return false, ErrCancelled
	}
	return result.answer, nil
}

// confirmPromptModel is the bubbletea model backing RunConfirm.
type confirmPromptModel struct {
	question  string
	selected  bool // true = yes
	answer    bool
	cancelled bool
	done      bool
}

func newConfirmPromptModel(question string, defaultYes bool) confirmPromptModel {
	return confirmPromptModel{
		question: question,
		selected: defaultYes,
	}
}

func (m confirmPromptModel) Init() tea.Cmd { return nil }

func (m confirmPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h", "y":
			m.selected = true
		case "right", "l", "n":
			m.selected = false
		case "enter":
			m.answer = m.selected
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.cancelled = true
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmPromptModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render(m.question))
	b.WriteString("\n")

	yes := " Yes "
	no := " No "
	if m.selected {
		yes = SelectedItemStyle.Render(yes)
		no = NormalItemStyle.Render(no)
	} else {
		yes = NormalItemStyle.Render(yes)
		no = SelectedItemStyle.Render(no)
	}

	b.WriteString("    ")
	b.WriteString(yes)
	b.WriteString("    ")
	b.WriteString(no)
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("←/→: select • y/n: choose • enter: confirm • esc: cancel"))

	return b.String()
}
