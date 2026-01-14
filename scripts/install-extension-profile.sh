#!/bin/bash
# Install VS Code extension in all profiles
# This script:
# 1. Installs via VS Code CLI (puts files in ~/.vscode/extensions/)
# 2. Removes old extension versions
# 3. Updates extensions.json in each profile directly (fixes pinned/version issues)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Find VS Code CLI
CLI=$(command -v code 2>/dev/null || true)
if [ -z "$CLI" ]; then
    CLI=$(command -v codium 2>/dev/null || true)
fi
if [ -z "$CLI" ]; then
    CLI=$(command -v code-insiders 2>/dev/null || true)
fi

if [ -z "$CLI" ]; then
    echo "❌ VS Code CLI not found (code, codium, or code-insiders)"
    exit 1
fi

# Find latest VSIX file
VSIX=$(ls -t "$REPO_ROOT/vscode-extension/flip-vscode-"*.vsix 2>/dev/null | head -n 1)
if [ -z "$VSIX" ]; then
    echo "❌ No VSIX file found. Run 'make package-extension' first."
    exit 1
fi

VERSION=$(basename "$VSIX" | sed 's/flip-vscode-//;s/\.vsix//')
EXTENSION_ID="danorama.flip-vscode"

echo "→ Installing flip-vscode v$VERSION"

# Determine OS-specific paths
if [[ "$OSTYPE" == "darwin"* ]]; then
    VSCODE_USER_DIR="$HOME/Library/Application Support/Code/User"
    VSCODE_EXT_DIR="$HOME/.vscode/extensions"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    VSCODE_USER_DIR="$HOME/.config/Code/User"
    VSCODE_EXT_DIR="$HOME/.vscode/extensions"
else
    echo "⚠ Unknown OS: $OSTYPE, using default paths"
    VSCODE_USER_DIR="$HOME/.config/Code/User"
    VSCODE_EXT_DIR="$HOME/.vscode/extensions"
fi

PROFILES_DIR="$VSCODE_USER_DIR/profiles"
STORAGE_FILE="$VSCODE_USER_DIR/globalStorage/storage.json"

# Step 1: Install extension via CLI (puts files in ~/.vscode/extensions/)
echo "→ Installing extension via VS Code CLI..."
"$CLI" --install-extension "$VSIX" --force 2>&1 | grep -v "^$" || true

# Step 2: Remove old versions from extensions directory
echo "→ Cleaning up old extension versions..."
for old_dir in "$VSCODE_EXT_DIR"/danorama.flip-vscode-*; do
    if [ -d "$old_dir" ]; then
        old_version=$(basename "$old_dir" | sed 's/danorama.flip-vscode-//')
        if [ "$old_version" != "$VERSION" ]; then
            rm -rf "$old_dir"
            echo "  Removed v$old_version"
        fi
    fi
done

# Step 3: Update extensions.json in each profile
if [ -d "$PROFILES_DIR" ]; then
    echo "→ Updating VS Code profiles..."
    
    # Get profile names from storage.json
    if [ -f "$STORAGE_FILE" ]; then
        PROFILE_INFO=$(jq -r '.userDataProfiles[]? | "\(.location):\(.name)"' "$STORAGE_FILE" 2>/dev/null || true)
    fi
    
    for profile_dir in "$PROFILES_DIR"/*/; do
        profile_id=$(basename "$profile_dir")
        extensions_file="$profile_dir/extensions.json"
        
        if [ ! -f "$extensions_file" ]; then
            continue
        fi
        
        # Get profile name from storage.json
        profile_name=$(echo "$PROFILE_INFO" | grep "^$profile_id:" | cut -d: -f2)
        if [ -z "$profile_name" ]; then
            profile_name="$profile_id"
        fi
        
        # Check if flip is in this profile
        if ! grep -q "\"$EXTENSION_ID\"" "$extensions_file" 2>/dev/null; then
            continue
        fi
        
        echo "  Updating profile: $profile_name"
        
        # Backup and update extensions.json
        cp "$extensions_file" "$extensions_file.bak"
        
        NEW_PATH="$VSCODE_EXT_DIR/danorama.flip-vscode-$VERSION"
        TIMESTAMP=$(date +%s)000
        
        jq --arg version "$VERSION" \
           --arg path "$NEW_PATH" \
           --arg relPath "danorama.flip-vscode-$VERSION" \
           --argjson ts "$TIMESTAMP" \
           '(.[] | select(.identifier.id == "danorama.flip-vscode")) |= (
              .version = $version |
              .location.path = $path |
              .relativeLocation = $relPath |
              .metadata.pinned = false |
              .metadata.installedTimestamp = $ts
           )' "$extensions_file.bak" > "$extensions_file"
        
        rm "$extensions_file.bak"
        echo "    ✓ Updated to v$VERSION"
    done
fi

# Step 4: Also update the default profile's extensions (if not using profiles)
DEFAULT_EXT_FILE="$VSCODE_USER_DIR/extensions.json"
if [ -f "$DEFAULT_EXT_FILE" ] && grep -q "\"$EXTENSION_ID\"" "$DEFAULT_EXT_FILE" 2>/dev/null; then
    echo "  Updating default profile..."
    cp "$DEFAULT_EXT_FILE" "$DEFAULT_EXT_FILE.bak"
    
    NEW_PATH="$VSCODE_EXT_DIR/danorama.flip-vscode-$VERSION"
    TIMESTAMP=$(date +%s)000
    
    jq --arg version "$VERSION" \
       --arg path "$NEW_PATH" \
       --arg relPath "danorama.flip-vscode-$VERSION" \
       --argjson ts "$TIMESTAMP" \
       '(.[] | select(.identifier.id == "danorama.flip-vscode")) |= (
          .version = $version |
          .location.path = $path |
          .relativeLocation = $relPath |
          .metadata.pinned = false |
          .metadata.installedTimestamp = $ts
       )' "$DEFAULT_EXT_FILE.bak" > "$DEFAULT_EXT_FILE"
    
    rm "$DEFAULT_EXT_FILE.bak"
    echo "    ✓ Updated to v$VERSION"
fi

echo ""
echo "✓ Extension v$VERSION installed successfully!"
echo "→ Please reload VS Code window (Cmd+Shift+P → 'Reload Window')"

