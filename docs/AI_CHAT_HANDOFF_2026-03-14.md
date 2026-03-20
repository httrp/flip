# AI Chat Handoff (2026-03-14)

Purpose: Persist current implementation context before `Reload Window`.

## Implemented in this session

### 1) AI UX terminology + flow alignment
- Standardized wording around:
  - **Template** (formerly prompt note)
  - **Request** (concrete task/question)
  - **Run notes** (per-run extra instructions)
- Added/used CLI aliases for clearer flags:
  - `--template` (alias of `--prompt`)
  - `--request`
  - `--run-notes` (alias of `--prompt-extra`)

### 2) Context-source support
- CLI and extension now support explicit context files and auto-context behavior:
  - `--context-file` (repeatable)
  - `--context-auto-brain` (research flow)
- VS Code flow includes context source selection step.

### 3) Reliable multiline input in extension
- Replaced fragile unsaved-untitled flow with temporary file workflow for multiline editor input.

### 4) Pre-run review screen (VS Code extension)
- Added modal confirmation dialog before execution for:
  - Research
  - Summarize
  - Improve
- Review contains key run config:
  - Template
  - Request / Instruction
  - Context
  - Run notes
  - Model
  - Title/File/Apply mode

### 5) Best-practice template presets (new)
- During **template creation** in AI flows, user can pick preset:
  - `Research template (best practice)`
  - `Summarize template (best practice)`
  - `Improve template (best practice)`
  - `Blank custom template`
- Preset pre-fills title + template body, still editable before save/default selection.

## Files changed
- `vscode-extension/src/commands/ai.ts`
- `internal/commands/ai.go`
- `docs/VSCODE.md`
- `docs/COMMANDS.md`
- `docs/FEATURES.md`

## Validation run
- `npm run compile` (in `vscode-extension`) ✅
- `make build-extension` (repo root) ✅

## Quick test checklist after reload
1. Run **Flip: AI Research**
2. Choose/create template
3. When creating, verify preset picker appears
4. Add request + optional context + run notes
5. Verify **Review AI run configuration** modal appears
6. Confirm run executes only after explicit confirmation

## Notes
- If no prompt/template note exists for a brain, flow still works with system defaults.
- Default prompt note is auto-used when marked (`prompt_default: true`) or named `default`.
