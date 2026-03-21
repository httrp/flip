package commands

import (
	"fmt"

	"github.com/httrp/flip/internal/ui"
)

// selectBrainForOperation prompts the user to choose a brain from the active workspace.
// If only one brain exists, it is returned without prompting.
func selectBrainForOperation(promptLabel string) (*Brain, error) {
	ws, err := getActiveWorkspace()
	if err != nil {
		return nil, err
	}

	if len(ws.Brains) == 0 {
		return nil, fmt.Errorf("no brains in active workspace")
	}

	if len(ws.Brains) == 1 {
		// Only one available, use it directly
		return &ws.Brains[0], nil
	}

	// Build choices, highlighting default brain
	items := make([]ui.SelectItem, 0, len(ws.Brains))
	for i := range ws.Brains {
		b := ws.Brains[i]
		marker := "  "
		if b.Name == ws.DefaultBrain {
			marker = "★ "
		}
		items = append(items, ui.SelectItem{
			Label: fmt.Sprintf("%s%s  —  %s", marker, b.Name, b.Path),
			Value: b.Name,
		})
	}

	idx, _, err := ui.RunSelect(promptLabel, items, 0)
	if err != nil {
		return nil, fmt.Errorf("selection cancelled")
	}
	if idx >= 0 && idx < len(ws.Brains) {
		return &ws.Brains[idx], nil
	}
	return nil, fmt.Errorf("invalid selection")
}
