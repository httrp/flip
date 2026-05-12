// vscode_search.go - VS Code search, recent, and sync commands
//
// This file contains commands for searching notes, listing recent files,
// and syncing brains via git. All commands output JSON for the VS Code extension.

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/httrp/flip/internal/git"
	"github.com/spf13/cobra"
)

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

// VSCodeSyncResult represents the result of a sync operation
type VSCodeSyncResult struct {
	Brains []VSCodeBrainSyncResult `json:"brains"`
}

// VSCodeBrainSyncResult represents sync result for a single brain
type VSCodeBrainSyncResult struct {
	Name           string   `json:"name"`
	Path           string   `json:"path"`
	Pulled         bool     `json:"pulled"`           // Remote changes were pulled
	HasChanges     bool     `json:"has_changes"`      // Local uncommitted changes
	ChangedFiles   []string `json:"changed_files,omitempty"`
	Committed      bool     `json:"committed"`
	CommitMessage  string   `json:"commit_message,omitempty"`
	Pushed         bool     `json:"pushed"`
	HasConflicts   bool     `json:"has_conflicts"`    // Merge conflicts detected
	ConflictFiles  []string `json:"conflict_files,omitempty"` // Files with conflicts
	Error          string   `json:"error,omitempty"`
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

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Bidirectional sync: pull, commit, and push changes (JSON)",
		Long:  "Full synchronization workflow: pulls remote changes, commits local changes, and pushes to remote. Detects and reports merge conflicts.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVSCodeSync(brainName)
		},
	}
	cmd.Flags().StringVar(&brainName, "brain", "", "Sync specific brain only")
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

func runVSCodeSync(brainName string) error {
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
			brainResult.Error = "not a git repository"
			result.Brains = append(result.Brains, brainResult)
			continue
		}

		if git.HasUncommittedChanges(brain.Path) {
			brainResult.HasChanges = true

			changes, err := git.GetChangedFiles(brain.Path)
			if err != nil {
				brainResult.Error = fmt.Sprintf("failed to get changed files: %v", err)
				result.Brains = append(result.Brains, brainResult)
				continue
			}
			brainResult.ChangedFiles = changes

			commitMsg := generateSmartCommitMessage(changes)
			brainResult.CommitMessage = commitMsg

			err = git.AddAndCommit(brain.Path, commitMsg)
			if err != nil {
				brainResult.Error = fmt.Sprintf("failed to commit: %v", err)
				result.Brains = append(result.Brains, brainResult)
				continue
			}
			brainResult.Committed = true
		}

		err = git.Pull(brain.Path)
		if err != nil {
			if git.HasMergeConflicts(brain.Path) {
				conflictFiles, _ := git.GetConflictedFiles(brain.Path)
				brainResult.HasConflicts = true
				brainResult.ConflictFiles = conflictFiles
				brainResult.Error = fmt.Sprintf("merge conflicts detected while pulling %q", brain.Name)
				result.Brains = append(result.Brains, brainResult)
				continue
			}
			brainResult.Error = fmt.Sprintf("failed to pull: %v", err)
			result.Brains = append(result.Brains, brainResult)
			continue
		}
		brainResult.Pulled = true

		err = gitPush(brain.Path)
		if err != nil {
			brainResult.Error = fmt.Sprintf("failed to push: %v", err)
			result.Brains = append(result.Brains, brainResult)
			continue
		}
		brainResult.Pushed = true

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
