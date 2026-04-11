#!/bin/sh
# Apply *.up.sql in order once per version; safe across repeated runs (no data drops).
set -eu
MIGRATIONS_DIR="${MIGRATIONS_DIR:-/migrations}"
export PGCONNECT_TIMEOUT="${PGCONNECT_TIMEOUT:-60}"

if [ -z "${DATABASE_URL:-}" ]; then
  echo "error: DATABASE_URL is not set" >&2
  exit 1
fi

psql "$DATABASE_URL" -v ON_ERROR_STOP=1 <<'EOSQL'
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
EOSQL

cd "$MIGRATIONS_DIR"
# shellcheck disable=SC2012
for f in $(ls -1 *.up.sql 2>/dev/null | LC_ALL=C sort); do
  version="${f%.up.sql}"
  applied="$(psql "$DATABASE_URL" -tAc "SELECT COUNT(*) FROM schema_migrations WHERE version = '${version}'" | tr -d '[:space:]')"
  if [ "$applied" = "1" ]; then
    echo "skip ${version} (already applied)"
    continue
  fi
  echo "apply ${version}"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$MIGRATIONS_DIR/$f"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations (version) VALUES ('${version}')"
done

echo "migrations complete"
