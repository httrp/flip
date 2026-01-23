# Changelog

All notable changes to flip will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Feature Module System**: Toggle features via environment variables (`FLIP_FEATURE_<NAME>=0/1`)
  - Core features: `core`, `git`, `health` (always enabled)
  - Optional features: `tasks`, `exercises`, `templates`, `vscode`, `definitions`, `meetings`, `migration`
- New command: `flip features` - List and manage feature modules
- New command: `flip features list [--json]` - Show features with optional JSON output
- New command: `flip features enable/disable <name>` - Toggle features (session only)
- **Centralized Constants**: `constants.go` with `DefaultWorkspaceName`, file extensions, directory names
- **Code Quality Standards**: Extended `CONTRIBUTING.md` with senior-level best practices
- **Feature-Aware Menus**: Interactive menus now respect feature flags
- **Structured Logging**: `logger.go` with levels, JSON/text formats, structured fields
  - Environment config: `FLIP_LOG_LEVEL`, `FLIP_LOG_FORMAT`
  - Global verbose flag: `flip -v` / `flip --verbose`
- **Centralized Error Handling**: `errors.go` with `FlipError` type
  - Error codes: CONFIG, BRAIN, WORKSPACE, FILE, GIT, PARSE, VALIDATION, etc.
  - User-friendly hints with `WithHint()`
  - Error chaining with `WithCause()`
- **Lazy Loading & Caching**: `lazy.go` for improved performance
  - `FileCache` with TTL and LRU eviction
  - `LazyDirReader` with extension filtering and pagination
  - Auto-excludes: `.git`, `node_modules`, `.obsidian`, `.trash`

### Changed
- Replaced magic strings with constants throughout codebase
- Menu files refactored to use dynamic item lists based on enabled features
- `main.go` now initializes features and logging from environment at startup
- `config.go` uses new structured error types

### Internal
- `menu_brain.go` split into 3 files: `menu_brain.go`, `menu_brain_edit.go`, `menu_brain_actions.go`
- Embedded JSON extracted from `health.go` to `brain_schema.json`
- `vscode.go` split into `vscode.go` and `vscode_sync.go`
- Platform-specific code extracted to `internal/platform/` package
- Test coverage improved:
  - `commands`: 0.3% → 2.3%
  - `tasks`: 1.6% → 7.0%
- New test files: `config_test.go`, `errors_test.go`, `logger_test.go`, `lazy_test.go`, `checker_test.go`

## [0.3.12] - 2025-01-XX

### Added
- Meeting enhancements from code review session
