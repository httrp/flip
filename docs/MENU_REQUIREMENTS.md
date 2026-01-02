# Menu System - Anforderungen & Spezifikation

> Dokumentation der Qualitätsanforderungen und Feature-Wünsche für das flip Menu System.
> Stand: Januar 2025 | Branch: `refactor/code-cleanup-2025-01`

## Kernprinzipien

1. **i18n First** - Alle Texte über `lang.GetText()`, keine hardcodierten Strings
2. **Einheitliche Experience** - Egal ob promptui oder bubbletea, das UX muss konsistent sein
3. **Informativ aber lean** - Keine überladenen Menüs, klare Struktur
4. **Command Preview** - Immer eine Zeile die den aktuellen Befehl beschreibt
5. **Feature Completeness** - Alle zentralen flip-Funktionen über Menu erreichbar
6. **VS Code Integration** - Tasks und Extension müssen weiter funktionieren
7. **Plattformunabhängig** - macOS, Linux, Windows (so weit wie möglich)
8. **Brain Location Agnostic** - Egal wo die Brains liegen, flip funktioniert

---

## Qualitätsanforderungen (Must-Have)

| ID | Anforderung | Status | Ziel |
|----|-------------|--------|------|
| Q1 | **Stabilität bei Fenstergrößenänderung** | ❌ promptui hat Probleme | Kein Flickering, korrekte Darstellung |
| Q2 | **Keine os.Exit() in Menu-Funktionen** | ❌ 63 os.Exit() calls | Fehler zurückgeben, sauber beenden |
| Q3 | **Testbarkeit** | ❌ Keine Menu-Tests | Unit-Tests für Navigation/Logik |
| Q4 | **Wartbare Dateigröße** | ❌ 2853 Zeilen in menu.go | Max 500 Zeilen pro Datei |
| Q5 | **Konsistente UX** | ⚠️ Gemischt | Einheitliches Look & Feel |
| Q6 | **Breadcrumb-Navigation** | ✅ Vorhanden | Beibehalten |
| Q7 | **i18n Support** | ⚠️ Teilweise | 100% aller UI-Texte lokalisiert |
| Q8 | **Command Description Line** | ✅ Vorhanden | In allen Menüs konsistent |

---

## Feature-Wünsche (Nice-to-Have)

| ID | Feature | Status | Priorität |
|----|---------|--------|-----------|
| F1 | **Keyboard Shortcuts** im Hauptmenü | ❌ Nur im Task Browser | Mittel |
| F2 | **Fuzzy Search** in Listen | ❌ Nicht vorhanden | Niedrig |
| F3 | **Persistente Filter** | ❌ Reset bei jedem Aufruf | Niedrig |
| F4 | **Theming/Colors** konfigurierbar | ❌ Hardcoded | Niedrig |

---

## Architektur-Ziele

### Aktuell (Problem)

```
menu.go (2853 lines)
  └─ 17 runXxxMenu() Funktionen
  └─ 36 promptui.Select Aufrufe
  └─ Rekursive Aufrufe
  └─ Gemischte Verantwortlichkeiten
```

### Ziel-Struktur

```
menu.go              (~200 lines)  Einstiegspunkt, runInteractiveMenu()
menu_browse.go       (~300 lines)  Browse & Search Menüs
menu_create.go       (~300 lines)  Create/Add Menüs
menu_manage.go       (~300 lines)  Manage Resources, Contexts
menu_brain.go        (~300 lines)  Brain-spezifische Menüs
menu_status.go       (~200 lines)  Status, About, Help
menu_vscode.go       (~200 lines)  VS Code Integration
menu_types.go        (~150 lines)  MenuItem, Templates, Helpers
menu_exercises.go    (existiert)   Exercise Submenus
menu_git.go          (existiert)   Git Operations
```

---

## Technische Entscheidungen

### Framework-Strategie: Hybrid

| Anwendungsfall | Framework | Begründung |
|----------------|-----------|------------|
| Einfache Auswahl (5-10 Items) | promptui | Wenig Code, funktioniert |
| Browser mit Filter/Sort | bubbletea | Bessere UX, WindowSize-Handling |
| Text-Eingabe | promptui.Prompt | Einfach, ausreichend |

### UX-Konsistenz zwischen Frameworks

Um trotz unterschiedlicher Frameworks eine einheitliche Experience zu gewährleisten:

1. **Gleiche Farbpalette** - lipgloss und promptui Templates nutzen gleiche ANSI-Farben
2. **Gleiche Struktur** - Header → Items → Footer/Help
3. **Gleiche Keyboard Conventions** - `q` = quit, `Enter` = select, `↑↓` = navigate
4. **Gleiche Status-Messages** - ✅ ❌ 💡 Emojis für Feedback

### i18n Implementation

```go
// Alle UI-Texte MÜSSEN über lang.GetText() gehen
Label: lang.GetText("menu.browse.recent_label"),
Description: lang.GetText("menu.browse.recent_desc"),

// NICHT erlaubt:
Label: "Recent Notes",  // ❌ Hardcoded
```

Ausnahmen:
- Emojis (🔍, 📋, etc.) - diese sind universal
- Technische Begriffe wenn keine sinnvolle Übersetzung existiert

---

## Was am Task Browser gut funktioniert (Referenz)

Der Task Browser (`task_browser.go`) ist die Referenz-Implementation für bubbletea:

1. ✅ Fixe Viewport-Berechnung (`headerFooterLines := 5`)
2. ✅ `WindowSizeMsg` Handling für Resize
3. ✅ Immer sichtbare Keyboard-Hints
4. ✅ Status-Messages für User-Feedback
5. ✅ Saubere Model-Update-View Trennung
6. ✅ Filtering und Sorting

---

## Integrations-Anforderungen

### VS Code Tasks

Die Tasks in `.vscode/tasks.json` müssen weiterhin funktionieren:
- `flip menu` - Interaktives Hauptmenü
- `flip task browse` - Task Browser
- `flip status` - Status-Übersicht

### VS Code Extension

Die Extension (`extension/`) nutzt Commands die stabil bleiben müssen:
- Alle `flip` CLI Befehle
- JSON-Output für maschinenlesbare Ausgaben

### Brain Location Independence

flip muss funktionieren unabhängig davon wo die Brains liegen:
- Lokale Pfade (`~/brains/work`)
- Netzwerk-Pfade (`/Volumes/shared/brain`)
- Cloud-synced Ordner (Dropbox, iCloud, OneDrive)
- Relative Pfade in `.flip.yaml`

---

## Migration-Strategie

### Phase 1: Aufteilen (ohne Framework-Änderung)
1. `menu.go` in logische Dateien splitten
2. Jede neue Datei bekommt `_test.go`
3. Alle bestehenden Befehle müssen weiter funktionieren

### Phase 2: Stabilisieren
1. os.Exit() durch error returns ersetzen
2. Konsistente Error-Handling
3. i18n für alle neuen/geänderten Strings

### Phase 3: Verbessern (optional)
1. Komplexe Menüs zu bubbletea migrieren falls nötig
2. Keyboard Shortcuts
3. Fuzzy Search

---

## Erfolgs-Kriterien

- [ ] `go build` ohne Fehler
- [ ] `go test ./...` grün
- [ ] Alle Menu-Funktionen erreichbar
- [ ] Keine Regression in VS Code Integration
- [ ] Code Review bestanden
- [ ] Dokumentation aktuell
