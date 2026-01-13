package migration

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
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
		// Logseq uses lowercase kebab-case (standardized)
		return t.toSlug(base)

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

// ============================================================================
// FRONTMATTER TRANSFORMER
// ============================================================================
// Converts between YAML frontmatter (Obsidian, Flip, Dendron, Foam) and
// Logseq property bullets (key:: value format)

// FrontmatterStyle represents the metadata style used by a brain type
type FrontmatterStyle int

const (
	FrontmatterYAML     FrontmatterStyle = iota // ---\nkey: value\n---
	FrontmatterLogseq                           // key:: value (property bullets)
	FrontmatterNone                             // No frontmatter
)

// GetFrontmatterStyle returns the frontmatter style for a brain type
func GetFrontmatterStyle(brainType health.BrainType) FrontmatterStyle {
	switch brainType {
	case health.BrainTypeLogseq:
		return FrontmatterLogseq
	case health.BrainTypeFlip, health.BrainTypeObsidian, health.BrainTypeDendron, health.BrainTypeFoam:
		return FrontmatterYAML
	default:
		return FrontmatterNone
	}
}

// TransformContent transforms the entire content of a note between brain types
// This includes frontmatter conversion and any content adjustments
func (t *Transformer) TransformContent(content string) string {
	if t.sourceType == t.targetType {
		return content
	}

	sourceStyle := GetFrontmatterStyle(t.sourceType)
	targetStyle := GetFrontmatterStyle(t.targetType)

	if sourceStyle == targetStyle {
		return content // Same frontmatter style, no conversion needed
	}

	switch {
	case sourceStyle == FrontmatterYAML && targetStyle == FrontmatterLogseq:
		return t.yamlToLogseq(content)
	case sourceStyle == FrontmatterLogseq && targetStyle == FrontmatterYAML:
		return t.logseqToYAML(content)
	default:
		return content
	}
}

// yamlToLogseq converts YAML frontmatter to Logseq property bullets
func (t *Transformer) yamlToLogseq(content string) string {
	// Check if content has YAML frontmatter
	if !strings.HasPrefix(content, "---\n") {
		return content // No frontmatter to convert
	}

	// Find the closing ---
	endIndex := strings.Index(content[4:], "\n---")
	if endIndex == -1 {
		return content // Invalid frontmatter
	}

	// Extract frontmatter and body
	frontmatter := content[4 : 4+endIndex]
	body := content[4+endIndex+4:] // Skip past \n---

	// Parse YAML properties
	properties := t.parseYAMLProperties(frontmatter)

	// Convert to Logseq format
	var result strings.Builder

	// Write properties as bullets at the top
	for _, prop := range properties {
		result.WriteString(prop.Key)
		result.WriteString(":: ")
		result.WriteString(prop.Value)
		result.WriteString("\n")
	}

	// Add body (trimming leading newlines but keeping one separator)
	body = strings.TrimLeft(body, "\n")
	if body != "" && len(properties) > 0 {
		result.WriteString("\n")
	}
	result.WriteString(body)

	return result.String()
}

// logseqToYAML converts Logseq property bullets to YAML frontmatter
func (t *Transformer) logseqToYAML(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	
	var properties []Property
	var bodyLines []string
	inProperties := true

	// Property bullet pattern: key:: value
	propRegex := regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)::(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Once we hit a non-property line, everything else is body
		if !inProperties {
			bodyLines = append(bodyLines, line)
			continue
		}

		// Check for property bullet
		if matches := propRegex.FindStringSubmatch(line); matches != nil {
			key := matches[1]
			value := strings.TrimSpace(matches[2])
			properties = append(properties, Property{Key: key, Value: value})
		} else if strings.TrimSpace(line) == "" && len(properties) > 0 {
			// Empty line after properties - switch to body mode
			inProperties = false
		} else if len(properties) == 0 && strings.TrimSpace(line) == "" {
			// Skip leading empty lines
			continue
		} else {
			// Non-property line - this is body content
			inProperties = false
			bodyLines = append(bodyLines, line)
		}
	}

	// If no properties found, return original content
	if len(properties) == 0 {
		return content
	}

	// Sort properties for consistent output
	sort.Slice(properties, func(i, j int) bool {
		return properties[i].Key < properties[j].Key
	})

	// Build YAML frontmatter
	var result strings.Builder
	result.WriteString("---\n")
	for _, prop := range properties {
		result.WriteString(prop.Key)
		result.WriteString(": ")
		result.WriteString(t.formatYAMLValue(prop.Value))
		result.WriteString("\n")
	}
	result.WriteString("---\n")

	// Add body
	if len(bodyLines) > 0 {
		result.WriteString("\n")
		result.WriteString(strings.Join(bodyLines, "\n"))
	}

	return result.String()
}

// Property represents a key-value metadata property
type Property struct {
	Key   string
	Value string
}

// parseYAMLProperties parses simple YAML key: value pairs
// Note: This handles simple cases, not full YAML (no nested objects, arrays)
func (t *Transformer) parseYAMLProperties(yaml string) []Property {
	var properties []Property
	
	scanner := bufio.NewScanner(strings.NewReader(yaml))
	
	// Simple key: value pattern
	simpleRegex := regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*):\s*(.*)$`)
	// Array item pattern
	arrayItemRegex := regexp.MustCompile(`^\s*-\s*(.+)$`)

	var currentKey string
	var arrayValues []string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for array item
		if currentKey != "" {
			if matches := arrayItemRegex.FindStringSubmatch(line); matches != nil {
				arrayValues = append(arrayValues, strings.TrimSpace(matches[1]))
				continue
			} else {
				// End of array - save it
				if len(arrayValues) > 0 {
					properties = append(properties, Property{
						Key:   currentKey,
						Value: t.formatLogseqArray(arrayValues),
					})
				}
				currentKey = ""
				arrayValues = nil
			}
		}

		// Check for simple key: value
		if matches := simpleRegex.FindStringSubmatch(line); matches != nil {
			key := matches[1]
			value := strings.TrimSpace(matches[2])

			if value == "" {
				// This might be a multi-line value or array
				currentKey = key
			} else {
				properties = append(properties, Property{Key: key, Value: value})
			}
		}
	}

	// Handle any remaining array
	if currentKey != "" && len(arrayValues) > 0 {
		properties = append(properties, Property{
			Key:   currentKey,
			Value: t.formatLogseqArray(arrayValues),
		})
	}

	return properties
}

// formatLogseqArray formats an array for Logseq (comma-separated in brackets)
func (t *Transformer) formatLogseqArray(values []string) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return values[0]
	}
	// Logseq uses [[link1]], [[link2]] for tags or [item1, item2] for simple arrays
	return "[" + strings.Join(values, ", ") + "]"
}

// formatYAMLValue ensures value is properly formatted for YAML
func (t *Transformer) formatYAMLValue(value string) string {
	// Check if it's a Logseq array format [item1, item2]
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		inner := value[1 : len(value)-1]
		items := strings.Split(inner, ", ")
		if len(items) > 1 {
			// Convert to YAML array on same line
			return value // Keep bracket format, it's valid YAML
		}
	}

	// Check if value needs quoting (contains special YAML chars)
	if strings.ContainsAny(value, ":{}[]#&*!|>'\"%@`") {
		// Use double quotes and escape internal quotes
		escaped := strings.ReplaceAll(value, `"`, `\"`)
		return `"` + escaped + `"`
	}

	return value
}

// GetContentTransformationSummary returns a description of content changes
func (t *Transformer) GetContentTransformationSummary() string {
	if t.sourceType == t.targetType {
		return "No content transformation needed"
	}

	sourceStyle := GetFrontmatterStyle(t.sourceType)
	targetStyle := GetFrontmatterStyle(t.targetType)

	if sourceStyle == targetStyle {
		return "Frontmatter style compatible, no conversion needed"
	}

	var changes []string

	switch {
	case sourceStyle == FrontmatterYAML && targetStyle == FrontmatterLogseq:
		changes = append(changes, "Convert YAML frontmatter to Logseq property bullets")
		changes = append(changes, "  ---                    →  title:: My Note")
		changes = append(changes, "  title: My Note         →  tags:: [[tag1]], [[tag2]]")
		changes = append(changes, "  tags: [tag1, tag2]     →")
		changes = append(changes, "  ---")

	case sourceStyle == FrontmatterLogseq && targetStyle == FrontmatterYAML:
		changes = append(changes, "Convert Logseq property bullets to YAML frontmatter")
		changes = append(changes, "  title:: My Note        →  ---")
		changes = append(changes, "  tags:: [[tag1]]        →  title: My Note")
		changes = append(changes, "                         →  tags: [[tag1]]")
		changes = append(changes, "                         →  ---")
	}

	return strings.Join(changes, "\n")
}

// GetFullTransformationSummary returns complete summary including filename and content changes
func (t *Transformer) GetFullTransformationSummary() string {
	if !t.NeedsTransformation() {
		return "No transformation needed (same brain type)"
	}

	var sections []string

	// Filename/structure changes
	structureChanges := t.GetTransformationSummary()
	if structureChanges != "" && structureChanges != "No transformation needed (same brain type)" {
		sections = append(sections, "📁 Structure Changes:")
		sections = append(sections, "   "+strings.ReplaceAll(structureChanges, "\n", "\n   "))
	}

	// Content changes
	contentChanges := t.GetContentTransformationSummary()
	if contentChanges != "" && !strings.HasPrefix(contentChanges, "No ") && !strings.HasPrefix(contentChanges, "Frontmatter style compatible") {
		sections = append(sections, "\n📝 Content Changes:")
		sections = append(sections, "   "+strings.ReplaceAll(contentChanges, "\n", "\n   "))
	}

	if len(sections) == 0 {
		return "Minor adjustments only"
	}

	return strings.Join(sections, "\n")
}

// TransformProperties transforms a map of properties to the target format string
func (t *Transformer) TransformProperties(properties map[string]string) string {
	targetStyle := GetFrontmatterStyle(t.targetType)

	// Sort keys for consistent output
	keys := make([]string, 0, len(properties))
	for k := range properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var result strings.Builder

	switch targetStyle {
	case FrontmatterYAML:
		result.WriteString("---\n")
		for _, key := range keys {
			result.WriteString(key)
			result.WriteString(": ")
			result.WriteString(t.formatYAMLValue(properties[key]))
			result.WriteString("\n")
		}
		result.WriteString("---\n")

	case FrontmatterLogseq:
		for _, key := range keys {
			result.WriteString(key)
			result.WriteString(":: ")
			result.WriteString(properties[key])
			result.WriteString("\n")
		}

	default:
		// No frontmatter style - return empty
		return ""
	}

	return result.String()
}
