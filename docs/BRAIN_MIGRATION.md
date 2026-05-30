# Brain Migration & Health Check Analysis

## Datum: 2025-11-09
## Feature: Brain-to-Brain Migration mit Format-Konvertierung

---

## 1. Problem Statement

**Ziel**: Notes sicher und intelligent zwischen verschiedenen Brain-Typen migrieren
- Logseq → Obsidian
- Obsidian → Flip
- Flip → Logseq
- etc.

**Herausforderungen**:
1. Unterschiedliche Dateistrukturen
2. Unterschiedliche Link-Formate
3. Unterschiedliche Asset-Verwaltung
4. Unterschiedliche Metadaten-Formate
5. Task-Format-Unterschiede

---

## 2. Brain-Typen Analyse

### 2.1 Logseq

**Struktur**:
```
graph-name/
├── journals/
│   ├── 2025_01_15.md          # Underscore-Format!
│   └── ...
├── pages/
│   ├── note-name.md           # Filename = Title
│   └── ...
├── assets/
│   ├── image_1234567890.png   # Timestamped
│   └── ...
└── logseq/
    ├── config.edn
    └── custom.css
```

**Eigenschaften**:
- **Links**: `[[Page Name]]` - Bi-directional, case-insensitive
- **Daily Notes**: `journals/YYYY_MM_DD.md` (Underscore!)
- **Assets**: `![](../assets/image_123.png)` - relative paths
- **Tasks**: 
  ```markdown
  - TODO Task
  - DOING In progress
  - DONE Completed
  - LATER Deferred
  - CANCELLED Cancelled
  ```
- **Metadaten**: Properties mit `::`
  ```markdown
  title:: My Note
  tags:: [[tag1]], [[tag2]]
  ```
- **Blocks**: Jeder Block hat UUID für Referenzen
- **Namespaces**: Hierarchie via `/` in Filename: `project/subtopic.md`

**Besonderheiten**:
- Outliner-basiert (alles sind Blocks)
- Graph ist bidirektional
- Starke Block-Referenzen
- EDN-Config (Clojure-basiert)

### 2.2 Obsidian

**Struktur**:
```
vault-name/
├── Daily Notes/               # Configurable folder
│   ├── 2025-01-15.md         # Dash-Format
│   └── ...
├── Projects/                  # User-defined folders
│   ├── Project A.md
│   └── Project A/            # Optional subfolder for assets
│       └── image.png
├── Templates/
└── .obsidian/                # Hidden config
    ├── workspace.json
    ├── plugins/
    └── themes/
```

**Eigenschaften**:
- **Links**: `[[Note Name]]` oder `[[Note Name|Alias]]`
- **Wikilinks**: `[[folder/note]]` für Subfolders
- **Embed**: `![[note]]` oder `![[image.png]]`
- **Daily Notes**: Format konfigurierbar (meist `YYYY-MM-DD`)
- **Assets**: 
  - Inline: `![alt](path/to/image.png)`
  - Wikilink: `![[image.png]]`
  - Attachments folder konfigurierbar
- **Tasks**: Standard Markdown
  ```markdown
  - [ ] Open task
  - [x] Completed task
  ```
- **Frontmatter**: YAML
  ```yaml
  ---
  title: My Note
  tags: [tag1, tag2]
  aliases: [alias1]
  ---
  ```
- **Folders**: Freie Struktur, Folders = Organisation

**Besonderheiten**:
- File-basiert (nicht Block-basiert)
- Graph ist bidirektional (mit Plugin)
- Sehr flexible Folder-Struktur
- Starke Plugin-Ecosystem

### 2.3 Dendron

**Struktur**:
```
workspace/
├── vault1/
│   ├── root.md
│   ├── daily.2025.01.15.md      # Hierarchical!
│   ├── project.web.ideas.md     # Dot notation!
│   └── assets/
│       └── images/
└── dendron.yml
```

**Eigenschaften**:
- **Hierarchie**: Dot-notation `parent.child.grandchild`
- **Links**: `[[note]]` oder `[[dendron://vault/note]]`
- **Daily Notes**: `daily.YYYY.MM.DD.md`
- **Tasks**: Standard Markdown checklist
- **Frontmatter**: YAML ähnlich wie Obsidian
- **Assets**: Meist in `/assets` oder `/assets/images`

**Besonderheiten**:
- Hierarchisches System (wie Dateipfade mit Punkten)
- Multi-vault Support
- Lookup-basiert (Hierarchie im Filename)

### 2.4 Flip (Current)

**Struktur**:
```
brain/
├── journal/
│   ├── 2025-01-15.md          # Dash-Format
│   └── ...
├── notes/
│   ├── note-name.md
│   └── ...
├── meetings/
│   ├── 2025-01-15-meeting-title.md
│   └── ...
├── tasks/
│   ├── tasks.md               # Unified tasks
│   ├── inbox.md
│   └── ...
├── templates/
│   ├── note-template.md
│   └── ...
├── definitions/
│   ├── task-definitions.yaml  # Simple format
│   ├── organizations.yaml     # Brain format
│   ├── projects.yaml
│   ├── contexts.yaml
│   └── people.yaml
├── assets/
│   └── images/
└── .flip.yaml                 # Brain config
```

**Eigenschaften**:
- **Links**: Standard Markdown `[text](path.md)`
- **Wikilinks**: Optional `[[note]]` (sollten wir unterstützen?)
- **Daily Notes**: `journal/YYYY-MM-DD.md`
- **Assets**: `/assets/images/` - relative paths
- **Tasks**: Eigenes Format mit Metadaten
  ```markdown
  - [ ] Task title
    status:: open
    priority:: high
    organization:: ORG
    project:: PROJ
  ```
- **Frontmatter**: YAML
- **Definitions**: YAML-basiert (org/project/context/people)

**Besonderheiten**:
- Strukturierte Ordner (journal, notes, meetings, tasks)
- Task-System mit Definitions
- Hybrides Definitions-System (Brain + Simple)

---

## 3. Kritische Analyse deiner Anforderungen

### ✅ Sehr gute Ideen:

1. **Definition Migration**: Absolut wichtig!
   - Logseq properties → Flip definitions
   - Obsidian tags → Flip definitions
   - Mapping-Konzept für Organisationen/Projekte

2. **Format-Anpassung**: Richtig erkannt!
   - Journal-Dateinamen (Underscore vs. Dash)
   - Link-Format (Wikilinks vs. relative paths)
   - Task-Format (TODO vs. checkbox vs. flip)

3. **Asset-Migration**: Kritischer Punkt!
   - Pfad-Anpassung
   - Kopieren vs. Referenzen
   - Deduplizierung

4. **Health Checks**: Sehr smart!
   - Broken links detection
   - Missing images detection
   - Orphaned files

### ⚠️ Herausforderungen & Kritik:

1. **"Geflogenheiten des Target Brain"**:
   - Problem: Zu generisch! Was genau heißt das?
   - Besser: **Konkrete Konvertierungsregeln definieren**
   - Beispiel: Logseq Blocks → Flip Notes (1:1 oder gruppieren?)

2. **Optionen-Overload**:
   - Zu viele Optionen verwirren
   - Besser: **Smart Defaults + Advanced Mode**
   - Beispiel: "Migrate All" vs. "Selective Migration"

3. **Bidirektionalität**:
   - Nicht erwähnt: Was wenn ich zurück migrieren will?
   - Sollten wir Migrations-Historie tracken?
   - Conflict resolution?

4. **Edge Cases**:
   - Was mit Block-References (Logseq-spezifisch)?
   - Was mit Obsidian Canvas-Files?
   - Was mit Plugin-spezifischen Daten?

5. **Performance**:
   - Großer Graph (1000+ Notes)?
   - Asset-Größe (GB an Bildern)?
   - Progress-Tracking notwendig!

---

## 4. Vorgeschlagene Feature-Struktur

### Phase 1: Health Check (Foundation)
**Warum zuerst?**
- Unabhängig von Migration
- Sofortiger Nutzen
- Basis für Migration-Validation

**Features**:
```bash
flip brain check
  ├── Check broken links
  ├── Check missing assets
  ├── Check orphaned files
  ├── Check duplicate files
  ├── Check task format consistency
  └── Check definitions consistency
```

**Output**:
```
🔍 Brain Health Check: my-brain

✅ Links: 245 OK, 3 broken
   • note-a.md:15 → missing-note.md
   • note-b.md:23 → ../assets/old.png (missing)

⚠️  Assets: 12 orphaned files
   • assets/unused-image.png (not referenced)

✓ Tasks: All valid
✓ Definitions: Consistent
```

### Phase 2: Migration Preparation (Dry-Run)
```bash
flip brain migrate --from my-brain --to new-obsidian-brain --dry-run
```

**Was passiert**:
1. Analysiere Source Brain Type (auto-detect)
2. Analysiere Target Brain Type (auto-detect oder specify)
3. Erstelle Migration Plan
4. Zeige Preview der Changes
5. Frage nach Bestätigung

**Migration Plan Output**:
```
📋 Migration Plan: my-brain (flip) → obsidian-vault (obsidian)

Files to migrate:
  ✓ 150 notes
  ✓ 45 journal entries
  ✓ 12 meetings
  ✓ 234 assets

Conversions needed:
  • Journal filenames: 2025-01-15.md → 2025-01-15.md ✓ (same)
  • Links: [text](path.md) → [[Note Name]]
  • Tasks: flip format → markdown checkboxes
  • Definitions: 
    - 5 organizations → #org/NAME tags
    - 12 projects → #project/NAME tags
    - 8 contexts → #context/NAME tags
    - 15 people → #person/NAME tags

Warnings:
  ⚠️  3 notes have block references (Logseq-specific) - will be converted to links
  ⚠️  2 canvas files will not be migrated (Obsidian-specific)

Estimated time: ~30 seconds
Space required: ~45 MB

Continue? [y/N]
```

### Phase 3: Actual Migration
```bash
flip brain migrate --from my-brain --to new-obsidian-brain
```

**Features**:
- Progress bar
- Atomic operation (all or nothing)
- Backup creation before migration
- Rollback capability
- Log file creation

### Phase 4: Post-Migration Validation
```bash
flip brain validate-migration --source my-brain --target new-obsidian-brain
```

**Checks**:
- All files migrated?
- All links working?
- All assets present?
- No data loss?

---

## 5. Konkrete Konvertierungs-Regeln

### 5.1 Links

| Source Format | Target Format | Conversion |
|---------------|---------------|------------|
| Logseq `[[Page Name]]` | Flip `[Page Name](page-name.md)` | Filename normalization |
| Flip `[text](note.md)` | Obsidian `[[note]]` | Extract filename, remove extension |
| Obsidian `[[note\|alias]]` | Flip `[alias](note.md)` | Split on pipe |

### 5.2 Assets

| Source | Target | Action |
|--------|--------|--------|
| Logseq `../assets/img.png` | Flip `assets/images/img.png` | Copy + update path |
| Obsidian `![[img.png]]` | Flip `![](assets/images/img.png)` | Convert wikilink |
| Absolute paths | Relative paths | Convert if asset exists |

### 5.3 Tasks

| Source | Target | Action |
|--------|--------|--------|
| Logseq `TODO` | Flip `- [ ] + status::open` | Add metadata |
| Obsidian `- [ ]` | Flip `- [ ] + status::open` | Add metadata |
| Flip format | Logseq `TODO` | Remove metadata, add marker |

### 5.4 Definitions/Tags

**Konzept**: Tag-basierte Systeme → Flip Definitions

```yaml
# Mapping strategy
tags:
  - "#project/webdev" → project: WEBDEV
  - "#org/acme" → organization: ACME
  - "#person/john" → person: JOHN

# Auto-create definitions during migration
definitions:
  organizations:
    - name: "ACME Corp"
      abbreviation: "ACME"
      source: "migrated from obsidian tag #org/acme"
```

### 5.5 Journal/Daily Notes

| Source Format | Target Format | Conversion |
|---------------|---------------|------------|
| Logseq `journals/2025_01_15.md` | Flip `journal/2025-01-15.md` | Rename + move |
| Obsidian `Daily/2025-01-15.md` | Flip `journal/2025-01-15.md` | Move |
| Dendron `daily.2025.01.15.md` | Flip `journal/2025-01-15.md` | Rename pattern |

---

## 6. Implementierungs-Vorschlag

### Architektur

```
internal/
├── migration/
│   ├── analyzer.go          # Brain type detection
│   ├── converter.go         # Format conversions
│   ├── migrator.go          # Main migration logic
│   ├── validator.go         # Pre/Post validation
│   ├── plan.go              # Migration plan generation
│   └── types/
│       ├── logseq.go        # Logseq-specific
│       ├── obsidian.go      # Obsidian-specific
│       ├── dendron.go       # Dendron-specific
│       └── flip.go          # Flip-specific
├── health/
│   ├── checker.go           # Health check orchestrator
│   ├── links.go             # Link validation
│   ├── assets.go            # Asset validation
│   └── report.go            # Report generation
```

### Commands

```go
// Health Check
flip brain check [brain-name]
flip brain check --report=json
flip brain check --fix-auto    // Auto-fix simple issues

// Migration
flip brain migrate --source <brain> --target <brain> [--dry-run]
flip brain migrate --source my-brain --target obsidian-vault --type obsidian
flip brain migrate --interactive  // Guided wizard

// Validation
flip brain validate <brain-name>
flip brain validate-migration --source <brain> --target <brain>
```

### Migration Wizard (Interactive Mode)

```
🔄 Brain Migration Wizard

Step 1: Source Brain
  → Select source brain: my-brain
  → Detected type: flip
  → Found: 150 notes, 45 journal entries, 234 assets

Step 2: Target Brain
  → Target brain: new-vault
  → Target type: [flip/obsidian/logseq] obsidian
  → Create new? [Y/n] Y

Step 3: Migration Options
  ✓ Migrate notes [Y/n]
  ✓ Migrate journal [Y/n]
  ✓ Migrate assets [Y/n]
  ✓ Migrate tasks [Y/n]
  ✓ Convert definitions to tags [Y/n]
  
Step 4: Conversion Options
  Link format: [relative/wikilink] wikilink
  Task format: [checkbox/logseq-markers] checkbox
  Journal folder: [Daily Notes]
  Assets folder: [attachments]

Step 5: Safety
  ✓ Create backup before migration [Y/n]
  ✓ Run health check first [Y/n]
  
Step 6: Review
  [Show detailed migration plan]
  
Ready to migrate? [y/N]
```

---

## 7. Edge Cases & Lösungen

### 7.1 Logseq Block-References
**Problem**: `((block-uuid))` gibt es nicht in anderen Systemen
**Lösung**: 
- Option 1: Convert to section link `[Block text](#section)`
- Option 2: Convert to full note link `[Note name](note.md)`
- Option 3: Keep as text with warning

### 7.2 Obsidian Canvas
**Problem**: `.canvas` files sind JSON, nicht Markdown
**Lösung**: 
- Skip mit Warning
- Oder: Extract links und migrate referenced notes

### 7.3 Duplicate Filenames
**Problem**: `note.md` in verschiedenen Ordnern
**Lösung**:
- Keep folder structure wenn möglich
- Oder: Rename mit Suffix `note-2.md`
- Fragen beim User

### 7.4 Circular References
**Problem**: Note A → Note B → Note A
**Lösung**: 
- Kein Problem, einfach beibehalten
- Graph bleibt bidirektional

### 7.5 Large Assets
**Problem**: 2GB Video-Datei in assets
**Lösung**:
- Progress indicator
- Option to skip large files
- Compression option?

---

## 8. Testing Strategy

### Unit Tests
- Link conversion functions
- Path normalization
- Format detection

### Integration Tests
- Full migration scenarios
- Rollback functionality
- Health check accuracy

### Test Brains
Erstelle kleine Test-Brains:
```
test-brains/
├── logseq-sample/      # 10 notes, mixed content
├── obsidian-sample/    # 10 notes, mixed content
├── flip-sample/        # 10 notes, mixed content
└── expected-outputs/   # Expected results
```

---

## 9. Phase 1 Implementation Plan

**Start simple, iterate quickly!**

### Milestone 1: Health Check (1-2 Tage)
- [x] Brain type detection
- [ ] Broken link checker
- [ ] Missing asset checker
- [ ] Report generation

### Milestone 2: Basic Migration (3-4 Tage)
- [ ] Flip → Obsidian (einfachster Fall)
- [ ] Link conversion
- [ ] Asset copying
- [ ] Dry-run mode

### Milestone 3: Validation (1-2 Tage)
- [ ] Pre-migration validation
- [ ] Post-migration validation
- [ ] Diff reporting

### Milestone 4: Polish (1-2 Tage)
- [ ] Interactive wizard
- [ ] Progress indicators
- [ ] Rollback functionality
- [ ] Documentation

**Total: ~1-2 Wochen für MVP**

---

## 10. Offene Fragen

1. **Backup Strategy**: Wohin? Ins Brain selbst oder extern?
2. **Rollback**: Wie granular? Vollständig oder partial?
3. **Definitions Mapping**: User-Input erforderlich oder Auto-Detect?
4. **Performance**: Ab welcher Größe ist Streaming notwendig?
5. **Konflikte**: Was wenn Target Brain schon existiert?

---

## 11. Nächste Schritte

1. ✅ **Analyse-Dokument erstellen** (dieses Dokument)
2. [ ] **Feedback einholen** von dir
3. [ ] **Priorisierung** der Features
4. [ ] **Branch erstellen** ✅ `feat/brain-migration`
5. [ ] **Prototyp** Health Check implementieren
6. [ ] **Test Brains** erstellen
7. [ ] **Iterative Development**

---

## 12. Fragen an dich

1. **Priorität**: Health Check oder Migration zuerst?
2. **Scope**: Welche Brain-Types sind am wichtigsten? (Logseq? Obsidian?)
3. **Auto-Detection**: Wie wichtig ist es, Brain-Type automatisch zu erkennen?
4. **Definitions**: Auto-Mapping oder immer User-Input?
5. **Rollback**: Wie wichtig ist vollständiger Rollback?
6. **Interactive vs. Command**: Wizard bevorzugt oder CLI mit Flags?

---

**Status**: 📝 Analyse-Phase
**Branch**: `feat/brain-migration`
**Next**: Feedback & Priorisierung
