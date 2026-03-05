package migration

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Executor applies a migration plan to the file system
type Executor struct {
	plan            *MigrationPlan
	sourceStructure *BrainStructure
	targetStructure *BrainStructure
	transformer     *Transformer
	assetMigrator   *AssetMigrator
	linkRewriter    *NoteLinkRewriter

	// Execution log for rollback
	log *ExecutionLog
}

// ExecutionLog records all operations for potential rollback
type ExecutionLog struct {
	StartTime    time.Time           `json:"start_time"`
	EndTime      time.Time           `json:"end_time"`
	Success      bool                `json:"success"`
	ItemsWritten []string            `json:"items_written"`
	Errors       []string            `json:"errors"`
	AssetStats   AssetMigrationStats `json:"asset_stats"`
	Metadata     map[string]string   `json:"metadata"`
}

// NewExecutor creates an executor for a migration plan
func NewExecutor(plan *MigrationPlan, sourceStruct, targetStruct *BrainStructure) *Executor {
	assetMigrator := NewAssetMigrator(
		plan.SourceBrainPath,
		plan.TargetBrainPath,
		sourceStruct,
		targetStruct,
	)

	// Build note path map for link rewriting
	notePathMap := BuildNotePathMap(plan)
	linkRewriter := NewNoteLinkRewriter(
		plan.SourceBrainPath,
		plan.TargetBrainPath,
		sourceStruct,
		targetStruct,
		notePathMap,
	)

	return &Executor{
		plan:            plan,
		sourceStructure: sourceStruct,
		targetStructure: targetStruct,
		transformer:     NewTransformer(sourceStruct.Type, targetStruct.Type),
		assetMigrator:   assetMigrator,
		linkRewriter:    linkRewriter,
		log: &ExecutionLog{
			ItemsWritten: make([]string, 0),
			Errors:       make([]string, 0),
			Metadata:     make(map[string]string),
		},
	}
}

// Execute runs the migration, copying notes and assets, updating references
func (e *Executor) Execute() (*ExecutionLog, error) {
	return e.ExecuteWithProgress(nil)
}

// ExecuteWithProgress runs the migration with optional progress reporting
func (e *Executor) ExecuteWithProgress(progressCallback ProgressCallback) (*ExecutionLog, error) {
	e.log.StartTime = time.Now()
	e.log.Metadata["mode"] = string(e.plan.Mode)
	e.log.Metadata["source"] = e.plan.SourceBrainPath
	e.log.Metadata["target"] = e.plan.TargetBrainPath

	// Ensure target brain root exists
	if err := os.MkdirAll(e.plan.TargetBrainPath, 0o755); err != nil {
		return e.log, fmt.Errorf("failed to create target brain root: %w", err)
	}

	// Create progress bar if items to process
	totalItems := len(e.plan.Items)
	var progressBar *ProgressBar
	if progressCallback == nil && totalItems > 5 { // Only show for substantial migrations
		progressBar = NewProgressBar(totalItems)
	}

	// Process each item in the plan
	for i, item := range e.plan.Items {
		if item.Skipped {
			continue
		}

		// Update progress
		if progressBar != nil {
			progressBar.Update(item.SourcePath)
		} else if progressCallback != nil {
			progressCallback(i+1, totalItems, item.SourcePath)
		}

		var err error
		switch item.Type {
		case "note", "journal":
			err = e.migrateNote(item)
		case "definition":
			err = e.migrateDefinition(item)
		case "asset":
			err = e.migrateAsset(item)
		default:
			err = fmt.Errorf("unknown item type: %s", item.Type)
		}

		if err != nil {
			e.log.Errors = append(e.log.Errors, fmt.Sprintf("%s: %v", item.SourcePath, err))
			// Continue on error (non-fatal)
		} else {
			e.log.ItemsWritten = append(e.log.ItemsWritten, item.TargetPath)
		}
	}

	// Finish progress bar
	if progressBar != nil {
		progressBar.Finish()
	}

	// Finalize log
	e.log.EndTime = time.Now()
	e.log.Success = len(e.log.Errors) == 0
	e.log.AssetStats = e.assetMigrator.GetStats()

	// Write execution log to target brain
	if err := e.writeExecutionLog(); err != nil {
		return e.log, fmt.Errorf("warning: failed to write execution log: %w", err)
	}

	// Run post-migration health check
	e.log.Metadata["health_check"] = "running"
	if healthReport, err := e.runHealthCheck(); err != nil {
		e.log.Errors = append(e.log.Errors, fmt.Sprintf("health check: %v", err))
		e.log.Metadata["health_check"] = "failed"
	} else {
		e.log.Metadata["health_check"] = "passed"
		e.log.Metadata["health_issues"] = fmt.Sprintf("%d", len(healthReport.Issues))
	}

	return e.log, nil
}

// migrateNote copies and transforms a note file
func (e *Executor) migrateNote(item PlanItem) error {
	sourcePath := filepath.Join(e.plan.SourceBrainPath, item.SourcePath)
	targetPath := filepath.Join(e.plan.TargetBrainPath, item.TargetPath)

	// Read source note
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read note: %w", err)
	}

	// Update asset references in content
	updatedContent, err := e.assetMigrator.UpdateAssetReferences(string(content), item.SourcePath)
	if err != nil {
		// Log warning but continue (partial updates may have succeeded)
		e.log.Errors = append(e.log.Errors, fmt.Sprintf("asset refs in %s: %v", item.SourcePath, err))
	}

	// Update internal note links (wikilinks, markdown links to other notes)
	updatedContent, err = e.linkRewriter.UpdateNoteLinks(updatedContent, item.SourcePath)
	if err != nil {
		e.log.Errors = append(e.log.Errors, fmt.Sprintf("note links in %s: %v", item.SourcePath, err))
	}

	// Transform content conventions between brain types (e.g. frontmatter styles)
	updatedContent = e.transformer.TransformContent(updatedContent)

	// Clean up source-specific artifacts (Logseq video embeds, task markers, etc.)
	updatedContent = e.transformer.CleanupLogseqArtifacts(updatedContent)

	// Create target directory
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("failed to create target dir: %w", err)
	}

	// Write target note
	if err := os.WriteFile(targetPath, []byte(updatedContent), 0o644); err != nil {
		return fmt.Errorf("failed to write note: %w", err)
	}

	return nil
}

// migrateAsset copies an asset file
func (e *Executor) migrateAsset(item PlanItem) error {
	sourcePath := filepath.Join(e.plan.SourceBrainPath, item.SourcePath)
	targetPath := filepath.Join(e.plan.TargetBrainPath, item.TargetPath)

	// Create target directory
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("failed to create asset dir: %w", err)
	}

	// Open source
	src, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source asset: %w", err)
	}
	defer src.Close()

	// Create target
	dst, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target asset: %w", err)
	}
	defer dst.Close()

	// Copy
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy asset: %w", err)
	}

	return nil
}

func (e *Executor) migrateDefinition(item PlanItem) error {
	sourcePath := filepath.Join(e.plan.SourceBrainPath, item.SourcePath)
	targetPath := filepath.Join(e.plan.TargetBrainPath, item.TargetPath)

	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read definition: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("failed to create definition dir: %w", err)
	}

	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write definition: %w", err)
	}

	return nil
}

// writeExecutionLog writes the execution log to .flip-migration-log.json in target brain
func (e *Executor) writeExecutionLog() error {
	logPath := filepath.Join(e.plan.TargetBrainPath, ".flip-migration-log.json")

	data, err := json.MarshalIndent(e.log, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(logPath, data, 0o644)
}

// GetLog returns the current execution log
func (e *Executor) GetLog() *ExecutionLog {
	return e.log
}

// runHealthCheck performs a health check on the migrated brain
func (e *Executor) runHealthCheck() (*HealthCheckReport, error) {
	report := &HealthCheckReport{
		Issues: make([]string, 0),
	}

	// 1. Check if target brain directory exists
	if _, err := os.Stat(e.plan.TargetBrainPath); os.IsNotExist(err) {
		report.Issues = append(report.Issues, "Target brain directory does not exist")
		return report, nil
	}

	// 2. Check if any files were written
	if len(e.log.ItemsWritten) == 0 {
		report.Issues = append(report.Issues, "No files were written during migration")
	}

	// 3. Verify written files exist on disk
	missingFiles := 0
	for _, item := range e.log.ItemsWritten {
		itemPath := filepath.Join(e.plan.TargetBrainPath, item)
		if _, err := os.Stat(itemPath); os.IsNotExist(err) {
			missingFiles++
		}
	}
	if missingFiles > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("%d written files are missing", missingFiles))
	}

	// 4. Build an index of all migrated markdown files for link checking
	mdFiles := make(map[string]bool) // lowercase basename (no ext) → true
	for _, item := range e.log.ItemsWritten {
		if strings.HasSuffix(strings.ToLower(item), ".md") {
			base := strings.TrimSuffix(strings.ToLower(filepath.Base(item)), ".md")
			mdFiles[base] = true
		}
	}

	// 5. Scan migrated files for broken internal links and leftover BROKEN markers
	brokenLinkCount := 0
	brokenMarkerCount := 0
	for _, item := range e.log.ItemsWritten {
		if !strings.HasSuffix(strings.ToLower(item), ".md") {
			continue
		}
		itemPath := filepath.Join(e.plan.TargetBrainPath, item)
		content, err := os.ReadFile(itemPath)
		if err != nil {
			continue
		}

		text := string(content)

		// Count BROKEN markers from previous repairs
		brokenMarkerCount += strings.Count(text, "<!-- BROKEN")

		// Check wikilinks: [[target]]
		for _, m := range wikiLinkRe.FindAllStringSubmatch(text, -1) {
			if len(m) < 2 {
				continue
			}
			target := strings.ToLower(m[1])
			// Strip alias
			if idx := strings.Index(target, "|"); idx > 0 {
				target = target[:idx]
			}
			if !mdFiles[target] {
				brokenLinkCount++
			}
		}
	}

	if brokenLinkCount > 0 {
		report.Issues = append(report.Issues,
			fmt.Sprintf("%d broken internal links detected in migrated files", brokenLinkCount))
	}
	if brokenMarkerCount > 0 {
		report.Issues = append(report.Issues,
			fmt.Sprintf("%d BROKEN markers found in migrated files (from prior repairs)", brokenMarkerCount))
	}

	return report, nil
}

// wikiLinkRe matches [[target]] or [[target|alias]] (not embeds ![[...]])
var wikiLinkRe = regexp.MustCompile(`(?:^|[^!])\[\[([^\]]+)\]\]`)

// HealthCheckReport summarizes post-migration health
type HealthCheckReport struct {
	Issues []string
}
