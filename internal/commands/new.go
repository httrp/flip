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
		Use:   "new [type]",
		Short: "Create something new",
		Long:  "Create new content (note, meeting-note, journal, task) or infrastructure (brain, workspace).\n\nExamples:\n  flip new               # Interactive menu\n  flip new note          # Create new note\n  flip new meeting-note  # Create new meeting note\n  flip new journal       # Create new journal\n  flip new task          # Create new task\n  flip new brain         # Create new brain\n  flip new workspace     # Create new workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewMenu(args)
		},
	}

	// Content creation subcommands
	noteCmd := &cobra.Command{
		Use:   "note",
		Short: "Create a new note",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateNote()
		},
	}
	cmd.AddCommand(noteCmd)

	meetingCmd := &cobra.Command{
		Use:     "meeting-note",
		Aliases: []string{"meeting"}, // Backward compatibility
		Short:   "Create a new meeting note",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateMeeting()
		},
	}
	cmd.AddCommand(meetingCmd)

	journalCmd := &cobra.Command{
		Use:   "journal",
		Short: "Create or open a daily journal",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateJournal()
		},
	}
	cmd.AddCommand(journalCmd)

	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Create a new task",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateTask()
		},
	}
	cmd.AddCommand(taskCmd)

	// Infrastructure subcommands
	brainCmd := &cobra.Command{
		Use:   "brain",
		Short: "Create a new brain",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewBrain()
		},
	}
	cmd.AddCommand(brainCmd)

	workspaceCmd := &cobra.Command{
		Use:   "workspace",
		Short: "Create a new workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewWorkspace()
		},
	}
	cmd.AddCommand(workspaceCmd)

	return cmd
}

// Interactive menu for 'flip new'
func runNewMenu(args []string) error {
	// If args provided, route to appropriate command
	if len(args) > 0 {
		switch args[0] {
		case "note":
			return runCreateNote()
		case "meeting-note", "meeting":
			return runCreateMeeting()
		case "journal":
			return runCreateJournal()
		case "task":
			return runCreateTask()
		case "brain":
			return runNewBrain()
		case "workspace":
			return runNewWorkspace()
		default:
			return fmt.Errorf("unknown type: %s", args[0])
		}
	}

	// Interactive menu if no args
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("+ What do you want to create?")
	fmt.Println("  1) Note")
	fmt.Println("  2) Meeting Note")
	fmt.Println("  3) Journal")
	fmt.Println("  4) Task")
	fmt.Println("  5) Brain")
	fmt.Println("  6) Workspace")
	fmt.Printf("Choose (1-6) [default: 1]: ")
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)
	switch choice {
	case "", "1":
		return runCreateNote()
	case "2":
		return runCreateMeeting()
	case "3":
		return runCreateJournal()
	case "4":
		return runCreateTask()
	case "5":
		return runNewBrain()
	case "6":
		return runNewWorkspace()
	default:
		fmt.Println("[X] Cancelled.")
		return nil
	}
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

	switch choice {
	case "1":
		fmt.Println()
		return runNewBrain()
	case "2":
		fmt.Println()
		// TODO: Implement add existing brain flow
		fmt.Println("→ Add existing brain (not yet implemented)")
		return nil
	default:
		return nil
	}
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
	fmt.Printf("  IMPORTANT: Path must be a NEW directory or include the brain name as the final folder.\n")
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

	// Comprehensive path validation
	if err := validateBrainPath(target, name); err != nil {
		fmt.Printf("\n✗ Error: %v\n\n", err)
		fmt.Println("💡 Tip: Use the recommended path or ensure:")
		fmt.Println("   - Path is a NEW directory (doesn't exist or is empty)")
		fmt.Println("   - Path is NOT inside another brain")
		fmt.Println("   - Path ends with the brain name")
		fmt.Println()
		return runNewBrain()
	}

	// Path validated - ensure it's absolute
	target, _ = filepath.Abs(target)

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

	switch followUp {
	case "1":
		fmt.Println()
		return runNewBrain()
	case "2":
		fmt.Println()
		// TODO: Implement add existing brain flow
		fmt.Println("→ Add existing brain (not yet implemented)")
		return nil
	default:
		return nil
	}
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

// isInsideExistingBrain checks if a path is inside an existing brain by looking for .flip.yaml in parent directories
func isInsideExistingBrain(targetPath string) (bool, string, error) {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return false, "", err
	}

	// Check each parent directory for .flip.yaml
	currentPath := filepath.Dir(absPath)
	for {
		markerPath := filepath.Join(currentPath, ".flip.yaml")
		if _, err := os.Stat(markerPath); err == nil {
			return true, currentPath, nil
		}

		// Move to parent
		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			// Reached root
			break
		}
		currentPath = parent
	}

	return false, "", nil
}

// containsExistingBrain checks if the target path contains any existing brains (subdirectories with .flip.yaml)
func containsExistingBrain(targetPath string) (bool, []string, error) {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return false, nil, err
	}

	// Path doesn't exist yet, so it can't contain brains
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return false, nil, nil
	}

	var foundBrains []string

	// Walk through subdirectories looking for .flip.yaml
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the target path itself
		if path == absPath {
			return nil
		}

		// Check if this directory has .flip.yaml
		if info.IsDir() {
			markerPath := filepath.Join(path, ".flip.yaml")
			if _, err := os.Stat(markerPath); err == nil {
				relPath, _ := filepath.Rel(absPath, path)
				foundBrains = append(foundBrains, relPath)
				// Don't descend into this brain
				return filepath.SkipDir
			}
		}

		return nil
	})

	return len(foundBrains) > 0, foundBrains, err
}

// validateBrainPath performs comprehensive validation of a brain path
func validateBrainPath(targetPath, brainName string) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// 1. Check if path is inside an existing brain
	isInside, parentBrain, err := isInsideExistingBrain(absPath)
	if err != nil {
		return fmt.Errorf("failed to check parent brains: %w", err)
	}
	if isInside {
		return fmt.Errorf("cannot create brain inside another brain\n   Parent brain found at: %s\n   Brains must not be nested.", parentBrain)
	}

	// 2. Check if path already exists
	stat, err := os.Stat(absPath)
	if err == nil {
		// Path exists
		if !stat.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", absPath)
		}

		// Check if directory is empty
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return fmt.Errorf("failed to read directory: %w", err)
		}

		if len(entries) > 0 {
			return fmt.Errorf("directory already exists and is not empty: %s\n   A brain MUST be created in a NEW, empty directory.\n   Please choose a different path.", absPath)
		}

		// Empty directory is OK, but check for nested brains
		containsBrains, foundBrains, err := containsExistingBrain(absPath)
		if err != nil {
			return fmt.Errorf("failed to check for nested brains: %w", err)
		}
		if containsBrains {
			return fmt.Errorf("path contains existing brain(s): %v\n   Cannot create brain containing other brains.", foundBrains)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check path: %w", err)
	}

	// 3. Warn if path doesn't end with brain name
	baseName := filepath.Base(absPath)
	if baseName != brainName && baseName != normalizePathName(brainName) {
		return fmt.Errorf("path should end with brain name '%s'\n   Current path: %s\n   Suggested: %s", brainName, absPath, filepath.Join(filepath.Dir(absPath), brainName))
	}

	return nil
}
