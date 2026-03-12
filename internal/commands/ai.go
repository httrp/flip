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
  FLIP_AI_PROVIDER=ollama|openai|anthropic|azure
  OLLAMA_HOST=http://localhost:11434 (for Ollama)
  OPENAI_API_KEY=sk-... (for OpenAI)
  ANTHROPIC_API_KEY=sk-ant-... (for Anthropic)`,
	}

	cmd.AddCommand(newAISummarizeCommand())
	cmd.AddCommand(newAIResearchCommand())
	cmd.AddCommand(newAIImproveCommand())
	cmd.AddCommand(newAIStatusCommand())

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
		}
	}

	fmt.Println()
	return nil
}

// newAISummarizeCommand creates notes summarizing content from brains
func newAISummarizeCommand() *cobra.Command {
	var (
		topic    string
		brains   []string
		output   string
		noStream bool
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
			return runAISummarize(topic, brains, output, !noStream)
		},
	}

	cmd.Flags().StringSliceVarP(&brains, "brains", "b", nil, "Specific brains to search (default: all)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: auto-generated)")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")

	return cmd
}

func runAISummarize(topic string, brainNames []string, outputPath string, stream bool) error {
	// Get topic interactively if not provided
	if topic == "" {
		prompt := promptui.Prompt{
			Label: "What topic do you want to summarize?",
		}
		var err error
		topic, err = prompt.Run()
		if err != nil {
			return err
		}
		if topic == "" {
			return fmt.Errorf("topic is required")
		}
	}

	fmt.Printf("\n🔍 Searching for notes about: %s\n", topic)

	// Get brains to search
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
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

			// Limit to avoid token overflow
			if relevantContent.Len() > 50000 {
				break
			}
		}
	}

	if noteCount == 0 {
		fmt.Println("❌ No notes found matching the topic")
		return nil
	}

	fmt.Printf("📄 Found %d relevant notes\n\n", noteCount)

	// Create AI client
	client, err := ai.NewClient()
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

	userPrompt := fmt.Sprintf("Please summarize the following notes about \"%s\":\n\n%s", topic, relevantContent.String())

	ctx := context.Background()
	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Temperature: 0.3, // Lower temperature for summarization
		MaxTokens:   4096,
	}

	fmt.Println("🤖 Generating summary...")
	fmt.Println()

	var result strings.Builder

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
			fmt.Print(chunk)
			result.WriteString(chunk)
		}
		fmt.Println()
	} else {
		// Non-streaming response
		resp, err := client.Complete(ctx, req)
		if err != nil {
			return fmt.Errorf("AI error: %w", err)
		}
		fmt.Println(resp.Content)
		result.WriteString(resp.Content)
	}

	// Save to file
	if outputPath == "" {
		// Generate filename
		slug := strings.ToLower(strings.ReplaceAll(topic, " ", "-"))
		if len(slug) > 30 {
			slug = slug[:30]
		}
		outputPath = fmt.Sprintf("summary-%s-%s.md", slug, time.Now().Format("2006-01-02"))
	}

	// Get default brain for output
	defaultBrain := aiGetDefaultBrain(activeWs)
	if defaultBrain != nil {
		outputPath = filepath.Join(defaultBrain.Path, "notes", outputPath)
	}

	// Create frontmatter
	frontmatter := fmt.Sprintf(`---
title: "Summary: %s"
date: %s
type: summary
topic: "%s"
sources: %d notes
ai_generated: true
---

`, topic, time.Now().Format("2006-01-02"), topic, noteCount)

	// Write file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(frontmatter+result.String()), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("\n\n✅ Summary saved to: %s\n", outputPath)

	return nil
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
		noStream bool
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
			return runAIResearch(topic, output, !noStream)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: auto-generated)")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")

	return cmd
}

func runAIResearch(topic string, outputPath string, stream bool) error {
	// Get topic interactively if not provided
	if topic == "" {
		prompt := promptui.Prompt{
			Label: "What topic do you want to research?",
		}
		var err error
		topic, err = prompt.Run()
		if err != nil {
			return err
		}
		if topic == "" {
			return fmt.Errorf("topic is required")
		}
	}

	fmt.Printf("\n🔬 Researching: %s\n\n", topic)

	// Create AI client
	client, err := ai.NewClient()
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

	userPrompt := fmt.Sprintf("Please provide a comprehensive overview of: %s\n\nInclude key concepts, practical examples, and best practices.", topic)

	ctx := context.Background()
	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
		Temperature: 0.5,
		MaxTokens:   4096,
	}

	fmt.Println("🤖 Generating research note...")
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

	// Save to file
	if outputPath == "" {
		slug := strings.ToLower(strings.ReplaceAll(topic, " ", "-"))
		slug = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				return r
			}
			return '-'
		}, slug)
		if len(slug) > 40 {
			slug = slug[:40]
		}
		outputPath = fmt.Sprintf("research-%s-%s.md", slug, time.Now().Format("2006-01-02"))
	}

	// Get default brain for output
	activeWs, _ := getActiveWorkspace()
	if activeWs != nil {
		defaultBrain := aiGetDefaultBrain(activeWs)
		if defaultBrain != nil {
			outputPath = filepath.Join(defaultBrain.Path, "notes", outputPath)
		}
	}

	// Create frontmatter
	frontmatter := fmt.Sprintf(`---
title: "%s"
date: %s
type: research
topic: "%s"
ai_generated: true
---

`, topic, time.Now().Format("2006-01-02"), topic)

	// Write file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(frontmatter+result.String()), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("\n\n✅ Research note saved to: %s\n", outputPath)

	return nil
}

// newAIImproveCommand improves existing notes with AI
func newAIImproveCommand() *cobra.Command {
	var (
		instruction string
		noStream    bool
		inPlace     bool
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
			return runAIImprove(args[0], instruction, !noStream, inPlace)
		},
	}

	cmd.Flags().StringVarP(&instruction, "instruction", "i", "", "Specific improvement instructions")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output")
	cmd.Flags().BoolVar(&inPlace, "in-place", false, "Update the file in place")

	return cmd
}

func runAIImprove(filePath string, instruction string, stream bool, inPlace bool) error {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
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

	// Create AI client
	client, err := ai.NewClient()
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

	userPrompt := fmt.Sprintf("Please improve the following note according to this instruction: %s\n\n---\n\n%s", instruction, string(content))

	ctx := context.Background()
	req := &ai.CompletionRequest{
		System: systemPrompt,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: userPrompt},
		},
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
