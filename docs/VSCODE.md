# VS Code Integration

> Use flip directly from VS Code with native dialogs and tasks.

---

## Quick Setup

```bash
# Install flip tasks to your VS Code workspace
flip vscode install
```

This creates `.vscode/tasks.json` with all flip commands.

---

## Using Flip Tasks

1. Open VS Code in your brain folder (or any folder)
2. Press `Cmd+Shift+P` (Mac) or `Ctrl+Shift+P` (Windows/Linux)
3. Type "Tasks: Run Task"
4. Select any "Flip: ..." task

### Available Tasks

| Task | Description | Shortcut Suggestion |
|------|-------------|---------------------|
| **Flip: Menu** | Open interactive menu | — |
| **Flip: Journal (Today)** | Open/create today's journal | `Cmd+J` |
| **Flip: New Task** | Create new task | `Cmd+T` |
| **Flip: Task Done** | Mark task as done | — |
| **Flip: Task List** | List all tasks | — |
| **Flip: New Note** | Create new note | `Cmd+N` |
| **Flip: Quick Note** | Quick note (minimal prompts) | — |
| **Flip: Search Notes** | Search all notes | `Cmd+F` |
| **Flip: File Info** | Info about current file | — |
| **Flip: Status** | Show brain/workspace status | — |
| **Flip: Switch Brain** | Change active brain | — |

---

## VS Code Extension (Optional)

For deeper integration, install the flip VS Code extension:

### Features
- Status bar showing active brain
- Native VS Code dialogs for note/task creation
- Quick Pick for brain switching
- AI research/summarize/improve with prompt notes

### Installation

The extension is in development. To install from source:

```bash
cd /path/to/flip/vscode-extension
npm install
npm run compile
# Then use VS Code's "Install from VSIX" or run in dev mode

# Alternatively, from project root:
make package-extension
make install-extension   # detects code/codium/code-insiders
```

---

## AI Prompt Notes

The extension can select or create prompt notes for AI commands.

### Flow
1. Pick brain
2. Pick prompt (or create one if none exist)
3. Optional extra instructions (multi-line editor)
4. Run AI command

Prompt notes are stored under notes/prompts. Default prompt notes are used automatically when set.

---

## Context-Aware Actions

flip can detect what type of file you're editing:

```bash
# Check file type
flip file-info /path/to/file.md

# Output:
# Type: exercise
# Brain: my-brain
# Brain Type: flip
```

### How It Works

1. flip reads the YAML frontmatter of your file
2. Looks for `type:` field (exercise, note, task, plan, journal)
3. Falls back to directory detection (`/exercises/` → exercise)

### Using in VS Code Tasks

The "Flip: Context-Aware Action" task automatically:
- Detects current file type
- Runs appropriate command
- Example: In an exercise file → opens exercise editor

---

## Keyboard Shortcuts

VS Code doesn't auto-create shortcuts for tasks. To add them:

1. Open `Keyboard Shortcuts` (`Cmd+K Cmd+S`)
2. Search for "Tasks: Run Task"
3. Or add to `keybindings.json`:

```json
{
  "key": "cmd+shift+j",
  "command": "workbench.action.tasks.runTask",
  "args": "Flip: Journal (Today)"
}
```

---

## Task Terminal Behavior

All flip tasks run in VS Code's integrated terminal with:

- **Dedicated panel** - Each task gets its own terminal
- **Auto-focus** - Terminal gets focus when task runs
- **Auto-clear** - Previous output cleared on each run

Interactive commands (like `flip menu`) work fully in the VS Code terminal.

---

## Troubleshooting

### "flip not found"

Make sure flip is in your PATH:

```bash
# Check if flip is available
which flip

# If not found, add to PATH or use absolute path
export PATH=$PATH:/path/to/flip/binary
```

### Tasks Not Appearing

1. Verify `.vscode/tasks.json` exists in your workspace
2. Run `flip vscode status` to check installation
3. Reload VS Code window (`Cmd+Shift+P` → "Reload Window")

### Terminal Closes Too Fast

Some tasks may close quickly. Check the task output:
1. `Cmd+Shift+P` → "Tasks: Show Running Tasks"
2. Or check `View → Terminal`

---

## Updating Tasks

When flip adds new features:

```bash
# Check if update needed
flip vscode status

# Update tasks.json
flip vscode update
```

---

## Multiple Workspaces

If you have multiple VS Code workspaces:

```bash
# Install to specific workspace
cd /path/to/workspace
flip vscode install

# Each workspace gets its own tasks.json
```

---

*For CLI reference, see [COMMANDS.md](COMMANDS.md)*
