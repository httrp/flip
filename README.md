# flip
steroids for my 2nd brain

## Overview

Flip is an intelligent assistant designed to supercharge personal knowledge management workflows.

Flip doesn't store your notes and tasks directly, but connects to your external 2nd brain.

## Scope

Use Flip to

- Create your new 2nd brain (structure and templates for notes and tasks)
- Track your tasks
- Summarize your notes
- Query your 2nd brain using natural language

## Tech

- **Primary**: Flip is primarily built in Golang
- **Format Agnostic**: Flip supports Markdown, plain text, and structured formats
- **Multi-Source**: You can connect to multiple knowledge repositories simultaneously (e.g. different folders for private and business notes)
- **LLM-Integration** Flip supports multiple (local) LLMs to query your 2nd brain using natural language

## Standards & Formats

### Note Types

#### Journal Notes
Journal notes follow the format `YYYY-MM-DD.md` and contain:

```markdown
# 2025-08-12

## General Thoughts
- 

## Tasks
- [ ] 

## Notes
- 

```

#### Meeting Notes
Meeting notes use the format `YYYY-MM-DD-[ORG]-meeting-title.md` or `YYYY-MM-DD-[ORG]-[CTX]-meeting-title.md`:

```markdown
# Meeting: [Title] - 2025-08-12

**Date:** 2025-08-12
**Time:** [HH:MM - HH:MM]
**Organization:** [ORG] - [[Organization Name]]
**Context:** [CTX] - [[Context Name]] _(optional)_
**Participants:** [PER1], [PER2], [[External Person]]
**Type:** [standup|planning|review|retrospective|other]

## Agenda
- 
- 

## Discussion
- 

## Decisions
- 

## Action Items
- [ ] [Task] - @[PER1] - [ORG]/[CTX] - due: YYYY-MM-DD
- [ ] [Task] - @[PER2] - [ORG] - due: YYYY-MM-DD

## Next Steps
- 
```

#### General Notes
General notes follow a flexible structure with mandatory frontmatter:

```markdown
---
title: "Note Title"
date: 2025-08-12
tags: [tag1, tag2, tag3]
type: note
status: active
---

# Note Title

## Summary
Brief summary of the note content.

## Content
Main note content here.

## Links
- [[Related Note 1]]
- [[Related Note 2]]
- [External Link](https://example.com)

## References
- 
```

### Task Management

#### Task Format
Tasks use standardized markdown checkboxes with metadata:

```markdown
- [ ] Task description [ORG] #tag1 #tag2 @[PER] due:2025-08-15 priority:high
- [ ] Context task [ORG]/[CTX] @[PER] due:2025-08-15 context:meeting
- [x] Completed task [ORG] ✅ 2025-08-12
- [~] Cancelled task [ORG]/[CTX] ❌ 2025-08-12
```

#### Task Metadata
- **Organization:** `[ORG]` - Required organizational context
- **Context:** `[ORG]/[CTX]` - Optional context or area
- **Tags:** `#work #personal #urgent`
- **Assignee:** `@[PER]` or `@[[Person Name]]` for external people
- **Due Date:** `due:YYYY-MM-DD`
- **Priority:** `priority:high|medium|low`
- **Estimate:** `est:2h` or `est:30m`
- **Context:** `context:meeting|email|call|idea`

#### Task States
- `[ ]` - Open/Todo
- `[x]` - Completed
- `[~]` - Cancelled  
- `[>]` - Forwarded/Rescheduled
- `[!]` - Important/Urgent
- `[?]` - Question/Needs clarification
- `[-]` - In Progress

### Definition Management

Flip manages all recurring entities in dedicated files within each repository:

#### Organizations (`definitions/organizations.yaml`)
```yaml
organizations:
  WORK:
    name: "Work"
    type: "professional"
    description: "Professional work and career"
    color: "#4ECDC4"
  PERSONAL:
    name: "Personal"
    type: "personal" 
    description: "Personal life and activities"
    color: "#45B7D1"
  LEARNING:
    name: "Learning"
    type: "development"
    description: "Learning and skill development"
    color: "#96CEB4"
```

#### Contexts (`definitions/contexts.yaml`)
```yaml
contexts:
  WORK:
    MEETINGS:
      name: "Meetings & Collaboration"
      status: "active"
      description: "Team meetings and collaborative work"
    PROJECTS:
      name: "Active Projects"
      status: "active"
      description: "Current work projects and deliverables"
  PERSONAL:
    HEALTH:
      name: "Health & Fitness"
      status: "ongoing"
      description: "Health and fitness related activities"
    FINANCE:
      name: "Personal Finance"
      status: "ongoing"
      description: "Financial planning and management"
  LEARNING:
    TECH:
      name: "Technology Learning"
      status: "active"
      description: "Learning new technologies and tools"
    LANGUAGES:
      name: "Language Learning"
      status: "active" 
      description: "Learning new languages"
```

#### People (`definitions/people.yaml`)
```yaml
people:
  SELF:
    name: "Your Name"
    organization: "WORK"
    role: "Your Role"
    email: "your.email@example.com"
  JS:
    name: "Jane Smith"
    organization: "WORK"
    role: "Colleague"
    email: "jane.smith@company.com"
  DR_MUELLER:
    name: "Dr. Mueller"
    organization: "PERSONAL"
    role: "Doctor"
    phone: "+49-123-456789"
```

### Usage Examples

#### Meeting Note Examples
```
2025-08-12-WORK-weekly-standup.md          # Work organization meeting
2025-08-12-WORK-PROJECTS-design-review.md  # Work projects context meeting
2025-08-12-PERSONAL-doctor-appointment.md  # Personal meeting
```

#### Task Examples
```markdown
- [ ] Review wireframes [WORK]/[PROJECTS] @JS due:2025-08-15 priority:high
- [ ] Prepare presentation [WORK] @SELF due:2025-08-14 context:meeting
- [ ] Call dentist [PERSONAL]/[HEALTH] @SELF due:2025-08-16 priority:low
- [ ] Study Go patterns [LEARNING]/[TECH] @SELF due:2025-08-13 est:1h
```

### Folder Structure

```
📁 your-second-brain/
├── 📁 definitions/
│   ├── organizations.yaml
│   ├── contexts.yaml
│   └── people.yaml
├── 📁 journal/
│   ├── 2025-08-12.md
│   ├── 2025-08-13.md
│   └── ...
├── 📁 meetings/
│   ├── 2025-08-12-WORK-weekly-standup.md
│   ├── 2025-08-12-WORK-PROJECTS-design-review.md
│   └── ...
├── 📁 notes/
│   ├── WORK-processes.md
│   ├── PERSONAL-goals.md
│   └── ...
├── 📁 tasks/
│   ├── inbox.md
│   ├── WORK-tasks.md
│   ├── PERSONAL-tasks.md
│   └── LEARNING-tasks.md
├── 📁 templates/
│   ├── journal-template.md
│   ├── meeting-template.md
│   └── note-template.md
└── 📁 assets/
    ├── images/
    └── documents/
```

### Linking & References

#### Internal Links
- `[[Note Title]]` - Link to another note
- `[[Note Title#Section]]` - Link to specific section
- `[[Note Title|Display Text]]` - Link with custom display text

#### Tags
- `#tag` - Simple tag
- `#category/subcategory` - Hierarchical tags
- `#project/alpha` - Project-specific tags

#### People
- `@[PER]` - Person reference using abbreviation
- `@[[Person Name]]` - Formal person link for external people
- Contact info stored in people.yaml

### Compatibility

These standards maintain compatibility with:
- **Logseq:** Block references, page links, task states
- **Obsidian:** Wikilinks, tags, frontmatter
- **Dendron:** Hierarchical structure, schemas
- **Standard Markdown:** Pure markdown fallback
- **Git:** Version control friendly file formats

### Configuration

Flip uses a `.flip.yaml` configuration file:

```yaml
# .flip.yaml
repositories:
  - name: "personal"
    path: "/path/to/personal/notes"
    type: "primary"
  - name: "work"
    path: "/path/to/work/notes"
    type: "secondary"

definitions:
  auto_resolve: true
  validate_on_create: true
  suggest_missing: true

templates:
  journal: "templates/journal-template.md"
  meeting: "templates/meeting-template.md"
  note: "templates/note-template.md"

defaults:
  author: "Your Name"
  timezone: "Europe/Berlin"
  date_format: "2006-01-02"
  default_organization: "WORK"

llm:
  provider: "ollama"
  model: "llama2"
  endpoint: "http://localhost:11434"
```