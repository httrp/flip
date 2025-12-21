# Flip Brain Specification v1.1

> **Status**: Draft  
> **Version**: 1.1  
> **Date**: 2025-12-21  
> **Authors**: Flip Team

---

## 1. Overview

A **Flip Brain** is a directory-based personal knowledge management system designed for:
- **Portability**: Plain-text files, no vendor lock-in
- **Structure**: Clear organization without rigidity
- **Simplicity**: Works with any text editor
- **Power**: Rich task management and metadata support

---

## 2. Directory Structure

### 2.1 Required Files

| Path | Description |
|------|-------------|
| `.flip-brain.yaml` | Brain marker and configuration (REQUIRED) |

A directory is recognized as a Flip Brain if and only if it contains a `.flip-brain.yaml` file.

### 2.2 Recommended Directories

| Directory | Purpose | Content |
|-----------|---------|---------|
| `journal/` | Daily entries | `YYYY-MM-DD.md` files |
| `notes/` | General notes | `{slug}.md` files |

### 2.3 Optional Directories

| Directory | Purpose | Content |
|-----------|---------|---------|
| `meetings/` | Meeting notes | `YYYY-MM-DD-{slug}.md` files |
| `tasks/` | Task files | `todo.{context}.md` or `inbox.md` |
| `definitions/` | Structured metadata | YAML files |
| `templates/` | Note templates | `*-template.md` files |
| `assets/` | Media & attachments | Any files |
| `assets/images/` | Images | Image files |
| `assets/documents/` | Documents | PDF, DOCX, etc. |

### 2.4 Visual Overview

```
my-brain/
├── .flip-brain.yaml          # REQUIRED: Brain marker
│
├── journal/                  # RECOMMENDED
│   ├── 2025-01-15.md
│   └── 2025-01-16.md
│
├── notes/                    # RECOMMENDED
│   ├── project-ideas.md
│   └── learning-go.md
│
├── meetings/                 # OPTIONAL
│   └── 2025-01-15-standup.md
│
├── tasks/                    # OPTIONAL
│   ├── inbox.md
│   ├── todo.work.md
│   └── todo.personal.md
│
├── definitions/              # OPTIONAL
│   ├── contexts.yaml
│   ├── organizations.yaml
│   ├── projects.yaml
│   └── people.yaml
│
├── templates/                # OPTIONAL
│   ├── journal-template.md
│   ├── meeting-template.md
│   └── note-template.md
│
└── assets/                   # OPTIONAL
    ├── images/
    └── documents/
```

---

## 3. File Naming Conventions

### 3.1 General Rules

- Use **lowercase** letters
- Use **hyphens** (`-`) as word separators (not underscores)
- Use **ASCII** characters only (no umlauts, special chars)
- File extension: `.md` for Markdown

### 3.2 Specific Patterns

| File Type | Pattern | Example |
|-----------|---------|---------|
| Journal | `YYYY-MM-DD.md` | `2025-01-15.md` |
| Meeting | `YYYY-MM-DD-{slug}.md` | `2025-01-15-standup.md` |
| Note | `{slug}.md` | `project-ideas.md` |
| Task | `todo.{context}.md` or `inbox.md` | `todo.work.md` |
| Template | `{type}-template.md` | `meeting-template.md` |
| Definition | `{type}.yaml` | `contexts.yaml` |

### 3.3 Date Format

**Always use ISO 8601**: `YYYY-MM-DD`

- ✅ `2025-01-15` (correct)
- ❌ `2025_01_15` (underscores - Logseq style)
- ❌ `15-01-2025` (DD-MM-YYYY)
- ❌ `01-15-2025` (MM-DD-YYYY)

---

## 4. Brain Configuration (.flip-brain.yaml)

### 4.1 Minimal Configuration

```yaml
brain:
  name: "my-brain"
  created: "2025-01-15T10:30:00+01:00"
```

### 4.2 Full Configuration

```yaml
brain:
  name: "my-brain"
  type: "personal"              # personal | work | project
  created: "2025-01-15T10:30:00+01:00"
  version: "1.1"

config:
  author: "Your Name"
  default_organization: "PERSONAL"
  
  # Task settings
  task_format: "flip"           # flip | logseq | dataview | emoji
  
  # Link settings
  wikilinks: false              # Enable [[Note]] syntax
  
  # Supported formats beyond markdown
  supported_formats:
    - md
    - excalidraw
    - drawio

# Directory customization (advanced)
directories:
  journal: "journal"
  notes: "notes"
  meetings: "meetings"
  tasks: "tasks"
  templates: "templates"
  assets: "assets"
```

### 4.3 Field Reference

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `brain.name` | Yes | string | Human-readable brain name |
| `brain.created` | Yes | ISO 8601 | Creation timestamp |
| `brain.type` | No | enum | `personal`, `work`, `project` |
| `brain.version` | No | string | Spec version (default: "1.1") |
| `config.author` | No | string | Default author for notes |
| `config.default_organization` | No | string | Default org abbreviation |
| `config.task_format` | No | enum | Preferred task format |
| `config.wikilinks` | No | bool | Enable wikilink support |

---

## 5. Markdown Format

### 5.1 Frontmatter (YAML Header)

All notes SHOULD have a YAML frontmatter:

```markdown
---
title: "Note Title"
date: 2025-01-15
tags: [tag1, tag2]
type: note
---

# Note Title

Content here...
```

### 5.2 Required Frontmatter Fields

| Field | Required | Type | Description |
|-------|----------|------|-------------|
| `title` | Yes | string | Note title |
| `date` | Yes | date | Creation date (YYYY-MM-DD) |

### 5.3 Optional Frontmatter Fields

| Field | Type | Description |
|-------|------|-------------|
| `tags` | array | Categorization tags |
| `type` | enum | `note`, `meeting`, `journal`, `task` |
| `status` | enum | `active`, `archived`, `draft` |
| `organization` | string | Organization abbreviation |
| `project` | string | Project name |
| `related` | array | Links to related notes |

### 5.4 Links

**Primary format** (recommended):
```markdown
[Display Text](relative/path/to/note.md)
```

**Alternative format** (if `config.wikilinks: true`):
```markdown
[[Note Name]]
[[Note Name|Display Text]]
```

### 5.5 Images and Assets

```markdown
![Alt text](../assets/images/diagram.png)
[Download PDF](../assets/documents/spec.pdf)
```

---

## 6. Task Format (Canonical)

### 6.1 Task Structure

```markdown
- [ ] Task title
  created: 2025-01-15
  due: 2025-01-20
  priority: high
  status: open
  context: work
  organization: DAN
  project: flip
  tags: [backend, urgent]
```

### 6.2 Task States

| Checkbox | Status | Description |
|----------|--------|-------------|
| `- [ ]` | open | Not started |
| `- [/]` | in-progress | Currently working |
| `- [x]` | done | Completed |
| `- [-]` | cancelled | Cancelled/dropped |

### 6.3 Task Metadata Fields

| Field | Type | Description |
|-------|------|-------------|
| `created` | date | When task was created |
| `due` | date | Due date |
| `priority` | enum | `low`, `medium`, `high`, `urgent` |
| `status` | enum | `open`, `in-progress`, `blocked`, `done`, `cancelled` |
| `context` | string | Context (e.g., `work`, `home`) |
| `organization` | string | Organization abbreviation |
| `project` | string | Project name |
| `tags` | array | Additional tags |
| `assigned` | string | Person responsible |
| `recurring` | string | Recurrence pattern |

### 6.4 Legacy Format Support (Read-Only)

The parser also supports these formats for **reading** (not writing):

**Dataview style:**
```markdown
- [ ] Task
  due:: 2025-01-20
  priority:: high
```

**Emoji style:**
```markdown
- [ ] Task 📅 2025-01-20 ⏫ #work
```

**Logseq style:**
```markdown
- TODO Task
- DONE Completed task
```

---

## 7. Definitions (YAML)

### 7.1 Contexts

```yaml
# definitions/contexts.yaml
contexts:
  - name: "work"
    description: "Work-related tasks"
    color: "#3B82F6"
  - name: "personal"
    description: "Personal tasks"
    color: "#10B981"
```

### 7.2 Organizations

```yaml
# definitions/organizations.yaml
organizations:
  - name: "Danorama"
    abbreviation: "DAN"
    description: "Main company"
    type: "employer"
```

### 7.3 Projects

```yaml
# definitions/projects.yaml
projects:
  - name: "Flip Development"
    abbreviation: "FLIP"
    organization: "DAN"
    status: "active"
```

### 7.4 People

```yaml
# definitions/people.yaml
people:
  - name: "John Doe"
    abbreviation: "JD"
    email: "john@example.com"
    organization: "DAN"
```

---

## 8. Templates

### 8.1 Template Philosophy

> **"Metadata-Rich, Content-Minimal"**

Templates should:
- ✅ Offer all possible metadata fields in the header
- ✅ Provide minimal content structure
- ❌ NOT include sections the user must delete

### 8.2 Template Variables

Templates use Go `text/template` syntax:

| Variable | Description |
|----------|-------------|
| `{{.Title}}` | Note title |
| `{{.Date}}` | Current date (YYYY-MM-DD) |
| `{{.DateTime}}` | Full timestamp |
| `{{.Tags}}` | Comma-separated tags |
| `{{.Organization}}` | Default organization |
| `{{.Author}}` | Author name |

### 8.3 Example Templates

**Journal Template:**
```markdown
---
title: "Journal {{.Date}}"
date: {{.Date}}
type: journal
mood: 
energy: 
tags: []
---

# {{.Date}}

```

**Meeting Template:**
```markdown
---
title: "{{.Title}}"
date: {{.Date}}
type: meeting
attendees: []
organization: {{.Organization}}
project: 
tags: []
decisions: []
action_items: []
---

# {{.Title}}

```

**Note Template:**
```markdown
---
title: "{{.Title}}"
date: {{.Date}}
type: note
tags: []
status: active
related: []
---

# {{.Title}}

```

---

## 9. Compatibility

### 9.1 Tool Compatibility

| Tool | Compatibility | Notes |
|------|--------------|-------|
| **VS Code** | ✅ Full | Native markdown support |
| **Obsidian** | ⚠️ Partial | Wikilinks need conversion |
| **Logseq** | ⚠️ Partial | Different task format |
| **Git** | ✅ Full | Plain text, small files |
| **Grep/Ripgrep** | ✅ Full | Searchable text |

### 9.2 Migration Paths

From Flip Brain to:
- **Obsidian**: Convert links `[](path.md)` → `[[Note]]`
- **Logseq**: Convert dates `2025-01-15` → `2025_01_15`

To Flip Brain from:
- **Obsidian**: Convert links `[[Note]]` → `[Note](notes/note.md)`
- **Logseq**: Convert dates `2025_01_15` → `2025-01-15`

---

## 10. Future Extensions

### 10.1 Multi-Format Support

Beyond Markdown, Flip Brains MAY contain:

| Extension | Tool | Type |
|-----------|------|------|
| `.excalidraw` | Excalidraw | Drawings |
| `.drawio` | Draw.io | Diagrams |
| `.tldraw` | tldraw | Whiteboard |
| `.canvas` | Obsidian | Mind maps |

These files are:
- ✅ Listed in brain contents
- ✅ Validated in link checks
- ❌ NOT parsed for content
- ❌ NOT searchable by flip

### 10.2 Schema System

Brains MAY include schemas for validation:

```
definitions/
└── schemas/
    ├── note.yaml
    ├── meeting.yaml
    └── task.yaml
```

---

## Appendix A: Quick Reference

### File Patterns
```
Journal:  journal/YYYY-MM-DD.md
Meeting:  meetings/YYYY-MM-DD-{slug}.md
Note:     notes/{slug}.md
Task:     tasks/todo.{context}.md
```

### Task Quick Format
```markdown
- [ ] Task title
  due: 2025-01-20
  priority: high
```

### Minimal Brain
```
my-brain/
├── .flip-brain.yaml    # Required
└── notes/              # Recommended
    └── first-note.md
```

---

*Flip Brain Specification v1.1 - December 2025*
