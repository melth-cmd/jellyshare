# Spec: Create and manage shares from the admin page

Status: implemented
Last updated: 2026-11-02

## Goal

Let the admin create and manage share links directly from `/admin`, for personal
use, without a separate Jellyfin-side integration.

## Non-goals

- A multi-tenant administration surface; one admin key is enough.
- Redesigning the public viewer experience; the admin page only controls what a
  link allows.
- Bulk editing. Editing is per share.

## Implemented decisions

- Admin page has three tabs: **Create**, **Shares** and **Analytics**.
- The Shares tab filters with a segmented control: **Active / Inactive / All**,
  defaulting to Active.
- New shares are owned by the first Jellyfin user the backend resolves at
  startup. The API still accepts an explicit `jellyfinUserId`.
- Search returns films and series only. Series expand into seasons, and seasons
  into episodes.
- Mass actions are revoke and delete. Individual actions are copy, edit,
  revoke, restore and duplicate.
- Stale links are deleted after a configurable retention window.
- New share tokens are a readable slug plus a 12-hex suffix, for example
  `mr-mrs-smith-s01-ep-01-2bf3cd802f25`. Old random tokens keep working.

## User flow

1. Admin opens `/admin`, already authenticated with the backend key.
2. The top bar shows the wordmark, Create, Shares and Analytics tabs, refresh,
   and sign out.
3. On **Create**, the admin searches for a film or series. A series expands to
   its seasons; a season expands to its episodes.
4. The admin chooses the whole series, one season, one episode, or a film.
5. The admin sets the link terms: expiry, play limit, concurrent viewers, quality
   caps, password, and whether downloads are allowed.
6. Creating shows the public URL with a copy button, Create another, and View in
   Shares.
7. On **Shares**, the admin filters Active, Inactive, or All, inspects links,
   edits, revokes, restores or duplicates one, or selects several for a mass
   action.
8. On **Analytics**, the admin reviews totals and per-link usage.

## Item selection

- Search returns top-level films and series, never a flat mix of episodes.
- Each series row expands to its seasons; each season expands to its episodes.
- Each level has its own share action: **Share series**, **Share season**, or
  click an episode.
- Seasons and episodes load lazily and are cached until Refresh; several series
  or seasons may stay open at once.
- If a series has no Season children, its episodes appear as one group.
- The selected item supplies `jellyfinItemId`; the backend fills the owning
  Jellyfin user automatically.

## Create form

Matches `models.CreateShareRequest`:

- `expiresInMinutes`, or `neverExpires`
- `maxTotalPlays` (optional)
- `maxConcurrentViewers` (optional)
- `maxVideoHeight` (optional)
- `maxVideoBitrate` (optional)
- `password` (optional)
- `allowDownload` (default on)

## Shares tab

- Segmented filter: Active / Inactive / All, default Active.
- Inactive means revoked, expired, or out of plays.
- Counts reflect the selected filter.
- Revoke and detail views work in every filter.
- Rows have checkboxes; shift-click selects a range, and select-all covers the
  loaded page.
- The mass-action bar offers revoke and delete. Delete is only allowed for
  inactive links; active ids are skipped server-side.
- Each row has Copy link, Edit, and Revoke; revoked rows show Restore. The
  detail sheet also offers Duplicate and Restore.
- Edit fields: add days to expiry, play limit, concurrent viewers, maximum
  picture, maximum bitrate, download permission, set password, remove password.
  Blank means unchanged; never-expiring links cannot be extended. A quality cap
  of Auto removes it.
- Duplicate copies the terms with a new token and a fresh lifetime.
- Restore clears a revoke only while the link is still usable; expired or
  used-up links return `409`.

## Viewer subtitle timing

- When a share plays a text subtitle as a WebVTT sidecar, the player offers a
  subtitle timing control. It shifts the cues by ±30 seconds in 0.2-second
  steps.
- The correction is client-side: the video stream and the saved position are
  untouched. The value is stored in the viewer's browser per share, item and
  subtitle track, so it survives closing and reopening the player.
- Keyboard shortcuts are `[` (earlier), `]` (later) and `\` (reset).
- Image subtitles burned into the picture have no sidecar, so they have no
  timing control.

## Retention

- Links are deleted once they have been expired or revoked for longer than
  `JELLYSHARE_SHARE_RETENTION_DAYS` (default 365; `0` disables cleanup).
- Live links and links that never expire are never deleted.

## Validation and errors

- Required: an item.
- Server errors show inline and do not discard form state.
- Mass actions report changed and skipped counts.
- The admin key and share passwords are never logged.

## Analytics tab

- **Overview**: total and active links, plays, views, unique viewers, average
  watch time, and a 30-day views chart.
- **Links**: per-link views, unique viewers, average watch time and plays.
- Backed by `GET /api/admin/analytics`, which aggregates the existing audit and
  session data in one set of queries.

## Resolved deferrals

- Editing quality caps and download permission after creation is supported
  (#10).
- Selecting several seasons or episodes in one batch was closed as not planned
  (#12); the picker stays single-item.
