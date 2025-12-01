package exercises

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Scanner finds exercises, sessions, and plans in a brain
type Scanner struct {
	parser *Parser
}

// NewScanner creates a new exercise scanner
func NewScanner() *Scanner {
	return &Scanner{
		parser: NewParser(),
	}
}

// ScanExercises finds all exercise definitions in a brain
func (s *Scanner) ScanExercises(brainPath string, brainType string) ([]*Exercise, error) {
	var exercises []*Exercise

	// Determine search paths based on brain type
	searchPaths := s.getExerciseSearchPaths(brainPath, brainType)

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue // Path doesn't exist, skip
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors, continue walking
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Only process .md files
			if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
				return nil
			}

			// Skip session files (they're in sessions/ subdirectory)
			if strings.Contains(path, "/sessions/") || strings.Contains(path, "\\sessions\\") {
				return nil
			}

			// Try to parse as exercise
			exercise, err := s.parser.ParseExercise(path)
			if err != nil {
				return nil // Skip invalid files
			}

			// Only include if frontmatter has type: exercise
			if s.isExerciseFile(path) {
				exercises = append(exercises, exercise)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to scan exercises: %w", err)
		}
	}

	return exercises, nil
}

// ScanSessions finds all sessions for a specific exercise
func (s *Scanner) ScanSessions(brainPath, exerciseID string, brainType string) ([]*ExerciseSession, error) {
	var sessions []*ExerciseSession

	// Determine search paths based on brain type
	searchPaths := s.getSessionSearchPaths(brainPath, exerciseID, brainType)

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
				return nil
			}

			// Try to parse as session
			session, err := s.parser.ParseSession(path)
			if err != nil {
				return nil
			}

			// Filter by exercise ID
			if session.ExerciseID == exerciseID {
				sessions = append(sessions, session)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to scan sessions: %w", err)
		}
	}

	return sessions, nil
}

// ScanAllSessions finds all sessions across all exercises
func (s *Scanner) ScanAllSessions(brainPath string, brainType string) ([]*ExerciseSession, error) {
	var sessions []*ExerciseSession

	searchPaths := s.getSessionSearchPaths(brainPath, "", brainType)

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
				return nil
			}

			session, err := s.parser.ParseSession(path)
			if err != nil {
				return nil
			}

			if s.isSessionFile(path) {
				sessions = append(sessions, session)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to scan all sessions: %w", err)
		}
	}

	return sessions, nil
}

// ScanPlans finds all exercise plans in a brain
func (s *Scanner) ScanPlans(brainPath string, brainType string) ([]*ExercisePlan, error) {
	var plans []*ExercisePlan

	searchPaths := s.getPlanSearchPaths(brainPath, brainType)

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
				return nil
			}

			// Skip session files
			if strings.Contains(path, "/sessions/") || strings.Contains(path, "\\sessions\\") {
				return nil
			}

			plan, err := s.parser.ParsePlan(path)
			if err != nil {
				return nil
			}

			if s.isPlanFile(path) {
				plans = append(plans, plan)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to scan plans: %w", err)
		}
	}

	return plans, nil
}

// ScanPlanSessions finds all sessions for a specific plan
func (s *Scanner) ScanPlanSessions(brainPath, planID string, brainType string) ([]*PlanSession, error) {
	var planSessions []*PlanSession

	searchPaths := s.getPlanSessionSearchPaths(brainPath, planID, brainType)

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
				return nil
			}

			planSession, err := s.parser.ParsePlanSession(path)
			if err != nil {
				return nil
			}

			if planSession.PlanID == planID {
				planSessions = append(planSessions, planSession)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to scan plan sessions: %w", err)
		}
	}

	return planSessions, nil
}

// getExerciseSearchPaths returns paths to search for exercise definitions
func (s *Scanner) getExerciseSearchPaths(brainPath, brainType string) []string {
	switch brainType {
	case "logseq":
		return []string{
			filepath.Join(brainPath, "pages"),
		}
	case "dendron":
		return []string{
			brainPath, // Dendron stores at root with hierarchical names
		}
	default:
		// Obsidian, Foam, Flip
		return []string{
			filepath.Join(brainPath, "exercises"),
		}
	}
}

// getSessionSearchPaths returns paths to search for sessions
func (s *Scanner) getSessionSearchPaths(brainPath, exerciseID string, brainType string) []string {
	switch brainType {
	case "logseq":
		return []string{
			filepath.Join(brainPath, "journals"),
		}
	default:
		if exerciseID != "" {
			return []string{
				filepath.Join(brainPath, "exercises", "sessions", exerciseID),
			}
		}
		return []string{
			filepath.Join(brainPath, "exercises", "sessions"),
		}
	}
}

// getPlanSearchPaths returns paths to search for plans
func (s *Scanner) getPlanSearchPaths(brainPath, brainType string) []string {
	switch brainType {
	case "logseq":
		return []string{
			filepath.Join(brainPath, "pages"),
		}
	case "dendron":
		return []string{
			brainPath,
		}
	default:
		return []string{
			filepath.Join(brainPath, "exercise-plans"),
		}
	}
}

// getPlanSessionSearchPaths returns paths to search for plan sessions
func (s *Scanner) getPlanSessionSearchPaths(brainPath, planID string, brainType string) []string {
	switch brainType {
	case "logseq":
		return []string{
			filepath.Join(brainPath, "journals"),
		}
	default:
		if planID != "" {
			return []string{
				filepath.Join(brainPath, "exercise-plans", "sessions", planID),
			}
		}
		return []string{
			filepath.Join(brainPath, "exercise-plans", "sessions"),
		}
	}
}

// isExerciseFile checks if a file contains an exercise definition
func (s *Scanner) isExerciseFile(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Quick check for "type: exercise" in frontmatter
	text := string(content)
	return strings.Contains(text, "type: exercise") || 
	       strings.Contains(text, "exercise_type:")
}

// isSessionFile checks if a file contains a session
func (s *Scanner) isSessionFile(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	text := string(content)
	return strings.Contains(text, "type: exercise-session") ||
	       strings.Contains(text, "exercise_id:")
}

// isPlanFile checks if a file contains a plan
func (s *Scanner) isPlanFile(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	text := string(content)
	return strings.Contains(text, "type: exercise-plan") ||
	       strings.Contains(text, "plan_id:")
}
