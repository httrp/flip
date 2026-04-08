# Flip in 5 Minutes 🚀

> Your personal knowledge base in plain text. No vendor lock-in. No subscriptions. Just Markdown.

---

## What is Flip?

**Flip** is a CLI tool for managing personal knowledge bases ("brains") in plain Markdown. It helps you:

- 📓 Write **daily journals**
- 📝 Create **structured notes** with metadata
- ✅ Manage **tasks** with a powerful browser & filters
- 🗓️ Track **meetings** and decisions
- 🔗 Connect everything with **standard Markdown links**

All data is stored as **plain Markdown** files. No vendor lock-in. No subscriptions. **You own your data forever.**

### Key Concepts

**Brain** = Your personal knowledge base folder
- Contains notes, tasks, journals, meetings
- Can be Flip format, Obsidian vault, Logseq graph, Dendron workspace, or plain Markdown
- Syncs with Dropbox, iCloud, OneDrive, or Git

**Workspace** = A collection of related brains
- Organize multiple brains (work, personal, projects, learning)
- Each workspace has an active brain for creating new content
- Independent configuration per workspace

**Example workflow:**
```
You have 3 workspaces:
  - "work" (active brain: projects)
  - "personal" (active brain: life)
  - "learning" (active brain: languages)

flip journal              # Creates entry in active workspace's brain
flip workspace switch personal
flip journal              # Now creates entry in "life" brain
```

---

## Installation

```bash
# macOS / Linux / Windows
go install github.com/httrp/flip/cmd/flip@latest

# Or build from source
git clone https://github.com/httrp/flip
cd flip
make build
```

---

## Quick Setup (Recommended)

Use the interactive quickstart:

```bash
flip quickstart
```

This guides you through:
1. Creating a workspace
2. Creating or connecting a brain
3. Writing your first entry

Done in ~2 minutes! 🚀

---

## Manual Setup: Your First Brain

### 1. Create a Workspace

```bash
flip workspace create work
flip workspace switch work
```

**Why workspaces?** Keep your work, personal, and learning separate.

### 2. Create Your First Brain

```bash
flip brain new
# Choose: flip (recommended) or another brain type
# Follow prompts for name & location
```

Or add an existing folder:

```bash
flip brain add ~/Documents/my-notes
```

This creates (for Flip format):
```
my-brain/
├── .flip-brain.yaml      # Brain configuration & metadata
├── .flip.yaml            # Workspace brain registry
├── journal/              # Daily journal entries (YYYY-MM-DD.md)
├── notes/                # Your notes with frontmatter
├── meetings/             # Meeting notes with attendees/decisions
├── tasks/                # Todo lists by context (todo.work.md, etc.)
├── definitions/          # Metadata: organizations, people, projects, contexts
├── templates/            # Templates for notes, meetings, etc.
└── assets/               # Images and documents
```

### 3. Open Today's Journal

```bash
flip journal
# Opens 2025-12-21.md in your editor
```

### 4. Create a Note

```bash
flip note "My First Idea"
# Creates notes/my-first-idea.md
```

### 5. Add a Task

```bash
flip task add "Learn flip basics" --due tomorrow --priority high
# Adds to your task list
```

### 6. Browse Tasks

```bash
flip task browse
# Interactive task browser with filters
```

---

## Daily Workflow

```bash
# Morning
flip journal                    # Open today's journal

# During the day
flip note "Meeting insights"    # Quick note
flip task add "Follow up"       # New task
flip meeting "Standup"          # Meeting notes

# Review
flip task browse                # See all tasks
flip brain check health         # Check for broken links
```

---

## Task Power Features

The task browser is flip's superpower:

```bash
flip task browse              # All tasks
flip task browse --due today  # Due today
flip task browse --context work  # Work tasks only
flip task browse --overdue    # Overdue tasks
```

**In the browser:**
- `Enter` - View/edit task
- `Space` - Toggle status
- `d` - Set due date
- `p` - Change priority
- `/` - Search
- `?` - Help

---

## File Format

Everything is **plain Markdown** with YAML headers:

```markdown
---
title: "My Note"
date: 2025-12-21
tags: [idea, project]
---

# My Note

Your content here...
```

**Tasks** use a simple format:
```markdown
- [ ] Task title
  due: 2025-12-25
  priority: high
```

---

## Why Flip?

| Feature | Flip | Others |
|---------|------|--------|
| **Data ownership** | ✅ Plain files | ❌ Proprietary DB |
| **Works offline** | ✅ Always | ⚠️ Sometimes |
| **Git-friendly** | ✅ Small text files | ⚠️ Merge conflicts |
| **Editor freedom** | ✅ Any editor | ❌ Specific app |
| **Cost** | ✅ Free forever | 💰 Subscriptions |

---

## Next Steps

1. **Explore**: Run `flip --help` to see all commands
2. **Customize**: Edit `.flip-brain.yaml` to configure your brain
3. **Learn**: Check `flip brain check health` to validate your brain
4. **Read**: See [FLIP_BRAIN_SPEC.md](FLIP_BRAIN_SPEC.md) for the full format spec

---

## Quick Reference

| Command | Description |
|---------|-------------|
| `flip brain init NAME` | Create new brain |
| `flip journal` | Open today's journal |
| `flip note "Title"` | Create new note |
| `flip meeting "Title"` | Create meeting note |
| `flip task add "Task"` | Add task |
| `flip task browse` | Browse tasks |
| `flip brain check health` | Check brain health |
| `flip --help` | Show all commands |

---

**That's it!** You're ready to use flip. 

Questions? Issues? → [GitHub Issues](https://github.com/your-org/flip/issues)

---

*Flip - Your brain, your files, your freedom.* 🧠
