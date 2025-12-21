# Example Brain - Flip Standard v1.1

This directory contains a complete example of a Flip Brain structure.
Use it as a reference or copy it to start your own brain.

## Structure

```
example-brain/
├── .flip-brain.yaml          # Brain configuration
├── journal/
│   └── 2025-01-15.md         # Example journal entry
├── notes/
│   ├── getting-started.md    # Welcome note
│   └── project-ideas.md      # Example note with links
├── meetings/
│   └── 2025-01-15-kickoff.md # Example meeting
├── tasks/
│   ├── inbox.md              # Task inbox
│   └── todo.work.md          # Work tasks
├── definitions/
│   ├── organizations.yaml
│   ├── contexts.yaml
│   ├── projects.yaml
│   ├── people.yaml
│   └── schemas/
│       ├── note.yaml
│       ├── meeting.yaml
│       ├── journal.yaml
│       └── task.yaml
└── templates/
    ├── journal-template.md
    ├── meeting-template.md
    └── note-template.md
```

## Quick Start

1. Copy this folder: `cp -r example-brain ~/my-brain`
2. Edit `.flip-brain.yaml` with your details
3. Start using flip: `cd ~/my-brain && flip journal`

## Notes

- All dates use ISO 8601 format: `YYYY-MM-DD`
- Files use lowercase with hyphens: `project-ideas.md`
- Templates follow "Metadata-rich, Content-minimal" principle
