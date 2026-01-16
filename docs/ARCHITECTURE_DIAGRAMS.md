# WebView Task Browser - Architecture Diagram

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        VS Code Extension                             │
│ ┌──────────────────────────────────────────────────────────────────┐ │
│ │                    Extension Host (extension.ts)                │ │
│ │  • Command: flip.taskBrowse                                     │ │
│ │  • Keyboard: Ctrl+Alt+B                                         │ │
│ └────────────────────────┬─────────────────────────────────────────┘ │
│                          │                                             │
│                          ▼                                             │
│ ┌──────────────────────────────────────────────────────────────────┐ │
│ │              TaskBrowserPanel (taskBrowser.ts)                 │ │
│ │  • WebView Panel Management                                    │ │
│ │  • Message Protocol Handler                                    │ │
│ │  • Task Data State Management                                  │ │
│ └────────────────────────┬─────────────────────────────────────────┘ │
│                          │                                             │
│                          ▼                                             │
│ ┌──────────────────────────────────────────────────────────────────┐ │
│ │              FlipClient (flip-client.ts)                       │ │
│ │  • browseTasksJson()                                           │ │
│ │  • listTasksJson(filters?)                                     │ │
│ └────────────────────────┬─────────────────────────────────────────┘ │
└────────────────────────┼─────────────────────────────────────────────┘
                         │
              ┌──────────┴──────────┐
              │                     │
              ▼                     ▼
         ┌─────────────┐      ┌────────────────┐
         │ CLI Binary  │      │ File System    │
         │  (flip)     │      │                │
         └──────┬──────┘      └────────────────┘
                │
         ┌──────┴──────┐
         │             │
         ▼             ▼
   ┌──────────┐  ┌──────────────┐
   │  task    │  │ task         │
   │  browse  │  │ list         │
   │ --json   │  │ --json       │
   └──────┬───┘  └──────┬───────┘
          │             │
          └──────┬──────┘
                 │
              JSON Output
                 │
                 ▼
    ┌────────────────────────┐
    │  TaskInfo[] + Metadata │
    │                        │
    │  • description         │
    │  • status              │
    │  • priority            │
    │  • due                 │
    │  • tags                │
    │  • frog                │
    │  • path/rel_path       │
    │  • line                │
    │  • brain_name          │
    │  • brain_type          │
    │  • organization        │
    │  • project             │
    │  • context             │
    └────────────────────────┘
                 ▲
                 │
              FlipResult<TasksResult>
                 │
                 ▼
    ┌──────────────────────────┐
    │   WebView HTML/CSS/JS    │
    │                          │
    │  ┌──────────────────┐   │
    │  │   Control Bar    │   │
    │  │  [Scope][Status] │   │
    │  └──────────────────┘   │
    │                          │
    │  ┌──────────────────┐   │
    │  │  Task List View  │   │
    │  │  • Filtered      │   │
    │  │  • Formatted     │   │
    │  │  • Clickable     │   │
    │  └──────────────────┘   │
    └──────────────────────────┘
```

## Message Flow Diagram

```
User Action
    │
    ├─► Change Scope/Status Filter
    │   │
    │   └─► postMessage({ command: 'filter', scope, status })
    │       │
    │       ▼
    │   _handleMessage() detects filter change
    │       │
    │       ├─► Update _scopeFilter and _statusFilter
    │       ├─► Call _applyFilters() - client-side only
    │       └─► Call _updateWebview() - re-render
    │
    ├─► Click on Task
    │   │
    │   └─► postMessage({ command: 'openTask', task })
    │       │
    │       ▼
    │   _handleMessage() detects openTask
    │       │
    │       └─► _openTask(task)
    │           │
    │           ├─► vscode.workspace.openTextDocument()
    │           ├─► vscode.window.showTextDocument()
    │           └─► Move cursor to task.line
    │
    └─► Click Refresh Button
        │
        └─► postMessage({ command: 'refresh' })
            │
            ▼
        _handleMessage() detects refresh
            │
            └─► _loadTasks()
                │
                ├─► postMessage({ command: 'loading' })
                ├─► client.browseTasksJson()
                │   │
                │   └─► flip task browse --json
                ├─► Update _allTasks
                ├─► _applyFilters()
                └─► _updateWebview()
```

## Data Flow - Filter Application

```
Raw Tasks from CLI (438 total)
        │
        ▼
    _allTasks = [... all 438 ...]
        │
        ▼
    _applyFilters()
        │
        ├─► Filter by statusFilter (if not "all")
        │   │
        │   └─► tasks.status === statusFilter
        │
        ├─► Filter by scopeFilter (if not "all")
        │   │
        │   └─► Additional filtering logic
        │
        └─► Result: _filteredTasks
            │
            ▼
        _updateWebview()
            │
            └─► _getHtmlContent()
                │
                ├─► Map _filteredTasks to HTML
                ├─► Apply styling
                ├─► Add event listeners
                └─► Render in WebView Panel
```

## Component Interaction

```
┌──────────────────────────────────┐
│   VS Code Editor                  │
│  ┌────────────────────────────┐  │
│  │  TaskBrowserPanel (Active) │  │
│  │                            │  │
│  │  ┌──────────────────────┐ │  │
│  │  │ Control Bar          │ │  │
│  │  │ [All▼][All▼][Refresh]│ │  │
│  │  └──────────────────────┘ │  │
│  │                            │  │
│  │  ┌──────────────────────┐ │  │
│  │  │ Task: Review goals   │ │  │
│  │  │ 🐸 ⏫ 📅 2026-01-20│ │  ◄────┐
│  │  │ log / tasks.md:42    │ │  │   │
│  │  │ Tags: planning       │ │  │   │
│  │  └──────────────────────┘ │  │   │
│  │                            │  │   │
│  │  ┌──────────────────────┐ │  │   │
│  │  │ Task: Code review    │ │  │   │
│  │  │ 🔄 📅 2026-01-18    │ │  │   │
│  │  │ log / tasks.md:45    │ │  │   │
│  │  └──────────────────────┘ │  │   │
│  │                            │  │   │
│  │  ... more tasks ...        │  │   │
│  │                            │  │   │
│  └────────────────────────────┘  │   │ Click
│                                    │   │
│  ┌────────────────────────────┐  │   │
│  │  File Editor              │◄─────┘
│  │ tasks.md                  │  │
│  │                           │  │
│  │ 42 | - [ ] Review goals   │  │ (cursor moves here)
│  │    | - 🐸 priority:high   │  │
│  │    | - due:2026-01-20     │  │
│  │                           │  │
│  └────────────────────────────┘  │
└──────────────────────────────────┘
```

## Redux-like State Management

```
TaskBrowserPanel Instance
    │
    ├─► _allTasks: TaskInfo[] = []
    │   (Raw data from CLI)
    │
    ├─► _filteredTasks: TaskInfo[] = []
    │   (After applying filters)
    │
    ├─► _scopeFilter: 'all' | 'my' = 'all'
    │
    ├─► _statusFilter: Status = 'all'
    │
    ├─► State Changes:
    │   ├─► _loadTasks() → Updates _allTasks
    │   ├─► _applyFilters() → Updates _filteredTasks
    │   ├─► _updateWebview() → Re-renders UI
    │   └─► _handleMessage() → Dispatches actions
    │
    └─► _panel.webview
        └─► Subscribes to state changes via _updateWebview()
```

## Error Handling Flow

```
Error Scenarios:
    │
    ├─► CLI fails (browseTasksJson)
    │   └─► OutputJSONError returned
    │       └─► result.success = false
    │           └─► _handleMessage() detects
    │               └─► postMessage({ command: 'error', message })
    │                   └─► WebView shows error UI
    │
    ├─► File not found (openTask)
    │   └─► workspace.openTextDocument() rejects
    │       └─► Try-catch handles
    │           └─► Error is logged (no user-facing error)
    │
    └─► WebView disconnects
        └─► _panel.onDidDispose() triggers
            └─► dispose() cleanup
                └─► _disposables are cleaned up
```

## Performance Considerations

```
Operation           Time        Notes
──────────────────────────────────────────────────
Load all tasks      ~300ms      One CLI call, 438 tasks
Apply filters       <1ms        Client-side, no I/O
Render WebView      ~50ms       DOM manipulation
Total (worst case)  ~350ms      Acceptable for user

Optimization done:
✓ Filters applied client-side (no CLI round-trips)
✓ No debouncing needed for filtering
✓ WebView retained when hidden (retainContextWhenHidden)
✓ Single large JSON fetch better than multiple small calls
```

---

This architecture ensures:
- ✅ Responsive user experience
- ✅ Clean separation of concerns
- ✅ Testable components
- ✅ Minimal overhead
- ✅ Native VS Code integration
