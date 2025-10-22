# Flip Task Management - Nächste Schritte

**Stand:** 22. Oktober 2025  
**Commits:** 825e98e (Implementation), ebdb2a8 (Menu Integration)

## ✅ Bereits implementiert (MVP Phase 1 - 85%)

### Core System
- ✅ **Task Data Structures** (`internal/tasks/task.go`)
  - Task, Status, Priority, TaskContext structs
  - Helper functions (IsOverdue, IsDueToday, IsDueThisWeek)
  - Conversion functions (StatusFromCheckbox, PriorityIcon, etc.)

- ✅ **Markdown Parser** (`internal/tasks/parser.go`)
  - Obsidian/Logseq kompatibel
  - Inline Metadata: `- [ ] Task 📅 2025-10-30 ⏫ #tag`
  - Indented Metadata: `key:: value`
  - FormatTask() für Markdown-Export

- ✅ **Brain Scanner** (`internal/tasks/scanner.go`)
  - Rekursives Scannen aller MD-Dateien
  - Multi-Brain Support
  - Context-Tracking (File, Line, Section, Brain)

- ✅ **Task Index & Filtering** (`internal/tasks/filter.go`)
  - Fast lookups (byStatus, byPriority, byTag, byProject, byDueDate)
  - Smart sorting (Overdue → Date → Priority → Created)
  - Statistics & Aggregations

### CLI Commands
- ✅ **flip task list** - Mit Filtern (--status, --priority, --tag, --project, --today, --week, --overdue)
- ✅ **flip task new** - Interactive Task Creation
- ✅ **flip task stats** - Statistiken mit Completion Rate
- ✅ **flip task search** - Suche nach Beschreibung
- ⏸️ **flip task done/start/update** - Platzhalter implementiert

### Menu Integration
- ✅ Browse & Search → Browse Tasks (mit 6 Filter-Optionen)
- ✅ New → Task (ruft runCreateTask auf)

### Testing
- ✅ Live-Test erfolgreich: 50 Tasks in realen Brains gescannt
- ✅ Stats zeigen korrekte Zahlen (44 open, 6 done, 12% completion)
- ✅ Keine Compile-Errors

---

## 🚧 Offene Punkte - Priorität HOCH

### 1. Task Update/Done Implementation (Critical)

**Warum wichtig:** Kernfunktionalität für Task Management

**Was fehlt:**
- File-Editing für Status-Updates
- Task by ID finden und updaten
- Checkbox-Status ändern ([ ] → [x] oder [/])
- Metadata aktualisieren (completed::, started::)

**Dateien zu erstellen/ändern:**
- `internal/tasks/updater.go` (neu)
  - `UpdateTaskInFile(filePath, lineNum, newTask)`
  - `ToggleTaskStatus(task *Task)`
  - `SetTaskStatus(task *Task, status Status)`

**Commands zu implementieren:**
```go
// internal/commands/task_helpers.go

func NewTaskDoneCommand() *cobra.Command {
    // Scan tasks, find by ID/interactive selection, mark as done
    // Update file with completed:: timestamp
}

func NewTaskStartCommand() *cobra.Command {
    // Mark task as in-progress
    // Add started:: timestamp
}

func NewTaskUpdateCommand() *cobra.Command {
    // Interactive update of task properties
    // Change description, due date, priority, tags
}
```

**Beispiel-Implementation:**
```go
// UpdateTaskInFile ersetzt eine Task-Zeile in einer Datei
func UpdateTaskInFile(filePath string, lineNum int, newTask *Task) error {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return err
    }
    
    lines := strings.Split(string(content), "\n")
    if lineNum <= 0 || lineNum > len(lines) {
        return fmt.Errorf("invalid line number")
    }
    
    // Replace task line
    lines[lineNum-1] = FormatTask(newTask)
    
    // Write back
    return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
}
```

**Herausforderungen:**
- Multi-line Tasks (Task + Metadata)
- Concurrent file access
- Preserving file structure

---

### 2. Task Templates System

**Warum wichtig:** Vereinfacht Workflow, etabliert Best Practices

**Was fehlt:**
- Template-Dateien für tasks/
- `flip task init` Command
- Template-Rendering

**Templates zu erstellen:**

**`~/.flip/templates/task-backlog.md`**
```markdown
# Task Backlog

Created: {{date}}

## High Priority
<!-- High priority tasks -->

## Medium Priority
<!-- Medium priority tasks -->

## Low Priority
<!-- Low priority tasks -->

## Someday/Maybe
<!-- Ideas for later -->
```

**`~/.flip/templates/task-today.md`**
```markdown
# Tasks - {{date}}

## 🎯 Top 3 Priorities
- [ ] 
- [ ] 
- [ ] 

## 📋 Other Tasks
<!-- Tasks due today -->

## ✅ Completed Today
<!-- Moved here when marked done -->
```

**`~/.flip/templates/task-project.md`**
```markdown
# {{project-name}} - Tasks

Project: [[{{project-name}}]]
Created: {{date}}

## 🎯 Current Sprint
<!-- Active tasks -->

## 📋 Backlog
<!-- Future tasks -->

## ✅ Completed
<!-- Done tasks -->
```

**Implementation:**
```go
// internal/commands/task_init.go

func NewTaskInitCommand() *cobra.Command {
    // Create tasks/ folder in active brain
    // Copy templates from ~/.flip/templates/
    // Render with current date, project name
}

// internal/tasks/templates.go

func RenderTemplate(templatePath, outputPath string, vars map[string]string) error {
    // Read template
    // Replace {{var}} with values
    // Write to output
}
```

---

### 3. Interactive Task Selection

**Warum wichtig:** Bessere UX für task done/update

**Was fehlt:**
- Task-Auswahl aus Liste
- Fuzzy Search für Tasks
- Quick Actions Menu

**Implementation:**
```go
// internal/commands/task_list.go - Erweitern

func selectAndActOnTask(taskList []*tasks.Task) error {
    // 1. Zeige Task-Liste mit promptui
    // 2. User wählt Task aus
    // 3. Zeige Actions: Done, Start, Update, Open File, Cancel
    // 4. Führe gewählte Action aus
    
    // Task selection
    items := make([]string, len(taskList))
    for i, task := range taskList {
        items[i] = fmt.Sprintf("%s | %s", task.Description, task.Context.FileName)
    }
    
    selectPrompt := promptui.Select{
        Label: "Select Task",
        Items: items,
        Size:  10,
    }
    
    idx, _, err := selectPrompt.Run()
    if err != nil {
        return nil
    }
    
    selectedTask := taskList[idx]
    
    // Action menu
    actionPrompt := promptui.Select{
        Label: "What do you want to do?",
        Items: []string{
            "✅ Mark as Done",
            "🔄 Mark as In Progress",
            "📝 Update Task",
            "📂 Open File",
            "❌ Cancel",
        },
    }
    
    actionIdx, _, _ := actionPrompt.Run()
    
    switch actionIdx {
    case 0: // Done
        return markTaskAsDone(selectedTask)
    case 1: // Start
        return markTaskAsStarted(selectedTask)
    case 2: // Update
        return updateTaskInteractive(selectedTask)
    case 3: // Open File
        return openInEditor(selectedTask.Context.FilePath)
    }
    
    return nil
}
```

---

## 🔮 Phase 2 Features (Nice-to-Have)

### 1. Performance Optimizations
- **SQLite Cache** für große Brains (1000+ Tasks)
- **Incremental Scanning** (nur geänderte Dateien)
- **File Hash Tracking** (git status integration)
- **Background Indexing** (optional daemon)

**Implementierung:**
```go
// internal/tasks/cache.go

type TaskCache struct {
    DB           *sql.DB
    LastScan     time.Time
    FileHashes   map[string]string
}

func (c *TaskCache) NeedsRescan(filePath string) bool {
    currentHash := hashFile(filePath)
    cachedHash := c.FileHashes[filePath]
    return currentHash != cachedHash
}

func (c *TaskCache) UpdateCache(tasks []Task) error {
    // Store in SQLite
    // Update file hashes
}
```

### 2. Advanced Task Features
- **Recurring Tasks**: `repeat: weekly`, `until: 2025-12-31`
- **Dependencies**: `depends-on: [[other-task-id]]`, `blocks: [[task-id]]`
- **Subtasks**: Nested task hierarchies
- **Time Tracking**: Start/Stop timer, Pomodoro integration
- **Task Links**: Bidirectional links zwischen Tasks und Notes

**Format:**
```markdown
- [ ] Weekly Report 🔁 every Monday
  repeat:: weekly
  on:: monday
  until:: 2025-12-31

- [ ] Backend API 
  depends-on:: [[task-123]], [[task-456]]
  blocks:: [[task-789]]
  - [ ] Subtask 1
  - [ ] Subtask 2
```

### 3. Reports & Analytics
- **Burndown Chart**: Task completion über Zeit
- **Velocity Tracking**: Tasks pro Woche
- **Time Reports**: Zeit pro Projekt/Tag
- **Heatmap**: Task-Aktivität über Zeit

**Commands:**
```bash
flip task report burndown --project "Alpha" --weeks 4
flip task report velocity --since "2025-01-01"
flip task report time --by project
```

### 4. Export & Import
- **Calendar Export**: `.ics` für Tasks mit Due Dates
- **GitHub Issues Sync**: Bi-directional sync
- **Todoist Import**: CSV Import
- **JSON API**: REST API für mobile apps

**Commands:**
```bash
flip task export --format ics --output tasks.ics
flip task import todoist --file export.csv
flip task sync github --repo user/repo
```

### 5. AI Features
- **Smart Scheduling**: ML-basierte Due Date Vorschläge
- **Priority Assistant**: Auto-Priorität basierend auf Context
- **Task Similarity**: Finde ähnliche Tasks
- **Time Estimation**: Lerne aus completed tasks

---

## 📝 Architektur-Notes für zukünftige Entwicklung

### File Structure Conventions
```
brain/
├── tasks/
│   ├── backlog.md          # Unsorted/future tasks
│   ├── today.md            # Today's focus
│   ├── this-week.md        # Week planning
│   ├── recurring.md        # Recurring task templates
│   └── projects/
│       ├── project-alpha.md
│       └── project-beta.md
├── projects/
│   └── alpha/
│       └── notes.md        # Can contain embedded tasks
└── journal/
    └── 2025-10-22.md      # Daily journal with tasks
```

### Task ID Strategy
- **Current**: Hash-based (SHA256 of file:line:description)
- **Problem**: Changes wenn Task moved
- **Alternative**: UUID (needs storage in frontmatter)
- **Recommendation**: Hybrid approach

```markdown
- [ ] Task description
  id:: uuid-123-456-789
  created:: 2025-10-22
```

### Metadata Priority
1. **Inline** (fast scan): Due date, Priority, Tags
2. **Indented** (details): Created, Effort, Project, Notes
3. **Frontmatter** (future): YAML für komplexe Struktur

### Query Performance
- **< 100 tasks**: Live scan OK
- **100-500 tasks**: Index helpful
- **> 500 tasks**: SQLite cache recommended
- **> 1000 tasks**: Background indexing needed

---

## 🎯 Empfohlene Reihenfolge

1. **Sofort** (1-2 Stunden):
   - Task Update/Done Implementation
   - Interactive Task Selection
   
2. **Nächste Session** (2-3 Stunden):
   - Template System
   - Task Init Command
   
3. **Später** (optional):
   - Performance Optimizations (bei Bedarf)
   - Advanced Features (Phase 2)

---

## 🧪 Testing Checklist

Vor jedem Release testen:
- [ ] `flip task list` zeigt Tasks korrekt
- [ ] `flip task new` erstellt Task in richtiger Datei
- [ ] `flip task done <id>` markiert Task als completed
- [ ] `flip task stats` zeigt korrekte Zahlen
- [ ] Menu → Browse Tasks funktioniert
- [ ] Menu → New → Task funktioniert
- [ ] Multi-Brain Scanning korrekt
- [ ] Emoji-Indikatoren (⏫ 📅) korrekt geparst
- [ ] Tags (#tag) korrekt extrahiert
- [ ] File-Context (Line, Section) korrekt getrackt

---

## 📚 Referenzen

**Kompatibilität:**
- [Obsidian Tasks Plugin](https://github.com/obsidian-tasks-group/obsidian-tasks) - Format-Inspiration
- [Logseq](https://docs.logseq.com/) - TODO/DOING/DONE Syntax
- [GitHub Markdown](https://docs.github.com/en/get-started/writing-on-github/getting-started-with-writing-and-formatting-on-github/basic-writing-and-formatting-syntax#task-lists) - Checkbox Standard

**Code-Beispiele:**
- `internal/tasks/` - Core Implementation
- `internal/commands/task_*.go` - CLI Commands
- `internal/commands/menu.go` - Menu Integration (Zeile 1434-1563)

---

**Status**: 85% MVP implementiert, 15% für Production-Ready
**Nächster Commit**: Task Update/Done Implementation
**Geschätzter Aufwand**: 2-3 Stunden für komplettes MVP
