#!/bin/bash
# Synchronize extension version between package.json and vscode_extension.go
# This ensures both files always have the same version number

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PACKAGE_JSON="$REPO_ROOT/vscode-extension/package.json"
VSCODE_GO="$REPO_ROOT/internal/commands/vscode_extension.go"

if [ ! -f "$PACKAGE_JSON" ]; then
    echo "❌ Error: $PACKAGE_JSON not found"
    exit 1
fi

if [ ! -f "$VSCODE_GO" ]; then
    echo "❌ Error: $VSCODE_GO not found"
    exit 1
fi

# Extract version from package.json
PACKAGE_VERSION=$(jq -r '.version' "$PACKAGE_JSON" 2>/dev/null)
if [ -z "$PACKAGE_VERSION" ]; then
    echo "❌ Error: Could not extract version from $PACKAGE_JSON"
    exit 1
fi

# Extract current version from vscode_extension.go
GO_VERSION=$(grep 'const ExtensionVersion = ' "$VSCODE_GO" | sed 's/.*"\([^"]*\)".*/\1/')

# Compare and sync if different
if [ "$GO_VERSION" != "$PACKAGE_VERSION" ]; then
    echo "🔄 Syncing extension version: $GO_VERSION → $PACKAGE_VERSION"
    
    # Update vscode_extension.go
    sed -i '' "s/const ExtensionVersion = \"[^\"]*\"/const ExtensionVersion = \"$PACKAGE_VERSION\"/" "$VSCODE_GO"
    
    if [ $? -eq 0 ]; then
        echo "✓ Updated ExtensionVersion in vscode_extension.go to $PACKAGE_VERSION"
    else
        echo "❌ Failed to update ExtensionVersion"
        exit 1
    fi
else
    echo "✓ Extension versions are in sync: $PACKAGE_VERSION"
fi
