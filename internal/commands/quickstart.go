package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func getText(key string) string {
	return lang.GetText(key)
}

// isProblematicTerminal detects terminals that don't work well with interactive prompts
// (Git Bash/mintty on Windows has issues with ANSI escape sequences)
func isProblematicTerminal() bool {
	// Check for Git Bash / MSYS / MinGW on Windows
	if runtime.GOOS == "windows" {
		msystem := os.Getenv("MSYSTEM")
		if msystem != "" {
			// MINGW64, MINGW32, MSYS, etc.
			return true
		}
		// Also check for mintty
		term := os.Getenv("TERM_PROGRAM")
		if term == "mintty" {
			return true
		}
	}
	return false
}

// simpleSelect provides a fallback numbered selection for problematic terminals
func simpleSelect(label string, items []string) (int, string, error) {
	fmt.Println()
	fmt.Println(label + ":")
	fmt.Println()
	for i, item := range items {
		fmt.Printf("  %d) %s\n", i+1, item)
	}
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter number (1-", len(items), "): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, "", err
		}
		input = strings.TrimSpace(input)
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(items) {
			fmt.Println("Please enter a valid number.")
			continue
		}
		return num - 1, items[num-1], nil
	}
}

// simplePrompt provides a fallback text input for problematic terminals
func simplePrompt(label string, defaultVal string) (string, error) {
	fmt.Println()
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" && defaultVal != "" {
		return defaultVal, nil
	}
	return input, nil
}

// selectPrompt wraps promptui.Select with fallback for problematic terminals
func selectPrompt(label string, items []string) (int, string, error) {
	if isProblematicTerminal() {
		return simpleSelect(label, items)
	}

	prompt := promptui.Select{
		Label: label,
		Items: items,
	}
	return prompt.Run()
}

// textPrompt wraps promptui.Prompt with fallback for problematic terminals
func textPrompt(label string, defaultVal string, validate func(string) error) (string, error) {
	if isProblematicTerminal() {
		for {
			result, err := simplePrompt(label, defaultVal)
			if err != nil {
				return "", err
			}
			if validate != nil {
				if err := validate(result); err != nil {
					fmt.Printf("Error: %s\n", err)
					continue
				}
			}
			return result, nil
		}
	}

	prompt := promptui.Prompt{
		Label:    label,
		Default:  defaultVal,
		Validate: validate,
	}
	return prompt.Run()
}

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
	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  " + getText("welcome_banner"))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Load existing config first to show current status
	config, err := loadWorkspaceConfig()
	if err != nil {
		config = &WorkspaceConfig{
			Version:    "2.0",
			Workspaces: []Workspace{},
		}
	}

	// Check if there's already a setup
	hasWorkspaces := len(config.Workspaces) > 0
	var activeWS *Workspace
	var activeBrainCount int

	if hasWorkspaces {
		// Find active workspace and count brains
		for i := range config.Workspaces {
			if config.Workspaces[i].Name == config.ActiveWorkspace {
				activeWS = &config.Workspaces[i]
				activeBrainCount = len(activeWS.Brains)
				break
			}
		}
	}

	// Show current setup status
	if hasWorkspaces && activeWS != nil {
		fmt.Println(getText("quickstart.current_setup_header"))
		fmt.Println()
		fmt.Printf("  "+getText("quickstart.current_workspace")+"\n", config.ActiveWorkspace)
		fmt.Println()
		fmt.Println("  " + getText("quickstart.current_brains"))
		if len(activeWS.Brains) > 0 {
			for _, brain := range activeWS.Brains {
				if brain.Name == activeWS.DefaultBrain {
					fmt.Printf("    "+getText("quickstart.brain_item_active")+"\n", brain.Name, brain.Type, brain.Path)
				} else {
					fmt.Printf("    "+getText("quickstart.brain_item")+"\n", brain.Name, brain.Type, brain.Path)
				}
			}
		} else {
			fmt.Println("    " + getText("quickstart.no_brains_yet"))
		}
		fmt.Println()

		// If setup looks complete, offer options
		if activeBrainCount > 0 {
			fmt.Println(getText("quickstart.setup_already_complete"))
			fmt.Println()

			setupOptions := []string{
				getText("quickstart.option_exit"),
				getText("quickstart.option_add_brain"),
				getText("quickstart.option_new_workspace"),
				getText("quickstart.option_reconfigure"),
			}

			actionIdx, _, err := selectPrompt(getText("quickstart.setup_options"), setupOptions)
			if err != nil {
				return fmt.Errorf("action selection failed: %w", err)
			}

			switch actionIdx {
			case 0: // Exit
				fmt.Println()
				fmt.Println("👋 Great! Your flip setup is ready to use.")
				fmt.Println()
				fmt.Println(getText("quickstart.next_steps"))
				return nil
			case 1: // Add brain - skip to brain step
				fmt.Println()
				return runAddBrainStep(config, config.ActiveWorkspace)
			case 2: // New workspace
				fmt.Println()
				return runWorkspaceStep(config, false)
			case 3: // Reconfigure - continue with full setup
				fmt.Println()
				fmt.Println("Starting fresh setup...")
				// Jump directly to workspace step with freshStart=true
				return runFreshStart(config)
			}
		}
	} else {
		// No workspaces yet - show intro
		fmt.Println(getText("quickstart.welcome_title"))
		fmt.Println()
		fmt.Println(getText("quickstart.no_workspaces_yet"))
		fmt.Println()
	}

	// Show educational content for new users
	fmt.Println(getText("quickstart.intro_workspace"))
	fmt.Println()
	fmt.Println(getText("quickstart.intro_brain"))
	fmt.Println()
	fmt.Println(getText("quickstart.brain_storage"))
	fmt.Println()
	fmt.Println(getText("quickstart.interaction_methods"))
	fmt.Println()
	fmt.Println(getText("quickstart.recommendations"))
	fmt.Println()
	fmt.Println("💡 " + getText("quickstart.flap_folder_info"))
	fmt.Println()

	// Run workspace step
	return runWorkspaceStep(config, false)
}

// runFreshStart handles the "Start fresh" flow - shows educational content and runs workspace step with forceNew
func runFreshStart(config *WorkspaceConfig) error {
	fmt.Println()

	// Show educational content
	fmt.Println(getText("quickstart.intro_workspace"))
	fmt.Println()
	fmt.Println(getText("quickstart.intro_brain"))
	fmt.Println()
	fmt.Println(getText("quickstart.brain_storage"))
	fmt.Println()
	fmt.Println(getText("quickstart.interaction_methods"))
	fmt.Println()
	fmt.Println(getText("quickstart.recommendations"))
	fmt.Println()
	fmt.Println("💡 " + getText("quickstart.flap_folder_info"))
	fmt.Println()

	// Run workspace step with forceNew=true
	return runWorkspaceStep(config, true)
}

// runWorkspaceStep handles workspace creation/selection
func runWorkspaceStep(config *WorkspaceConfig, forceNew bool) error {
	// Step 1: Create workspace with better UX
	fmt.Println("━━━ " + getText("quickstart.step_workspace") + " ━━━")
	fmt.Println()

	// Choose between default and custom workspace name
	wsChoiceItems := []string{
		getText("quickstart.workspace_default_option"),
		getText("quickstart.workspace_custom_option"),
	}

	wsChoice, _, err := selectPrompt(getText("quickstart.prompt_workspace_name"), wsChoiceItems)
	if err != nil {
		return fmt.Errorf("workspace choice failed: %w", err)
	}

	var wsName string
	if wsChoice == 0 {
		// Use default
		wsName = "default"
		fmt.Println()
		fmt.Println("✓ Using workspace 'default'")
	} else {
		// Custom name with suggestions
		fmt.Println()
		fmt.Println(getText("quickstart.workspace_suggestions"))
		fmt.Println()

		validate := func(input string) error {
			if strings.ToLower(input) == "flap" {
				return fmt.Errorf("%s", getText("quickstart.reserved_flap"))
			}
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("workspace name cannot be empty")
			}
			return nil
		}

		wsName, err = textPrompt(getText("quickstart.workspace_prompt_custom"), "", validate)
		if err != nil {
			return fmt.Errorf("workspace prompt failed: %w", err)
		}
		wsName = strings.TrimSpace(wsName)
	}

	// Check if workspace already exists
	workspaceExists := false
	var existingWSIdx int = -1
	for i, ws := range config.Workspaces {
		if ws.Name == wsName {
			workspaceExists = true
			existingWSIdx = i
			break
		}
	}

	if workspaceExists {
		existingBrainCount := len(config.Workspaces[existingWSIdx].Brains)

		if forceNew && existingBrainCount > 0 {
			// Fresh start mode - ask if user wants to clear the workspace
			fmt.Println()
			fmt.Printf("⚠️  Workspace '%s' already exists with %d brain(s):\n", wsName, existingBrainCount)
			for _, b := range config.Workspaces[existingWSIdx].Brains {
				fmt.Printf("   • %s (%s)\n", b.Name, b.Path)
			}
			fmt.Println()

			clearOptions := []string{
				fmt.Sprintf("Clear workspace and start fresh (remove %d brain(s) from config)", existingBrainCount),
				"Keep existing brains and add more",
				"Cancel and go back",
			}

			clearIdx, _, err := selectPrompt("What would you like to do?", clearOptions)
			if err != nil || clearIdx == 2 {
				return fmt.Errorf("setup cancelled")
			}

			if clearIdx == 0 {
				// Clear the workspace brains
				fmt.Println()
				fmt.Printf("🗑️  Clearing workspace '%s'...\n", wsName)
				fmt.Println("   (Note: Your brain folders and files are NOT deleted, only removed from flip's config)")
				config.Workspaces[existingWSIdx].Brains = []Brain{}
				if err := saveWorkspaceConfig(config); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
				fmt.Printf("✓ Workspace '%s' cleared\n\n", wsName)
			} else {
				fmt.Printf("\n✓ Keeping existing brains in workspace '%s'\n\n", wsName)
			}
		} else {
			fmt.Printf("\n✓ Workspace '%s' already exists, using it\n\n", wsName)
		}

		config.ActiveWorkspace = wsName
		if err := saveWorkspaceConfig(config); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if !workspaceExists {
		home, _ := os.UserHomeDir()
		wsPath := filepath.Join(home, "flap", "workspaces", wsName)

		wsPath, err = textPrompt(getText("quickstart.prompt_workspace_path"), wsPath, nil)
		if err != nil {
			return fmt.Errorf("path prompt failed: %w", err)
		}
		wsPath = strings.TrimSpace(wsPath)

		// Create workspace directory
		if err := os.MkdirAll(wsPath, 0755); err != nil {
			return fmt.Errorf("failed to create workspace directory: %w", err)
		}

		// Add workspace to config
		newWS := Workspace{
			Name:   wsName,
			Brains: []Brain{},
		}
		config.Workspaces = append(config.Workspaces, newWS)

		// Set as active workspace
		config.ActiveWorkspace = wsName

		// Save config
		if err := saveWorkspaceConfig(config); err != nil {
			return fmt.Errorf("failed to save workspace config: %w", err)
		}

		fmt.Println()
		fmt.Printf("✓ "+getText("quickstart.workspace_created")+"\n", wsName, wsPath)
		fmt.Println()
	}

	// Continue to brain step
	return runAddBrainStep(config, wsName)
}

// runAddBrainStep handles adding a brain to a workspace
func runAddBrainStep(config *WorkspaceConfig, wsName string) error {
	// Step 2: Add a brain
	fmt.Println("━━━ " + getText("quickstart.step_brain") + " ━━━")
	fmt.Println()

	brainTypeItems := []string{
		getText("quickstart.option_new"),
		getText("quickstart.option_existing"),
		getText("quickstart.option_scan"),
	}

	brainTypeIdx, _, err := selectPrompt(getText("quickstart.brain_options"), brainTypeItems)
	if err != nil {
		return fmt.Errorf("brain type selection failed: %w", err)
	}

	// Option 3: Scan for brains
	if brainTypeIdx == 2 {
		home, _ := os.UserHomeDir()

		// Educational text before scan
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println(getText("quickstart.what_is_brain"))
		fmt.Println(getText("quickstart.brain_explanation"))
		fmt.Println()
		fmt.Println(getText("quickstart.supported_types"))
		fmt.Println(getText("quickstart.flip_type"))
		fmt.Println(getText("quickstart.obsidian_type"))
		fmt.Println(getText("quickstart.logseq_type"))
		fmt.Println(getText("quickstart.dendron_type"))
		fmt.Println(getText("quickstart.foam_type"))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("🔍 Scanning for existing brains...")
		fmt.Println("   Looking for: Flip, Obsidian, Logseq, Dendron, Foam brains")
		fmt.Println("   (Ignoring generic markdown folders)")
		fmt.Println()

		// Scan from home directory - only show recognized brain types, not "potential" brains
		if err := runScanWithOptions(home, ScanOptions{
			MinBrainFiles:   5,
			MinContentFiles: 50,
			MaxDepth:        5,
			ShowPotential:   false, // Only show recognized brain types!
		}); err != nil {
			fmt.Printf("⚠️  Scan completed with warnings: %v\n", err)
		}

		// After scan, offer to add a brain
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		afterScanOptions := []string{
			"Add a brain from the list above (enter path)",
			"Continue with setup complete (brains already added)",
			"Return to main options",
		}

		afterIdx, _, err := selectPrompt("What would you like to do?", afterScanOptions)
		if err != nil || afterIdx == 2 {
			// Return to brain options
			return runAddBrainStep(config, wsName)
		}

		// Option: Continue with setup complete
		if afterIdx == 1 {
			fmt.Println()
			fmt.Println(getText("quickstart.setup_complete"))
			fmt.Printf("  • Workspace: %s\n", wsName)
			fmt.Println()
			fmt.Println(getText("quickstart.next_steps"))
			fmt.Println()
			return nil
		}

		// User wants to add a brain - go to the path entry flow
		fmt.Println()
		fmt.Println("💡 Copy the path from the scan results above")
		fmt.Println()

		validateBrainPath := func(input string) error {
			if input == "" {
				return fmt.Errorf("%s", getText("quickstart.no_path_provided"))
			}
			// Expand ~ to home directory
			if strings.HasPrefix(input, "~/") {
				input = filepath.Join(home, input[2:])
			}
			absPath, err := filepath.Abs(input)
			if err != nil {
				return fmt.Errorf(getText("quickstart.invalid_path"), err)
			}
			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return fmt.Errorf(getText("quickstart.path_not_exist"), absPath)
			}
			return nil
		}

		brainPathInput, err := textPrompt("Enter brain path", "", validateBrainPath)
		if err != nil {
			return fmt.Errorf("brain path prompt failed: %w", err)
		}

		// Expand ~ if present
		if strings.HasPrefix(brainPathInput, "~/") {
			brainPathInput = filepath.Join(home, brainPathInput[2:])
		}

		brainTarget, _ := filepath.Abs(brainPathInput)
		defaultName := filepath.Base(brainTarget)

		validateBrainName := func(input string) error {
			if strings.ToLower(input) == "flap" {
				return fmt.Errorf("%s", getText("quickstart.reserved_flap"))
			}
			return nil
		}

		brainName, err := textPrompt(getText("quickstart.prompt_existing_brain_name"), defaultName, validateBrainName)
		if err != nil {
			return fmt.Errorf("brain name prompt failed: %w", err)
		}

		// Initialize and add brain
		fmt.Println(fmt.Sprintf(getText("quickstart.initializing_brain"), brainName, brainTarget))
		if err := runDirectoryInitWithName(brainTarget, brainName, "", false); err != nil {
			fmt.Printf("! Warning: %v\n", err)
		}

		// Show completion message
		fmt.Println()
		fmt.Println(getText("quickstart.setup_complete"))
		fmt.Printf("  • Workspace: %s\n", wsName)
		fmt.Printf("  • Brain: %s (%s)\n", brainName, brainTarget)
		fmt.Println()
		fmt.Println(getText("quickstart.next_steps"))
		fmt.Println()
		return nil
	}

	// Option 2: Connect existing brain
	if brainTypeIdx == 1 {
		home, _ := os.UserHomeDir()

		// Educational text
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println(getText("quickstart.what_is_brain"))
		fmt.Println(getText("quickstart.brain_explanation"))
		fmt.Println()
		fmt.Println(getText("quickstart.supported_types"))
		fmt.Println(getText("quickstart.flip_type"))
		fmt.Println(getText("quickstart.obsidian_type"))
		fmt.Println(getText("quickstart.logseq_type"))
		fmt.Println(getText("quickstart.dendron_type"))
		fmt.Println(getText("quickstart.foam_type"))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		fmt.Println("📂 Where is your existing brain located?")
		fmt.Println()

		pathOptions := []string{
			"Enter path manually",
			fmt.Sprintf("~/Documents (%s)", filepath.Join(home, "Documents")),
			fmt.Sprintf("~/Dropbox (%s)", filepath.Join(home, "Dropbox")),
			fmt.Sprintf("~/ (Home: %s)", home),
		}

		// Add platform-specific common paths
		if runtime.GOOS == "windows" {
			pathOptions = append(pathOptions, fmt.Sprintf("OneDrive (%s)", filepath.Join(home, "OneDrive")))
		}

		pathIdx, _, err := selectPrompt("Choose a starting location or enter path", pathOptions)
		if err != nil {
			return fmt.Errorf("path selection failed: %w", err)
		}

		var brainPathInput string

		if pathIdx == 0 {
			// Manual entry
			validateBrainPath := func(input string) error {
				if input == "" {
					return fmt.Errorf("%s", getText("quickstart.no_path_provided"))
				}
				// Expand ~ to home directory
				if strings.HasPrefix(input, "~/") {
					input = filepath.Join(home, input[2:])
				}
				absPath, err := filepath.Abs(input)
				if err != nil {
					return fmt.Errorf(getText("quickstart.invalid_path"), err)
				}
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					return fmt.Errorf(getText("quickstart.path_not_exist"), absPath)
				}
				return nil
			}

			brainPathInput, err = textPrompt(getText("quickstart.prompt_existing_brain_path"), "", validateBrainPath)
			if err != nil {
				return fmt.Errorf("brain path prompt failed: %w", err)
			}
		} else {
			// Extract actual path from selection
			var basePath string
			switch pathIdx {
			case 1:
				basePath = filepath.Join(home, "Documents")
			case 2:
				basePath = filepath.Join(home, "Dropbox")
			case 3:
				basePath = home
			case 4:
				basePath = filepath.Join(home, "OneDrive")
			}

			// List subdirectories and let user choose
			entries, err := os.ReadDir(basePath)
			if err != nil {
				return fmt.Errorf("failed to read directory: %w", err)
			}

			var subdirs []string
			subdirs = append(subdirs, "[Use this folder: "+basePath+"]")
			subdirs = append(subdirs, "[Enter different path]")

			for _, entry := range entries {
				if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
					subdirs = append(subdirs, entry.Name())
				}
			}

			if len(subdirs) > 2 {
				fmt.Println()
				subIdx, subName, err := selectPrompt("Select a folder", subdirs)
				if err != nil {
					return fmt.Errorf("folder selection failed: %w", err)
				}

				if subIdx == 0 {
					brainPathInput = basePath
				} else if subIdx == 1 {
					// Manual entry fallback
					brainPathInput, err = textPrompt("Enter full path", "", nil)
					if err != nil {
						return fmt.Errorf("path prompt failed: %w", err)
					}
				} else {
					brainPathInput = filepath.Join(basePath, subName)
				}
			} else {
				brainPathInput = basePath
			}
		}

		// Expand ~ if present
		if strings.HasPrefix(brainPathInput, "~/") {
			brainPathInput = filepath.Join(home, brainPathInput[2:])
		}

		brainTarget, _ := filepath.Abs(brainPathInput)
		defaultName := filepath.Base(brainTarget)

		validateBrainName := func(input string) error {
			if strings.ToLower(input) == "flap" {
				return fmt.Errorf("%s", getText("quickstart.reserved_flap"))
			}
			return nil
		}

		brainName, err := textPrompt(getText("quickstart.prompt_existing_brain_name"), defaultName, validateBrainName)
		if err != nil {
			return fmt.Errorf("brain name prompt failed: %w", err)
		}

		// Initialize if not already a brain (pass brainName to avoid asking again)
		fmt.Println(fmt.Sprintf(getText("quickstart.initializing_brain"), brainName, brainTarget))
		if err := runDirectoryInitWithName(brainTarget, brainName, "", false); err != nil {
			fmt.Printf("! Warning: %v\n", err)
		}

		// Show completion message
		fmt.Println()
		fmt.Println(getText("quickstart.setup_complete"))
		fmt.Printf("  • Workspace: %s\n", wsName)
		fmt.Printf("  • Brain: %s (%s)\n", brainName, brainTarget)
		fmt.Println()
		fmt.Println(getText("quickstart.next_steps"))
		fmt.Println()
		return nil
	}

	// Create new brain
	allCreativeNames := []string{
		// Classic second brain concepts
		"zettelkasten", "memex", "commonplace-book", "digital-garden", "knowledge-base", "wiki",
		"personal-wiki", "pkm", "slipbox", "exobrain", "antinet",
		// Creative/poetic names
		"atlas", "odyssey", "aurora", "echo", "zenith", "muse", "oracle", "serendipity", "epiphany",
		"noesis", "elysium", "satori", "logos", "cosmos", "paradox", "quasar", "zeitgeist", "sophia", "mythos",
		"nexus", "vault", "archive", "library", "repository", "codex", "compendium",
	}

	// Filter out already-used brain names
	usedNames := make(map[string]bool)
	for _, ws := range config.Workspaces {
		for _, brain := range ws.Brains {
			usedNames[strings.ToLower(brain.Name)] = true
		}
	}

	var creativeNames []string
	creativeNames = append(creativeNames, "[Enter custom name]")
	for _, name := range allCreativeNames {
		if !usedNames[strings.ToLower(name)] {
			creativeNames = append(creativeNames, name)
		}
	}

	idx, brainName, err := selectPrompt(getText("quickstart.suggested_names"), creativeNames)
	if err != nil {
		return fmt.Errorf("brain name selection failed: %w", err)
	}

	// If custom name was selected, prompt for it
	if idx == 0 {
		validateCustomName := func(input string) error {
			if strings.ToLower(input) == "flap" {
				return fmt.Errorf("%s", getText("quickstart.reserved_flap"))
			}
			if input == "" {
				return fmt.Errorf("brain name cannot be empty")
			}
			return nil
		}
		brainName, err = textPrompt(getText("quickstart.prompt_brain_name"), "", validateCustomName)
		if err != nil {
			return fmt.Errorf("custom name prompt failed: %w", err)
		}
	}

	home, _ := os.UserHomeDir()
	brainPath := filepath.Join(home, "flap", "brains", brainName)

	brainPath, err = textPrompt(getText("quickstart.prompt_brain_path"), brainPath, nil)
	if err != nil {
		return fmt.Errorf("brain path prompt failed: %w", err)
	}
	brainPath = strings.TrimSpace(brainPath)

	fmt.Println()
	fmt.Println(fmt.Sprintf(getText("quickstart.creating_brain"), brainName, brainPath))
	if err := createBrainStructure(brainPath, "flip"); err != nil {
		return fmt.Errorf("failed to create structure: %w", err)
	}
	// Pass brainName to avoid asking for it again
	if err := runDirectoryInitWithName(brainPath, brainName, "", true); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	fmt.Println()
	fmt.Println(fmt.Sprintf(getText("quickstart.brain_created"), brainName, wsName))
	fmt.Println()
	fmt.Println(getText("quickstart.setup_complete"))
	fmt.Printf("  • Workspace: %s\n", wsName)
	fmt.Printf("  • Brain: %s\n", brainName)
	fmt.Println()
	fmt.Println(getText("quickstart.next_steps"))
	fmt.Println()
	return nil
}
