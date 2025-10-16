package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func NewNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create something new (brain/workspace)",
		Long:  "Interactive menu to create a new brain or workspace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewMenu(args)
		},
	}
	// Alias: flip new brain
	brainCmd := &cobra.Command{
		Use:   "brain",
		Short: "Create a new brain (alias)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewBrain()
		},
	}
	cmd.AddCommand(brainCmd)
	return cmd
}

// Interactive menu for 'flip new'
func runNewMenu(args []string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("+ What do you want to create?")
	fmt.Println("  1) Brain")
	fmt.Println("  2) Workspace")
	fmt.Printf("Choose (1/2) [default: 1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" || choice == "1" {
		return runNewBrain()
	} else if choice == "2" {
		return runNewWorkspace()
	}
	fmt.Println("[X] Cancelled.")
	return nil
}

// Workspace creation logic (same UX as brain)
func runNewWorkspace() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("+ Create a new workspace")

	// Creative name selection
	creativeNames := []string{
		"atlas", "odyssey", "aurora", "echo", "zenith", "soliloquy", "muse", "oracle", "serendipity", "epiphany", "noesis", "elysium", "satori", "logos", "cosmos", "paradox", "quasar", "zeitgeist", "sophia", "mythos",
	}
	fmt.Println("? Choose a name for your new brain:")
	for i, n := range creativeNames {
		fmt.Printf("  %d) %s\n", i+1, n)
	}
	fmt.Printf("Or enter your own name [default: %s]: ", creativeNames[0])
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = creativeNames[0]
	}
	// Prevent 'flap' as brain name
	if strings.ToLower(name) == "flap" {
		fmt.Println("[X] 'flap' is reserved for the main folder. Please choose another name.")
		return nil
	}
	// Default path: $HOME/flap/workspaces/<name>, but allow any custom path
	home, _ := os.UserHomeDir()
	recommended := filepath.Join(home, "flap", "workspaces", name)
	fmt.Printf("? Path for your workspace [default: %s]: ", recommended)
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)
	if path == "" {
		path = recommended
	}
	target := path

	// Safety: Prevent using the flip project/source folder as the target
	projectMarkers := []string{"go.mod", "internal/commands/init.go", "internal/brain/creator.go"}
	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(target, marker)); err == nil {
			return fmt.Errorf("you cannot use the flip project/source folder as your workspace; please choose a different directory")
		}
	}

	// Create workspace directory
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	fmt.Printf("[OK] Workspace '%s' created at %s\n", name, target)
	return nil
}

// Brain creation logic (used by 'flip new brain' and menu)
func runNewBrain() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("+ Create a new brain")

	// Name (default: 'default')
	fmt.Printf("? Name for your new workspace [default: default]: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	// Prevent 'flap' as workspace name
	if strings.ToLower(name) == "flap" {
		fmt.Println("[X] 'flap' is reserved for the main folder. Please choose another name.")
		return nil
	}
	// Default path: $HOME/flap/brains/<name>, but allow any custom path
	home, _ := os.UserHomeDir()
	recommended := filepath.Join(home, "flap", "brains", name)
	fmt.Printf("? Path for your brain [default: %s]: ", recommended)
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)
	if path == "" {
		path = recommended
	}
	target := path

	// Safety: Prevent using the flip project/source folder as the target
	projectMarkers := []string{"go.mod", "internal/commands/init.go", "internal/brain/creator.go"}
	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(target, marker)); err == nil {
			return fmt.Errorf("you cannot use the flip project/source folder as your brain; please choose a different directory")
		}
	}

	// Choose dialect/structure
	fmt.Println("? Choose structure:")
	fmt.Println("  1) flip (recommended)")
	fmt.Println("  2) dendron")
	fmt.Println("  3) obsidian")
	fmt.Println("  4) logseq")
	fmt.Printf("Choose (1/2/3/4) [default: 1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	if choice == "" {
		choice = "1"
	}

	var dialect string
	switch choice {
	case "2":
		dialect = "dendron"
	case "3":
		dialect = "obsidian"
	case "4":
		dialect = "logseq"
	default:
		dialect = "flip"
	}

	fmt.Printf("\n> Creating new brain '%s' at %s with '%s' structure...\n", name, target, dialect)

	if err := createBrainStructure(target, dialect); err != nil {
		return fmt.Errorf("failed to create structure: %w", err)
	}

	// Initialize with flip (this will auto-add to default workspace)
	if err := runDirectoryInit(target, "", true); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	fmt.Println("[OK] New brain created and initialized!")
	return nil
}

// createBrainStructure legt die passende Struktur für den Dialekt an
func createBrainStructure(target, dialect string) error {
	dirs := []string{}
	switch dialect {
	case "dendron":
		dirs = []string{"notes", "vault1", "vault2"}
		if err := os.WriteFile(filepath.Join(target, "dendron.yml"), []byte("version: 2.0\n"), 0644); err != nil {
			return err
		}
	case "obsidian":
		dirs = []string{"Daily Notes", "Templates", "Projects", "Meetings"}
		if err := os.MkdirAll(filepath.Join(target, ".obsidian"), 0755); err != nil {
			return err
		}
	case "logseq":
		dirs = []string{"journals", "pages"}
		if err := os.MkdirAll(filepath.Join(target, ".logseq"), 0755); err != nil {
			return err
		}
	case "flip":
		dirs = []string{"journal", "meetings", "notes", "tasks", "definitions", "templates", "assets/images", "assets/documents"}
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(target, dir), 0755); err != nil {
			return err
		}
	}
	return nil
}
