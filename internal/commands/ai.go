package commands

// ai.go - AI Integration Commands
//
// Provides AI-powered features for flip:
//   - flip ai summarize: Summarize notes about a topic from your brains
//   - flip ai research: Create a research note on a topic using AI
//   - flip ai improve: Improve/rewrite a note with AI assistance
//   - flip ai status: Check AI provider status
//
// Configuration via environment variables:
//   FLIP_AI_PROVIDER: ollama (default), openai, anthropic, azure
//   FLIP_FEATURE_AI: 0 to disable AI features

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/ai"
	"github.com/httrp/flip/internal/brain"
	"github.com/manifoldco/promptui"
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
  FLIP_AI_PROVIDER=ollama|openai|anthropic|groq|azure
  OLLAMA_HOST=http://localhost:11434 (for Ollama)
  OPENAI_API_KEY=sk-... (for OpenAI)
  ANTHROPIC_API_KEY=sk-ant-... (for Anthropic)
  GROQ_API_KEY=gsk_... (for Groq - fast & free)`,
	}

	cmd.AddCommand(newAISummarizeCommand())
	cmd.AddCommand(newAIResearchCommand())
	cmd.AddCommand(newAIImproveCommand())
	cmd.AddCommand(newAIStatusCommand())
	cmd.AddCommand(newAIModelsCommand())

	return cmd
}

// newAIStatusCommand shows AI provider status
func newAIStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check AI provider status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAIStatus()
		},
	}
}

func runAIStatus() error {
	fmt.Println()
	fmt.Println("🤖 AI Provider Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	cfg := ai.LoadConfig()
	fmt.Printf("Provider: %s\n", cfg.Provider)
	fmt.Printf("Model:    %s\n", cfg.Model)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ai.NewClient()
	if err != nil {
		fmt.Printf("Status:   ❌ Configuration error: %v\n", err)
		return nil
	}

	if client.IsAvailable(ctx) {
		fmt.Println("Status:   ✅ Available")

		// Try to list models
		models, err := client.ListModels(ctx)
		if err == nil && len(models) > 0 {
			fmt.Printf("Models:   %d available\n", len(models))
			// Show first few models
			for i, m := range models {
				if i >= 5 {
					fmt.Printf("          ...and %d more\n", len(models)-5)
					break
				}
				fmt.Printf("          - %s\n", m.Name)
			}
		}
	} else {
		fmt.Println("Status:   ⚠️  Not available")
		fmt.Println()
		fmt.Println("Troubleshooting:")
		switch cfg.Provider {
		case "ollama":
			fmt.Println("  • Make sure Ollama is running: ollama serve")
			fmt.Println("  • Check OLLAMA_HOST if using a different address")
		case "openai":
			fmt.Println("  • Check that OPENAI_API_KEY is set correctly")
		case "anthropic":
			fmt.Println("  • Check that ANTHROPIC_API_KEY is set correctly")
		case "groq":
			fmt.Println("  • Check that GROQ_API_KEY is set correctly")
			fmt.Println("  • Get a free key at: https://console.groq.com")
		}
	}

	fmt.Println()
	return nil
}

type aiModelsResponse struct {
	Provider     string        `json:"provider"`
	DefaultModel string        `json:"default_model"`
	Models       []ai.ModelInfo `json:"models"`
}

// newAIModelsCommand lists available models for the current provider
func newAIModelsCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "models",
		Short: "List available AI models",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			return runAIModels()
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON (for VS Code integration)")

	return cmd
}

func runAIModels() error {
	cfg := ai.LoadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := ai.NewClient()
	if err != nil {
		if JSONOutput {
			OutputJSONError("ai models", err)
			return nil
		}
		return err
	}

	models, err := client.ListModels(ctx)
	if err != nil {
		if JSONOutput {
			OutputJSONError("ai models", err)
			return nil
		}
		return err
	}

	resp := aiModelsResponse{
		Provider:     cfg.Provider,
		DefaultModel: cfg.Model,
		Models:       models,
	}

	if JSONOutput {
		OutputJSONSuccess("ai models", resp)
		return nil
	}

	fmt.Println("\n🤖 Available Models")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Provider: %s\n", cfg.Provider)
	fmt.Printf("Default:  %s\n\n", cfg.Model)

	if len(models) == 0 {
		fmt.Println("No models found.")
		return nil
	}

	for _, m := range models {
		label := m.ID
		if m.Name != "" && m.Name != m.ID {
			label = fmt.Sprintf("%s (%s)", m.Name, m.ID)
		}
		fmt.Printf("- %s\n", label)
		if m.Description != "" {
			fmt.Printf("  %s\n", m.Description)
		}
	}

	return nil
}

// newAISummarizeCommand creates notes summarizing content from brains
func newAISummarizeCommand() *cobra.Command {
	var (
		topic    string
		brains   []string
		output   string
		brain    string
		title    string
		model    string
		prompt   string
		promptExtra string
		promptCreateTitle string
		promptCreateBody  string
		promptCreateDefault bool
		timeout  int
		link     bool
		noLink   bool
		noStream bool
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "summarize [topic]",
		Short: "Summarize your notes about a topic",
		Long: `Search your brains for notes about a topic and create a summary.

This command:
1. Searches your brains for notes matching the topic
2. Sends the relevant content to your AI provider
3. Creates a new note with the AI-generated summary

Examples:
  flip ai summarize "project management"
  flip ai summarize "golang best practices" --brains danobrain
  flip ai summarize "meeting notes Q1" -o summary.md`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				topic = args[0]
			}
			JSONOutput = jsonOut
			return runAISummarize(topic, brains, output, brain, title, model, prompt, promptExtra, promptCreateTitle, promptCreateBody, promptCreateDefault, timeout, link, noLink, !noStream)
		},
	}

	cmd.Flags().StringSliceVarP(&brains, "brains", "b", nil, "Specific brains to search (default: all)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: auto-generated)")
	cmd.Flags().StringVar(&brain, "brain", "", "Target brain to save the output (skip selection prompt)")
	cmd.Flags().StringVar(&title, "title", "", "Custom title for the generated note")
	cmd.Flags().StringVar(&model, "model", "", "Override model for this request")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt note to apply (path or name)")
	cmd.Flags().StringVar(&promptExtra, "prompt-extra", "", "Additional prompt instructions for this request")
	cmd.Flags().StringVar(&promptCreateTitle, "prompt-create-title", "", "Create a prompt note with this title")
	cmd.Flags().StringVar(&promptCreateBody, "prompt-create-body", "", "Prompt body used when creating a prompt note")
	cmd.Flags().BoolVar(&promptCreateDefault, "prompt-create-default", false, "Mark created prompt note as default")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Override AI timeout in seconds")
	cmd.Flags().BoolVar(&link, "link", false, "Always add link to today's journal without prompting")
	cmd.Flags().BoolVar(&noLink, "no-link", false, "Do not add a link to today's journal")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON (for VS Code integration)")

	return cmd
}

type summarySourceNote struct {
	BrainName string
	BrainPath string
	NotePath  string
	RelPath   string
	Title     string
}

func runAISummarize(topic string, brainNames []string, outputPath string, brainName string, title string, modelOverride string, promptOverride string, promptExtra string, promptCreateTitle string, promptCreateBody string, promptCreateDefault bool, timeoutSec int, link bool, noLink bool, stream bool) error {
	// Get active workspace first
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if len(activeWs.Brains) == 0 {
		fmt.Println("⚠️  No brains found in active workspace. Please add a brain first.")
		return nil
	}

	// STEP 1: Select target brain for output
	var targetBrain *Brain
	if brainName != "" {
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == brainName {
				targetBrain = &activeWs.Brains[i]
				break
			}
		}
		if targetBrain == nil {
			return fmt.Errorf("brain not found: %s", brainName)
		}
	} else {
		targetBrain, err = confirmOrSelectBrain(activeWs)
		if err != nil {
			return err
		}
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, _ := detector.DetectBrainType(targetBrain.Path)

	promptContent, promptNote, err := resolvePromptNoteForBrain(targetBrain, detection.Type, promptOverride, promptCreateTitle, promptCreateBody, promptCreateDefault, !JSONOutput)
	if err != nil {
		return err
	}

	// Get topic interactively if not provided
	if topic == "" {
		prompt := promptui.Prompt{
			Label: "What topic do you want to summarize?",
		}
		topic, err = prompt.Run()
		if err != nil {
			return err
		}
		if topic == "" {
			return fmt.Errorf("topic is required")
		}
	}

	PrintOrJSON("\n🔍 Searching for notes about: %s\n", topic)

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

	if noteCount == 0 {
		PrintlnOrJSON("❌ No notes found matching the topic")
		return nil
	}

	PrintOrJSON("📄 Found %d relevant notes\n\n", noteCount)

	// Create AI client (allow per-request timeout override)
	cfg := ai.LoadConfig()
	if timeoutSec > 0 {
		cfg.Timeout = timeoutSec
	}
	client, err := ai.NewClientWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
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

	ctx := context.Background()
	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Model:       modelOverride,
		Temperature: 0.3, // Lower temperature for summarization
		MaxTokens:   4096,
	}

	PrintlnOrJSON("🤖 Generating summary...")
	PrintlnOrJSON()

	var result strings.Builder

	usedModel := ""
	if modelOverride != "" {
		usedModel = modelOverride
	}

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

	// Save to file
	noteDate := time.Now().Format("2006-01-02")
	noteDir := filepath.Join(targetBrain.Path, "notes")

	interactiveTitle := !JSONOutput && strings.TrimSpace(title) == ""
	noteTitle, filename, err := resolveAITitleAndFilename(topic, title, noteDir, noteDate, interactiveTitle)
	if err != nil {
		return err
	}

	// Use selected target brain for output
	if outputPath == "" {
		outputPath = filepath.Join(targetBrain.Path, "notes", filename)
	} else if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(targetBrain.Path, "notes", outputPath)
	}

	// Get AI config for metadata
	if usedModel == "" {
		usedModel = cfg.Model
	}

	promptMeta := ""
	if promptNote != nil {
		relPrompt := relativePathFromBrain(promptNote.Path, targetBrain.Path)
		promptMeta = fmt.Sprintf("prompt_note: \"%s\"\n", filepath.ToSlash(relPrompt))
	}
	if strings.TrimSpace(promptExtra) != "" {
		promptMeta += "prompt_extra: true\n"
	}

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
	if !noLink {
		relPath := relativePathFromBrain(outputPath, targetBrain.Path)
		linkTitle := formatAILinkTitle(noteTitle, "Summary", cfg.Provider)
		if err := AddLinkToJournal(JournalLinkOptions{
			ItemType:    "note",
			ItemName:    linkTitle,
			ItemPath:    relPath,
			Brain:       targetBrain,
			Interactive: !link,
			BrainType:   detection.Type,
		}); err != nil {
			fmt.Printf("⚠️  Could not add journal link: %v\n", err)
		}
	}

	return nil
}

func buildSummarySourcesSection(summaryPath string, targetBrain *Brain, sources []summarySourceNote, force bool) string {
	if len(sources) == 0 && !force {
		return ""
	}

	lines := []string{"", "", "## Sources"}
	if len(sources) == 0 {
		lines = append(lines, "- None")
		return strings.Join(lines, "\n")
	}
	for _, source := range sources {
		label := source.Title
		if source.BrainName != "" && source.BrainName != targetBrain.Name {
			label = source.BrainName + ": " + label
		}

		linkPath := source.NotePath
		if source.BrainName == targetBrain.Name {
			rel, err := filepath.Rel(filepath.Dir(summaryPath), source.NotePath)
			if err == nil {
				linkPath = rel
			}
		}
		linkPath = filepath.ToSlash(linkPath)
		lines = append(lines, fmt.Sprintf("- [%s](%s)", label, linkPath))
	}

	return strings.Join(lines, "\n")
}

func buildPromptSection(notePath string, targetBrain *Brain, promptNote *PromptNoteInfo, promptExtra string, force bool) string {
	if promptNote == nil && strings.TrimSpace(promptExtra) == "" && !force {
		return ""
	}

	lines := []string{"", "", "## Prompt"}
	if promptNote != nil {
		label := promptNote.Title
		if label == "" {
			label = promptNote.Name
		}

		linkPath := promptNote.Path
		if targetBrain != nil && targetBrain.Path != "" {
			rel, err := filepath.Rel(filepath.Dir(notePath), promptNote.Path)
			if err == nil {
				linkPath = rel
			}
		}
		linkPath = filepath.ToSlash(linkPath)
		lines = append(lines, fmt.Sprintf("- Prompt note: [%s](%s)", label, linkPath))
	}

	extra := strings.TrimSpace(promptExtra)
	if extra != "" {
		lines = append(lines, "", "### Extra Instructions", "", extra)
	}
	if promptNote == nil && extra == "" {
		lines = append(lines, "- None")
	}

	return strings.Join(lines, "\n")
}

// searchNotesInBrain searches for notes matching a query in a brain
func searchNotesInBrain(brainPath string, query string) ([]string, error) {
	var matches []string
	query = strings.ToLower(query)
	keywords := strings.Fields(query)

	err := filepath.Walk(brainPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories and non-markdown files
		if info.IsDir() {
			// Skip hidden directories and common non-note directories
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "templates" || name == "assets" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Check filename
		lowerName := strings.ToLower(info.Name())
		for _, kw := range keywords {
			if strings.Contains(lowerName, kw) {
				matches = append(matches, path)
				return nil
			}
		}

		// Check file content (limited scan)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lowerContent := strings.ToLower(string(content))
		matchCount := 0
		for _, kw := range keywords {
			if strings.Contains(lowerContent, kw) {
				matchCount++
			}
		}

		// Require at least half the keywords to match
		if matchCount >= len(keywords)/2+1 || (len(keywords) == 1 && matchCount == 1) {
			matches = append(matches, path)
		}

		return nil
	})

	return matches, err
}

// newAIResearchCommand creates research notes using AI
func newAIResearchCommand() *cobra.Command {
	var (
		topic    string
		output   string
		brain    string
		title    string
		model    string
		prompt   string
		promptExtra string
		promptCreateTitle string
		promptCreateBody  string
		promptCreateDefault bool
		timeout  int
		link     bool
		noLink   bool
		noStream bool
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "research [topic]",
		Short: "Create a research note on a topic using AI",
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
			return runAIResearch(topic, output, brain, title, model, prompt, promptExtra, promptCreateTitle, promptCreateBody, promptCreateDefault, timeout, link, noLink, !noStream)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: auto-generated)")
	cmd.Flags().StringVar(&brain, "brain", "", "Target brain to save the output (skip selection prompt)")
	cmd.Flags().StringVar(&title, "title", "", "Custom title for the generated note")
	cmd.Flags().StringVar(&model, "model", "", "Override model for this request")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt note to apply (path or name)")
	cmd.Flags().StringVar(&promptExtra, "prompt-extra", "", "Additional prompt instructions for this request")
	cmd.Flags().StringVar(&promptCreateTitle, "prompt-create-title", "", "Create a prompt note with this title")
	cmd.Flags().StringVar(&promptCreateBody, "prompt-create-body", "", "Prompt body used when creating a prompt note")
	cmd.Flags().BoolVar(&promptCreateDefault, "prompt-create-default", false, "Mark created prompt note as default")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Override AI timeout in seconds")
	cmd.Flags().BoolVar(&link, "link", false, "Always add link to today's journal without prompting")
	cmd.Flags().BoolVar(&noLink, "no-link", false, "Do not add a link to today's journal")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON (for VS Code integration)")

	return cmd
}

func runAIResearch(topic string, outputPath string, brainName string, title string, modelOverride string, promptOverride string, promptExtra string, promptCreateTitle string, promptCreateBody string, promptCreateDefault bool, timeoutSec int, link bool, noLink bool, stream bool) error {
	// Get active workspace first
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if len(activeWs.Brains) == 0 {
		fmt.Println("⚠️  No brains found in active workspace. Please add a brain first.")
		return nil
	}

	// STEP 1: Select target brain for output
	var targetBrain *Brain
	if brainName != "" {
		for i := range activeWs.Brains {
			if activeWs.Brains[i].Name == brainName {
				targetBrain = &activeWs.Brains[i]
				break
			}
		}
		if targetBrain == nil {
			return fmt.Errorf("brain not found: %s", brainName)
		}
	} else {
		targetBrain, err = confirmOrSelectBrain(activeWs)
		if err != nil {
			return err
		}
	}

	// Detect brain type
	detector := brain.NewDetector()
	detection, _ := detector.DetectBrainType(targetBrain.Path)

	promptContent, promptNote, err := resolvePromptNoteForBrain(targetBrain, detection.Type, promptOverride, promptCreateTitle, promptCreateBody, promptCreateDefault, !JSONOutput)
	if err != nil {
		return err
	}

	// Get topic interactively if not provided
	if topic == "" {
		prompt := promptui.Prompt{
			Label: "What topic do you want to research?",
		}
		topic, err = prompt.Run()
		if err != nil {
			return err
		}
		if topic == "" {
			return fmt.Errorf("topic is required")
		}
	}

	PrintOrJSON("\n🔬 Researching: %s\n\n", topic)

	// Create AI client (allow per-request timeout override)
	cfg := ai.LoadConfig()
	if timeoutSec > 0 {
		cfg.Timeout = timeoutSec
	}
	client, err := ai.NewClientWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
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

	userPrompt := fmt.Sprintf("Please provide a comprehensive overview of: %s\n\nInclude key concepts, practical examples, and best practices.", topic)

	ctx := context.Background()
	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Model:       modelOverride,
		Temperature: 0.5,
		MaxTokens:   4096,
	}

	PrintlnOrJSON("🤖 Generating research note...")
	PrintlnOrJSON()

	var result strings.Builder

	usedModel := ""
	if modelOverride != "" {
		usedModel = modelOverride
	}

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
			if !JSONOutput {
				fmt.Print(chunk)
			}
			result.WriteString(chunk)
		}
		if !JSONOutput {
			fmt.Println()
		}
	} else {
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

	// Save to file
	noteDate := time.Now().Format("2006-01-02")
	noteDir := filepath.Join(targetBrain.Path, "notes")

	interactiveTitle := !JSONOutput && strings.TrimSpace(title) == ""
	noteTitle, filename, err := resolveAITitleAndFilename(topic, title, noteDir, noteDate, interactiveTitle)
	if err != nil {
		return err
	}

	// Use selected target brain for output
	if outputPath == "" {
		outputPath = filepath.Join(targetBrain.Path, "notes", filename)
	} else if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(targetBrain.Path, "notes", outputPath)
	}

	// Get AI config for metadata
	if usedModel == "" {
		usedModel = cfg.Model
	}

	promptMeta := ""
	if promptNote != nil {
		relPrompt := relativePathFromBrain(promptNote.Path, targetBrain.Path)
		promptMeta = fmt.Sprintf("prompt_note: \"%s\"\n", filepath.ToSlash(relPrompt))
	}
	if strings.TrimSpace(promptExtra) != "" {
		promptMeta += "prompt_extra: true\n"
	}

	// Create frontmatter with AI metadata
frontmatter := fmt.Sprintf(`---
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

`, noteTitle, noteDate, topic, time.Now().Format(time.RFC3339), cfg.Provider, usedModel, promptMeta)

	promptSection := buildPromptSection(outputPath, targetBrain, promptNote, promptExtra, true)
	sourcesSection := buildSummarySourcesSection(outputPath, targetBrain, nil, true)
	finalContent := promptSection + sourcesSection + result.String()

	// Write file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
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
		return nil
	}

	fmt.Printf("\n\n✅ Research note saved to: %s\n", outputPath)

	// Add link to journal
	if !noLink {
		relPath := relativePathFromBrain(outputPath, targetBrain.Path)
		linkTitle := formatAILinkTitle(noteTitle, "Research", cfg.Provider)
		if err := AddLinkToJournal(JournalLinkOptions{
			ItemType:    "note",
			ItemName:    linkTitle,
			ItemPath:    relPath,
			Brain:       targetBrain,
			Interactive: !link,
			BrainType:   detection.Type,
		}); err != nil {
			fmt.Printf("⚠️  Could not add journal link: %v\n", err)
		}
	}

	return nil
}

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
	)

	cmd := &cobra.Command{
		Use:   "improve [file]",
		Short: "Improve a note with AI assistance",
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
			return runAIImprove(args[0], instruction, model, prompt, promptExtra, timeout, !noStream, inPlace)
		},
	}

	cmd.Flags().StringVarP(&instruction, "instruction", "i", "", "Specific improvement instructions")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&inPlace, "in-place", false, "Update the file in place")
	cmd.Flags().StringVar(&model, "model", "", "Override model for this request")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt note to apply (path or name)")
	cmd.Flags().StringVar(&promptExtra, "prompt-extra", "", "Additional prompt instructions for this request")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Override AI timeout in seconds")

	return cmd
}

func runAIImprove(filePath string, instruction string, modelOverride string, promptOverride string, promptExtra string, timeoutSec int, stream bool, inPlace bool) error {
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
		prompt := promptui.Prompt{
			Label:   "How should the note be improved?",
			Default: "improve clarity and structure",
		}
		instruction, err = prompt.Run()
		if err != nil {
			return err
		}
	}

	fmt.Printf("\n📝 Improving: %s\n", filePath)
	fmt.Printf("💡 Instruction: %s\n\n", instruction)

	// Create AI client (allow per-request timeout override)
	cfg := ai.LoadConfig()
	if timeoutSec > 0 {
		cfg.Timeout = timeoutSec
	}
	client, err := ai.NewClientWithConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
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

	ctx := context.Background()
	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Model:       modelOverride,
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

// aiGetDefaultBrain returns the default brain from a workspace
func aiGetDefaultBrain(ws *Workspace) *Brain {
	if ws == nil {
		return nil
	}
	for i := range ws.Brains {
		if ws.Brains[i].Name == ws.DefaultBrain {
			return &ws.Brains[i]
		}
	}
	if len(ws.Brains) > 0 {
		return &ws.Brains[0]
	}
	return nil
}

func resolvePromptNoteForBrain(brainInfo *Brain, brainType brain.BrainType, override string, promptCreateTitle string, promptCreateBody string, promptCreateDefault bool, interactive bool) (string, *PromptNoteInfo, error) {
	if brainInfo == nil {
		override = strings.TrimSpace(override)
		if override == "" {
			return "", nil, nil
		}
		lower := strings.ToLower(override)
		if lower == "none" || lower == "off" || lower == "no" {
			return "", nil, nil
		}
		if filepath.IsAbs(override) {
			content, err := loadPromptNoteContent(override)
			if err != nil {
				return "", nil, err
			}
			return content, &PromptNoteInfo{Path: override}, nil
		}
		return "", nil, fmt.Errorf("prompt note requires a brain context or absolute path")
	}

	if strings.TrimSpace(promptCreateTitle) != "" {
		promptNote, err := createPromptNote(brainInfo.Path, brainType, promptCreateTitle, promptCreateBody, promptCreateDefault)
		if err != nil {
			return "", nil, err
		}
		content, err := loadPromptNoteContent(promptNote.Path)
		if err != nil {
			return "", nil, err
		}
		return content, &promptNote, nil
	}

	prompts, defaultPrompt, err := findPromptNotes(brainInfo.Path, brainType)
	if err != nil {
		return "", nil, err
	}

	resolved, handled, err := resolvePromptOverride(override, brainInfo.Path, brainType, prompts)
	if err != nil {
		return "", nil, err
	}
	if handled {
		if resolved.Path == "" {
			return "", nil, nil
		}
		content, err := loadPromptNoteContent(resolved.Path)
		if err != nil {
			return "", nil, err
		}
		return content, &resolved, nil
	}

	if defaultPrompt != nil {
		content, err := loadPromptNoteContent(defaultPrompt.Path)
		if err != nil {
			return "", nil, err
		}
		return content, defaultPrompt, nil
	}

	if !interactive || len(prompts) == 0 {
		if interactive && len(prompts) == 0 {
			createSelector := promptui.Select{
				Label: "No prompt notes found. Create one?",
				Items: []string{"Yes", "No"},
			}
			_, createChoice, err := createSelector.Run()
			if err != nil {
				return "", nil, err
			}
			if createChoice == "Yes" {
				titlePrompt := promptui.Prompt{
					Label: "Prompt title",
					Validate: func(input string) error {
						if strings.TrimSpace(input) == "" {
							return fmt.Errorf("title cannot be empty")
						}
						return nil
					},
				}
				title, err := titlePrompt.Run()
				if err != nil {
					return "", nil, err
				}
				bodyPrompt := promptui.Prompt{
					Label: "Prompt instructions",
				}
				body, err := bodyPrompt.Run()
				if err != nil {
					return "", nil, err
				}
				defaultPromptSelect := promptui.Select{
					Label: "Set as default prompt?",
					Items: []string{"No", "Yes"},
				}
				_, defaultChoice, err := defaultPromptSelect.Run()
				if err != nil {
					return "", nil, err
				}
				setDefault := defaultChoice == "Yes"
				created, err := createPromptNote(brainInfo.Path, brainType, title, body, setDefault)
				if err != nil {
					return "", nil, err
				}
				content, err := loadPromptNoteContent(created.Path)
				if err != nil {
					return "", nil, err
				}
				return content, &created, nil
			}
		}
		return "", nil, nil
	}

	items := []string{"No prompt"}
	for _, prompt := range prompts {
		items = append(items, prompt.Title)
	}

	selector := promptui.Select{
		Label: "Select prompt note (optional)",
		Items: items,
		Size:  calculateMenuSize(len(items)),
	}
	idx, _, err := selector.Run()
	if err != nil {
		return "", nil, err
	}
	if idx == 0 {
		return "", nil, nil
	}

	selected := prompts[idx-1]
	content, err := loadPromptNoteContent(selected.Path)
	if err != nil {
		return "", nil, err
	}

	return content, &selected, nil
}

func applyPromptStack(systemPrompt string, promptContent string, promptExtra string) string {
	parts := []string{}
	if strings.TrimSpace(promptContent) != "" {
		parts = append(parts, strings.TrimSpace(promptContent))
	}
	if strings.TrimSpace(promptExtra) != "" {
		parts = append(parts, strings.TrimSpace(promptExtra))
	}
	parts = append(parts, systemPrompt)
	return strings.Join(parts, "\n\n")
}

func resolveAITitleAndFilename(topic string, title string, targetDir string, dateStr string, interactive bool) (string, string, error) {
	resolvedTitle := strings.TrimSpace(title)
	if resolvedTitle == "" {
		resolvedTitle = suggestTitleFromTopic(topic)
	}

	if interactive {
		prompt := promptui.Prompt{
			Label:   "Title",
			Default: resolvedTitle,
			Validate: func(input string) error {
				if strings.TrimSpace(input) == "" {
					return fmt.Errorf("title cannot be empty")
				}
				filename := aiFilenameFromTitle(strings.TrimSpace(input), dateStr)
				if aiFileExists(filepath.Join(targetDir, filename)) {
					return fmt.Errorf("a note with this title already exists")
				}
				return nil
			},
		}
		value, err := prompt.Run()
		if err != nil {
			return "", "", err
		}
		resolvedTitle = strings.TrimSpace(value)
		filename := aiFilenameFromTitle(resolvedTitle, dateStr)
		return resolvedTitle, filename, nil
	}

	slug := aiSlugFromTitle(resolvedTitle)
	filename := uniqueAIFilename(slug, dateStr, targetDir)
	return resolvedTitle, filename, nil
}

func suggestTitleFromTopic(topic string) string {
	cleaned := strings.TrimSpace(topic)
	if cleaned == "" {
		return "New Note"
	}

	for _, sep := range []string{".", "?", "!"} {
		if idx := strings.Index(cleaned, sep); idx > 15 {
			cleaned = cleaned[:idx]
			break
		}
	}

	words := strings.Fields(cleaned)
	if len(words) > 8 {
		cleaned = strings.Join(words[:8], " ")
	}

	cleaned = strings.Trim(cleaned, "\"'")
	if cleaned == "" {
		return "New Note"
	}
	return cleaned
}

func aiFilenameFromTitle(title string, dateStr string) string {
	slug := aiSlugFromTitle(title)
	if slug == "" {
		slug = "note"
	}
	return fmt.Sprintf("%s-%s.md", slug, dateStr)
}

func aiSlugFromTitle(title string) string {
	input := strings.ToLower(strings.TrimSpace(title))
	if input == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}

	slug := strings.Trim(b.String(), "-")
	if len(slug) > 40 {
		slug = slug[:40]
		slug = strings.Trim(slug, "-")
	}
	return slug
}

func uniqueAIFilename(slug string, dateStr string, targetDir string) string {
	base := fmt.Sprintf("%s-%s.md", slug, dateStr)
	if !aiFileExists(filepath.Join(targetDir, base)) {
		return base
	}
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d-%s.md", slug, i, dateStr)
		if !aiFileExists(filepath.Join(targetDir, candidate)) {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%s.md", slug, dateStr)
}

func aiFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func formatAILinkTitle(title string, action string, provider string) string {
	if provider == "" {
		return title
	}
	return fmt.Sprintf("%s (%s: %s)", title, action, provider)
}

func resolveBrainForFile(filePath string) (*Brain, brain.BrainType) {
	brainPath := detectBrainPath(filePath)
	if brainPath == "" {
		return nil, brain.BrainTypeFlip
	}

	detector := brain.NewDetector()
	detection, _ := detector.DetectBrainType(brainPath)

	ws, err := getActiveWorkspace()
	if err == nil {
		absBrainPath, _ := filepath.Abs(brainPath)
		for i := range ws.Brains {
			brainAbs, _ := filepath.Abs(ws.Brains[i].Path)
			if filepath.Clean(brainAbs) == filepath.Clean(absBrainPath) {
				return &ws.Brains[i], detection.Type
			}
		}
	}

	return &Brain{
		Name: filepath.Base(brainPath),
		Path: brainPath,
		Type: string(detection.Type),
	}, detection.Type
}
