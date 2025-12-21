package commands

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/migration"
	"github.com/spf13/cobra"
)

func brainMigrateRollbackCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "rollback [target-brain-path]",
		Short: "Rollback a migration by removing migrated files",
		Long:  "Reads the migration log and removes all files that were written during migration. This is destructive and cannot be undone.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := args[0]
			return runMigrationRollback(targetPath, force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runMigrationRollback(targetPath string, force bool) error {
	// Make absolute
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if target exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("target brain not found: %s", absPath)
	}

	// Read migration log
	logPath := filepath.Join(absPath, ".flip-migration-log.json")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("no migration log found at: %s\nThis brain may not have been migrated with flip.", logPath)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		return fmt.Errorf("failed to read migration log: %w", err)
	}

	var execLog migration.ExecutionLog
	if err := json.Unmarshal(logData, &execLog); err != nil {
		return fmt.Errorf("failed to parse migration log: %w", err)
	}

	// Show summary
	fmt.Printf("\n🔄 Rollback Migration\n")
	fmt.Printf("   Brain: %s\n", absPath)
	fmt.Printf("   Migrated: %s\n", execLog.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Duration: %v\n", execLog.EndTime.Sub(execLog.StartTime))
	fmt.Printf("   Files to remove: %d\n", len(execLog.ItemsWritten))
	fmt.Printf("   Success: %v\n", execLog.Success)

	if len(execLog.Errors) > 0 {
		fmt.Printf("\n⚠️  Original migration had %d errors\n", len(execLog.Errors))
	}

	// Confirmation
	if !force {
		fmt.Printf("\n⚠️  This will DELETE all migrated files. This cannot be undone.\n")
		fmt.Printf("? Continue with rollback? (yes/no) [default: no]: ")

		var confirm string
		fmt.Scanln(&confirm)
		confirm = strings.TrimSpace(strings.ToLower(confirm))

		if confirm != "yes" && confirm != "y" {
			fmt.Println("\n✓ Cancelled")
			return nil
		}
	}

	// Execute rollback
	fmt.Println("\n🗑️  Removing migrated files...")
	removed := 0
	failed := 0
	notFound := 0

	for _, item := range execLog.ItemsWritten {
		itemPath := filepath.Join(absPath, item)

		if _, err := os.Stat(itemPath); os.IsNotExist(err) {
			notFound++
			fmt.Printf("  ⚠️  Not found: %s\n", item)
			continue
		}

		if err := os.Remove(itemPath); err != nil {
			failed++
			fmt.Printf("  ❌ Failed to remove %s: %v\n", item, err)
		} else {
			removed++
		}
	}

	// Remove empty directories
	fmt.Println("\n🧹 Cleaning empty directories...")
	cleanedDirs := removeEmptyDirs(absPath)

	// Remove migration log itself
	if err := os.Remove(logPath); err != nil {
		fmt.Printf("⚠️  Could not remove migration log: %v\n", err)
	} else {
		fmt.Println("✓ Migration log removed")
	}

	// Summary
	fmt.Printf("\n✅ Rollback complete\n")
	fmt.Printf("   Removed: %d files\n", removed)
	if cleanedDirs > 0 {
		fmt.Printf("   Cleaned: %d directories\n", cleanedDirs)
	}
	if notFound > 0 {
		fmt.Printf("   Not found: %d files (already deleted?)\n", notFound)
	}
	if failed > 0 {
		fmt.Printf("   Failed: %d files\n", failed)
		return fmt.Errorf("rollback completed with %d failures", failed)
	}

	return nil
}

// removeEmptyDirs recursively removes empty directories
func removeEmptyDirs(root string) int {
	count := 0

	// Walk bottom-up to handle nested dirs
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == root {
			return nil
		}

		// Check if directory is empty
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil
		}

		if len(entries) == 0 {
			if err := os.Remove(path); err == nil {
				count++
			}
		}

		return nil
	})

	return count
}
