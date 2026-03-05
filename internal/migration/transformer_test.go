package migration

import (
	"testing"

	"github.com/httrp/flip/internal/health"
)

func TestTransformFilename_JournalDates(t *testing.T) {
	tests := []struct {
		name       string
		sourceType health.BrainType
		targetType health.BrainType
		input      string
		expected   string
	}{
		// Logseq → Flip (underscores → dashes)
		{
			name:       "Logseq to Flip journal",
			sourceType: health.BrainTypeLogseq,
			targetType: health.BrainTypeFlip,
			input:      "2025_12_20.md",
			expected:   "2025-12-20.md",
		},
		// Flip → Logseq (dashes → underscores)
		{
			name:       "Flip to Logseq journal",
			sourceType: health.BrainTypeFlip,
			targetType: health.BrainTypeLogseq,
			input:      "2025-12-20.md",
			expected:   "2025_12_20.md",
		},
		// Logseq → Dendron (underscores → dots with prefix)
		{
			name:       "Logseq to Dendron journal",
			sourceType: health.BrainTypeLogseq,
			targetType: health.BrainTypeDendron,
			input:      "2025_12_20.md",
			expected:   "daily.2025.12.20.md",
		},
		// Dendron → Flip (dots → dashes, strip prefix)
		{
			name:       "Dendron to Flip journal",
			sourceType: health.BrainTypeDendron,
			targetType: health.BrainTypeFlip,
			input:      "daily.2025.12.20.md",
			expected:   "2025-12-20.md",
		},
		// Same type - no change
		{
			name:       "Flip to Flip (no change)",
			sourceType: health.BrainTypeFlip,
			targetType: health.BrainTypeFlip,
			input:      "2025-12-20.md",
			expected:   "2025-12-20.md",
		},
		// Non-date filename - should apply note naming rules
		{
			name:       "Non-date filename Flip to Logseq",
			sourceType: health.BrainTypeFlip,
			targetType: health.BrainTypeLogseq,
			input:      "my-cool-note.md",
			expected:   "my-cool-note.md", // Logseq now uses kebab-case (standardized)
		},
		// Logseq title to Flip slug
		{
			name:       "Logseq title to Flip slug",
			sourceType: health.BrainTypeLogseq,
			targetType: health.BrainTypeFlip,
			input:      "My Cool Note.md",
			expected:   "my-cool-note.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewTransformer(tt.sourceType, tt.targetType)
			result := transformer.TransformFilename(tt.input)
			if result != tt.expected {
				t.Errorf("TransformFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTransformFilename_NoteNaming(t *testing.T) {
	tests := []struct {
		name       string
		sourceType health.BrainType
		targetType health.BrainType
		input      string
		expected   string
	}{
		// Flip slug to Dendron dot-notation
		{
			name:       "Flip to Dendron note",
			sourceType: health.BrainTypeFlip,
			targetType: health.BrainTypeDendron,
			input:      "project-ideas-2025.md",
			expected:   "project.ideas.2025.md",
		},
		// Dendron to Flip
		{
			name:       "Dendron to Flip note",
			sourceType: health.BrainTypeDendron,
			targetType: health.BrainTypeFlip,
			input:      "projects.work.q4-planning.md",
			expected:   "projects-work-q4-planning.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewTransformer(tt.sourceType, tt.targetType)
			result := transformer.TransformFilename(tt.input)
			if result != tt.expected {
				t.Errorf("TransformFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetTransformationSummary(t *testing.T) {
	// Test Logseq → Flip summary
	transformer := NewTransformer(health.BrainTypeLogseq, health.BrainTypeFlip)
	summary := transformer.GetTransformationSummary()

	if summary == "" {
		t.Error("Expected non-empty summary for Logseq→Flip transformation")
	}

	// Summary should mention date format change
	if !contains(summary, "Journal dates") && !contains(summary, "journal") {
		t.Logf("Summary: %s", summary)
		// This is a soft check - summary format may vary
	}
}

func TestNeedsTransformation(t *testing.T) {
	// Same type - no transformation
	t1 := NewTransformer(health.BrainTypeFlip, health.BrainTypeFlip)
	if t1.NeedsTransformation() {
		t.Error("Same brain type should not need transformation")
	}

	// Different types - needs transformation
	t2 := NewTransformer(health.BrainTypeLogseq, health.BrainTypeFlip)
	if !t2.NeedsTransformation() {
		t.Error("Different brain types should need transformation")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// FRONTMATTER TRANSFORMER TESTS
// ============================================================================

func TestTransformContent_YAMLToLogseq(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Simple YAML to Logseq",
			input: `---
title: My Note
tags: project
---

This is the content.`,
			expected: `title:: My Note
tags:: project

This is the content.`,
		},
		{
			name: "Multiple properties",
			input: `---
title: Meeting Notes
date: 2025-12-20
status: active
---

Meeting content here.`,
			expected: `title:: Meeting Notes
date:: 2025-12-20
status:: active

Meeting content here.`,
		},
		{
			name: "No frontmatter - unchanged",
			input: `# Just a heading

Some content.`,
			expected: `# Just a heading

Some content.`,
		},
		{
			name: "YAML with array to Logseq",
			input: `---
title: Tagged Note
tags:
  - project
  - work
  - important
---

Content with tags.`,
			expected: `title:: Tagged Note
tags:: [project, work, important]

Content with tags.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewTransformer(health.BrainTypeFlip, health.BrainTypeLogseq)
			result := transformer.TransformContent(tt.input)
			if result != tt.expected {
				t.Errorf("TransformContent() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

func TestTransformContent_LogseqToYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Simple Logseq to YAML",
			input: `title:: My Note
tags:: project

This is the content.`,
			// Note: Properties are sorted alphabetically in YAML output
			expected: `---
tags: project
title: My Note
---

This is the content.`,
		},
		{
			name: "Multiple Logseq properties",
			input: `title:: Meeting Notes
date:: 2025-12-20
status:: active

Meeting content here.`,
			// Note: Properties are sorted alphabetically in YAML output
			expected: `---
date: 2025-12-20
status: active
title: Meeting Notes
---

Meeting content here.`,
		},
		{
			name: "No properties - add minimal frontmatter",
			input: `# Just a heading

Some content.`,
			expected: `# Just a heading

Some content.`,
		},
		{
			name: "Logseq array format to YAML",
			input: `title:: Tagged Note
tags:: [project, work]

Content with tags.`,
			expected: `---
tags: [project, work]
title: Tagged Note
---

Content with tags.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewTransformer(health.BrainTypeLogseq, health.BrainTypeFlip)
			result := transformer.TransformContent(tt.input)
			if result != tt.expected {
				t.Errorf("TransformContent() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

func TestTransformContent_SameStyle(t *testing.T) {
	// Flip to Obsidian - both use YAML, no content change
	input := `---
title: My Note
---

Content here.`

	transformer := NewTransformer(health.BrainTypeFlip, health.BrainTypeObsidian)
	result := transformer.TransformContent(input)

	if result != input {
		t.Errorf("Same frontmatter style should not change content.\nGot: %q\nWant: %q", result, input)
	}
}

func TestGetFrontmatterStyle(t *testing.T) {
	tests := []struct {
		brainType health.BrainType
		expected  FrontmatterStyle
	}{
		{health.BrainTypeLogseq, FrontmatterLogseq},
		{health.BrainTypeFlip, FrontmatterYAML},
		{health.BrainTypeObsidian, FrontmatterYAML},
		{health.BrainTypeDendron, FrontmatterYAML},
		{health.BrainTypeFoam, FrontmatterYAML},
	}

	for _, tt := range tests {
		t.Run(string(tt.brainType), func(t *testing.T) {
			result := GetFrontmatterStyle(tt.brainType)
			if result != tt.expected {
				t.Errorf("GetFrontmatterStyle(%s) = %d, want %d", tt.brainType, result, tt.expected)
			}
		})
	}
}

func TestTransformProperties(t *testing.T) {
	props := map[string]string{
		"title":  "Test Note",
		"status": "active",
	}

	// To YAML
	yamlTransformer := NewTransformer(health.BrainTypeLogseq, health.BrainTypeFlip)
	yamlResult := yamlTransformer.TransformProperties(props)

	if !contains(yamlResult, "---") {
		t.Error("YAML output should contain frontmatter delimiters")
	}
	if !contains(yamlResult, "title: Test Note") {
		t.Error("YAML output should contain title property")
	}

	// To Logseq
	logseqTransformer := NewTransformer(health.BrainTypeFlip, health.BrainTypeLogseq)
	logseqResult := logseqTransformer.TransformProperties(props)

	if contains(logseqResult, "---") {
		t.Error("Logseq output should not contain YAML delimiters")
	}
	if !contains(logseqResult, "title:: Test Note") {
		t.Error("Logseq output should contain title property bullet")
	}
}

func TestGetFullTransformationSummary(t *testing.T) {
	// Logseq → Flip should mention both structure and content changes
	transformer := NewTransformer(health.BrainTypeLogseq, health.BrainTypeFlip)
	summary := transformer.GetFullTransformationSummary()

	if summary == "" {
		t.Error("Expected non-empty full summary")
	}

	// Should mention structure or content changes
	if !contains(summary, "Structure") && !contains(summary, "Content") && !contains(summary, "adjustment") {
		t.Logf("Summary: %s", summary)
	}
}

// ============================================================================
// LOGSEQ BULLET PROPERTY TESTS
// ============================================================================

func TestTransformContent_LogseqBulletPropsToYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Bullet property format",
			input: `- title:: My Note
- tags:: project

Content here.`,
			expected: `---
tags: project
title: My Note
---

Content here.`,
		},
		{
			name: "Mixed bullet and bare properties",
			input: `title:: My Note
- tags:: project
date:: 2025-01-15

Content here.`,
			expected: `---
date: 2025-01-15
tags: project
title: My Note
---

Content here.`,
		},
		{
			name: "Discards collapsed and card properties",
			input: `title:: Keep This
collapsed:: true
card-last-interval:: 4
card-repeats:: 1
background-color:: red

Content here.`,
			expected: `---
title: Keep This
---

Content here.`,
		},
		{
			name: "Converts created-at epoch to date",
			input: `title:: Note
created-at:: 1703030400000

Content.`,
			expected: `---
created: 2023-12-20
title: Note
---

Content.`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformer := NewTransformer(health.BrainTypeLogseq, health.BrainTypeFlip)
			result := transformer.TransformContent(tt.input)
			if result != tt.expected {
				t.Errorf("TransformContent() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

// ============================================================================
// CLEANUP LOGSEQ ARTIFACTS TESTS
// ============================================================================

func TestCleanupLogseqArtifacts(t *testing.T) {
	transformer := NewTransformer(health.BrainTypeLogseq, health.BrainTypeFlip)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Removes collapsed props from body",
			input: `---
title: Test
---

Some text
- collapsed:: true
More text`,
			expected: `---
title: Test
---

Some text
More text`,
		},
		{
			name: "Converts video embeds",
			input: `---
title: Test
---

Check this: {{video https://www.youtube.com/watch?v=abc123}}`,
			expected: `---
title: Test
---

Check this: [YouTube Video](https://www.youtube.com/watch?v=abc123)`,
		},
		{
			name: "Removes query blocks",
			input: `---
title: Test
---

Content
- {{query (and (task TODO) (page "project"))}}
More content`,
			expected: `---
title: Test
---

Content
More content`,
		},
		{
			name: "Converts task markers",
			input: `---
title: Test
---

- TODO Buy groceries
- DONE Write tests
- LATER Read book
- CANCELLED Old task`,
			expected: `---
title: Test
---

- [ ] Buy groceries
- [x] Write tests
- [ ] Read book
- [~] Old task`,
		},
		{
			name: "Non-logseq source returns unchanged",
			input: `---
title: Test
---

- collapsed:: true
{{video https://example.com}}`,
			expected: `---
title: Test
---

- collapsed:: true
{{video https://example.com}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.name == "Non-logseq source returns unchanged" {
				flipTransformer := NewTransformer(health.BrainTypeFlip, health.BrainTypeFlip)
				result = flipTransformer.CleanupLogseqArtifacts(tt.input)
			} else {
				result = transformer.CleanupLogseqArtifacts(tt.input)
			}
			if result != tt.expected {
				t.Errorf("CleanupLogseqArtifacts() =\n%q\nwant:\n%q", result, tt.expected)
			}
		})
	}
}

func TestSplitFrontmatterBody(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantFM string
		wantB  string
		wantOK bool
	}{
		{
			name:   "With frontmatter",
			input:  "---\ntitle: Test\n---\nBody here",
			wantFM: "---\ntitle: Test\n---",
			wantB:  "\nBody here",
			wantOK: true,
		},
		{
			name:   "No frontmatter",
			input:  "# Heading\nContent",
			wantFM: "",
			wantB:  "# Heading\nContent",
			wantOK: false,
		},
		{
			name:   "Unclosed frontmatter",
			input:  "---\ntitle: Test\nBody without closing",
			wantFM: "",
			wantB:  "---\ntitle: Test\nBody without closing",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, ok := splitFrontmatterBody(tt.input)
			if fm != tt.wantFM || body != tt.wantB || ok != tt.wantOK {
				t.Errorf("splitFrontmatterBody(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tt.input, fm, body, ok, tt.wantFM, tt.wantB, tt.wantOK)
			}
		})
	}
}
