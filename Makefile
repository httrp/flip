# Makefile for flip CLI and VS Code Extension
# Works on macOS, Linux, and Windows (with Make installed)

# Detect OS and set binary name
ifeq ($(OS),Windows_NT)
    BINARY=flip.exe
else
    BINARY=flip
endif

GOPATH=$(shell go env GOPATH)
INSTALL_PATH=$(GOPATH)/bin
VSIX_DIR=vscode-extension
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
ALLOW_1_0 ?= 0
LDFLAGS=-X github.com/httrp/flip/internal/commands.FlipVersion=$(VERSION)
# Extract version dynamically from package.json
EXTENSION_VERSION=$(shell grep '"version"' $(VSIX_DIR)/package.json | head -1 | sed 's/.*"version": "\([^"]*\)".*/\1/')
VSIX_FILE=$(VSIX_DIR)/flip-vscode-$(EXTENSION_VERSION).vsix
VSIX_GLOB=$(VSIX_DIR)/flip-vscode-*.vsix

.PHONY: all ci build build-cli build-extension clean clean-cli clean-extension test lint smoke public-check install uninstall dev-link package-extension install-extension uninstall-extension help setup-hooks bump-extension-patch bump-extension-minor bump-extension-major bump-extension bump-extension-install extension-deps

# Default: show help
help:
	@echo "Flip - Build Targets:"
	@echo ""
	@echo "  make setup               Full setup: build + install CLI + install extension"
	@echo "  make all                 Full check: build + lint + test + smoke"
	@echo "  make ci                  Run exactly what CI runs"
	@echo ""
	@echo "  make build               Build everything (CLI + Extension)"
	@echo "  make build VERSION=0.4.0 Build with explicit CLI version"
	@echo "  make build-cli           Build flip CLI only"
	@echo "  make build-extension     Build VS Code Extension only"
	@echo "  make package-extension   Package Extension as .vsix"
	@echo "  make install-extension   Install VS Code Extension (detects code/codium)"
	@echo "  make uninstall-extension Uninstall VS Code Extension"
	@echo "  make install             Install flip CLI to GOPATH/bin"
	@echo "  make dev-link            Create /usr/local/bin symlink (macOS/Linux)"
	@echo "  make setup-hooks         Install git pre-commit hooks"
	@echo "  make test                Run Go tests"
	@echo "  make lint                Run Go linter (vet)"
	@echo "  make public-check        Validate public repo hygiene"
	@echo "  make smoke               Run smoke tests"
	@echo "  make clean               Clean all build artifacts"
	@echo "  make clean-cli           Clean CLI binary only"
	@echo "  make clean-extension     Clean Extension build files"
	@echo "  make uninstall           Remove installed flip"
	@echo "  make bump-extension      Bump extension version (patch)"
	@echo "  make bump-extension-patch Bump extension version (patch)"
	@echo "  make bump-extension-minor Bump extension version (minor)"
	@echo "  make bump-extension-major Bump extension version (major, requires ALLOW_1_0=1)"
	@echo "  make bump-extension-install Bump (patch) + build + install extension"

# Full setup: build + install CLI + install extension (for new machines)
setup: build install install-extension
	@echo ""
	@echo "✓ Full setup complete!"
	@echo "  • flip CLI installed to $(INSTALL_PATH)"
	@echo "  • VS Code Extension installed"
	@echo ""
	@echo "Next: Reload VS Code window and run 'flip quickstart'"

# Full check before commit (recommended before pushing)
all: public-check build lint test smoke
	@echo ""
	@echo "✓ All checks passed! Safe to commit."

# Run exactly what CI runs
ci: public-check build lint test smoke
	@echo ""
	@echo "✓ CI simulation complete"

# Build everything
build: sync-version build-cli build-extension
	@echo "✓ Build complete: $(BINARY) + VS Code Extension"

# Sync extension version between package.json and Go
sync-version:
	@bash scripts/sync-extension-version.sh

# Bump extension version (package.json only), then sync Go const
bump-extension: bump-extension-patch

bump-extension-patch:
	@cd $(VSIX_DIR) && npm version patch --no-git-tag-version
	@$(MAKE) sync-version
	@echo "✓ Extension version bumped (patch)"

bump-extension-minor:
	@cd $(VSIX_DIR) && npm version minor --no-git-tag-version
	@$(MAKE) sync-version
	@echo "✓ Extension version bumped (minor)"

bump-extension-major:
	@if [ "$(ALLOW_1_0)" != "1" ]; then \
		echo "❌ Major bump blocked (pre-1.0 mode)."; \
		echo "   Use patch/minor, or run: make bump-extension-major ALLOW_1_0=1"; \
		exit 1; \
	fi
	@cd $(VSIX_DIR) && npm version major --no-git-tag-version
	@$(MAKE) sync-version
	@echo "✓ Extension version bumped (major)"

bump-extension-install: bump-extension-patch build install-extension

# Build CLI only
build-cli:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/flip
	@echo "✓ CLI built: $(BINARY)"

# Ensure extension dependencies are installed
extension-deps:
	@cd $(VSIX_DIR) && if [ ! -d node_modules ]; then \
		echo "Installing extension dependencies..."; \
		if ! npm ci; then \
			echo "npm ci failed, retrying once..."; \
			npm ci || (echo "❌ Could not install extension dependencies (network/registry issue)." && exit 1); \
		fi; \
	fi

# Build Extension (compile TypeScript)
build-extension: extension-deps
	@cd $(VSIX_DIR) && npm run compile
	@echo "✓ Extension built"

# Package Extension as .vsix file
package-extension: build-extension
	@echo "Building VSIX package (version $(EXTENSION_VERSION))..."
	@rm -f "$(VSIX_FILE)"
	@cd $(VSIX_DIR) && npx @vscode/vsce package --skip-license --out flip-vscode-$(EXTENSION_VERSION).vsix
	@if [ -f "$(VSIX_FILE)" ]; then \
		echo "✓ Extension packaged: $(VSIX_FILE)"; \
	else \
		echo "❌ Could not create VSIX: $(VSIX_FILE)"; \
		exit 1; \
	fi

# Install VS Code Extension (works with VS Code or VSCodium)
# Installs to both default profile AND "da" profile if it exists
install-extension: package-extension
	@CLI=$$(command -v code || true); \
	if [ -z "$$CLI" ]; then \
		CLI=$$(command -v codium || true); \
	fi; \
	if [ -z "$$CLI" ]; then \
		CLI=$$(command -v code-insiders || true); \
	fi; \
	if [ -z "$$CLI" ]; then \
		echo "❌ VS Code CLI not found. Install VS Code and ensure 'code' is in PATH."; \
		exit 1; \
	fi; \
	VSIX="$(VSIX_FILE)"; \
	if [ ! -f "$$VSIX" ]; then \
		echo "❌ VSIX file not found: $$VSIX"; \
		exit 1; \
	fi; \
	echo "Installing $$VSIX..."; \
	"$$CLI" --install-extension "$$VSIX" --force && echo "✓ Extension installed (default profile)"; \
	"$$CLI" --install-extension "$$VSIX" --force --profile "da" 2>/dev/null && echo "✓ Extension installed (da profile)" || true

# Uninstall the extension by identifier
uninstall-extension:
	@ID=danorama.flip-vscode; \
	CLI=$$(command -v code || true); \
	if [ -z "$$CLI" ]; then \
		CLI=$$(command -v codium || true); \
	fi; \
	if [ -z "$$CLI" ]; then \
		CLI=$$(command -v code-insiders || true); \
	fi; \
	if [ -z "$$CLI" ]; then \
		echo "❌ VS Code CLI not found. Please ensure 'code' or 'codium' is in PATH."; \
		exit 1; \
	fi; \
	echo "Uninstalling extension: $$ID via $$CLI"; \
	"$$CLI" --uninstall-extension "$$ID" && echo "✓ Extension uninstalled"

# Install to GOPATH/bin (requires GOPATH/bin in PATH)
install: build-cli
	go install -ldflags "$(LDFLAGS)" ./cmd/flip
	@echo "✓ flip installed to $(INSTALL_PATH)"

# Create symlink in /usr/local/bin (requires sudo, macOS/Linux only)
dev-link: build-cli
	@echo "Creating symlink in /usr/local/bin (may require password)..."
	sudo ln -sf $(PWD)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "✓ flip linked to /usr/local/bin/flip"
	@echo "  You can now run 'flip' from anywhere"
	@echo "  Changes will be active after 'make build-cli'"

test:
	go test ./...

lint:
	go vet ./...

smoke:
	bash scripts/smoke.sh

public-check:
	bash scripts/check-public-repo.sh

# Clean everything
clean: clean-cli clean-extension
	@echo "✓ All build artifacts cleaned"

# Clean CLI binary only
clean-cli:
	rm -f $(BINARY)

# Clean Extension build files
clean-extension:
	cd $(VSIX_DIR) && rm -rf out/ && rm -f *.vsix
	@echo "✓ Extension artifacts cleaned"

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY)
	sudo rm -f /usr/local/bin/$(BINARY)
	@echo "✓ flip uninstalled"

# Setup git hooks
setup-hooks:
	@echo "Installing git hooks..."
	@echo '#!/bin/bash' > .git/hooks/pre-commit
	@echo 'REPO_ROOT="$$(git rev-parse --show-toplevel)"' >> .git/hooks/pre-commit
	@echo 'if [ -x "$$REPO_ROOT/scripts/check-extension-version.sh" ]; then' >> .git/hooks/pre-commit
	@echo '    "$$REPO_ROOT/scripts/check-extension-version.sh"' >> .git/hooks/pre-commit
	@echo 'fi' >> .git/hooks/pre-commit
	@echo 'if [ -x "$$REPO_ROOT/scripts/check-public-repo.sh" ]; then' >> .git/hooks/pre-commit
	@echo '    "$$REPO_ROOT/scripts/check-public-repo.sh"' >> .git/hooks/pre-commit
	@echo 'fi' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✓ Git hooks installed"
