package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RepoStatus represents the status of a git repository
type RepoStatus struct {
	IsRepo          bool
	HasChanges      bool
	UntrackedFiles  int
	ModifiedFiles   int
	StagedFiles     int
	Branch          string
	LastCommitHash  string
	LastCommitMsg   string
	LastCommitDate  time.Time
	RemoteURL       string
	AheadBehind     string // e.g., "ahead 2, behind 1"
}

// CommitOptions configures how a commit is created
type CommitOptions struct {
	Message     string
	AddAll      bool   // Run git add . before commit
	AuthorName  string // Optional: override author name
	AuthorEmail string // Optional: override author email
}

// IsGitRepo checks if the given path is inside a git repository
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// GetRepoRoot returns the root directory of the git repository
func GetRepoRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(output)), nil
}

// GetStatus retrieves comprehensive git status for a repository
func GetStatus(repoPath string) (*RepoStatus, error) {
	status := &RepoStatus{
		IsRepo: IsGitRepo(repoPath),
	}

	if !status.IsRepo {
		return status, nil
	}

	// Get current branch
	branch, err := getCurrentBranch(repoPath)
	if err == nil {
		status.Branch = branch
	}

	// Get status information
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		status.HasChanges = len(lines) > 0 && lines[0] != ""

		for _, line := range lines {
			if len(line) < 2 {
				continue
			}
			statusCode := line[:2]
			if statusCode[0] == '?' {
				status.UntrackedFiles++
			} else if statusCode[0] != ' ' {
				status.StagedFiles++
			} else if statusCode[1] != ' ' {
				status.ModifiedFiles++
			}
		}
	}

	// Get last commit info
	if lastCommit, err := getLastCommit(repoPath); err == nil {
		status.LastCommitHash = lastCommit.Hash
		status.LastCommitMsg = lastCommit.Message
		status.LastCommitDate = lastCommit.Date
	}

	// Get remote URL
	if remote, err := getRemoteURL(repoPath); err == nil {
		status.RemoteURL = remote
	}

	// Get ahead/behind info
	if aheadBehind, err := getAheadBehind(repoPath); err == nil {
		status.AheadBehind = aheadBehind
	}

	return status, nil
}

// CommitInfo represents information about a git commit
type CommitInfo struct {
	Hash    string
	Message string
	Author  string
	Date    time.Time
}

// GetLastCommit retrieves information about the last commit
func GetLastCommit(repoPath string) (*CommitInfo, error) {
	return getLastCommit(repoPath)
}

func getLastCommit(repoPath string) (*CommitInfo, error) {
	// Format: hash|message|author|date
	cmd := exec.Command("git", "log", "-1", "--pretty=format:%h|%s|%an|%ci")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get last commit: %w", err)
	}

	parts := strings.Split(string(output), "|")
	if len(parts) < 4 {
		return nil, fmt.Errorf("unexpected git log output")
	}

	date, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[3])

	return &CommitInfo{
		Hash:    parts[0],
		Message: parts[1],
		Author:  parts[2],
		Date:    date,
	}, nil
}

// GetCommitHistory retrieves the last N commits
func GetCommitHistory(repoPath string, count int) ([]*CommitInfo, error) {
	cmd := exec.Command("git", "log", fmt.Sprintf("-%d", count), "--pretty=format:%h|%s|%an|%ci")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commits := make([]*CommitInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		date, _ := time.Parse("2006-01-02 15:04:05 -0700", parts[3])

		commits = append(commits, &CommitInfo{
			Hash:    parts[0],
			Message: parts[1],
			Author:  parts[2],
			Date:    date,
		})
	}

	return commits, nil
}

// Commit creates a new git commit
func Commit(repoPath string, options CommitOptions) error {
	if !IsGitRepo(repoPath) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Add files if requested
	if options.AddAll {
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add files: %w", err)
		}
	}

	// Check if there's anything to commit
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = repoPath
	if err := cmd.Run(); err == nil {
		// No changes staged
		return fmt.Errorf("no changes to commit")
	}

	// Build commit command
	args := []string{"commit", "-m", options.Message}

	// Set author if provided
	if options.AuthorName != "" && options.AuthorEmail != "" {
		args = append(args, "--author", fmt.Sprintf("%s <%s>", options.AuthorName, options.AuthorEmail))
	}

	cmd = exec.Command("git", args...)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// AddAndCommit is a convenience function that adds all changes and commits
func AddAndCommit(repoPath, message string) error {
	return Commit(repoPath, CommitOptions{
		Message: message,
		AddAll:  true,
	})
}

// AddFile stages a specific file
func AddFile(repoPath, filePath string) error {
	if !IsGitRepo(repoPath) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Make path relative to repo root if it's absolute
	if filepath.IsAbs(filePath) {
		repoRoot, err := GetRepoRoot(repoPath)
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(repoRoot, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		filePath = relPath
	}

	cmd := exec.Command("git", "add", filePath)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add file: %w", err)
	}

	return nil
}

// Helper functions

func getCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getRemoteURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getAheadBehind(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	parts := strings.Fields(string(output))
	if len(parts) != 2 {
		return "", nil
	}

	ahead := parts[0]
	behind := parts[1]

	if ahead == "0" && behind == "0" {
		return "up to date", nil
	}

	var result []string
	if ahead != "0" {
		result = append(result, fmt.Sprintf("ahead %s", ahead))
	}
	if behind != "0" {
		result = append(result, fmt.Sprintf("behind %s", behind))
	}

	return strings.Join(result, ", "), nil
}

// GetChangedFiles returns a list of changed files
func GetChangedFiles(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	files := make([]string, 0, len(lines))

	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		// Extract filename (after status code and space)
		filename := strings.TrimSpace(line[3:])
		files = append(files, filename)
	}

	return files, nil
}

// HasUncommittedChanges checks if there are any uncommitted changes
func HasUncommittedChanges(repoPath string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}
