package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/manifoldco/promptui"
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
			fmt.Printf("> Current scan path: %s\n", scanPath)
			fmt.Printf("? Enter new path to scan or press Enter to use this: ")
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
	fmt.Printf("> Scanning %s for 2nd brain workspaces...\n\n", scanPath)

	type FoundBrain struct {
		Number      int
		Description string
		Type        brain.BrainType
		Path        string
		MdCount     int
		LastMod     string
		Indicators  []string
	}

	var foundBrains []FoundBrain
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
			lastMod := getLastModified(path)
			foundBrains = append(foundBrains, FoundBrain{
				Number:      len(foundBrains) + 1,
				Description: result.Description,
				Type:        result.Type,
				Path:        path,
				MdCount:     mdCount,
				LastMod:     lastMod,
				Indicators:  result.Indicators,
			})
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if len(foundBrains) == 0 {
		fmt.Println("\n✗ No 2nd brain workspaces found.")
		return nil
	}

	// Display found brains with summary
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	fmt.Printf("✓ Found %d brain(s)\n\n", len(foundBrains))

	// Interactive loop to browse and add brains
	for {
		// Create list items for promptui
		type BrainListItem struct {
			Display string
			Index   int
		}

		var items []BrainListItem
		for i, fb := range foundBrains {
			display := fmt.Sprintf("%s (%s) - %d files", fb.Description, fb.Type, fb.MdCount)
			items = append(items, BrainListItem{Display: display, Index: i})
		}
		items = append(items, BrainListItem{Display: "◀️  Exit", Index: -1})

		// Select a brain to view details
		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "▸ {{ .Display | cyan | bold }}",
			Inactive: "  {{ .Display }}",
			Selected: "{{ .Display | green | bold }}",
		}

		prompt := promptui.Select{
			Label:     "Select a brain to view details",
			Items:     items,
			Templates: templates,
			Size:      10,
		}

		idx, _, err := prompt.Run()
		if err != nil {
			return nil
		}

		selectedIdx := items[idx].Index

		// Exit option selected
		if selectedIdx == -1 {
			fmt.Println("\n✓ Done")
			return nil
		}

		// Show brain details
		fb := foundBrains[selectedIdx]
		fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("Brain Details")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Name:         %s\n", fb.Description)
		fmt.Printf("Type:         %s\n", fb.Type)
		fmt.Printf("Path:         %s\n", fb.Path)
		fmt.Printf("Files:        %d markdown files\n", fb.MdCount)
		fmt.Printf("Last updated: %s\n", fb.LastMod)
		fmt.Printf("Indicators:   %s\n", strings.Join(fb.Indicators, ", "))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		// Ask what to do with this brain
		actionPrompt := promptui.Select{
			Label: "What would you like to do?",
			Items: []string{
				"✨ Initialize and add to workspace",
				"◀️  Back to brain list",
			},
		}

		actionIdx, _, err := actionPrompt.Run()
		if err != nil {
			return nil
		}

		if actionIdx == 0 {
			// Initialize and add brain
			config, err := ensureActiveWorkspace()
			if err != nil {
				return err
			}

			brainName := filepath.Base(fb.Path)
			fmt.Printf("\n→ Adding '%s' to workspace '%s'...\n", brainName, config.ActiveWorkspace)

			if err := runDirectoryInitWithName(fb.Path, brainName, "", false); err != nil {
				fmt.Printf("✗ Failed: %v\n\n", err)
				continue
			}

			fmt.Printf("✓ Brain '%s' added successfully!\n\n", brainName)

			// Ask if user wants to continue
			continuePrompt := promptui.Select{
				Label: "Continue browsing?",
				Items: []string{"Yes, show brain list", "No, exit"},
			}

			contIdx, _, err := continuePrompt.Run()
			if err != nil || contIdx == 1 {
				fmt.Println("\n✓ Done")
				return nil
			}
		}
		// If "Back to brain list" selected (actionIdx == 1), loop continues
	}
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
	return time.Unix(lastMod, 0).Format("2006-01-02 15:04")
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
