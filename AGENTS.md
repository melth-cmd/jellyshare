# AGENTS.md

Instructions for any LLM or coding agent working in this repository.

## What this repo is

A private, personal build of a Jellyfin link-sharing backend. It serves:

- a Go HTTP API (`cmd/server`, `internal/`),
- a Svelte admin UI at `/admin` (`web/src/components/Admin*.svelte`),
- public share pages for viewers who have no Jellyfin account.

This is not a public fork. Do not add remotes, links, badges, plugin
integrations, or references to any upstream project or its companion plugin.
There is no plugin: share creation belongs in the admin page.

## Commands

Run these from the repo root.

```bash
# Backend
go build ./...
go test ./...

# Frontend (Svelte 4 + Vite)
cd web && npm install && npm run build

# Dev stack (Docker, hot reload)
make dev-up          # backend :8098, frontend :5173
make dev-logs
make dev-down

# Refresh dev database from production
make dev-db-sync

# Local demo with a throwaway Jellyfin
make demo-up
```

After backend changes run `go build ./...` and `go test ./...`. After UI
changes run the frontend build. Mention in your summary which checks you ran.

## Layout

- `cmd/server/main.go` — router, middleware, all route definitions.
- `internal/handlers/` — HTTP handlers (`admin.go` is the admin API).
- `internal/database/` — SQL via `sqlx`.
- `internal/models/` — data structs and JSON wire shapes.
- `internal/jellyfin/` — Jellyfin API client.
- `internal/proxy/`, `internal/download/` — streaming and download paths.
- `internal/middleware/` — auth, rate limiting, logging, sessions.
- `migrations/` — numbered up/down pairs; never edit an applied migration.
- `web/src/components/` — Svelte single-file components.
- `web/src/tokens.css` — shared design tokens.

## Conventions

- Go 1.23, standard library plus chi, sqlx, and PostgreSQL. Keep handlers thin:
  decode and validate, call the database or Jellyfin, write JSON.
- Use `writeError(w, status, message)` for error responses.
- The JSON field names in `internal/models` are a public API for the admin UI and
  public share pages. Do not rename them casually.
- New migrations use the next number, e.g. `000013_...up.sql` with a matching
  `.down.sql`.
- Frontend is Svelte 4, plain JavaScript, no TypeScript. Match the existing
  component style and reuse `tokens.css` variables.
- Admin endpoints are under `/api/admin` and require `X-Backend-Key`.

## Rules

- Keep changes focused and incremental; do not refactor unrelated code.
- Ask before adding a new dependency, changing the database schema in a
  destructive way, or altering public share behavior.
- Never commit `.env` or real credentials. `.env` is gitignored on purpose.
- Do not weaken authentication, rate limiting, or the share token checks.
- Update `PROGRESS.md` after finishing a meaningful chunk of work, and keep
  `SPEC.md` in sync when requirements change. Follow the step order in
  `PROGRESS.md` unless the user asks to jump ahead.
- Commit with clear messages and push to `origin main`. Do not force-push.
