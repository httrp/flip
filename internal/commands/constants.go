package commands

// constants.go - Centralized constants to avoid magic strings
//
// All hardcoded strings that appear in multiple places should be defined here.

// Workspace constants
const (
	// DefaultWorkspaceName is the name of the default workspace
	DefaultWorkspaceName = "default"

	// DefaultBrainConfigFile is the name of the brain configuration file
	DefaultBrainConfigFile = ".flip.yaml"

	// WorkspaceConfigFileName is the name of the workspace config file
	WorkspaceConfigFileName = "workspaces.json"
)

// Brain type identifiers
const (
	BrainTypeFlip     = "flip"
	BrainTypeObsidian = "obsidian"
	BrainTypeLogseq   = "logseq"
	BrainTypeDendron  = "dendron"
	BrainTypeFoam     = "foam"
	BrainTypeUnknown  = "unknown"
)

// File extensions
const (
	ExtMarkdown = ".md"
	ExtYAML     = ".yaml"
	ExtJSON     = ".json"
)

// Directory names
const (
	DirJournal     = "journal"
	DirNotes       = "notes"
	DirMeetings    = "meetings"
	DirTasks       = "tasks"
	DirTemplates   = "templates"
	DirDefinitions = "definitions"
	DirAssets      = "assets"
)

// Menu labels (fallback if lang not loaded)
const (
	LabelBack     = "Back"
	LabelCancel   = "Cancel"
	LabelContinue = "Continue"
)
