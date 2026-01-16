#!/usr/bin/env python3
"""
Repair journal links to use correct relative paths from journal to target files
"""

import os
import re
import sys
from pathlib import Path

def fix_links_in_journal(journal_file, brain_path):
    """Fix all markdown links in a journal file to use correct relative paths"""
    
    if not os.path.exists(journal_file):
        return False
    
    journal_dir = os.path.dirname(journal_file)
    
    with open(journal_file, 'r') as f:
        content = f.read()
    
    original_content = content
    
    def fix_link(match):
        text = match.group(1)
        path = match.group(2)
        
        # Skip if already using relative path (..)
        if path.startswith('../') or path.startswith('..\\'):
            return match.group(0)
        
        # Skip if it's a URL or anchor
        if path.startswith('http://') or path.startswith('https://') or path.startswith('#'):
            return match.group(0)
        
        # If path is brain-relative (like "tasks/todo.md"), convert to journal-relative
        if not path.startswith('/'):
            try:
                full_path = os.path.join(brain_path, path)
                full_path = os.path.normpath(full_path)
                
                if os.path.exists(full_path):
                    rel_from_journal = os.path.relpath(full_path, journal_dir)
                    # Normalize to forward slashes for markdown
                    rel_from_journal = rel_from_journal.replace('\\', '/')
                    return f'[{text}]({rel_from_journal})'
            except:
                pass
        
        return match.group(0)
    
    # Replace all [text](path) patterns
    new_content = re.sub(r'\[([^\]]+)\]\(([^)]+)\)', fix_link, content)
    
    if new_content != original_content:
        with open(journal_file, 'w') as f:
            f.write(new_content)
        return True
    
    return False

# Main
if __name__ == '__main__':
    repo_root = Path(__file__).parent.parent
    workspace_root = repo_root.parent
    
    fixed_count = 0
    
    print("🔍 Scanning for brains and journal files...")
    
    # Find all journal files in the workspace
    for journal_file in workspace_root.glob('**/journal/**/*.md'):
        # Get the brain root (parent of journal directory)
        brain_path = journal_file.parent.parent
        
        print(f"→ Checking: {journal_file}")
        
        if fix_links_in_journal(str(journal_file), str(brain_path)):
            print(f"  ✓ Fixed links")
            fixed_count += 1
        else:
            print(f"  - No changes needed")
    
    print("")
    print(f"✓ Journal link repair complete! ({fixed_count} files fixed)")
    print("  Fixed links to use correct relative paths from journal to target files")
