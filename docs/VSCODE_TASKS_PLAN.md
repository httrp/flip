# VS Code Tasks Integration Plan

> Feature Branch: `feat/vscode-tasks`
> Goal: Make flip functionality accessible via VS Code Tasks with context-aware behavior

## Overview

This document outlines the VS Code Tasks integration for flip, including:
- Task structure and organization
- Context-aware behavior and fallbacks
- Required flip enhancements
- Multi-dialect (brain type) support

---

## Existing flip Commands

The following flip commands are already implemented and ready for VS Code integration:

### Core Commands
- `flip menu` - Interactive main menu
- `flip brain list` - List available brains
- `flip brain switch` - Switch active brain

### Task Management
- `flip task new` - Create new task
- `flip task done` - Mark task as done (interactive)
- `flip task start` - Mark task as in-progress (interactive)
- `flip task update` - Update task properties (interactive)
- `flip task search [keywords]` - Search tasks

### Note Management
- `flip note new` - Create new note
- `flip note search [keywords]` - Search notes

### Exercise Management
- `flip exercise new` - Create new exercise (with brain selection)
- `flip exercise edit` - Edit exercise and add variants
- `flip exercise list` - List exercises
- `flip exercise show` - Show exercise details
- `flip exercise track` - Track exercise session

### Exercise Plans
- `flip exercise plan new` - Create new plan
- `flip exercise plan edit` - Edit plan (add exercises)
- `flip exercise plan list` - List plans
- `flip exercise plan show` - Show plan details

---

## Required New flip Features

To enable smart, context-aware VS Code Tasks, flip needs these new capabilities:

### 1. Context Detection Command
```bash
flip context-detect <filepath>
```
**Purpose:** Detect the type of a file based on YAML frontmatter `type:` field

**Output:** One of:
- `exercise` - Exercise file
- `note` - Note file
- `task` - Task file
- `plan` - Exercise plan file
- `unknown` - Unknown type or no frontmatter

**Implementation:** Parse YAML frontmatter from given file, extract `type:` field

### 2. Universal Search Command
```bash
flip search --interactive [--type=<type>]
```
**Purpose:** Unified search across all flip entities (notes, tasks, exercises, plans)

**Behavior:**
- Without flags: Search all types, show categorized results
- With `--type=exercise`: Search only exercises
- With `--type=note`: Search only notes
- `--interactive`: Present results as interactive selection menu, allow opening result

**Output:** Display results grouped by type or in interactive picker

### 3. Recent Items Command (optional)
```bash
flip list-recent --type=<type> --count=<n>
```
**Purpose:** Show recently accessed items of a specific type

**Behavior:**
- Useful for fallback behavior when in wrong context
- Default count: 5
- Output: List format suitable for quick selection

### 4. Insert Mode Support (optional)
Add `--insert` flag to creation commands:
```bash
flip task new --insert       # Creates task and inserts at ${cursor} in current file
flip exercise new --insert   # Creates exercise and inserts at ${cursor}
```
**Purpose:** Insert new items directly into current file at cursor position

---

## VS Code Tasks Structure

### Global Tasks (Available Everywhere)

| Label | Command | Shortcut | Description |
|-------|---------|----------|-------------|
| Flip: Menu | `flip menu` | — | Open interactive flip menu |
| Flip: Universal Search | `flip search --interactive` | `Cmd+Shift+F` | Search all flip items |
| Flip: Quick Task New | `flip task new` | `Cmd+Shift+T` | Create new task |
| Flip: Quick Task Done | `flip task done` | `Cmd+Shift+D` | Mark task as done |
| Flip: Quick Note New | `flip note new` | — | Create new note |
| Flip: Quick Exercise New | `flip exercise new` | — | Create new exercise |
| Flip: Task Search | `flip task search` | — | Search tasks |

### Context-Aware Tasks with Fallback

#### When in Exercise File (`type: exercise`)
| Label | Primary Command | Fallback | Shortcut |
|-------|-----------------|----------|----------|
| Flip: Add Variant | `flip exercise edit --file ${file}` | Show variant picker | `Cmd+Shift+V` |
| Flip: Track Session | `flip exercise track --file ${file}` | Show exercise picker | — |

#### When in Note File (`type: note`)
| Label | Primary Command | Fallback | Shortcut |
|-------|-----------------|----------|----------|
| Flip: Insert Task | `flip task new --insert` | `flip task new` | `Cmd+Shift+K` |

#### When in Task File (`type: task`)
| Label | Primary Command | Fallback | Shortcut |
|-------|-----------------|----------|----------|
| Flip: Mark Done | `flip task done --file ${file}` | Show task picker | `Cmd+Shift+D` |
| Flip: Edit Task | `flip task update --file ${file}` | Show task picker | — |
| Flip: Add Subtask | `flip task new --parent ${file} --insert` | Create standalone task | — |

#### When in Plan File (`type: plan`)
| Label | Primary Command | Fallback | Shortcut |
|-------|-----------------|----------|----------|
| Flip: Add Exercises | `flip exercise plan edit --file ${file}` | Show plan picker | — |

#### Fallback (Wrong or Unknown Context)
If in wrong context, task should:
1. Run `flip context-detect ${file}`
2. If doesn't match task requirements:
   - Show quick selection menu with recent items of correct type
   - Or offer to open file picker for correct type
   - Example: "Not in exercise file. Open recent exercise?" → Show list

---

## Multi-Dialect Support (Brain Type Handling)

flip supports multiple "brain dialects":
- **flip**: `notes/`, `tasks/`, `exercises/` directories
- **obsidian**: Custom folder structure with tags/properties
- **logseq**: Page-based with properties
- **dendron**: Hierarchical with dendron conventions

### Strategy

1. **YAML-Based Detection**: All brain types use YAML frontmatter for file type identification
   - Frontmatter `type:` field works across all dialects
   - `flip context-detect` works universally

2. **Brain-Agnostic Operations**: Commands operate on YAML structure, not file paths
   - Works regardless of brain type
   - Brain detection happens once at command start

3. **Workspace Awareness**: Tasks work within active workspace
   - Commands read from `~/.config/flip/workspaces.json`
   - Fallback searches scope to current brain

---

## Implementation Roadmap

### Phase 1: Core Enhancement (flip CLI)
- [ ] Add `flip context-detect <filepath>` command
- [ ] Add `flip search --interactive [--type=...]` command
- [ ] Add support for `--file` flag in edit/track commands
- [ ] Test across flip, obsidian, logseq, dendron brain types

### Phase 2: VS Code Integration
- [ ] Create `.vscode/tasks.json` with global tasks
- [ ] Add shell script helpers for context detection and fallback logic
- [ ] Implement context-aware task definitions with `when` clauses
- [ ] Test task execution in different file contexts

### Phase 3: Polish
- [ ] Add keybindings configuration (optional)
- [ ] Create user documentation
- [ ] Test fallback behavior edge cases
- [ ] Performance optimization if needed

---

## File Structure

```
flip/
├── .vscode/
│   ├── tasks.json              # VS Code task definitions
│   └── extensions.json         # Optional: recommended extensions
├── docs/
│   ├── VSCODE_TASKS_PLAN.md   # This file
│   └── VSCODE_TASKS_SETUP.md  # User setup guide
├── scripts/
│   └── vscode-helpers.sh       # Shell helpers for context detection
└── internal/commands/
    ├── context_detect.go       # NEW: Context detection command
    └── search.go               # NEW or ENHANCE: Universal search
```

---

## Example Task Definition (Preview)

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Flip: Menu",
      "type": "shell",
      "command": "flip",
      "args": ["menu"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated"
      },
      "group": {
        "kind": "test",
        "isDefault": true
      }
    },
    {
      "label": "Flip: Universal Search",
      "type": "shell",
      "command": "flip",
      "args": ["search", "--interactive"],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated"
      }
    },
    {
      "label": "Flip: Add Variant to Exercise",
      "type": "shell",
      "command": "bash",
      "args": [
        "-c",
        "flip context-detect '${file}' | grep -q 'exercise' && flip exercise edit --file '${file}' || flip search --type=exercise --interactive"
      ],
      "presentation": {
        "reveal": "always",
        "panel": "dedicated"
      },
      "runOptions": {
        "runOn": "folderOpen"
      }
    }
  ]
}
```

---

## Notes

- All tasks use `reveal: always` to show output immediately
- Context detection happens via shell piping to keep tasks lightweight
- Fallback behavior handled with bash conditionals
- Tasks are brain-type agnostic through YAML frontmatter detection
- Terminal presentation supports both interactive input (prompts) and output

---

## References

- [flip Command Structure](../README.md)
- [VS Code Tasks Documentation](https://code.visualstudio.com/docs/editor/tasks)
- [Brain Type Detection](../internal/brain/detection.go)
