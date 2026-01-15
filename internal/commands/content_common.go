package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/httrp/flip/internal/brain"
	"github.com/manifoldco/promptui"
	"gopkg.in/yaml.v2"
)

// brainConfigYAML represents the structure of .flip-brain.yaml file
type brainConfigYAML struct {
	Brain struct {
		Name    string `yaml:"name"`
		Type    string `yaml:"type"`
		Created string `yaml:"created"`
		Version string `yaml:"version"`
	} `yaml:"brain"`
	Config struct {
		DefaultOrganization string `yaml:"default_organization"`
		Author              string `yaml:"author"`
	} `yaml:"config"`
}

// getBrainAuthor reads the author from the brain config file
func getBrainAuthor(brainPath string) string {
	configPath := filepath.Join(brainPath, ".flip-brain.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "" // Return empty if config doesn't exist
	}

	var config brainConfigYAML
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "" // Return empty if parsing fails
	}

	return config.Config.Author
}

// getNotesDirectory returns the appropriate directory for notes based on brain type
func getNotesDirectory(brainPath string, brainType brain.BrainType) string {
	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: pages/ directory
		return filepath.Join(brainPath, "pages")

	case brain.BrainTypeObsidian:
		// Obsidian: root or Notes/ if it exists
		notesDir := filepath.Join(brainPath, "Notes")
		if _, err := os.Stat(notesDir); err == nil {
			return notesDir
		}
		return brainPath

	case brain.BrainTypeDendron:
		// Dendron: root directory
		return brainPath

	case brain.BrainTypeFoam:
		// Foam: notes/ directory (common convention)
		notesDir := filepath.Join(brainPath, "notes")
		if _, err := os.Stat(notesDir); err == nil {
			return notesDir
		}
		// Fallback to root if notes/ doesn't exist
		return brainPath

	case brain.BrainTypeFlip:
		// Flip: notes/ directory
		return filepath.Join(brainPath, "notes")

	default:
		// Generic: root directory
		return brainPath
	}
}

// getJournalDirectory returns the appropriate directory for journal entries
func getJournalDirectory(brainPath string, brainType brain.BrainType) string {
	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: journals/ directory (special!)
		return filepath.Join(brainPath, "journals")

	case brain.BrainTypeObsidian:
		// Obsidian: Daily Notes/ or Journal/ if it exists
		dailyNotesDir := filepath.Join(brainPath, "Daily Notes")
		if _, err := os.Stat(dailyNotesDir); err == nil {
			return dailyNotesDir
		}
		// Fallback to Journal/
		journalDir := filepath.Join(brainPath, "Journal")
		if _, err := os.Stat(journalDir); err == nil {
			return journalDir
		}
		// Create Daily Notes if doesn't exist
		return dailyNotesDir

	case brain.BrainTypeDendron:
		// Dendron: root directory
		return brainPath

	case brain.BrainTypeFoam:
		// Foam: journal/ directory (common convention)
		journalDir := filepath.Join(brainPath, "journal")
		if _, err := os.Stat(journalDir); err == nil {
			return journalDir
		}
		// Also check for Daily Notes pattern
		dailyDir := filepath.Join(brainPath, "daily")
		if _, err := os.Stat(dailyDir); err == nil {
			return dailyDir
		}
		// Create journal/ if doesn't exist
		return journalDir

	case brain.BrainTypeFlip:
		// Flip: journal/ directory
		return filepath.Join(brainPath, "journal")

	default:
		// Generic: journal/ directory
		return filepath.Join(brainPath, "journal")
	}
}

// getMeetingsDirectory returns the appropriate directory for meeting notes
func getMeetingsDirectory(brainPath string, brainType brain.BrainType) string {
	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: pages/ directory (meetings are just pages)
		return filepath.Join(brainPath, "pages")

	case brain.BrainTypeObsidian:
		// Obsidian: Meetings/ directory
		return filepath.Join(brainPath, "Meetings")

	case brain.BrainTypeDendron:
		// Dendron: root directory (uses hierarchy notation)
		return brainPath

	case brain.BrainTypeFoam:
		// Foam: meetings/ directory (common convention)
		meetingsDir := filepath.Join(brainPath, "meetings")
		if _, err := os.Stat(meetingsDir); err == nil {
			return meetingsDir
		}
		// Also check for notes/meetings pattern
		notesDir := filepath.Join(brainPath, "notes", "meetings")
		if _, err := os.Stat(notesDir); err == nil {
			return notesDir
		}
		// Create meetings/ if doesn't exist
		return meetingsDir

	case brain.BrainTypeFlip:
		// Flip: meetings/ directory
		return filepath.Join(brainPath, "meetings")

	default:
		// Generic: meetings/ directory
		return filepath.Join(brainPath, "meetings")
	}
}

// promptForSubfolder asks if user wants to use a subfolder
// Only applicable for notes and meetings, not journals
func promptForSubfolder(baseDir string, brainType brain.BrainType) (string, error) {
	// Warn about brain types that typically don't use subfolders
	if brainType == brain.BrainTypeLogseq {
		fmt.Println("⚠️  Note: Logseq typically uses a flat structure in pages/")
		fmt.Println("   Creating subfolders may affect navigation in Logseq.")
		fmt.Println()
	} else if brainType == brain.BrainTypeDendron {
		fmt.Println("⚠️  Note: Dendron uses dot notation for hierarchy (e.g., project.meeting.2025)")
		fmt.Println("   Using subfolders is not the Dendron way. Consider using hierarchical names instead.")
		fmt.Println()
	}

	// Get existing subdirectories
	var subdirs []string
	entries, err := os.ReadDir(baseDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				subdirs = append(subdirs, entry.Name())
			}
		}
	}

	// Sort subdirectories alphabetically (case-insensitive)
	if len(subdirs) > 0 {
		sortedDirs := make([]string, len(subdirs))
		copy(sortedDirs, subdirs)
		for i := 0; i < len(sortedDirs)-1; i++ {
			for j := i + 1; j < len(sortedDirs); j++ {
				if strings.ToLower(sortedDirs[i]) > strings.ToLower(sortedDirs[j]) {
					sortedDirs[i], sortedDirs[j] = sortedDirs[j], sortedDirs[i]
				}
			}
		}
		subdirs = sortedDirs
	}

	// Build menu items
	items := []string{"📂 (Notes folder - no subfolder)"}
	if len(subdirs) > 0 {
		items = append(items, subdirs...)
	}
	items = append(items, "+ Create new subfolder")

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "📁 {{ . | green }}",
	}

	prompt := promptui.Select{
		Label:     "Select target folder (existing subfolders shown alphabetically)",
		Items:     items,
		Templates: templates,
		Size:      10,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("folder selection cancelled: %w", err)
	}

	// Root directory selected
	if idx == 0 {
		return baseDir, nil
	}

	// Create new subfolder
	if idx == len(items)-1 {
		promptNew := promptui.Prompt{
			Label: "New subfolder name",
			Validate: func(input string) error {
				if strings.TrimSpace(input) == "" {
					return fmt.Errorf("folder name cannot be empty")
				}
				// Check for invalid characters
				if strings.ContainsAny(input, "/\\:*?\"<>|") {
					return fmt.Errorf("invalid characters in folder name")
				}
				return nil
			},
		}

		newFolder, err := promptNew.Run()
		if err != nil {
			return "", fmt.Errorf("folder name prompt cancelled: %w", err)
		}
		newFolder = strings.TrimSpace(newFolder)
		targetPath := filepath.Join(baseDir, newFolder)

		// Create the new folder
		if err := os.MkdirAll(targetPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create subfolder: %w", err)
		}
		fmt.Printf("✓ Created subfolder: %s\n\n", newFolder)
		return targetPath, nil
	}

	// Existing subfolder selected
	return filepath.Join(baseDir, subdirs[idx-1]), nil
}

// generateID creates a unique ID for notes (UUID format for Dendron compatibility)
func generateID() string {
	return uuid.New().String()
}

// sanitizeFilename creates a safe filename from a title
func sanitizeFilename(title string) string {
	safeName := strings.ToLower(title)

	// Replace German umlauts with ASCII equivalents
	replacements := map[string]string{
		"ä": "ae", "ö": "oe", "ü": "ue", "ß": "ss",
		"Ä": "ae", "Ö": "oe", "Ü": "ue",
	}
	for from, to := range replacements {
		safeName = strings.ReplaceAll(safeName, from, to)
	}

	// Replace spaces and special chars with hyphen, but keep structure
	safeName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == ' ' || r == '_' || r == '.' || r == ',' || r == ';' || r == ':' {
			return '-'
		}
		// Drop other special characters
		return -1
	}, safeName)

	// Collapse multiple hyphens into one
	for strings.Contains(safeName, "--") {
		safeName = strings.ReplaceAll(safeName, "--", "-")
	}

	// Trim hyphens from start/end
	safeName = strings.Trim(safeName, "-")

	return safeName
}
