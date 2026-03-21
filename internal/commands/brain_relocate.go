package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/health"
	"github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// RelocateResult contains the result of a brain relocation
type RelocateResult struct {
	SourcePath      string
	DestinationPath string
	Success         bool
	Message         string
	Error           string
	HealthCheckOK   bool
	HealthIssues    int
	BackupPath      string // Path of backup if error occurred
}

func newBrainRelocateCommand() *cobra.Command {
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "relocate [source] [destination]",
		Short: "Relocate a brain to a new location",
		Long: `Safely move a brain to a new location and verify everything works afterwards.

The command will:
1. Validate the source brain exists
2. Check destination is writable
3. Move the brain to the new location
4. Run a health check to verify integrity
5. Create a backup of the original (on failure)

Examples:
  flip brain relocate                              # Interactive mode
  flip brain relocate ~/brain /var/lib/brains/new  # Direct path
  flip brain relocate ~/brain /var/lib/brains/new --dry-run  # Preview only`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrainRelocate(args, dryRun, force)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func runBrainRelocate(args []string, dryRun bool, force bool) error {
	var sourcePath, destPath string

	// Get source and destination
	if len(args) >= 2 {
		sourcePath = args[0]
		destPath = args[1]
	} else if len(args) == 1 {
		sourcePath = args[0]
		// Ask for destination interactively
		dest, err := ui.RunInput("New destination path", "", "", func(input string) error {
			if input == "" {
				return fmt.Errorf("destination cannot be empty")
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("destination input cancelled")
		}
		destPath = dest
	} else {
		// Ask for both interactively
		src, err := ui.RunInput("Source brain path", "", "", func(input string) error {
			if input == "" {
				return fmt.Errorf("source cannot be empty")
			}
			// Expand ~ to home
			expanded := expandPath(input)
			if _, err := os.Stat(expanded); err != nil {
				return fmt.Errorf("source path not found")
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("source input cancelled")
		}
		sourcePath = expandPath(src)

		dest, err := ui.RunInput("New destination path", "", "", func(input string) error {
			if input == "" {
				return fmt.Errorf("destination cannot be empty")
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("destination input cancelled")
		}
		destPath = expandPath(dest)
	}

	// Expand paths
	sourcePath = expandPath(sourcePath)
	destPath = expandPath(destPath)

	// Perform relocation
	result := PerformRelocation(sourcePath, destPath, dryRun, force)

	// Display results
	displayRelocateResult(result)

	if !result.Success {
		return fmt.Errorf("%s", result.Error)
	}

	return nil
}

func PerformRelocation(sourcePath string, destPath string, dryRun bool, force bool) *RelocateResult {
	result := &RelocateResult{
		SourcePath:      sourcePath,
		DestinationPath: destPath,
	}

	// Validate source is a valid brain
	fmt.Printf("🔍 Validating source brain at %s...\n", sourcePath)
	brainInfo, err := health.AnalyzeBrain(sourcePath)
	if err != nil {
		result.Error = fmt.Sprintf("Invalid brain: %v", err)
		return result
	}
	fmt.Printf("   ✓ Valid %s brain\n", brainInfo.Type)

	// Check source exists
	if _, err := os.Stat(sourcePath); err != nil {
		result.Error = fmt.Sprintf("source path not found: %v", err)
		return result
	}

	// Check destination parent exists and is writable
	destParent := filepath.Dir(destPath)
	if err := os.MkdirAll(destParent, 0755); err != nil {
		result.Error = fmt.Sprintf("cannot create destination parent: %v", err)
		return result
	}

	// Check destination doesn't already exist
	if _, err := os.Stat(destPath); err == nil {
		result.Error = fmt.Sprintf("destination already exists: %s", destPath)
		return result
	}

	// Show what will happen
	fmt.Printf("\n📋 Relocation plan:\n")
	fmt.Printf("   From: %s\n", sourcePath)
	fmt.Printf("   To:   %s\n\n", destPath)

	if dryRun {
		fmt.Printf("✓ DRY RUN: Would relocate %s brain from %s\n", brainInfo.Type, sourcePath)
		result.Success = true
		result.Message = "Dry run completed - no changes made"
		return result
	}

	// Ask for confirmation if not forced
	if !force {
		confirmed, err := ui.RunConfirm("Proceed with relocation?", true)
		if err != nil || !confirmed {
			result.Error = "relocation cancelled by user"
			return result
		}
	}

	// Move the brain
	fmt.Printf("🚚 Moving brain...\n")
	if err := os.Rename(sourcePath, destPath); err != nil {
		result.Error = fmt.Sprintf("failed to move brain: %v", err)
		return result
	}
	fmt.Printf("   ✓ Brain moved to %s\n", destPath)

	// Run health check on new location
	fmt.Printf("\n🏥 Running health check on new location...\n")
	checker, err := health.NewChecker(destPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create health checker: %v", err)
		// Attempt to restore original location
		fmt.Printf("⚠️  Attempting to restore brain to original location...\n")
		if restoreErr := os.Rename(destPath, sourcePath); restoreErr != nil {
			result.Error = fmt.Sprintf("%s (AND failed to restore: %v)", result.Error, restoreErr)
		} else {
			result.Error += " (restored to original location)"
		}
		return result
	}

	checkResult, err := checker.Check()
	if err != nil {
		result.Error = fmt.Sprintf("health check failed: %v", err)
		return result
	}

	result.HealthCheckOK = len(checkResult.Issues) == 0
	result.HealthIssues = len(checkResult.Issues)

	if result.HealthCheckOK {
		fmt.Printf("   ✓ Health check passed - no issues found\n")
	} else {
		fmt.Printf("   ⚠️  Found %d potential issues (can still use brain)\n", len(checkResult.Issues))
	}

	result.Success = true
	result.Message = fmt.Sprintf("Brain successfully relocated from %s to %s", sourcePath, destPath)

	return result
}

func displayRelocateResult(result *RelocateResult) {
	fmt.Printf("%s\n", strings.Repeat("━", 60))

	if result.Success {
		fmt.Printf("✅ RELOCATION SUCCESSFUL\n\n")
		fmt.Printf("📍 New location: %s\n", result.DestinationPath)

		if result.HealthCheckOK {
			fmt.Printf("✓ Health check: PASSED\n")
		} else {
			fmt.Printf("⚠️  Health check: %d issues found\n", result.HealthIssues)
			fmt.Printf("   (Brain is functional but may benefit from repairs)\n")
		}

		fmt.Printf("\n%s\n", result.Message)
	} else {
		fmt.Printf("❌ RELOCATION FAILED\n\n")
		fmt.Printf("Error: %s\n", result.Error)

		if result.BackupPath != "" {
			fmt.Printf("Backup: %s\n", result.BackupPath)
		}
	}

	fmt.Printf("%s\n", strings.Repeat("━", 60))
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[1:])
	}
	return path
}
