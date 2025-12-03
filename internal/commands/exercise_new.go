package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/exercises"
	"github.com/httrp/flip/internal/lang"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

// ExerciseNewCmd creates a new exercise definition
var ExerciseNewCmd = &cobra.Command{
	Use:   "new",
	Short: lang.GetText("exercise.new.short"),
	Long:  lang.GetText("exercise.new.long"),
	Run:   runExerciseNew,
}

func runExerciseNew(cmd *cobra.Command, args []string) {
	activeBrain, err := getActiveBrain()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	
	brainPath := activeBrain.Path
	detection, err := brain.NewDetector().DetectBrainType(brainPath)
		if err != nil {
			fmt.Printf("Error detecting brain: %v\n", err)
			os.Exit(1)
		}

		if !detection.Compatible {
			fmt.Println("Error: Not in a compatible brain directory")
		fmt.Println("Error: Not in a brain directory")
		os.Exit(1)
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
	
	contextSelect := promptui.Select{
		Label: "Select context",
		Items: contexts,
		Size:  12,
	}
	
	_, selectedContext, err := contextSelect.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	
	if selectedContext == "➕ New context" {
		contextPrompt := promptui.Prompt{
			Label: "New context name (e.g., Sport, Music, Basketball, Drums)",
		}
		exercise.Context, _ = contextPrompt.Run()
	} else if selectedContext != "⊝ No context" {
		exercise.Context = selectedContext
	}

	// Name
	namePrompt := promptui.Prompt{
		Label: "Exercise Name",
	}
	exercise.Name, err = namePrompt.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
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
		
		actionPrompt := promptui.Select{
			Label: "What would you like to do?",
			Items: []string{
				"Open existing exercise",
				"Create anyway with different name",
				"Cancel",
			},
		}
		
		actionIdx, _, err := actionPrompt.Run()
		if err != nil || actionIdx == 2 {
			fmt.Println("Cancelled")
			return
		}
		
		if actionIdx == 0 {
			// Find and open existing
			for _, ex := range allExercises {
				if strings.EqualFold(ex.Name, exercise.Name) {
					fmt.Printf("Opening: %s\n", ex.FilePath)
					if err := promptAndOpenEditor(ex.FilePath); err != nil {
						fmt.Printf("Error: %v\n", err)
					}
					return
				}
			}
		}
		
		// Ask for new name
		newNamePrompt := promptui.Prompt{
			Label: "New exercise name",
		}
		exercise.Name, err = newNamePrompt.Run()
		if err != nil {
			return
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
		
		continuePrompt := promptui.Select{
			Label: "Continue creating new exercise?",
			Items: []string{"Yes, create new", "No, open similar", "Cancel"},
		}
		
		continueIdx, _, err := continuePrompt.Run()
		if err != nil || continueIdx == 2 {
			fmt.Println("Cancelled")
			return
		}
		
		if continueIdx == 1 {
			// Select which similar to open
			similarNames := make([]string, len(similar))
			for i, ex := range similar {
				contextStr := "no context"
				if ex.Context != "" {
					contextStr = ex.Context
				}
				similarNames[i] = fmt.Sprintf("%s (%s)", ex.Name, contextStr)
			}
			
			selectPrompt := promptui.Select{
				Label: "Select exercise to open",
				Items: similarNames,
			}
			
			idx, _, err := selectPrompt.Run()
			if err == nil && idx < len(similar) {
				fmt.Printf("Opening: %s\n", similar[idx].FilePath)
				if err := promptAndOpenEditor(similar[idx].FilePath); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
			}
			return
		}
	}

	// Description
	descPrompt := promptui.Prompt{
		Label: "Description (optional)",
	}
	exercise.Description, _ = descPrompt.Run()

	// Goal
	goalPrompt := promptui.Prompt{
		Label: "Goal (optional)",
	}
	exercise.Goal, _ = goalPrompt.Run()

	// Tags
	tagsPrompt := promptui.Prompt{
		Label: "Tags (comma-separated, optional)",
	}
	tagsInput, _ := tagsPrompt.Run()
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
		varNamePrompt := promptui.Prompt{
			Label: fmt.Sprintf("Variant Name (optional, e.g., 'Balance & Control')"),
		}
		variant.Name, _ = varNamePrompt.Run()
		
		// Variant description
		varDescPrompt := promptui.Prompt{
			Label: "Description (optional, explain what makes this variant unique)",
		}
		variant.Description, _ = varDescPrompt.Run()
		
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
			
			selectPrompt := promptui.Select{
				Label: "Select tracking property",
				Items: propOptions,
				Size:  15,
			}
			
			_, selected, err := selectPrompt.Run()
			if err != nil || selected == "✓ Done" {
				break
			}
			
			if selected == "➕ Add new property" {
				// Create new property
				namePrompt := promptui.Prompt{
					Label: "Property name",
				}
				propName, err := namePrompt.Run()
				if err != nil || strings.TrimSpace(propName) == "" {
					continue
				}
				propName = strings.TrimSpace(propName)
				
				unitPrompt := promptui.Prompt{
					Label:   "Unit (e.g., bpm, count, kg, text)",
					Default: "text",
				}
				propUnit, err := unitPrompt.Run()
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
		
		addMorePrompt := promptui.Select{
			Label: "Add another variant?",
			Items: []string{"Yes", "No"},
		}
		addIdx, _, err := addMorePrompt.Run()
		if err != nil || addIdx == 1 {
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
		fmt.Printf("Error creating exercise: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Exercise created: %s\n\n", exPath)
	
	// Ask what to do next
	nextPrompt := promptui.Select{
		Label: "What would you like to do next?",
		Items: []string{"View/Edit in editor", "Done - Track later with 'flip exercise track'"},
	}
	
	nextIdx, _, err := nextPrompt.Run()
	if err != nil {
		return
	}
	
	switch nextIdx {
	case 0: // View/Edit
		if err := promptAndOpenEditor(exPath); err != nil {
			fmt.Printf("⚠️  Could not open editor: %v\n", err)
		}
	case 1: // Done
		fmt.Printf("💡 To track a session, run: flip exercise track %s\n", exercise.ID)
	}
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
