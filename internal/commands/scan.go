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
		fmt.Println("No 2nd brain workspaces found.")
		return nil
	}

	// Display found brains
	fmt.Printf("✓ Found %d brain(s):\n\n", len(foundBrains))
	for _, fb := range foundBrains {
		fmt.Printf("%d) %s\n", fb.Number, fb.Description)
		fmt.Printf("   Type: %s | Files: %d | Updated: %s\n", fb.Type, fb.MdCount, fb.LastMod)
		fmt.Printf("   Path: %s\n\n", fb.Path)
	}

	// Interactive selection
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("? What would you like to do?")
	
	for {
		fmt.Println()
		fmt.Println("  1) Add one or more brains to workspace")
		fmt.Println("  2) View details of a brain")
		fmt.Println("  0) Exit")
		fmt.Printf("Choose (0/1/2): ")
		
		var choice string
		fmt.Scanln(&choice)
		choice = strings.TrimSpace(choice)
		
		if choice == "0" || strings.ToLower(choice) == "exit" || strings.ToLower(choice) == "quit" {
			fmt.Println("\n✓ Done")
			return nil
		}
		
		if choice == "2" {
			// View details
			fmt.Printf("\nEnter brain number to view details (1-%d): ", len(foundBrains))
			var num int
			fmt.Scanln(&num)
			
			if num < 1 || num > len(foundBrains) {
				fmt.Println("✗ Invalid number")
				continue
			}
			
			fb := foundBrains[num-1]
			fmt.Println("\n━━━ Brain Details ━━━")
			fmt.Printf("Name: %s\n", fb.Description)
			fmt.Printf("Type: %s\n", fb.Type)
			fmt.Printf("Path: %s\n", fb.Path)
			fmt.Printf("Markdown files: %d\n", fb.MdCount)
			fmt.Printf("Last updated: %s\n", fb.LastMod)
			fmt.Printf("Indicators: %s\n", strings.Join(fb.Indicators, ", "))
			fmt.Println()
			continue
		}
		
		if choice == "1" {
			// Add brains
			fmt.Println()
			fmt.Printf("Enter brain numbers to add (comma-separated, e.g. '1,3,5' or 'all'): ")
			var input string
			fmt.Scanln(&input)
			input = strings.TrimSpace(input)
			
			var toAdd []int
			if strings.ToLower(input) == "all" {
				for i := range foundBrains {
					toAdd = append(toAdd, i)
				}
			} else {
				parts := strings.Split(input, ",")
				for _, p := range parts {
					var num int
					fmt.Sscanf(strings.TrimSpace(p), "%d", &num)
					if num >= 1 && num <= len(foundBrains) {
						toAdd = append(toAdd, num-1)
					}
				}
			}
			
			if len(toAdd) == 0 {
				fmt.Println("✗ No valid brains selected")
				continue
			}
			
			// Ensure workspace exists
			config, err := ensureActiveWorkspace()
			if err != nil {
				return err
			}
			
			// Add each brain
			added := 0
			for _, idx := range toAdd {
				fb := foundBrains[idx]
				brainName := filepath.Base(fb.Path)
				
				// Initialize the brain
				fmt.Printf("\n→ Adding '%s'...\n", brainName)
				if err := runDirectoryInitWithName(fb.Path, brainName, "", false); err != nil {
					fmt.Printf("  ✗ Failed: %v\n", err)
					continue
				}
				added++
			}
			
			fmt.Printf("\n✓ Added %d brain(s) to workspace '%s'\n", added, config.ActiveWorkspace)
			return nil
		}
		
		fmt.Println("✗ Invalid choice")
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
