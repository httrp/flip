# Brain Types Reference

**Last Updated:** 2025-12-20  
**Purpose:** Authoritative reference for all supported knowledge management systems (brain types/dialects)

This document serves as the single source of truth for brain type conventions. All code implementing brain-specific behavior should reference this document.

---

## Overview

Flip supports multiple "brain types" (also called "dialects") - different markdown-based knowledge management systems. Each has unique:

- **Folder structures** (journal/ vs journals/ vs Daily Notes/)
- **File naming conventions** (YYYY-MM-DD.md vs YYYY_MM_DD.md vs journal.YYYY-MM-DD.md)
- **Metadata formats** (YAML frontmatter vs property bullets vs Dendron ID system)
- **Link styles** (wikilinks, markdown links, block references)
- **Asset organization** (centralized assets/ vs attachments/ vs near-note)

**Core Principle:** Flip must ALWAYS detect the brain type and adapt all operations accordingly.

---

## Supported Brain Types

### 1. **Flip** (Native)

**Status:** ✅ Fully supported  
**Marker File:** `.flip-brain.yaml`, `.flip.yaml`

**Philosophy:** Opinionated but flexible structure optimized for flip workflows.

#### Directory Structure
```
brain-root/
├── .flip-brain.yaml        # Brain marker with metadata
├── .flip.yaml              # Optional config
├── journal/                # Daily journal entries
│   └── YYYY-MM-DD.md
├── meetings/               # Meeting notes
│   └── YYYY-MM-DD-meeting-title.md
├── notes/                  # General notes
│   └── YYYY-MM-DD-note-title.md
├── tasks/                  # Task lists
│   └── todo.*.md
├── templates/              # Note templates
│   ├── note-template.md
│   ├── journal-template.md
│   └── meeting-template.md
├── definitions/            # Organizational metadata
│   ├── organizations.yaml
│   ├── projects.yaml
│   ├── contexts.yaml
│   └── people.yaml
└── assets/                 # Media and attachments
    ├── images/
    └── documents/
```

#### Naming Conventions
- **Journal:** `YYYY-MM-DD.md` in `journal/`
- **Notes:** `YYYY-MM-DD-slug-title.md` in `notes/`
- **Meetings:** `YYYY-MM-DD-meeting-title.md` in `meetings/`

#### Metadata Format
```yaml
---
title: "Note Title"
created: YYYY-MM-DD
updated: YYYY-MM-DD
type: note|journal|meeting
author: Name
tags: [tag1, tag2]
---
```

#### Link Style
- **Preferred:** Wikilinks `[[note-name]]`
- **Supported:** Markdown links `[text](path.md)`

#### Asset Strategy
- **Location:** Centralized `assets/` folder
- **Subfolders:** `assets/images/`, `assets/documents/`
- **References:** `![](../assets/images/file.png)` or `[[assets/images/file.png]]`

---

### 2. **Obsidian**

**Status:** ✅ Fully supported  
**Marker Directory:** `.obsidian/`

**Philosophy:** User-defined structure, maximum flexibility, vault-wide settings.

#### Directory Structure
```
vault-root/
├── .obsidian/              # Obsidian config (marker)
│   ├── app.json
│   ├── workspace.json
│   └── config
├── Daily Notes/            # Daily notes (user-configurable path)
│   └── YYYY-MM-DD.md
├── Projects/               # User-organized folders
├── Templates/              # Templates
├── attachments/            # Default asset location (user-configurable)
└── [user-defined folders]  # Complete freedom
```

#### Naming Conventions
- **Daily Notes:** `YYYY-MM-DD.md` (configurable format)
- **Notes:** `Note Title.md` (no enforced pattern, user choice)
- **Flexible:** Users can use ANY structure

#### Metadata Format
```yaml
---
title: Note Title
date: YYYY-MM-DD
tags: [tag1, tag2]
aliases: [alternative-name]
cssclasses: [custom-class]
---
```

**Features:**
- Very permissive YAML
- Inline properties: `key:: value`
- Dataview plugin support

#### Link Style
- **Preferred:** Wikilinks `[[Note Name]]` or `[[Note Name|Alias]]`
- **Supported:** Markdown links `[text](path.md)`
- **Embeds:** `![[note]]`, `![[image.png]]`

#### Asset Strategy
- **Default:** `attachments/` folder (configurable in settings)
- **Alternative:** Next to note (user preference)
- **Flexible:** Users can configure any location

**Critical Notes:**
- NO enforced structure - respect user's organization
- Daily notes path is configurable per vault
- Asset location is user-defined
- Template location is configurable

---

### 3. **Logseq**

**Status:** ✅ Fully supported  
**Marker Directory:** `logseq/` or `.logseq/`  
**Config:** `logseq/config.edn`

**Philosophy:** Outliner-first, block-based, strict journal/pages separation.

#### Directory Structure
```
graph-root/
├── logseq/                 # Config directory (marker)
│   ├── config.edn
│   ├── metadata.edn
│   └── custom.css
├── journals/               # Daily journal entries
│   └── YYYY_MM_DD.md       # Note: UNDERSCORES!
├── pages/                  # Content pages
│   └── Note Name.md        # Spaces allowed, Title Case
├── assets/                 # Media files
│   └── image_YYYY_MM_DD_UUID.png
└── templates/              # Templates (optional)
```

#### Naming Conventions
- **Journals:** `YYYY_MM_DD.md` in `journals/` ⚠️ **UNDERSCORES, not dashes!**
- **Pages:** `Title Case With Spaces.md` in `pages/`
- **Case-sensitive:** Page names match exactly

#### Metadata Format

**Traditional (Property Bullets):**
```markdown
title:: Page Title
tags:: tag1, tag2
created:: [[YYYY-MM-DD]]
author:: Name

- Content starts here as bullets
- # Heading level 1
  - ## Heading level 2
```

**Modern (YAML Frontmatter - DB graphs):**
```yaml
---
title: Page Title
tags: [tag1, tag2]
created: YYYY-MM-DD
---
```

**Both formats work!** Logseq has evolved to support both.

#### Link Style
- **Preferred:** Wikilinks `[[Page Name]]`
- **Block References:** `((block-uuid))` - unique to Logseq
- **Page References:** `#[[Page Name]]` for tags
- **Embeds:** `{{embed [[Page Name]]}}`

#### Asset Strategy
- **Location:** `assets/` folder
- **Auto-naming:** Logseq adds timestamps to asset filenames
- **References:** `![](../assets/file.png)` or `[[../assets/file.png]]`

**Critical Notes:**
- UNDERSCORE dates in journals (migration pitfall!)
- Outliner format - everything is bullets
- Block-level UUIDs for references
- Case-sensitive page names

---

### 4. **Dendron**

**Status:** ✅ Supported  
**Marker File:** `dendron.yml`  
**Workspace:** VS Code extension-based

**Philosophy:** Hierarchical dot-notation, schema-driven, multi-vault.

#### Directory Structure
```
workspace-root/
├── dendron.yml             # Workspace config (marker)
├── dendron.code-workspace  # VS Code workspace
├── .dendron/               # Dendron metadata
│   ├── cache.json
│   └── port
├── vault/                  # Default vault (can have multiple)
│   ├── root.md             # Root note
│   ├── daily.YYYY.MM.DD.md # Daily journal (dots!)
│   ├── project.alpha.md    # Hierarchy via dots
│   ├── project.alpha.tasks.md
│   └── project.beta.md
└── assets/                 # Assets
    └── images/
```

#### Naming Conventions
- **Hierarchy:** `parent.child.grandchild.md` (dot-notation)
- **Daily:** `daily.YYYY.MM.DD.md` ⚠️ **DOTS, not dashes!**
- **Meetings:** `meet.YYYY.MM.DD.md` or `meet.YYYY.MM.DD.topic.md`
- **Flat structure:** All notes in vault root, hierarchy via naming

#### Metadata Format
```yaml
---
id: uuid-or-timestamp      # Required unique ID
title: Note Title
desc: 'Description'
updated: 1640000000        # Unix timestamp (seconds)
created: 1640000000        # Unix timestamp (seconds)
tags: [tag1, tag2]
---
```

**Required Fields:**
- `id`: Unique identifier (UUID recommended, not nanoseconds)
- `title`: Human-readable title
- `desc`: Description (can be empty string)
- `updated`: Unix timestamp in seconds
- `created`: Unix timestamp in seconds

#### Link Style
- **Preferred:** Wikilinks `[[note-name]]`
- **Cross-vault:** `dendron://vault-name/note-name`
- **Hierarchical:** References respect dot-notation

#### Asset Strategy
- **Location:** `assets/` folder
- **References:** `![](assets/images/file.png)`

**Critical Notes:**
- DOT-notation for hierarchy
- DOTS in daily format (YYYY.MM.DD not YYYY-MM-DD)
- Unix timestamps (seconds, not milliseconds/nanoseconds)
- UUID-based IDs strongly recommended
- Schema system for templates (advanced)

---

### 5. **Foam**

**Status:** ⚠️ Partial support (needs integration)  
**Marker Directory:** `.foam/` or `.vscode/foam.code-snippets`

**Philosophy:** Lightweight, VS Code-based, GitHub-friendly, no dependencies.

#### Directory Structure
```
foam-root/
├── .foam/                  # Optional marker directory
├── .vscode/                # VS Code config
│   ├── foam.code-snippets  # Foam snippets (marker)
│   ├── settings.json
│   └── extensions.json
├── docs/                   # Documentation (common pattern)
├── journal/                # Daily notes (user-configured)
│   └── YYYY-MM-DD.md
├── notes/                  # General notes
├── templates/              # Note templates
└── attachments/            # Assets (user-configured)
```

#### Naming Conventions
- **Flexible:** No enforced structure, user-defined
- **Daily Notes:** Typically `YYYY-MM-DD.md` (configurable)
- **Notes:** User choice, often `note-title.md` or `Note Title.md`

#### Metadata Format
```yaml
---
type: note
tags: [tag1, tag2]
date: YYYY-MM-DD
---
```

**Minimal requirements:** Foam is very permissive with frontmatter.

#### Link Style
- **Preferred:** Wikilinks `[[Note Title]]`
- **GitHub-compatible:** Also supports `[Note Title](note-title.md)`
- **Tag style:** `#tag`

#### Asset Strategy
- **Default:** `attachments/` (user-configurable)
- **Flexible:** Can be any folder
- **GitHub-friendly:** Relative paths

**Critical Notes:**
- Extremely minimal - just wikilinks + markdown
- No enforced structure beyond VS Code workspace
- GitHub Pages integration common
- Similar to Obsidian in flexibility but simpler

**Detection Challenges:**
- May look like generic markdown folder
- Requires `.vscode/foam.code-snippets` or `.foam/` marker
- Fallback to "unknown markdown" if no clear markers

---

## Brain Type Detection Algorithm

### Detection Priority (in order)

1. **Flip** - Look for `.flip-brain.yaml` or `.flip.yaml`
2. **Logseq** - Look for `logseq/config.edn` or `.logseq/` + `journals/` + `pages/`
3. **Obsidian** - Look for `.obsidian/` directory
4. **Dendron** - Look for `dendron.yml`
5. **Foam** - Look for `.foam/` or `.vscode/foam.code-snippets`
6. **Unknown** - Contains `.md` files but no clear markers

### Implementation

```go
// DetectBrainType analyzes directory structure
func DetectBrainType(path string) (BrainType, error) {
    // 1. Check Flip
    if exists(path, ".flip-brain.yaml") || exists(path, ".flip.yaml") {
        return BrainTypeFlip, nil
    }
    
    // 2. Check Logseq (strict: needs config OR journals+pages)
    if exists(path, "logseq/config.edn") {
        return BrainTypeLogseq, nil
    }
    if existsDir(path, "journals") && existsDir(path, "pages") {
        return BrainTypeLogseq, nil // Logseq without config
    }
    
    // 3. Check Obsidian
    if existsDir(path, ".obsidian") {
        return BrainTypeObsidian, nil
    }
    
    // 4. Check Dendron
    if exists(path, "dendron.yml") {
        return BrainTypeDendron, nil
    }
    
    // 5. Check Foam
    if existsDir(path, ".foam") || exists(path, ".vscode/foam.code-snippets") {
        return BrainTypeFoam, nil
    }
    
    // 6. Unknown (but potentially compatible if it has markdown)
    if hasMarkdownFiles(path) {
        return BrainTypeUnknown, nil
    }
    
    return BrainTypeEmpty, nil
}
```

---

## Brain-Aware Operations

### Critical Operations That MUST Be Brain-Aware

#### 1. Directory Resolution
```go
// getNotesDirectory returns the correct notes folder
func getNotesDirectory(brainPath string, brainType BrainType) string {
    switch brainType {
    case BrainTypeFlip:
        return filepath.Join(brainPath, "notes")
    case BrainTypeLogseq:
        return filepath.Join(brainPath, "pages")
    case BrainTypeObsidian:
        return brainPath // User-defined, no enforced structure
    case BrainTypeDendron:
        return filepath.Join(brainPath, "vault") // Or root
    case BrainTypeFoam:
        return filepath.Join(brainPath, "notes") // Common convention
    default:
        return brainPath
    }
}

// getJournalDirectory returns the correct journal/daily notes folder
func getJournalDirectory(brainPath string, brainType BrainType) string {
    switch brainType {
    case BrainTypeFlip:
        return filepath.Join(brainPath, "journal")
    case BrainTypeLogseq:
        return filepath.Join(brainPath, "journals") // Plural!
    case BrainTypeObsidian:
        return filepath.Join(brainPath, "Daily Notes") // Configurable
    case BrainTypeDendron:
        return brainPath // daily.YYYY.MM.DD.md in root
    case BrainTypeFoam:
        return filepath.Join(brainPath, "journal") // Convention
    default:
        return filepath.Join(brainPath, "journal")
    }
}
```

#### 2. File Naming
```go
// generateJournalFilename creates date-based filenames
func generateJournalFilename(date time.Time, brainType BrainType) string {
    switch brainType {
    case BrainTypeFlip:
        return date.Format("2006-01-02") + ".md"
    case BrainTypeLogseq:
        return date.Format("2006_01_02") + ".md" // UNDERSCORES!
    case BrainTypeObsidian:
        return date.Format("2006-01-02") + ".md"
    case BrainTypeDendron:
        return "daily." + date.Format("2006.01.02") + ".md" // DOTS!
    case BrainTypeFoam:
        return date.Format("2006-01-02") + ".md"
    default:
        return date.Format("2006-01-02") + ".md"
    }
}
```

#### 3. Metadata Generation
```go
// generateFrontmatter creates appropriate metadata
func generateFrontmatter(title string, brainType BrainType, fields map[string]string) string {
    switch brainType {
    case BrainTypeLogseq:
        // Property bullets (traditional)
        return fmt.Sprintf("title:: %s\ndate:: %s\n", title, fields["date"])
    case BrainTypeDendron:
        // Dendron requires id, updated, created
        return fmt.Sprintf(`---
id: %s
title: %s
desc: ''
updated: %s
created: %s
---`, fields["id"], title, fields["updated"], fields["created"])
    default:
        // YAML frontmatter
        return fmt.Sprintf(`---
title: %s
date: %s
---`, title, fields["date"])
    }
}
```

---

## Migration Considerations

### Critical Date Format Differences

**Problem:** Date formats differ significantly!

- **Flip/Obsidian/Foam:** `YYYY-MM-DD` (dashes)
- **Logseq:** `YYYY_MM_DD` (underscores) ⚠️
- **Dendron:** `YYYY.MM.DD` (dots) ⚠️

**Solution:** When migrating journals:
```go
// Convert journal filename format
func convertJournalFilename(filename string, fromType, toType BrainType) string {
    // Extract date
    date := extractDateFromFilename(filename, fromType)
    
    // Regenerate with target format
    return generateJournalFilename(date, toType)
}
```

### Asset Path Conventions

**Problem:** Asset references vary!

- **Flip/Logseq/Dendron:** `assets/` folder
- **Obsidian:** `attachments/` (configurable)
- **Foam:** `attachments/` (common)

**Solution:** Asset migrator must:
1. Detect source asset locations
2. Copy to target conventions
3. Rewrite all asset references in notes

---

## Template System

### Per-Brain-Type Templates

Templates MUST exist for each brain type:

```
~/.config/flip/templates/
├── flip/
│   ├── note.md
│   ├── journal.md
│   ├── meeting.md
│   └── task.md
├── obsidian/
│   ├── note.md
│   ├── journal.md
│   └── meeting.md
├── logseq/
│   ├── note.md
│   ├── journal.md
│   └── meeting.md
├── dendron/
│   ├── note.md
│   ├── journal.md
│   └── meeting.md
└── foam/
    ├── note.md
    ├── journal.md
    └── meeting.md
```

**Placeholders:** (available in all templates)
- `{{title}}` - Note title
- `{{date}}` - Current date (YYYY-MM-DD)
- `{{time}}` - Current time (HH:MM)
- `{{id}}` - UUID (Dendron requires this)
- `{{updated}}` - Unix timestamp (Dendron requires this)
- `{{created}}` - Unix timestamp (Dendron requires this)
- `{{author}}` - Author name from brain config
- `{{tags}}` - Tags placeholder

---

## Testing Matrix

For each brain type, test:

1. **Detection** - Correctly identifies brain type
2. **Note Creation** - Creates note with correct format
3. **Journal Creation** - Uses correct date format and folder
4. **Meeting Creation** - Follows conventions
5. **Asset Handling** - Stores in correct location
6. **Link Resolution** - Finds referenced notes
7. **Migration** - Converts formats correctly
8. **Health Check** - Validates brain structure

---

## Adding New Brain Types

Checklist for adding support:

- [ ] Add to `BrainType` enum (`internal/brain/detector.go` and `internal/health/detector.go`)
- [ ] Add detection logic to `DetectBrainType()` functions
- [ ] Add directory structure to `internal/migration/structure.go` `GetBrainStructure()`
- [ ] Add templates to `~/.config/flip/templates/{brain-type}/`
- [ ] Update `getNotesDirectory()`, `getJournalDirectory()` functions
- [ ] Update `generateJournalFilename()`, `generateNoteFilename()` functions
- [ ] Update `generateFrontmatter()` logic
- [ ] Add to brain creator `createDirectoriesCompatible()`
- [ ] Add tests for detection, creation, migration
- [ ] Document in this file

---

## References

- **Obsidian:** https://help.obsidian.md/
- **Logseq:** https://docs.logseq.com/
- **Dendron:** https://wiki.dendron.so/
- **Foam:** https://foambubble.github.io/foam/

---

**Maintained by:** flip development team  
**Last Review:** 2025-12-20
