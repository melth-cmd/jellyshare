#!/usr/bin/env bash
# Generate subtitle-sync test clips for playback diagnosis (Phase 0).
#
# Each clip burns in its own timecode, beeps every 10 seconds, and carries a
# subtitle whose text is its own cue timestamp, so the sync error can be read
# from a screenshot instead of guessed.
#
# Usage: scripts/make-sync-clip.sh [--out DIR] [--duration SECONDS] [--force]
# Env:   FFMPEG (default: ffmpeg), FONT (default: first match for "monospace")
#
# The clips are written to <repo>/.sync-clips/library, one folder per variant so
# Jellyfin indexes each as a separate item. Point a test Jellyfin library at that
# folder, then create one Jellyshare link per item.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="$REPO_ROOT/.sync-clips"
DURATION=120
FORCE=0
FFMPEG="${FFMPEG:-ffmpeg}"

while [ $# -gt 0 ]; do
  case "$1" in
    --out) OUT_DIR="$2"; shift 2 ;;
    --duration) DURATION="$2"; shift 2 ;;
    --force) FORCE=1; shift ;;
    -h|--help) awk 'NR>1 && /^#/ { sub(/^# ?/, ""); print; next } NR>1 { exit }' "${BASH_SOURCE[0]}"; exit 0 ;;
    *) echo "error: unknown argument '$1'" >&2; exit 1 ;;
  esac
done

if ! [[ "$DURATION" =~ ^[1-9][0-9]*$ ]] || [ "$DURATION" -lt 20 ]; then
  echo "error: --duration must be a whole number of seconds, at least 20" >&2
  exit 1
fi

if ! command -v "$FFMPEG" >/dev/null 2>&1; then
  echo "error: '$FFMPEG' not found; install ffmpeg or set FFMPEG" >&2
  exit 1
fi

if [ -z "${FONT:-}" ]; then
  FONT="$(fc-match -f '%{file}' monospace 2>/dev/null || true)"
fi
if [ -z "${FONT:-}" ] || [ ! -f "$FONT" ]; then
  echo "error: no usable font found; set FONT to a .ttf/.otf path" >&2
  exit 1
fi

LIBRARY="$OUT_DIR/library"
WORK="$OUT_DIR/work"
if [ -d "$LIBRARY" ] && [ "$FORCE" -eq 0 ]; then
  echo "error: $LIBRARY already exists; pass --force to regenerate" >&2
  exit 1
fi
rm -rf "$LIBRARY" "$WORK"
mkdir -p "$LIBRARY" "$WORK"

# Escape a path for ffmpeg's filtergraph parser.
filter_path() { printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/:/\\:/g' -e "s/'/\\\\'/g"; }
FONT_ESCAPED="$(filter_path "$FONT")"

# Draw the burned-in clock: timecode, frame number and a 10-second flash/marker.
draw_overlay() {
  cat <<EOF
drawtext=fontfile='$FONT_ESCAPED':text='%{pts\\:hms}':fontcolor=white:fontsize=96:box=1:boxcolor=black@0.7:boxborderw=24:x=(w-text_w)/2:y=(h-text_h)/2,
drawtext=fontfile='$FONT_ESCAPED':text='frame %{n}  fps $1':fontcolor=white:fontsize=36:box=1:boxcolor=black@0.7:boxborderw=12:x=(w-text_w)/2:y=h*0.72,
drawbox=x=0:y=0:w=iw:h=48:color=white@0.9:t=fill:enable='lt(mod(t\,10)\,0.2)',
drawbox=x='(w-120)*mod(t\,10)/10':y=h-140:w=120:h=120:color=orange:t=fill
EOF
}

# A 1 kHz beep for the first half second of every 10 seconds.
AUDIO_SRC="aevalsrc=if(lt(mod(t\\,10)\\,0.5)\\,0.6*sin(2*PI*1000*t)\\,0):s=48000:d=$DURATION"

# render_base OUTFILE FPS [extra_encode_args...]
render_base() {
  local out="$1" fps="$2"
  shift 2
  "$FFMPEG" -hide_banner -loglevel error -y \
    -f lavfi -i "color=c=0x101010:s=1280x720:r=$fps:d=$DURATION" \
    -f lavfi -i "$AUDIO_SRC" \
    -vf "$(draw_overlay "$fps")" \
    -c:v libx264 -preset veryfast -crf 20 -pix_fmt yuv420p "$@" \
    -c:a aac -b:a 128k -shortest "$out"
}

# Write an SRT whose cue text is its own start time. Cues run 3s every 10s.
# $3 scales every timestamp, which is how a 25fps-timed file is produced; $4
# shifts every cue, for a file whose subtitles run early.
write_srt() {
  local out="$1" duration="$2" scale="${3:-1}" shift="${4:-0}"
  awk -v dur="$duration" -v scale="$scale" -v shift="$shift" '
    function ts(x, sep,  h, m, s, ms) {
      ms = int((x - int(x)) * 1000 + 0.5)
      x = int(x)
      if (ms == 1000) { x += 1; ms = 0 }
      h = int(x / 3600); m = int((x % 3600) / 60); s = x % 60
      return sprintf("%02d:%02d:%02d%s%03d", h, m, s, sep, ms)
    }
    BEGIN {
      for (t = 0; t + 3 <= dur; t += 10) {
        start = t * scale + shift
        if (start < 0) continue
        printf "%d\n%s --> %s\nSUB %s\n\n", ++n, ts(start, ","), ts((t + 3) * scale + shift, ","), ts(t, ".")
      }
    }' >"$out"
}

echo "Rendering base clips (${DURATION}s) ..."
render_base "$WORK/base-24000.mp4" 24 -g 48
render_base "$WORK/base-23976.mp4" 24000/1001 -g 48
render_base "$WORK/base-longgop.mp4" 24 -g 240 -keyint_min 240 -sc_threshold 0

publish() { # publish SLUG TITLE SOURCE [extra args...]
  local slug="$1" title="$2" source="$3"
  shift 3
  local dir="$LIBRARY/$slug"
  mkdir -p "$dir"
  cp "$source" "$dir/$slug.mp4"
  if [ $# -gt 0 ]; then
    "$FFMPEG" -hide_banner -loglevel error -y -i "$dir/$slug.mp4" "$@" -c copy "$dir/.tmp.mp4"
    mv "$dir/.tmp.mp4" "$dir/$slug.mp4"
  fi
  echo "  $title"
}

publish "Sync Normal (2026)" "24 fps, correct timings" "$WORK/base-24000.mp4"
write_srt "$LIBRARY/Sync Normal (2026)/Sync Normal (2026).en.srt" "$DURATION"

publish "Sync Long Gop (2026)" "24 fps, 10s keyframe interval" "$WORK/base-longgop.mp4"
write_srt "$LIBRARY/Sync Long Gop (2026)/Sync Long Gop (2026).en.srt" "$DURATION"

publish "Sync Offset Plus 10 (2026)" "24 fps, output timestamps shifted +10s" \
  "$WORK/base-24000.mp4" -output_ts_offset 10
write_srt "$LIBRARY/Sync Offset Plus 10 (2026)/Sync Offset Plus 10 (2026).en.srt" "$DURATION"

# A negative presentation start cannot be stored in MP4, so the "subtitles run
# early" direction is produced on the subtitle side instead.
publish "Sync Subs Early 2 (2026)" "24 fps, subtitle cues 2s early" "$WORK/base-24000.mp4"
write_srt "$LIBRARY/Sync Subs Early 2 (2026)/Sync Subs Early 2 (2026).en.srt" "$DURATION" 1 -2

# Video is 23.976 fps but the subtitles are timed for a 25 fps cut, so they run
# early by 25/23.976 and the error grows ~2.5s per minute.
publish "Sync Fps Mismatch (2026)" "23.976 fps video, 25 fps timings" "$WORK/base-23976.mp4"
write_srt "$LIBRARY/Sync Fps Mismatch (2026)/Sync Fps Mismatch (2026).en.srt" "$DURATION" \
  "$(awk 'BEGIN { printf "%.9f", 25 / (24000 / 1001) }')"

# Embedded subtitle in MKV: the same cues, but Jellyfin has to render them.
mkdir -p "$LIBRARY/Sync Embedded (2026)"
cp "$WORK/base-24000.mp4" "$WORK/embedded.mkv"
write_srt "$WORK/embedded.en.srt" "$DURATION"
"$FFMPEG" -hide_banner -loglevel error -y \
  -i "$WORK/embedded.mkv" -i "$WORK/embedded.en.srt" \
  -map 0 -map 1 -c copy -c:s srt -metadata:s:s:0 language=eng \
  "$LIBRARY/Sync Embedded (2026)/Sync Embedded (2026).mkv"
echo "  Embedded (2026)"

rm -rf "$WORK"

echo
echo "Clips written to: $LIBRARY"
echo "Add that folder as a Jellyfin library, scan it, then create one Jellyshare"
echo "link per item. Each clip shows its own timecode and beeps every 10s; the"
echo "subtitle prints the timestamp it was authored for."
