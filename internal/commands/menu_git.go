package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/git"
)

// handleMergeConflicts helps user resolve merge conflicts
func handleMergeConflicts(brain Brain) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("⚠️  MERGE CONFLICTS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Get list of conflicted files
	conflicts, err := git.GetConflictedFiles(brain.Path)
	if err != nil || len(conflicts) == 0 {
		fmt.Println("Unable to determine conflicted files.")
		fmt.Println("Please resolve conflicts manually.")
		return
	}

	fmt.Println("\nConflicted files:")
	markdownFiles := 0
	for _, file := range conflicts {
		isMarkdown := strings.HasSuffix(strings.ToLower(file), ".md") ||
			strings.HasSuffix(strings.ToLower(file), ".markdown")
		marker := ""
		if isMarkdown {
			marker = " 📝"
			markdownFiles++
		}
		fmt.Printf("  • %s%s\n", file, marker)
	}

	fmt.Println()

	// For markdown files, suggest merging both versions to prevent data loss
	if markdownFiles > 0 {
		fmt.Println("📝 Detected markdown/content files with conflicts.")
		fmt.Println("⚠️  To prevent data loss, we recommend merging both versions.")
		fmt.Println()
	}

	fmt.Println("How would you like to proceed?")

	options := []string{
		"Merge both versions (SAFE - keeps all content)",
		"Keep my local version only",
		"Use remote version only",
		"Abort and revert changes",
		"Skip - I'll resolve manually",
	}

	idx, err := runStringSelect("Choose action", options)
	if err != nil {
		return
	}

	switch idx {
	case 0: // Merge both
		fmt.Println("\n→ Merging both versions (safe merge)...")
		fmt.Println("   Both local and remote content will be preserved.")
		fmt.Println("   You can review and clean up the merged content later.")
		fmt.Println()
		allResolved := true
		for _, file := range conflicts {
			if err := git.ResolveConflictMergeBoth(brain.Path, file); err != nil {
				fmt.Printf("   ✗ Failed to resolve %s: %v\n", file, err)
				allResolved = false
			} else {
				fmt.Printf("   ✓ Resolved %s (merged both versions)\n", file)
			}
		}

		if allResolved {
			fmt.Println("\n→ Continuing rebase...")
			if err := git.ContinueRebase(brain.Path); err != nil {
				fmt.Printf("   ✗ Failed to continue: %v\n", err)
				fmt.Println("   Please complete the rebase manually.")
			} else {
				fmt.Println("   ✓ Rebase completed successfully!")
				fmt.Println("\n💡 Review merged files and clean up duplicate content:")
				for _, file := range conflicts {
					fmt.Printf("   - %s\n", file)
				}
			}
		}

	case 1: // Keep local
		fmt.Println("\n→ Keeping local versions...")
		fmt.Println("   ⚠️  Remote changes will be DISCARDED!")
		allResolved := true
		for _, file := range conflicts {
			if err := git.ResolveConflictUseOurs(brain.Path, file); err != nil {
				fmt.Printf("   ✗ Failed to resolve %s: %v\n", file, err)
				allResolved = false
			} else {
				fmt.Printf("   ✓ Resolved %s (kept local)\n", file)
			}
		}

		if allResolved {
			fmt.Println("\n→ Continuing rebase...")
			if err := git.ContinueRebase(brain.Path); err != nil {
				fmt.Printf("   ✗ Failed to continue: %v\n", err)
				fmt.Println("   Please complete the rebase manually.")
			} else {
				fmt.Println("   ✓ Rebase completed successfully!")
			}
		}

	case 2: // Use remote
		fmt.Println("\n→ Using remote versions...")
		fmt.Println("   ⚠️  Local changes will be DISCARDED!")
		allResolved := true
		for _, file := range conflicts {
			if err := git.ResolveConflictUseTheirs(brain.Path, file); err != nil {
				fmt.Printf("   ✗ Failed to resolve %s: %v\n", file, err)
				allResolved = false
			} else {
				fmt.Printf("   ✓ Resolved %s (used remote)\n", file)
			}
		}

		if allResolved {
			fmt.Println("\n→ Continuing rebase...")
			if err := git.ContinueRebase(brain.Path); err != nil {
				fmt.Printf("   ✗ Failed to continue: %v\n", err)
				fmt.Println("   Please complete the rebase manually.")
			} else {
				fmt.Println("   ✓ Rebase completed successfully!")
			}
		}

	case 3: // Abort
		fmt.Println("\n→ Aborting merge/rebase...")
		if err := git.AbortMerge(brain.Path); err != nil {
			fmt.Printf("   ✗ Failed to abort: %v\n", err)
		} else {
			fmt.Println("   ✓ Changes reverted successfully")
		}

	case 4: // Manual
		fmt.Println("\n→ Skipping automatic resolution")
		fmt.Printf("\n💡 To resolve manually:\n")
		fmt.Printf("   1. cd %s\n", brain.Path)
		fmt.Printf("   2. Edit conflicted files\n")
		fmt.Printf("   3. git add <resolved-files>\n")
		fmt.Printf("   4. git rebase --continue\n")
		fmt.Printf("\n   Or abort with: git rebase --abort\n")
	}

	fmt.Println()
}

// checkRemoteUpdatesOnStart checks if there are remote updates and offers to pull
func checkRemoteUpdatesOnStart() {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return
	}

	if config.ActiveWorkspace == "" {
		return
	}

	// Find active workspace
	var activeWs *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWs = &config.Workspaces[i]
			break
		}
	}

	if activeWs == nil {
		return
	}

	// Check each brain for remote updates and uncommitted changes
	type BrainWithUpdates struct {
		Brain           Brain
		HasLocalChanges bool
	}
	brainsWithUpdates := []BrainWithUpdates{}

	for _, brain := range activeWs.Brains {
		if git.IsGitRepo(brain.Path) && git.HasRemoteUpdates(brain.Path) {
			brainsWithUpdates = append(brainsWithUpdates, BrainWithUpdates{
				Brain:           brain,
				HasLocalChanges: git.HasUncommittedChanges(brain.Path),
			})
		}
	}

	// No updates, continue
	if len(brainsWithUpdates) == 0 {
		return
	}

	// Show available updates
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📥 Remote updates available:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	hasConflictRisk := false
	for _, bwu := range brainsWithUpdates {
		status := "✓"
		note := ""
		if bwu.HasLocalChanges {
			status = "⚠️"
			note = " (has uncommitted changes)"
			hasConflictRisk = true
		}
		fmt.Printf("%s %s%s\n", status, bwu.Brain.Name, note)
	}
	fmt.Println()

	if hasConflictRisk {
		fmt.Println("⚠️  Note: Some brains have uncommitted changes.")
		fmt.Println("   They will be automatically stashed and restored after pulling.")
		fmt.Println()
	}

	// Ask if user wants to pull
	idx, err := runStringSelect("Would you like to pull these updates now?", []string{"Yes, pull all", "No, pull later"})
	if err != nil || idx != 0 {
		fmt.Println()
		fmt.Println("💡 You can pull later from the menu")
		fmt.Println()
		return
	}

	// Pull each brain
	fmt.Println()
	for _, bwu := range brainsWithUpdates {
		fmt.Printf("📥 Pulling updates for '%s'...\n", bwu.Brain.Name)

		// Use autostash if there are local changes
		opts := git.PullOptions{
			Rebase:    true,
			AutoStash: bwu.HasLocalChanges,
		}

		err := git.PullWithOptions(bwu.Brain.Path, opts)
		if err != nil {
			if git.HasMergeConflicts(bwu.Brain.Path) {
				fmt.Printf("   ✗ Merge conflicts detected!\n")
				handleMergeConflicts(bwu.Brain)
			} else {
				fmt.Printf("   ✗ Failed to pull: %v\n", err)
			}
			continue
		}

		fmt.Printf("   ✓ Updated successfully\n")
	}

	fmt.Println()
	fmt.Println("✅ Pull operation completed!")
	fmt.Println()
}

// checkUncommittedChangesOnExit checks for uncommitted changes and offers to commit them
func checkUncommittedChangesOnExit() {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return
	}

	if config.ActiveWorkspace == "" {
		return
	}

	// Find active workspace
	var activeWs *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWs = &config.Workspaces[i]
			break
		}
	}

	if activeWs == nil {
		return
	}

	// Check each brain for uncommitted changes
	type BrainWithChanges struct {
		Brain   Brain
		Changes []string
	}
	brainsWithChanges := []BrainWithChanges{}

	for _, brain := range activeWs.Brains {
		if git.IsGitRepo(brain.Path) && git.HasUncommittedChanges(brain.Path) {
			changes, err := git.GetChangedFiles(brain.Path)
			if err != nil {
				changes = []string{"<unable to get file list>"}
			}
			brainsWithChanges = append(brainsWithChanges, BrainWithChanges{
				Brain:   brain,
				Changes: changes,
			})
		}
	}

	// No changes, exit cleanly
	if len(brainsWithChanges) == 0 {
		return
	}

	// Show uncommitted changes
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("⚠️  Uncommitted changes detected:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	for _, bwc := range brainsWithChanges {
		fmt.Printf("📁 %s\n", bwc.Brain.Name)
		for _, change := range bwc.Changes {
			fmt.Printf("   %s\n", change)
		}
		fmt.Println()
	}

	// Ask if user wants to commit
	idx, err := runStringSelect("Would you like to commit these changes now?", []string{"Yes, commit all", "No, commit later"})
	if err != nil || idx != 0 {
		fmt.Println()
		fmt.Println("💡 You can commit later with: flip brain git-status")
		fmt.Println()
		return
	}

	// Commit each brain
	fmt.Println()
	for _, bwc := range brainsWithChanges {
		fmt.Printf("📝 Committing changes in '%s'...\n", bwc.Brain.Name)

		// Generate smart commit message
		commitMsg := generateSmartCommitMessage(bwc.Changes)

		// Perform commit
		err := git.AddAndCommit(bwc.Brain.Path, commitMsg)
		if err != nil {
			fmt.Printf("   ✗ Failed to commit: %v\n", err)
			continue
		}

		fmt.Printf("   ✓ Committed: %s\n", commitMsg)
	}

	fmt.Println()
	fmt.Println("✅ All changes committed successfully!")
}

// generateSmartCommitMessage creates a meaningful commit message based on changed files
func generateSmartCommitMessage(changes []string) string {
	if len(changes) == 0 {
		return "Update content"
	}

	// Categorize changes by type and status
	type FileChange struct {
		Name   string
		Status string // A=added, M=modified, D=deleted, etc.
	}

	type ChangeCategory struct {
		newNotes        []FileChange
		updatedNotes    []FileChange
		newMeetings     []FileChange
		updatedMeetings []FileChange
		newJournal      []FileChange
		updatedJournal  []FileChange
		newTasks        []FileChange
		updatedTasks    []FileChange
		other           []FileChange
	}

	cat := ChangeCategory{}

	for _, change := range changes {
		var status string
		var filePath string

		// Parse git status format (e.g., "M  file.md", "?? file.md", "A  file.md")
		if len(change) > 3 && (change[1] == ' ' || change[0] == '?') {
			if change[0] == '?' {
				status = "A" // Untracked = new
			} else if change[0] != ' ' {
				status = string(change[0])
			} else if change[1] != ' ' {
				status = string(change[1])
			} else {
				status = "M"
			}
			filePath = strings.TrimSpace(change[2:])
		} else {
			status = "M"
			filePath = change
		}

		fileName := filepath.Base(filePath)
		filePathLower := strings.ToLower(filePath)

		fc := FileChange{Name: fileName, Status: status}
		isNew := (status == "A" || status == "?")

		// Categorize by path
		switch {
		case strings.Contains(filePathLower, "/notes/"):
			if isNew {
				cat.newNotes = append(cat.newNotes, fc)
			} else {
				cat.updatedNotes = append(cat.updatedNotes, fc)
			}
		case strings.Contains(filePathLower, "/meetings/"):
			if isNew {
				cat.newMeetings = append(cat.newMeetings, fc)
			} else {
				cat.updatedMeetings = append(cat.updatedMeetings, fc)
			}
		case strings.Contains(filePathLower, "/journal/"):
			if isNew {
				cat.newJournal = append(cat.newJournal, fc)
			} else {
				cat.updatedJournal = append(cat.updatedJournal, fc)
			}
		case strings.Contains(filePathLower, "/tasks/"):
			if isNew {
				cat.newTasks = append(cat.newTasks, fc)
			} else {
				cat.updatedTasks = append(cat.updatedTasks, fc)
			}
		default:
			cat.other = append(cat.other, fc)
		}
	}

	// Build structured commit message
	var sections []string

	// New Notes
	if len(cat.newNotes) > 0 {
		section := "New Notes\n========="
		for _, fc := range cat.newNotes {
			section += fmt.Sprintf("\n- %s - Added new note", fc.Name)
		}
		sections = append(sections, section)
	}

	// Updated Notes
	if len(cat.updatedNotes) > 0 {
		section := "Updated Notes\n============="
		for _, fc := range cat.updatedNotes {
			section += fmt.Sprintf("\n- %s - Updated content", fc.Name)
		}
		sections = append(sections, section)
	}

	// New Meetings
	if len(cat.newMeetings) > 0 {
		section := "New Meetings\n============"
		for _, fc := range cat.newMeetings {
			section += fmt.Sprintf("\n- %s - Added meeting notes", fc.Name)
		}
		sections = append(sections, section)
	}

	// Updated Meetings
	if len(cat.updatedMeetings) > 0 {
		section := "Updated Meetings\n================"
		for _, fc := range cat.updatedMeetings {
			section += fmt.Sprintf("\n- %s - Updated meeting notes", fc.Name)
		}
		sections = append(sections, section)
	}

	// New Journal Entries
	if len(cat.newJournal) > 0 {
		section := "New Journal Entries\n==================="
		for _, fc := range cat.newJournal {
			section += fmt.Sprintf("\n- %s - Created journal entry", fc.Name)
		}
		sections = append(sections, section)
	}

	// Updated Journal Entries
	if len(cat.updatedJournal) > 0 {
		section := "Updated Journal Entries\n======================="
		for _, fc := range cat.updatedJournal {
			section += fmt.Sprintf("\n- %s - Updated journal entry", fc.Name)
		}
		sections = append(sections, section)
	}

	// New Tasks
	if len(cat.newTasks) > 0 {
		section := "New Tasks\n========="
		for _, fc := range cat.newTasks {
			section += fmt.Sprintf("\n- %s - Added task file", fc.Name)
		}
		sections = append(sections, section)
	}

	// Updated Tasks
	if len(cat.updatedTasks) > 0 {
		section := "Updated Tasks\n============="
		for _, fc := range cat.updatedTasks {
			section += fmt.Sprintf("\n- %s - Updated tasks", fc.Name)
		}
		sections = append(sections, section)
	}

	// Other Files
	if len(cat.other) > 0 {
		section := "Other Changes\n============="
		for _, fc := range cat.other {
			action := "Updated"
			if fc.Status == "A" || fc.Status == "?" {
				action = "Added"
			} else if fc.Status == "D" {
				action = "Deleted"
			}
			section += fmt.Sprintf("\n- %s - %s", fc.Name, action)
		}
		sections = append(sections, section)
	}

	if len(sections) == 0 {
		return "Update content"
	}

	// Join sections with double newline
	return strings.Join(sections, "\n\n")
}
