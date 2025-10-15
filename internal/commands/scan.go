package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/spf13/cobra"
)

func NewScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan a directory for 2nd brain workspaces",
		Long:  "Searches a directory (default: home) for Obsidian vaults, Logseq graphs, Dendron workspaces, and Flip brains.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var scanPath string
			if len(args) > 0 {
				scanPath = args[0]
			} else {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get home directory: %w", err)
				}
				scanPath = home
			}
			// Always show and allow to change path interactively
			fmt.Printf("🔍 Current scan path: %s\n", scanPath)
			fmt.Printf("❓ Enter new path to scan or press Enter to use this: ")
			var input string
			fmt.Scanln(&input)
			input = strings.TrimSpace(input)
			if input != "" {
				scanPath = input
			}
			return runScan(scanPath)
		},
	}
	return cmd
}

func runScan(scanPath string) error {
	fmt.Printf("🔍 Scanning %s for 2nd brain workspaces...\n\n", scanPath)
	found := 0
	maxDepth := 5
	excludeDirs := []string{".vscode", ".oh-my-zsh", "Library", "AppData", "Program Files", "node_modules", "Applications"}

	err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !info.IsDir() {
			return nil
		}

		// Limit scan depth for performance
		if depth(path, scanPath) > maxDepth {
			return filepath.SkipDir
		}

		// Exclude system and irrelevant folders
		for _, ex := range excludeDirs {
			if strings.Contains(path, string(os.PathSeparator)+ex+string(os.PathSeparator)) || strings.HasSuffix(path, string(os.PathSeparator)+ex) {
				return filepath.SkipDir
			}
		}

		detector := brain.NewDetector()
		result, err := detector.DetectBrainType(path)
		if err != nil {
			return nil
		}

		// Only show if clear marker and plausible number of markdown files
		mdCount := countMarkdownFiles(path)
		isValid := false
		switch result.Type {
		case brain.BrainTypeObsidian:
			isValid = hasDir(path, ".obsidian") && mdCount > 5
		case brain.BrainTypeLogseq:
			isValid = hasDir(path, ".logseq") && (hasDir(path, "journals") || hasDir(path, "pages")) && mdCount > 5
		case brain.BrainTypeDendron:
			isValid = fileExists(path, "dendron.yml") && mdCount > 5
		case brain.BrainTypeFlip:
			isValid = fileExists(path, ".flip-brain.yaml") || fileExists(path, ".flip.yaml")
		}

		if isValid {
			found++
			lastMod := getLastModified(path)
			fmt.Printf("%d) %s\n   Type: %s\n   Path: %s\n   Markdown files: %d\n   Last updated: %s\n   Indicators: %s\n\n", found, result.Description, result.Type, result.Path, mdCount, lastMod, strings.Join(result.Indicators, ", "))
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if found == 0 {
		fmt.Println("No 2nd brain workspaces found.")
	}
	return nil
}

// getLastModified returns the last modification time of any file in the directory (recursive)
func getLastModified(base string) string {
	var lastMod int64
	_ = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			mod := info.ModTime().Unix()
			if mod > lastMod {
				lastMod = mod
			}
		}
		return nil
	})
	if lastMod == 0 {
		return "unknown"
	}
	return fmt.Sprintf("%s", time.Unix(lastMod, 0).Format("2006-01-02 15:04"))
}
func hasDir(base, name string) bool {
	info, err := os.Stat(filepath.Join(base, name))
	return err == nil && info.IsDir()
}

func fileExists(base, name string) bool {
	info, err := os.Stat(filepath.Join(base, name))
	return err == nil && !info.IsDir()
}

func countMarkdownFiles(base string) int {
	count := 0
	_ = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			count++
		}
		// Don't go too deep
		if depth(path, base) > 2 {
			return filepath.SkipDir
		}
		return nil
	})
	return count
}

func depth(path, base string) int {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return 0
	}
	if rel == "." {
		return 0
	}
	return strings.Count(rel, string(os.PathSeparator))
}
