# flip Feature-Konzept: Exercises

## Motivation
Viele Lern- und Entwicklungsziele (Instrument, Sprache, Fitness, ...) erfordern regelmäßige, wiederholte Übungen. Aufgaben (Tasks) sind oft einmalig, Exercises dagegen sind auf Wiederholung und Fortschritt ausgelegt.

## Unterschiede zu Tasks
- **Tasks:** Einmalige, klar definierte Aufgaben, werden als "done" markiert
- **Exercises:** Wiederholbare Aktivitäten, Fokus auf Häufigkeit, Fortschritt, Varianten

---

## Übung-Typen

Nicht alle Exercises funktionieren gleich. Es gibt verschiedene Arten von Übungen:

### 1. **Wiederholungs-basiert** (Repetition)
Zählt einfach, wie oft die Übung gemacht wurde. Keine Varianten.
- *Beispiel:* "Morgen-Lauf", "Buch lesen", "Programmieren üben"
- *Tracking:* Datum + Dauer optional
- *Auswertung:* "Diese Woche 4x gemacht, letzte Woche 6x"
- *Minimal:* Button "Done!" → erledigt für heute

### 2. **Varianten-basiert** (Variations)
Verschiedene Ausprägungen der gleichen Übung.
- *Beispiel:* "Gitarre üben" → Varianten: Akkorde, Tonleitern, Melodie
- *Tracking:* Welche Variante, Dauer, optional Notiz
- *Auswertung:* "Akkorde: 10x, Tonleitern: 5x"
- *Minimal:* Auswahl der Variante, dann Datum

### 3. **Zielwert-basiert** (Target)
Üben mit messbarem Ziel oder Fortschritt.
- *Beispiel:* "Planks halten" (Ziel: 60 Sekunden), "Vokabeln lernen" (Ziel: 100/Tag)
- *Tracking:* Aktueller Wert, wie nah am Ziel
- *Auswertung:* Fortschritt über Zeit visualisieren
- *Minimal:* Eingabe des Wertes, Datum

### 4. **Skill-basiert** (Skill)
Übungen, die zu einer Fertigkeit gehören, mit Levels/Stufen.
- *Beispiel:* "Sprache A": Anfänger → Fortgeschrittener → Fließend
- *Tracking:* Level, Subskill, Zeit auf Level
- *Auswertung:* "B1 Level erreicht", Zeit pro Level
- *Minimal:* Level/Stufe und Datum

### 5. **Projekt-basiert** (Project)
Größere Projekte mit mehreren kleinen Übungsschritten.
- *Beispiel:* "Lied lernen" mit einzelnen Schritten: Intro, Verse, Chorus
- *Tracking:* Welcher Schritt, wie viel Progress
- *Auswertung:* "50% des Lieds gelernt"
- *Minimal:* Progress-Prozent oder Schritt-Auswahl

---

## Real-World Szenarien

### Szenario 1: Schlagzeugunterricht
**Problem:** "Ich war beim Unterricht und habe neue Übungen gelernt → wie dokumentiere ich das?"

**Lösung:**
```bash
# Nach dem Unterricht
$ flip exercise new

Name: Double Bass Drum Pedal
Type: (1) Repetition (2) Variations (3) Target (4) Skill (5) Project
> 2

Variants: Slow (80 BPM), Medium (100 BPM), Fast (120 BPM)
Description: Gelerntes vom Unterricht mit Lehrer X
Materials: (optional Link zu Video vom Unterricht)
```

**Session gleich erfassen:**
```
✓ Exercise created: double-bass-drum-pedal

? Track session now? (y/n) y
? Variant: Slow (80 BPM)
? Duration: 30 min
? Notes: Gerade vom Unterricht gelernt, noch unbeholfen
✓ Session logged
```

**Ergebnis im Brain:**
```
exercises/
├── double-bass-drum-pedal.yml
└── sessions/double-bass-drum-pedal/
    └── 2025-12-01.yml  # heute
```

---

### Szenario 2: Übungsstunde mit Varianten-Mix
**Problem:** "Ich übe heute 1 Stunde Schlagzeug mit mehreren Übungen und Varianten → wie tracke ich das?"

**Lösung A: Multiple Sessions in einer Übungsstunde**
```bash
# Session 1: Double Bass
$ flip exercise track double-bass-drum-pedal --variant "Fast (120 BPM)" --duration 15
✓ Session logged

# Session 2: Snare Rolls
$ flip exercise track snare-rolls --variant "Single Stroke" --duration 15
✓ Session logged

# Session 3: Blast Beats
$ flip exercise track blast-beats --duration 20
✓ Session logged

# Session 4: Back zu Double Bass (andere Variante)
$ flip exercise track double-bass-drum-pedal --variant "Medium (100 BPM)" --duration 10
✓ Session logged
```

**Ergebnis:** 4 separate Sessions, alle heute mit unterschiedlichen Varianten/Übungen

---

**Lösung B: Session-Gruppen/Workout (optional, später)**
```bash
# Könnte auch so sein:
$ flip workout new "Schlagzeug Mittwoch"
? Übungen: 
  1. double-bass-drum-pedal (Slow & Fast)
  2. snare-rolls (Single Stroke)
  3. blast-beats
? Duration: 60 min

# Später einfach:
$ flip workout track "Schlagzeug Mittwoch"
✓ All 4 sessions logged (verteilt auf die Übungen)
```

---

### Szenario 3: Fitness Studio mit Plan
**Problem:** "Ich habe einen Trainingsplan mit festen Übungen → wie tracke ich mehrere Übungen pro Tag?"

**Lösung: Exercise Plans**

**1. Plan erstellen (einmalig):**
```bash
$ flip exercise plan new "Anfänger Plan Montag/Mittwoch/Freitag"
? Exercises in this plan:
  1. Squats (Target: 3 sets x 12 reps)
  2. Bench Press (Target: 3 sets x 10 reps)
  3. Rows (Target: 3 sets x 10 reps)
? Save as: beginner-strength-mwf

✓ Plan created
```

**2. Tägliches Tracking:**
```bash
# Option A: Alle auf einmal (schnell)
$ flip exercise plan track beginner-strength-mwf
? Squats: 3 sets, completed ✓
? Bench Press: 3 sets, completed ✓
? Rows: Wie viele sets? 2.5 (unvollständig)
✓ Workout logged (alle 3 Sessions erstellt)

# Option B: Einzeln
$ flip exercise track squats --value "3x12" --unit "reps"
$ flip exercise track bench-press --value "3x10" --unit "reps"
$ flip exercise track rows --value "2.5x10" --unit "reps"
✓ 3 Sessions logged
```

**3. Auswertung:**
```
$ flip exercise plan stats beginner-strength-mwf

Beginner Plan - Tracking:
├─ Squats: 10 sessions (100% plan)
├─ Bench Press: 10 sessions (100% plan)
└─ Rows: 8 sessions (80% plan - 2 skipped)

Adherence: 93% (27 of 29 planned sessions done)
Last workout: gestern
Streak: 6 Workouts in a row
```

---

## Wichtige Erkenntnisse aus Szenarien

### 1. **Multiple Sessions pro Tag sind normal**
- Nicht "eine Exercise pro Tag", sondern "multiple Sessions pro Tag"
- Tracking muss schnell gehen (CLI, Shorthand)

### 2. **Varianten-Mix ist wichtig**
- Gleiche Übung, aber verschiedene Varianten/Schwierigkeiten
- Muss einfach auswählbar sein

### 3. **Pläne/Workouts brauchen Struktur**
- Manche Exercises gehören zusammen
- Ein "Workout" kann 3-5 Exercises sein
- Plan-Adherence ist ein wichtiger Metric

### 4. **Schnelle Erfassung ist kritisch**
- Im Gym/nach Unterricht: keine Zeit für komplexe Eingaben
- UI/CLI muss 3-5 Sekunden pro Session sein

### 5. **Auswertung muss praktisch sein**
- "Habe ich meinen Plan eingehalten?" (% adherence)
- "Welche Variante mache ich am häufigsten?"
- "Trend: besser oder schlechter?"

---

## Aufbau einer Exercise

### Minimale Definition (für schnelle Erfassung)
```
Name:        "Morgen-Lauf"
Type:        repetition
Frequency:   weekly (optional, nur für UI-Hinweise)
```

### Erweiterte Definition (optional)
```
Name:          "Gitarre: Akkorde üben"
Type:          variations
Description:   "Tägliches Warmup für Akkordwechsel"
Goal:          "Pro Akkord 5min spielen"
Variants:      ["Em", "Am", "C", "G"]
Tags:          ["musik", "gitarre"]
Duration:      20-30 min (erwartung)
Materials:     [link zu PDF, Video]
Related:       ["Gitarre: Tonleitern", "Gitarre Anfänger Plan"]
CreatedDate:   2025-12-01
Status:        active/inactive/paused
```

---

## Tracking: Was wird erfasst?

### Die Exercise Session (minimal)
```
ExerciseID:    "gitarre-akkorde"
Date:          2025-12-01
Type:          variations
Variant:       "Em"  (nur bei Typ "variations")
Duration:      15 min
Notes:         "Heute besserer Rhythmus"
```

### Was wird NICHT erfasst
- Keine "fertig/nicht fertig" Markierung
- Keine Deadlines
- Kein "vergessen"-Status

---

## Auswertung & Reports

### Einfache Ansichten
1. **Progress This Week**: "4 Sessions", "Letzte: gestern"
2. **Streak**: "7 Tage in Folge"
3. **History**: "Dezember: 12 Sessions, November: 15"
4. **By Variant** (nur relevant): "Em: 8x, Am: 6x, C: 4x"

### Export (später)
- CSV: Date, Variant (optional), Duration, Notes
- Einfach zu analysieren in Spreadsheet/Graph

---

## Eigenschaften einer Exercise
---

## Datenmodell (Go)

### Core: Exercise Definition
```go
type ExerciseType string

const (
    TypeRepetition ExerciseType = "repetition"  // einfach zählen
    TypeVariations ExerciseType = "variations"  // verschiedene Varianten
    TypeTarget     ExerciseType = "target"      // Messwert/Ziel
    TypeSkill      ExerciseType = "skill"       // Skill Level
    TypeProject    ExerciseType = "project"     // Multi-Step Projekt
)

type Exercise struct {
    ID           string            // z.B. "gitarre-akkorde-2025"
    Name         string            // "Gitarre: Akkorde üben"
    Type         ExerciseType      // variations
    Description  string            // Ausführliche Beschreibung
    Goal         string            // Optional: was ist das Ziel?
    
    // Für Type-spezifische Infos
    Variants     []string          // ["Em", "Am", "C", "G"]
    TargetValue  int               // z.B. 60 (Sekunden)
    TargetUnit   string            // z.B. "sec", "reps", "count"
    SkillLevels  []string          // z.B. ["Beginner", "Intermediate", "Advanced"]
    ProjectSteps []string          // z.B. ["Intro", "Verse", "Chorus"]
    
    // Allgemeines
    Tags         []string          // ["musik", "gitarre"]
    Duration     string            // "20-30 min" (Guidance)
    Materials    []Material        // Videos, Links, PDFs
    Related      []string          // IDs anderer Exercises
    Status       string            // "active", "inactive", "paused"
    
    // Tracking
    CreatedDate  time.Time
    LastSession  time.Time         // Wann letzte Session?
    SessionCount int               // Gesamt Sessions bisher
}

type Material struct {
    Type  string // "video", "pdf", "link", "note"
    Title string
    URL   string
    Path  string // Lokal in Brain
}

type ExerciseSession struct {
    ID         string    // UUID
    ExerciseID string    // Referenz zur Exercise
    Date       time.Time // Wann?
    Duration   int       // Minuten
    
    // Type-spezifische Daten
    Variant    string    // Nur für Typ "variations"
    Value      int       // Nur für Typ "target" (z.B. 45 Sekunden)
    Level      string    // Nur für Typ "skill"
    Progress   int       // Nur für Typ "project" (0-100%)
    
    Notes      string    // Optionale Notizen/Reflexion
}

// ExercisePlan: Gruppiert mehrere Exercises
type ExercisePlan struct {
    ID           string    // z.B. "beginner-strength-mwf"
    Name         string    // "Anfänger Plan Montag/Mittwoch/Freitag"
    Description  string    // Optional
    
    PlanItems    []PlanItem // Welche Exercises sind im Plan?
    
    Schedule     string    // z.B. "MWF", "daily", "weekly"
    Status       string    // "active", "completed", "paused"
    
    CreatedDate  time.Time
    StartDate    time.Time
    EndDate      time.Time // Optional: wann soll Plan fertig sein?
    
    // Stats
    SessionCount int       // Wie viele Sessions insgesamt?
}

// Ein Item im Plan
type PlanItem struct {
    ExerciseID string    // Referenz zur Exercise
    Order      int       // Reihenfolge (1, 2, 3, ...)
    Target     string    // Optional: "3x12 reps", "30 min", etc.
}

// Plan-Adherence tracking
type PlanSession struct {
    ID       string    // UUID
    PlanID   string    // Referenz zum Plan
    Date     time.Time // Wann wurde das Workout gemacht?
    
    Items    []PlanSessionItem // Für jede Exercise im Plan: was wurde gemacht?
    Notes    string    // Allgemeine Notiz zum Workout
}

type PlanSessionItem struct {
    ExerciseID string    // Welche Exercise?
    Completed  bool      // Fertig gemacht?
    Actual     string    // Was wurde tatsächlich gemacht? "3x12 reps", "25 min", etc.
    Notes      string    // Optional
}
```

### Speicherung im Brain
```
brain/
├── exercises/              # Exercise Definitionen
│   ├── double-bass-drum-pedal.yml
│   ├── snare-rolls.yml
│   └── sessions/           # Sessions für jede Exercise
│       ├── double-bass-drum-pedal/
│       │   ├── 2025-12-01.yml
│       │   ├── 2025-12-02.yml
│       │   └── 2025-12-03.yml
│       └── snare-rolls/
│           └── 2025-12-01.yml
│
├── exercise-plans/         # Plans (neu!)
│   ├── beginner-strength-mwf.yml
│   └── sessions/           # Plan-Sessions (Workouts)
│       ├── beginner-strength-mwf/
│       │   ├── 2025-12-01.yml  # Montag
│       │   ├── 2025-12-03.yml  # Mittwoch
│       │   └── 2025-12-05.yml  # Freitag
```
│       │   ├── 2025-12-02.yml
│       │   └── 2025-12-03.yml
│       └── exercise-2/
└── exercise-plans/         # Optional: Lernpläne
    └── guitar-basics.yml
```

### Beispiel YAML: Exercise (Schlagzeug)
```yaml
# exercises/double-bass-drum-pedal.yml
id: double-bass-drum-pedal
name: "Double Bass Drum Pedal"
type: variations
description: |
  Gelerntes vom Schlagzeugunterricht mit Lehrer Max.
  Fokus auf Präzision und Kontrolle bei verschiedenen Tempi.
goal: "Alle Varianten fließend spielen ohne Fehler"
variants:
  - "Slow (80 BPM)"
  - "Medium (100 BPM)"
  - "Fast (120 BPM)"
tags:
  - schlagzeug
  - drums
  - unterricht
duration: "20-30 min"
materials:
  - type: video
    title: "Double Bass Drum Tutorial"
    url: "https://youtube.com/..."
related: []
status: active
created_date: 2025-12-01
last_session: 2025-12-01
session_count: 3
```

### Beispiel YAML: Session
```yaml
# exercises/sessions/double-bass-drum-pedal/2025-12-01.yml
exercise_id: double-bass-drum-pedal
date: 2025-12-01
duration: 15
variant: "Slow (80 BPM)"
notes: "Gerade vom Unterricht gelernt, noch unbeholfen"
```

### Beispiel YAML: Plan (Fitness)
```yaml
# exercise-plans/beginner-strength-mwf.yml
id: beginner-strength-mwf
name: "Anfänger Krafttraining Mo/Mi/Fr"
description: |
  3er Split für Anfänger ohne Equipment.
  Total body Workouts.
schedule: "MWF"  # Montag, Mittwoch, Freitag
status: active
created_date: 2025-12-01
start_date: 2025-12-01
end_date: 2026-03-01  # 3 Monate Plan
plan_items:
  - exercise_id: squats
    order: 1
    target: "3 sets x 12 reps"
  - exercise_id: bench-press
    order: 2
    target: "3 sets x 10 reps"
  - exercise_id: rows
    order: 3
    target: "3 sets x 10 reps"
session_count: 12
```

### Beispiel YAML: Plan-Session
```yaml
# exercise-plans/sessions/beginner-strength-mwf/2025-12-01.yml
plan_id: beginner-strength-mwf
date: 2025-12-01  # Montag
items:
  - exercise_id: squats
    completed: true
    actual: "3 sets x 12 reps"
    notes: ""
  - exercise_id: bench-press
    completed: true
    actual: "3 sets x 10 reps"
    notes: "Letzte Serie etwas schwer"
  - exercise_id: rows
    completed: false
    actual: "2 sets x 10 reps"
    notes: "Keine Zeit mehr, nur Warm-up gemacht"
notes: "Gutes Workout, nächstes Mal mehr Zeit für Rows"
```
notes: "Heute besserer Rhythmus, aber noch unsaubere Übergänge"
```

---
## CLI-Kommandos

### Exercise-Management
```bash
# Neue Exercise anlegen
flip exercise new

# Alle Exercises anzeigen
flip exercise list

# Details + Fortschritt
flip exercise show gitarre-akkorde

# Exercise bearbeiten
flip exercise edit gitarre-akkorde

# Exercise archivieren
flip exercise archive gitarre-akkorde
```

### Session Tracking (Kern-Funktion!)
```bash
# Schnell & einfach - heute eine Session erfassen
flip exercise track gitarre-akkorde
flip exercise track gitarre-akkorde --variant Em --duration 15
flip exercise track planks --value 45 --unit sec

# Spezifisches Datum
flip exercise track gitarre-akkorde --date 2025-12-01 --variant Am --duration 20

# Mit Notiz
flip exercise track gitarre-akkorde --variant Em --duration 20 --notes "Besserer Rhythmus heute"
```

### Plan Management
```bash
# Neuen Plan erstellen
flip exercise plan new "Fitness Mo/Mi/Fr"

# Plan anzeigen
flip exercise plan show beginner-strength-mwf

# Plan für heute tracken (alle Exercises des Plans)
flip exercise plan track beginner-strength-mwf

# Plan-Statistiken
flip exercise plan stats beginner-strength-mwf
```

### Auswertung & Reports
```bash
# Statistiken für Exercise
flip exercise stats gitarre-akkorde

# Report: Alle Sessions im Dezember
flip exercise report --month december

# Vergleich: diese Woche vs. letzte Woche
flip exercise compare

# Export für Spreadsheet
flip exercise export --format csv --from 2025-11-01 --to 2025-12-01
```

---

## Zusammenfassung: Wie es funktioniert

### Die 3 Schritte (für den User)

1. **Einmalig:** Exercise anlegen (oder Plan erstellen)
   ```bash
   flip exercise new
   # oder
   flip exercise plan new
   ```

2. **Regelmäßig:** Session tracken (super schnell!)
   ```bash
   flip exercise track double-bass-drum-pedal --variant "Fast (120 BPM)" --duration 15
   # oder für Plans:
   flip exercise plan track beginner-strength-mwf
   ```

3. **Auswertung:** Fortschritt anschauen
   ```bash
   flip exercise show double-bass-drum-pedal
   flip exercise plan stats beginner-strength-mwf
   ```

### Datenflow

```
Exercise    ┐
            ├──> Sessions (im Brain als YAML)
Plan        ┐                      ↓
            ├──> Plan Sessions      Auswertung
                 (Exercise IDs)     (CSV/Stats)
```

---

## Vorteile dieses Designs

✅ **Drei intuitive Ebenen:** Exercise, Session, Plan
✅ **Schnell zu erfassen:** 10-30 Sekunden pro Session
✅ **Flexibel:** 5 verschiedene Übungs-Typen
✅ **Plan-basiert:** Trainings-/Lernpläne first-class feature
✅ **Einfache Auswertung:** Adherence, Stats, Export
✅ **Brain-nativ:** Pure YAML, kein separate DB
✅ **Skalierbar:** Von 1 bis 100+ Exercises möglich
✅ **Zukunftssicher:** Lädt zu Social, Coaching, Communities ein

---

## Nächste Implementierungs-Schritte

1. **Phase 1 (MVP):**
   - Exercise CRUD (new, list, show, delete)
   - Session tracking (track, list by exercise)
   - Basic stats (sessions/week, streak, last session)

2. **Phase 2 (Plans):**
   - Plan CRUD
   - Plan session tracking
   - Adherence reporting

3. **Phase 3 (Advanced):**
   - Export (CSV, JSON)
   - Reports (monthly, trends)
   - Integrations (Google Sheets, etc.)

4. **Phase 4 (Social):**
   - Share plans/exercises
   - Community exercises
   - Progress challenges
