package migration

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// MigrationMode represents the scope of migration
type MigrationMode string

const (
	ModeSingle  MigrationMode = "single"
	ModePartial MigrationMode = "partial"
	ModeFull    MigrationMode = "full"
)

// PlanItem represents a single item (note or asset) slated for migration
type PlanItem struct {
	SourcePath string
	TargetPath string
	Type       string // note|asset|definition|journal
	Skipped    bool
	Reason     string // if skipped
}

// ConflictStrategy determines how to handle items where the target already exists
type ConflictStrategy string

const (
	ConflictBlock     ConflictStrategy = "block"     // Default: block migration if conflicts exist
	ConflictSkip      ConflictStrategy = "skip"      // Skip conflicting items silently
	ConflictOverwrite ConflictStrategy = "overwrite"  // Overwrite existing target files
)

// ApplyConflictStrategy marks plan items according to the chosen strategy.
// It operates on items whose Reason is "target already exists".
func ApplyConflictStrategy(plan *MigrationPlan, strategy ConflictStrategy) {
	if strategy == ConflictBlock {
		return // default: leave conflicts as-is, execution will refuse
	}
	for i := range plan.Items {
		if plan.Items[i].Reason != "target already exists" {
			continue
		}
		switch strategy {
		case ConflictSkip:
			plan.Items[i].Skipped = true
			plan.Items[i].Reason = "skipped (target already exists)"
		case ConflictOverwrite:
			plan.Items[i].Reason = "will overwrite existing target"
		}
	}
	// Remove target-conflict entries from plan.Conflicts list so execution
	// is not blocked for the strategies that explicitly handle them.
	if strategy == ConflictSkip || strategy == ConflictOverwrite {
		filtered := plan.Conflicts[:0]
		for _, c := range plan.Conflicts {
			if !strings.Contains(c, "target already exists") {
				filtered = append(filtered, c)
			}
		}
		plan.Conflicts = filtered
	}
}

// MigrationPlan is the dry-run structure describing what will happen
type MigrationPlan struct {
	SourceBrainPath string
	TargetBrainPath string
	SourceType      string
	TargetType      string
	Mode            MigrationMode
	Depth           int
	Items           []PlanItem
	Conflicts       []string
	Warnings        []string
	Errors          []string
	NotesCount      int
	AssetsCount     int
}

// Planner creates migration plans without modifying data
type Planner struct {
	sourceStructure *BrainStructure
	targetStructure *BrainStructure
	sourceRoot      string
	targetRoot      string
	transformer     *Transformer
}

// NewPlanner constructs a Planner
func NewPlanner(sourceRoot, targetRoot string, sourceStruct, targetStruct *BrainStructure) *Planner {
	return &Planner{
		sourceStructure: sourceStruct,
		targetStructure: targetStruct,
		sourceRoot:      sourceRoot,
		targetRoot:      targetRoot,
		transformer:     NewTransformer(sourceStruct.Type, targetStruct.Type),
	}
}

// BuildPlan builds a migration plan according to mode and optional filters
func (p *Planner) BuildPlan(mode MigrationMode, singleNote string, folders []string, depth int) (*MigrationPlan, error) {
	plan := &MigrationPlan{
		SourceBrainPath: p.sourceRoot,
		TargetBrainPath: p.targetRoot,
		SourceType:      p.sourceStructure.Type.String(),
		TargetType:      p.targetStructure.Type.String(),
		Mode:            mode,
		Depth:           depth,
		Items:           make([]PlanItem, 0, 128),
		Conflicts:       make([]string, 0),
		Warnings:        make([]string, 0),
		Errors:          make([]string, 0),
	}

	switch mode {
	case ModeSingle:
		if singleNote == "" {
			return nil, fmt.Errorf("single note mode requires note path or title")
		}
		if err := p.addSingleNote(plan, singleNote, depth); err != nil {
			return nil, err
		}
	case ModePartial:
		if len(folders) == 0 {
			return nil, fmt.Errorf("partial mode requires at least one folder")
		}
		if err := p.addFolderSelection(plan, folders, depth); err != nil {
			return nil, err
		}
	case ModeFull:
		if err := p.addFullBrain(plan); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown migration mode: %s", mode)
	}

	p.finalizeCounts(plan)
	p.detectConflicts(plan)
	p.detectTargetConflicts(plan)
	return plan, nil
}

// addSingleNote adds one note and (optionally) linked notes (depth 1 for now)
func (p *Planner) addSingleNote(plan *MigrationPlan, noteRef string, depth int) error {
	// Resolve possible path variants
	// Accept either a direct path or a title (convert to slug)
	cand := noteRef
	if !strings.HasSuffix(cand, ".md") {
		slug := strings.ToLower(strings.ReplaceAll(cand, " ", "-"))
		// Try notes/, journal/, root
		candidates := []string{
			filepath.Join(p.sourceRoot, p.sourceStructure.NotesDir, slug+".md"),
			filepath.Join(p.sourceRoot, slug+".md"),
		}
		found := ""
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				found = c
				break
			}
		}
		if found == "" {
			return fmt.Errorf("note not found: %s", noteRef)
		}
		cand = found
	} else if !filepath.IsAbs(cand) {
		cand = filepath.Join(p.sourceRoot, cand)
	}

	if _, err := os.Stat(cand); err != nil {
		return fmt.Errorf("note not found: %s", cand)
	}

	rel, _ := filepath.Rel(p.sourceRoot, cand)
	plan.Items = append(plan.Items, PlanItem{
		SourcePath: rel,
		TargetPath: p.computeTargetNotePath(rel),
		Type:       "note",
	})

	// Depth expansion placeholder (future: parse links)
	if depth > 0 {
		plan.Warnings = append(plan.Warnings, "link expansion not implemented yet (depth>0 ignored)")
	}
	return nil
}

// addFolderSelection adds notes from specific folders (non-recursive for now)
func (p *Planner) addFolderSelection(plan *MigrationPlan, folders []string, depth int) error {
	for _, folder := range folders {
		abs := folder
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(p.sourceRoot, folder)
		}
		if stat, err := os.Stat(abs); err == nil && stat.IsDir() {
			entries, err := os.ReadDir(abs)
			if err != nil {
				return err
			}
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
					continue
				}
				srcRel, _ := filepath.Rel(p.sourceRoot, filepath.Join(abs, e.Name()))
				plan.Items = append(plan.Items, PlanItem{
					SourcePath: srcRel,
					TargetPath: p.computeTargetNotePath(srcRel),
					Type:       "note",
				})
			}
		} else {
			plan.Errors = append(plan.Errors, fmt.Sprintf("folder not found: %s", folder))
		}
	}
	if depth > 0 {
		plan.Warnings = append(plan.Warnings, "depth >0 expansion not implemented yet for partial mode")
	}
	return nil
}

// skippedDirs are internal/hidden directories that should never be migrated.
// Note: .orphaned IS migrated (it contains user content moved by health repair).
var skippedDirs = map[string]bool{
	".git":       true,
	".obsidian":  true,
	".logseq":    true,
	".foam":      true,
	".trash":     true,
	".vscode":    true,
	".dendron":   true,
	"logseq":     true, // Logseq config directory (no dot)
	"node_modules": true,
}

// boilerplateFiles are files commonly generated by brain tools that should be
// skipped during migration to avoid conflicts and clutter.
var boilerplateFiles = map[string]bool{
	"README.md":               true,
	"WELCOME.md":              true,
	"LICENSE.md":              true,
	"LICENSE":                 true,
	"dendron.yml":             true,
	"dendron.code-workspace":  true,
	".gitignore":              true,
}

// addFullBrain walks entire source brain and collects notes & assets
func (p *Planner) addFullBrain(plan *MigrationPlan) error {
	return filepath.WalkDir(p.sourceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if d.IsDir() {
			name := d.Name()
			// Skip known internal directories but allow .orphaned/
			if skippedDirs[name] {
				return filepath.SkipDir
			}
			// Skip other hidden dirs that aren't .orphaned
			if strings.HasPrefix(name, ".") && name != ".orphaned" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(p.sourceRoot, path)
		lower := strings.ToLower(d.Name())

		// Skip boilerplate files (mark in plan so user sees what was skipped)
		if boilerplateFiles[d.Name()] {
			plan.Items = append(plan.Items, PlanItem{
				SourcePath: rel,
				TargetPath: "",
				Type:       "note",
				Skipped:    true,
				Reason:     "boilerplate file (common to all brains)",
			})
			return nil
		}

		switch {
		case strings.HasSuffix(lower, ".md"):
			plan.Items = append(plan.Items, PlanItem{
				SourcePath: rel,
				TargetPath: p.computeTargetNotePath(rel),
				Type:       classifyNote(rel, p.sourceStructure),
			})
		case isDefinitionsYAML(rel):
			plan.Items = append(plan.Items, PlanItem{
				SourcePath: rel,
				TargetPath: rel,
				Type:       "definition",
			})
		case isAssetExt(filepath.Ext(lower)):
			plan.Items = append(plan.Items, PlanItem{
				SourcePath: rel,
				TargetPath: p.targetStructure.GetAssetTargetPath(d.Name(), ""),
				Type:       "asset",
			})
		}
		return nil
	})
}

// finalizeCounts computes counts for summary
func (p *Planner) finalizeCounts(plan *MigrationPlan) {
	notes := 0
	assets := 0
	for _, it := range plan.Items {
		if it.Type == "note" || it.Type == "journal" || it.Type == "definition" {
			notes++
		} else if it.Type == "asset" {
			assets++
		}
	}
	plan.NotesCount = notes
	plan.AssetsCount = assets
}

// detectConflicts identifies target path collisions within the plan
func (p *Planner) detectConflicts(plan *MigrationPlan) {
	seen := make(map[string]string)
	for _, it := range plan.Items {
		if it.Skipped {
			continue
		}
		if prior, exists := seen[it.TargetPath]; exists {
			plan.Conflicts = append(plan.Conflicts, fmt.Sprintf("%s conflicts with %s at %s", it.SourcePath, prior, it.TargetPath))
		} else {
			seen[it.TargetPath] = it.SourcePath
		}
	}
}

// detectTargetConflicts checks if any target paths already exist on disk
// in the target brain. These are flagged as warnings so the user can decide
// whether to --skip-conflicts or --overwrite.
func (p *Planner) detectTargetConflicts(plan *MigrationPlan) {
	for i, it := range plan.Items {
		if it.Skipped {
			continue
		}
		targetPath := filepath.Join(p.targetRoot, it.TargetPath)
		if _, err := os.Stat(targetPath); err == nil {
			plan.Conflicts = append(plan.Conflicts,
				fmt.Sprintf("target already exists: %s (from %s)", it.TargetPath, it.SourcePath))
			plan.Items[i].Reason = "target already exists"
		}
	}
}

// computeTargetNotePath maps a source relative path to target conventions
// Uses the Transformer for proper filename conversion between brain types
func (p *Planner) computeTargetNotePath(sourceRel string) string {
	dir := filepath.Dir(sourceRel)
	base := filepath.Base(sourceRel)

	// Transform the filename using brain-type aware converter
	newFilename := p.transformer.TransformFilename(base)

	// Determine target directory based on note type
	targetDir := p.mapSourceDirToTarget(dir)

	if targetDir != "" && targetDir != "." {
		return filepath.Join(targetDir, newFilename)
	}
	return newFilename
}

// mapSourceDirToTarget converts source directory to appropriate target directory
func (p *Planner) mapSourceDirToTarget(sourceDir string) string {
	// Handle journal directory mapping
	if p.sourceStructure.JournalDir != "" &&
		(sourceDir == p.sourceStructure.JournalDir || strings.HasPrefix(sourceDir, p.sourceStructure.JournalDir+"/")) {
		if p.targetStructure.JournalDir != "" {
			// Replace source journal dir with target journal dir
			return strings.Replace(sourceDir, p.sourceStructure.JournalDir, p.targetStructure.JournalDir, 1)
		}
		return sourceDir
	}

	// Handle notes directory mapping
	if p.sourceStructure.NotesDir != "" &&
		(sourceDir == p.sourceStructure.NotesDir || strings.HasPrefix(sourceDir, p.sourceStructure.NotesDir+"/")) {
		if p.targetStructure.NotesDir != "" {
			return strings.Replace(sourceDir, p.sourceStructure.NotesDir, p.targetStructure.NotesDir, 1)
		}
		return sourceDir
	}

	// Handle meetings directory mapping
	if p.sourceStructure.MeetingsDir != "" &&
		(sourceDir == p.sourceStructure.MeetingsDir || strings.HasPrefix(sourceDir, p.sourceStructure.MeetingsDir+"/")) {
		if p.targetStructure.MeetingsDir != "" {
			return strings.Replace(sourceDir, p.sourceStructure.MeetingsDir, p.targetStructure.MeetingsDir, 1)
		}
		return sourceDir
	}

	// Keep original directory if no mapping applies
	if p.targetStructure.NotesDir != "" && sourceDir == "." {
		return p.targetStructure.NotesDir
	}

	return sourceDir
}

func classifyNote(rel string, sourceStruct *BrainStructure) string {
	if sourceStruct == nil {
		return "note"
	}
	if sourceStruct.JournalDir != "" && strings.HasPrefix(rel, sourceStruct.JournalDir+"/") {
		return "journal"
	}
	return "note"
}

func isAssetExt(ext string) bool {
	ext = strings.ToLower(ext)
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".pdf", ".doc", ".docx", ".mp4", ".mov", ".webm", ".mp3", ".wav", ".ogg":
		return true
	}
	return false
}

func isDefinitionsYAML(relPath string) bool {
	clean := filepath.ToSlash(strings.ToLower(relPath))
	if !strings.HasPrefix(clean, "definitions/") {
		return false
	}
	return strings.HasSuffix(clean, ".yaml") || strings.HasSuffix(clean, ".yml")
}
