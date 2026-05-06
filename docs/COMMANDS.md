# Flip Command Reference

> Complete reference for all flip commands. For quick start, see [QUICKSTART.md](QUICKSTART.md).

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--theme <type>` | Icon theme: `ascii`, `emoji`, or `mixed` (default) |
| `--help` | Show help for any command |

---

## Brain Management

Brains are your knowledge bases (folders with notes, tasks, journals). Use **workspaces** to organize multiple brains.

### Create & Add Brains

```bash
# Create new brain interactively
flip brain new
# Available from: flip new → brain

# Add an existing folder as a brain
flip brain add <path> [--name <name>]
# Available from: flip new → workspace → add existing brain

# Initialize current directory as brain
flip brain init [<path>]  [--name <name>]

# Scan home folder for existing brains
flip brain scan
```

### Manage Brains

```bash
# List all brains in active workspace
flip brain list

# Set default brain for new content
flip brain set-default <name>

# Switch to different brain
flip brain switch

# Remove brain from flip (files stay intact)
flip brain remove <name>

# Check brain health
flip brain health

# Scan for new brain folders
flip brain scan
```

---

## Workspace Management

**Workspaces** organize multiple brains. Each workspace has its own active brain for content creation.

### Setup

First time getting started?
```bash
flip quickstart    # Interactive setup guide (creates workspace + brain)
```

### Commands

```bash
# List all workspaces
flip workspace list

# Create new workspace
flip workspace create <name>

# Switch active workspace
flip workspace switch <name>

# Remove workspace (brains stay intact)
flip workspace remove <name>

# Rename workspace
flip workspace rename <old> <new>
```

---

## Content Creation

### Journal

Daily journal entries, one file per day.

```bash
# Open/create today's journal
flip journal

# Yesterday's journal
flip journal yesterday

# Specific date
flip journal 2025-01-01
flip journal "last monday"
```

### Notes

Structured notes with frontmatter metadata.

```bash
# Create note (interactive)
flip note

# Create note with title
flip note "My Note Title"

# Quick note (minimal prompts)
flip quicknote "Quick thought"

# List recent notes
flip note list

# Search notes
flip note search <query>
```

### Meeting Notes

Meeting notes with attendees, agenda, and action items.

```bash
# Create meeting note (interactive)
flip meeting-note
```

---

## Export

Convert markdown files to shareable formats without cluttering your brain folders.

```bash
# Convert markdown to PDF (default output: ~/Desktop/flip-output)
flip export convert notes/my-note.md --format pdf

# Convert to HTML or DOCX
flip export convert notes/my-note.md --format html
flip export convert notes/my-note.md --format docx

# Override output directory
flip export convert notes/my-note.md --format pdf --output-dir ~/Documents/exports

# Use landscape PDF layout
flip export convert notes/my-note.md --format pdf --landscape

# Use specific template name or path
flip export convert notes/my-note.md --format pdf --template eisvogel
flip export convert notes/my-note.md --format pdf --template /path/to/eisvogel.latex

# DOCX style via reference document
flip export convert notes/my-note.md --format docx --reference-doc /path/to/reference.docx

# Discover templates and reference docs
flip export template list --format pdf
flip export template list --format html
flip export template list --format docx

# Check dependencies and install hints
flip export doctor --format pdf
```

### Export Output Behavior

- By default, exports are written to `~/Desktop/flip-output`
- If the input belongs to a known brain, export paths include brain name and date
- Use `--output` or `--output-dir` to override defaults

### Template Discovery

Flip searches templates in:

- `PANDOC_DATA_DIR/templates`
- `pandoc --data-dir` + `/templates`
- `./templates` (current working directory)
- `~/.pandoc/templates` (legacy)
- `~/.local/share/pandoc/templates`

---

## AI Commands

AI-powered research, summaries, and improvements.

```bash
# Research a topic
flip ai research "Topic"

# Summarize notes about a topic
flip ai summarize "Topic"

# Improve a note
flip ai improve notes/my-note.md
```

### AI Note Conventions

- Filename: `<short-title-slug>-YYYY-MM-DD.md` (no `research-` or `summary-` prefix)
- Title: required; suggested from topic and can be confirmed/edited
- Journal link label: `Title (Research: <provider>)` or `Title (Summary: <provider>)`
- Top sections: `## Template Context` and `## Sources` are always present

### Prompt Notes

Template notes live under templates/ai and can be reused across AI commands.

```bash
# Create a prompt note
flip note --subfolder prompts --title "Research (Deep)"
```

### Useful Flags

```bash
# Select template note (path or name)
--template "Research (Deep)"

# Add concrete request for this run
--request "Analyze migration options for project X"

# Add optional run notes for this request
--run-notes "Focus on 2025-2026 trends"

# Add explicit context files (repeat flag)
--context-file notes/architecture.md --context-file notes/decisions.md

# Research only: auto-search current brain for additional context
--context-auto-brain

# Create a template note on the fly
--template-create-title "Research (Deep)"
--template-create-body "Structured research with sources and risks"
--template-create-default

# Override model and timeout
--model "llama3.2"
--timeout 240
```

---

## Task Management

Tasks can live in any markdown file or dedicated task files.

### Basic Commands

```bash
# Create new task (interactive)
flip task new

# List all tasks
flip task list

# List open tasks only
flip task list --open

# Mark task as done
flip task done

# Start a task (mark in-progress)
flip task start

# Update task properties
flip task update
```

### Task Browser (TUI)

Full-featured terminal task browser.

```bash
# Open task browser
flip task browse

# Keyboard shortcuts in browser:
#   ↑/↓     Navigate
#   Enter   Toggle done/open
#   d       Mark done
#   s       Start task
#   /       Search
#   f       Filter by status
#   p       Filter by priority
#   q       Quit
```

### Task Syntax in Markdown

```markdown
- [ ] Open task
- [x] Completed task
- [/] In-progress task
- [-] Cancelled task
- [>] Deferred task

# With metadata:
- [ ] Task description [priority: high] [due: 2025-01-15] [project: flip]
```

---

## Exercise Tracking

Track practice sessions for repeatable activities.

```bash
# Create new exercise
flip exercise new

# List exercises
flip exercise list

# Edit exercise (add variants)
flip exercise edit

# Show exercise details
flip exercise show <id>

# Track a session
flip exercise track
```

### Exercise Plans

Combine exercises into training plans.

```bash
# Create exercise plan
flip exercise plan new

# Edit plan (add exercises)
flip exercise plan edit

# List plans
flip exercise plan list

# Show plan details
flip exercise plan show <id>
```

---

## Definitions

Reusable metadata for tasks and notes.

```bash
# Organizations
flip definitions org add
flip definitions org list

# Projects
flip definitions project add
flip definitions project list

# Contexts
flip definitions context add
flip definitions context list

# People
flip definitions people add
flip definitions people list
```

---

## Templates

Manage note templates.

```bash
# List available templates
flip template list

# Show template content
flip template show <name>
```

---

## Utilities

### Status

```bash
# Show workspace/brain overview
flip status
```

### File Info

Get metadata about any brain file (for scripting/VS Code).

```bash
flip file-info <filepath>
flip file-info <filepath> --json
```

### Interactive Menu

```bash
# Open full interactive menu
flip menu

# Or just run flip with no arguments
flip
```

---

## VS Code Integration

### Install VS Code Tasks

```bash
# Install flip tasks to current workspace
flip vscode install

# Check status
flip vscode status

# Update tasks
flip vscode update
```

### Available VS Code Tasks

After installing, use `Cmd+Shift+P` → "Tasks: Run Task":

- **Flip: Menu** - Open interactive menu
- **Flip: Journal (Today)** - Open today's journal
- **Flip: New Task** - Create task
- **Flip: Task Done** - Mark task done
- **Flip: New Note** - Create note
- **Flip: File Info** - Info about current file

---

## Configuration

flip stores configuration in `~/.config/flip/`:

```
~/.config/flip/
├── workspaces.json     # Workspace and brain configuration
└── settings.json       # User preferences (future)
```

Each brain can have a `.flip-brain.yaml` file:

```yaml
brain:
  name: my-brain
  type: flip
  version: "1.0"

config:
  author: Your Name
  default_organization: Personal
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |

---

## Examples

### Daily Workflow

```bash
# Morning: Open journal
flip journal

# Create note from meeting
flip note "Project Kickoff Notes"

# Add follow-up task
flip task new

# End of day: Review tasks
flip task browse
```

### Multi-Brain Setup

```bash
# Add work and personal brains
flip brain add ~/work-notes --name work
flip brain add ~/personal-notes --name personal

# Create workspace for work
flip workspace create work
flip workspace switch work
# (now only work brain is active)

# Switch to default to see all
flip workspace switch default
```

---

*For more details, run `flip <command> --help`*
