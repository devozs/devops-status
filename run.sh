#!/usr/bin/env bash
# One-shot local dev: Docker (Postgres + tunnel), Go API, Nuxt — from repo root.
# Ctrl+C stops the frontend and tears down the backend child processes.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

if [[ -f .env.dev ]]; then
  set -a
  # shellcheck source=/dev/null
  source .env.dev
  if [[ -f .env.dev.local ]]; then
    # shellcheck source=/dev/null
    source .env.dev.local
  fi
  set +a
fi

PORT="${PORT:-8080}"

echo "==> make dev-infra-up (postgres + cloudflared tunnel)"
make dev-infra-up

# Apply SQL migrations (required after pulls that add columns, e.g. k8s api_verify_*). Skip with RUN_SH_SKIP_DB_MIGRATE=1
if [[ "${RUN_SH_SKIP_DB_MIGRATE:-}" != "1" ]]; then
  echo "==> make db-migrate"
  make db-migrate
fi

# Same process group so SIGINT/SIGTERM can stop backend + make children.
set -m

cleanup() {
  echo ""
  echo "==> stopping background jobs (backend)…"
  kill 0 2>/dev/null || true
}
trap cleanup EXIT INT TERM

echo "==> make be-run (background)"
make be-run &

if command -v curl >/dev/null 2>&1; then
  echo "==> waiting for http://127.0.0.1:${PORT}/healthz …"
  for _ in $(seq 1 90); do
    if curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
      echo "==> backend is healthy"
      break
    fi
    sleep 1
  done
else
  echo "==> curl not found; sleeping 5s before starting frontend"
  sleep 5
fi

echo "==> make fe-dev (foreground — Ctrl+C exits and stops backend)"
make fe-dev
