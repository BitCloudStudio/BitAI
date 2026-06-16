#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT"

if [[ ! -f "$ROOT/scripts/init.sh" ]]; then
  echo "未找到 scripts/init.sh" >&2
  exit 1
fi

exec bash "$ROOT/scripts/init.sh" "$@"
