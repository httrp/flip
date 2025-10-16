package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewIntroCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "intro",
		Short: "Introduction to Flip, brains, and workspaces",
		Long:  "A short guided introduction explaining Flip's model and how to get started quickly.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIntro()
		},
	}
}

func runIntro() error {
	fmt.Println("===============================================================")
	fmt.Println("Welcome to Flip")
	fmt.Println("===============================================================")
	fmt.Println()
	fmt.Println("Flip manages external knowledge bases without moving your notes.")
	fmt.Println()
	fmt.Println("Key concepts:")
	fmt.Printf("  %s Brain: A single knowledge base directory (Obsidian, Logseq, Dendron, markdown)\n", IconBrain)
	fmt.Printf("  %s Workspace: A named collection of related brains\n", IconWorkspace)
	fmt.Println("     - The 'default' workspace contains all brains Flip knows about on this machine")
	fmt.Println()
	fmt.Println("Quick start:")
	fmt.Println("  flip quickstart           # Guided onboarding")
	fmt.Println("  flip brain new            # Create a new brain with a compatible structure")
	fmt.Println("  flip brain add <path>     # Connect an existing brain directory")
	fmt.Println("  flip status               # See your workspaces and brains")
	fmt.Println()
	fmt.Println("Tips:")
	fmt.Println("  - Use 'flip workspace create <name>' to organize brains by context (e.g., personal/work)")
	fmt.Println("  - Set a default brain per workspace with 'flip brain set-default <name>'")
	fmt.Println("  - Repair your registry anytime with 'flip workspace repair'")
	fmt.Println()
	return nil
}
