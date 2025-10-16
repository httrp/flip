package main

import (
	"fmt"
	"os"

	"github.com/httrp/flip/internal/commands"
	"github.com/spf13/cobra"
)

func main() {
	var theme string
	var rootCmd = &cobra.Command{
		Use:   "flip",
		Short: "Steroids for your 2nd brain",
		Long:  "Flip is an intelligent assistant designed to supercharge personal knowledge management workflows.",
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

	// Add commands
	rootCmd.AddCommand(commands.NewMenuCommand())
	rootCmd.AddCommand(commands.NewStatusCommand())
	rootCmd.AddCommand(commands.NewQuickstartCommand())
	rootCmd.AddCommand(commands.NewIntroCommand())
	rootCmd.AddCommand(commands.NewWorkspaceCommand())
	// Brain command with subcommands (init/scan moved here)
	brainCmd := commands.NewBrainCommand()
	brainCmd.AddCommand(commands.NewInitCommand())
	brainCmd.AddCommand(commands.NewScanCommand())
	rootCmd.AddCommand(brainCmd)
	// Register 'new' command at root level (not as brain subcommand)
	rootCmd.AddCommand(commands.NewNewCommand())

	// Global flags
	rootCmd.PersistentFlags().StringVar(&theme, "theme", "", "icon theme: ascii|emoji|mixed (default: mixed)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
