package commands

import (
	"archive/zip"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// ExtensionVersion must match the version in vscode-extension/package.json
const ExtensionVersion = "0.3.16"
const ExtensionID = "danorama.flip-vscode"

//go:embed assets/flip-vscode.vsix
var embeddedVSIX embed.FS

// Extension installation commands are added to the vscode command group

func newVSCodeExtensionInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install-extension",
		Short: "Install the Flip VS Code extension",
		Long: `Install the Flip VS Code extension for native integration.

The extension provides:
- Command palette integration (Cmd+Shift+P → "Flip:")
- Keyboard shortcuts (Cmd+Alt+J for journal, etc.)
- Status bar showing active brain
- Native VS Code dialogs instead of terminal prompts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installExtension()
		},
	}
}

func newVSCodeExtensionStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "extension-status",
		Short: "Show VS Code extension installation status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showExtensionStatus()
		},
	}
}

func newVSCodeExtensionUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall-extension",
		Short: "Uninstall the Flip VS Code extension",
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallExtension()
		},
	}
}

// getInstalledExtensionVersion returns the installed version or empty string if not installed
func getInstalledExtensionVersion() string {
	cmd := exec.Command("code", "--list-extensions", "--show-versions")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(strings.ToLower(line), strings.ToLower(ExtensionID)) {
			// Format: publisher.name@version
			parts := strings.Split(line, "@")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// isVSCodeAvailable checks if the 'code' command is available
func isVSCodeAvailable() bool {
	_, err := exec.LookPath("code")
	return err == nil
}

func showExtensionStatus() error {
	fmt.Println("\nVS Code Extension Status")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !isVSCodeAvailable() {
		fmt.Println("❌ VS Code 'code' command not found")
		fmt.Println("   Install VS Code and ensure 'code' is in your PATH")
		return nil
	}

	installedVersion := getInstalledExtensionVersion()

	if installedVersion == "" {
		fmt.Println("❌ Extension: not installed")
	} else if installedVersion == ExtensionVersion {
		fmt.Printf("✅ Extension: v%s (up to date)\n", installedVersion)
	} else {
		fmt.Printf("⚠️  Extension: v%s (update available: v%s)\n", installedVersion, ExtensionVersion)
	}

	fmt.Printf("📦 Bundled version: v%s\n", ExtensionVersion)

	// Check if running in VS Code terminal
	if os.Getenv("TERM_PROGRAM") == "vscode" || os.Getenv("VSCODE_IPC_HOOK_CLI") != "" {
		fmt.Println("🖥️  Running in VS Code terminal")
	}

	return nil
}

func installExtension() error {
	fmt.Println("\n📦 Installing Flip VS Code Extension...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !isVSCodeAvailable() {
		return fmt.Errorf("VS Code 'code' command not found. Install VS Code and ensure 'code' is in your PATH")
	}

	// Get VS Code extensions directory to check if extension exists in current profile
	extensionsDir, _ := getVSCodeExtensionsDir()
	extensionDir := filepath.Join(extensionsDir, fmt.Sprintf("danorama.flip-vscode-%s", ExtensionVersion))
	extensionExistsLocally := false
	if _, err := os.Stat(extensionDir); err == nil {
		extensionExistsLocally = true
	}

	// Check current installation via CLI
	installedVersion := getInstalledExtensionVersion()

	// Only skip if installed AND exists locally in this profile
	if installedVersion == ExtensionVersion && extensionExistsLocally {
		_ = updateProfileExtensionsJSON(extensionDir)
		fmt.Printf("✅ Extension v%s is already installed and up to date in this profile.\n", installedVersion)
		return nil
	}

	if installedVersion != "" && !extensionExistsLocally {
		fmt.Printf("📍 Extension found in VS Code but not in this profile, installing locally...\n")
	} else if installedVersion != "" {
		fmt.Printf("📤 Updating from v%s to v%s...\n", installedVersion, ExtensionVersion)
	} else {
		fmt.Printf("📥 Installing v%s...\n", ExtensionVersion)
	}

	// Extract embedded VSIX to temp file
	vsixData, err := embeddedVSIX.ReadFile("assets/flip-vscode.vsix")
	if err != nil {
		return fmt.Errorf("failed to read embedded extension: %w", err)
	}

	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "flip-vscode.vsix")
	if err := os.WriteFile(tempFile, vsixData, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(tempFile)

	// Try direct profile installation first (works with multiple profiles)
	if err := installToVSCodeProfile(tempFile); err == nil {
		fmt.Println("\n✅ Extension installed successfully!")
		fmt.Println("\n💡 Reload VS Code to activate (Cmd+Shift+P → Reload Window)")
		return nil
	}

	// Fallback: Try CLI installation (doesn't work with profiles but is simpler)
	fmt.Println("⚠️  Direct installation failed, trying CLI method...")
	cmd := exec.Command("code", "--install-extension", tempFile, "--force")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install extension: %w", err)
	}

	fmt.Println("\n✅ Extension installed successfully!")
	fmt.Println("\n⚠️  NOTE: If the extension doesn't appear in VS Code immediately:")
	fmt.Println("   1. Reload VS Code (Cmd+Shift+P → 'Reload Window')")
	fmt.Println("   2. Or manually install via: Extensions → Install from VSIX...")
	fmt.Println("      → " + tempFile)
	fmt.Println("\n💡 Available commands (Cmd+Shift+P):")
	fmt.Println("   • Flip: Open/Create Journal  (Cmd+Alt+J)")
	fmt.Println("   • Flip: New Note            (Cmd+Alt+N)")
	fmt.Println("   • Flip: Quick Note          (Cmd+Alt+Q)")
	fmt.Println("   • Flip: New Task")
	fmt.Println("   • Flip: Switch Brain")
	fmt.Println("   • Flip: Show Status")

	return nil
}

// installToVSCodeProfile extracts the VSIX directly into VS Code's extensions directory
// This method works with multiple profiles (unlike the `code` CLI)
func installToVSCodeProfile(vsixPath string) error {
	// Get VS Code extensions directory
	extensionsDir, err := getVSCodeExtensionsDir()
	if err != nil {
		return fmt.Errorf("could not find VS Code extensions directory: %w", err)
	}

	// Target extension directory
	extensionDir := filepath.Join(extensionsDir, fmt.Sprintf("danorama.flip-vscode-%s", ExtensionVersion))

	// Remove any old flip-vscode installations (any version)
	entries, _ := os.ReadDir(extensionsDir)
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "danorama.flip-vscode-") {
			oldDir := filepath.Join(extensionsDir, entry.Name())
			fmt.Printf("Removing old installation at %s\n", oldDir)
			if err := os.RemoveAll(oldDir); err != nil {
				fmt.Printf("Warning: could not remove %s: %v\n", oldDir, err)
			}
		}
	}

	// Create extension directory
	if err := os.MkdirAll(extensionDir, 0755); err != nil {
		return fmt.Errorf("failed to create extension directory: %w", err)
	}

	// Extract VSIX (it's a ZIP file)
	reader, err := zip.OpenReader(vsixPath)
	if err != nil {
		return fmt.Errorf("failed to open VSIX: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		// VSIX files have "extension/" prefix - strip it
		// Skip non-extension files like [Content_Types].xml
		name := file.Name
		if strings.HasPrefix(name, "extension/") {
			name = strings.TrimPrefix(name, "extension/")
		} else {
			// Skip metadata files at root level
			continue
		}

		if name == "" {
			continue
		}

		targetPath := filepath.Join(extensionDir, name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(targetPath, 0755)
		} else {
			// Create parent directories
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			// Extract file
			source, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open file in VSIX: %w", err)
			}

			dest, err := os.Create(targetPath)
			if err != nil {
				source.Close()
				return fmt.Errorf("failed to create file: %w", err)
			}

			_, err = io.Copy(dest, source)
			source.Close()
			dest.Close()

			if err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}
		}
	}

	fmt.Printf("✅ Installed to: %s\n", extensionDir)

	// Update extensions.json in all VS Code profiles
	fmt.Println("\n→ Updating VS Code profiles...")
	if err := updateProfileExtensionsJSON(extensionDir); err != nil {
		fmt.Printf("⚠️  Warning: could not update all profiles: %v\n", err)
	}

	return nil
}

// getVSCodeExtensionsDir returns the path to VS Code's extensions directory
// Uses ~/.vscode/extensions which is shared across all profiles
func getVSCodeExtensionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}

	// Use ~/.vscode/extensions (shared across all profiles on all platforms)
	extensionsDir := filepath.Join(home, ".vscode", "extensions")
	if err := os.MkdirAll(extensionsDir, 0755); err != nil {
		return "", err
	}

	fmt.Println("📂 Using VS Code extensions directory (shared across all profiles)")
	return extensionsDir, nil
}

func uninstallExtension() error {
	fmt.Println("\n🗑️  Uninstalling Flip VS Code Extension...")

	if !isVSCodeAvailable() {
		return fmt.Errorf("VS Code 'code' command not found")
	}

	installedVersion := getInstalledExtensionVersion()
	if installedVersion == "" {
		fmt.Println("ℹ️  Extension is not installed.")
		return nil
	}

	cmd := exec.Command("code", "--uninstall-extension", ExtensionID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall extension: %w", err)
	}

	fmt.Println("✅ Extension uninstalled.")
	return nil
}

// getVSCodeUserDir returns the VS Code user data directory
func getVSCodeUserDir() (string, error) {
	var userDir string

	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		userDir = filepath.Join(home, "Library", "Application Support", "Code", "User")
	case "linux":
		home, _ := os.UserHomeDir()
		userDir = filepath.Join(home, ".config", "Code", "User")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		userDir = filepath.Join(appData, "Code", "User")
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return userDir, nil
}

// updateProfileExtensionsJSON updates the extensions.json in each VS Code profile
// This is necessary because the VS Code CLI doesn't properly update profile-specific extensions.json
func updateProfileExtensionsJSON(extensionPath string) error {
	userDir, err := getVSCodeUserDir()
	if err != nil {
		return err
	}

	profilesDir := filepath.Join(userDir, "profiles")
	if _, err := os.Stat(profilesDir); os.IsNotExist(err) {
		// No profiles, nothing to update
		return nil
	}

	// Read profile names from storage.json
	profileNames := make(map[string]string)
	storageFile := filepath.Join(userDir, "globalStorage", "storage.json")
	if data, err := os.ReadFile(storageFile); err == nil {
		var storage map[string]interface{}
		if json.Unmarshal(data, &storage) == nil {
			if profiles, ok := storage["userDataProfiles"].([]interface{}); ok {
				for _, p := range profiles {
					if profile, ok := p.(map[string]interface{}); ok {
						loc, _ := profile["location"].(string)
						name, _ := profile["name"].(string)
						if loc != "" && name != "" {
							profileID := filepath.Base(strings.ReplaceAll(loc, "\\", "/"))
							profileNames[profileID] = name
						}
					}
				}
			}
		}
	}

	// Iterate through all profiles
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil // Not an error if we can't read profiles
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		profileID := entry.Name()
		extensionsFile := filepath.Join(profilesDir, profileID, "extensions.json")

		if updated := updateExtensionsFile(extensionsFile, extensionPath); updated {
			profileName := profileNames[profileID]
			if profileName == "" {
				profileName = profileID
			}
			fmt.Printf("  ✓ Updated profile: %s\n", profileName)
		}
	}

	// Also update default profile's extensions.json if it exists
	defaultExtFile := filepath.Join(userDir, "extensions.json")
	if _, err := os.Stat(defaultExtFile); err == nil {
		if updateExtensionsFile(defaultExtFile, extensionPath) {
			fmt.Println("  ✓ Updated default profile")
		}
	}

	return nil
}

// updateExtensionsFile updates a single extensions.json file
// Uses map[string]interface{} to preserve all fields (VS Code adds many extra fields)
func updateExtensionsFile(filePath, extensionPath string) bool {
	var extensions []map[string]interface{}
	if data, err := os.ReadFile(filePath); err == nil {
		if err := json.Unmarshal(data, &extensions); err != nil {
			return false
		}
	} else if !os.IsNotExist(err) {
		return false
	}

	// Find and update flip extension
	found := false
	for i, ext := range extensions {
		identifier, ok := ext["identifier"].(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := identifier["id"].(string)
		if id != ExtensionID {
			continue
		}

		found = true
		// Update version
		extensions[i]["version"] = ExtensionVersion
		extensions[i]["relativeLocation"] = fmt.Sprintf("danorama.flip-vscode-%s", ExtensionVersion)

		// Update location (preserve other fields like $mid, scheme)
		if location, ok := ext["location"].(map[string]interface{}); ok {
			location["path"] = extensionPath
			extensions[i]["location"] = location
		} else {
			extensions[i]["location"] = map[string]interface{}{"path": extensionPath}
		}

		// Update metadata (preserve other fields)
		if metadata, ok := ext["metadata"].(map[string]interface{}); ok {
			metadata["pinned"] = false
			metadata["installedTimestamp"] = time.Now().UnixMilli()
			extensions[i]["metadata"] = metadata
		} else {
			extensions[i]["metadata"] = map[string]interface{}{
				"pinned":             false,
				"installedTimestamp": time.Now().UnixMilli(),
			}
		}
		break
	}

	if !found {
		extensions = append(extensions, map[string]interface{}{
			"identifier": map[string]interface{}{
				"id": ExtensionID,
			},
			"version":          ExtensionVersion,
			"relativeLocation": fmt.Sprintf("danorama.flip-vscode-%s", ExtensionVersion),
			"location": map[string]interface{}{
				"path": extensionPath,
			},
			"metadata": map[string]interface{}{
				"pinned":             false,
				"installedTimestamp": time.Now().UnixMilli(),
			},
		})
	}

	// Write back with same formatting
	newData, err := json.MarshalIndent(extensions, "", "  ")
	if err != nil {
		return false
	}

	if err := os.WriteFile(filePath, newData, 0644); err != nil {
		return false
	}

	return true
}
