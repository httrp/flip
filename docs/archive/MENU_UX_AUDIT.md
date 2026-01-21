# Menu UX Audit & Improvements

## Gütekriterien für Terminal-Menus (promptui)

### 1. ⚡ Schnell
**Status: ✅ Gut**
- Menu lädt sofort
- Keine merkbaren Verzögerungen
- `HideHelp: true` reduziert visuellen Overhead

**Verbesserungspotential:**
- [ ] Lazy-Loading für Workspace-Config bei Submenus

### 2. 🎯 Konsistent
**Status: ⚠️ Teilweise**

**Probleme gefunden:**
- Verschiedene Template-Stile (createSimpleSelectTemplates vs createMenuItemWithCommandTemplates)
- Manche Menus haben Details-Bereich, andere nicht
- Inkonsistente Back-Button-Bezeichnungen

**Lösung:**
- Einheitliche Templates verwenden
- Konsistente Labels für Navigation (← Back, ↑ Back to Main)

### 3. 📐 Übersichtlich (auch bei wenig Platz)
**Status: ⚠️ Teilweise**

**Vorhandene Features:**
- `getTerminalSize()` für dynamische Anpassung
- `truncateString()` für lange Texte
- `calculateMenuSize()` für optimale Listenhöhe

**Probleme:**
- Manche Labels sind zu lang für schmale Terminals
- Command-Anzeige kann abgeschnitten werden
- Details-Bereich nimmt viel Platz

**Lösung:**
- Adaptive Templates die auf Terminal-Breite reagieren
- Details nur bei ausreichend Platz anzeigen

### 4. 📏 Zeilenumbrüche vermeiden
**Status: ⚠️ Problematisch**

**Probleme:**
- Keine explizite Wrap-Verhinderung
- Lange Pfade können umbrechen
- Emoji + Text + Command kann zu lang werden

**Lösung:**
- Templates mit dynamischer Breitenberechnung
- Ellipsis (...) für zu lange Texte
- Kompaktmodus für schmale Terminals

### 5. 🔗 Verwandte Funktionen ähnlich abbilden
**Status: ⚠️ Teilweise**

**Probleme:**
- Create/Edit/Delete nicht immer gruppiert
- Git-Operationen an verschiedenen Stellen
- Exercises unter Browse (statt eigene Kategorie?)

**Lösung:**
- Menu-Struktur überdenken
- Konsistente Icons pro Funktionstyp

### 6. 🧭 Intuitiv - wo findet man welche Funktion
**Status: ⚠️ Verbesserungswürdig**

**Aktuelle Struktur:**
```
Main Menu
├── Create        → Notes, Meetings, Journal, Tasks
├── Browse        → Recent, Search, Tasks, Exercises
├── Manage        → Brains, Workspaces, Templates
├── Status        → Git, Health Check
└── Help          → CLI Reference, About
```

**Probleme:**
- "Browse" enthält "Exercises" (warum nicht unter Manage?)
- "Status" vs "Manage" Abgrenzung unklar
- Keine Quick-Access für häufige Aktionen

### 7. 💬 Command-Beschreibung immer sichtbar
**Status: ✅ Implementiert**
- `createMenuItemWithCommandTemplates()` zeigt Command in derselben Zeile
- Details-Bereich zeigt Beschreibung

**Verbesserung:**
- Nur bei ausreichend Platz anzeigen
- Für alle Menus konsistent verwenden

### 8. 📁 Pfad-Autocomplete/Vorschläge
**Status: ✅ Gut (path_browser.go)**
- Interaktiver Directory Browser
- Manuelle Eingabe möglich
- Common Paths als Shortcuts

**Verbesserungspotential:**
- Tab-Completion in Prompt-Eingabe
- Fuzzy-Matching für Pfade
- Merken der letzten verwendeten Pfade

### 9. 🌍 Plattformunabhängig
**Status: ⚠️ Zu prüfen**

**Potentielle Probleme:**
- `os.UserHomeDir()` - sollte überall funktionieren ✅
- `filepath.Join()` - korrekt ✅
- Hardcoded Pfadtrenner - problematisch
- `open` Command für Editor - macOS-spezifisch!
- Terminal-Size mit `golang.org/x/term` - sollte funktionieren

**Zu prüfen:**
- [ ] `openInEditor()` Funktion
- [ ] Alle exec.Command() Aufrufe
- [ ] Pfad-Separatoren

---

## Verbesserungen umzusetzen

### Phase 1: Template-Konsolidierung
1. Adaptive Templates die auf Terminalbreite reagieren
2. Einheitliche Verwendung in allen Menus
3. Wrap-Verhinderung einbauen

### Phase 2: Menu-Struktur
1. menu.go weiter aufteilen nach logischen Gruppen
2. Konsistente Navigation (Back-Buttons)
3. Breadcrumbs überall

### Phase 3: Plattformunabhängigkeit
1. `openInEditor()` plattformübergreifend
2. Pfad-Handling prüfen

### Phase 4: Erweiterte Features
1. Fuzzy-Search in langen Listen
2. Recent/Favorites für schnellen Zugriff
3. Keyboard Shortcuts anzeigen
