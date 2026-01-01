package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// TasksVersion is incremented when tasks.json changes
// This allows flip to detect outdated installations
const TasksVersion = "1.0.0"

// VSCodeTasksJSON contains the embedded tasks configuration
// This is the single source of truth for flip's VS Code tasks
var VSCodeTasksJSON = `{
  "version": "2.0.0",
  "_flipTasksVersion": "` + TasksVersion + `",
  "tasks": [
    {
      "label": "Flip: Menu",
      "type": "shell",
      "command": "flip",
      "args": ["menu"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: File Info",
      "type": "shell",
      "command": "flip",
      "args": ["file-info", "${file}"],
      "presentation": {
        "reveal": "always",
        "panel": "shared",
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Journal (Today)",
      "type": "shell",
      "command": "flip",
      "args": ["journal"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: New Task",
      "type": "shell",
      "command": "flip",
      "args": ["task", "new"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Task Done",
      "type": "shell",
      "command": "flip",
      "args": ["task", "done"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Task List",
      "type": "shell",
      "command": "flip",
      "args": ["task", "list"],
      "presentation": {
        "reveal": "always",
        "panel": "shared",
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: New Note",
      "type": "shell",
      "command": "flip",
      "args": ["note"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Quick Note",
      "type": "shell",
      "command": "flip",
      "args": ["quicknote"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Search Notes",
      "type": "shell",
      "command": "flip",
      "args": ["note", "search"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: New Exercise",
      "type": "shell",
      "command": "flip",
      "args": ["exercise", "new"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Track Exercise",
      "type": "shell",
      "command": "flip",
      "args": ["exercise", "track"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated",
        "focus": true,
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: List Exercises",
      "type": "shell",
      "command": "flip",
      "args": ["exercise", "list"],
      "presentation": {
        "reveal": "always",
        "panel": "shared",
        "clear": true
      },
      "problemMatcher": []
    },
    {
      "label": "Flip: Status",
      "type": "shell",
      "command": "flip",
      "args": ["status"],
      "presentation": {
        "reveal": "always",
        "panel": "shared",
        "clear": true
      },
      "problemMatcher": []
    }
  ]
}`

// NewVSCodeCommand creates the vscode command group
func NewVSCodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vscode",
		Short: "VS Code integration",
		Long:  "Install, update, or remove flip tasks for VS Code",
	}

	cmd.AddCommand(newVSCodeInstallCommand())
	cmd.AddCommand(newVSCodeStatusCommand())
	cmd.AddCommand(newVSCodeUninstallCommand())
	cmd.AddCommand(newVSCodeInfoCommand())

	return cmd
}

func newVSCodeInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Output flip info for VS Code extension (JSON)",
		Long:  "Returns JSON with all information needed by the VS Code extension",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVSCodeInfo()
		},
	}
}

func runVSCodeInfo() error {
	config, err := loadWorkspaceConfig()
	if err != nil {
		OutputJSONError("vscode-info", err)
		return nil
	}

	info := VSCodeInfo{
		FlipVersion:     FlipVersion,
		ActiveWorkspace: config.ActiveWorkspace,
		Brains:          []BrainInfo{},
		Commands: []string{
			"journal", "note", "quicknote", "meeting-note",
			"search", "recent", "status", "brain switch",
			"task new", "task done", "task list",
		},
	}

	// Find active workspace and its brains
	for _, ws := range config.Workspaces {
		if ws.Name == config.ActiveWorkspace {
			info.ActiveBrain = ws.DefaultBrain

			for _, b := range ws.Brains {
				brainInfo := BrainInfo{
					Name:   b.Name,
					Path:   b.Path,
					Type:   b.Type,
					Active: b.Name == ws.DefaultBrain,
				}
				info.Brains = append(info.Brains, brainInfo)
			}
			break
		}
	}

	OutputJSONSuccess("vscode-info", info)
	return nil
}

func newVSCodeInstallCommand() *cobra.Command {
	var local bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install flip tasks for VS Code",
		Long: `Install flip tasks to make them available in VS Code.

By default, tasks are installed globally (available in all VS Code windows).
Use --local to install only in the current directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if local {
				return installTasksLocal()
			}
			return installTasksGlobal()
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Install to .vscode/tasks.json in current directory")
	return cmd
}

func newVSCodeStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show VS Code tasks installation status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showVSCodeStatus()
		},
	}
}

func newVSCodeUninstallCommand() *cobra.Command {
	var local bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove flip tasks from VS Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			if local {
				return uninstallTasksLocal()
			}
			return uninstallTasksGlobal()
		},
	}

	cmd.Flags().BoolVar(&local, "local", false, "Remove from .vscode/tasks.json in current directory")
	return cmd
}

// getVSCodeUserTasksPath returns the path to VS Code's user tasks.json
func getVSCodeUserTasksPath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, "Library", "Application Support", "Code", "User")
	case "linux":
		// Check for flatpak VS Code first
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		flatpakPath := filepath.Join(home, ".var", "app", "com.visualstudio.code", "config", "Code", "User")
		if _, err := os.Stat(flatpakPath); err == nil {
			configDir = flatpakPath
		} else {
			// Standard Linux path
			configDir = filepath.Join(home, ".config", "Code", "User")
		}
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(appData, "Code", "User")
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return filepath.Join(configDir, "tasks.json"), nil
}

func installTasksGlobal() error {
	tasksPath, err := getVSCodeUserTasksPath()
	if err != nil {
		return fmt.Errorf("could not determine VS Code config path: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(tasksPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	// Check if file exists and merge if needed
	if _, err := os.Stat(tasksPath); err == nil {
		return mergeTasksFile(tasksPath, true)
	}

	// Fresh install
	if err := os.WriteFile(tasksPath, []byte(VSCodeTasksJSON), 0644); err != nil {
		return fmt.Errorf("could not write tasks.json: %w", err)
	}

	fmt.Println("✅ Flip tasks installed globally")
	fmt.Printf("   Location: %s\n", tasksPath)
	fmt.Println("   Tasks available in all VS Code windows")
	fmt.Println()
	fmt.Println("💡 Use Ctrl+Shift+P → 'Tasks: Run Task' → 'Flip: ...'")

	return nil
}

func installTasksLocal() error {
	tasksPath := filepath.Join(".vscode", "tasks.json")

	// Ensure .vscode directory exists
	if err := os.MkdirAll(".vscode", 0755); err != nil {
		return fmt.Errorf("could not create .vscode directory: %w", err)
	}

	// Check if file exists and merge if needed
	if _, err := os.Stat(tasksPath); err == nil {
		return mergeTasksFile(tasksPath, false)
	}

	// Fresh install
	if err := os.WriteFile(tasksPath, []byte(VSCodeTasksJSON), 0644); err != nil {
		return fmt.Errorf("could not write tasks.json: %w", err)
	}

	cwd, _ := os.Getwd()
	fmt.Println("✅ Flip tasks installed locally")
	fmt.Printf("   Location: %s\n", filepath.Join(cwd, tasksPath))
	fmt.Println("   Tasks available when this folder is open in VS Code")

	return nil
}

// mergeTasksFile merges flip tasks into an existing tasks.json
func mergeTasksFile(tasksPath string, isGlobal bool) error {
	// Read existing file
	existingData, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("could not read existing tasks.json: %w", err)
	}

	// Parse existing tasks
	var existing map[string]interface{}
	if err := json.Unmarshal(existingData, &existing); err != nil {
		return fmt.Errorf("could not parse existing tasks.json: %w", err)
	}

	// Parse flip tasks
	var flipTasks map[string]interface{}
	if err := json.Unmarshal([]byte(VSCodeTasksJSON), &flipTasks); err != nil {
		return fmt.Errorf("could not parse flip tasks: %w", err)
	}

	// Get existing tasks array
	existingTasksRaw, ok := existing["tasks"].([]interface{})
	if !ok {
		existingTasksRaw = []interface{}{}
	}

	// Get flip tasks array
	flipTasksRaw, ok := flipTasks["tasks"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid flip tasks configuration")
	}

	// Remove existing flip tasks (those starting with "Flip:")
	var filteredTasks []interface{}
	for _, task := range existingTasksRaw {
		taskMap, ok := task.(map[string]interface{})
		if !ok {
			continue
		}
		label, ok := taskMap["label"].(string)
		if !ok || !strings.HasPrefix(label, "Flip:") {
			filteredTasks = append(filteredTasks, task)
		}
	}

	// Add flip tasks
	mergedTasks := append(filteredTasks, flipTasksRaw...)

	// Update the map
	existing["tasks"] = mergedTasks
	existing["_flipTasksVersion"] = TasksVersion

	// Write back
	output, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal merged tasks: %w", err)
	}

	if err := os.WriteFile(tasksPath, output, 0644); err != nil {
		return fmt.Errorf("could not write merged tasks.json: %w", err)
	}

	location := "globally"
	if !isGlobal {
		location = "locally"
	}
	fmt.Printf("✅ Flip tasks updated %s\n", location)
	fmt.Printf("   Location: %s\n", tasksPath)
	fmt.Printf("   Version: %s\n", TasksVersion)

	return nil
}

func uninstallTasksGlobal() error {
	tasksPath, err := getVSCodeUserTasksPath()
	if err != nil {
		return fmt.Errorf("could not determine VS Code config path: %w", err)
	}

	return removeFlipTasks(tasksPath, true)
}

func uninstallTasksLocal() error {
	return removeFlipTasks(filepath.Join(".vscode", "tasks.json"), false)
}

func removeFlipTasks(tasksPath string, isGlobal bool) error {
	// Check if file exists
	if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
		fmt.Println("ℹ️  No tasks.json found, nothing to uninstall")
		return nil
	}

	// Read existing file
	existingData, err := os.ReadFile(tasksPath)
	if err != nil {
		return fmt.Errorf("could not read tasks.json: %w", err)
	}

	// Parse existing tasks
	var existing map[string]interface{}
	if err := json.Unmarshal(existingData, &existing); err != nil {
		return fmt.Errorf("could not parse tasks.json: %w", err)
	}

	// Get existing tasks array
	existingTasksRaw, ok := existing["tasks"].([]interface{})
	if !ok {
		fmt.Println("ℹ️  No tasks found in file")
		return nil
	}

	// Remove flip tasks
	var filteredTasks []interface{}
	removedCount := 0
	for _, task := range existingTasksRaw {
		taskMap, ok := task.(map[string]interface{})
		if !ok {
			continue
		}
		label, ok := taskMap["label"].(string)
		if ok && strings.HasPrefix(label, "Flip:") {
			removedCount++
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}

	if removedCount == 0 {
		fmt.Println("ℹ️  No Flip tasks found to remove")
		return nil
	}

	// Remove flip version marker
	delete(existing, "_flipTasksVersion")

	// Update the map
	existing["tasks"] = filteredTasks

	// Write back
	output, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("could not marshal tasks: %w", err)
	}

	if err := os.WriteFile(tasksPath, output, 0644); err != nil {
		return fmt.Errorf("could not write tasks.json: %w", err)
	}

	location := "globally"
	if !isGlobal {
		location = "locally"
	}
	fmt.Printf("✅ Removed %d Flip tasks %s\n", removedCount, location)

	return nil
}

func showVSCodeStatus() error {
	fmt.Println("VS Code Tasks Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Check global installation
	globalPath, err := getVSCodeUserTasksPath()
	if err != nil {
		fmt.Printf("⚠️  Could not determine global path: %v\n", err)
	} else {
		globalVersion := getInstalledVersion(globalPath)
		if globalVersion != "" {
			if globalVersion == TasksVersion {
				fmt.Printf("✅ Global: %s (v%s - current)\n", globalPath, globalVersion)
			} else {
				fmt.Printf("⚠️  Global: %s (v%s - update available: v%s)\n", globalPath, globalVersion, TasksVersion)
			}
		} else {
			fmt.Printf("❌ Global: not installed\n")
		}
	}

	// Check local installation
	localPath := filepath.Join(".vscode", "tasks.json")
	localVersion := getInstalledVersion(localPath)
	if localVersion != "" {
		cwd, _ := os.Getwd()
		if localVersion == TasksVersion {
			fmt.Printf("✅ Local:  %s (v%s - current)\n", filepath.Join(cwd, localPath), localVersion)
		} else {
			fmt.Printf("⚠️  Local:  %s (v%s - update available: v%s)\n", filepath.Join(cwd, localPath), localVersion, TasksVersion)
		}
	} else {
		fmt.Printf("❌ Local:  not installed\n")
	}

	fmt.Println()
	fmt.Printf("📦 Bundled version: v%s\n", TasksVersion)

	// Check if running in VS Code
	if IsRunningInVSCode() {
		fmt.Println("🖥️  Running in VS Code terminal")
	}

	return nil
}

// getInstalledVersion reads the _flipTasksVersion from a tasks.json file
func getInstalledVersion(tasksPath string) string {
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return ""
	}

	var tasks map[string]interface{}
	if err := json.Unmarshal(data, &tasks); err != nil {
		return ""
	}

	// Check for flip version marker
	if version, ok := tasks["_flipTasksVersion"].(string); ok {
		return version
	}

	// Check if there are any Flip tasks (old installation without version)
	if tasksRaw, ok := tasks["tasks"].([]interface{}); ok {
		for _, task := range tasksRaw {
			if taskMap, ok := task.(map[string]interface{}); ok {
				if label, ok := taskMap["label"].(string); ok && strings.HasPrefix(label, "Flip:") {
					return "unknown"
				}
			}
		}
	}

	return ""
}

// IsRunningInVSCode returns true if flip is running in a VS Code terminal
func IsRunningInVSCode() bool {
	// VS Code sets TERM_PROGRAM=vscode
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return true
	}
	// Also check for VSCODE_* environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "VSCODE_") {
			return true
		}
	}
	return false
}

// CheckVSCodeTasksUpdate checks if tasks need updating and shows a hint
// Call this from main.go during startup
func CheckVSCodeTasksUpdate() {
	if !IsRunningInVSCode() {
		return
	}

	// Check global installation
	globalPath, err := getVSCodeUserTasksPath()
	if err != nil {
		return
	}

	globalVersion := getInstalledVersion(globalPath)
	
	// No installation - suggest installing
	if globalVersion == "" {
		fmt.Println("💡 Tip: Run 'flip vscode install' to enable VS Code tasks")
		fmt.Println()
		return
	}

	// Outdated installation - suggest updating
	if globalVersion != TasksVersion && globalVersion != "unknown" {
		fmt.Printf("💡 Tip: Flip tasks outdated (v%s → v%s). Run 'flip vscode install' to update\n", globalVersion, TasksVersion)
		fmt.Println()
	}
}
