# Schema Evolution System

The Schema Evolution System provides version tracking and migration capabilities for note templates in flip brains.

## Overview

Schemas define the expected structure of notes:
- **Fields**: Metadata properties (title, date, tags, etc.)
- **Sections**: Required content sections (## Summary, ## Notes, etc.)
- **Migrations**: Automatic or manual upgrade paths between versions

## Schema Definition

Schemas are stored in `definitions/schemas/` as YAML files:

```yaml
# definitions/schemas/journal.yaml
version: "2.0.0"
name: journal
description: Daily journal entry schema

fields:
  - name: date
    type: date
    required: true
  - name: mood
    type: string
    required: false
    default: neutral
  - name: tags
    type: tags
    required: false
    aliases:
      - labels  # Old name for migration

sections:
  - name: summary
    heading: "## Summary"
    required: true
  - name: gratitude
    heading: "## Gratitude"
    required: false

migrations:
  - from_version: "1.0.0"
    to_version: "2.0.0"
    description: "Add mood field, rename labels to tags"
    automatic: true
    field_changes:
      - action: add
        new_name: mood
        default: neutral
      - action: rename
        old_name: labels
        new_name: tags
```

## Note Schema Declaration

Notes declare their schema in the frontmatter:

```yaml
# YAML Frontmatter (Flip/Obsidian)
---
schema: journal
schema_version: 2.0.0
date: 2025-12-20
mood: happy
tags: [personal, reflection]
---

## Summary
...
```

```markdown
# Logseq Properties
schema:: journal
schema_version:: 2.0.0
date:: 2025-12-20
mood:: happy
tags:: [[personal]], [[reflection]]

## Summary
...
```

## Validation

The SchemaManager validates notes against their declared schema:

```go
sm := schema.NewSchemaManager(brainPath)
sm.LoadSchemas()

result, err := sm.ValidateNote(notePath)
if !result.Valid {
    for _, err := range result.Errors {
        fmt.Println("Error:", err)
    }
}

if result.NeedsMigration {
    fmt.Printf("Note needs migration: %s → %s\n", 
        result.CurrentVersion, result.LatestVersion)
}
```

## Migration

Automatic migrations can be applied when a note is outdated:

```go
result, err := sm.MigrateNote(notePath, dryRun)
if result.Success {
    fmt.Println("Migration completed:")
    for _, change := range result.Changes {
        fmt.Println(" - " + change)
    }
}
```

## CLI Commands (Planned)

```bash
# Validate all notes against their schemas
flip schema validate

# Migrate outdated notes
flip schema migrate --dry-run
flip schema migrate

# Create schema from template
flip schema create --from templates/journal.md --name journal

# List all schemas
flip schema list
```

## Field Types

| Type | Description | Example |
|------|-------------|---------|
| `string` | Free text | `title: "My Note"` |
| `date` | ISO date | `date: 2025-12-20` |
| `tags` | List of tags | `tags: [work, project]` |
| `number` | Numeric value | `priority: 3` |
| `boolean` | True/false | `draft: true` |

## Best Practices

1. **Version Semantically**: Use semantic versioning (MAJOR.MINOR.PATCH)
2. **Document Migrations**: Always describe what changes between versions
3. **Preserve Data**: Avoid removing fields without migration paths
4. **Test Migrations**: Use `--dry-run` before applying to real notes
5. **Gradual Rollout**: Migrate notes incrementally, not all at once
