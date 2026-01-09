package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/httrp/flip/internal/git"
	"github.com/httrp/flip/internal/tasks"
	"github.com/spf13/cobra"
)

// TasksVersion is incremented when tasks.json changes
// This allows flip to detect outdated installations
const TasksVersion = "1.1.0"

// VSCodeTasksJSON contains the embedded tasks configuration
// This is the single source of truth for flip's VS Code tasks
var VSCodeTasksJSON = `{
  "version": "2.0.0",
  "_flipTasksVersion": "` + TasksVersion + `",
  "tasks": [
    {
      "label": "Flip: Menu",
      "type": "shell",
      "command": "flip",
      "args": ["menu"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: File Info",
      "type": "shell",
      "command": "flip",
      "args": ["file-info", "${file}"],
      "presentation": {
        "reveal": "always",
        "panel": "shared",
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Journal (Today)",
      "type": "shell",
      "command": "flip",
      "args": ["journal"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: New Task",
      "type": "shell",
      "command": "flip",
      "args": ["task", "new"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Task Done",
      "type": "shell",
      "command": "flip",
      "args": ["task", "done"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Task List",
      "type": "shell",
      "command": "flip",
      "args": ["task", "list"],
      "presentation": {
        "reveal": "always",
        "panel": "shared",
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: New Note",
      "type": "shell",
      "command": "flip",
      "args": ["note"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Quick Note",
      "type": "shell",
      "command": "flip",
      "args": ["quicknote"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Search Notes",
      "type": "shell",
      "command": "flip",
      "args": ["note", "search"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: New Exercise",
      "type": "shell",
      "command": "flip",
      "args": ["exercise", "new"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Track Exercise",
      "type": "shell",
      "command": "flip",
      "args": ["exercise", "track"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: List Exercises",
      "type": "shell",
      "command": "flip",
      "args": ["exercise", "list"],
      "presentation": {
        "reveal": "always",
        "panel": "shared",
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Status",
      "type": "shell",
      "command": "flip",
      "args": ["status"],
      "presentation": {
        "reveal": "always",
        "panel": "shared",
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Meeting Note",
      "type": "shell",
      "command": "flip",
      "args": ["meeting-note"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Definitions",
      "type": "shell",
      "command": "flip",
      "args": ["definitions"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Switch Brain",
      "type": "shell",
      "command": "flip",
      "args": ["brain", "switch"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Brain Info",
      "type": "shell",
      "command": "flip",
      "args": ["brain", "info"],
      "presentation": {
        "reveal": "always",
        "panel": "shared",
        "clear": true
      },
      "problemMatcher": []
    }
  ]
}`

// NewVSCodeCommand creates the vscode command group
func NewVSCodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vscode",
		Short: "VS Code integration",
		Long:  "Install, update, or remove flip tasks and extension for VS Code",
	}

	// Tasks commands
	cmd.AddCommand(newVSCodeInstallCommand())
	cmd.AddCommand(newVSCodeStatusCommand())
	cmd.AddCommand(newVSCodeUninstallCommand())
	cmd.AddCommand(newVSCodeInfoCommand())
	cmd.AddCommand(newVSCodeNotesCommand())
	cmd.AddCommand(newVSCodeTasksCommand())
	cmd.AddCommand(newVSCodeSearchCommand())
	cmd.AddCommand(newVSCodeRecentCommand())
	cmd.AddCommand(newVSCodeSyncCommand())
	cmd.AddCommand(NewVSCodeMeetingsCommand())
	cmd.AddCommand(NewVSCodeDefinitionsCommand())

	// Extension commands
	cmd.AddCommand(newVSCodeExtensionInstallCommand())
	cmd.AddCommand(newVSCodeExtensionStatusCommand())
	cmd.AddCommand(newVSCodeExtensionUninstallCommand())

	return cmd
}

func newVSCodeInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Output flip info for VS Code extension (JSON)",
		Long:  "Returns JSON with all information needed by the VS Code extension",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVSCodeInfo()
		},
	}
	// Accept --json flag (for consistency, output is always JSON)
	cmd.Flags().Bool("json", false, "Output as JSON (default behavior)")
	return cmd
}

// NoteInfo represents a note with its link format
type NoteInfo struct {
	Name       string `json:"name"`        // Display name
	Path       string `json:"path"`        // Full file path
	RelPath    string `json:"rel_path"`    // Relative path from brain root
	LinkFormat string `json:"link_format"` // Ready-to-insert link format
	BrainName  string `json:"brain_name"`  // Which brain this note belongs to
	BrainType  string `json:"brain_type"`  // Type of brain (logseq, foam, etc.)
}

// NotesResult is returned by vscode notes command
type NotesResult struct {
	Notes      []NoteInfo `json:"notes"`
	BrainName  string     `json:"brain_name"`
	BrainType  string     `json:"brain_type"`
	BrainPath  string     `json:"brain_path"`
	LinkSyntax string     `json:"link_syntax"` // e.g., "[[name]]" or "[name](path)"
}

func newVSCodeNotesCommand() *cobra.Command {
	var brainName string

	cmd := &cobra.Command{
		Use:   "notes",
		Short: "List notes with link formats for VS Code extension",
		Long:  "Returns JSON list of notes with appropriate link format for each brain type",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVSCodeNotes(brainName)
		},
	}
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain to list notes from (default: active)")
	cmd.Flags().Bool("json", false, "Output as JSON (default behavior)")
	return cmd
}

func runVSCodeNotes(brainName string) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		OutputJSONError("vscode-notes", err)
		return nil
	}

	// Find the target brain
	var targetBrain *Brain
	for _, ws := range config.Workspaces {
		if ws.Name == config.ActiveWorkspace {
			for i, b := range ws.Brains {
				if brainName == "" && b.Name == ws.DefaultBrain {
					targetBrain = &ws.Brains[i]
					break
				} else if b.Name == brainName {
					targetBrain = &ws.Brains[i]
					break
				}
			}
			break
		}
	}

	if targetBrain == nil {
		OutputJSONError("vscode-notes", fmt.Errorf("brain not found"))
		return nil
	}

	result := NotesResult{
		Notes:     []NoteInfo{},
		BrainName: targetBrain.Name,
		BrainType: targetBrain.Type,
		BrainPath: targetBrain.Path,
	}

	// Set link syntax based on brain type
	switch targetBrain.Type {
	case "logseq":
		result.LinkSyntax = "[[Page Name]]"
	case "dendron":
		result.LinkSyntax = "[[hierarchy.note]]"
	case "foam", "obsidian":
		result.LinkSyntax = "[[note-name]]"
	default: // flip and others
		result.LinkSyntax = "[[note-name]]"
	}

	// Find all markdown files
	err = filepath.Walk(targetBrain.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden directories and common non-note directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only include .md files
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		relPath, _ := filepath.Rel(targetBrain.Path, path)
		name := strings.TrimSuffix(filepath.Base(path), ".md")

		// Generate link format based on brain type
		var linkFormat string
		switch targetBrain.Type {
		case "logseq":
			// Logseq uses page names, journals have special format
			if strings.Contains(relPath, "journals") {
				// Convert 2026_01_06 to readable format
				linkFormat = "[[" + name + "]]"
			} else {
				linkFormat = "[[" + name + "]]"
			}
		case "dendron":
			// Dendron uses dot-separated hierarchy
			hierarchy := strings.ReplaceAll(relPath, string(filepath.Separator), ".")
			hierarchy = strings.TrimSuffix(hierarchy, ".md")
			linkFormat = "[[" + hierarchy + "]]"
		case "foam", "obsidian":
			// Foam/Obsidian use wiki-style links
			linkFormat = "[[" + name + "]]"
		default: // flip
			linkFormat = "[[" + name + "]]"
		}

		result.Notes = append(result.Notes, NoteInfo{
			Name:       name,
			Path:       path,
			RelPath:    relPath,
			LinkFormat: linkFormat,
			BrainName:  targetBrain.Name,
			BrainType:  targetBrain.Type,
		})

		return nil
	})

	if err != nil {
		OutputJSONError("vscode-notes", err)
		return nil
	}

	OutputJSONSuccess("vscode-notes", result)
	return nil
}

// TaskInfo represents a task for VS Code extension
type TaskInfo struct {
	Description  string   `json:"description"`
	Status       string   `json:"status"` // "open", "in-progress", "done"
	Priority     string   `json:"priority,omitempty"`
	Due          string   `json:"due,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Frog         bool     `json:"frog,omitempty"`
	Path         string   `json:"path"`
	RelPath      string   `json:"rel_path"`
	Line         int      `json:"line"`
	BrainName    string   `json:"brain_name"`
	BrainType    string   `json:"brain_type"`
	Organization string   `json:"organization,omitempty"`
	Project      string   `json:"project,omitempty"`
	ContextTag   string   `json:"context,omitempty"`
}

// TasksResult is returned by vscode tasks command
type TasksResult struct {
	Tasks      []TaskInfo `json:"tasks"`
	TotalCount int        `json:"total_count"`
	BrainName  string     `json:"brain_name,omitempty"`
	BrainPath  string     `json:"brain_path,omitempty"`
	Query      string     `json:"query,omitempty"`
}

func newVSCodeTasksCommand() *cobra.Command {
	var brainName string
	var status string
	var priority string
	var frogOnly bool
	var query string

	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "List tasks for VS Code extension (JSON)",
		Long:  "Returns JSON list of tasks, optionally filtered",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build query from args if provided
			if len(args) > 0 && query == "" {
				query = strings.Join(args, " ")
			}
			return runVSCodeTasks(brainName, status, priority, frogOnly, query)
		},
	}
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain to list tasks from (default: all brains)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (open, in-progress, done)")
	cmd.Flags().StringVar(&priority, "priority", "", "Filter by priority (high, medium, low)")
	cmd.Flags().BoolVar(&frogOnly, "frog", false, "Show only 🐸 eat-the-frog tasks")
	cmd.Flags().StringVar(&query, "query", "", "Search query (matches description)")
	cmd.Flags().Bool("json", false, "Output as JSON (default behavior)")
	return cmd
}

func runVSCodeTasks(brainName, status, priority string, frogOnly bool, query string) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		OutputJSONError("vscode-tasks", err)
		return nil
	}

	result := TasksResult{
		Tasks: []TaskInfo{},
		Query: query,
	}

	// Find brains to scan
	var brainsToScan []Brain
	for _, ws := range config.Workspaces {
		if ws.Name == config.ActiveWorkspace {
			for _, b := range ws.Brains {
				if brainName == "" || b.Name == brainName {
					brainsToScan = append(brainsToScan, b)
				}
			}
			break
		}
	}

	if len(brainsToScan) == 0 {
		OutputJSONError("vscode-tasks", fmt.Errorf("no brains found"))
		return nil
	}

	// If single brain specified, include in result
	if brainName != "" && len(brainsToScan) == 1 {
		result.BrainName = brainsToScan[0].Name
		result.BrainPath = brainsToScan[0].Path
	}

	// Scan each brain
	for _, brain := range brainsToScan {
		scanner := tasks.NewScanner(brain.Path)
		brainTasks, err := scanner.ScanBrain()
		if err != nil {
			continue // Skip brain on error
		}

		for _, t := range brainTasks {
			// Apply filters
			if status != "" {
				taskStatus := statusString(t.Status)
				if taskStatus != status {
					continue
				}
			}

			if priority != "" {
				taskPriority := priorityString(t.Priority)
				if taskPriority != priority {
					continue
				}
			}

			if frogOnly && !t.Frog {
				continue
			}

			if query != "" {
				if !strings.Contains(strings.ToLower(t.Description), strings.ToLower(query)) {
					continue
				}
			}

			relPath, _ := filepath.Rel(brain.Path, t.Context.FilePath)
			taskInfo := TaskInfo{
				Description:  t.Description,
				Status:       statusString(t.Status),
				Priority:     priorityString(t.Priority),
				Tags:         t.Tags,
				Frog:         t.Frog,
				Path:         t.Context.FilePath,
				RelPath:      relPath,
				Line:         t.Context.LineNumber,
				BrainName:    brain.Name,
				BrainType:    brain.Type,
				Organization: t.Organization,
				Project:      t.Project,
				ContextTag:   t.ContextTag,
			}

			if t.Due != nil {
				taskInfo.Due = t.Due.Format("2006-01-02")
			}

			result.Tasks = append(result.Tasks, taskInfo)
		}
	}

	result.TotalCount = len(result.Tasks)
	OutputJSONSuccess("vscode-tasks", result)
	return nil
}

// statusString converts tasks.Status to string
func statusString(s tasks.Status) string {
	switch s {
	case tasks.StatusOpen:
		return "open"
	case tasks.StatusInProgress:
		return "in-progress"
	case tasks.StatusDone:
		return "done"
	default:
		return "open"
	}
}

// priorityString converts tasks.Priority to string
func priorityString(p tasks.Priority) string {
	switch p {
	case tasks.PriorityHigh:
		return "high"
	case tasks.PriorityMedium:
		return "medium"
	case tasks.PriorityLow:
		return "low"
	default:
		return ""
	}
}

// VSCodeSearchItem represents a search result for VS Code extension
type VSCodeSearchItem struct {
	Title     string   `json:"title"`
	Path      string   `json:"path"`
	RelPath   string   `json:"rel_path"`
	BrainName string   `json:"brain_name"`
	BrainType string   `json:"brain_type"`
	Type      string   `json:"type,omitempty"` // note, journal, meeting, task
	Tags      []string `json:"tags,omitempty"`
	Modified  string   `json:"modified,omitempty"`   // ISO date
	Created   string   `json:"created,omitempty"`    // ISO date
	MatchLine int      `json:"match_line,omitempty"` // Line number of match
	MatchText string   `json:"match_text,omitempty"` // Snippet around match
	Score     float64  `json:"score,omitempty"`      // Relevance score
}

// VSCodeSearchResults is returned by vscode search command
type VSCodeSearchResults struct {
	Results    []VSCodeSearchItem `json:"results"`
	TotalCount int                `json:"total_count"`
	Query      string             `json:"query,omitempty"`
	SearchType string             `json:"search_type"` // "fulltext", "recent", "tag"
}

func newVSCodeSearchCommand() *cobra.Command {
	var query string
	var tag string
	var brainName string
	var limit int
	var noteType string

	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search notes across all brains (JSON)",
		Long:  "Full-text search across all brains in the active workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build query from args if provided
			if len(args) > 0 && query == "" {
				query = strings.Join(args, " ")
			}
			return runVSCodeSearch(query, tag, brainName, noteType, limit)
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "Search query (matches title and content)")
	cmd.Flags().StringVar(&tag, "tag", "", "Filter by tag")
	cmd.Flags().StringVar(&brainName, "brain", "", "Search in specific brain only")
	cmd.Flags().StringVar(&noteType, "type", "", "Filter by type (note, journal, meeting)")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum results to return")
	cmd.Flags().Bool("json", false, "Output as JSON (default behavior)")
	return cmd
}

func newVSCodeRecentCommand() *cobra.Command {
	var brainName string
	var limit int
	var noteType string

	cmd := &cobra.Command{
		Use:   "recent",
		Short: "List recently modified notes (JSON)",
		Long:  "Returns notes sorted by modification time, most recent first",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVSCodeRecent(brainName, noteType, limit)
		},
	}
	cmd.Flags().StringVar(&brainName, "brain", "", "Filter by specific brain")
	cmd.Flags().StringVar(&noteType, "type", "", "Filter by type (note, journal, meeting)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum results to return")
	cmd.Flags().Bool("json", false, "Output as JSON (default behavior)")
	return cmd
}

func newVSCodeSyncCommand() *cobra.Command {
	var brainName string
	var push bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Commit and optionally push changes in brains (JSON)",
		Long:  "Commits all uncommitted changes in brains with auto-generated messages. Optionally pushes to remote.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVSCodeSync(brainName, push)
		},
	}
	cmd.Flags().StringVar(&brainName, "brain", "", "Sync specific brain only")
	cmd.Flags().BoolVar(&push, "push", false, "Push to remote after commit")
	cmd.Flags().Bool("json", false, "Output as JSON (default behavior)")
	return cmd
}

func runVSCodeSearch(query, tag, brainName, noteType string, limit int) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		OutputJSONError("vscode-search", err)
		return nil
	}

	result := VSCodeSearchResults{
		Results:    []VSCodeSearchItem{},
		Query:      query,
		SearchType: "fulltext",
	}

	if tag != "" {
		result.SearchType = "tag"
	}

	// Find brains to search
	var brainsToSearch []Brain
	for _, ws := range config.Workspaces {
		if ws.Name == config.ActiveWorkspace {
			for _, b := range ws.Brains {
				if brainName == "" || b.Name == brainName {
					brainsToSearch = append(brainsToSearch, b)
				}
			}
			break
		}
	}

	if len(brainsToSearch) == 0 {
		OutputJSONError("vscode-search", fmt.Errorf("no brains found"))
		return nil
	}

	// Search each brain
	for _, brain := range brainsToSearch {
		results, err := searchBrain(brain, query, tag, noteType, limit)
		if err != nil {
			continue // Skip brain on error
		}
		result.Results = append(result.Results, results...)
	}

	// Sort by relevance (score) or modification time
	// For now, just limit results
	if len(result.Results) > limit {
		result.Results = result.Results[:limit]
	}

	result.TotalCount = len(result.Results)
	OutputJSONSuccess("vscode-search", result)
	return nil
}

func runVSCodeRecent(brainName, noteType string, limit int) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		OutputJSONError("vscode-recent", err)
		return nil
	}

	result := VSCodeSearchResults{
		Results:    []VSCodeSearchItem{},
		SearchType: "recent",
	}

	// Find brains to search
	var brainsToSearch []Brain
	for _, ws := range config.Workspaces {
		if ws.Name == config.ActiveWorkspace {
			for _, b := range ws.Brains {
				if brainName == "" || b.Name == brainName {
					brainsToSearch = append(brainsToSearch, b)
				}
			}
			break
		}
	}

	if len(brainsToSearch) == 0 {
		OutputJSONError("vscode-recent", fmt.Errorf("no brains found"))
		return nil
	}

	// Collect all markdown files with mod times
	type fileWithTime struct {
		result  VSCodeSearchItem
		modTime int64
	}
	var allFiles []fileWithTime

	for _, brain := range brainsToSearch {
		err := filepath.Walk(brain.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			// Skip hidden directories
			if info.IsDir() {
				name := info.Name()
				if strings.HasPrefix(name, ".") || name == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}

			// Only markdown files
			if !strings.HasSuffix(strings.ToLower(path), ".md") {
				return nil
			}

			// Determine note type from path
			detectedType := detectNoteType(path, brain.Path)
			if noteType != "" && detectedType != noteType {
				return nil
			}

			relPath, _ := filepath.Rel(brain.Path, path)
			title := strings.TrimSuffix(filepath.Base(path), ".md")

			// Try to extract title from frontmatter
			if extractedTitle := extractTitleFromFile(path); extractedTitle != "" {
				title = extractedTitle
			}

			allFiles = append(allFiles, fileWithTime{
				result: VSCodeSearchItem{
					Title:     title,
					Path:      path,
					RelPath:   relPath,
					BrainName: brain.Name,
					BrainType: brain.Type,
					Type:      detectedType,
					Modified:  info.ModTime().Format("2006-01-02T15:04:05"),
				},
				modTime: info.ModTime().Unix(),
			})

			return nil
		})
		if err != nil {
			continue
		}
	}

	// Sort by modification time (newest first)
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].modTime > allFiles[j].modTime
	})

	// Take top N
	for i, f := range allFiles {
		if i >= limit {
			break
		}
		result.Results = append(result.Results, f.result)
	}

	result.TotalCount = len(result.Results)
	OutputJSONSuccess("vscode-recent", result)
	return nil
}

// VSCodeSyncResult represents the result of a sync operation
type VSCodeSyncResult struct {
	Brains []VSCodeBrainSyncResult `json:"brains"`
}

// VSCodeBrainSyncResult represents sync result for a single brain
type VSCodeBrainSyncResult struct {
	Name          string   `json:"name"`
	Path          string   `json:"path"`
	HasChanges    bool     `json:"has_changes"`
	ChangedFiles  []string `json:"changed_files,omitempty"`
	Committed     bool     `json:"committed"`
	CommitMessage string   `json:"commit_message,omitempty"`
	Pushed        bool     `json:"pushed"`
	Error         string   `json:"error,omitempty"`
}

func runVSCodeSync(brainName string, push bool) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		OutputJSONError("vscode-sync", err)
		return nil
	}

	if config.ActiveWorkspace == "" {
		OutputJSONError("vscode-sync", fmt.Errorf("no active workspace"))
		return nil
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
		OutputJSONError("vscode-sync", fmt.Errorf("active workspace not found"))
		return nil
	}

	result := VSCodeSyncResult{
		Brains: []VSCodeBrainSyncResult{},
	}

	for _, brain := range activeWs.Brains {
		// Filter by brain name if specified
		if brainName != "" && brain.Name != brainName {
			continue
		}

		brainResult := VSCodeBrainSyncResult{
			Name: brain.Name,
			Path: brain.Path,
		}

		// Check if git repo
		if !git.IsGitRepo(brain.Path) {
			brainResult.HasChanges = false
			brainResult.Error = "not a git repository"
			result.Brains = append(result.Brains, brainResult)
			continue
		}

		// Check for uncommitted changes
		if !git.HasUncommittedChanges(brain.Path) {
			brainResult.HasChanges = false
			result.Brains = append(result.Brains, brainResult)
			continue
		}

		brainResult.HasChanges = true

		// Get changed files
		changes, err := git.GetChangedFiles(brain.Path)
		if err != nil {
			brainResult.Error = fmt.Sprintf("failed to get changed files: %v", err)
			result.Brains = append(result.Brains, brainResult)
			continue
		}
		brainResult.ChangedFiles = changes

		// Generate commit message
		commitMsg := generateSmartCommitMessage(changes)
		brainResult.CommitMessage = commitMsg

		// Commit
		err = git.AddAndCommit(brain.Path, commitMsg)
		if err != nil {
			brainResult.Error = fmt.Sprintf("failed to commit: %v", err)
			result.Brains = append(result.Brains, brainResult)
			continue
		}
		brainResult.Committed = true

		// Push if requested
		if push {
			err = git.Pull(brain.Path) // Pull first to avoid conflicts
			if err != nil {
				brainResult.Error = fmt.Sprintf("failed to pull before push: %v", err)
				result.Brains = append(result.Brains, brainResult)
				continue
			}

			err = gitPush(brain.Path)
			if err != nil {
				brainResult.Error = fmt.Sprintf("failed to push: %v", err)
				result.Brains = append(result.Brains, brainResult)
				continue
			}
			brainResult.Pushed = true
		}

		result.Brains = append(result.Brains, brainResult)
	}

	OutputJSONSuccess("vscode-sync", result)
	return nil
}

// gitPush pushes to remote
func gitPush(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "push")
	return cmd.Run()
}

func searchBrain(brain Brain, query, tag, noteType string, limit int) ([]VSCodeSearchItem, error) {
	var results []VSCodeSearchItem
	queryLower := strings.ToLower(query)

	err := filepath.Walk(brain.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden directories
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only markdown files
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		// Determine note type from path
		detectedType := detectNoteType(path, brain.Path)
		if noteType != "" && detectedType != noteType {
			return nil
		}

		relPath, _ := filepath.Rel(brain.Path, path)
		title := strings.TrimSuffix(filepath.Base(path), ".md")

		// Read file content for search
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		contentStr := string(content)

		// Extract title from frontmatter
		if extractedTitle := extractTitleFromContent(contentStr); extractedTitle != "" {
			title = extractedTitle
		}

		// Extract tags from content
		tags := extractTagsFromContent(contentStr)

		// Tag filter
		if tag != "" {
			hasTag := false
			for _, t := range tags {
				if strings.EqualFold(t, tag) {
					hasTag = true
					break
				}
			}
			if !hasTag {
				return nil
			}
		}

		// Query matching
		if query != "" {
			titleMatch := strings.Contains(strings.ToLower(title), queryLower)
			contentMatch := strings.Contains(strings.ToLower(contentStr), queryLower)

			if !titleMatch && !contentMatch {
				return nil
			}

			// Find match line and snippet
			matchLine := 0
			matchText := ""
			if contentMatch {
				lines := strings.Split(contentStr, "\n")
				for i, line := range lines {
					if strings.Contains(strings.ToLower(line), queryLower) {
						matchLine = i + 1
						matchText = strings.TrimSpace(line)
						if len(matchText) > 100 {
							matchText = matchText[:100] + "..."
						}
						break
					}
				}
			}

			results = append(results, VSCodeSearchItem{
				Title:     title,
				Path:      path,
				RelPath:   relPath,
				BrainName: brain.Name,
				BrainType: brain.Type,
				Type:      detectedType,
				Tags:      tags,
				Modified:  info.ModTime().Format("2006-01-02T15:04:05"),
				MatchLine: matchLine,
				MatchText: matchText,
			})
		} else if tag != "" {
			// Tag-only search
			results = append(results, VSCodeSearchItem{
				Title:     title,
				Path:      path,
				RelPath:   relPath,
				BrainName: brain.Name,
				BrainType: brain.Type,
				Type:      detectedType,
				Tags:      tags,
				Modified:  info.ModTime().Format("2006-01-02T15:04:05"),
			})
		}

		return nil
	})

	return results, err
}

func detectNoteType(path, brainPath string) string {
	relPath := strings.ToLower(path)

	if strings.Contains(relPath, "journal") || strings.Contains(relPath, "daily") {
		return "journal"
	}
	if strings.Contains(relPath, "meeting") {
		return "meeting"
	}
	if strings.Contains(relPath, "task") {
		return "task"
	}
	return "note"
}

func extractTitleFromFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return extractTitleFromContent(string(content))
}

func extractTitleFromContent(content string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				break
			}
		}

		if inFrontmatter && strings.HasPrefix(line, "title:") {
			title := strings.TrimPrefix(line, "title:")
			title = strings.TrimSpace(title)
			title = strings.Trim(title, "\"'")
			return title
		}

		// Logseq style
		if strings.HasPrefix(line, "- title::") {
			title := strings.TrimPrefix(line, "- title::")
			return strings.TrimSpace(title)
		}
	}

	// Fallback: first # heading
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}

	return ""
}

func extractTagsFromContent(content string) []string {
	var tags []string
	seen := make(map[string]bool)

	// Find #tags in content (but not in code blocks)
	tagPattern := regexp.MustCompile(`(?:^|\s)#([a-zA-Z][a-zA-Z0-9_-]*)`)
	matches := tagPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tag := strings.ToLower(match[1])
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}

	// Also check frontmatter tags
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				break
			}
		}

		if inFrontmatter && strings.HasPrefix(trimmed, "tags:") {
			// Parse tags: [tag1, tag2] or tags: tag1, tag2
			tagStr := strings.TrimPrefix(trimmed, "tags:")
			tagStr = strings.TrimSpace(tagStr)
			tagStr = strings.Trim(tagStr, "[]")

			parts := strings.Split(tagStr, ",")
			for _, p := range parts {
				tag := strings.TrimSpace(p)
				tag = strings.ToLower(tag)
				if tag != "" && !seen[tag] {
					tags = append(tags, tag)
					seen[tag] = true
				}
			}
		}
	}

	return tags
}

func runVSCodeInfo() error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		OutputJSONError("vscode-info", err)
		return nil
	}

	info := VSCodeInfo{
		FlipVersion:     FlipVersion,
		ActiveWorkspace: config.ActiveWorkspace,
		Brains:          []BrainInfo{},
		Commands: []string{
			"journal", "note", "quicknote", "meeting-note",
			"search", "recent", "status", "brain switch",
			"task new", "task done", "task list",
		},
	}

	// Find active workspace and its brains
	for _, ws := range config.Workspaces {
		if ws.Name == config.ActiveWorkspace {
			info.ActiveBrain = ws.DefaultBrain

			for _, b := range ws.Brains {
				brainInfo := BrainInfo{
					Name:   b.Name,
					Path:   b.Path,
					Type:   b.Type,
					Active: b.Name == ws.DefaultBrain,
				}
				info.Brains = append(info.Brains, brainInfo)
			}
			break
		}
	}

	OutputJSONSuccess("vscode-info", info)
	return nil
}

func newVSCodeInstallCommand() *cobra.Command {
	var local bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install flip tasks for VS Code",
		Long: `Install flip tasks to make them available in VS Code.

By default, tasks are installed globally (available in all VS Code windows).
Use --local to install only in the current directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if local {
				return installTasksLocal()
			}
			return installTasksGlobal()
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Install to .vscode/tasks.json in current directory")
	return cmd
}

func newVSCodeStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show VS Code tasks installation status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showVSCodeStatus()
		},
	}
}

func newVSCodeUninstallCommand() *cobra.Command {
	var local bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove flip tasks from VS Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			if local {
				return uninstallTasksLocal()
			}
			return uninstallTasksGlobal()
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Remove from .vscode/tasks.json in current directory")
	return cmd
}

// getVSCodeUserTasksPath returns the path to VS Code's user tasks.json
func getVSCodeUserTasksPath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, "Library", "Application Support", "Code", "User")
	case "linux":
		// Check for flatpak VS Code first
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		flatpakPath := filepath.Join(home, ".var", "app", "com.visualstudio.code", "config", "Code", "User")
		if _, err := os.Stat(flatpakPath); err == nil {
			configDir = flatpakPath
		} else {
			// Standard Linux path
			configDir = filepath.Join(home, ".config", "Code", "User")
		}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(appData, "Code", "User")
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return filepath.Join(configDir, "tasks.json"), nil
}

func installTasksGlobal() error {
	tasksPath, err := getVSCodeUserTasksPath()
	if err != nil {
		return fmt.Errorf("could not determine VS Code config path: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(tasksPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	// Check if file exists and merge if needed
	if _, err := os.Stat(tasksPath); err == nil {
		return mergeTasksFile(tasksPath, true)
	}

	// Fresh install
	if err := os.WriteFile(tasksPath, []byte(VSCodeTasksJSON), 0644); err != nil {
		return fmt.Errorf("could not write tasks.json: %w", err)
	}

	fmt.Println("✅ Flip tasks installed globally")
	fmt.Printf("   Location: %s\n", tasksPath)
	fmt.Println("   Tasks available in all VS Code windows")
	fmt.Println()
	fmt.Println("💡 Use Ctrl+Shift+P → 'Tasks: Run Task' → 'Flip: ...'")

	return nil
}

func installTasksLocal() error {
	tasksPath := filepath.Join(".vscode", "tasks.json")

	// Ensure .vscode directory exists
	if err := os.MkdirAll(".vscode", 0755); err != nil {
		return fmt.Errorf("could not create .vscode directory: %w", err)
	}

	// Check if file exists and merge if needed
	if _, err := os.Stat(tasksPath); err == nil {
		return mergeTasksFile(tasksPath, false)
	}

	// Fresh install
	if err := os.WriteFile(tasksPath, []byte(VSCodeTasksJSON), 0644); err != nil {
		return fmt.Errorf("could not write tasks.json: %w", err)
	}

	cwd, _ := os.Getwd()
	fmt.Println("✅ Flip tasks installed locally")
	fmt.Printf("   Location: %s\n", filepath.Join(cwd, tasksPath))
	fmt.Println("   Tasks available when this folder is open in VS Code")

	return nil
}

// mergeTasksFile merges flip tasks into an existing tasks.json
func mergeTasksFile(tasksPath string, isGlobal bool) error {
	// Read existing file
	existingData, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("could not read existing tasks.json: %w", err)
	}

	// Parse existing tasks
	var existing map[string]interface{}
	if err := json.Unmarshal(existingData, &existing); err != nil {
		return fmt.Errorf("could not parse existing tasks.json: %w", err)
	}

	// Parse flip tasks
	var flipTasks map[string]interface{}
	if err := json.Unmarshal([]byte(VSCodeTasksJSON), &flipTasks); err != nil {
		return fmt.Errorf("could not parse flip tasks: %w", err)
	}

	// Get existing tasks array
	existingTasksRaw, ok := existing["tasks"].([]interface{})
	if !ok {
		existingTasksRaw = []interface{}{}
	}

	// Get flip tasks array
	flipTasksRaw, ok := flipTasks["tasks"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid flip tasks configuration")
	}

	// Remove existing flip tasks (those starting with "Flip:")
	var filteredTasks []interface{}
	for _, task := range existingTasksRaw {
		taskMap, ok := task.(map[string]interface{})
		if !ok {
			continue
		}
		label, ok := taskMap["label"].(string)
		if !ok || !strings.HasPrefix(label, "Flip:") {
			filteredTasks = append(filteredTasks, task)
		}
	}

	// Add flip tasks
	mergedTasks := append(filteredTasks, flipTasksRaw...)

	// Update the map
	existing["tasks"] = mergedTasks
	existing["_flipTasksVersion"] = TasksVersion

	// Write back
	output, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal merged tasks: %w", err)
	}

	if err := os.WriteFile(tasksPath, output, 0644); err != nil {
		return fmt.Errorf("could not write merged tasks.json: %w", err)
	}

	location := "globally"
	if !isGlobal {
		location = "locally"
	}
	fmt.Printf("✅ Flip tasks updated %s\n", location)
	fmt.Printf("   Location: %s\n", tasksPath)
	fmt.Printf("   Version: %s\n", TasksVersion)

	return nil
}

func uninstallTasksGlobal() error {
	tasksPath, err := getVSCodeUserTasksPath()
	if err != nil {
		return fmt.Errorf("could not determine VS Code config path: %w", err)
	}

	return removeFlipTasks(tasksPath, true)
}

func uninstallTasksLocal() error {
	return removeFlipTasks(filepath.Join(".vscode", "tasks.json"), false)
}

func removeFlipTasks(tasksPath string, isGlobal bool) error {
	// Check if file exists
	if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
		fmt.Println("ℹ️  No tasks.json found, nothing to uninstall")
		return nil
	}

	// Read existing file
	existingData, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("could not read tasks.json: %w", err)
	}

	// Parse existing tasks
	var existing map[string]interface{}
	if err := json.Unmarshal(existingData, &existing); err != nil {
		return fmt.Errorf("could not parse tasks.json: %w", err)
	}

	// Get existing tasks array
	existingTasksRaw, ok := existing["tasks"].([]interface{})
	if !ok {
		fmt.Println("ℹ️  No tasks found in file")
		return nil
	}

	// Remove flip tasks
	var filteredTasks []interface{}
	removedCount := 0
	for _, task := range existingTasksRaw {
		taskMap, ok := task.(map[string]interface{})
		if !ok {
			continue
		}
		label, ok := taskMap["label"].(string)
		if ok && strings.HasPrefix(label, "Flip:") {
			removedCount++
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}

	if removedCount == 0 {
		fmt.Println("ℹ️  No Flip tasks found to remove")
		return nil
	}

	// Remove flip version marker
	delete(existing, "_flipTasksVersion")

	// Update the map
	existing["tasks"] = filteredTasks

	// Write back
	output, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal tasks: %w", err)
	}

	if err := os.WriteFile(tasksPath, output, 0644); err != nil {
		return fmt.Errorf("could not write tasks.json: %w", err)
	}

	location := "globally"
	if !isGlobal {
		location = "locally"
	}
	fmt.Printf("✅ Removed %d Flip tasks %s\n", removedCount, location)

	return nil
}

func showVSCodeStatus() error {
	fmt.Println("VS Code Tasks Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Check global installation
	globalPath, err := getVSCodeUserTasksPath()
	if err != nil {
		fmt.Printf("⚠️  Could not determine global path: %v\n", err)
	} else {
		globalVersion := getInstalledVersion(globalPath)
		if globalVersion != "" {
			if globalVersion == TasksVersion {
				fmt.Printf("✅ Global: %s (v%s - current)\n", globalPath, globalVersion)
			} else {
				fmt.Printf("⚠️  Global: %s (v%s - update available: v%s)\n", globalPath, globalVersion, TasksVersion)
			}
		} else {
			fmt.Printf("❌ Global: not installed\n")
		}
	}

	// Check local installation
	localPath := filepath.Join(".vscode", "tasks.json")
	localVersion := getInstalledVersion(localPath)
	if localVersion != "" {
		cwd, _ := os.Getwd()
		if localVersion == TasksVersion {
			fmt.Printf("✅ Local:  %s (v%s - current)\n", filepath.Join(cwd, localPath), localVersion)
		} else {
			fmt.Printf("⚠️  Local:  %s (v%s - update available: v%s)\n", filepath.Join(cwd, localPath), localVersion, TasksVersion)
		}
	} else {
		fmt.Printf("❌ Local:  not installed\n")
	}

	fmt.Println()
	fmt.Printf("📦 Bundled version: v%s\n", TasksVersion)

	// Check if running in VS Code
	if IsRunningInVSCode() {
		fmt.Println("🖥️  Running in VS Code terminal")
	}

	return nil
}

// getInstalledVersion reads the _flipTasksVersion from a tasks.json file
func getInstalledVersion(tasksPath string) string {
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return ""
	}

	var tasks map[string]interface{}
	if err := json.Unmarshal(data, &tasks); err != nil {
		return ""
	}

	// Check for flip version marker
	if version, ok := tasks["_flipTasksVersion"].(string); ok {
		return version
	}

	// Check if there are any Flip tasks (old installation without version)
	if tasksRaw, ok := tasks["tasks"].([]interface{}); ok {
		for _, task := range tasksRaw {
			if taskMap, ok := task.(map[string]interface{}); ok {
				if label, ok := taskMap["label"].(string); ok && strings.HasPrefix(label, "Flip:") {
					return "unknown"
				}
			}
		}
	}

	return ""
}

// IsRunningInVSCode returns true if flip is running in a VS Code terminal
func IsRunningInVSCode() bool {
	// VS Code sets TERM_PROGRAM=vscode
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return true
	}
	// Also check for VSCODE_* environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "VSCODE_") {
			return true
		}
	}
	return false
}

// CheckVSCodeTasksUpdate checks if tasks need updating and shows a hint
// Call this from main.go during startup
// NOTE: Checks if --json flag is present in os.Args to skip in JSON mode
func CheckVSCodeTasksUpdate() {
	// Skip hints if --json flag is present
	for _, arg := range os.Args {
		if arg == "--json" || arg == "-json" {
			return
		}
	}

	if !IsRunningInVSCode() {
		return
	}

	// Check global installation
	globalPath, err := getVSCodeUserTasksPath()
	if err != nil {
		return
	}

	globalVersion := getInstalledVersion(globalPath)

	// No installation - suggest installing
	if globalVersion == "" {
		fmt.Println("💡 Tip: Run 'flip vscode install' to enable VS Code tasks")
		fmt.Println()
		return
	}

	// Outdated installation - suggest updating
	if globalVersion != TasksVersion && globalVersion != "unknown" {
		fmt.Printf("💡 Tip: Flip tasks outdated (v%s → v%s). Run 'flip vscode install' to update\n", globalVersion, TasksVersion)
		fmt.Println()
	}
}
