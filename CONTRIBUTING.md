# Contributing to flip

Thank you for considering contributing to flip! This document provides guidelines and information for contributors.

## Code of Conduct

Be respectful, constructive, and collaborative. We're all here to make flip better.

---

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional but recommended)

### Setup

```bash
# Clone the repository
git clone https://github.com/httrp/flip.git
cd flip

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Create dev symlink (optional)
make dev-link
```

### Project Structure

```
flip/
├── cmd/flip/main.go        # Entry point
├── internal/
│   ├── commands/           # CLI commands (cobra)
│   │   ├── menu.go         # Interactive menu
│   │   ├── task*.go        # Task commands
│   │   ├── note.go         # Note commands
│   │   └── ...
│   ├── tasks/              # Task parsing/scanning
│   ├── exercises/          # Exercise system
│   ├── brain/              # Brain detection
│   ├── health/             # Health checking
│   ├── migration/          # Brain format migration
│   ├── schema/             # Schema validation
│   ├── lang/               # i18n strings
│   └── templates/          # Embedded templates
├── docs/                   # Documentation
├── scripts/                # Build/test scripts
└── vscode-extension/       # VS Code extension
```

---

## Development Workflow

### 1. Create a Branch

```bash
# Feature
git checkout -b feat/my-feature

# Bug fix
git checkout -b fix/issue-description

# Refactor
git checkout -b refactor/area-name
```

### 2. Make Changes

Follow the coding standards below.

### 3. Test

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/tasks/...

# Run with verbose output
go test -v ./...
```

### 4. Commit

Use conventional commits:

```bash
# Features
git commit -m "feat(tasks): add priority filtering"

# Fixes
git commit -m "fix(journal): handle timezone correctly"

# Refactoring
git commit -m "refactor(menu): extract helper functions"

# Documentation
git commit -m "docs: update command reference"
```

### 5. Push & PR

```bash
git push origin feat/my-feature
# Then create PR on GitHub
```

---

## Coding Standards

### Go Code

```go
// Use meaningful names
func scanTasksInBrain(brainPath string) ([]Task, error) { ... }

// Return errors, don't panic
if err != nil {
    return nil, fmt.Errorf("failed to scan tasks: %w", err)
}

// Document exported functions
// ScanTasks scans all markdown files in the brain for tasks.
// It returns tasks sorted by priority (high first).
func ScanTasks(brainPath string) ([]Task, error) { ... }
```

### Error Handling

```go
// ✅ Good: Return errors
func doSomething() error {
    if err != nil {
        return fmt.Errorf("context: %w", err)
    }
    return nil
}

// ❌ Bad: Exit directly
func doSomething() {
    if err != nil {
        fmt.Println("Error:", err)
        os.Exit(1)  // Don't do this in library code
    }
}
```

### Output

```go
// Use localized strings where possible
fmt.Println(lang.GetText("task.created"))

// For user-facing output, consider icons
fmt.Println("✅ Task created")

// For JSON output, use the json_output helper
outputJSON(result)
```

### File Organization

- One command per file (e.g., `task_new.go`, `task_list.go`)
- Keep files under 500 lines when practical
- Extract shared code to helper files

---

## Testing

### Unit Tests

```go
func TestParseTask(t *testing.T) {
    input := "- [ ] My task [priority: high]"
    task, err := ParseTask(input)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if task.Priority != PriorityHigh {
        t.Errorf("expected high priority, got %v", task.Priority)
    }
}
```

### Test Files

- Place tests in `*_test.go` files
- Use table-driven tests for multiple cases
- Mock external dependencies (file system, etc.)

---

## Documentation

### Code Comments

```go
// Package tasks provides task parsing, scanning, and management.
package tasks

// Task represents a todo item with optional metadata.
type Task struct {
    Description string
    Priority    Priority
    Due         *time.Time
}
```

### User Documentation

- Update `docs/COMMANDS.md` for new commands
- Update `docs/FEATURES.md` for new features
- Keep README.md concise (link to docs for details)

---

## Common Tasks

### Adding a New Command

1. Create `internal/commands/mycommand.go`
2. Implement `NewMyCommand() *cobra.Command`
3. Register in `cmd/flip/main.go`
4. Add to `docs/COMMANDS.md`
5. Add tests

### Adding Localization

1. Add key to `internal/lang/en.json`
2. Use `lang.GetText("my.key")` in code

### Adding a VS Code Task

1. Edit `VSCodeTasksJSON` in `internal/commands/vscode.go`
2. Bump `TasksVersion` constant
3. Update `docs/VSCODE.md`

---

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create release tag
4. GitHub Actions builds releases

---

## Getting Help

- Open an issue for bugs or feature requests
- Check existing issues before creating new ones
- For questions, use GitHub Discussions

---

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
