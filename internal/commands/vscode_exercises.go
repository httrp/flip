package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sort"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ExerciseItemResult represents an exercise for JSON output
type ExerciseItemResult struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Context      string    `json:"context,omitempty"`
	Description  string    `json:"description,omitempty"`
	Goal         string    `json:"goal,omitempty"`
	Status       string    `json:"status,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	VariantCount int       `json:"variant_count"`
	SessionCount int       `json:"session_count"`
	LastSession  string    `json:"last_session,omitempty"`
	FilePath     string    `json:"file_path"`
	RelPath      string    `json:"rel_path"`
	Created      time.Time `json:"created"`
}

// ExerciseVariantResult represents an exercise variant for JSON output
type ExerciseVariantResult struct {
	Name               string            `json:"name,omitempty"`
	Description        string            `json:"description,omitempty"`
	TrackingProperties map[string]string `json:"tracking_properties,omitempty"`
}

// ExerciseDetailResult is the response for vscode exercises show
type ExerciseDetailResult struct {
	ExerciseItemResult
	Variants  []ExerciseVariantResult `json:"variants,omitempty"`
	Duration  string                  `json:"duration,omitempty"`
	Related   []string                `json:"related,omitempty"`
	BrainName string                  `json:"brain_name"`
	BrainPath string                  `json:"brain_path"`
}

// ExercisesListResult is the response from vscode exercises list
type ExercisesListResult struct {
	Exercises []ExerciseItemResult `json:"exercises"`
	BrainName string               `json:"brain_name"`
	BrainPath string               `json:"brain_path"`
	Total     int                  `json:"total"`
}

// NewVSCodeExercisesCommand creates the vscode exercises subcommand
func NewVSCodeExercisesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exercises",
		Short: "Exercise-related VS Code integration commands",
	}

	cmd.AddCommand(NewVSCodeExercisesListCommand())
	cmd.AddCommand(NewVSCodeExercisesShowCommand())
	cmd.AddCommand(NewVSCodeExercisesTrackCommand())
	cmd.AddCommand(NewVSCodeExercisesNewCommand())
	cmd.AddCommand(NewVSCodeExercisesMigrateSessionsCommand())

	return cmd
}

// NewVSCodeExercisesListCommand lists all exercises for VS Code
func NewVSCodeExercisesListCommand() *cobra.Command {
	var jsonOutput bool
	var brainName string
	var filterContext string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all exercises",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput

			activeBrain, err := getBrainByNameOrActive(brainName)
			if err != nil {
				OutputJSONError("exercises-list", err)
				return nil
			}

			detection, err := brain.NewDetector().DetectBrainType(activeBrain.Path)
			if err != nil {
				OutputJSONError("exercises-list", err)
				return nil
			}

			scanner := exercises.NewScanner()
			allExercises, err := scanner.ScanExercises(activeBrain.Path, string(detection.Type))
			if err != nil {
				OutputJSONError("exercises-list", err)
				return nil
			}

			// Filter and convert
			var items []ExerciseItemResult
			for _, ex := range allExercises {
				// Filter by context if specified
				if filterContext != "" && !strings.EqualFold(ex.Context, filterContext) {
					continue
				}

				relPath := strings.TrimPrefix(ex.FilePath, activeBrain.Path+"/")
				lastSession := ""
				if ex.LastSession != nil {
					lastSession = ex.LastSession.Format("2006-01-02")
				}

				items = append(items, ExerciseItemResult{
					ID:           ex.ID,
					Name:         ex.Name,
					Context:      ex.Context,
					Description:  ex.Description,
					Goal:         ex.Goal,
					Status:       ex.Status,
					Tags:         ex.Tags,
					VariantCount: len(ex.Variants),
					SessionCount: ex.SessionCount,
					LastSession:  lastSession,
					FilePath:     ex.FilePath,
					RelPath:      relPath,
					Created:      ex.Created,
				})
			}

			// Sort by name
			sort.Slice(items, func(i, j int) bool {
				return items[i].Name < items[j].Name
			})

			result := ExercisesListResult{
				Exercises: items,
				BrainName: activeBrain.Name,
				BrainPath: activeBrain.Path,
				Total:     len(items),
			}

			OutputJSONSuccess("exercises-list", result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain to use")
	cmd.Flags().StringVarP(&filterContext, "context", "c", "", "Filter by context")

	return cmd
}

// NewVSCodeExercisesShowCommand shows exercise details
func NewVSCodeExercisesShowCommand() *cobra.Command {
	var jsonOutput bool
	var brainName string

	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show exercise details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput
			exerciseID := args[0]

			activeBrain, err := getBrainByNameOrActive(brainName)
			if err != nil {
				OutputJSONError("exercises-show", err)
				return nil
			}

			detection, err := brain.NewDetector().DetectBrainType(activeBrain.Path)
			if err != nil {
				OutputJSONError("exercises-show", err)
				return nil
			}

			scanner := exercises.NewScanner()
			allExercises, err := scanner.ScanExercises(activeBrain.Path, string(detection.Type))
			if err != nil {
				OutputJSONError("exercises-show", err)
				return nil
			}

			// Find the exercise
			var found *exercises.Exercise
			for _, ex := range allExercises {
				if ex.ID == exerciseID || strings.EqualFold(ex.Name, exerciseID) {
					found = ex
					break
				}
			}

			if found == nil {
				OutputJSONError("exercises-show", fmt.Errorf("exercise not found: %s", exerciseID))
				return nil
			}

			relPath := strings.TrimPrefix(found.FilePath, activeBrain.Path+"/")
			lastSession := ""
			if found.LastSession != nil {
				lastSession = found.LastSession.Format("2006-01-02")
			}

			// Convert variants
			variants := make([]ExerciseVariantResult, len(found.Variants))
			for i, v := range found.Variants {
				variants[i] = ExerciseVariantResult{
					Name:               v.Name,
					Description:        v.Description,
					TrackingProperties: v.TrackingProperties,
				}
			}

			result := ExerciseDetailResult{
				ExerciseItemResult: ExerciseItemResult{
					ID:           found.ID,
					Name:         found.Name,
					Context:      found.Context,
					Description:  found.Description,
					Goal:         found.Goal,
					Status:       found.Status,
					Tags:         found.Tags,
					VariantCount: len(found.Variants),
					SessionCount: found.SessionCount,
					LastSession:  lastSession,
					FilePath:     found.FilePath,
					RelPath:      relPath,
					Created:      found.Created,
				},
				Variants:  variants,
				Duration:  found.Duration,
				Related:   found.Related,
				BrainName: activeBrain.Name,
				BrainPath: activeBrain.Path,
			}

			OutputJSONSuccess("exercises-show", result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain to use")

	return cmd
}

// ExerciseTrackResult is the response from tracking
type ExerciseTrackResult struct {
	Success      bool   `json:"success"`
	ExerciseID   string `json:"exercise_id"`
	ExerciseName string `json:"exercise_name"`
	JournalPath  string `json:"journal_path"`
	Date         string `json:"date"`
}

type ExerciseMigrateResult struct {
	Success bool `json:"success"`
	exerciseMigrationResult
}

// NewVSCodeExercisesTrackCommand tracks a session
func NewVSCodeExercisesTrackCommand() *cobra.Command {
	var jsonOutput bool
	var brainName string
	var exerciseID string
	var duration int
	var variantName string
	var notes string
	var properties []string

	cmd := &cobra.Command{
		Use:   "track",
		Short: "Track an exercise session",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput

			if exerciseID == "" {
				OutputJSONError("exercises-track", fmt.Errorf("exercise ID required"))
				return nil
			}

			activeBrain, err := getBrainByNameOrActive(brainName)
			if err != nil {
				OutputJSONError("exercises-track", err)
				return nil
			}

			detection, err := brain.NewDetector().DetectBrainType(activeBrain.Path)
			if err != nil {
				OutputJSONError("exercises-track", err)
				return nil
			}

			scanner := exercises.NewScanner()
			allExercises, err := scanner.ScanExercises(activeBrain.Path, string(detection.Type))
			if err != nil {
				OutputJSONError("exercises-track", err)
				return nil
			}

			// Find the exercise
			var found *exercises.Exercise
			for _, ex := range allExercises {
				if ex.ID == exerciseID || strings.EqualFold(ex.Name, exerciseID) {
					found = ex
					break
				}
			}

			if found == nil {
				OutputJSONError("exercises-track", fmt.Errorf("exercise not found: %s", exerciseID))
				return nil
			}

			// Get journal path for today
			today := time.Now()
			journalPath := getJournalPathForDate(activeBrain.Path, detection.Type, today)

			// Ensure journal exists
			if err := ensureJournalExistsForDate(journalPath, today); err != nil {
				OutputJSONError("exercises-track", err)
				return nil
			}

			parsedProperties := parseExerciseProperties(properties)
			// Build the session entry
			sessionEntry := buildExerciseSessionEntry(found, journalPath, detection.Type, duration, variantName, notes, parsedProperties)

			// Append to journal in Exercises section
			if err := appendToJournalSection(journalPath, "## Exercises", sessionEntry); err != nil {
				OutputJSONError("exercises-track", err)
				return nil
			}

			result := ExerciseTrackResult{
				Success:      true,
				ExerciseID:   found.ID,
				ExerciseName: found.Name,
				JournalPath:  journalPath,
				Date:         today.Format("2006-01-02"),
			}

			OutputJSONSuccess("exercises-track", result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain to use")
	cmd.Flags().StringVar(&exerciseID, "exercise", "", "Exercise ID or name")
	cmd.Flags().IntVar(&duration, "duration", 0, "Duration in minutes")
	cmd.Flags().StringVar(&variantName, "variant", "", "Variant name")
	cmd.Flags().StringVar(&notes, "notes", "", "Session notes")
	cmd.Flags().StringArrayVar(&properties, "prop", []string{}, "Additional property in key=value format (repeatable)")

	return cmd
}

// ExerciseNewResult is the response from creating an exercise
type ExerciseNewResult struct {
	Success  bool   `json:"success"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	FilePath string `json:"file_path"`
	RelPath  string `json:"rel_path"`
}

// NewVSCodeExercisesNewCommand creates a new exercise
func NewVSCodeExercisesNewCommand() *cobra.Command {
	var jsonOutput bool
	var brainName string
	var name string
	var context string
	var description string
	var goal string
	var variantsJSON string

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new exercise",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput

			if name == "" {
				OutputJSONError("exercises-new", fmt.Errorf("exercise name required"))
				return nil
			}

			activeBrain, err := getBrainByNameOrActive(brainName)
			if err != nil {
				OutputJSONError("exercises-new", err)
				return nil
			}

			detection, err := brain.NewDetector().DetectBrainType(activeBrain.Path)
			if err != nil {
				OutputJSONError("exercises-new", err)
				return nil
			}

			// Generate ID from name
			id := generateExerciseIDFromName(name)

			variants := []exercises.ExerciseVariant{}
			if strings.TrimSpace(variantsJSON) != "" {
				if err := json.Unmarshal([]byte(variantsJSON), &variants); err != nil {
					OutputJSONError("exercises-new", fmt.Errorf("invalid variants json: %w", err))
					return nil
				}
			}

			// Create the exercise file
			filePath, err := createExerciseFile(activeBrain.Path, string(detection.Type), id, name, context, description, goal, variants)
			if err != nil {
				OutputJSONError("exercises-new", err)
				return nil
			}

			relPath := strings.TrimPrefix(filePath, activeBrain.Path+"/")

			result := ExerciseNewResult{
				Success:  true,
				ID:       id,
				Name:     name,
				FilePath: filePath,
				RelPath:  relPath,
			}

			OutputJSONSuccess("exercises-new", result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain to use")
	cmd.Flags().StringVar(&name, "name", "", "Exercise name")
	cmd.Flags().StringVar(&context, "context", "", "Context (e.g., Sport, Music)")
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().StringVar(&goal, "goal", "", "Goal")
	cmd.Flags().StringVar(&variantsJSON, "variants-json", "", "Variants as JSON array")

	return cmd
}

// NewVSCodeExercisesMigrateSessionsCommand migrates legacy exercise journal entries.
func NewVSCodeExercisesMigrateSessionsCommand() *cobra.Command {
	var jsonOutput bool
	var brainName string
	var apply bool
	var backup bool
	var days int

	cmd := &cobra.Command{
		Use:   "migrate-sessions",
		Short: "Migrate legacy exercise journal entries to compact format",
		RunE: func(cmd *cobra.Command, args []string) error {
			JSONOutput = jsonOutput

			activeBrain, err := getBrainByNameOrActive(brainName)
			if err != nil {
				OutputJSONError("exercises-migrate-sessions", err)
				return nil
			}

			detection, err := brain.NewDetector().DetectBrainType(activeBrain.Path)
			if err != nil {
				OutputJSONError("exercises-migrate-sessions", err)
				return nil
			}

			result, err := migrateExerciseSessionsInBrain(activeBrain.Path, detection.Type, apply, backup, days)
			if err != nil {
				OutputJSONError("exercises-migrate-sessions", err)
				return nil
			}
			result.BrainName = activeBrain.Name
			result.BrainPath = activeBrain.Path

			OutputJSONSuccess("exercises-migrate-sessions", ExerciseMigrateResult{
				Success:                 true,
				exerciseMigrationResult: result,
			})
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().StringVar(&brainName, "brain", "", "Brain to use")
	cmd.Flags().BoolVar(&apply, "apply", false, "Apply migration (default dry-run)")
	cmd.Flags().BoolVar(&backup, "backup", true, "Create backups when applying")
	cmd.Flags().IntVar(&days, "days", 0, "Only process journal files from the last N days (0 = all)")

	return cmd
}

// Helper functions

func buildExerciseSessionEntry(ex *exercises.Exercise, journalPath string, brainType brain.BrainType, duration int, variantName, notes string, properties map[string]interface{}) string {
	return buildCompactExerciseEntry(ex.ID, ex.Name, ex.FilePath, journalPath, brainType, variantName, duration, properties, notes)
}

func parseExerciseProperties(propertyValues []string) map[string]interface{} {
	parsed := make(map[string]interface{})
	for _, item := range propertyValues {
		kv := strings.SplitN(item, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		if key == "" || value == "" {
			continue
		}

		if intVal, err := strconv.Atoi(value); err == nil {
			parsed[key] = intVal
		} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			parsed[key] = floatVal
		} else {
			parsed[key] = value
		}
	}
	return parsed
}

func appendToJournalSection(journalPath, sectionHeader, content string) error {
	// Read journal
	data, err := os.ReadFile(journalPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var result []string
	inserted := false
	sectionFound := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		result = append(result, line)

		// Found the section - insert content after section header
		if strings.HasPrefix(line, sectionHeader) && !inserted {
			sectionFound = true
			// Add empty line after section header if next line isn't empty
			if i+1 < len(lines) && lines[i+1] != "" {
				result = append(result, "")
			}
			// Add the content
			result = append(result, content)
			inserted = true
		}
	}

	// If section not found, append at end
	if !sectionFound {
		result = append(result, "")
		result = append(result, sectionHeader)
		result = append(result, "")
		result = append(result, content)
	}

	return os.WriteFile(journalPath, []byte(strings.Join(result, "\n")), 0644)
}

func generateExerciseIDFromName(name string) string {
	// Convert to lowercase, replace spaces with hyphens
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	// Remove special characters
	id = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, id)
	return id
}

func createExerciseFile(brainPath, brainType, id, name, context, description, goal string, variants []exercises.ExerciseVariant) (string, error) {
	// Determine exercises folder
	exercisesDir := filepath.Join(brainPath, "exercises")
	if brainType == "logseq" {
		exercisesDir = filepath.Join(brainPath, "pages")
	}

	// Create directory if needed
	if err := os.MkdirAll(exercisesDir, 0755); err != nil {
		return "", err
	}

	// Create filename
	filename := fmt.Sprintf("%s.md", id)
	filePath := filepath.Join(exercisesDir, filename)

	frontmatter := map[string]interface{}{
		"id":            id,
		"type":          "exercise",
		"name":          name,
		"status":        "active",
		"created":       time.Now().Format(time.RFC3339),
		"session_count": 0,
	}
	if context != "" {
		frontmatter["context"] = context
	}
	if description != "" {
		frontmatter["description"] = description
	}
	if goal != "" {
		frontmatter["goal"] = goal
	}
	if len(variants) > 0 {
		frontmatter["variants"] = variants
	}

	frontmatterYAML, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", err
	}

	// Build content
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(frontmatterYAML)
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", name))
	if description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", description))
	}
	sb.WriteString("## Variants\n\n")
	if len(variants) == 0 {
		sb.WriteString("_Add exercise variants here_\n\n")
	} else {
		for i, variant := range variants {
			if strings.TrimSpace(variant.Name) != "" {
				sb.WriteString(fmt.Sprintf("### Variant %d: %s\n\n", i+1, variant.Name))
			} else {
				sb.WriteString(fmt.Sprintf("### Variant %d\n\n", i+1))
			}

			if strings.TrimSpace(variant.Description) != "" {
				sb.WriteString(fmt.Sprintf("%s\n\n", variant.Description))
			}

			if len(variant.TrackingProperties) > 0 {
				sb.WriteString("**Tracking Properties:**\n")
				keys := make([]string, 0, len(variant.TrackingProperties))
				for key := range variant.TrackingProperties {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				for _, key := range keys {
					sb.WriteString(fmt.Sprintf("- **%s**: %s\n", key, variant.TrackingProperties[key]))
				}
				sb.WriteString("\n")
			}
		}
	}
	sb.WriteString("## Notes\n\n")
	sb.WriteString("_Add notes and observations here_\n")

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}
