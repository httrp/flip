# flip

> Steroids for your 2nd brain 🧠

**flip** is a CLI tool for managing personal knowledge bases. It works with your existing markdown folders - Obsidian vaults, Logseq graphs, Dendron workspaces, or plain directories.

## Features

- 📓 **Daily Journals** - One command to open today's entry
- 📝 **Structured Notes** - Templates, frontmatter, search
- ✅ **Task Management** - Markdown-based tasks with TUI browser
- 🏋️ **Exercise Tracking** - Practice sessions, plans, progress
- 🔗 **Multi-Brain** - Connect and switch between knowledge bases
- 🎨 **VS Code Integration** - Tasks, commands, status bar
- 🤖 **AI Assist** - Research, summaries, and improvements

## Quick Start

```bash
# Install
go install github.com/httrp/flip/cmd/flip@latest

# Or build from source
git clone https://github.com/httrp/flip.git
cd flip && make build

# Create your first brain
flip brain init ~/my-brain

# Open interactive menu
flip
```

## Common Commands

```bash
flip journal              # Today's journal
flip note "My Idea"       # New note
flip task new             # New task
flip task browse          # Task browser (TUI)
flip exercise new         # New exercise
flip status               # Overview
```

## Documentation

| Document | Description |
|----------|-------------|
| [QUICKSTART](docs/QUICKSTART.md) | Get started in 5 minutes |
| [COMMANDS](docs/COMMANDS.md) | Full command reference |
| [VSCODE](docs/VSCODE.md) | VS Code integration guide |
| [FEATURES](docs/FEATURES.md) | Feature overview |

## Supported Brain Types

| Type | Detection | Status |
|------|-----------|--------|
| **Flip** | `.flip-brain.yaml` | ✅ Full support |
| **Obsidian** | `.obsidian/` | ✅ Full support |
| **Logseq** | `.logseq/` | ✅ Full support |
| **Dendron** | `dendron.yml` | ✅ Full support |
| **Foam** | `.foam/` | ✅ Full support |
| **Plain Markdown** | Any folder | ✅ Full support |

## VS Code Integration

```bash
# Install VS Code tasks
flip vscode install

# Then: Cmd+Shift+P → "Tasks: Run Task" → "Flip: ..."
```

## Getting Started

**First time?** Run the interactive quickstart:

```bash
flip quickstart
```

This walks you through:
1. Creating or joining a workspace
2. Creating a new brain or connecting an existing one
3. Writing your first journal entry

For manual setup, see [QUICKSTART.md](docs/QUICKSTART.md).

## Development

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup & build instructions
- Testing & lint
- Adding new commands
- VS Code extension development

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Install dev symlink
make dev-link
```

## Core Concepts

### Brain
A **brain** is a folder containing your knowledge base (notes, tasks, journals, meetings). Flip works with plain Markdown files and supports multiple brain formats:
- **Flip** (`flip` native format with `.flip-brain.yaml`)
- **Obsidian** (`.obsidian/` vaults)
- **Logseq** (`.logseq/` graphs)
- **Dendron** (`dendron.yml` workspaces)
- **Foam** (`.foam/` setups)
- **Plain Markdown** (any folder with `.md` files)

### Workspace
A **workspace** groups related brains together. Each workspace has:
- An **active brain** (used for `flip journal`, `flip note`, etc.)
- Multiple brains you can switch between
- Independent configuration per workspace

**Example:**
```
Workspace "work":
  - brain: project-alpha (default)
  - brain: project-beta
  - brain: documentation

Workspace "personal":
  - brain: life (default)
  - brain: learning
```

## Architecture

```
flip/
├── cmd/flip/                # CLI entry point
├── internal/
│   ├── commands/            # Cobra CLI commands (brain, task, journal, etc.)
│   ├── brain/               # Brain detection & creation
│   │   ├── detector.go      # Identify brain types (Obsidian, Logseq, etc.)
│   │   └── creator.go       # Create new brains
│   ├── tasks/               # Task scanning & management
│   ├── exercises/           # Exercise tracking
│   ├── health/              # Brain validation & repair
│   ├── ai/                  # AI integrations (OpenAI, Anthropic, etc.)
│   ├── migration/           # Convert between brain formats
│   ├── templates/           # Template loading & rendering
│   ├── platform/            # Cross-platform utilities
│   ├── ui/                  # Interactive prompts (TUI)
│   └── lang/                # Multi-language support
├── vscode-extension/        # VS Code integration (TypeScript)
├── docs/                    # Documentation & specs
└── test-brains/            # Test fixtures
```

### Key Design Patterns

1. **Workspace Configuration** (`~/.config/flip/workspaces.yaml`)
   - Manages workspace collections
   - Tracks active workspace & default brain per workspace
   - Persisted locally, no cloud sync

2. **Brain Detection**
   - Automatic brain type identification via marker files
   - Backward compatible with existing Obsidian, Logseq, etc. setups
   - Supports nested brain prevention

3. **Health Checks**
   - Validates path integrity across sync services (iCloud, Dropbox, OneDrive)
   - Auto-repairs broken links
   - Checks for duplicate files created by sync conflicts

## License

MIT - see LICENSE.

---

*Built with ❤️ for knowledge workers who love plain text.*
