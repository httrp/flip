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

### Changed
- Replaced magic strings with constants throughout codebase
- Menu files refactored to use dynamic item lists based on enabled features
- `main.go` now initializes features from environment at startup

### Internal
- `menu_brain.go` split into 3 files: `menu_brain.go`, `menu_brain_edit.go`, `menu_brain_actions.go`
- Embedded JSON extracted from `health.go` to `brain_schema.json`
- `vscode.go` split into `vscode.go` and `vscode_sync.go`
- Platform-specific code extracted to `internal/platform/` package
- Added 8 new tests for feature system

## [0.3.12] - 2025-01-XX

### Added
- Meeting enhancements from code review session
