package health

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// OrphanedFileInfo contains information about an orphaned file
type OrphanedFileInfo struct {
	Path         string     // relative path from .orphaned/
	Category     string     // meetings, notes, tasks, etc.
	Filename     string     // just the filename
	Date         *time.Time // extracted from filename if available (e.g., 2026-03-04)
	CreatedDate  *time.Time // extracted from file metadata if available
	TargetPath   string     // where it should be restored to
	JournalMatch string     // matching journal file if date found
}

// RestoreResult contains the result of a restore operation
type RestoreResult struct {
	File          *OrphanedFileInfo
	Success       bool
	Message       string
	Error         string
	LinkedToEntry string // journal entry it was linked to
}

// Restorer can restore orphaned files back to their original locations
type Restorer struct {
	brainPath string
	brainType BrainType
}

// NewRestorer creates a new restorer for a brain
func NewRestorer(brainPath string) (*Restorer, error) {
	info, err := AnalyzeBrain(brainPath)
	if err != nil {
		return nil, err
	}

	return &Restorer{
		brainPath: brainPath,
		brainType: info.Type,
	}, nil
}

// ListOrphanedFiles discovers all files in .orphaned/ folder
func (r *Restorer) ListOrphanedFiles() ([]*OrphanedFileInfo, error) {
	orphanedPath := filepath.Join(r.brainPath, ".orphaned")

	// Check if .orphaned exists
	if _, err := os.Stat(orphanedPath); err != nil {
		if os.IsNotExist(err) {
			return []*OrphanedFileInfo{}, nil
		}
		return nil, err
	}

	var files []*OrphanedFileInfo

	err := filepath.WalkDir(orphanedPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only include markdown files (and potentially other formats)
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".md" && ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".pdf" {
			return nil
		}

		// Get relative path from .orphaned
		relPath, err := filepath.Rel(orphanedPath, path)
		if err != nil {
			return nil
		}

		// Parse the orphaned file info
		info := r.ParseOrphanedFile(relPath)
		files = append(files, info)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// ParseOrphanedFile extracts information from an orphaned file path
func (r *Restorer) ParseOrphanedFile(relPath string) *OrphanedFileInfo {
	info := &OrphanedFileInfo{
		Path:     relPath,
		Filename: filepath.Base(relPath),
	}

	// Determine category from first directory component
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) > 0 {
		info.Category = parts[0]
	}

	// Try to extract date from filename (YYYY-MM-DD pattern)
	datePattern := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	if match := datePattern.FindStringSubmatch(info.Filename); match != nil {
		dateStr := fmt.Sprintf("%s-%s-%s", match[1], match[2], match[3])
		if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
			info.Date = &parsed
		}
	}

	// Extract created date from file metadata if available
	srcPath := filepath.Join(r.brainPath, ".orphaned", relPath)
	if createdDate := r.extractCreatedDate(srcPath); createdDate != nil {
		info.CreatedDate = createdDate
	}

	// Determine target path (restore to original location)
	info.TargetPath = filepath.Join(info.Category, info.Filename)

	// Determine which date to use for journal linking (prefer created date if available)
	linkDate := info.Date
	if info.CreatedDate != nil {
		linkDate = info.CreatedDate
	}

	// Check if matching journal exists or will need to be created
	if linkDate != nil {
		journalName := linkDate.Format("2006-01-02") + ".md"
		journalPath := filepath.Join(r.brainPath, "journal", journalName)
		if _, err := os.Stat(journalPath); err == nil {
			info.JournalMatch = filepath.Join("journal", journalName)
		} else {
			// Journal doesn't exist yet, but we'll create it
			info.JournalMatch = filepath.Join("journal", journalName)
		}
	}

	return info
}

// RestoreSingleFile restores one orphaned file to its original location
func (r *Restorer) RestoreSingleFile(file *OrphanedFileInfo, linkToJournal bool) *RestoreResult {
	result := &RestoreResult{
		File:    file,
		Success: false,
	}

	// Source path (in .orphaned)
	srcPath := filepath.Join(r.brainPath, ".orphaned", file.Path)

	// Target path (original location)
	targetPath := filepath.Join(r.brainPath, file.TargetPath)

	// Create target directory if needed
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		result.Error = fmt.Sprintf("failed to create target directory: %v", err)
		return result
	}

	// Move the file
	if err := os.Rename(srcPath, targetPath); err != nil {
		result.Error = fmt.Sprintf("failed to move file: %v", err)
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Restored to %s", file.TargetPath)

	// Determine which date to use for journal linking
	journalDate := file.CreatedDate
	if journalDate == nil && file.Date != nil {
		journalDate = file.Date
	}

	// Only link to journal if we have a date - don't use today as fallback
	// Notes without dates are simply restored without journal linking
	if linkToJournal && journalDate != nil {
		journalName := journalDate.Format("2006-01-02") + ".md"
		journalRelPath := filepath.Join("journal", journalName)
		linkedEntry, err := r.linkToJournal(targetPath, journalRelPath)
		if err != nil {
			result.Message += fmt.Sprintf(" (Warning: could not link to journal: %v)", err)
		} else {
			result.LinkedToEntry = linkedEntry
			result.Message += fmt.Sprintf(" and linked in %s", linkedEntry)
		}
	} else if linkToJournal && journalDate == nil {
		result.Message += " (no date found, skipped journal link)"
	}

	return result
}

// linkToJournal adds a link to the restored file in the journal entry
func (r *Restorer) linkToJournal(restoredPath string, journalPath string) (string, error) {
	fullJournalPath := filepath.Join(r.brainPath, journalPath)
	journalDir := filepath.Dir(fullJournalPath)

	// Create journal directory if needed
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create journal directory: %w", err)
	}

	// Create journal file if it doesn't exist
	var content []byte
	if _, err := os.Stat(fullJournalPath); err != nil {
		if os.IsNotExist(err) {
			// Create new journal file with frontmatter and date as heading
			dateStr := strings.TrimSuffix(filepath.Base(fullJournalPath), ".md")
			frontmatter := fmt.Sprintf("---\ndate: %s\n---\n\n# %s\n\n", dateStr, dateStr)
			content = []byte(frontmatter)
		} else {
			return "", err
		}
	} else {
		// Read existing journal content
		var readErr error
		content, readErr = os.ReadFile(fullJournalPath)
		if readErr != nil {
			return "", readErr
		}
	}

	// Get relative path from journal to restored file (journalDir already defined above)
	relFromJournal, err := filepath.Rel(journalDir, restoredPath)
	if err != nil {
		return "", err
	}

	// Extract filename without extension for display text
	filenameWithoutExt := strings.TrimSuffix(filepath.Base(restoredPath), filepath.Ext(restoredPath))

	// Determine emoji based on file type
	emoji := "📝"
	if strings.Contains(restoredPath, "meeting") {
		emoji = "🤝"
	} else if strings.Contains(restoredPath, "task") {
		emoji = "📋"
	} else if strings.Contains(restoredPath, "exercise") {
		emoji = "💪"
	}

	// Create link based on brain type - flip uses markdown links, not wikilinks
	var linkText string
	switch r.brainType {
	case "logseq":
		linkText = fmt.Sprintf("[[%s]]", filenameWithoutExt)
	case "obsidian":
		linkText = fmt.Sprintf("[[%s]]", filenameWithoutExt)
	default:
		// flip and others: standard markdown link with emoji
		linkText = fmt.Sprintf("%s [%s](%s)", emoji, filenameWithoutExt, relFromJournal)
	}

	// Append link to journal (after YAML frontmatter if exists)
	journalContent := string(content)
	lines := strings.Split(journalContent, "\n")

	// Find where to insert the link (after frontmatter)
	insertIdx := 0
	frontmatterCount := 0

	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			frontmatterCount++
			if frontmatterCount == 2 {
				insertIdx = i + 1
				break
			}
		}
	}

	// Add a blank line and the link if not already there
	linkLine := fmt.Sprintf("- %s", linkText)
	if !strings.Contains(journalContent, linkText) {
		if insertIdx < len(lines) && lines[insertIdx] != "" {
			lines = append(lines[:insertIdx+1], append([]string{"", linkLine}, lines[insertIdx+1:]...)...)
		} else if insertIdx < len(lines) {
			lines = append(lines[:insertIdx], append([]string{linkLine}, lines[insertIdx:]...)...)
		} else {
			lines = append(lines, "", linkLine)
		}

		// Write back
		newContent := strings.Join(lines, "\n")
		if err := os.WriteFile(fullJournalPath, []byte(newContent), 0644); err != nil {
			return "", err
		}
	}

	return journalPath, nil
}

// extractCreatedDate tries to extract the created date from file metadata
func (r *Restorer) extractCreatedDate(filePath string) *time.Time {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	fileContent := string(content)

	// Try to extract from YAML frontmatter
	if strings.HasPrefix(fileContent, "---") {
		endIdx := strings.Index(fileContent[3:], "---")
		if endIdx > 0 {
			frontmatter := fileContent[3 : 3+endIdx]

			// Look for created or date field
			for _, line := range strings.Split(frontmatter, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "created:") || strings.HasPrefix(line, "date:") {
					dateStr := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
					// Remove quotes if present
					dateStr = strings.Trim(dateStr, "\"'")
					// Parse date (try YYYY-MM-DD format)
					if t, err := time.Parse("2006-01-02", dateStr); err == nil {
						return &t
					}
					// Try other date formats
					formats := []string{
						"2006-01-02T15:04:05Z07:00",
						"2006-01-02 15:04:05",
						"2006-01-02",
					}
					for _, format := range formats {
						if t, err := time.Parse(format, dateStr); err == nil {
							return &t
						}
					}
				}
			}
		}
	}

	// Try to extract from Logseq properties format (property:: value)
	if strings.Contains(fileContent, "created::") {
		for _, line := range strings.Split(fileContent, "\n") {
			if strings.HasPrefix(line, "created::") {
				dateStr := strings.TrimPrefix(line, "created::")
				dateStr = strings.TrimSpace(dateStr)
				// Parse Logseq timestamp format (usually milliseconds)
				// Convert to YYYY-MM-DD if possible
				if t, err := time.Parse("2006-01-02", dateStr); err == nil {
					return &t
				}
			}
		}
	}

	return nil
}

// RestoreMultipleFiles restores multiple orphaned files
func (r *Restorer) RestoreMultipleFiles(files []*OrphanedFileInfo, linkToJournal bool) []*RestoreResult {
	results := make([]*RestoreResult, 0, len(files))

	for _, file := range files {
		result := r.RestoreSingleFile(file, linkToJournal)
		results = append(results, result)
	}

	return results
}

// CleanupEmptyOrphanedDirs removes empty directories from .orphaned folder
func (r *Restorer) CleanupEmptyOrphanedDirs() error {
	orphanedPath := filepath.Join(r.brainPath, ".orphaned")

	// Check if .orphaned exists
	if _, err := os.Stat(orphanedPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Walk directory and remove empty ones from bottom up
	return filepath.WalkDir(orphanedPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() || path == orphanedPath {
			return nil
		}

		// Try to remove empty directory
		_ = os.Remove(path)

		return nil
	})
}
