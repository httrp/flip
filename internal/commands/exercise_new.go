package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	ui "github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

// ExerciseNewCmd creates a new exercise definition
var ExerciseNewCmd = &cobra.Command{
	Use:   "new",
	Short: lang.GetText("exercise.new.short"),
	Long:  lang.GetText("exercise.new.long"),
	RunE:  runExerciseNew,
}

func runExerciseNew(cmd *cobra.Command, args []string) error {
	// Ask user which brain to use (from active workspace)
	selectedBrain, err := selectBrainForOperation(lang.GetText("prompts.select_brain_for_new_exercise"))
	if err != nil {
		return fmt.Errorf("selecting brain: %w", err)
	}

	brainPath := selectedBrain.Path
	detection, err := brain.NewDetector().DetectBrainType(brainPath)
	if err != nil {
		return fmt.Errorf("detecting brain: %w", err)
	}
	if !detection.Compatible {
		return fmt.Errorf("not in a compatible brain directory")
	}

	exercise := &exercises.Exercise{
		Type:     "exercise",
		Created:  time.Now(),
		Variants: []exercises.ExerciseVariant{},
	}

	// Context FIRST - scan existing contexts
	scanner := exercises.NewScanner()
	allExercises, err := scanner.ScanExercises(brainPath, string(detection.Type))
	if err != nil {
		fmt.Printf("Warning: Could not scan existing exercises: %v\n", err)
		allExercises = []*exercises.Exercise{}
	}

	// Collect unique contexts
	contextMap := make(map[string]bool)
	for _, ex := range allExercises {
		if ex.Context != "" {
			contextMap[ex.Context] = true
		}
	}

	var contexts []string
	for ctx := range contextMap {
		contexts = append(contexts, ctx)
	}
	contexts = append(contexts, "➕ New context", "⊝ No context")

	contextItems := make([]ui.SelectItem, len(contexts))
	for i, c := range contexts {
		contextItems[i] = ui.SelectItem{Label: c, Value: c}
	}

	_, selectedContext, err := ui.RunSelect("Select context", contextItems, 12)
	if err != nil {
		return fmt.Errorf("selecting context: %w", err)
	}

	if selectedContext == "➕ New context" {
		exercise.Context, _ = ui.RunInput("New context name (e.g., Sport, Music, Basketball, Drums)", "", "", nil)
	} else if selectedContext != "⊝ No context" {
		exercise.Context = selectedContext
	}

	// Name
	exercise.Name, err = ui.RunInput("Exercise Name", "", "", nil)
	if err != nil {
		return fmt.Errorf("entering name: %w", err)
	}

	// Check for duplicates or similar exercises
	exactMatch := false
	for _, ex := range allExercises {
		if strings.EqualFold(ex.Name, exercise.Name) {
			exactMatch = true
			break
		}
	}

	if exactMatch {
		fmt.Printf("\n⚠️  Exercise '%s' already exists!\n\n", exercise.Name)

		actionItems := []ui.SelectItem{
			{Label: "Open existing exercise", Value: "open"},
			{Label: "Create anyway with different name", Value: "rename"},
			{Label: "Cancel", Value: "cancel"},
		}

		_, actionVal, err := ui.RunSelect("What would you like to do?", actionItems, 5)
		if err != nil || actionVal == "cancel" {
			fmt.Println("Cancelled")
			return nil
		}

		if actionVal == "open" {
			// Find and open existing
			for _, ex := range allExercises {
				if strings.EqualFold(ex.Name, exercise.Name) {
					fmt.Printf("Opening: %s\n", ex.FilePath)
					if err := promptAndOpenEditor(ex.FilePath); err != nil {
						fmt.Printf("Error: %v\n", err)
					}
					return nil
				}
			}
		}

		// Ask for new name
		exercise.Name, err = ui.RunInput("New exercise name", "", "", nil)
		if err != nil {
			return nil
		}
	}

	// Check for similar exercises (threshold 0.6)
	similar := exercises.FindSimilarExercises(exercise.Name, allExercises, 0.6)
	if len(similar) > 0 && !exactMatch {
		fmt.Printf("\n💡 Found %d similar exercise(s):\n", len(similar))
		for _, ex := range similar {
			contextStr := "no context"
			if ex.Context != "" {
				contextStr = ex.Context
			}
			fmt.Printf("   - %s (%s)\n", ex.Name, contextStr)
		}
		fmt.Println()

		continueItems := []ui.SelectItem{
			{Label: "Yes, create new", Value: "yes"},
			{Label: "No, open similar", Value: "open"},
			{Label: "Cancel", Value: "cancel"},
		}

		_, continueVal, err := ui.RunSelect("Continue creating new exercise?", continueItems, 5)
		if err != nil || continueVal == "cancel" {
			fmt.Println("Cancelled")
			return nil
		}

		if continueVal == "open" {
			// Select which similar to open
			similarNames := make([]string, len(similar))
			for i, ex := range similar {
				contextStr := "no context"
				if ex.Context != "" {
					contextStr = ex.Context
				}
				similarNames[i] = fmt.Sprintf("%s (%s)", ex.Name, contextStr)
			}

			similarItems := make([]ui.SelectItem, len(similarNames))
			for i, name := range similarNames {
				similarItems[i] = ui.SelectItem{Label: name, Value: fmt.Sprintf("%d", i)}
			}

			idx, _, err := ui.RunSelect("Select exercise to open", similarItems, 10)
			if err == nil && idx < len(similar) {
				fmt.Printf("Opening: %s\n", similar[idx].FilePath)
				if err := promptAndOpenEditor(similar[idx].FilePath); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			}
			return nil
		}
	}

	// Description
	exercise.Description, _ = ui.RunInput("Description (optional)", "", "", nil)

	// Goal
	exercise.Goal, _ = ui.RunInput("Goal (optional)", "", "", nil)

	// Tags
	tagsInput, _ := ui.RunInput("Tags (comma-separated, optional)", "", "", nil)
	if tagsInput != "" {
		tags := strings.Split(tagsInput, ",")
		for _, tag := range tags {
			exercise.Tags = append(exercise.Tags, strings.TrimSpace(tag))
		}
	}

	// Variants - always create at least one
	fmt.Println("\n━━━ Variant Configuration ━━━")
	fmt.Println("Every exercise needs at least one variant.")
	fmt.Println("Variants define different ways to practice this exercise.")
	fmt.Println()

	variantNum := 1
	for {
		fmt.Printf("━━ Variant %d ━━\n", variantNum)

		variant := exercises.ExerciseVariant{
			TrackingProperties: make(map[string]string),
		}

		// Optional variant name
		variant.Name, _ = ui.RunInput("Variant Name (optional, e.g., 'Balance & Control')", "", "", nil)

		// Variant description
		variant.Description, _ = ui.RunInput("Description (optional, explain what makes this variant unique)", "", "", nil)

		// Tracking properties for this variant
		propMgr, err := exercises.NewPropertyManager(brainPath)
		if err != nil {
			fmt.Printf("Warning: Could not load property manager: %v\n", err)
		}

		fmt.Println("\nTracking properties for this variant:")
		for {
			// Get available properties
			propOptions := propMgr.GetPropertyNames()
			propOptions = append(propOptions, "➕ Add new property", "✓ Done")

			propItems := make([]ui.SelectItem, len(propOptions))
			for i, opt := range propOptions {
				propItems[i] = ui.SelectItem{Label: opt, Value: opt}
			}

			_, selected, err := ui.RunSelect("Select tracking property", propItems, 15)
			if err != nil || selected == "✓ Done" {
				break
			}

			if selected == "➕ Add new property" {
				// Create new property
				propName, err := ui.RunInput("Property name", "", "", nil)
				if err != nil || strings.TrimSpace(propName) == "" {
					continue
				}
				propName = strings.TrimSpace(propName)

				propUnit, err := ui.RunInput("Unit (e.g., bpm, count, kg, text)", "", "text", nil)
				if err != nil {
					continue
				}
				propUnit = strings.TrimSpace(propUnit)

				// Add to manager
				if err := propMgr.Add(propName, propUnit); err != nil {
					fmt.Printf("Warning: Could not save property: %v\n", err)
				}

				variant.TrackingProperties[propName] = propUnit
				fmt.Printf("✓ Added: %s (%s)\n", propName, propUnit)
			} else {
				// Selected existing property
				// Parse "name (unit)" format
				propName := strings.Split(selected, " (")[0]
				prop := propMgr.GetByName(propName)
				if prop != nil {
					variant.TrackingProperties[prop.Name] = prop.Unit
					fmt.Printf("✓ Added: %s (%s)\n", prop.Name, prop.Unit)
				}
			}
		}

		exercise.Variants = append(exercise.Variants, variant)

		// Ask if user wants to add another variant
		if variantNum == 1 {
			fmt.Println()
		}

		addMoreItems := []ui.SelectItem{
			{Label: "Yes", Value: "yes"},
			{Label: "No", Value: "no"},
		}
		_, addMoreVal, err := ui.RunSelect("Add another variant?", addMoreItems, 3)
		if err != nil || addMoreVal == "no" {
			break
		}

		fmt.Println()
		variantNum++
	}

	// Generate ID
	exercise.ID = generateExerciseID(exercise.Name)

	// Write exercise file
	parser := exercises.NewParser()
	exPath := parser.GetExerciseFilePath(brainPath, exercise.ID, string(detection.Type))
	if err := parser.WriteExercise(exercise, exPath); err != nil {
		return fmt.Errorf("creating exercise: %w", err)
	}

	fmt.Printf("✓ Exercise created: %s\n\n", exPath)

	// Ask what to do next
	nextItems := []ui.SelectItem{
		{Label: "View/Edit in editor", Value: "edit"},
		{Label: "Done - Track later with 'flip exercise track'", Value: "done"},
	}

	_, nextVal, err := ui.RunSelect("What would you like to do next?", nextItems, 3)
	if err != nil {
		return nil
	}

	switch nextVal {
	case "edit":
		if err := promptAndOpenEditor(exPath); err != nil {
			fmt.Printf("⚠️  Could not open editor: %v\n", err)
		}
	case "done":
		fmt.Printf("💡 To track a session, run: flip exercise track %s\n", exercise.ID)
	}
	return nil
}

func generateExerciseID(name string) string {
	// Convert name to lowercase, replace spaces with hyphens
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
