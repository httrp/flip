# Menu UX Issues - Action Plan

**Date:** 2025-10-24  
**Reporter:** User Feedback  
**Priority:** High - UX blockers

---

## 🐛 Identified Issues

### 1. Task Browser - Mark Done (d) doesn't persist ⚠️ CRITICAL
**Status:** 🔴 Broken  
**File:** `internal/commands/task_browser.go:130`

**Problem:**
```go
case "d":
    // Toggle done
    if m.cursor < len(m.filteredTasks) {
        task := m.filteredTasks[m.cursor]
        if task.Status == tasks.StatusDone {
            task.Status = tasks.StatusOpen
        } else {
            task.Status = tasks.StatusDone
        }
        // TODO: Save back to file  ← NOT IMPLEMENTED!
    }
```

**Impact:** Users can't actually mark tasks as done from the browser.

**Fix Required:**
- Implement `saveTaskToFile()` function
- Update markdown file with new status
- Handle different status markers (TODO, DONE, IN-PROGRESS, etc.)
- Preserve file structure and other content

---

###

 2. Old Task Menu in Main Menu ⚠️ CRITICAL
**Status:** 🔴 Outdated  
**File:** `internal/commands/menu.go:1626`

**Problem:** 
- `runTaskBrowseMenu()` uses old list-based approach
- New TUI browser (`task_browser.go`) exists but not integrated
- Menu shows "Browse Tasks" but calls old function

**Current Flow:**
```
Main Menu → Browse → Tasks → runTaskBrowseMenu() → Old list view
```

**Should Be:**
```
Main Menu → Browse → Tasks → runTaskBrowser() → New TUI
```

**Fix Required:**
- Replace `runTaskBrowseMenu()` with call to new browser
- Or remove submenu entirely, go directly to TUI

---

### 3. Status Header Repeats Unnecessarily 🟡 ANNOYING
**Status:** 🟡 UX Issue  
**Files:** Multiple menu functions

**Problem:**
```go
func runBrowseSearchMenu() error {
    fmt.Println()
    displayStatusHeader()  // ← Shown again
    fmt.Println()
    // ...
}
```

Every submenu calls `displayStatusHeader()` again, even though it was just shown.

**Impact:** Header appears multiple times as user navigates menus.

**Fix Required:**
- Only show header once at start
- Remove from submenu functions
- Add option to refresh if needed

---

### 4. Exit Doesn't Always Exit 🟡 CONFUSING
**Status:** 🟡 UX Issue  
**Files:** Multiple menu functions

**Problem:**
Some menu items labeled "Back" or "Exit" don't actually exit, they return to parent menu.

**Examples:**
- Exit in submenu → Goes to main menu (should say "Back")
- Back in main menu → Should say "Exit"

**Fix Required:**
- Consistent labeling: "Back" for submenus, "Exit" for main
- Actual exit should terminate program
- Visual indicator of menu depth

---

### 5. Too Many Top-Level Options 🟢 OPTIMIZATION
**Status:** 🟢 Enhancement  
**File:** `internal/commands/menu.go:87`

**Current Main Menu:**
```
1. Quickstart Guide
2. Browse & Search
3. Create New
4. Manage Resources
5. Switch Context
6. Show Status
7. Help & Commands
8. Exit
```

**Problem:** 8 options = too much cognitive load

**Proposed Grouping:**
```
1. 📝 Create (Note/Meeting/Journal)
2. 📚 Browse (Tasks/Notes/Meetings)
3. ⚙️  Manage (Brains/Workspace/Templates)
4. ℹ️  Help & Status
5. 🚪 Exit
```

---

### 6. Missing Description Footer 🟡 MISSING FEATURE
**Status:** 🟡 Regression  
**File:** `internal/commands/menu.go:50-60`

**Problem:** Old menu had dynamic footer showing description of highlighted item.

**Old Behavior:**
```
  Create Note
▸ Create Meeting      <-- Selected
  Create Journal

[Create a new meeting note with agenda and participants]
```

**Current:** No description shown.

**Fix Required:**
- Add `Details` template to `promptui.SelectTemplates`
- Show `Description` field dynamically
- Format nicely with word wrap

---

### 7. Missing Command Hints 🟢 ENHANCEMENT
**Status:** 🟢 Feature Request

**Request:** Show CLI command behind menu items

**Example:**
```
▸ Create Meeting Note     (flip meeting-note)
  Create Journal Entry    (flip journal)
  Browse Tasks            (flip task browse)
```

**Benefits:**
- Users learn CLI commands
- Bridge between TUI and CLI
- Power users can skip menu

**Implementation:**
- Add `Command` field to menu items
- Display in gray/dimmed after label
- Make optional/configurable

---

## 📋 Action Plan

### Phase 1: Critical Fixes (Today)
- [ ] **Issue #1**: Implement `saveTaskToFile()` in task browser
- [ ] **Issue #2**: Integrate new task browser into menu
- [ ] **Issue #4**: Fix Exit/Back labeling consistency

### Phase 2: UX Improvements (This Week)
- [ ] **Issue #3**: Remove duplicate status headers
- [ ] **Issue #6**: Add description footer back
- [ ] **Issue #5**: Consolidate main menu options

### Phase 3: Enhancements (Next Week)
- [ ] **Issue #7**: Add command hints to menu items
- [ ] Add keyboard shortcuts to descriptions
- [ ] Improve menu navigation (breadcrumbs?)

---

## 🔧 Technical Details

### Task Save Implementation

```go
// saveTaskToFile updates a task's status in its markdown file
func saveTaskToFile(task *tasks.Task) error {
    // 1. Read file
    content, err := os.ReadFile(task.FilePath)
    if err != nil {
        return fmt.Errorf("failed to read file: %w", err)
    }

    // 2. Find task line
    lines := strings.Split(string(content), "\n")
    if task.LineNumber < 1 || task.LineNumber > len(lines) {
        return fmt.Errorf("invalid line number: %d", task.LineNumber)
    }

    // 3. Update status marker
    oldLine := lines[task.LineNumber-1]
    newLine := updateTaskStatus(oldLine, task.Status)
    lines[task.LineNumber-1] = newLine

    // 4. Write back
    newContent := strings.Join(lines, "\n")
    return os.WriteFile(task.FilePath, []byte(newContent), 0644)
}

// updateTaskStatus replaces status marker in task line
func updateTaskStatus(line string, newStatus tasks.TaskStatus) string {
    // Handle: - [ ], - [x], - [>], - [~], - [-]
    switch newStatus {
    case tasks.StatusOpen:
        return replaceTaskMarker(line, "[ ]")
    case tasks.StatusDone:
        return replaceTaskMarker(line, "[x]")
    case tasks.StatusInProgress:
        return replaceTaskMarker(line, "[>]")
    case tasks.StatusDeferred:
        return replaceTaskMarker(line, "[~]")
    case tasks.StatusCancelled:
        return replaceTaskMarker(line, "[-]")
    default:
        return line
    }
}

// replaceTaskMarker finds and replaces task checkbox
func replaceTaskMarker(line, newMarker string) string {
    // Regex: - \[.\] or * \[.\] or + \[.\]
    re := regexp.MustCompile(`^(\s*[-*+]\s+)\[[^\]]\]`)
    return re.ReplaceAllString(line, "${1}"+newMarker)
}
```

### Menu Template with Description

```go
func createMenuItemSelectTemplates() *promptui.SelectTemplates {
    return &promptui.SelectTemplates{
        Label:    "{{ . }}",
        Active:   "▸ {{ .Label | cyan | bold }}",
        Inactive: "  {{ .Label }}",
        Selected: "{{ .Label | green | bold }}",
        Details:  "\n{{ \"─\" | faint }}{{ \"─\" | faint }}{{ \"─\" | faint }}\n{{ .Description | faint }}", // ← ADD THIS
    }
}
```

### Command Hints

```go
type MenuItem struct {
    Label       string
    Description string
    Command     string  // ← ADD THIS
    Action      func() error
}

// Template:
Active:   "▸ {{ .Label | cyan | bold }}  {{ if .Command }}{{ .Command | faint }}{{ end }}",
```

---

## ✅ Testing Checklist

After implementing fixes:

- [ ] Task browser 'd' key persists changes to file
- [ ] File content preserved (no data loss)
- [ ] Special characters in tasks handled correctly
- [ ] Multiple status changes work in sequence
- [ ] Main menu links to new task browser TUI
- [ ] Exit from main menu terminates program
- [ ] Back from submenus returns to parent
- [ ] Status header only shows once per navigation path
- [ ] Description footer updates on item change
- [ ] Command hints display correctly
- [ ] Menu is less cluttered (fewer top-level items)
- [ ] All keyboard shortcuts work as expected

---

**Generated:** 2025-10-24  
**Priority:** High - Multiple UX blockers  
**Estimated Effort:** 4-6 hours for Phase 1
