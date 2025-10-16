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
		Short: "Create a new brain with compatible structure",
		Long:  "Creates a new brain folder with a structure compatible to Obsidian, Logseq, Dendron, or Flip.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNew()
		},
	}
	return cmd
}

func runNew() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("+ Create a new brain")

	// Name
	fmt.Printf("? Name for your new brain [default: flap]: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = "flap"
	}

	// Suggested default path
	home, _ := os.UserHomeDir()
	recommended := filepath.Join(home, name)
	fmt.Printf("? Where to create it? [default: %s]: ", recommended)
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)
	if path == "" {
		path = recommended
	}
	target := path

	fmt.Printf("! Selected path: %s\n", target)
	fmt.Printf("Do you want to create your brain here? (y/N): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("[X] Cancelled by user.")
		return nil
	}

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

	// Ask if user wants to add to an additional workspace
	fmt.Printf("\n# Add to another workspace? (y/N): ")
	yn, _ := reader.ReadString('\n')
	yn = strings.TrimSpace(strings.ToLower(yn))
	if yn == "y" || yn == "yes" {
		// Ask for workspace name
		fmt.Printf("Add to existing workspace or create new? (existing/new) [default: existing]: ")
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(strings.ToLower(choice))

		workspaceName := ""
		if choice == "new" {
			fmt.Printf("New workspace name: ")
			wsInput, _ := reader.ReadString('\n')
			workspaceName = strings.TrimSpace(wsInput)
			if workspaceName == "" {
				workspaceName = name
			}
			// Create new workspace
			if err := runWorkspaceCreate(workspaceName); err != nil {
				fmt.Printf("%s Could not create workspace: %v\n", IconWarning, err)
				return nil
			}
		} else {
			fmt.Printf("Workspace name: ")
			wsInput, _ := reader.ReadString('\n')
			workspaceName = strings.TrimSpace(wsInput)
		}

		if workspaceName != "" && workspaceName != "default" {
			// Verify workspace exists and switch to it
			if err := runWorkspaceSwitch(workspaceName); err != nil {
				fmt.Printf("%s Workspace '%s' not found or cannot switch: %v\n", IconWarning, workspaceName, err)
				return nil
			}

			// Add brain to the now-active workspace
			if err := runBrainAdd(target, name, true); err != nil {
				fmt.Printf("%s Could not add brain: %v\n", IconWarning, err)
			} else {
				fmt.Printf("%s Brain '%s' added to workspace '%s'\n", IconCheck, name, workspaceName)
			}
		}
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
