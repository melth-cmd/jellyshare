# API Reference

## Admin Endpoints

All admin endpoints require the `X-Backend-Key` header with your API key.

### Create Share
```http
POST /api/admin/shares
Content-Type: application/json
X-Backend-Key: your-api-key

{
  "jellyfinItemId": "abc123",
  "jellyfinUserId": "user123",
  "expiresInMinutes": 1440,
  "neverExpires": false,
  "password": "optional-password",
  "maxTotalPlays": 5,
  "maxConcurrentViewers": 2,
  "maxVideoHeight": 720,
  "maxVideoBitrate": 4000000,
  "allowDownload": true
}
```

`expiresInMinutes` defaults to 30 days when omitted. `allowDownload` defaults to
`true` when omitted, so a client that predates it keeps offering downloads;
`JELLYSHARE_ALLOW_DOWNLOADS=false` overrides it for every share. `jellyfinUserId` is
optional: when omitted the backend uses the Jellyfin user it resolved at
startup, which is what the admin page does.

### List Shares
```http
GET /api/admin/shares?limit=50
GET /api/admin/shares?limit=50&beforeCreatedAt=2026-10-18T15:04:05.123456Z&beforeId=8f0c6a4e-…
X-Backend-Key: your-api-key
```

Newest first, a page at a time: `limit` defaults to 50 and is capped at 200. The
next page continues below the last link of the previous one - pass its
`createdAt` and `id` as `beforeCreatedAt` and `beforeId` - so links created or
cleaned up in between do not shift the pages. Add `status=active|inactive|all`
(default `all`) to filter. Active means not revoked, not expired, and plays
remaining; inactive is everything else.

The response carries the page in `shares`, `hasMore` when another page follows,
`status` echoing the filter, and counts over all links rather than the page:

```json
{
  "shares": [],
  "total": 12,
  "live": 8,
  "status": "active",
  "counts": { "active": 8, "inactive": 4, "all": 12 },
  "hasMore": false
}
```

`total` is the number matching the filter; `live` is the active count. With
`?jellyfinUserId=…` it returns that user's links whole.

### Get Share Details
```http
GET /api/admin/shares/{id}
X-Backend-Key: your-api-key
```

### Revoke Share
```http
POST /api/admin/shares/{id}/revoke
X-Backend-Key: your-api-key
```

Add `?jellyfinUserId=...` to scope the call to one owner: the share is only revoked
if it belongs to that user, otherwise the call is refused with 403. Use this to keep
one Jellyfin user from revoking another user's share. Omitting it keeps full admin
reach. The same applies to `GET /api/admin/shares/{id}/analytics`.

### Revoke or Delete Shares in Bulk
```http
POST /api/admin/shares/bulk/revoke
POST /api/admin/shares/bulk/delete
Content-Type: application/json
X-Backend-Key: your-api-key

{ "ids": ["baf9de87-…", "6d019bd4-…"] }
```

Up to 200 unique ids per request. Revoke terminates any active sessions. Delete
only removes links that are already inactive; ids that are still active are
returned in `skipped` and left untouched.

```json
{ "revoked": ["baf9de87-…"], "skipped": [] }
{ "deleted": ["baf9de87-…"], "skipped": ["6d019bd4-…"] }
```

### Update Share
```http
PATCH /api/admin/shares/{id}
Content-Type: application/json
X-Backend-Key: your-api-key

{
  "maxTotalPlays": 10,
  "maxConcurrentViewers": 2,
  "extendMinutes": 1440,
  "password": "new-password",
  "removePassword": false,
  "allowDownload": false,
  "maxVideoHeight": 720,
  "maxVideoBitrate": 4000000
}
```

All fields are optional; omitted fields stay as they are. `extendMinutes` adds to
the existing expiry and does nothing to a link that never expires. A
`maxVideoHeight` or `maxVideoBitrate` of `0` removes that cap.

### Restore or Duplicate a Share
```http
POST /api/admin/shares/{id}/restore
POST /api/admin/shares/{id}/duplicate
X-Backend-Key: your-api-key
```

Restore clears a revoke only when the link is still usable; an expired or
used-up link returns `409`. Duplicate copies the terms with a new token and a
fresh lifetime, and returns the same shape as Create Share.

### Share Analytics
```http
GET /api/admin/shares/{id}/analytics
X-Backend-Key: your-api-key
```

Returns total views, unique viewers, average watch time and views per day for the
last 30 days.

### Server Stats
```http
GET /api/admin/stats
X-Backend-Key: your-api-key
```

### Library Search and Browse

```http
GET /api/admin/jellyfin/items?q=alien&limit=20
GET /api/admin/jellyfin/items/{id}/poster?maxWidth=240
GET /api/admin/jellyfin/series/{id}/seasons
GET /api/admin/jellyfin/seasons/{id}/episodes
GET /api/admin/jellyfin/series/{id}/episodes
X-Backend-Key: your-api-key
```

`items` searches films and series (recently added when `q` is empty). The series
and season calls feed the create page's collapsible tree; the series-level
episodes call is the fallback for libraries without Season objects. Posters are
proxied with the Jellyfin key kept server-side, so the browser only sends the
admin key.

## Public Endpoints

### Get Share Info
```http
GET /api/public/shares/{token}
```

`downloadBytes` is the size of the file a download hands over. A link that no
longer opens answers `410` with why, since when and which link it was:

```json
{ "error": "share has expired", "reason": "expired", "title": "The Long Walk Home", "endedAt": "2026-10-18T16:58:42Z" }
```

`reason` is `revoked` or `expired`; a revoke is named over an expiry. A season link
also carries `seriesName`. Every public endpoint answers a dead link this way.

### Validate Password
```http
POST /api/public/shares/{token}/password
Content-Type: application/json

{
  "password": "share-password"
}
```

### Start Playback
```http
POST /api/public/shares/{token}/play
```

Optional query parameters, all validated against the item and then pinned to the
session - the stream proxy never reads them from a later request:
`videoCodecs` (comma-separated list the browser can decode, e.g. `h264,hevc`),
`audioStreamIndex`, `subtitleStreamIndex`, and `audioLanguage` / `subtitleLanguage`,
which take precedence for an episode because the track list was probed on a
different one.

The response carries `subtitleUrl` when a text subtitle was selected.

### List Episodes (Season and Series shares)
```http
GET /api/public/shares/{token}/episodes
```

Each episode carries `sizeBytes`, its file's size; a season ZIP stores the files
uncompressed, so its size is theirs added up.

### Start Episode Playback
```http
POST /api/public/shares/{token}/episodes/{episodeId}/play
```

### Progress
```http
GET /api/public/shares/{token}/progress
```

Where the link was left: `{"last": "main", "items": {"main": {"position": 88, "duration": 150}}}`,
or `{"last": "", "items": {}}` before anything was watched. The film is keyed
`main`, episodes by their id. Like the episode list, a password link answers only
once the password session cookie is set. It is written by the heartbeat and finish
calls, which take `positionSeconds` and `durationSeconds`; a report without a
duration is not stored.

### Download
```http
POST /api/public/shares/{token}/download
POST /api/public/shares/{token}/episodes/{episodeId}/download
POST /api/public/shares/{token}/episodes/download
GET  /api/public/downloads/{ticket}
```

The POSTs - a film or episode share itself, one episode of a season or series,
or every episode as one ZIP - check the share and its password, charge one play,
and answer with a `downloadUrl`. That URL carries a signed ticket valid for two
hours; the GET streams the original file with Range support, so an interrupted
download resumes without being charged again. The ZIP is stored, not
compressed, files episodes by season, and leaves out episodes the library lists
without a file.

Refused with 403 when the share's play limit is spent or downloads are off for
it. A download takes no concurrent-viewer slot.

### Finish Playback
```http
POST /api/public/sessions/{sessionId}/finish
```

### Media, Images and Subtitles
```http
GET /api/public/stream/{sessionId}/{path}
GET /api/public/images/{token}/{type}
GET /api/public/images/{token}/episode/{episodeId}
GET /api/public/subtitles/{sessionId}/{index}.vtt
```

`{type}` is `poster`, `backdrop`, `logo` or `thumb`. `maxWidth` and
`maxHeight` (up to 1920) and `quality` (up to 100) pass through to Jellyfin.
An episode's still is served only for an episode inside the shared season or
series, on a live link that is unlocked if it has a password.

### Heartbeat (keep session alive)
```http
POST /api/public/sessions/{sessionId}/heartbeat
Content-Type: application/json

{
  "positionSeconds": 120
}
```
