package health

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// BrainType represents the type of brain/vault
type BrainType string

const (
	BrainTypeFlip     BrainType = "flip"
	BrainTypeLogseq   BrainType = "logseq"
	BrainTypeObsidian BrainType = "obsidian"
	BrainTypeDendron  BrainType = "dendron"
	BrainTypeUnknown  BrainType = "unknown"
)

// BrainInfo contains information about a brain
type BrainInfo struct {
	Path       string
	Type       BrainType
	Name       string
	NoteCount  int
	AssetCount int
}

// DetectBrainType analyzes a directory and determines the brain type
func DetectBrainType(brainPath string) (BrainType, error) {
	// Check for Flip brain
	if _, err := os.Stat(filepath.Join(brainPath, ".flip.yaml")); err == nil {
		return BrainTypeFlip, nil
	}

	// Check for Logseq brain
	logseqConfig := filepath.Join(brainPath, "logseq", "config.edn")
	if _, err := os.Stat(logseqConfig); err == nil {
		return BrainTypeLogseq, nil
	}

	// Check for Obsidian vault
	obsidianConfig := filepath.Join(brainPath, ".obsidian")
	if stat, err := os.Stat(obsidianConfig); err == nil && stat.IsDir() {
		return BrainTypeObsidian, nil
	}

	// Check for Dendron workspace
	dendronConfig := filepath.Join(brainPath, "dendron.yml")
	if _, err := os.Stat(dendronConfig); err == nil {
		return BrainTypeDendron, nil
	}

	// Check for common patterns
	hasJournals, _ := exists(filepath.Join(brainPath, "journals"))
	hasPages, _ := exists(filepath.Join(brainPath, "pages"))

	if hasJournals && hasPages {
		// Likely Logseq without config
		return BrainTypeLogseq, nil
	}

	return BrainTypeUnknown, fmt.Errorf("could not determine brain type for: %s", brainPath)
}

// AnalyzeBrain performs a quick analysis of the brain
func AnalyzeBrain(brainPath string) (*BrainInfo, error) {
	brainType, err := DetectBrainType(brainPath)
	if err != nil {
		return nil, err
	}

	info := &BrainInfo{
		Path: brainPath,
		Type: brainType,
		Name: filepath.Base(brainPath),
	}

	// Count notes (markdown files)
	noteCount := 0
	assetCount := 0

	err = filepath.Walk(brainPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if fileInfo.IsDir() {
			// Skip hidden and config directories
			if strings.HasPrefix(fileInfo.Name(), ".") ||
				fileInfo.Name() == "logseq" ||
				fileInfo.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(fileInfo.Name()))

		// Count markdown files
		if ext == ".md" {
			noteCount++
		}

		// Count assets (images, videos, pdfs)
		if isAsset(ext) {
			assetCount++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	info.NoteCount = noteCount
	info.AssetCount = assetCount

	return info, nil
}

// isAsset checks if file extension is an asset type
func isAsset(ext string) bool {
	assets := []string{
		".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg",
		".mp4", ".mov", ".avi", ".webm",
		".pdf", ".doc", ".docx",
		".mp3", ".wav", ".ogg",
	}

	for _, a := range assets {
		if ext == a {
			return true
		}
	}
	return false
}

// exists checks if a path exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// GetBrainTypeString returns a human-readable brain type
func (bt BrainType) String() string {
	switch bt {
	case BrainTypeFlip:
		return "Flip Brain"
	case BrainTypeLogseq:
		return "Logseq Graph"
	case BrainTypeObsidian:
		return "Obsidian Vault"
	case BrainTypeDendron:
		return "Dendron Workspace"
	default:
		return "Unknown"
	}
}

// LinkPattern represents different link formats
type LinkPattern struct {
	Name    string
	Regex   *regexp.Regexp
	Extract func(string) string // Extract the target from match
}

// GetLinkPatterns returns link patterns for a brain type
func GetLinkPatterns(brainType BrainType) []LinkPattern {
	patterns := []LinkPattern{
		// Markdown links: [text](path.md)
		{
			Name:  "markdown",
			Regex: regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`),
			Extract: func(match string) string {
				re := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
				matches := re.FindStringSubmatch(match)
				if len(matches) >= 3 {
					return matches[2] // Return path part
				}
				return ""
			},
		},
		// Wikilinks: [[note]] or [[note|alias]]
		{
			Name:  "wikilink",
			Regex: regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`),
			Extract: func(match string) string {
				re := regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)
				matches := re.FindStringSubmatch(match)
				if len(matches) >= 2 {
					return matches[1] // Return note name
				}
				return ""
			},
		},
	}

	// Logseq-specific: Block references ((uuid))
	if brainType == BrainTypeLogseq {
		patterns = append(patterns, LinkPattern{
			Name:  "block-ref",
			Regex: regexp.MustCompile(`\(\(([a-f0-9-]+)\)\)`),
			Extract: func(match string) string {
				re := regexp.MustCompile(`\(\(([a-f0-9-]+)\)\)`)
				matches := re.FindStringSubmatch(match)
				if len(matches) >= 2 {
					return matches[1]
				}
				return ""
			},
		})
	}

	return patterns
}
