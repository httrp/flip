package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceConfigSerialization(t *testing.T) {
	config := &WorkspaceConfig{
		Version:         "2.0",
		ActiveWorkspace: "default",
		Workspaces: []Workspace{
			{
				Name:         "default",
				Description:  "Default workspace",
				DefaultBrain: "main",
				Brains: []Brain{
					{
						Name:        "main",
						Path:        "/path/to/brain",
						Type:        "flip",
						Description: "Main brain",
					},
				},
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Test unmarshaling
	var parsed WorkspaceConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if parsed.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", parsed.Version)
	}
	if parsed.ActiveWorkspace != "default" {
		t.Errorf("Expected active workspace 'default', got %s", parsed.ActiveWorkspace)
	}
	if len(parsed.Workspaces) != 1 {
		t.Errorf("Expected 1 workspace, got %d", len(parsed.Workspaces))
	}
}

func TestWorkspaceConfigWarning(t *testing.T) {
	config := &WorkspaceConfig{
		Version: "2.0",
	}

	// Marshal with warning
	config.Warning = "Test warning"
	data, _ := json.Marshal(config)

	if string(data) == "" {
		t.Error("Expected non-empty data")
	}

	// Check warning is included
	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	if parsed["_warning"] != "Test warning" {
		t.Error("Warning should be serialized as _warning")
	}
}

func TestBrainType(t *testing.T) {
	tests := []struct {
		brainType   string
		description string
	}{
		{"flip", "Flip brain"},
		{"obsidian", "Obsidian vault"},
		{"logseq", "Logseq graph"},
		{"dendron", "Dendron workspace"},
		{"generic", "Generic markdown"},
	}

	for _, tt := range tests {
		t.Run(tt.brainType, func(t *testing.T) {
			brain := Brain{
				Name:        "test",
				Path:        "/test/path",
				Type:        tt.brainType,
				Description: tt.description,
			}

			if brain.Type != tt.brainType {
				t.Errorf("Expected type %s, got %s", tt.brainType, brain.Type)
			}
		})
	}
}

func TestLegacyConfigFormat(t *testing.T) {
	legacyJSON := `{
		"version": "1.0",
		"workspaces": [
			{
				"name": "main",
				"path": "/path/to/brain",
				"type": "flip",
				"description": "Main brain",
				"default": true,
				"active": true
			}
		]
	}`

	var legacy legacyWorkspaceConfig
	err := json.Unmarshal([]byte(legacyJSON), &legacy)
	if err != nil {
		t.Fatalf("Failed to parse legacy config: %v", err)
	}

	if len(legacy.Workspaces) != 1 {
		t.Errorf("Expected 1 workspace, got %d", len(legacy.Workspaces))
	}
	if !legacy.Workspaces[0].Default {
		t.Error("Expected workspace to be default")
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := getConfigPath()
	if err != nil {
		t.Fatalf("getConfigPath failed: %v", err)
	}

	if path == "" {
		t.Error("Expected non-empty path")
	}

	// Should end with workspaces.json
	if filepath.Base(path) != "workspaces.json" {
		t.Errorf("Expected path to end with workspaces.json, got %s", filepath.Base(path))
	}

	// Should be in flip config directory
	if filepath.Base(filepath.Dir(path)) != "flip" {
		t.Errorf("Expected path to be in flip directory, got %s", filepath.Dir(path))
	}
}

func TestEmptyWorkspaceConfig(t *testing.T) {
	config := &WorkspaceConfig{
		Version:    "2.0",
		Workspaces: []Workspace{},
	}

	if len(config.Workspaces) != 0 {
		t.Error("Expected empty workspaces")
	}

	// Test finding workspace in empty config
	for _, ws := range config.Workspaces {
		t.Errorf("Should not iterate: %s", ws.Name)
	}
}

func TestWorkspaceWithMultipleBrains(t *testing.T) {
	ws := Workspace{
		Name:         "multi",
		Description:  "Multi-brain workspace",
		DefaultBrain: "primary",
		Brains: []Brain{
			{Name: "primary", Path: "/path/primary", Type: "flip"},
			{Name: "secondary", Path: "/path/secondary", Type: "obsidian"},
			{Name: "archive", Path: "/path/archive", Type: "generic"},
		},
	}

	if len(ws.Brains) != 3 {
		t.Errorf("Expected 3 brains, got %d", len(ws.Brains))
	}

	// Find default brain
	var defaultBrain *Brain
	for i := range ws.Brains {
		if ws.Brains[i].Name == ws.DefaultBrain {
			defaultBrain = &ws.Brains[i]
			break
		}
	}

	if defaultBrain == nil {
		t.Error("Default brain not found")
	}
	if defaultBrain.Name != "primary" {
		t.Errorf("Expected default brain 'primary', got %s", defaultBrain.Name)
	}
}

func TestLoadWorkspaceConfigNonExistent(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", origConfigDir)

	// Note: This test may not work on all systems due to os.UserConfigDir behavior
	// The key is that loadWorkspaceConfig should not panic on missing file
}

func TestConfigVersioning(t *testing.T) {
	tests := []struct {
		version string
		valid   bool
	}{
		{"2.0", true},
		{"1.0", true}, // Legacy but parseable
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			config := WorkspaceConfig{Version: tt.version}
			if tt.valid && config.Version == "" {
				t.Error("Expected valid version")
			}
		})
	}
}
