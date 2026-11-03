# Branding

The app mark is the inherited one from the upstream project. This page lists
every place it appears so a replacement can be done without touching unrelated
icons.

## Asset files

| File | Size | Referenced by |
| --- | --- | --- |
| `web/public/brand/favicon.ico` | 16×16, 32×32, 48×48 | `web/index.html` |
| `web/public/brand/jellyshare-mark-small.svg` | SVG, `viewBox="0 0 120 120"` | `web/index.html` |
| `web/public/brand/apple-touch-icon.png` | 180×180 | `web/index.html` |
| `web/public/brand/icon-192.png` | 192×192 | `web/public/brand/manifest.webmanifest` |
| `web/public/brand/icon-512.png` | 512×512 | `web/public/brand/manifest.webmanifest` |
| `web/public/brand/icon-maskable-512.png` | 512×512, maskable | `web/public/brand/manifest.webmanifest` |
| `web/public/brand/manifest.webmanifest` | JSON | `web/index.html` |
| `web/src/components/BrandMark.svelte` | inline SVG | four components |

## Where the inline mark is drawn

`BrandMark.svelte` is imported and rendered in exactly four places:

- `web/src/components/AdminLogin.svelte:47`
- `web/src/components/ErrorView.svelte:80`
- `web/src/components/ShareView.svelte:1134`
- `web/src/components/AdminDashboard.svelte:431`

The component is decorative (`aria-hidden="true"`); the word “Jellyshare” beside
it carries the name. Its three colours come from `tokens.css`:
`--color-ink-2` (the line), `--color-ink` (the two nodes) and `--color-accent`
(the apex).

## What not to touch

- `shareInfo.logoUrl` in `ShareView.svelte` is the **Jellyfin media logo** (the
  film or series title art), not the app mark.
- `CastIcon.svelte` is the **Google Cast** mark.
- `CheckIcon.svelte`, `DownloadIcon.svelte` and `PlayIcon.svelte` are UI icons.
- The footer wordmark is text, not an image.

## Replacing the mark

1. **Prepare the files at the same sizes and names.** Keeping the names means
   `index.html` and `manifest.webmanifest` need no changes.
   - `favicon.ico`: 16, 32 and 48 px.
   - `jellyshare-mark-small.svg`: square viewBox, readable at 16–24 px.
   - `apple-touch-icon.png`: 180×180, opaque background (iOS rounds the corners).
   - `icon-192.png` and `icon-512.png`: square, opaque or transparent as the
     design needs.
   - `icon-maskable-512.png`: 512×512 with the important content inside the
     central safe zone (about 80% of the square); the background must fill the
     whole canvas.
2. **Replace the six files** in `web/public/brand/`.
3. **Replace the inline SVG** in `web/src/components/BrandMark.svelte`. Keep the
   `viewBox`, the `aria-hidden`/`focusable` attributes and the `.mark` sizing
   rule. Map the new colours to the existing tokens, or add tokens to
   `tokens.css` if the mark needs new ones. If you would rather load a file than
   draw inline, replace the `<svg>` with an `<img>`; the trade-off is that it
   will no longer follow the theme tokens.
4. **If you rename a file**, update the matching `<link>` in `web/index.html` or
   the `icons` array in `web/public/brand/manifest.webmanifest`.
5. **If the background colour changes**, update the `theme-color` meta tag in
   `web/index.html` and `background_color`/`theme_color` in
   `manifest.webmanifest`.
6. **Build and check** with `cd web && npm run build`, then look at:
   - the browser tab (16–32 px),
   - the admin login page,
   - a public share page,
   - the error page,
   - the nav mark at its normal size (about 22 px),
   - a PWA install or “Add to Home Screen” preview,
   - the maskable icon on Android (no clipped edges).

## Checklist

- [ ] Six files replaced in `web/public/brand/`.
- [ ] `BrandMark.svelte` updated.
- [ ] `index.html` and `manifest.webmanifest` still resolve.
- [ ] `theme-color` and manifest colours match the new artwork.
- [ ] Frontend build passes.
- [ ] Tab, admin, share, error and PWA icons checked.
