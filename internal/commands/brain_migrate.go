package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/health"
	"github.com/httrp/flip/internal/migration"
	"github.com/spf13/cobra"
)

// brainMigrateCmd implements the migration dry-run planner and future execution
func brainMigrateCmd() *cobra.Command {
	var source string
	var target string
	var mode string
	var note string
	var folders []string
	var depth int
	var output string
	var execute bool
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Plan or execute migration between brains (dry-run by default)",
		Long:  "Generate a migration plan (notes, assets, conflicts) for converting one brain structure into another. By default performs a dry-run. Use --execute to apply after reviewing plan.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if source == "" || target == "" {
				return fmt.Errorf("--source and --target are required")
			}
			sourceAbs, _ := filepath.Abs(source)
			targetAbs, _ := filepath.Abs(target)

			// Detect source & target structures using health detector for type heuristics
			sourceType, err := health.DetectBrainType(sourceAbs)
			if err != nil {
				fmt.Printf("Warning: source brain type unknown (%v), proceeding with defaults\n", err)
			}
			targetType, err := health.DetectBrainType(targetAbs)
			if err != nil {
				fmt.Printf("Warning: target brain type unknown (%v), proceeding with defaults\n", err)
			}

			sourceStruct := migration.GetBrainStructure(sourceType)
			targetStruct := migration.GetBrainStructure(targetType)

			// Planner
			planner := migration.NewPlanner(sourceAbs, targetAbs, sourceStruct, targetStruct)
			mMode := migration.MigrationMode(mode)
			plan, err := planner.BuildPlan(mMode, note, folders, depth)
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(plan); err != nil {
					return err
				}
			} else {
				printPlan(plan)
			}

			if len(plan.Conflicts) > 0 && execute {
				return fmt.Errorf("cannot execute migration with conflicts present; resolve first")
			}

			if execute {
				// Confirm before execution
				fmt.Printf("\n⚠️  About to migrate %d notes and %d assets.\n", plan.NotesCount, plan.AssetsCount)
				fmt.Printf("   Source: %s (%s)\n", plan.SourceBrainPath, plan.SourceType)
				fmt.Printf("   Target: %s (%s)\n", plan.TargetBrainPath, plan.TargetType)
				if len(plan.Warnings) > 0 {
					fmt.Printf("   Warnings: %d\n", len(plan.Warnings))
				}
				fmt.Printf("\n? Continue with migration? (yes/no) [default: no]: ")

				var confirm string
				fmt.Scanln(&confirm)
				confirm = strings.TrimSpace(strings.ToLower(confirm))

				if confirm != "yes" && confirm != "y" {
					fmt.Println("\n✓ Cancelled")
					return nil
				}

				// Execute migration
				fmt.Println("\n🚀 Executing migration...")
				executor := migration.NewExecutor(plan, sourceStruct, targetStruct)
				execLog, err := executor.Execute()

				// Print summary
				fmt.Printf("\n✅ Migration complete\n")
				fmt.Printf("   Written: %d items\n", len(execLog.ItemsWritten))
				fmt.Printf("   Errors: %d\n", len(execLog.Errors))
				fmt.Printf("   Assets: %d copied\n", execLog.AssetStats.CopiedAssets)
				fmt.Printf("   Duration: %v\n", execLog.EndTime.Sub(execLog.StartTime))
				fmt.Printf("   Log: %s/.flip-migration-log.json\n", plan.TargetBrainPath)

				if len(execLog.Errors) > 0 {
					fmt.Println("\nErrors encountered:")
					for i, e := range execLog.Errors {
						if i < 10 {
							fmt.Printf("  - %s\n", e)
						} else {
							fmt.Printf("  ... and %d more (see log file)\n", len(execLog.Errors)-10)
							break
						}
					}
				}

				if err != nil {
					return fmt.Errorf("migration completed with warnings: %w", err)
				}

				// Offer to add migrated brain to workspace
				if execLog.Success && len(execLog.Errors) == 0 {
					fmt.Printf("\n? Add migrated brain to workspace? (yes/no) [default: yes]: ")
					var addToWorkspace string
					fmt.Scanln(&addToWorkspace)
					addToWorkspace = strings.TrimSpace(strings.ToLower(addToWorkspace))

					if addToWorkspace == "" || addToWorkspace == "yes" || addToWorkspace == "y" {
						brainName := filepath.Base(plan.TargetBrainPath)
						if err := runBrainAdd(plan.TargetBrainPath, brainName, false); err != nil {
							fmt.Printf("⚠️  Failed to add brain to workspace: %v\n", err)
							fmt.Printf("   You can add it manually with: flip brain add %s\n", plan.TargetBrainPath)
						} else {
							fmt.Printf("✅ Brain '%s' added to workspace\n", brainName)
						}
					}
				}
			} else {
				if output != "" {
					if err := writePlanFile(plan, output); err != nil {
						return err
					}
					fmt.Printf("Plan written to %s\n", output)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Source brain root path")
	cmd.Flags().StringVar(&target, "target", "", "Target brain root path")
	cmd.Flags().StringVar(&mode, "mode", "full", "Migration mode: single|partial|full")
	cmd.Flags().StringVar(&note, "note", "", "Single note path or title (for mode=single)")
	cmd.Flags().StringSliceVar(&folders, "folders", []string{}, "Folders to include (for mode=partial)")
	cmd.Flags().IntVar(&depth, "depth", 0, "Link expansion depth (not yet implemented)")
	cmd.Flags().BoolVar(&execute, "execute", false, "Execute migration (apply) instead of dry-run")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON plan to stdout")
	cmd.Flags().StringVar(&output, "out", "", "Write JSON plan to file")

	return cmd
}

func printPlan(plan *migration.MigrationPlan) {
	fmt.Printf("Migration Plan: %s (%s) -> %s (%s)\n", plan.SourceBrainPath, plan.SourceType, plan.TargetBrainPath, plan.TargetType)
	fmt.Printf("Mode: %s  Notes: %d  Assets: %d\n", plan.Mode, plan.NotesCount, plan.AssetsCount)
	if len(plan.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, w := range plan.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
	if len(plan.Errors) > 0 {
		fmt.Println("Errors:")
		for _, e := range plan.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}
	if len(plan.Conflicts) > 0 {
		fmt.Println("Conflicts:")
		for _, c := range plan.Conflicts {
			fmt.Printf("  - %s\n", c)
		}
	}

	maxItems := 50
	fmt.Printf("Items (showing up to %d):\n", maxItems)
	for i, it := range plan.Items {
		if i >= maxItems {
			fmt.Printf("  ... (%d more)\n", len(plan.Items)-maxItems)
			break
		}
		status := ""
		if it.Skipped {
			status = " (skipped: " + it.Reason + ")"
		}
		fmt.Printf("  [%s] %s -> %s%s\n", it.Type, it.SourcePath, it.TargetPath, status)
	}
}

func writePlanFile(plan *migration.MigrationPlan, path string) error {
	b, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	if !strings.HasSuffix(path, ".json") {
		path += ".json"
	}
	return os.WriteFile(path, b, 0o644)
}
