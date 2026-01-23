package main

import (
	"fmt"
	"os"

	"github.com/httrp/flip/internal/commands"
	"github.com/spf13/cobra"
)

func main() {
	// Initialize logging from environment variables
	commands.InitLogging()

	// Initialize features from environment variables
	commands.InitFeaturesFromEnv()

	var theme string
	var verbose bool
	var rootCmd = &cobra.Command{
		Use:   "flip",
		Short: "Steroids for your 2nd brain",
		Long:  "Flip is an intelligent assistant designed to supercharge personal knowledge management workflows.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Set debug level if verbose flag is set
			if verbose {
				commands.SetLogLevel(commands.LogLevelDebug)
			}

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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose/debug output")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
