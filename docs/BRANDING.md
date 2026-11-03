# Branding

The app mark is the neon purple-blue vortex play icon. It is a raster design, so
the icons are generated from the 1254×1254 source and the nav draws a cropped
PNG rather than an inline SVG.

## Asset files

| File | Size | Referenced by |
| --- | --- | --- |
| `web/public/brand/favicon.ico` | 16×16, 32×32, 48×48 | `web/index.html` |
| `web/public/brand/apple-touch-icon.png` | 180×180 | `web/index.html` |
| `web/public/brand/icon-192.png` | 192×192 | `web/public/brand/manifest.webmanifest` |
| `web/public/brand/icon-512.png` | 512×512 | `web/public/brand/manifest.webmanifest` |
| `web/public/brand/icon-maskable-512.png` | 512×512, maskable | `web/public/brand/manifest.webmanifest` |
| `web/public/brand/mark-128.png` | 128×128 | `web/src/components/BrandMark.svelte` |
| `web/public/brand/manifest.webmanifest` | JSON | `web/index.html` |

The favicon and the nav tile are 900×900 centre crops, so the vortex fills more
of the small frame; 900 is the tightest crop that keeps the whole vortex. The
192 and 512 icons use the full square composition. The maskable icon is also the
full square because the vortex already sits inside the 80% safe zone.

## Where the nav mark is drawn

`BrandMark.svelte` is an `<img>` pointing at `/brand/mark-128.png`. It is
imported and rendered in exactly four places:

- `web/src/components/AdminLogin.svelte:47`
- `web/src/components/ErrorView.svelte:80`
- `web/src/components/ShareView.svelte:1134`
- `web/src/components/AdminDashboard.svelte:431`

The image is decorative (`alt=""`, `aria-hidden="true"`); the word “Jellyshare”
beside it carries the name. The component clips it to a small rounded square
with `border-radius: 0.22em`.

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
   - `apple-touch-icon.png`: 180×180, opaque background (iOS rounds the corners).
   - `icon-192.png` and `icon-512.png`: square.
   - `icon-maskable-512.png`: 512×512 with the important content inside the
     central safe zone (about 80% of the square); the background must fill the
     whole canvas.
   - `mark-128.png`: square, cropped so the mark reads at about 22 px.
2. **Replace the six files** in `web/public/brand/`.
3. **If you rename a file**, update the matching `<link>` in `web/index.html`,
   the `icons` array in `web/public/brand/manifest.webmanifest`, or the `src` in
   `web/src/components/BrandMark.svelte`.
4. **If the background colour changes**, update the `theme-color` meta tag in
   `web/index.html` and `background_color`/`theme_color` in
   `manifest.webmanifest`. The current icon background is `#010421`; the app
   theme is `#060d11`.
5. **Build and check** with `cd web && npm run build`, then look at:
   - the browser tab (16–32 px),
   - the admin login page,
   - a public share page,
   - the error page,
   - the nav mark at its normal size (about 22 px),
   - a PWA install or “Add to Home Screen” preview,
   - the maskable icon on Android (no clipped edges).

## Source

The 1254×1254 original is not committed. A copy is kept at
`/tmp/opencode/jellyshare-brand-source.png` on the build host. Regenerate the
raster set from it with ImageMagick if the artwork changes.

## Checklist

- [ ] Six files replaced in `web/public/brand/`.
- [ ] `BrandMark.svelte` still points at the nav tile.
- [ ] `index.html` and `manifest.webmanifest` still resolve.
- [ ] `theme-color` and manifest colours match the new artwork.
- [ ] Frontend build passes.
- [ ] Tab, admin, share, error and PWA icons checked.
