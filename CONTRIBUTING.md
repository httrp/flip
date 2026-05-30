# Contributing to flip

Thanks for your interest in flip. This document covers development setup, the build/test workflow and the project conventions.

If something here is wrong or missing, a PR or issue is welcome.

---

## Prerequisites

- Go 1.21+
- Node.js 18+ (only needed if you build the VS Code extension)
- Git
- Make (recommended)

## Quick start

```bash
git clone https://github.com/httrp/flip.git
cd flip
make setup        # builds CLI + extension and installs both
```

After `make setup`, reload your VS Code window and run `flip quickstart`.

### Manual steps

```bash
make build        # build CLI + extension
make install      # install CLI into $GOPATH/bin
make install-extension   # build and install the VS Code extension
```

Make sure `$GOPATH/bin` (typically `~/go/bin`) is on your `PATH`:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Verify:

```bash
which flip
flip status
```

### Platform notes

- **macOS / Linux** — `make dev-link` creates a `/usr/local/bin/flip` symlink so the binary is picked up after every `make build-cli` without re-running `make install`. Requires `sudo`.
- **Windows (Git Bash)** — use Git Bash, not CMD or PowerShell. Add `export PATH="$HOME/go/bin:$PATH"` to `~/.bashrc`. `make dev-link` is not supported.

If `flip` is not found from VS Code's terminal, it usually means the terminal does not see `$GOPATH/bin`. Reload the VS Code window after fixing your shell config.

---

## Project layout

```
cmd/flip/                  # CLI entry point
internal/
  commands/                # cobra commands (one file per command where practical)
  brain/                   # brain detection and creation
  tasks/                   # task parsing and scanning
  exercises/               # exercise tracking
  health/                  # brain validation and repair
  ai/                      # AI provider integrations
  migration/               # brain format migration
  templates/               # embedded templates
  platform/                # cross-platform helpers
  ui/                      # interactive prompts
  lang/                    # i18n strings
vscode-extension/          # VS Code integration (TypeScript)
docs/                      # user and design documentation
scripts/                   # build, hooks and CI helpers
test-brains/               # test fixtures
```

---

## Development workflow

### Branches

- `feat/<short-name>` — features
- `fix/<short-name>` — bug fixes
- `refactor/<area>` — refactors
- `docs/<area>` — documentation only

### Commits

Conventional commits are encouraged but not strictly enforced:

```
feat(tasks): add priority filtering
fix(journal): handle timezone correctly
docs: update command reference
```

### Build, test, lint

```bash
make build         # build CLI + extension
make build-cli     # CLI only
make build-extension

make test          # go test ./...
make lint          # go vet ./...
make smoke         # smoke tests (scripts/smoke.sh)

make all           # public-check + build + lint + test + smoke
make ci            # what CI runs
```

After changing the CLI, the new binary is available immediately if `~/go/bin` is on your PATH (and you ran `make install`). After changing the extension, reload the VS Code window.

---

## Coding conventions

### Errors

Return wrapped errors with context. Avoid `os.Exit` outside of `main`.

```go
if err != nil {
    return fmt.Errorf("loading workspace config: %w", err)
}
```

In interactive menus, printing an error and continuing is fine.

### Constants over magic strings

Prefer named constants in `internal/commands/constants.go` over repeated literal strings (`"default"`, directory names, file extensions, …).

### File size

Aim to keep files under ~500 LOC. Split when they grow beyond that — for example, `menu_brain.go` was split into `menu_brain.go`, `menu_brain_edit.go` and `menu_brain_actions.go`.

### Output

Prefer localized strings:

```go
fmt.Println(lang.GetText("task.created"))
```

For JSON output, use the existing `outputJSON` helper.

### Tests

- Place tests in `*_test.go` next to the code they cover.
- Use table-driven tests where it helps.
- Don't depend on the user's real `~/.config/flip` — use temporary directories.

---

## Adding things

### A new command

1. Create `internal/commands/<command>.go` exposing a `New<Command>Command() *cobra.Command`.
2. Register it in `cmd/flip/main.go`.
3. Add or update its entry in [docs/COMMANDS.md](docs/COMMANDS.md).
4. Add tests where it's reasonable (parsing, business logic).

### Localization

1. Add a key to `internal/lang/en.json`.
2. Use `lang.GetText("my.key")` in code.

### A new VS Code task

1. Edit `VSCodeTasksJSON` in `internal/commands/vscode.go`.
2. Bump the `TasksVersion` constant.
3. Update [docs/VSCODE.md](docs/VSCODE.md) if the user-visible behaviour changes.

---

## Public-repo hygiene

This repo is mirrored to a public profile. Internal working notes must stay out of tracked files.

Do not commit:

- ad-hoc review docs (e.g. `CODE_REVIEW_*.md`)
- AI handoff or planning notes (`docs/AI_*`, `docs/*_PLAN.md`)
- private or scratch material under `docs/archive/`

Allowed:

- user-facing docs (`README.md`, `docs/QUICKSTART.md`, `docs/COMMANDS.md`, …)
- test fixtures under `test-brains/`
- example content under `docs/example-brain/`

Enforced by `make public-check` locally and the **Public Repo Guard** step in CI.

---

## Release process

1. Update affected version constants and `vscode-extension/package.json` (or use `make bump-extension-*`).
2. Update [CHANGELOG.md](CHANGELOG.md).
3. Tag the release.
4. GitHub Actions builds release artifacts.

---

## Getting help

- Bugs and feature requests: [GitHub Issues](https://github.com/httrp/flip/issues)
- Questions: [GitHub Discussions](https://github.com/httrp/flip/discussions)

---

## License

By contributing you agree that your contributions are licensed under the MIT License (see [LICENSE](LICENSE)).
