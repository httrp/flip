# Flip Brain Standard – Evaluation 2025

## Datum: 2025-12-20
## Autor: Copilot (Code Review)
## Branch: feat/flip-brain-standard

---

## 1. Gütekriterien für Brain Standards

Bevor wir den Flip Standard bewerten, definieren wir objektive Kriterien:

### 1.1 Portabilität (P)
- **P1**: Wie einfach ist der Export zu anderen Tools?
- **P2**: Werden Standardformate verwendet (Markdown, YAML, JSON)?
- **P3**: Sind Dateien ohne spezielle Software lesbar?
- **P4**: Gibt es Vendor Lock-in?

### 1.2 Lesbarkeit (L)
- **L1**: Sind Dateien für Menschen lesbar ohne Tooling?
- **L2**: Ist die Ordnerstruktur intuitiv?
- **L3**: Sind Metadaten verständlich?
- **L4**: Können Dateien mit Standard-Editoren bearbeitet werden?

### 1.3 Strukturierung (S)
- **S1**: Gibt es klare Konventionen?
- **S2**: Ist die Struktur konsistent?
- **S3**: Skaliert die Struktur bei vielen Dateien?
- **S4**: Ist Navigation möglich ohne Tool-Support?

### 1.4 Flexibilität (F)
- **F1**: Können Benutzer die Struktur anpassen?
- **F2**: Sind Erweiterungen möglich ohne Breaking Changes?
- **F3**: Unterstützt der Standard verschiedene Use Cases?

### 1.5 Interoperabilität (I)
- **I1**: Kompatibilität mit Obsidian?
- **I2**: Kompatibilität mit Logseq?
- **I3**: Kompatibilität mit Dendron/Foam?
- **I4**: Standard-Git-Workflow möglich?

### 1.6 Maschinenlesbarkeit (M)
- **M1**: Können Metadaten programmatisch extrahiert werden?
- **M2**: Ist das Schema validierbar?
- **M3**: Sind Migrationen automatisierbar?

---

## 2. Flip Brain Standard – Bestandsaufnahme

### 2.1 Verzeichnisstruktur

```
brain/
├── .flip-brain.yaml          # Brain-Marker & Konfiguration
├── journal/                  # Tägliche Einträge
│   └── YYYY-MM-DD.md
├── notes/                    # Allgemeine Notizen
├── meetings/                 # Meeting-Protokolle
│   └── YYYY-MM-DD-title.md
├── tasks/                    # Task-Dateien
│   └── todo.{context}.md
├── definitions/              # Strukturierte Definitionen
│   ├── contexts.yaml
│   ├── organizations.yaml
│   ├── people.yaml
│   └── schemas/              # Schema-Versionen
├── templates/                # Vorlagen
│   └── *-template.md
└── assets/                   # Medien
    ├── documents/
    └── images/
```

### 2.2 Brain-Marker (.flip-brain.yaml)

```yaml
brain:
  name: "danobrain"
  type: "personal"           # personal | work | project
  created: "2025-11-29T..."
  version: "1.0"
config:
  default_organization: "PERSONAL"
  author: "Your Name"
```

### 2.3 Task-Format

```markdown
- [ ] Task-Titel 📅 2025-01-15 ⏫ #context
  created:: 2025-01-10
  due:: 2025-01-15
  priority:: high
  organization:: ORG
```

**Unterstützte Formate:**
1. Emoji-Style: `📅 due ⏫ priority`
2. Dataview-Style: `due:: value`
3. Logseq-Style: `TODO/DONE`
4. Simple Markdown: `- [ ] task`

### 2.4 Templates (Go text/template)

```markdown
---
title: "{{.Title}}"
date: {{.Date}}
tags: {{.Tags}}
---

# {{.Title}}
```

### 2.5 Definitions (YAML)

```yaml
# contexts.yaml
contexts:
  - name: "danorama"
    description: "Arbeit bei Danorama"
    color: "#blue"
```

---

## 3. Vergleich mit anderen Brain-Typen

### 3.1 Feature-Matrix

| Feature | Flip | Obsidian | Logseq | Dendron |
|---------|------|----------|--------|---------|
| **Marker-Datei** | `.flip-brain.yaml` | `.obsidian/` | `logseq/` | `dendron.yml` |
| **Journal-Format** | `YYYY-MM-DD.md` | Konfigurierbar | `YYYY_MM_DD.md` | `daily.YYYY.MM.DD.md` |
| **Link-Format** | Markdown `[]()` | Wikilinks `[[]]` | Wikilinks | Wikilinks |
| **Task-Format** | Hybrid (multi) | Standard MD | `TODO/DONE` | Standard MD |
| **Frontmatter** | YAML | YAML | Properties `::` | YAML |
| **Hierarchie** | Folders | Folders | Pages | Dot-notation |
| **Definitions** | YAML extern | Tags/Frontmatter | Properties | Tags |
| **Schema-System** | ✅ Vorhanden | ❌ Nein | ❌ Nein | ❌ Nein |
| **Templates** | Go text/template | Templater Plugin | Native | Native |

### 3.2 Stärken-Schwächen-Analyse

#### Obsidian
- ✅ Riesiges Plugin-Ökosystem
- ✅ Flexible Ordnerstruktur
- ✅ Starke Wikilinks
- ❌ Closed-Source App
- ❌ Plugin-Abhängigkeit für viele Features

#### Logseq
- ✅ Block-basiert (granular)
- ✅ Outliner-UX
- ✅ Starke Bidirektionalität
- ❌ Underscore-Datumsformat (unüblich)
- ❌ EDN-Config (kompliziert)
- ❌ Block-UUIDs machen Dateien weniger lesbar

#### Dendron
- ✅ Hierarchisches System
- ✅ Enterprise-ready
- ❌ Dot-Notation im Dateinamen (unüblich)
- ❌ Projekt-Entwicklung eingestellt

#### Flip
- ✅ Strukturierte Ordner
- ✅ Schema-System
- ✅ Standard-Markdown
- ✅ Externe Definitions
- ❌ Weniger verbreitet
- ❌ Kein Plugin-Ökosystem

---

## 4. Bewertung: Flip Brain Standard

### 4.1 Scoreboard (1-5, 5=best)

| Kriterium | Score | Begründung |
|-----------|-------|------------|
| **P1** Exportierbarkeit | ⭐⭐⭐⭐⭐ | Standard-Markdown, keine proprietären Formate |
| **P2** Standardformate | ⭐⭐⭐⭐⭐ | Markdown + YAML = universell |
| **P3** Lesbarkeit ohne Tool | ⭐⭐⭐⭐ | Gut, aber `::` Syntax braucht Kontext |
| **P4** Vendor Lock-in | ⭐⭐⭐⭐⭐ | Keins! Alles Plain-Text |
| | |
| **L1** Human-Readable | ⭐⭐⭐⭐ | Klar, aber Task-Metadaten verbose |
| **L2** Intuitive Struktur | ⭐⭐⭐⭐⭐ | journal/notes/meetings/tasks ist super |
| **L3** Metadaten | ⭐⭐⭐⭐ | YAML klar, inline `::` weniger |
| **L4** Editor-Kompatibilität | ⭐⭐⭐⭐⭐ | Jeder Texteditor funktioniert |
| | |
| **S1** Klare Konventionen | ⭐⭐⭐ | Dokumentation fehlt teilweise |
| **S2** Konsistenz | ⭐⭐⭐ | Mehrere Task-Formate = Inkonsistenz |
| **S3** Skalierbarkeit | ⭐⭐⭐⭐ | Gut durch Ordner, Tasks zentral könnten wachsen |
| **S4** Navigation ohne Tool | ⭐⭐⭐⭐ | Ordnerstruktur hilft |
| | |
| **F1** Anpassbarkeit | ⭐⭐⭐⭐ | Definitions flexibel |
| **F2** Erweiterbarkeit | ⭐⭐⭐⭐⭐ | Schema-System ermöglicht Migrationen |
| **F3** Use Cases | ⭐⭐⭐⭐ | Personal, Work, Project |
| | |
| **I1** Obsidian-Kompatibel | ⭐⭐⭐ | Markdown ja, aber keine Wikilinks |
| **I2** Logseq-Kompatibel | ⭐⭐ | Andere Task-Syntax, Datumsformat |
| **I3** Dendron/Foam | ⭐⭐⭐ | Ähnliches Konzept |
| **I4** Git-Workflow | ⭐⭐⭐⭐⭐ | Perfekt, kleine Dateien, keine Binaries |
| | |
| **M1** Metadaten-Extraktion | ⭐⭐⭐⭐⭐ | YAML + Parser vorhanden |
| **M2** Schema-Validierung | ⭐⭐⭐ | Vorhanden, aber nicht genutzt |
| **M3** Automatisierung | ⭐⭐⭐⭐ | Gute Grundlage |

**Gesamtscore: 77/100 (77%)**

---

## 5. Was ist gut und warum?

### 5.1 🌟 Strukturierte Ordner

**Die klare Trennung in journal/, notes/, meetings/, tasks/ ist exzellent!**

Warum?
- **Navigierbar**: Man findet Dinge ohne Suche
- **Skalierbar**: Jeder Ordner wächst unabhängig
- **Fokussiert**: Meetings sind bei Meetings, nicht vermischt
- **Tool-agnostisch**: Funktioniert in jedem Dateimanager

Vergleich:
- Obsidian: Freie Struktur = oft chaotisch
- Logseq: Nur journals/ und pages/ = alles gemischt
- Flip: ✅ Beste Struktur!

### 5.2 🌟 Schema-System

**Unique Selling Point! Kein anderes Tool hat das.**

```yaml
# definitions/schemas/note.yaml
version: "1.0"
fields:
  - name: title
    required: true
  - name: date
    type: date
```

Warum?
- **Versioniert**: Migrationen möglich
- **Validierbar**: Automatische Checks
- **Dokumentiert**: Schema ist Dokumentation

### 5.3 🌟 Externe Definitions

**Trennung von Daten und Metadaten ist smart!**

```yaml
# definitions/organizations.yaml
organizations:
  - name: "Danorama"
    abbreviation: "DAN"
```

Warum?
- **Zentral**: Eine Quelle der Wahrheit
- **Referenzierbar**: Tasks können ORG: DAN nutzen
- **Exportierbar**: Leicht zu migrieren

### 5.4 🌟 Standard-Markdown

**Keine Wikilink-Abhängigkeit = maximale Portabilität!**

- Obsidian: `[[Note]]` → Funktioniert nur in Obsidian
- Logseq: `[[Page]]` → Funktioniert nur in Logseq
- Flip: `[Note](notes/note.md)` → Funktioniert ÜBERALL

### 5.5 🌟 Brain-Marker-Datei

**.flip-brain.yaml ist perfekt:**
- Eindeutig identifizierbar
- Enthält Konfiguration
- Versteckt (`.` Prefix möglich)
- YAML = lesbar + parsbar

---

## 6. Was sollte verbessert werden?

### 6.1 ⚠️ Task-Format-Chaos

**Problem**: 4 verschiedene Formate werden unterstützt

```markdown
# Format 1: Emoji
- [ ] Task 📅 2025-01-15 ⏫

# Format 2: Dataview
- [ ] Task
  due:: 2025-01-15
  priority:: high

# Format 3: Logseq
- TODO Task

# Format 4: Simple
- [ ] Task
```

**Warum schlecht?**
- Inkonsistenz in derselben Brain
- Parser-Komplexität
- Verwirrung für Benutzer
- Schwieriger zu migrieren

**Empfehlung**: 
Ein primäres Format definieren, andere nur für Import unterstützen.

**Vorschlag – Flip Canonical Task Format:**
```markdown
- [ ] Task-Titel
  due: 2025-01-15
  priority: high
  context: work
```
- Kein `::` (Logseq-spezifisch)
- Kein Emoji (schwer zu tippen)
- YAML-like Einrückung (konsistent mit Frontmatter)

### 6.2 ⚠️ Fehlende Wikilink-Unterstützung

**Problem**: `[[Note]]` Links werden nicht nativ unterstützt

**Warum ein Problem?**
- Obsidian/Logseq-User erwarten Wikilinks
- Schneller zu tippen als `[Note](path/to/note.md)`
- Bidirektionale Links populär

**Empfehlung**:
- Wikilinks als **Alternative** unterstützen
- Bei Health Check in Markdown konvertieren können
- Nicht erzwingen, aber ermöglichen

### 6.3 ⚠️ Schema-System untergenutzt

**Problem**: Schema-System existiert, wird aber kaum genutzt

**Beobachtung aus echten Brains:**
- danobrain hat kein `definitions/schemas/` Verzeichnis
- Keine Schema-Validierung aktiv

**Empfehlung**:
- Default-Schemas bei `flip brain init` erstellen
- `flip brain check schema` Command
- Automatische Validierung bei Note-Erstellung

### 6.4 ⚠️ Template-Syntax zu technisch

**Problem**: Go text/template ist mächtig, aber nicht benutzerfreundlich

```markdown
{{if .Tags}}tags: {{.Tags}}{{end}}
{{range .Items}}...{{end}}
```

**Empfehlung**:
- Einfache Variable-Syntax als Default: `${title}`, `${date}`
- Go-Templates für Power-User
- Dokumentation verbessern

### 6.5 ⚠️ Definitions-Konsistenz

**Problem**: Unterschiedliche Definition-Formate

```yaml
# contexts.yaml (Brain format)
contexts:
  - name: "danorama"
    description: "..."

# vs. simple format
contexts:
  danorama:
    description: "..."
```

**Empfehlung**:
- Ein Format definieren
- Migration-Tool für das andere

### 6.6 ⚠️ Fehlende Dokumentation

**Problem**: Kein `FLIP_BRAIN_SPEC.md` oder ähnliches

Was fehlt:
- Formale Spezifikation der Ordnerstruktur
- Verpflichtende vs. optionale Verzeichnisse
- Dateinamens-Konventionen
- Metadaten-Format-Definition

---

## 7. Konkrete Verbesserungsvorschläge

### 7.1 Flip Brain Specification v1.1

Neue Datei erstellen: `docs/FLIP_BRAIN_SPEC.md`

```markdown
# Flip Brain Specification v1.1

## Required Structure
- .flip-brain.yaml (REQUIRED)
- journal/ (RECOMMENDED)
- notes/ (RECOMMENDED)

## Optional Structure
- meetings/
- tasks/
- definitions/
- templates/
- assets/

## File Naming
- Journal: YYYY-MM-DD.md (ISO 8601)
- Meeting: YYYY-MM-DD-{slug}.md
- Note: {slug}.md (lowercase, hyphens)

## Task Format
[Canonical format hier definieren]

## Link Format
- Primary: Markdown [text](path.md)
- Alternative: Wikilinks [[Note]] (resolved to notes/)
```

### 7.2 Default-Schemas erstellen

Bei `flip brain init`:
```yaml
# definitions/schemas/note.yaml
version: "1.0"
fields:
  - name: title
    type: string
    required: true
  - name: date
    type: date
    required: true
  - name: tags
    type: array
    required: false
```

### 7.3 Canonical Task Format

Ein Format als Standard, Aliases für Import:

```yaml
# .flip-brain.yaml
config:
  task_format: "flip"  # flip | logseq | dataview | emoji
```

### 7.4 Wikilink Support

```go
// Neuer Flag in Brain Config
config:
  wikilinks: true  # Enable [[Note]] syntax
```

Health Check kann dann:
- Wikilinks auflösen (notes/, pages/, meetings/ durchsuchen)
- Optional zu Markdown-Links konvertieren

---

## 8. Priorisierte Roadmap

### Phase 1: Dokumentation (Niedrig-hängend)
1. ✏️ `FLIP_BRAIN_SPEC.md` erstellen
2. ✏️ Task-Format kanonisch definieren
3. ✏️ README aktualisieren

### Phase 2: Schema-Aktivierung
1. 🔧 Default-Schemas bei `init` erstellen
2. 🔧 `flip brain check schema` implementieren
3. 🔧 Schema-Validierung bei Note-Erstellung

### Phase 3: Wikilink-Support
1. 🔧 Wikilink-Parsing in Health Check
2. 🔧 Optional: Wikilink-zu-Markdown Konverter
3. 🔧 Config-Option für Wikilinks

### Phase 4: Task-Vereinheitlichung
1. 🔧 Canonical Task Format implementieren
2. 🔧 Legacy-Format-Migration-Tool
3. 🔧 Format-Detection verbessern

---

## 9. Zusätzliche Design-Prinzipien

*Ergänzt am 21.12.2025 basierend auf Stakeholder-Feedback*

### 9.1 🎯 Template-Philosophie: "Metadata-Rich, Content-Minimal"

**Prinzip**: Templates sollten **viele Meta-Informationen** im YAML-Header anbieten, aber **minimalen Inhalt** vorgeben.

**Warum?**
- Benutzer müssen nichts löschen
- Alle relevanten Felder sind sichtbar (man vergisst nichts)
- Schneller Start – einfach ausfüllen
- Konsistente Metadaten über alle Notes

**Aktuelles Problem:**
```markdown
# {{.Title}}

## Attendees
- 

## Agenda
1. 

## Notes

## Action Items
- [ ] 
```
→ Zu viel Struktur, die man oft löschen muss!

**Besserer Ansatz:**
```markdown
---
title: "{{.Title}}"
date: {{.Date}}
type: meeting
attendees: []
tags: []
organization: {{.Organization}}
project: 
related: []
---

# {{.Title}}

```
→ Header ist reich, Content ist leer. Der Benutzer füllt, was er braucht.

**Empfehlung für Template-Guideline:**
| Bereich | Soll enthalten | Soll NICHT enthalten |
|---------|----------------|----------------------|
| YAML Header | Alle möglichen Metadaten-Felder | - |
| Content | Nur Titel `# {{.Title}}` | Vorgegebene Sections |
| Optional | Ein `<!-- Notes: -->` Kommentar | Lange Erklärungen |

### 9.2 🔧 Tasks: Funktionalität bewahren, Format kanonisieren

**Kernaussage**: Das Task-System ist ein **Herzstück von flip**. Der Task Browser mit all seinen Features (Filterung, Gruppierung, Status-Übergänge) darf **nicht beschnitten** werden!

**Klarstellung**: Ein kanonisches Format bedeutet NICHT Feature-Verlust!

```
┌─────────────────────────────────────────────────────────┐
│  KANONISCHES FORMAT  ≠  WENIGER FEATURES                │
│                                                         │
│  Format = Wie Tasks GESPEICHERT werden                  │
│  Features = Was die APP damit macht                     │
│                                                         │
│  → Parser kann weiterhin ALLE Formate LESEN             │
│  → Neue Tasks werden im kanonischen Format GESCHRIEBEN  │
│  → Migration-Tool für Legacy-Formate                    │
└─────────────────────────────────────────────────────────┘
```

**Vorschlag: Kanonisches Task-Format v2**

```markdown
- [ ] Task-Titel
  created: 2025-12-21
  due: 2025-01-15
  priority: high
  status: open
  context: work
  organization: DAN
  project: flip
  tags: [backend, urgent]
```

**Was bleibt erhalten:**
- ✅ Alle Metadaten-Felder (due, priority, status, context, org, project, tags)
- ✅ Task Browser Filterung (nach allen Feldern)
- ✅ Status-Übergänge (open → in-progress → done)
- ✅ Gruppierung (nach Projekt, Kontext, etc.)
- ✅ Recurring Tasks (falls implementiert)

**Was sich ändert:**
- 📝 Einheitliche Syntax (kein Mix aus `::`, Emojis, etc.)
- 📝 YAML-ähnliche Einrückung (konsistent mit Frontmatter)
- 📝 Parser priorisiert kanonisches Format

**Backward Compatibility:**
```go
// Parser-Strategie
1. Versuche kanonisches Format zu parsen
2. Fallback: Dataview-Style (due::)
3. Fallback: Emoji-Style (📅)
4. Fallback: Logseq-Style (TODO/DONE)
5. Fallback: Simple Markdown
```

### 9.3 🕸️ Vernetzte Notizen: Graph statt Liste

**Realität**: Ein Brain ist ein **Netz**, keine flache Liste!

```
                    ┌─────────┐
                    │ Note A  │
                    └────┬────┘
                         │ links to
              ┌──────────┼──────────┐
              ▼          ▼          ▼
         ┌────────┐ ┌────────┐ ┌────────┐
         │ Note B │ │ Note C │ │ Image  │
         └───┬────┘ └────────┘ └────────┘
             │ links to
             ▼
        ┌─────────┐
        │ Note D  │
        └─────────┘
```

**Was Notes enthalten können:**
- 🔗 Links zu anderen Notes: `[Meeting](../meetings/2025-01-15-standup.md)`
- 🖼️ Bilder: `![Diagram](../assets/images/architecture.png)`
- 📎 Dokumente: `[PDF](../assets/documents/spec.pdf)`
- 🌐 Externe URLs: `[Docs](https://example.com)`

**Flip's Stärke hier:**
- ✅ Standard-Markdown-Links funktionieren überall
- ✅ Health Check findet broken links
- ✅ Relative Pfade = portabel

**Verbesserungspotential:**

| Feature | Status | Priorität |
|---------|--------|-----------|
| Backlinks anzeigen | ❌ Nicht vorhanden | Mittel |
| Graph-Visualisierung | ❌ Nicht vorhanden | Niedrig |
| Wikilinks `[[Note]]` | ⚠️ Nur Health Check | Mittel |
| Auto-complete für Links | ❌ Nicht vorhanden | Hoch |

**Empfehlung:**
1. **Kurzfristig**: Wikilink-Support in Health Check ausbauen
2. **Mittelfristig**: `flip brain graph` Command für Visualisierung
3. **Langfristig**: Backlink-Tracking (welche Notes verlinken hierher?)

### 9.4 📐 Zukunft: Multi-Format Support

**Vision**: Neben Markdown auch visuelle Formate unterstützen

```
brain/
├── notes/
│   ├── architecture.md           # Text-Note
│   ├── system-diagram.excalidraw # Zeichnung
│   ├── flowchart.drawio          # Diagramm
│   └── whiteboard.tldraw         # Whiteboard
```

**Kandidaten für Support:**

| Format | Tool | Use Case | Komplexität |
|--------|------|----------|-------------|
| `.excalidraw` | Excalidraw | Hand-drawn diagrams | Mittel |
| `.drawio` | Draw.io | Technical diagrams | Niedrig |
| `.tldraw` | tldraw | Whiteboarding | Mittel |
| `.canvas` | Obsidian Canvas | Mind-maps | Hoch (proprietär) |

**Wie integrieren?**

```yaml
# .flip-brain.yaml
config:
  supported_formats:
    - md          # Markdown (always)
    - excalidraw  # Excalidraw drawings
    - drawio      # Draw.io diagrams
```

**Was flip tun sollte:**
- ✅ Dateien in `notes/`, `assets/` erkennen
- ✅ In Listen/Suche anzeigen
- ✅ Nicht versuchen zu parsen (nur Markdown)
- ✅ Health Check: Links zu diesen Dateien validieren
- ⚠️ Öffnen mit externem Tool (`flip open diagram.excalidraw`)

**Was flip NICHT tun sollte:**
- ❌ Eigenen Editor bauen
- ❌ Format-spezifische Features
- ❌ Konvertierung zwischen Formaten

### 9.5 👋 Onboarding: "Flip in 5 Minuten"

**Ziel**: Jeder soll sich **schnell einarbeiten** und flip **feiern** für das durchdachte Format!

**Aktueller Stand:**
- README.md existiert ✅
- DEVELOPMENT.md für Entwickler ✅
- Keine "Getting Started" Guide ❌
- Keine Format-Dokumentation ❌

**Vorschlag: Onboarding-Materialien**

```
docs/
├── QUICKSTART.md           # "Flip in 5 Minuten"
├── FLIP_BRAIN_SPEC.md      # Formale Spezifikation
├── BRAIN_STANDARD_EVALUATION.md  # Diese Analyse
└── examples/
    └── example-brain/      # Vorzeige-Brain zum Rumspielen
```

**QUICKSTART.md Struktur:**
```markdown
# Flip in 5 Minuten 🚀

## Was ist flip?
Ein CLI-Tool für strukturiertes Personal Knowledge Management.

## Installation
brew install flip  # oder go install

## Dein erstes Brain
flip brain init my-brain
cd my-brain

## Täglicher Workflow
flip journal         # Heute's Journal öffnen
flip note "Idee"     # Neue Notiz erstellen
flip task add "..."  # Task hinzufügen
flip task browse     # Tasks durchstöbern

## Das war's!
Dein Brain ist jetzt bereit. Alle Dateien sind plain Markdown.
```

**Warum wichtig?**
- Erste Erfahrung prägt den Eindruck
- Niedrige Einstiegshürde = mehr Adoption
- Dokumentation = Vertrauen ins Projekt

---

## 10. Fazit

### Der Flip Brain Standard ist **solide** (77/100)!

**Top 3 Stärken:**
1. 🏆 Strukturierte Ordner (beste im Markt)
2. 🏆 Schema-System (einzigartig)
3. 🏆 Standard-Markdown (maximale Portabilität)

**Top 3 Verbesserungsbereiche:**
1. ⚠️ Task-Format vereinheitlichen (ohne Feature-Verlust!)
2. ⚠️ Dokumentation/Spezifikation (QUICKSTART, SPEC)
3. ⚠️ Schema-System aktivieren

**Neue Design-Prinzipien:**
1. 📝 Templates: Metadata-rich, Content-minimal
2. 🔧 Tasks: Kanonisches Format, volle Backward-Compatibility
3. 🕸️ Vernetzung: Graph-Denken, nicht Listen-Denken
4. 📐 Multi-Format: Offen für .excalidraw, .drawio, etc.
5. 👋 Onboarding: "Flip in 5 Minuten" als Ziel

**Vergleich mit Konkurrenz:**
- vs. Obsidian: Weniger Plugins, aber portabler
- vs. Logseq: Weniger Block-Power, aber lesbarere Dateien
- vs. Dendron: Ähnlich strukturiert, aber mit Schema-System

**Vision:**
> Flip soll der **portabelste, strukturierteste und am besten dokumentierte** PKM-Standard werden – 
> ein Format, das jeder in 5 Minuten versteht und das trotzdem mächtige Features wie den Task Browser bietet.

---

## 11. Aktualisierte Roadmap

### Phase 1: Dokumentation & Templates (Niedrig-hängend)
1. ✏️ `FLIP_BRAIN_SPEC.md` erstellen
2. ✏️ `QUICKSTART.md` – "Flip in 5 Minuten"
3. ✏️ Templates überarbeiten (Metadata-rich, Content-minimal)
4. ✏️ Task-Format kanonisch dokumentieren

### Phase 2: Schema & Onboarding
1. 🔧 Default-Schemas bei `init` erstellen
2. 🔧 Example-Brain als Vorlage
3. 🔧 `flip brain check schema` implementieren

### Phase 3: Vernetzung & Links
1. 🔧 Wikilink-Support ausbauen
2. 🔧 `flip brain graph` für Visualisierung
3. 🔧 Backlink-Tracking (optional)

### Phase 4: Multi-Format & Zukunft
1. 🔧 `.excalidraw`, `.drawio` Dateien erkennen
2. 🔧 `flip open` für externe Formate
3. 🔧 Health Check für Multi-Format Links

---

*Erstellt während Code Review 2025, Branch: feat/flip-brain-standard*
*Erweitert: 21.12.2025 – Design-Prinzipien & Stakeholder-Feedback*
