# Flip VS Code Extension

Native VS Code integration for [flip](https://github.com/httrp/flip) - your intelligent 2nd brain assistant.

## Features

- **Command Palette Integration**: Access all flip commands via `Ctrl+Shift+P`
- **Keyboard Shortcuts**: Quick access to common actions
- **Status Bar**: See your active brain at a glance
- **Native Dialogs**: Use VS Code's input dialogs instead of terminal prompts

## Requirements

- [flip CLI](https://github.com/httrp/flip) must be installed and in your PATH
- At least one brain configured via `flip brain init`

## Commands

| Command | Shortcut | Description |
|---------|----------|-------------|
| Flip: Open/Create Journal | `Ctrl+Alt+J` | Open today's journal entry |
| Flip: New Note | `Ctrl+Alt+N` | Create a new note |
| Flip: Quick Note | `Ctrl+Alt+Q` | Create a quick note |
| Flip: New Meeting Note | - | Create a meeting note |
| Flip: Search Notes | - | Search across notes |
| Flip: Show Status | - | Show current status |
| Flip: Switch Brain | - | Switch to different brain |

## Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `flip.executablePath` | `flip` | Path to flip executable |
| `flip.timeout` | `10000` | Timeout for commands (ms) |
| `flip.showStatusBar` | `true` | Show brain in status bar |

## Development

```bash
cd vscode-extension
npm install
npm run compile
```

Press `F5` to launch Extension Development Host.

## Building VSIX

```bash
npm run package
```

This creates `flip-vscode-x.x.x.vsix` which can be installed via:
- `code --install-extension flip-vscode-x.x.x.vsix`
- Or through flip: `flip vscode install` (when VSIX is embedded)
