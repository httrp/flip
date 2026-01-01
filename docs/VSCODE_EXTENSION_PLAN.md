# VS Code Extension Plan

> Branch: `feat/vscode-integration`
> Status: Planning Phase
> Goal: Native VS Code integration for flip with proper UX

## Overview

The VS Code extension provides a native UI layer for flip, using VS Code's input dialogs, quick picks, and notifications instead of terminal-based interaction.

**Key Principle:** The extension is a thin UI layer. All logic stays in flip.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    VS Code Extension                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │  Commands   │  │  UI Layer   │  │  Status Bar     │  │
│  │  (palette)  │  │  (dialogs)  │  │  (brain info)   │  │
│  └──────┬──────┘  └──────┬──────┘  └────────┬────────┘  │
│         │                │                   │           │
│         └────────────────┴───────────────────┘           │
│                          │                               │
│                    ┌─────┴─────┐                         │
│                    │  flip CLI │  (subprocess)           │
│                    └───────────┘                         │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  flip CLI   │  (installed binary)
                    │  --json     │
                    │  --non-int  │
                    └─────────────┘
```

---

## Phase 1: flip CLI Enhancements

Before building the extension, flip needs these capabilities:

### 1.1 JSON Output Mode

All commands need `--json` flag for structured output:

```bash
# Current (human-readable)
flip journal
# → Opens editor, prints "Journal created: /path/to/file.md"

# New (machine-readable)
flip journal --json
# → {"status": "created", "path": "/path/to/2025-12-22.md", "brain": "log"}
```

**Commands needing JSON output:**
- [ ] `flip journal --json`
- [ ] `flip note --json`
- [ ] `flip quicknote --json`
- [ ] `flip meeting-note --json`
- [ ] `flip brain list --json`
- [ ] `flip brain switch --json`
- [ ] `flip status --json`
- [ ] `flip search --json`
- [ ] `flip recent --json`

### 1.2 Non-Interactive Mode

Commands need `--non-interactive` or accept all params via flags:

```bash
# Current (interactive prompts)
flip note
# → Prompts for title, tags, etc.

# New (all params via flags)
flip note --title "My Note" --tags "work,project" --json
# → Creates note without prompts, returns JSON
```

**Required flags per command:**
| Command | Required Flags |
|---------|---------------|
| `flip note` | `--title`, `--tags`, `--brain` |
| `flip meeting-note` | `--title`, `--attendees`, `--brain` |
| `flip quicknote` | `--content`, `--brain` |
| `flip brain switch` | `--name` |

### 1.3 Discovery Command

New command for extension to discover capabilities:

```bash
flip vscode-info --json
# → {
#     "version": "1.0.0",
#     "brains": [...],
#     "activeBrain": "log",
#     "activeWorkspace": "default",
#     "commands": ["journal", "note", "quicknote", ...]
#   }
```

---

## Phase 2: Extension Foundation

### 2.1 Project Structure

```
flip/
├── vscode-extension/          # Extension subfolder
│   ├── src/
│   │   ├── extension.ts       # Entry point
│   │   ├── flip-client.ts     # CLI wrapper
│   │   ├── commands/          # Command implementations
│   │   │   ├── journal.ts
│   │   │   ├── note.ts
│   │   │   └── ...
│   │   ├── ui/                # UI helpers
│   │   │   ├── input.ts       # Input dialogs
│   │   │   └── quickpick.ts   # Quick picks
│   │   └── status-bar.ts      # Status bar item
│   ├── package.json           # Extension manifest
│   ├── tsconfig.json
│   └── README.md
├── cmd/flip/
├── internal/
└── ...
```

### 2.2 flip-client.ts (Core)

```typescript
interface FlipResult<T> {
  success: boolean;
  data?: T;
  error?: string;
}

class FlipClient {
  private timeout = 10000; // 10s default
  
  async execute<T>(command: string, args: string[]): Promise<FlipResult<T>> {
    // Run: flip <command> <args> --json
    // Parse JSON output
    // Handle errors gracefully
  }
  
  async getInfo(): Promise<FlipInfo> { ... }
  async createJournal(brain?: string): Promise<{path: string}> { ... }
  async createNote(params: NoteParams): Promise<{path: string}> { ... }
  // ...
}
```

### 2.3 Commands (package.json)

```json
{
  "contributes": {
    "commands": [
      { "command": "flip.journal", "title": "Flip: Open/Create Journal" },
      { "command": "flip.note", "title": "Flip: New Note" },
      { "command": "flip.quicknote", "title": "Flip: Quick Note" },
      { "command": "flip.meetingNote", "title": "Flip: New Meeting Note" },
      { "command": "flip.search", "title": "Flip: Search" },
      { "command": "flip.recent", "title": "Flip: Recent Files" },
      { "command": "flip.status", "title": "Flip: Show Status" },
      { "command": "flip.switchBrain", "title": "Flip: Switch Brain" }
    ]
  }
}
```

---

## Phase 3: UI Implementation

### 3.1 Example: New Note Flow

```typescript
// commands/note.ts
async function createNote() {
  // 1. Get title via VS Code input
  const title = await vscode.window.showInputBox({
    prompt: 'Note title',
    placeHolder: 'My new note'
  });
  if (!title) return;

  // 2. Get tags via VS Code input
  const tags = await vscode.window.showInputBox({
    prompt: 'Tags (comma-separated)',
    placeHolder: 'work, project'
  });

  // 3. Brain selection (if multiple)
  const brains = await flipClient.getBrains();
  let brain = brains.active;
  if (brains.list.length > 1) {
    brain = await vscode.window.showQuickPick(brains.list, {
      placeHolder: 'Select brain'
    });
  }

  // 4. Create note via flip CLI
  const result = await flipClient.createNote({ title, tags, brain });
  
  // 5. Open the created file
  if (result.success) {
    const doc = await vscode.workspace.openTextDocument(result.data.path);
    await vscode.window.showTextDocument(doc);
  } else {
    vscode.window.showErrorMessage(`Failed to create note: ${result.error}`);
  }
}
```

### 3.2 Status Bar

```
┌─────────────────────────────────────────────────────────┐
│  [Flip: log 🧠]                                          │
└─────────────────────────────────────────────────────────┘
         ↑
    Click to switch brain or see status
```

### 3.3 Keyboard Shortcuts (Defaults)

| Shortcut | Command |
|----------|---------|
| `Ctrl+Alt+J` | Journal |
| `Ctrl+Alt+N` | New Note |
| `Ctrl+Alt+Q` | Quick Note |
| `Ctrl+Alt+M` | Meeting Note |
| `Ctrl+Alt+F` | Search |

---

## Phase 4: Advanced Features (Future)

- [ ] Tree view for brain contents
- [ ] Preview panel for notes
- [ ] Auto-complete for tags
- [ ] Task integration (flip tasks in VS Code Problems panel?)
- [ ] Git status indicator per brain
- [ ] Context menu entries (right-click on files)

---

## Design Decisions

### Extension Location: Monorepo (subfolder)

**Decision:** `flip/vscode-extension/` subfolder

**Rationale:**
- Easier to keep in sync with flip changes
- Single PR can update both CLI and extension
- Shared documentation
- One CI pipeline

**Trade-off:** Slightly more complex release process

### Version Coupling

**Decision:** Extension major version matches flip major version

**Example:**
- flip v1.2.3 + extension v1.x.x = compatible
- flip v2.0.0 + extension v1.x.x = may break

Extension will check flip version on startup and warn if incompatible.

### Distribution

**Phase 1:** Manual VSIX install via `flip vscode install`
**Phase 2:** VS Code Marketplace (public)

---

## Error Handling

| Scenario | Extension Behavior |
|----------|-------------------|
| flip not in PATH | Show install instructions |
| flip version too old | Show update instructions |
| No workspace/brain | Prompt to initialize |
| Command fails | Show error notification with details |
| Timeout (>10s) | Cancel with message |

---

## Testing Strategy

1. **Unit tests:** flip-client.ts mocking
2. **Integration tests:** Real flip CLI in test environment
3. **E2E tests:** VS Code Extension Testing framework
4. **Manual testing:** Checklist before release

---

## Implementation Order

### Sprint 1: flip CLI (1-2 days)
- [ ] Add `--json` flag infrastructure
- [ ] Implement for: journal, note, quicknote, status
- [ ] Add `flip vscode-info` command

### Sprint 2: Extension Foundation (1 day)
- [ ] Scaffold extension with `yo code`
- [ ] Implement flip-client.ts
- [ ] Basic journal command (proof of concept)

### Sprint 3: Core Commands (1-2 days)
- [ ] Note creation with full UI
- [ ] Quick note
- [ ] Meeting note
- [ ] Search with results picker
- [ ] Recent files

### Sprint 4: Polish (1 day)
- [ ] Status bar
- [ ] Error handling
- [ ] Settings
- [ ] Documentation

### Sprint 5: Distribution
- [ ] VSIX build in CI
- [ ] `flip vscode install` installs extension
- [ ] (Optional) Marketplace publishing

---

## Open Questions

1. **Multi-root workspaces:** How handle multiple brains in one VS Code window?
2. **Remote development:** Does flip work in SSH/Container/WSL scenarios?
3. **Settings sync:** Should extension settings be per-workspace or global?

---

## References

- [VS Code Extension API](https://code.visualstudio.com/api)
- [Extension Guidelines](https://code.visualstudio.com/api/references/extension-guidelines)
- [Publishing Extensions](https://code.visualstudio.com/api/working-with-extensions/publishing-extension)
