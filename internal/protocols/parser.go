package protocols

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MeetingContent represents parsed content from a meeting note
type MeetingContent struct {
	// Frontmatter fields
	Title        string   `yaml:"title"`
	Date         string   `yaml:"date"`
	Time         string   `yaml:"time"`
	Duration     string   `yaml:"duration"`
	Author       string   `yaml:"author"`
	Organization string   `yaml:"organization"`
	Project      string   `yaml:"project"`
	Context      string   `yaml:"context"`
	Series       string   `yaml:"series"`
	Tags         []string `yaml:"tags"`
	Type         string   `yaml:"type"`
	Created      string   `yaml:"created"`
	Updated      string   `yaml:"updated"`

	// Parsed sections
	Participants []string
	Agenda       string
	Notes        string
	ActionItems  []ActionItem
	Decisions    []Decision
	Related      []string

	// Metadata
	FilePath string
	ParsedAt time.Time
}

// ActionItem represents a task from meeting action items
type ActionItem struct {
	Text      string
	Completed bool
	Assignee  string
	DueDate   string
	Priority  string
}

// Decision represents a decision made in the meeting
type Decision struct {
	Text    string
	Context string
	Impact  string
}

// ParseMeetingFile reads and parses a meeting markdown file
func ParseMeetingFile(filePath string) (*MeetingContent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	meeting := &MeetingContent{
		FilePath: filePath,
		ParsedAt: time.Now(),
	}

	// Extract frontmatter
	if err := extractFrontmatter(content, meeting); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Extract sections
	extractSections(content, meeting)

	return meeting, nil
}

// extractFrontmatter parses YAML frontmatter from markdown
func extractFrontmatter(content string, meeting *MeetingContent) error {
	if !strings.HasPrefix(content, "---\n") {
		return fmt.Errorf("no frontmatter found")
	}

	// Find end of frontmatter
	endIdx := strings.Index(content[4:], "\n---")
	if endIdx == -1 {
		return fmt.Errorf("malformed frontmatter")
	}

	frontmatter := content[4 : 4+endIdx]

	// Parse YAML
	if err := yaml.Unmarshal([]byte(frontmatter), meeting); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	return nil
}

// extractSections parses markdown sections from the content
func extractSections(content string, meeting *MeetingContent) {
	lines := strings.Split(content, "\n")

	var currentSection string
	var sectionContent strings.Builder

	for _, line := range lines {
		// Check for section headers
		if strings.HasPrefix(line, "## ") {
			// Save previous section
			saveSectionContent(currentSection, sectionContent.String(), meeting)

			// Start new section
			currentSection = strings.TrimPrefix(line, "## ")
			sectionContent.Reset()
			continue
		}

		// Skip frontmatter
		if strings.HasPrefix(line, "---") {
			continue
		}

		// Skip main heading (# Title)
		if strings.HasPrefix(line, "# ") && currentSection == "" {
			continue
		}

		// Accumulate section content
		if currentSection != "" {
			sectionContent.WriteString(line + "\n")
		}
	}

	// Save last section
	if currentSection != "" {
		saveSectionContent(currentSection, sectionContent.String(), meeting)
	}
}

// saveSectionContent processes and stores section content
func saveSectionContent(sectionName, content string, meeting *MeetingContent) {
	content = strings.TrimSpace(content)

	switch sectionName {
	case "Participants":
		meeting.Participants = parseParticipants(content)
	case "Agenda":
		meeting.Agenda = content
	case "Notes":
		meeting.Notes = content
		// Extract decisions from notes (simple pattern matching for now)
		meeting.Decisions = extractDecisions(content)
	case "Action Items":
		meeting.ActionItems = parseActionItems(content)
	case "Related":
		meeting.Related = parseRelated(content)
	}
}

// parseParticipants extracts participant names from section
func parseParticipants(content string) []string {
	var participants []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and list markers
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")

		if line != "" {
			participants = append(participants, line)
		}
	}

	return participants
}

// parseActionItems extracts action items from markdown task list
func parseActionItems(content string) []ActionItem {
	var items []ActionItem
	lines := strings.Split(content, "\n")

	taskPattern := regexp.MustCompile(`^[-*]\s+\[([x ])\]\s+(.+)`)

	for _, line := range lines {
		matches := taskPattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) == 3 {
			item := ActionItem{
				Text:      strings.TrimSpace(matches[2]),
				Completed: matches[1] == "x",
			}

			// Extract metadata from task text
			item.Assignee = extractAssignee(item.Text)
			item.DueDate = extractDueDate(item.Text)
			item.Priority = extractPriority(item.Text)

			items = append(items, item)
		}
	}

	return items
}

// extractDecisions finds decision patterns in meeting notes
func extractDecisions(content string) []Decision {
	var decisions []Decision

	// Simple pattern matching for common decision phrases
	decisionPatterns := []string{
		`(?i)we (decided|agreed|chose) (to )?(.+)`,
		`(?i)decision: (.+)`,
		`(?i)the team (agreed|decided) (.+)`,
	}

	for _, pattern := range decisionPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 0 {
				// Get the full matched text as decision
				decision := Decision{
					Text: strings.TrimSpace(match[0]),
				}
				decisions = append(decisions, decision)
			}
		}
	}

	return decisions
}

// parseRelated extracts related note links
func parseRelated(content string) []string {
	var related []string

	// Match both [[wiki-links]] and [markdown](links)
	wikiLinkPattern := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	mdLinkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	// Extract wiki links
	for _, match := range wikiLinkPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			related = append(related, match[1])
		}
	}

	// Extract markdown links
	for _, match := range mdLinkPattern.FindAllStringSubmatch(content, -1) {
		if len(match) > 2 {
			related = append(related, match[1])
		}
	}

	return related
}

// Helper functions to extract metadata from action item text

func extractAssignee(text string) string {
	// Match @username or (@name)
	patterns := []string{`@(\w+)`, `\(@([^)]+)\)`}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(text); len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

func extractDueDate(text string) string {
	// Match dates in format: due: 2026-01-20 or Deadline: YYYY-MM-DD
	datePattern := regexp.MustCompile(`(?i)(due|deadline):\s*(\d{4}-\d{2}-\d{2})`)

	if matches := datePattern.FindStringSubmatch(text); len(matches) > 2 {
		return matches[2]
	}

	return ""
}

func extractPriority(text string) string {
	// Match Priority: High/Medium/Low
	priorityPattern := regexp.MustCompile(`(?i)priority:\s*(high|medium|low)`)

	if matches := priorityPattern.FindStringSubmatch(text); len(matches) > 1 {
		return strings.ToLower(matches[1])
	}

	return ""
}

// GetSeriesName returns the series name if this is part of a series
func (m *MeetingContent) GetSeriesName() string {
	return m.Series
}

// IsSeriesMeeting checks if this meeting is part of a series
func (m *MeetingContent) IsSeriesMeeting() bool {
	return m.Series != ""
}

// GetDateParsed returns the meeting date as time.Time
func (m *MeetingContent) GetDateParsed() (time.Time, error) {
	return time.Parse("2006-01-02", m.Date)
}
