#!/usr/bin/env bash
# Dump the production database into .backups/ and keep the newest five dumps.
#
# Usage: scripts/backup-db.sh
# Env:   BACKUP_DIR   destination (default: <repo>/.backups)
#        BACKUP_KEEP  how many dumps to keep (default: 5)
#        PROD_DB_CONTAINER (default: jfshare-db)
set -euo pipefail

# Dumps contain share tokens and password hashes; keep them private even on a
# shared host.
umask 077

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$REPO_ROOT/.backups}"
BACKUP_KEEP="${BACKUP_KEEP:-5}"
PROD_DB_CONTAINER="${PROD_DB_CONTAINER:-jfshare-db}"

if ! [[ "$BACKUP_KEEP" =~ ^[1-9][0-9]*$ ]]; then
  echo "error: BACKUP_KEEP must be a positive integer" >&2
  exit 1
fi

if [ "$(docker inspect -f '{{.State.Running}}' "$PROD_DB_CONTAINER" 2>/dev/null || true)" != "true" ]; then
  echo "error: database container '$PROD_DB_CONTAINER' is not running" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR" 2>/dev/null || true

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
destination="$BACKUP_DIR/jfshare-$timestamp.sql.gz"
temporary="$destination.tmp"
trap 'rm -f "$temporary"' EXIT

echo "Dumping $PROD_DB_CONTAINER ..."
docker exec "$PROD_DB_CONTAINER" sh -c '
  PGPASSWORD="$POSTGRES_PASSWORD" pg_dump \
    -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --no-acl
' | gzip -9 > "$temporary"

# A dump that cannot be decompressed is worse than no dump: it looks like a
# backup until the day it is needed.
gzip -t "$temporary"
chmod 600 "$temporary"
mv "$temporary" "$destination"
trap - EXIT

# Filenames sort chronologically because the timestamp is fixed-width UTC.
mapfile -t dumps < <(
  find "$BACKUP_DIR" -maxdepth 1 -type f -name 'jfshare-*.sql.gz' -printf '%f\n' | sort
)
remove_count=$(( ${#dumps[@]} - BACKUP_KEEP ))
if (( remove_count > 0 )); then
  for old in "${dumps[@]:0:remove_count}"; do
    rm -f -- "$BACKUP_DIR/$old"
    echo "Removed old backup: $old"
  done
fi

echo "Backup written: $destination"
echo "Keeping the newest $BACKUP_KEEP dump(s) in $BACKUP_DIR"
