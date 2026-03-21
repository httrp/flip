package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	ui "github.com/httrp/flip/internal/ui"
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


// MenuItem represents a menu item with label, description, command, and action
type MenuItem struct {
	Label       string
	Description string
	Command     string // CLI command for this action
	Action      func() error
}

// ===== Menu Helpers =====

// runMenuItemSelect presents a menu of MenuItems using bubbletea and returns the selected index
func runMenuItemSelect(label string, items []MenuItem) (int, error) {
	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item.Label, Value: fmt.Sprintf("%d", i)}
	}
	idx, _, err := ui.RunSelect(label, selectItems, calculateMenuSize(len(items)))
	return idx, err
}

// runStringSelect presents a menu of string items using bubbletea and returns the selected index
func runStringSelect(label string, items []string) (int, error) {
	selectItems := make([]ui.SelectItem, len(items))
	for i, item := range items {
		selectItems[i] = ui.SelectItem{Label: item, Value: fmt.Sprintf("%d", i)}
	}
	idx, _, err := ui.RunSelect(label, selectItems, calculateMenuSize(len(items)))
	return idx, err
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
