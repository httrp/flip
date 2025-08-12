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
		Short: "Initialize a new 2nd brain repository",
		Long:  "Creates a new folder structure with templates and configuration for your 2nd brain.",
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
	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", directory, err)
	}

	absPath, err := filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fmt.Printf("🧠 Flip - Initialize your 2nd brain\n\n")
	fmt.Printf("📁 Target directory: %s\n", absPath)

	// Check if directory is empty
	if !force && !isDirEmpty(absPath) {
		fmt.Printf("⚠️  Directory is not empty. Use --force to initialize anyway.\n")
		return fmt.Errorf("directory not empty")
	}

	// Use template or interactive config
	var config BrainConfig
	if template != "" {
		config = getTemplateConfig(template)
		fmt.Printf("📋 Using template: %s\n", template)
	} else {
		config = promptForConfig(filepath.Base(absPath))
	}

	return initializeBrain(absPath, config)
}

func runInteractiveInit(template string, force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Printf("🧠 Flip - Initialize your 2nd brain\n\n")
	fmt.Printf("📁 Current directory: %s\n", cwd)

	// Check for existing brain
	if isBrainInitialized(cwd) {
		fmt.Printf("✅ Brain already initialized in this directory\n")
		return nil
	}

	// Ask where to initialize
	fmt.Printf("\n❓ Where would you like to initialize your 2nd brain?\n")
	fmt.Printf("  1) Here in current directory\n")
	fmt.Printf("  2) Create new folder\n")
	choice := promptForInput("Choose (1/2)", "1")

	if choice == "2" || choice == "new" || choice == "folder" {
		// Create new folder mode
		folderName := promptForInput("📂 Folder name", "my-second-brain")
		basePath := promptForInput("📁 Base path", cwd)

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
		fmt.Printf("📄 Found files: ")
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

		if !promptYesNo("❓ Initialize 2nd brain here?", true) {
			fmt.Printf("❌ Cancelled\n")
			return nil
		}
	}

	// Use template or interactive config
	var config BrainConfig
	if template != "" {
		config = getTemplateConfig(template)
		fmt.Printf("📋 Using template: %s\n", template)
	} else {
		config = promptForConfig(filepath.Base(cwd))
	}

	return initializeBrain(cwd, config)
}

func isDirEmpty(path string) bool {
	files, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(files) == 0
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
	fmt.Printf("\n📝 Configuration:\n")

	name := promptForInput("❓ Brain name", defaultName)

	fmt.Printf("❓ Type:\n")
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

	author := promptForInput("❓ Author name", "Your Name")

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
	fmt.Printf("\n✨ Creating your 2nd brain...\n")

	// Convert to brain.Config
	brainConfig := brain.Config{
		Name:                config.Name,
		Type:                config.Type,
		DefaultOrganization: config.DefaultOrganization,
		Author:              config.Author,
	}

	creator := brain.NewCreator()
	if err := creator.CreateWithConfig(path, brainConfig); err != nil {
		return fmt.Errorf("failed to create 2nd brain: %w", err)
	}

	fmt.Printf("📁 Created: definitions/, journal/, meetings/, notes/, tasks/, templates/\n")
	fmt.Printf("📄 Created: .flip-brain.yaml, .flip.yaml\n")
	fmt.Printf("✅ Ready! Try: flip create journal\n")

	return nil
}
