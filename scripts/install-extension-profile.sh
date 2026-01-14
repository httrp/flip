#!/bin/bash
# Install VS Code extension in all profiles or current window

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

# Find all VS Code profiles
PROFILES_DIR="$HOME/Library/Application Support/Code/User/profiles"
if [ ! -d "$PROFILES_DIR" ]; then
    echo "❌ VS Code profiles directory not found at $PROFILES_DIR"
    exit 1
fi

echo "→ Scanning VS Code profiles..."

PROFILES=()
for profile_dir in "$PROFILES_DIR"/*/; do
    profile_id=$(basename "$profile_dir")
    if [ "$profile_id" != "profiles" ]; then
        PROFILES+=("$profile_id")
    fi
done

if [ ${#PROFILES[@]} -eq 0 ]; then
    echo "❌ No VS Code profiles found"
    exit 1
fi

echo "→ Found ${#PROFILES[@]} profile(s)"

# Install in each profile that has flip-vscode
SUCCESS=0
FAILED=0

for profile_id in "${PROFILES[@]}"; do
    profile_path="$PROFILES_DIR/$profile_id"
    extensions_file="$profile_path/extensions.json"
    
    if [ ! -f "$extensions_file" ]; then
        continue
    fi
    
    # Check if flip is installed in this profile
    if grep -q '"id":"danorama.flip-vscode"' "$extensions_file" 2>/dev/null; then
        echo ""
        echo "→ Updating extension in profile: $profile_id"
        echo "  Uninstalling old version first..."
        
        "$CLI" --uninstall-extension danorama.flip-vscode 2>/dev/null || true
        sleep 1
        
        echo "  Installing v$VERSION..."
        if "$CLI" --install-extension "$VSIX" --force 2>&1 | grep -q "successfully installed"; then
            echo "  ✓ Extension v$VERSION installed in profile: $profile_id"
            ((SUCCESS++))
        else
            echo "  ⚠ Installation may have issues in profile: $profile_id"
            ((FAILED++))
        fi
    fi
done

echo ""
if [ $SUCCESS -gt 0 ]; then
    echo "✓ Extension v$VERSION installed in $SUCCESS profile(s)"
fi
if [ $FAILED -gt 0 ]; then
    echo "⚠ $FAILED profile(s) had issues"
fi

if [ $SUCCESS -eq 0 ] && [ $FAILED -eq 0 ]; then
    echo "⚠ Flip extension not found in any profile. Installing in default..."
    "$CLI" --uninstall-extension danorama.flip-vscode 2>/dev/null || true
    sleep 1
    "$CLI" --install-extension "$VSIX" --force
    echo "✓ Extension v$VERSION installed"
fi

