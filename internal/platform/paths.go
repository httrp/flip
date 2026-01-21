// Package platform provides cross-platform path resolution and OS-specific utilities.
// This package abstracts away platform differences for cloud storage paths,
// configuration directories, and other OS-specific locations.
package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CloudStorageInfo represents a cloud storage service with its paths
type CloudStorageInfo struct {
	Name      string
	Available bool
	Path      string
}

// GetCloudStoragePaths returns available cloud storage paths for the current platform.
// Returns a map of display name -> path for cloud services that exist on the system.
func GetCloudStoragePaths() map[string]string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	paths := make(map[string]string)

	// Common paths across all platforms
	commonPaths := map[string]string{
		"Home Directory": home,
		"Documents":      filepath.Join(home, "Documents"),
	}

	for name, path := range commonPaths {
		if pathExists(path) {
			paths[name] = path
		}
	}

	// Platform-specific cloud storage paths
	switch runtime.GOOS {
	case "darwin":
		addMacOSCloudPaths(paths, home)
	case "windows":
		addWindowsCloudPaths(paths, home)
	case "linux":
		addLinuxCloudPaths(paths, home)
	}

	return paths
}

// addMacOSCloudPaths adds macOS-specific cloud storage paths
func addMacOSCloudPaths(paths map[string]string, home string) {
	cloudPaths := map[string]string{
		"Dropbox":                    filepath.Join(home, "Dropbox"),
		"iCloud Drive":               filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs"),
		"iCloud Drive (Mobile Docs)": filepath.Join(home, "Library", "Mobile Documents"),
		"Google Drive":               filepath.Join(home, "Google Drive"),
		"OneDrive":                   filepath.Join(home, "OneDrive"),
		"OneDrive - Personal":        filepath.Join(home, "OneDrive - Personal"),
	}

	for name, path := range cloudPaths {
		if pathExists(path) {
			paths[name] = path
		}
	}

	// Check for Obsidian vaults in common locations
	obsidianPaths := []string{
		filepath.Join(home, "Documents", "Obsidian"),
		filepath.Join(home, "Dropbox", "Obsidian"),
		filepath.Join(home, "Library", "Mobile Documents", "iCloud~md~obsidian", "Documents"),
	}
	for _, p := range obsidianPaths {
		if pathExists(p) {
			paths["Obsidian Vaults"] = p
			break
		}
	}

	// Check for Logseq
	logseqPaths := []string{
		filepath.Join(home, "Documents", "Logseq"),
		filepath.Join(home, "Dropbox", "Logseq"),
	}
	for _, p := range logseqPaths {
		if pathExists(p) {
			paths["Logseq"] = p
			break
		}
	}
}

// addWindowsCloudPaths adds Windows-specific cloud storage paths
func addWindowsCloudPaths(paths map[string]string, home string) {
	cloudPaths := map[string]string{
		"Dropbox":             filepath.Join(home, "Dropbox"),
		"Google Drive":        filepath.Join(home, "Google Drive"),
		"OneDrive":            filepath.Join(home, "OneDrive"),
		"OneDrive - Personal": filepath.Join(home, "OneDrive - Personal"),
		"iCloud Drive":        filepath.Join(home, "iCloudDrive"),
	}

	for name, path := range cloudPaths {
		if pathExists(path) {
			paths[name] = path
		}
	}

	// Windows-specific: Check for Google Drive File Stream
	gdfsPaths := []string{
		"G:\\My Drive",
		"G:\\Shared drives",
	}
	for _, p := range gdfsPaths {
		if pathExists(p) {
			paths["Google Drive File Stream"] = p
			break
		}
	}

	// Check for Obsidian
	obsidianPath := filepath.Join(home, "Documents", "Obsidian")
	if pathExists(obsidianPath) {
		paths["Obsidian Vaults"] = obsidianPath
	}

	// Check for Logseq
	logseqPath := filepath.Join(home, "Documents", "Logseq")
	if pathExists(logseqPath) {
		paths["Logseq"] = logseqPath
	}
}

// addLinuxCloudPaths adds Linux-specific cloud storage paths
func addLinuxCloudPaths(paths map[string]string, home string) {
	cloudPaths := map[string]string{
		"Dropbox":      filepath.Join(home, "Dropbox"),
		"Google Drive": filepath.Join(home, "google-drive"), // rclone mount point
		"OneDrive":     filepath.Join(home, "OneDrive"),     // rclone mount point
	}

	for name, path := range cloudPaths {
		if pathExists(path) {
			paths[name] = path
		}
	}

	// Check for Flatpak/Snap specific paths
	flatpakConfig := filepath.Join(home, ".var", "app")
	if pathExists(flatpakConfig) {
		paths["Flatpak Apps"] = flatpakConfig
	}

	// Check for Obsidian
	obsidianPath := filepath.Join(home, "Documents", "Obsidian")
	if pathExists(obsidianPath) {
		paths["Obsidian Vaults"] = obsidianPath
	}

	// Check for Logseq
	logseqPath := filepath.Join(home, "Documents", "Logseq")
	if pathExists(logseqPath) {
		paths["Logseq"] = logseqPath
	}
}

// GetVSCodeConfigPath returns the VS Code configuration directory for the current platform.
func GetVSCodeConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "Code", "User"), nil
	case "linux":
		// Check for Flatpak VS Code first
		flatpakPath := filepath.Join(home, ".var", "app", "com.visualstudio.code", "config", "Code", "User")
		if pathExists(flatpakPath) {
			return flatpakPath, nil
		}
		// Standard Linux path
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, "Code", "User"), nil
	default:
		// Fallback for other Unix-like systems
		return filepath.Join(home, ".config", "Code", "User"), nil
	}
}

// GetFlipConfigDir returns the flip configuration directory for the current platform.
func GetFlipConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "flip"), nil
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "flip"), nil
	case "linux":
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(home, ".config")
		}
		return filepath.Join(configDir, "flip"), nil
	default:
		return filepath.Join(home, ".flip"), nil
	}
}

// pathExists checks if a path exists
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsRunningInVSCode returns true if the application is running in a VS Code terminal.
func IsRunningInVSCode() bool {
	// VS Code sets TERM_PROGRAM=vscode
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return true
	}
	// Also check for VSCODE_* environment variables
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "VSCODE_") {
			return true
		}
	}
	return false
}

// GetOS returns the current operating system as a user-friendly string.
func GetOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}
