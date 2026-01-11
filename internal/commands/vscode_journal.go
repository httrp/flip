package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/spf13/cobra"
)

// MissingJournalEntry represents a file that should be in the journal
type MissingJournalEntry struct {
	Path     string `json:"path"`
	RelPath  string `json:"rel_path"`
	Title    string `json:"title"`
	Type     string `json:"type"` // "note", "meeting", "task"
	Modified string `json:"modified"`
	Date     string `json:"date"` // YYYY-MM-DD for grouping
}

// DayResult contains results for a single day
type DayResult struct {
	Date           string                `json:"date"`
	JournalPath    string                `json:"journal_path"`
	JournalExists  bool                  `json:"journal_exists"`
	MissingEntries []MissingJournalEntry `json:"missing_entries"`
	AlreadyLinked  int                   `json:"already_linked"`
	Ignored        int                   `json:"ignored"`
}

// JournalSyncResult is the response from journal sync check
type JournalSyncResult struct {
	Days          []DayResult `json:"days"`
	TotalMissing  int         `json:"total_missing"`
	TotalChecked  int         `json:"total_checked"`
	TotalLinked   int         `json:"total_linked"`
	TotalIgnored  int         `json:"total_ignored"`
	DaysChecked   int         `json:"days_checked"`
	BrainName     string      `json:"brain_name"`
	BrainPath     string      `json:"brain_path"`
}

// NewVSCodeJournalCommand creates the vscode journal subcommand
func NewVSCodeJournalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journal",
		Short: "Journal-related VS Code integration commands",
	}

	cmd.AddCommand(NewJournalSyncCheckCommand())

	return cmd
}

// NewJournalSyncCheckCommand checks for files missing from today's journal
func NewJournalSyncCheckCommand() *cobra.Command {
	var days int
	var brainName string

	cmd := &cobra.Command{
		Use:   "sync-check",
		Short: "Check for files created/modified recently that are missing from the journal",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get brain (by name or active)
			var targetBrain *Brain
			if brainName != "" {
				ws, err := getActiveWorkspace()
				if err != nil {
					OutputJSONError("journal-sync-check", err)
					return nil
				}
				for i := range ws.Brains {
					if ws.Brains[i].Name == brainName {
						targetBrain = &ws.Brains[i]
						break
					}
				}
				if targetBrain == nil {
					OutputJSONError("journal-sync-check", fmt.Errorf("brain '%s' not found", brainName))
					return nil
				}
			} else {
				var err error
				targetBrain, err = getActiveBrain()
				if err != nil {
					OutputJSONError("journal-sync-check", err)
					return nil
				}
			}

			// Detect brain type
			detector := brain.NewDetector()
			detection, err := detector.DetectBrainType(targetBrain.Path)
			if err != nil {
				OutputJSONError("journal-sync-check", fmt.Errorf("failed to detect brain type: %w", err))
				return nil
			}

			// Collect all markdown files with their mod times
			type fileInfo struct {
				path    string
				modTime time.Time
			}
			allFiles := []fileInfo{}

			dirsToScan := getContentDirectories(targetBrain.Path, detection.Type)
			for _, dir := range dirsToScan {
				if _, err := os.Stat(dir); os.IsNotExist(err) {
					continue
				}

				filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return nil
					}
					if !strings.HasSuffix(strings.ToLower(path), ".md") {
						return nil
					}
					if strings.Contains(path, "journal") {
						return nil
					}
					info, err := d.Info()
					if err != nil {
						return nil
					}
					allFiles = append(allFiles, fileInfo{path: path, modTime: info.ModTime()})
					return nil
				})
			}

			// Process each day
			dayResults := []DayResult{}
			totalMissing := 0
			totalChecked := 0
			totalLinked := 0
			totalIgnored := 0

			now := time.Now()
			for i := 0; i < days; i++ {
				checkDate := now.AddDate(0, 0, -i)
				dateStr := checkDate.Format("2006-01-02")

				journalPath := getJournalPathForDate(targetBrain.Path, detection.Type, checkDate)
				journalExists := false
				var journalContent string

				if _, err := os.Stat(journalPath); err == nil {
					journalExists = true
					content, _ := os.ReadFile(journalPath)
					journalContent = string(content)
				}

				dayMissing := []MissingJournalEntry{}
				dayLinked := 0
				dayIgnored := 0

				for _, f := range allFiles {
					if !isSameDay(f.modTime, checkDate) {
						continue
					}
					totalChecked++

					content, err := os.ReadFile(f.path)
					if err != nil {
						continue
					}
					contentStr := string(content)

					if hasNoJournalTag(contentStr) {
						dayIgnored++
						totalIgnored++
						continue
					}

					relPath, _ := filepath.Rel(targetBrain.Path, f.path)
					title := extractTitleForJournalSync(contentStr, filepath.Base(f.path))
					fileType := detectFileType(contentStr, f.path)

					if journalExists && isLinkedInJournal(journalContent, f.path, relPath, title) {
						dayLinked++
						totalLinked++
						continue
					}

					dayMissing = append(dayMissing, MissingJournalEntry{
						Path:     f.path,
						RelPath:  relPath,
						Title:    title,
						Type:     fileType,
						Modified: f.modTime.Format("15:04"),
						Date:     dateStr,
					})
					totalMissing++
				}

				// Only include days with missing entries or if it's today
				if len(dayMissing) > 0 || i == 0 {
					dayResults = append(dayResults, DayResult{
						Date:           dateStr,
						JournalPath:    journalPath,
						JournalExists:  journalExists,
						MissingEntries: dayMissing,
						AlreadyLinked:  dayLinked,
						Ignored:        dayIgnored,
					})
				}
			}

			result := JournalSyncResult{
				Days:         dayResults,
				TotalMissing: totalMissing,
				TotalChecked: totalChecked,
				TotalLinked:  totalLinked,
				TotalIgnored: totalIgnored,
				DaysChecked:  days,
				BrainName:    targetBrain.Name,
				BrainPath:    targetBrain.Path,
			}

			OutputJSONSuccess("journal-sync-check", result)
			return nil
		},
	}

	cmd.Flags().IntVar(&days, "days", 3, "Number of days to check (default: 3)")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain name to check (default: active brain)")
	cmd.Flags().Bool("json", true, "Output JSON (always true for vscode commands)")

	return cmd
}

// getJournalPathForDate returns the journal file path for a specific date
func getJournalPathForDate(brainPath string, brainType brain.BrainType, date time.Time) string {
	dateStr := date.Format("2006-01-02")
	
	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: journals/2025_01_10.md
		return filepath.Join(brainPath, "journals", date.Format("2006_01_02")+".md")
	case brain.BrainTypeObsidian:
		// Obsidian: varies, common pattern is daily/2025-01-10.md
		return filepath.Join(brainPath, "daily", dateStr+".md")
	default: // Flip brain
		return filepath.Join(brainPath, "journal", dateStr+".md")
	}
}

// getContentDirectories returns directories that contain user content
func getContentDirectories(brainPath string, brainType brain.BrainType) []string {
	switch brainType {
	case brain.BrainTypeLogseq:
		return []string{
			filepath.Join(brainPath, "pages"),
		}
	case brain.BrainTypeObsidian:
		return []string{
			brainPath, // Obsidian stores everything in root
		}
	default: // Flip brain
		return []string{
			filepath.Join(brainPath, "notes"),
			filepath.Join(brainPath, "meetings"),
			filepath.Join(brainPath, "tasks"),
		}
	}
}

// isSameDay checks if two times are on the same day
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// hasNoJournalTag checks if content has the no-journal tag
func hasNoJournalTag(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Check various tag formats
		if strings.HasPrefix(line, "- tags::") || strings.HasPrefix(line, "tags:") {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "no-journal") {
				return true
			}
		}
	}
	return false
}

// extractTitleForJournalSync extracts title from file content (local to avoid conflict)
func extractTitleForJournalSync(content, filename string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Logseq format: - title:: Something
		if strings.HasPrefix(line, "- title::") {
			return strings.TrimSpace(strings.TrimPrefix(line, "- title::"))
		}
		// YAML frontmatter: title: Something
		if strings.HasPrefix(line, "title:") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			return strings.Trim(title, "\"'")
		}
		// Markdown heading: # Something
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	// Fallback to filename without extension
	return strings.TrimSuffix(filename, ".md")
}

// detectFileType determines the type of file from content
func detectFileType(content, path string) string {
	lowerContent := strings.ToLower(content)
	lowerPath := strings.ToLower(path)

	if strings.Contains(lowerPath, "meeting") || strings.Contains(lowerContent, "type:: meeting") {
		return "meeting"
	}
	if strings.Contains(lowerPath, "task") || strings.Contains(lowerContent, "- todo ") {
		return "task"
	}
	return "note"
}

// isLinkedInJournal checks if a file is already linked in the journal
func isLinkedInJournal(journalContent, absPath, relPath, title string) bool {
	// Check for various link formats
	filename := filepath.Base(absPath)
	filenameNoExt := strings.TrimSuffix(filename, ".md")

	// Logseq link format: [[filename]]
	if strings.Contains(journalContent, "[["+filenameNoExt+"]]") {
		return true
	}
	// Full path link
	if strings.Contains(journalContent, "[["+relPath+"]]") {
		return true
	}
	// Markdown link format: [title](path)
	if strings.Contains(journalContent, "("+relPath+")") {
		return true
	}
	// Title mentioned
	if title != "" && strings.Contains(journalContent, title) {
		return true
	}

	return false
}
