# Brain Type Awareness Audit

**Date:** 2025-12-20  
**Purpose:** Systematic audit of ALL commands to verify brain-type awareness

## Audit Status

### ✅ FULLY Brain-Type Aware

These commands detect brain type and adapt behavior:

1. **`flip note`** ✅
   - File: [internal/commands/note.go](../internal/commands/note.go)
   - Detection: Line 87 - `detector.DetectBrainType(activeBrain.Path)`
   - Adaptations:
     - Directory: `getNotesDirectory()` respects brain type
     - Filename: `generateNoteFilename()` uses brain-specific formats
     - Content: `generateNoteContent()` uses brain-specific frontmatter

2. **`flip journal`** ✅
   - File: [internal/commands/journal.go](../internal/commands/journal.go)
   - Detection: Line 72 - `detector.DetectBrainType(activeBrain.Path)`
   - Adaptations:
     - Directory: `journals/` (Logseq) vs `journal/` (Flip) vs `Daily Notes/` (Obsidian)
     - Filename: `YYYY_MM_DD.md` (Logseq) vs `YYYY-MM-DD.md` (others)
     - Content: Property bullets (Logseq) vs YAML frontmatter

3. **`flip meeting`** ✅
   - File: [internal/commands/meeting.go](../internal/commands/meeting.go)
   - Detection: Line 75 - `detector.DetectBrainType(activeBrain.Path)`
   - Adaptations:
     - Directory: `meetings/` (Flip) vs `pages/` (Logseq) vs root (Dendron)
     - Filename: Brain-specific formats
     - Content: Brain-specific frontmatter

4. **`flip exercise list`** ✅
   - File: [internal/commands/exercise_list.go](../internal/commands/exercise_list.go)
   - Detection: Line 43 - `detector.DetectBrainType(brainPath)`
   - Uses brain type for path scanning

5. **`flip exercise new`** ✅
   - File: [internal/commands/exercise_new.go](../internal/commands/exercise_new.go)
   - Detection: Line 33
   - Brain-aware content generation

6. **`flip exercise show`** ✅
   - File: [internal/commands/exercise_show.go](../internal/commands/exercise_show.go)
   - Detection: Line 43

7. **`flip exercise edit`** ✅
   - File: [internal/commands/exercise_edit.go](../internal/commands/exercise_edit.go)
   - Detection: Line 32

8. **`flip exercise track`** ✅
   - File: [internal/commands/exercise_track.go](../internal/commands/exercise_track.go)
   - Detection: Line 34
   - Has `generateMinimalJournalHeader()` that adapts to brain type

9. **`flip exercise-plan *`** ✅
   - All plan commands detect brain type
   - Files: exercise_plan_*.go

10. **`flip brain migrate`** ✅
    - Uses BrainStructure system
    - Full brain-type awareness

11. **`flip template`** ✅
    - Loads brain-specific templates
    - Fully brain-type aware

### ✅ Brain-Agnostic by Design

These commands intentionally use a universal format that works across all brain types:

1. **`flip task new`** ✅ (Brain-Agnostic)
   - File: [internal/commands/task_new.go](../internal/commands/task_new.go)
   - **Intentionally NOT brain-specific!**
   - Uses universal format:
     - Checkbox: `- [x] Task description`
     - Properties: `key:: value` (Logseq-native, readable everywhere)
     - Location: `tasks/` directory (flip's own convention)
   - **Rationale:** One consistent format = one parser = maintainable code

2. **`flip task list`** ✅ (Brain-Agnostic)
   - File: [internal/commands/task_list.go](../internal/commands/task_list.go)
   - Scans `tasks/` directory in all brains
   - Universal parser works with one format

3. **`flip task browse`** ✅ (Brain-Agnostic)
   - File: [internal/commands/task_browse_cmd.go](../internal/commands/task_browse_cmd.go)
   - Same universal format

4. **`flip quicknote`** ❓ (Need to check)
   - File: [internal/commands/quicknote.go](../internal/commands/quicknote.go)
   - Quick check needed

### ⚠️ Partially Brain-Type Aware

These commands use SOME brain-aware components but may have gaps:

1. **`flip init`** ⚠️
   - Detects brain type but may not create all necessary structures
   - Should validate against BRAIN_TYPES_REFERENCE.md

2. **`flip new`** (wizard) ⚠️
   - Uses brain.Creator which is brain-aware
   - But may not validate all conventions

## Impact Analysis

### Architecture Decision: Task System

The task system is **intentionally brain-agnostic**:

- **Format:** `- [x] Task` + `property::` syntax works everywhere
- **Directory:** `tasks/` is flip's own convention, doesn't conflict with other tools
- **Parser:** Single parser for all brains = consistent behavior
- **Benefit:** Users can switch brain types without losing task data

This is the correct architectural choice - flip provides a unified task experience.

### Medium Priority

**Quicknote:**
- Less critical but should be brain-aware
- May create notes in wrong location/format

## Documentation Accuracy

### Official Documentation Sources ✅

Brain type conventions in BRAIN_TYPES_REFERENCE.md were derived from:

1. **Logseq** ✅
   - Source: Official Logseq docs + GitHub code inspection
   - Verified:
     - `journals/` with `YYYY_MM_DD.md` format (underscores!)
     - `pages/` for content
     - Property bullets: `key:: value`
     - Block references: `((uuid))`

2. **Obsidian** ✅
   - Source: https://help.obsidian.md/
   - Verified:
     - `.obsidian/` marker
     - Flexible structure
     - YAML frontmatter support
     - Configurable daily notes path
     - `attachments/` default (but configurable)

3. **Dendron** ✅
   - Source: https://wiki.dendron.so/
   - Verified:
     - `dendron.yml` marker
     - Dot-notation hierarchy
     - Required frontmatter: `id`, `title`, `desc`, `updated`, `created`
     - Unix timestamps (seconds, not ms/ns)
     - Daily format: `daily.YYYY.MM.DD.md` (dots!)

4. **Foam** ⚠️ PARTIAL
   - Source: https://foambubble.github.io/foam/
   - Verified:
     - Wikilinks core feature
     - VS Code based
     - `.foam/` or `.vscode/foam.code-snippets` marker
   - **Assumptions:**
     - `journal/` for daily notes (common convention, NOT documented)
     - `notes/` for general notes (common convention)
     - `attachments/` for assets (common convention)
   - **Note:** Foam is VERY minimal - most structure is user-defined

5. **Flip** ✅
   - Source: Our own design
   - Fully documented

### Potential Inaccuracies ⚠️

1. **Foam Conventions**
   - Most conventions are ASSUMPTIONS based on common patterns
   - Foam doesn't enforce structure
   - Should add disclaimer in docs

2. **Obsidian Daily Notes**
   - Path is configurable, we assume `Daily Notes/`
   - Format is configurable, we assume `YYYY-MM-DD`
   - Need to handle detection of actual config

3. **Logseq Asset Paths**
   - Sometimes `../assets`, sometimes `assets/`
   - Our detection handles both

## Recommendations

### Immediate (Priority 1)

1. **Fix Task System** ❗
   - Add brain type detection to all task commands
   - Implement brain-specific task formats
   - Update task scanner to respect brain conventions

2. **Document Limitations**
   - Add disclaimer about Foam assumptions
   - Note Obsidian configurability
   - Be transparent about conventions vs. requirements

### Short-term (Priority 2)

3. **Add Brain-Type Validation**
   - Create test suite that validates each brain type
   - Test note/task/journal creation in each type
   - Verify conventions match docs

4. **Improve Obsidian Detection**
   - Parse `.obsidian/app.json` for actual daily notes path
   - Parse for actual attachments path
   - Respect user configuration

### Long-term (Priority 3)

5. **Brain-Specific Config Override**
   - Allow `.flip-brain.yaml` to override detected conventions
   - User can specify non-standard paths

6. **Convention Validation Mode**
   - `flip brain validate` command
   - Checks if brain follows expected conventions
   - Warns about deviations

## Action Items

- [ ] Fix task_new.go to detect and respect brain type
- [ ] Fix task_list.go to search brain-appropriate locations
- [ ] Fix task_browse_cmd.go scanner
- [ ] Check quicknote.go
- [ ] Add disclaimer to Foam section in BRAIN_TYPES_REFERENCE.md
- [ ] Add test for each brain type's note creation
- [ ] Document configuration override system

## Testing Checklist

For each brain type, verify:

- [ ] Flip
  - [ ] Note creation
  - [ ] Journal creation
  - [ ] Meeting creation
  - [ ] Task creation
  - [ ] Exercise creation
- [ ] Obsidian
  - [ ] Note creation
  - [ ] Journal creation
  - [ ] Meeting creation
  - [ ] Task creation
- [ ] Logseq
  - [ ] Note creation (in pages/)
  - [ ] Journal creation (in journals/, YYYY_MM_DD format)
  - [ ] Meeting creation
  - [ ] Task creation (property bullets)
- [ ] Dendron
  - [ ] Note creation (dot notation)
  - [ ] Journal creation (daily.YYYY.MM.DD format)
  - [ ] Task creation
- [ ] Foam
  - [ ] Note creation
  - [ ] Journal creation
  - [ ] Task creation

---

**Conclusion:** We have GOOD brain-type awareness in core note/journal/meeting commands and exercises, but **CRITICAL GAP in tasks system**. Documentation is mostly accurate but needs disclaimer about Foam assumptions.
