package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCommand shows CLI version information.
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show flip version",
		Long:  "Display the current flip CLI version and bundled VS Code extension version.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("flip %s\n", FlipVersion)
			fmt.Printf("extension %s\n", ExtensionVersion)
		},
	}
}
