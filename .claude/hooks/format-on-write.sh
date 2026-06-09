#!/usr/bin/env bash
# PostToolUse(Edit|Write) : gofmt sur les fichiers Go écrits.
# Ne touche jamais au code généré (DO NOT EDIT) ni aux fichiers non-Go.
set -euo pipefail

input=$(cat)
file=$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty' 2>/dev/null || true)
[ -z "$file" ] && exit 0

case "$file" in
  *.go)
    if command -v gofmt >/dev/null 2>&1 && [ -f "$file" ]; then
      gofmt -w "$file"
      command -v goimports >/dev/null 2>&1 && goimports -w "$file" >/dev/null 2>&1 || true
    fi
    ;;
esac
exit 0
