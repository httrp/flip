#!/usr/bin/env bash
set -euo pipefail

# Guardrail for public repository hygiene.
# Fails if clearly internal working docs are tracked.

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# Keep this list intentionally narrow to avoid false positives.
DENY_PATTERNS=(
  '^CODE_REVIEW_.*\.md$'
  '^docs/AI_CHAT_HANDOFF_.*\.md$'
  '^docs/AI_FEATURES_ARCHITECTURE\.md$'
  '^docs/TEMPLATE_RESEARCH\.md$'
  '^docs/VSCODE_EXTENSION_PLAN\.md$'
  '^docs/VSCODE_TASKS_PLAN\.md$'
  '^docs/archive/.*$'
)

files="$(git ls-files)"
violations=()

for pattern in "${DENY_PATTERNS[@]}"; do
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    violations+=("$line")
  done < <(printf '%s\n' "$files" | grep -E "$pattern" || true)
done

if [[ ${#violations[@]} -gt 0 ]]; then
  printf 'Public repo guard failed. Remove or relocate these files:\n' >&2
  printf ' - %s\n' "${violations[@]}" | sort -u >&2
  printf '\nHint: keep internal working docs outside tracked paths (for example in local notes).\n' >&2
  exit 1
fi

echo "✓ Public repo guard passed"
