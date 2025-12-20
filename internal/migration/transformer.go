package migration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/httrp/flip/internal/health"
)

// Transformer handles content and filename conversions between brain types
type Transformer struct {
	sourceType health.BrainType
	targetType health.BrainType
}

// NewTransformer creates a transformer for converting between brain types
func NewTransformer(sourceType, targetType health.BrainType) *Transformer {
	return &Transformer{
		sourceType: sourceType,
		targetType: targetType,
	}
}

// TransformFilename converts a filename from source to target brain conventions
// This handles journal dates, note naming patterns, etc.
func (t *Transformer) TransformFilename(filename string) string {
	if t.sourceType == t.targetType {
		return filename // No transformation needed
	}

	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	// Try to detect and transform date-based filenames (journals)
	if transformed := t.transformDateFilename(base); transformed != "" {
		return transformed + ext
	}

	// Transform note naming conventions
	return t.transformNoteName(base) + ext
}

// transformDateFilename handles journal/daily note filename conversions
// Returns empty string if not a date-based filename
func (t *Transformer) transformDateFilename(base string) string {
	// Try to parse as date from source format
	date, sourceFormat := t.parseDateFromFilename(base)
	if date.IsZero() {
		return "" // Not a date filename
	}

	// Get target format
	targetFormat := t.getDateFormat(t.targetType)
	if targetFormat == sourceFormat {
		return "" // Same format, no change needed
	}

	// Handle Dendron's special daily.YYYY.MM.DD pattern
	if t.targetType == health.BrainTypeDendron {
		return "daily." + date.Format(targetFormat)
	}

	// Strip "daily." prefix if coming from Dendron
	return date.Format(targetFormat)
}

// parseDateFromFilename extracts date from various filename formats
func (t *Transformer) parseDateFromFilename(base string) (time.Time, string) {
	// Strip "daily." prefix for Dendron
	cleanBase := strings.TrimPrefix(base, "daily.")

	// Try different date formats
	formats := []struct {
		pattern string
		layout  string
	}{
		// Logseq: YYYY_MM_DD (underscores)
		{`^\d{4}_\d{2}_\d{2}$`, "2006_01_02"},
		// Flip/Obsidian/Foam: YYYY-MM-DD (dashes)
		{`^\d{4}-\d{2}-\d{2}$`, "2006-01-02"},
		// Dendron: YYYY.MM.DD (dots)
		{`^\d{4}\.\d{2}\.\d{2}$`, "2006.01.02"},
	}

	for _, f := range formats {
		re := regexp.MustCompile(f.pattern)
		if re.MatchString(cleanBase) {
			if date, err := time.Parse(f.layout, cleanBase); err == nil {
				return date, f.layout
			}
		}
	}

	return time.Time{}, ""
}

// getDateFormat returns the date format string for a brain type
func (t *Transformer) getDateFormat(brainType health.BrainType) string {
	switch brainType {
	case health.BrainTypeLogseq:
		return "2006_01_02" // Underscores!
	case health.BrainTypeDendron:
		return "2006.01.02" // Dots!
	default:
		return "2006-01-02" // Dashes (Flip, Obsidian, Foam)
	}
}

// transformNoteName handles general note naming conventions
func (t *Transformer) transformNoteName(base string) string {
	switch t.targetType {
	case health.BrainTypeLogseq:
		// Logseq prefers "Title Case" with spaces
		return t.toTitleCase(base)

	case health.BrainTypeDendron:
		// Dendron uses dot.notation.hierarchy
		return t.toDotNotation(base)

	case health.BrainTypeFlip:
		// Flip uses lowercase-slug-format
		return t.toSlug(base)

	default:
		// Obsidian/Foam: keep as-is (flexible)
		return base
	}
}

// toTitleCase converts slug-format to Title Case
func (t *Transformer) toTitleCase(s string) string {
	// Replace dashes/underscores with spaces
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	// Title case each word
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

// toSlug converts to lowercase-slug-format
func (t *Transformer) toSlug(s string) string {
	// Replace spaces and dots with dashes
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Lowercase
	s = strings.ToLower(s)

	// Remove multiple dashes
	re := regexp.MustCompile(`-+`)
	s = re.ReplaceAllString(s, "-")

	return strings.Trim(s, "-")
}

// toDotNotation converts to dendron.style.notation
func (t *Transformer) toDotNotation(s string) string {
	// Replace separators with dots
	s = strings.ReplaceAll(s, "-", ".")
	s = strings.ReplaceAll(s, "_", ".")
	s = strings.ReplaceAll(s, " ", ".")

	// Lowercase
	s = strings.ToLower(s)

	// Remove multiple dots
	re := regexp.MustCompile(`\.+`)
	s = re.ReplaceAllString(s, ".")

	return strings.Trim(s, ".")
}

// TransformJournalPath converts full journal path between brain conventions
func (t *Transformer) TransformJournalPath(sourcePath, sourceRoot, targetRoot string) string {
	// Get relative path from source root
	relPath, err := filepath.Rel(sourceRoot, sourcePath)
	if err != nil {
		return sourcePath
	}

	// Get source and target structures
	sourceStruct := GetBrainStructure(t.sourceType)
	targetStruct := GetBrainStructure(t.targetType)

	// Check if this is in the journal directory
	sourceJournalDir := sourceStruct.JournalDir
	if sourceJournalDir != "" && strings.HasPrefix(relPath, sourceJournalDir) {
		// Extract filename
		filename := filepath.Base(relPath)

		// Transform filename
		newFilename := t.TransformFilename(filename)

		// Build new path with target journal directory
		targetJournalDir := targetStruct.JournalDir
		if targetJournalDir == "" {
			targetJournalDir = "journal" // Default fallback
		}

		return filepath.Join(targetRoot, targetJournalDir, newFilename)
	}

	// Not a journal file, just transform filename
	dir := filepath.Dir(relPath)
	filename := filepath.Base(relPath)
	newFilename := t.TransformFilename(filename)

	return filepath.Join(targetRoot, dir, newFilename)
}

// NeedsTransformation checks if any transformation is needed between types
func (t *Transformer) NeedsTransformation() bool {
	return t.sourceType != t.targetType
}

// GetTransformationSummary returns a human-readable description of what will change
func (t *Transformer) GetTransformationSummary() string {
	if !t.NeedsTransformation() {
		return "No transformation needed (same brain type)"
	}

	changes := []string{}

	// Date format changes
	sourceFormat := t.getDateFormat(t.sourceType)
	targetFormat := t.getDateFormat(t.targetType)
	if sourceFormat != targetFormat {
		changes = append(changes, fmt.Sprintf("Journal dates: %s → %s", 
			t.formatExample(sourceFormat), 
			t.formatExample(targetFormat)))
	}

	// Directory changes
	sourceStruct := GetBrainStructure(t.sourceType)
	targetStruct := GetBrainStructure(t.targetType)
	
	if sourceStruct.JournalDir != targetStruct.JournalDir {
		changes = append(changes, fmt.Sprintf("Journal directory: %s/ → %s/",
			sourceStruct.JournalDir, targetStruct.JournalDir))
	}
	
	if sourceStruct.NotesDir != targetStruct.NotesDir {
		changes = append(changes, fmt.Sprintf("Notes directory: %s/ → %s/",
			sourceStruct.NotesDir, targetStruct.NotesDir))
	}

	if len(changes) == 0 {
		return "Minor naming convention adjustments"
	}

	return strings.Join(changes, "\n")
}

// formatExample shows an example date in the given format
func (t *Transformer) formatExample(format string) string {
	example := time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC)
	return example.Format(format)
}
