#!/usr/bin/env bash
# Restore a dump produced by scripts/backup-db.sh into a target database.
#
# Usage: scripts/restore-db.sh <dump.sql.gz> [target-container]
#
# The target defaults to the dev database, jfshare-db-dev. Restoring into the
# production container requires FORCE_PRODUCTION=1 because it replaces live data.
set -euo pipefail

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
  echo "usage: $0 <dump.sql.gz> [target-container]" >&2
  echo "  target defaults to jfshare-db-dev" >&2
  exit 1
fi

DUMP="$1"
TARGET="${2:-jfshare-db-dev}"
PRODUCTION_CONTAINER="${PRODUCTION_CONTAINER:-jfshare-db}"

if [ ! -f "$DUMP" ]; then
  echo "error: dump not found: $DUMP" >&2
  exit 1
fi
if ! gzip -t "$DUMP" 2>/dev/null; then
  echo "error: not a readable gzip dump: $DUMP" >&2
  exit 1
fi
if [ "$TARGET" = "$PRODUCTION_CONTAINER" ] && [ "${FORCE_PRODUCTION:-0}" != "1" ]; then
  echo "error: refusing to overwrite the production database '$TARGET'." >&2
  echo "       set FORCE_PRODUCTION=1 only when you mean it." >&2
  exit 1
fi
if [ "$(docker inspect -f '{{.State.Running}}' "$TARGET" 2>/dev/null || true)" != "true" ]; then
  echo "error: database container '$TARGET' is not running" >&2
  exit 1
fi

echo "Restoring $DUMP into $TARGET ..."

# Drop connections first: the dev server keeps a pool open, and DROP SCHEMA
# would otherwise wait on it.
docker exec "$TARGET" sh -c '
  PGPASSWORD="$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 -q \
    -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid();" \
    -c "DROP SCHEMA public CASCADE;" \
    -c "CREATE SCHEMA public;"
'

gzip -dc "$DUMP" | docker exec -i "$TARGET" sh -c '
  PGPASSWORD="$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 -q \
    -U "$POSTGRES_USER" -d "$POSTGRES_DB"
'

echo "Restore complete: $TARGET"
