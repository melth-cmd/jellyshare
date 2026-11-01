#!/usr/bin/env bash
# Copy the production database into the dev database.
#
# Production runs in the `jfshare-db` container; the hot-reload dev stack runs
# its own Postgres in `jfshare-db-dev`. This replaces the dev schema with a
# fresh dump, so shares visible in production are visible while testing.
#
# Usage: scripts/sync-dev-db.sh
set -euo pipefail

PROD_DB_CONTAINER="${PROD_DB_CONTAINER:-jfshare-db}"
DEV_DB_CONTAINER="${DEV_DB_CONTAINER:-jfshare-db-dev}"

require_running() {
  local name="$1"
  if ! docker inspect "$name" >/dev/null 2>&1; then
    echo "error: container '$name' does not exist" >&2
    exit 1
  fi
  if [ "$(docker inspect -f '{{.State.Running}}' "$name")" != "true" ]; then
    echo "error: container '$name' is not running" >&2
    exit 1
  fi
}

require_running "$PROD_DB_CONTAINER"
require_running "$DEV_DB_CONTAINER"

echo "Resetting $DEV_DB_CONTAINER ..."
# Drop every connection first: the dev server keeps a pool open, and DROP
# SCHEMA would otherwise wait on it. The pool reconnects after the restore.
docker exec "$DEV_DB_CONTAINER" sh -c '
  PGPASSWORD="$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 -q \
    -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid();" \
    -c "DROP SCHEMA public CASCADE;" \
    -c "CREATE SCHEMA public;"
'

echo "Dumping $PROD_DB_CONTAINER ..."
docker exec "$PROD_DB_CONTAINER" sh -c '
  PGPASSWORD="$POSTGRES_PASSWORD" pg_dump \
    -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --no-acl
' | docker exec -i "$DEV_DB_CONTAINER" sh -c '
  PGPASSWORD="$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 -q \
    -U "$POSTGRES_USER" -d "$POSTGRES_DB"
'

echo "Done. Dev database now matches production."
