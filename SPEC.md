# Spec: Create and manage shares from the admin page

Status: implemented
Last updated: 2026-11-01

## Goal

Let the admin create and manage share links directly from `/admin`, for personal
use, without a separate Jellyfin-side integration.

## Non-goals

- A multi-tenant administration surface; one admin key is enough.
- Changing the public viewer experience.
- Bulk editing. Editing is per share.

## Implemented decisions

- Admin page has two tabs: **Create** and **Shares**.
- The Shares tab filters with a segmented control: **Active / Inactive / All**,
  defaulting to Active.
- New shares are owned by the first Jellyfin user the backend resolves at
  startup. The API still accepts an explicit `jellyfinUserId`.
- Search returns films and series only. Series expand into seasons, and seasons
  into episodes.
- Mass actions are revoke and delete. Individual actions are copy, edit, and
  revoke.
- Stale links are deleted after a configurable retention window.

## User flow

1. Admin opens `/admin`, already authenticated with the backend key.
2. The top bar shows the wordmark, Create and Shares tabs, refresh, and sign out.
3. On **Create**, the admin searches for a film or series. A series expands to
   its seasons; a season expands to its episodes.
4. The admin chooses the whole series, one season, one episode, or a film.
5. The admin sets the link terms: expiry, play limit, concurrent viewers, quality
   caps, password, and whether downloads are allowed.
6. Creating shows the public URL with a copy button, Create another, and View in
   Shares.
7. On **Shares**, the admin filters Active, Inactive, or All, inspects links,
   edits or revokes one, or selects several for a mass action.

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
- Each row has Copy link, Edit, and Revoke. Edit is also available from the
  detail sheet.
- Edit fields: add days to expiry, play limit, concurrent viewers, set password,
  remove password. Blank means unchanged; never-expiring links cannot be
  extended.

## Retention

- Links are deleted once they have been expired or revoked for longer than
  `JFSHARE_SHARE_RETENTION_DAYS` (default 365; `0` disables cleanup).
- Live links and links that never expire are never deleted.

## Validation and errors

- Required: an item.
- Server errors show inline and do not discard form state.
- Mass actions report changed and skipped counts.
- The admin key and share passwords are never logged.

## Out of scope for now

- Editing quality caps or download permission after creation.
- Selecting multiple seasons without the whole series, or multiple episodes
  without the whole season.
