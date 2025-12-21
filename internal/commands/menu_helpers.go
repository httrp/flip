package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"golang.org/x/term"
)

// getTerminalSize returns width and height of terminal, with safe defaults
func getTerminalSize() (width, height int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Safe defaults if we can't detect terminal size
		return 80, 24
	}
	return w, h
}

// truncateString truncates a string to fit in terminal width, leaving room for UI elements
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// createSimpleSelectTemplates creates templates without multi-line details
func createSimpleSelectTemplates() *promptui.SelectTemplates {
	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ . | cyan | bold }}",
		Inactive: "  {{ . }}",
		Selected: "{{ . | green | bold }}",
	}
}

// MenuItem represents a menu item with label, description, command, and action
type MenuItem struct {
	Label       string
	Description string
	Command     string // CLI command for this action
	Action      func() error
}

// createMenuItemSelectTemplates creates templates for menu items with inline description
func createMenuItemSelectTemplates() *promptui.SelectTemplates {
	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ .Label | cyan | bold }}",
		Inactive: "  {{ .Label }}",
		Selected: "{{ .Label | green | bold }}",
	}
}

// createMenuItemWithCommandTemplates creates templates showing commands alongside labels
func createMenuItemWithCommandTemplates() *promptui.SelectTemplates {
	width, _ := getTerminalSize()
	// Reserve space for: prompt (4 chars) + label (variable) + command (variable) + padding (4 chars)
	maxLabelWidth := width - 50 // Reserve ~50 chars for command display
	if maxLabelWidth < 30 {
		maxLabelWidth = 30
	}

	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   fmt.Sprintf("▸ {{ printf \"%%-%ds\" .Label | cyan | bold }}  {{ .Command | faint }}", maxLabelWidth),
		Inactive: fmt.Sprintf("  {{ printf \"%%-%ds\" .Label }}  {{ .Command | faint }}", maxLabelWidth),
		Selected: "{{ .Label | green | bold }}",
		Details:  "\n{{ \"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\" | faint }}\n{{ .Description | faint }}",
	}
}

// calculateMenuSize determines optimal menu size based on terminal height
func calculateMenuSize(itemCount int) int {
	_, height := getTerminalSize()
	// Leave room for: title (2 lines), help text (2 lines), prompt (1 line), padding (3 lines)
	availableLines := height - 8
	if availableLines < 5 {
		availableLines = 5 // Minimum
	}
	if availableLines > 15 {
		availableLines = 15 // Maximum for usability
	}
	if itemCount < availableLines {
		return itemCount
	}
	return availableLines
}

// showBreadcrumb displays navigation breadcrumb
func showBreadcrumb(path ...string) {
	if len(path) == 0 {
		return
	}
	fmt.Printf("  %s\n", strings.Join(path, " › "))
	fmt.Println()
}
