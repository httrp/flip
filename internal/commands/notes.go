package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// NoteFile represents a note file with metadata
type NoteFile struct {
	Path         string
	RelativePath string
	BrainName    string
	ModTime      time.Time
	Size         int64
}

func NewRecentCommand() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "recent",
		Short: "Show recently modified notes across all brains",
		Long:  "Lists the most recently modified notes across all brains in the active workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showRecentNotes(limit)
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "Number of recent notes to show")

	return cmd
}

func NewSearchCommand() *cobra.Command {
	var searchContent bool

	cmd := &cobra.Command{
		Use:   "search [keywords...]",
		Short: "Search for notes across all brains",
		Long:  "Search for notes by filename or content across all brains in the active workspace.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			return searchNotes(query, searchContent)
		},
	}

	cmd.Flags().BoolVarP(&searchContent, "content", "c", false, "Search in file content (slower)")

	return cmd
}

func showRecentNotes(limit int) error {
	// Load workspace config
	config, err := loadWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load workspace: %w", err)
	}

	if config.ActiveWorkspace == "" {
		return fmt.Errorf("no active workspace")
	}

	// Find active workspace
	var activeWs *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWs = &config.Workspaces[i]
			break
		}
	}

	if activeWs == nil {
		return fmt.Errorf("active workspace not found")
	}

	if len(activeWs.Brains) == 0 {
		return fmt.Errorf("no brains in workspace")
	}

	// Collect all notes from all brains
	fmt.Println("📝 Scanning notes across workspace...")
	var allNotes []NoteFile

	for _, brain := range activeWs.Brains {
		notes, err := collectNotesFromBrain(brain.Path, brain.Name)
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to scan brain '%s': %v\n", brain.Name, err)
			continue
		}
		allNotes = append(allNotes, notes...)
	}

	if len(allNotes) == 0 {
		fmt.Println("\n✗ No notes found in workspace")
		return nil
	}

	// Sort by modification time (newest first)
	sort.Slice(allNotes, func(i, j int) bool {
		return allNotes[i].ModTime.After(allNotes[j].ModTime)
	})

	// Limit results
	if len(allNotes) > limit {
		allNotes = allNotes[:limit]
	}

	// Display and allow selection
	return displayAndSelectNotes(allNotes, fmt.Sprintf("Recent Notes (Top %d)", limit))
}

func searchNotes(query string, searchContent bool) error {
	// Load workspace config
	config, err := loadWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load workspace: %w", err)
	}

	if config.ActiveWorkspace == "" {
		return fmt.Errorf("no active workspace")
	}

	// Find active workspace
	var activeWs *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWs = &config.Workspaces[i]
			break
		}
	}

	if activeWs == nil {
		return fmt.Errorf("active workspace not found")
	}

	if len(activeWs.Brains) == 0 {
		return fmt.Errorf("no brains in workspace")
	}

	// Collect all notes from all brains
	searchType := "filename"
	if searchContent {
		searchType = "filename and content"
	}
	fmt.Printf("🔍 Searching for '%s' in %s across workspace...\n", query, searchType)

	var matchingNotes []NoteFile
	queryLower := strings.ToLower(query)

	for _, brain := range activeWs.Brains {
		notes, err := collectNotesFromBrain(brain.Path, brain.Name)
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to scan brain '%s': %v\n", brain.Name, err)
			continue
		}

		for _, note := range notes {
			// Check filename
			if strings.Contains(strings.ToLower(filepath.Base(note.Path)), queryLower) {
				matchingNotes = append(matchingNotes, note)
				continue
			}

			// Check content if requested
			if searchContent {
				if contentMatches(note.Path, queryLower) {
					matchingNotes = append(matchingNotes, note)
				}
			}
		}
	}

	if len(matchingNotes) == 0 {
		fmt.Printf("\n✗ No notes found matching '%s'\n", query)
		return nil
	}

	// Sort by modification time (newest first)
	sort.Slice(matchingNotes, func(i, j int) bool {
		return matchingNotes[i].ModTime.After(matchingNotes[j].ModTime)
	})

	// Display and allow selection
	return displayAndSelectNotes(matchingNotes, fmt.Sprintf("Search Results for '%s'", query))
}

func collectNotesFromBrain(brainPath, brainName string) ([]NoteFile, error) {
	var notes []NoteFile

	// NOTE: We use filesystem ModTime for performance.
	// Git log calls are too slow for large repositories (can take minutes).
	// Future: Add --use-git-time flag for users who need accurate git timestamps.

	// Supported note extensions
	noteExtensions := map[string]bool{
		".md":       true,
		".markdown": true,
		".txt":      true,
		".org":      true,
		".rst":      true,
	}

	err := filepath.Walk(brainPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories
		if info.IsDir() {
			// Skip common exclude directories
			name := info.Name()
			if strings.HasPrefix(name, ".") ||
				name == "node_modules" ||
				name == "vendor" ||
				name == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's a note file
		ext := strings.ToLower(filepath.Ext(path))
		if !noteExtensions[ext] {
			return nil
		}

		// Get relative path from brain root
		relPath, _ := filepath.Rel(brainPath, path)

		notes = append(notes, NoteFile{
			Path:         path,
			RelativePath: relPath,
			BrainName:    brainName,
			ModTime:      info.ModTime(),
			Size:         info.Size(),
		})

		return nil
	})

	return notes, err
}

func contentMatches(filePath, query string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	return strings.Contains(strings.ToLower(string(content)), query)
}

func displayAndSelectNotes(notes []NoteFile, title string) error {
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	fmt.Printf("✓ Found %d note(s)\n\n", len(notes))

	// Create menu items
	type MenuItem struct {
		Display string
		Index   int
	}

	var items []MenuItem
	for i, note := range notes {
		// Format modification time
		timeStr := formatRelativeTime(note.ModTime)

		// Build display string
		display := fmt.Sprintf("%-40s  %s  [%s]  %s",
			truncate(note.RelativePath, 40),
			timeStr,
			note.BrainName,
			formatFileSize(note.Size),
		)

		items = append(items, MenuItem{Display: display, Index: i})
	}
	items = append(items, MenuItem{Display: "◀️  Back", Index: -1})

	// Interactive selection
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ .Display | cyan | bold }}",
		Inactive: "  {{ .Display }}",
		Selected: "{{ .Display | green | bold }}",
	}

	prompt := promptui.Select{
		Label:     title,
		Items:     items,
		Templates: templates,
		Size:      15,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return nil
	}

	selectedIdx := items[idx].Index

	// Exit option selected
	if selectedIdx == -1 {
		return nil
	}

	selectedNote := notes[selectedIdx]

	// Show note details
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Note Details")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("File:         %s\n", selectedNote.RelativePath)
	fmt.Printf("Brain:        %s\n", selectedNote.BrainName)
	fmt.Printf("Full Path:    %s\n", selectedNote.Path)
	fmt.Printf("Modified:     %s (%s)\n", selectedNote.ModTime.Format("2006-01-02 15:04:05"), formatRelativeTime(selectedNote.ModTime))
	fmt.Printf("Size:         %s\n", formatFileSize(selectedNote.Size))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Ask what to do
	actionPrompt := promptui.Select{
		Label: "What would you like to do?",
		Items: []string{
			"📝 Open in editor",
			"📋 Copy path to clipboard",
			"◀️  Back to list",
		},
	}

	actionIdx, _, err := actionPrompt.Run()
	if err != nil {
		return nil
	}

	switch actionIdx {
	case 0: // Open in editor
		return openInEditor(selectedNote.Path)
	case 1: // Copy to clipboard
		fmt.Printf("Path: %s\n", selectedNote.Path)
		fmt.Println("(Path displayed above - copy manually)")
		return nil
	case 2: // Back to list
		return displayAndSelectNotes(notes, title)
	}

	return nil
}

func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}
	if duration < 365*24*time.Hour {
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}

	years := int(duration.Hours() / 24 / 365)
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
