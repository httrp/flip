package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WorkspaceConfig is the root configuration
type WorkspaceConfig struct {
	Warning         string      `json:"_warning,omitempty"` // Warning message for manual edits
	Version         string      `json:"version"`
	ActiveWorkspace string      `json:"active_workspace"` // Name of currently active workspace
	Workspaces      []Workspace `json:"workspaces"`
}

// Workspace is a collection of related brains
type Workspace struct {
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	DefaultBrain string  `json:"default_brain,omitempty"` // Name of default brain in this workspace
	Brains       []Brain `json:"brains"`
}

// Brain represents a single 2nd brain directory
type Brain struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`        // flip, obsidian, logseq, dendron, etc.
	Description string `json:"description"` // Human-readable type description
}

// Legacy types for migration
type legacyWorkspaceConfig struct {
	Version    string                 `json:"version"`
	Workspaces []legacyWorkspaceEntry `json:"workspaces"`
}

type legacyWorkspaceEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     bool   `json:"default"`
	Active      bool   `json:"active"`
}

// getConfigPath returns the path to the flip configuration directory
func getConfigPath() (string, error) {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	configDir := filepath.Join(configRoot, "flip")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	return filepath.Join(configDir, "workspaces.json"), nil
}

// loadWorkspaceConfig loads the workspace configuration from file
func loadWorkspaceConfig() (*WorkspaceConfig, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &WorkspaceConfig{
			Version:    "2.0",
			Workspaces: []Workspace{},
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Try to parse as new format first
	var config WorkspaceConfig
	if err := json.Unmarshal(data, &config); err == nil && config.Version == "2.0" {
		return &config, nil
	}

	// Try legacy format migration
	var legacy legacyWorkspaceConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Migrate from v1.0 to v2.0
	fmt.Println("> Migrating config from v1.0 to v2.0...")
	newConfig := &WorkspaceConfig{
		Version:    "2.0",
		Workspaces: []Workspace{},
	}

	// Convert each legacy workspace entry to a workspace with one brain
	for _, legacyWS := range legacy.Workspaces {
		ws := Workspace{
			Name:         legacyWS.Name,
			Description:  "",
			DefaultBrain: legacyWS.Name, // The brain has the same name
			Brains: []Brain{
				{
					Name:        legacyWS.Name,
					Path:        legacyWS.Path,
					Type:        legacyWS.Type,
					Description: legacyWS.Description,
				},
			},
		}
		newConfig.Workspaces = append(newConfig.Workspaces, ws)

		// Set active workspace to the first one marked as default
		if legacyWS.Default && newConfig.ActiveWorkspace == "" {
			newConfig.ActiveWorkspace = ws.Name
		}
	}

	// Save migrated config
	if err := saveWorkspaceConfig(newConfig); err != nil {
		fmt.Printf("%s Warning: Could not save migrated config: %v\n", IconWarning, err)
	} else {
		fmt.Printf("%s Config migration complete\n", IconCheck)
	}

	return newConfig, nil
}

// saveWorkspaceConfig saves the workspace configuration to file
func saveWorkspaceConfig(config *WorkspaceConfig) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Add warning message if not present
	if config.Warning == "" {
		config.Warning = "⚠️  DO NOT EDIT MANUALLY! Use 'flip workspace' and 'flip brain' commands. Manual edits can break flip completely!"
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getActiveWorkspace returns the currently active workspace
func getActiveWorkspace() (*Workspace, error) {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return nil, err
	}

	if config.ActiveWorkspace == "" {
		if len(config.Workspaces) > 0 {
			// Default to first workspace if none active
			return &config.Workspaces[0], nil
		}
		return nil, fmt.Errorf("no workspaces configured")
	}

	for i, ws := range config.Workspaces {
		if ws.Name == config.ActiveWorkspace {
			return &config.Workspaces[i], nil
		}
	}

	return nil, fmt.Errorf("active workspace '%s' not found", config.ActiveWorkspace)
}

// getActiveBrain returns the default brain from the active workspace
func getActiveBrain() (*Brain, error) {
	ws, err := getActiveWorkspace()
	if err != nil {
		return nil, err
	}

	if len(ws.Brains) == 0 {
		return nil, fmt.Errorf("no brains in active workspace")
	}

	// Return default brain if set
	if ws.DefaultBrain != "" {
		for i := range ws.Brains {
			if ws.Brains[i].Name == ws.DefaultBrain {
				return &ws.Brains[i], nil
			}
		}
	}

	// Otherwise return first brain
	return &ws.Brains[0], nil
}

// ensureDefaultWorkspace ensures the "default" workspace exists and returns it
func ensureDefaultWorkspace() (*Workspace, error) {
	config, err := loadWorkspaceConfig()
	if err != nil {
		return nil, err
	}

	// Check if default workspace exists
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == "default" {
			return &config.Workspaces[i], nil
		}
	}

	// Create default workspace
	defaultWS := Workspace{
		Name:        "default",
		Description: "All brains known to flip on this machine",
		Brains:      []Brain{},
	}

	config.Workspaces = append(config.Workspaces, defaultWS)

	// Set as active if no other workspace is active
	if config.ActiveWorkspace == "" {
		config.ActiveWorkspace = "default"
	}

	if err := saveWorkspaceConfig(config); err != nil {
		return nil, err
	}

	// Reload and return pointer to stored workspace
	config, err = loadWorkspaceConfig()
	if err != nil {
		return nil, err
	}
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == "default" {
			return &config.Workspaces[i], nil
		}
	}
	return nil, fmt.Errorf("default workspace not found after creation")
}

// addBrainToDefaultWorkspace adds a brain to the default workspace if not already present
func addBrainToDefaultWorkspace(brain Brain) error {
	// Ensure default workspace exists
	if _, err := ensureDefaultWorkspace(); err != nil {
		return err
	}

	// Load config after ensuring default workspace exists
	config, err := loadWorkspaceConfig()
	if err != nil {
		return err
	}

	// Find default workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == "default" {
			// Check if brain already exists (by path)
			for _, b := range config.Workspaces[i].Brains {
				if b.Path == brain.Path {
					// Already exists, update it
					for j := range config.Workspaces[i].Brains {
						if config.Workspaces[i].Brains[j].Path == brain.Path {
							config.Workspaces[i].Brains[j] = brain
						}
					}
					return saveWorkspaceConfig(config)
				}
			}

			// Add new brain
			config.Workspaces[i].Brains = append(config.Workspaces[i].Brains, brain)

			// Set as default brain if it's the first one
			if len(config.Workspaces[i].Brains) == 1 {
				config.Workspaces[i].DefaultBrain = brain.Name
			}

			return saveWorkspaceConfig(config)
		}
	}

	return fmt.Errorf("default workspace not found")
}

// normalizePathName converts a name to a filesystem-safe version
// Removes spaces, special characters, converts to lowercase
func normalizePathName(name string) string {
	// Convert to lowercase
	normalized := strings.ToLower(name)

	// Replace spaces and underscores with hyphens
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")

	// Remove special characters, keep only alphanumeric and hyphens
	var result strings.Builder
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	// Remove consecutive hyphens
	finalResult := result.String()
	for strings.Contains(finalResult, "--") {
		finalResult = strings.ReplaceAll(finalResult, "--", "-")
	}

	// Trim hyphens from start and end
	finalResult = strings.Trim(finalResult, "-")

	return finalResult
}
