#!/usr/bin/env bash
# One-shot local dev: Docker (Postgres + tunnel), Go API, Nuxt — from repo root.
# Ctrl+C stops the frontend and tears down the backend child processes.
#
#   ./run.sh --kill   Stop dev listeners (API + Nuxt ports) and docker infra; do not start anything.
#   RUN_SH_KILL_SKIP_INFRA=1 ./run.sh --kill   Only free API/FE ports; leave Postgres/tunnel running.
#   RUN_SH_SKIP_DB_MIGRATE=1 ./run.sh   Skip db-migrate (faster restarts; use after migrations are applied).
#   Frontend uses nvm Node 22 by default (scripts/dev-node-env.sh). Override: RUN_SH_NVM_NODE_VERSION=20
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

load_env_dev() {
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
}

# Match Makefile be-run / fe-dev: force-free listeners (catches go run / node after process-group gaps).
kill_listeners_on_port() {
  local port="$1"
  if ! command -v lsof >/dev/null 2>&1; then
    echo "warn: lsof not installed; cannot clear port ${port}" >&2
    return 0
  fi
  lsof -t -i:"${port}" | xargs kill -9 2>/dev/null || true
}

if [[ "${1:-}" == "--kill" ]]; then
  load_env_dev
  PORT="${PORT:-8080}"
  FE_PORT="${RUN_SH_FE_PORT:-3000}"
  echo "==> run.sh --kill: stopping API (:${PORT}), frontend (:${FE_PORT})"
  kill_listeners_on_port "$PORT"
  kill_listeners_on_port "$FE_PORT"
  if [[ "${RUN_SH_KILL_SKIP_INFRA:-}" == "1" ]]; then
    echo "==> RUN_SH_KILL_SKIP_INFRA=1 — skipping make infra-down"
  else
    echo "==> make infra-down (postgres + tunnel)"
    make infra-down
  fi
  echo "==> done"
  exit 0
fi

load_env_dev

PORT="${PORT:-8080}"
FE_PORT="${RUN_SH_FE_PORT:-3000}"

echo "==> make dev-infra-up (postgres + cloudflared tunnel)"
make dev-infra-up

# ---------------------------------------------------------------------------
# Tunnel health-check: ensure the Cloudflare quick tunnel is actually serving
# traffic, not just "running" with a stale/broken session. If the URL is
# unreachable, recreate the container to obtain a fresh hostname.
# ---------------------------------------------------------------------------
ensure_tunnel_healthy() {
  local compose_dir="$ROOT/devops-status-dev"
  local max_wait=30          # seconds to wait for URL in logs after (re)start
  local url=""

  # 1. Extract the most recent trycloudflare URL from container logs.
  extract_tunnel_url() {
    docker logs devops-status-tunnel 2>&1 \
      | grep -oE 'https://[a-zA-Z0-9.-]+\.trycloudflare\.com' \
      | tail -1 || true
  }

  url="$(extract_tunnel_url)"

  # 2. If we got a URL, probe it.  A working tunnel returns any HTTP status
  #    from the backend (or Cloudflare 502/503). DNS failure (curl exit 6) or
  #    connection refused (exit 7) means the session is dead.
  if [[ -n "$url" ]]; then
    echo "==> tunnel URL from logs: $url"
    if curl -sf --max-time 8 "$url/healthz" >/dev/null 2>&1; then
      echo "==> tunnel is healthy"
      _write_tunnel_env "$url"
      return 0
    fi
    echo "==> tunnel URL is unreachable — recycling container for a new hostname"
  else
    echo "==> no tunnel URL found in logs — recycling container"
  fi

  # 3. Recreate the tunnel container to get a brand-new Cloudflare session.
  (cd "$compose_dir" && docker compose rm -sf tunnel && \
   TUNNEL_TARGET_PORT="${PORT}" docker compose up -d tunnel)

  # 4. Wait for the new URL to appear in logs.
  echo "==> waiting for new tunnel URL (up to ${max_wait}s)…"
  url=""
  for _ in $(seq 1 "$max_wait"); do
    url="$(extract_tunnel_url)"
    [[ -n "$url" ]] && break
    sleep 1
  done

  if [[ -z "$url" ]]; then
    echo "warn: could not obtain a tunnel URL after ${max_wait}s — K8s onboarding will fail" >&2
    return 0
  fi

  # 5. Verify the fresh tunnel actually works (backend may not be up yet so
  #    accept any HTTP response; we only reject DNS/connection failures).
  echo "==> new tunnel URL: $url — verifying reachability…"
  local ok=0
  for _ in $(seq 1 15); do
    # --head + --max-time avoids blocking; any HTTP response (even 502) is fine.
    if curl -sI --max-time 5 "$url/healthz" >/dev/null 2>&1; then
      ok=1; break
    fi
    sleep 1
  done

  if [[ "$ok" == "1" ]]; then
    echo "==> tunnel is healthy"
  else
    echo "warn: tunnel URL obtained but not yet reachable (backend may not be up yet)" >&2
  fi
  _write_tunnel_env "$url"
}

_write_tunnel_env() {
  local url="$1"
  local local_file="$ROOT/.env.dev.local"
  touch "$local_file"
  if grep -q '^EXTERNAL_URL=' "$local_file" 2>/dev/null; then
    sed -i "s|^EXTERNAL_URL=.*|EXTERNAL_URL=${url}|" "$local_file"
  else
    { echo "# Auto-managed by run.sh — tunnel public URL"; echo "EXTERNAL_URL=${url}"; } >> "$local_file"
  fi
  export EXTERNAL_URL="$url"
  echo "==> .env.dev.local updated: EXTERNAL_URL=$url"
}

ensure_tunnel_healthy

# Apply SQL migrations (required after pulls that add columns, e.g. k8s api_verify_*). Skip with RUN_SH_SKIP_DB_MIGRATE=1
if [[ "${RUN_SH_SKIP_DB_MIGRATE:-}" != "1" ]]; then
  echo "==> make db-migrate"
  make db-migrate
fi

# Do not use `kill 0` in EXIT cleanup: it signals this shell's process group and can
# re-enter the trap hundreds of times. Stop the known backend job + free ports instead.
BE_MAKE_PID=""

cleanup() {
  if [[ "${_RUN_SH_CLEANUP_DONE:-}" == "1" ]]; then
    return 0
  fi
  _RUN_SH_CLEANUP_DONE=1
  echo ""
  echo "==> stopping background jobs and clearing dev ports…"
  if [[ -n "${BE_MAKE_PID:-}" ]] && kill -0 "$BE_MAKE_PID" 2>/dev/null; then
    kill "$BE_MAKE_PID" 2>/dev/null || true
    wait "$BE_MAKE_PID" 2>/dev/null || true
  fi
  sleep 0.2
  kill_listeners_on_port "$PORT"
  kill_listeners_on_port "$FE_PORT"
}
trap cleanup EXIT INT TERM

echo "==> make be-run (background)"
make be-run &
BE_MAKE_PID=$!

if command -v curl >/dev/null 2>&1; then
  echo "==> waiting for http://127.0.0.1:${PORT}/healthz …"
  for _ in $(seq 1 120); do
    if curl -sf "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
      echo "==> backend is healthy"
      break
    fi
    sleep 0.25
  done
else
  echo "==> curl not found; sleeping 5s before starting frontend"
  sleep 5
fi

echo "==> frontend (scripts/fe-dev.sh — Ctrl+C exits and stops backend)"
export RUN_SH_FE_PORT="$FE_PORT"
bash "$ROOT/scripts/fe-dev.sh"
