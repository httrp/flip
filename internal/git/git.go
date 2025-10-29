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

	// Skip ahead/behind info by default to avoid authentication prompts
	// Users can explicitly check sync status via Pull command
	// if aheadBehind, err := getAheadBehind(repoPath); err == nil {
	// 	status.AheadBehind = aheadBehind
	// }

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
	// First check if an upstream branch is configured
	// This prevents triggering authentication prompts for repos without remotes
	checkCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	checkCmd.Dir = repoPath
	upstreamOutput, err := checkCmd.Output()
	if err != nil {
		// No upstream configured - this is fine for local-only repos
		return "", err
	}

	upstream := strings.TrimSpace(string(upstreamOutput))
	if upstream == "" {
		return "", fmt.Errorf("no upstream branch")
	}

	// Check if the upstream ref exists locally (from last fetch)
	// This avoids triggering authentication by only comparing local refs
	checkRefCmd := exec.Command("git", "rev-parse", "--verify", "--quiet", upstream)
	checkRefCmd.Dir = repoPath
	if err := checkRefCmd.Run(); err != nil {
		// Upstream ref doesn't exist locally - need to fetch first
		// Return empty instead of forcing network access
		return "", fmt.Errorf("upstream ref not found locally")
	}

	// Only check ahead/behind if upstream ref exists locally
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD..."+upstream)
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

// PullOptions configures how a pull is performed
type PullOptions struct {
	Rebase       bool // Use rebase instead of merge
	AutoStash    bool // Automatically stash/unstash local changes
	AllowDirty   bool // Allow pull even with uncommitted changes
}

// Pull pulls changes from the remote repository
func Pull(repoPath string) error {
	return PullWithOptions(repoPath, PullOptions{
		Rebase:     true, // Default to rebase for cleaner history
		AutoStash:  false,
		AllowDirty: false,
	})
}

// PullWithOptions pulls changes with specific options
func PullWithOptions(repoPath string, opts PullOptions) error {
	if !IsGitRepo(repoPath) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Check for uncommitted changes
	if HasUncommittedChanges(repoPath) && !opts.AllowDirty && !opts.AutoStash {
		return fmt.Errorf("you have uncommitted changes. Please commit or stash them first")
	}

	// Build pull command
	args := []string{"pull"}
	if opts.Rebase {
		args = append(args, "--rebase")
	}
	if opts.AutoStash {
		args = append(args, "--autostash")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Check if it's a merge conflict
		if HasMergeConflicts(repoPath) {
			return fmt.Errorf("merge conflicts detected - please resolve them manually")
		}
		return fmt.Errorf("failed to pull: %w", err)
	}

	return nil
}

// HasMergeConflicts checks if there are unresolved merge conflicts
func HasMergeConflicts(repoPath string) bool {
	// Check for conflict markers in git status
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Look for "UU" (both modified) or "AA" (both added) status
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) >= 2 {
			status := line[:2]
			if status == "UU" || status == "AA" || status == "DD" {
				return true
			}
		}
	}

	// Also check if we're in the middle of a rebase/merge
	_, rebaseErr := os.Stat(filepath.Join(repoPath, ".git", "rebase-merge"))
	_, mergeErr := os.Stat(filepath.Join(repoPath, ".git", "MERGE_HEAD"))
	
	return rebaseErr == nil || mergeErr == nil
}

// GetConflictedFiles returns a list of files with merge conflicts
func GetConflictedFiles(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// AbortMerge aborts an ongoing merge or rebase
func AbortMerge(repoPath string) error {
	// Try rebase abort first
	cmd := exec.Command("git", "rebase", "--abort")
	cmd.Dir = repoPath
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Try merge abort
	cmd = exec.Command("git", "merge", "--abort")
	cmd.Dir = repoPath
	return cmd.Run()
}

// ResolveConflictUseOurs resolves conflict by keeping local version
func ResolveConflictUseOurs(repoPath, filePath string) error {
	cmd := exec.Command("git", "checkout", "--ours", filePath)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to resolve conflict: %w", err)
	}

	// Stage the resolved file
	return AddFile(repoPath, filePath)
}

// ResolveConflictUseTheirs resolves conflict by keeping remote version
func ResolveConflictUseTheirs(repoPath, filePath string) error {
	cmd := exec.Command("git", "checkout", "--theirs", filePath)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to resolve conflict: %w", err)
	}

	// Stage the resolved file
	return AddFile(repoPath, filePath)
}

// ResolveConflictMergeBoth merges both versions for content files (Markdown)
// This is safer for content as it prevents data loss
func ResolveConflictMergeBoth(repoPath, filePath string) error {
	fullPath := filepath.Join(repoPath, filePath)
	
	// Read the conflicted file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	
	// Check if file actually has conflict markers
	contentStr := string(content)
	if !strings.Contains(contentStr, "<<<<<<<") {
		// No conflict markers, file is already resolved
		return AddFile(repoPath, filePath)
	}
	
	// Extract both versions from conflict markers
	// Format: <<<<<<< HEAD (local) ... ======= ... >>>>>>> remote
	var merged strings.Builder
	lines := strings.Split(contentStr, "\n")
	
	inConflict := false
	var localContent []string
	var remoteContent []string
	collectingLocal := false
	
	for _, line := range lines {
		if strings.HasPrefix(line, "<<<<<<<") {
			inConflict = true
			collectingLocal = true
			continue
		} else if strings.HasPrefix(line, "=======") && inConflict {
			collectingLocal = false
			continue
		} else if strings.HasPrefix(line, ">>>>>>>") && inConflict {
			// End of conflict - merge both versions
			merged.WriteString("<!-- ═══════════════════════════════════════════════════════ -->\n")
			merged.WriteString("<!-- MERGED CONTENT: Both local and remote versions included -->\n")
			merged.WriteString("<!-- Please review and clean up as needed -->\n")
			merged.WriteString("<!-- ═══════════════════════════════════════════════════════ -->\n\n")
			
			if len(localContent) > 0 {
				merged.WriteString("<!-- LOCAL VERSION: -->\n")
				merged.WriteString(strings.Join(localContent, "\n"))
				merged.WriteString("\n\n")
			}
			
			if len(remoteContent) > 0 {
				merged.WriteString("<!-- REMOTE VERSION: -->\n")
				merged.WriteString(strings.Join(remoteContent, "\n"))
				merged.WriteString("\n")
			}
			
			merged.WriteString("\n<!-- END MERGED CONTENT -->\n\n")
			
			// Reset for next conflict
			inConflict = false
			localContent = nil
			remoteContent = nil
			continue
		}
		
		if inConflict {
			if collectingLocal {
				localContent = append(localContent, line)
			} else {
				remoteContent = append(remoteContent, line)
			}
		} else {
			merged.WriteString(line)
			merged.WriteString("\n")
		}
	}
	
	// Write merged content back
	if err := os.WriteFile(fullPath, []byte(merged.String()), 0644); err != nil {
		return fmt.Errorf("failed to write merged file: %w", err)
	}
	
	// Stage the resolved file
	return AddFile(repoPath, filePath)
}

// ContinueRebase continues a rebase after conflicts are resolved
func ContinueRebase(repoPath string) error {
	cmd := exec.Command("git", "rebase", "--continue")
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Fetch fetches changes from remote without merging
func Fetch(repoPath string) error {
	if !IsGitRepo(repoPath) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	cmd := exec.Command("git", "fetch")
	cmd.Dir = repoPath
	// Run silently
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	return nil
}

// HasRemoteUpdates checks if there are updates available from remote
// Returns true if local branch is behind remote
func HasRemoteUpdates(repoPath string) bool {
	// First try to fetch (silently)
	_ = Fetch(repoPath)

	// Check if behind remote
	aheadBehind, err := getAheadBehind(repoPath)
	if err != nil {
		return false
	}

	return strings.Contains(aheadBehind, "behind")
}
