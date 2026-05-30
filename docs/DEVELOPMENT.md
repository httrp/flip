# Development Guide

This is the architecture / internals companion to [CONTRIBUTING.md](../CONTRIBUTING.md). Read CONTRIBUTING first for setup, build, test and conventions.

## Layout

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
  protocols/               # meeting protocols
  templates/               # embedded templates
  platform/                # cross-platform helpers
  ui/                      # interactive prompts
  schema/                  # schema validation
  lang/                    # i18n strings
vscode-extension/          # VS Code integration (TypeScript)
```

## Key concepts

- **Workspace config** lives in `~/.config/flip/workspaces.yaml` (Linux/macOS) or the platform-equivalent path. It tracks workspaces, the active workspace and the default brain per workspace. There is no cloud sync.
- **Brain detection** uses marker files: `.flip-brain.yaml`, `.obsidian/`, `.logseq/`, `dendron.yml`, `.foam/`. Detection lives in `internal/brain/detector.go`.
- **Health checks** validate path integrity, broken links, sync-conflict files (iCloud/Dropbox/OneDrive) and structural issues.
- **Migration** converts between brain formats, including link-style transforms and Logseq artifact cleanup.

## Local development loop

```bash
make build-cli         # rebuild CLI
make test              # go test ./...
make lint              # go vet ./...
```

If you ran `make dev-link` once, the binary at `/usr/local/bin/flip` follows your latest local build automatically (macOS/Linux only).

## Versioning

- CLI version is injected at build time via `-ldflags` from `VERSION` (defaults to `git describe --tags --always --dirty`).
- The VS Code extension version lives in `vscode-extension/package.json` and is mirrored into `internal/commands/vscode_extension.go` by `scripts/sync-extension-version.sh`.
- `scripts/check-extension-version.sh` (also wired into the optional pre-commit hook) catches drift between the two.
- `scripts/check-public-repo.sh` blocks accidental commits of internal working notes (see [CONTRIBUTING.md](../CONTRIBUTING.md#public-repo-hygiene)).

## Release flow

1. Run `make all` to build, lint, test and smoke locally.
2. Update [CHANGELOG.md](../CHANGELOG.md) under the new version.
3. Tag the release (`git tag v0.x.y && git push --tags`).
4. CI publishes artifacts.

For extension-only releases, `make bump-extension-patch` (or `-minor`) keeps `package.json` and the Go constant in sync.
