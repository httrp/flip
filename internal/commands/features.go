package commands

// features.go - Feature Module System
//
// Flip uses a modular architecture where features can be enabled/disabled.
// This allows for lean releases and gradual feature rollout.
//
// Usage:
//   if features.IsEnabled(FeatureExercises) {
//       // register exercise commands
//   }

import (
	"os"
	"strings"
)

// Feature represents a toggleable feature module
type Feature string

// Available features
const (
	// Core features (always enabled)
	FeatureCore   Feature = "core"   // Brain, workspace, status
	FeatureGit    Feature = "git"    // Git integration
	FeatureHealth Feature = "health" // Health checks

	// Optional features (can be disabled)
	FeatureTasks       Feature = "tasks"       // Task management
	FeatureExercises   Feature = "exercises"   // Exercise tracking
	FeatureTemplates   Feature = "templates"   // Template system
	FeatureVSCode      Feature = "vscode"      // VS Code integration
	FeatureDefinitions Feature = "definitions" // Org/Person definitions
	FeatureMeetings    Feature = "meetings"    // Meeting notes
	FeatureMigration   Feature = "migration"   // Brain migration tools
	FeatureAI          Feature = "ai"          // AI integration (summarize, research)
)

// featureRegistry holds the enabled state of each feature
var featureRegistry = map[Feature]bool{
	// Core - always enabled
	FeatureCore:   true,
	FeatureGit:    true,
	FeatureHealth: true,

	// Optional - enabled by default, can be disabled
	FeatureTasks:       true,
	FeatureExercises:   true,
	FeatureTemplates:   true,
	FeatureVSCode:      true,
	FeatureDefinitions: true,
	FeatureMeetings:    true,
	FeatureMigration:   true,
	FeatureAI:          true, // AI enabled by default (zero-config with Ollama)
}

// coreFeatures cannot be disabled
var coreFeatures = map[Feature]bool{
	FeatureCore:   true,
	FeatureGit:    true,
	FeatureHealth: true,
}

// IsEnabled returns true if the feature is enabled
func IsEnabled(f Feature) bool {
	// Check environment override first
	envKey := "FLIP_FEATURE_" + strings.ToUpper(string(f))
	if envVal := os.Getenv(envKey); envVal != "" {
		return envVal == "1" || strings.ToLower(envVal) == "true"
	}

	enabled, exists := featureRegistry[f]
	return exists && enabled
}

// EnableFeature enables a feature (for testing or runtime config)
func EnableFeature(f Feature) {
	featureRegistry[f] = true
}

// DisableFeature disables a feature (core features cannot be disabled)
func DisableFeature(f Feature) {
	if coreFeatures[f] {
		return // Cannot disable core features
	}
	featureRegistry[f] = false
}

// GetEnabledFeatures returns a list of all enabled features
func GetEnabledFeatures() []Feature {
	var enabled []Feature
	for f, isEnabled := range featureRegistry {
		if isEnabled {
			enabled = append(enabled, f)
		}
	}
	return enabled
}

// GetDisabledFeatures returns a list of all disabled features
func GetDisabledFeatures() []Feature {
	var disabled []Feature
	for f, isEnabled := range featureRegistry {
		if !isEnabled {
			disabled = append(disabled, f)
		}
	}
	return disabled
}

// FeatureInfo provides metadata about a feature
type FeatureInfo struct {
	Name        Feature
	Description string
	IsCore      bool
	Enabled     bool
}

// GetFeatureInfo returns information about all features
func GetFeatureInfo() []FeatureInfo {
	return []FeatureInfo{
		{FeatureCore, "Core brain and workspace management", true, IsEnabled(FeatureCore)},
		{FeatureGit, "Git integration for brains", true, IsEnabled(FeatureGit)},
		{FeatureHealth, "Brain health checks", true, IsEnabled(FeatureHealth)},
		{FeatureTasks, "Task management and tracking", false, IsEnabled(FeatureTasks)},
		{FeatureExercises, "Exercise and practice tracking", false, IsEnabled(FeatureExercises)},
		{FeatureTemplates, "Note and meeting templates", false, IsEnabled(FeatureTemplates)},
		{FeatureVSCode, "VS Code integration", false, IsEnabled(FeatureVSCode)},
		{FeatureDefinitions, "Organization/Person definitions", false, IsEnabled(FeatureDefinitions)},
		{FeatureMeetings, "Meeting note creation", false, IsEnabled(FeatureMeetings)},
		{FeatureMigration, "Brain migration tools", false, IsEnabled(FeatureMigration)},
	}
}

// InitFeaturesFromEnv initializes features from environment variables
// Set FLIP_FEATURE_<NAME>=0 to disable a feature
// Example: FLIP_FEATURE_EXERCISES=0 flip menu
func InitFeaturesFromEnv() {
	for f := range featureRegistry {
		if coreFeatures[f] {
			continue // Skip core features
		}

		envKey := "FLIP_FEATURE_" + strings.ToUpper(string(f))
		if envVal := os.Getenv(envKey); envVal != "" {
			if envVal == "0" || strings.ToLower(envVal) == "false" {
				featureRegistry[f] = false
			} else if envVal == "1" || strings.ToLower(envVal) == "true" {
				featureRegistry[f] = true
			}
		}
	}
}
