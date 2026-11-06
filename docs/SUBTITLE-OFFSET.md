# Subtitle offset in the link player

Status: implemented
Last updated: 2026-11-06

## Problem

Sidecar WebVTT subtitles can drift from the HLS timeline after a seek or a
transcode. The drift is usually a few seconds and can be early or late. The
viewer has no way to correct it in the link player.

## Goal

Add a viewer-facing control to the link player that shifts the sidecar subtitle
cues by a configurable number of seconds, so a viewer can line them up again
without the admin re-creating the link.

## Non-goals

- Fixing the root cause (keyframe spacing / HLS timeline / transcode start).
- Burned-in or image subtitles (PGS, VOBSUB); those cannot be shifted.
- A server-side per-session offset, admin controls, or a global default.
- Casting. The cast path drops subtitles today, so there is nothing to offset.

## Approach

Shift the cues in the browser. The sidecar is a normal `<track>` element, so
once it loads, `TextTrack.cues` contains `VTTCue` objects whose `startTime` and
`endTime` are writable. Adding the same constant to every cue moves the whole
subtitle track without re-fetching it.

A constant shift preserves cue order, so the browser's active-cue lookup stays
valid. No blob URL, no re-parse, no backend change, and no network round trip
when the viewer nudges the value.

Why not rewrite the VTT on the server: it would need a request per adjustment,
a cache key that includes the offset, and a VTT timing parser in Go. It is a
reasonable fallback if cue mutation turns out to be unreliable in a target
browser, but it is more moving parts for a bandaid.

## User experience

- A subtitle-timing button appears in the player header tools only when
  `playbackData.subtitleUrl` is set. It shows the current offset when non-zero.
- The button opens a small popover:
  - title: "Subtitle timing"
  - `−0.2s` and `+0.2s` buttons
  - current value, e.g. `+1.4s`
  - a range slider for coarse adjustment
  - `Reset`
  - a one-line hint: positive values show subtitles later.
- Range is clamped to ±30 seconds. Step is 0.2 seconds.
- Keyboard: `[` moves earlier, `]` moves later, `\` resets. Ignore the keys
  while a form control has focus.
- The current value is shown in the popover only; there is no separate status
  line over the picture.
- The value is stored in `localStorage` under
  `jellyshare:subtitle-offset:<offsetKey>`, where `offsetKey` is
  `<share token>:<item key>:<subtitle index>`. It survives closing and
  reopening the player, and each subtitle track keeps its own value.
- If storage is blocked, the offset still works for the current page.

## Files

- `web/src/components/Player.svelte` — offset state, control, cue shifting,
  keyboard shortcuts, styles.
- `web/src/components/ShareView.svelte` — pass an `offsetKey` built from the
  share token, the current item, and the selected subtitle index.
- `web/src/subtitleOffset.js` (new) — small pure helpers: clamp, format, and
  shift cues while remembering each cue's original times.
- `web/src/i18n/en.js`, `web/src/i18n/th.js` — new player strings.
- `PROGRESS.md` — log the change after implementation.
- `SPEC.md` — note that viewers can adjust subtitle timing on a share.

## Implementation steps

1. Add `web/src/subtitleOffset.js`:
   - `clampOffset(value)` → number in `[-30, 30]`, rounded to 0.2.
   - `formatOffset(value)` → `+1.4s`, `0.0s`, `−1.4s`.
   - `shiftCues(cues, offset, baseTimes)` → sets each cue's `startTime` and
     `endTime` from its stored original, clamping at 0. `baseTimes` is a
     `WeakMap` so a reloaded track with new cue objects starts clean.
2. In `Player.svelte`, add `export let subtitleOffsetKey = null;` and state for
   the offset and popover visibility.
3. On mount, read the stored offset for `subtitleOffsetKey`; on change, write it
   back inside `try/catch`.
4. Add `subtitleTrack()` (the `kind === 'subtitles'` track) and
   `applySubtitleOffset()`. Call it from the track's `load` handler, from
   `showSubtitles()`, and whenever the offset changes.
5. Add the header button and popover. Stop click propagation so a click in the
   popover does not reach the video's double-click-to-fullscreen handler.
6. Add `[`, `]`, and `\` to `handleKeydown`, skipping form controls.
7. In `ShareView.svelte`, pass
   `subtitleOffsetKey={`${token}:${currentEpisodeId || MAIN}:${selectedSubtitleIndex}`}`.
8. Add English and Thai strings for the button, popover, hint, reset, and
   status message.
9. Run the frontend build and test manually.

## Edge cases

- `track.cues` is null until the track loads; apply on `load` and retry then.
- A track reload creates new cue objects; the `WeakMap` keeps their originals
  separate from the old ones.
- A cue shifted before 0 is clamped to 0. If its end is also at or before 0 it
  never becomes active, which is the desired result.
- A constant shift keeps the cue list in non-decreasing start order, including
  after clamping.
- Autoplay to the next episode creates a new player and a new `offsetKey`, so
  the offset does not leak between episodes.
- Changing the subtitle track recreates the session and the player, so the new
  track gets its own stored offset.
- Burned-in subtitles have no `subtitleUrl`, so the control stays hidden.
- Safari's native HLS path uses the same `TextTrack` API.

## Verification

- `cd web && npm run build`.
- Manual, on the dev stack:
  - open a link with sidecar subtitles and adjust the offset both ways;
  - seek, pause, and confirm the cues stay shifted;
  - toggle subtitles off and on in the native menu;
  - close and reopen the player, and reload the page, to confirm persistence;
  - check a burned-in subtitle link: the control is absent;
  - check Safari/iOS if available.
- No Go changes are planned, so `go build ./...` and `go test ./...` are not
  required unless the fallback is used.

## Risks

- Cue mutation is part of the spec but browser behavior can vary. If a target
  browser refuses it, fall back to the server-side rewrite below.
- The popover must sit above the native control bar in fullscreen; test on a
  phone and on desktop.
- The offset is per browser, not per link. A viewer on a second device starts
  at zero. That is acceptable for a bandaid; the alternative is a server-side
  session field.

## Fallback: server-side VTT rewrite

If client-side cue mutation is unreliable:

1. Add an `offset` query parameter to `ServeSubtitle` in
   `internal/proxy/stream.go`.
2. Parse it as a float, clamp to ±60, and shift the timing lines of the VTT
   body before writing it.
3. Keep the response private and make the offset part of the cache key (it is
   already part of the URL).
4. In the player, set the track `src` to `subtitleUrl + '?offset=' + value` and
   re-enable the track after `load`.

This costs a request per adjustment and a small VTT parser, but it avoids
relying on cue mutability.

## Resolved decisions

- Step size: 0.2 seconds, for both the buttons and the slider.
- Scope: per share + item + track, so a correction does not leak to another
  link or another subtitle language.
- Visibility: every viewer sees the control when a sidecar exists, not only
  with `?debug=1`.
- Feedback: the current value in the popover is enough; there is no separate
  status line over the picture.
