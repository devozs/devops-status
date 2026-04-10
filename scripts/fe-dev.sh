#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# shellcheck source=/dev/null
source "$ROOT/scripts/dev-node-env.sh"

if ! ensure_npm_on_path; then
  echo "error: npm not found. Install Node or use nvm/mise/asdf/fnm (see scripts/dev-node-env.sh)." >&2
  exit 1
fi

FE_PORT="${RUN_SH_FE_PORT:-3000}"
if command -v lsof >/dev/null 2>&1; then
  lsof -t -i:"$FE_PORT" | xargs kill -9 2>/dev/null || true
fi

cd devops-status-fe
exec npm run dev
