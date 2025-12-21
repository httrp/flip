package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/manifoldco/promptui"
	"golang.org/x/term"
)

// ===== Platform-Independent Helpers =====

// openInFileManager opens a path in the system's file manager (cross-platform)
func openInFileManager(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	default: // Linux, BSD, etc.
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Start()
}

// openInDefaultApp opens a file with the system's default application (cross-platform)
func openInDefaultApp(filePath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", filePath)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filePath)
	default: // Linux, BSD, etc.
		cmd = exec.Command("xdg-open", filePath)
	}

	return cmd.Start()
}

// Note: For editor opening, use openInEditor() from note.go which has
// comprehensive fallback logic (VS Code → system default → $EDITOR → CLI editors)

// ===== Terminal Helpers =====

// getTerminalSize returns width and height of terminal, with safe defaults
func getTerminalSize() (width, height int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Safe defaults if we can't detect terminal size
		return 80, 24
	}
	return w, h
}

// isNarrowTerminal returns true if terminal width is less than threshold
func isNarrowTerminal(threshold int) bool {
	width, _ := getTerminalSize()
	return width < threshold
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

// truncateToFit truncates string to fit terminal width with padding
func truncateToFit(s string, reservedChars int) string {
	width, _ := getTerminalSize()
	maxLen := width - reservedChars
	if maxLen < 10 {
		maxLen = 10
	}
	return truncateString(s, maxLen)
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

// ===== Menu Templates =====

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
// Adapts to terminal width - hides commands on narrow terminals
func createMenuItemWithCommandTemplates() *promptui.SelectTemplates {
	width, height := getTerminalSize()

	// Narrow terminal mode (< 80 chars): show only label, no command
	if width < 80 {
		return &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "▸ {{ .Label | cyan | bold }}",
			Inactive: "  {{ .Label }}",
			Selected: "{{ .Label | green | bold }}",
			// Compact details for narrow screens
			Details: "{{ .Command | faint }}",
		}
	}

	// Medium terminal (80-120 chars): compact command display
	if width < 120 {
		maxLabelWidth := width - 35
		if maxLabelWidth < 25 {
			maxLabelWidth = 25
		}
		return &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   fmt.Sprintf("▸ {{ printf \"%%-%ds\" .Label | cyan | bold }} {{ .Command | faint }}", maxLabelWidth),
			Inactive: fmt.Sprintf("  {{ printf \"%%-%ds\" .Label }} {{ .Command | faint }}", maxLabelWidth),
			Selected: "{{ .Label | green | bold }}",
			// No details section to save vertical space
		}
	}

	// Wide terminal (≥120 chars): full display with description
	maxLabelWidth := 45
	// Only show details section if we have enough vertical space
	detailsSection := ""
	if height > 20 {
		detailsSection = "\n{{ \"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\" | faint }}\n{{ .Description | faint }}"
	}

	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   fmt.Sprintf("▸ {{ printf \"%%-%ds\" .Label | cyan | bold }}  {{ .Command | faint }}", maxLabelWidth),
		Inactive: fmt.Sprintf("  {{ printf \"%%-%ds\" .Label }}  {{ .Command | faint }}", maxLabelWidth),
		Selected: "{{ .Label | green | bold }}",
		Details:  detailsSection,
	}
}

// createCompactSelectTemplates creates minimal templates for quick selection
func createCompactSelectTemplates() *promptui.SelectTemplates {
	return &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "{{ . | green }}",
	}
}

// ===== Menu Size Helpers =====

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

// ===== Navigation Helpers =====

// showBreadcrumb displays navigation breadcrumb
func showBreadcrumb(path ...string) {
	if len(path) == 0 {
		return
	}
	// Truncate breadcrumb for narrow terminals
	width, _ := getTerminalSize()
	breadcrumb := strings.Join(path, " › ")
	if len(breadcrumb) > width-4 {
		// Show only last 2 parts if too long
		if len(path) > 2 {
			path = path[len(path)-2:]
			breadcrumb = "… › " + strings.Join(path, " › ")
		}
		breadcrumb = truncateString(breadcrumb, width-4)
	}
	fmt.Printf("  %s\n", breadcrumb)
	fmt.Println()
}
