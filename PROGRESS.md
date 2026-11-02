# Progress and Plan

## Current status

2026-11-01

- Repo detached from upstream: fresh single-commit history, private
  `melth-cmd/jellyfin-share-backend`, backups of the old `.git` kept outside the
  repo.
- Rebranded module path, container image, Dockerfile label, README clone URL,
  and footer links to `melth-cmd/jellyfin-share-backend`.
- Removed all companion plugin references.
- Added `AGENTS.md`, `SPEC.md`, `PROGRESS.md`.
- `go build ./...` and `go vet ./...` pass.

## Decisions

- Admin tabs: **Create** and **Shares**. The tab buttons live at the right end of
  the existing top pill, before the refresh and sign-out buttons.
- Shares tab uses a segmented filter: **Active / Inactive / All**, defaulting to
  Active.
- New shares are owned by the first Jellyfin user the backend already resolves at
  startup. No user picker for now.
- Dev testing uses the hot-reload stack on port **8098**, separate from
  production on 8097.
- Dev database is a **copy of production**, refreshed on demand, so existing
  shares are visible while testing.
- The create form exposes the same terms the API already accepts: expiry, never
  expires, play limit, concurrent viewers, quality cap, password, downloads.
- Mass actions work on explicitly selected rows: revoke or delete. There is no
  bulk edit; editing is per share. Delete is only allowed for inactive links.
- Stale links are deleted after 365 days past expiry or revocation, adjustable
  with `JFSHARE_SHARE_RETENTION_DAYS`.

## Iterating without rebuilding the compose stack

The repo already has a hot-reload stack: `docker-compose.dev.yml` runs the Go
server under `air` and the Svelte app under Vite, both bind-mounted from the
working tree. After Step 0, the loop is:

```bash
make dev-up                  # once per session
scripts/sync-dev-db.sh       # optional, refresh data from production
```

- Open `http://localhost:5173/admin` for the UI (Vite HMR).
- Go edits rebuild and restart the dev server automatically via `air`.
- SQL migration edits also restart the dev server; migrations run at startup.
- Production stays up at `http://localhost:8097` and is untouched.

The dev stack reads `JELLYFIN_URL` and `JELLYFIN_API_KEY` from `.env`, so it
talks to the real Jellyfin library. Picking a real film or series works in dev.

## Plan

### Step 0 — Fast dev loop

- [x] Remap the dev backend to host port `8098` (container port stays `8097`) so
      it can run beside production.
- [x] Make the Vite proxy target configurable: it follows `JFSHARE_PORT`
      (8097 inside the dev container), and `VITE_API_TARGET` overrides it when
      Vite runs on the host.
- [x] Add `scripts/sync-dev-db.sh`: dump `jfshare-db`, restore into
      `jfshare-db-dev` (drop and recreate the public schema first).
- [x] Add a `make dev-db-sync` target.
- [x] Confirm air picks up Go changes and Vite picks up Svelte changes.

Done when: editing a Go file and a Svelte file is reflected in the browser with
no compose rebuild, and dev data matches production after a sync.

Status: done. Dev backend is on `http://localhost:8098`, Vite on
`http://localhost:5173`, and the sync script copies production data. Go changes
rebuild via `air`; Svelte changes are served by Vite immediately.

### Step 1 — Backend: Jellyfin library search

- [x] Add `SearchItems` to `internal/jellyfin/client.go`, using the Jellyfin
      `/Items` endpoint with `recursive=true` and
      `includeItemTypes=Movie,Series,Season,Episode`.
- [x] Return the fields the picker needs: id, name, type, year, series/season
      names, overview, runtime, and whether a poster exists.
- [x] Add `GET /api/admin/jellyfin/items?q=...&limit=...` under the existing
      admin auth middleware.
- [x] Add handler tests with an `httptest` Jellyfin stub.

- [x] Add `GET /api/admin/jellyfin/items/{id}/poster` so the picker can show
      posters without exposing the Jellyfin key to the browser.

Done when: a search for part of a real title returns usable rows, and an empty
query returns a sensible default (recent items or an empty prompt).

Status: done. Search and poster proxying are live on the dev stack. Verified
against the real library: recent items, a title search, poster bytes, and 401
without the admin key. Client tests cover query building and field mapping.

### Step 2 — Admin shell and tabs

- [x] Split out `CreatePanel.svelte`; the shares index stays in
      `AdminDashboard.svelte` for now, without rewriting its list/detail behavior.
- [x] Replace the current top pill with a wider bar: wordmark on the left, tab
      buttons on the right, then refresh and sign out.
- [x] Style the tabs to match the existing design tokens.
- [x] Keep the selected tab in session state; do not lose it on refresh.
- [x] On mobile, keep the bar usable (tabs wrap or collapse; counts may hide as
      they already do).

Done when: the admin can switch between Create and Shares freely and the bar
looks right on desktop and phone.

Status: done. The pill has Create and Shares tab buttons before refresh and sign
out, the selected tab survives a reload via `sessionStorage`, and on narrow
screens the wordmark text and link count give way to the tabs and actions.
`CreatePanel` is a placeholder until Step 4.

### Step 3 — Shares tab: Active / Inactive / All

- [x] Add a `status=active|inactive|all` query parameter to
      `GET /api/admin/shares`, applied in `GetSharesPage` and `CountShares`.
- [x] Treat inactive as revoked, expired, or play limit spent.
- [x] Return counts for the current filter so the header is accurate.
- [x] Add the segmented filter to the Shares tab and reset pagination when it
      changes.
- [x] Keep Revoke and the detail sheet working in every filter.

Done when: selecting Inactive shows only dead links, All shows everything, and
the Active count matches the real live links.

Status: done. The API returns `counts {active, inactive, all}` and the UI shows
them on the filter buttons. Revoking from Active removes the row in place and
moves the count to Inactive. Verified with a test share in the dev database:
created → Active 1, revoked → Active 0, then deleted the test row.

### Step 4 — Create share form

- [x] Item picker: debounced search, poster and title rows, select one item.
- [x] Show the selected item with its poster, title, type, year, and runtime.
- [x] Expiry: preset chips (24 h, 7 d, 30 d, Never) plus a custom option.
- [x] Play limit and concurrent viewer fields, optional.
- [x] Quality cap: Auto and common heights; optional bitrate cap.
- [x] Password field, optional.
- [x] Downloads toggle, default on.
- [x] Inline validation; keep form state on server errors.
- [x] Submit to `POST /api/admin/shares`; on success show the public URL with a
      copy button, and offer "Create another".

Done when: a share created from the admin page opens and plays through the
existing public share page.

Status: done. The form searches the live library, shows posters through the
admin image proxy, and exposes expiry, play and viewer limits, quality caps,
password, and downloads. The backend now fills the Jellyfin user automatically
when the request leaves it out. Success shows the public URL with a copy button,
Create another, and View in Shares. Verified by creating a share through the API
without a user id, checking its settings in the Active list, and deleting the
test rows.

### Step 5 — Series-first picker (revise Step 4)

Why the current picker misses things: the Jellyfin search endpoint matches each
item's own name. A series matches its title, but its seasons are named "Season
1" and its episodes are named after the episode, so neither contains the series
title. Searching "smith" returns *Mr. & Mrs. Smith* as a series but none of its
episodes; browsing its children returns all 8. Flat search is also capped at 20
and mixes in unrelated episodes whose titles match, which crowds out what the
admin was looking for.

That is not lost data - it is the wrong access pattern. Browse the tree instead
of searching it flat.

- [x] Narrow the search to top-level items: `Movie,Series` only. Empty query
      still lists recently added films and series. Update the client tests.
- [x] Add `GET /api/admin/jellyfin/series/{id}/seasons`: id, name, season
      number, poster flag, sorted by number. Include Specials (season 0).
- [x] Add `GET /api/admin/jellyfin/seasons/{id}/episodes`: id, name, episode
      number, overview, runtime, poster flag, sorted by number.
- [x] Add `BrowseSeasons` / `BrowseEpisodes` to the Jellyfin client. Map to the
      same picker row shape as `SearchItem`; do not disturb the existing public
      episode code.
- [x] Fallback: if a series returns no Season children, fetch its episodes
      recursively and show one "Episodes" group. The series itself stays
      shareable.
- [x] Series result row: poster, title, year, a disclosure chevron, and a
      **Share series** button. Expanding it loads seasons lazily.
- [x] Season row: name ("Season 1", "Specials"), a disclosure chevron, and a
      **Share season** button. Expanding it loads episodes lazily.
- [x] Episode row: clicking it selects that episode. No extra expand step.
- [x] Cache seasons and episodes per id for the life of the panel; the Refresh
      button clears the cache and reloads the current search. Multiple series
      and seasons may be open at once.
- [x] Keep expansion state when the admin chooses an item and then presses
      **Change item**.
- [x] Loading and error states inside each expanded node, with a retry that only
      retries that node.
- [x] Accessibility: real buttons, `aria-expanded` on disclosure controls, no
      nested interactive elements, keyboard reachable.
- [x] Responsive: indent the tree on phones; keep the result list scrollable
      rather than growing the page without bound.
- [x] Tests: client query/mapping for seasons and episodes, handler tests, and a
      live check against a multi-season series plus one with specials.

Done when: searching a series shows it; expanding shows every season; expanding
a season shows every episode; and sharing a series, a season, or an episode all
work from the tree.

Status: done. Search now returns only films and series. Series rows expand into
seasons with Share season buttons; seasons expand into episodes that select on
click. Seasons and episodes load lazily, cache per id, and a series with no
Season objects falls back to a flat episode group. Verified against the live
library: Mr. & Mrs. Smith returned 1 season and 8 episodes; Attack on Titan
returned 4 seasons with 25, 12, 22, and 28 episodes.

### Step 6 — Selection, mass actions, and per-share actions

Selection is frontend state; the mass actions get dedicated endpoints so a
partial failure is reported honestly. There is no bulk edit.

- [x] Add a checkbox column to the Shares rows, plus a select-all box for the
      loaded page.
- [x] Support shift-click range selection across the rows currently loaded.
- [x] Clear selection when the tab or Active/Inactive/All filter changes, after
      an action, and when the list is reloaded.
- [x] Add a persistent mass-action bar at the top of the Shares tab, under the
      filter: select-all, "N selected", Revoke selected, Delete selected, Clear.
      Actions stay disabled until something is selected.
- [x] Mass revoke: `POST /api/admin/shares/bulk/revoke` with `{ "ids": [...] }`
      in one SQL statement, returning which ids changed.
- [x] Mass delete: `POST /api/admin/shares/bulk/delete`, allowed only for
      inactive links. Cascades sessions and progress; audit rows keep their
      history because `share_id` is set NULL. Report skipped ids.
- [x] Per-share buttons next to each row: Copy link, Edit, Revoke. Revoke keeps
      its inline confirm.
- [x] Individual edit opens a dialog using the existing
      `PATCH /api/admin/shares/{id}`: extend expiry, change the play limit,
      change concurrent viewers, set or remove the password. Works from the
      detail sheet too.
- [x] Confirm before mass revoke and mass delete.

Implementation notes:

- Select-all in the mass-action bar applies to the rows currently loaded, not
  every page in the filter. The bar says so when more pages exist.
- Shift-click uses the last plain-clicked row as the anchor and applies the new
  checked state to the whole range.
- Mass revoke returns `{ revoked: [...], skipped: [...] }`; revoked rows also
  have their active sessions terminated, like the single-share action.
- Mass delete returns `{ deleted: [...], skipped: [...] }`; active rows are
  skipped server-side, not just hidden in the UI.
- Bulk ids are deduplicated, capped at 200 per request, and validated as UUIDs.
- The edit dialog uses the existing `PATCH /api/admin/shares/{id}`. Leaving a
  field blank means "leave it alone". Extending a never-expiring link is
  disabled with a note, because Jellyfin's SQL keeps its NULL expiry.
- Editing reloads the current filter so counts and statuses stay truthful;
  selection clears.
- The edit dialog is also reachable from the detail sheet. Fields: add days to
  expiry, play limit, concurrent viewers, set password, remove password.
  Quality and download changes are not part of this step.

Done when: ticking a few shares and choosing Revoke or Delete does what the
label says, shift-click selects the range, and each row can be edited or revoked
on its own.

Status: done. The Shares tab has a checkbox column, select-all, shift-click
ranges, and a mass-action bar with revoke/delete plus a visible confirmation.
Rows have Copy link, Edit, and Revoke; Edit is also reachable from the detail
sheet. Bulk revoke terminates sessions; bulk delete removes only inactive links
and reports skipped active ids. Verified live with three temporary shares:
2 revoked, 2 deleted, the active one skipped, then cleaned up.

### Step 7 — Retention cleanup for stale links

Expiry already makes a link dead; there is no need to also mark it revoked. The
useful cleanup is deleting the row after it has been dead long enough.

- [x] Add `JFSHARE_SHARE_RETENTION_DAYS`, default 365.
- [x] Replace the unused `CleanupExpiredShares` with
      `DeleteStaleShares(ctx, retention)`: delete where `expires_at` or
      `revoked_at` is older than the cutoff. Never-expiring, non-revoked links
      are never touched.
- [x] Run it from the existing five-minute cleanup loop and log how many rows
      went.
- [x] Document the setting in `.env.example` and the README.

Done when: an old expired or revoked link disappears after the retention
window, while live and never-expiring links stay.

Status: done. `JFSHARE_SHARE_RETENTION_DAYS` defaults to 365 and `0` disables
cleanup. `DeleteStaleShares` replaces the unused `CleanupExpiredShares` and runs
from the existing five-minute loop. Verified in the dev database: a link expired
730 days ago and one revoked 730 days ago were deleted; a fresh link and a
never-expiring link were untouched. Test rows removed afterwards.

### Step 8 — Copy and empty states

- [x] Update dashboard copy that still implies links are created from Jellyfin.
- [x] Empty states per tab and per filter.
- [x] Make sure errors and disabled states are understandable.
- [x] Selection bar and checkboxes work on phone widths.

Status: done. Empty states now explain whether there are no live links, no
inactive links, or nothing at all, and point at the right tab. The mass-action
bar says what Select all covers, explains why its actions are disabled, uses
full-width buttons on phones, and reports skipped active links in words. Copy
failures now say so instead of silently doing nothing.

### Step 9 — Docs and verification

- [x] Update `SPEC.md` to match what was built.
- [x] Update `README.md` admin section with the new flow and dev workflow.
- [x] Run `go build ./...`, `go test ./...`, and `cd web && npm run build`.
- [x] Scripted checklist: create movie, series, season, and episode shares;
      password unlock; play limit; revoke, edit, mass revoke, mass delete;
      filter counts; retention cleanup.
- [ ] Visual and phone-width pass in the browser.
- [x] Commit and push to `origin main`.

Status: done. SPEC.md is now an implemented spec. README covers the Create and
Shares tabs, optional owner, status filtering and counts, bulk endpoints, library
browse endpoints, retention, and the dev loop on 8098 with `make dev-db-sync`.

Scripted verification against the dev stack: movie, series, season, and episode
shares created and resolved through the public API; a password share unlocked
and played with its cookie; the second play on a one-play link returned 403; all
five test links were revoked and deleted. `go vet`, `go test ./...`, and the
frontend build pass. Visual and phone-width checks remain a human pass.

## Roadmap after v0.1.0

v0.1.0 is the personal-use baseline and the Steps above are complete. The two
open milestones hold the next work; turn an item into an issue when it starts.
Keep each release small and only tag it after the scripted and visual checks
pass.

### 0.1.1 — operations and hardening

**Database backup and restore**
- Add `scripts/backup-db.sh`: `pg_dump` from `jfshare-db`, timestamped and
  compressed, with a retention count.
- Add `scripts/restore-db.sh` and a systemd timer/cron example.
- Restore a dump into the dev database as the test, not just checking that the
  file exists.
- Done when a nightly dump runs unattended and a restore has been verified.

**Pin the production image**
- Change `docker-compose.yml` from `:latest` to the current release tag and
  document the bump step.
- Confirm the GHCR package visibility and that the host can pull it.
- Done when the running container's image tag matches a release, not `latest`.

**CI security and dependencies**
- Add `govulncheck` and `npm audit` (fail on high/critical) to the GitHub
  Actions build.
- Pin action versions and document the update cadence.
- Done when a vulnerable dependency or toolchain fails CI.

**Deployment checklist**
- Add a short section covering TLS, which ports may be exposed, proxy headers,
  admin-key rotation, backups, retention, and upgrades.
- Done when a fresh deployment can follow it without reading the source.

**Startup warnings**
- Warn, without printing secrets, when the admin key is short, the public URL
  is HTTP in a non-dev setup, or required Jellyfin settings are missing.
- Done when a misconfigured production start says what is wrong.

**Fixes from real use**
- Triage anything found while sharing links in the `bug` label, fix the ones
  that affect playback or access first, and keep the rest for 0.2.0.

### 0.2.0 — next features

**Analytics UI**
- Surface the existing `GET /api/admin/shares/{id}/analytics`: views, unique
  viewers, average watch time, and views per day in the detail sheet.
- Add a small dashboard summary from the same data; keep it dependency-free
  unless a chart actually needs one.
- Done when the admin can see how a share was used without calling the API.

**Copy link plus password and QR code**
- Let the admin copy one block with the URL, password, and expiry for sending
  over chat.
- Add a QR code for phone hand-off; ask before adding a client dependency.
- Done when a new link can be handed over in one copy or one scan.

**Full edit coverage**
- Extend `PATCH /api/admin/shares/{id}` and `UpdateShare` with `allowDownload`,
  `maxVideoHeight`, and `maxVideoBitrate`.
- Add those fields to the edit dialog and keep the current reload/count
  behavior.
- Done when every create-form term except the shared item can be changed later.

**Un-revoke and duplicate**
- Add a restore endpoint that clears `revoked_at` only when the link is still
  usable, and a duplicate endpoint that copies the terms with a new token.
- Decide explicitly what restoring an expired link means; extending it may be
  the more useful action.
- Done when a mistaken revoke can be undone and terms can be reused.

**Partial multi-select in the picker**
- Add selection checkboxes to seasons and episodes, separate from the
  single-item Share buttons.
- Create one share per selected item with the same terms, then report per-item
  success and failure.
- Done when several episodes can be shared without sharing the whole season.

**Better search**
- Add fuzzy or alias matching, recently shared items, and remembering the last
  chosen item between visits.
- Check what Jellyfin's search already handles before adding client-side
  ranking.
- Done when finding a known title takes fewer keystrokes than today.

**Subtitle control on the share page** *(#15)*
- Move track selection below Play and replace the collapsed "Change" row with a
  visible, labelled Subtitles dropdown; decide where audio lives.
- Keep the chosen track feeding playback, autoplay and casting, and check the
  series/season and phone layouts.
- Done when a viewer can see and change subtitles without opening a disclosure.

**Backlog from use**
- Revisit the items deliberately left out of v0.1.0 and promote whichever the
  family workflow actually needs.

### 0.3.0 — viewer localization

**Thai for the public share page** *(#16)*
- Translate only the viewer-facing `/s/{token}` experience and player chrome:
  password gate, Play/Resume/Download, terms, refusals, dead links, downloads
  and cast notes. Admin stays English.
- Default the link page to Thai; add Thai-capable fonts through the existing
  tokens, and pin the date locale/calendar explicitly.
- Done when a Thai-speaking viewer can use a link, including its errors, without
  falling back to English.

## Testing notes

- Backend: `go test ./...`
- Frontend: `cd web && npm run build`
- Admin API smoke test (after Step 1):

  ```bash
  curl -s -H "X-Backend-Key: $BACKEND_API_KEY" \
    "http://localhost:8098/api/admin/jellyfin/items?q=alien&limit=5"
  ```

- Dev UI: `http://localhost:5173/admin`
- Production UI: `http://localhost:8097/admin`

## Log

### 2026-11-01

- Detached repo from upstream, rebranded, removed plugin references.
- Added agent instructions, spec, and this plan.
- Step 0 done: dev stack on 8098, Vite proxy configurable, `make dev-db-sync`
  copies production data. Fixed the dev image build (Node 20 + npm latest was
  incompatible) and `air` building with VCS stamping inside the container.
- Verified hot reload for Go and Svelte and synced the dev database.
- Step 1 done: `SearchItems` in the Jellyfin client, `GET /api/admin/jellyfin/items`
  and a poster proxy, plus client and handler tests.
- Step 2 done: Create/Shares tabs in the top pill, `CreatePanel` placeholder, and
  tab state kept in `sessionStorage`. Frontend build passes.
- Step 3 done: server-side Active/Inactive/All filter with counts, filter buttons
  in the Shares tab, and revoked rows leave the Active view in place. Added bulk
  actions and retention cleanup to the plan, now Steps 6 and 7.
- Step 4 done: create form with library picker, posters, terms, quality, password,
  downloads, success/copy state, and automatic Jellyfin user assignment. Shared
  clipboard helper works over plain HTTP on the LAN. Frontend build and API
  creation checks pass.
- Diagnosed missing seasons/episodes in the picker: Jellyfin's flat search
  matches each item's own name, so a series title never matches its seasons or
  episodes. Added Step 5: a series-first, collapsible tree that browses seasons
  and episodes instead of searching for them.
- Step 5 done: top-level-only search, season/episode browse endpoints, and a
  lazy, cached tree with Share series / Share season / episode selection plus
  a no-seasons fallback. Verified against a one-season and a four-season show.
- Fixed the password loop on plain-HTTP deployments: the session cookie was
  always Secure, so browsers dropped it and every Play asked again. Secure now
  follows TLS or X-Forwarded-Proto. Verified: cookie set without Secure over
  HTTP, Play with the cookie returns 200, without it 401.
- Step 6 done: row checkboxes, select-all, shift-click ranges, mass revoke and
  delete endpoints/UI, per-row Edit/Revoke/Copy, and an edit dialog. Bulk
  endpoints tested live with mixed active/inactive ids. Added handler tests for
  bulk id validation.
- Step 7 done: configurable retention (365 days by default, `0` disables),
  `DeleteStaleShares` in the cleanup loop, and docs. Verified against stale
  expired, stale revoked, fresh, and never-expiring rows.
- Steps 8 and 9 done: empty/error/disclosure copy polished, docs rewritten to
  match the built admin flow, and a scripted end-to-end pass covered movie,
  series, season, episode, password, play-limit, bulk, and cleanup paths.
- **v0.2.0 scope complete**: analytics tab, QR/copy details, visible track pickers,
  full edit coverage, restore and duplicate. Remaining 0.2.0 items moved to
  0.3.0; Thai localization moved to 0.4.0.
- #10 done: the edit dialog now changes download permission, maximum picture and
  maximum bitrate; `PATCH /api/admin/shares/{id}` accepts `allowDownload`,
  `maxVideoHeight` and `maxVideoBitrate` (0 clears a cap). The list exposes the
  stored download choice and bitrate so the editor can prefill them.
- #11 done: `POST /api/admin/shares/{id}/restore` clears a revoke only when the
  link is still usable, and `POST /api/admin/shares/{id}/duplicate` copies the
  terms with a new token and a fresh lifetime. Rows show Restore for revoked
  links; the detail sheet has Duplicate and Restore.
- #15 done: the public share page now shows Audio/Subtitles selects below Play,
  always visible, instead of a collapsed Change row above it.
- #9 done: the create success panel can copy a block with the URL, password and
  expiry, and renders a QR code for the link. Added `qrcode-generator`.
- #8 done: new Analytics tab in the admin pill with Overview and Links sub-tabs,
  backed by `GET /api/admin/analytics` (aggregate totals, 30-day views, and
  per-link views/unique viewers/average watch time).
- Fixed the blank admin page after the Svelte 5 upgrade: `main.js` still used the
  Svelte 4 `new App({ target })` API. It now uses `mount(App, { target })`.
  Verified by rendering the production bundle in jsdom: the admin login page
  mounts instead of staying on the boot screen. v0.1.2 released and deployed;
  the live bundle was fetched from the public URL and rendered in jsdom, both
  the login page and the dashboard mount correctly.
- **v0.1.1 released** (`779b8d6` + tag) with the deployment checklist, startup
  warnings, Go 1.27, frontend Svelte 5/Vite 7, CI scans, dependency updates,
  backups, and the pinned `0.1.1` image. Release workflow and image build passed.
- Deployment checklist #5 is done: `docs/DEPLOYMENT.md` covers TLS/proxy, port
  exposure, secret generation, GHCR auth, verification headers, upgrades and
  rollback, backups, key rotation, incident response, and monitoring; the README
  links it from the admin section.
- Startup warnings #6 are done: `JFSHARE_ENV=production|development` (default
  production) controls non-fatal warnings for a short admin key, a non-HTTPS
  public URL, and the default database password. Warnings never print secret
  values; tests cover production, development, safe config, and unknown values.
- v0.1.1 prep: compose and the local-build fallback pin `0.1.1`, the frontend
  package version is 0.1.1, and a local production image build completed with
  the Go 1.27 Dockerfile.
- CI security work for #4 is done: `go vet`, `govulncheck` (pinned v1.9.0), and
  `npm audit --audit-level=high` now run in GitHub Actions on pushes, PRs, and a
  weekly schedule; actions are pinned to SHAs and Dependabot watches Go, npm,
  and actions. Upgraded the frontend from Svelte 4/Vite 5 to Svelte 5/Vite 7 and
  hls.js 1.7; `npm ci` + production build pass in a clean Node 20 container and
  the high/critical audit baseline is clean. CI also exposed Go 1.23 stdlib
  findings, so `go.mod` and both Dockerfiles now use Go 1.27; local
  `govulncheck` is clean and the Build workflow is green.
- Backup work for #2 is done: `scripts/backup-db.sh` writes checked gzip dumps
  into gitignored `.backups/` and keeps five, `scripts/restore-db.sh` refuses
  production without `FORCE_PRODUCTION=1`, and a user systemd timer runs nightly
  at 03:30. Tested retention with dummy dumps, restored a real dump into a
  throwaway Postgres container, and verified counts and migration 12. Docs and
  Makefile targets added.
- Security pass: forwarded IPs now require a private proxy and use the rightmost
  X-Forwarded-For; admin digests compare in constant time; CORS no longer claims
  credentials; request bodies are capped; password guesses get a 10/min limit;
  unlock cookies are bound to the current password hash; dead-link images are
  refused; admin responses are no-store; public bearer tokens are masked in
  logs; security headers and HTTPS-only HSTS were added; read-header timeout
  set. Live checks: headers, password rotation invalidating the old cookie,
  mask in logs, and 404 for a dead share's poster. `.env` tightened to 0600.
