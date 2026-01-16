#!/bin/bash
# Bump extension version and keep sync with Go code
# Usage: scripts/bump-version.sh [major|minor|patch]

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

# Get current version
CURRENT_VERSION=$(jq -r '.version' "$PACKAGE_JSON")
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Determine new version
BUMP_TYPE="${1:-patch}"
case "$BUMP_TYPE" in
    major)
        NEW_VERSION="$((MAJOR + 1)).0.0"
        ;;
    minor)
        NEW_VERSION="$MAJOR.$((MINOR + 1)).0"
        ;;
    patch)
        NEW_VERSION="$MAJOR.$MINOR.$((PATCH + 1))"
        ;;
    *)
        echo "❌ Invalid bump type: $BUMP_TYPE (use: major, minor, patch)"
        exit 1
        ;;
esac

echo "📦 Bumping version: $CURRENT_VERSION → $NEW_VERSION"

# Update package.json
jq ".version = \"$NEW_VERSION\"" "$PACKAGE_JSON" > "$PACKAGE_JSON.tmp"
mv "$PACKAGE_JSON.tmp" "$PACKAGE_JSON"
echo "✓ Updated package.json"

# Update vscode_extension.go
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/const ExtensionVersion = \"[^\"]*\"/const ExtensionVersion = \"$NEW_VERSION\"/" "$VSCODE_GO"
else
    sed -i "s/const ExtensionVersion = \"[^\"]*\"/const ExtensionVersion = \"$NEW_VERSION\"/" "$VSCODE_GO"
fi
echo "✓ Updated vscode_extension.go"

echo ""
echo "✅ Version bumped to $NEW_VERSION"
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff"
echo "  2. Test: make all"
echo "  3. Commit: git add -A && git commit -m \"chore: bump version to $NEW_VERSION\""
echo "  4. Tag: git tag v$NEW_VERSION"
echo "  5. Push: git push && git push --tags"
