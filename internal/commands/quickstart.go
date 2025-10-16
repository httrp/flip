package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var langTexts map[string]string

func getText(key string) string {
	if langTexts == nil {
		langTexts = make(map[string]string)
		data, err := ioutil.ReadFile("lang/en.json")
		if err == nil {
			json.Unmarshal(data, &langTexts)
		}
	}
	if val, ok := langTexts[key]; ok {
		return val
	}
	return key
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
	fmt.Println("\n" + getText("welcome_banner"))
	fmt.Println(getText("intro_workspace"))
	fmt.Println(getText("intro_brain"))
	fmt.Println()

	// Step 1: Create workspace
	fmt.Println(getText("step_workspace"))

	validate := func(input string) error {
		if strings.ToLower(input) == "flap" {
			return fmt.Errorf(getText("reserved_flap"))
		}
		return nil
	}

	promptWorkspace := promptui.Prompt{
		Label:    getText("prompt_workspace_name"),
		Default:  "default",
		Validate: validate,
	}

	wsName, err := promptWorkspace.Run()
	if err != nil {
		return fmt.Errorf("workspace prompt failed: %w", err)
	}
	wsName = strings.TrimSpace(wsName)

	home, _ := os.UserHomeDir()
	wsPath := filepath.Join(home, "flap", "workspaces", wsName)

	promptPath := promptui.Prompt{
		Label:   getText("prompt_workspace_path"),
		Default: wsPath,
	}

	wsPath, err = promptPath.Run()
	if err != nil {
		return fmt.Errorf("path prompt failed: %w", err)
	}
	wsPath = strings.TrimSpace(wsPath)

	if err := os.MkdirAll(wsPath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}
	fmt.Printf(getText("workspace_created")+"\n\n", wsName, wsPath)

	// Step 2: Add a brain
	fmt.Println("\n" + getText("step_brain"))

	selectBrainType := promptui.Select{
		Label: getText("brain_options"),
		Items: []string{
			"Create a new brain (knowledge base)",
			"Connect an existing brain (Obsidian, Logseq, Markdown, Dendron, ...)",
		},
	}

	brainTypeIdx, _, err := selectBrainType.Run()
	if err != nil {
		return fmt.Errorf("brain type selection failed: %w", err)
	}

	if brainTypeIdx == 1 {
		// Connect existing brain
		promptBrainPath := promptui.Prompt{
			Label: getText("prompt_existing_brain_path"),
			Validate: func(input string) error {
				if input == "" {
					return fmt.Errorf(getText("no_path_provided"))
				}
				absPath, err := filepath.Abs(input)
				if err != nil {
					return fmt.Errorf(getText("invalid_path"), err)
				}
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					return fmt.Errorf(getText("path_not_exist"), absPath)
				}
				return nil
			},
		}

		brainPathInput, err := promptBrainPath.Run()
		if err != nil {
			return fmt.Errorf("brain path prompt failed: %w", err)
		}

		brainTarget, _ := filepath.Abs(brainPathInput)
		defaultName := filepath.Base(brainTarget)

		promptBrainName := promptui.Prompt{
			Label:   getText("prompt_existing_brain_name"),
			Default: defaultName,
			Validate: func(input string) error {
				if strings.ToLower(input) == "flap" {
					return fmt.Errorf(getText("reserved_flap"))
				}
				return nil
			},
		}

		brainName, err := promptBrainName.Run()
		if err != nil {
			return fmt.Errorf("brain name prompt failed: %w", err)
		}

		// Initialize if not already a brain
		fmt.Printf(getText("initializing_brain")+"\n", brainName, brainTarget)
		if err := runDirectoryInit(brainTarget, "", false); err != nil {
			fmt.Printf("! Warning: %v\n", err)
		}
		fmt.Printf(getText("brain_added")+"\n", brainName, wsName)
		fmt.Println(getText("status_tip"))
		return nil
	}

	// Create new brain
	creativeNames := []string{
		"atlas", "odyssey", "aurora", "echo", "zenith", "soliloquy", "muse", "oracle", "serendipity", "epiphany",
		"noesis", "elysium", "satori", "logos", "cosmos", "paradox", "quasar", "zeitgeist", "sophia", "mythos",
		"[Enter custom name]",
	}

	selectBrainName := promptui.Select{
		Label: getText("choose_brain_name"),
		Items: creativeNames,
		Size:  10,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "▸ {{ . | cyan | bold }}",
			Inactive: "  {{ . }}",
			Selected: "{{ . | green | bold }}",
		},
	}

	idx, brainName, err := selectBrainName.Run()
	if err != nil {
		return fmt.Errorf("brain name selection failed: %w", err)
	}

	// If custom name was selected, prompt for it
	if idx == len(creativeNames)-1 {
		promptCustomName := promptui.Prompt{
			Label:   "Enter custom brain name",
			Default: "my-brain",
			Validate: func(input string) error {
				if strings.ToLower(input) == "flap" {
					return fmt.Errorf(getText("reserved_flap"))
				}
				if input == "" {
					return fmt.Errorf("brain name cannot be empty")
				}
				return nil
			},
		}
		brainName, err = promptCustomName.Run()
		if err != nil {
			return fmt.Errorf("custom name prompt failed: %w", err)
		}
	}

	brainPath := filepath.Join(home, "flap", "brains", brainName)

	promptBrainPath := promptui.Prompt{
		Label:   getText("prompt_brain_path"),
		Default: brainPath,
	}

	brainPath, err = promptBrainPath.Run()
	if err != nil {
		return fmt.Errorf("brain path prompt failed: %w", err)
	}
	brainPath = strings.TrimSpace(brainPath)

	fmt.Printf(getText("creating_brain")+"\n", brainName, brainPath)
	if err := createBrainStructure(brainPath, "flip"); err != nil {
		return fmt.Errorf("failed to create structure: %w", err)
	}
	if err := runDirectoryInit(brainPath, "", true); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}
	fmt.Printf(getText("brain_created")+"\n", brainName, wsName)
	fmt.Println(getText("status_tip"))
	return nil
}
