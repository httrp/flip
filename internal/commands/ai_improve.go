package commands

// ai_improve.go - AI improve command

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/httrp/flip/internal/ai"
	"github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// newAIImproveCommand improves existing notes with AI
func newAIImproveCommand() *cobra.Command {
	var (
		instruction string
		noStream    bool
		inPlace     bool
		model       string
		prompt      string
		promptExtra string
		timeout     int
		prepareOnly bool
	)

	cmd := &cobra.Command{
		Use:          "improve [file]",
		SilenceUsage: true,
		Short:        "Improve a note with AI assistance",
		Long: `Use AI to improve, rewrite, or enhance an existing note.

This command:
1. Reads the content of the specified note
2. Sends it to the AI with your improvement instructions
3. Shows the improved version (optionally saves in place)

Examples:
  flip ai improve notes/my-note.md
  flip ai improve notes/draft.md --instruction "make it more concise"
  flip ai improve notes/rough.md --instruction "fix grammar and improve flow" --in-place`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAIImprove(AIImproveOptions{
				AIBaseOptions: AIBaseOptions{
					ModelOverride:  model,
					PromptOverride: prompt,
					PromptExtra:    promptExtra,
					TimeoutSec:     timeout,
					Stream:         !noStream,
					PrepareOnly:    prepareOnly,
				},
				FilePath:    args[0],
				Instruction: instruction,
				InPlace:     inPlace,
			})
		},
	}

	cmd.Flags().StringVarP(&instruction, "instruction", "i", "", "Specific improvement instructions")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&inPlace, "in-place", false, "Update the file in place")
	cmd.Flags().StringVar(&model, "model", "", "Override model for this request")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt note to apply (path or name)")
	cmd.Flags().StringVar(&prompt, "template", "", "Template note to apply (alias of --prompt)")
	cmd.Flags().StringVar(&promptExtra, "prompt-extra", "", "Additional prompt instructions for this request")
	cmd.Flags().StringVar(&promptExtra, "run-notes", "", "Additional run notes for this request (alias of --prompt-extra)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Override AI timeout in seconds")
	cmd.Flags().BoolVar(&prepareOnly, "prepare-only", false, "Return prompts as JSON without calling AI (for Copilot integration)")

	return cmd
}

func runAIImprove(opts AIImproveOptions) error {
	filePath := opts.FilePath
	instruction := opts.Instruction
	modelOverride := opts.ModelOverride
	promptOverride := opts.PromptOverride
	promptExtra := opts.PromptExtra
	timeoutSec := opts.TimeoutSec
	stream := opts.Stream
	inPlace := opts.InPlace
	prepareOnly := opts.PrepareOnly

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	promptContent := ""
	brainInfo, brainType := resolveBrainForFile(filePath)
	promptContent, _, err = resolvePromptNoteForBrain(brainInfo, brainType, promptOverride, "", "", false, promptOverride == "")
	if err != nil {
		return err
	}

	// Get instruction interactively if not provided
	if instruction == "" {
		instruction, err = ui.RunInput("How should the note be improved?", "", "improve clarity and structure", nil)
		if err != nil {
			return err
		}
	}

	if !prepareOnly {
		fmt.Printf("\n📝 Improving: %s\n", filePath)
		fmt.Printf("💡 Instruction: %s\n\n", instruction)
	}

	// Create AI client (allow per-request timeout override) — skip for prepare-only
	cfg := ai.LoadConfig()
	if timeoutSec > 0 {
		cfg.Timeout = timeoutSec
	}

	var client *ai.Client
	var effectiveModel string
	ctx := context.Background()

	if prepareOnly {
		effectiveModel = modelOverride
	} else {
		var cErr error
		client, cErr = ai.NewClientWithConfig(cfg)
		if cErr != nil {
			return fmt.Errorf("failed to create AI client: %w", cErr)
		}
		var mErr error
		effectiveModel, mErr = resolveAIModelForRequest(ctx, client, cfg, modelOverride)
		if mErr != nil {
			return mErr
		}
	}

	// Build the prompt
	systemPrompt := `You are a skilled editor helping to improve markdown notes.
When given a note and improvement instructions:
- Apply the requested improvements
- Preserve the overall structure and meaning
- Keep any existing frontmatter (YAML between --- markers)
- Maintain markdown formatting
- Output only the improved note, nothing else`

	systemPrompt = applyPromptStack(systemPrompt, promptContent, promptExtra)

	userPrompt := fmt.Sprintf("Please improve the following note according to this instruction: %s\n\n---\n\n%s", instruction, string(content))

	// --prepare-only: return prompts + metadata as JSON so the extension can use Copilot
	if prepareOnly {
		result := map[string]interface{}{
			"system_prompt": systemPrompt,
			"user_prompt":   userPrompt,
			"file_path":     filePath,
			"in_place":      inPlace,
			"instruction":   instruction,
		}
		if brainInfo != nil {
			result["brain_name"] = brainInfo.Name
			result["brain_path"] = brainInfo.Path
		}
		jsonBytes, jErr := json.MarshalIndent(result, "", "  ")
		if jErr != nil {
			return fmt.Errorf("failed to marshal prepare-only result: %w", jErr)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Model:       effectiveModel,
		Temperature: 0.4,
		MaxTokens:   8192,
	}

	fmt.Println("🤖 Improving note...")
	fmt.Println()

	var result strings.Builder

	if stream {
		streamResp, err := client.CompleteStream(ctx, req)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}
		defer streamResp.Close()

		for {
			chunk, err := streamResp.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("stream error: %w", err)
			}
			fmt.Print(chunk)
			result.WriteString(chunk)
		}
		fmt.Println()
	} else {
		resp, err := client.Complete(ctx, req)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}
		fmt.Println(resp.Content)
		result.WriteString(resp.Content)
	}

	if inPlace {
		if err := os.WriteFile(filePath, []byte(result.String()), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("\n✅ File updated: %s\n", filePath)
	} else {
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("💡 Use --in-place to save changes to the file")
	}

	return nil
}
