# Task Metadata - Organization, Project & Context

## Overview

Tasks can now be organized by **Organization**, **Project**, and **Context** using flexible syntax options. Metadata can be defined at the note level (frontmatter) and overridden at the task level.

## Syntax Options

### 1. Frontmatter (Note-Level)

Applies to ALL tasks in the file:

```markdown
---
organization: P1174
project: Cloud-Migration
context: BACKEND
---

# Tasks

- [ ] Task 1  (inherits: P1174, Cloud-Migration, BACKEND)
- [ ] Task 2  (inherits: P1174, Cloud-Migration, BACKEND)
```

### 2. Inline Compact Syntax

Short bracket notation:

```markdown
- [ ] [P1174] Client task (organization only)
- [ ] [P1174:BACKEND] Client backend work (org + context)
- [ ] [P1174:Cloud:BACKEND] Full specification (org + project + context)
- [ ] [DANORAMA:HR] Company HR task
```

### 3. Inline Explicit Syntax

Key:value format:

```markdown
- [ ] Review code org:P1174 ctx:BACKEND
- [ ] Monthly report org:DANORAMA ctx:FINANCE
- [ ] Deploy service org:P1174 ctx:DEVOPS proj:Cloud
```

### 4. Indented Metadata

Multi-line format:

```markdown
- [ ] Complex task with metadata
  organization:: P1174
  project:: Cloud-Migration
  context:: BACKEND
  due:: 2025-12-01
```

## Precedence Rules

When multiple sources define the same metadata:

**Inline > Indented > Frontmatter**

Example:
```markdown
---
organization: DEFAULT
---

- [ ] [OVERRIDE] Task  
  organization:: IGNORED

Result: organization = OVERRIDE (inline wins)
```

## Filtering Tasks

### CLI Filters

```bash
# Filter by organization
flip task list --org P1174

# Filter by context
flip task list --context BACKEND

# Filter by project
flip task list --project Cloud-Migration

# Combined filters
flip task list --org P1174 --context FINANCE

# With other filters
flip task list --org P1174 --status open --priority high
```

### Display Format

Tasks show metadata inline:

```
⏫ [ ] Deploy API endpoint [P1174] ctx:BACKEND 📅 Nov 15 #urgent
```

## Use Cases

### Client Project Organization

```markdown
---
organization: P1174
project: Cloud-Migration
---

## Backend Tasks
- [ ] [P1174:BACKEND] Deploy new API
- [ ] [P1174:BACKEND] Update database schema

## Frontend Tasks  
- [ ] [P1174:FRONTEND] Update UI components
```

### Multi-Organization Work

```markdown
## Client Work
- [ ] [P1174:FINANCE] Monthly report for client
- [ ] [CLIENT-XYZ:DEVOPS] Infrastructure setup

## Internal Work
- [ ] [DANORAMA:HR] Recruitment interviews
- [ ] [DANORAMA:FINANCE] Budget planning

## Personal
- [ ] [PERSONAL:HEALTH] Fitness goal
```

### Context-Based Filtering

Track different areas of work:

- `BACKEND` - Backend development
- `FRONTEND` - Frontend development  
- `DEVOPS` - Infrastructure & deployment
- `FINANCE` - Financial tasks
- `HR` - Human resources
- `RESEARCH` - Research tasks
- `DOCUMENTATION` - Documentation work

## Key Aliases

For flexibility, multiple keys are supported:

- Organization: `organization`, `org`
- Context: `context`, `ctx`
- Project: `project`, `proj`

All work in frontmatter, inline key:value, and indented metadata.

## Examples

### Example 1: Client Consulting

```markdown
---
organization: P1174
project: ERP-Migration
---

# Weekly Tasks

## Technical Work
- [ ] ctx:BACKEND Database migration script
- [ ] ctx:FRONTEND Update dashboard UI
- [ ] ctx:DEVOPS Deploy to staging

## Business Tasks
- [ ] ctx:FINANCE Review budget allocation
- [ ] ctx:PLANNING Weekly status meeting
```

Filter: `flip task list --org P1174 --context BACKEND`

### Example 2: Mixed Organization Workflow

```markdown
# Today's Tasks

- [ ] [P1174:BACKEND] ⏫ Client API deployment 📅 2025-11-15 @john
- [ ] [DANORAMA:HR] Schedule interviews #recruiting
- [ ] org:PERSONAL ctx:HEALTH Morning workout
- [ ] [INTERNAL] Update flip documentation
```

Filter examples:
- `flip task list --org P1174` → Client tasks only
- `flip task list --org PERSONAL` → Personal tasks
- `flip task list --context HR` → All HR tasks

## Tips

1. **Consistency**: Choose one syntax style per note for readability
2. **Uppercase**: Use UPPERCASE for organizations and contexts for clarity
3. **Hyphens**: Allowed in all identifiers: `CLIENT-XYZ`, `MULTI-WORD`
4. **Frontmatter**: Best for single-organization notes
5. **Inline**: Best for mixed-organization task lists
6. **Context**: Use meaningful context tags that match your workflow

## Validation

Values are case-sensitive and support:
- Uppercase letters: A-Z
- Numbers: 0-9  
- Hyphens: -

Examples: `P1174`, `CLIENT-XYZ`, `BACKEND`, `MULTI-WORD-CONTEXT`
