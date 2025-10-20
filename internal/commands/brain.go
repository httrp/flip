package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/brain"
	"github.com/spf13/cobra"
)

func NewBrainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "brain",
		Short: "Manage brains within the active workspace",
		Long:  "Add, list, and manage individual 2nd brain directories within the active workspace",
	}

	cmd.AddCommand(newBrainAddCommand())
	// Alias: 'flip brain new' calls the same logic as 'flip new brain'
	newCmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new brain (alias)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNewBrain()
		},
	}
	cmd.AddCommand(newCmd)
	cmd.AddCommand(newBrainListCommand())
	cmd.AddCommand(newBrainRemoveCommand())
	cmd.AddCommand(newBrainSetDefaultCommand())
	cmd.AddCommand(newBrainRenameCommand())
	cmd.AddCommand(newBrainRepairCommand())

	return cmd
}

func newBrainAddCommand() *cobra.Command {
	var name string
	var setDefault bool

	cmd := &cobra.Command{
		Use:   "add [path]",
		Short: "Add a brain to the active workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrainAdd(args[0], name, setDefault)
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Custom name for the brain")
	cmd.Flags().BoolVarP(&setDefault, "default", "d", false, "Set as default brain in workspace")

	return cmd
}

func newBrainListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List brains in active workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrainList()
		},
	}
}

func newBrainRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [name]",
		Short: "Remove a brain from the active workspace (files stay intact)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrainRemove(args[0])
		},
	}
}

func newBrainSetDefaultCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set-default [name]",
		Short: "Set the default brain in active workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrainSetDefault(args[0])
		},
	}
}

// Implementation functions

func runBrainAdd(path, name string, setDefault bool) error {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if path exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(absPath)
	if err != nil {
		return fmt.Errorf("failed to detect brain type: %w", err)
	}

	fmt.Printf("%s Detecting brain at: %s\n", IconArrow, absPath)
	fmt.Printf("%s Brain type: %s\n", IconBrain, detection.Description)

	if len(detection.Indicators) > 0 {
		fmt.Println("   Indicators:")
		for _, indicator := range detection.Indicators {
			fmt.Printf("   - %s\n", indicator)
		}
	}

	if !detection.Compatible {
		fmt.Printf("%s This directory is not compatible with flip.\n", IconError)
		fmt.Printf("   Use 'flip init %s --force' to initialize it.\n", absPath)
		return fmt.Errorf("incompatible brain")
	}

	// Generate name if not provided
	if name == "" {
		name = filepath.Base(absPath)
	}

	// Load config and get active workspace
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	ws, err := getActiveWorkspace()
	if err != nil {
		return err
	}

	// Check if brain already exists in this workspace
	for _, b := range ws.Brains {
		if b.Path == absPath {
			fmt.Printf("%s Brain already in workspace: %s\n", IconCheck, b.Name)
			return nil
		}
		if b.Name == name {
			return fmt.Errorf("brain name '%s' already exists in workspace '%s'", name, ws.Name)
		}
	}

	// Add new brain
	newBrain := Brain{
		Name:        name,
		Path:        absPath,
		Type:        string(detection.Type),
		Description: detection.Description,
	}

	// Find workspace in config and add brain
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == ws.Name {
			config.Workspaces[i].Brains = append(config.Workspaces[i].Brains, newBrain)

			// Set as default if requested or if it's the first brain
			if setDefault || len(config.Workspaces[i].Brains) == 1 {
				config.Workspaces[i].DefaultBrain = name
			}
			break
		}
	}

	// Save configuration
	if err := saveWorkspaceConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("%s Brain '%s' added to workspace '%s'\n", IconCheck, name, ws.Name)
	if setDefault || len(ws.Brains) == 0 {
		fmt.Printf("%s Set as default brain\n", IconDefault)
	}

	return nil
}

func runBrainList() error {
	ws, err := getActiveWorkspace()
	if err != nil {
		return err
	}

	fmt.Printf("%s Brains in workspace '%s':\n\n", IconBrain, ws.Name)

	if len(ws.Brains) == 0 {
		fmt.Println("No brains in this workspace.")
		fmt.Println("Use 'flip brain add <path>' to add one.")
		return nil
	}

	for _, brain := range ws.Brains {
		marker := "  "
		if brain.Name == ws.DefaultBrain {
			marker = IconDefault
		}

		fmt.Printf("%s %s\n", marker, brain.Name)
		fmt.Printf("   Type: %s\n", brain.Description)
		fmt.Printf("   Path: %s\n", brain.Path)

		// Check if path still exists
		if _, err := os.Stat(brain.Path); os.IsNotExist(err) {
			fmt.Printf("   %s Path no longer exists\n", IconError)
		} else {
			fmt.Printf("   %s Available\n", IconCheck)
		}
		fmt.Println()
	}

	return nil
}

func runBrainRemove(name string) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	ws, err := getActiveWorkspace()
	if err != nil {
		return err
	}

	// Find and remove brain
	found := false
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == ws.Name {
			newBrains := make([]Brain, 0, len(config.Workspaces[i].Brains))

			for _, b := range config.Workspaces[i].Brains {
				if b.Name == name {
					found = true
					fmt.Printf("%s Removing brain '%s' from workspace '%s'\n", IconWarning, name, ws.Name)
					fmt.Printf("   (Files at %s stay intact)\n", b.Path)
				} else {
					newBrains = append(newBrains, b)
				}
			}

			config.Workspaces[i].Brains = newBrains

			// If we removed the default brain, set a new default
			if config.Workspaces[i].DefaultBrain == name {
				if len(newBrains) > 0 {
					config.Workspaces[i].DefaultBrain = newBrains[0].Name
					fmt.Printf("%s Set '%s' as new default brain\n", IconDefault, newBrains[0].Name)
				} else {
					config.Workspaces[i].DefaultBrain = ""
				}
			}

			break
		}
	}

	if !found {
		return fmt.Errorf("brain '%s' not found in workspace '%s'", name, ws.Name)
	}

	if err := saveWorkspaceConfig(config); err != nil {
		return err
	}

	fmt.Printf("%s Brain '%s' removed\n", IconCheck, name)
	return nil
}

func runBrainSetDefault(name string) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	ws, err := getActiveWorkspace()
	if err != nil {
		return err
	}

	// Check if brain exists
	found := false
	for _, b := range ws.Brains {
		if b.Name == name {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("brain '%s' not found in workspace '%s'", name, ws.Name)
	}

	// Set as default
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == ws.Name {
			config.Workspaces[i].DefaultBrain = name
			break
		}
	}

	if err := saveWorkspaceConfig(config); err != nil {
		return err
	}

	fmt.Printf("%s '%s' set as default brain in workspace '%s'\n", IconDefault, name, ws.Name)
	return nil
}

func newBrainRenameCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rename [old-name] [new-name]",
		Short: "Rename a brain in the active workspace",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrainRename(args[0], args[1])
		},
	}
}

func runBrainRename(oldName, newName string) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	ws, err := getActiveWorkspace()
	if err != nil {
		return err
	}

	// Check if new name already exists
	for _, b := range ws.Brains {
		if b.Name == newName {
			return fmt.Errorf("brain '%s' already exists in workspace '%s'", newName, ws.Name)
		}
	}

	// Find the brain to get its path
	var brainPath string
	found := false
	
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == ws.Name {
			for j := range config.Workspaces[i].Brains {
				if config.Workspaces[i].Brains[j].Name == oldName {
					brainPath = config.Workspaces[i].Brains[j].Path
					found = true
					break
				}
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("brain '%s' not found in workspace '%s'", oldName, ws.Name)
	}

	// Warning: This will update the name across all workspaces
	fmt.Printf("\n%s Warning: This will update the brain name in ALL workspaces that reference this brain.\n", IconWarning)
	
	// Check if brain is referenced in other workspaces
	otherWorkspaces := []string{}
	for _, w := range config.Workspaces {
		if w.Name != ws.Name {
			for _, b := range w.Brains {
				if b.Path == brainPath {
					otherWorkspaces = append(otherWorkspaces, w.Name)
					break
				}
			}
		}
	}
	
	if len(otherWorkspaces) > 0 {
		fmt.Printf("   Brain is also referenced in: %s\n", strings.Join(otherWorkspaces, ", "))
	}
	fmt.Println()

	// Update name in all workspaces that reference this brain
	for i := range config.Workspaces {
		for j := range config.Workspaces[i].Brains {
			if config.Workspaces[i].Brains[j].Path == brainPath {
				config.Workspaces[i].Brains[j].Name = newName
				
				// Update default brain reference if needed
				if config.Workspaces[i].DefaultBrain == oldName {
					config.Workspaces[i].DefaultBrain = newName
				}
			}
		}
	}

	if err := saveWorkspaceConfig(config); err != nil {
		return err
	}

	fmt.Printf("%s Brain renamed in config: '%s' -> '%s'\n", IconCheck, oldName, newName)

	// Check if .flip-brain.yaml exists and update it
	brainConfigPath := filepath.Join(brainPath, ".flip-brain.yaml")
	if _, err := os.Stat(brainConfigPath); err == nil {
		// Read the file
		data, err := os.ReadFile(brainConfigPath)
		if err == nil {
			// Simple string replacement for the name field
			content := string(data)
			oldLine := fmt.Sprintf("  name: \"%s\"", oldName)
			newLine := fmt.Sprintf("  name: \"%s\"", newName)
			
			if strings.Contains(content, oldLine) {
				content = strings.Replace(content, oldLine, newLine, 1)
				if err := os.WriteFile(brainConfigPath, []byte(content), 0644); err != nil {
					fmt.Printf("%s Warning: Could not update .flip-brain.yaml: %v\n", IconWarning, err)
				} else {
					fmt.Printf("%s Updated .flip-brain.yaml\n", IconCheck)
				}
			}
		}
	}

	fmt.Printf("\n%s Brain renamed: '%s' -> '%s'\n", IconCheck, oldName, newName)
	return nil
}

func newBrainRepairCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "repair",
		Short: "Repair brains in active workspace (validate paths, re-detect types)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrainRepair()
		},
	}
}

func runBrainRepair() error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	ws, err := getActiveWorkspace()
	if err != nil {
		return err
	}

	fmt.Printf("> Repairing brains in workspace '%s'...\n\n", ws.Name)

	detector := brain.NewDetector()
	repaired := 0
	removed := 0

	for i := range config.Workspaces {
		if config.Workspaces[i].Name != ws.Name {
			continue
		}

		newBrains := make([]Brain, 0, len(config.Workspaces[i].Brains))

		for _, b := range config.Workspaces[i].Brains {
			// Check if path exists
			if _, err := os.Stat(b.Path); os.IsNotExist(err) {
				fmt.Printf("[X] Removing '%s' - path no longer exists: %s\n", b.Name, b.Path)
				removed++
				continue
			}

			// Re-detect type
			detection, err := detector.DetectBrainType(b.Path)
			if err != nil {
				fmt.Printf("! Warning: Could not detect type for '%s': %v\n", b.Name, err)
				newBrains = append(newBrains, b)
				continue
			}

			// Update if changed
			if string(detection.Type) != b.Type || detection.Description != b.Description {
				fmt.Printf("[OK] Updated '%s': %s -> %s\n", b.Name, b.Type, detection.Type)
				b.Type = string(detection.Type)
				b.Description = detection.Description
				repaired++
			}

			newBrains = append(newBrains, b)
		}

		config.Workspaces[i].Brains = newBrains

		// Fix default brain if it was removed
		if config.Workspaces[i].DefaultBrain != "" {
			found := false
			for _, b := range newBrains {
				if b.Name == config.Workspaces[i].DefaultBrain {
					found = true
					break
				}
			}
			if !found && len(newBrains) > 0 {
				config.Workspaces[i].DefaultBrain = newBrains[0].Name
				fmt.Printf("# Set new default brain: '%s'\n", newBrains[0].Name)
			}
		}

		break
	}

	if err := saveWorkspaceConfig(config); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("[OK] Repair complete: %d updated, %d removed\n", repaired, removed)
	return nil
}
