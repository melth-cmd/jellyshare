# Localization plan — Thai for the public share page

Status: planning. Nothing is translated yet.

## Scope

Translate the viewer-facing experience only:

- `/s/{token}` share page
- the player that replaces it
- the dead-link / error page
- cast and playback messages shown to the viewer

The admin page stays English. The brand name **Jellyshare** stays as it is.

## Not translated

These come from Jellyfin or are not language:

- **Media metadata**: title, series name, overview, tagline, genres, studios,
  directors, actors, roles, ratings, runtime, video quality, episode names and
  episode overviews. Jellyfin serves whatever language the library holds; the
  share page cannot translate it.
- **Track display titles** from Jellyfin. The page builds its own labels from
  the language tag, and those labels will be localized.
- **Brand and technical units**: Jellyshare, KB/MB/GB/TB, h/m, resolutions.
- **Console logs and code comments.**

## Where the strings live

| Area | File | Approx. lines | Strings |
|---|---|---|---|
| Boot label | `web/index.html` | 70 | 1 |
| App shell | `web/src/App.svelte` | 60–80 | 1 |
| Error page | `web/src/components/ErrorView.svelte` | 1–80 | ~20 |
| Share page script | `web/src/components/ShareView.svelte` | 200–1100 | ~70 |
| Share page markup | `web/src/components/ShareView.svelte` | 1100–1700 | ~50 |
| Player script | `web/src/components/Player.svelte` | 1–560 | ~20 |
| Player markup | `web/src/components/Player.svelte` | 560–620 | ~15 |
| Cast helpers | `web/src/cast.js` | 50–70, 140–210, 290–344 | ~12 |
| Track language names | `web/src/cast.js` | 300–310 | `Intl.DisplayNames` |
| Dates and times | `web/src/components/ShareView.svelte`, `ErrorView.svelte` | — | `Intl.DateTimeFormat` |

## String inventory

### 1. Shell and errors

- Boot: “Opening the link”.
- Error titles and bodies: Not found, No longer available, Access denied,
  Couldn’t reach the server, Couldn’t load the link.
- Gone-link variants: Switched off, This link has expired, “The link to …”,
  “This link”, “was switched off by …”, “expired …”.
- Footer: “Shared via”.
- Page titles: `{title} · Jellyshare`, `{error} · Jellyshare`.

### 2. Password gate and terms

- Password label, Show/Hide, “Whoever shared this link set a password for it.”,
  Checking, Unlock.
- Password errors: enter the password, wrong password, too many tries, couldn’t
  check, offline.
- Plays: “No plays left”, “One play”, “Last of N plays”, “N of M plays left”,
  “Every play on this link has been used.”, “This link has a single play.”,
  “This is the last play.”, “Watching here or on a TV, or downloading, uses it.”
- Expiry: “expires in N days/hours/minutes”, “no expiry”, “Until {date}”.
- Viewer limits: one person, up to N people, everyone is watching.
- Counting note for series.
- Resume notes: “Where anyone with this link last stopped…”, “Next after the
  last episode anyone with this link finished.”
- Spent link: “The one play on this link has been used.”, “All N plays…”.

### 3. Playback controls and downloads

- Play, Resume, Starting, Restart {episode}, Start from the beginning.
- Download, Download all, Preparing, Download it, Don’t download.
- Download cost sentences, including “the film”, “every episode”, “the link’s
  only play”, “the last play”, “one of the N plays left”.
- Cast: Cast to a TV, Connecting, Stop casting, Stop, Playing on {device},
  Connected to {device}, “Keep this page open while it plays.”, “Pick an
  episode below to play it there.”
- Tracks: Audio, Subtitles, None, Mono, Stereo, 5.1, 7.1, N channels, forced,
  Track, Subtitle.
- Watched, episode list heading, Loading episodes, Try again, “This link
  doesn’t include any episodes yet.”
- Episode labels: “Episode N”, “Season N, episode N”.
- Series play labels: “Play Episode N”, “Continue with Episode N”, “Resume
  Episode N”.
- Casting without subtitles note.

### 4. Player chrome

- Back (Esc), Back to the share, Fullscreen (F), Exit fullscreen (F).
- Playback stopped, Back to the share.
- Next episode, Play now, Cancel, “Don’t play the next episode”.
- Recovering: “Connection interrupted, reconnecting”, “Recovering the stream”.
- Playback errors: stream didn’t arrive, browser can’t play, stream stopped,
  last play used, open in Chrome/Edge/Safari, go back and start again.
- Session-ended reasons: switched off, expired, viewing ended.

### 5. Cast messages

- “To watch on a TV, start playing and choose AirPlay in the player.”
- “To watch on a TV, open this link in Chrome or Edge.”
- “The TV couldn’t be reached.” / “Check it’s on and on the same Wi-Fi…”
- “The TV couldn’t play the video.” / “Try again, or watch here instead.”
- “Casting didn’t work.” / “Try again, or watch here instead.”
- “Shared media” and “Subtitles” fallbacks.
- Track language names via `Intl.DisplayNames`.

### 6. Metadata labels and formatting

- Direction, Studio, Starring, Show all N, Show fewer.
- Audience N/10, Critics N%.
- Duration: “1h 23m”, “23m”.
- Clock: “1:00 of 2:00”.
- Dates: absolute date/time, “on {date}”, “in N days”.
- Bytes: KB/MB/GB/TB.

## Batch workflow

Each batch is a separate pass:

1. I list the English source and a proposed Thai translation, key by key.
2. You review the Thai and mark anything that should change.
3. I apply the approved strings, wire the keys, and build.
4. We check the page in the browser, then move to the next batch.

No batch is applied until you have reviewed it. The English dictionary is the
source of truth; the Thai dictionary is filled in batch by batch.

## Batch order

1. **Shell, errors and the pill** — boot label, error page, gone-link variants,
   footer, page titles, the EN/TH pill.
2. **Password gate and link terms** — password form and errors, plays and
   expiry headlines, warnings, viewer limits, counting note, resume notes,
   spent link.
3. **Playback controls and downloads** — Play/Resume/Starting, download buttons
   and confirms, cast buttons and status, track selectors, watched, episode
   list and labels.
4. **Player chrome** — back, fullscreen, playback stopped, next-episode card,
   recovery messages, playback errors, session-ended reasons.
5. **Cast messages and language names** — TV hints, cast errors, “Shared media”,
   “Subtitles”, and `Intl.DisplayNames` for track languages.
6. **Metadata labels and formatting** — Direction/Studio/Starring, Show all,
   Audience/Critics, duration, clock, dates and bytes.

## Proposed implementation

1. **Dictionary files** — `web/src/i18n/en.js` and `web/src/i18n/th.js`, flat
   keys such as `share.play`, `error.404.title`, `terms.playsLeft`.
2. **Runtime** — `web/src/i18n/index.js` with a `locale` store, a `t()` store
   function, plural handling, and date/number helpers. No new dependency.
3. **Locale pill** — a small `EN / TH` pill in the public page nav and the error
   page nav. The choice is stored in `localStorage` and applied to
   `<html lang>`. The player inherits the choice; a pill can also sit in the
   player bar if wanted.
4. **Default** — Thai, per the issue, with English one tap away.
5. **Dates** — `th-TH` for Thai and `en-GB` for English, with the calendar
   chosen explicitly.
6. **Fonts** — add a Thai-capable family to the existing font tokens so Latin
   text keeps Geist/Instrument Serif and Thai text falls back to the Thai face.
7. **Batches** — translate and review in the six groups above, one batch per
   pass, with the English source and proposed Thai side by side.

## Decisions

1. **Default locale**: Thai. English is one tap away in the pill.
2. **Thai calendar**: Gregorian. Thai month names, Gregorian year.
3. **Pill**: flags only. Use small inline SVG flags with accessible labels, not
   emoji, so they render the same on every platform.
4. **Thai font**: IBM Plex Sans Thai, used for both body and headings where the
   Latin faces have no Thai glyphs.
5. **Player pill**: no. The player inherits the page-level choice.
6. **Boot label**: translated before Svelte mounts with a tiny inline script
   that reads the stored locale.

## Still to settle during scaffolding

- The exact flag artwork and how the pill reads to a screen reader.
- Whether the pill sits in the top-right nav on every public page, including
  the error page.
- The Thai font weights to load (regular, medium, semibold) and whether the
  display headings need a separate Thai face.
