#!/bin/bash
# Pre-commit check: Ensure extension version is bumped when extension files change
# This prevents accidentally committing extension changes without updating the version

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PACKAGE_JSON="$REPO_ROOT/vscode-extension/package.json"
VSCODE_GO="$REPO_ROOT/internal/commands/vscode_extension.go"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Not a git repository, skipping check"
    exit 0
fi

# Get staged extension source files
STAGED_EXTENSION_FILES=$(git diff --cached --name-only -- 'vscode-extension/src/**' 'vscode-extension/*.ts' 2>/dev/null || true)

# Any staged extension-related files that require version consistency checks
STAGED_RELATED_FILES=$(git diff --cached --name-only -- \
    'vscode-extension/src/**' \
    'vscode-extension/*.ts' \
    'vscode-extension/package.json' \
    'internal/commands/vscode_extension.go' 2>/dev/null || true)

is_staged() {
    local path="$1"
    git diff --cached --name-only -- "$path" 2>/dev/null | grep -q .
}

get_package_version() {
    if is_staged 'vscode-extension/package.json'; then
        git show :vscode-extension/package.json | jq -r '.version' 2>/dev/null
    else
        jq -r '.version' "$PACKAGE_JSON" 2>/dev/null
    fi
}

get_go_version() {
    if is_staged 'internal/commands/vscode_extension.go'; then
        git show :internal/commands/vscode_extension.go | grep 'const ExtensionVersion = ' | sed 's/.*"\([^"]*\)".*/\1/'
    else
        grep 'const ExtensionVersion = ' "$VSCODE_GO" | sed 's/.*"\([^"]*\)".*/\1/'
    fi
}

if [ -z "$STAGED_RELATED_FILES" ]; then
    # No extension-related files staged, nothing to check
    exit 0
fi

# Check if package.json is also staged (meaning version might have been updated)
PACKAGE_JSON_STAGED=$(git diff --cached --name-only -- 'vscode-extension/package.json' 2>/dev/null || true)

if [ -z "$PACKAGE_JSON_STAGED" ]; then
    echo ""
    echo "⚠️  WARNING: Extension source files were changed but package.json was not modified!"
    echo ""
    echo "   Changed files:"
    echo "$STAGED_EXTENSION_FILES" | sed 's/^/     - /'
    echo ""
    echo "   Please bump the version in vscode-extension/package.json"
    echo "   Current version: $(jq -r '.version' "$PACKAGE_JSON" 2>/dev/null)"
    echo ""
    echo "   To skip this check (not recommended): git commit --no-verify"
    echo ""
    exit 1
fi

# package.json is staged, check if version actually changed
VERSION_CHANGED=$(git diff --cached -- 'vscode-extension/package.json' | grep -E '^\+.*"version"' || true)

if [ -z "$VERSION_CHANGED" ]; then
    echo ""
    echo "⚠️  WARNING: Extension source files were changed but version in package.json is unchanged!"
    echo ""
    echo "   Changed files:"
    echo "$STAGED_EXTENSION_FILES" | sed 's/^/     - /'
    echo ""
    echo "   Please bump the version in vscode-extension/package.json"
    echo "   Current version: $(jq -r '.version' "$PACKAGE_JSON" 2>/dev/null)"
    echo ""
    echo "   To skip this check (not recommended): git commit --no-verify"
    echo ""
    exit 1
fi

# Keep package.json and Go constant synchronized whenever extension-related files are staged
PACKAGE_VERSION=$(get_package_version)
GO_VERSION=$(get_go_version)

if [ -z "$PACKAGE_VERSION" ] || [ -z "$GO_VERSION" ]; then
    echo ""
    echo "❌ ERROR: Could not read extension versions from package.json or vscode_extension.go"
    echo ""
    exit 1
fi

if [ "$PACKAGE_VERSION" != "$GO_VERSION" ]; then
    echo ""
    echo "❌ ERROR: Extension versions are out of sync"
    echo ""
    echo "   package.json:        $PACKAGE_VERSION"
    echo "   vscode_extension.go: $GO_VERSION"
    echo ""
    echo "   Run: scripts/sync-extension-version.sh"
    echo ""
    exit 1
fi

echo "✓ Extension version check passed"
exit 0
