package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/health"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newBrainCheckCommand() *cobra.Command {
	var fix bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check brain health and configuration",
		Long:  "Validate brain structure, detect configuration issues, and optionally repair them",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrainCheck(fix, verbose)
		},
	}

	cmd.Flags().BoolVarP(&fix, "fix", "f", false, "Automatically fix issues where possible")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information")

	// Add subcommands
	cmd.AddCommand(newBrainHealthCommand())
	cmd.AddCommand(newBrainSchemaCommand())

	return cmd
}

func newBrainHealthCommand() *cobra.Command {
	var brainPath string
	var jsonOutput bool
	var fix bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "health [path]",
		Short: "Check brain content health (broken links, missing assets, orphaned files)",
		Long: `Performs comprehensive health checks on brain content:

- Detects broken links (wikilinks and markdown links)
- Finds missing assets (images, PDFs, etc.)
- Identifies orphaned files (not linked from anywhere)
- Supports Flip, Logseq, Obsidian, Dendron, and Foam brains

Use --fix to automatically repair issues where possible:
- Broken links: Marked with strikethrough and <!-- BROKEN --> comment
- Orphaned files: Moved to .orphaned/ folder
- Format issues: Normalized (trailing whitespace, line endings)

Examples:
  flip brain check health              # Check current brain
  flip brain check health ~/my-vault   # Check specific brain
  flip brain check health --json       # Output JSON for scripting
  flip brain check health --fix        # Check and repair issues
  flip brain check health --fix --dry-run  # Preview repairs without applying`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				brainPath = args[0]
			} else {
				brainPath = "."
			}
			return runBrainHealthCheck(brainPath, jsonOutput, fix, dryRun)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	cmd.Flags().BoolVarP(&fix, "fix", "f", false, "Automatically fix issues where possible")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview repairs without applying them (use with --fix)")

	return cmd
}

type BrainIssue struct {
	BrainName string
	Severity  string // "error", "warning", "info"
	Category  string // "nested", "missing", "path", "config"
	Message   string
	Fixable   bool
	FixAction func() error
}

// healthJSONResponse is used when --json is requested
type healthJSONResponse struct {
	Result  *health.CheckResult      `json:"result"`
	Repairs *healthJSONRepairPayload `json:"repairs,omitempty"`
	Notes   map[string]any           `json:"notes,omitempty"`
}

type healthJSONRepairPayload struct {
	Results            []health.RepairResult `json:"results"`
	Stats              health.RepairStats    `json:"stats"`
	NotRepairableCount int                   `json:"not_repairable_count"`
}

func runBrainCheck(fix, verbose bool) error {
	fmt.Println("🔍 Brain Health Check")
	fmt.Println(strings.Repeat("━", 60))

	config, err := loadWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	ws, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get active workspace: %w", err)
	}

	if len(ws.Brains) == 0 {
		fmt.Println("✓ No brains to check in active workspace")
		return nil
	}

	fmt.Printf("Checking %d brain(s) in workspace '%s'...\n\n", len(ws.Brains), ws.Name)

	var issues []BrainIssue
	brainCount := 0
	okCount := 0

	for _, b := range ws.Brains {
		brainCount++
		fmt.Printf("Checking: %s\n", b.Name)

		brainIssues := checkBrain(b, ws, config, verbose)
		if len(brainIssues) == 0 {
			fmt.Printf("  %s OK - No issues found\n", IconCheck)
			okCount++
		} else {
			for _, issue := range brainIssues {
				issues = append(issues, issue)
				icon := IconWarning
				if issue.Severity == "error" {
					icon = IconError
				} else if issue.Severity == "info" {
					icon = IconInfo
				}
				fmt.Printf("  %s [%s] %s\n", icon, strings.ToUpper(issue.Severity), issue.Message)
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println(strings.Repeat("━", 60))
	fmt.Printf("Summary: %d brain(s) checked, %d OK, %d with issues\n", brainCount, okCount, brainCount-okCount)

	if len(issues) > 0 {
		errorCount := 0
		warningCount := 0
		fixableCount := 0

		for _, issue := range issues {
			if issue.Severity == "error" {
				errorCount++
			} else if issue.Severity == "warning" {
				warningCount++
			}
			if issue.Fixable {
				fixableCount++
			}
		}

		fmt.Printf("  • %d error(s)\n", errorCount)
		fmt.Printf("  • %d warning(s)\n", warningCount)
		if fixableCount > 0 {
			fmt.Printf("  • %d issue(s) can be auto-fixed\n", fixableCount)
		}

		if fix && fixableCount > 0 {
			fmt.Println("\n🔧 Attempting to fix issues...")
			fixed := 0
			for _, issue := range issues {
				if issue.Fixable && issue.FixAction != nil {
					fmt.Printf("  Fixing: %s\n", issue.Message)
					if err := issue.FixAction(); err != nil {
						fmt.Printf("    %s Failed: %v\n", IconError, err)
					} else {
						fmt.Printf("    %s Fixed\n", IconCheck)
						fixed++
					}
				}
			}
			fmt.Printf("\n✓ Fixed %d issue(s)\n", fixed)
		} else if fixableCount > 0 {
			fmt.Printf("\n💡 Run 'flip brain check --fix' to automatically repair issues\n")
		}
	} else {
		fmt.Println("✓ All brains are healthy!")
	}

	return nil
}

func checkBrain(b Brain, ws *Workspace, config *WorkspaceConfig, verbose bool) []BrainIssue {
	var issues []BrainIssue

	// 1. Check if path exists
	absPath, err := filepath.Abs(b.Path)
	if err != nil {
		issues = append(issues, BrainIssue{
			BrainName: b.Name,
			Severity:  "error",
			Category:  "path",
			Message:   fmt.Sprintf("Invalid path: %v", err),
			Fixable:   false,
		})
		return issues
	}

	stat, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		issues = append(issues, BrainIssue{
			BrainName: b.Name,
			Severity:  "error",
			Category:  "path",
			Message:   fmt.Sprintf("Path does not exist: %s", absPath),
			Fixable:   false,
		})
		return issues
	} else if err != nil {
		issues = append(issues, BrainIssue{
			BrainName: b.Name,
			Severity:  "error",
			Category:  "path",
			Message:   fmt.Sprintf("Cannot access path: %v", err),
			Fixable:   false,
		})
		return issues
	}

	if !stat.IsDir() {
		issues = append(issues, BrainIssue{
			BrainName: b.Name,
			Severity:  "error",
			Category:  "path",
			Message:   "Path is not a directory",
			Fixable:   false,
		})
		return issues
	}

	// 2. Check for brain marker file
	markerPath := filepath.Join(absPath, ".flip.yaml")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		issues = append(issues, BrainIssue{
			BrainName: b.Name,
			Severity:  "warning",
			Category:  "config",
			Message:   ".flip.yaml marker file missing",
			Fixable:   true,
			FixAction: func() error {
				// Create a basic .flip.yaml
				content := `# ==================================================================
# FLIP BRAIN CONFIGURATION
# ==================================================================
# This file was auto-generated by 'flip brain check --fix'
# Edit with caution or use flip commands for safe changes
# ==================================================================

repositories:
  - name: "primary"
    path: "."
    type: "primary"

definitions:
  auto_resolve: true
  validate_on_create: true
  suggest_missing: true

templates:
  journal: "templates/journal-template.md"
  meeting: "templates/meeting-template.md"
  note: "templates/note-template.md"

defaults:
  author: "Your Name"
  timezone: "Europe/Berlin"
  date_format: "2006-01-02"
  default_organization: "PERSONAL"
`
				return os.WriteFile(markerPath, []byte(content), 0644)
			},
		})
	}

	// 3. Check if brain is nested inside another brain
	isInside, parentPath, err := isInsideExistingBrain(absPath)
	if err == nil && isInside {
		issues = append(issues, BrainIssue{
			BrainName: b.Name,
			Severity:  "error",
			Category:  "nested",
			Message:   fmt.Sprintf("Brain is nested inside another brain at: %s", parentPath),
			Fixable:   false,
		})
	}

	// 4. Check if brain contains other brains
	containsBrains, foundBrains, err := containsExistingBrain(absPath)
	if err == nil && containsBrains {
		issues = append(issues, BrainIssue{
			BrainName: b.Name,
			Severity:  "warning",
			Category:  "nested",
			Message:   fmt.Sprintf("Brain contains nested brains: %v", foundBrains),
			Fixable:   false,
		})
	}

	// 5. Check for required directories (flip brain structure)
	if b.Type == "flip" || strings.Contains(strings.ToLower(b.Description), "flip") {
		requiredDirs := []string{"journal", "meetings", "notes", "tasks", "templates", "definitions"}
		missingDirs := []string{}

		for _, dir := range requiredDirs {
			dirPath := filepath.Join(absPath, dir)
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				missingDirs = append(missingDirs, dir)
			}
		}

		if len(missingDirs) > 0 {
			issues = append(issues, BrainIssue{
				BrainName: b.Name,
				Severity:  "warning",
				Category:  "missing",
				Message:   fmt.Sprintf("Missing directories: %v", missingDirs),
				Fixable:   true,
				FixAction: func() error {
					for _, dir := range missingDirs {
						dirPath := filepath.Join(absPath, dir)
						if err := os.MkdirAll(dirPath, 0755); err != nil {
							return err
						}
					}
					return nil
				},
			})
		}
	}

	// 6. Check for path normalization (symlinks, relative paths)
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err == nil && resolvedPath != absPath {
		issues = append(issues, BrainIssue{
			BrainName: b.Name,
			Severity:  "info",
			Category:  "path",
			Message:   fmt.Sprintf("Path is a symlink, resolves to: %s", resolvedPath),
			Fixable:   true,
			FixAction: func() error {
				// Update config with resolved path
				for i := range config.Workspaces {
					if config.Workspaces[i].Name == ws.Name {
						for j := range config.Workspaces[i].Brains {
							if config.Workspaces[i].Brains[j].Name == b.Name {
								config.Workspaces[i].Brains[j].Path = resolvedPath
								return saveWorkspaceConfig(config)
							}
						}
					}
				}
				return fmt.Errorf("brain not found in config")
			},
		})
	}

	// 7. Detect brain type if set to "unknown"
	if b.Type == "" || b.Type == "unknown" {
		detector := brain.NewDetector()
		detection, err := detector.DetectBrainType(absPath)
		if err == nil && detection.Type != brain.BrainTypeUnknown {
			issues = append(issues, BrainIssue{
				BrainName: b.Name,
				Severity:  "info",
				Category:  "config",
				Message:   fmt.Sprintf("Brain type can be detected as: %s", detection.Type),
				Fixable:   true,
				FixAction: func() error {
					// Update config with detected type
					for i := range config.Workspaces {
						if config.Workspaces[i].Name == ws.Name {
							for j := range config.Workspaces[i].Brains {
								if config.Workspaces[i].Brains[j].Name == b.Name {
									config.Workspaces[i].Brains[j].Type = string(detection.Type)
									config.Workspaces[i].Brains[j].Description = detection.Description
									return saveWorkspaceConfig(config)
								}
							}
						}
					}
					return fmt.Errorf("brain not found in config")
				},
			})
		}
	}

	return issues
}

func runBrainHealthCheck(brainPath string, jsonOutput, fix, dryRun bool) error {
	// Get absolute path
	absPath := brainPath
	if brainPath == "." {
		var err error
		absPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	} else {
		var err error
		absPath, err = filepath.Abs(brainPath)
		if err != nil {
			return fmt.Errorf("failed to resolve path: %w", err)
		}
	}

	// Create checker
	if !jsonOutput {
		fmt.Println("🔍 Analyzing brain content...")
	}
	checker, err := health.NewChecker(absPath)
	if err != nil {
		return fmt.Errorf("failed to create checker: %w", err)
	}

	// Run checks
	result, err := checker.Check()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// JSON mode (with optional repairs)
	if jsonOutput {
		resp := &healthJSONResponse{Result: result}

		if fix && len(result.Issues) > 0 {
			repairPayload, err := performRepairs(absPath, result.Issues, dryRun)
			if err != nil {
				return err
			}
			resp.Repairs = repairPayload
		}

		// Wrap in standard FlipResult format for consistency with other commands
		output := map[string]interface{}{
			"success": true,
			"command": "brain-check-health",
			"data":    resp,
		}

		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal health report: %w", err)
		}

		fmt.Println(string(data))
		return nil
	}

	reporter := health.NewReporter(result)
	reporter.Print()

	// Handle --fix mode
	if fix && len(result.Issues) > 0 {
		fmt.Println()
		fmt.Println(strings.Repeat("━", 60))

		if dryRun {
			fmt.Println("🔍 DRY RUN - Previewing repairs (no changes will be made)")
		} else {
			fmt.Println("🔧 Attempting to repair issues...")
		}
		fmt.Println()

		repairer, err := health.NewRepairer(absPath, dryRun)
		if err != nil {
			return fmt.Errorf("failed to create repairer: %w", err)
		}

		// Show what can be repaired
		repairableActions := repairer.GetRepairableIssues()
		fmt.Println("📋 Available repair actions:")
		for _, action := range repairableActions {
			fmt.Printf("   • %s: %s\n", action.IssueType, action.Description)
		}
		fmt.Println()

		// Filter to repairable issues
		repairableIssues := make([]health.Issue, 0)
		notRepairableCount := 0
		for _, issue := range result.Issues {
			if repairer.CanRepair(issue.Type) {
				repairableIssues = append(repairableIssues, issue)
			} else {
				notRepairableCount++
			}
		}

		if len(repairableIssues) == 0 {
			fmt.Println("ℹ️  No issues can be automatically repaired")
			if notRepairableCount > 0 {
				fmt.Printf("   (%d issues require manual intervention)\n", notRepairableCount)
			}
		} else {
			fmt.Printf("🔧 Repairing %d issues...\n\n", len(repairableIssues))

			repairResults := repairer.RepairIssues(repairableIssues)

			// Show results
			for _, res := range repairResults {
				icon := IconCheck
				if !res.Success {
					icon = IconError
				}

				shortFile := res.Issue.File
				if len(shortFile) > 40 {
					shortFile = "..." + shortFile[len(shortFile)-37:]
				}

				fmt.Printf("   %s %s\n", icon, shortFile)
				if res.Message != "" && res.Message != "Repaired successfully" {
					fmt.Printf("      └─ %s\n", res.Message)
				}
				if res.SkipReason != "" {
					fmt.Printf("      └─ Skipped: %s\n", res.SkipReason)
				}
			}

			// Summary
			stats := health.CalculateStats(repairResults)
			fmt.Println()
			fmt.Println(strings.Repeat("─", 40))
			if dryRun {
				fmt.Printf("📊 Would repair: %d/%d issues\n", stats.Repaired, stats.TotalIssues)
				fmt.Println("   Run without --dry-run to apply changes")
			} else {
				fmt.Printf("📊 Repaired: %d/%d issues\n", stats.Repaired, stats.TotalIssues)
				if stats.Failed > 0 {
					fmt.Printf("   ⚠️  %d repairs failed\n", stats.Failed)
				}
			}
			if notRepairableCount > 0 {
				fmt.Printf("   ℹ️  %d issues require manual intervention\n", notRepairableCount)
			}
		}
	} else if !fix && result.Stats.ErrorCount > 0 {
		// Hint to use --fix
		fmt.Println()
		fmt.Println("💡 Run 'flip brain check health --fix' to automatically repair some issues")
		fmt.Println("   Use '--fix --dry-run' to preview changes first")
	}

	// Exit with error code if there are errors (but not if we're fixing)
	if result.Stats.ErrorCount > 0 && !fix {
		return fmt.Errorf("found %d errors", result.Stats.ErrorCount)
	}

	return nil
}

// performRepairs runs repair actions without printing, returning structured data for JSON mode
func performRepairs(brainPath string, issues []health.Issue, dryRun bool) (*healthJSONRepairPayload, error) {
	repairer, err := health.NewRepairer(brainPath, dryRun)
	if err != nil {
		return nil, fmt.Errorf("failed to create repairer: %w", err)
	}

	repairableIssues := make([]health.Issue, 0)
	notRepairableCount := 0
	for _, issue := range issues {
		if repairer.CanRepair(issue.Type) {
			repairableIssues = append(repairableIssues, issue)
		} else {
			notRepairableCount++
		}
	}

	repairResults := repairer.RepairIssues(repairableIssues)
	stats := health.CalculateStats(repairResults)

	return &healthJSONRepairPayload{
		Results:            repairResults,
		Stats:              stats,
		NotRepairableCount: notRepairableCount,
	}, nil
}

func newBrainSchemaCommand() *cobra.Command {
	var brainPath string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "schema [path]",
		Short: "Validate notes against schema definitions",
		Long: `Validates note frontmatter against schema definitions.

Checks that:
- Required fields are present
- Field types are correct (string, date, array, etc.)
- Date formats are valid (YYYY-MM-DD)

Schemas are defined in definitions/schemas/*.yaml

Examples:
  flip brain check schema              # Check current brain
  flip brain check schema ~/my-brain   # Check specific brain
  flip brain check schema -v           # Verbose output`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				brainPath = args[0]
			} else {
				brainPath = "."
			}
			return runBrainSchemaCheck(brainPath, verbose)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information")

	return cmd
}

func runBrainSchemaCheck(brainPath string, verbose bool) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(brainPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Detect brain type
	detector := brain.NewDetector()
	result, err := detector.DetectBrainType(absPath)
	if err != nil {
		return fmt.Errorf("failed to detect brain type: %w", err)
	}

	if result.Type == brain.BrainTypeUnknown || result.Type == brain.BrainTypeEmpty {
		return fmt.Errorf("no valid brain found at %s", absPath)
	}

	// Check for schemas directory
	schemasDir := filepath.Join(absPath, "definitions", "schemas")
	if _, err := os.Stat(schemasDir); os.IsNotExist(err) {
		fmt.Println("🔍 Schema Check")
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println()
		fmt.Println("⚠️  No schemas defined")
		fmt.Println()
		fmt.Println("   Schema directory not found: definitions/schemas/")
		fmt.Println()
		fmt.Println("💡 To enable schema validation:")
		fmt.Println("   1. Create a new brain with 'flip brain init' (includes default schemas)")
		fmt.Println("   2. Or manually create definitions/schemas/ with schema files")
		fmt.Println()
		return nil
	}

	// Load schemas
	schemas, err := loadSchemas(schemasDir)
	if err != nil {
		return fmt.Errorf("failed to load schemas: %w", err)
	}

	if len(schemas) == 0 {
		fmt.Println("⚠️  No schema files found in definitions/schemas/")
		return nil
	}

	// Print header
	fmt.Println("🔍 Schema Validation")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("  Brain:   %s\n", filepath.Base(absPath))
	fmt.Printf("  Path:    %s\n", absPath)
	fmt.Printf("  Schemas: %d loaded\n", len(schemas))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()

	// Print loaded schemas
	if verbose {
		fmt.Println("📋 Loaded Schemas:")
		for name, schema := range schemas {
			fmt.Printf("   • %s (v%s) - %d fields\n", name, schema.Version, len(schema.Fields))
		}
		fmt.Println()
	}

	// Find and validate notes
	var totalFiles, validFiles, invalidFiles, skippedFiles int
	var issues []schemaIssue

	// Check each directory that might have notes
	noteDirs := []struct {
		dir        string
		schemaName string
	}{
		{"notes", "note"},
		{"journal", "journal"},
		{"meetings", "meeting"},
	}

	for _, nd := range noteDirs {
		dirPath := filepath.Join(absPath, nd.dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		schema, hasSchema := schemas[nd.schemaName]
		if !hasSchema {
			if verbose {
				fmt.Printf("⏭️  Skipping %s/ (no %s schema)\n", nd.dir, nd.schemaName)
			}
			continue
		}

		err := filepath.WalkDir(dirPath, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") {
				return nil
			}

			// Skip templates
			if strings.Contains(path, "template") {
				skippedFiles++
				return nil
			}

			totalFiles++
			relPath, _ := filepath.Rel(absPath, path)

			fileIssues := validateFileAgainstSchema(path, schema)
			if len(fileIssues) == 0 {
				validFiles++
				if verbose {
					fmt.Printf("✅ %s\n", relPath)
				}
			} else {
				invalidFiles++
				for _, issue := range fileIssues {
					issue.File = relPath
					issues = append(issues, issue)
				}
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	// Print results
	fmt.Println()
	fmt.Println("📊 Results")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("  Files checked:  %d\n", totalFiles)
	fmt.Printf("  Valid:          %d ✅\n", validFiles)
	fmt.Printf("  Invalid:        %d ❌\n", invalidFiles)
	fmt.Printf("  Skipped:        %d ⏭️\n", skippedFiles)
	fmt.Println()

	if len(issues) > 0 {
		fmt.Println("❌ Issues Found:")
		fmt.Println()

		currentFile := ""
		for _, issue := range issues {
			if issue.File != currentFile {
				currentFile = issue.File
				fmt.Printf("  📄 %s\n", issue.File)
			}
			fmt.Printf("     • %s: %s\n", issue.Field, issue.Message)
		}
		fmt.Println()
	}

	if invalidFiles == 0 {
		fmt.Println("✅ All notes conform to their schemas!")
	}

	return nil
}

type schemaIssue struct {
	File    string
	Field   string
	Message string
}

type schemaDefinition struct {
	Version string        `yaml:"version"`
	Name    string        `yaml:"name"`
	Fields  []schemaField `yaml:"fields"`
}

type schemaField struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Required bool   `yaml:"required"`
}

func loadSchemas(schemasDir string) (map[string]*schemaDefinition, error) {
	schemas := make(map[string]*schemaDefinition)

	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(schemasDir, entry.Name()))
		if err != nil {
			continue
		}

		var schema schemaDefinition
		if err := parseYAML(data, &schema); err != nil {
			continue
		}

		// Use filename without extension as key if name not set
		name := schema.Name
		if name == "" {
			name = strings.TrimSuffix(entry.Name(), ".yaml")
		}
		schemas[name] = &schema
	}

	return schemas, nil
}

func validateFileAgainstSchema(path string, schema *schemaDefinition) []schemaIssue {
	var issues []schemaIssue

	data, err := os.ReadFile(path)
	if err != nil {
		return []schemaIssue{{Message: fmt.Sprintf("cannot read file: %v", err)}}
	}

	content := string(data)

	// Check if file has frontmatter
	if !strings.HasPrefix(content, "---") {
		return []schemaIssue{{Field: "frontmatter", Message: "missing YAML frontmatter"}}
	}

	// Extract frontmatter
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return []schemaIssue{{Field: "frontmatter", Message: "invalid frontmatter format"}}
	}

	frontmatter := make(map[string]interface{})
	if err := parseYAML([]byte(parts[1]), &frontmatter); err != nil {
		return []schemaIssue{{Field: "frontmatter", Message: fmt.Sprintf("invalid YAML: %v", err)}}
	}

	// Check each required field
	for _, field := range schema.Fields {
		value, exists := frontmatter[field.Name]

		if field.Required && !exists {
			issues = append(issues, schemaIssue{
				Field:   field.Name,
				Message: "required field missing",
			})
			continue
		}

		if !exists {
			continue
		}

		// Type validation
		switch field.Type {
		case "date":
			if str, ok := value.(string); ok {
				if !isValidDate(str) {
					issues = append(issues, schemaIssue{
						Field:   field.Name,
						Message: fmt.Sprintf("invalid date format '%s' (expected YYYY-MM-DD)", str),
					})
				}
			}
		case "array":
			if _, ok := value.([]interface{}); !ok {
				// Also accept nil for empty arrays
				if value != nil {
					issues = append(issues, schemaIssue{
						Field:   field.Name,
						Message: "expected array",
					})
				}
			}
		}
	}

	return issues
}

func isValidDate(s string) bool {
	// Accept YYYY-MM-DD format
	if len(s) != 10 {
		return false
	}
	if s[4] != '-' || s[7] != '-' {
		return false
	}
	// Basic check - could be more thorough
	for i, c := range s {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func parseYAML(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
