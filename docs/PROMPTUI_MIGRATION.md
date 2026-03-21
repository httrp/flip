# promptui → bubbletea Migration Plan

> Status: **Phase 0+1 empfohlen**, Phasen 2-6 nach Bedarf  
> Erstellt: 2026-03-21  
> Kontext: Code Review 2026, Menu-Modernisierung (commits f23f8ab, 24e7b8f, da7beec)

## Ausgangslage

- **Menü-System** bereits auf bubbletea migriert (menu_new.go + ui/app.go) mit Loop-Modus
- **~300 promptui-Referenzen** verbleiben in **45 Dateien** unter `internal/commands/`
- `promptui` (github.com/manifoldco/promptui) ist seit 2020 kaum maintained
- bubbletea-Stack vorhanden: `bubbletea` v1.3.10, `bubbles` v1.0.0, `lipgloss` v1.1.0
- `charmbracelet/huh` ist **nicht** als Dependency vorhanden — kein Form-Framework

## Vorhandene bubbletea-Komponenten (internal/ui/)

| Datei | Komponente | Beschreibung |
|-------|-----------|--------------|
| app.go | `AppModel`, `RunApp()` | Root-Navigation mit MenuConfig, Stack, Loop-Modus |
| menu.go | `MenuModel`, `MenuItem` | List-basiertes Menü mit Filtering + Delegate |
| components.go | `InputModel` | Text-Input (braucht: Default, Validate) |
| components.go | `ConfirmModel` | Ja/Nein-Abfrage (braucht: Standalone-Runner) |
| components.go | `LoadingModel` | Spinner mit Nachricht |
| streaming.go | Streaming Viewport | Viewport + Spinner für AI-Streaming |
| styles.go | Lipgloss Styles | Geteilte Style-Definitionen |

## Entscheidungen

- **Kein `huh`** — Lightweight `RunSelect`/`RunInput`/`RunConfirm` Wrapper mit `bubbles` + `bubbletea`
- **Kein AltScreen für Inline-Prompts** — Nur das Hauptmenü nutzt AltScreen
- **Sequenzielle Prompts** — Komplexe Wizards nutzen aufeinanderfolgende `RunSelect`→`RunInput`-Aufrufe
- **Phasen 2-6 nur nach Bedarf** — Die promptui-Inline-Prompts funktionieren, Migration ist technische Hygiene

---

## Phase 0 — Foundation: Reusable Prompt Components

**Ziel:** 3 standalone Prompt-Runner in `internal/ui/` bauen, die von allen Command-Dateien nutzbar sind.

### 0.1 — InputModel erweitern

```go
// RunInput zeigt einen Text-Prompt und gibt den eingegebenen Wert zurück.
// Läuft inline (kein AltScreen).
func RunInput(prompt, placeholder, defaultVal string, validate func(string) error) (string, error)
```

- Default-Wert als Pre-Fill im textinput
- Validate-Callback: Fehlermeldung inline anzeigen bei ungültiger Eingabe
- Ctrl-C/Esc → leerer String + `ErrCancelled`

### 0.2 — SelectModel (neu)

```go
type SelectItem struct {
    Label string
    Value string // Rückgabewert (kann == Label sein)
}

// RunSelect zeigt eine Auswahlliste und gibt Index + Value zurück.
// Läuft inline (kein AltScreen).
func RunSelect(label string, items []SelectItem, size int) (int, string, error)
```

- Lightweight — kein Breadcrumb, keine Command-Hints
- Pfeil-Navigation + Filterung (optional)
- Size begrenzt sichtbare Items
- Ersetzt alle `promptui.Select`

### 0.3 — ConfirmModel Standalone-Runner

```go
// RunConfirm zeigt eine Ja/Nein-Frage.
func RunConfirm(question string, defaultYes bool) (bool, error)
```

- Nutzt bestehenden `ConfirmModel`, wrapped in `tea.NewProgram`

### 0.4 — Shared Error

```go
var ErrCancelled = errors.New("cancelled")
```

**Verifikation:** `go build ./...` — nur neue Dateien, keine bestehenden Änderungen.

---

## Phase 1 — Legacy-Menü-Dateien löschen (Group 1)

**11 Dateien, ~56 promptui-Refs.** Vom neuen Menü-System komplett ersetzt.

### 1.1 — Prüfen welche alten Funktionen noch referenziert werden

Aktuell ruft `executeCommandV2()` in menu_new.go noch auf:
- `runEditWorkspaceMenu()` → menu_brain_edit.go
- `runManageBrainsMenu()` → menu_brain_edit.go  
- `runViewWorkspaceConfig()` → menu_manage.go
- `runEditManageMenu()` → menu_manage.go
- `runBrainMigrationMenu()` → brain_migrate_menu.go

### 1.2 — Sicher löschbare Dateien (keine externen Referenzen)

- `menu.go` — altes Hauptmenü (jetzt `menu-legacy`)
- `menu_browse.go` — altes Browse-Menü
- `menu_create.go` — altes Create-Menü
- `menu_status.go` — altes Status-Menü
- `menu_exercises.go` — altes Exercise-Menü
- `menu_brain.go` — altes Brain-Menü
- `menu_brain_view.go` — altes Brain-Detail-Menü
- `menu_git.go` — altes Git-Menü

### 1.3 — Zu migrierende Dateien (noch referenziert)

- `menu_brain_edit.go` (15 refs) — Workspace/Brain CRUD: Selects + Prompts → `RunSelect`/`RunInput`/`RunConfirm`
- `menu_manage.go` (teilweise) — `runViewWorkspaceConfig()` und `runEditManageMenu()` extrahieren
- `menu_helpers.go` — Utility-Funktionen behalten (`getTerminalSize`, `truncateString`, `openInFileManager`), `promptui.SelectTemplates`-Factories löschen

### 1.4 — `menu-legacy` Command entfernen

- features_cmd.go: `NewMenuCommand()` (menu-legacy) Registrierung entfernen
- cmd/flip/main.go: Bereits auf `NewMenuV2Command()` umgestellt ✓

**Verifikation:** `go build ./...`, `go test ./...`, manueller Test aller menu_new.go Dispatcher-Commands.

---

## Phase 2 — Brain/Workspace Selection (Group 2)

**4 Dateien, ~21 Refs. Hohe Wiederverwendung.**

| Datei | Refs | Was |
|-------|------|-----|
| brain_select.go | 2 | `selectBrainForOperation()` — Brain aus Workspace wählen |
| brain_relocate.go | 5 | Pfad-Eingaben + Bestätigung für Brain-Umzug |
| brain_restore.go | 5 | Backup-Auswahl + Link-Verhalten |
| brain_migrate_menu.go | 9 | Migrations-Wizard (Scope, Source/Target, Mode, Confirm) |

**Ansatz:** Alle `promptui.Select` → `RunSelect`, alle `promptui.Prompt` → `RunInput`/`RunConfirm`.

---

## Phase 3 — AI & Simple Commands (Groups 5 + 7 partial)

**7 Dateien, ~25 Refs. Einfache Wins.**

| Datei | Refs | Was |
|-------|------|-----|
| ai_setup.go | 4 | Provider-Auswahl, API-Key, Bestätigung |
| ai_helpers.go | 8 | System-Prompt, Model-Auswahl, Custom-Prompt |
| ai_improve.go | 2 | Custom Improvement Instruction |
| ai_research.go | 2 | Research-Frage |
| ai_summarize.go | 2 | Custom Summarization Instruction |
| git_helpers.go | 3 | Branch/File-Auswahl |
| quickstart.go | 5 | Onboarding-Wizard |

---

## Phase 4 — Content Creation (Group 3)

**9 Dateien, ~54 Refs. Mittlere Komplexität.**

| Datei | Refs | Komplexität |
|-------|------|-------------|
| note.go | 4 | Niedrig — Titel, Brain-Auswahl, Aktion |
| quicknote.go | 4 | Niedrig — Titel, Tags, Open-in-Editor |
| journal.go | 3 | Niedrig — Datum-Modus + Custom-Datum |
| journal_link.go | 2 | Niedrig — Link-Typ |
| content_common.go | 4 | Niedrig — Ordner-Auswahl + Neu erstellen |
| notes.go | 4 | Niedrig — Notizen-Liste + Aktion |
| path_browser.go | 7 | Mittel — Dateisystem-Browser |
| template.go | 13 | Mittel — Template CRUD |
| meeting.go | 13 | **Hoch** — 12-Schritt Wizard (Org, Serie, Titel, Teilnehmer, Projekt, Kontext, Tags, Dauer) |

---

## Phase 5 — Tasks (Group 4)

**4 Dateien, ~47 Refs.**

| Datei | Refs | Komplexität |
|-------|------|-------------|
| task_new.go | 8 | Mittel — Beschreibung, Datum, Priorität, Tags |
| task_list.go | 10 | Mittel — Auflistung, Auswahl, Aktionen |
| task_helpers.go | 11 | Mittel — Shared Selects + Property-Editor |
| task_new_prompts.go | 18 | **Hoch** — Context/Project/Org-Selektoren mit Inline-Erstellung |

---

## Phase 6 — Exercises & Definitions (Groups 6 + 7)

**12 Dateien, ~103 Refs. Höchste Komplexität.**

| Datei | Refs | Komplexität |
|-------|------|-------------|
| exercise_new.go | 18 | **Hoch** — Komplett-Wizard |
| exercise_edit.go | 14 | Hoch |
| exercise_show.go | 2 | Niedrig |
| exercise_track.go | 7 | Mittel |
| exercise_plan_new.go | 6 | Mittel |
| exercise_plan_edit.go | 4 | Mittel |
| definitions.go | 2 | Niedrig |
| definitions_browse.go | 9 | Mittel |
| definitions_manage.go | 27 | **Hoch** — CRUD für 4 Entity-Typen |
| scan.go | 6 | Mittel |

---

## Phase 7 — Cleanup

1. `go mod tidy` — `promptui` aus go.mod/go.sum entfernen
2. `menu_helpers.go` — verbleibende `promptui.SelectTemplates` löschen
3. `features_cmd.go` — `menu-legacy` Registrierung entfernen (falls in Phase 1 noch nicht geschehen)
4. **Verifikation:** `grep -r promptui internal/` → 0 Ergebnisse

---

## Zusammenfassung

| Phase | Dateien | Refs | Aufwand | Empfehlung |
|-------|---------|------|---------|------------|
| 0 — Foundation | - | - | ~1 Session | **Sofort** |
| 1 — Legacy Menus löschen | 11 | ~56 | ~1 Session | **Sofort** |
| 2 — Brain/Workspace | 4 | ~21 | ~1 Session | Nach Bedarf |
| 3 — AI & Simple | 7 | ~25 | ~1 Session | Nach Bedarf |
| 4 — Content Creation | 9 | ~54 | ~1-2 Sessions | Nach Bedarf |
| 5 — Tasks | 4 | ~47 | ~1-2 Sessions | Nach Bedarf |
| 6 — Exercises & Defs | 12 | ~103 | ~2-3 Sessions | Nach Bedarf |
| 7 — Cleanup | - | - | 10 Min | Am Ende |
| **Gesamt** | **45** | **~300** | **~8-11 Sessions** | |

**Empfehlung:** Phase 0 + 1 sofort umsetzen (Foundation + Legacy aufräumen). Phasen 2-6 inkrementell, wenn man ohnehin in den jeweiligen Dateien arbeitet.
