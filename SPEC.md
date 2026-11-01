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

## Decisions

- Admin page has two tabs: **Create** and **Shares**.
- The Shares tab filters with a segmented control: **Active / Inactive / All**,
  defaulting to Active.
- New shares are owned by the first Jellyfin user the backend resolves at
  startup. No user picker.
- The create form exposes the same terms the API accepts: expiry, never expires,
  play limit, concurrent viewers, quality cap, password, downloads.

## Current state

- The backend already has `POST /api/admin/shares` (`internal/handlers/admin.go`)
  which creates a share from a Jellyfin item id and user id.
- `/admin` can list, inspect, and revoke shares, but has no create form.
- `UpdateShare` exists in the handler but is not routed in `cmd/server/main.go`.

## User flow

1. Admin opens `/admin`, already authenticated with the backend key.
2. The top bar shows the wordmark, Create and Shares tabs, refresh, and sign out.
3. On **Create**, the admin searches the Jellyfin library and picks a film,
   episode, season, or series.
4. Admin sets the link terms: expiry, play limit, concurrent viewers, quality
   cap, password, and whether downloads are allowed.
5. Admin creates the share and gets the public URL with a copy button.
6. On **Shares**, the admin sees the new link and can filter Active, Inactive, or
   All.

## Requirements

### Item selection

- Search returns top-level films and series only, never a flat mix of episodes.
- Each series row expands to its seasons; each season expands to its episodes.
- Each level has its own share action: **Share series**, **Share season**, or
  click an episode.
- Seasons and episodes load lazily and are cached until Refresh; several series
  or seasons may stay open at once.
- If a series has no Season children, show its episodes as one group.
- The selected item supplies `jellyfinItemId`; the backend fills the owning
  Jellyfin user automatically.
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

### Shares tab

- Segmented filter: Active / Inactive / All, default Active.
- Inactive means revoked, expired, or play limit spent.
- Counts reflect the selected filter.
- Revoke and detail views work in every filter.
- Rows can be selected with checkboxes; shift-click selects a range.
- A mass-action bar at the top of the list offers revoke and delete for the
  selection. There is no bulk edit. Delete is only allowed for inactive links.
- Each row has Copy link, Edit, and Revoke buttons. Edit opens a dialog for
  expiry, play limit, concurrent viewers, and password.

### Retention

- Links are deleted once they have been expired or revoked for longer than
  `JFSHARE_SHARE_RETENTION_DAYS` (default 365).
- Live links and links that never expire are never deleted.

### After creation

- Show the public link and copy it.
- The new share appears in the Shares tab without a full page reload.

### Validation and errors

- Required: an item.
- Show server errors inline; do not lose the form state.
- Never log the admin key or the share password.

## Open questions

- Any personal defaults worth prefilling (expiry, play limit, downloads)?
- Should the quality cap offer bitrate presets or only heights?
