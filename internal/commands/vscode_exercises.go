package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/spf13/cobra"
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

	return cmd
}

// getBrainByNameOrActive returns the brain by name or the active brain
func getBrainByNameOrActive(brainName string) (*Brain, error) {
	if brainName != "" {
		brains, err := getAllBrainsInWorkspace()
		if err != nil {
			return nil, err
		}
		path, ok := brains[brainName]
		if !ok {
			return nil, fmt.Errorf("brain '%s' not found", brainName)
		}
		return &Brain{Name: brainName, Path: path}, nil
	}
	return getActiveBrain()
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

// NewVSCodeExercisesTrackCommand tracks a session
func NewVSCodeExercisesTrackCommand() *cobra.Command {
	var jsonOutput bool
	var brainName string
	var exerciseID string
	var duration int
	var variantName string
	var notes string

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

			// Build the session entry
			sessionEntry := buildExerciseSessionEntry(found, duration, variantName, notes)

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

			// Create the exercise file
			filePath, err := createExerciseFile(activeBrain.Path, string(detection.Type), id, name, context, description, goal)
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

	return cmd
}

// Helper functions

func buildExerciseSessionEntry(ex *exercises.Exercise, duration int, variantName, notes string) string {
	var sb strings.Builder

	// Header with wikilink
	sb.WriteString(fmt.Sprintf("### 🏋️ [[%s]]\n", ex.Name))

	if variantName != "" {
		sb.WriteString(fmt.Sprintf("**Variant:** %s\n", variantName))
	}

	if duration > 0 {
		sb.WriteString(fmt.Sprintf("**Duration:** %d min\n", duration))
	}

	if notes != "" {
		sb.WriteString(fmt.Sprintf("\n%s\n", notes))
	}

	sb.WriteString("\n")

	return sb.String()
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

func createExerciseFile(brainPath, brainType, id, name, context, description, goal string) (string, error) {
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

	// Build content
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", id))
	sb.WriteString("type: exercise\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", name))
	if context != "" {
		sb.WriteString(fmt.Sprintf("context: %s\n", context))
	}
	if description != "" {
		sb.WriteString(fmt.Sprintf("description: %s\n", description))
	}
	if goal != "" {
		sb.WriteString(fmt.Sprintf("goal: %s\n", goal))
	}
	sb.WriteString("status: active\n")
	sb.WriteString(fmt.Sprintf("created: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("session_count: 0\n")
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", name))
	if description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", description))
	}
	sb.WriteString("## Variants\n\n")
	sb.WriteString("_Add exercise variants here_\n\n")
	sb.WriteString("## Notes\n\n")
	sb.WriteString("_Add notes and observations here_\n")

	if err := os.WriteFile(filePath, []byte(sb.String()), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}
