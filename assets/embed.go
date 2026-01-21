// Package assets provides embedded static files for flip.
// This includes VS Code tasks configuration and other assets.
package assets

import (
	"embed"
	"strings"
)

//go:embed vscode-tasks.json
var vscodeTasks embed.FS

// GetVSCodeTasksJSON returns the VS Code tasks configuration JSON.
// The {{VERSION}} placeholder is replaced with the provided version string.
func GetVSCodeTasksJSON(version string) (string, error) {
	data, err := vscodeTasks.ReadFile("vscode-tasks.json")
	if err != nil {
		return "", err
	}
	
	// Replace version placeholder
	content := strings.Replace(string(data), "{{VERSION}}", version, 1)
	return content, nil
}
