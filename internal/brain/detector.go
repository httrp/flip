package brain

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type BrainType string

const (
	BrainTypeObsidian BrainType = "obsidian"
	BrainTypeLogseq   BrainType = "logseq"
	BrainTypeDendron  BrainType = "dendron"
	BrainTypeFlip     BrainType = "flip"
	BrainTypeEmpty    BrainType = "empty"
	BrainTypeUnknown  BrainType = "unknown"
)

type DetectionResult struct {
	Type        BrainType
	Path        string
	Description string
	Compatible  bool
	Indicators  []string
}

type Detector struct{}

func NewDetector() *Detector {
	return &Detector{}
}

// DetectBrainType analyzes a directory to determine what type of 2nd brain system it contains
func (d *Detector) DetectBrainType(path string) (*DetectionResult, error) {
	result := &DetectionResult{
		Path:       path,
		Type:       BrainTypeUnknown,
		Compatible: false,
		Indicators: []string{},
	}

	// Check if directory exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		result.Type = BrainTypeEmpty
		result.Description = "Directory does not exist"
		result.Compatible = true
		return result, nil
	}

	// Check if directory is empty
	isEmpty, err := d.isDirectoryEmpty(path)
	if err != nil {
		return nil, err
	}

	if isEmpty {
		result.Type = BrainTypeEmpty
		result.Description = "Empty directory"
		result.Compatible = true
		return result, nil
	}

	// Check for existing brain systems
	if d.isObsidianVault(path, result) {
		result.Type = BrainTypeObsidian
		result.Description = "Obsidian Vault detected"
		result.Compatible = true
		return result, nil
	}

	if d.isLogseqGraph(path, result) {
		result.Type = BrainTypeLogseq
		result.Description = "Logseq Graph detected"
		result.Compatible = true
		return result, nil
	}

	if d.isDendronWorkspace(path, result) {
		result.Type = BrainTypeDendron
		result.Description = "Dendron Workspace detected"
		result.Compatible = true
		return result, nil
	}

	if d.isFlipBrain(path, result) {
		result.Type = BrainTypeFlip
		result.Description = "Flip Brain detected"
		result.Compatible = true
		return result, nil
	}

	// Check if it's a general markdown/text directory that could be compatible
	if d.hasMarkdownFiles(path, result) {
		result.Type = BrainTypeUnknown
		result.Description = "Contains markdown files - potentially compatible"
		result.Compatible = true
		return result, nil
	}

	result.Description = "Unknown directory structure"
	result.Compatible = false
	return result, nil
}

func (d *Detector) isDirectoryEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func (d *Detector) isObsidianVault(path string, result *DetectionResult) bool {
	indicators := []string{}

	// Check for .obsidian folder
	obsidianDir := filepath.Join(path, ".obsidian")
	if _, err := os.Stat(obsidianDir); err == nil {
		indicators = append(indicators, ".obsidian/ folder found")
	}

	// Check for common Obsidian files
	commonFiles := []string{
		".obsidian/config",
		".obsidian/workspace.json",
		".obsidian/app.json",
		".obsidian/appearance.json",
	}

	for _, file := range commonFiles {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			indicators = append(indicators, file+" found")
		}
	}

	result.Indicators = append(result.Indicators, indicators...)
	return len(indicators) > 0
}

func (d *Detector) isLogseqGraph(path string, result *DetectionResult) bool {
	indicators := []string{}

	// Check for .logseq folder
	logseqDir := filepath.Join(path, ".logseq")
	if _, err := os.Stat(logseqDir); err == nil {
		indicators = append(indicators, ".logseq/ folder found")
	}

	// Check for common Logseq files
	commonFiles := []string{
		".logseq/config.edn",
		".logseq/metadata.edn",
		"logseq/config.edn",
		"pages/contents.md",
		"journals/",
	}

	for _, file := range commonFiles {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			indicators = append(indicators, file+" found")
		}
	}

	result.Indicators = append(result.Indicators, indicators...)
	return len(indicators) > 0
}

func (d *Detector) isDendronWorkspace(path string, result *DetectionResult) bool {
	indicators := []string{}

	// Check for dendron.yml
	dendronConfig := filepath.Join(path, "dendron.yml")
	if _, err := os.Stat(dendronConfig); err == nil {
		indicators = append(indicators, "dendron.yml found")
	}

	// Check for .dendron files
	dendronFiles := []string{
		".dendron.cache.json",
		".dendron.port",
		".dendron.ws",
	}

	for _, file := range dendronFiles {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			indicators = append(indicators, file+" found")
		}
	}

	// Check for typical Dendron structure
	vaultDirs, err := filepath.Glob(filepath.Join(path, "vault*"))
	if err == nil && len(vaultDirs) > 0 {
		indicators = append(indicators, "vault directories found")
	}

	result.Indicators = append(result.Indicators, indicators...)
	return len(indicators) > 0
}

func (d *Detector) isFlipBrain(path string, result *DetectionResult) bool {
	indicators := []string{}

	// Check for flip marker file
	flipMarker := filepath.Join(path, ".flip-brain.yaml")
	if _, err := os.Stat(flipMarker); err == nil {
		indicators = append(indicators, ".flip-brain.yaml found")
	}

	// Check for flip configuration
	flipConfig := filepath.Join(path, ".flip.yaml")
	if _, err := os.Stat(flipConfig); err == nil {
		indicators = append(indicators, ".flip.yaml found")
	}

	result.Indicators = append(result.Indicators, indicators...)
	return len(indicators) > 0
}

func (d *Detector) hasMarkdownFiles(path string, result *DetectionResult) bool {
	indicators := []string{}
	mdCount := 0

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if there's an error
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			mdCount++
			if mdCount <= 3 { // Only show first few as indicators
				relPath, _ := filepath.Rel(path, filePath)
				indicators = append(indicators, "markdown file: "+relPath)
			}
		}

		// Don't walk too deep to avoid performance issues
		if strings.Count(filePath, string(os.PathSeparator))-strings.Count(path, string(os.PathSeparator)) > 2 {
			return filepath.SkipDir
		}

		return nil
	})

	if err == nil && mdCount > 0 {
		if len(indicators) < mdCount {
			indicators = append(indicators, fmt.Sprintf("... and %d more markdown files", mdCount-len(indicators)))
		}
	}

	result.Indicators = append(result.Indicators, indicators...)
	return mdCount > 0
}

// GetCompatibilityInfo returns information about how flip can work with this brain type
func (d *Detector) GetCompatibilityInfo(brainType BrainType) string {
	switch brainType {
	case BrainTypeObsidian:
		return "Flip can enhance your Obsidian vault with structured templates and task management while maintaining full compatibility."
	case BrainTypeLogseq:
		return "Flip can work alongside Logseq, using compatible block references and daily notes structure."
	case BrainTypeDendron:
		return "Flip can integrate with Dendron's hierarchical note structure and schema system."
	case BrainTypeFlip:
		return "This is already a Flip-managed brain. You can enhance it with additional features."
	case BrainTypeEmpty:
		return "Perfect! Flip can create a new brain structure that's compatible with major tools."
	case BrainTypeUnknown:
		return "Flip can work with existing markdown files and enhance them with structure and templates."
	default:
		return "Compatibility assessment needed."
	}
}
