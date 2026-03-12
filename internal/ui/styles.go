package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette - consistent across all UI
var (
	ColorPrimary   = lipgloss.Color("39")  // Cyan
	ColorSecondary = lipgloss.Color("99")  // Purple
	ColorAccent    = lipgloss.Color("205") // Pink
	ColorSuccess   = lipgloss.Color("82")  // Green
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorError     = lipgloss.Color("196") // Red
	ColorMuted     = lipgloss.Color("241") // Gray
	ColorHighlight = lipgloss.Color("230") // Light yellow
	ColorBg        = lipgloss.Color("57")  // Purple background for selection
)

// Text styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	TextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)
)

// Menu item styles
var (
	SelectedItemStyle = lipgloss.NewStyle().
				Background(ColorBg).
				Foreground(ColorHighlight).
				Bold(true).
				Padding(0, 1)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	ItemDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			PaddingLeft(3)

	CommandStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)
)

// Status header styles
var (
	StatusStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	StatusLabelStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	StatusValueStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	BreadcrumbStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginBottom(1)
)

// Box styles for panels
var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(1, 2)

	FocusedBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)
)

// Spinner style for loading states
var SpinnerStyle = lipgloss.NewStyle().Foreground(ColorPrimary)

// AI-specific styles (for streaming responses)
var (
	AIResponseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			PaddingLeft(2)

	AIThinkingStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	AICodeBlockStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("252")).
				Padding(1, 2)
)
