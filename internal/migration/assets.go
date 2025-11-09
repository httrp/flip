package migration

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// AssetMigrator handles copying assets between brains with structure awareness
type AssetMigrator struct {
	sourceStructure *BrainStructure
	targetStructure *BrainStructure
	sourceBrainRoot string
	targetBrainRoot string

	// Tracking
	assetMap     map[string]string // source path -> target path
	copiedAssets map[string]bool   // track what we've already copied
	errors       []error           // collect errors
}

// NewAssetMigrator creates a new asset migrator
func NewAssetMigrator(
	sourceBrain, targetBrain string,
	sourceStructure, targetStructure *BrainStructure,
) *AssetMigrator {
	return &AssetMigrator{
		sourceStructure: sourceStructure,
		targetStructure: targetStructure,
		sourceBrainRoot: sourceBrain,
		targetBrainRoot: targetBrain,
		assetMap:        make(map[string]string),
		copiedAssets:    make(map[string]bool),
		errors:          make([]error, 0),
	}
}

// MigrateAsset copies an asset from source to target brain
// Returns the new path (relative to target brain root)
func (am *AssetMigrator) MigrateAsset(assetRef, sourceNotePath string) (string, error) {
	// Check if we've already migrated this asset
	if targetPath, exists := am.assetMap[assetRef]; exists {
		return targetPath, nil
	}

	// Step 1: Find the asset in source brain (try multiple locations)
	sourceAssetPath, err := am.findAssetInSource(assetRef)
	if err != nil {
		am.errors = append(am.errors, err)
		return "", fmt.Errorf("asset not found: %s", assetRef)
	}

	// Step 2: Determine target path based on target brain's conventions
	assetFilename := filepath.Base(sourceAssetPath)
	targetRelPath := am.targetStructure.GetAssetTargetPath(assetFilename, sourceNotePath)

	// Step 3: Copy the asset
	targetAbsPath := filepath.Join(am.targetBrainRoot, targetRelPath)
	if err := am.copyAssetFile(sourceAssetPath, targetAbsPath); err != nil {
		am.errors = append(am.errors, err)
		return "", fmt.Errorf("failed to copy asset: %w", err)
	}

	// Track the migration
	am.assetMap[assetRef] = targetRelPath
	am.copiedAssets[targetAbsPath] = true

	return targetRelPath, nil
}

// findAssetInSource tries to locate an asset in the source brain
// Checks multiple possible locations based on source brain structure
func (am *AssetMigrator) findAssetInSource(assetRef string) (string, error) {
	// Get possible paths from source structure
	possiblePaths := am.sourceStructure.ResolveAssetPath(am.sourceBrainRoot, assetRef)

	// Try each path
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			// Found it!
			return path, nil
		}
	}

	// Also try relative to source brain root (without any asset dir)
	directPath := filepath.Join(am.sourceBrainRoot, assetRef)
	if _, err := os.Stat(directPath); err == nil {
		return directPath, nil
	}

	return "", fmt.Errorf("asset not found in any of %d locations: %s",
		len(possiblePaths), assetRef)
}

// copyAssetFile copies a file from source to target
func (am *AssetMigrator) copyAssetFile(sourcePath, targetPath string) error {
	// Check if already copied
	if am.copiedAssets[targetPath] {
		return nil // Already copied, skip
	}

	// Create target directory if needed
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Open source file
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create target file
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target file: %w", err)
	}
	defer targetFile.Close()

	// Copy contents
	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

// UpdateAssetReferences updates asset references in markdown content
// Converts paths and link formats according to target brain conventions
func (am *AssetMigrator) UpdateAssetReferences(
	content string,
	sourceNotePath string,
) (string, error) {
	updated := content
	errors := make([]error, 0)

	// Pattern 1: Markdown images: ![alt](path)
	markdownPattern := `!\[([^\]]*)\]\(([^)]+)\)`
	updated = replaceWithFunc(updated, markdownPattern, func(match string) string {
		alt, path := extractMarkdownImage(match)
		
		// Migrate the asset
		newPath, err := am.MigrateAsset(path, sourceNotePath)
		if err != nil {
			errors = append(errors, err)
			return match // Keep original if migration fails
		}

		// Return updated reference
		return fmt.Sprintf("![%s](%s)", alt, newPath)
	})

	// Pattern 2: Wikilink embeds: ![[image.png]]
	wikilinkPattern := `!\[\[([^\]]+)\]\]`
	updated = replaceWithFunc(updated, wikilinkPattern, func(match string) string {
		assetName := extractWikilinkEmbed(match)
		
		// Migrate the asset
		newPath, err := am.MigrateAsset(assetName, sourceNotePath)
		if err != nil {
			errors = append(errors, err)
			return match
		}

		// Convert format if target brain prefers markdown
		if am.targetStructure.PreferredLinkStyle == LinkStyleMarkdown {
			return fmt.Sprintf("![%s](%s)", assetName, newPath)
		}
		
		// Keep as wikilink but update path
		return fmt.Sprintf("![[%s]]", filepath.Base(newPath))
	})

	// Pattern 3: Plain markdown links to assets (PDFs, etc.)
	assetLinkPattern := `\[([^\]]+)\]\(([^)]+\.(?:pdf|doc|docx|zip|xlsx))\)`
	updated = replaceWithFunc(updated, assetLinkPattern, func(match string) string {
		text, path := extractMarkdownLink(match)
		
		// Migrate the asset
		newPath, err := am.MigrateAsset(path, sourceNotePath)
		if err != nil {
			errors = append(errors, err)
			return match
		}

		return fmt.Sprintf("[%s](%s)", text, newPath)
	})

	if len(errors) > 0 {
		am.errors = append(am.errors, errors...)
		return updated, fmt.Errorf("encountered %d errors updating asset references", len(errors))
	}

	return updated, nil
}

// GetAssetMap returns the mapping of source to target asset paths
func (am *AssetMigrator) GetAssetMap() map[string]string {
	return am.assetMap
}

// GetErrors returns all errors encountered during migration
func (am *AssetMigrator) GetErrors() []error {
	return am.errors
}

// GetStats returns migration statistics
func (am *AssetMigrator) GetStats() AssetMigrationStats {
	return AssetMigrationStats{
		TotalAssets:   len(am.assetMap),
		CopiedAssets:  len(am.copiedAssets),
		FailedAssets:  len(am.errors),
		AssetStrategy: string(am.targetStructure.AssetStrategy),
	}
}

// AssetMigrationStats contains statistics about asset migration
type AssetMigrationStats struct {
	TotalAssets   int
	CopiedAssets  int
	FailedAssets  int
	AssetStrategy string
}

// Helper functions for extracting parts from regex matches

func extractMarkdownImage(match string) (alt, path string) {
	// ![alt](path) -> alt, path
	start := strings.Index(match, "[")
	mid := strings.Index(match, "](")
	end := strings.LastIndex(match, ")")
	
	if start >= 0 && mid > start && end > mid {
		alt = match[start+2 : mid]  // Skip "!["
		path = match[mid+2 : end]   // Skip "]("
	}
	return
}

func extractWikilinkEmbed(match string) string {
	// ![[asset.png]] -> asset.png
	return strings.TrimSuffix(strings.TrimPrefix(match, "![["), "]]")
}

func extractMarkdownLink(match string) (text, path string) {
	// [text](path) -> text, path
	start := strings.Index(match, "[")
	mid := strings.Index(match, "](")
	end := strings.LastIndex(match, ")")
	
	if start >= 0 && mid > start && end > mid {
		text = match[start+1 : mid]
		path = match[mid+2 : end]
	}
	return
}

// replaceWithFunc is a helper that applies a function to each regex match
// Note: This is simplified - in production we'd use regexp.ReplaceAllStringFunc
func replaceWithFunc(content, pattern string, fn func(string) string) string {
	// Simple implementation - for production, use proper regex
	// This is a placeholder for the concept
	return content // TODO: Implement proper regex replacement
}
