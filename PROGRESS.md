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

- [ ] Add a checkbox column to the Shares rows, plus a select-all box for the
      loaded page.
- [ ] Support shift-click range selection across the rows currently loaded.
- [ ] Clear selection when the tab or Active/Inactive/All filter changes, after
      an action, and when the list is reloaded.
- [ ] Add a persistent mass-action bar at the top of the Shares tab, under the
      filter: select-all, "N selected", Revoke selected, Delete selected, Clear.
      Actions stay disabled until something is selected.
- [ ] Mass revoke: `POST /api/admin/shares/bulk/revoke` with `{ "ids": [...] }`
      in one SQL statement, returning which ids changed.
- [ ] Mass delete: `POST /api/admin/shares/bulk/delete`, allowed only for
      inactive links. Cascades sessions and progress; audit rows keep their
      history because `share_id` is set NULL. Report skipped ids.
- [ ] Per-share buttons next to each row: Copy link, Edit, Revoke. Revoke keeps
      its inline confirm.
- [ ] Individual edit opens a dialog using the existing
      `PATCH /api/admin/shares/{id}`: extend expiry, change the play limit,
      change concurrent viewers, set or remove the password. Works from the
      detail sheet too.
- [ ] Confirm before mass revoke and mass delete.

Done when: ticking a few shares and choosing Revoke or Delete does what the
label says, shift-click selects the range, and each row can be edited or revoked
on its own.

### Step 7 — Retention cleanup for stale links

Expiry already makes a link dead; there is no need to also mark it revoked. The
useful cleanup is deleting the row after it has been dead long enough.

- [ ] Add `JFSHARE_SHARE_RETENTION_DAYS`, default 365.
- [ ] Replace the unused `CleanupExpiredShares` with
      `DeleteStaleShares(ctx, retention)`: delete where `expires_at` or
      `revoked_at` is older than the cutoff. Never-expiring, non-revoked links
      are never touched.
- [ ] Run it from the existing five-minute cleanup loop and log how many rows
      went.
- [ ] Document the setting in `.env.example` and the README.

Done when: an old expired or revoked link disappears after the retention
window, while live and never-expiring links stay.

### Step 8 — Copy and empty states

- [ ] Update dashboard copy that still implies links are created from Jellyfin.
- [ ] Empty states per tab and per filter.
- [ ] Make sure errors and disabled states are understandable.
- [ ] Selection bar and checkboxes work on phone widths.

### Step 9 — Docs and verification

- [ ] Update `SPEC.md` to match what was built.
- [ ] Update `README.md` admin section with the new flow and dev workflow.
- [ ] Run `go build ./...`, `go test ./...`, and `cd web && npm run build`.
- [ ] Manual checklist: create movie share, create series share, password link,
      expiry, play limit, revoke one, edit one, mass revoke, mass delete,
      filter switching, retention cleanup, mobile layout.
- [ ] Commit and push to `origin main`.

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
