# Brain Health Check

## Overview

The Brain Health Check feature helps maintain the integrity of your knowledge base by detecting:

- **Broken Links**: Wikilinks `[[note]]` and markdown links `[text](path.md)` that point to non-existent files
- **Missing Assets**: References to images, PDFs, or other files that don't exist
- **Orphaned Files**: Notes and assets that aren't linked from anywhere

## Supported Brain Types

The health checker automatically detects and works with:

- **Flip Brain** - Structured folders (journal/, notes/, meetings/, tasks/, definitions/)
- **Logseq Graph** - Block-based with journals/ and pages/
- **Obsidian Vault** - Flexible folder structure with .obsidian/ config
- **Dendron Workspace** - Hierarchical dot-notation with dendron.yml

## Usage

### Check Current Brain

```bash
flip brain check health
```

### Check Specific Brain

```bash
flip brain check health ~/my-vault
flip brain check health /path/to/logseq-graph
```

### JSON Output (for scripting)

```bash
flip brain check health --json
```

## Output Example

```
🧠 Brain Health Check
────────────────────────────────────────────────────────────
  Brain:  danobrain
  Type:   Flip Brain
  Path:   /home/user/danobrain
  Notes:  14 markdown files
  Assets: 0 files
────────────────────────────────────────────────────────────

📊 Scan Statistics
  Files scanned:   19
  Links checked:   5
  Assets checked:  0

❌ Errors (4)
  journal/2025-10-29.md:11
    → Broken wikilink: [[Kiana]]
      Target note not found in brain

  notes/example.md:15
    → Broken link: ../images/diagram.png
      Target not found: images/diagram.png

⚠️  Warnings (2)
  notes/orphaned-note.md
    → Orphaned note (not linked from anywhere)
      Consider linking this note or moving it to archive

────────────────────────────────────────────────────────────
  Summary: 4 errors 2 warnings
────────────────────────────────────────────────────────────
```

## Link Formats Supported

### Wikilinks
- Basic: `[[Note Name]]`
- With alias: `[[Note Name|Display Text]]`
- Works across all brain types

### Markdown Links
- Relative: `[text](../folder/note.md)`
- Absolute (from brain root): `[text](/notes/note.md)`
- Images: `![alt](assets/image.png)`

### Logseq Block References
- Block refs: `((block-uuid))`
- Detected but not yet validated (coming soon)

## Smart Detection

The health checker is smart about:

- **Templates**: Skipped from link checking (template placeholders like `[[{{.Organization}}]]` are ignored)
- **Entry Points**: Journal entries, inbox.md, tasks.md, etc. are expected to be orphaned
- **Brain Type**: Automatically detects Logseq vs Obsidian vs Flip and uses appropriate link resolution
- **Wikilink Resolution**: Searches appropriate folders based on brain type (pages/, notes/, etc.)

## Architecture

### Core Components

1. **detector.go** - Brain type detection and link pattern matching
   - Auto-detects brain type from directory structure
   - Provides regex patterns for different link formats
   - Handles brain-specific conventions

2. **checker.go** - Link validation and orphan detection
   - Indexes all files in brain
   - Validates each link target
   - Tracks which files are referenced
   - Identifies orphaned files

3. **reporter.go** - Formatted output
   - Colorized terminal output
   - Groups issues by severity
   - Shows detailed file paths and line numbers
   - Summary statistics

## Exit Codes

- `0` - No errors found (warnings are OK)
- `1` - Errors found (broken links or missing assets)

This allows using the health check in CI/CD pipelines:

```bash
flip brain check health || echo "Brain has integrity issues!"
```

## Future Enhancements

- [ ] Auto-fix simple issues (--fix flag)
- [ ] Validate Logseq block references
- [ ] Check for circular references
- [ ] Detect duplicate file names across folders
- [ ] Performance: Parallel scanning for large brains
- [ ] Support for exclude patterns (.flipignore)
- [ ] Integration with VS Code (show issues in Problems panel)

## Implementation Notes

The health checker uses a three-phase approach:

1. **Index Phase**: Walk directory tree and catalog all files
2. **Validation Phase**: Check each link in each markdown file
3. **Analysis Phase**: Identify orphaned files by comparing referenced vs. all files

This is memory-efficient even for large brains (10k+ notes) as it only keeps file paths in memory, not content.

## Testing

Tested on:
- Flip Brain (danobrain) - 14 notes ✅
- Logseq Graph - Coming soon
- Obsidian Vault - Coming soon
- Dendron Workspace - Coming soon
