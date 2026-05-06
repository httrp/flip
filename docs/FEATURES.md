# Flip Features Overview

> Last updated: 2025-12-22

Flip is a CLI tool for managing personal knowledge bases ("brains"). It works with existing markdown folders and supports multiple brain types.

## Core Philosophy

- **Standard Markdown** - Uses `[text](path.md)` links, no proprietary formats
- **Brain Agnostic** - Works with Obsidian, Logseq, Dendron, Foam, or plain folders
- **Non-Destructive** - Never modifies your files without explicit consent
- **Workspace-Based** - Organize multiple brains into logical workspaces

---

## 🧠 Brain Management

| Command | Description |
|---------|-------------|
| `flip brain add <path>` | Add existing folder as brain |
| `flip brain new` | Create new brain with templates |
| `flip brain init` | Initialize brain in current directory |
| `flip brain list` | List all brains in workspace |
| `flip brain scan` | Auto-discover brain folders |
| `flip brain repair` | Fix broken paths |
| `flip brain health` | Check brain health status |
| `flip brain migrate` | Migrate between brain formats |

### Supported Brain Types

| Type | Detection | Link Style |
|------|-----------|------------|
| **Flip** | `.flip-brain.yaml` or `.flip.yaml` | Markdown `[](path.md)` |
| **Obsidian** | `.obsidian/` folder | Wikilinks `[[]]` |
| **Logseq** | `.logseq/` folder (or `journals/` + `pages/`) | Wikilinks `[[]]` |
| **Dendron** | `dendron.yml` | Wikilinks `[[]]` |
| **Foam** | `.foam/` folder | Wikilinks `[[]]` |
| **Plain** | Just markdown files | Markdown `[](path.md)` |

---

## 📁 Workspace Management

| Command | Description |
|---------|-------------|
| `flip workspace create` | Create new workspace |
| `flip workspace list` | List all workspaces |
| `flip workspace switch` | Switch active workspace |
| `flip workspace remove` | Remove workspace |
| `flip workspace rename` | Rename workspace |

---

## ✏️ Content Creation

### Notes
| Command | Description |
|---------|-------------|
| `flip note` | Create note with prompts |
| `flip quicknote` | Create note quickly (minimal prompts) |
| `flip note list` | List recent notes |
| `flip note search` | Search notes |

### Journals
| Command | Description |
|---------|-------------|
| `flip journal` | Create/open today's journal |
| `flip journal yesterday` | Open yesterday's journal |
| `flip journal <date>` | Open specific date |

### Meetings
| Command | Description |
|---------|-------------|
| `flip meeting-note` | Create meeting note |

---

## 🤖 AI Assist

Create research notes, summaries, and improvements with AI.

| Command | Description |
|---------|-------------|
| `flip ai research` | Create research note from topic |
| `flip ai summarize` | Summarize notes about a topic |
| `flip ai improve` | Improve an existing note |

### AI Templates

Reusable template notes live in templates/ai. You can set a default template or add optional run notes per request.

AI notes use short filenames based on the title (`<slug>-YYYY-MM-DD.md`) and include `## Template Context` and `## Sources` sections at the top.

### Tasks
| Command | Description |
|---------|-------------|
| `flip task new` | Create new task |
| `flip task list` | List tasks |
| `flip task browse` | Interactive TUI browser |
| `flip task done <id>` | Mark task complete |

---

## 🏋️ Exercise Tracking

Track repeatable practice sessions (music, languages, skills).

| Command | Description |
|---------|-------------|
| `flip exercise new` | Create new exercise |
| `flip exercise list` | List exercises |
| `flip exercise log` | Log a practice session |
| `flip exercise stats` | View practice statistics |
| `flip exercise plan new` | Create training plan |

---

## 🔄 Migration

Convert between brain formats while preserving content.

```bash
# Dry-run (plan only)
flip brain migrate --source ~/obsidian-vault --target ~/new-flip-brain

# Execute migration
flip brain migrate --source ~/obsidian-vault --target ~/new-flip-brain --execute
```

### Migration Features
- ✅ Wikilinks → Markdown links (and vice versa)
- ✅ Asset relocation
- ✅ Folder structure mapping
- ✅ Conflict detection
- ✅ Rollback support
- ✅ Progress tracking

---

## 📦 Git Integration

Automatic git operations for brains.

| Command | Description |
|---------|-------------|
| `flip brain git status` | Git status of default brain |
| `flip brain git log` | Recent commits |
| `flip brain git commit` | Commit changes |
| `flip brain git pull` | Pull remote changes |

### Auto-Sync Features
- Uncommitted changes warning on exit
- Remote updates check on start
- Conflict detection

---

## 🎨 Interactive Menu

```bash
flip menu
```

Full-featured TUI menu with:
- Breadcrumb navigation
- Status header (workspace, brain, git)
- All commands accessible
- Keyboard shortcuts

---

## 📋 Definitions

Manage reusable metadata for tasks and notes.

| Command | Description |
|---------|-------------|
| `flip definitions org add` | Add organization |
| `flip definitions org list` | List organizations |
| `flip definitions context add` | Add context |
| `flip definitions people add` | Add person |

---

## 🔧 Templates

| Command | Description |
|---------|-------------|
| `flip template list` | List available templates |
| `flip template show <name>` | Show template content |

### Built-in Templates
- `note-template.md`
- `journal-template.md`
- `meeting-template.md`
- `task-template.md`
- `prompt-template.md`
- `exercise-template.md`
- `exercise-plan-template.md`

---

## 📤 Markdown Export

Export markdown notes to shareable document formats with pandoc.

| Command | Description |
|---------|-------------|
| `flip export convert <file.md> --format pdf` | Convert markdown to PDF |
| `flip export convert <file.md> --format html` | Convert markdown to HTML |
| `flip export convert <file.md> --format docx` | Convert markdown to DOCX |
| `flip export template list --format pdf` | Discover available templates |
| `flip export doctor` | Check dependencies and install hints |

### Export Defaults
- Output defaults to `~/Desktop/flip-output` to keep brains clean.
- If input is inside a known brain, output path includes brain name and date.
- PDF auto-detects common templates like Eisvogel when available.

---

## 🏥 Health Check

Automatic brain health monitoring:

- ✅ Path existence validation
- ✅ Required folder structure
- ✅ Git repository status
- ✅ Sync service detection (iCloud, Dropbox, etc.)
- ✅ Conflict file detection
- ⚠️ Warning aggregation in status

---

## 🌐 Internationalization

Built-in language support:
- English (default)
- German

All UI text externalized in `lang/` folder.

---

## 📊 Status Overview

```bash
flip status
```

Shows:
- Active workspace & brain
- All workspaces with brain counts
- Brain health status
- Git status per brain
- Sync service detection
- Issues & warnings

---

## 🚀 Quick Start

```bash
# 1. Start interactive menu
flip menu

# 2. Or use quickstart wizard
flip quickstart

# 3. Or CLI commands
flip workspace create "personal"
flip brain add ~/my-notes --name "notes"
flip brain set-default notes
flip journal  # Start writing!
```

---

## Configuration

Config stored in: `~/.config/flip/config.json`

```json
{
  "workspaces": [...],
  "activeWorkspace": "default",
  "theme": "mixed"
}
```

---

## Coming Soon

- [ ] Backlinks discovery
- [ ] Graph visualization
- [ ] Content-only brains (MkDocs, Hugo support)
- [ ] Multi-format export
- [ ] AI-powered search

---

## Documentation Index

| Document | Description |
|----------|-------------|
| [QUICKSTART.md](QUICKSTART.md) | Getting started guide |
| [FLIP_BRAIN_SPEC.md](FLIP_BRAIN_SPEC.md) | Brain format specification |
| [TASK_REFERENCE.md](TASK_REFERENCE.md) | Task format reference |
| [BRAIN_MIGRATION.md](BRAIN_MIGRATION.md) | Migration guide |
| [BRAIN_TYPES_REFERENCE.md](BRAIN_TYPES_REFERENCE.md) | Supported brain types |
| [BRAIN_HEALTH_CHECK.md](BRAIN_HEALTH_CHECK.md) | Health check details |
