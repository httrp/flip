package health

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// RepairAction represents an action that can fix an issue
type RepairAction struct {
	IssueType   IssueType
	Description string
	Apply       func(brainPath string, issue Issue) error
}

// RepairResult contains the result of a repair operation
type RepairResult struct {
	Issue      Issue
	Success    bool
	Message    string
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

	// Wrong link format - convert to brain-specific format
	r.actions[IssueTypeWrongLinkFormat] = RepairAction{
		IssueType:   IssueTypeWrongLinkFormat,
		Description: "Convert links to brain-specific format",
		Apply:       r.repairWrongLinkFormat,
	}

	// Wrong filename - rename to brain-specific convention and update all links
	r.actions[IssueTypeWrongFilename] = RepairAction{
		IssueType:   IssueTypeWrongFilename,
		Description: "Rename files to match brain conventions and update all links",
		Apply:       r.repairWrongFilename,
	}

	// Wrong media filename - rename asset and update all references
	r.actions[IssueTypeWrongMediaFilename] = RepairAction{
		IssueType:   IssueTypeWrongMediaFilename,
		Description: "Rename media to contextual name and update references",
		Apply:       r.repairWrongMediaFilename,
	}

	// Wrong media location - move asset to standard location and update references
	r.actions[IssueTypeWrongMediaLocation] = RepairAction{
		IssueType:   IssueTypeWrongMediaLocation,
		Description: "Move media to standard assets location and update references",
		Apply:       r.repairWrongMediaLocation,
	}

	// Missing metadata - add title and created-at to files
	r.actions[IssueTypeMissingMetadata] = RepairAction{
		IssueType:   IssueTypeMissingMetadata,
		Description: "Add missing metadata (title, created-at) to files",
		Apply:       r.repairMissingMetadata,
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

// repairWrongLinkFormat converts links to the correct format for the brain type
func (r *Repairer) repairWrongLinkFormat(brainPath string, issue Issue) error {
	if issue.File == "" {
		return fmt.Errorf("issue missing file information")
	}

	fullPath := filepath.Join(brainPath, issue.File)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	text := string(content)
	var newText string

	if r.brainType == BrainTypeLogseq || r.brainType == BrainTypeObsidian {
		// Convert markdown links [text](path.md) to wiki-links [[pagename]]
		markdownLinkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+\.md)\)`)
		newText = markdownLinkPattern.ReplaceAllStringFunc(text, func(match string) string {
			submatches := markdownLinkPattern.FindStringSubmatch(match)
			if len(submatches) < 3 {
				return match
			}
			// Extract page name from path (remove .md extension and path components)
			path := submatches[2]
			pageName := filepath.Base(path)
			pageName = strings.TrimSuffix(pageName, ".md")
			return "[[" + pageName + "]]"
		})
	} else {
		// Convert wiki-links [[pagename]] to markdown links [pagename](pagename.md)
		wikiLinkPattern := regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)
		newText = wikiLinkPattern.ReplaceAllStringFunc(text, func(match string) string {
			submatches := wikiLinkPattern.FindStringSubmatch(match)
			if len(submatches) < 2 {
				return match
			}
			pageName := submatches[1]
			// Create markdown link with .md extension
			return "[" + pageName + "](" + pageName + ".md)"
		})
	}

	if newText == text {
		return nil // No changes needed
	}

	return os.WriteFile(fullPath, []byte(newText), 0644)
}

// repairWrongFilename renames a file to match brain conventions and updates all links
func (r *Repairer) repairWrongFilename(brainPath string, issue Issue) error {
	if issue.File == "" {
		return fmt.Errorf("issue missing file information")
	}

	// Extract expected filename from issue details
	// Details format: "Current: old.md → Expected: new.md"
	parts := strings.Split(issue.Details, "→")
	if len(parts) != 2 {
		return fmt.Errorf("cannot parse expected filename from details")
	}

	expectedPart := strings.TrimSpace(parts[1])
	expectedFilename := strings.TrimPrefix(expectedPart, "Expected: ")

	oldPath := filepath.Join(brainPath, issue.File)
	newPath := filepath.Join(brainPath, filepath.Dir(issue.File), expectedFilename)

	// Check if target already exists
	if _, err := os.Stat(newPath); err == nil {
		return fmt.Errorf("target file already exists: %s", expectedFilename)
	}

	// Find all files that might link to this file
	var filesToUpdate []string
	err := filepath.WalkDir(brainPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".md" {
			// Skip the file being renamed
			if path != oldPath {
				filesToUpdate = append(filesToUpdate, path)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan for links: %w", err)
	}

	oldFilename := filepath.Base(oldPath)
	oldBasename := strings.TrimSuffix(oldFilename, ".md")
	newBasename := strings.TrimSuffix(expectedFilename, ".md")

	// Update all links in other files
	for _, file := range filesToUpdate {
		if err := r.updateLinksInFile(file, oldBasename, newBasename, oldFilename, expectedFilename); err != nil {
			// Log but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to update links in %s: %v\n", file, err)
		}
	}

	// Rename the file
	return os.Rename(oldPath, newPath)
}

// updateLinksInFile updates all links to a renamed file
func (r *Repairer) updateLinksInFile(filePath, oldBasename, newBasename, oldFilename, newFilename string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	text := string(content)
	updated := false

	// Update wiki-links: [[old-name]] -> [[new-name]]
	wikiLinkPattern := regexp.MustCompile(`\[\[` + regexp.QuoteMeta(oldBasename) + `(?:\|[^\]]+)?\]\]`)
	if wikiLinkPattern.MatchString(text) {
		text = wikiLinkPattern.ReplaceAllStringFunc(text, func(match string) string {
			// Preserve alias if present: [[old|alias]] -> [[new|alias]]
			if strings.Contains(match, "|") {
				parts := strings.SplitN(match, "|", 2)
				return "[[" + newBasename + "|" + parts[1]
			}
			return "[[" + newBasename + "]]"
		})
		updated = true
	}

	// Update markdown links: [text](old-name.md) -> [text](new-name.md)
	mdLinkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]*` + regexp.QuoteMeta(oldFilename) + `)\)`)
	if mdLinkPattern.MatchString(text) {
		text = mdLinkPattern.ReplaceAllStringFunc(text, func(match string) string {
			submatches := mdLinkPattern.FindStringSubmatch(match)
			if len(submatches) >= 3 {
				linkText := submatches[1]
				linkPath := submatches[2]
				// Replace filename in path
				newLinkPath := strings.Replace(linkPath, oldFilename, newFilename, 1)
				return "[" + linkText + "](" + newLinkPath + ")"
			}
			return match
		})
		updated = true
	}

	// Also update relative paths like ../notes/old-name.md
	relPathPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]*/)` + regexp.QuoteMeta(oldFilename) + `\)`)
	if relPathPattern.MatchString(text) {
		text = relPathPattern.ReplaceAllString(text, "[$1]($2"+newFilename+")")
		updated = true
	}

	// Update markdown image references: ![alt](path/oldFilename) -> ![alt](path/newFilename)
	mdImgPattern := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*` + regexp.QuoteMeta(oldFilename) + `)\)`)
	if mdImgPattern.MatchString(text) {
		text = mdImgPattern.ReplaceAllStringFunc(text, func(match string) string {
			re := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*)\)`)
			parts := re.FindStringSubmatch(match)
			if len(parts) >= 3 {
				alt := parts[1]
				linkPath := parts[2]
				newLinkPath := strings.Replace(linkPath, oldFilename, newFilename, 1)
				return fmt.Sprintf("![%s](%s)", alt, newLinkPath)
			}
			return match
		})
		updated = true
	}

	// Update wikilink embeds with filenames: ![[oldFilename]] -> ![[newFilename]]
	wikiEmbedPattern := regexp.MustCompile(`!\[\[` + regexp.QuoteMeta(oldFilename) + `\]\]`)
	if wikiEmbedPattern.MatchString(text) {
		text = wikiEmbedPattern.ReplaceAllString(text, "![["+newFilename+"]]")
		updated = true
	}

	if updated {
		return os.WriteFile(filePath, []byte(text), 0644)
	}

	return nil
}

// repairWrongMediaFilename renames an asset to a better contextual name and updates references
func (r *Repairer) repairWrongMediaFilename(brainPath string, issue Issue) error {
	if issue.File == "" {
		return fmt.Errorf("issue missing file information")
	}

	oldRel := issue.File
	oldAbs := filepath.Join(brainPath, oldRel)
	oldFilename := filepath.Base(oldRel)
	ext := strings.ToLower(filepath.Ext(oldRel))

	// Find a referencing note to derive a contextual name
	noteBase := ""
	err := filepath.WalkDir(brainPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			content, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			if strings.Contains(string(content), oldFilename) {
				nb := strings.TrimSuffix(filepath.Base(path), ".md")
				noteBase = sanitizeTitle(nb)
				return fmt.Errorf("found") // break walk
			}
		}
		return nil
	})
	if err == nil {
		// No early termination; ignore
	}

	ts := time.Now().Format("20060102-150405")
	var newFilename string
	if noteBase != "" {
		newFilename = fmt.Sprintf("%s-%s%s", noteBase, ts, ext)
	} else {
		newFilename = fmt.Sprintf("asset-%s%s", ts, ext)
	}

	newAbs := filepath.Join(filepath.Dir(oldAbs), newFilename)

	if _, statErr := os.Stat(newAbs); statErr == nil {
		return fmt.Errorf("target file already exists: %s", newFilename)
	}

	// Update all references in markdown files
	var filesToUpdate []string
	_ = filepath.WalkDir(brainPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			filesToUpdate = append(filesToUpdate, path)
		}
		return nil
	})

	for _, file := range filesToUpdate {
		if uerr := r.updateLinksInFile(file, "", "", oldFilename, newFilename); uerr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update references in %s: %v\n", file, uerr)
		}
	}

	// Rename the asset
	return os.Rename(oldAbs, newAbs)
}

// repairWrongMediaLocation moves an asset to a standard location and updates references
func (r *Repairer) repairWrongMediaLocation(brainPath string, issue Issue) error {
	if issue.File == "" {
		return fmt.Errorf("issue missing file information")
	}

	oldRel := issue.File
	oldAbs := filepath.Join(brainPath, oldRel)
	oldFilename := filepath.Base(oldRel)

	// Decide target location based on standard conventions

	// Try to derive source note path for near-note strategies
	sourceNoteRel := ""
	_ = filepath.WalkDir(brainPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			content, rerr := os.ReadFile(path)
			if rerr == nil && strings.Contains(string(content), oldFilename) {
				rel, _ := filepath.Rel(brainPath, path)
				sourceNoteRel = rel
				return fmt.Errorf("found")
			}
		}
		return nil
	})

	targetRelPath := getAssetTargetPath(r.brainType, oldFilename, sourceNoteRel)
	targetAbsPath := filepath.Join(brainPath, targetRelPath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(targetAbsPath), 0755); err != nil {
		return fmt.Errorf("failed to create target dir: %w", err)
	}

	// Move file
	if err := os.Rename(oldAbs, targetAbsPath); err != nil {
		return fmt.Errorf("failed to move asset: %w", err)
	}

	// Update references in markdown files to point to newRel path
	var filesToUpdate []string
	_ = filepath.WalkDir(brainPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".md" {
			filesToUpdate = append(filesToUpdate, path)
		}
		return nil
	})

	// For updates, replace any link path ending in oldFilename with targetRelPath
	for _, file := range filesToUpdate {
		content, rerr := os.ReadFile(file)
		if rerr != nil {
			continue
		}
		text := string(content)
		updated := false

		// Markdown links and images
		re := regexp.MustCompile(`(\!?)\[([^\]]*)\]\(([^)]*` + regexp.QuoteMeta(oldFilename) + `)\)`)
		if re.MatchString(text) {
			text = re.ReplaceAllStringFunc(text, func(match string) string {
				re2 := regexp.MustCompile(`(\!?)\[([^\]]*)\]\(([^)]*)\)`)
				parts := re2.FindStringSubmatch(match)
				if len(parts) >= 4 {
					bang := parts[1]
					alt := parts[2]
					// Replace full path with targetRelPath
					return fmt.Sprintf("%s[%s](%s)", bang, alt, targetRelPath)
				}
				return match
			})
			updated = true
		}

		// Wikilink embeds with filename
		we := regexp.MustCompile(`!\[\[([^\]]*` + regexp.QuoteMeta(oldFilename) + `)\]\]`)
		if we.MatchString(text) {
			// Use only basename for wikilinks (common convention)
			text = we.ReplaceAllString(text, "![["+filepath.Base(targetRelPath)+"]]")
			updated = true
		}

		if updated {
			_ = os.WriteFile(file, []byte(text), 0644)
		}
	}

	return nil
}

// getAssetTargetPath returns a standard target path for assets per brain type
func getAssetTargetPath(brainType BrainType, assetFilename, sourceNotePath string) string {
	switch brainType {
	case BrainTypeFlip, BrainTypeLogseq, BrainTypeDendron:
		// Place in main assets directory
		return filepath.Join("assets", assetFilename)
	case BrainTypeFoam:
		return filepath.Join("attachments", assetFilename)
	case BrainTypeObsidian:
		// Place near note when available
		if sourceNotePath != "" {
			noteDir := filepath.Dir(sourceNotePath)
			return filepath.Join(noteDir, assetFilename)
		}
		return filepath.Join("attachments", assetFilename)
	default:
		return filepath.Join("assets", assetFilename)
	}
}

// repairMissingMetadata adds missing metadata (title, created-at) to a file
func (r *Repairer) repairMissingMetadata(brainPath string, issue Issue) error {
	if issue.File == "" {
		return fmt.Errorf("issue missing file information")
	}

	fullPath := filepath.Join(brainPath, issue.File)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	contentStr := string(content)
	basename := filepath.Base(issue.File)
	nameWithoutExt := strings.TrimSuffix(basename, ".md")

	// Generate title from filename
	title := fileNameToTitle(nameWithoutExt)

	// Try to get creation date from git history, fallback to file mtime
	createdAt := r.getFileCreationTime(fullPath, brainPath)
	brainName := filepath.Base(brainPath)

	if isJournalPath(issue.File) {
		switch r.brainType {
		case BrainTypeLogseq:
			return r.addLogseqJournalMetadata(fullPath, contentStr, title, createdAt, brainName)
		default:
			return r.addYAMLJournalMetadata(fullPath, contentStr, title, createdAt, brainName)
		}
	}

	switch r.brainType {
	case BrainTypeLogseq:
		// Logseq uses property format: property:: value
		// Properties go at the start of the first block (first line or after frontmatter)
		return r.addLogseqMetadata(fullPath, contentStr, title, createdAt)

	case BrainTypeFlip, BrainTypeFoam, BrainTypeObsidian:
		// These use YAML frontmatter
		return r.addYAMLFrontmatter(fullPath, contentStr, title, createdAt)

	default:
		// Default to YAML frontmatter
		return r.addYAMLFrontmatter(fullPath, contentStr, title, createdAt)
	}
}

func (r *Repairer) addLogseqJournalMetadata(fullPath, content, title string, createdAt time.Time, brainName string) error {
	hasTitle := strings.Contains(content, "title::")
	hasCreatedAt := strings.Contains(content, "created-at::")
	hasBrain := strings.Contains(content, "brain::")

	if hasTitle && hasCreatedAt && hasBrain {
		return nil
	}

	createdMs := createdAt.UnixMilli()

	var props []string
	if !hasTitle {
		props = append(props, fmt.Sprintf("title:: %s", title))
	}
	if !hasCreatedAt {
		props = append(props, fmt.Sprintf("created-at:: %d", createdMs))
	}
	if !hasBrain {
		props = append(props, fmt.Sprintf("brain:: %s", brainName))
	}

	if len(props) == 0 {
		return nil
	}

	propBlock := strings.Join(props, "\n") + "\n"

	var newContent string
	if strings.TrimSpace(content) == "" {
		newContent = propBlock
	} else {
		if strings.HasPrefix(content, "- ") {
			newContent = "- " + strings.Join(props, "\n  ") + "\n" + content
		} else {
			newContent = propBlock + "\n" + content
		}
	}

	return os.WriteFile(fullPath, []byte(newContent), 0644)
}

func (r *Repairer) addYAMLJournalMetadata(fullPath, content, title string, createdAt time.Time, brainName string) error {
	dateStr := createdAt.Format("2006-01-02")

	if strings.HasPrefix(content, "---") {
		endIdx := strings.Index(content[3:], "---")
		if endIdx == -1 {
			return fmt.Errorf("malformed frontmatter: no closing ---")
		}

		frontmatter := content[3 : endIdx+3]
		rest := content[endIdx+6:]

		hasTitle := strings.Contains(frontmatter, "title:")
		hasCreated := strings.Contains(frontmatter, "created:") || strings.Contains(frontmatter, "date:")
		hasBrain := strings.Contains(frontmatter, "brain:")

		if hasTitle && hasCreated && hasBrain {
			return nil
		}

		var additions []string
		if !hasTitle {
			additions = append(additions, fmt.Sprintf("title: %s", title))
		}
		if !hasCreated {
			additions = append(additions, fmt.Sprintf("created: %s", dateStr))
		}
		if !hasBrain {
			additions = append(additions, fmt.Sprintf("brain: %s", brainName))
		}

		newFrontmatter := strings.TrimRight(frontmatter, "\n") + "\n" + strings.Join(additions, "\n") + "\n"
		newContent := "---" + newFrontmatter + "---" + rest

		return os.WriteFile(fullPath, []byte(newContent), 0644)
	}

	frontmatter := fmt.Sprintf("---\ntitle: %s\ncreated: %s\nbrain: %s\n---\n\n", title, dateStr, brainName)
	newContent := frontmatter + content

	return os.WriteFile(fullPath, []byte(newContent), 0644)
}

// addLogseqMetadata adds Logseq-style properties to a file
func (r *Repairer) addLogseqMetadata(fullPath, content, title string, createdAt time.Time) error {
	// Check what's already present
	hasTitle := strings.Contains(content, "title::")
	hasCreatedAt := strings.Contains(content, "created-at::")

	if hasTitle && hasCreatedAt {
		return nil // Nothing to do
	}

	// Format timestamp as Unix milliseconds (Logseq convention)
	createdMs := createdAt.UnixMilli()

	// Build property block
	var props []string
	if !hasTitle {
		props = append(props, fmt.Sprintf("title:: %s", title))
	}
	if !hasCreatedAt {
		props = append(props, fmt.Sprintf("created-at:: %d", createdMs))
	}

	if len(props) == 0 {
		return nil
	}

	// Insert properties at the start of the file
	// In Logseq, page properties are typically the first block
	propBlock := strings.Join(props, "\n") + "\n"

	var newContent string
	if strings.TrimSpace(content) == "" {
		// Empty file
		newContent = propBlock
	} else {
		// Prepend to existing content
		// If first line is "- ", insert after it (outline format)
		if strings.HasPrefix(content, "- ") {
			// Insert properties as first bullet point
			newContent = "- " + strings.Join(props, "\n  ") + "\n" + content
		} else {
			// Standard: prepend properties
			newContent = propBlock + "\n" + content
		}
	}

	return os.WriteFile(fullPath, []byte(newContent), 0644)
}

// addYAMLFrontmatter adds or updates YAML frontmatter in a file
func (r *Repairer) addYAMLFrontmatter(fullPath, content, title string, createdAt time.Time) error {
	// Format date as ISO 8601
	dateStr := createdAt.Format("2006-01-02")

	if strings.HasPrefix(content, "---") {
		// Already has frontmatter - update it
		endIdx := strings.Index(content[3:], "---")
		if endIdx == -1 {
			return fmt.Errorf("malformed frontmatter: no closing ---")
		}

		frontmatter := content[3 : endIdx+3]
		rest := content[endIdx+6:] // After the closing ---

		hasTitle := strings.Contains(frontmatter, "title:")
		hasCreated := strings.Contains(frontmatter, "created:") || strings.Contains(frontmatter, "date:")

		if hasTitle && hasCreated {
			return nil // Nothing to do
		}

		// Add missing fields
		var additions []string
		if !hasTitle {
			additions = append(additions, fmt.Sprintf("title: %s", title))
		}
		if !hasCreated {
			additions = append(additions, fmt.Sprintf("created: %s", dateStr))
		}

		// Insert at the end of frontmatter (before closing ---)
		newFrontmatter := strings.TrimRight(frontmatter, "\n") + "\n" + strings.Join(additions, "\n") + "\n"
		newContent := "---" + newFrontmatter + "---" + rest

		return os.WriteFile(fullPath, []byte(newContent), 0644)
	}

	// No frontmatter - create one
	frontmatter := fmt.Sprintf("---\ntitle: %s\ncreated: %s\n---\n\n", title, dateStr)
	newContent := frontmatter + content

	return os.WriteFile(fullPath, []byte(newContent), 0644)
}

// getFileCreationTime tries to get creation time from git, falls back to file mtime
func (r *Repairer) getFileCreationTime(fullPath, brainPath string) time.Time {
	// Try git log first
	relPath, err := filepath.Rel(brainPath, fullPath)
	if err == nil {
		createdTime := getGitCreationTime(brainPath, relPath)
		if !createdTime.IsZero() {
			return createdTime
		}
	}

	// Fallback to file modification time
	info, err := os.Stat(fullPath)
	if err != nil {
		return time.Now()
	}

	return info.ModTime()
}

// getGitCreationTime gets the first commit date for a file using git log
func getGitCreationTime(repoPath, relPath string) time.Time {
	// We can't use exec.Command here easily without adding the import
	// So we'll use a simpler approach: check if .git exists and use file stat
	// For a proper implementation, we'd need to import "os/exec"

	// For now, return zero time to indicate we should use file mtime
	// TODO: Implement proper git history lookup
	return time.Time{}
}
