package commands

// ai_helpers.go - Shared helpers for AI commands
//
// Contains types, utilities, and shared logic used across
// summarize, research, and improve commands.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/ai"
	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/ui"
)

// summarySourceNote represents a note used as a source for AI summarization or research.
type summarySourceNote struct {
	BrainName string
	BrainPath string
	NotePath  string
	RelPath   string
	Title     string
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

	lines := []string{"", "", "## Template Context"}
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
		lines = append(lines, fmt.Sprintf("- Template note: [%s](%s)", label, linkPath))
	}

	extra := strings.TrimSpace(promptExtra)
	if extra != "" {
		lines = append(lines, "", "### Run Notes", "", extra)
	}
	if promptNote == nil && extra == "" {
		lines = append(lines, "- None")
	}

	return strings.Join(lines, "\n")
}

func loadContextFiles(filePaths []string, targetBrain *Brain) ([]summarySourceNote, string, error) {
	if len(filePaths) == 0 {
		return nil, "", nil
	}

	var notes []summarySourceNote
	var content strings.Builder
	seen := make(map[string]bool)

	for _, raw := range filePaths {
		candidate := strings.TrimSpace(raw)
		if candidate == "" {
			continue
		}

		abs := candidate
		if !filepath.IsAbs(abs) {
			resolved, err := filepath.Abs(abs)
			if err != nil {
				return nil, "", fmt.Errorf("failed to resolve context file '%s': %w", candidate, err)
			}
			abs = resolved
		}

		if seen[abs] {
			continue
		}
		seen[abs] = true

		fileContent, err := os.ReadFile(abs)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read context file '%s': %w", candidate, err)
		}

		content.WriteString(fmt.Sprintf("\n--- %s ---\n", filepath.Base(abs)))
		content.Write(fileContent)
		content.WriteString("\n")

		note := summarySourceNote{
			NotePath: abs,
			Title:    strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs)),
		}
		if targetBrain != nil {
			relPath, err := filepath.Rel(targetBrain.Path, abs)
			if err == nil && !strings.HasPrefix(relPath, "..") {
				note.BrainName = targetBrain.Name
				note.BrainPath = targetBrain.Path
				note.RelPath = relPath
			}
		}
		notes = append(notes, note)
	}

	return notes, strings.TrimSpace(content.String()), nil
}

func collectAutoContextFromBrain(targetBrain *Brain, request string) ([]summarySourceNote, string, error) {
	if targetBrain == nil {
		return nil, "", nil
	}

	notes, err := searchNotesInBrain(targetBrain.Path, request)
	if err != nil {
		return nil, "", err
	}
	if len(notes) == 0 {
		return nil, "", nil
	}

	var sourceNotes []summarySourceNote
	var content strings.Builder

	maxNotes := len(notes)
	if maxNotes > 6 {
		maxNotes = 6
	}

	for i := 0; i < maxNotes; i++ {
		notePath := notes[i]
		noteContent, readErr := os.ReadFile(notePath)
		if readErr != nil {
			continue
		}

		relPath, _ := filepath.Rel(targetBrain.Path, notePath)
		content.WriteString(fmt.Sprintf("\n--- %s/%s ---\n", targetBrain.Name, relPath))
		content.Write(noteContent)
		content.WriteString("\n")

		sourceNotes = append(sourceNotes, summarySourceNote{
			BrainName: targetBrain.Name,
			BrainPath: targetBrain.Path,
			NotePath:  notePath,
			RelPath:   relPath,
			Title:     strings.TrimSuffix(filepath.Base(notePath), filepath.Ext(notePath)),
		})

		if content.Len() > 30000 {
			break
		}
	}

	return sourceNotes, strings.TrimSpace(content.String()), nil
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
			createYes, err := ui.RunConfirm("No prompt notes found. Create one?", false)
			if err != nil {
				return "", nil, err
			}
			if createYes {
				title, err := ui.RunInput("Prompt title", "", "", func(input string) error {
					if strings.TrimSpace(input) == "" {
						return fmt.Errorf("title cannot be empty")
					}
					return nil
				})
				if err != nil {
					return "", nil, err
				}
				body, err := ui.RunInput("Prompt instructions", "", "", nil)
				if err != nil {
					return "", nil, err
				}
				setDefault, err := ui.RunConfirm("Set as default prompt?", false)
				if err != nil {
					return "", nil, err
				}
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

	selectItems := []ui.SelectItem{{Label: "No prompt", Value: "__none__"}}
	for _, prompt := range prompts {
		selectItems = append(selectItems, ui.SelectItem{Label: prompt.Title, Value: prompt.Title})
	}

	_, choice, err := ui.RunSelect("Select prompt note (optional)", selectItems, calculateMenuSize(len(selectItems)))
	if err != nil {
		return "", nil, err
	}
	if choice == "__none__" {
		return "", nil, nil
	}

	// Find selected prompt by title
	var selected PromptNoteInfo
	for _, p := range prompts {
		if p.Title == choice {
			selected = p
			break
		}
	}
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

func resolveAITitleAndFilename(topic string, title string, targetDir string, interactive bool) (string, string, error) {
	resolvedTitle := strings.TrimSpace(title)
	if resolvedTitle == "" {
		resolvedTitle = suggestTitleFromTopic(topic)
	}

	if interactive {
		value, err := ui.RunInput("Title", "", resolvedTitle, func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("title cannot be empty")
			}
			filename := aiFilenameFromTitle(strings.TrimSpace(input))
			if aiFileExists(filepath.Join(targetDir, filename)) {
				return fmt.Errorf("a note with this title already exists")
			}
			return nil
		})
		if err != nil {
			return "", "", err
		}
		resolvedTitle = strings.TrimSpace(value)
		filename := aiFilenameFromTitle(resolvedTitle)
		return resolvedTitle, filename, nil
	}

	slug := aiSlugFromTitle(resolvedTitle)
	filename := uniqueAIFilename(slug, targetDir)
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

func aiFilenameFromTitle(title string) string {
	slug := aiSlugFromTitle(title)
	if slug == "" {
		slug = "note"
	}
	return fmt.Sprintf("%s.md", slug)
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

func uniqueAIFilename(slug string, targetDir string) string {
	base := fmt.Sprintf("%s.md", slug)
	if !aiFileExists(filepath.Join(targetDir, base)) {
		return base
	}
	for i := 2; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d.md", slug, i)
		if !aiFileExists(filepath.Join(targetDir, candidate)) {
			return candidate
		}
	}
	return fmt.Sprintf("%s.md", slug)
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

func resolveAIModelForRequest(ctx context.Context, client *ai.Client, cfg *ai.Config, modelOverride string) (string, error) {
	modelOverride = strings.TrimSpace(modelOverride)
	if modelOverride != "" {
		return modelOverride, nil
	}

	// For all providers: list available models and let user pick interactively
	models, err := client.ListModels(ctx)
	if err != nil {
		// If listing fails, fall back to default
		return "", nil
	}

	if len(models) == 0 {
		if cfg.Provider == "ollama" {
			return "", fmt.Errorf("AI error: no Ollama models installed. Run `flip ai setup` (or `ollama pull %s`) and try again", cfg.Model)
		}
		return "", nil
	}

	// For Ollama: check if configured model exists, fallback if not
	if cfg.Provider == "ollama" {
		configured := strings.TrimSpace(cfg.Model)
		if configured == "" {
			return firstModelID(models), nil
		}

		found := false
		for _, m := range models {
			if strings.EqualFold(m.ID, configured) || strings.EqualFold(m.Name, configured) {
				found = true
				break
			}
		}
		if !found {
			fallback := firstModelID(models)
			if fallback != "" && !JSONOutput {
				fmt.Printf("⚠️  Ollama model '%s' not found locally, using '%s'\n", configured, fallback)
			}
			if fallback != "" {
				return fallback, nil
			}
		}
	}

	// Interactive model selection for all providers (when not in JSON mode)
	if !JSONOutput && len(models) > 1 {
		items := make([]string, len(models))
		configuredModel := strings.TrimSpace(cfg.Model)
		for i, m := range models {
			desc := m.Name
			if m.Description != "" {
				desc = fmt.Sprintf("%s – %s", m.Name, m.Description)
			}
			if m.ContextSize > 0 {
				desc = fmt.Sprintf("%s (%dk ctx)", desc, m.ContextSize/1000)
			}
			items[i] = desc
			if strings.EqualFold(m.ID, configuredModel) || strings.EqualFold(m.Name, configuredModel) {
				_ = i // future: use as default index
			}
		}

		selectItems := make([]ui.SelectItem, len(items))
		for i, item := range items {
			selectItems[i] = ui.SelectItem{Label: item, Value: models[i].ID}
		}
		_, choice, err := ui.RunSelect("Select model", selectItems, 10)
		if err != nil {
			return "", err
		}
		return choice, nil
	}

	return "", nil
}

func firstModelID(models []ai.ModelInfo) string {
	for _, m := range models {
		if strings.TrimSpace(m.ID) != "" {
			return m.ID
		}
		if strings.TrimSpace(m.Name) != "" {
			return m.Name
		}
	}
	return ""
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

// selectAndDetectBrain selects a target brain by name (or interactively) and detects its type.
// This is shared by summarize and research commands.
func selectAndDetectBrain(ws *Workspace, brainName string) (*Brain, *brain.DetectionResult, error) {
	var targetBrain *Brain
	if brainName != "" {
		for i := range ws.Brains {
			if ws.Brains[i].Name == brainName {
				targetBrain = &ws.Brains[i]
				break
			}
		}
		if targetBrain == nil {
			return nil, nil, fmt.Errorf("brain not found: %s", brainName)
		}
	} else {
		var err error
		targetBrain, err = confirmOrSelectBrain(ws)
		if err != nil {
			return nil, nil, err
		}
	}

	detector := brain.NewDetector()
	detection, _ := detector.DetectBrainType(targetBrain.Path)
	return targetBrain, detection, nil
}

// setupAIClientAndModel creates an AI client (unless prepareOnly) and resolves the effective model.
func setupAIClientAndModel(cfg *ai.Config, modelOverride string, prepareOnly bool) (*ai.Client, string, context.Context, error) {
	ctx := context.Background()
	if prepareOnly {
		return nil, modelOverride, ctx, nil
	}
	client, err := ai.NewClientWithConfig(cfg)
	if err != nil {
		return nil, "", ctx, fmt.Errorf("failed to create AI client: %w", err)
	}
	effectiveModel, err := resolveAIModelForRequest(ctx, client, cfg, modelOverride)
	if err != nil {
		return nil, "", ctx, err
	}
	return client, effectiveModel, ctx, nil
}

// buildPromptMeta builds the frontmatter lines for prompt_note and prompt_extra.
func buildPromptMeta(promptNote *PromptNoteInfo, promptExtra string, brainPath string) string {
	var meta string
	if promptNote != nil {
		relPrompt := relativePathFromBrain(promptNote.Path, brainPath)
		meta = fmt.Sprintf("prompt_note: \"%s\"\n", filepath.ToSlash(relPrompt))
	}
	if strings.TrimSpace(promptExtra) != "" {
		meta += "prompt_extra: true\n"
	}
	return meta
}

// resolveAIOutputPath resolves the note title, filename and full output path.
// Returns noteTitle, resolvedOutputPath, noteDate.
func resolveAIOutputPath(topic, title, outputPath, brainPath string, interactive bool) (string, string, string, error) {
	noteDate := filepath.Base(brainPath) // dummy; overridden below
	_ = noteDate
	noteDir := filepath.Join(brainPath, "notes")
	noteTitle, filename, err := resolveAITitleAndFilename(topic, title, noteDir, interactive)
	if err != nil {
		return "", "", "", err
	}
	resolved := outputPath
	if resolved == "" {
		resolved = filepath.Join(brainPath, "notes", filename)
	} else if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(brainPath, "notes", resolved)
	}
	return noteTitle, resolved, filename, nil
}

// addAIJournalLink adds a link to today's journal for the generated AI note.
func addAIJournalLink(outputPath string, targetBrain *Brain, brainType brain.BrainType, noteTitle, action, provider string, noLink, interactive bool) {
	if noLink {
		return
	}
	relPath := relativePathFromBrain(outputPath, targetBrain.Path)
	linkTitle := formatAILinkTitle(noteTitle, action, provider)
	if err := AddLinkToJournal(JournalLinkOptions{
		ItemType:    "note",
		ItemName:    linkTitle,
		ItemPath:    relPath,
		Brain:       targetBrain,
		Interactive: interactive,
		BrainType:   brainType,
	}); err != nil {
		if !JSONOutput {
			fmt.Printf("⚠️  Could not add journal link: %v\n", err)
		}
	}
}

// EnsureAISetup checks if an AI provider and model are configured,
// and prompts the user to set it up if they aren't.
func EnsureAISetup(jsonOutput bool) error {
	cfg := ai.LoadConfig()
	if cfg.Provider == "" || cfg.Model == "" {
		if jsonOutput {
			// Trigger VS Code extension to show setup UI
			OutputJSONError("ai check", fmt.Errorf("AI not set up. Run 'flip ai setup'"))
			return fmt.Errorf("setup required")
		}

		fmt.Println("⚠️  AI provider and model are not fully configured yet.")

		yes, err := ui.RunConfirm("Would you like to run the setup now?", true)
		if err != nil || !yes {
			return fmt.Errorf("AI configuration is required to use this command")
		}

		return runAISetup("", "", "", "", false, false)
	}
	return nil
}
