#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck disable=SC1091
set -a && source "$ROOT/versions.env" && set +a
RANGE="${1:-origin/main..HEAD}"
if ! command -v gitlint >/dev/null 2>&1; then
  echo "install: pip install gitlint==${GITLINT_VERSION:-0.19.1}"
  exit 1
fi
git log "$RANGE" --format=%B | gitlint --ignore=body-is-missing