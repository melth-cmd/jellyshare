# Operations

## Backups

The database is the only copy of share links, password hashes, sessions and
playback progress. Dumps are written to `.backups/` in the repository; the
directory is gitignored and excluded from the Docker build context.

```bash
make backup-db
# or: scripts/backup-db.sh
```

The script keeps the newest **five** dumps and removes older ones. Override with
`BACKUP_DIR`, `BACKUP_KEEP`, or `PROD_DB_CONTAINER` if needed. A dump is
decompressed and checked before it is kept, so a truncated file is not mistaken
for a backup. Dumps are written `0600` inside a `0700` directory because they
contain share tokens and password hashes.

### Nightly timer

The units in `scripts/systemd/` run a backup at 03:30 local time with up to five
minutes of jitter. `Persistent=true` catches a run missed while the machine was
off.

```bash
mkdir -p ~/.config/systemd/user
cp scripts/systemd/jellyshare-backup.service scripts/systemd/jellyshare-backup.timer ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now jellyshare-backup.timer
systemctl --user list-timers jellyshare-backup.timer
```

The unit assumes the checkout is at `~/jellyshare`; adjust `WorkingDirectory`
and `ExecStart` if it is elsewhere. A user timer keeps running after logout only
when lingering is enabled (`loginctl enable-linger "$USER"`).

### Restore

```bash
# Restore into the dev database (the safe default)
make restore-db FILE=.backups/jellyshare-20261102T050239Z.sql.gz

# Restore production only when you mean it
docker compose stop jellyshare
FORCE_PRODUCTION=1 scripts/restore-db.sh .backups/jellyshare-20261102T050239Z.sql.gz jellyshare-db
docker compose start jellyshare
```

The restore drops and recreates the target's `public` schema, then loads the
dump. Stop the app first when restoring production; its open connections are
terminated and it should reconnect after the restore completes. The dev target
defaults to `jellyshare-db-dev`.

## Upgrading

Releases are pinned in `docker-compose.yml`; production never follows `latest`.

```bash
# 1. Back up first
make backup-db

# 2. Change the image tag in docker-compose.yml to the new release, then
docker compose pull jellyshare
docker compose up -d

# 3. Check it came up and a link still opens
curl -fsS http://localhost:8097/health
```

To roll back, put the previous tag back in `docker-compose.yml`, pull, and
`docker compose up -d` again. Keep the previous dump in `.backups/` in case the
new release needs a database change.

The local fallback for a host without GHCR access is `make prod-rebuild`, which
builds the pinned tag from this checkout.
