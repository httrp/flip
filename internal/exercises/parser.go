package exercises

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Parser handles reading and writing exercise-related markdown files
type Parser struct{}

// NewParser creates a new exercise parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseExercise reads an exercise definition from a markdown file
func (p *Parser) ParseExercise(filePath string) (*Exercise, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read exercise file: %w", err)
	}

	frontmatter, body, err := p.extractFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	var exercise Exercise
	if err := yaml.Unmarshal([]byte(frontmatter), &exercise); err != nil {
		return nil, fmt.Errorf("failed to unmarshal exercise frontmatter: %w", err)
	}

	exercise.FilePath = filePath
	exercise.BrainPath = p.getBrainPath(filePath)

	// Parse body for additional context (optional)
	_ = body // Future: extract goals, materials from markdown body

	return &exercise, nil
}

// ParseSession reads a session from a markdown file
func (p *Parser) ParseSession(filePath string) (*ExerciseSession, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	frontmatter, body, err := p.extractFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	var session ExerciseSession
	if err := yaml.Unmarshal([]byte(frontmatter), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session frontmatter: %w", err)
	}

	session.FilePath = filePath
	session.BrainPath = p.getBrainPath(filePath)

	// Parse body for notes if not in frontmatter
	if session.Notes == "" && len(body) > 0 {
		session.Notes = strings.TrimSpace(body)
	}

	return &session, nil
}

// ParsePlan reads a plan from a markdown file
func (p *Parser) ParsePlan(filePath string) (*ExercisePlan, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	frontmatter, _, err := p.extractFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	var plan ExercisePlan
	if err := yaml.Unmarshal([]byte(frontmatter), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan frontmatter: %w", err)
	}

	plan.FilePath = filePath
	plan.BrainPath = p.getBrainPath(filePath)

	return &plan, nil
}

// ParsePlanSession reads a plan session from a markdown file
func (p *Parser) ParsePlanSession(filePath string) (*PlanSession, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan session file: %w", err)
	}

	frontmatter, _, err := p.extractFrontmatter(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	var planSession PlanSession
	if err := yaml.Unmarshal([]byte(frontmatter), &planSession); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan session frontmatter: %w", err)
	}

	planSession.FilePath = filePath
	planSession.BrainPath = p.getBrainPath(filePath)

	return &planSession, nil
}

// WriteExercise writes an exercise to a markdown file
func (p *Parser) WriteExercise(exercise *Exercise, filePath string) error {
	// Prepare frontmatter
	frontmatter, err := yaml.Marshal(exercise)
	if err != nil {
		return fmt.Errorf("failed to marshal exercise: %w", err)
	}

	// Build markdown content
	var content strings.Builder
	content.WriteString("---\n")
	content.Write(frontmatter)
	content.WriteString("---\n\n")
	content.WriteString(fmt.Sprintf("# %s\n\n", exercise.Name))

	if exercise.Description != "" {
		content.WriteString("## Description\n")
		content.WriteString(exercise.Description + "\n\n")
	}

	if exercise.Goal != "" {
		content.WriteString("## Goal\n")
		content.WriteString(exercise.Goal + "\n\n")
	}

	if len(exercise.Materials) > 0 {
		content.WriteString("## Materials\n")
		for _, mat := range exercise.Materials {
			if mat.URL != "" {
				content.WriteString(fmt.Sprintf("- [%s](%s)\n", mat.Title, mat.URL))
			} else if mat.Path != "" {
				content.WriteString(fmt.Sprintf("- [[%s]]\n", mat.Path))
			}
		}
		content.WriteString("\n")
	}

	// Variants
	if len(exercise.Variants) > 0 {
		content.WriteString("## Variants\n\n")
		for i, variant := range exercise.Variants {
			if variant.Name != "" {
				content.WriteString(fmt.Sprintf("### Variant %d: %s\n\n", i+1, variant.Name))
			} else {
				content.WriteString(fmt.Sprintf("### Variant %d\n\n", i+1))
			}
			
			if variant.Description != "" {
				content.WriteString(variant.Description + "\n\n")
			}
			
			if len(variant.TrackingProperties) > 0 {
				content.WriteString("**Tracking Properties:**\n")
				for key, value := range variant.TrackingProperties {
					content.WriteString(fmt.Sprintf("- **%s**: %s\n", key, value))
				}
				content.WriteString("\n")
			}
		}
	}

	// Write file
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write exercise file: %w", err)
	}

	return nil
}

// WriteSession writes a session to a markdown file
func (p *Parser) WriteSession(session *ExerciseSession, filePath string) error {
	// Prepare frontmatter
	frontmatter, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Build markdown content
	var content strings.Builder
	content.WriteString("---\n")
	content.Write(frontmatter)
	content.WriteString("---\n\n")
	content.WriteString(fmt.Sprintf("# Session: %s - %s\n\n", session.ExerciseID, session.Date.Format("2006-01-02")))

	if session.Duration > 0 {
		content.WriteString(fmt.Sprintf("**Duration**: %d min  \n", session.Duration))
	}
	
	// Write custom properties
	if len(session.Properties) > 0 {
		for key, value := range session.Properties {
			content.WriteString(fmt.Sprintf("**%s**: %v  \n", key, value))
		}
	}

	content.WriteString("\n")

	if session.Notes != "" {
		content.WriteString("## Notes\n")
		content.WriteString(session.Notes + "\n")
	}

	// Write file
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// WritePlan writes a plan to a markdown file
func (p *Parser) WritePlan(plan *ExercisePlan, filePath string) error {
	// Prepare frontmatter
	frontmatter, err := yaml.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	// Build markdown content
	var content strings.Builder
	content.WriteString("---\n")
	content.Write(frontmatter)
	content.WriteString("---\n\n")
	content.WriteString(fmt.Sprintf("# %s\n\n", plan.Name))

	if plan.Description != "" {
		content.WriteString(plan.Description + "\n\n")
	}

	content.WriteString("## Exercises\n")
	for i, item := range plan.Items {
		line := fmt.Sprintf("%d. %s", i+1, item.Exercise)
		if item.Target != "" {
			line += fmt.Sprintf(" - %s", item.Target)
		}
		content.WriteString(line + "\n")
	}
	content.WriteString("\n")

	// Write file
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	return nil
}

// WritePlanSession writes a plan session to a markdown file
func (p *Parser) WritePlanSession(planSession *PlanSession, filePath string) error {
	// Prepare frontmatter
	frontmatter, err := yaml.Marshal(planSession)
	if err != nil {
		return fmt.Errorf("failed to marshal plan session: %w", err)
	}

	// Build markdown content
	var content strings.Builder
	content.WriteString("---\n")
	content.Write(frontmatter)
	content.WriteString("---\n\n")
	content.WriteString(fmt.Sprintf("# Workout: %s - %s\n\n", planSession.PlanID, planSession.Date.Format("2006-01-02")))

	content.WriteString("## Exercises\n")
	for _, item := range planSession.Items {
		status := "✗"
		if item.Completed {
			status = "✓"
		}
		line := fmt.Sprintf("- [%s] %s", status, item.Exercise)
		if item.Actual != "" {
			line += fmt.Sprintf(" - %s", item.Actual)
		}
		content.WriteString(line + "\n")
		if item.Notes != "" {
			content.WriteString(fmt.Sprintf("  - Notes: %s\n", item.Notes))
		}
	}
	content.WriteString("\n")

	if planSession.Notes != "" {
		content.WriteString("## Overall Notes\n")
		content.WriteString(planSession.Notes + "\n")
	}

	// Write file
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write plan session file: %w", err)
	}

	return nil
}

// extractFrontmatter extracts YAML frontmatter from markdown content
func (p *Parser) extractFrontmatter(content []byte) (frontmatter string, body string, err error) {
	text := string(content)

	// Check if starts with ---
	if !strings.HasPrefix(text, "---\n") {
		return "", text, fmt.Errorf("no frontmatter found")
	}

	// Find closing ---
	rest := text[4:]
	endIdx := strings.Index(rest, "\n---\n")
	if endIdx == -1 {
		return "", text, fmt.Errorf("frontmatter not closed")
	}

	frontmatter = rest[:endIdx]
	body = strings.TrimSpace(rest[endIdx+5:])

	return frontmatter, body, nil
}

// getBrainPath extracts the brain root path from a file path
func (p *Parser) getBrainPath(filePath string) string {
	// Walk up until we find a brain marker (.obsidian, .logseq, .flip-brain.yaml, etc.)
	dir := filepath.Dir(filePath)
	for {
		// Check for brain markers
		markers := []string{".obsidian", ".logseq", ".flip-brain.yaml", "dendron.yml", ".foam"}
		for _, marker := range markers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir
			}
		}

		// Move up one level
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	// Fallback: return directory of file
	return filepath.Dir(filePath)
}

// GetExerciseFilePath generates the file path for an exercise based on brain type
func (p *Parser) GetExerciseFilePath(brainPath, exerciseID string, brainType string) string {
	switch brainType {
	case "logseq":
		// Logseq uses pages/
		return filepath.Join(brainPath, "pages", fmt.Sprintf("Exercise - %s.md", exerciseID))
	case "dendron":
		// Dendron uses hierarchical naming
		return filepath.Join(brainPath, fmt.Sprintf("exercise.%s.md", exerciseID))
	default:
		// Obsidian, Foam, Flip: exercises/ folder
		return filepath.Join(brainPath, "exercises", fmt.Sprintf("%s.md", exerciseID))
	}
}

// GetSessionFilePath generates the file path for a session
func (p *Parser) GetSessionFilePath(brainPath, exerciseID string, date time.Time, brainType string) string {
	dateStr := date.Format("2006-01-02")

	switch brainType {
	case "logseq":
		// Logseq: journals/YYYY_MM_DD.md (append to daily note)
		return filepath.Join(brainPath, "journals", fmt.Sprintf("%s.md", strings.ReplaceAll(dateStr, "-", "_")))
	default:
		// Obsidian, Foam, Flip: exercises/sessions/exercise-id/YYYY-MM-DD.md
		return filepath.Join(brainPath, "exercises", "sessions", exerciseID, fmt.Sprintf("%s.md", dateStr))
	}
}

// GetPlanFilePath generates the file path for a plan
func (p *Parser) GetPlanFilePath(brainPath, planID string, brainType string) string {
	switch brainType {
	case "logseq":
		return filepath.Join(brainPath, "pages", fmt.Sprintf("Exercise Plan - %s.md", planID))
	case "dendron":
		return filepath.Join(brainPath, fmt.Sprintf("exercise-plan.%s.md", planID))
	default:
		return filepath.Join(brainPath, "exercise-plans", fmt.Sprintf("%s.md", planID))
	}
}

// GetPlanSessionFilePath generates the file path for a plan session
func (p *Parser) GetPlanSessionFilePath(brainPath, planID string, date time.Time, brainType string) string {
	dateStr := date.Format("2006-01-02")

	switch brainType {
	case "logseq":
		// Append to daily journal
		return filepath.Join(brainPath, "journals", fmt.Sprintf("%s.md", strings.ReplaceAll(dateStr, "-", "_")))
	default:
		return filepath.Join(brainPath, "exercise-plans", "sessions", planID, fmt.Sprintf("%s.md", dateStr))
	}
}

// ParseJournalExerciseBlocks extracts exercise sessions from a journal file
func (p *Parser) ParseJournalExerciseBlocks(filePath string) ([]*ExerciseSession, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read journal file: %w", err)
	}

	// Extract date from filename (YYYY-MM-DD or YYYY_MM_DD)
	filename := filepath.Base(filePath)
	dateStr := strings.TrimSuffix(filename, ".md")
	dateStr = strings.ReplaceAll(dateStr, "_", "-")
	
	journalDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		// If we can't parse date from filename, use current date
		journalDate = time.Now()
	}

	var sessions []*ExerciseSession
	lines := strings.Split(string(content), "\n")

	var currentSession *ExerciseSession
	var blockStart int

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if compactSession := p.parseCompactExerciseLine(trimmed, journalDate, filePath, i); compactSession != nil {
			sessions = append(sessions, compactSession)
			continue
		}

		if legacySession, consumed := p.parseLegacyVSCodeExerciseEntry(lines, i, journalDate, filePath); legacySession != nil {
			sessions = append(sessions, legacySession)
			i += consumed
			continue
		}

		// Legacy CLI block header: "## Exercise: Name"
		if strings.HasPrefix(trimmed, "## Exercise:") {
			if currentSession != nil {
				currentSession.BlockEnd = i - 1
				sessions = append(sessions, currentSession)
			}

			currentSession = &ExerciseSession{
				Date:        journalDate,
				Properties:  make(map[string]interface{}),
				BlockStart:  i,
				JournalDate: journalDate.Format("2006-01-02"),
				FilePath:    filePath,
			}
			blockStart = i
			continue
		}

		if currentSession != nil && strings.HasPrefix(trimmed, "- ") {
			propLine := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if strings.Contains(propLine, "::") {
				parts := strings.SplitN(propLine, "::", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					valueStr := strings.TrimSpace(parts[1])
					p.applySessionField(currentSession, key, valueStr)
				}
			}
		}

		if currentSession != nil && (trimmed == "" || (strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "## Exercise:"))) {
			if i > blockStart+1 {
				currentSession.BlockEnd = i - 1
				sessions = append(sessions, currentSession)
				currentSession = nil
			}
		}
	}

	if currentSession != nil {
		currentSession.BlockEnd = len(lines) - 1
		sessions = append(sessions, currentSession)
	}

	return sessions, nil
}

func (p *Parser) parseCompactExerciseLine(line string, journalDate time.Time, filePath string, lineNumber int) *ExerciseSession {
	// New format: - [[Exercise Name]] (30 min) | key=val | ...
	if strings.HasPrefix(line, "- [[") {
		return p.parseWikilinkExerciseLine(line, journalDate, filePath, lineNumber)
	}

	// flip default format: - [Exercise Name](../exercises/id.md) (30 min) | key=val | ...
	if strings.HasPrefix(line, "- [") {
		return p.parseMarkdownLinkExerciseLine(line, journalDate, filePath, lineNumber)
	}

	if !strings.HasPrefix(line, "- exercise::") {
		return nil
	}

	raw := strings.TrimSpace(strings.TrimPrefix(line, "- exercise::"))
	if raw == "" {
		return nil
	}

	session := &ExerciseSession{
		Date:        journalDate,
		Properties:  make(map[string]interface{}),
		BlockStart:  lineNumber,
		BlockEnd:    lineNumber,
		JournalDate: journalDate.Format("2006-01-02"),
		FilePath:    filePath,
	}

	parts := strings.Split(raw, "|")
	for idx, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}

		if strings.Contains(token, "=") {
			kv := strings.SplitN(token, "=", 2)
			key := strings.TrimSpace(kv[0])
			val := ""
			if len(kv) > 1 {
				val = strings.TrimSpace(kv[1])
			}
			p.applySessionField(session, key, val)
			continue
		}

		if idx == 0 {
			session.ExerciseID = token
		}
	}

	if session.ExerciseID == "" {
		return nil
	}

	return session
}

// parseWikilinkExerciseLine parses the new compact format:
// - [[Exercise Name]] (30 min) | variant=X | distance_km=6.2 | notes=...
func (p *Parser) parseWikilinkExerciseLine(line string, journalDate time.Time, filePath string, lineNumber int) *ExerciseSession {
	// Strip leading "- "
	raw := strings.TrimPrefix(line, "- ")

	// Extract [[Name]]
	if !strings.HasPrefix(raw, "[[") {
		return nil
	}
	closeIdx := strings.Index(raw, "]]")
	if closeIdx == -1 {
		return nil
	}
	exerciseName := strings.TrimSpace(raw[2:closeIdx])
	if exerciseName == "" {
		return nil
	}

	session := &ExerciseSession{
		Date:        journalDate,
		Properties:  make(map[string]interface{}),
		BlockStart:  lineNumber,
		BlockEnd:    lineNumber,
		JournalDate: journalDate.Format("2006-01-02"),
		FilePath:    filePath,
		ExerciseID:  p.normalizeExerciseID(exerciseName),
	}

	// Rest after "]]"
	rest := strings.TrimSpace(raw[closeIdx+2:])

	// Optional duration: (30 min)
	if strings.HasPrefix(rest, "(") {
		closeP := strings.Index(rest, ")")
		if closeP != -1 {
			durStr := strings.TrimSpace(rest[1:closeP])
			// "30 min" → extract the number
			durStr = strings.TrimSuffix(strings.TrimSpace(durStr), " min")
			durStr = strings.TrimSuffix(durStr, "min")
			p.applySessionField(session, "duration", strings.TrimSpace(durStr))
			rest = strings.TrimSpace(rest[closeP+1:])
		}
	}

	// Remaining key=value pairs separated by |
	if strings.HasPrefix(rest, "|") {
		rest = strings.TrimSpace(rest[1:])
	}
	for _, part := range strings.Split(rest, "|") {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if strings.Contains(token, "=") {
			kv := strings.SplitN(token, "=", 2)
			key := strings.TrimSpace(kv[0])
			val := ""
			if len(kv) > 1 {
				val = strings.TrimSpace(kv[1])
			}
			p.applySessionField(session, key, val)
		}
	}

	return session
}

// parseMarkdownLinkExerciseLine parses compact format with markdown links:
// - [Exercise Name](../exercises/exercise-id.md) (30 min) | variant=X | key=val
func (p *Parser) parseMarkdownLinkExerciseLine(line string, journalDate time.Time, filePath string, lineNumber int) *ExerciseSession {
	raw := strings.TrimSpace(strings.TrimPrefix(line, "- "))

	openBracket := strings.Index(raw, "[")
	labelEnd := strings.Index(raw, "](")
	linkEnd := strings.Index(raw, ")")
	if openBracket != 0 || labelEnd <= 1 || linkEnd <= labelEnd+2 {
		return nil
	}

	linkLabel := strings.TrimSpace(raw[1:labelEnd])
	linkTarget := strings.TrimSpace(raw[labelEnd+2 : linkEnd])
	if linkLabel == "" || linkTarget == "" {
		return nil
	}

	lowerTarget := strings.ToLower(filepath.ToSlash(linkTarget))
	if !strings.Contains(lowerTarget, "/exercises/") && !strings.HasPrefix(lowerTarget, "exercises/") {
		return nil
	}

	session := &ExerciseSession{
		Date:        journalDate,
		Properties:  make(map[string]interface{}),
		BlockStart:  lineNumber,
		BlockEnd:    lineNumber,
		JournalDate: journalDate.Format("2006-01-02"),
		FilePath:    filePath,
	}

	fileName := strings.TrimSpace(filepath.Base(linkTarget))
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if fileName != "" && fileName != "." && fileName != ".." {
		session.ExerciseID = fileName
	} else {
		session.ExerciseID = p.normalizeExerciseID(linkLabel)
	}

	rest := strings.TrimSpace(raw[linkEnd+1:])

	// Optional duration: (30 min)
	if strings.HasPrefix(rest, "(") {
		closeP := strings.Index(rest, ")")
		if closeP != -1 {
			durStr := strings.TrimSpace(rest[1:closeP])
			durStr = strings.TrimSuffix(strings.TrimSpace(durStr), " min")
			durStr = strings.TrimSuffix(durStr, "min")
			p.applySessionField(session, "duration", strings.TrimSpace(durStr))
			rest = strings.TrimSpace(rest[closeP+1:])
		}
	}

	if strings.HasPrefix(rest, "|") {
		rest = strings.TrimSpace(rest[1:])
	}
	for _, part := range strings.Split(rest, "|") {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}
		if strings.Contains(token, "=") {
			kv := strings.SplitN(token, "=", 2)
			key := strings.TrimSpace(kv[0])
			val := ""
			if len(kv) > 1 {
				val = strings.TrimSpace(kv[1])
			}
			p.applySessionField(session, key, val)
		}
	}

	if session.ExerciseID == "" {
		return nil
	}

	return session
}

func (p *Parser) parseLegacyVSCodeExerciseEntry(lines []string, start int, journalDate time.Time, filePath string) (*ExerciseSession, int) {
	line := strings.TrimSpace(lines[start])
	if !strings.HasPrefix(line, "### 🏋️ [[") {
		return nil, 0
	}

	begin := strings.Index(line, "[[")
	end := strings.Index(line, "]]")
	if begin == -1 || end == -1 || end <= begin+2 {
		return nil, 0
	}

	name := strings.TrimSpace(line[begin+2 : end])
	session := &ExerciseSession{
		Date:        journalDate,
		Properties:  make(map[string]interface{}),
		BlockStart:  start,
		JournalDate: journalDate.Format("2006-01-02"),
		FilePath:    filePath,
		ExerciseID:  p.normalizeExerciseID(name),
	}

	consumed := 0
	for i := start + 1; i < len(lines); i++ {
		current := strings.TrimSpace(lines[i])
		if strings.HasPrefix(current, "### ") || strings.HasPrefix(current, "## ") {
			break
		}
		if current == "" {
			consumed = i - start
			break
		}

		if strings.HasPrefix(current, "**Variant:**") {
			session.VariantName = strings.TrimSpace(strings.TrimPrefix(current, "**Variant:**"))
		} else if strings.HasPrefix(current, "**Duration:**") {
			value := strings.TrimSpace(strings.TrimPrefix(current, "**Duration:**"))
			p.applySessionField(session, "duration", value)
		} else {
			if session.Notes == "" {
				session.Notes = current
			} else {
				session.Notes += " " + current
			}
		}

		consumed = i - start
	}

	session.BlockEnd = start + consumed
	return session, consumed
}

func (p *Parser) applySessionField(session *ExerciseSession, key, valueStr string) {
	clean := strings.TrimSpace(valueStr)
	if clean == "" {
		return
	}

	switch strings.ToLower(strings.TrimSpace(key)) {
	case "exercise", "exercise-id", "exercise_id", "id":
		session.ExerciseID = clean
	case "variant", "variant_name":
		session.VariantName = clean
	case "duration", "duration_min":
		norm := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(clean, " min"), "min"))
		if duration, err := strconv.Atoi(norm); err == nil {
			session.Duration = duration
		}
	case "notes", "note":
		session.Notes = clean
	case "name":
		// name is optional metadata for compact entries; keep it out of Properties.
	default:
		session.Properties[key] = p.parseSessionValue(clean)
	}
}

func (p *Parser) parseSessionValue(value string) interface{} {
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}
	return value
}

func (p *Parser) normalizeExerciseID(name string) string {
	id := strings.ToLower(strings.TrimSpace(name))
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, id)
	return id
}
