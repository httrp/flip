package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/httrp/flip/internal/brain"
	"github.com/httrp/flip/internal/ui"
	"github.com/spf13/cobra"
)

func NewScanCommand() *cobra.Command {
	var minBrainFiles int
	var minContentFiles int
	var maxDepth int
	var showPotential bool

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
			return runScanWithOptions(scanPath, ScanOptions{
				MinBrainFiles:   minBrainFiles,
				MinContentFiles: minContentFiles,
				MaxDepth:        maxDepth,
				ShowPotential:   showPotential,
			})
		},
	}

	cmd.Flags().IntVar(&minBrainFiles, "min-brain-files", 10, "Minimum markdown files for recognized brain types (Obsidian, Logseq, Dendron)")
	cmd.Flags().IntVar(&minContentFiles, "min-content-files", 50, "Minimum content files for potential brains without markers")
	cmd.Flags().IntVar(&maxDepth, "max-depth", 4, "Maximum directory depth to scan")
	cmd.Flags().BoolVar(&showPotential, "show-potential", true, "Show potential brains (folders with many content files)")

	return cmd
}

// ScanOptions contains configurable thresholds for scanning
type ScanOptions struct {
	MinBrainFiles   int  // Minimum files for recognized brain types
	MinContentFiles int  // Minimum files for potential brains
	MaxDepth        int  // Maximum scan depth
	ShowPotential   bool // Whether to show potential brains
}

func runScan(scanPath string) error {
	return runScanWithOptions(scanPath, ScanOptions{
		MinBrainFiles:   10,
		MinContentFiles: 50,
		MaxDepth:        4,
		ShowPotential:   true,
	})
}

func runScanWithOptions(scanPath string, options ScanOptions) error {
	fmt.Printf("> Scanning %s for 2nd brain workspaces...\n", scanPath)
	if !options.ShowPotential {
		fmt.Printf("> Showing only recognized brain types (Obsidian, Logseq, Dendron, Flip)\n")
	}

	// Start progress animation
	stopProgress := make(chan bool)
	go showProgress(stopProgress)

	type FoundBrain struct {
		Number        int
		Description   string
		Type          brain.BrainType
		Path          string
		DisplayPath   string // Shortened path for display
		MdCount       int
		LastMod       string
		Indicators    []string
		IsInWorkspace bool   // Whether this brain is already in active workspace
		WorkspaceName string // Name of workspace if already added
		IsPotential   bool   // Whether this is a content-rich folder without markers
		// Sync & Version Control
		SyncService   string // Dropbox, Google Drive, iCloud, etc.
		IsGitRepo     bool   // Whether this is a git repository
		GitRemoteURL  string // Git remote URL (if exists)
		GitRemoteName string // Git remote name (usually "origin")
		GitBranch     string // Current git branch
	}

	var foundBrains []FoundBrain

	// Comprehensive exclude list for scanning
	excludeDirs := []string{
		// Development environments
		"node_modules", ".npm", ".yarn", ".pnpm",
		"vendor", "venv", "env", ".venv", ".env",
		"go", "pkg", "bin", "target", "build", "dist",
		".gradle", ".maven", ".cargo", ".rustup",
		"Cargo.lock", "package-lock.json",

		// Version control & IDEs
		".git", ".svn", ".hg",
		".vscode", ".idea", ".eclipse", ".vs",

		// System directories (macOS)
		"Library", "Applications", "System", "Volumes",
		"private", "opt", "usr", "var", "tmp",
		".Trash", ".DocumentRevisions-V100", ".Spotlight-V100",
		".TemporaryItems", ".fseventsd",
		"Downloads", "Music", "Movies", "Pictures", // Usually not brain locations

		// System directories (Windows)
		"AppData", "Program Files", "Program Files (x86)",
		"Windows", "ProgramData",

		// Package managers & caches
		".cache", ".local", ".config",
		"__pycache__", ".pytest_cache",
		".next", ".nuxt", ".output",
		"coverage", ".coverage",

		// Common hidden/config directories
		".oh-my-zsh", ".zsh", ".vim", ".emacs.d",
		".ssh", ".gnupg", ".docker",

		// Cloud sync internal folders (don't scan inside these)
		".dropbox.cache", ".dropbox",
		".icloud", "iCloud~",

		// Other
		"Parallels", "VirtualBox VMs", "Docker",
		"snap", "flatpak",
	}

	err := filepath.WalkDir(scanPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if !d.IsDir() {
			return nil
		}

		// Skip hidden directories at root level (anything starting with .)
		if strings.HasPrefix(d.Name(), ".") && path != scanPath {
			return filepath.SkipDir
		}

		// Limit scan depth for performance
		if depth(path, scanPath) > options.MaxDepth {
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

		// Check if this is a subdirectory of a git repo
		// If so, check if the git root is a better match
		gitRoot := getGitRoot(path)
		if gitRoot != "" && gitRoot != path {
			// Check if git root has brain markers or significant content
			gitRootResult, _ := detector.DetectBrainType(gitRoot)
			gitRootMdCount := countMarkdownFiles(gitRoot)
			gitRootContentCount := countContentFiles(gitRoot)

			// Special case: If git root has journals+pages structure (Logseq), prefer it
			gitHasLogseqStructure := hasDir(gitRoot, "journals") && hasDir(gitRoot, "pages")

			// If git root has brain markers OR significant content OR Logseq structure, skip this subdirectory
			// (we'll find the git root later in the scan)
			hasGitRootMarkers := gitRootResult.Type != brain.BrainTypeUnknown
			hasGitRootContent := gitRootContentCount >= 30 || gitRootMdCount >= 10

			if hasGitRootMarkers || hasGitRootContent || gitHasLogseqStructure {
				// Skip this subdirectory, we'll pick up the git root instead
				return filepath.SkipDir
			}
		}

		// Check for content files (markdown, txt, org, etc.)
		contentCount := countContentFiles(path)
		mdCount := countMarkdownFiles(path)

		// Only show if clear marker and plausible number of files, OR lots of content files
		isValid := false
		isPotential := false

		switch result.Type {
		case brain.BrainTypeObsidian:
			isValid = hasDir(path, ".obsidian") && mdCount > options.MinBrainFiles
		case brain.BrainTypeLogseq:
			// Logseq can be detected either by .logseq marker OR by journals+pages structure
			hasMarker := hasDir(path, ".logseq")
			hasStructure := hasDir(path, "journals") && hasDir(path, "pages")

			// If no marker, verify it's actually a Logseq graph by checking journal content
			if !hasMarker && hasStructure {
				journalCount := countMarkdownFiles(filepath.Join(path, "journals"))
				pagesCount := countMarkdownFiles(filepath.Join(path, "pages"))
				// Only recognize as Logseq if journals has significant content (typical Logseq usage)
				hasStructure = journalCount >= 5 || pagesCount >= 5
			}

			isValid = (hasMarker || hasStructure) && mdCount > options.MinBrainFiles
			if isValid && !hasMarker && hasStructure {
				result.Description = "Logseq Graph (no marker)"
				result.Indicators = []string{"journals/ and pages/ directories"}
			}
		case brain.BrainTypeDendron:
			isValid = fileExists(path, "dendron.yml") && mdCount > options.MinBrainFiles
		case brain.BrainTypeFoam:
			isValid = hasDir(path, ".foam") && mdCount > options.MinBrainFiles
		case brain.BrainTypeFlip:
			isValid = (fileExists(path, ".flip-brain.yaml") || fileExists(path, ".flip.yaml")) && mdCount > 3
		case brain.BrainTypeUnknown:
			// Check if this looks like a Logseq structure without .logseq marker
			hasJournals := hasDir(path, "journals")
			hasPages := hasDir(path, "pages")
			if hasJournals && hasPages {
				// Verify it's actually Logseq by checking for content in journals/pages
				journalCount := countMarkdownFiles(filepath.Join(path, "journals"))
				pagesCount := countMarkdownFiles(filepath.Join(path, "pages"))

				// Only recognize as Logseq if there's significant content in journals or pages
				if (journalCount >= 5 || pagesCount >= 5) && contentCount > options.MinBrainFiles {
					// This looks like Logseq without the marker
					isValid = true
					result.Type = brain.BrainTypeLogseq
					result.Description = "Logseq Graph (no marker)"
					result.Indicators = []string{"journals/ and pages/ directories"}
					break
				}
			}

			// Otherwise check for content-rich folders (if enabled)
			if options.ShowPotential && contentCount >= options.MinContentFiles {
				isValid = true
				isPotential = true
				result.Type = brain.BrainTypeFlip // Default type for potential brains
				result.Description = "Content Folder"
				result.Indicators = []string{fmt.Sprintf("%d content files", contentCount)}
			}
		}

		if isValid {
			// Don't add system-critical paths as brains (but continue scanning their children)
			if IsSystemCriticalPath(path) {
				return nil
			}

			lastMod := getLastModified(path)
			isInWs, wsName := checkIfInWorkspace(path)
			displayPath := shortenPath(path)

			// Detect sync service and git info
			syncService := detectSyncService(path)
			gitInfo := detectGitInfo(path)

			foundBrains = append(foundBrains, FoundBrain{
				Number:        len(foundBrains) + 1,
				Description:   result.Description,
				Type:          result.Type,
				Path:          path,
				DisplayPath:   displayPath,
				MdCount:       mdCount,
				LastMod:       lastMod,
				Indicators:    result.Indicators,
				IsInWorkspace: isInWs,
				WorkspaceName: wsName,
				IsPotential:   isPotential,
				SyncService:   syncService,
				IsGitRepo:     gitInfo.IsRepo,
				GitRemoteURL:  gitInfo.RemoteURL,
				GitRemoteName: gitInfo.RemoteName,
				GitBranch:     gitInfo.Branch,
			})
		}
		return nil
	})

	// Stop progress animation
	stopProgress <- true
	fmt.Print("\r\033[K") // Clear the progress line

	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if len(foundBrains) == 0 {
		fmt.Println("\n✗ No 2nd brain workspaces found.")
		return nil
	}

	// Sort results: recognized brain types first (by confidence), then potential folders
	// Within each category, sort by file count (descending)
	sort.Slice(foundBrains, func(i, j int) bool {
		iPotential := foundBrains[i].IsPotential
		jPotential := foundBrains[j].IsPotential

		// Recognized types come before potential folders
		if iPotential != jPotential {
			return !iPotential // false (recognized) comes before true (potential)
		}

		// Within same category, sort by markdown file count (more files = higher confidence)
		return foundBrains[i].MdCount > foundBrains[j].MdCount
	})

	// Renumber after sorting
	for idx := range foundBrains {
		foundBrains[idx].Number = idx + 1
	}

	// Display found brains with summary
	fmt.Println()
	displayStatusHeader()
	fmt.Println()
	fmt.Printf("✓ Found %d brain(s)\n\n", len(foundBrains))

	// Interactive loop to browse and add brains
	for {
		// Create list items for selection
		type BrainListItem struct {
			Display string
			Index   int
		}

		var items []BrainListItem
		for i, fb := range foundBrains {
			// Build display string with path and workspace status
			statusPrefix := ""
			if fb.IsInWorkspace {
				statusPrefix = "✓ "
			}
			if fb.IsPotential {
				statusPrefix = "? " + statusPrefix
			}

			wsInfo := ""
			if fb.IsInWorkspace {
				wsInfo = fmt.Sprintf(" [%s]", fb.WorkspaceName)
			}

			// Add sync and git badges
			badges := ""
			if fb.SyncService != "" {
				badges += " ☁️"
			}
			if fb.IsGitRepo {
				badges += " 📦"
			}

			display := fmt.Sprintf("%s%s (%s) - %d files - %s%s%s",
				statusPrefix, fb.Description, fb.Type, fb.MdCount, fb.DisplayPath, wsInfo, badges)
			items = append(items, BrainListItem{Display: display, Index: i})
		}
		items = append(items, BrainListItem{Display: "◀️  Exit", Index: -1})

		// Select a brain to view details
		selectItems := make([]ui.SelectItem, len(items))
		for i, item := range items {
			selectItems[i] = ui.SelectItem{Label: item.Display, Value: fmt.Sprintf("%d", item.Index)}
		}

		idx, _, err := ui.RunSelect("Select a brain to view details", selectItems, 10)
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
		fmt.Printf("Type:         %s", fb.Type)
		if fb.IsPotential {
			fmt.Printf(" (potential - no specific brain marker detected)")
		}
		fmt.Println()
		fmt.Printf("Path:         %s\n", fb.Path)
		fmt.Printf("Files:        %d markdown files\n", fb.MdCount)
		fmt.Printf("Last updated: %s\n", fb.LastMod)
		fmt.Printf("Indicators:   %s\n", strings.Join(fb.Indicators, ", "))
		if fb.IsInWorkspace {
			fmt.Printf("Status:       ✓ Already in workspace '%s'\n", fb.WorkspaceName)
		} else {
			fmt.Printf("Status:       Not yet added to workspace\n")
		}

		// Show sync service info
		if fb.SyncService != "" {
			fmt.Printf("Sync:         ☁️  %s\n", fb.SyncService)
		} else {
			fmt.Printf("Sync:         Local only (no cloud sync detected)\n")
		}

		// Show git info
		if fb.IsGitRepo {
			fmt.Printf("Git:          📦 Repository")
			if fb.GitBranch != "" {
				fmt.Printf(" (branch: %s)", fb.GitBranch)
			}
			fmt.Println()
			if fb.GitRemoteURL != "" {
				fmt.Printf("Git Remote:   %s (%s)\n", fb.GitRemoteURL, fb.GitRemoteName)
			} else {
				fmt.Printf("Git Remote:   No remote configured (local only)\n")
			}
		} else {
			fmt.Printf("Git:          Not a git repository\n")
		}

		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()

		// Ask what to do with this brain
		actionItems := []ui.SelectItem{}
		if !fb.IsInWorkspace {
			actionItems = append(actionItems, ui.SelectItem{Label: "✨ Initialize and add to workspace", Value: "init"})
		}
		actionItems = append(actionItems, ui.SelectItem{Label: "◀️  Back to brain list", Value: "back"})

		_, actionChoice, err := ui.RunSelect("What would you like to do?", actionItems, 0)
		if err != nil {
			return nil
		}

		// If brain is already in workspace, only "Back" is available
		if fb.IsInWorkspace || actionChoice == "back" {
			continue // Back to brain list
		}

		if actionChoice == "init" {
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
			continueBrowsing, err := ui.RunConfirm("Continue browsing?", true)
			if err != nil || !continueBrowsing {
				fmt.Println("\n✓ Done")
				return nil
			}
		}
		// If "Back to brain list" selected, loop continues
	}
}

// getLastModified returns the last modification time of any file in the directory (recursive)
func getLastModified(base string) string {
	var lastMod int64
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return nil
			}
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

// IsSystemCriticalPath checks if a path is a system-critical directory that should never be a brain
func IsSystemCriticalPath(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Get home directory for comparison
	homeDir, _ := os.UserHomeDir()
	usersDir := filepath.Dir(homeDir) // Usually /Users on macOS, /home on Linux

	// List of critical paths that should never be brains
	criticalPaths := []string{
		"/",                 // Root filesystem
		"/System",           // macOS system
		"/Library",          // macOS system library
		"/Applications",     // macOS applications
		"/usr",              // Unix system
		"/var",              // Unix variables
		"/tmp",              // Temporary
		"/opt",              // Optional software
		"/bin",              // Binaries
		"/sbin",             // System binaries
		"/etc",              // System configuration
		"/dev",              // Devices
		"/proc",             // Process info (Linux)
		"/sys",              // System info (Linux)
		homeDir,             // User home directory
		usersDir,            // /Users or /home directory
		"C:\\",              // Windows root
		"C:\\Windows",       // Windows system
		"C:\\Program Files", // Windows programs
		"C:\\Program Files (x86)",
	}

	// Clean paths for comparison
	absPath = filepath.Clean(absPath)

	// Check exact matches
	for _, critical := range criticalPaths {
		if critical != "" && absPath == filepath.Clean(critical) {
			return true
		}
	}

	return false
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
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
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

// shortenPath returns a shortened version of a path for display
func shortenPath(fullPath string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(fullPath, home) {
		shortened := "~" + strings.TrimPrefix(fullPath, home)
		// Further shorten if still too long (> 50 chars)
		if len(shortened) > 50 {
			parts := strings.Split(shortened, string(os.PathSeparator))
			if len(parts) > 3 {
				return "~/" + parts[1] + "/.../" + strings.Join(parts[len(parts)-2:], "/")
			}
		}
		return shortened
	}

	// Show last 3 path components if path is very long
	if len(fullPath) > 60 {
		parts := strings.Split(fullPath, string(os.PathSeparator))
		if len(parts) > 4 {
			return ".../" + strings.Join(parts[len(parts)-3:], "/")
		}
	}

	return fullPath
} // checkIfInWorkspace checks if a brain path is already in the active workspace
func checkIfInWorkspace(brainPath string) (bool, string) {
	config, err := loadWorkspaceConfig()
	if err != nil || config.ActiveWorkspace == "" {
		return false, ""
	}

	// Find active workspace
	for _, ws := range config.Workspaces {
		if ws.Name == config.ActiveWorkspace {
			// Check if path matches any brain in this workspace
			for _, b := range ws.Brains {
				// Normalize paths for comparison
				absPath, _ := filepath.Abs(b.Path)
				absBrainPath, _ := filepath.Abs(brainPath)
				if absPath == absBrainPath {
					return true, ws.Name
				}
			}
			return false, ""
		}
	}

	return false, ""
}

// countContentFiles counts markdown, text, org and other content files
func countContentFiles(base string) int {
	count := 0
	extensions := []string{".md", ".txt", ".org", ".markdown", ".rst"}

	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			lower := strings.ToLower(d.Name())
			for _, ext := range extensions {
				if strings.HasSuffix(lower, ext) {
					count++
					break
				}
			}
		}
		// Don't go too deep
		if depth(path, base) > 2 {
			return filepath.SkipDir
		}
		return nil
	})
	return count
}

// detectSyncService checks if a path is inside a cloud sync folder
func detectSyncService(path string) string {
	// Common sync service patterns (macOS paths)
	syncPatterns := map[string][]string{
		"Dropbox":      {"/Dropbox/", "Dropbox (Personal)", "Dropbox (Business)"},
		"Google Drive": {"/Google Drive/", "/GoogleDrive/", "Google Drive File Stream"},
		"iCloud Drive": {"/Library/Mobile Documents/com~apple~CloudDocs/", "/iCloud Drive/"},
		"OneDrive":     {"/OneDrive/", "/OneDrive - /"},
		"SharePoint":   {"/SharePoint/"},
		"Box":          {"/Box/", "/Box Sync/"},
		"Nextcloud":    {"/Nextcloud/"},
		"ownCloud":     {"/ownCloud/"},
		"Syncthing":    {"/Syncthing/"},
		"Resilio Sync": {"/Resilio Sync/", "/.sync/"},
		"pCloud":       {"/pCloudDrive/", "/pCloud/"},
		"MEGA":         {"/MEGA/", "/MEGAsync/"},
		"SugarSync":    {"/SugarSync/"},
		"SpiderOak":    {"/SpiderOak/"},
	}

	// Normalize path for comparison
	normalizedPath := filepath.Clean(path)

	for service, patterns := range syncPatterns {
		for _, pattern := range patterns {
			if strings.Contains(normalizedPath, pattern) {
				return service
			}
		}
	}

	return ""
}

// getGitRoot returns the git repository root for a given path, or empty string if not in a git repo
func getGitRoot(path string) string {
	// Walk up the directory tree looking for .git
	current := path
	for {
		gitDir := filepath.Join(current, ".git")
		if stat, err := os.Stat(gitDir); err == nil && stat.IsDir() {
			return current
		}

		// Move to parent directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}
		current = parent
	}
	return ""
}

// GitInfo holds git repository information
type GitInfo struct {
	IsRepo     bool
	RemoteName string
	RemoteURL  string
	Branch     string
}

// detectGitInfo checks if a path is a git repository and extracts remote info
func detectGitInfo(path string) GitInfo {
	info := GitInfo{}

	// Check if .git directory exists
	gitDir := filepath.Join(path, ".git")
	if stat, err := os.Stat(gitDir); err != nil || !stat.IsDir() {
		return info
	}

	info.IsRepo = true

	// Read current branch
	headFile := filepath.Join(gitDir, "HEAD")
	if headData, err := os.ReadFile(headFile); err == nil {
		headStr := strings.TrimSpace(string(headData))
		if strings.HasPrefix(headStr, "ref: refs/heads/") {
			info.Branch = strings.TrimPrefix(headStr, "ref: refs/heads/")
		}
	}

	// Read remote info (look for origin first, then any remote)
	configFile := filepath.Join(gitDir, "config")
	if configData, err := os.ReadFile(configFile); err == nil {
		lines := strings.Split(string(configData), "\n")
		var currentRemote string

		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Check for remote section
			if strings.HasPrefix(line, "[remote \"") && strings.HasSuffix(line, "\"]") {
				currentRemote = strings.TrimSuffix(strings.TrimPrefix(line, "[remote \""), "\"]")
			}

			// Check for URL in current remote
			if currentRemote != "" && strings.HasPrefix(line, "url = ") {
				url := strings.TrimPrefix(line, "url = ")

				// Prefer origin, but keep first remote found
				if currentRemote == "origin" || info.RemoteName == "" {
					info.RemoteName = currentRemote
					info.RemoteURL = url
				}

				if currentRemote == "origin" {
					break // Found origin, no need to continue
				}
			}
		}
	}

	return info
}

// showProgress displays an animated progress indicator while scanning
func showProgress(stop chan bool) {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	messages := []string{
		"Searching for brains",
		"Analyzing directories",
		"Detecting brain types",
		"Checking markers",
		"Scanning workspaces",
	}

	i := 0
	msgIdx := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			msg := messages[msgIdx%len(messages)]
			spinner := spinners[i%len(spinners)]
			fmt.Printf("\r%s %s...", spinner, msg)
			i++
			if i%10 == 0 {
				msgIdx++
			}
		}
	}
}
