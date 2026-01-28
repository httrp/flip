# Flip - Development Setup Guide

This guide covers how to set up **flip** for development and daily use across **macOS**, **Linux**, and **Windows**.

## Prerequisites

You need:
- **Go** 1.21+ ([Download](https://golang.org/dl))
- **Node.js** 18+ (for VS Code Extension building)
- **Git**

## Installation

### Quick Setup (New Machine)

```bash
git clone https://github.com/httrp/flip.git
cd flip
make setup
```

This does everything: builds CLI + Extension, installs both. After that:
1. Reload VS Code window (`Ctrl+Shift+P` → "Reload Window")
2. Run `flip quickstart`

### Manual Setup

#### 1. Clone & Build

```bash
git clone https://github.com/httrp/flip.git
cd flip
make build
```

This creates:
- `flip` (or `flip.exe` on Windows) - CLI binary
- VS Code Extension (`.vsix` file)

### 2. Install flip CLI

The standard way - puts `flip` in your `$GOPATH/bin`:

```bash
# Install to $GOPATH/bin (usually ~/go/bin)
make install

# Or manually:
go install ./cmd/flip
```

**Verify installation:**
```bash
which flip
flip status
```

### 3. Add to Your Shell

For `flip` to be available in every new terminal, add `$GOPATH/bin` to your PATH.

#### macOS & Linux

Edit your shell config (`~/.bashrc`, `~/.zshrc`, or `~/.bash_profile`):

```bash
# Add this line:
export PATH="$HOME/go/bin:$PATH"

# Then reload:
source ~/.bashrc  # or ~/.zshrc
```

#### Windows (Git Bash)

Edit `~/.bashrc`:

```bash
# Add this line:
export PATH="$HOME/go/bin:$PATH"
```

Then restart your Git Bash terminal.

### 4. Install VS Code Extension

```bash
# Build and install extension
make install-extension
```

Or manually:
```bash
# Package extension
make package-extension

# Install to VS Code
code --install-extension vscode-extension/flip-vscode-*.vsix --force
```

**Verify in VS Code:**
- Open VS Code
- Run: `Ctrl+Shift+P` → "Reload Window"
- Run: `Ctrl+Shift+P` → "Flip: Menu" (should work without errors)

### 5. Initialize Your First Brain

```bash
# Start interactive setup
flip quickstart

# Or manually:
flip brain new ~/my-first-brain
flip workspace create default
```

## Development Workflow

### Making Changes

```bash
# Edit Go code in ./internal or ./cmd
# Edit TypeScript in ./vscode-extension/src

# Build CLI only
make build-cli

# Build Extension only
make build-extension

# Full build
make build
```

**After building CLI:** The new binary is immediately available via `flip` command (from `~/go/bin`).

**After building Extension:** Reload VS Code window (`Ctrl+Shift+P` → "Reload Window").

### Testing

```bash
# Run tests
make test

# Run linter
make lint

# Run smoke tests
make smoke

# Full CI check (before pushing)
make all
```

## Troubleshooting

### "flip: command not found"

**Problem:** `flip` is not in your PATH.

**Solution:**
```bash
# Check if installation succeeded
ls -la ~/go/bin/flip

# Check PATH
echo $PATH | grep go/bin

# If missing, add to ~/.bashrc and reload
export PATH="$HOME/go/bin:$PATH"
source ~/.bashrc
```

### VS Code Extension: "Failed to get brain info"

**Problem:** Extension can't find the `flip` CLI.

**Solution:**
1. Make sure `flip` is installed: `which flip` returns a path
2. Reload VS Code: `Ctrl+Shift+P` → "Reload Window"
3. Check VS Code terminal: `Ctrl+`` → terminal should have `flip` in PATH

### Windows-Specific Issues

**Make command not found:**
```bash
# Install make via Scoop
scoop install make
```

**Go not found:**
```bash
# Install Go via Scoop
scoop install go
```

## Platform-Specific Notes

### macOS

- Use Homebrew for dependencies: `brew install go node`
- You can optionally create a symlink: `make dev-link` (requires sudo)

### Linux (Fedora, Ubuntu, etc.)

- Install Go: `sudo dnf install golang` (Fedora) or `sudo apt install golang-go` (Ubuntu)
- Install Node: Use nvm or your distro's package manager
- Shell config: Add PATH to `~/.bashrc`

### Windows (Git Bash)

- Install via scoop recommended: `scoop install go nodejs`
- Make sure to use **Git Bash**, not CMD or PowerShell
- Shell config: Edit `~/.bashrc` (Git Bash config)
- VS Code should detect flip from Git Bash PATH automatically

## Next Steps

1. **Read the docs:** [COMMANDS.md](docs/COMMANDS.md)
2. **Try the quickstart:** `flip quickstart`
3. **Set up your workspace:** `flip workspace create`
4. **Create first brain:** `flip brain new`
5. **Try the menu:** `flip menu`

## Contributing

When making changes:
- Test on at least 2 platforms (macOS, Linux, or Windows)
- Run `make all` before committing
- Don't add platform-specific workarounds to the repo (document them in this guide instead)
- Update language files in `internal/lang/` for user-facing text

## Support

- Issues: [GitHub Issues](https://github.com/httrp/flip/issues)
- Discussions: [GitHub Discussions](https://github.com/httrp/flip/discussions)
