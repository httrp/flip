package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/git"
)

// displayStatusHeader shows a compact one-line status
func displayStatusHeader() {
	config, err := loadWorkspaceConfig()
	if err != nil || len(config.Workspaces) == 0 {
		fmt.Println("⚠️  No workspace configured · Run: flip quickstart")
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
		fmt.Println("⚠️  No active workspace")
		return
	}

	// Build compact status line
	parts := []string{}
	
	// Workspace
	parts = append(parts, fmt.Sprintf("📂 %s", activeWS.Name))

	// Default brain
	var defaultBrain *Brain
	if activeWS.DefaultBrain != "" {
		parts = append(parts, fmt.Sprintf("🧠 %s", activeWS.DefaultBrain))
		for i := range activeWS.Brains {
			if activeWS.Brains[i].Name == activeWS.DefaultBrain {
				defaultBrain = &activeWS.Brains[i]
				break
			}
		}
	} else if len(activeWS.Brains) > 0 {
		parts = append(parts, fmt.Sprintf("🧠 %s", activeWS.Brains[0].Name))
		defaultBrain = &activeWS.Brains[0]
	}

	// Git status (compact)
	if defaultBrain != nil {
		gitParts := getCompactGitStatus(defaultBrain)
		parts = append(parts, gitParts...)
		
		// Today's journal check
		if hasJournalToday(defaultBrain.Path) {
			parts = append(parts, "📓 ✓")
		}
		
		// Open tasks count
		if taskCount := getOpenTaskCount(defaultBrain.Path); taskCount > 0 {
			parts = append(parts, fmt.Sprintf("✅ %d", taskCount))
		}
	}

	fmt.Println(strings.Join(parts, " · "))
}

// getCompactGitStatus returns compact git info
func getCompactGitStatus(brain *Brain) []string {
	status, err := git.GetStatus(brain.Path)
	if err != nil || !status.IsRepo {
		return nil
	}

	parts := []string{}

	// Branch
	if status.Branch != "" {
		parts = append(parts, fmt.Sprintf("⎇ %s", status.Branch))
	}

	// Changes indicator (very compact)
	if status.HasChanges {
		total := status.ModifiedFiles + status.StagedFiles + status.UntrackedFiles
		parts = append(parts, fmt.Sprintf("⚡%d", total))
	} else {
		parts = append(parts, "✓")
	}

	// Remote sync
	if status.AheadBehind != "" && status.AheadBehind != "up to date" {
		parts = append(parts, "🔄")
	}

	return parts
}

// hasJournalToday checks if today's journal exists
func hasJournalToday(brainPath string) bool {
	today := time.Now().Format("2006-01-02")
	journalPath := filepath.Join(brainPath, "journal", today+".md")
	_, err := os.Stat(journalPath)
	return err == nil
}

// getOpenTaskCount counts open tasks in the brain
func getOpenTaskCount(brainPath string) int {
	tasksDir := filepath.Join(brainPath, "tasks")
	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		return 0
	}

	count := 0
	filepath.Walk(tasksDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		
		// Count unchecked task markers
		content := string(data)
		count += strings.Count(content, "- [ ]")
		count += strings.Count(content, "* [ ]")
		
		return nil
	})

	return count
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
