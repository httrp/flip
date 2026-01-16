# Task Browser WebView Panel

## Overview

The Task Browser is a new VS Code WebView Panel feature (v0.2.5+) that provides an interactive way to browse, filter, and manage tasks across all your brains.

## Features

- **Dual-Axis Filtering**: Filter tasks by scope (All/My) and status (All/Open/In Progress/Done)
- **Real-time Display**: See task counts and instantly filtered results
- **Task Navigation**: Click any task to open it in the editor at the exact line
- **Rich Task Metadata**: Display priority, due dates, tags, frog status, and organization
- **Brain Awareness**: See which brain each task belongs to
- **Keyboard Shortcut**: `Ctrl+Alt+B` (or `Cmd+Alt+B` on macOS)

## Backend Integration

The WebView Panel uses two new CLI commands for data retrieval:

### `flip task browse --json`
Returns all tasks with full metadata as JSON, ready for WebView rendering.

```bash
flip task browse --json
```

Example output:
```json
{
  "success": true,
  "command": "task-browse",
  "data": {
    "tasks": [
      {
        "description": "Review quarterly goals",
        "status": "open",
        "priority": "high",
        "due": "2026-01-20",
        "tags": ["planning", "quarterly"],
        "frog": true,
        "path": "/path/to/brain/tasks.md",
        "rel_path": "tasks.md",
        "line": 42,
        "brain_name": "log",
        "brain_type": "logseq",
        "organization": "PERSONAL"
      }
    ],
    "total_count": 1
  }
}
```

### `flip task list --json [filters]`
Returns filtered tasks with optional status, priority, or frog filters.

```bash
flip task list --json --status open --priority high
```

## UI Components

The WebView Panel includes:

- **Control Bar**: Scope and Status filter dropdowns with refresh button
- **Task List**: Scrollable list of filtered tasks with rich display
- **Task Items**: 
  - Icons for status, priority, and frog status
  - Task description with due date
  - Brain name and file location
  - Tags (if available)

## Implementation Details

### Files Added/Modified

**New Files:**
- `vscode-extension/src/ui/taskBrowser.ts` - WebView Panel implementation
- `vscode-extension/src/commands/task-browse.ts` - Command handler

**Modified Files:**
- `vscode-extension/src/flip-client.ts` - Added `browseTasksJson()` and `listTasksJson()` methods
- `vscode-extension/src/extension.ts` - Registered `flip.taskBrowse` command
- `vscode-extension/package.json` - Added command and keybinding

**Backend:**
- `internal/commands/task_browse_cmd.go` - Added `--json` flag to `task browse`
- `internal/commands/task_list.go` - Added `--json` flag to `task list`

### Message Protocol

The WebView communicates with the extension via VS Code's message API:

```typescript
// From WebView to Extension
vscode.postMessage({
  command: 'filter',
  scope: 'all' | 'my',
  status: 'all' | 'open' | 'in-progress' | 'done'
});

vscode.postMessage({
  command: 'openTask',
  task: { path: string, line: number }
});

vscode.postMessage({ command: 'refresh' });

// From Extension to WebView
webview.postMessage({ command: 'loading' });
webview.postMessage({ command: 'error', message: string });
```

## Usage

1. Open the Task Browser with `Ctrl+Alt+B` (Windows/Linux) or `Cmd+Alt+B` (macOS)
2. Select filters from the dropdown menus:
   - **Scope**: Choose between "All Tasks" or "My Tasks"
   - **Status**: Choose between "All", "Open", "In Progress", or "Done"
3. Click on any task to open it in the editor
4. Use the "🔄 Refresh" button to reload tasks from disk

## Future Enhancements

- [ ] Sorting options (by due date, priority, brain)
- [ ] Search/filter by text
- [ ] Inline task editing
- [ ] Task creation from the panel
- [ ] Real-time file watching for auto-refresh
- [ ] Export task lists
- [ ] Task statistics and analytics

## Styling

The WebView Panel uses VS Code's built-in color scheme variables for consistency:

- `--vscode-editor-background` - Background color
- `--vscode-editor-foreground` - Text color
- `--vscode-button-background` - Button color
- `--vscode-list-hoverBackground` - Hover effect
- `--vscode-descriptionForeground` - Secondary text

This ensures the panel looks native to the user's VS Code theme.
