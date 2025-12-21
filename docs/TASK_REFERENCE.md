# Flip Task System Reference

> Complete documentation of the Flip task format, features, and workflows.

---

## 1. Task Format Overview

### 1.1 Canonical Format (v1.1)

This is the **recommended** format for new tasks:

```markdown
- [ ] Task title
  created: 2025-12-21
  due: 2025-01-15
  priority: high
  status: open
  context: work
  organization: DAN
  project: flip
  tags: [backend, urgent]
```

**Key characteristics:**
- Standard markdown checkbox `- [ ]`
- YAML-like indented metadata (2 spaces)
- No `::` suffix (cleaner than Dataview)
- No emojis (easier to type)

### 1.2 Supported Legacy Formats

The parser **reads** these formats for backward compatibility:

#### Dataview Style
```markdown
- [ ] Task title
  due:: 2025-01-15
  priority:: high
```

#### Emoji Style
```markdown
- [ ] Task title 📅 2025-01-15 ⏫ #work #urgent
```

#### Logseq Style
```markdown
- TODO Task title
- DOING In progress task
- DONE Completed task
- LATER Deferred task
```

#### Simple Markdown
```markdown
- [ ] Task title
- [x] Completed task
```

---

## 2. Task States

### 2.1 Checkbox States

| Checkbox | State | Description |
|----------|-------|-------------|
| `- [ ]` | Open | Not started |
| `- [/]` | In Progress | Currently working on |
| `- [x]` | Done | Completed |
| `- [-]` | Cancelled | Dropped/cancelled |

### 2.2 Status Field

The `status` field provides more granularity:

| Status | Description |
|--------|-------------|
| `open` | Not started (default) |
| `in-progress` | Currently working on |
| `blocked` | Waiting on something |
| `done` | Completed |
| `cancelled` | Dropped |

### 2.3 Status Flow

```
open → in-progress → done
         ↓
       blocked → in-progress → done
         ↓
       cancelled
```

---

## 3. Metadata Fields

### 3.1 Core Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `created` | date | Auto | When task was created |
| `due` | date | No | Due date |
| `priority` | enum | No | Task priority |
| `status` | enum | No | Current status |

### 3.2 Organization Fields

| Field | Type | Description |
|-------|------|-------------|
| `context` | string | Context (work, home, errands) |
| `organization` | string | Organization abbreviation |
| `project` | string | Project name |
| `assigned` | string | Person responsible |

### 3.3 Additional Fields

| Field | Type | Description |
|-------|------|-------------|
| `tags` | array | Additional categorization |
| `recurring` | string | Recurrence pattern |
| `estimate` | string | Time estimate (e.g., "2h") |
| `actual` | string | Actual time spent |
| `notes` | string | Additional notes |
| `blocked_by` | string | What's blocking this task |

---

## 4. Priority Levels

| Priority | Description | Emoji (legacy) |
|----------|-------------|----------------|
| `urgent` | Do immediately | 🔴 |
| `high` | Do today | ⏫ |
| `medium` | Do this week | 🔼 |
| `low` | Do eventually | 🔽 |

---

## 5. Task Files

### 5.1 File Organization

```
tasks/
├── inbox.md           # Unsorted tasks (capture)
├── todo.work.md       # Work tasks
├── todo.personal.md   # Personal tasks
├── todo.someday.md    # Someday/Maybe
└── archive/
    └── 2025-01.md     # Archived completed tasks
```

### 5.2 Task File Structure

```markdown
# Work Tasks

## Active

- [ ] Important task
  due: 2025-01-15
  priority: high

- [/] In progress task
  status: in-progress

## Completed

- [x] Done task
  completed: 2025-01-10
```

---

## 6. Task Browser Features

The `flip task browse` command provides:

### 6.1 Views

| View | Description |
|------|-------------|
| All | All tasks across all files |
| Today | Tasks due today |
| This Week | Tasks due this week |
| Overdue | Past due tasks |
| By Context | Grouped by context |
| By Project | Grouped by project |
| By Priority | Grouped by priority |

### 6.2 Filters

```bash
flip task browse                    # All tasks
flip task browse --due today        # Due today
flip task browse --due this-week    # Due this week
flip task browse --overdue          # Overdue
flip task browse --context work     # Work context
flip task browse --project flip     # Specific project
flip task browse --priority high    # High priority only
flip task browse --status open      # Open tasks only
```

### 6.3 Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | View/edit task |
| `Space` | Toggle done/open |
| `i` | Mark in-progress |
| `d` | Set due date |
| `p` | Change priority |
| `c` | Change context |
| `t` | Add tags |
| `/` | Search |
| `n` | New task |
| `a` | Archive completed |
| `?` | Help |
| `q` | Quit |

---

## 7. Task Workflows

### 7.1 Quick Capture

```bash
# Add task from command line
flip task add "Call dentist" --due tomorrow --context personal

# Add to inbox for later sorting
flip task add "Research topic X"
```

### 7.2 Morning Review

```bash
# See what's due today
flip task browse --due today

# Check overdue items
flip task browse --overdue
```

### 7.3 Weekly Review

```bash
# All tasks by project
flip task browse --group project

# Archive completed tasks
flip task archive
```

### 7.4 Context Switching

```bash
# At work
flip task browse --context work

# At home
flip task browse --context personal

# Running errands
flip task browse --context errands
```

---

## 8. Integration with Notes

### 8.1 Tasks in Meeting Notes

```markdown
---
title: "Sprint Planning"
date: 2025-01-15
type: meeting
action_items: []
---

# Sprint Planning

## Notes
- Discussed feature X

## Action Items

- [ ] Implement feature X
  due: 2025-01-20
  assigned: @john
  project: webapp
```

### 8.2 Tasks in Journal Entries

```markdown
---
title: "Journal 2025-01-15"
date: 2025-01-15
type: journal
---

# 2025-01-15

## Tasks for today
- [ ] Finish report
  due: 2025-01-15
  priority: high
```

### 8.3 Task Collection

All tasks from all files are collected by the task browser.

---

## 9. Definitions Integration

### 9.1 Using Organizations

```yaml
# definitions/organizations.yaml
organizations:
  - name: "Danorama"
    abbreviation: "DAN"
```

```markdown
- [ ] Submit report
  organization: DAN
```

### 9.2 Using Projects

```yaml
# definitions/projects.yaml
projects:
  - name: "Flip Development"
    abbreviation: "FLIP"
    organization: "DAN"
```

```markdown
- [ ] Fix bug #123
  project: FLIP
```

### 9.3 Using Contexts

```yaml
# definitions/contexts.yaml
contexts:
  - name: "work"
    description: "Work tasks"
```

```markdown
- [ ] Review PR
  context: work
```

---

## 10. Best Practices

### 10.1 Task Titles

- ✅ Start with a **verb**: "Call dentist", "Review document"
- ✅ Be **specific**: "Email John about project deadline"
- ❌ Avoid vague titles: "Stuff", "Things to do"

### 10.2 Due Dates

- Set due dates only when there's a **real deadline**
- Use `priority: high` for urgent tasks without hard deadlines
- Review and adjust due dates in weekly review

### 10.3 Context Usage

- Use contexts for **where** or **when** you can do the task
- Common contexts: `work`, `home`, `errands`, `computer`, `phone`
- Keep the number of contexts small (5-7 max)

### 10.4 Projects vs Tags

- **Projects**: Larger efforts with multiple tasks
- **Tags**: Cross-cutting concerns or labels
- Example: `project: website-redesign`, `tags: [frontend, urgent]`

---

## Appendix: Format Conversion

### From Dataview to Canonical

```markdown
# Before (Dataview)
- [ ] Task
  due:: 2025-01-15
  priority:: high

# After (Canonical)
- [ ] Task
  due: 2025-01-15
  priority: high
```

### From Emoji to Canonical

```markdown
# Before (Emoji)
- [ ] Task 📅 2025-01-15 ⏫ #work

# After (Canonical)
- [ ] Task
  due: 2025-01-15
  priority: high
  context: work
```

---

*Flip Task System Reference v1.1 - December 2025*
