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

// listFoldersInBrain lists all directories in a brain (non-recursive, level 1 only)
func listFoldersInBrain(brainPath string) []string {
	folders := make([]string, 0)
	entries, err := os.ReadDir(brainPath)
	if err != nil {
		return folders
	}

	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			folders = append(folders, entry.Name())
		}
	}
	return folders
}

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

	// First ask: workspace brains or external paths?
	scopeSelect := promptui.Select{
		Label: "Migration scope",
		Items: []string{
			"Between workspace brains (recommended)",
			"Custom paths (external brains)",
		},
		Templates: createSimpleSelectTemplates(),
		Size:      2,
		HideHelp:  true,
	}
	scopeIdx, _, err := scopeSelect.Run()
	if err != nil {
		return runManageResourcesMenu()
	}

	var sourceAbs, targetAbs string

	if scopeIdx == 0 {
		// Workspace-first approach
		workspace, err := getActiveWorkspace()
		if err != nil || len(workspace.Brains) < 2 {
			fmt.Println("\n⚠️  Need at least 2 brains in workspace for this option.")
			fmt.Println("Switching to custom paths mode...")
			fmt.Println()
			scopeIdx = 1 // Fall back to custom paths
		}

		if scopeIdx == 0 {
			// Select source brain from workspace
			brainNames := make([]string, len(workspace.Brains))
			for i, b := range workspace.Brains {
				brainNames[i] = fmt.Sprintf("%s (%s)", b.Name, b.Path)
			}

			sourceSelect := promptui.Select{
				Label:     "Source brain",
				Items:     brainNames,
				Templates: createSimpleSelectTemplates(),
				Size:      calculateMenuSize(len(brainNames)),
				HideHelp:  true,
			}
			sourceIdx, _, err := sourceSelect.Run()
			if err != nil {
				return runManageResourcesMenu()
			}
			sourceAbs = workspace.Brains[sourceIdx].Path

			// Select target brain from workspace (excluding source)
			targetBrainNames := make([]string, 0, len(workspace.Brains)-1)
			targetBrainIndices := make([]int, 0, len(workspace.Brains)-1)
			for i, b := range workspace.Brains {
				if i != sourceIdx {
					targetBrainNames = append(targetBrainNames, fmt.Sprintf("%s (%s)", b.Name, b.Path))
					targetBrainIndices = append(targetBrainIndices, i)
				}
			}

			targetSelect := promptui.Select{
				Label:     "Target brain",
				Items:     targetBrainNames,
				Templates: createSimpleSelectTemplates(),
				Size:      calculateMenuSize(len(targetBrainNames)),
				HideHelp:  true,
			}
			targetIdx, _, err := targetSelect.Run()
			if err != nil {
				return runManageResourcesMenu()
			}
			targetAbs = workspace.Brains[targetBrainIndices[targetIdx]].Path
		}
	}

	if scopeIdx == 1 {
		// Custom paths mode with intelligent suggestions
		// Show available brain suggestions
		homeDir, _ := os.UserHomeDir()
		suggestedPaths := []string{
			filepath.Join(homeDir, "Documents"),
			filepath.Join(homeDir, "notes"),
			filepath.Join(homeDir, "flap"),
			filepath.Join(homeDir, "obsidian"),
			filepath.Join(homeDir, "logseq"),
			"./",
		}
		
		fmt.Println("\n💡 Common brain locations:")
		for _, p := range suggestedPaths {
			if _, err := os.Stat(p); err == nil {
				fmt.Printf("   • %s\n", p)
			}
		}
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
		sourceAbs, _ = filepath.Abs(sourcePath)
		if _, err := os.Stat(sourceAbs); os.IsNotExist(err) {
			fmt.Printf("\n❌ Source brain not found: %s\n", sourceAbs)
			fmt.Println("Press Enter to continue...")
			fmt.Scanln()
			return runManageResourcesMenu()
		}

		// Show all known brains as suggestions for target
		config, err := loadWorkspaceConfig()
		if err == nil && len(config.Workspaces) > 0 {
			fmt.Println("\n💡 Known brains in your configuration:")
			for _, ws := range config.Workspaces {
				for _, b := range ws.Brains {
					if b.Path != sourceAbs { // Don't suggest source as target
						fmt.Printf("   • %s (%s in %s)\n", b.Name, b.Path, ws.Name)
					}
				}
			}
			fmt.Println()
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

		targetAbs, _ = filepath.Abs(targetPath)
	}

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
		// List available folders in source brain
		availableFolders := listFoldersInBrain(sourceAbs)
		if len(availableFolders) == 0 {
			fmt.Println("\n⚠️  No folders found in source brain")
			fmt.Println("Press Enter to continue...")
			fmt.Scanln()
			return runManageResourcesMenu()
		}

		fmt.Println("\n📁 Available folders in source brain:")
		for i, f := range availableFolders {
			fmt.Printf("   %d) %s\n", i+1, f)
		}
		fmt.Println()

		// Ask user to select folders (comma-separated numbers or names)
		fmt.Print("Select folders (comma-separated numbers or names): ")
		var folderInput string
		fmt.Scanln(&folderInput)
		
		// Parse input (support both numbers and names)
		for _, input := range strings.Split(folderInput, ",") {
			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}
			
			// Check if it's a number
			if num, err := fmt.Sscanf(input, "%d", new(int)); err == nil && num == 1 {
				// It's a number - get folder by index
				var idx int
				fmt.Sscanf(input, "%d", &idx)
				if idx > 0 && idx <= len(availableFolders) {
					folders = append(folders, availableFolders[idx-1])
				}
			} else {
				// It's a name - add directly
				folders = append(folders, input)
			}
		}
		
		if len(folders) == 0 {
			fmt.Println("\n❌ No folders specified")
			fmt.Println("Press Enter to continue...")
			fmt.Scanln()
			return runManageResourcesMenu()
		}
		
		fmt.Printf("\n✓ Selected folders: %s\n", strings.Join(folders, ", "))
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

	// Show preview of items (first 10)
	if len(plan.Items) > 0 {
		fmt.Println("\n📄 Preview (first 10 items):")
		previewCount := 10
		if len(plan.Items) < previewCount {
			previewCount = len(plan.Items)
		}
		for i := 0; i < previewCount; i++ {
			item := plan.Items[i]
			fmt.Printf("   %d. %s → %s (%s)\n", i+1, item.SourcePath, item.TargetPath, item.Type)
		}
		if len(plan.Items) > previewCount {
			fmt.Printf("   ... and %d more items\n", len(plan.Items)-previewCount)
		}
	}

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
