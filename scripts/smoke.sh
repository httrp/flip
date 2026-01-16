#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

# Use existing binary if available, otherwise build
if [ ! -f ./flip ]; then
  echo "> Building flip binary..."
  go build -o flip ./cmd/flip
else
  echo "> Using existing flip binary"
fi

echo "> Testing flip vscode info..."
if ! timeout 5 ./flip vscode info --json > /dev/null 2>&1; then
  echo "[X] flip vscode info failed"
  exit 1
fi
echo "[OK] flip vscode info completed"

echo "> Testing flip task list..."
if ! timeout 10 ./flip task list --json > /dev/null 2>&1; then
  echo "[X] flip task list failed"
  exit 1
fi
echo "[OK] flip task list completed"

echo "> Testing flip task browse (JSON)..."
if ! timeout 10 ./flip task browse --json > /dev/null 2>&1; then
  echo "[X] flip task browse --json failed"
  exit 1
fi
echo "[OK] flip task browse --json completed"

echo ""
echo "[OK] All smoke tests passed"
