package commands

import (
	"archive/zip"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// ExtensionVersion must match the version in vscode-extension/package.json
const ExtensionVersion = "0.1.3"
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
		targetPath := filepath.Join(extensionDir, file.Name)

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
	return nil
}

// getVSCodeExtensionsDir returns the path to VS Code's extensions directory
// Handles multiple profiles - installs to the standard extensions directory
// which is shared across all profiles
func getVSCodeExtensionsDir() (string, error) {
	var vscodeDir string

	switch runtime.GOOS {
	case "darwin":
		// macOS: ~/Library/Application Support/Code
		home, _ := os.UserHomeDir()
		vscodeDir = filepath.Join(home, "Library", "Application Support", "Code")

	case "linux":
		// Linux: ~/.config/Code
		home, _ := os.UserHomeDir()
		vscodeDir = filepath.Join(home, ".config", "Code")

	case "windows":
		// Windows: %APPDATA%\Code
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		vscodeDir = filepath.Join(appData, "Code")

	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// Use standard extensions directory (shared across all profiles)
	// This is the most reliable location that works with all VS Code versions
	extensionsDir := filepath.Join(vscodeDir, "extensions")
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
