package exercises

import (
	"time"
)

// Exercise represents a repeatable practice/training activity
type Exercise struct {
	// Core identification
	ID      string `yaml:"id" json:"id"`
	Type    string `yaml:"type" json:"type"` // Always "exercise" for type detection
	Name    string `yaml:"name" json:"name"`
	Context string `yaml:"context,omitempty" json:"context,omitempty"` // User-defined: Sport, Music, Language, Basketball, etc.

	// Metadata (compatible with Obsidian/Logseq/Dendron)
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	Goal        string    `yaml:"goal,omitempty" json:"goal,omitempty"`
	Tags        []string  `yaml:"tags,omitempty" json:"tags,omitempty"`
	Created     time.Time `yaml:"created" json:"created"`
	Status      string    `yaml:"status,omitempty" json:"status,omitempty"` // active, inactive, paused

	// Variants - different variations of this exercise
	Variants []ExerciseVariant `yaml:"variants,omitempty" json:"variants,omitempty"`

	// Optional guidance
	Duration string `yaml:"duration,omitempty" json:"duration,omitempty"` // Expected duration "20-30 min"

	// Materials & Resources
	Materials []Material `yaml:"materials,omitempty" json:"materials,omitempty"`

	// Relationships (wikilinks compatible)
	Related []string `yaml:"related,omitempty" json:"related,omitempty"` // IDs or [[wikilinks]]

	// Tracking stats (updated by flip)
	LastSession  *time.Time `yaml:"last_session,omitempty" json:"last_session,omitempty"`
	SessionCount int        `yaml:"session_count,omitempty" json:"session_count,omitempty"`

	// File context (internal, not in YAML)
	FilePath  string `yaml:"-" json:"-"` // Absolute path to .md file
	BrainPath string `yaml:"-" json:"-"` // Root path of brain
	BrainType string `yaml:"-" json:"-"` // obsidian, logseq, dendron, etc.
}

// Material represents a learning resource
type Material struct {
	Type  string `yaml:"type" json:"type"`   // video, pdf, link, note
	Title string `yaml:"title" json:"title"` // Display name
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
	Path  string `yaml:"path,omitempty" json:"path,omitempty"` // Local path in brain
}

// ExerciseVariant represents a specific variation of an exercise
type ExerciseVariant struct {
	Name               string            `yaml:"name,omitempty" json:"name,omitempty"`                               // Optional: "Balance & Control", "Dynamic Waves", etc.
	Description        string            `yaml:"description,omitempty" json:"description,omitempty"`                 // What makes this variant unique
	TrackingProperties map[string]string `yaml:"tracking_properties,omitempty" json:"tracking_properties,omitempty"` // e.g. "tempo": "bpm", "focus": "text"
}

// ExerciseSession represents a single practice session (stored in journal)
type ExerciseSession struct {
	// Core identification
	ID         string    `yaml:"id,omitempty" json:"id,omitempty"` // UUID (optional for simple sessions)
	ExerciseID string    `yaml:"exercise_id" json:"exercise_id"`   // Reference to exercise
	Date       time.Time `yaml:"date" json:"date"`

	// Session data
	Duration    int    `yaml:"duration,omitempty" json:"duration,omitempty"`       // Minutes
	VariantName string `yaml:"variant_name,omitempty" json:"variant_name,omitempty"` // Which variant was practiced
	Notes       string `yaml:"notes,omitempty" json:"notes,omitempty"`

	// Flexible tracking data - any key-value pairs user wants to track
	Properties map[string]interface{} `yaml:"properties,omitempty" json:"properties,omitempty"` // e.g. {"speed": 120, "level": 3, "reps": 50}

	// Brain compatibility fields
	Tags     []string `yaml:"tags,omitempty" json:"tags,omitempty"`         // For Obsidian Dataview
	Exercise string   `yaml:"exercise,omitempty" json:"exercise,omitempty"` // Wikilink to exercise: "[[Exercise Name]]"

	// File context (internal) - journal entry info
	FilePath    string `yaml:"-" json:"-"` // Path to journal file
	BrainPath   string `yaml:"-" json:"-"`
	BlockStart  int    `yaml:"-" json:"-"` // Line number where this session starts in journal
	BlockEnd    int    `yaml:"-" json:"-"` // Line number where this session ends in journal
	JournalDate string `yaml:"-" json:"-"` // Journal date (YYYY-MM-DD format)
}

// ExercisePlan groups multiple exercises into a structured training/learning plan
type ExercisePlan struct {
	// Core identification
	ID   string `yaml:"id" json:"id"`
	Name string `yaml:"name" json:"name"`

	// Metadata
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	Tags        []string  `yaml:"tags,omitempty" json:"tags,omitempty"`
	Created     time.Time `yaml:"created" json:"created"`
	Status      string    `yaml:"status,omitempty" json:"status,omitempty"` // active, completed, paused

	// Schedule
	Schedule  []string   `yaml:"schedule,omitempty" json:"schedule,omitempty"` // ["monday", "wednesday", "friday"] or ["daily"]
	StartDate *time.Time `yaml:"start_date,omitempty" json:"start_date,omitempty"`
	EndDate   *time.Time `yaml:"end_date,omitempty" json:"end_date,omitempty"`

	// Plan items (exercises in this plan)
	Items []PlanItem `yaml:"items" json:"items"`

	// Stats
	SessionCount int `yaml:"session_count,omitempty" json:"session_count,omitempty"`

	// File context
	FilePath  string `yaml:"-" json:"-"`
	BrainPath string `yaml:"-" json:"-"`
}

// PlanItem represents one exercise in a plan
type PlanItem struct {
	ExerciseID string `yaml:"exercise_id" json:"exercise_id"`         // ID of exercise
	Exercise   string `yaml:"exercise,omitempty" json:"exercise,omitempty"` // Wikilink: "[[Exercise Name]]"
	Order      int    `yaml:"order" json:"order"`                     // Position in plan
	Target     string `yaml:"target,omitempty" json:"target,omitempty"` // Expected target "3x12 reps", "30 min"
}

// PlanSession represents a workout/training session for a plan
type PlanSession struct {
	// Core identification
	ID     string    `yaml:"id,omitempty" json:"id,omitempty"`
	PlanID string    `yaml:"plan_id" json:"plan_id"` // Reference to plan
	Date   time.Time `yaml:"date" json:"date"`

	// Session items (what was actually done)
	Items []PlanSessionItem `yaml:"items" json:"items"`

	// Overall notes
	Notes string `yaml:"notes,omitempty" json:"notes,omitempty"`

	// Brain compatibility
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Plan string   `yaml:"plan,omitempty" json:"plan,omitempty"` // Wikilink to plan

	// File context
	FilePath  string `yaml:"-" json:"-"`
	BrainPath string `yaml:"-" json:"-"`
}

// PlanSessionItem represents what was done for one exercise in a plan session
type PlanSessionItem struct {
	ExerciseID string `yaml:"exercise_id" json:"exercise_id"`
	Exercise   string `yaml:"exercise,omitempty" json:"exercise,omitempty"` // Wikilink
	Completed  bool   `yaml:"completed" json:"completed"`
	Actual     string `yaml:"actual,omitempty" json:"actual,omitempty"` // What was actually done
	Notes      string `yaml:"notes,omitempty" json:"notes,omitempty"`
}
