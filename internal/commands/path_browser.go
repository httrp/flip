package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/httrp/flip/internal/ui"
)

// PathBrowserOption represents an option in the path browser
type PathBrowserOption struct {
	Display string
	Path    string
	IsDir   bool
}

// BrowseDirectory provides an interactive directory browser for path selection
// Returns the selected path or error if cancelled
func BrowseDirectory(startPath string, selectDirs bool) (string, error) {
	// Normalize start path
	if startPath == "" || startPath == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		startPath = home
	}

	// Expand ~ in path
	if strings.HasPrefix(startPath, "~/") {
		home, _ := os.UserHomeDir()
		startPath = filepath.Join(home, startPath[2:])
	}

	// Make absolute
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	currentPath := absPath

	for {
		// Read directory contents
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return "", fmt.Errorf("failed to read directory: %w", err)
		}

		// Build options
		var options []PathBrowserOption

		// Add parent directory option if not at root
		if currentPath != filepath.Dir(currentPath) {
			options = append(options, PathBrowserOption{
				Display: "📁 ../  (Go up)",
				Path:    filepath.Dir(currentPath),
				IsDir:   true,
			})
		}

		// Add "Select this directory" if we're selecting dirs
		if selectDirs {
			options = append(options, PathBrowserOption{
				Display: "✅ [Select this directory]",
				Path:    currentPath,
				IsDir:   true,
			})
		}

		// Add subdirectories
		var dirs []PathBrowserOption
		var files []PathBrowserOption

		for _, entry := range entries {
			// Skip hidden files/dirs unless at home
			name := entry.Name()
			if strings.HasPrefix(name, ".") && !strings.HasPrefix(currentPath, os.Getenv("HOME")) {
				continue
			}

			fullPath := filepath.Join(currentPath, name)
			isDir := entry.IsDir()

			option := PathBrowserOption{
				Path:  fullPath,
				IsDir: isDir,
			}

			if isDir {
				option.Display = fmt.Sprintf("📁 %s/", name)
				dirs = append(dirs, option)
			} else if !selectDirs {
				// Only show files if we're selecting files
				option.Display = fmt.Sprintf("📄 %s", name)
				files = append(files, option)
			}
		}

		// Sort alphabetically
		sort.Slice(dirs, func(i, j int) bool {
			return strings.ToLower(dirs[i].Display) < strings.ToLower(dirs[j].Display)
		})
		sort.Slice(files, func(i, j int) bool {
			return strings.ToLower(files[i].Display) < strings.ToLower(files[j].Display)
		})

		// Combine: parent, select, dirs, files
		options = append(options, dirs...)
		options = append(options, files...)

		// Add manual entry option
		options = append(options, PathBrowserOption{
			Display: "⌨️  [Enter path manually]",
			Path:    "",
			IsDir:   false,
		})

		// Add cancel option
		options = append(options, PathBrowserOption{
			Display: "❌ [Cancel]",
			Path:    "",
			IsDir:   false,
		})

		// Show current path and select
		fmt.Println()
		fmt.Printf("📂 Current: %s\n", currentPath)
		fmt.Println()

		selectItems := make([]ui.SelectItem, len(options))
		for i, opt := range options {
			selectItems[i] = ui.SelectItem{Label: opt.Display, Value: fmt.Sprintf("%d", i)}
		}

		idx, _, err := ui.RunSelect("Select directory or file", selectItems, calculateMenuSize(len(options)))
		if err != nil {
			return "", fmt.Errorf("selection cancelled: %w", err)
		}

		selected := options[idx]

		// Handle special options
		switch {
		case strings.Contains(selected.Display, "[Cancel]"):
			return "", fmt.Errorf("cancelled by user")

		case strings.Contains(selected.Display, "[Enter path manually]"):
			manualPath, err := ui.RunInput("Enter full path", "", currentPath, nil)
			if err != nil {
				continue // Go back to browser
			}

			// Expand and validate
			if strings.HasPrefix(manualPath, "~/") {
				home, _ := os.UserHomeDir()
				manualPath = filepath.Join(home, manualPath[2:])
			}

			absManual, err := filepath.Abs(manualPath)
			if err != nil {
				fmt.Printf("\n❌ Invalid path: %v\n", err)
				continue
			}

			// Check if path exists
			stat, err := os.Stat(absManual)
			if err != nil {
				fmt.Printf("\n❌ Path does not exist: %s\n", absManual)
				continue
			}

			// If selecting dirs, return it; otherwise continue browsing
			if selectDirs && stat.IsDir() {
				return absManual, nil
			} else if !selectDirs && !stat.IsDir() {
				return absManual, nil
			} else if selectDirs && !stat.IsDir() {
				fmt.Println("\n❌ Selected path is a file, but a directory is required")
				continue
			} else {
				// File selected but we need a dir, navigate to it
				currentPath = filepath.Dir(absManual)
				continue
			}

		case strings.Contains(selected.Display, "[Select this directory]"):
			return currentPath, nil

		case selected.IsDir:
			// Navigate into directory
			currentPath = selected.Path

		default:
			// File selected
			if selectDirs {
				fmt.Println("\n❌ Please select a directory, not a file")
				continue
			}
			return selected.Path, nil
		}
	}
}

// BrowseDirectoryWithCommonPaths provides directory browsing with shortcuts to common locations
func BrowseDirectoryWithCommonPaths(selectDirs bool, commonPaths map[string]string) (string, error) {
	// Show common paths first
	var options []string
	var pathMap = make(map[string]string)

	for label, path := range commonPaths {
		display := fmt.Sprintf("⭐ %s", label)
		options = append(options, display)
		pathMap[display] = path
	}

	// Add browse option
	browseOption := "📂 Browse filesystem..."
	options = append(options, browseOption)

	// Add manual entry
	manualOption := "⌨️  Enter path manually"
	options = append(options, manualOption)

	selectItems := make([]ui.SelectItem, len(options))
	for i, opt := range options {
		selectItems[i] = ui.SelectItem{Label: opt, Value: opt}
	}

	idx, selected, err := ui.RunSelect("Choose a starting location", selectItems, calculateMenuSize(len(options)))
	if err != nil {
		return "", fmt.Errorf("selection cancelled: %w", err)
	}
	_ = idx

	switch {
	case selected == browseOption:
		home, _ := os.UserHomeDir()
		return BrowseDirectory(home, selectDirs)

	case selected == manualOption:
		path, err := ui.RunInput("Enter full path", "", "", nil)
		if err != nil {
			return "", err
		}

		// Expand tilde
		if strings.HasPrefix(path, "~/") {
			home, _ := os.UserHomeDir()
			path = filepath.Join(home, path[2:])
		}

		return path, nil

	default:
		// Common path selected
		if path, ok := pathMap[selected]; ok {
			// If it's a shortcut, still allow browsing from there
			return BrowseDirectory(path, selectDirs)
		}
		return options[idx], nil
	}
}
