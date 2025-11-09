package migration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// NoteLinkRewriter updates internal note links during migration
type NoteLinkRewriter struct {
	sourceStructure *BrainStructure
	targetStructure *BrainStructure
	sourceBrainRoot string
	targetBrainRoot string

	// Map source note paths to target note paths
	notePathMap map[string]string
}

// NewNoteLinkRewriter creates a link rewriter with note mapping
func NewNoteLinkRewriter(
	sourceBrain, targetBrain string,
	sourceStruct, targetStruct *BrainStructure,
	noteMap map[string]string,
) *NoteLinkRewriter {
	return &NoteLinkRewriter{
		sourceStructure: sourceStruct,
		targetStructure: targetStruct,
		sourceBrainRoot: sourceBrain,
		targetBrainRoot: targetBrain,
		notePathMap:     noteMap,
	}
}

// UpdateNoteLinks updates wikilinks and markdown links to other notes
func (nlr *NoteLinkRewriter) UpdateNoteLinks(content, currentNotePath string) (string, error) {
	updated := content

	// Pattern 1: Wikilinks [[Note Title]] or [[Note Title|Alias]]
	wikilinkPattern := regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)
	updated = wikilinkPattern.ReplaceAllStringFunc(updated, func(match string) string {
		parts := wikilinkPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		noteName := strings.TrimSpace(parts[1])
		alias := ""
		if len(parts) >= 3 && parts[2] != "" {
			alias = strings.TrimSpace(parts[2])
		}

		// Resolve to source note path
		sourcePath := nlr.resolveNoteName(noteName)
		if sourcePath == "" {
			return match // Not found, keep original
		}

		// Get target path from map
		targetPath, exists := nlr.notePathMap[sourcePath]
		if !exists {
			return match // Not migrated, keep original
		}

		// Convert to target link format
		return nlr.formatNoteLink(targetPath, noteName, alias)
	})

	// Pattern 2: Markdown links to .md files [text](path.md) or [text](path)
	markdownLinkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+\.md)\)`)
	updated = markdownLinkPattern.ReplaceAllStringFunc(updated, func(match string) string {
		parts := markdownLinkPattern.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		linkText := parts[1]
		linkPath := parts[2]

		// Normalize path (relative to current note)
		sourcePath := nlr.resolveRelativePath(currentNotePath, linkPath)

		// Get target path from map
		targetPath, exists := nlr.notePathMap[sourcePath]
		if !exists {
			return match // Not migrated, keep original
		}

		// Convert to target link format
		return nlr.formatNoteLink(targetPath, linkText, "")
	})

	return updated, nil
}

// resolveNoteName tries to find source note path from a note name/title
func (nlr *NoteLinkRewriter) resolveNoteName(noteName string) string {
	// Normalize search term
	searchSlug := strings.ToLower(strings.ReplaceAll(noteName, " ", "-"))
	searchNorm := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(noteName, " ", ""), "-", ""))
	searchNorm = strings.ReplaceAll(searchNorm, "_", "")

	for sourcePath := range nlr.notePathMap {
		base := filepath.Base(sourcePath)
		baseNoExt := strings.TrimSuffix(base, filepath.Ext(base))

		// 1. Case-insensitive exact match
		if strings.EqualFold(baseNoExt, noteName) {
			return sourcePath
		}

		// 2. Slug match (spaces -> dashes)
		if strings.EqualFold(baseNoExt, searchSlug) {
			return sourcePath
		}

		// 3. Normalized match (remove all separators)
		baseNorm := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(baseNoExt, "-", ""), "_", ""))
		baseNorm = strings.ReplaceAll(baseNorm, " ", "")
		if baseNorm == searchNorm {
			return sourcePath
		}
	}

	return ""
}

// resolveRelativePath resolves a relative link path from current note
func (nlr *NoteLinkRewriter) resolveRelativePath(currentNotePath, linkPath string) string {
	// If absolute (starts with /), treat as relative to brain root
	if strings.HasPrefix(linkPath, "/") {
		return strings.TrimPrefix(linkPath, "/")
	}

	// Otherwise relative to current note's directory
	currentDir := filepath.Dir(currentNotePath)
	resolved := filepath.Join(currentDir, linkPath)
	cleaned := filepath.Clean(resolved)

	// Check if it exists in map
	if _, exists := nlr.notePathMap[cleaned]; exists {
		return cleaned
	}

	// Try normalized version (maybe path has changed)
	for srcPath := range nlr.notePathMap {
		if strings.HasSuffix(srcPath, filepath.Base(linkPath)) {
			return srcPath
		}
	}

	return cleaned
}

// formatNoteLink creates a link in target brain's preferred format
func (nlr *NoteLinkRewriter) formatNoteLink(targetPath, displayText, alias string) string {
	targetName := filepath.Base(targetPath)
	targetNameNoExt := strings.TrimSuffix(targetName, filepath.Ext(targetName))

	// Use provided alias or display text, fall back to target name
	displayName := alias
	if displayName == "" {
		displayName = displayText
	}
	if displayName == "" {
		displayName = targetNameNoExt
	}

	// Format according to target brain preferences
	if nlr.targetStructure.PreferredLinkStyle == LinkStyleWikilink {
		// Wikilink format
		if displayName != targetNameNoExt {
			return fmt.Sprintf("[[%s|%s]]", targetNameNoExt, displayName)
		}
		return fmt.Sprintf("[[%s]]", targetNameNoExt)
	}

	// Markdown format
	// Use relative path from target structure
	linkPath := targetPath
	if nlr.targetStructure.NotesDir != "" && strings.HasPrefix(targetPath, nlr.targetStructure.NotesDir+"/") {
		// Link relative to notes dir
		linkPath = strings.TrimPrefix(targetPath, nlr.targetStructure.NotesDir+"/")
	}

	return fmt.Sprintf("[%s](%s)", displayName, linkPath)
}

// BuildNotePathMap creates a map of source -> target note paths from migration plan
func BuildNotePathMap(plan *MigrationPlan) map[string]string {
	noteMap := make(map[string]string)

	for _, item := range plan.Items {
		if item.Type == "note" || item.Type == "journal" || item.Type == "definition" {
			noteMap[item.SourcePath] = item.TargetPath
		}
	}

	return noteMap
}
