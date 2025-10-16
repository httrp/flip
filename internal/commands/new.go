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
	fmt.Println()

	// Check if 'default' is already taken
	config, _ := loadWorkspaceConfig()
	defaultTaken := false
	for _, ws := range config.Workspaces {
		if ws.Name == "default" {
			defaultTaken = true
			break
		}
	}

	// Suggest 'default' if available, otherwise 'workspace'
	defaultName := "default"
	if defaultTaken {
		defaultName = "workspace"
	}

	// Step 1: Get workspace name
	fmt.Printf("? Name for your new workspace [default: %s]\n", defaultName)
	fmt.Printf("  (or enter '0' to cancel): ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	// Handle cancellation
	if name == "0" || strings.ToLower(name) == "cancel" {
		fmt.Println("\n✗ Cancelled")
		return nil
	}

	if name == "" {
		name = defaultName
	}

	// Check if workspace already exists
	for _, ws := range config.Workspaces {
		if ws.Name == name {
			fmt.Printf("\n✗ Workspace '%s' already exists\n", name)
			return nil
		}
	}

	// Prevent 'flap' as workspace name
	if strings.ToLower(name) == "flap" {
		fmt.Println("\n✗ 'flap' is reserved for the main folder. Please choose another name.")
		return nil
	}

	// Step 2: Get workspace path (optional, just for information)
	home, _ := os.UserHomeDir()
	recommended := filepath.Join(home, "flap", "workspaces", name)
	fmt.Printf("\n→ Workspace name: %s\n", name)
	fmt.Printf("? Path for workspace metadata [default: %s]\n", recommended)
	fmt.Printf("  (or enter '0' to cancel): ")
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)

	// Handle cancellation
	if path == "0" || strings.ToLower(path) == "cancel" {
		fmt.Println("\n✗ Cancelled")
		return nil
	}

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

	// Now create everything (after all inputs are validated)
	// Create workspace directory (optional, for metadata/organization)
	if err := os.MkdirAll(target, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Add to config
	newWS := Workspace{
		Name:   name,
		Brains: []Brain{},
	}

	config.Workspaces = append(config.Workspaces, newWS)

	// Set as active if it's the first workspace
	if len(config.Workspaces) == 1 {
		config.ActiveWorkspace = name
	}

	if err := saveWorkspaceConfig(config); err != nil {
		return fmt.Errorf("failed to save workspace config: %w", err)
	}

	fmt.Printf("\n✓ Workspace '%s' created\n", name)
	if config.ActiveWorkspace == name {
		fmt.Printf("✓ Set as active workspace\n")
	}

	// Ask if user wants to add a brain now
	fmt.Println()
	fmt.Println("? Would you like to add a brain to this workspace now?")
	fmt.Println("  1) Yes, create a new brain")
	fmt.Println("  2) Yes, add an existing brain")
	fmt.Println("  0) No, finish")
	fmt.Printf("Choose (0/1/2) [default: 0]: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "1" {
		fmt.Println()
		return runNewBrain()
	} else if choice == "2" {
		fmt.Println()
		// TODO: Implement add existing brain flow
		fmt.Println("→ Add existing brain (not yet implemented)")
		return nil
	}

	return nil
}

// Brain creation logic (used by 'flip new brain' and menu)
func runNewBrain() error {
	// Ensure we have an active workspace
	config, err := ensureActiveWorkspace()
	if err != nil {
		return err
	}
	if config == nil {
		return fmt.Errorf("no active workspace available")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("+ Create a new brain")
	fmt.Println()

	// Step 1: Ask how to choose the name
	fmt.Println("? How would you like to name your brain?")
	fmt.Println("  1) Enter a custom name")
	fmt.Println("  2) Choose from creative suggestions")
	fmt.Println("  0) Cancel")
	fmt.Printf("Choose (0/1/2) [default: 1]: ")

	nameChoice, _ := reader.ReadString('\n')
	nameChoice = strings.TrimSpace(nameChoice)
	if nameChoice == "" {
		nameChoice = "1"
	}

	// Handle cancellation
	if nameChoice == "0" || strings.ToLower(nameChoice) == "cancel" || strings.ToLower(nameChoice) == "abort" {
		fmt.Println("\n✗ Cancelled")
		return nil
	}

	var name string

	if nameChoice == "2" || strings.ToLower(nameChoice) == "suggestions" {
		// Step 2a: Show creative suggestions
		// Use config from ensureActiveWorkspace
		existingBrainNames := make(map[string]bool)
		for _, ws := range config.Workspaces {
			for _, b := range ws.Brains {
				existingBrainNames[b.Name] = true
			}
		}

		// Creative name selection
		creativeNames := []string{
			"atlas", "odyssey", "aurora", "echo", "zenith", "soliloquy", "muse", "oracle", "serendipity", "epiphany",
			"noesis", "elysium", "satori", "logos", "cosmos", "paradox", "quasar", "zeitgeist", "sophia", "mythos",
			// Classic PKM names
			"zettelkasten", "memex", "roam", "second-brain", "pkm", "garden", "vault", "archive", "library", "index",
		}

		// Filter out already used names
		var availableNames []string
		for _, n := range creativeNames {
			if !existingBrainNames[n] {
				availableNames = append(availableNames, n)
			}
		}

		if len(availableNames) == 0 {
			fmt.Println("\n✗ All suggested names are already in use. Please enter a custom name.")
			nameChoice = "1"
		} else {
			fmt.Println("\n? Choose a name from suggestions:")
			for i, n := range availableNames {
				if i >= 20 { // Limit display to 20 options
					break
				}
				fmt.Printf("  %d) %s\n", i+1, n)
			}
			fmt.Println("  0) Back (enter custom name instead)")
			fmt.Printf("Choose (0-%d): ", min(len(availableNames), 20))

			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			// Handle back/cancel
			if choice == "0" || strings.ToLower(choice) == "back" || strings.ToLower(choice) == "custom" {
				nameChoice = "1" // Switch to custom name entry
			} else if choice != "" {
				// Parse number choice
				idx := 0
				fmt.Sscanf(choice, "%d", &idx)
				if idx > 0 && idx <= len(availableNames) && idx <= 20 {
					name = availableNames[idx-1]
				} else {
					fmt.Println("\n✗ Invalid choice. Please try again.")
					return runNewBrain()
				}
			} else {
				// Default to first suggestion
				name = availableNames[0]
			}
		}
	}

	// Step 2b: Custom name entry
	if nameChoice == "1" || name == "" {
		fmt.Println("\n? Enter a custom name for your brain:")
		fmt.Printf("Brain name: ")
		name, _ = reader.ReadString('\n')
		name = strings.TrimSpace(name)

		if name == "" {
			fmt.Println("\n✗ No name provided. Cancelled.")
			return nil
		}

		// Prevent 'flap' as brain name
		if strings.ToLower(name) == "flap" {
			fmt.Println("\n✗ 'flap' is reserved for the main folder. Please choose another name.")
			return runNewBrain()
		}
	}

	fmt.Printf("\n→ Brain name: %s\n", name)

	// Step 3: Ask for path
	home, _ := os.UserHomeDir()
	recommended := filepath.Join(home, "flap", "brains", name)
	fmt.Printf("\n? Path for your brain [default: %s]\n", recommended)
	fmt.Printf("  (or enter '0' to cancel): ")
	path, _ := reader.ReadString('\n')
	path = strings.TrimSpace(path)

	// Handle cancellation
	if path == "0" || strings.ToLower(path) == "cancel" || strings.ToLower(path) == "abort" {
		fmt.Println("\n✗ Cancelled")
		return nil
	}

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
	fmt.Println("\n? Choose structure:")
	fmt.Println("  1) flip (recommended)")
	fmt.Println("  2) dendron")
	fmt.Println("  3) obsidian")
	fmt.Println("  4) logseq")
	fmt.Println("  0) Cancel")
	fmt.Printf("Choose (0/1/2/3/4) [default: 1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	// Handle cancellation
	if choice == "0" || strings.ToLower(choice) == "cancel" || strings.ToLower(choice) == "abort" {
		fmt.Println("\n✗ Cancelled")
		return nil
	}

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

	// ALL INPUTS COLLECTED - Now create everything
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("→ Brain name: %s\n", name)
	fmt.Printf("→ Path: %s\n", target)
	fmt.Printf("→ Structure: %s\n", dialect)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Println("\n> Creating brain structure...")

	// Create the brain structure
	if err := createBrainStructure(target, dialect); err != nil {
		// Cleanup on error
		os.RemoveAll(target)
		return fmt.Errorf("failed to create structure: %w", err)
	}

	// Initialize with flip (pass the name to avoid asking again)
	fmt.Println("> Initializing brain...")
	if err := runDirectoryInitWithName(target, name, "", true); err != nil {
		// Cleanup on error
		os.RemoveAll(target)
		return fmt.Errorf("failed to initialize: %w", err)
	}

	fmt.Println("\n✓ New brain created and initialized successfully!")

	// Ask if user wants to create another brain
	fmt.Println()
	fmt.Println("? Would you like to create or add another brain?")
	fmt.Println("  1) Yes, create another new brain")
	fmt.Println("  2) Yes, add an existing brain")
	fmt.Println("  0) No, finish")
	fmt.Printf("Choose (0/1/2) [default: 0]: ")

	followUp, _ := reader.ReadString('\n')
	followUp = strings.TrimSpace(followUp)

	if followUp == "1" {
		fmt.Println()
		return runNewBrain()
	} else if followUp == "2" {
		fmt.Println()
		// TODO: Implement add existing brain flow
		fmt.Println("→ Add existing brain (not yet implemented)")
		return nil
	}

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

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
