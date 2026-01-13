# Flip Brain Standard - Naming Conventions & Conventions

**Version:** 1.1  
**Last Updated:** 2026-01-13  
**Status:** Official Standard for Flip Native Brains

---

## Table of Contents

1. [Overview](#overview)
2. [Directory Structure](#directory-structure)
3. [Naming Conventions](#naming-conventions)
4. [Metadata Format](#metadata-format)
5. [Link Style](#link-style)
6. [Asset Organization](#asset-organization)
7. [Rationale & Design Decisions](#rationale--design-decisions)

---

## Overview

The **Flip Standard** defines conventions for native Flip brains (marked by `.flip-brain.yaml`). These conventions are designed for:

- **Consistency**: Predictable file organization across all brains
- **Portability**: Easy migration to other brain types (Obsidian, Logseq, Dendron, etc.)
- **Searchability**: Clear, readable filenames with metadata context
- **Automation**: CLI commands can reliably detect and process files

---

## Directory Structure

```
brain-root/
├── .flip-brain.yaml              # Brain marker with metadata
├── .flip.yaml                    # Optional local configuration
├── journal/                      # Daily journal entries (one file per day)
│   ├── 2026-01-12.md
│   ├── 2026-01-13.md
│   └── 2026-01-14.md
├── meetings/                     # Meeting notes with participants
│   ├── 2026-01-12-meeting-kickoff-ece.md
│   ├── 2026-01-09-meeting-team-standup.md
│   └── 2025-11-19-meeting-project-review.md
├── notes/                        # General notes and knowledge base
│   ├── 2026-01-07-business-development.md
│   ├── 2025-12-11-bonusprogramm.md
│   └── 2026-01-10-ideal-customer-profile.md
├── tasks/                        # Task management
│   ├── todo.inbox.md
│   ├── todo.active.md
│   ├── todo.projects.md
│   └── todo.archive.md
├── templates/                    # Note templates (for creation)
│   ├── note-template.md
│   ├── journal-template.md
│   ├── meeting-template.md
│   └── task-template.md
├── definitions/                  # Organizational metadata (YAML)
│   ├── organizations.yaml        # Companies, organizations, teams
│   ├── people.yaml              # Contacts and team members
│   ├── projects.yaml            # Project definitions
│   ├── contexts.yaml            # Work contexts/areas
│   └── schemas/                 # Custom field schemas (optional)
└── assets/                       # Media and attachments (centralized)
    ├── images/
    │   ├── screenshots/
    │   └── diagrams/
    └── documents/
        ├── pdfs/
        └── archives/
```

---

## Naming Conventions

### General Rules

1. **Case**: Always **lowercase**
2. **Separators**: **Kebab-case** (hyphens, not underscores or spaces)
3. **No Redundancy**: Don't repeat metadata that's already in frontmatter

### By File Type

#### Journal Files

**Pattern:** `YYYY-MM-DD.md`  
**Location:** `journal/`  
**Example:** `2026-01-13.md`

**Rationale:**
- Date is the primary identifier for journals
- Single daily entry = filename is date
- Sortable alphabetically = chronological order
- No title needed (implicit in date)

#### Meeting Notes

**Pattern:** `YYYY-MM-DD-meeting-{slug-title}.md`  
**Location:** `meetings/`  
**Examples:**
- `2026-01-12-meeting-kickoff-ece-copilot.md`
- `2026-01-09-meeting-team-standup.md`
- `2025-11-19-meeting-project-review.md`

**Rationale:**
- Date first = chronological sorting
- `meeting-` prefix = distinguishes from other files with dates
- Slug title = human-readable without being too long
- All lowercase = compatibility across OS
- **NO Datums-Präfix for Logseq pages** (only journals need dates)

#### General Notes

**Pattern:** `{slug-title}.md`  
**Location:** `notes/`  
**Examples:**
- `business-development.md`
- `bonusprogramm.md`
- `ideal-customer-profile.md`

**Rationale:**
- Title only = clean, readable filenames
- Creation date in frontmatter = metadata, not filename
- Slug-case title = clear, concise
- Lowercase = consistent with all other files
- Hyphen separators = standard web convention
- No redundancy = avoid duplicating metadata

### Note Creation & Journal Linking

When creating a new note with `flip note`, you will be prompted:
```
Add link to today's journal?
├─ Add link (default)
└─ Skip
```

**Default behavior:** Link is added automatically unless you select "Skip"

#### What happens:
1. Note is created in `notes/{title}.md`
2. System checks if `journal/YYYY-MM-DD.md` exists for today
3. If journal doesn't exist, it's created with template
4. Note link is added to journal's "## Activities" section with emoji prefix
5. Link format depends on brain type:
   - **Flip/Foam:** Markdown link `[title](notes/title.md)`
   - **Logseq:** Wikilink `[[title]]`
   - **Obsidian:** Wikilink `[[title]]`

#### When to skip journal links:
- Bulk imports or template notes
- Reference material without temporal context
- Notes created for organizational purposes (not daily work)

#### Example journal entry:
```markdown
# 2026-01-13 - Monday

## Activities

📝 [Business Development Strategy](notes/business-development-strategy.md)
📝 [Q1 Planning Session](notes/q1-planning.md)
```

#### Health check:
The health check (`flip brain check`) will:
- ℹ️ Inform about recently created notes not mentioned in any journal
- Not flag as error (journal links are optional)
- Only check Flip-native brains (not Logseq/Obsidian)

#### Task Files

**Pattern:** `todo.{category}.md`  
**Location:** `tasks/`  
**Examples:**
- `todo.inbox.md` - New tasks to be triaged
- `todo.active.md` - Currently working tasks
- `todo.projects.md` - Project-level tasks
- `todo.archive.md` - Completed tasks

**Rationale:**
- `todo.` prefix = easy to search/filter
- Category clear = helps with task management workflow

---

## Metadata Format

### Frontmatter (YAML)

All files use YAML frontmatter at the top:

```yaml
---
title: "Display Title (Can Have Capitals and Spaces)"
type: note|journal|meeting|task
created: YYYY-MM-DD
updated: YYYY-MM-DD
author: Name
tags: [tag1, tag2, tag3]
---
```

### Common Fields

| Field | Type | Required | Purpose |
|-------|------|----------|---------|
| `title` | string | ✅ | Human-readable title |
| `type` | string | ✅ | Content type (note, journal, meeting, task, exercise) |
| `created` | date | ✅ | Creation date |
| `updated` | date | ✅ | Last modification |
| `author` | string | ✅ | Author name |
| `tags` | array | ❌ | Categorization tags |
| `organization` | string | ❌ | Associated organization |
| `context` | string | ❌ | Work context/area |

### Meeting-Specific Fields

```yaml
---
title: "Meeting Title"
type: meeting
date: YYYY-MM-DD
time: HH:MM
duration: {minutes}
participants: [name1, name2, name3]
organization: "Organization Name"
series: "Series Name" (optional, if recurring)
---
```

### Note-Specific Fields

```yaml
---
title: "Note Title"
type: note
created: YYYY-MM-DD
tags: [topic1, topic2]
related: [[other-note-1], [other-note-2]]
---
```

---

## Link Style

### Wikilinks (Preferred)

**Format:** `[[note-title]]`  
**Usage:** Preferred within Flip brains  
**Example:** `[[business-development]]`

**Benefits:**
- Auto-complete support
- Bidirectional linking
- Graph visualization friendly
- Clean syntax

### Markdown Links (Fallback)

**Format:** `[display text](../path/to/file.md)`  
**Usage:** For external links or when filename needed  
**Example:** `[Read more](../notes/related-note.md)`

### Block References

**Format:** `[[file-name#section-name]]`  
**Usage:** Link to specific sections  
**Example:** `[[meeting-kickoff-ece#decisions]]`

---

## Asset Organization

### Centralized Assets Directory

```
assets/
├── images/
│   ├── screenshots/     # App screenshots, UI captures
│   ├── diagrams/        # Charts, graphs, drawings
│   └── photos/          # Photographs and visual media
└── documents/
    ├── pdfs/            # PDF files
    ├── spreadsheets/    # Excel, CSV
    └── archives/        # ZIP, compressed files
```

### Asset Naming

**Pattern:** `{type}_{description}_{date}.{ext}`  
**Examples:**
- `screenshot_logseq-dashboard_2026-01-13.png`
- `diagram_project-timeline_2025-12-11.svg`
- `document_contract-2025_2025-11-01.pdf`

**Rationale:**
- Type first = easy filtering
- Date last = version tracking
- Lowercase = consistency
- Underscores (not hyphens) = common for assets

### Referencing Assets

**In Notes:**
```markdown
![Alt text](../assets/images/screenshot_logseq-dashboard_2026-01-13.png)

Or with wikilinks:
[[assets/images/screenshot_logseq-dashboard_2026-01-13.png]]
```

**Best Practice:**
- Use relative paths (`../assets/...`)
- Always include alt text for images
- Keep assets organized by type

---

## Rationale & Design Decisions

### Why Lowercase + Kebab-Case?

**Decision:** All filenames in lowercase, separated by hyphens

**Rationale:**
1. **Cross-Platform**: Avoids case-sensitivity issues (Windows vs Linux)
2. **Web Standard**: URLs use lowercase (consistency if published)
3. **Searchability**: Easier to type and remember
4. **Git Friendly**: Fewer conflicts in version control
5. **Readability**: Hyphens are clearer than underscores or camelCase

### Why Date Prefix for Notes?

**Decision:** Include creation date in note filenames

**Rationale:**
1. **Chronological Sorting**: Files naturally sort by creation date
2. **Version Tracking**: Easy to see when notes were created
3. **Context**: Date provides temporal context
4. **Portability**: Date-based naming is standard across knowledge systems

### Why Separate Meetings Directory?

**Decision:** Dedicated `meetings/` folder instead of mixed with notes

**Rationale:**
1. **Distinct Type**: Meetings have unique metadata (participants, time)
2. **Findability**: Easy to locate all meeting notes
3. **Workflow**: Different creation process than general notes
4. **Portability**: Logical structure for migration to other systems

### Why Centralized Assets?

**Decision:** All assets in one `assets/` folder (vs. co-located near notes)

**Rationale:**
1. **Organization**: Media library is clear and managed
2. **Reusability**: Multiple notes can reference same asset
3. **Backup**: Easier to backup/sync media separately
4. **Cleanup**: Simple to identify unused assets
5. **Performance**: Large asset folders don't clutter individual notes

### Why YAML Frontmatter?

**Decision:** All content uses YAML frontmatter metadata

**Rationale:**
1. **Flexibility**: Easy to add custom fields
2. **Machine-Readable**: CLI can parse and act on metadata
3. **Standard**: Widely supported (Hugo, Jekyll, Obsidian)
4. **Human-Readable**: Clear key-value pairs
5. **Portable**: Can extract metadata when migrating

---

## Special Cases & Exceptions

### Logseq Brain (log)

**Deviations from Standard:**
- **Pages**: Lowercase kebab-case WITHOUT date prefix (`foam.md` not `2026-01-13-foam.md`)
- **Journals**: `YYYY_MM_DD.md` (underscores, Logseq standard)
- **Reason**: Logseq has different conventions; must respect native format

### Dendron Brain

**Deviations from Standard:**
- **Notes**: Dot-notation (`project.ideas.2025.md` instead of `project-ideas-2025.md`)
- **Reason**: Dendron's hierarchical naming is fundamental to the system

### Migration Between Types

When migrating between brain types, use the transformer:

```bash
flip brain migrate --from flip --to obsidian --source /path/to/flip-brain
```

The CLI automatically converts:
- `YYYY-MM-DD-title.md` → brain-specific naming
- Wikilinks → markdown links (if needed)
- Metadata format → target system format

---

## Enforcement & Health Checks

### Health Check Detects

The `flip brain health` command checks:

✅ **Filename conventions:**
- Files in correct directories
- Proper naming patterns
- No mixed case
- No problematic spaces

✅ **Broken links:**
- Wikilinks to non-existent files
- Invalid markdown links

✅ **Missing assets:**
- Referenced images that don't exist
- Broken asset paths

✅ **Orphaned files:**
- Files not referenced anywhere
- Potentially unused notes

### Auto-Repair

The `flip brain repair` command can auto-fix:

```bash
flip brain repair --path /path/to/brain --dry-run  # Preview changes
flip brain repair --path /path/to/brain --apply    # Apply changes
```

---

## Summary: The Flip Standard

| Aspect | Standard | Reason |
|--------|----------|--------|
| **Case** | Lowercase only | Cross-platform compatibility |
| **Separators** | Kebab-case (hyphens) | Web standard, readable |
| **Journal** | `YYYY-MM-DD.md` | Date is primary ID |
| **Meetings** | `YYYY-MM-DD-meeting-title.md` | Distinguishable, temporal reference |
| **Notes** | `title.md` | Metadata in frontmatter, not filename |
| **Metadata** | YAML frontmatter | Flexible, machine-readable |
| **Links** | Wikilinks preferred | Auto-complete, bidirectional |
| **Assets** | Centralized `assets/` | Organization, reusability |

---

## Version History

- **v1.1** (2026-01-13): Added Logseq page naming clarification, health check details
- **v1.0** (2026-01-12): Initial standard definition
