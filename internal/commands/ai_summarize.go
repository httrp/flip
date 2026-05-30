package commands

// ai_summarize.go - AI summarize command

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/ai"
	"github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// newAISummarizeCommand creates notes summarizing content from brains
func newAISummarizeCommand() *cobra.Command {
	var (
		topic               string
		brains              []string
		output              string
		brain               string
		title               string
		model               string
		prompt              string
		promptExtra         string
		contextFiles        []string
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
		Use:          "summarize [topic]",
		SilenceUsage: true,
		Short:        "Summarize your notes about a topic",
		Long: `Search your brains for notes about a topic and create a summary.

This command:
1. Searches your brains for notes matching the topic
2. Sends the relevant content to your AI provider
3. Creates a new note with the AI-generated summary

Examples:
  flip ai summarize "project management"
  flip ai summarize "golang best practices" --brains my-brain
  flip ai summarize "meeting notes Q1" -o summary.md`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				topic = args[0]
			}
			JSONOutput = jsonOut
			return runAISummarize(AISummarizeOptions{
				AIBaseOptions: AIBaseOptions{
					ModelOverride:  model,
					PromptOverride: prompt,
					PromptExtra:    promptExtra,
					TimeoutSec:     timeout,
					Stream:         !noStream,
					PrepareOnly:    prepareOnly,
				},
				Topic:        topic,
				SearchBrains: brains,
				OutputPath:   output,
				BrainName:    brain,
				Title:        title,
				ContextFiles: contextFiles,
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

	cmd.Flags().StringSliceVarP(&brains, "brains", "b", nil, "Specific brains to search (default: all)")
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
	cmd.Flags().BoolVar(&prepareOnly, "prepare-only", false, "Return prompts as JSON without calling AI (for Copilot integration)")

	return cmd
}

func runAISummarize(opts AISummarizeOptions) error {
	topic := opts.Topic
	brainNames := opts.SearchBrains
	outputPath := opts.OutputPath
	brainName := opts.BrainName
	title := opts.Title
	modelOverride := opts.ModelOverride
	promptOverride := opts.PromptOverride
	promptExtra := opts.PromptExtra
	contextFiles := opts.ContextFiles
	promptCreateTitle := opts.PromptCreate.Title
	promptCreateBody := opts.PromptCreate.Body
	promptCreateDefault := opts.PromptCreate.Default
	timeoutSec := opts.TimeoutSec
	link := opts.Link
	noLink := opts.NoLink
	stream := opts.Stream
	prepareOnly := opts.PrepareOnly

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
		topic, err = ui.RunInput("What should be summarized? (request)", "", "", nil)
		if err != nil {
			return err
		}
		if topic == "" {
			return fmt.Errorf("topic is required")
		}
	}

	if !prepareOnly {
		PrintOrJSON("\n🔍 Searching for notes about: %s\n", topic)
	}

	var searchBrains []Brain

	if len(brainNames) > 0 {
		// Use specified brains
		for _, name := range brainNames {
			for _, b := range activeWs.Brains {
				if b.Name == name {
					searchBrains = append(searchBrains, b)
					break
				}
			}
		}
	} else {
		// Use all brains in active workspace
		searchBrains = activeWs.Brains
	}

	if len(searchBrains) == 0 {
		return fmt.Errorf("no brains to search")
	}

	// Search for matching notes
	var relevantContent strings.Builder
	var noteCount int
	var sourceNotes []summarySourceNote
	sourceSeen := make(map[string]bool)

	for _, brain := range searchBrains {
		notes, err := searchNotesInBrain(brain.Path, topic)
		if err != nil {
			continue
		}

		for _, note := range notes {
			content, err := os.ReadFile(note)
			if err != nil {
				continue
			}

			// Add note content with header
			relPath, _ := filepath.Rel(brain.Path, note)
			relevantContent.WriteString(fmt.Sprintf("\n--- %s/%s ---\n", brain.Name, relPath))
			relevantContent.WriteString(string(content))
			relevantContent.WriteString("\n")
			noteCount++

			key := brain.Name + "|" + relPath
			if !sourceSeen[key] {
				sourceSeen[key] = true
				titleFromFile := strings.TrimSuffix(filepath.Base(note), filepath.Ext(note))
				sourceNotes = append(sourceNotes, summarySourceNote{
					BrainName: brain.Name,
					BrainPath: brain.Path,
					NotePath:  note,
					RelPath:   relPath,
					Title:     titleFromFile,
				})
			}

			// Limit to avoid token overflow
			if relevantContent.Len() > 50000 {
				break
			}
		}
	}

	// Add explicit context files
	contextNotes, contextContent, err := loadContextFiles(contextFiles, targetBrain)
	if err != nil {
		return err
	}
	if strings.TrimSpace(contextContent) != "" {
		relevantContent.WriteString("\n\n--- Additional context files ---\n")
		relevantContent.WriteString(contextContent)
		relevantContent.WriteString("\n")
		noteCount += len(contextNotes)
		sourceNotes = append(sourceNotes, contextNotes...)
	}

	if noteCount == 0 {
		PrintlnOrJSON("❌ No notes found matching the topic")
		return nil
	}

	if !prepareOnly {
		PrintOrJSON("📄 Found %d relevant notes\n\n", noteCount)
	}

	// Create AI client (allow per-request timeout override) — skip for prepare-only
	cfg := ai.LoadConfig()
	if timeoutSec > 0 {
		cfg.Timeout = timeoutSec
	}

	client, effectiveModel, ctx, err := setupAIClientAndModel(cfg, modelOverride, prepareOnly)
	if err != nil {
		return err
	}

	// Build the prompt
	systemPrompt := `You are a helpful assistant that creates clear summaries of notes.
Given a collection of notes about a topic, create a well-structured summary that:
- Highlights the key points and insights
- Groups related information together
- Uses clear headings and bullet points
- Preserves important details and references
- Is written in markdown format

Keep the summary concise but comprehensive.`

	systemPrompt = applyPromptStack(systemPrompt, promptContent, promptExtra)

	userPrompt := fmt.Sprintf("Please summarize the following notes about \"%s\":\n\n%s", topic, relevantContent.String())

	// Resolve title, filename, output path early (needed for prepare-only)
	noteDate := time.Now().Format("2006-01-02")
	interactiveTitle := !JSONOutput && strings.TrimSpace(title) == ""
	noteTitle, outputPath, _, err := resolveAIOutputPath(topic, title, outputPath, targetBrain.Path, interactiveTitle)
	if err != nil {
		return err
	}

	// --prepare-only: return prompts + metadata as JSON so the extension can use Copilot
	if prepareOnly {
		promptSection := buildPromptSection(outputPath, targetBrain, promptNote, promptExtra, true)
		sourcesSection := buildSummarySourcesSection(outputPath, targetBrain, sourceNotes, true)

		usedModel := effectiveModel
		if usedModel == "" {
			usedModel = cfg.Model
		}
		promptMeta := buildPromptMeta(promptNote, promptExtra, targetBrain.Path)
		frontmatter := fmt.Sprintf("---\ntitle: \"%s\"\ndate: %s\ntype: summary\ntopic: \"%s\"\nsources: %d notes\nai_generated: true\nai_action: summary\nai_created_at: \"%s\"\nai_provider: \"%s\"\nai_model: \"%s\"\n%s---\n\n",
			noteTitle, noteDate, topic, noteCount, time.Now().Format(time.RFC3339), "__PROVIDER__", "__MODEL__", promptMeta)

		result := map[string]interface{}{
			"system_prompt":   systemPrompt,
			"user_prompt":     userPrompt,
			"output_path":     outputPath,
			"frontmatter":     frontmatter,
			"prompt_section":  promptSection,
			"sources_section": sourcesSection,
			"brain_name":      targetBrain.Name,
			"brain_path":      targetBrain.Path,
			"title":           noteTitle,
			"topic":           topic,
			"source_count":    noteCount,
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal prepare-only result: %w", err)
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
		Temperature: 0.3, // Lower temperature for summarization
		MaxTokens:   4096,
	}

	PrintlnOrJSON("🤖 Generating summary...")
	PrintlnOrJSON()

	var result strings.Builder

	usedModel := effectiveModel

	if stream {
		// Stream the response
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
			if !JSONOutput {
				fmt.Print(chunk)
			}
			result.WriteString(chunk)
		}
		if !JSONOutput {
			fmt.Println()
		}
	} else {
		// Non-streaming response
		resp, err := client.Complete(ctx, req)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}
		if !JSONOutput {
			fmt.Println(resp.Content)
		}
		result.WriteString(resp.Content)
		if resp.Model != "" {
			usedModel = resp.Model
		}
	}

	// Save to file — title, filename, outputPath already resolved above

	// Get AI config for metadata
	if usedModel == "" {
		usedModel = cfg.Model
	}

	promptMeta := buildPromptMeta(promptNote, promptExtra, targetBrain.Path)

	// Create frontmatter with AI metadata
	frontmatter := fmt.Sprintf(`---
title: "%s"
date: %s
type: summary
topic: "%s"
sources: %d notes
ai_generated: true
ai_action: summary
ai_created_at: "%s"
ai_provider: "%s"
ai_model: "%s"
%s---

`, noteTitle, noteDate, topic, noteCount, time.Now().Format(time.RFC3339), cfg.Provider, usedModel, promptMeta)

	// Append sources/prompts section at the top
	promptSection := buildPromptSection(outputPath, targetBrain, promptNote, promptExtra, true)
	sourcesSection := buildSummarySourcesSection(outputPath, targetBrain, sourceNotes, true)
	finalContent := promptSection + sourcesSection + result.String()

	// Write file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(frontmatter+finalContent), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if JSONOutput {
		OutputJSONSuccess("ai summarize", NoteResult{
			Action:    "created",
			Path:      outputPath,
			Title:     noteTitle,
			BrainName: targetBrain.Name,
			BrainPath: targetBrain.Path,
			BrainType: string(detection.Type),
		})
		return nil
	}

	fmt.Printf("\n\n✅ Summary saved to: %s\n", outputPath)

	// Add link to journal
	addAIJournalLink(outputPath, targetBrain, detection.Type, noteTitle, "Summary", cfg.Provider, noLink, !link)

	return nil
}
