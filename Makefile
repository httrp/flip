# Makefile for flip CLI and VS Code Extension
# Works on macOS, Linux, and Windows (with Make installed)

BINARY=flip
GOPATH=$(shell go env GOPATH)
INSTALL_PATH=$(GOPATH)/bin
VSIX_DIR=vscode-extension
VSIX_FILE=$(VSIX_DIR)/flip-vscode-*.vsix

.PHONY: build build-cli build-extension clean clean-cli clean-extension test lint smoke install uninstall dev-link package-extension install-extension uninstall-extension help

# Default: show help
help:
	@echo "Flip - Build Targets:"
	@echo ""
	@echo "  make build               Build everything (CLI + Extension)"
	@echo "  make build-cli           Build flip CLI only"
	@echo "  make build-extension     Build VS Code Extension only"
	@echo "  make package-extension   Package Extension as .vsix"
	@echo "  make install-extension   Install VS Code Extension (detects code/codium)"
	@echo "  make uninstall-extension Uninstall VS Code Extension"
	@echo "  make install             Install flip CLI to GOPATH/bin"
	@echo "  make dev-link            Create /usr/local/bin symlink (macOS/Linux)"
	@echo "  make test                Run Go tests"
	@echo "  make lint                Run Go linter (vet)"
	@echo "  make smoke               Run smoke tests"
	@echo "  make clean               Clean all build artifacts"
	@echo "  make clean-cli           Clean CLI binary only"
	@echo "  make clean-extension     Clean Extension build files"
	@echo "  make uninstall           Remove installed flip"

# Build everything
build: build-cli build-extension
	@echo "✓ Build complete: $(BINARY) + VS Code Extension"

# Build CLI only
build-cli:
	go build -o $(BINARY) ./cmd/flip
	@echo "✓ CLI built: $(BINARY)"

# Build Extension (compile TypeScript)
build-extension:
	cd $(VSIX_DIR) && npm run compile
	@echo "✓ Extension built"

# Package Extension as .vsix file
package-extension: build-extension
	cd $(VSIX_DIR) && yes | npm run package
	@echo "✓ Extension packaged as .vsix"

# Install VS Code Extension (works with VS Code or VSCodium)
install-extension: package-extension
	@echo "Detecting VS Code CLI (code/codium/code-insiders)..."
	@CLI=$$(command -v code || true); \
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
	VSIX=$$(ls -t $(VSIX_DIR)/flip-vscode-*.vsix 2>/dev/null | head -n 1); \
	if [ -z "$$VSIX" ]; then \
		echo "❌ No VSIX found. Run 'make package-extension' first."; \
		exit 1; \
	fi; \
	echo "Installing extension: $$VSIX via $$CLI"; \
	"$$CLI" --install-extension "$$VSIX" --force && echo "✓ Extension installed"

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
	go install ./cmd/flip
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
