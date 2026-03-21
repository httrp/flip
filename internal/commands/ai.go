package commands

// ai.go - AI Integration Commands
//
// Provides AI-powered features for flip:
//   - flip ai summarize: Summarize notes about a topic from your brains
//   - flip ai research: Create a research note on a topic using AI
//   - flip ai improve: Improve/rewrite a note with AI assistance
//   - flip ai status: Check AI provider status
//
// Split into:
//   ai_status.go    - status + models commands
//   ai_setup.go     - setup, config, key management
//   ai_summarize.go - summarize command
//   ai_research.go  - research command
//   ai_improve.go   - improve command
//   ai_helpers.go   - shared types and utilities

import (
	"github.com/spf13/cobra"
)

// NewAICommand creates the AI command group
func NewAICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI-powered features",
		Long: `AI-powered features for note-taking and knowledge management.

flip uses AI to help you:
  - Summarize your notes about a topic
  - Research and create new notes
  - Improve existing notes

Configuration:
  FLIP_AI_PROVIDER=ollama|openai|anthropic|groq|mistral|azure
  OLLAMA_HOST=http://localhost:11434 (for Ollama)
  OPENAI_API_KEY=sk-... (for OpenAI)
  ANTHROPIC_API_KEY=sk-ant-... (for Anthropic)
  GROQ_API_KEY=gsk_... (for Groq - fast & free)
  MISTRAL_API_KEY=... (for Mistral)`,
	}

	cmd.AddCommand(newAISummarizeCommand())
	cmd.AddCommand(newAIResearchCommand())
	cmd.AddCommand(newAIImproveCommand())
	cmd.AddCommand(newAIStatusCommand())
	cmd.AddCommand(newAIModelsCommand())
	cmd.AddCommand(newAISetupCommand())
	cmd.AddCommand(newAIRenewKeyCommand())

	return cmd
}
