package commands

// ai_types.go - Parameter types for AI commands
//
// Replaces long parameter lists with structured options,
// making function signatures readable and extensible.

// AIBaseOptions holds options shared across all AI commands.
type AIBaseOptions struct {
	ModelOverride string
	PromptOverride string
	PromptExtra   string
	TimeoutSec    int
	Stream        bool
	PrepareOnly   bool
}

// PromptCreateOptions holds options for creating new prompt notes inline.
type PromptCreateOptions struct {
	Title   string
	Body    string
	Default bool
}

// AISummarizeOptions holds all options for the summarize command.
type AISummarizeOptions struct {
	AIBaseOptions
	Topic        string
	SearchBrains []string
	OutputPath   string
	BrainName    string
	Title        string
	ContextFiles []string
	PromptCreate PromptCreateOptions
	Link         bool
	NoLink       bool
}

// AIResearchOptions holds all options for the research command.
type AIResearchOptions struct {
	AIBaseOptions
	Topic            string
	OutputPath       string
	BrainName        string
	Title            string
	ContextFiles     []string
	ContextAutoBrain bool
	PromptCreate     PromptCreateOptions
	Link             bool
	NoLink           bool
}

// AIImproveOptions holds all options for the improve command.
type AIImproveOptions struct {
	AIBaseOptions
	FilePath    string
	Instruction string
	InPlace     bool
}
