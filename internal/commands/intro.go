package commands

import (
	"fmt"

	"github.com/httrp/flip/internal/lang"
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
	// Simply print the intro template - all text is in lang/intro.en.txt
	fmt.Println(lang.GetTemplate("intro"))
	return nil
}
