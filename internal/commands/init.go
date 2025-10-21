package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/brain"
	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	var template string
	var force bool

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new brain repository",
		Long:  "Creates a new folder structure with templates and configuration for your brain.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var targetPath string

			if len(args) > 0 {
				// Directory name provided
				targetPath = args[0]
				return runDirectoryInit(targetPath, template, force)
			} else {
				// Interactive mode in current directory
				return runInteractiveInit(template, force)
			}
		},
	}

	cmd.Flags().StringVarP(&template, "template", "t", "", "Template to use (personal|work|learning)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force initialization even if directory is not empty")

	return cmd
}

type BrainConfig struct {
	Name                string
	Type                string
	DefaultOrganization string
	Author              string
}

func runDirectoryInit(directory, template string, force bool) error {
	return runDirectoryInitWithName(directory, "", template, force)
}

// runDirectoryInitWithName initializes a directory with an optional predefined name
func runDirectoryInitWithName(directory, brainName, template string, force bool) error {
	// Ensure we have an active workspace
	if _, err := ensureActiveWorkspace(); err != nil {
		return err
	}

	// Get absolute path
	absPath, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Prevent initialization in flip source/project folder
	projectMarkers := []string{"go.mod", "internal/commands/init.go", "internal/brain/creator.go"}
	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(absPath, marker)); err == nil {
			return fmt.Errorf("you are trying to initialize in the flip project/source folder; please choose a different directory")
		}
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", absPath, err)
	}

	// Detect existing brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(absPath)
	if err != nil {
		return fmt.Errorf("failed to detect brain type: %w", err)
	}

	fmt.Printf("%s Flip - Initialize your brain\n\n", IconBrain)
	fmt.Printf("Target directory: %s\n", absPath)
	fmt.Printf("%s Detection result: %s\n", IconArrow, detection.Description)

	if len(detection.Indicators) > 0 {
		fmt.Println("   Indicators found:")
		for _, indicator := range detection.Indicators {
			fmt.Printf("   - %s\n", indicator)
		}
	}

	fmt.Printf("%s Compatibility: %s\n\n", IconSuccess, detector.GetCompatibilityInfo(detection.Type))

	// Check if we can proceed
	if !detection.Compatible && !force {
		return fmt.Errorf("directory contains incompatible content. Use --force to override")
	}

	if detection.Type != brain.BrainTypeEmpty && detection.Type != brain.BrainTypeFlip && !force {
		fmt.Printf("%s This directory already contains a %s setup.\n", IconWarning, detection.Type)
		fmt.Println("Flip will add its structure while preserving existing files.")

		if !askForConfirmation("Continue with flip initialization?") {
			fmt.Println("Initialization cancelled.")
			return nil
		}
	}

	return initializeBrainAtPathWithName(absPath, brainName, template, detection.Type)
}

func askForConfirmation(question string) bool {
	fmt.Printf("%s (y/N): ", question)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func initializeBrainAtPathWithName(path, brainName, template string, existingType brain.BrainType) error {
	// Use template or interactive config
	var config BrainConfig
	if template != "" {
		config = getTemplateConfig(template)
		fmt.Printf("i Using template: %s\n", template)
	} else {
		// If brainName is provided, use it; otherwise prompt
		if brainName != "" {
			config = BrainConfig{
				Name:                brainName,
				Type:                "personal", // Default type
				DefaultOrganization: "PERSONAL",
				Author:              "Your Name",
			}
		} else {
			config = promptForConfig(filepath.Base(path))
		}
	}

	// Create brain with compatibility considerations
	creator := brain.NewCreator()
	brainConfig := brain.Config{
		Name:                config.Name,
		Type:                config.Type,
		DefaultOrganization: config.DefaultOrganization,
		Author:              config.Author,
	}

	// Adapt creation based on existing brain type
	if err := creator.CreateCompatible(path, brainConfig, existingType); err != nil {
		return err
	}

	// Auto-add to default workspace
	fmt.Println("\n~ Adding brain to default workspace...")
	if err := autoAddBrainToDefault(path, config.Name); err != nil {
		fmt.Printf("%s Warning: Could not add to default workspace: %v\n", IconWarning, err)
	} else {
		fmt.Println("[OK] Brain added to default workspace")
	}

	return nil
}

func runInteractiveInit(template string, force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Printf("%s Flip - Initialize your brain\n\n", IconBrain)
	fmt.Printf("Current directory: %s\n", cwd)

	// Check for existing brain
	if isBrainInitialized(cwd) {
		fmt.Printf("%s Brain already initialized in this directory\n", IconCheck)
		return nil
	}

	// Ask where to initialize
	fmt.Printf("\n? Where would you like to initialize your brain?\n")
	fmt.Printf("  1) Here in current directory\n")
	fmt.Printf("  2) Create new folder\n")
	choice := promptForInput("Choose (1/2)", "1")

	if choice == "2" || choice == "new" || choice == "folder" {
		// Create new folder mode
		folderName := promptForInput("Folder name", "my-brain")
		basePath := promptForInput("Base path", cwd)

		targetPath := filepath.Join(basePath, folderName)
		return runDirectoryInit(targetPath, template, force)
	}

	// Initialize in current directory
	// Check directory contents
	files, err := os.ReadDir(cwd)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if len(files) > 0 && !force {
		fmt.Printf("Found files: ")
		for i, file := range files {
			if i < 3 {
				fmt.Printf("%s", file.Name())
				if i < len(files)-1 && i < 2 {
					fmt.Printf(", ")
				}
			}
		}
		if len(files) > 3 {
			fmt.Printf("... (%d more)", len(files)-3)
		}
		fmt.Printf("\n\n")

		if !promptYesNo("? Initialize brain here?", true) {
			fmt.Printf("%s Cancelled\n", IconError)
			return nil
		}
	}

	// Use template or interactive config
	var config BrainConfig
	if template != "" {
		config = getTemplateConfig(template)
		fmt.Printf("i Using template: %s\n", template)
	} else {
		config = promptForConfig(filepath.Base(cwd))
	}

	return initializeBrain(cwd, config)
}

func isBrainInitialized(path string) bool {
	markerPath := filepath.Join(path, ".flip-brain.yaml")
	_, err := os.Stat(markerPath)
	return err == nil
}

func promptYesNo(question string, defaultValue bool) bool {
	reader := bufio.NewReader(os.Stdin)
	defaultStr := "y"
	if !defaultValue {
		defaultStr = "n"
	}

	fmt.Printf("%s (Y/n) [default: %s]: ", question, defaultStr)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" {
		return defaultValue
	}
	return response == "y" || response == "yes"
}

func promptForInput(question, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)
	if defaultValue != "" {
		fmt.Printf("%s [default: %s]: ", question, defaultValue)
	} else {
		fmt.Printf("%s: ", question)
	}

	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response == "" {
		return defaultValue
	}
	return response
}

func promptForConfig(defaultName string) BrainConfig {
	fmt.Printf("\nConfiguration:\n")

	name := promptForInput("Brain name", defaultName)

	fmt.Printf("Type:\n")
	fmt.Printf("  1) personal - Personal knowledge and tasks\n")
	fmt.Printf("  2) work - Professional work and projects\n")
	fmt.Printf("  3) learning - Learning and skill development\n")
	typeChoice := promptForInput("Choose (1/2/3)", "1")

	var brainType, defaultOrg string
	switch typeChoice {
	case "2", "work":
		brainType = "work"
		defaultOrg = "WORK"
	case "3", "learning":
		brainType = "learning"
		defaultOrg = "LEARNING"
	default:
		brainType = "personal"
		defaultOrg = "PERSONAL"
	}

	author := promptForInput("Author name", "Your Name")

	return BrainConfig{
		Name:                name,
		Type:                brainType,
		DefaultOrganization: defaultOrg,
		Author:              author,
	}
}

func getTemplateConfig(template string) BrainConfig {
	switch template {
	case "work":
		return BrainConfig{
			Name:                "Work Brain",
			Type:                "work",
			DefaultOrganization: "WORK",
			Author:              "Your Name",
		}
	case "learning":
		return BrainConfig{
			Name:                "Learning Brain",
			Type:                "learning",
			DefaultOrganization: "LEARNING",
			Author:              "Your Name",
		}
	default: // personal
		return BrainConfig{
			Name:                "Personal Brain",
			Type:                "personal",
			DefaultOrganization: "PERSONAL",
			Author:              "Your Name",
		}
	}
}

func initializeBrain(path string, config BrainConfig) error {
	fmt.Printf("\n> Creating your brain...\n")

	// Convert to brain.Config
	brainConfig := brain.Config{
		Name:                config.Name,
		Type:                config.Type,
		DefaultOrganization: config.DefaultOrganization,
		Author:              config.Author,
	}

	creator := brain.NewCreator()
	if err := creator.CreateWithConfig(path, brainConfig); err != nil {
		return fmt.Errorf("failed to create brain: %w", err)
	}

	fmt.Printf("[OK] Created: definitions/, journal/, meetings/, notes/, tasks/, templates/\n")
	fmt.Printf("[OK] Created: .flip-brain.yaml, .flip.yaml\n")

	// Auto-add to default workspace
	fmt.Println("\n~ Adding brain to default workspace...")
	if err := autoAddBrainToDefault(path, config.Name); err != nil {
		fmt.Printf("%s Warning: Could not add to default workspace: %v\n", IconWarning, err)
	} else {
		fmt.Println("[OK] Brain added to default workspace")
	}

	fmt.Printf("%s Ready! Try: flip create journal\n", IconCheck)

	return nil
}

// autoAddBrainToDefault detects brain type and adds it to the default workspace
func autoAddBrainToDefault(path, name string) error {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(absPath)
	if err != nil {
		return err
	}

	// Create brain entry
	newBrain := Brain{
		Name:        name,
		Path:        absPath,
		Type:        string(detection.Type),
		Description: detection.Description,
	}

	// Add to default workspace
	return addBrainToDefaultWorkspace(newBrain)
}
