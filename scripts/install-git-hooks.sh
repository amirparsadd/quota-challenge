#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOKS_SRC="$ROOT/scripts/git-hooks"
GIT_DIR="$(git -C "$ROOT" rev-parse --git-dir)"
HOOKS_DST="$GIT_DIR/hooks"

if [[ ! -d "$HOOKS_SRC" ]]; then
  echo "missing hook templates: $HOOKS_SRC" >&2
  exit 1
fi

mkdir -p "$HOOKS_DST"
for hook in "$HOOKS_SRC"/*; do
  [[ -f "$hook" ]] || continue
  name="$(basename "$hook")"
  chmod +x "$hook"
  ln -sf "$hook" "$HOOKS_DST/$name"
  echo "hook installed: $HOOKS_DST/$name -> $hook"
done