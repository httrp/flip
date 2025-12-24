package commands

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONOutput controls whether commands output JSON instead of human-readable text
var JSONOutput bool

// Common JSON response structures for VS Code extension integration

// CommandResult is the base response for all commands
type CommandResult struct {
	Success bool        `json:"success"`
	Command string      `json:"command"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// JournalResult contains journal creation/open result
type JournalResult struct {
	Action    string `json:"action"` // "created" or "opened"
	Path      string `json:"path"`
	Date      string `json:"date"`
	BrainName string `json:"brain_name"`
	BrainPath string `json:"brain_path"`
	BrainType string `json:"brain_type"`
}

// NoteResult contains note creation result
type NoteResult struct {
	Action    string   `json:"action"` // "created"
	Path      string   `json:"path"`
	Title     string   `json:"title"`
	Tags      []string `json:"tags,omitempty"`
	BrainName string   `json:"brain_name"`
	BrainPath string   `json:"brain_path"`
	BrainType string   `json:"brain_type"`
}

// QuicknoteResult contains quicknote creation result
type QuicknoteResult struct {
	Action    string `json:"action"` // "created"
	Path      string `json:"path"`
	BrainName string `json:"brain_name"`
	BrainPath string `json:"brain_path"`
}

// StatusResult contains status information
type StatusResult struct {
	Workspace    WorkspaceInfo `json:"workspace"`
	ActiveBrain  *BrainInfo    `json:"active_brain,omitempty"`
	Brains       []BrainInfo   `json:"brains"`
	FlipVersion  string        `json:"flip_version"`
	IsVSCode     bool          `json:"is_vscode"`
}

// WorkspaceInfo contains workspace details
type WorkspaceInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Active bool   `json:"active"`
}

// BrainInfo contains brain details
type BrainInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Type      string `json:"type"`
	Active    bool   `json:"active"`
	GitBranch string `json:"git_branch,omitempty"`
	GitStatus string `json:"git_status,omitempty"`
}

// SearchResult contains search results
type SearchResult struct {
	Query   string       `json:"query"`
	Count   int          `json:"count"`
	Results []SearchHit  `json:"results"`
}

// SearchHit is a single search result
type SearchHit struct {
	Path      string `json:"path"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet,omitempty"`
	LineNum   int    `json:"line_num,omitempty"`
	BrainName string `json:"brain_name"`
}

// RecentResult contains recent files
type RecentResult struct {
	Count int          `json:"count"`
	Files []RecentFile `json:"files"`
}

// RecentFile is a recently modified file
type RecentFile struct {
	Path      string `json:"path"`
	Title     string `json:"title"`
	Modified  string `json:"modified"`
	BrainName string `json:"brain_name"`
}

// VSCodeInfo contains information for VS Code extension
type VSCodeInfo struct {
	FlipVersion     string        `json:"flip_version"`
	ActiveWorkspace string        `json:"active_workspace"`
	ActiveBrain     string        `json:"active_brain"`
	Brains          []BrainInfo   `json:"brains"`
	Commands        []string      `json:"commands"`
}

// OutputJSON outputs a CommandResult as JSON to stdout
func OutputJSON(result CommandResult) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(result)
}

// OutputJSONSuccess outputs a successful result
func OutputJSONSuccess(command string, data interface{}) {
	OutputJSON(CommandResult{
		Success: true,
		Command: command,
		Data:    data,
	})
}

// OutputJSONError outputs an error result
func OutputJSONError(command string, err error) {
	OutputJSON(CommandResult{
		Success: false,
		Command: command,
		Error:   err.Error(),
	})
}

// PrintOrJSON prints human-readable output or JSON based on JSONOutput flag
// If JSONOutput is true, it will defer output until OutputJSONSuccess/Error is called
// If JSONOutput is false, it prints immediately
func PrintOrJSON(format string, args ...interface{}) {
	if !JSONOutput {
		fmt.Printf(format, args...)
	}
}

// PrintlnOrJSON is like PrintOrJSON but with newline
func PrintlnOrJSON(args ...interface{}) {
	if !JSONOutput {
		fmt.Println(args...)
	}
}
