package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/httrp/flip/internal/git"
	"github.com/manifoldco/promptui"
)

// autoCommitFile attempts to auto-commit a newly created file
func autoCommitFile(brainPath, filePath, contentType string) error {
	// Check if brain is a git repo
	if !git.IsGitRepo(brainPath) {
		return nil // Silently skip if not a git repo
	}

	// Ask user if they want to commit
	shouldCommit := promptForCommit()
	if !shouldCommit {
		return nil
	}

	// Stage the file
	if err := git.AddFile(brainPath, filePath); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	// Generate commit message
	filename := filepath.Base(filePath)
	commitMsg := fmt.Sprintf("Add %s: %s", contentType, filename)

	// Create commit
	if err := git.Commit(brainPath, git.CommitOptions{
		Message: commitMsg,
		AddAll:  false, // We already added the specific file
	}); err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	fmt.Printf("\n%s Git: Committed changes\n", IconCheck)
	fmt.Printf("   Message: %s\n", commitMsg)

	return nil
}

// promptForCommit asks the user if they want to commit changes
func promptForCommit() bool {
	// Check if FLIP_AUTO_COMMIT environment variable is set
	autoCommit := os.Getenv("FLIP_AUTO_COMMIT")
	if autoCommit == "always" {
		return true
	} else if autoCommit == "never" {
		return false
	}

	// Interactive prompt
	prompt := promptui.Select{
		Label: "Commit this change to git?",
		Items: []string{"Yes", "No"},
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "▸ {{ . | cyan }}",
			Inactive: "  {{ . }}",
			Selected: "{{ . | green }}",
		},
		HideHelp: true,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return false
	}

	return idx == 0
}

// getGitCommitMessage prompts for a custom commit message
func getGitCommitMessage(defaultMsg string) (string, error) {
	prompt := promptui.Prompt{
		Label:   "Commit message",
		Default: defaultMsg,
	}

	return prompt.Run()
}
