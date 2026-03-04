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
	IssueTypeBrokenLink         IssueType = "broken-link"
	IssueTypeMissingAsset       IssueType = "missing-asset"
	IssueTypeOrphanedFile       IssueType = "orphaned-file"
	IssueTypeDuplicate          IssueType = "duplicate"
	IssueTypeFormat             IssueType = "format"
	IssueTypeWrongLinkFormat    IssueType = "wrong-link-format"
	IssueTypeWrongFilename      IssueType = "wrong-filename"
	IssueTypeWrongMediaFilename IssueType = "wrong-media-filename"
	IssueTypeWrongMediaLocation IssueType = "wrong-media-location"
	IssueTypeMissingMetadata    IssueType = "missing-metadata"
	IssueTypeMalformedTitle     IssueType = "malformed-title"
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
		brainPath:       brainPath,
		brainType:       info.Type,
		brainInfo:       info,
		allFiles:        make(map[string]bool),
		assetFiles:      make(map[string]bool),
		linkedFiles:     make(map[string]bool),
		markdownFiles:   make([]string, 0),
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

	// Step 2.6: Check for missing metadata (title, created-at)
	for _, mdFile := range c.markdownFiles {
		metadataIssues := c.checkMissingMetadata(mdFile)
		result.Issues = append(result.Issues, metadataIssues...)
	}

	// Step 3: Check for orphaned files
	orphanedIssues := c.checkOrphanedFiles()
	result.Issues = append(result.Issues, orphanedIssues...)

	// Step 3.5: Check media naming and location conventions
	mediaIssues := c.checkMediaConventions()
	result.Issues = append(result.Issues, mediaIssues...)

	// Step 3.7: Check for notes with recent creation that might be missing journal links (Flip brains only)
	if c.brainInfo.Type == "flip" || c.brainInfo.Type == "flip-brain" {
		journalLinkIssues := c.checkPotentialMissingJournalLinks()
		result.Issues = append(result.Issues, journalLinkIssues...)
	}

	// Step 3.8: Check for malformed journal titles (space-separated dates)
	if c.brainInfo.Type == "flip" || c.brainInfo.Type == "flip-brain" {
		titleIssues := c.checkJournalTitles()
		result.Issues = append(result.Issues, titleIssues...)
	}

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
			target = strings.TrimPrefix(target, "/")
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
					Details:  "Target not found",
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

		// Broken wikilink - try to find suggestions
		suggestion := c.suggestWikilinkFix(target)
		details := "Target note not found in brain"
		if suggestion != "" {
			details += fmt.Sprintf(". Did you mean: [[%s]]?", suggestion)
		}

		return &Issue{
			Type:     IssueTypeBrokenLink,
			Severity: SeverityError,
			File:     sourceFile,
			Line:     lineNum,
			Message:  fmt.Sprintf("Broken wikilink: [[%s]]", target),
			Details:  details,
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

	// Link is broken - try to find suggestions
	issueType := IssueTypeBrokenLink
	if isAsset(filepath.Ext(target)) {
		issueType = IssueTypeMissingAsset
	}

	suggestion := c.suggestMarkdownLinkFix(target, sourceFile)
	details := fmt.Sprintf("Target not found: %s", targetPath)
	if suggestion != "" {
		details += fmt.Sprintf(". Did you mean: %s?", suggestion)
	}

	return &Issue{
		Type:     issueType,
		Severity: SeverityError,
		File:     sourceFile,
		Line:     lineNum,
		Message:  fmt.Sprintf("Broken link: %s", target),
		Details:  details,
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

// suggestWikilinkFix suggests a correct wikilink based on similarity
func (c *Checker) suggestWikilinkFix(target string) string {
	targetLower := strings.ToLower(target)
	var candidates []struct {
		path  string
		score int
	}

	// Collect all markdown files
	for mdFile := range c.allFiles {
		if !strings.HasSuffix(mdFile, ".md") {
			continue
		}

		fileBase := strings.TrimSuffix(filepath.Base(mdFile), ".md")
		fileLower := strings.ToLower(fileBase)

		// Calculate similarity score
		score := levenshteinSimilarity(targetLower, fileLower)

		// Also boost exact substring matches
		if strings.Contains(fileLower, targetLower) || strings.Contains(targetLower, fileLower) {
			score += 50
		}

		// Only consider reasonable matches (>50% similar)
		if score > 50 {
			candidates = append(candidates, struct {
				path  string
				score int
			}{fileBase, score})
		}
	}

	// Return the best match
	if len(candidates) > 0 {
		// Sort by score descending
		maxScore := 0
		bestCandidate := ""
		for _, c := range candidates {
			if c.score > maxScore {
				maxScore = c.score
				bestCandidate = c.path
			}
		}
		return bestCandidate
	}

	return ""
}

// suggestMarkdownLinkFix suggests a correct markdown link based on similarity
func (c *Checker) suggestMarkdownLinkFix(target, sourceFile string) string {
	// Extract just the filename from the link
	targetFile := filepath.Base(target)
	targetLower := strings.ToLower(strings.TrimSuffix(targetFile, ".md"))

	var candidates []struct {
		path  string
		score int
	}

	// Collect all files in brain
	for file := range c.allFiles {
		if !strings.HasSuffix(file, ".md") {
			continue
		}

		fileBase := strings.TrimSuffix(filepath.Base(file), ".md")
		fileLower := strings.ToLower(fileBase)

		// Calculate similarity score
		score := levenshteinSimilarity(targetLower, fileLower)

		// Boost exact substring matches
		if strings.Contains(fileLower, targetLower) || strings.Contains(targetLower, fileLower) {
			score += 50
		}

		if score > 50 {
			candidates = append(candidates, struct {
				path  string
				score int
			}{file, score})
		}
	}

	// Return the best match
	if len(candidates) > 0 {
		maxScore := 0
		bestCandidate := ""
		for _, c := range candidates {
			if c.score > maxScore {
				maxScore = c.score
				bestCandidate = c.path
			}
		}
		return bestCandidate
	}

	return ""
}

// levenshteinSimilarity calculates string similarity as a percentage (0-100)
// Based on Levenshtein distance
func levenshteinSimilarity(s1, s2 string) int {
	if len(s1) == 0 && len(s2) == 0 {
		return 100
	}

	distance := levenshteinDistance(s1, s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	if maxLen == 0 {
		return 100
	}

	similarity := 100 - (distance * 100 / maxLen)
	return similarity
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	d := make([][]int, len(s1)+1)
	for i := range d {
		d[i] = make([]int, len(s2)+1)
	}

	// Initialize
	for i := 0; i <= len(s1); i++ {
		d[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		d[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}

	return d[len(s1)][len(s2)]
}

// min returns the minimum of multiple integers
func min(nums ...int) int {
	result := nums[0]
	for _, n := range nums[1:] {
		if n < result {
			result = n
		}
	}
	return result
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
	// Check filename conventions based on brain type
	basename := filepath.Base(relPath)

	// Skip hidden files and special names
	if strings.HasPrefix(basename, ".") {
		return nil
	}

	// Logseq: Skip journal files - they use YYYY_MM_DD.md format (Logseq standard)
	if c.brainType == BrainTypeLogseq && strings.HasPrefix(relPath, "journals/") {
		return nil
	}

	// Logseq specific: Pages should NOT have YYYY_MM_DD___ prefix
	// Check this FIRST before general kebab-case check
	if c.brainType == BrainTypeLogseq {
		// Check if file is in pages/ directory with date prefix
		// Format: YYYY_MM_DD___title.md (positions: 0-3_5-6_8-9___rest)
		if strings.HasPrefix(relPath, "pages/") &&
			len(basename) > 13 &&
			basename[4] == '_' &&
			basename[7] == '_' &&
			basename[10:13] == "___" {
			// This is a pages/ file with YYYY_MM_DD___ prefix - wrong!
			correctName := basename[13:] // Remove date prefix (YYYY_MM_DD___ = 13 chars)
			// Also convert to kebab-case
			correctName = toKebabCase(strings.TrimSuffix(correctName, ".md")) + ".md"

			return &Issue{
				Type:     IssueTypeWrongFilename,
				Severity: SeverityWarning,
				File:     relPath,
				Message:  fmt.Sprintf("Logseq pages should not have date prefix. Expected: %s", correctName),
				Details:  fmt.Sprintf("Rename to: %s (date belongs in frontmatter, not filename)", correctName),
			}
		}
	}

	// All brain types: markdown files should be lowercase kebab-case
	if strings.HasSuffix(basename, ".md") {
		nameWithoutExt := strings.TrimSuffix(basename, ".md")

		// Check for various naming violations
		hasUppercase := false
		hasSpaces := strings.Contains(nameWithoutExt, " ")
		hasUnderscores := strings.Contains(nameWithoutExt, "_")

		for _, char := range nameWithoutExt {
			if char >= 'A' && char <= 'Z' {
				hasUppercase = true
				break
			}
		}

		// Generate suggested kebab-case filename
		if hasUppercase || hasSpaces || hasUnderscores {
			suggested := toKebabCase(nameWithoutExt)

			var reason string
			if hasUppercase && hasSpaces {
				reason = "Filename contains uppercase letters and spaces"
			} else if hasUppercase && hasUnderscores {
				reason = "Filename contains uppercase letters and underscores"
			} else if hasUppercase {
				reason = "Filename contains uppercase letters"
			} else if hasSpaces {
				reason = "Filename contains spaces"
			} else {
				reason = "Filename contains underscores"
			}

			return &Issue{
				Type:     IssueTypeWrongFilename,
				Severity: SeverityWarning,
				File:     relPath,
				Message:  fmt.Sprintf("Filename should be lowercase kebab-case: %s.md", suggested),
				Details:  fmt.Sprintf("%s. Rename to: %s.md", reason, suggested),
			}
		}
	}

	return nil
}

// checkMissingMetadata checks if a file has required metadata (title, created-at, etc.)
func (c *Checker) checkMissingMetadata(relPath string) []Issue {
	var issues []Issue

	// Skip template files
	if strings.Contains(relPath, "template") {
		return issues
	}

	// Journal files: check for minimum journal metadata (brain field)
	if isJournalPath(relPath) {
		return c.checkJournalMetadata(relPath)
	}

	fullPath := filepath.Join(c.brainPath, relPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return issues
	}

	contentStr := string(content)
	basename := filepath.Base(relPath)

	// Check for metadata based on brain type
	switch c.brainType {
	case BrainTypeLogseq:
		// Logseq uses property format: property:: value
		hasTitle := strings.Contains(contentStr, "title::") || strings.Contains(contentStr, "title:: ")
		hasCreatedAt := strings.Contains(contentStr, "created-at::")

		// Check if file has any content (not just empty)
		trimmed := strings.TrimSpace(contentStr)
		if len(trimmed) == 0 {
			return issues // Empty files don't need metadata warnings
		}

		var missing []string
		if !hasTitle {
			missing = append(missing, "title::")
		}
		if !hasCreatedAt {
			missing = append(missing, "created-at::")
		}

		if len(missing) > 0 {
			// Generate suggested title from filename
			titleSuggestion := fileNameToTitle(strings.TrimSuffix(basename, ".md"))

			issues = append(issues, Issue{
				Type:     IssueTypeMissingMetadata,
				Severity: SeverityInfo,
				File:     relPath,
				Message:  fmt.Sprintf("Missing Logseq properties: %s", strings.Join(missing, ", ")),
				Details:  fmt.Sprintf("Add to first block: title:: %s, created-at:: <unix-ms-timestamp>", titleSuggestion),
			})
		}

	case BrainTypeFlip, BrainTypeFoam, BrainTypeObsidian:
		// These use YAML frontmatter
		hasFrontmatter := strings.HasPrefix(contentStr, "---")
		if !hasFrontmatter {
			// Check if file has any content
			trimmed := strings.TrimSpace(contentStr)
			if len(trimmed) == 0 {
				return issues // Empty files don't need metadata warnings
			}

			titleSuggestion := fileNameToTitle(strings.TrimSuffix(basename, ".md"))
			issues = append(issues, Issue{
				Type:     IssueTypeMissingMetadata,
				Severity: SeverityInfo,
				File:     relPath,
				Message:  "Missing YAML frontmatter",
				Details:  fmt.Sprintf("Add frontmatter: ---\ntitle: %s\ncreated: <date>\n---", titleSuggestion),
			})
		} else {
			// Check if frontmatter has required fields
			// Find end of frontmatter
			endIdx := strings.Index(contentStr[3:], "---")
			if endIdx > 0 {
				frontmatter := contentStr[3 : endIdx+3]
				hasTitle := strings.Contains(frontmatter, "title:")
				hasCreated := strings.Contains(frontmatter, "created:") || strings.Contains(frontmatter, "date:")

				var missing []string
				if !hasTitle {
					missing = append(missing, "title")
				}
				if !hasCreated {
					missing = append(missing, "created")
				}

				if len(missing) > 0 {
					issues = append(issues, Issue{
						Type:     IssueTypeMissingMetadata,
						Severity: SeverityInfo,
						File:     relPath,
						Message:  fmt.Sprintf("Frontmatter missing fields: %s", strings.Join(missing, ", ")),
						Details:  "Add missing metadata fields to frontmatter",
					})
				}
			}
		}
	}

	return issues
}

func (c *Checker) checkJournalMetadata(relPath string) []Issue {
	var issues []Issue

	fullPath := filepath.Join(c.brainPath, relPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return issues
	}

	contentStr := strings.TrimSpace(string(content))
	if contentStr == "" {
		return issues
	}

	switch c.brainType {
	case BrainTypeLogseq:
		if !strings.Contains(contentStr, "brain::") {
			issues = append(issues, Issue{
				Type:     IssueTypeMissingMetadata,
				Severity: SeverityInfo,
				File:     relPath,
				Message:  "Journal missing metadata field: brain::",
				Details:  "Add journal property: brain:: <brain-name>",
			})
		}

	case BrainTypeFlip, BrainTypeFoam, BrainTypeObsidian:
		hasFrontmatter := strings.HasPrefix(contentStr, "---")
		if !hasFrontmatter {
			issues = append(issues, Issue{
				Type:     IssueTypeMissingMetadata,
				Severity: SeverityInfo,
				File:     relPath,
				Message:  "Journal missing YAML frontmatter",
				Details:  "Add frontmatter with at least: brain: <brain-name>",
			})
			return issues
		}

		endIdx := strings.Index(contentStr[3:], "---")
		if endIdx > 0 {
			frontmatter := contentStr[3 : endIdx+3]
			if !strings.Contains(frontmatter, "brain:") {
				issues = append(issues, Issue{
					Type:     IssueTypeMissingMetadata,
					Severity: SeverityInfo,
					File:     relPath,
					Message:  "Journal frontmatter missing field: brain",
					Details:  "Add brain: <brain-name> to journal frontmatter",
				})
			}
		}
	}

	return issues
}

func isJournalPath(relPath string) bool {
	normalized := strings.ToLower(filepath.ToSlash(relPath))
	return strings.HasPrefix(normalized, "journal/") ||
		strings.HasPrefix(normalized, "journals/") ||
		strings.Contains(normalized, "/journal/") ||
		strings.Contains(normalized, "/journals/")
}

// fileNameToTitle converts a kebab-case filename to a readable title
func fileNameToTitle(filename string) string {
	// Replace hyphens and underscores with spaces
	title := strings.ReplaceAll(filename, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Capitalize each word
	words := strings.Fields(title)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// toKebabCase converts a string to lowercase kebab-case
func toKebabCase(s string) string {
	// Replace spaces and underscores with hyphens
	result := strings.ReplaceAll(s, " ", "-")
	result = strings.ReplaceAll(result, "_", "-")
	// Convert to lowercase
	result = strings.ToLower(result)
	// Replace multiple hyphens with single hyphen
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")
	return result
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

// checkPotentialMissingJournalLinks checks if recent notes might not be linked in journals
// This is informational only, as users can intentionally skip journal links
func (c *Checker) checkPotentialMissingJournalLinks() []Issue {
	var issues []Issue

	notesDir := filepath.Join(c.brainPath, "notes")
	journalDir := filepath.Join(c.brainPath, "journal")

	// Check if notes directory exists
	if _, err := os.Stat(notesDir); err != nil {
		return issues
	}

	// Check if journal directory exists
	if _, err := os.Stat(journalDir); err != nil {
		return issues // No journal directory, can't check
	}

	// Get files from past 7 days
	now := time.Now()
	recentDate := now.AddDate(0, 0, -7) // 7 days ago

	// Read note files and check if they're mentioned in journals
	entries, err := os.ReadDir(notesDir)
	if err != nil {
		return issues
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		// Get file modification time
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Only check recently modified notes (past 7 days)
		if info.ModTime().Before(recentDate) {
			continue
		}

		noteName := entry.Name()

		// Extract date if exists (for notes, should be YYYY-MM-DD.md format initially created)
		// For Flip brains, note files are just title.md
		// We can't reliably determine which journal entry should contain it
		// So we just note that recent notes exist

		// Check if this note is referenced in any journal file
		noteTitle := strings.TrimSuffix(noteName, ".md")
		found := false

		journalEntries, err := os.ReadDir(journalDir)
		if err != nil {
			continue
		}

		for _, jentry := range journalEntries {
			if jentry.IsDir() || !strings.HasSuffix(jentry.Name(), ".md") {
				continue
			}

			// Read journal file
			journalPath := filepath.Join(journalDir, jentry.Name())
			content, err := os.ReadFile(journalPath)
			if err != nil {
				continue
			}

			// Check if note name is mentioned in journal
			if strings.Contains(string(content), noteName) || strings.Contains(string(content), noteTitle) {
				found = true
				break
			}
		}

		// Only report as info if not found (optional suggestion)
		if !found {
			relPath := filepath.Join("notes", noteName)
			issue := Issue{
				Type:     IssueTypeFormat,
				Severity: SeverityInfo,
				File:     relPath,
				Message:  "Recent note might not be linked in journal",
				Details:  "This note was recently created/modified but isn't referenced in any journal entry. This is optional - you can skip journal links if not needed.",
			}
			issues = append(issues, issue)
		}
	}

	return issues
}
// checkJournalTitles checks for malformed journal titles (space-separated dates like "2026 01 13")
func (c *Checker) checkJournalTitles() []Issue {
	issues := make([]Issue, 0)
	journalDir := filepath.Join(c.brainPath, "journal")

	// If no journal directory exists, skip
	if _, err := os.Stat(journalDir); err != nil {
		return issues
	}

	// Pattern to match malformed titles: "YYYY MM DD" (space-separated dates)
	malformedPattern := regexp.MustCompile(`title:\s*(\d{4})\s+(\d{2})\s+(\d{2})(?:\s|$)`)

	// Walk through all files in journal directory
	filepath.WalkDir(journalDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		// Only check markdown files
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Check for malformed title
		if matches := malformedPattern.FindStringSubmatch(string(content)); matches != nil {
			relPath, _ := filepath.Rel(c.brainPath, path)
			correctTitle := fmt.Sprintf("%s-%s-%s", matches[1], matches[2], matches[3])
			malformedTitle := fmt.Sprintf("%s %s %s", matches[1], matches[2], matches[3])

			issue := Issue{
				Type:     IssueTypeMalformedTitle,
				Severity: SeverityWarning,
				File:     relPath,
				Message:  fmt.Sprintf("Journal title has incorrect format: '%s'", malformedTitle),
				Details:  fmt.Sprintf("Change 'title: %s' to 'title: %s' (use hyphens not spaces, ISO 8601 format)", malformedTitle, correctTitle),
			}
			issues = append(issues, issue)
		}

		return nil
	})

	return issues
}