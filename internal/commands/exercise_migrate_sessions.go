package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/spf13/cobra"
)

// ExerciseMigrateSessionsCmd migrates legacy journal exercise entries to compact one-line format.
var ExerciseMigrateSessionsCmd = &cobra.Command{
	Use:   "migrate-sessions",
	Short: "Migrate legacy exercise journal entries to compact format",
	Long:  "Converts legacy exercise tracking entries in journals to compact one-line format for better readability and analytics.",
	RunE:  runExerciseMigrateSessions,
}

type exerciseMigrationResult struct {
	DryRun               bool     `json:"dry_run"`
	BrainName            string   `json:"brain_name,omitempty"`
	BrainPath            string   `json:"brain_path,omitempty"`
	FilesScanned         int      `json:"files_scanned"`
	FilesChanged         int      `json:"files_changed"`
	SessionsMigrated     int      `json:"sessions_migrated"`
	LegacyCLISessions    int      `json:"legacy_cli_sessions"`
	LegacyVSCodeSessions int      `json:"legacy_vscode_sessions"`
	AlreadyCompact       int      `json:"already_compact"`
	BackupsCreated       int      `json:"backups_created"`
	Errors               []string `json:"errors,omitempty"`
}

func init() {
	ExerciseMigrateSessionsCmd.Flags().Bool("apply", false, "Apply migration (default is dry-run)")
	ExerciseMigrateSessionsCmd.Flags().Int("days", 0, "Only migrate journal entries from the last N days (0 = all)")
	ExerciseMigrateSessionsCmd.Flags().Bool("backup", true, "Create a .bak backup file before rewriting")
}

func runExerciseMigrateSessions(cmd *cobra.Command, args []string) error {
	apply, _ := cmd.Flags().GetBool("apply")
	days, _ := cmd.Flags().GetInt("days")
	backup, _ := cmd.Flags().GetBool("backup")

	selectedBrain, err := selectBrainForOperation("Select brain for exercise session migration")
	if err != nil {
		return fmt.Errorf("selecting brain: %w", err)
	}

	detection, err := brain.NewDetector().DetectBrainType(selectedBrain.Path)
	if err != nil {
		return fmt.Errorf("detecting brain: %w", err)
	}
	if !detection.Compatible {
		return fmt.Errorf("not in a compatible brain directory")
	}

	result, err := migrateExerciseSessionsInBrain(selectedBrain.Path, detection.Type, apply, backup, days)
	if err != nil {
		return err
	}
	result.BrainName = selectedBrain.Name
	result.BrainPath = selectedBrain.Path

	mode := "DRY-RUN"
	if apply {
		mode = "APPLY"
	}

	fmt.Printf("\nExercise session migration (%s)\n", mode)
	fmt.Printf("Brain: %s\n", selectedBrain.Name)
	fmt.Printf("Files scanned: %d\n", result.FilesScanned)
	fmt.Printf("Files changed: %d\n", result.FilesChanged)
	fmt.Printf("Sessions migrated: %d\n", result.SessionsMigrated)
	fmt.Printf("Legacy CLI sessions: %d\n", result.LegacyCLISessions)
	fmt.Printf("Legacy VS Code sessions: %d\n", result.LegacyVSCodeSessions)
	fmt.Printf("Already compact: %d\n", result.AlreadyCompact)
	if apply {
		fmt.Printf("Backups created: %d\n", result.BackupsCreated)
	}
	if len(result.Errors) > 0 {
		fmt.Println("\nWarnings:")
		for _, e := range result.Errors {
			fmt.Printf("- %s\n", e)
		}
	}
	if !apply {
		fmt.Println("\nRun again with --apply to write changes.")
	}

	return nil
}

func migrateExerciseSessionsInBrain(brainPath string, brainType brain.BrainType, apply, backup bool, days int) (exerciseMigrationResult, error) {
	result := exerciseMigrationResult{DryRun: !apply}

	journalFiles, err := findJournalFilesForMigration(brainPath, brainType, days)
	if err != nil {
		return result, err
	}

	result.FilesScanned = len(journalFiles)
	parser := exercises.NewParser()

	for _, filePath := range journalFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("read %s: %v", filePath, err))
			continue
		}

		sessions, err := parser.ParseJournalExerciseBlocks(filePath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		type replacement struct {
			endLine int
			line    string
		}
		replacements := map[int]replacement{}

		for _, session := range sessions {
			if session.BlockStart < 0 || session.BlockStart >= len(lines) {
				continue
			}

			startTrimmed := strings.TrimSpace(lines[session.BlockStart])
			switch {
			case strings.HasPrefix(startTrimmed, "- exercise::"):
				result.AlreadyCompact++
				continue
			case strings.HasPrefix(startTrimmed, "- [["):
				result.AlreadyCompact++
				continue
			case strings.HasPrefix(startTrimmed, "- ["):
				result.AlreadyCompact++
				continue
			case strings.HasPrefix(startTrimmed, "## Exercise:"):
				result.LegacyCLISessions++
			case strings.HasPrefix(startTrimmed, "### 🏋️ [["):
				result.LegacyVSCodeSessions++
			default:
				continue
			}

			exerciseID := strings.TrimSpace(session.ExerciseID)
			if exerciseID == "" {
				continue
			}

			exerciseFilePath := parser.GetExerciseFilePath(brainPath, exerciseID, string(brainType))
			entry := strings.TrimSpace(buildCompactExerciseEntry(
				exerciseID,
				extractLegacyExerciseName(startTrimmed, exerciseID),
				exerciseFilePath,
				filePath,
				brainType,
				session.VariantName,
				session.Duration,
				session.Properties,
				session.Notes,
			))

			replacements[session.BlockStart] = replacement{endLine: session.BlockEnd, line: entry}
		}

		if len(replacements) == 0 {
			continue
		}

		result.FilesChanged++
		result.SessionsMigrated += len(replacements)

		if !apply {
			continue
		}

		if backup {
			backupPath, err := createMigrationBackup(filePath, content)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("backup %s: %v", filePath, err))
				continue
			}
			if backupPath != "" {
				result.BackupsCreated++
			}
		}

		var rewritten []string
		for i := 0; i < len(lines); {
			repl, ok := replacements[i]
			if !ok {
				rewritten = append(rewritten, lines[i])
				i++
				continue
			}

			rewritten = append(rewritten, repl.line)
			if repl.endLine < i {
				i++
			} else {
				i = repl.endLine + 1
			}
		}

		newContent := strings.Join(rewritten, "\n")
		if strings.HasSuffix(string(content), "\n") && !strings.HasSuffix(newContent, "\n") {
			newContent += "\n"
		}
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("write %s: %v", filePath, err))
			continue
		}
	}

	return result, nil
}

func findJournalFilesForMigration(brainPath string, brainType brain.BrainType, days int) ([]string, error) {
	journalDir := getJournalDirectory(brainPath, brainType)
	if _, err := os.Stat(journalDir); err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading journal dir: %w", err)
	}

	var threshold time.Time
	if days > 0 {
		threshold = time.Now().AddDate(0, 0, -days)
	}

	files := []string{}
	err := filepath.WalkDir(journalDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		if !isJournalFileForType(d.Name(), brainType) {
			return nil
		}
		if days > 0 {
			fileDate, ok := journalDateFromFilename(d.Name(), brainType)
			if !ok || fileDate.Before(threshold) {
				return nil
			}
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func isJournalFileForType(fileName string, brainType brain.BrainType) bool {
	base := strings.TrimSuffix(fileName, ".md")
	switch brainType {
	case brain.BrainTypeDendron:
		return strings.HasPrefix(base, "journal.")
	case brain.BrainTypeLogseq:
		parts := strings.Split(base, "_")
		return len(parts) == 3
	default:
		if strings.HasPrefix(base, "journal.") {
			return true
		}
		parts := strings.Split(base, "-")
		return len(parts) == 3
	}
}

func journalDateFromFilename(fileName string, brainType brain.BrainType) (time.Time, bool) {
	base := strings.TrimSuffix(fileName, ".md")
	base = strings.TrimPrefix(base, "journal.")
	if brainType == brain.BrainTypeLogseq {
		base = strings.ReplaceAll(base, "_", "-")
	}
	if _, err := strconv.Atoi(strings.ReplaceAll(base, "-", "")); err != nil {
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", base)
	if err != nil {
		return time.Time{}, false
	}
	return d, true
}

func createMigrationBackup(filePath string, content []byte) (string, error) {
	backupPath := filePath + ".exercise-migration.bak"
	if _, err := os.Stat(backupPath); err == nil {
		now := time.Now().Format("20060102-150405")
		backupPath = filePath + ".exercise-migration." + now + ".bak"
	}
	if err := os.WriteFile(backupPath, content, 0644); err != nil {
		return "", err
	}
	return backupPath, nil
}

func extractLegacyExerciseName(startLine, fallbackID string) string {
	if strings.HasPrefix(startLine, "## Exercise:") {
		name := strings.TrimSpace(strings.TrimPrefix(startLine, "## Exercise:"))
		if name != "" {
			return name
		}
	}
	if strings.HasPrefix(startLine, "### 🏋️ [[") {
		begin := strings.Index(startLine, "[[")
		end := strings.Index(startLine, "]]")
		if begin != -1 && end != -1 && end > begin+2 {
			name := strings.TrimSpace(startLine[begin+2 : end])
			if name != "" {
				return name
			}
		}
	}
	return fallbackID
}
