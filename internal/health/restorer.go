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
	Path           string    // relative path from .orphaned/
	Category       string    // meetings, notes, tasks, etc.
	Filename       string    // just the filename
	Date           *time.Time // extracted from filename if available (e.g., 2026-03-04)
	TargetPath     string    // where it should be restored to
	JournalMatch   string    // matching journal file if date found
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

	// Determine target path (restore to original location)
	info.TargetPath = filepath.Join(info.Category, info.Filename)

	// Check if matching journal exists
	if info.Date != nil {
		journalName := info.Date.Format("2006-01-02") + ".md"
		journalPath := filepath.Join(r.brainPath, "journal", journalName)
		if _, err := os.Stat(journalPath); err == nil {
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

	// Link to journal if requested and match found
	if linkToJournal && file.JournalMatch != "" {
		linkedEntry, err := r.linkToJournal(targetPath, file.JournalMatch)
		if err != nil {
			result.Message += fmt.Sprintf(" (Warning: could not link to journal: %v)", err)
		} else {
			result.LinkedToEntry = linkedEntry
			result.Message += fmt.Sprintf(" and linked in %s", linkedEntry)
		}
	}

	return result
}

// linkToJournal adds a link to the restored file in the journal entry
func (r *Restorer) linkToJournal(restoredPath string, journalPath string) (string, error) {
	fullJournalPath := filepath.Join(r.brainPath, journalPath)

	// Read journal content
	content, err := os.ReadFile(fullJournalPath)
	if err != nil {
		return "", err
	}

	// Get relative path from brain root for the link
	relPath, err := filepath.Rel(r.brainPath, restoredPath)
	if err != nil {
		return "", err
	}

	// Extract filename without extension for wiki link
	filenameWithoutExt := strings.TrimSuffix(filepath.Base(restoredPath), filepath.Ext(restoredPath))

	// Create link based on brain type
	var linkText string
	switch r.brainType {
	case "flip":
		linkText = fmt.Sprintf("[[%s]]", filenameWithoutExt)
	case "logseq":
		linkText = fmt.Sprintf("[[%s]]", filenameWithoutExt)
	default:
		linkText = fmt.Sprintf("[%s](%s)", filenameWithoutExt, relPath)
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
