package protocols

import (
	"fmt"
	"strings"
	"time"
)

// ProtocolType defines the type of protocol to generate
type ProtocolType string

const (
	ProtocolTypeSingle  ProtocolType = "single"
	ProtocolTypeRolling ProtocolType = "rolling"
)

// Protocol represents a generated meeting protocol
type Protocol struct {
	Type         ProtocolType
	GeneratedAt  time.Time
	Meetings     []*MeetingContent
	SeriesName   string
	TimeRange    TimeRange
	OutputFormat string
}

// TimeRange represents a time period for rolling protocols
type TimeRange struct {
	From time.Time
	To   time.Time
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

	return &Protocol{
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

	// All action items aggregated
	allActionItems := p.aggregateActionItems()
	if len(allActionItems) > 0 {
		sb.WriteString("### Action Items Overview\n\n")

		completed := []ActionItem{}
		open := []ActionItem{}

		for _, item := range allActionItems {
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
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	// All decisions timeline
	allDecisions := p.aggregateDecisions()
	if len(allDecisions) > 0 {
		sb.WriteString("### Decisions Timeline\n\n")
		for i, decision := range allDecisions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, decision.Text))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*Rolling protocol generated: %s*\n", p.GeneratedAt.Format("2006-01-02 15:04")))

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
