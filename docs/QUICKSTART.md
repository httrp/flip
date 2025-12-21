# Flip in 5 Minutes 🚀

> Your personal knowledge base in plain text. No vendor lock-in. No subscriptions. Just Markdown.

---

## What is Flip?

**Flip** is a CLI tool for managing your personal knowledge base ("brain"). It helps you:

- 📓 Write **daily journals**
- 📝 Create **structured notes**
- ✅ Manage **tasks** with a powerful browser
- 🗓️ Track **meetings** and decisions
- 🔗 Keep everything **connected**

All data is stored as **plain Markdown** files. You own your data forever.

---

## Installation

```bash
# macOS / Linux
go install github.com/your-org/flip@latest

# Or build from source
git clone https://github.com/your-org/flip
cd flip
make build
```

---

## Your First Brain

### 1. Create a Brain

```bash
flip brain init my-brain
cd my-brain
```

This creates:
```
my-brain/
├── .flip-brain.yaml      # Brain configuration
├── journal/              # Daily entries
├── notes/                # Your notes
├── meetings/             # Meeting notes
├── tasks/                # Task lists
├── definitions/          # Metadata (orgs, people, projects)
└── templates/            # Note templates
```

### 2. Open Today's Journal

```bash
flip journal
# Opens 2025-12-21.md in your editor
```

### 3. Create a Note

```bash
flip note "My First Idea"
# Creates notes/my-first-idea.md
```

### 4. Add a Task

```bash
flip task add "Learn flip basics" --due tomorrow --priority high
# Adds to your task list
```

### 5. Browse Tasks

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
