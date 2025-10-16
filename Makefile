# Simple Makefile for flip CLI

BINARY=flip

.PHONY: build test lint smoke clean

build:
	go build -o $(BINARY) ./cmd/flip

test:
	go test ./...

lint:
	go vet ./...

smoke:
	bash scripts/smoke.sh

clean:
	rm -f $(BINARY)
