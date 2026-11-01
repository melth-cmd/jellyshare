# Spec: Create shares from the admin page

Status: draft
Last updated: 2026-11-01

## Goal

Let the admin create share links directly from `/admin`, without the Jellyfin
companion plugin. This is a personal-use replacement for the plugin's create
flow.

## Non-goals

- Building or maintaining a Jellyfin plugin.
- Changing the public viewer experience.
- Multi-tenant administration; one admin key is enough.

## Current state

- The backend already has `POST /api/admin/shares` (`internal/handlers/admin.go`)
  which creates a share from a Jellyfin item id and user id.
- `/admin` can list, inspect, and revoke shares, but has no create form.
- `UpdateShare` exists in the handler but is not routed in `cmd/server/main.go`.

## User flow

1. Admin opens `/admin`, already authenticated with the backend key.
2. Admin opens a "New share" panel or dialog.
3. Admin searches or browses the Jellyfin library and picks a film, episode,
   season, or series.
4. Admin sets the link terms: expiry, play limit, concurrent viewers, quality
   cap, password, and whether downloads are allowed.
5. Admin creates the share and gets the public URL, with a copy button.

## Requirements

### Item selection

- Search Jellyfin by title; show poster, title, type, and year.
- Support movies, episodes, seasons, and series.
- The selected item supplies `jellyfinItemId`; the owning user supplies
  `jellyfinUserId`.
- The backend needs read access to the Jellyfin library from the admin API.

### Form fields

Match `models.CreateShareRequest`:

- `expiresInMinutes` or `neverExpires`
- `maxTotalPlays` (optional)
- `maxConcurrentViewers` (optional)
- `maxVideoHeight` (optional)
- `maxVideoBitrate` (optional)
- `password` (optional)
- `allowDownload` (default on)

### After creation

- Show the public link and copy it.
- The new share appears in the existing list without a full page reload.

### Validation and errors

- Required: item and user.
- Show server errors inline; do not lose the form state.
- Never log the admin key or the share password.

## Open questions

- Where does the admin pick the Jellyfin user? Fixed to a configured default, or
  a user selector?
- Should the item search be server-side (new `/api/admin/library` endpoint) or
  reuse an existing Jellyfin search through the admin API?
- Any personal defaults worth prefilling (expiry, play limit, downloads)?
