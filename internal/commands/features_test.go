package commands

import (
	"os"
	"testing"
)

func TestIsEnabled_CoreFeatures(t *testing.T) {
	// Core features should always be enabled
	coreFeatures := []Feature{FeatureCore, FeatureGit, FeatureHealth}

	for _, f := range coreFeatures {
		if !IsEnabled(f) {
			t.Errorf("Core feature %s should be enabled by default", f)
		}
	}
}

func TestIsEnabled_OptionalFeatures(t *testing.T) {
	// Optional features should be enabled by default
	optionalFeatures := []Feature{
		FeatureTasks,
		FeatureExercises,
		FeatureTemplates,
		FeatureVSCode,
		FeatureDefinitions,
		FeatureMeetings,
		FeatureMigration,
		FeatureAI,
	}

	for _, f := range optionalFeatures {
		if !IsEnabled(f) {
			t.Errorf("Optional feature %s should be enabled by default", f)
		}
	}
}

func TestDisableFeature_CoreCannotBeDisabled(t *testing.T) {
	// Try to disable core feature
	DisableFeature(FeatureCore)

	// Should still be enabled
	if !IsEnabled(FeatureCore) {
		t.Error("Core feature should not be disableable")
	}
}

func TestDisableFeature_OptionalCanBeDisabled(t *testing.T) {
	// Disable an optional feature
	DisableFeature(FeatureExercises)
	defer EnableFeature(FeatureExercises) // Cleanup

	if IsEnabled(FeatureExercises) {
		t.Error("Optional feature should be disableable")
	}
}

func TestEnableFeature(t *testing.T) {
	// Disable then re-enable
	DisableFeature(FeatureTasks)
	EnableFeature(FeatureTasks)

	if !IsEnabled(FeatureTasks) {
		t.Error("Feature should be re-enableable")
	}
}

func TestIsEnabled_EnvironmentOverride(t *testing.T) {
	// Set environment variable to disable a feature
	os.Setenv("FLIP_FEATURE_EXERCISES", "0")
	defer os.Unsetenv("FLIP_FEATURE_EXERCISES")

	if IsEnabled(FeatureExercises) {
		t.Error("Environment variable should override feature state")
	}
}

func TestIsEnabled_EnvironmentEnable(t *testing.T) {
	// Disable feature first
	DisableFeature(FeatureExercises)
	defer EnableFeature(FeatureExercises)

	// Set environment variable to enable
	os.Setenv("FLIP_FEATURE_EXERCISES", "true")
	defer os.Unsetenv("FLIP_FEATURE_EXERCISES")

	if !IsEnabled(FeatureExercises) {
		t.Error("Environment variable should enable feature")
	}
}

func TestGetEnabledFeatures(t *testing.T) {
	enabled := GetEnabledFeatures()

	if len(enabled) == 0 {
		t.Error("Should have at least core features enabled")
	}

	// Check that core features are in the list
	hasCore := false
	for _, f := range enabled {
		if f == FeatureCore {
			hasCore = true
			break
		}
	}

	if !hasCore {
		t.Error("Core feature should be in enabled list")
	}
}

func TestGetFeatureInfo(t *testing.T) {
	info := GetFeatureInfo()

	if len(info) == 0 {
		t.Error("Feature info should not be empty")
	}

	// Check that core features are marked as core
	for _, fi := range info {
		if fi.Name == FeatureCore && !fi.IsCore {
			t.Error("Core feature should be marked as IsCore=true")
		}
		if fi.Name == FeatureExercises && fi.IsCore {
			t.Error("Exercises should not be marked as core")
		}
	}
}

func TestConstants(t *testing.T) {
	// Verify constants are set correctly
	if DefaultWorkspaceName != "default" {
		t.Errorf("DefaultWorkspaceName should be 'default', got %s", DefaultWorkspaceName)
	}

	if DefaultBrainConfigFile != ".flip.yaml" {
		t.Errorf("DefaultBrainConfigFile should be '.flip.yaml', got %s", DefaultBrainConfigFile)
	}

	if ExtMarkdown != ".md" {
		t.Errorf("ExtMarkdown should be '.md', got %s", ExtMarkdown)
	}
}

func TestConstantsBrainTypes(t *testing.T) {
	// Test brain type constants
	brainTypes := map[string]string{
		"flip":     BrainTypeFlip,
		"obsidian": BrainTypeObsidian,
		"logseq":   BrainTypeLogseq,
		"dendron":  BrainTypeDendron,
		"foam":     BrainTypeFoam,
		"unknown":  BrainTypeUnknown,
	}

	for expected, actual := range brainTypes {
		if actual != expected {
			t.Errorf("BrainType constant mismatch: expected %s, got %s", expected, actual)
		}
	}
}

func TestConstantsFileExtensions(t *testing.T) {
	// All extensions should start with a dot
	extensions := []string{ExtMarkdown, ExtYAML, ExtJSON}
	for _, ext := range extensions {
		if len(ext) == 0 || ext[0] != '.' {
			t.Errorf("Extension '%s' should start with a dot", ext)
		}
	}
}

func TestConstantsDirectoryNames(t *testing.T) {
	// Directory names should be lowercase and not empty
	dirs := map[string]string{
		"journal":     DirJournal,
		"notes":       DirNotes,
		"meetings":    DirMeetings,
		"tasks":       DirTasks,
		"templates":   DirTemplates,
		"definitions": DirDefinitions,
		"assets":      DirAssets,
	}

	for expected, actual := range dirs {
		if actual != expected {
			t.Errorf("Directory constant mismatch: expected %s, got %s", expected, actual)
		}
	}
}

func TestFeatureEnvVarParsing(t *testing.T) {
	tests := []struct {
		envValue string
		expected bool
	}{
		{"0", false},
		{"false", false},
		{"FALSE", false},
		{"1", true},
		{"true", true},
		{"TRUE", true},
		// Note: "yes" is NOT supported - only "1" and "true"
	}

	feature := FeatureExercises

	for _, tt := range tests {
		t.Run(tt.envValue, func(t *testing.T) {
			// Reset feature state
			EnableFeature(feature)

			os.Setenv("FLIP_FEATURE_EXERCISES", tt.envValue)
			defer os.Unsetenv("FLIP_FEATURE_EXERCISES")

			result := IsEnabled(feature)
			if result != tt.expected {
				t.Errorf("For env value '%s': expected %v, got %v", tt.envValue, tt.expected, result)
			}
		})
	}
}

func TestFeatureEnvVarEmpty(t *testing.T) {
	// Empty env var means use registry default (enabled)
	feature := FeatureExercises
	EnableFeature(feature)

	os.Unsetenv("FLIP_FEATURE_EXERCISES")

	if !IsEnabled(feature) {
		t.Error("Empty env var should use default (enabled)")
	}
}

func TestAllFeaturesHaveDescriptions(t *testing.T) {
	info := GetFeatureInfo()

	for _, fi := range info {
		if fi.Description == "" {
			t.Errorf("Feature %s has no description", fi.Name)
		}
		if len(fi.Description) < 10 {
			t.Errorf("Feature %s description too short: %s", fi.Name, fi.Description)
		}
	}
}
