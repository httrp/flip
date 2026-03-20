package schema

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Schema represents the structure definition for notes
type Schema struct {
	Version     string            `yaml:"version"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at"`
	UpdatedAt   time.Time         `yaml:"updated_at"`
	Checksum    string            `yaml:"checksum"`
	Fields      []SchemaField     `yaml:"fields"`
	Sections    []SchemaSection   `yaml:"sections,omitempty"`
	Migrations  []SchemaMigration `yaml:"migrations,omitempty"`
}

// SchemaField represents a metadata field in the schema
type SchemaField struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"` // string, date, tags, number, boolean
	Required    bool     `yaml:"required,omitempty"`
	Default     string   `yaml:"default,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Aliases     []string `yaml:"aliases,omitempty"` // Old names for migration
}

// SchemaSection represents a required section in the note structure
type SchemaSection struct {
	Name        string `yaml:"name"`
	Heading     string `yaml:"heading"`     // e.g., "## Summary"
	Required    bool   `yaml:"required,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// SchemaMigration defines a migration from one version to another
type SchemaMigration struct {
	FromVersion  string            `yaml:"from_version"`
	ToVersion    string            `yaml:"to_version"`
	Description  string            `yaml:"description"`
	FieldChanges []FieldChange     `yaml:"field_changes,omitempty"`
	Automatic    bool              `yaml:"automatic"` // Can be auto-applied
}

// FieldChange represents a change to a field during migration
type FieldChange struct {
	Action   string `yaml:"action"`    // rename, remove, add, transform
	OldName  string `yaml:"old_name,omitempty"`
	NewName  string `yaml:"new_name,omitempty"`
	Default  string `yaml:"default,omitempty"`
	Transform string `yaml:"transform,omitempty"` // For complex transforms
}

// SchemaVersion tracks which schema version a note uses
type SchemaVersion struct {
	SchemaName    string    `yaml:"schema"`
	SchemaVersion string    `yaml:"schema_version"`
	LastMigrated  time.Time `yaml:"last_migrated,omitempty"`
}

// SchemaManager handles schema registration and evolution
type SchemaManager struct {
	brainPath string
	schemas   map[string]*Schema // name -> schema
}

// NewSchemaManager creates a new schema manager for a brain
func NewSchemaManager(brainPath string) *SchemaManager {
	return &SchemaManager{
		brainPath: brainPath,
		schemas:   make(map[string]*Schema),
	}
}

// LoadSchemas loads all schema definitions from the brain
func (sm *SchemaManager) LoadSchemas() error {
	schemasDir := filepath.Join(sm.brainPath, "definitions", "schemas")
	
	// Check if schemas directory exists
	if _, err := os.Stat(schemasDir); os.IsNotExist(err) {
		// No schemas defined - that's okay
		return nil
	}

	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		return fmt.Errorf("failed to read schemas directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(schemasDir, entry.Name())
		schema, err := sm.loadSchemaFile(path)
		if err != nil {
			return fmt.Errorf("failed to load schema %s: %w", entry.Name(), err)
		}

		sm.schemas[schema.Name] = schema
	}

	return nil
}

// loadSchemaFile loads a single schema from a YAML file
func (sm *SchemaManager) loadSchemaFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var schema Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	// Compute checksum from fields and sections
	schema.Checksum = sm.computeChecksum(&schema)

	return &schema, nil
}

// computeChecksum creates a hash of the schema structure
func (sm *SchemaManager) computeChecksum(schema *Schema) string {
	// Create a canonical representation
	var parts []string
	for _, f := range schema.Fields {
		parts = append(parts, fmt.Sprintf("field:%s:%s:%v", f.Name, f.Type, f.Required))
	}
	for _, s := range schema.Sections {
		parts = append(parts, fmt.Sprintf("section:%s:%s:%v", s.Name, s.Heading, s.Required))
	}
	
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(hash[:8]) // Short hash
}

// GetSchema returns a schema by name
func (sm *SchemaManager) GetSchema(name string) (*Schema, bool) {
	schema, ok := sm.schemas[name]
	return schema, ok
}

// ListSchemas returns all registered schema names
func (sm *SchemaManager) ListSchemas() []string {
	names := make([]string, 0, len(sm.schemas))
	for name := range sm.schemas {
		names = append(names, name)
	}
	return names
}

// ValidateNote checks if a note conforms to its declared schema
func (sm *SchemaManager) ValidateNote(notePath string) (*ValidationResult, error) {
	content, err := os.ReadFile(notePath)
	if err != nil {
		return nil, err
	}

	result := &ValidationResult{
		NotePath: notePath,
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Extract schema version from frontmatter
	schemaVersion := extractSchemaVersion(string(content))
	if schemaVersion == nil {
		result.Warnings = append(result.Warnings, "No schema version declared")
		return result, nil
	}

	result.DeclaredSchema = schemaVersion.SchemaName
	result.DeclaredVersion = schemaVersion.SchemaVersion

	// Get the schema
	schema, ok := sm.schemas[schemaVersion.SchemaName]
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, 
			fmt.Sprintf("Unknown schema: %s", schemaVersion.SchemaName))
		return result, nil
	}

	// Check version
	if schemaVersion.SchemaVersion != schema.Version {
		result.NeedsMigration = true
		result.CurrentVersion = schemaVersion.SchemaVersion
		result.LatestVersion = schema.Version
	}

	// Validate fields
	frontmatter := extractFrontmatter(string(content))
	for _, field := range schema.Fields {
		if field.Required {
			if _, ok := frontmatter[field.Name]; !ok {
				// Check aliases
				found := false
				for _, alias := range field.Aliases {
					if _, ok := frontmatter[alias]; ok {
						found = true
						result.Warnings = append(result.Warnings,
							fmt.Sprintf("Field '%s' uses deprecated name '%s'", field.Name, alias))
						break
					}
				}
				if !found {
					result.Valid = false
					result.Errors = append(result.Errors,
						fmt.Sprintf("Missing required field: %s", field.Name))
				}
			}
		}
	}

	// Validate sections
	for _, section := range schema.Sections {
		if section.Required {
			if !hasSection(string(content), section.Heading) {
				result.Valid = false
				result.Errors = append(result.Errors,
					fmt.Sprintf("Missing required section: %s", section.Heading))
			}
		}
	}

	return result, nil
}

// ValidationResult contains the result of schema validation
type ValidationResult struct {
	NotePath        string
	Valid           bool
	DeclaredSchema  string
	DeclaredVersion string
	CurrentVersion  string
	LatestVersion   string
	NeedsMigration  bool
	Errors          []string
	Warnings        []string
}

// MigrateNote updates a note to the latest schema version
func (sm *SchemaManager) MigrateNote(notePath string, dryRun bool) (*MigrationResult, error) {
	validation, err := sm.ValidateNote(notePath)
	if err != nil {
		return nil, err
	}

	result := &MigrationResult{
		NotePath:    notePath,
		FromVersion: validation.CurrentVersion,
		ToVersion:   validation.LatestVersion,
		Changes:     []string{},
	}

	if !validation.NeedsMigration {
		result.Success = true
		result.Message = "Already at latest version"
		return result, nil
	}

	schema, ok := sm.schemas[validation.DeclaredSchema]
	if !ok {
		return nil, fmt.Errorf("schema not found: %s", validation.DeclaredSchema)
	}

	// Find migration path
	migration := findMigration(schema, validation.CurrentVersion, validation.LatestVersion)
	if migration == nil {
		result.Success = false
		result.Message = "No migration path available"
		return result, nil
	}

	if !migration.Automatic {
		result.Success = false
		result.Message = "Migration requires manual intervention"
		result.ManualSteps = migration.Description
		return result, nil
	}

	// Read current content
	content, err := os.ReadFile(notePath)
	if err != nil {
		return nil, err
	}

	// Apply migration
	newContent := string(content)
	for _, change := range migration.FieldChanges {
		switch change.Action {
		case "rename":
			renamed, ok := renameField(newContent, change.OldName, change.NewName)
			if !ok {
				result.Changes = append(result.Changes,
					fmt.Sprintf("Warning: field %s not found for rename to %s", change.OldName, change.NewName))
			} else {
				newContent = renamed
			}
			result.Changes = append(result.Changes,
				fmt.Sprintf("Renamed field: %s → %s", change.OldName, change.NewName))
		case "add":
			if change.Default != "" {
				newContent = addField(newContent, change.NewName, change.Default)
				result.Changes = append(result.Changes,
					fmt.Sprintf("Added field: %s = %s", change.NewName, change.Default))
			}
		case "remove":
			newContent = removeField(newContent, change.OldName)
			result.Changes = append(result.Changes,
				fmt.Sprintf("Removed field: %s", change.OldName))
		}
	}

	// Update schema version in frontmatter
	newContent = updateSchemaVersion(newContent, validation.DeclaredSchema, validation.LatestVersion)

	if !dryRun {
		if err := os.WriteFile(notePath, []byte(newContent), 0644); err != nil {
			return nil, err
		}
	}

	result.Success = true
	result.Message = "Migration completed"
	return result, nil
}

// MigrationResult contains the result of a note migration
type MigrationResult struct {
	NotePath    string
	FromVersion string
	ToVersion   string
	Success     bool
	Message     string
	ManualSteps string
	Changes     []string
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func extractSchemaVersion(content string) *SchemaVersion {
	// Look for schema_version in YAML frontmatter or property bullets
	patterns := []string{
		`schema:\s*([^\n]+)`,
		`schema_version:\s*([^\n]+)`,
		`schema::\s*([^\n]+)`,
		`schema_version::\s*([^\n]+)`,
	}

	var schemaName, version string

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(content); len(matches) > 1 {
			value := strings.TrimSpace(matches[1])
			if strings.Contains(pattern, "version") {
				version = value
			} else {
				schemaName = value
			}
		}
	}

	if schemaName == "" {
		return nil
	}

	return &SchemaVersion{
		SchemaName:    schemaName,
		SchemaVersion: version,
	}
}

func extractFrontmatter(content string) map[string]string {
	result := make(map[string]string)

	// YAML frontmatter
	if strings.HasPrefix(content, "---\n") {
		end := strings.Index(content[4:], "\n---")
		if end > 0 {
			fm := content[4 : 4+end]
			for _, line := range strings.Split(fm, "\n") {
				if idx := strings.Index(line, ":"); idx > 0 {
					key := strings.TrimSpace(line[:idx])
					value := strings.TrimSpace(line[idx+1:])
					result[key] = value
				}
			}
		}
	}

	// Property bullets (Logseq style)
	propRegex := regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)::(.*)$`)
	for _, line := range strings.Split(content, "\n") {
		if matches := propRegex.FindStringSubmatch(line); len(matches) > 2 {
			result[matches[1]] = strings.TrimSpace(matches[2])
		}
	}

	return result
}

func hasSection(content, heading string) bool {
	// Check for exact heading match
	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(heading) + `\s*$`)
	return pattern.MatchString(content)
}

func findMigration(schema *Schema, from, to string) *SchemaMigration {
	for _, m := range schema.Migrations {
		if m.FromVersion == from && m.ToVersion == to {
			return &m
		}
	}
	return nil
}

func renameField(content, oldName, newName string) (string, bool) {
	// YAML: old_name: value → new_name: value
	yamlPattern := regexp.MustCompile(`(?m)^(\s*)` + regexp.QuoteMeta(oldName) + `:`)
	if yamlPattern.MatchString(content) {
		return yamlPattern.ReplaceAllString(content, "${1}"+newName+":"), true
	}

	// Logseq: old_name:: value → new_name:: value
	logseqPattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(oldName) + `::`)
	if logseqPattern.MatchString(content) {
		return logseqPattern.ReplaceAllString(content, newName+"::"), true
	}

	return content, false
}

func addField(content, name, value string) string {
	// Add to YAML frontmatter if exists
	if strings.HasPrefix(content, "---\n") {
		end := strings.Index(content[4:], "\n---")
		if end > 0 {
			fm := content[4 : 4+end]
			newFm := fm + "\n" + name + ": " + value
			return "---\n" + newFm + content[4+end:]
		}
	}
	
	// Otherwise add as property bullet at top
	return name + ":: " + value + "\n" + content
}

func removeField(content, name string) string {
	// Remove from YAML
	yamlPattern := regexp.MustCompile(`(?m)^(\s*)` + regexp.QuoteMeta(name) + `:.*\n`)
	content = yamlPattern.ReplaceAllString(content, "")

	// Remove property bullet
	logseqPattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + `::.*\n`)
	content = logseqPattern.ReplaceAllString(content, "")

	return content
}

func updateSchemaVersion(content, schemaName, version string) string {
	// Update in YAML
	versionPattern := regexp.MustCompile(`(?m)^(\s*)schema_version:.*$`)
	if versionPattern.MatchString(content) {
		return versionPattern.ReplaceAllString(content, "${1}schema_version: "+version)
	}

	// Update property bullet
	propPattern := regexp.MustCompile(`(?m)^schema_version::.*$`)
	if propPattern.MatchString(content) {
		return propPattern.ReplaceAllString(content, "schema_version:: "+version)
	}

	// Add if not exists (add after schema field)
	schemaPattern := regexp.MustCompile(`(?m)^(\s*schema:.*)$`)
	if schemaPattern.MatchString(content) {
		return schemaPattern.ReplaceAllString(content, "$1\n${1}schema_version: "+version)
	}

	return content
}

// CreateDefaultSchema creates a basic schema from a template file
func CreateDefaultSchema(templatePath, name string) (*Schema, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}

	schema := &Schema{
		Version:     "1.0.0",
		Name:        name,
		Description: fmt.Sprintf("Schema extracted from %s", filepath.Base(templatePath)),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Fields:      []SchemaField{},
		Sections:    []SchemaSection{},
	}

	// Extract fields from template frontmatter
	frontmatter := extractFrontmatter(string(content))
	for key, value := range frontmatter {
		field := SchemaField{
			Name:     key,
			Type:     inferFieldType(value),
			Required: !strings.Contains(value, "optional"),
		}
		schema.Fields = append(schema.Fields, field)
	}

	// Extract sections from headings
	headingPattern := regexp.MustCompile(`(?m)^(#{2,3})\s+(.+)$`)
	matches := headingPattern.FindAllStringSubmatch(string(content), -1)
	for _, match := range matches {
		section := SchemaSection{
			Name:    strings.ToLower(strings.ReplaceAll(match[2], " ", "-")),
			Heading: match[0],
		}
		schema.Sections = append(schema.Sections, section)
	}

	return schema, nil
}

func inferFieldType(value string) string {
	value = strings.ToLower(value)
	
	if strings.Contains(value, "{{date}}") || strings.Contains(value, "date") {
		return "date"
	}
	if strings.Contains(value, "[[") || strings.Contains(value, "tag") {
		return "tags"
	}
	if strings.Contains(value, "true") || strings.Contains(value, "false") {
		return "boolean"
	}
	
	return "string"
}
