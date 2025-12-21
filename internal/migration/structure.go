package migration

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/health"
)

// BrainStructure defines how a brain type organizes its content
// This is CRITICAL for high compatibility - each brain has different conventions
type BrainStructure struct {
	Type health.BrainType

	// Directory conventions - where content lives
	NotesDir       string // Where notes are stored (can be empty for flat structure)
	JournalDir     string // Where journal entries go
	MeetingsDir    string // Where meetings go (Flip-specific)
	TasksDir       string // Where tasks are stored
	AssetsDir      string // Default assets location
	TemplatesDir   string // Where templates live
	AttachmentsDir string // Alternative name for assets (Obsidian uses this)

	// File naming conventions - critical for compatibility
	JournalFormat string // Date format for journal entries (YYYY-MM-DD vs YYYY_MM_DD!)
	NoteFormat    string // How notes are named

	// Asset handling - THIS IS KEY FOR YOUR CONCERN
	AssetStrategy  AssetPlacementStrategy // Where assets should go
	AssetLocations []string               // Multiple possible asset locations to check

	// Link format preferences
	PreferredLinkStyle LinkStyle
	SupportsWikilinks  bool
	SupportsMarkdown   bool
}

// AssetPlacementStrategy determines where assets should be placed
// Different brain types have VERY different conventions here!
type AssetPlacementStrategy string

const (
	// Assets in single top-level folder (Logseq, Flip default)
	AssetStrategyRootFolder AssetPlacementStrategy = "root-folder"

	// Assets next to the note that references them (Obsidian common pattern)
	AssetStrategyNearNote AssetPlacementStrategy = "near-note"

	// Assets in subfolder of note's directory
	AssetStrategySubfolder AssetPlacementStrategy = "subfolder"

	// Assets organized by type (images/, pdfs/, etc.)
	AssetStrategyByType AssetPlacementStrategy = "by-type"
)

// LinkStyle defines how links are written
type LinkStyle string

const (
	LinkStyleWikilink LinkStyle = "wikilink" // [[Note Name]]
	LinkStyleMarkdown LinkStyle = "markdown" // [text](path.md)
	LinkStyleMixed    LinkStyle = "mixed"    // Both allowed
)

// GetBrainStructure returns the structure conventions for a brain type
// This encodes all the knowledge about how each brain type works
func GetBrainStructure(brainType health.BrainType) *BrainStructure {
	switch brainType {
	case health.BrainTypeFlip:
		return &BrainStructure{
			Type:               health.BrainTypeFlip,
			NotesDir:           "notes",
			JournalDir:         "journal",
			MeetingsDir:        "meetings",
			TasksDir:           "tasks",
			AssetsDir:          "assets",
			TemplatesDir:       "templates",
			JournalFormat:      "2006-01-02",                                            // YYYY-MM-DD
			NoteFormat:         "slug-with-date",                                        // 2025-10-27-note-title.md
			AssetStrategy:      AssetStrategyRootFolder,                                 // assets/image.png
			AssetLocations:     []string{"assets", "assets/images", "assets/documents"}, // Check these in order
		PreferredLinkStyle: LinkStyleMarkdown,                                       // Standard Markdown links [text](path.md)
		SupportsWikilinks:  true,                                                    // Can READ wikilinks (for migration)
			SupportsMarkdown:   true,
		}

	case health.BrainTypeLogseq:
		return &BrainStructure{
			Type:               health.BrainTypeLogseq,
			NotesDir:           "pages",
			JournalDir:         "journals",
			TasksDir:           "", // Tasks are blocks, not separate files
			AssetsDir:          "assets",
			TemplatesDir:       "templates",
			JournalFormat:      "2006_01_02",                    // YYYY_MM_DD (UNDERSCORES not dashes!)
			NoteFormat:         "title-case",                    // "Title Case.md" with spaces
			AssetStrategy:      AssetStrategyRootFolder,         // Always assets/ folder
			AssetLocations:     []string{"assets", "../assets"}, // Logseq sometimes uses ../assets
			PreferredLinkStyle: LinkStyleWikilink,               // Strongly prefers wikilinks
			SupportsWikilinks:  true,
			SupportsMarkdown:   true, // Supports markdown syntax in content, but prefers wikilinks for references
		}

	case health.BrainTypeObsidian:
		return &BrainStructure{
			Type:               health.BrainTypeObsidian,
			NotesDir:           "",            // NO enforced structure - user-defined!
			JournalDir:         "",            // Often "Daily Notes" or "Journal" but configurable
			TasksDir:           "",            // User-defined
			AssetsDir:          "attachments", // Most common convention
			AttachmentsDir:     "attachments",
			TemplatesDir:       "templates",
			JournalFormat:      "2006-01-02",                                               // YYYY-MM-DD (but configurable)
			NoteFormat:         "free-form",                                                // ANY structure allowed
			AssetStrategy:      AssetStrategyNearNote,                                      // Often keeps assets NEXT to notes!
			AssetLocations:     []string{"attachments", "assets", "files", ".attachments"}, // Check many locations
			PreferredLinkStyle: LinkStyleWikilink,
			SupportsWikilinks:  true,
			SupportsMarkdown:   true,
		}

	case health.BrainTypeDendron:
		return &BrainStructure{
			Type:               health.BrainTypeDendron,
			NotesDir:           "", // Flat structure with dot-notation hierarchy
			JournalDir:         "", // daily.YYYY.MM.DD.md pattern
			TasksDir:           "",
			AssetsDir:          "assets",
			TemplatesDir:       "templates",
			JournalFormat:      "2006.01.02",            // YYYY.MM.DD (DOTS not dashes!)
			NoteFormat:         "dot-notation",          // parent.child.grandchild.md
			AssetStrategy:      AssetStrategyRootFolder, // assets/ folder
			AssetLocations:     []string{"assets", "vault/assets"},
			PreferredLinkStyle: LinkStyleWikilink,
			SupportsWikilinks:  true,
			SupportsMarkdown:   true,
		}

	case health.BrainTypeFoam:
		return &BrainStructure{
			Type:               health.BrainTypeFoam,
			NotesDir:           "notes", // Common convention, but flexible
			JournalDir:         "journal",
			MeetingsDir:        "meetings", // Common pattern
			TasksDir:           "",
			AssetsDir:          "attachments", // Common convention
			AttachmentsDir:     "attachments",
			TemplatesDir:       "templates",
			JournalFormat:      "2006-01-02",                            // YYYY-MM-DD (standard)
			NoteFormat:         "free-form",                             // Very flexible like Obsidian
			AssetStrategy:      AssetStrategyRootFolder,                 // Usually attachments/ folder
			AssetLocations:     []string{"attachments", "assets", "docs/attachments"}, // Multiple common locations
			PreferredLinkStyle: LinkStyleWikilink,                       // Wikilinks are core to Foam
			SupportsWikilinks:  true,
			SupportsMarkdown:   true, // Also supports regular markdown links (GitHub-friendly)
		}

	default:
		// Unknown brain type - use safe defaults
		return &BrainStructure{
			Type:               health.BrainTypeUnknown,
			NotesDir:           "",
			JournalDir:         "",
			AssetsDir:          "assets",
			AssetStrategy:      AssetStrategyRootFolder,
			AssetLocations:     []string{"assets", "attachments", "files"},
			PreferredLinkStyle: LinkStyleMarkdown, // Most compatible
			SupportsWikilinks:  true,
			SupportsMarkdown:   true,
		}
	}
}

// ResolveAssetPath finds where an asset is actually located in a brain
// Critical for migration - assets can be in many places!
func (bs *BrainStructure) ResolveAssetPath(brainRoot, assetRef string) []string {
	possiblePaths := make([]string, 0)

	// Handle absolute refs from brain root
	if strings.HasPrefix(assetRef, "/") {
		absPath := filepath.Join(brainRoot, strings.TrimPrefix(assetRef, "/"))
		possiblePaths = append(possiblePaths, absPath)
	}

	// Try each possible asset location
	for _, location := range bs.AssetLocations {
		testPath := filepath.Join(brainRoot, location, filepath.Base(assetRef))
		possiblePaths = append(possiblePaths, testPath)
	}

	// Try relative to brain root
	possiblePaths = append(possiblePaths, filepath.Join(brainRoot, assetRef))

	return possiblePaths
}

// GetAssetTargetPath determines where an asset should be placed in target brain
// This respects the target brain's conventions
func (bs *BrainStructure) GetAssetTargetPath(assetFilename, sourceNotePath string) string {
	switch bs.AssetStrategy {
	case AssetStrategyRootFolder:
		// Place in main assets directory
		return filepath.Join(bs.AssetsDir, assetFilename)

	case AssetStrategyNearNote:
		// Place in same directory as note (Obsidian style)
		noteDir := filepath.Dir(sourceNotePath)
		return filepath.Join(noteDir, assetFilename)

	case AssetStrategySubfolder:
		// Place in subfolder of note's directory
		noteDir := filepath.Dir(sourceNotePath)
		return filepath.Join(noteDir, "assets", assetFilename)

	case AssetStrategyByType:
		// Organize by file type (images/, documents/, etc.)
		ext := strings.ToLower(filepath.Ext(assetFilename))
		typeDir := getAssetTypeDir(ext)
		return filepath.Join(bs.AssetsDir, typeDir, assetFilename)

	default:
		return filepath.Join(bs.AssetsDir, assetFilename)
	}
}

// getAssetTypeDir returns subdirectory for asset based on extension
func getAssetTypeDir(ext string) string {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return "images"
	case ".pdf", ".doc", ".docx":
		return "documents"
	case ".mp4", ".mov", ".avi", ".webm":
		return "videos"
	case ".mp3", ".wav", ".ogg":
		return "audio"
	default:
		return "files"
	}
}

// ConvertLinkFormat converts a link from one format to another
// Preserves aliases and text where possible
func (bs *BrainStructure) ConvertLinkFormat(link, targetPath string, targetStyle LinkStyle) string {
	// Detect current link format
	if strings.HasPrefix(link, "[[") && strings.HasSuffix(link, "]]") {
		// It's a wikilink: [[Note]] or [[Note|Alias]]
		content := strings.TrimSuffix(strings.TrimPrefix(link, "[["), "]]")

		// Handle aliases
		parts := strings.Split(content, "|")
		actualNote := parts[0]
		alias := actualNote
		if len(parts) > 1 {
			alias = parts[1]
		}

		if targetStyle == LinkStyleMarkdown {
			// Convert to markdown: [alias](path)
			return fmt.Sprintf("[%s](%s)", alias, targetPath)
		}
		// Keep as wikilink (maybe update path)
		return link

	} else if strings.Contains(link, "](") {
		// It's a markdown link: [text](path)
		start := strings.Index(link, "[")
		mid := strings.Index(link, "](")
		end := strings.LastIndex(link, ")")

		if start >= 0 && mid > start && end > mid {
			text := link[start+1 : mid]

			if targetStyle == LinkStyleWikilink {
				// Convert to wikilink
				noteName := filepath.Base(targetPath)
				noteName = strings.TrimSuffix(noteName, filepath.Ext(noteName))
				if text != noteName {
					return fmt.Sprintf("[[%s|%s]]", noteName, text)
				}
				return fmt.Sprintf("[[%s]]", noteName)
			}
			// Keep as markdown, update path
			return fmt.Sprintf("[%s](%s)", text, targetPath)
		}
	}

	// Unknown format, return as-is
	return link
}

// GetJournalPath returns the path for a journal entry in this brain
func (bs *BrainStructure) GetJournalPath(date string) string {
	if bs.JournalDir == "" {
		return date + ".md"
	}
	return filepath.Join(bs.JournalDir, date+".md")
}

// GetNotePath returns the path for a note in this brain
func (bs *BrainStructure) GetNotePath(noteTitle string) string {
	// Convert title to appropriate format
	slug := strings.ToLower(noteTitle)
	slug = strings.ReplaceAll(slug, " ", "-")

	if bs.NotesDir == "" {
		return slug + ".md"
	}
	return filepath.Join(bs.NotesDir, slug+".md")
}

// String returns a human-readable description
func (bs *BrainStructure) String() string {
	return fmt.Sprintf("%s Structure (Assets: %s, Links: %s)",
		bs.Type.String(),
		bs.AssetStrategy,
		bs.PreferredLinkStyle,
	)
}
