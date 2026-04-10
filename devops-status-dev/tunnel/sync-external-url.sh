#!/usr/bin/env bash
# Extract the Cloudflare quick-tunnel URL from container logs and write repo-root .env.dev.local
# so make / Go pick up EXTERNAL_URL (Makefile uses: include .env.dev then -include .env.dev.local).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
COMPOSE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$COMPOSE_DIR"

if ! docker ps --format '{{.Names}}' | grep -qx devops-status-tunnel; then
  echo "Tunnel container is not running. Start it: make tunnel-up (from repo root)" >&2
  exit 1
fi

echo "Waiting for public URL in tunnel logs (up to 45s)..."
for _ in $(seq 1 45); do
  url="$(docker logs devops-status-tunnel 2>&1 | grep -oE 'https://[a-zA-Z0-9.-]+\.trycloudflare\.com' | tail -1 || true)"
  if [[ -n "${url}" ]]; then
    local_file="${REPO_ROOT}/.env.dev.local"
    touch "$local_file"
    if grep -q '^EXTERNAL_URL=' "$local_file" 2>/dev/null; then
      sed -i "s|^EXTERNAL_URL=.*|EXTERNAL_URL=${url}|" "$local_file"
    else
      { echo "# Generated / updated by devops-status-dev/tunnel/sync-external-url.sh"; echo "EXTERNAL_URL=${url}"; } >> "$local_file"
    fi
    echo "Updated ${local_file}"
    echo "EXTERNAL_URL=${url}"
    echo ""
    echo "make be-run auto-detects this URL from container logs; no restart needed."
    exit 0
  fi
  sleep 1
done

echo "Could not find a *.trycloudflare.com URL. Check: docker compose -f devops-status-dev/docker-compose.yml logs tunnel" >&2
exit 1
