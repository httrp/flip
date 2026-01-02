package commands

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ExtensionVersion must match the version in vscode-extension/package.json
const ExtensionVersion = "0.1.0"
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

	// Check current installation
	installedVersion := getInstalledExtensionVersion()
	
	if installedVersion == ExtensionVersion {
		fmt.Printf("✅ Extension v%s is already installed and up to date.\n", installedVersion)
		return nil
	}

	if installedVersion != "" {
		fmt.Printf("📤 Updating from v%s to v%s...\n", installedVersion, ExtensionVersion)
	} else {
		fmt.Printf("📥 Installing v%s...\n", ExtensionVersion)
	}

	// Extract embedded VSIX to temp file
	vsixData, err := embeddedVSIX.ReadFile("assets/flip-vscode.vsix")
	if err != nil {
		return fmt.Errorf("failed to read embedded extension: %w", err)
	}

	tempFile := filepath.Join(os.TempDir(), "flip-vscode.vsix")
	if err := os.WriteFile(tempFile, vsixData, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(tempFile)

	// Install via code CLI
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
