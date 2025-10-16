package commands

import (
	"fmt"
)

// displayStatusHeader shows the current workspace and default brain
func displayStatusHeader() {
	config, err := loadWorkspaceConfig()
	if err != nil || len(config.Workspaces) == 0 {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  No workspace configured")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return
	}

	var activeWS *Workspace
	for i := range config.Workspaces {
		if config.Workspaces[i].Name == config.ActiveWorkspace {
			activeWS = &config.Workspaces[i]
			break
		}
	}

	if activeWS == nil {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  No active workspace")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		return
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📂 Workspace: %s", activeWS.Name)

	// Show default brain if set
	if activeWS.DefaultBrain != "" {
		fmt.Printf(" | 🧠 Brain: %s", activeWS.DefaultBrain)
	} else if len(activeWS.Brains) > 0 {
		fmt.Printf(" | 🧠 Brain: %s", activeWS.Brains[0].Name)
	} else {
		fmt.Printf(" | 🧠 Brain: (none)")
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
