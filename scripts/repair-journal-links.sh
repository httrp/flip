#!/bin/bash
# Repair journal links to use correct relative paths from journal to target files
# This fixes old journal entries that have incorrect relative paths

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Function to repair links in a journal file
repair_journal_file() {
    local journal_file="$1"
    local brain_path="$2"
    
    if [ ! -f "$journal_file" ]; then
        return
    fi
    
    local journal_dir=$(dirname "$journal_file")
    local temp_file="${journal_file}.tmp"
    
    # Read the file and fix all markdown links
    # Pattern: [text](path) where path should be relative to journal
    # We'll fix paths like "tasks/todo.md" to be relative from journal dir
    
    python3 << 'PYTHON_EOF'
import sys
import os
import re

journal_file = sys.argv[1]
brain_path = sys.argv[2]
journal_dir = os.path.dirname(journal_file)

with open(journal_file, 'r') as f:
    content = f.read()

# Find all markdown links: [text](path)
# We need to fix paths that are relative to brain root to be relative to journal
def fix_link(match):
    text = match.group(1)
    path = match.group(2)
    
    # Skip if already using relative path (..)
    if path.startswith('../') or path.startswith('..\\'):
        return match.group(0)
    
    # Skip if it's a URL
    if path.startswith('http://') or path.startswith('https://') or path.startswith('#'):
        return match.group(0)
    
    # If path is brain-relative (like "tasks/todo.md"), convert to journal-relative
    if not path.startswith('/'):
        # This is a relative path from brain root
        # Convert to path relative from journal
        try:
            full_path = os.path.join(brain_path, path)
            rel_from_journal = os.path.relpath(full_path, journal_dir)
            # Normalize to forward slashes
            rel_from_journal = rel_from_journal.replace('\\', '/')
            return f'[{text}]({rel_from_journal})'
        except:
            pass
    
    return match.group(0)

# Replace all [text](path) patterns
new_content = re.sub(r'\[([^\]]+)\]\(([^)]+)\)', fix_link, content)

with open(journal_file, 'w') as f:
    f.write(new_content)

print(f"Fixed: {journal_file}")
PYTHON_EOF
}

export -f repair_journal_file

# Find all brains
if [ ! -d "$REPO_ROOT" ]; then
    echo "❌ Repository not found: $REPO_ROOT"
    exit 1
fi

echo "🔍 Scanning for brains and journal files..."

# Find all journal directories and repair them
find "$REPO_ROOT"/../ -type f -name "*.md" -path "*/journal/*" 2>/dev/null | while read journal_file; do
    # Find the brain root (go up from journal directory until we find a .flip or similar)
    brain_path=$(dirname "$(dirname "$journal_file")")
    
    # Check if this looks like a brain
    if [ -d "$brain_path" ]; then
        echo "→ Repairing: $journal_file"
        repair_journal_file "$journal_file" "$brain_path"
    fi
done

echo ""
echo "✓ Journal link repair complete!"
echo "  Fixed links to use correct relative paths from journal to target files"
