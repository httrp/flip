package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/templates"
	"gopkg.in/yaml.v3"
)

// PromptNoteInfo represents a prompt note stored in a brain.
type PromptNoteInfo struct {
	Name      string
	Title     string
	Path      string
	RelPath   string
	IsDefault bool
}

func getAITemplateDirectory(brainPath string) string {
	return filepath.Join(brainPath, "templates", "ai")
}

func getPromptSearchDirectories(brainPath string, brainType brain.BrainType) []string {
	_ = brainType
	return []string{getAITemplateDirectory(brainPath)}
}

func findPromptNotes(brainPath string, brainType brain.BrainType) ([]PromptNoteInfo, *PromptNoteInfo, error) {
	searchDirs := getPromptSearchDirectories(brainPath, brainType)
	var prompts []PromptNoteInfo
	seenPaths := make(map[string]bool)
	seenNames := make(map[string]bool)

	for _, promptsDir := range searchDirs {
		info, err := os.Stat(promptsDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, nil, err
		}
		if !info.IsDir() {
			continue
		}

		err = filepath.Walk(promptsDir, func(path string, fileInfo os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return nil
			}

			if fileInfo.IsDir() {
				name := fileInfo.Name()
				if strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(fileInfo.Name()), ".md") {
				return nil
			}

			absPath, absErr := filepath.Abs(path)
			if absErr != nil {
				absPath = path
			}
			if seenPaths[absPath] {
				return nil
			}

			metaTitle, isDefault := readPromptNoteMeta(path)
			name := strings.TrimSuffix(fileInfo.Name(), filepath.Ext(fileInfo.Name()))
			nameKey := strings.ToLower(name)
			if seenNames[nameKey] {
				return nil
			}
			seenNames[nameKey] = true
			seenPaths[absPath] = true

			title := metaTitle
			if title == "" {
				title = name
			}

			relPath, _ := filepath.Rel(brainPath, path)
			prompts = append(prompts, PromptNoteInfo{
				Name:      name,
				Title:     title,
				Path:      path,
				RelPath:   relPath,
				IsDefault: isDefault || strings.EqualFold(name, "default"),
			})

			return nil
		})
		if err != nil {
			return nil, nil, err
		}
	}

	sort.Slice(prompts, func(i, j int) bool {
		return strings.ToLower(prompts[i].Title) < strings.ToLower(prompts[j].Title)
	})

	var defaultNote *PromptNoteInfo
	for i := range prompts {
		if prompts[i].IsDefault {
			note := prompts[i]
			defaultNote = &note
			break
		}
	}

	return prompts, defaultNote, nil
}

func readPromptNoteMeta(filePath string) (string, bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}

	frontmatter, err := extractFrontmatter(content)
	if err != nil {
		return "", false
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &data); err != nil {
		return "", false
	}

	title := ""
	if rawTitle, ok := data["title"].(string); ok {
		title = rawTitle
	} else if rawName, ok := data["name"].(string); ok {
		title = rawName
	}

	isDefault := false
	if rawDefault, ok := data["prompt_default"]; ok {
		isDefault = isTruthyValue(rawDefault)
	}
	if rawDefault, ok := data["default"]; ok {
		isDefault = isDefault || isTruthyValue(rawDefault)
	}

	return title, isDefault
}

func loadPromptNoteContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	text := stripFrontmatter(string(content))
	return strings.TrimSpace(text), nil
}

func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return content
	}

	rest := ""
	if strings.HasPrefix(content, "---\r\n") {
		rest = content[5:]
	} else {
		rest = content[4:]
	}

	markers := []string{"\n---\n", "\n---\r\n", "\r\n---\r\n"}
	for _, marker := range markers {
		if idx := strings.Index(rest, marker); idx != -1 {
			return strings.TrimLeft(rest[idx+len(marker):], "\r\n")
		}
	}

	return content
}

func isTruthyValue(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true") || strings.EqualFold(strings.TrimSpace(v), "yes")
	default:
		return false
	}
}

func resolvePromptOverride(override string, brainPath string, brainType brain.BrainType, prompts []PromptNoteInfo) (PromptNoteInfo, bool, error) {
	override = strings.TrimSpace(override)
	if override == "" {
		return PromptNoteInfo{}, false, nil
	}

	lower := strings.ToLower(override)
	if lower == "none" || lower == "off" || lower == "no" {
		return PromptNoteInfo{}, true, nil
	}

	if filepath.IsAbs(override) {
		if _, err := os.Stat(override); err == nil {
			return PromptNoteInfo{Path: override}, true, nil
		}
		return PromptNoteInfo{}, false, fmt.Errorf("prompt note not found: %s", override)
	}

	candidate := override
	if !strings.HasSuffix(strings.ToLower(candidate), ".md") {
		candidate += ".md"
	}

	// Try direct brain-relative path first (e.g. templates/ai/foo.md)
	brainRelCandidate := filepath.Join(brainPath, candidate)
	if _, err := os.Stat(brainRelCandidate); err == nil {
		return PromptNoteInfo{Path: brainRelCandidate}, true, nil
	}

	for _, promptsDir := range getPromptSearchDirectories(brainPath, brainType) {
		pathCandidate := filepath.Join(promptsDir, candidate)
		if _, err := os.Stat(pathCandidate); err == nil {
			return PromptNoteInfo{Path: pathCandidate}, true, nil
		}
	}

	for _, prompt := range prompts {
		if strings.EqualFold(prompt.Name, override) || strings.EqualFold(prompt.Title, override) {
			return prompt, true, nil
		}
	}

	return PromptNoteInfo{}, false, fmt.Errorf("prompt note not found: %s", override)
}

func createPromptNote(brainPath string, brainType brain.BrainType, title string, body string, setDefault bool) (PromptNoteInfo, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return PromptNoteInfo{}, fmt.Errorf("prompt title is required")
	}

	filename := sanitizeFilename(title) + ".md"
	promptsDir := getAITemplateDirectory(brainPath)
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return PromptNoteInfo{}, err
	}

	filePath := filepath.Join(promptsDir, filename)
	if _, err := os.Stat(filePath); err == nil {
		return PromptNoteInfo{}, fmt.Errorf("prompt note already exists: %s", filePath)
	}

	content := renderPromptTemplate(brainType, title, body, setDefault)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return PromptNoteInfo{}, err
	}

	relPath, _ := filepath.Rel(brainPath, filePath)
	return PromptNoteInfo{
		Name:      strings.TrimSuffix(filename, ".md"),
		Title:     title,
		Path:      filePath,
		RelPath:   relPath,
		IsDefault: setDefault,
	}, nil
}

func renderPromptTemplate(brainType brain.BrainType, title string, body string, setDefault bool) string {
	now := time.Now()
	vars := map[string]string{
		"title":   title,
		"date":    now.Format("2006-01-02"),
		"id":      uuid.New().String(),
		"created": fmt.Sprintf("%d", now.Unix()),
		"updated": fmt.Sprintf("%d", now.Unix()),
	}

	tmpl, _ := templates.Load(brainType, templates.TemplateTypePrompt)
	content := templates.Render(tmpl, vars)
	if setDefault {
		content = insertPromptDefault(content, brainType)
	}

	body = strings.TrimSpace(body)
	if body != "" {
		content = strings.TrimRight(content, "\n") + "\n\n" + body + "\n"
	}

	return content
}

func insertPromptDefault(content string, brainType brain.BrainType) string {
	if brainType == brain.BrainTypeLogseq {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "- title::") {
				insertIdx := i + 1
				lines = append(lines[:insertIdx], append([]string{"- prompt_default:: true"}, lines[insertIdx:]...)...)
				return strings.Join(lines, "\n")
			}
		}
		return "- prompt_default:: true\n" + content
	}

	if strings.HasPrefix(content, "---") {
		frontmatter, err := extractFrontmatter([]byte(content))
		if err != nil {
			return content
		}
		updated := frontmatter + "\nprompt_default: true\n"
		return strings.Replace(content, frontmatter, updated, 1)
	}

	return content
}
