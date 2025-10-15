package commands

import (
	"fmt"
	"os"

	"github.com/httrp/flip/internal/brain"
	"github.com/spf13/cobra"
)

func NewWorkspaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"ws"},
		Short:   "Manage workspace collections",
		Long:    "A workspace is a collection of related 2nd brain directories",
	}

	cmd.AddCommand(newWorkspaceCreateCommand())
	cmd.AddCommand(newWorkspaceListCommand())
	cmd.AddCommand(newWorkspaceSwitchCommand())
	cmd.AddCommand(newWorkspaceRemoveCommand())
	cmd.AddCommand(newWorkspaceRenameCommand())
	cmd.AddCommand(newWorkspaceRepairCommand())

	return cmd
}

func newWorkspaceCreateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceCreate(args[0])
		},
	}
}

func newWorkspaceListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceList()
		},
	}
}

func newWorkspaceSwitchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "switch [name]",
		Short: "Switch to a different workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceSwitch(args[0])
		},
	}
}

func newWorkspaceRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [name]",
		Short: "Remove a workspace (brains stay intact)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceRemove(args[0])
		},
	}
}

func newWorkspaceRenameCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rename [old-name] [new-name]",
		Short: "Rename a workspace",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceRename(args[0], args[1])
		},
	}
}

// Implementation functions

func runWorkspaceCreate(name string) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	// Check if workspace already exists
	for _, ws := range config.Workspaces {
		if ws.Name == name {
			return fmt.Errorf("workspace '%s' already exists", name)
		}
	}

	// Create new workspace
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
		return err
	}

	fmt.Printf("✅ Workspace '%s' created\n", name)
	if config.ActiveWorkspace == name {
		fmt.Printf("🌟 Set as active workspace\n")
	}

	return nil
}

func runWorkspaceList() error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	if len(config.Workspaces) == 0 {
		fmt.Println("No workspaces configured.")
		fmt.Println("Use 'flip workspace create <name>' to create one.")
		return nil
	}

	fmt.Printf("📚 Workspaces:\n\n")

	for _, ws := range config.Workspaces {
		activeMarker := "  "
		if ws.Name == config.ActiveWorkspace {
			activeMarker = "🎯"
		}

		fmt.Printf("%s %s\n", activeMarker, ws.Name)
		if ws.Description != "" {
			fmt.Printf("   %s\n", ws.Description)
		}
		fmt.Printf("   Brains: %d", len(ws.Brains))
		if ws.DefaultBrain != "" {
			fmt.Printf(" (default: %s)", ws.DefaultBrain)
		}
		fmt.Println()

		for _, brain := range ws.Brains {
			marker := "   •"
			if brain.Name == ws.DefaultBrain {
				marker = "   ⭐"
			}

			// Check if path exists
			status := "✅"
			if _, err := os.Stat(brain.Path); os.IsNotExist(err) {
				status = "❌"
			}

			fmt.Printf("%s %s (%s) %s\n", marker, brain.Name, brain.Type, status)
			fmt.Printf("      %s\n", brain.Path)
		}
		fmt.Println()
	}

	return nil
}

func runWorkspaceSwitch(name string) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	// Check if workspace exists
	found := false
	for _, ws := range config.Workspaces {
		if ws.Name == name {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	config.ActiveWorkspace = name

	if err := saveWorkspaceConfig(config); err != nil {
		return err
	}

	fmt.Printf("🎯 Switched to workspace '%s'\n", name)
	return nil
}

func runWorkspaceRemove(name string) error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	// Find and remove workspace
	found := false
	newWorkspaces := make([]Workspace, 0, len(config.Workspaces))

	for _, ws := range config.Workspaces {
		if ws.Name == name {
			found = true
			fmt.Printf("🗑️  Removing workspace '%s' (brains stay intact)\n", ws.Name)
		} else {
			newWorkspaces = append(newWorkspaces, ws)
		}
	}

	if !found {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	config.Workspaces = newWorkspaces

	// If we removed the active workspace, switch to first available
	if config.ActiveWorkspace == name {
		if len(config.Workspaces) > 0 {
			config.ActiveWorkspace = config.Workspaces[0].Name
			fmt.Printf("🎯 Switched to workspace '%s'\n", config.ActiveWorkspace)
		} else {
			config.ActiveWorkspace = ""
		}
	}

	if err := saveWorkspaceConfig(config); err != nil {
		return err
	}

	fmt.Printf("✅ Workspace '%s' removed\n", name)
	return nil
}

func runWorkspaceRename(oldName, newName string) error {
	// Protect default workspace
	if oldName == "default" {
		return fmt.Errorf("cannot rename 'default' workspace - it's reserved")
	}

	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	// Check if new name already exists
	for _, ws := range config.Workspaces {
		if ws.Name == newName {
			return fmt.Errorf("workspace '%s' already exists", newName)
		}
	}

	// Check if new name is reserved
	if newName == "default" {
		return fmt.Errorf("cannot use 'default' as workspace name - it's reserved")
	}

	// Find and rename
	found := false
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == oldName {
			config.Workspaces[i].Name = newName
			found = true

			// Update active workspace if needed
			if config.ActiveWorkspace == oldName {
				config.ActiveWorkspace = newName
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("workspace '%s' not found", oldName)
	}

	if err := saveWorkspaceConfig(config); err != nil {
		return err
	}

	fmt.Printf("[OK] Workspace renamed: '%s' -> '%s'\n", oldName, newName)
	return nil
}

func newWorkspaceRepairCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "repair",
		Short: "Repair all workspaces (validate paths, re-detect brain types)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkspaceRepair()
		},
	}
}

func runWorkspaceRepair() error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	fmt.Println("> Repairing all workspaces...")
	fmt.Println()

	detector := brain.NewDetector()
	totalRepaired := 0
	totalRemoved := 0

	for i := range config.Workspaces {
		ws := &config.Workspaces[i]
		fmt.Printf("* Workspace: %s\n", ws.Name)

		newBrains := make([]Brain, 0, len(ws.Brains))
		repaired := 0
		removed := 0

		for _, b := range ws.Brains {
			// Check if path exists
			if _, err := os.Stat(b.Path); os.IsNotExist(err) {
				fmt.Printf("  [X] Removing '%s' - path no longer exists\n", b.Name)
				removed++
				continue
			}

			// Re-detect type
			detection, err := detector.DetectBrainType(b.Path)
			if err != nil {
				fmt.Printf("  ! Warning: Could not detect type for '%s': %v\n", b.Name, err)
				newBrains = append(newBrains, b)
				continue
			}

			// Update if changed
			if string(detection.Type) != b.Type || detection.Description != b.Description {
				fmt.Printf("  [OK] Updated '%s': %s -> %s\n", b.Name, b.Type, detection.Type)
				b.Type = string(detection.Type)
				b.Description = detection.Description
				repaired++
			}

			newBrains = append(newBrains, b)
		}

		ws.Brains = newBrains
		totalRepaired += repaired
		totalRemoved += removed

		// Fix default brain if it was removed
		if ws.DefaultBrain != "" {
			found := false
			for _, b := range newBrains {
				if b.Name == ws.DefaultBrain {
					found = true
					break
				}
			}
			if !found && len(newBrains) > 0 {
				ws.DefaultBrain = newBrains[0].Name
				fmt.Printf("  # Set new default brain: '%s'\n", newBrains[0].Name)
			} else if !found {
				ws.DefaultBrain = ""
			}
		}

		fmt.Println()
	}

	if err := saveWorkspaceConfig(config); err != nil {
		return err
	}

	fmt.Printf("[OK] Repair complete: %d brains updated, %d removed\n", totalRepaired, totalRemoved)
	return nil
}
