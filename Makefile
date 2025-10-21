# Simple Makefile for flip CLI

BINARY=flip
GOPATH=$(shell go env GOPATH)
INSTALL_PATH=$(GOPATH)/bin

.PHONY: build test lint smoke clean install uninstall dev-link

build:
	go build -o $(BINARY) ./cmd/flip

# Install to GOPATH/bin (requires GOPATH/bin in PATH)
install:
	go install ./cmd/flip

# Create symlink in /usr/local/bin (requires sudo)
dev-link: build
	@echo "Creating symlink in /usr/local/bin (may require password)..."
	sudo ln -sf $(PWD)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "✓ flip linked to /usr/local/bin/flip"
	@echo "  You can now run 'flip' from anywhere"
	@echo "  Changes will be active after 'make build'"

test:
	go test ./...

lint:
	go vet ./...

smoke:
	bash scripts/smoke.sh

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY)
	sudo rm -f /usr/local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
