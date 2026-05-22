package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	ui "github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// ExerciseTrackCmd tracks a session for an exercise
var ExerciseTrackCmd = &cobra.Command{
	Use:   "track [exercise-id]",
	Short: lang.GetText("exercise.track.short"),
	Long:  lang.GetText("exercise.track.long"),
	RunE:  runExerciseTrack,
}

func runExerciseTrack(cmd *cobra.Command, args []string) error {
	activeBrain, err := getActiveBrain()
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	brainPath := activeBrain.Path
	detection, err := brain.NewDetector().DetectBrainType(brainPath)
	if err != nil {
		return fmt.Errorf("error detecting brain: %w", err)
	}
	if !detection.Compatible {
		return fmt.Errorf("not in a compatible brain directory")
	}

	scanner := exercises.NewScanner()

	// Get exercise ID
	var exerciseID string
	if len(args) > 0 {
		exerciseID = args[0]
	} else {
		// List exercises and let user choose
		allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
		if err != nil {
			return fmt.Errorf("error scanning exercises: %w", err)
		}

		if len(allExercises) == 0 {
			fmt.Println("\n⚠️  No exercises found")
			fmt.Println("Create one with: flip exercise new")
			return nil
		}

		exerciseNames := make([]string, len(allExercises))
		exerciseMap := make(map[string]*exercises.Exercise)
		for i, ex := range allExercises {
			contextStr := "no context"
			if ex.Context != "" {
				contextStr = ex.Context
			}
			exerciseNames[i] = fmt.Sprintf("%s (%s)", ex.Name, contextStr)
			exerciseMap[exerciseNames[i]] = ex
		}

		selectItems := make([]ui.SelectItem, len(exerciseNames))
		for i, name := range exerciseNames {
			selectItems[i] = ui.SelectItem{Label: name, Value: name}
		}

		_, selected, err := ui.RunSelect("Select Exercise", selectItems, 10)
		if err != nil {
			return fmt.Errorf("error selecting exercise: %w", err)
		}

		exerciseID = exerciseMap[selected].ID
	}

	// Load exercise
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		return fmt.Errorf("error loading exercises: %w", err)
	}

	var exercise *exercises.Exercise
	for _, ex := range allExercises {
		if ex.ID == exerciseID {
			exercise = ex
			break
		}
	}

	if exercise == nil {
		return fmt.Errorf("exercise not found: %s", exerciseID)
	}

	// Create session properties
	now := time.Now()
	properties := make(map[string]interface{})
	var variantName string

	// Select variant if multiple exist
	if len(exercise.Variants) > 1 {
		variantLabels := make([]string, len(exercise.Variants))
		for i, v := range exercise.Variants {
			if v.Name != "" {
				variantLabels[i] = fmt.Sprintf("Variant %d: %s", i+1, v.Name)
			} else {
				variantLabels[i] = fmt.Sprintf("Variant %d", i+1)
			}
		}

		variantItems := make([]ui.SelectItem, len(variantLabels))
		for i, label := range variantLabels {
			variantItems[i] = ui.SelectItem{Label: label, Value: fmt.Sprintf("%d", i)}
		}

		variantIdx, _, err := ui.RunSelect("Select variant", variantItems, 10)
		if err != nil {
			return fmt.Errorf("error selecting variant: %w", err)
		}

		selectedVariant := exercise.Variants[variantIdx]
		variantName = selectedVariant.Name

		// Prompt for tracking properties from selected variant
		if len(selectedVariant.TrackingProperties) > 0 {
			fmt.Println("\nEnter values for tracking properties (leave empty to skip):")
			for name, unit := range selectedVariant.TrackingProperties {
				valueStr, err := ui.RunInput(fmt.Sprintf("%s (%s)", name, unit), "", "", nil)
				if err != nil {
					continue
				}
				if valueStr != "" {
					// Try to parse as number, otherwise keep as string
					if intVal, err := strconv.Atoi(valueStr); err == nil {
						properties[name] = intVal
					} else if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {
						properties[name] = floatVal
					} else {
						properties[name] = valueStr
					}
				}
			}
		}
	} else if len(exercise.Variants) == 1 {
		// Only one variant, use it automatically
		variant := exercise.Variants[0]
		variantName = variant.Name

		if len(variant.TrackingProperties) > 0 {
			fmt.Println("\nEnter values for tracking properties (leave empty to skip):")
			for name, unit := range variant.TrackingProperties {
				valueStr, err := ui.RunInput(fmt.Sprintf("%s (%s)", name, unit), "", "", nil)
				if err != nil {
					continue
				}
				if valueStr != "" {
					if intVal, err := strconv.Atoi(valueStr); err == nil {
						properties[name] = intVal
					} else if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {
						properties[name] = floatVal
					} else {
						properties[name] = valueStr
					}
				}
			}
		}
	}

	// Duration
	durationStr, err := ui.RunInput("Duration (minutes)", "", "30", nil)
	if err != nil {
		return fmt.Errorf("error reading duration: %w", err)
	}
	duration, _ := strconv.Atoi(durationStr)

	// Notes (optional)
	notes, _ := ui.RunInput("Notes (optional)", "", "", nil)

	// Append to journal
	journalDir := getJournalDirectory(brainPath, detection.Type)
	journalFilename := generateJournalFilename(now, detection.Type)
	journalPath := filepath.Join(journalDir, journalFilename)

	// Ensure journal directory exists
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return fmt.Errorf("error creating journal directory: %w", err)
	}

	entry := buildCompactExerciseEntry(
		exercise.ID,
		exercise.Name,
		exercise.FilePath,
		journalPath,
		detection.Type,
		variantName,
		duration,
		properties,
		notes,
	)

	// Append or create journal file
	var journalContent string
	if _, err := os.Stat(journalPath); err == nil {
		// File exists, append
		data, err := os.ReadFile(journalPath)
		if err != nil {
			return fmt.Errorf("error reading journal: %w", err)
		}
		journalContent = string(data) + entry
	} else {
		// File doesn't exist, create with minimal header
		journalContent = generateMinimalJournalHeader(now, detection.Type) + entry
	}

	if err := os.WriteFile(journalPath, []byte(journalContent), 0644); err != nil {
		return fmt.Errorf("error writing journal: %w", err)
	}

	fmt.Printf("✓ Exercise tracked in journal: %s\n", journalPath)
	return nil
}

// generateMinimalJournalHeader creates a minimal journal header if file doesn't exist
func generateMinimalJournalHeader(date time.Time, brainType brain.BrainType) string {
	dateStr := date.Format("2006-01-02")
	weekday := date.Format("Monday")

	switch brainType {
	case brain.BrainTypeLogseq:
		return fmt.Sprintf("- %s, %s\n", weekday, dateStr)
	case brain.BrainTypeObsidian:
		return fmt.Sprintf("---\ndate: %s\n---\n\n# %s, %s\n", dateStr, weekday, dateStr)
	default:
		return fmt.Sprintf("# %s, %s\n", weekday, dateStr)
	}
}

func buildCompactExerciseEntry(exerciseID, exerciseName, exerciseFilePath, journalPath string, brainType brain.BrainType, variantName string, duration int, properties map[string]interface{}, notes string) string {
	header := buildExerciseLinkHeader(exerciseID, exerciseName, exerciseFilePath, journalPath, brainType)
	if duration > 0 {
		header += fmt.Sprintf(" (%d min)", duration)
	}

	parts := []string{header}

	if variantName != "" {
		parts = append(parts, fmt.Sprintf("variant=%s", sanitizeCompactValue(variantName)))
	}

	// Add tracking properties (skip duration_min – already in header)
	keys := make([]string, 0, len(properties))
	for key := range properties {
		if strings.ToLower(key) != "duration_min" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, sanitizeCompactValue(fmt.Sprintf("%v", properties[key]))))
	}

	if notes != "" {
		parts = append(parts, fmt.Sprintf("notes=%s", sanitizeCompactValue(notes)))
	}

	return "\n- " + strings.Join(parts, " | ") + "\n"
}

func buildExerciseLinkHeader(exerciseID, exerciseName, exerciseFilePath, journalPath string, brainType brain.BrainType) string {
	cleanName := sanitizeCompactValue(exerciseName)

	switch brainType {
	case brain.BrainTypeLogseq, brain.BrainTypeObsidian:
		return fmt.Sprintf("[[%s]]", cleanName)
	default:
		targetPath := exerciseFilePath
		if strings.TrimSpace(targetPath) == "" {
			targetPath = filepath.Join(filepath.Dir(journalPath), "..", "exercises", fmt.Sprintf("%s.md", exerciseID))
		}

		relPath, err := filepath.Rel(filepath.Dir(journalPath), targetPath)
		if err != nil || strings.TrimSpace(relPath) == "" {
			relPath = filepath.Join("..", "exercises", fmt.Sprintf("%s.md", exerciseID))
		}

		return fmt.Sprintf("[%s](%s)", cleanName, filepath.ToSlash(relPath))
	}
}

func sanitizeCompactValue(value string) string {
	clean := strings.TrimSpace(value)
	clean = strings.ReplaceAll(clean, "|", "/")
	clean = strings.ReplaceAll(clean, "\n", " ")
	clean = strings.Join(strings.Fields(clean), " ")
	return clean
}
