package health

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Issue represents a health check issue
type Issue struct {
	Type     IssueType
	Severity Severity
	File     string
	Line     int
	Message  string
	Details  string
}

// IssueType categorizes the type of issue
type IssueType string

const (
	IssueTypeBrokenLink   IssueType = "broken-link"
	IssueTypeMissingAsset IssueType = "missing-asset"
	IssueTypeOrphanedFile IssueType = "orphaned-file"
	IssueTypeDuplicate    IssueType = "duplicate"
	IssueTypeFormat       IssueType = "format"
)

// Severity indicates how critical an issue is
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// CheckResult contains the results of a health check
type CheckResult struct {
	BrainInfo *BrainInfo
	Issues    []Issue
	Stats     CheckStats
}

// CheckStats contains statistics about the check
type CheckStats struct {
	FilesScanned  int
	LinksChecked  int
	AssetsChecked int
	ErrorCount    int
	WarningCount  int
	InfoCount     int
}

// Checker performs health checks on a brain
type Checker struct {
	brainPath string
	brainType BrainType
	brainInfo *BrainInfo

	// Tracking sets
	allFiles      map[string]bool // All files in brain
	markdownFiles []string        // All markdown files
	assetFiles    map[string]bool // All asset files
	linkedFiles   map[string]bool // Files that are linked to
}

// NewChecker creates a new health checker
func NewChecker(brainPath string) (*Checker, error) {
	info, err := AnalyzeBrain(brainPath)
	if err != nil {
		return nil, err
	}

	return &Checker{
		brainPath:     brainPath,
		brainType:     info.Type,
		brainInfo:     info,
		allFiles:      make(map[string]bool),
		assetFiles:    make(map[string]bool),
		linkedFiles:   make(map[string]bool),
		markdownFiles: make([]string, 0),
	}, nil
}

// Check performs all health checks
func (c *Checker) Check() (*CheckResult, error) {
	result := &CheckResult{
		BrainInfo: c.brainInfo,
		Issues:    make([]Issue, 0),
	}

	// Step 1: Index all files
	if err := c.indexFiles(); err != nil {
		return nil, fmt.Errorf("failed to index files: %w", err)
	}

	result.Stats.FilesScanned = len(c.allFiles)

	// Step 2: Check all markdown files for broken links
	for _, mdFile := range c.markdownFiles {
		issues, linkCount, err := c.checkFileLinks(mdFile)
		if err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to check %s: %v\n", mdFile, err)
			continue
		}
		result.Issues = append(result.Issues, issues...)
		result.Stats.LinksChecked += linkCount
	}

	// Step 3: Check for orphaned files
	orphanedIssues := c.checkOrphanedFiles()
	result.Issues = append(result.Issues, orphanedIssues...)

	// Calculate stats
	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityError:
			result.Stats.ErrorCount++
		case SeverityWarning:
			result.Stats.WarningCount++
		case SeverityInfo:
			result.Stats.InfoCount++
		}
	}

	result.Stats.AssetsChecked = len(c.assetFiles)

	return result, nil
}

// indexFiles builds an index of all files in the brain
func (c *Checker) indexFiles() error {
	return filepath.Walk(c.brainPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			// Skip hidden and config directories
			dirName := info.Name()
			if strings.HasPrefix(dirName, ".") ||
				dirName == "logseq" ||
				dirName == "node_modules" ||
				dirName == ".obsidian" ||
				dirName == ".trash" {
				return filepath.SkipDir
			}
			return nil
		}

		// Make path relative to brain root
		relPath, err := filepath.Rel(c.brainPath, path)
		if err != nil {
			return nil
		}

		// Track all files
		c.allFiles[relPath] = true

		ext := strings.ToLower(filepath.Ext(info.Name()))

		// Track markdown files
		if ext == ".md" {
			c.markdownFiles = append(c.markdownFiles, relPath)
		}

		// Track asset files
		if isAsset(ext) {
			c.assetFiles[relPath] = true
		}

		return nil
	})
}

// checkFileLinks checks a single markdown file for broken links
func (c *Checker) checkFileLinks(relPath string) ([]Issue, int, error) {
	issues := make([]Issue, 0)
	linkCount := 0

	// Skip template files from link checking
	if strings.Contains(relPath, "template") {
		return issues, 0, nil
	}

	fullPath := filepath.Join(c.brainPath, relPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, 0, err
	}

	lines := strings.Split(string(content), "\n")
	patterns := GetLinkPatterns(c.brainType)

	for lineNum, line := range lines {
		for _, pattern := range patterns {
			matches := pattern.Regex.FindAllString(line, -1)

			for _, match := range matches {
				linkCount++
				target := pattern.Extract(match)

				if target == "" {
					continue
				}

				// Skip template placeholders
				if strings.Contains(target, "{{") || strings.Contains(target, "}}") {
					continue
				}

				// Check if link target exists
				issue := c.checkLinkTarget(relPath, target, lineNum+1, pattern.Name)
				if issue != nil {
					issues = append(issues, *issue)
				} else {
					// Mark as linked
					c.linkedFiles[target] = true
				}
			}
		}
	}

	return issues, linkCount, nil
}

// checkLinkTarget verifies if a link target exists
func (c *Checker) checkLinkTarget(sourceFile, target string, lineNum int, linkType string) *Issue {
	// Handle different link types
	var targetPath string

	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		// External link - skip for now
		return nil
	}

	if strings.HasPrefix(target, "/") {
		// Absolute path from brain root
		targetPath = strings.TrimPrefix(target, "/")
	} else {
		// Relative path from source file directory
		sourceDir := filepath.Dir(sourceFile)
		targetPath = filepath.Join(sourceDir, target)
	}

	// Normalize path
	targetPath = filepath.Clean(targetPath)

	// For wikilinks, try multiple locations and extensions
	if linkType == "wikilink" {
		found := c.findWikilink(target)
		if found != "" {
			c.linkedFiles[found] = true
			return nil
		}

		return &Issue{
			Type:     IssueTypeBrokenLink,
			Severity: SeverityError,
			File:     sourceFile,
			Line:     lineNum,
			Message:  fmt.Sprintf("Broken wikilink: [[%s]]", target),
			Details:  "Target note not found in brain",
		}
	}

	// Check if target exists
	if c.allFiles[targetPath] {
		c.linkedFiles[targetPath] = true
		return nil
	}

	// Try with .md extension if not present
	if !strings.HasSuffix(targetPath, ".md") {
		mdPath := targetPath + ".md"
		if c.allFiles[mdPath] {
			c.linkedFiles[mdPath] = true
			return nil
		}
	}

	// Link is broken
	issueType := IssueTypeBrokenLink
	if isAsset(filepath.Ext(target)) {
		issueType = IssueTypeMissingAsset
	}

	return &Issue{
		Type:     issueType,
		Severity: SeverityError,
		File:     sourceFile,
		Line:     lineNum,
		Message:  fmt.Sprintf("Broken link: %s", target),
		Details:  fmt.Sprintf("Target not found: %s", targetPath),
	}
}

// findWikilink searches for a wikilink target in common locations
func (c *Checker) findWikilink(target string) string {
	// Common locations based on brain type
	var searchPaths []string

	baseName := strings.TrimSuffix(target, ".md")

	switch c.brainType {
	case BrainTypeLogseq:
		searchPaths = []string{
			filepath.Join("pages", baseName+".md"),
			filepath.Join("journals", baseName+".md"),
			baseName + ".md",
		}
	case BrainTypeObsidian, BrainTypeFlip:
		// Obsidian and Flip wikilinks can reference any file anywhere
		// We need to search all markdown files (case-insensitive)
		targetLower := strings.ToLower(baseName)
		
		// First try exact basename match across all folders
		for mdFile := range c.allFiles {
			if !strings.HasSuffix(mdFile, ".md") {
				continue
			}
			fileBase := strings.TrimSuffix(filepath.Base(mdFile), ".md")
			if strings.ToLower(fileBase) == targetLower {
				return mdFile
			}
		}
		
		// Try slug-style match (spaces to hyphens)
		targetSlug := strings.ToLower(strings.ReplaceAll(baseName, " ", "-"))
		for mdFile := range c.allFiles {
			if !strings.HasSuffix(mdFile, ".md") {
				continue
			}
			fileBase := strings.TrimSuffix(filepath.Base(mdFile), ".md")
			fileSlug := strings.ToLower(strings.ReplaceAll(fileBase, " ", "-"))
			if fileSlug == targetSlug {
				return mdFile
			}
		}
		
		return ""
	}

	// Check each possible path (for Logseq)
	for _, path := range searchPaths {
		if c.allFiles[path] {
			return path
		}
	}

	return ""
}

// checkOrphanedFiles finds files that are not linked from anywhere
func (c *Checker) checkOrphanedFiles() []Issue {
	issues := make([]Issue, 0)

	// Check markdown files
	for _, mdFile := range c.markdownFiles {
		// Skip certain files that are expected to be entry points
		if c.isEntryPoint(mdFile) {
			continue
		}

		if !c.linkedFiles[mdFile] {
			issues = append(issues, Issue{
				Type:     IssueTypeOrphanedFile,
				Severity: SeverityWarning,
				File:     mdFile,
				Message:  "Orphaned note (not linked from anywhere)",
				Details:  "Consider linking this note or moving it to archive",
			})
		}
	}

	// Check asset files
	for assetFile := range c.assetFiles {
		if !c.linkedFiles[assetFile] {
			issues = append(issues, Issue{
				Type:     IssueTypeOrphanedFile,
				Severity: SeverityInfo,
				File:     assetFile,
				Message:  "Orphaned asset (not referenced from any note)",
				Details:  "Consider removing unused assets to save space",
			})
		}
	}

	return issues
}

// isEntryPoint checks if a file is expected to be an entry point (not linked)
func (c *Checker) isEntryPoint(relPath string) bool {
	baseName := strings.ToLower(filepath.Base(relPath))

	// Common entry point files
	entryPoints := []string{
		"readme.md",
		"index.md",
		"home.md",
		"welcome.md",
		"inbox.md",
		"tasks.md",
		"today.md",
		"this-week.md",
		"someday.md",
	}

	for _, ep := range entryPoints {
		if baseName == ep {
			return true
		}
	}

	// Templates are not meant to be linked
	if strings.Contains(relPath, "template") {
		return true
	}

	// Journal entries are entry points
	if strings.Contains(relPath, "journal") {
		return true
	}

	// Task files are entry points
	if strings.Contains(relPath, "tasks/") {
		return true
	}

	return false
}
