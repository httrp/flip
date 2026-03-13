package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/httrp/flip/internal/brain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// FileInfo represents metadata about a file in a brain
type FileInfo struct {
	// File information
	Path     string `json:"path"`
	FileName string `json:"file_name"`
	Exists   bool   `json:"exists"`

	// Content type from frontmatter
	Type string `json:"type"` // exercise, note, task, plan, session, journal, meeting, unknown

	// Brain information
	BrainPath string `json:"brain_path,omitempty"`
	BrainName string `json:"brain_name,omitempty"`
	BrainType string `json:"brain_type,omitempty"` // flip, obsidian, logseq, dendron, foam

	// Additional frontmatter fields (useful for context)
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Context string `json:"context,omitempty"`

	// Error information
	Error string `json:"error,omitempty"`
}

// NewFileInfoCommand creates the file-info command
func NewFileInfoCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "file-info <filepath>",
		Short: "Get information about a file in a brain",
		Long: `Analyze a file and return information about its type, associated brain, and metadata.

This command reads the YAML frontmatter from a markdown file and returns:
- File type (exercise, note, task, plan, session, journal, meeting)
- Brain path and type (flip, obsidian, logseq, dendron, foam)
- Basic metadata (id, name, context)

Useful for:
- VS Code tasks that need context-aware behavior
- Scripts that operate on brain files
- Debugging and inspection

Examples:
  flip file-info ./exercises/guitar-scales.md
  flip file-info ~/brains/work/notes/meeting-notes.md --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info := getFileInfo(args[0])

			if jsonOutput {
				output, err := json.MarshalIndent(info, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				fmt.Println(string(output))
			} else {
				printFileInfo(info)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

// getFileInfo analyzes a file and returns its metadata
func getFileInfo(filePath string) *FileInfo {
	info := &FileInfo{
		Path:   filePath,
		Exists: false,
		Type:   "unknown",
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		info.Error = fmt.Sprintf("failed to resolve path: %v", err)
		return info
	}
	info.Path = absPath
	info.FileName = filepath.Base(absPath)

	// Check if file exists
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			info.Error = "file does not exist"
		} else {
			info.Error = fmt.Sprintf("failed to stat file: %v", err)
		}
		return info
	}

	if fileInfo.IsDir() {
		info.Error = "path is a directory, not a file"
		return info
	}

	info.Exists = true

	// Only process markdown files
	if !strings.HasSuffix(strings.ToLower(absPath), ".md") {
		info.Type = "non-markdown"
		return info
	}

	// Read file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		info.Error = fmt.Sprintf("failed to read file: %v", err)
		return info
	}

	// Extract frontmatter
	frontmatter, err := extractFrontmatter(content)
	if err != nil {
		// No frontmatter - try to infer type from path
		info.Type = inferTypeFromPath(absPath)
	} else {
		// Parse frontmatter to get type and other fields
		parseFrontmatterFields(frontmatter, info)

		// If frontmatter didn't have a type, try path-based inference
		if info.Type == "unknown" {
			info.Type = inferTypeFromPath(absPath)
		}
	}

	// Detect brain
	brainPath := detectBrainPath(absPath)
	if brainPath != "" {
		info.BrainPath = brainPath
		info.BrainName = filepath.Base(brainPath)

		// Detect brain type
		detector := brain.NewDetector()
		detection, err := detector.DetectBrainType(brainPath)
		if err == nil && detection.Compatible {
			info.BrainType = string(detection.Type)
		}
	}

	return info
}

// extractFrontmatter extracts YAML frontmatter from markdown content
func extractFrontmatter(content []byte) (string, error) {
	text := string(content)

	// Check if starts with ---
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		return "", fmt.Errorf("no frontmatter found")
	}

	// Find closing ---
	var rest string
	if strings.HasPrefix(text, "---\r\n") {
		rest = text[5:]
	} else {
		rest = text[4:]
	}

	// Try both Unix and Windows line endings
	endIdx := strings.Index(rest, "\n---\n")
	if endIdx == -1 {
		endIdx = strings.Index(rest, "\n---\r\n")
	}
	if endIdx == -1 {
		endIdx = strings.Index(rest, "\r\n---\r\n")
	}
	if endIdx == -1 {
		return "", fmt.Errorf("frontmatter not closed")
	}

	return rest[:endIdx], nil
}

// parseFrontmatterFields extracts relevant fields from frontmatter YAML
func parseFrontmatterFields(frontmatter string, info *FileInfo) {
	// Generic map to capture any frontmatter
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &data); err != nil {
		return
	}

	// Extract type
	if t, ok := data["type"].(string); ok {
		info.Type = strings.ToLower(t)
	}

	// Extract id
	if id, ok := data["id"].(string); ok {
		info.ID = id
	}

	// Extract name (or title as fallback)
	if name, ok := data["name"].(string); ok {
		info.Name = name
	} else if title, ok := data["title"].(string); ok {
		info.Name = title
	}

	// Extract context
	if ctx, ok := data["context"].(string); ok {
		info.Context = ctx
	}
}

// inferTypeFromPath tries to determine file type from its directory path
// This is used when frontmatter doesn't specify a type
func inferTypeFromPath(filePath string) string {
	// Normalize path separators
	normalized := strings.ToLower(filepath.ToSlash(filePath))

	// Check directory patterns (order matters - more specific first)
	patterns := []struct {
		pattern string
		typ     string
	}{
		{"/exercises/", "exercise"},
		{"/exercise/", "exercise"},
		{"/plans/", "plan"},
		{"/sessions/", "session"},
		{"/tasks/", "task"},
		{"/todo/", "task"},
		{"/prompts/", "prompt"},
		{"/notes/", "note"},
		{"/journal/", "journal"},
		{"/journals/", "journal"},
		{"/daily/", "journal"},
		{"/meetings/", "meeting"},
		{"/pages/", "note"}, // Logseq convention
	}

	for _, p := range patterns {
		if strings.Contains(normalized, p.pattern) {
			return p.typ
		}
	}

	return "unknown"
}

// detectBrainPath walks up the directory tree to find the brain root
func detectBrainPath(filePath string) string {
	dir := filepath.Dir(filePath)

	for {
		// Check for brain markers
		markers := []string{
			".flip-brain.yaml", // flip
			".obsidian",        // Obsidian
			".logseq",          // Logseq
			"dendron.yml",      // Dendron
			".foam",            // Foam
		}

		for _, marker := range markers {
			markerPath := filepath.Join(dir, marker)
			if _, err := os.Stat(markerPath); err == nil {
				return dir
			}
		}

		// Move up one level
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding a brain marker
			return ""
		}
		dir = parent
	}
}

// printFileInfo displays file info in human-readable format
func printFileInfo(info *FileInfo) {
	fmt.Println()
	fmt.Println("📄 File Information")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("   Path:      %s\n", info.Path)
	fmt.Printf("   File:      %s\n", info.FileName)
	fmt.Printf("   Exists:    %v\n", info.Exists)

	if info.Error != "" {
		fmt.Printf("   ❌ Error:   %s\n", info.Error)
		fmt.Println()
		return
	}

	// Type with icon
	typeIcon := getTypeIcon(info.Type)
	fmt.Printf("   Type:      %s %s\n", typeIcon, info.Type)

	if info.ID != "" {
		fmt.Printf("   ID:        %s\n", info.ID)
	}
	if info.Name != "" {
		fmt.Printf("   Name:      %s\n", info.Name)
	}
	if info.Context != "" {
		fmt.Printf("   Context:   %s\n", info.Context)
	}

	if info.BrainPath != "" {
		fmt.Println()
		fmt.Println("🧠 Brain")
		fmt.Println("────────────────────────────────────────────────────────────────────────")
		fmt.Printf("   Path:      %s\n", info.BrainPath)
		fmt.Printf("   Name:      %s\n", info.BrainName)
		if info.BrainType != "" {
			fmt.Printf("   Type:      %s\n", info.BrainType)
		}
	}

	fmt.Println()
}

// getTypeIcon returns an appropriate icon for a file type
func getTypeIcon(fileType string) string {
	switch fileType {
	case "exercise":
		return "🏋️"
	case "plan":
		return "📋"
	case "session":
		return "⏱️"
	case "note":
		return "📝"
	case "task":
		return "✅"
	case "journal":
		return "📔"
	case "meeting":
		return "👥"
	case "prompt":
		return "💡"
	case "non-markdown":
		return "📎"
	default:
		return "❓"
	}
}
