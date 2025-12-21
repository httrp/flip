package health

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RepairAction represents an action that can fix an issue
type RepairAction struct {
	IssueType   IssueType
	Description string
	Apply       func(brainPath string, issue Issue) error
}

// RepairResult contains the result of a repair operation
type RepairResult struct {
	Issue     Issue
	Success   bool
	Message   string
	SkipReason string
}

// Repairer can fix certain health issues
type Repairer struct {
	brainPath string
	brainType BrainType
	dryRun    bool
	actions   map[IssueType]RepairAction
}

// NewRepairer creates a new repairer for a brain
func NewRepairer(brainPath string, dryRun bool) (*Repairer, error) {
	info, err := AnalyzeBrain(brainPath)
	if err != nil {
		return nil, err
	}

	r := &Repairer{
		brainPath: brainPath,
		brainType: info.Type,
		dryRun:    dryRun,
		actions:   make(map[IssueType]RepairAction),
	}

	// Register repair actions
	r.registerActions()

	return r, nil
}

// registerActions sets up all available repair actions
func (r *Repairer) registerActions() {
	// Broken internal links - remove or comment out
	r.actions[IssueTypeBrokenLink] = RepairAction{
		IssueType:   IssueTypeBrokenLink,
		Description: "Comment out or remove broken internal links",
		Apply:       r.repairBrokenLink,
	}

	// Orphaned files - move to orphaned folder or delete
	r.actions[IssueTypeOrphanedFile] = RepairAction{
		IssueType:   IssueTypeOrphanedFile,
		Description: "Move orphaned files to .orphaned folder",
		Apply:       r.repairOrphanedFile,
	}

	// Format issues - normalize formatting
	r.actions[IssueTypeFormat] = RepairAction{
		IssueType:   IssueTypeFormat,
		Description: "Normalize file formatting",
		Apply:       r.repairFormatIssue,
	}
}

// CanRepair checks if an issue type can be repaired
func (r *Repairer) CanRepair(issueType IssueType) bool {
	_, ok := r.actions[issueType]
	return ok
}

// GetRepairableIssues returns a description of what can be repaired
func (r *Repairer) GetRepairableIssues() []RepairAction {
	result := make([]RepairAction, 0, len(r.actions))
	for _, action := range r.actions {
		result = append(result, action)
	}
	return result
}

// RepairIssues attempts to fix a list of issues
func (r *Repairer) RepairIssues(issues []Issue) []RepairResult {
	results := make([]RepairResult, 0, len(issues))

	for _, issue := range issues {
		result := r.repairSingleIssue(issue)
		results = append(results, result)
	}

	return results
}

// repairSingleIssue attempts to fix a single issue
func (r *Repairer) repairSingleIssue(issue Issue) RepairResult {
	action, ok := r.actions[issue.Type]
	if !ok {
		return RepairResult{
			Issue:      issue,
			Success:    false,
			SkipReason: "No repair action available for this issue type",
		}
	}

	if r.dryRun {
		return RepairResult{
			Issue:   issue,
			Success: true,
			Message: fmt.Sprintf("[DRY-RUN] Would: %s", action.Description),
		}
	}

	if err := action.Apply(r.brainPath, issue); err != nil {
		return RepairResult{
			Issue:   issue,
			Success: false,
			Message: fmt.Sprintf("Failed: %v", err),
		}
	}

	return RepairResult{
		Issue:   issue,
		Success: true,
		Message: "Repaired successfully",
	}
}

// ============================================================================
// REPAIR IMPLEMENTATIONS
// ============================================================================

// repairBrokenLink comments out or removes a broken link
func (r *Repairer) repairBrokenLink(brainPath string, issue Issue) error {
	if issue.File == "" || issue.Line == 0 {
		return fmt.Errorf("issue missing file or line information")
	}

	fullPath := filepath.Join(brainPath, issue.File)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	if issue.Line > len(lines) {
		return fmt.Errorf("line %d out of range (file has %d lines)", issue.Line, len(lines))
	}

	// Get the problematic line (1-indexed)
	lineIdx := issue.Line - 1
	line := lines[lineIdx]

	// Strategy: Comment out the broken link by wrapping it
	// For wikilinks: [[broken]] → ~~[[broken]]~~ (BROKEN LINK)
	// For markdown links: [text](broken.md) → ~~[text](broken.md)~~ (BROKEN LINK)

	// Try to find the specific link pattern
	brokenTarget := extractLinkTarget(issue.Message)
	if brokenTarget == "" {
		// Fallback: comment out the entire line
		lines[lineIdx] = fmt.Sprintf("<!-- BROKEN LINK: %s -->", line)
	} else {
		// More surgical: just mark the broken link
		lines[lineIdx] = markBrokenLink(line, brokenTarget)
	}

	// Write back
	newContent := strings.Join(lines, "\n")
	return os.WriteFile(fullPath, []byte(newContent), 0644)
}

// repairOrphanedFile moves an orphaned file to .orphaned folder
func (r *Repairer) repairOrphanedFile(brainPath string, issue Issue) error {
	if issue.File == "" {
		return fmt.Errorf("issue missing file information")
	}

	srcPath := filepath.Join(brainPath, issue.File)
	
	// Create .orphaned directory if needed
	orphanedDir := filepath.Join(brainPath, ".orphaned")
	if err := os.MkdirAll(orphanedDir, 0755); err != nil {
		return fmt.Errorf("failed to create .orphaned directory: %w", err)
	}

	// Preserve directory structure in orphaned folder
	relDir := filepath.Dir(issue.File)
	if relDir != "." {
		targetDir := filepath.Join(orphanedDir, relDir)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}
	}

	dstPath := filepath.Join(orphanedDir, issue.File)

	// Move the file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move file: %w", err)
	}

	return nil
}

// repairFormatIssue normalizes file formatting
func (r *Repairer) repairFormatIssue(brainPath string, issue Issue) error {
	if issue.File == "" {
		return fmt.Errorf("issue missing file information")
	}

	fullPath := filepath.Join(brainPath, issue.File)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Normalize the content
	normalized := normalizeContent(string(content))

	return os.WriteFile(fullPath, []byte(normalized), 0644)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// extractLinkTarget extracts the link target from an issue message
func extractLinkTarget(message string) string {
	// Try to extract from "Link to 'xyz' not found" pattern
	patterns := []string{
		`Link to '([^']+)' not found`,
		`\[\[([^\]]+)\]\]`,
		`\[([^\]]+)\]\([^)]+\)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(message); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// markBrokenLink marks a broken link without removing it
func markBrokenLink(line, target string) string {
	// Wikilink pattern
	wikiPattern := regexp.MustCompile(`\[\[` + regexp.QuoteMeta(target) + `(\|[^\]]+)?\]\]`)
	if wikiPattern.MatchString(line) {
		return wikiPattern.ReplaceAllString(line, `~~[[$0]]~~ <!-- BROKEN -->`)
	}

	// Markdown link pattern
	mdPattern := regexp.MustCompile(`\[([^\]]+)\]\(` + regexp.QuoteMeta(target) + `[^)]*\)`)
	if mdPattern.MatchString(line) {
		return mdPattern.ReplaceAllString(line, `~~$0~~ <!-- BROKEN -->`)
	}

	// Fallback: just append comment
	return line + " <!-- CONTAINS BROKEN LINK -->"
}

// normalizeContent applies standard formatting to markdown content
func normalizeContent(content string) string {
	// Ensure file ends with exactly one newline
	content = strings.TrimRight(content, "\n") + "\n"

	// Normalize line endings to LF
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// Remove trailing whitespace from lines
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	// Collapse multiple blank lines to maximum of 2
	result := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		if line == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, line)
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// RepairStats contains statistics about a repair operation
type RepairStats struct {
	TotalIssues   int
	Repaired      int
	Failed        int
	Skipped       int
	NotRepairable int
}

// CalculateStats calculates statistics from repair results
func CalculateStats(results []RepairResult) RepairStats {
	stats := RepairStats{
		TotalIssues: len(results),
	}

	for _, r := range results {
		if r.Success {
			if strings.HasPrefix(r.Message, "[DRY-RUN]") {
				// Don't count dry-run as actual repair
				stats.Repaired++
			} else {
				stats.Repaired++
			}
		} else if r.SkipReason != "" {
			if strings.Contains(r.SkipReason, "No repair action") {
				stats.NotRepairable++
			} else {
				stats.Skipped++
			}
		} else {
			stats.Failed++
		}
	}

	return stats
}
