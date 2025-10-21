package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/templates"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func NewNoteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "note",
		Short: "Create a new note",
		Long:  "Create a new note in your active brain with appropriate template and naming conventions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateNote()
		},
	}
	return cmd
}

// runCreateNote creates a new note in the active brain
func runCreateNote() error {
	fmt.Println("\n📝 Create New Note")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Load config
	config, err := loadWorkspaceConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if we have any workspaces
	if len(config.Workspaces) == 0 {
		fmt.Println("⚠️  No workspaces found. Please run 'flip quickstart' first.")
		return nil
	}

	// Get active workspace
	activeWs, err := getActiveWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get active workspace: %w", err)
	}

	// Check if workspace has any brains
	if len(activeWs.Brains) == 0 {
		fmt.Println("⚠️  No brains found in active workspace. Please add a brain first.")
		return nil
	}

	// STEP 1: Confirm/select brain FIRST
	activeBrain, err := confirmOrSelectBrain(activeWs)
	if err != nil {
		return err
	}

	// Detect brain type early (for context)
	detector := brain.NewDetector()
	detection, err := detector.DetectBrainType(activeBrain.Path)
	if err != nil {
		return fmt.Errorf("failed to detect brain type: %w", err)
	}

	fmt.Printf("Brain type: %s\n\n", detection.Type)

	// STEP 2: Prompt for note title
	promptTitle := promptui.Prompt{
		Label: "Note title",
		Validate: func(input string) error {
			if strings.TrimSpace(input) == "" {
				return fmt.Errorf("title cannot be empty")
			}
			return nil
		},
	}

	title, err := promptTitle.Run()
	if err != nil {
		return fmt.Errorf("title prompt cancelled: %w", err)
	}
	title = strings.TrimSpace(title)

	// Generate filename based on brain type
	filename := generateNoteFilename(title, detection.Type)

	// Determine target directory
	targetDir := getNotesDirectory(activeBrain.Path, detection.Type)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create notes directory: %w", err)
	}

	// Full file path
	filePath := filepath.Join(targetDir, filename)

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		fmt.Printf("\n⚠️  File already exists: %s\n", filePath)
		return nil
	}

	// STEP 3: Show preview of what will be created
	fmt.Printf("\n📄 Will create: %s\n", filename)
	fmt.Printf("   Location: %s\n\n", targetDir)

	// Generate content based on brain type
	content := generateNoteContent(title, detection.Type)

	// STEP 4: Write file (only after title is confirmed)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	fmt.Printf("✓ Note created successfully!\n")
	fmt.Printf("  Path: %s\n", filePath)
	fmt.Printf("  Brain: %s (%s)\n\n", activeBrain.Name, detection.Type)

	// STEP 5: Ask if user wants to edit the note
	if err := promptAndOpenEditor(filePath); err != nil {
		// Don't fail if editor opening fails, note is already created
		fmt.Printf("⚠️  Could not open editor: %v\n", err)
	}

	return nil
}

// confirmOrSelectBrain shows all brains in workspace with default pre-selected
func confirmOrSelectBrain(ws *Workspace) (*Brain, error) {
	// If there's only one brain, use it directly
	if len(ws.Brains) == 1 {
		brain := &ws.Brains[0]
		fmt.Printf("📍 Using brain: %s (%s)\n", brain.Name, brain.Type)
		fmt.Printf("   Path: %s\n\n", brain.Path)
		return brain, nil
	}

	// Find default brain index
	defaultIdx := 0
	for i := range ws.Brains {
		if ws.Brains[i].Name == ws.DefaultBrain {
			defaultIdx = i
			break
		}
	}

	// Build list of all brains with type information
	brainItems := make([]string, len(ws.Brains))
	for i, b := range ws.Brains {
		if i == defaultIdx {
			// Mark default brain
			brainItems[i] = fmt.Sprintf("%s (%s) [default]", b.Name, b.Type)
		} else {
			brainItems[i] = fmt.Sprintf("%s (%s)", b.Name, b.Type)
		}
	}

	// Show selection with default pre-selected
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ . | cyan }}",
		Inactive: "  {{ . }}",
		Selected: "📍 {{ . | green }}",
	}

	promptBrain := promptui.Select{
		Label:     fmt.Sprintf("Select brain (workspace: %s)", ws.Name),
		Items:     brainItems,
		Templates: templates,
		CursorPos: defaultIdx, // Start at default brain
		Size:      10,
	}

	brainIdx, _, err := promptBrain.Run()
	if err != nil {
		return nil, fmt.Errorf("brain selection cancelled: %w", err)
	}

	selectedBrain := &ws.Brains[brainIdx]
	fmt.Printf("   Path: %s\n\n", selectedBrain.Path)

	return selectedBrain, nil
}

// generateNoteFilename creates a filename based on brain type conventions
func generateNoteFilename(title string, brainType brain.BrainType) string {
	// Sanitize title for filename
	safeName := strings.ToLower(title)

	// Replace spaces and special chars with hyphen, but keep structure
	safeName = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		// Replace any non-alphanumeric with hyphen
		return '-'
	}, safeName)

	// Clean up multiple consecutive hyphens
	for strings.Contains(safeName, "--") {
		safeName = strings.ReplaceAll(safeName, "--", "-")
	}

	// Trim leading/trailing hyphens
	safeName = strings.Trim(safeName, "-")

	now := time.Now()

	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: YYYY_MM_DD___title.md
		return fmt.Sprintf("%s___%s.md", now.Format("2006_01_02"), safeName)

	case brain.BrainTypeObsidian:
		// Obsidian: simple title.md (no date prefix usually)
		return fmt.Sprintf("%s.md", safeName)

	case brain.BrainTypeDendron:
		// Dendron: notes.YYYY-MM-DD-title.md
		return fmt.Sprintf("notes.%s-%s.md", now.Format("2006-01-02"), safeName)

	case brain.BrainTypeFlip:
		// Flip: YYYY-MM-DD-title.md
		return fmt.Sprintf("%s-%s.md", now.Format("2006-01-02"), safeName)

	default:
		// Generic: YYYY-MM-DD-title.md
		return fmt.Sprintf("%s-%s.md", now.Format("2006-01-02"), safeName)
	}
}

// getNotesDirectory returns the appropriate directory for notes based on brain type
func getNotesDirectory(brainPath string, brainType brain.BrainType) string {
	switch brainType {
	case brain.BrainTypeLogseq:
		// Logseq: pages/ directory
		return filepath.Join(brainPath, "pages")

	case brain.BrainTypeObsidian:
		// Obsidian: root or Notes/ if it exists
		notesDir := filepath.Join(brainPath, "Notes")
		if _, err := os.Stat(notesDir); err == nil {
			return notesDir
		}
		return brainPath

	case brain.BrainTypeDendron:
		// Dendron: root directory
		return brainPath

	case brain.BrainTypeFlip:
		// Flip: notes/ directory
		return filepath.Join(brainPath, "notes")

	default:
		// Generic: root directory
		return brainPath
	}
}

// generateNoteContent creates note content with appropriate frontmatter/metadata
func generateNoteContent(title string, brainType brain.BrainType) string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")

	// Try to load template from file
	tmpl, err := templates.Load(brainType, templates.TemplateTypeNote)
	if err != nil {
		// Fallback to hardcoded template if file not found
		fmt.Printf("Warning: Could not load template, using default (%v)\n", err)
		return generateDefaultNoteContent(title, brainType)
	}

	// Prepare template variables
	vars := map[string]string{
		"title":   title,
		"date":    dateStr,
		"tags":    "note",
		"id":      generateID(),
		"updated": fmt.Sprintf("%d", now.Unix()),
		"created": fmt.Sprintf("%d", now.Unix()),
	}

	return templates.Render(tmpl, vars)
}

// generateDefaultNoteContent provides fallback templates when template files don't exist
func generateDefaultNoteContent(title string, brainType brain.BrainType) string {
	now := time.Now()
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04")

	switch brainType {
	case brain.BrainTypeLogseq:
		return fmt.Sprintf(`- title:: %s
- created:: %s %s
- tags:: 

## %s

## Related
- [[related-note]]

## Tasks
- TODO Example task

`, title, dateStr, timeStr, title)

	case brain.BrainTypeObsidian:
		return fmt.Sprintf(`---
title: %s
created: %s
tags: []
---

# %s

## Related
- [[related-note]]

## Tasks
- [ ] Example task

`, title, dateStr, title)

	case brain.BrainTypeDendron:
		return fmt.Sprintf(`---
id: %s
title: %s
desc: ''
updated: %d
created: %d
---

# %s

## Related
- [[related-note]]

## Tasks
- [ ] Example task

`, generateID(), title, now.Unix(), now.Unix(), title)

	case brain.BrainTypeFlip:
		return fmt.Sprintf(`---
title: %s
created: %s
updated: %s
type: note
tags: []
---

# %s

## Related
- [[related-note]]

## Tasks
- [ ] Example task

`, title, dateStr, dateStr, title)

	default:
		return fmt.Sprintf(`---
title: %s
date: %s
---

# %s

## Related
- [[related-note]]

## Tasks
- [ ] Example task

`, title, dateStr, title)
	}
}

// generateID creates a unique ID for notes (used by Dendron)
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// promptAndOpenEditor asks user if they want to edit the note and opens appropriate editor
func promptAndOpenEditor(filePath string) error {
	promptEdit := promptui.Select{
		Label: "Open note in editor?",
		Items: []string{"Yes", "No"},
	}

	idx, _, err := promptEdit.Run()
	if err != nil {
		return err
	}

	if idx != 0 {
		// User chose "No"
		return nil
	}

	return openInEditor(filePath)
}

// openInEditor opens the file in the most appropriate editor
func openInEditor(filePath string) error {
	// Check if we're in VS Code terminal (TERM_PROGRAM env var)
	termProgram := os.Getenv("TERM_PROGRAM")
	vscodeIPC := os.Getenv("VSCODE_IPC_HOOK_CLI")

	// Priority 1: VS Code (if running in VS Code terminal)
	if termProgram == "vscode" || vscodeIPC != "" {
		if err := tryOpenInVSCode(filePath); err == nil {
			fmt.Println("📝 Opening in VS Code...")
			return nil
		}
	}

	// Priority 2: Try VS Code anyway (might be installed)
	if err := tryOpenInVSCode(filePath); err == nil {
		fmt.Println("📝 Opening in VS Code...")
		return nil
	}

	// Priority 3: System default editor via 'open' (macOS) or 'xdg-open' (Linux)
	if err := trySystemOpen(filePath); err == nil {
		fmt.Println("📝 Opening in default editor...")
		return nil
	}

	// Priority 4: EDITOR environment variable
	if editor := os.Getenv("EDITOR"); editor != "" {
		if err := tryEditor(editor, filePath); err == nil {
			fmt.Printf("📝 Opening in %s...\n", editor)
			return nil
		}
	}

	// Priority 5: Common CLI editors
	for _, editor := range []string{"nano", "vim", "vi"} {
		if err := tryEditor(editor, filePath); err == nil {
			fmt.Printf("📝 Opening in %s...\n", editor)
			return nil
		}
	}

	return fmt.Errorf("no suitable editor found")
}

// tryOpenInVSCode attempts to open file in VS Code
func tryOpenInVSCode(filePath string) error {
	// Try 'code' command
	cmd := exec.Command("code", filePath)
	return cmd.Run()
}

// trySystemOpen uses system default (macOS 'open', Linux 'xdg-open')
func trySystemOpen(filePath string) error {
	// Try platform-specific commands
	// macOS
	cmd := exec.Command("open", filePath)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Linux
	cmd = exec.Command("xdg-open", filePath)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Windows
	cmd = exec.Command("cmd", "/c", "start", filePath)
	if err := cmd.Run(); err == nil {
		return nil
	}

	return fmt.Errorf("could not open file with system default")
}

// tryEditor tries to open file with a specific editor
func tryEditor(editor, filePath string) error {
	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
