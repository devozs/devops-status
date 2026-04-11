#!/bin/sh
# Run *.sql in SEEDS_DIR in sorted order. Idempotent via ON CONFLICT / IF NOT EXISTS in each file.
set -eu
SEEDS_DIR="${SEEDS_DIR:-/seeds}"
export PGCONNECT_TIMEOUT="${PGCONNECT_TIMEOUT:-60}"

if [ -z "${DATABASE_URL:-}" ]; then
  echo "error: DATABASE_URL is not set" >&2
  exit 1
fi

cd "$SEEDS_DIR"
# shellcheck disable=SC2012
files=$(ls -1 *.sql 2>/dev/null | LC_ALL=C sort || true)
if [ -z "$files" ]; then
  echo "no seed files in $SEEDS_DIR, skipping"
  exit 0
fi

for f in $files; do
  echo "seed: $f"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$SEEDS_DIR/$f"
done

echo "seeds complete"
