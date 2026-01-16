# Flip v0.2.x Features

## Overview
This document describes the features added in v0.2.0 through v0.2.4, focused on VS Code Extension improvements and Brain Management.

## v0.2.0 - Status as Markdown
**Feature:** Show flip status as a comprehensive markdown document instead of a simple notification.

**Commands:**
- `flip.status` - Opens status as markdown document with:
  - Active workspace and brain info
  - All available brains list
  - Git status for each brain
  - Task and note statistics

**Benefits:**
- Better overview at a glance
- Persistent view (can stay open in editor)
- Rich formatting with emojis and structure

---

## v0.2.1 - About & Help Documentation
**Feature:** Integrated help and about information accessible from VS Code.

**Commands:**
- `flip.about` - Shows version, links, features, and supported brain types
- `flip.help` - Comprehensive quick reference guide with:
  - All available commands with descriptions
  - Keyboard shortcuts
  - Tips and best practices
  - Task priority reference

**Benefits:**
- No need to leave VS Code for help
- Quick access to command reference
- Onboarding information for new users

---

## v0.2.2 - Quick Mark Done
**Feature:** Quickly mark a task as done without prompts.

**Commands:**
- `flip.taskDone` - Directly marks currently selected/open task as done
- Shows confirmation message

**Benefits:**
- Faster workflow for task completion
- No interruption dialogs
- Can be bound to keyboard shortcut

---

## v0.2.3 - Brain Management
**Feature:** Full brain management from VS Code Extension.

**Commands:**
- `flip.manageBrains` - Interactive menu for brain operations
- `flip.listBrains` - Show all brains with status
- `flip.addBrain` - Add existing brain to workspace
- `flip.initBrain` - Create new brain
- `flip.removeBrain` - Remove brain from workspace
- `flip.setDefaultBrain` - Set which brain is active

**Workflow:**
```
User runs flip.manageBrains
  ↓
Shows options: List, Add, Init, Remove, Set Default
  ↓
Selected action prompts for details
  ↓
Runs flip CLI command
  ↓
Refreshes brain list
  ↓
Shows confirmation
```

**Benefits:**
- No need to use CLI for brain management
- Multi-select for choosing brains
- Validation and error handling
- Automatic refresh after operations

---

## v0.2.4 - Manage & Settings
**Feature:** Centralized management and configuration from VS Code.

**Commands:**
- `flip.manage` - Main manage menu with options for:
  - Settings & Configuration
  - Cleanup temporary files
  - Refresh cache
  - Show diagnostics
- `flip.settings` - Configure extension with:
  - Executable path
  - Status bar visibility
  - Icon theme (mixed, color, mono)
  - Default journal type (daily, weekly, combined)
  - Auto-create tasks file setting

**Settings Available:**
```json
{
  "flip.executablePath": "flip",
  "flip.showStatusBar": true,
  "flip.iconTheme": "mixed",
  "flip.defaultJournalType": "daily",
  "flip.autoCreateTasksFile": false
}
```

**Diagnostics:**
Shows system information including:
- Flip CLI status
- Active brain
- Number of brains
- Extension version
- VS Code version
- Platform and architecture
- Current configuration

**Benefits:**
- Easy configuration management
- System diagnostics for troubleshooting
- Cache refresh for clearing stale data
- Cleanup for removing temporary files

---

## Technical Details

### Architecture Changes

1. **New Files:**
   - `vscode-extension/src/commands/brain.ts` - Brain management operations
   - `vscode-extension/src/commands/manage.ts` - Settings and management menu

2. **Updated Files:**
   - `vscode-extension/src/flip-client.ts` - Added `runCommand()` method for generic command execution
   - `vscode-extension/src/extension.ts` - Registered new commands
   - `vscode-extension/package.json` - Added command definitions (v0.2.0→0.2.4)

3. **Key Implementation:**
   - All brain operations use the existing flip CLI via `runCommand()`
   - Settings use VS Code's `WorkspaceConfiguration` API
   - Markdown views use `vscode.workspace.openTextDocument()`
   - Error handling with user-friendly messages

### Version History
- v0.2.0: Status Markdown
- v0.2.1: About & Help
- v0.2.2: Quick Mark Done
- v0.2.3: Brain Management
- v0.2.4: Manage & Settings

---

## Future Enhancements

### Planned (Not Yet Implemented)
1. **Browse** - Task browser WebView Panel
2. **Integrated Settings** - VS Code settings panel instead of command-based
3. **Keyboard Shortcuts** - Configurable keybindings for all commands

### Considerations
- Performance optimization for large workspaces
- Caching of brain list to reduce CLI calls
- WebView-based UI for more complex interactions

---

## Testing Checklist

When testing these features:

- [ ] Status view shows all brains and git status
- [ ] About/Help documentation is readable and complete
- [ ] Mark Done updates task status immediately
- [ ] Can add/init/remove/list brains without errors
- [ ] Settings persist across sessions
- [ ] Diagnostics show correct system info
- [ ] Error messages are clear and actionable
- [ ] RefreshFlipClient() updates brain list correctly

---

## Usage Examples

### Brain Management
```
User opens flip.manageBrains
→ Selects "Init Brain"
→ Enters brain name: "My Project"
→ Asked if should be default
→ Brain created and activated
```

### Settings
```
User opens flip.settings
→ Selects "Icon Theme"
→ Chooses "Color"
→ Setting saved to global VS Code config
→ Restart extension applies changes
```

### Status Check
```
User opens flip.status
→ Markdown document opens with:
   - Active: danobrain
   - 4 other brains available
   - Git status for each brain
   - 47 tasks, 12 exercises, 156 notes
→ Can bookmark this view
```
