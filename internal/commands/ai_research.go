package commands

// ai_research.go - AI research command

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/ai"
	"github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// newAIResearchCommand creates research notes using AI
func newAIResearchCommand() *cobra.Command {
	var (
		topic               string
		output              string
		brain               string
		title               string
		model               string
		prompt              string
		promptExtra         string
		contextFiles        []string
		contextAutoBrain    bool
		promptCreateTitle   string
		promptCreateBody    string
		promptCreateDefault bool
		timeout             int
		link                bool
		noLink              bool
		noStream            bool
		jsonOut             bool
		prepareOnly         bool
	)

	cmd := &cobra.Command{
		Use:          "research [topic]",
		SilenceUsage: true,
		Short:        "Create a research note on a topic using AI",
		Long: `Create a new note with AI-researched content about a topic.

This command:
1. Sends your topic to the AI provider
2. Gets comprehensive information about the topic
3. Creates a new note with the research results

Examples:
  flip ai research "Kubernetes networking"
  flip ai research "best practices for Go error handling"
  flip ai research "introduction to graph databases" -o graph-db-intro.md`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				topic = args[0]
			}
			JSONOutput = jsonOut
			return runAIResearch(AIResearchOptions{
				AIBaseOptions: AIBaseOptions{
					ModelOverride:  model,
					PromptOverride: prompt,
					PromptExtra:    promptExtra,
					TimeoutSec:     timeout,
					Stream:         !noStream,
					PrepareOnly:    prepareOnly,
				},
				Topic:            topic,
				OutputPath:       output,
				BrainName:        brain,
				Title:            title,
				ContextFiles:     contextFiles,
				ContextAutoBrain: contextAutoBrain,
				PromptCreate: PromptCreateOptions{
					Title:   promptCreateTitle,
					Body:    promptCreateBody,
					Default: promptCreateDefault,
				},
				Link:   link,
				NoLink: noLink,
			})
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: auto-generated)")
	cmd.Flags().StringVar(&brain, "brain", "", "Target brain to save the output (skip selection prompt)")
	cmd.Flags().StringVar(&title, "title", "", "Custom title for the generated note")
	cmd.Flags().StringVar(&model, "model", "", "Override model for this request")
	cmd.Flags().StringVar(&topic, "request", "", "Specific request for this run (alias of topic argument)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt note to apply (path or name)")
	cmd.Flags().StringVar(&prompt, "template", "", "Template note to apply (alias of --prompt)")
	cmd.Flags().StringVar(&promptExtra, "prompt-extra", "", "Additional prompt instructions for this request")
	cmd.Flags().StringVar(&promptExtra, "run-notes", "", "Additional run notes for this request (alias of --prompt-extra)")
	cmd.Flags().StringSliceVar(&contextFiles, "context-file", nil, "Additional context file(s) to include (repeat flag)")
	cmd.Flags().BoolVar(&contextAutoBrain, "context-auto-brain", false, "Automatically search relevant notes in target brain and include as context")
	cmd.Flags().StringVar(&promptCreateTitle, "prompt-create-title", "", "Create a prompt note with this title")
	cmd.Flags().StringVar(&promptCreateTitle, "template-create-title", "", "Create a template note with this title (alias of --prompt-create-title)")
	cmd.Flags().StringVar(&promptCreateBody, "prompt-create-body", "", "Prompt body used when creating a prompt note")
	cmd.Flags().StringVar(&promptCreateBody, "template-create-body", "", "Template body used when creating a template note (alias of --prompt-create-body)")
	cmd.Flags().BoolVar(&promptCreateDefault, "prompt-create-default", false, "Mark created prompt note as default")
	cmd.Flags().BoolVar(&promptCreateDefault, "template-create-default", false, "Mark created template note as default (alias of --prompt-create-default)")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Override AI timeout in seconds")
	cmd.Flags().BoolVar(&link, "link", false, "Always add link to today's journal without prompting")
	cmd.Flags().BoolVar(&noLink, "no-link", false, "Do not add a link to today's journal")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON (for VS Code integration)")
	cmd.Flags().BoolVar(&prepareOnly, "prepare-only", false, "Build prompts and create placeholder note, but skip AI call (output JSON with prompts)")

	return cmd
}

func runAIResearch(opts AIResearchOptions) error {
	topic := opts.Topic
	outputPath := opts.OutputPath
	brainName := opts.BrainName
	title := opts.Title
	modelOverride := opts.ModelOverride
	promptOverride := opts.PromptOverride
	promptExtra := opts.PromptExtra
	contextFiles := opts.ContextFiles
	contextAutoBrain := opts.ContextAutoBrain
	promptCreateTitle := opts.PromptCreate.Title
	promptCreateBody := opts.PromptCreate.Body
	promptCreateDefault := opts.PromptCreate.Default
	timeoutSec := opts.TimeoutSec
	link := opts.Link
	noLink := opts.NoLink
	stream := opts.Stream
	prepareOnly := opts.PrepareOnly

	if !prepareOnly {
		if err := EnsureAISetup(JSONOutput); err != nil {
			return err
		}
	}
	// Get active workspace first
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if len(activeWs.Brains) == 0 {
		fmt.Println("⚠️  No brains found in active workspace. Please add a brain first.")
		return nil
	}

	// STEP 1: Select target brain for output + detect type
	targetBrain, detection, err := selectAndDetectBrain(activeWs, brainName)
	if err != nil {
		return err
	}

	promptContent, promptNote, err := resolvePromptNoteForBrain(targetBrain, detection.Type, promptOverride, promptCreateTitle, promptCreateBody, promptCreateDefault, !JSONOutput)
	if err != nil {
		return err
	}

	// Get topic interactively if not provided
	if topic == "" {
		topic, err = ui.RunInput("What do you want to research? (request)", "", "", nil)
		if err != nil {
			return err
		}
		if topic == "" {
			return fmt.Errorf("topic is required")
		}
	}

	if !prepareOnly {
		PrintOrJSON("\n🔬 Researching: %s\n\n", topic)
	}

	// Create AI client (allow per-request timeout override) — skip for prepare-only
	cfg := ai.LoadConfig()
	if timeoutSec > 0 {
		cfg.Timeout = timeoutSec
	} else if cfg.Timeout < 120 {
		// Research tasks need more time than the default 60s
		cfg.Timeout = 120
	}

	client, effectiveModel, ctx, err := setupAIClientAndModel(cfg, modelOverride, prepareOnly)
	if err != nil {
		return err
	}

	// Build the prompt
	systemPrompt := `You are a knowledgeable research assistant. When asked about a topic:
- Provide comprehensive, accurate information
- Structure the content with clear headings and sections
- Include key concepts, definitions, and explanations
- Add practical examples where relevant
- Note any important considerations or best practices
- Format output in clean markdown
- Be thorough but concise`

	systemPrompt = applyPromptStack(systemPrompt, promptContent, promptExtra)

	var contextParts []string
	var sourceNotes []summarySourceNote

	if contextAutoBrain {
		autoNotes, autoContent, err := collectAutoContextFromBrain(targetBrain, topic)
		if err != nil {
			return err
		}
		if strings.TrimSpace(autoContent) != "" {
			contextParts = append(contextParts, "### Brain context\n"+autoContent)
			sourceNotes = append(sourceNotes, autoNotes...)
		}
	}

	contextFileNotes, contextFileContent, err := loadContextFiles(contextFiles, targetBrain)
	if err != nil {
		return err
	}
	if strings.TrimSpace(contextFileContent) != "" {
		contextParts = append(contextParts, "### Context files\n"+contextFileContent)
		sourceNotes = append(sourceNotes, contextFileNotes...)
	}

	userPrompt := fmt.Sprintf("Please provide a comprehensive overview of: %s\n\nInclude key concepts, practical examples, and best practices.", topic)
	if len(contextParts) > 0 {
		userPrompt += "\n\nUse the following internal context if relevant:\n\n" + strings.Join(contextParts, "\n\n")
	}

	// Resolve title, filename and output path BEFORE the AI call
	noteDate := time.Now().Format("2006-01-02")
	interactiveTitle := !JSONOutput && strings.TrimSpace(title) == ""
	noteTitle, outputPath, _, err := resolveAIOutputPath(topic, title, outputPath, targetBrain.Path, interactiveTitle)
	if err != nil {
		return err
	}

	usedModel := effectiveModel
	if usedModel == "" {
		usedModel = cfg.Model
	}

	promptMeta := buildPromptMeta(promptNote, promptExtra, targetBrain.Path)

	// Build frontmatter (used for both placeholder and final file)
	buildFrontmatter := func(provider string, model string) string {
		return fmt.Sprintf(`---
title: "%s"
date: %s
type: research
topic: "%s"
ai_generated: true
ai_action: research
ai_created_at: "%s"
ai_provider: "%s"
ai_model: "%s"
%s---

`, noteTitle, noteDate, topic, time.Now().Format(time.RFC3339), provider, model, promptMeta)
	}

	// --prepare-only: return prompts + metadata as JSON so the extension can use Copilot
	if prepareOnly {
		promptSection := buildPromptSection(outputPath, targetBrain, promptNote, promptExtra, true)
		sourcesSection := buildSummarySourcesSection(outputPath, targetBrain, sourceNotes, true)
		result := map[string]interface{}{
			"system_prompt":   systemPrompt,
			"user_prompt":     userPrompt,
			"output_path":     outputPath,
			"frontmatter":     buildFrontmatter("__PROVIDER__", "__MODEL__"),
			"prompt_section":  promptSection,
			"sources_section": sourcesSection,
			"brain_name":      targetBrain.Name,
			"brain_path":      targetBrain.Path,
			"title":           noteTitle,
			"topic":           topic,
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal prepare-only result: %w", err)
		}
		fmt.Println(string(jsonBytes))
		return nil
	}

	// Write placeholder file immediately so the user sees progress
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	placeholderContent := buildFrontmatter(cfg.Provider, usedModel) + fmt.Sprintf("> ⏳ **AI is working…** (%s / %s)\n>\n> Researching: *%s*\n>\n> This note will be updated automatically when the AI finishes.\n", cfg.Provider, usedModel, topic)
	if err := os.WriteFile(outputPath, []byte(placeholderContent), 0644); err != nil {
		return fmt.Errorf("failed to write placeholder file: %w", err)
	}

	PrintOrJSON("\n📝 Note created: %s\n", outputPath)

	// Try to open the file in the editor (best-effort)
	if !JSONOutput {
		if _, err := exec.LookPath("code"); err == nil {
			_ = exec.Command("code", outputPath).Start()
		}
	}

	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Model:       effectiveModel,
		Temperature: 0.5,
		MaxTokens:   4096,
	}

	PrintlnOrJSON("🤖 Generating research note...")
	PrintlnOrJSON()

	var result strings.Builder
	var aiErr error

	if stream {
		streamResp, err := client.CompleteStream(ctx, req)
		if err != nil {
			aiErr = fmt.Errorf("AI error: %w", err)
		} else {
			defer streamResp.Close()

			for {
				chunk, err := streamResp.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					aiErr = fmt.Errorf("stream error: %w", err)
					break
				}
				if !JSONOutput {
					fmt.Print(chunk)
				}
				result.WriteString(chunk)
			}
			if !JSONOutput {
				fmt.Println()
			}
		}
	} else {
		resp, err := client.Complete(ctx, req)
		if err != nil {
			aiErr = fmt.Errorf("AI error: %w", err)
		} else {
			if !JSONOutput {
				fmt.Println(resp.Content)
			}
			result.WriteString(resp.Content)
			if resp.Model != "" {
				usedModel = resp.Model
			}
		}
	}

	// Update the file with final content (or error message)
	frontmatter := buildFrontmatter(cfg.Provider, usedModel)
	promptSection := buildPromptSection(outputPath, targetBrain, promptNote, promptExtra, true)
	sourcesSection := buildSummarySourcesSection(outputPath, targetBrain, sourceNotes, true)

	var finalContent string
	if aiErr != nil {
		// Write error info into the note so the user sees what happened
		errMsg := fmt.Sprintf("> ❌ **AI request failed**\n>\n> %s\n>\n> You can retry with: `flip ai research \"%s\" --timeout %d`\n\n",
			aiErr.Error(), topic, cfg.Timeout*2)
		if result.Len() > 0 {
			// Partial content was received before the error
			errMsg += "> ⚠️ Partial content received before error:\n\n"
		}
		finalContent = promptSection + sourcesSection + errMsg + result.String()
	} else {
		finalContent = promptSection + sourcesSection + result.String()
	}

	if err := os.WriteFile(outputPath, []byte(frontmatter+finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if JSONOutput {
		OutputJSONSuccess("ai research", NoteResult{
			Action:    "created",
			Path:      outputPath,
			Title:     noteTitle,
			BrainName: targetBrain.Name,
			BrainPath: targetBrain.Path,
			BrainType: string(detection.Type),
		})
	} else if aiErr != nil {
		fmt.Printf("\n\n⚠️  Research note saved with error: %s\n", outputPath)
		if strings.Contains(aiErr.Error(), "deadline exceeded") || strings.Contains(aiErr.Error(), "Timeout") {
			fmt.Printf("💡 Tip: Try again with a longer timeout: flip ai research \"%s\" --timeout %d\n", topic, cfg.Timeout*2)
		}
	} else {
		fmt.Printf("\n\n✅ Research note saved to: %s\n", outputPath)
	}

	// Add link to journal (always attempt, even on AI error – the note file exists)
	addAIJournalLink(outputPath, targetBrain, detection.Type, noteTitle, "Research", cfg.Provider, noLink, !link && !JSONOutput)

	// Return the AI error after everything else is done
	if aiErr != nil {
		return fmt.Errorf("failed to create research note: %w", aiErr)
	}

	return nil
}
