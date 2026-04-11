#!/bin/sh
set -e
# Skip migrations (e.g. local binary testing without Postgres in container).
if [ "${SKIP_DB_MIGRATE:-}" = "1" ]; then
  exec /server
fi
if [ -n "${DATABASE_URL:-}" ]; then
  /migrate.sh
  if [ "${SKIP_DB_SEED:-}" != "1" ]; then
    /seed.sh
  fi
else
  echo "warning: DATABASE_URL not set, skipping DB migrations" >&2
fi
exec /server
