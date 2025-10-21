# flip
steroids for my 2nd brain

## Overview

Flip is an intelligent assistant designed to supercharge personal knowledge management workflows.

**Flip doesn't store your notes and tasks directly** - instead, it works with your existing external 2nd brain directories (Obsidian vaults, Logseq graphs, Dendron workspaces, or plain markdown folders).

## Installation

### For Development & Daily Use (Recommended)

If you're actively developing flip while using it:

```bash
# Clone the repository
git clone https://github.com/danorama-dh/flip.git
cd flip

# Option 1: Create dev symlink (recommended for development)
make dev-link
# This creates a symlink /usr/local/bin/flip -> your local build
# After any code changes, just run 'make build' and changes are live

# Option 2: Install to GOPATH/bin
make install
# Requires ~/go/bin in your PATH
```

### For End Users (Coming Soon)

Once flip is ready for public release:

```bash
# Via Homebrew (macOS/Linux)
brew install danorama/tap/flip

# Via go install
go install github.com/danorama-dh/flip/cmd/flip@latest

# Manual download
# Download binary from releases page
curl -L https://github.com/danorama-dh/flip/releases/latest/download/flip-darwin-amd64 -o flip
chmod +x flip
sudo mv flip /usr/local/bin/
```

### Verify Installation

```bash
flip --version
flip status
```

## Architecture

Flip organizes your brains using a simple two-level structure:

- **Brain**: A single brain directory (e.g., Obsidian vault, Logseq graph, or markdown folder)
- **Workspace**: A collection of related brains that you work with together

Think of it like VS Code workspaces: one workspace can contain multiple folders. In Flip, one workspace can contain multiple brains.

**The "default" Workspace:**
- Automatically created when you initialize or create your first brain
- Contains ALL brains that flip knows about on this machine
- Perfect for getting a complete overview across all your knowledge bases

Example:
```
~ Workspace: "default"
|- + Brain: personal-notes
|- + Brain: work-notes
|- + Brain: journal
`- + Brain: projects

~ Workspace: "personal"
|- + Brain: personal-notes #
`- + Brain: journal

~ Workspace: "work"
|- + Brain: work-notes #
`- + Brain: projects
```

## Quick Start

### Check Status

```bash
# See overview of all workspaces and brains
flip
# or
flip status
```

### Create Your First Brain

```bash
# Guided setup - creates new brain
# Automatically adds to "default" workspace
flip new
```

### Or Initialize Existing Directory

```bash
# Initialize a directory as a brain
# Automatically adds to "default" workspace
flip init ~/my-notes
```

### Organize with Workspaces

```bash
# Create a workspace for related brains
flip workspace create personal

# Switch to it
flip workspace switch personal

# Add brains to the active workspace
flip brain add ~/Obsidian/MyVault --name my-vault --default
flip brain add ~/Documents/Journal --name journal

# The "default" workspace always contains ALL brains
flip workspace switch default
flip brain list  # shows all brains on this machine
```

### Switch Between Workspaces

```bash
# List all workspaces
flip workspace list

# Switch to different workspace
flip workspace switch work

# Commands now operate on the active workspace
flip brain list
```

### Manage Brains

```bash
# List brains in active workspace
flip brain list

# Set which brain is default for new content
flip brain set-default my-vault

# Remove a brain (files stay intact)
flip brain remove journal
```

## What Flip Does

- Detect existing systems: Automatically recognizes Obsidian, Logseq, Dendron, or markdown folders
- Smart initialization: Creates compatible structure without breaking existing setups  
- System-specific templates: Generates templates using each system's native syntax
- Multi-workspace management: Connect multiple 2nd brain folders simultaneously
- Preserve compatibility: Maintains full compatibility with your existing tools

## Scope

Use Flip to

- Initialize new 2nd brain structures (compatible with major tools)
- Connect and manage multiple external knowledge repositories  
- Generate system-specific templates and structures
- *[Coming soon]* Track your tasks across workspaces
- *[Coming soon]* Summarize your notes
- *[Coming soon]* Query your 2nd brain using natural language

## Tech

- **Primary**: Flip is primarily built in Golang
- **Format Agnostic**: Flip supports Markdown, plain text, and structured formats
- **Multi-Source**: You can connect to multiple knowledge repositories simultaneously (e.g. different folders for private and business notes)
- **LLM-Integration** Flip supports multiple (local) LLMs to query your 2nd brain using natural language

## Supported Systems

## Commands Reference

### Workspace Commands

Workspaces are collections of related brains.

```bash
flip workspace create <name>           # Create new workspace
flip workspace list                    # List all workspaces
flip workspace switch <name>           # Switch active workspace
flip workspace remove <name>           # Remove workspace (brains stay)
flip workspace rename <old> <new>      # Rename workspace
```

### Brain Commands

Brains are individual 2nd brain directories. Commands operate within the active workspace.

```bash
flip brain add <path> [--name NAME] [--default]  # Add brain to active workspace
flip brain list                                  # List brains in active workspace
flip brain set-default <name>                    # Set default brain
flip brain remove <name>                         # Remove brain (files stay)
```

### Other Commands

```bash
flip status              # Show overview (default when running just 'flip')
flip new                 # Create new brain with guided setup
flip init <path>         # Initialize existing directory
flip scan [path]         # Scan for existing brains
```

## Key Concepts

- **Brain**: A single knowledge base directory (Obsidian vault, Logseq graph, etc.)
- **Workspace**: A collection of related brains you work with together
- **Default Workspace**: Automatically created; contains ALL brains flip knows about on this machine
- **Active Workspace**: The workspace you're currently working in. All brain commands operate within this workspace.
- **Default Brain**: The brain where new content is created by default (within a workspace).
- **Auto-Registration**: Every brain you create or initialize is automatically added to the "default" workspace.

Flip automatically detects and works with:

### 🟠 Obsidian Vaults
- **Detection**: `.obsidian/` folder, `app.json`, `workspace.json`
- **Compatibility**: Full support for wikilinks, YAML frontmatter, templates
- **Structure**: Creates `Daily Notes/`, `Templates/`, `Projects/`, `Meetings/`
- **Templates**: Uses Obsidian template syntax (`{{date:YYYY-MM-DD}}`, `{{title}}`)

### 🔵 Logseq Graphs  
- **Detection**: `.logseq/` folder, `config.edn`, `journals/`, `pages/`
- **Compatibility**: Block references, page links, task states
- **Structure**: Respects existing `journals/` and `pages/`, adds flip organization
- **Templates**: Block-based structure (`- ## Section`, `TODO` tasks)

### 🟢 Dendron Workspaces
- **Detection**: `dendron.yml`, `.dendron.cache.json`, vault directories
- **Compatibility**: Hierarchical structure, schemas
- **Structure**: Works with vault structure, adds flip definitions
- **Templates**: Hierarchical naming, schema-compatible

### ⚫ Plain Markdown Folders
- **Detection**: Contains `.md` files but no specific system markers
- **Compatibility**: Works with any markdown-based system
- **Structure**: Creates standard flip structure
- **Templates**: Generic markdown with flip conventions

### Empty Directories
- **Perfect for**: New 2nd brain setups
- **Creates**: Full flip structure compatible with migration to other systems
- **Future-proof**: Easy to migrate to Obsidian, Logseq, or Dendron later

## Standards & Formats

### Note Types

#### Journal Notes
Journal notes follow the format `YYYY-MM-DD.md` and contain:

```markdown
# 2025-08-12

## General Thoughts
- 

## Tasks
- [ ] 

## Notes
- 

```

#### Meeting Notes
Meeting notes use the format `YYYY-MM-DD-[ORG]-meeting-title.md` or `YYYY-MM-DD-[ORG]-[CTX]-meeting-title.md`:

```markdown
# Meeting: [Title] - 2025-08-12

**Date:** 2025-08-12
**Time:** [HH:MM - HH:MM]
**Organization:** [ORG] - [[Organization Name]]
**Context:** [CTX] - [[Context Name]] _(optional)_
**Participants:** [PER1], [PER2], [[External Person]]
**Type:** [standup|planning|review|retrospective|other]

## Agenda
- 
- 

## Discussion
- 

## Decisions
- 

## Action Items
- [ ] [Task] - @[PER1] - [ORG]/[CTX] - due: YYYY-MM-DD
- [ ] [Task] - @[PER2] - [ORG] - due: YYYY-MM-DD

## Next Steps
- 
```

#### General Notes
General notes follow a flexible structure with mandatory frontmatter:

```markdown
---
title: "Note Title"
date: 2025-08-12
tags: [tag1, tag2, tag3]
type: note
status: active
---

# Note Title

## Summary
Brief summary of the note content.

## Content
Main note content here.

## Links
- [[Related Note 1]]
- [[Related Note 2]]
- [External Link](https://example.com)

## References
- 
```

### Task Management

#### Task Format
Tasks use standardized markdown checkboxes with metadata:

```markdown
- [ ] Task description [ORG] #tag1 #tag2 @[PER] due:2025-08-15 priority:high
- [ ] Context task [ORG]/[CTX] @[PER] due:2025-08-15 context:meeting
- [x] Completed task [ORG] [OK] 2025-08-12
- [~] Cancelled task [ORG]/[CTX] [X] 2025-08-12
```

#### Task Metadata
- **Organization:** `[ORG]` - Required organizational context
- **Context:** `[ORG]/[CTX]` - Optional context or area
- **Tags:** `#work #personal #urgent`
- **Assignee:** `@[PER]` or `@[[Person Name]]` for external people
- **Due Date:** `due:YYYY-MM-DD`
- **Priority:** `priority:high|medium|low`
- **Estimate:** `est:2h` or `est:30m`
- **Context:** `context:meeting|email|call|idea`

#### Task States
- `[ ]` - Open/Todo
- `[x]` - Completed
- `[~]` - Cancelled  
- `[>]` - Forwarded/Rescheduled
- `[!]` - Important/Urgent
- `[?]` - Question/Needs clarification
- `[-]` - In Progress

### Definition Management

Flip manages all recurring entities in dedicated files within each repository:

#### Organizations (`definitions/organizations.yaml`)
```yaml
organizations:
  WORK:
    name: "Work"
    type: "professional"
    description: "Professional work and career"
    color: "#4ECDC4"
  PERSONAL:
    name: "Personal"
    type: "personal" 
    description: "Personal life and activities"
    color: "#45B7D1"
  LEARNING:
    name: "Learning"
    type: "development"
    description: "Learning and skill development"
    color: "#96CEB4"
```

#### Contexts (`definitions/contexts.yaml`)
```yaml
contexts:
  WORK:
    MEETINGS:
      name: "Meetings & Collaboration"
      status: "active"
      description: "Team meetings and collaborative work"
    PROJECTS:
      name: "Active Projects"
      status: "active"
      description: "Current work projects and deliverables"
  PERSONAL:
    HEALTH:
      name: "Health & Fitness"
      status: "ongoing"
      description: "Health and fitness related activities"
    FINANCE:
      name: "Personal Finance"
      status: "ongoing"
      description: "Financial planning and management"
  LEARNING:
    TECH:
      name: "Technology Learning"
      status: "active"
      description: "Learning new technologies and tools"
    LANGUAGES:
      name: "Language Learning"
      status: "active" 
      description: "Learning new languages"
```

#### People (`definitions/people.yaml`)
```yaml
people:
  SELF:
    name: "Your Name"
    organization: "WORK"
    role: "Your Role"
    email: "your.email@example.com"
  JS:
    name: "Jane Smith"
    organization: "WORK"
    role: "Colleague"
    email: "jane.smith@company.com"
  DR_MUELLER:
    name: "Dr. Mueller"
    organization: "PERSONAL"
    role: "Doctor"
    phone: "+49-123-456789"
```

### Usage Examples

#### Meeting Note Examples
```
2025-08-12-WORK-weekly-standup.md          # Work organization meeting
2025-08-12-WORK-PROJECTS-design-review.md  # Work projects context meeting
2025-08-12-PERSONAL-doctor-appointment.md  # Personal meeting
```

#### Task Examples
```markdown
- [ ] Review wireframes [WORK]/[PROJECTS] @JS due:2025-08-15 priority:high
- [ ] Prepare presentation [WORK] @SELF due:2025-08-14 context:meeting
- [ ] Call dentist [PERSONAL]/[HEALTH] @SELF due:2025-08-16 priority:low
- [ ] Study Go patterns [LEARNING]/[TECH] @SELF due:2025-08-13 est:1h
```

### Folder Structure

```
your-second-brain/
├── definitions/
│   ├── organizations.yaml
│   ├── contexts.yaml
│   └── people.yaml
├── journal/
│   ├── 2025-08-12.md
│   ├── 2025-08-13.md
│   └── ...
├── meetings/
│   ├── 2025-08-12-WORK-weekly-standup.md
│   ├── 2025-08-12-WORK-PROJECTS-design-review.md
│   └── ...
├── notes/
│   ├── WORK-processes.md
│   ├── PERSONAL-goals.md
│   └── ...
├── tasks/
│   ├── inbox.md
│   ├── WORK-tasks.md
│   ├── PERSONAL-tasks.md
│   └── LEARNING-tasks.md
├── templates/
│   ├── journal-template.md
│   ├── meeting-template.md
│   └── note-template.md
└── assets/
    ├── images/
    └── documents/
```

### Linking & References

#### Internal Links
- `[[Note Title]]` - Link to another note
- `[[Note Title#Section]]` - Link to specific section
- `[[Note Title|Display Text]]` - Link with custom display text

#### Tags
- `#tag` - Simple tag
- `#category/subcategory` - Hierarchical tags
- `#project/alpha` - Project-specific tags

#### People
- `@[PER]` - Person reference using abbreviation
- `@[[Person Name]]` - Formal person link for external people
- Contact info stored in people.yaml

### Compatibility

These standards maintain compatibility with:
- **Logseq:** Block references, page links, task states
- **Obsidian:** Wikilinks, tags, frontmatter
- **Dendron:** Hierarchical structure, schemas
- **Standard Markdown:** Pure markdown fallback
- **Git:** Version control friendly file formats

### Configuration

Flip uses a `.flip.yaml` configuration file:

```yaml
# .flip.yaml
repositories:
  - name: "personal"
    path: "/path/to/personal/notes"
    type: "primary"
  - name: "work"
    path: "/path/to/work/notes"
    type: "secondary"

definitions:
  auto_resolve: true
  validate_on_create: true
  suggest_missing: true

templates:
  journal: "templates/journal-template.md"
  meeting: "templates/meeting-template.md"
  note: "templates/note-template.md"

defaults:
  author: "Your Name"
  timezone: "Europe/Berlin"
  date_format: "2006-01-02"
  default_organization: "WORK"

llm:
  provider: "ollama"
  model: "llama2"
  endpoint: "http://localhost:11434"
```

## Commands Reference

### Initialization Commands

```bash
# Initialize new brain in current directory
flip init

# Initialize new brain in specific directory
flip init /path/to/new/brain

# Initialize in existing directory (interactive mode)
flip init /path/to/existing/vault

# Force initialization even if directory contains files
flip init /path/to/existing/vault --force

# Use predefined template
flip init /path/to/new/brain --template personal
flip init /path/to/new/brain --template work
flip init /path/to/new/brain --template learning
```

### Workspace Management

```bash
# Add external workspace
flip workspace add /path/to/brain

# Add with custom name
flip workspace add /path/to/obsidian/vault --name "my-vault"

# Add and set as default
flip workspace add /path/to/brain --name "primary" --default

# List all workspaces
flip workspace list

# Remove workspace (files stay intact)
flip workspace remove "workspace-name"
```

### Configuration

Flip stores workspace configuration in `~/.config/flip/workspaces.json`:

```json
{
  "version": "1.0",
  "workspaces": [
    {
      "name": "my-vault",
      "path": "/Users/you/Documents/Obsidian Vault",
      "type": "obsidian",
      "description": "Obsidian Vault detected",
      "default": true
    },
    {
      "name": "work-notes", 
      "path": "/Users/you/work/logseq-graph",
      "type": "logseq",
      "description": "Logseq Graph detected",
      "default": false
    }
  ]
}
```

## Scan for Existing Workspaces

Flip kann dein Home-Verzeichnis oder einen beliebigen Pfad nach existierenden 2nd brain Workspaces durchsuchen:

```bash
# Scan Home-Verzeichnis (Standard)
flip scan

# Scan einen bestimmten Ordner
flip scan ~/Documents
flip scan /Volumes/ExternalDrive/Notes
```

**Scan-Erkennungskriterien:**
- **Obsidian Vault:** `.obsidian/` Ordner + viele `.md`-Dateien
- **Logseq Graph:** `.logseq/` Ordner + `journals/` oder `pages/` + viele `.md`-Dateien
- **Dendron Workspace:** `dendron.yml` im Root + viele `.md`-Dateien
- **Flip Brain:** `.flip-brain.yaml` oder `.flip.yaml` im Root

Systemordner und irrelevante Verzeichnisse werden automatisch übersprungen. Die Scan-Tiefe ist begrenzt, um Performance zu gewährleisten. Treffer werden nur angezeigt, wenn sie die typischen Marker und ausreichend viele Markdown-Dateien enthalten.

---

## Architecture: Flip, Workspaces, and Configuration

- **Flip application (binary):** Installed once and available in your PATH. Runs commands like `flip init`, `flip new`, `flip workspace add`, `flip scan`.
- **Workspaces (2nd brain folders):** Your note/task directories, living anywhere on your system (e.g., `~/flap`, `~/Documents/WorkBrain`, external drives). Each can be Flip-enabled (via `.flip-brain.yaml`/`.flip.yaml`) or native (Obsidian/Logseq/Dendron).
- **Workspace registry (workspaces.json):** Central registry listing all connected workspaces. Stored in your user config directory (NOT in the project):
  - macOS/Linux: `~/.config/flip/workspaces.json` (via `os.UserConfigDir()`)
  - Windows: `%APPDATA%\flip\workspaces.json`

### How it fits together
1. Create or initialize a brain:
   - `flip new` creates a new folder with a chosen structure (flip/dendron/obsidian/logseq) and initializes it.
   - `flip init /path/to/existing` enables an existing folder and makes it flip-compatible.
2. Connect it once:
   - `flip workspace add /path/to/brain` adds it to the central registry (workspaces.json).
3. Use flip globally:
   - Run global actions across all connected workspaces (search, summarize, tasks) regardless of their physical locations.

This keeps your data independent and portable (folders live anywhere), while flip provides a central, tool-agnostic control plane.