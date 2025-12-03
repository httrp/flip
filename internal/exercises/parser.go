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
	
	for i, line := range lines {
		// Check for exercise block header: "## Exercise: Name"
		if strings.HasPrefix(line, "## Exercise:") {
			// Save previous session if exists
			if currentSession != nil {
				currentSession.BlockEnd = i - 1
				sessions = append(sessions, currentSession)
			}
			
			// Start new session
			_ = strings.TrimSpace(strings.TrimPrefix(line, "## Exercise:"))
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
		
		// Parse properties within exercise block
		if currentSession != nil && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			// Property line format: "- key:: value"
			propLine := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
			
			if strings.Contains(propLine, "::") {
				parts := strings.SplitN(propLine, "::", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					valueStr := strings.TrimSpace(parts[1])
					
					// Handle special keys
					switch key {
					case "exercise-id":
						currentSession.ExerciseID = valueStr
					case "variant":
						currentSession.VariantName = valueStr
					case "duration":
						// Parse "30 min" format
						durationStr := strings.TrimSuffix(valueStr, " min")
						durationStr = strings.TrimSuffix(durationStr, "min")
						if duration, err := strconv.Atoi(strings.TrimSpace(durationStr)); err == nil {
							currentSession.Duration = duration
						}
					case "notes":
						currentSession.Notes = valueStr
					default:
						// Store as property - try to parse as number
						if intVal, err := strconv.Atoi(valueStr); err == nil {
							currentSession.Properties[key] = intVal
						} else if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {
							currentSession.Properties[key] = floatVal
						} else {
							currentSession.Properties[key] = valueStr
						}
					}
				}
			}
		}
		
		// Check if we've left the exercise block (empty line or new header)
		if currentSession != nil && (line == "" || (strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "## Exercise:"))) {
			if i > blockStart+1 { // At least some content after header
				currentSession.BlockEnd = i - 1
				sessions = append(sessions, currentSession)
				currentSession = nil
			}
		}
	}
	
	// Save last session if exists
	if currentSession != nil {
		currentSession.BlockEnd = len(lines) - 1
		sessions = append(sessions, currentSession)
	}
	
	return sessions, nil
}
