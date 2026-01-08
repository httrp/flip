package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
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

		selectPrompt := promptui.Select{
			Label: "Select Exercise",
			Items: exerciseNames,
		}
		_, selected, err := selectPrompt.Run()
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

		variantPrompt := promptui.Select{
			Label: "Select variant",
			Items: variantLabels,
		}
		variantIdx, _, err := variantPrompt.Run()
		if err != nil {
			return fmt.Errorf("error selecting variant: %w", err)
		}

		selectedVariant := exercise.Variants[variantIdx]
		variantName = selectedVariant.Name

		// Prompt for tracking properties from selected variant
		if len(selectedVariant.TrackingProperties) > 0 {
			fmt.Println("\nEnter values for tracking properties (leave empty to skip):")
			for name, unit := range selectedVariant.TrackingProperties {
				prompt := promptui.Prompt{
					Label: fmt.Sprintf("%s (%s)", name, unit),
				}
				valueStr, err := prompt.Run()
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
				prompt := promptui.Prompt{
					Label: fmt.Sprintf("%s (%s)", name, unit),
				}
				valueStr, err := prompt.Run()
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
	durationPrompt := promptui.Prompt{
		Label:   "Duration (minutes)",
		Default: "30",
	}
	durationStr, err := durationPrompt.Run()
	if err != nil {
		return fmt.Errorf("error reading duration: %w", err)
	}
	duration, _ := strconv.Atoi(durationStr)

	// Notes (optional)
	notesPrompt := promptui.Prompt{
		Label: "Notes (optional)",
	}
	notes, _ := notesPrompt.Run()

	// Append to journal
	journalDir := getJournalDirectory(brainPath, detection.Type)
	journalFilename := generateJournalFilename(now, detection.Type)
	journalPath := filepath.Join(journalDir, journalFilename)

	// Ensure journal directory exists
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return fmt.Errorf("error creating journal directory: %w", err)
	}

	// Build exercise block
	var block strings.Builder
	block.WriteString(fmt.Sprintf("\n## Exercise: %s\n", exercise.Name))
	block.WriteString(fmt.Sprintf("- exercise-id:: %s\n", exercise.ID))
	if variantName != "" {
		block.WriteString(fmt.Sprintf("- variant:: %s\n", variantName))
	}
	block.WriteString(fmt.Sprintf("- duration:: %d min\n", duration))

	for key, value := range properties {
		block.WriteString(fmt.Sprintf("- %s:: %v\n", key, value))
	}

	if notes != "" {
		block.WriteString(fmt.Sprintf("- notes:: %s\n", notes))
	}

	// Append or create journal file
	var journalContent string
	if _, err := os.Stat(journalPath); err == nil {
		// File exists, append
		data, err := os.ReadFile(journalPath)
		if err != nil {
			return fmt.Errorf("error reading journal: %w", err)
		}
		journalContent = string(data) + block.String()
	} else {
		// File doesn't exist, create with minimal header
		journalContent = generateMinimalJournalHeader(now, detection.Type) + block.String()
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
