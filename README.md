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

## Development

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

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## Architecture

```
flip/
├── cmd/flip/           # CLI entry point
├── internal/
│   ├── commands/       # Cobra commands
│   ├── tasks/          # Task system
│   ├── exercises/      # Exercise tracking
│   ├── brain/          # Brain detection
│   ├── health/         # Health checks
│   └── lang/           # Localization
├── docs/               # Documentation
└── vscode-extension/   # VS Code extension (TypeScript)
```

## License

MIT

---

*Built with ❤️ for knowledge workers who love plain text.*
