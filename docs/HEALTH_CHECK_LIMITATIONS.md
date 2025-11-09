# Health Check - Known Limitations & TODOs

## ✅ Working Well
- Flip brain detection and link checking
- Wikilink resolution with slug-style matching
- Orphaned file detection
- Markdown link validation
- Asset detection

## ⚠️ Known Limitations

### Not Yet Tested
- [ ] Logseq brains with block-references
- [ ] Obsidian vaults with complex folder structures
- [ ] Dendron workspaces with dot-notation
- [ ] Large brains (1000+ notes) - performance unknown
- [ ] Unicode filenames (emojis, umlauts, etc.)
- [ ] Symlinked files/folders in brain

### Edge Cases Not Handled
- [ ] Logseq block-references `((uuid))` - detected but not validated
- [ ] Obsidian Canvas files `.canvas` - currently ignored
- [ ] Complex relative paths `../../../folder/file.md`
- [ ] Circular symlinks could cause infinite loops
- [ ] Duplicate filenames in different folders (ambiguous wikilinks)

### Missing Features
- [ ] `--fix` flag for auto-repairs
- [ ] JSON output (implemented but not tested)
- [ ] Exclude patterns (`.flipignore`, `.gitignore` support)
- [ ] Progress bar for large brains
- [ ] Parallel scanning for performance
- [ ] Verbose mode with detailed diagnostics

## 🎯 Priority for Migration Feature

**High Priority** (needed for migration):
1. Handle large brains efficiently (streaming instead of loading all)
2. Better error messages (with suggestions)
3. Dry-run validation (don't modify files)

**Medium Priority** (nice to have):
1. Test with real Logseq/Obsidian brains
2. JSON output for programmatic use
3. Better handling of relative paths

**Low Priority** (future):
1. Auto-fix capabilities
2. Exclude patterns
3. Progress bars

## 🚀 Next Steps

**Before Migration:**
1. Add safety check: Warn if brain is very large (>1000 files)
2. Add safety check: Warn if brain has unusual structure
3. Test with at least one real Logseq/Obsidian brain

**During Migration Development:**
- Use health check for pre/post validation
- Extend health check as needed for migration requirements
- Add specific checks for migration safety

## 📝 Notes

The health check is **good enough as foundation** for migration feature.
We can improve it iteratively as we discover edge cases during migration work.

Philosophy: **Ship early, iterate based on real usage.**
