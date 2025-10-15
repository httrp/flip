package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func NewQuickstartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "quickstart",
		Short: "Guided onboarding for flip",
		Long:  "Walk through flip setup step-by-step with explanations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQuickstart()
		},
	}
}

func runQuickstart() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("===============================================================")
	fmt.Println("===  Welcome to Flip - Your Brain Management Assistant  ===")
	fmt.Println("===============================================================")
	fmt.Println()
	fmt.Println("Flip helps you manage your knowledge bases (brains) efficiently.")
	fmt.Println()
	fmt.Println("Key Concepts:")
	fmt.Println("  + BRAIN: A single knowledge base directory")
	fmt.Println("            (Obsidian vault, Logseq graph, markdown folder, etc.)")
	fmt.Println()
	fmt.Println("  ~ WORKSPACE: A collection of related brains")
	fmt.Println("                (like VS Code workspaces with multiple folders)")
	fmt.Println()
	fmt.Println("  # DEFAULT WORKSPACE: Automatically contains ALL brains")
	fmt.Println("                        flip knows about on this machine")
	fmt.Println()
	fmt.Println("---------------------------------------------------------------")
	fmt.Println()

	// Check existing setup
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	hasWorkspaces := len(config.Workspaces) > 0
	hasBrains := false
	totalBrains := 0
	for _, ws := range config.Workspaces {
		totalBrains += len(ws.Brains)
		if len(ws.Brains) > 0 {
			hasBrains = true
		}
	}

	if hasWorkspaces && hasBrains {
		fmt.Printf("[OK] You already have %d workspace(s) with %d brain(s)!\n\n", len(config.Workspaces), totalBrains)
		fmt.Println("Current setup:")
		runStatus()
		fmt.Println()

		fmt.Print("? Would you like to add another brain? (y/N): ")
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println()
			fmt.Println("[OK] All set! Use 'flip status' anytime to see your setup.")
			return nil
		}

		return guideAddBrain(reader)
	}

	// No setup yet - guide through first brain creation
	fmt.Println("! No brains configured yet. Let's create your first one!")
	fmt.Println()

	return guideFirstBrain(reader)
}

func guideFirstBrain(reader *bufio.Reader) error {
	fmt.Println("=== Creating Your First Brain ===")
	fmt.Println()
	fmt.Println("We'll create a new brain directory with a flip-compatible structure.")
	fmt.Println("This brain will be automatically added to the 'default' workspace.")
	fmt.Println()

	// Ask for name
	fmt.Print("? What should we call your brain? [default: flap]: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = "flap"
	}

	// Ask for location
	home, _ := os.UserHomeDir()
	suggestedPath := filepath.Join(home, name)

	fmt.Printf("? Where should we create it? [default: %s]: ", suggestedPath)
	pathInput, _ := reader.ReadString('\n')
	pathInput = strings.TrimSpace(pathInput)
	if pathInput == "" {
		pathInput = suggestedPath
	}

	target, err := filepath.Abs(pathInput)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(target); err == nil {
		fmt.Println()
		fmt.Printf("! Directory '%s' already exists.\n", target)
		fmt.Print("? Initialize it as a brain? (y/N): ")
		confirm, _ := reader.ReadString('\n')
		confirm = strings.TrimSpace(strings.ToLower(confirm))
		if confirm != "y" && confirm != "yes" {
			fmt.Println("[X] Cancelled.")
			return nil
		}
	}

	fmt.Println()
	fmt.Printf("> Creating brain '%s' at: %s\n", name, target)
	fmt.Println()

	// Create structure
	if err := createBrainStructure(target, "flip"); err != nil {
		return fmt.Errorf("failed to create structure: %w", err)
	}

	// Initialize
	if err := runDirectoryInit(target, "", true); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	fmt.Println()
	fmt.Println("===============================================================")
	fmt.Println("===  Brain Created Successfully!  ===")
	fmt.Println("===============================================================")
	fmt.Println()
	fmt.Println("What just happened:")
	fmt.Println()
	fmt.Printf("  1. Created brain directory at: %s\n", target)
	fmt.Println("  2. Added flip-compatible folder structure:")
	fmt.Println("       journal/     - Daily notes")
	fmt.Println("       meetings/    - Meeting notes")
	fmt.Println("       notes/       - General notes")
	fmt.Println("       tasks/       - Task lists")
	fmt.Println("       definitions/ - People, orgs, contexts")
	fmt.Println("       templates/   - Note templates")
	fmt.Println()
	fmt.Println("  3. Created 'default' workspace")
	fmt.Printf("  4. Added '%s' brain to 'default' workspace\n", name)
	fmt.Println()
	fmt.Println("---------------------------------------------------------------")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  flip status               # See your setup")
	fmt.Println("  flip workspace create personal  # Create another workspace")
	fmt.Println("  flip brain add <path>     # Add existing brain")
	fmt.Println()

	return nil
}

func guideAddBrain(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Println("=== Adding a Brain ===")
	fmt.Println()
	fmt.Println("You can:")
	fmt.Println("  1) Create a new brain")
	fmt.Println("  2) Connect an existing directory")
	fmt.Println()

	fmt.Print("? Choose (1/2) [default: 1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	if choice == "2" {
		return guideConnectExisting(reader)
	}

	return guideCreateNew(reader)
}

func guideConnectExisting(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Print("? Path to existing brain directory: ")
	pathInput, _ := reader.ReadString('\n')
	pathInput = strings.TrimSpace(pathInput)

	if pathInput == "" {
		fmt.Println("[X] No path provided.")
		return nil
	}

	target, err := filepath.Abs(pathInput)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", target)
	}

	fmt.Print("? Name for this brain [default: auto-detect]: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = filepath.Base(target)
	}

	// Initialize if not already a brain
	fmt.Println()
	fmt.Printf("> Initializing '%s' at: %s\n", name, target)
	if err := runDirectoryInit(target, "", false); err != nil {
		fmt.Printf("! Warning: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("[OK] Brain '%s' added to default workspace!\n", name)
	fmt.Println()
	fmt.Println("Use 'flip status' to see your setup.")

	return nil
}

func guideCreateNew(reader *bufio.Reader) error {
	fmt.Println()
	fmt.Print("? Name for new brain: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = "brain"
	}

	home, _ := os.UserHomeDir()
	suggestedPath := filepath.Join(home, name)

	fmt.Printf("? Where to create it? [default: %s]: ", suggestedPath)
	pathInput, _ := reader.ReadString('\n')
	pathInput = strings.TrimSpace(pathInput)
	if pathInput == "" {
		pathInput = suggestedPath
	}

	target, err := filepath.Abs(pathInput)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	fmt.Println()
	fmt.Printf("> Creating brain '%s' at: %s\n", name, target)

	// Create and initialize
	if err := createBrainStructure(target, "flip"); err != nil {
		return fmt.Errorf("failed to create structure: %w", err)
	}

	if err := runDirectoryInit(target, "", true); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	fmt.Println()
	fmt.Printf("[OK] Brain '%s' created and added to default workspace!\n", name)
	fmt.Println()
	fmt.Println("Use 'flip status' to see your setup.")

	return nil
}
