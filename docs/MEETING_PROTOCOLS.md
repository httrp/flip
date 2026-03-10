# Meeting Protocols - Design Document

## Overview

Generate structured meeting protocols from meeting notes with AI-powered content extraction.

## Use Cases

### 1. Single Meeting Protocol (Ergebnisprotokoll)
- **Input**: One meeting note (e.g., `2026-01-15-meeting-kickoff-2026.md`)
- **Output**: Formatted protocol with decisions, action items, attendees
- **Format**: Markdown, PDF, DOCX

### 2. Rolling Protocol (Terminserie)
- **Input**: Multiple meetings of same series (e.g., all "Copilot Jour fixe")
- **Output**: Aggregated protocol showing progression over time
- **Features**:
  - Chronological order
  - Track action items across meetings
  - Show decision history
  - Highlight open vs. closed items

### 3. Decision Documentation
- **Extract decisions** from meeting notes (AI-powered)
- **Track decision history** across series
- **Link to related notes** and contexts

## Data Model

### Protocol Metadata
```yaml
protocol_type: single|rolling
generated_at: 2026-01-19T20:00:00Z
meetings_included: 
  - date: 2026-01-15
    file: meetings/2026-01-15-meeting-kickoff-2026.md
series_name: "Copilot Jour fixe"  # for rolling
time_range:
  from: 2026-01-01
  to: 2026-01-31
```

### Meeting Content Structure
```yaml
title: "Meeting Title"
date: 2026-01-15
participants: [name1, name2]
organization: "ORG"
decisions:
  - text: "We decided to use TypeScript"
    context: "Technical Stack Discussion"
    impact: "high"
action_items:
  - task: "Setup CI/CD pipeline"
    assignee: "John Doe"
    due_date: 2026-01-20
    status: open|in-progress|done
notes: |
  Free-form discussion notes...
```

## CLI Commands

### Generate Single Protocol
```bash
flip meeting protocol <meeting-file>
flip meeting protocol meetings/2026-01-15-meeting-kickoff-2026.md
flip meeting protocol meetings/2026-01-15-meeting-kickoff-2026.md --format pdf
flip meeting protocol meetings/2026-01-15-meeting-kickoff-2026.md --output ~/Desktop/
```

### Generate Rolling Protocol
```bash
flip meeting protocol --series "Copilot Jour fixe"
flip meeting protocol --series "Copilot Jour fixe" --from 2026-01-01 --to 2026-01-31
flip meeting protocol --series "Copilot Jour fixe" --write
flip meeting protocol --series "Copilot Jour fixe" --sync
```

`--write` stores the rolling protocol at the canonical series path:

```text
meetings/protocols/<series-slug>-rolling-protocol.md
```

`--sync` updates that canonical file in place, preserves explicitly marked manual sections, and refreshes backlinks from included meeting notes.

### List Available Series
```bash
flip meeting series list
flip meeting series list --json
```

## AI Integration

### Content Extraction

#### 1. Decision Extraction
```
INPUT (Free text from ## Notes section):
"We discussed the tech stack. After evaluating React vs Vue, 
the team agreed to go with React because of better TypeScript 
support and larger community."

OUTPUT:
decision:
  text: "Use React as frontend framework"
  reasoning: "Better TypeScript support, larger community"
  alternatives_considered: ["Vue.js"]
  impact: "high"
  participants: [extracted from context]
```

#### 2. Action Item Extraction
```
INPUT:
"John mentioned he will set up the CI/CD pipeline by Friday.
Sarah volunteered to review the authentication flow."

OUTPUT:
action_items:
  - task: "Set up CI/CD pipeline"
    assignee: "John"
    due_date: 2026-01-20  # (inferred from "Friday")
    status: "open"
  - task: "Review authentication flow"
    assignee: "Sarah"
    status: "open"
```

#### 3. Summary Generation
```
INPUT: Full meeting notes (can be lengthy)

OUTPUT:
summary: |
  The team met to discuss the Q1 roadmap. Key decisions included
  adopting React for the frontend and setting up automated CI/CD.
  Three action items were assigned with deadlines by end of week.
```

### AI Provider Support

#### Local/Open Source (Priority 1)
- **Ollama** integration (local LLM)
- Models: llama3, mistral, codellama
- No API keys needed
- Privacy-preserving

#### Cloud Providers (Priority 2)
- OpenAI (GPT-4)
- Anthropic Claude
- Google Gemini
- Configurable via `.flip.yaml`

#### Configuration
```yaml
# .flip.yaml
ai:
  provider: ollama|openai|anthropic|gemini
  model: llama3|gpt-4|claude-3-sonnet
  endpoint: http://localhost:11434  # for ollama
  api_key: ${OPENAI_API_KEY}  # for cloud providers
  
  # Feature toggles
  features:
    decision_extraction: true
    action_item_extraction: true
    summary_generation: true
    sentiment_analysis: false  # future
```

## Output Formats

### 1. Markdown (Default)
```markdown
# Meeting Protocol: Kickoff 2026

**Date**: 2026-01-15  
**Organization**: IAK  
**Participants**: John Doe, Jane Smith  

## Summary
AI-generated summary here...

## Decisions
1. **Use React as frontend framework**
   - *Reasoning*: Better TypeScript support, larger community
   - *Impact*: High
   - *Alternatives*: Vue.js

## Action Items
- [ ] Set up CI/CD pipeline (@john, due: 2026-01-20)
- [ ] Review authentication flow (@sarah)

## Discussion Notes
[Original notes from ## Notes section]

---
*Protocol generated: 2026-01-19 20:00*
```

### 2. PDF Export
- Template-based (Pandoc Eisvogel or similar)
- Professional formatting
- Company branding support (future)

### 3. DOCX Export
- For sharing with non-technical stakeholders
- Compatible with MS Word / LibreOffice

### 4. HTML
- Self-contained HTML
- Shareable via email/intranet

## Rolling Protocol Format

```markdown
---
protocol_type: rolling
series_name: "Copilot Jour fixe"
generated_at: 2026-01-19T20:00:00Z
meetings_count: 4
time_range:
  from: 2026-01-01
  to: 2026-01-31
---

# Rolling Protocol: Copilot Jour fixe
**Period**: 2026-01-01 to 2026-01-31  
**Meetings**: 4  
**Organization**: ECE  

## Overview
This rolling protocol covers 4 meetings of the "Copilot Jour fixe" series.

## Changes Since Last Meeting

Latest meeting: Session 4 (2026-01-29)

### Information
- Budget risk was raised

### Decisions
- Approved Q1 budget increase

### Actions
- NEW: Hire frontend developer (@mike)
- DONE: Setup Jira board (@john)

## Timeline

### 2026-01-08: Session 1
**Participants**: John, Sarah, Mike

**Decisions**:
- Adopted new sprint cycle (2 weeks)

**Action Items**:
- [x] Setup Jira board (@john) ✅ Completed
- [ ] Document onboarding process (@sarah) ⏳ In Progress

---

### 2026-01-15: Session 2
**Participants**: John, Sarah

**Decisions**:
- Approved Q1 budget increase

**Action Items**:
- [ ] Hire frontend developer (@mike) 🆕 New
- [ ] Document onboarding process (@sarah) ⏳ Carryover from 2026-01-08

---

## Summary

### Information Overview
- Budget risk was raised (last updated: 2026-01-29, seen 2 times, Session 4)

### Action Items Overview

### ✅ Completed (1)
- Setup Jira board (@john, due: 2026-01-10, completed: 2026-01-09)

### ⏳ In Progress (1)
- Document onboarding process (@sarah, assigned: 2026-01-08)

### 🆕 Open (1)
- Hire frontend developer (@mike, assigned: 2026-01-15)

## Decisions Timeline
1. **2026-01-08**: Adopted 2-week sprint cycle
2. **2026-01-15**: Approved Q1 budget increase

## Manual Notes

<!-- flip:manual-notes:start -->
_Add manual notes here. This section is preserved by flip meeting protocol --sync._
<!-- flip:manual-notes:end -->

## Open Questions

<!-- flip:open-questions:start -->
_Track unresolved questions here. This section is preserved by flip meeting protocol --sync._
<!-- flip:open-questions:end -->

## Follow-Ups

<!-- flip:follow-ups:start -->
_Track stakeholder follow-ups here. This section is preserved by flip meeting protocol --sync._
<!-- flip:follow-ups:end -->

---
*Rolling protocol generated: 2026-01-19 20:00*
```

## Sync Semantics

- `flip meeting protocol --series "..." --write` generates the canonical rolling protocol file.
- `flip meeting protocol --series "..." --sync` regenerates the protocol and preserves content inside the `flip:manual-notes`, `flip:open-questions`, and `flip:follow-ups` markers.
- The generated analytical sections are canonical output and may be replaced on each sync.
- Each included meeting note receives or updates a backlink to the canonical rolling protocol in its `Related` section.
- The preserved sections are intended for hand-maintained narrative context, unresolved questions, and stakeholder follow-up tracking.

## Implementation Plan

### Phase 1: Core Protocol Generation (Markdown Only, No AI) ⏳ Current
- [x] Create feature branch
- [x] Design document
- [ ] Implement protocol parser (frontmatter + sections)
- [ ] Implement basic Markdown protocol generator
- [ ] Implement `flip meeting protocol <file>` command
- [ ] Add series detection logic
- [ ] Implement rolling protocol aggregation
- [ ] Tests & documentation

### Phase 2: AI Integration 🔮 Future
- [ ] Design AI extraction interface
- [ ] Implement Ollama integration (local)
- [ ] Decision extraction prompt engineering
- [ ] Action item extraction
- [ ] Summary generation
- [ ] Add OpenAI/Anthropic providers (optional)

### Phase 3: Export Formats 🔮 Future
- [ ] PDF export (via Pandoc)
- [ ] DOCX export
- [ ] HTML export
- [ ] Template customization

### Phase 4: Advanced Features 🔮 Future
- [ ] Company branding support
- [ ] Email integration (send protocol)
- [ ] Diff view (what changed since last meeting)
- [ ] VS Code extension integration (generate from editor)

## File Structure

```
flip/
├── internal/
│   ├── protocols/
│   │   ├── generator.go        # Core protocol generation
│   │   ├── rolling.go          # Rolling protocol logic
│   │   ├── parser.go           # Meeting content parser
│   │   ├── exporter.go         # Export to PDF/DOCX/HTML
│   │   └── ai/
│   │       ├── extractor.go    # AI extraction interface
│   │       ├── ollama.go       # Ollama provider
│   │       ├── openai.go       # OpenAI provider
│   │       └── prompts.go      # Prompt templates
│   └── commands/
│       └── meeting_protocol.go # CLI command implementation
└── templates/
    └── protocols/
        ├── single.md.tmpl      # Single meeting template
        ├── rolling.md.tmpl     # Rolling protocol template
        └── eisvogel-protocol.latex  # PDF template
```

## Testing Strategy

### Unit Tests
- Parse meeting frontmatter
- Extract sections (Notes, Decisions, Action Items)
- Aggregate multiple meetings
- Template rendering

### Integration Tests
- End-to-end protocol generation
- PDF/DOCX export
- AI extraction (with mock LLM)

### Manual Testing
- Generate protocols from real danobrain meetings
- Test series detection with "Copilot Jour fixe"
- Verify PDF quality

## Dependencies

### Required
- `gopkg.in/yaml.v3` (already in use)
- Template engine (Go templates, already available)

### Optional (for export)
- **Pandoc** (external) - for PDF/DOCX conversion
- **wkhtmltopdf** (alternative) - HTML to PDF

### Optional (AI)
- Ollama client library
- OpenAI Go SDK
- Anthropic Go SDK

## Configuration

### Global Config (.flip.yaml)
```yaml
protocols:
  default_format: markdown  # markdown|pdf|docx|html
  output_dir: ~/Documents/Protocols
  
  template:
    single: templates/protocols/single.md.tmpl
    rolling: templates/protocols/rolling.md.tmpl
  
  pdf:
    engine: pandoc  # pandoc|wkhtmltopdf
    template: eisvogel  # for pandoc
  
  ai:
    enabled: true
    provider: ollama
    model: llama3
    endpoint: http://localhost:11434
```

### Per-Meeting Override
```yaml
# In meeting frontmatter
protocol:
  template: custom-protocol-template.md
  include_raw_notes: false
  ai_extract: true
```

## Security & Privacy

### Data Handling
- **Local-first**: All processing happens locally by default
- **No telemetry**: Meeting content never sent to external servers without explicit opt-in
- **AI Provider Choice**: Users control which AI provider to use
  - Ollama (local): Fully private
  - Cloud (OpenAI/etc): User explicitly enables

### Sensitive Content
- Warn if cloud AI enabled for meetings with certain tags/organizations
- Option to exclude sensitive sections from AI processing
- Redaction support for exports

## Future Ideas

### Advanced AI Features
- **Sentiment analysis**: Detect conflicts/concerns in discussions
- **Topic modeling**: Auto-tag meetings by discussed topics
- **Speaker diarization**: Identify who said what (from transcripts)
- **Follow-up suggestions**: "Based on this meeting, you should..."

### Integrations
- **Calendar sync**: Pull meeting info from Google Calendar/Outlook
- **Transcript import**: Parse Zoom/Teams transcripts
- **Slack/Teams notifications**: Post protocol summary
- **Email sending**: Direct protocol distribution

### Analytics
- **Meeting efficiency metrics**: Duration, decisions per meeting, action item completion rate
- **Series health**: Track recurring meeting productivity
- **Decision tracking**: Graph of decisions over time

---

**Status**: Draft  
**Last Updated**: 2026-01-19  
**Author**: Dominik Hattrup, GitHub Copilot
