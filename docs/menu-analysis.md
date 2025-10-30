# Flip Menu Structure Analysis

## CLI Commands Overview

### Root Commands
- `flip` / `flip menu` - Interactive menu (default)
- `flip status` - Show workspace/brain overview
- `flip quickstart` - Guided quick setup
- `flip intro` - Introduction to flip

### Workspace Commands (`flip workspace`)
- `create <name>` - Create new workspace
- `list` - List all workspaces
- `switch <name>` - Switch active workspace
- `remove <name>` - Remove workspace
- `rename <old> <new>` - Rename workspace
- `repair` - Repair workspace configuration

### Brain Commands (`flip brain`)
- `new` - Create new brain (alias to `flip new brain`)
- `add <path>` - Add existing brain directory
- `list` - List brains in active workspace
- `remove <name>` - Remove brain from workspace
- `set-default <name>` - Set default brain
- `rename <old> <new>` - Rename brain
- `repair` - Repair brain configurations
- `check` - Check brain health
- `init <path>` - Initialize directory as brain
- `scan [path]` - Scan for existing brains
- `recent` - Show recent notes
- `search <query>` - Search notes
- `git-status [--all]` - Show git status
- `git-log [-n count] [--all]` - Show git commit history

### Content Creation Commands
- `flip new` - Create new brain (guided setup)
- `flip note` - Create new note
- `flip quicknote` - Create quick note
- `flip meeting` - Create new meeting note
- `flip journal` - Create journal entry
- `flip task` - Task management

### Task Commands (`flip task`)
- `browse` - Interactive task browser (TUI)
- `list [--today|--week|--overdue]` - List tasks
- `new` - Create new task
- `done <id>` - Mark task as done
- `start <id>` - Start working on task
- `update <id>` - Update task
- `search <query>` - Search tasks
- `stats` - Show task statistics

### Template Commands (`flip template`)
- Template management commands

## Current Menu Structure

### Main Menu (`runInteractiveMenu`)
1. **Quickstart** → `runQuickstart()`
2. **Browse & Search** → `runBrowseSearchMenu()`
3. **Create New** → `runCreateNewMenu()`
4. **Manage Resources** → `runManageResourcesMenu()`
5. **Switch Context** → `runSwitchContextMenu()`
6. **Status** → `runStatus()`
7. **Help** → Show commands
8. **Exit** → Check uncommitted changes & exit

### Browse & Search Menu (`runBrowseSearchMenu`)
1. **Recent notes** → `showRecentNotes(20)`
2. **Search notes** → `searchNotes(query, searchContent)`
3. **Tasks** → `runTaskBrowseMenu()`
4. **Back**

### Create New Menu (`runCreateNewMenu`)
1. **Create note** → `runCreateNote()`
2. **Create meeting** → `runCreateMeeting()`
3. **Create journal** → `runCreateJournal()`
4. **Create task** → `runCreateTask()`
5. **Back**

### Manage Resources Menu (`runManageResourcesMenu`)
1. **Manage Workspaces** → `runEditWorkspaceMenu()`
2. **Manage Brains** → `runManageBrainsMenu()`
3. **View/Edit workspace config** → `runViewWorkspaceConfig()`
4. **Templates** → `runEditManageMenu()`
5. **Git: Show status** → `runBrainGitStatus(false)`
6. **Git: Pull updates** → `checkRemoteUpdatesOnStart()`
7. **Git: Commit changes** → `checkUncommittedChangesOnExit()`
8. **Back**

### Manage Brains Menu (`runManageBrainsMenu`)
1. **Create new brain** → `runNewBrain()`
2. **Add existing brain** → `runDirectoryInit(path, "", false)`
3. **Scan for brains** → `runScan(path)`
4. **View/Edit brains** → `runEditBrainMenu()`
5. **Back**

### Edit Brain Menu (`runEditBrainMenu`)
1. **View brain details** → Show config & options
2. **Rename brain** → `runBrainRename(old, new)`
3. **Set default brain** → `runBrainSetDefault(name)`
4. **Repair brains** → `runBrainRepair()`
5. **Remove brain** → `runBrainRemove(name)`
6. **Show Git status** → `runBrainGitStatus(false)`
7. **Show Git commit history** → `runBrainGitLog(10, false)`
8. **Commit changes** → `checkUncommittedChangesOnExit()`
9. **Pull updates** → `checkRemoteUpdatesOnStart()`
10. **Back**

### Switch Context Menu (`runSwitchContextMenu`)
1. **Switch workspace** → `runSwitchWorkspaceMenu()`
2. **Switch default brain** → Select brain prompt
3. **Back**

### Edit Workspace Menu (`runEditWorkspaceMenu`)
1. **Rename workspace** → `runWorkspaceRename(old, new)`
2. **Repair workspace** → `runWorkspaceRepair()`
3. **Remove workspace** → `runWorkspaceRemove(name)`
4. **Back**

### Task Browse Menu (`runTaskBrowseMenu`)
1. **All open tasks** → `runListTasks(filter)`
2. **Tasks due today** → `runListTasks(filter)`
3. **Tasks due this week** → `runListTasks(filter)`
4. **Overdue tasks** → `runListTasks(filter)`
5. **High priority tasks** → `runListTasks(filter)`
6. **Task statistics** → `runTaskStats()`
7. **Back**

## Technical Implementation

### Menu System
- Uses `promptui` library for interactive selection
- Template-based rendering with `createMenuItemSelectTemplates()`
- Dynamic menu sizing with `calculateMenuSize(itemCount)`
- Status header displayed with `displayStatusHeader()`

### Menu Item Structure
```go
menuItems := []struct {
    Label       string
    Description string
    Action      func() error
}{
    // ...
}
```

### Navigation Pattern
- Each menu function returns `error`
- Navigation by returning the next menu function
- Back buttons call parent menu function
- Exit handled by returning `nil` or error

### Helper Functions
- `createMenuItemSelectTemplates()` - Menu item templates
- `createSimpleSelectTemplates()` - Simple selection templates  
- `calculateMenuSize(itemCount)` - Adaptive menu height
- `getTerminalSize()` - Terminal dimensions
- `truncateString(s, maxLen)` - String truncation

### Status Header
- Displays active workspace, default brain, brain count
- Shows Git status (branch, changes, last commit)
- Formatted with lipgloss styling

## Issues & Observations

### Current Problems
1. **Deep nesting** - Git functions buried 4+ levels deep
2. **Duplicate functionality** - Git options in multiple places
3. **Inconsistent grouping** - Related features scattered
4. **Unclear naming** - "Manage Resources" vs "Manage Brains"
5. **Missing quick actions** - No quick create from main menu
6. **Unused menus** - `runCreateAddMenu()` exists but not used
7. **Brain menu confusion** - Multiple brain management menus

### Navigation Issues
- Too many clicks to reach common operations
- Back navigation sometimes inconsistent
- "Edit Manage Menu" confusing name/purpose
- Template management buried in wrong section

### Git Integration Issues (FIXED)
- ✅ Authentication prompts removed
- ✅ ahead/behind checks disabled in status
- ✅ Git functions now in Manage Resources
- Still in Edit Brain menu (duplicate)

## Recommendations

### Immediate Improvements
1. Flatten menu structure (reduce nesting)
2. Group related operations logically
3. Add quick actions to main menu
4. Remove duplicate Git entries
5. Rename confusing menu items
6. Consolidate brain management

### Proposed Main Menu Structure
```
Main Menu
├─ Quick Actions
│  ├─ Create Note
│  ├─ Create Meeting
│  ├─ Create Journal
│  └─ Create Task
├─ Browse & Search
│  ├─ Recent Notes
│  ├─ Search Notes
│  └─ Browse Tasks
├─ Workspaces
│  ├─ Switch Workspace
│  ├─ Create Workspace
│  └─ Manage Workspaces
├─ Brains
│  ├─ Add/Create Brain
│  ├─ Switch Default Brain
│  └─ Manage Brains
├─ Git Operations
│  ├─ Status
│  ├─ Commit Changes
│  └─ Pull/Push
├─ Settings
│  ├─ Templates
│  ├─ View Config
│  └─ Preferences
├─ Status
├─ Help
└─ Exit
```

### Benefits
- 2-3 levels max depth
- Clear categorization
- Quick access to common operations
- Consistent naming
- Better discoverability
