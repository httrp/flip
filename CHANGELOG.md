# Changelog

All notable changes to flip will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.10.0] - 2026-05-25

### Added
- `flip version` command, prints CLI and bundled extension versions.
- Logseq artifact cleanup during migration (UI properties, `{{video}}` and `{{query}}` embeds, TODO/DONE markers).
- Logseq artifact detection and auto-repair in `flip brain health`.
- `ExercisesDir` field in `BrainStructure` and routing of `meeting-`/`exercise-` prefixed files during migration.
- Feature module system: toggle modules via `FLIP_FEATURE_<NAME>=0/1`. New `flip features list|enable|disable` commands.
- Structured logging (`FLIP_LOG_LEVEL`, `FLIP_LOG_FORMAT`, `--verbose`).
- Centralized error type `FlipError` with codes and user-facing hints.
- File cache and lazy directory reader for large brains.

### Changed
- Build: CLI version is injected via `-ldflags` using `VERSION` (defaults to `git describe`).
- Extension packaging: added `files` whitelist and bundled `LICENSE`.
- Replaced magic strings with constants (`DefaultWorkspaceName`, file extensions, directory names).
- Menus respect feature flags and build their items dynamically.

### Fixed
- Extension/CLI version drift is now caught by `scripts/check-extension-version.sh`.
- Logseq `- key:: value` bullet properties are correctly parsed.
- Logseq `created-at` millisecond epochs are converted to `YYYY-MM-DD`.
- `{{query ...}}` regex handles nested parentheses.
- UI-only Logseq properties (collapsed, card-*, background-color, heading) are no longer carried over during migration.

### Internal
- `menu_brain.go` split into three files; `vscode.go` split into `vscode.go` and `vscode_sync.go`.
- Health-check JSON schema extracted from `health.go` to `brain_schema.json`.
- Platform-specific code moved to `internal/platform/`.
- Test coverage improved in `commands` and `tasks` packages.

## [0.3.30] - 2026-05-24

### Added
- **Empty Variant Protection (VS Code)**: Exercise creation dialog no longer stores empty variants

### Fixed
- **Exercise Journal Links (Flip)**: New tracked entries now use flip-style markdown links (`[Name](../exercises/id.md)`) instead of wikilinks
- **Session Parsing Compatibility**: Parser now supports compact markdown-link exercise entries while keeping backward compatibility
- **Session Migration Compatibility**: Migration treats compact link-based entries as already compact and keeps brain-specific link formatting

## [0.3.29] - 2026-05-24

### Added
- **Exercise Tracking Improvements**: New compact exercise entry format and migration command integration in CLI/VS Code flows

### Changed
- **Exercise Creation UX (VS Code)**: New exercise dialog now asks whether variants should be created immediately and supports direct tracking-property setup

### Fixed
- **Duplicate `duration_min`**: Prevented duplicate duration data in tracked exercise sessions from VS Code flow

## [0.3.12]

### Added
- Meeting enhancements from code review session
