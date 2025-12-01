# Brain Compatibility Research: Exercises/Habits/Tracking

## Obsidian

### Plugins & Patterns
- **Tracker Plugin**: Uses codeblocks with `tracker` keyword for tracking habits/metrics
  ```markdown
  ```tracker
  searchType: tag
  searchTarget: exercise/running
  ```
  
- **Dataview Plugin**: Queries metadata from frontmatter
  ```yaml
  ---
  exercise: Running
  duration: 30
  date: 2025-12-01
  ---
  ```

- **Templater**: Templates with dynamic dates, prompts
- **Periodic Notes**: Daily notes with habit tracking
- **Habit Tracker**: Uses checkboxes in daily notes

### Standard Patterns
- **Frontmatter YAML** for metadata
- **Daily Notes** format: `YYYY-MM-DD.md`
- **Tags** for categorization: `#exercise/running`
- **Dataview queries** in notes for aggregation

---

## Logseq

### Block-Based Tracking
- **Journals**: Daily entries with blocks
  ```markdown
  - Exercise:: [[Running]]
    - Duration:: 30min
    - Intensity:: Medium
  ```

- **Namespaced Properties**: `property:: value`
- **Block References**: `((block-id))`
- **Queries**: Datalog-based queries
  ```clojure
  #+BEGIN_QUERY
  {:title "Exercise Log"
   :query [:find (pull ?b [*])
           :where [?b :block/properties ?p]
                  [(get ?p :exercise) ?e]]}
  #+END_QUERY
  ```

### File Structure
- `journals/YYYY_MM_DD.md` for daily entries
- `pages/Exercise.md` for definitions/templates

---

## Dendron

### Hierarchical Notes
- Schema-based structure
- Hierarchical naming: `exercise.running.2025-12-01.md`
- Frontmatter with schema validation
  ```yaml
  ---
  id: ex-run-20251201
  type: exercise-session
  exercise: running
  duration: 30
  date: 2025-12-01
  ---
  ```

### Templates
- Template inheritance
- Schema enforcement
- Backlinks and references

---

## Foam

### Wiki-Style
- Wikilinks: `[[Exercise Running]]`
- Backlinks for relationships
- Daily notes: `YYYY-MM-DD.md`
- No specific tracking system (relies on markdown + backlinks)

---

## Common Patterns Across All

1. **Frontmatter YAML** ✅ Universal
2. **Daily Notes** ✅ Common pattern
3. **Tags** ✅ Widely supported
4. **Markdown** ✅ Core format
5. **Backlinks/Wikilinks** ✅ Standard navigation

---

## Recommendations for flip Exercises

### ✅ Use Standard Markdown + Frontmatter
```yaml
---
type: exercise
id: double-bass-drum-pedal
name: "Double Bass Drum Pedal"
exercise_type: variations
tags:
  - exercise/drums
  - drums
  - practice
variants:
  - "Slow (80 BPM)"
  - "Medium (100 BPM)"
  - "Fast (120 BPM)"
created: 2025-12-01
---

# Double Bass Drum Pedal

## Description
Gelerntes vom Schlagzeugunterricht mit Lehrer Max.
Fokus auf Präzision und Kontrolle bei verschiedenen Tempi.

## Goal
Alle Varianten fließend spielen ohne Fehler

## Materials
- [Double Bass Drum Tutorial](https://youtube.com/...)

## Sessions
- [[exercises/sessions/double-bass-drum-pedal/2025-12-01]]
```

### ✅ Session Format (Compatible)
```yaml
---
type: exercise-session
exercise_id: double-bass-drum-pedal
exercise: "[[Double Bass Drum Pedal]]"
date: 2025-12-01
duration: 15
variant: "Slow (80 BPM)"
tags:
  - exercise-session
  - drums
---

# Session: Double Bass Drum Pedal - 2025-12-01

**Variant**: Slow (80 BPM)  
**Duration**: 15 min  

## Notes
Gerade vom Unterricht gelernt, noch unbeholfen.
```

### ✅ Plan Format
```yaml
---
type: exercise-plan
id: beginner-strength-mwf
name: "Anfänger Krafttraining Mo/Mi/Fr"
schedule: ["monday", "wednesday", "friday"]
tags:
  - exercise-plan
  - fitness
exercises:
  - "[[Squats]]"
  - "[[Bench Press]]"
  - "[[Rows]]"
created: 2025-12-01
---

# Anfänger Krafttraining Mo/Mi/Fr

3er Split für Anfänger ohne Equipment.

## Exercises
1. [[Squats]] - 3 sets x 12 reps
2. [[Bench Press]] - 3 sets x 10 reps
3. [[Rows]] - 3 sets x 10 reps
```

---

## Brain-Type Specific Adaptations

### Obsidian
- Use frontmatter heavily
- Add Dataview-compatible fields
- Wikilinks to exercises: `[[Exercise Name]]`
- Support inline tags: `#exercise/running`

### Logseq
- Use namespaced properties: `exercise:: [[Running]]`
- Block-based structure in journals
- Support queries with `:exercise` property

### Dendron
- Hierarchical file naming: `exercise.running.2025-12-01.md`
- Schema validation in frontmatter
- Template inheritance

### Foam
- Simple wikilinks
- Minimal frontmatter
- Backlink-based navigation

---

## File Organization (Universal)

```
brain/
├── exercises/
│   ├── double-bass-drum-pedal.md
│   ├── snare-rolls.md
│   └── squats.md
│
├── exercises/sessions/
│   ├── double-bass-drum-pedal/
│   │   ├── 2025-12-01.md
│   │   ├── 2025-12-02.md
│   │   └── 2025-12-03.md
│   └── squats/
│       └── 2025-12-01.md
│
└── exercise-plans/
    ├── beginner-strength-mwf.md
    └── sessions/
        └── beginner-strength-mwf/
            └── 2025-12-01.md
```

**Alternative** (Logseq-style):
```
brain/
├── pages/
│   ├── Exercise - Double Bass Drum Pedal.md
│   └── Exercise Plan - Beginner Strength MWF.md
│
└── journals/
    ├── 2025_12_01.md  # Contains session blocks
    └── 2025_12_02.md
```

---

## Decision: flip Exercise Storage Strategy

### Primary Format: Markdown with Frontmatter
- ✅ Compatible with all brain types
- ✅ Queryable via Dataview (Obsidian), Logseq queries
- ✅ Human-readable
- ✅ Version control friendly
- ✅ No proprietary format

### Brain-Type Detection
When flip detects a brain type, it adapts:
- **Obsidian**: Use `exercises/` folder, heavy frontmatter, wikilinks
- **Logseq**: Use `pages/` + `journals/`, block-based sessions
- **Dendron**: Hierarchical naming, schema frontmatter
- **Foam**: Minimal frontmatter, wikilinks
- **Flip**: Full YAML + Markdown hybrid

### Universal Fields (All Brains)
```yaml
---
type: exercise | exercise-session | exercise-plan
id: unique-id
date: YYYY-MM-DD
tags: [...]
---
```
