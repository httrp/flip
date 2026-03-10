package protocols

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ProtocolType defines the type of protocol to generate
type ProtocolType string

const (
	ProtocolTypeSingle                ProtocolType = "single"
	ProtocolTypeRolling               ProtocolType = "rolling"
	RollingProtocolManualNotesStart                = "<!-- flip:manual-notes:start -->"
	RollingProtocolManualNotesEnd                  = "<!-- flip:manual-notes:end -->"
	RollingProtocolOpenQuestionsStart              = "<!-- flip:open-questions:start -->"
	RollingProtocolOpenQuestionsEnd                = "<!-- flip:open-questions:end -->"
	RollingProtocolFollowUpsStart                  = "<!-- flip:follow-ups:start -->"
	RollingProtocolFollowUpsEnd                    = "<!-- flip:follow-ups:end -->"
)

const rollingProtocolManualNotesPlaceholder = "_Add manual notes here. This section is preserved by flip meeting protocol --sync._"
const rollingProtocolOpenQuestionsPlaceholder = "_Track unresolved questions here. This section is preserved by flip meeting protocol --sync._"
const rollingProtocolFollowUpsPlaceholder = "_Track stakeholder follow-ups here. This section is preserved by flip meeting protocol --sync._"

type PreservedSection struct {
	Heading     string
	StartMarker string
	EndMarker   string
	Placeholder string
}

var RollingProtocolPreservedSections = []PreservedSection{
	{
		Heading:     "Manual Notes",
		StartMarker: RollingProtocolManualNotesStart,
		EndMarker:   RollingProtocolManualNotesEnd,
		Placeholder: rollingProtocolManualNotesPlaceholder,
	},
	{
		Heading:     "Open Questions",
		StartMarker: RollingProtocolOpenQuestionsStart,
		EndMarker:   RollingProtocolOpenQuestionsEnd,
		Placeholder: rollingProtocolOpenQuestionsPlaceholder,
	},
	{
		Heading:     "Follow-Ups",
		StartMarker: RollingProtocolFollowUpsStart,
		EndMarker:   RollingProtocolFollowUpsEnd,
		Placeholder: rollingProtocolFollowUpsPlaceholder,
	},
}

// Protocol represents a generated meeting protocol
type Protocol struct {
	Type         ProtocolType
	GeneratedAt  time.Time
	Meetings     []*MeetingContent
	SeriesName   string
	TimeRange    TimeRange
	OutputFormat string
	Information  []InformationEntry
	Decisions    []DecisionEntry
	Actions      []ActionSummary
}

// TimeRange represents a time period for rolling protocols
type TimeRange struct {
	From time.Time
	To   time.Time
}

// InformationEntry represents a noteworthy information item in a rolling protocol.
type InformationEntry struct {
	Key              string
	Text             string
	MeetingDate      string
	MeetingTitle     string
	FirstMeetingDate string
	LastMeetingDate  string
	LastMeetingTitle string
	Occurrences      int
}

// DecisionEntry represents a decision with source context.
type DecisionEntry struct {
	Text         string
	MeetingDate  string
	MeetingTitle string
}

// ActionSummary tracks a task across meetings.
type ActionSummary struct {
	Key              string
	Text             string
	Assignee         string
	DueDate          string
	Priority         string
	FirstMeetingDate string
	LastMeetingDate  string
	LastMeetingTitle string
	Completed        bool
	Occurrences      int
}

// LatestChanges summarizes what changed in the most recent meeting.
type LatestChanges struct {
	MeetingDate      string
	MeetingTitle     string
	Information      []InformationEntry
	Decisions        []DecisionEntry
	NewActions       []ActionSummary
	CompletedActions []ActionSummary
	ReopenedActions  []ActionSummary
}

// GenerateSingleProtocol creates a protocol for a single meeting
func GenerateSingleProtocol(meeting *MeetingContent) *Protocol {
	return &Protocol{
		Type:         ProtocolTypeSingle,
		GeneratedAt:  time.Now(),
		Meetings:     []*MeetingContent{meeting},
		OutputFormat: "markdown",
	}
}

// GenerateRollingProtocol creates a protocol for multiple meetings in a series
func GenerateRollingProtocol(meetings []*MeetingContent, seriesName string) *Protocol {
	if len(meetings) == 0 {
		return nil
	}

	// Determine time range
	var from, to time.Time
	for i, m := range meetings {
		date, err := m.GetDateParsed()
		if err != nil {
			continue
		}

		if i == 0 || date.Before(from) {
			from = date
		}
		if i == 0 || date.After(to) {
			to = date
		}
	}

	protocol := &Protocol{
		Type:        ProtocolTypeRolling,
		GeneratedAt: time.Now(),
		Meetings:    meetings,
		SeriesName:  seriesName,
		TimeRange: TimeRange{
			From: from,
			To:   to,
		},
		OutputFormat: "markdown",
	}

	protocol.Information = protocol.aggregateInformation()
	protocol.Decisions = protocol.aggregateDecisionEntries()
	protocol.Actions = protocol.aggregateActionSummaries()

	return protocol
}

// RenderMarkdown generates the markdown output for the protocol
func (p *Protocol) RenderMarkdown() string {
	if p.Type == ProtocolTypeSingle {
		return p.renderSingleProtocol()
	}
	return p.renderRollingProtocol()
}

// renderSingleProtocol generates markdown for a single meeting
func (p *Protocol) renderSingleProtocol() string {
	if len(p.Meetings) == 0 {
		return ""
	}

	m := p.Meetings[0]
	var sb strings.Builder

	// Title
	sb.WriteString(fmt.Sprintf("# Meeting Protocol: %s\n\n", m.Title))

	// Metadata
	sb.WriteString(fmt.Sprintf("**Date**: %s", m.Date))
	if m.Time != "" {
		sb.WriteString(fmt.Sprintf(" %s", m.Time))
	}
	sb.WriteString("\n")

	if m.Duration != "" {
		sb.WriteString(fmt.Sprintf("**Duration**: %s  \n", m.Duration))
	}

	if m.Organization != "" {
		sb.WriteString(fmt.Sprintf("**Organization**: %s  \n", m.Organization))
	}

	if m.Project != "" {
		sb.WriteString(fmt.Sprintf("**Project**: %s  \n", m.Project))
	}

	if m.Context != "" {
		sb.WriteString(fmt.Sprintf("**Context**: %s  \n", m.Context))
	}

	// Participants
	if len(m.Participants) > 0 {
		sb.WriteString(fmt.Sprintf("**Participants**: %s  \n", strings.Join(m.Participants, ", ")))
	}

	sb.WriteString("\n")

	// Decisions (if any found)
	if len(m.Decisions) > 0 {
		sb.WriteString("## Decisions\n\n")
		for i, decision := range m.Decisions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, decision.Text))
		}
		sb.WriteString("\n")
	}

	// Action Items
	if len(m.ActionItems) > 0 {
		sb.WriteString("## Action Items\n\n")
		for _, item := range m.ActionItems {
			checkbox := "[ ]"
			if item.Completed {
				checkbox = "[x]"
			}

			sb.WriteString(fmt.Sprintf("- %s %s", checkbox, item.Text))

			// Add metadata if available
			meta := []string{}
			if item.Assignee != "" {
				meta = append(meta, fmt.Sprintf("@%s", item.Assignee))
			}
			if item.DueDate != "" {
				meta = append(meta, fmt.Sprintf("due: %s", item.DueDate))
			}
			if item.Priority != "" {
				meta = append(meta, fmt.Sprintf("priority: %s", item.Priority))
			}

			if len(meta) > 0 {
				sb.WriteString(fmt.Sprintf(" (%s)", strings.Join(meta, ", ")))
			}

			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Agenda
	if m.Agenda != "" && strings.TrimSpace(m.Agenda) != "" {
		sb.WriteString("## Agenda\n\n")
		sb.WriteString(m.Agenda)
		sb.WriteString("\n\n")
	}

	// Discussion Notes
	if m.Notes != "" && strings.TrimSpace(m.Notes) != "" {
		sb.WriteString("## Discussion Notes\n\n")
		sb.WriteString(m.Notes)
		sb.WriteString("\n\n")
	}

	// Related
	if len(m.Related) > 0 {
		sb.WriteString("## Related\n\n")
		for _, link := range m.Related {
			sb.WriteString(fmt.Sprintf("- [[%s]]\n", link))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*Protocol generated: %s*\n", p.GeneratedAt.Format("2006-01-02 15:04")))

	return sb.String()
}

// renderRollingProtocol generates markdown for a series of meetings
func (p *Protocol) renderRollingProtocol() string {
	var sb strings.Builder

	sb.WriteString(p.renderRollingFrontmatter())

	// Title
	sb.WriteString(fmt.Sprintf("# Rolling Protocol: %s\n\n", p.SeriesName))

	// Metadata
	sb.WriteString(fmt.Sprintf("**Period**: %s to %s  \n",
		p.TimeRange.From.Format("2006-01-02"),
		p.TimeRange.To.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("**Meetings**: %d  \n", len(p.Meetings)))

	// Get organization from first meeting (if consistent)
	if len(p.Meetings) > 0 && p.Meetings[0].Organization != "" {
		sb.WriteString(fmt.Sprintf("**Organization**: %s  \n", p.Meetings[0].Organization))
	}

	sb.WriteString("\n")

	// Overview
	sb.WriteString("## Overview\n\n")
	sb.WriteString(fmt.Sprintf("This rolling protocol covers %d meetings of the \"%s\" series.\n\n",
		len(p.Meetings), p.SeriesName))

	changes := p.latestChanges()
	if changes != nil {
		sb.WriteString("## Changes Since Last Meeting\n\n")
		sb.WriteString(fmt.Sprintf("Latest meeting: %s (%s)\n\n", changes.MeetingTitle, changes.MeetingDate))

		if len(changes.Information) > 0 {
			sb.WriteString("### Information\n\n")
			for _, info := range changes.Information {
				sb.WriteString(fmt.Sprintf("- %s\n", info.Text))
			}
			sb.WriteString("\n")
		}

		if len(changes.Decisions) > 0 {
			sb.WriteString("### Decisions\n\n")
			for _, decision := range changes.Decisions {
				sb.WriteString(fmt.Sprintf("- %s\n", decision.Text))
			}
			sb.WriteString("\n")
		}

		if len(changes.NewActions) > 0 || len(changes.CompletedActions) > 0 || len(changes.ReopenedActions) > 0 {
			sb.WriteString("### Actions\n\n")
			for _, item := range changes.NewActions {
				sb.WriteString(fmt.Sprintf("- NEW: %s", item.Text))
				if item.Assignee != "" {
					sb.WriteString(fmt.Sprintf(" (@%s)", item.Assignee))
				}
				sb.WriteString("\n")
			}
			for _, item := range changes.CompletedActions {
				sb.WriteString(fmt.Sprintf("- DONE: %s", item.Text))
				if item.Assignee != "" {
					sb.WriteString(fmt.Sprintf(" (@%s)", item.Assignee))
				}
				sb.WriteString("\n")
			}
			for _, item := range changes.ReopenedActions {
				sb.WriteString(fmt.Sprintf("- REOPENED: %s", item.Text))
				if item.Assignee != "" {
					sb.WriteString(fmt.Sprintf(" (@%s)", item.Assignee))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	// Timeline - individual meetings
	sb.WriteString("## Timeline\n\n")

	for i, m := range p.Meetings {
		sb.WriteString(fmt.Sprintf("### %s: %s\n\n", m.Date, m.Title))

		if len(m.Participants) > 0 {
			sb.WriteString(fmt.Sprintf("**Participants**: %s\n\n", strings.Join(m.Participants, ", ")))
		}

		// Decisions for this meeting
		if len(m.Decisions) > 0 {
			sb.WriteString("**Decisions**:\n")
			for _, decision := range m.Decisions {
				sb.WriteString(fmt.Sprintf("- %s\n", decision.Text))
			}
			sb.WriteString("\n")
		}

		// Action items for this meeting
		if len(m.ActionItems) > 0 {
			sb.WriteString("**Action Items**:\n")
			for _, item := range m.ActionItems {
				status := "⏳"
				if item.Completed {
					status = "✅"
				}

				sb.WriteString(fmt.Sprintf("- %s %s", status, item.Text))
				if item.Assignee != "" {
					sb.WriteString(fmt.Sprintf(" (@%s)", item.Assignee))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		// Separator between meetings (not after last one)
		if i < len(p.Meetings)-1 {
			sb.WriteString("---\n\n")
		}
	}

	// Aggregated summaries
	sb.WriteString("\n## Summary\n\n")

	if len(p.Information) > 0 {
		sb.WriteString("### Information Overview\n\n")
		for _, info := range p.Information {
			sb.WriteString(fmt.Sprintf("- %s", info.Text))
			meta := []string{}
			if info.LastMeetingDate != "" {
				meta = append(meta, fmt.Sprintf("last updated: %s", info.LastMeetingDate))
			}
			if info.Occurrences > 1 {
				meta = append(meta, fmt.Sprintf("seen %d times", info.Occurrences))
			}
			if info.LastMeetingTitle != "" {
				meta = append(meta, info.LastMeetingTitle)
			}
			if len(meta) > 0 {
				sb.WriteString(fmt.Sprintf(" (%s)", strings.Join(meta, ", ")))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// All action items aggregated
	if len(p.Actions) > 0 {
		sb.WriteString("### Action Items Overview\n\n")

		completed := []ActionSummary{}
		open := []ActionSummary{}

		for _, item := range p.Actions {
			if item.Completed {
				completed = append(completed, item)
			} else {
				open = append(open, item)
			}
		}

		if len(completed) > 0 {
			sb.WriteString(fmt.Sprintf("#### ✅ Completed (%d)\n\n", len(completed)))
			for _, item := range completed {
				sb.WriteString(fmt.Sprintf("- %s", item.Text))
				if item.Assignee != "" {
					sb.WriteString(fmt.Sprintf(" (@%s)", item.Assignee))
				}
				if item.LastMeetingDate != "" {
					sb.WriteString(fmt.Sprintf(" (last seen: %s)", item.LastMeetingDate))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		if len(open) > 0 {
			sb.WriteString(fmt.Sprintf("#### ⏳ Open (%d)\n\n", len(open)))
			for _, item := range open {
				sb.WriteString(fmt.Sprintf("- %s", item.Text))
				if item.Assignee != "" {
					sb.WriteString(fmt.Sprintf(" (@%s)", item.Assignee))
				}
				if item.DueDate != "" {
					sb.WriteString(fmt.Sprintf(" (due: %s)", item.DueDate))
				}
				if item.LastMeetingDate != "" {
					sb.WriteString(fmt.Sprintf(" (last seen: %s)", item.LastMeetingDate))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	// All decisions timeline
	if len(p.Decisions) > 0 {
		sb.WriteString("### Decisions Timeline\n\n")
		for i, decision := range p.Decisions {
			sb.WriteString(fmt.Sprintf("%d. [%s] %s", i+1, decision.MeetingDate, decision.Text))
			if decision.MeetingTitle != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", decision.MeetingTitle))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Preserved manual sections
	sb.WriteString(p.renderRollingPreservedSections())

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*Rolling protocol generated: %s*\n", p.GeneratedAt.Format("2006-01-02 15:04")))

	return sb.String()
}

func (p *Protocol) renderRollingFrontmatter() string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("protocol_type: rolling\n")
	sb.WriteString(fmt.Sprintf("series_name: %q\n", p.SeriesName))
	sb.WriteString(fmt.Sprintf("generated_at: %s\n", p.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("meetings_count: %d\n", len(p.Meetings)))
	sb.WriteString("time_range:\n")
	sb.WriteString(fmt.Sprintf("  from: %s\n", p.TimeRange.From.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("  to: %s\n", p.TimeRange.To.Format("2006-01-02")))
	sb.WriteString("meetings_included:\n")
	for _, meeting := range p.Meetings {
		sb.WriteString(fmt.Sprintf("  - date: %s\n", meeting.Date))
		sb.WriteString(fmt.Sprintf("    title: %q\n", meeting.Title))
		if meeting.FilePath != "" {
			sb.WriteString(fmt.Sprintf("    file: %q\n", meeting.FilePath))
		}
	}
	sb.WriteString("---\n\n")
	return sb.String()
}

func (p *Protocol) renderRollingPreservedSections() string {
	var sb strings.Builder
	for _, section := range RollingProtocolPreservedSections {
		sb.WriteString(fmt.Sprintf("## %s\n\n", section.Heading))
		sb.WriteString(section.StartMarker)
		sb.WriteString("\n")
		sb.WriteString(section.Placeholder)
		sb.WriteString("\n")
		sb.WriteString(section.EndMarker)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

// aggregateActionItems collects all action items from all meetings
func (p *Protocol) aggregateActionItems() []ActionItem {
	var all []ActionItem
	for _, m := range p.Meetings {
		all = append(all, m.ActionItems...)
	}
	return all
}

// aggregateDecisions collects all decisions from all meetings
func (p *Protocol) aggregateDecisions() []Decision {
	var all []Decision
	for _, m := range p.Meetings {
		all = append(all, m.Decisions...)
	}
	return all
}

func (p *Protocol) aggregateInformation() []InformationEntry {
	index := make(map[string]*InformationEntry)
	order := []string{}

	for _, meeting := range p.Meetings {
		for _, info := range extractInformationEntries(meeting) {
			if info.Key == "" {
				continue
			}

			entry, ok := index[info.Key]
			if !ok {
				entry = &InformationEntry{
					Key:              info.Key,
					Text:             info.Text,
					MeetingDate:      info.MeetingDate,
					MeetingTitle:     info.MeetingTitle,
					FirstMeetingDate: info.MeetingDate,
					LastMeetingDate:  info.MeetingDate,
					LastMeetingTitle: info.MeetingTitle,
				}
				index[info.Key] = entry
				order = append(order, info.Key)
			}

			entry.Text = info.Text
			entry.MeetingDate = info.MeetingDate
			entry.MeetingTitle = info.MeetingTitle
			entry.LastMeetingDate = info.MeetingDate
			entry.LastMeetingTitle = info.MeetingTitle
			entry.Occurrences++
		}
	}

	result := make([]InformationEntry, 0, len(order))
	for _, key := range order {
		result = append(result, *index[key])
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].LastMeetingDate != result[j].LastMeetingDate {
			return result[i].LastMeetingDate > result[j].LastMeetingDate
		}
		if result[i].Occurrences != result[j].Occurrences {
			return result[i].Occurrences > result[j].Occurrences
		}
		return result[i].Text < result[j].Text
	})

	return result
}

func (p *Protocol) aggregateDecisionEntries() []DecisionEntry {
	var all []DecisionEntry
	for _, meeting := range p.Meetings {
		for _, decision := range meeting.Decisions {
			all = append(all, DecisionEntry{
				Text:         decision.Text,
				MeetingDate:  meeting.Date,
				MeetingTitle: meeting.Title,
			})
		}
	}
	return all
}

func (p *Protocol) aggregateActionSummaries() []ActionSummary {
	index := make(map[string]*ActionSummary)
	order := []string{}

	for _, meeting := range p.Meetings {
		for _, item := range meeting.ActionItems {
			key := normalizeActionKey(item)
			if key == "" {
				continue
			}
			entry, ok := index[key]
			if !ok {
				entry = &ActionSummary{
					Key:              key,
					Text:             item.Text,
					Assignee:         item.Assignee,
					DueDate:          item.DueDate,
					Priority:         item.Priority,
					FirstMeetingDate: meeting.Date,
				}
				index[key] = entry
				order = append(order, key)
			}

			entry.LastMeetingDate = meeting.Date
			entry.LastMeetingTitle = meeting.Title
			entry.Occurrences++
			entry.Completed = item.Completed
			if entry.Assignee == "" {
				entry.Assignee = item.Assignee
			}
			if entry.DueDate == "" {
				entry.DueDate = item.DueDate
			}
			if entry.Priority == "" {
				entry.Priority = item.Priority
			}
		}
	}

	result := make([]ActionSummary, 0, len(order))
	for _, key := range order {
		result = append(result, *index[key])
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Completed != result[j].Completed {
			return !result[i].Completed
		}
		return result[i].Text < result[j].Text
	})

	return result
}

func (p *Protocol) latestChanges() *LatestChanges {
	if len(p.Meetings) == 0 {
		return nil
	}

	latest := p.Meetings[len(p.Meetings)-1]
	changes := &LatestChanges{
		MeetingDate:  latest.Date,
		MeetingTitle: latest.Title,
	}

	changes.Information = extractInformationEntries(latest)

	for _, decision := range latest.Decisions {
		changes.Decisions = append(changes.Decisions, DecisionEntry{
			Text:         decision.Text,
			MeetingDate:  latest.Date,
			MeetingTitle: latest.Title,
		})
	}

	previous := map[string]ActionSummary{}
	for _, meeting := range p.Meetings[:len(p.Meetings)-1] {
		for _, item := range meeting.ActionItems {
			key := normalizeActionKey(item)
			if key == "" {
				continue
			}
			previous[key] = ActionSummary{
				Key:       key,
				Text:      item.Text,
				Assignee:  item.Assignee,
				DueDate:   item.DueDate,
				Priority:  item.Priority,
				Completed: item.Completed,
			}
		}
	}

	for _, item := range latest.ActionItems {
		key := normalizeActionKey(item)
		if key == "" {
			continue
		}

		summary := ActionSummary{
			Key:              key,
			Text:             item.Text,
			Assignee:         item.Assignee,
			DueDate:          item.DueDate,
			Priority:         item.Priority,
			FirstMeetingDate: latest.Date,
			LastMeetingDate:  latest.Date,
			LastMeetingTitle: latest.Title,
			Completed:        item.Completed,
			Occurrences:      1,
		}

		prev, existed := previous[key]
		if !existed {
			changes.NewActions = append(changes.NewActions, summary)
			continue
		}

		if !prev.Completed && item.Completed {
			changes.CompletedActions = append(changes.CompletedActions, summary)
		}
		if prev.Completed && !item.Completed {
			changes.ReopenedActions = append(changes.ReopenedActions, summary)
		}
	}

	return changes
}

func normalizeActionKey(item ActionItem) string {
	text := strings.ToLower(strings.TrimSpace(item.Text))
	text = strings.ReplaceAll(text, "  ", " ")
	if text == "" {
		return ""
	}
	if item.Assignee != "" {
		return text + "::" + strings.ToLower(strings.TrimSpace(item.Assignee))
	}
	return text
}

func extractInformationEntries(meeting *MeetingContent) []InformationEntry {
	statements := extractInformationStatements(meeting.Notes)
	entries := make([]InformationEntry, 0, len(statements))
	for _, statement := range statements {
		key := normalizeInformationKey(statement)
		if key == "" {
			continue
		}
		entries = append(entries, InformationEntry{
			Key:              key,
			Text:             statement,
			MeetingDate:      meeting.Date,
			MeetingTitle:     meeting.Title,
			FirstMeetingDate: meeting.Date,
			LastMeetingDate:  meeting.Date,
			LastMeetingTitle: meeting.Title,
			Occurrences:      1,
		})
	}
	return entries
}

func extractInformationStatements(notes string) []string {
	lines := strings.Split(notes, "\n")
	statements := []string{}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		isBullet := strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")
		if isBullet {
			trimmed = strings.TrimSpace(trimmed[2:])
		}

		for _, part := range splitInformationSentenceCandidates(trimmed) {
			cleaned := cleanInformationText(part)
			if cleaned == "" {
				continue
			}
			statements = append(statements, cleaned)
		}
	}

	return statements
}

func splitInformationSentenceCandidates(line string) []string {
	replacer := strings.NewReplacer("?", ".", "!", ".", ";", ".")
	normalized := replacer.Replace(line)
	parts := strings.Split(normalized, ".")
	if len(parts) == 1 {
		return []string{line}
	}
	return parts
}

func cleanInformationText(text string) string {
	cleaned := strings.TrimSpace(text)
	cleaned = strings.TrimLeft(cleaned, "-*• ")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, ".")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned
}

func normalizeInformationKey(text string) string {
	cleaned := strings.ToLower(cleanInformationText(text))
	cleaned = strings.NewReplacer(",", "", ":", "", "(", "", ")", "", "\"", "", "'", "").Replace(cleaned)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned
}
