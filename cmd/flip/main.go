package main

import (
	"fmt"
	"os"

	"github.com/httrp/flip/internal/commands"
	"github.com/spf13/cobra"
)

func main() {
	// Initialize features from environment variables
	commands.InitFeaturesFromEnv()

	var theme string
	var rootCmd = &cobra.Command{
		Use:   "flip",
		Short: "Steroids for your 2nd brain",
		Long:  "Flip is an intelligent assistant designed to supercharge personal knowledge management workflows.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Check for VS Code tasks updates (after flags are parsed)
			// Only shows hint if in VS Code, outdated, and not in JSON mode
			if commands.IsEnabled(commands.FeatureVSCode) {
				commands.CheckVSCodeTasksUpdate()
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to mixed theme unless explicitly set
			if theme == "" {
				commands.SetIconTheme("mixed")
			} else {
				commands.SetIconTheme(theme)
			}
			// If no subcommand provided, run interactive menu
			return commands.NewMenuCommand().RunE(cmd, args)
		},
	}

	// Register commands based on enabled features
	commands.RegisterFeatureCommands(rootCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&theme, "theme", "", "icon theme: ascii|emoji|mixed (default: mixed)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
