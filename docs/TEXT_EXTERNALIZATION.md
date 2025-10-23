# Text Externalization Pattern

## Overview

Flip uses a hybrid approach for text management:
- **Short labels/messages**: `lang/en.json` with dot-notation keys
- **Long content (intro, help, etc.)**: `lang/*.en.txt` template files

## API

### lang.GetText(key string) string
Retrieves text by key. Supports dot-notation for nested keys.

```go
// Flat key (backwards compatible)
lang.GetText("welcome_banner")

// Dot-notation (hierarchical)
lang.GetText("menu.main.create_label")  // Returns: "📝 Create"
lang.GetText("prompts.continue")         // Returns: "Press Enter to return to menu..."
```

### lang.GetTextf(key string, args ...interface{}) string
Formatted text with sprintf-style placeholders.

```go
lang.GetTextf("workspace_created", wsName, wsPath)
// "✓ Workspace '%s' created at %s"
```

### lang.GetTemplate(name string) string
Returns complete text template files for longer content.

```go
lang.GetTemplate("intro")     // Reads internal/lang/intro.en.txt
lang.GetTemplate("commands")  // Reads internal/lang/commands.en.txt
```

## Migration Pattern

### Before (hardcoded):
```go
{
    Label:       "📝 Create",
    Description: "Create new content: note, meeting-note, journal, task",
    Action:      runCreateNewMenu,
}
```

### After (externalized):
```go
{
    Label:       lang.GetText("menu.main.create_label"),
    Description: lang.GetText("menu.main.create_desc"),
    Action:      runCreateNewMenu,
}
```

## JSON Structure

`internal/lang/en.json` is organized hierarchically:

```json
{
  "menu": {
    "main": {
      "create_label": "📝 Create",
      "create_desc": "Create new content: note, meeting-note, journal, task"
    },
    "browse": {
      "search_label": "🔍 Search",
      "search_desc": "Search for notes by name or content"
    }
  },
  "prompts": {
    "continue": "Press Enter to return to menu...",
    "search_in": "Search in"
  },
  "errors": {
    "no_workspaces": "📭 No workspaces found. Create one first!"
  }
}
```

## Template Files

For longer content, create `*.en.txt` files in `internal/lang/`:

- `intro.en.txt` - Introduction text
- `commands.en.txt` - Command reference
- `help.en.txt` - Help documentation

Templates are plain text (can include emojis, formatting, etc.).

## Remaining Work

**Migrated Menus:**
- ✅ Main Menu
- ✅ Browse & Search Menu  
- ✅ Create Menu
- ✅ Help Menu (uses template)

**TODO - Apply same pattern:**
- Manage Resources Menu
- Switch Context Menu
- Edit/Manage Menu
- Brain Management Menu
- Workspace Management Menu
- Task Management Menu
- All prompt strings ("Press Enter...")
- All error messages

## Benefits

1. **Maintainability**: All user-facing text in one place
2. **i18n-ready**: Easy to add de.json, fr.json later
3. **Consistency**: Reuse common strings (e.g., "Press Enter...")
4. **Separation of Concerns**: Logic ≠ Presentation
5. **Easy Updates**: Change text without touching code
6. **IDE-friendly**: JSON has good editor support
