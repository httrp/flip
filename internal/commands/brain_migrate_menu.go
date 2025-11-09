package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/health"
	"github.com/httrp/flip/internal/migration"
	"github.com/manifoldco/promptui"
)

// runBrainMigrationMenu shows migration wizard
func runBrainMigrationMenu() error {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔄 Brain Migration")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Migrate notes and assets between different brain types:")
	fmt.Println("  • Logseq → Flip")
	fmt.Println("  • Obsidian → Flip")
	fmt.Println("  • Dendron → Flip")
	fmt.Println("  • Or any combination")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  ✓ Smart link rewriting (wikilinks → markdown or vice versa)")
	fmt.Println("  ✓ Asset migration with proper placement")
	fmt.Println("  ✓ Note-to-note link updates")
	fmt.Println("  ✓ Dry-run planning before execution")
	fmt.Println("  ✓ Rollback support")
	fmt.Println()

	// Ask for source path
	promptSource := promptui.Prompt{
		Label:   "Source brain path",
		Default: "",
	}
	sourcePath, err := promptSource.Run()
	if err != nil {
		return runManageResourcesMenu()
	}
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		fmt.Println("\n❌ Source path required")
		fmt.Println("Press Enter to continue...")
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	// Resolve and check
	sourceAbs, _ := filepath.Abs(sourcePath)
	if _, err := os.Stat(sourceAbs); os.IsNotExist(err) {
		fmt.Printf("\n❌ Source brain not found: %s\n", sourceAbs)
		fmt.Println("Press Enter to continue...")
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	// Ask for target path
	promptTarget := promptui.Prompt{
		Label:   "Target brain path",
		Default: "",
	}
	targetPath, err := promptTarget.Run()
	if err != nil {
		return runManageResourcesMenu()
	}
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		fmt.Println("\n❌ Target path required")
		fmt.Println("Press Enter to continue...")
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	targetAbs, _ := filepath.Abs(targetPath)

	// Migration mode
	modeSelect := promptui.Select{
		Label: "Migration mode",
		Items: []string{
			"Full brain - Migrate everything",
			"Partial - Select specific folders",
			"Single note - Migrate one note only",
		},
		Templates: createSimpleSelectTemplates(),
		Size:      3,
		HideHelp:  true,
	}
	modeIdx, _, err := modeSelect.Run()
	if err != nil {
		return runManageResourcesMenu()
	}

	var mode migration.MigrationMode
	var note string
	var folders []string

	switch modeIdx {
	case 0:
		mode = migration.ModeFull
	case 1:
		mode = migration.ModePartial
		// Ask for folders
		fmt.Print("\nEnter folder names (comma-separated): ")
		var folderInput string
		fmt.Scanln(&folderInput)
		for _, f := range strings.Split(folderInput, ",") {
			folders = append(folders, strings.TrimSpace(f))
		}
		if len(folders) == 0 {
			fmt.Println("\n❌ No folders specified")
			fmt.Println("Press Enter to continue...")
			fmt.Scanln()
			return runManageResourcesMenu()
		}
	case 2:
		mode = migration.ModeSingle
		// Ask for note
		promptNote := promptui.Prompt{
			Label: "Note name or path",
		}
		note, err = promptNote.Run()
		if err != nil || note == "" {
			fmt.Println("\n❌ Note required for single mode")
			fmt.Println("Press Enter to continue...")
			fmt.Scanln()
			return runManageResourcesMenu()
		}
	}

	// Detect brain types
	fmt.Println("\n🔍 Detecting brain types...")
	sourceType, _ := health.DetectBrainType(sourceAbs)
	targetType, _ := health.DetectBrainType(targetAbs)
	fmt.Printf("   Source: %s\n", sourceType.String())
	fmt.Printf("   Target: %s\n", targetType.String())

	// Build plan
	fmt.Println("\n📋 Building migration plan...")
	sourceStruct := migration.GetBrainStructure(sourceType)
	targetStruct := migration.GetBrainStructure(targetType)
	planner := migration.NewPlanner(sourceAbs, targetAbs, sourceStruct, targetStruct)
	plan, err := planner.BuildPlan(mode, note, folders, 0)
	if err != nil {
		fmt.Printf("\n❌ Failed to build plan: %v\n", err)
		fmt.Println("Press Enter to continue...")
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	// Show plan summary
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Migration Plan Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Mode:        %s\n", plan.Mode)
	fmt.Printf("Notes:       %d\n", plan.NotesCount)
	fmt.Printf("Assets:      %d\n", plan.AssetsCount)
	fmt.Printf("Total items: %d\n", len(plan.Items))

	if len(plan.Conflicts) > 0 {
		fmt.Printf("\n⚠️  Conflicts: %d\n", len(plan.Conflicts))
		for _, c := range plan.Conflicts {
			fmt.Printf("   - %s\n", c)
		}
		fmt.Println("\n❌ Cannot proceed with conflicts. Resolve manually and try again.")
		fmt.Println("Press Enter to continue...")
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	if len(plan.Warnings) > 0 {
		fmt.Printf("\nWarnings: %d\n", len(plan.Warnings))
		for _, w := range plan.Warnings {
			fmt.Printf("   ⚠️  %s\n", w)
		}
	}

	// Ask to execute
	fmt.Println()
	executeSelect := promptui.Select{
		Label: "Execute migration now?",
		Items: []string{
			"Yes - Execute migration",
			"No - Save plan and exit",
			"Cancel",
		},
		Templates: createSimpleSelectTemplates(),
		Size:      3,
		HideHelp:  true,
	}
	execIdx, _, err := executeSelect.Run()
	if err != nil || execIdx == 2 {
		fmt.Println("\n❌ Cancelled")
		fmt.Println("Press Enter to continue...")
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	if execIdx == 1 {
		// Save plan
		planPath := filepath.Join(targetAbs, "migration-plan.json")
		if err := writePlanFile(plan, planPath); err != nil {
			fmt.Printf("\n❌ Failed to save plan: %v\n", err)
		} else {
			fmt.Printf("\n✓ Plan saved to: %s\n", planPath)
			fmt.Println("   You can review and execute later with:")
			fmt.Printf("   flip brain migrate --execute --source %s --target %s\n", sourceAbs, targetAbs)
		}
		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
		return runManageResourcesMenu()
	}

	// Execute migration
	fmt.Println("\n🚀 Executing migration...")
	executor := migration.NewExecutor(plan, sourceStruct, targetStruct)
	execLog, err := executor.Execute()

	// Show results
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Migration Results")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Status:   ")
	if execLog.Success {
		fmt.Println("✅ Success")
	} else {
		fmt.Println("⚠️  Completed with errors")
	}
	fmt.Printf("Written:  %d files\n", len(execLog.ItemsWritten))
	fmt.Printf("Assets:   %d copied\n", execLog.AssetStats.CopiedAssets)
	fmt.Printf("Duration: %v\n", execLog.EndTime.Sub(execLog.StartTime))

	if len(execLog.Errors) > 0 {
		fmt.Printf("\nErrors:   %d\n", len(execLog.Errors))
		for i, e := range execLog.Errors {
			if i < 5 {
				fmt.Printf("   - %s\n", e)
			} else {
				fmt.Printf("   ... and %d more (see log)\n", len(execLog.Errors)-5)
				break
			}
		}
	}

	fmt.Printf("\nLog file: %s/.flip-migration-log.json\n", targetAbs)

	if err != nil {
		fmt.Printf("\n⚠️  Warning: %v\n", err)
	}

	// Offer rollback
	if execLog.Success {
		fmt.Println("\n💡 To rollback this migration:")
		fmt.Printf("   flip brain migrate rollback %s\n", targetAbs)
	}

	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
	return runManageResourcesMenu()
}
