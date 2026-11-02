# Development

## Prerequisites

- Go 1.27+ (see `go.mod`)
- Node.js 20+ (the production image builds the frontend with `node:20`)
- Docker and Docker Compose

## Checks

CI runs on every push and pull request, plus weekly for new vulnerabilities:
`go vet`, `go build`, `go test`, `govulncheck ./...`, the frontend build, and
`npm audit --audit-level=high`. Dependabot opens weekly updates for Go modules,
npm packages, and GitHub Actions.

## Setup

```bash
# Start the hot-reload dev stack
make dev-up

# Backend:  http://<host-ip>:8098  (port 8097 is left for production)
# Frontend: http://<host-ip>:5173
```

The dev stack reads `JELLYFIN_URL` and `JELLYFIN_API_KEY` from `.env`, so it talks
to the real Jellyfin library. Go changes rebuild through `air`; Svelte changes
update through Vite. Production can keep running on 8097 at the same time.

```bash
# Optional: replace dev data with a dump of the production database
make dev-db-sync
```

The dev backend binds to all interfaces so another machine can open the UI; set
`DEV_HOST_IP` and `DEV_PUBLIC_BASE_URL` to override the defaults.

## Local demo

```bash
make demo-up      # build this checkout and start it next to a test Jellyfin
make demo-down    # stop it, keep its data
make demo-reset   # stop it and delete everything it created
```

`scripts/local-demo.sh` generates a few short test clips with metadata, runs
Jellyfin's setup wizard, creates an API key and starts the app on
`http://localhost:8097` (admin key `demo-admin-key`) next to Jellyfin on
`http://localhost:8096` (user `demo` / `demo`). It needs Docker, ffmpeg, curl
and jq, and keeps everything in `.local/` and `.env.local`, both ignored by git.
The demo's libraries look nothing up online, so media dropped into
`.local/media/movies` or `.local/media/shows` needs NFO files and local artwork
(`poster.jpg`, `backdrop.jpg`); it shows up after a library refresh. That is how the demo media is made.

## Working on the interface

[`design.md`](design.md) is the locked design system: palette, type, spacing,
motion, component voice, and the reasoning behind each. Read it before changing
the UI, and amend it rather than overriding it locally — a page that drifts from
it is the thing the system exists to prevent.

Two rules that are easy to break by accident:

- **Every colour and font comes from `web/src/tokens.css`.** Components reference
  tokens by name (`var(--color-accent)`); an inline hex or a bare `font-family`
  is how a design system erodes. If a value does not exist yet, add it to
  `tokens.css` first.
- **Buttons filled with `--color-accent` keep their `outline-offset`.** The focus
  ring only reaches 1.3:1 against the accent itself, so the offset is what puts it
  on the page background where it reads at 9.1:1. Removing it makes the control
  unusable by keyboard.

The four faces (Instrument Serif, Geist, Geist Mono, IBM Plex Sans Thai) load
from Google Fonts. An instance with no outbound internet access falls back to the
system stack — the layout holds, but the typography is not what was designed.
Self-host the fonts if that matters to you.

## Project Structure

```
.
├── cmd/server/          # Application entrypoint
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database operations & migrations
│   ├── download/        # Download tickets and file naming
│   ├── handlers/        # HTTP handlers (admin & public)
│   ├── jellyfin/        # Jellyfin API client
│   ├── middleware/      # Auth, rate limiting, sessions, request logging
│   ├── models/          # Data models
│   ├── proxy/           # Stream, image & subtitle proxy
│   └── slug/            # Readable share URL slugs
├── migrations/          # SQL migrations
├── scripts/             # Backup, restore, dev sync and the local demo
├── web/
│   └── src/
│       ├── i18n/        # English and Thai dictionaries
│       ├── tokens.css   # The design system's only source of colour/type/spacing
│       └── components/  # Svelte components
├── design.md            # Locked design system — read before touching the UI
└── docker-compose.yml   # Production compose
```

## Building

```bash
# Build production Docker image
docker build -t jellyshare .

# Build frontend only
cd web && npm run build
```
