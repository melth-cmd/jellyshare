# Deployment checklist

Use this before exposing Jellyshare to the internet, and again after a
server move or key rotation. The app is designed to sit behind a TLS-terminating
reverse proxy; it should never be the thing holding port 80/443 itself.

## Before the first start

- [ ] **DNS and TLS are ready.** `PUBLIC_BASE_URL` is the real `https://` URL
      and the certificate is valid for it.
- [ ] **Only 80/443 are public.** Port `8097` and the database port are not
      forwarded or exposed by the firewall.
- [ ] **`.env` is private.** `chmod 600 .env`; it must never be committed.
- [ ] **Keys are generated, not copied from the example.**
      `openssl rand -hex 32` for `BACKEND_API_KEY`; a real
      `POSTGRES_PASSWORD`; a Jellyfin API key from the Jellyfin dashboard.
- [ ] **`JELLYSHARE_ENV=production`** (the default) and
      **`PUBLIC_BASE_URL=https://...`** are set.
- [ ] **The image is pinned** in `docker-compose.yml` to a release tag, not
      `latest`.
- [ ] **Backups are running** and a restore has been tested. See the Backups
      section in the README.

## Reverse proxy

Caddy is a good default. Terminate TLS at Caddy and proxy to the app; Caddy sets
`X-Forwarded-Proto` and appends the client address to `X-Forwarded-For`.

```caddyfile
share.example.com {
    encode zstd gzip
    reverse_proxy 127.0.0.1:8097
}
```

If Caddy runs in the same Docker network as the app, proxy to `jellyshare:8097`
instead. Do not publish `8097` on the host.

The backend trusts forwarded client addresses only when the request comes from a
loopback or private proxy, and only uses the rightmost `X-Forwarded-For` value.
A direct internet client cannot choose its own rate-limit bucket.

## Start and verify

```bash
gh auth token | docker login ghcr.io -u melth-cmd --password-stdin
docker compose pull jellyshare
docker compose up -d

curl -fsS http://127.0.0.1:8097/health
docker compose logs --tail=50 jellyshare
```

Expected headers on the public URL:

```bash
curl -sI https://share.example.com/health | grep -Ei \
  'strict-transport|x-content-type|x-frame|referrer|permissions|x-robots'
```

- `Strict-Transport-Security` present over HTTPS only.
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: SAMEORIGIN`
- `Referrer-Policy: no-referrer`
- `X-Robots-Tag: noindex, nofollow`

Then create one test share from `/admin`, open it from outside the LAN, and
revoke it. Keep the previous image tag ready for rollback.

## Upgrades and rollback

1. Run `make backup-db`.
2. Change the pinned tag in `docker-compose.yml`.
3. `docker compose pull jellyshare && docker compose up -d`.
4. Check `/health`, logs, and one real link.
5. Roll back by restoring the previous tag, pulling, and bringing it up again.

No database migration should be run without a fresh backup. If a release adds
one, it runs automatically at startup.

## Backups

- `make backup-db` writes `.backups/jellyshare-<timestamp>.sql.gz` and keeps five.
- The user systemd timer runs nightly; confirm it with
  `systemctl --user list-timers jellyshare-backup.timer`.
- Copy `.backups/` off the host. The timer protects against bad upgrades and
  accidental deletes, not disk or host loss.
- Test a restore into the dev database before relying on the dump.

## Key rotation and incidents

Rotate immediately if `.env`, a share link, or the admin key may have leaked.

- **Backend key** — change `BACKEND_API_KEY` and restart. Admin sessions and
  unlock cookies are invalidated; viewers may need the share password again.
- **Jellyfin API key** — create a new key in Jellyfin, update `.env`, restart,
  then remove the old key.
- **Database password** — change the role password and `POSTGRES_PASSWORD`
  together, then restart both services.
- **A share link** — revoke it from `/admin`; active sessions are terminated.
- **Suspected compromise** — revoke the affected links, rotate the keys above,
  review `docker compose logs`, and restore a known-good dump only if data was
  changed.

## Ongoing monitoring

- `/health` from an external monitor.
- Container restart policy and disk usage (media, Postgres, `.backups`).
- Nightly backup timer success.
- Certificate expiry.
- `docker compose logs` for repeated 401/403/429 responses.

See the Configuration table in the README for every environment variable.
