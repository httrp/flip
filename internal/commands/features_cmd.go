package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NewFeaturesCommand creates the features management command
func NewFeaturesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "features",
		Short: "List and manage feature modules",
		Long: `Show which features are enabled or disabled.

Features can be toggled via environment variables:
  FLIP_FEATURE_<NAME>=0  Disable a feature
  FLIP_FEATURE_<NAME>=1  Enable a feature

Example:
  FLIP_FEATURE_EXERCISES=0 flip menu   # Run without exercises feature`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFeaturesList()
		},
	}

	cmd.AddCommand(newFeaturesListCommand())
	cmd.AddCommand(newFeaturesEnableCommand())
	cmd.AddCommand(newFeaturesDisableCommand())

	return cmd
}

func newFeaturesListCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all features and their status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonOutput {
				return runFeaturesListJSON()
			}
			return runFeaturesList()
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func newFeaturesEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <feature>",
		Short: "Show how to enable a feature",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feature := Feature(strings.ToLower(args[0]))
			return showFeatureEnableHelp(feature, true)
		},
	}
}

func newFeaturesDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <feature>",
		Short: "Show how to disable a feature",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			feature := Feature(strings.ToLower(args[0]))
			return showFeatureEnableHelp(feature, false)
		},
	}
}

func runFeaturesList() error {
	fmt.Println()
	fmt.Println("Flip Feature Modules")

	// Core features
	fmt.Println("🔒 Core Features (always enabled)")
	fmt.Println("─────────────────────────────────")
	for _, fi := range GetFeatureInfo() {
		if fi.IsCore {
			fmt.Printf("  ✓ %-15s %s\n", fi.Name, fi.Description)
		}
	}

	fmt.Println()

	// Optional features
	fmt.Println("⚡ Optional Features")
	fmt.Println("─────────────────────────────────")
	for _, fi := range GetFeatureInfo() {
		if !fi.IsCore {
			status := "✓"
			if !fi.Enabled {
				status = "○"
			}
			envVar := "FLIP_FEATURE_" + strings.ToUpper(string(fi.Name))
			envStatus := ""
			if os.Getenv(envVar) != "" {
				envStatus = fmt.Sprintf(" (env: %s)", os.Getenv(envVar))
			}
			fmt.Printf("  %s %-15s %s%s\n", status, fi.Name, fi.Description, envStatus)
		}
	}

	fmt.Println()
	fmt.Println("─────────────────────────────────")
	fmt.Println("Toggle features with environment variables:")
	fmt.Println("  FLIP_FEATURE_EXERCISES=0 flip menu   # Disable exercises")
	fmt.Println("  FLIP_FEATURE_TASKS=0 flip status     # Disable tasks")
	fmt.Println()

	return nil
}

func runFeaturesListJSON() error {
	fmt.Println("[")
	features := GetFeatureInfo()
	for i, fi := range features {
		comma := ","
		if i == len(features)-1 {
			comma = ""
		}
		fmt.Printf(`  {"name": "%s", "description": "%s", "core": %t, "enabled": %t}%s`+"\n",
			fi.Name, fi.Description, fi.IsCore, fi.Enabled, comma)
	}
	fmt.Println("]")
	return nil
}

func showFeatureEnableHelp(feature Feature, enable bool) error {
	// Find feature info
	var found *FeatureInfo
	for _, fi := range GetFeatureInfo() {
		if fi.Name == feature {
			found = &fi
			break
		}
	}

	if found == nil {
		fmt.Printf("❌ Unknown feature: %s\n\n", feature)
		fmt.Println("Available features:")
		for _, fi := range GetFeatureInfo() {
			if !fi.IsCore {
				fmt.Printf("  - %s\n", fi.Name)
			}
		}
		return nil
	}

	if found.IsCore {
		fmt.Printf("ℹ️  '%s' is a core feature and cannot be %s\n", feature, map[bool]string{true: "disabled", false: "enabled"}[!enable])
		return nil
	}

	envVar := "FLIP_FEATURE_" + strings.ToUpper(string(feature))
	value := "0"
	action := "disable"
	if enable {
		value = "1"
		action = "enable"
	}

	fmt.Println()
	fmt.Printf("To %s '%s':\n", action, feature)
	fmt.Println()
	fmt.Println("  # For a single command:")
	fmt.Printf("  %s=%s flip menu\n", envVar, value)
	fmt.Println()
	fmt.Println("  # Permanently (add to ~/.zshrc or ~/.bashrc):")
	fmt.Printf("  export %s=%s\n", envVar, value)
	fmt.Println()

	return nil
}

// RegisterFeatureCommands conditionally registers commands based on feature flags
// Call this from main.go to enable/disable command registration
func RegisterFeatureCommands(rootCmd *cobra.Command) {
	// Always register core commands
	rootCmd.AddCommand(NewMenuV2Command()) // Primary bubbletea-based menu
	rootCmd.AddCommand(NewStatusCommand())
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewQuickstartCommand())
	rootCmd.AddCommand(NewIntroCommand())
	rootCmd.AddCommand(NewWorkspaceCommand())
	rootCmd.AddCommand(NewFeaturesCommand()) // Always available

	// Brain command with core subcommands
	brainCmd := NewBrainCommand()
	brainCmd.AddCommand(NewInitCommand())
	brainCmd.AddCommand(NewScanCommand())
	brainCmd.AddCommand(NewRecentCommand())
	brainCmd.AddCommand(NewSearchCommand())
	rootCmd.AddCommand(brainCmd)

	// Conditionally register optional features
	if IsEnabled(FeatureTasks) {
		rootCmd.AddCommand(NewTaskCommand())
	}

	if IsEnabled(FeatureExercises) {
		rootCmd.AddCommand(NewExerciseCommand())
	}

	if IsEnabled(FeatureTemplates) {
		rootCmd.AddCommand(NewTemplateCommand())
		rootCmd.AddCommand(NewMediaCommand()) // Media is part of templates/content
		rootCmd.AddCommand(NewExportCommand())
	}

	if IsEnabled(FeatureVSCode) {
		rootCmd.AddCommand(NewVSCodeCommand())
		rootCmd.AddCommand(NewFileInfoCommand())
	}

	if IsEnabled(FeatureDefinitions) {
		rootCmd.AddCommand(NewDefinitionsCommand())
	}

	if IsEnabled(FeatureMeetings) {
		rootCmd.AddCommand(NewMeetingCommand())
		rootCmd.AddCommand(NewNewCommand())
		rootCmd.AddCommand(NewNoteCommand())
		rootCmd.AddCommand(NewQuicknoteCommand())
		rootCmd.AddCommand(NewJournalCommand())
	}

	if IsEnabled(FeatureAI) {
		rootCmd.AddCommand(NewAICommand())
	}
}
