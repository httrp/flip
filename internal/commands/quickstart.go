import (
       "encoding/json"
       "io/ioutil"
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
package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	fmt.Println(getText("welcome_banner"))
	fmt.Println("===============================================================")
	fmt.Println()
	fmt.Println(getText("intro_workspace"))
	fmt.Println()
	fmt.Println(getText("intro_brain"))
	fmt.Println()
	fmt.Println("---------------------------------------------------------------")
	fmt.Println()

	// Step 1: Workspace creation
       fmt.Println(getText("step_workspace"))
       fmt.Printf(getText("prompt_workspace_name"))
       wsName, _ := reader.ReadString('\n')
       wsName = strings.TrimSpace(wsName)
       if wsName == "" {
	       wsName = "default"
       }
       if strings.ToLower(wsName) == "flap" {
	       fmt.Println(getText("reserved_flap"))
	       return nil
       }
       home, _ := os.UserHomeDir()
       wsPath := filepath.Join(home, "flap", "workspaces", wsName)
       fmt.Printf(getText("prompt_workspace_path"), wsPath)
       wsPathInput, _ := reader.ReadString('\n')
       wsPathInput = strings.TrimSpace(wsPathInput)
       if wsPathInput != "" {
	       wsPath = wsPathInput
       }
       if err := os.MkdirAll(wsPath, 0755); err != nil {
	       return fmt.Errorf("failed to create workspace directory: %w", err)
       }
       fmt.Printf(getText("workspace_created")+"\n\n", wsName, wsPath)

	// Step 2: Add a brain
	fmt.Println(getText("step_brain"))
	fmt.Println(getText("brain_options"))
	fmt.Print(getText("choose_option"))
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

       if choice == "2" {
	       // Connect existing brain
	       fmt.Print(getText("prompt_existing_brain_path"))
	       brainPathInput, _ := reader.ReadString('\n')
	       brainPathInput = strings.TrimSpace(brainPathInput)
	       if brainPathInput == "" {
		       fmt.Println(getText("no_path_provided"))
		       return nil
	       }
	       brainTarget, err := filepath.Abs(brainPathInput)
	       if err != nil {
		       return fmt.Errorf(getText("invalid_path"), err)
	       }
	       if _, err := os.Stat(brainTarget); os.IsNotExist(err) {
		       return fmt.Errorf(getText("path_not_exist"), brainTarget)
	       }
	       fmt.Print(getText("prompt_existing_brain_name"))
	       brainName, _ := reader.ReadString('\n')
	       brainName = strings.TrimSpace(brainName)
	       if brainName == "" {
		       brainName = filepath.Base(brainTarget)
	       }
	       if strings.ToLower(brainName) == "flap" {
		       fmt.Println(getText("reserved_flap"))
		       return nil
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
	       "atlas", "odyssey", "aurora", "echo", "zenith", "soliloquy", "muse", "oracle", "serendipity", "epiphany", "noesis", "elysium", "satori", "logos", "cosmos", "paradox", "quasar", "zeitgeist", "sophia", "mythos",
       }
       fmt.Println(getText("choose_brain_name"))
       for i, n := range creativeNames {
	       fmt.Printf(getText("brain_name_option"), i+1, n)
       }
       fmt.Printf(getText("prompt_brain_name"), creativeNames[0])
       brainName, _ := reader.ReadString('\n')
       brainName = strings.TrimSpace(brainName)
       if brainName == "" {
	       brainName = creativeNames[0]
       }
       if strings.ToLower(brainName) == "flap" {
	       fmt.Println(getText("reserved_flap"))
	       return nil
       }
       brainPath := filepath.Join(home, "flap", "brains", brainName)
       fmt.Printf(getText("prompt_brain_path"), brainPath)
       brainPathInput, _ := reader.ReadString('\n')
       brainPathInput = strings.TrimSpace(brainPathInput)
       if brainPathInput != "" {
	       brainPath = brainPathInput
       }
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
