package commands

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BrainHealth represents the health status of a brain
type BrainHealth struct {
	Path        string
	HasIssues   bool
	Issues      []string
	Warnings    []string
	SyncService string
	IsGitRepo   bool
	GitStatus   GitHealthStatus
	SyncStatus  SyncHealthStatus
}

// GitHealthStatus holds git-specific health information
type GitHealthStatus struct {
	HasUncommitted   bool
	UncommittedCount int
	IsAhead          bool
	AheadCount       int
	IsBehind         bool
	BehindCount      int
	HasRemote        bool
	RemoteURL        string
	Branch           string
}

// SyncHealthStatus holds sync-specific health information
type SyncHealthStatus struct {
	HasConflicts        bool
	ConflictFiles       []string
	HasSyncErrors       bool
	SyncErrorIndicators []string
}

// checkBrainHealth performs comprehensive health check on a brain
func checkBrainHealth(brainPath string) BrainHealth {
	health := BrainHealth{
		Path:     brainPath,
		Issues:   []string{},
		Warnings: []string{},
	}

	// Check if path exists
	if _, err := os.Stat(brainPath); os.IsNotExist(err) {
		health.HasIssues = true
		health.Issues = append(health.Issues, "Path does not exist")
		return health
	}

	// Detect sync service
	health.SyncService = detectSyncService(brainPath)

	// Check git status
	gitInfo := detectGitInfo(brainPath)
	health.IsGitRepo = gitInfo.IsRepo

	if gitInfo.IsRepo {
		health.GitStatus = checkGitHealth(brainPath, gitInfo)

		// Add issues based on git status
		if health.GitStatus.HasUncommitted {
			health.Warnings = append(health.Warnings,
				fmt.Sprintf("%d uncommitted changes", health.GitStatus.UncommittedCount))
		}
		if health.GitStatus.IsAhead {
			health.Warnings = append(health.Warnings,
				fmt.Sprintf("%d commits ahead of remote", health.GitStatus.AheadCount))
		}
		if health.GitStatus.IsBehind {
			health.HasIssues = true
			health.Issues = append(health.Issues,
				fmt.Sprintf("%d commits behind remote - needs pull", health.GitStatus.BehindCount))
		}
		if !health.GitStatus.HasRemote {
			health.Warnings = append(health.Warnings, "No git remote configured")
		}
	}

	// Check sync status
	if health.SyncService != "" {
		health.SyncStatus = checkSyncHealth(brainPath, health.SyncService)

		if health.SyncStatus.HasConflicts {
			health.HasIssues = true
			health.Issues = append(health.Issues,
				fmt.Sprintf("%d conflict files found", len(health.SyncStatus.ConflictFiles)))
		}
		if health.SyncStatus.HasSyncErrors {
			health.HasIssues = true
			health.Issues = append(health.Issues,
				fmt.Sprintf("Sync errors detected: %s",
					strings.Join(health.SyncStatus.SyncErrorIndicators, ", ")))
		}
	}

	return health
}

// checkGitHealth checks git repository status
func checkGitHealth(brainPath string, gitInfo GitInfo) GitHealthStatus {
	status := GitHealthStatus{
		Branch:    gitInfo.Branch,
		RemoteURL: gitInfo.RemoteURL,
		HasRemote: gitInfo.RemoteURL != "",
	}

	// Change to brain directory for git commands
	oldDir, _ := os.Getwd()
	os.Chdir(brainPath)
	defer os.Chdir(oldDir)

	// Check for uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			status.HasUncommitted = true
			status.UncommittedCount = len(lines)
		}
	}

	// Check ahead/behind status (only if remote exists)
	if status.HasRemote {
		// Fetch to get latest remote info (silently)
		exec.Command("git", "fetch", "--quiet").Run()

		// Get ahead/behind counts
		cmd = exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
		if output, err := cmd.Output(); err == nil {
			parts := strings.Fields(strings.TrimSpace(string(output)))
			if len(parts) == 2 {
				var ahead, behind int
				fmt.Sscanf(parts[0], "%d", &ahead)
				fmt.Sscanf(parts[1], "%d", &behind)

				if ahead > 0 {
					status.IsAhead = true
					status.AheadCount = ahead
				}
				if behind > 0 {
					status.IsBehind = true
					status.BehindCount = behind
				}
			}
		}
	}

	return status
}

// checkSyncHealth checks for sync-related issues
func checkSyncHealth(brainPath string, syncService string) SyncHealthStatus {
	status := SyncHealthStatus{
		ConflictFiles:       []string{},
		SyncErrorIndicators: []string{},
	}

	// Patterns for conflict files from various sync services
	conflictPatterns := []string{
		// Dropbox
		" (conflicted copy ",
		" (Conflicted Copy ",
		// OneDrive / SharePoint
		"-conflict-",
		// iCloud
		" 2.", // iCloud duplicates files with " 2" suffix
		// General
		".conflict",
	}

	// Walk directory to find conflict files
	filepath.WalkDir(brainPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Skip hidden and system directories
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Check for conflict patterns in filename
		filename := d.Name()
		for _, pattern := range conflictPatterns {
			if strings.Contains(filename, pattern) {
				status.HasConflicts = true
				relPath, _ := filepath.Rel(brainPath, path)
				status.ConflictFiles = append(status.ConflictFiles, relPath)
				break
			}
		}

		return nil
	})

	// Check for sync service specific error indicators
	syncErrorFiles := map[string][]string{
		"Dropbox":      {".dropbox.cache/error.txt"},
		"Google Drive": {"desktop.ini"},                           // Often indicates sync issues
		"OneDrive":     {".849C9593-D756-4E56-8D6E-42412F2A707B"}, // OneDrive error marker
	}

	if errorFiles, ok := syncErrorFiles[syncService]; ok {
		for _, errorFile := range errorFiles {
			if _, err := os.Stat(filepath.Join(brainPath, errorFile)); err == nil {
				status.HasSyncErrors = true
				status.SyncErrorIndicators = append(status.SyncErrorIndicators, errorFile)
			}
		}
	}

	return status
}

// formatHealthStatus returns a colored status indicator
func formatHealthStatus(health BrainHealth) string {
	if health.HasIssues {
		return "⚠️"
	}
	if len(health.Warnings) > 0 {
		return "⚡"
	}
	return "✓"
}

// formatHealthDetails returns a detailed health summary
func formatHealthDetails(health BrainHealth) string {
	if !health.HasIssues && len(health.Warnings) == 0 {
		return "Healthy"
	}

	details := []string{}

	if len(health.Issues) > 0 {
		details = append(details, strings.Join(health.Issues, "; "))
	}

	if len(health.Warnings) > 0 {
		details = append(details, strings.Join(health.Warnings, "; "))
	}

	return strings.Join(details, " | ")
}
