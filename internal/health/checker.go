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
	IssueTypeBrokenLink      IssueType = "broken-link"
	IssueTypeMissingAsset    IssueType = "missing-asset"
	IssueTypeOrphanedFile    IssueType = "orphaned-file"
	IssueTypeDuplicate       IssueType = "duplicate"
	IssueTypeFormat          IssueType = "format"
	IssueTypeWrongLinkFormat IssueType = "wrong-link-format"
	IssueTypeWrongFilename   IssueType = "wrong-filename"
    IssueTypeWrongMediaFilename IssueType = "wrong-media-filename"
    IssueTypeWrongMediaLocation IssueType = "wrong-media-location"
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
	// Track references from markdown files to asset files
	assetReferences map[string][]string // asset rel path -> list of source md files
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
		assetReferences: make(map[string][]string),
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

	// Step 2.5: Check filename conventions for note files
	for _, mdFile := range c.markdownFiles {
		if issue := c.checkFilenameConvention(mdFile); issue != nil {
			result.Issues = append(result.Issues, *issue)
		}
	}

	// Step 3: Check for orphaned files
	orphanedIssues := c.checkOrphanedFiles()
	result.Issues = append(result.Issues, orphanedIssues...)

	// Step 3.5: Check media naming and location conventions
	mediaIssues := c.checkMediaConventions()
	result.Issues = append(result.Issues, mediaIssues...)

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
	return filepath.WalkDir(c.brainPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if d.IsDir() {
			// Skip hidden and config directories
			dirName := d.Name()
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

		ext := strings.ToLower(filepath.Ext(d.Name()))

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
		// Check for wrong link formats
		formatIssues := c.checkLinkFormat(relPath, line, lineNum+1)
		issues = append(issues, formatIssues...)

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
		// Also check image references (markdown image and wikilink embeds)
		imgIssues, imgCount := c.checkImageRefs(relPath, line, lineNum+1)
		if len(imgIssues) > 0 {
			issues = append(issues, imgIssues...)
		}
		linkCount += imgCount
	}

	return issues, linkCount, nil
}

// checkImageRefs finds image references and validates targets
func (c *Checker) checkImageRefs(sourceRelPath, line string, lineNum int) ([]Issue, int) {
	issues := make([]Issue, 0)
	count := 0

	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "    ") {
		return issues, 0
	}

	// Markdown image: ![alt](path)
	mdImgRe := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	mdMatches := mdImgRe.FindAllStringSubmatch(line, -1)
	for _, m := range mdMatches {
		if len(m) >= 3 {
			count++
			target := m[2]
			// Skip external
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
				continue
			}
			// Resolve relative to source file
			sourceDir := filepath.Dir(sourceRelPath)
			targetPath := filepath.Clean(filepath.Join(sourceDir, target))

			if c.allFiles[targetPath] {
				c.linkedFiles[targetPath] = true
				c.assetReferences[targetPath] = append(c.assetReferences[targetPath], sourceRelPath)
			} else {
				issues = append(issues, Issue{
					Type:     IssueTypeMissingAsset,
					Severity: SeverityError,
					File:     sourceRelPath,
					Line:     lineNum,
					Message:  fmt.Sprintf("Missing asset: %s", target),
					Details:  fmt.Sprintf("Target not found: %s", targetPath),
				})
			}
		}
	}

	// Wikilink embed: ![[asset.ext]] or ![[path/asset.ext]]
	wikiEmbedRe := regexp.MustCompile(`!\[\[([^\]]+)\]\]`)
	weMatches := wikiEmbedRe.FindAllStringSubmatch(line, -1)
	for _, m := range weMatches {
		if len(m) >= 2 {
			count++
			target := m[1]
			// Resolve possibly relative path
			if strings.HasPrefix(target, "/") {
				target = strings.TrimPrefix(target, "/")
			}
			sourceDir := filepath.Dir(sourceRelPath)
			targetPath := filepath.Clean(filepath.Join(sourceDir, target))

			// Try base name search if not found directly
			if !c.allFiles[targetPath] {
				base := filepath.Base(target)
				// search all files for basename match
				found := ""
				for f := range c.allFiles {
					if filepath.Base(f) == base {
						found = f
						break
					}
				}
				if found != "" {
					targetPath = found
				}
			}

			if c.allFiles[targetPath] {
				c.linkedFiles[targetPath] = true
				c.assetReferences[targetPath] = append(c.assetReferences[targetPath], sourceRelPath)
			} else {
				issues = append(issues, Issue{
					Type:     IssueTypeMissingAsset,
					Severity: SeverityError,
					File:     sourceRelPath,
					Line:     lineNum,
					Message:  fmt.Sprintf("Missing asset: %s", target),
					Details:  fmt.Sprintf("Target not found"),
				})
			}
		}
	}

	return issues, count
}

// checkMediaConventions validates asset filenames and locations
func (c *Checker) checkMediaConventions() []Issue {
	issues := make([]Issue, 0)

	// Allowed locations based on brain structure
	bsLocs := getAllowedAssetLocations(c.brainType)

	// Generic filename patterns to flag
	genericPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^image(\d+)?\.(png|jpg|jpeg|gif|webp|svg)$`),
		regexp.MustCompile(`(?i)^screenshot.*\.(png|jpg|jpeg|gif|webp)$`),
		regexp.MustCompile(`(?i)^pasted[-_ ]?image.*\.(png|jpg|jpeg|gif|webp)$`),
		regexp.MustCompile(`(?i)^bildschirmfoto.*\.(png|jpg|jpeg|gif|webp)$`),
	}

	for asset := range c.assetFiles {
		base := filepath.Base(asset)
		dir := filepath.Dir(asset)

		// Check wrong location: not under any allowed AssetLocations
		allowed := false
		for _, loc := range bsLocs {
			// Normalize: check prefix match allowing nested dirs
			if strings.HasPrefix(dir+"/", strings.TrimSuffix(loc, "/")+"/") || dir == loc || base == loc {
				allowed = true
				break
			}
		}
		if !allowed {
			issues = append(issues, Issue{
				Type:     IssueTypeWrongMediaLocation,
				Severity: SeverityWarning,
				File:     asset,
				Line:     0,
				Message:  fmt.Sprintf("Asset outside standard locations (%s)", strings.Join(bsLocs, ", ")),
				Details:  fmt.Sprintf("Current: %s", asset),
			})
		}

		// Check generic filename
		isGeneric := false
		for _, re := range genericPatterns {
			if re.MatchString(base) {
				isGeneric = true
				break
			}
		}
		if isGeneric {
			// Suggest a better name using first referencing note, if available
			noteBase := ""
			if refs, ok := c.assetReferences[asset]; ok && len(refs) > 0 {
				nb := strings.TrimSuffix(filepath.Base(refs[0]), ".md")
				noteBase = sanitizeTitle(extractTitleFromFilename(nb, c.brainType))
			}
			ts := time.Now().Format("20060102-150405")
			ext := strings.ToLower(filepath.Ext(base))
			suggested := ""
			if noteBase != "" {
				suggested = fmt.Sprintf("%s-%s%s", noteBase, ts, ext)
			} else {
				suggested = fmt.Sprintf("asset-%s%s", ts, ext)
			}

			issues = append(issues, Issue{
				Type:     IssueTypeWrongMediaFilename,
				Severity: SeverityInfo,
				File:     asset,
				Line:     0,
				Message:  "Generic media filename",
				Details:  fmt.Sprintf("Current: %s → Suggested: %s", base, suggested),
			})
		}
	}

	return issues
}

// getAllowedAssetLocations returns acceptable asset folders for a brain type
func getAllowedAssetLocations(brainType BrainType) []string {
	switch brainType {
	case BrainTypeFlip:
		return []string{"assets", "assets/images", "assets/documents"}
	case BrainTypeLogseq:
		return []string{"assets", "../assets"}
	case BrainTypeObsidian:
		return []string{"attachments", "assets", "files", ".attachments"}
	case BrainTypeDendron:
		return []string{"assets", "vault/assets"}
	case BrainTypeFoam:
		return []string{"attachments", "assets", "docs/attachments"}
	default:
		return []string{"assets", "attachments", "files"}
	}
}

// checkLinkFormat checks if a line uses the wrong link format for this brain type
func (c *Checker) checkLinkFormat(relPath, line string, lineNum int) []Issue {
	issues := make([]Issue, 0)

	// Skip lines that are clearly not links or are in code blocks
	if strings.HasPrefix(strings.TrimSpace(line), "```") || strings.HasPrefix(strings.TrimSpace(line), "    ") {
		return issues
	}

	// Define patterns for detecting wrong formats
	markdownLinkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+\.md)\)`)
	wikiLinkRegex := regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

	switch c.brainType {
	case BrainTypeLogseq, BrainTypeObsidian:
		// These should use [[wiki-links]], not [markdown](links) for internal files
		matches := markdownLinkRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 3 {
				linkPath := match[2]
				// Skip external links and assets
				if strings.HasPrefix(linkPath, "http") || !strings.HasSuffix(linkPath, ".md") {
					continue
				}
				issues = append(issues, Issue{
					Type:     IssueTypeWrongLinkFormat,
					Severity: SeverityWarning,
					File:     relPath,
					Line:     lineNum,
					Message:  fmt.Sprintf("Markdown link should be wiki-link in %s brain", c.brainType),
					Details:  fmt.Sprintf("Found: %s → Should be: [[%s]]", match[0], extractPageName(linkPath)),
				})
			}
		}

	case BrainTypeFlip:
		// Flip should use [markdown](links), not [[wiki-links]]
		matches := wikiLinkRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				pageName := match[1]
				// Skip if it looks like an external reference
				if strings.Contains(pageName, "http") {
					continue
				}
				issues = append(issues, Issue{
					Type:     IssueTypeWrongLinkFormat,
					Severity: SeverityWarning,
					File:     relPath,
					Line:     lineNum,
					Message:  "Wiki-link should be markdown link in Flip brain",
					Details:  fmt.Sprintf("Found: [[%s]] → Should be: [%s](%s.md)", pageName, pageName, pageName),
				})
			}
		}
	}

	return issues
}

// extractPageName extracts the page name from a path (filename without extension)
func extractPageName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".md")
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

// checkFilenameConvention checks if a file follows brain-specific naming conventions
func (c *Checker) checkFilenameConvention(relPath string) *Issue {
	// Only check specific patterns that are clearly wrong
	
	// Logseq: Pages should NOT have YYYY_MM_DD___ prefix (only journals should)
	if c.brainType == BrainTypeLogseq {
		basename := filepath.Base(relPath)
		
		// Check if file is in pages/ directory with date prefix
		if strings.HasPrefix(relPath, "pages/") && 
		   len(basename) > 14 && 
		   basename[4] == '_' && 
		   basename[7] == '_' && 
		   basename[10:14] == "___" {
			// This is a pages/ file with YYYY_MM_DD___ prefix - wrong!
			correctName := basename[14:] // Remove date prefix
			
			return &Issue{
				Type:     IssueTypeWrongFilename,
				Severity: SeverityWarning,
				File:     relPath,
				Message:  fmt.Sprintf("Logseq pages should not have date prefix. Expected: %s", correctName),
				Details:  fmt.Sprintf("Rename to: %s (date belongs in frontmatter, not filename)", correctName),
			}
		}
	}
	
	return nil
}

// extractTitleFromFilename extracts the title part from a filename based on brain type
func extractTitleFromFilename(basename string, brainType BrainType) string {
	switch brainType {
	case BrainTypeLogseq:
		// Strip YYYY_MM_DD___ prefix if present
		if len(basename) > 14 && basename[4] == '_' && basename[7] == '_' && basename[10] == '_' {
			if len(basename) > 13 && basename[11:14] == "___" {
				return basename[14:]
			}
		}
		return basename

	case BrainTypeFlip, BrainTypeFoam:
		// Strip YYYY-MM-DD- prefix if present
		if len(basename) > 11 && basename[4] == '-' && basename[7] == '-' && basename[10] == '-' {
			return basename[11:]
		}
		return basename

	case BrainTypeDendron:
		// Strip notes. prefix and YYYY-MM-DD- if present
		if strings.HasPrefix(basename, "notes.") {
			withoutPrefix := basename[6:]
			if len(withoutPrefix) > 11 && withoutPrefix[4] == '-' && withoutPrefix[7] == '-' && withoutPrefix[10] == '-' {
				return withoutPrefix[11:]
			}
			return withoutPrefix
		}
		return basename

	case BrainTypeObsidian:
		// No date prefix expected
		return basename

	default:
		return basename
	}
}

// generateExpectedFilename generates the expected filename based on brain conventions
// relPath is used to extract existing date if the file already has one
func generateExpectedFilename(title string, brainType BrainType, relPath string) string {
	// Sanitize title to kebab-case
	safeName := sanitizeTitle(title)

	// Extract existing date from filename if present
	filename := filepath.Base(relPath)
	basename := strings.TrimSuffix(filename, ".md")
	dateStr := extractDateFromFilename(basename, brainType)

	// Use existing date or current date
	if dateStr == "" {
		now := time.Now()
		dateStr = now.Format("2006-01-02")
	}

	switch brainType {
	case BrainTypeLogseq:
		// YYYY_MM_DD___title.md
		logseqDate := strings.ReplaceAll(dateStr, "-", "_")
		return fmt.Sprintf("%s___%s.md", logseqDate, safeName)

	case BrainTypeObsidian:
		// title.md (no date)
		return fmt.Sprintf("%s.md", safeName)

	case BrainTypeDendron:
		// notes.YYYY-MM-DD-title.md
		return fmt.Sprintf("notes.%s-%s.md", dateStr, safeName)

	case BrainTypeFoam, BrainTypeFlip:
		// YYYY-MM-DD-title.md
		return fmt.Sprintf("%s-%s.md", dateStr, safeName)

	default:
		// Default: YYYY-MM-DD-title.md
		return fmt.Sprintf("%s-%s.md", dateStr, safeName)
	}
}

// extractDateFromFilename tries to extract a date from the filename
func extractDateFromFilename(basename string, brainType BrainType) string {
	switch brainType {
	case BrainTypeLogseq:
		// YYYY_MM_DD___title pattern
		if len(basename) > 10 && basename[4] == '_' && basename[7] == '_' {
			dateStr := basename[0:10]
			return strings.ReplaceAll(dateStr, "_", "-")
		}

	case BrainTypeFlip, BrainTypeFoam:
		// YYYY-MM-DD-title pattern
		if len(basename) > 10 && basename[4] == '-' && basename[7] == '-' {
			return basename[0:10]
		}

	case BrainTypeDendron:
		// notes.YYYY-MM-DD-title pattern
		if strings.HasPrefix(basename, "notes.") {
			withoutPrefix := basename[6:]
			if len(withoutPrefix) > 10 && withoutPrefix[4] == '-' && withoutPrefix[7] == '-' {
				return withoutPrefix[0:10]
			}
		}
	}

	return ""
}

// sanitizeTitle converts a title to kebab-case following the same rules as note creation
func sanitizeTitle(title string) string {
	safeName := strings.ToLower(title)

	// Replace spaces and special chars with hyphen
	safeName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == ' ' || r == '_' || r == '.' || r == ',' || r == ';' || r == ':' {
			return '-'
		}
		// Drop other special characters
		return -1
	}, safeName)

	// Collapse multiple hyphens into one
	for strings.Contains(safeName, "--") {
		safeName = strings.ReplaceAll(safeName, "--", "-")
	}

	// Trim hyphens from start/end
	safeName = strings.Trim(safeName, "-")

	return safeName
}
