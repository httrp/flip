package commands

import (
	"fmt"

	"github.com/httrp/flip/internal/health"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func newBrainRestoreCommand() *cobra.Command {
	var autoRestore bool

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore files from .orphaned/ folder back to their original locations",
		Long: `Restore orphaned files that were moved to the .orphaned/ folder:

The tool will:
1. Show all files currently in .orphaned/
2. Allow you to select which ones to restore
3. Move them back to their original locations
4. Optionally link them in the corresponding journal entry (if date match found)

Examples:
  flip brain restore              # Interactive selection
  flip brain restore --auto       # Auto-restore all with journal linking`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrainRestore(autoRestore)
		},
	}

	cmd.Flags().BoolVar(&autoRestore, "auto", false, "Automatically restore all orphaned files and link them to journals")

	return cmd
}

func runBrainRestore(autoRestore bool) error {
	// Get active brain
	ws, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get active workspace: %w", err)
	}

	if len(ws.Brains) == 0 {
		return fmt.Errorf("no brains found in workspace")
	}

	// Select brain
	var selectedBrain *Brain
	if len(ws.Brains) == 1 {
		selectedBrain = &ws.Brains[0]
		fmt.Printf("📍 Using brain: %s\n\n", selectedBrain.Name)
	} else {
		var err error
		selectedBrain, err = confirmOrSelectBrain(ws)
		if err != nil {
			return fmt.Errorf("brain selection failed: %w", err)
		}
	}

	// Create restorer
	restorer, err := health.NewRestorer(selectedBrain.Path)
	if err != nil {
		return fmt.Errorf("failed to create restorer: %w", err)
	}

	// List orphaned files
	fmt.Println("🔍 Scanning for orphaned files...")
	orphanedFiles, err := restorer.ListOrphanedFiles()
	if err != nil {
		return fmt.Errorf("failed to list orphaned files: %w", err)
	}

	if len(orphanedFiles) == 0 {
		fmt.Println("✅ No orphaned files found - everything is clean!")
		return nil
	}

	fmt.Printf("Found %d orphaned file(s):\n\n", len(orphanedFiles))

	// If auto mode, restore all
	if autoRestore {
		return restoreAllFiles(restorer, orphanedFiles)
	}

	// Otherwise, interactive selection
	return interactiveRestore(restorer, orphanedFiles)
}

func restoreAllFiles(restorer *health.Restorer, files []*health.OrphanedFileInfo) error {
	fmt.Println("🔄 Restoring all orphaned files with journal linking...")

	results := restorer.RestoreMultipleFiles(files, true) // link to journal automatically

	successCount := 0
	failCount := 0

	for _, result := range results {
		if result.Success {
			fmt.Printf("✅ %s\n", result.Message)
			successCount++
		} else {
			fmt.Printf("❌ %s: %s\n", result.File.Filename, result.Error)
			failCount++
		}
	}

	// Cleanup empty directories
	if err := restorer.CleanupEmptyOrphanedDirs(); err != nil {
		fmt.Printf("Warning: failed to cleanup empty .orphaned directories: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("📊 Results: %d restored, %d failed\n", successCount, failCount)

	if failCount == 0 && successCount > 0 {
		fmt.Println("✅ All files successfully restored!")
	}

	return nil
}

func interactiveRestore(restorer *health.Restorer, orphanedFiles []*health.OrphanedFileInfo) error {
	// Group by category
	byCategory := make(map[string][]*health.OrphanedFileInfo)
	for _, f := range orphanedFiles {
		byCategory[f.Category] = append(byCategory[f.Category], f)
	}

	// Show categories
	fmt.Println("📂 Files by category:")
	for category, files := range byCategory {
		journalMatches := 0
		for _, f := range files {
			if f.JournalMatch != "" {
				journalMatches++
			}
		}
		fmt.Printf("   • %s/ (%d files, %d can be linked to journal)\n", category, len(files), journalMatches)
	}

	fmt.Println()

	// Selection loop
	for len(orphanedFiles) > 0 {
		items := make([]string, len(orphanedFiles))
		for i, f := range orphanedFiles {
			journalInfo := ""
			if f.JournalMatch != "" {
				journalInfo = fmt.Sprintf(" → %s", f.JournalMatch)
			}
			items[i] = fmt.Sprintf("%s/%s%s", f.Category, f.Filename, journalInfo)
		}
		items = append(items, "✅ Done", "❌ Cancel")

		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "▸ {{ . | cyan }}",
			Inactive: "  {{ . }}",
			Selected: "📌 {{ . | green }}",
		}

		prompt := promptui.Select{
			Label:     "Choose a file to restore (or Done/Cancel)",
			Items:     items,
			Templates: templates,
			Size:      15,
		}

		idx, _, err := prompt.Run()
		if err != nil {
			return fmt.Errorf("selection cancelled: %w", err)
		}

		// Handle special items
		if idx == len(items)-2 { // "Done"
			fmt.Println()
			fmt.Println("✅ All selected files have been restored!")
			break
		}
		if idx == len(items)-1 { // "Cancel"
			fmt.Println("❌ Cancelled")
			return nil
		}

		// Restore the selected file
		selectedFile := orphanedFiles[idx]

		// Ask if want to link to journal
		linkToJournal := false
		if selectedFile.JournalMatch != "" {
			templates := &promptui.SelectTemplates{
				Label:    "{{ . }}",
				Active:   "▸ {{ . | cyan }}",
				Inactive: "  {{ . }}",
				Selected: "📌 {{ . | green }}",
			}

			linkPrompt := promptui.Select{
				Label:     "Link to corresponding journal entry?",
				Items:     []string{"Yes, link to " + selectedFile.JournalMatch, "No, just restore"},
				Templates: templates,
			}

			linkIdx, _, err := linkPrompt.Run()
			if err == nil {
				linkToJournal = linkIdx == 0
			}
		}

		// Restore
		result := restorer.RestoreSingleFile(selectedFile, linkToJournal)
		if result.Success {
			fmt.Printf("\n✅ %s\n", result.Message)
		} else {
			fmt.Printf("\n❌ Error: %s\n", result.Error)
		}

		// Remove from list
		orphanedFiles = append(orphanedFiles[:idx], orphanedFiles[idx+1:]...)

		fmt.Println()
	}

	// Cleanup empty directories
	if err := restorer.CleanupEmptyOrphanedDirs(); err != nil {
		fmt.Printf("Warning: failed to cleanup empty .orphaned directories: %v\n", err)
	}

	return nil
}
