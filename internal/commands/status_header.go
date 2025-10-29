package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/httrp/flip/internal/git"
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
	var defaultBrain *Brain
	if activeWS.DefaultBrain != "" {
		fmt.Printf(" | 🧠 Brain: %s", activeWS.DefaultBrain)
		for i := range activeWS.Brains {
			if activeWS.Brains[i].Name == activeWS.DefaultBrain {
				defaultBrain = &activeWS.Brains[i]
				break
			}
		}
	} else if len(activeWS.Brains) > 0 {
		fmt.Printf(" | 🧠 Brain: %s", activeWS.Brains[0].Name)
		defaultBrain = &activeWS.Brains[0]
	} else {
		fmt.Printf(" | 🧠 Brain: (none)")
	}

	// Show git status for default brain
	if defaultBrain != nil {
		showBrainGitStatus(defaultBrain)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// showBrainGitStatus displays git status information for a brain
func showBrainGitStatus(brain *Brain) {
	status, err := git.GetStatus(brain.Path)
	if err != nil || !status.IsRepo {
		return // Silently skip if not a git repo
	}

	// Build status line
	var statusParts []string

	// Branch info
	if status.Branch != "" {
		statusParts = append(statusParts, fmt.Sprintf("⎇ %s", status.Branch))
	}

	// Changes indicator
	if status.HasChanges {
		changeInfo := []string{}
		if status.ModifiedFiles > 0 {
			changeInfo = append(changeInfo, fmt.Sprintf("%d modified", status.ModifiedFiles))
		}
		if status.StagedFiles > 0 {
			changeInfo = append(changeInfo, fmt.Sprintf("%d staged", status.StagedFiles))
		}
		if status.UntrackedFiles > 0 {
			changeInfo = append(changeInfo, fmt.Sprintf("%d untracked", status.UntrackedFiles))
		}
		statusParts = append(statusParts, "⚡ "+strings.Join(changeInfo, ", "))
	} else {
		statusParts = append(statusParts, "✓ clean")
	}

	// Last commit info
	if status.LastCommitHash != "" {
		timeSince := formatTimeSince(status.LastCommitDate)
		commitMsg := truncate(status.LastCommitMsg, 40)
		statusParts = append(statusParts, fmt.Sprintf("📝 %s: %s (%s)", status.LastCommitHash, commitMsg, timeSince))
	}

	// Remote sync status
	if status.AheadBehind != "" && status.AheadBehind != "up to date" {
		statusParts = append(statusParts, "🔄 "+status.AheadBehind)
	}

	if len(statusParts) > 0 {
		fmt.Println()
		fmt.Printf("   %s\n", strings.Join(statusParts, " | "))
	}
}

// formatTimeSince formats a time duration in a human-readable way
func formatTimeSince(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}

	years := int(duration.Hours() / 24 / 365)
	if years == 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}
