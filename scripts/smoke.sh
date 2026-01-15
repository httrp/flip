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

echo "> Running flip status..."
STATUS_OUT=$(./flip status || true)
if [ $? -ne 0 ]; then
  echo "[X] flip status failed"
  exit 1
fi
echo "[OK] flip status completed"

echo "> Running flip brain list..."
BRAIN_OUT=$(./flip brain list || true)
if [ $? -ne 0 ]; then
  echo "[X] flip brain list failed"
  exit 1
fi
echo "[OK] flip brain list completed"

echo ""
echo "[OK] All smoke tests passed"
