# Development Guide

## Development Setup

This guide is for developers who want to actively develop flip while using it daily.

### Initial Setup

```bash
# 1. Clone and build
cd ~/universe/da/flip
make build

# 2. Create development symlink
make dev-link
# This creates: /usr/local/bin/flip -> /Users/dh/universe/da/flip/flip
# Requires sudo password once
```

### Daily Development Workflow

```bash
# 1. Make code changes in your editor
# 2. Rebuild
make build

# 3. Test immediately (flip command uses your latest build)
flip status

# No installation step needed! The symlink points to your local binary.
```

### Project Structure

```
flip/
├── cmd/flip/           # Main application entry point
├── internal/
│   ├── commands/       # CLI commands (cobra)
│   │   ├── brain.go
│   │   ├── workspace.go
│   │   ├── menu.go     # Interactive menus
│   │   ├── init.go
│   │   ├── new.go
│   │   └── ...
│   └── brain/          # Brain detection & metadata
├── lang/               # i18n translations
├── scripts/            # Helper scripts
└── Makefile
```

### Key Commands

```bash
# Build binary
make build

# Run tests
make test

# Lint code
make lint

# Smoke tests
make smoke

# Clean build artifacts
make clean

# Remove installation/symlink
make uninstall
```

### Configuration Files

Flip stores configuration in:
- **Config**: `~/Library/Application Support/flip/workspaces.json`
- **Brain Metadata**: `.flip-brain.yaml` in each brain directory

### Testing Changes

```bash
# Quick test
make build && flip status

# Full test suite
make test

# Smoke test (integration)
make smoke
```

## Git Workflow

```bash
# Feature development
git checkout -b feature/my-feature
# ... make changes ...
make build && make test
git add -A
git commit -m "feat: Add my feature"

# Code cleanup
git checkout -b fix/cleanup
# ... fixes ...
git commit -m "fix: Resolve linter warnings"

# Push to remote
git push origin feature/my-feature
```

## Release Preparation (Future)

When ready to publish flip:

### 1. Version Management

```bash
# Update version in code
# cmd/flip/main.go - add version constant
const Version = "0.1.0"

# Tag release
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

### 2. Build Releases

Use [GoReleaser](https://goreleaser.com/) for multi-platform builds:

```bash
# Install goreleaser
brew install goreleaser

# Create .goreleaser.yaml config
# Test release build
goreleaser release --snapshot --clean

# Real release (on tagged commit)
goreleaser release --clean
```

### 3. Distribution Options

**Option A: GitHub Releases**
- GoReleaser automatically creates GitHub releases
- Users download binary manually
- Simple, no infrastructure needed

**Option B: Homebrew Tap**
```bash
# Create tap repository
# danorama/homebrew-tap
# Formula: flip.rb points to GitHub releases
```

**Option C: Go Install**
- Users run: `go install github.com/danorama-dh/flip/cmd/flip@latest`
- Requires public GitHub repo
- No additional setup needed

### 4. Documentation for Users

Update README.md:
- Installation instructions
- Usage examples
- Configuration guide
- Troubleshooting

## Current Status (October 2025)

### ✅ Completed Features
- Interactive menu system (status, new, edit/manage)
- Workspace management (create, switch, rename, remove)
- Brain management (add, rename, set default, remove)
- Multi-workspace support
- Brain detection (Obsidian, Logseq, Dendron, Markdown)
- Configuration persistence
- Directory initialization
- Quickstart wizard

### 🚧 In Development
- Task tracking across workspaces
- Note summarization
- Natural language querying
- LLM integration

### 📋 TODO
- [ ] Version command
- [ ] Config migration system
- [ ] Backup/restore functionality
- [ ] Export/import workspaces
- [ ] Plugin system
- [ ] Web UI (optional)

## Contributing (Future)

When ready to accept contributions:

1. Fork repository
2. Create feature branch
3. Make changes
4. Run tests: `make test && make lint`
5. Submit pull request

### Code Style
- Follow Go conventions
- Use `gofmt` for formatting
- Add tests for new features
- Update documentation

## Questions?

Current maintainer: dh (@danorama-dh)
