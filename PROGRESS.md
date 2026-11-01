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

## Iterating without rebuilding the compose stack

The repo already has a hot-reload stack: `docker-compose.dev.yml` runs the Go
server under `air` and the Svelte app under Vite, both bind-mounted from the
working tree. After Step 0, the loop is:

```bash
make dev-up                  # once per session
scripts/sync-dev-db.sh       # optional, refresh data from production
```

- Open `http://192.168.1.133:5173/admin` for the UI (Vite HMR).
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

Status: done. Dev backend is on `http://192.168.1.133:8098`, Vite on
`http://192.168.1.133:5173`, and the sync script copies production data. Go changes
rebuild via `air`; Svelte changes are served by Vite immediately.

### Step 1 — Backend: Jellyfin library search

- [ ] Add `SearchItems` to `internal/jellyfin/client.go`, using the Jellyfin
      `/Items` endpoint with `recursive=true` and
      `includeItemTypes=Movie,Series,Season,Episode`.
- [ ] Return the fields the picker needs: id, name, type, year, series/season
      names, overview, runtime, poster URL.
- [ ] Add `GET /api/admin/jellyfin/items?q=...&limit=...` under the existing
      admin auth middleware.
- [ ] Add handler tests with an `httptest` Jellyfin stub.

Done when: a search for part of a real title returns usable rows, and an empty
query returns a sensible default (recent items or an empty prompt).

### Step 2 — Admin shell and tabs

- [ ] Split `AdminDashboard.svelte` into a shell plus a shares panel and a create
      panel, without rewriting the existing list/detail behavior.
- [ ] Replace the current top pill with a wider bar: wordmark on the left, tab
      buttons on the right, then refresh and sign out.
- [ ] Style the tabs to match the existing design tokens.
- [ ] Keep the selected tab in session state; do not lose it on refresh.
- [ ] On mobile, keep the bar usable (tabs wrap or collapse; counts may hide as
      they already do).

Done when: the admin can switch between Create and Shares freely and the bar
looks right on desktop and phone.

### Step 3 — Shares tab: Active / Inactive / All

- [ ] Add a `status=active|inactive|all` query parameter to
      `GET /api/admin/shares`, applied in `GetSharesPage` and `CountShares`.
- [ ] Treat inactive as revoked, expired, or play limit spent.
- [ ] Return counts for the current filter so the header is accurate.
- [ ] Add the segmented filter to the Shares tab and reset pagination when it
      changes.
- [ ] Keep Revoke and the detail sheet working in every filter.

Done when: selecting Inactive shows only dead links, All shows everything, and
the Active count matches the real live links.

### Step 4 — Create share form

- [ ] Item picker: debounced search, poster and title rows, select one item.
- [ ] Show the selected item with its poster, title, type, year, and runtime.
- [ ] Expiry: preset chips (24 h, 7 d, 30 d, Never) plus a custom option.
- [ ] Play limit and concurrent viewer fields, optional.
- [ ] Quality cap: Auto and common heights; optional bitrate cap.
- [ ] Password field, optional.
- [ ] Downloads toggle, default on.
- [ ] Inline validation; keep form state on server errors.
- [ ] Submit to `POST /api/admin/shares`; on success show the public URL with a
      copy button, and offer "Create another".

Done when: a share created from the admin page opens and plays through the
existing public share page.

### Step 5 — Copy and empty states

- [ ] Update dashboard copy that still implies links are created from Jellyfin.
- [ ] Empty states per tab and per filter.
- [ ] Make sure errors and disabled states are understandable.

### Step 6 — Docs and verification

- [ ] Update `SPEC.md` to match what was built.
- [ ] Update `README.md` admin section with the new flow and dev workflow.
- [ ] Run `go build ./...`, `go test ./...`, and `cd web && npm run build`.
- [ ] Manual checklist: create movie share, create series share, password link,
      expiry, play limit, revoke, filter switching, mobile layout.
- [ ] Commit and push to `origin main`.

## Testing notes

- Backend: `go test ./...`
- Frontend: `cd web && npm run build`
- Admin API smoke test (after Step 1):

  ```bash
  curl -s -H "X-Backend-Key: $BACKEND_API_KEY" \
    "http://192.168.1.133:8098/api/admin/jellyfin/items?q=alien&limit=5"
  ```

- Dev UI: `http://192.168.1.133:5173/admin`
- Production UI: `http://192.168.1.133:8097/admin`

## Log

### 2026-11-01

- Detached repo from upstream, rebranded, removed plugin references.
- Added agent instructions, spec, and this plan.
- Step 0 done: dev stack on 8098, Vite proxy configurable, `make dev-db-sync`
  copies production data. Fixed the dev image build (Node 20 + npm latest was
  incompatible) and `air` building with VCS stamping inside the container.
- Verified hot reload for Go and Svelte and synced the dev database.
