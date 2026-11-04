#!/usr/bin/env bash
# Trace one Jellyshare playback session for the Phase 0 sync investigation.
#
# Prints the session and share row, the Jellyfin item and its tracks, the
# subtitle source, Jellyfin and ffmpeg versions, the live Jellyfin session, every
# ffmpeg job started for this playback, the HLS playlist summary and the first
# segment's timestamps. File paths are redacted and nothing secret is printed.
#
# Usage: scripts/playback-trace.sh <session-id>
# Env:   JELLYFIN_CONTAINER   (default: jellyfin)
#        JELLYSHARE_DB_CONTAINER (default: jellyshare-db)
#        JELLYSHARE_CONTAINER (default: jellyshare, used to read the API key)
#        JELLYFIN_URL         (default: the published 8096 port on 127.0.0.1)
#        JELLYFIN_API_KEY     (default: the key in the jellyshare container)
#        OUT                  (default: /tmp/playback-trace-<session>.txt)
set -euo pipefail
umask 077

JELLYFIN_CONTAINER="${JELLYFIN_CONTAINER:-jellyfin}"
DB_CONTAINER="${JELLYSHARE_DB_CONTAINER:-jellyshare-db}"
JS_CONTAINER="${JELLYSHARE_CONTAINER:-jellyshare}"

if [ $# -ne 1 ]; then
  echo "usage: scripts/playback-trace.sh <session-id>" >&2
  exit 1
fi
SESSION_ID="$1"
if ! [[ "$SESSION_ID" =~ ^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$ ]]; then
  echo "error: '$SESSION_ID' is not a session id" >&2
  exit 1
fi

OUT="${OUT:-/tmp/playback-trace-$SESSION_ID.txt}"
exec > >(tee "$OUT") 2>&1

section() { printf '\n=== %s ===\n' "$1"; }

# Strip library paths from anything printed; keep transcode cache names, which
# identify a job but reveal nothing about the viewer or the library layout.
redact() {
  sed -E 's#file:"[^"]*"#file:"<path>"#g; s#(/data|/media|/movies|/tv|/music|/downloads|/mnt|/srv|/home)/[^ ",]*#<path>#g'
}

sql() {
  docker exec "$DB_CONTAINER" sh -c \
    'PGPASSWORD=$POSTGRES_PASSWORD psql -U $POSTGRES_USER -d $POSTGRES_DB -tA -F "|" -c "$1"' sh "$1"
}

jf() {
  curl -sS --max-time 10 -H "$AUTH_HEADER" "$JF_URL$1"
}

echo "Playback trace for session $SESSION_ID"
echo "Generated $(date -u +%Y-%m-%dT%H:%M:%SZ) UTC"

# ---------------------------------------------------------------------------
# Jellyfin connection
# ---------------------------------------------------------------------------
api_key="${JELLYFIN_API_KEY:-}"
if [ -z "$api_key" ]; then
  api_key="$(docker exec "$JS_CONTAINER" printenv JELLYSHARE_JELLYFIN_API_KEY 2>/dev/null || true)"
fi
if [ -z "${JELLYFIN_URL:-}" ]; then
  port="$(docker port "$JELLYFIN_CONTAINER" 8096/tcp 2>/dev/null | head -1 | sed 's/.*://' || true)"
  JELLYFIN_URL="http://127.0.0.1:${port:-8096}"
fi
JF_URL="$JELLYFIN_URL"
AUTH_HEADER="Authorization: MediaBrowser Client=\"playback-trace\", Device=\"playback-trace\", DeviceId=\"playback-trace\", Version=\"1.0\", Token=\"$api_key\""

# ---------------------------------------------------------------------------
# Session and share
# ---------------------------------------------------------------------------
section "Session"
row="$(sql "SELECT s.id, s.started_at, s.last_heartbeat_at, s.finished_at, s.terminated_reason,
                  s.jellyfin_item_id, s.media_source_id, s.audio_stream_index,
                  s.subtitle_stream_index, s.vtt_subtitle_index, s.video_bitrate,
                  s.video_codec, s.max_video_height,
                  sh.title, sh.item_type, sh.max_video_height, sh.max_video_bitrate,
                  sh.allow_download, sh.expires_at, sh.revoked_at
           FROM share_sessions s JOIN shares sh ON sh.id = s.share_id
           WHERE s.id = '$SESSION_ID'")"
if [ -z "$row" ]; then
  echo "session not found in $DB_CONTAINER"
  exit 1
fi
IFS='|' read -r _ started heartbeat finished reason item_id media_source_id \
  audio_idx subtitle_idx vtt_idx bitrate codec max_height \
  title item_type share_max_height share_max_bitrate allow_download expires revoked <<<"$row"

START_UTC="$started"
END_UTC="$finished"
printf '%-22s %s\n' "started" "$started"
printf '%-22s %s\n' "last heartbeat" "${heartbeat:-—}"
printf '%-22s %s\n' "finished" "${finished:-still active}"
printf '%-22s %s\n' "terminated" "${reason:-—}"
printf '%-22s %s (%s)\n' "item" "$title" "$item_type"
printf '%-22s %s\n' "jellyfin item id" "$item_id"
printf '%-22s %s\n' "media source id" "$media_source_id"
printf '%-22s %s\n' "codec pinned" "${codec:-—}"
printf '%-22s %s\n' "video bitrate" "${bitrate:-—}"
printf '%-22s %s\n' "max height" "${max_height:-—}"
printf '%-22s %s\n' "audio index" "${audio_idx:-—}"
printf '%-22s %s\n' "subtitle index" "${subtitle_idx:-—}"
printf '%-22s %s\n' "sidecar vtt index" "${vtt_idx:-—}"
if [ -n "$vtt_idx" ]; then
  echo "subtitle mode         sidecar (WebVTT <track>)"
elif [ -n "$subtitle_idx" ]; then
  echo "subtitle mode         burn-in (SubtitleMethod=Encode)"
else
  echo "subtitle mode         none"
fi
printf '%-22s %s\n' "share cap height" "${share_max_height:-—}"
printf '%-22s %s\n' "share cap bitrate" "${share_max_bitrate:-—}"
printf '%-22s %s\n' "downloads allowed" "$allow_download"
printf '%-22s %s\n' "share expires" "${expires:-never}"
printf '%-22s %s\n' "share revoked" "${revoked:-no}"

# ---------------------------------------------------------------------------
# Item and tracks
# ---------------------------------------------------------------------------
section "Source"
item_json="$(jf "/Items?Ids=$item_id&Fields=MediaSources,Path,MediaStreams" || true)"
if [ -n "$item_json" ]; then
  printf '%s' "$item_json" | python3 -c '
import json, sys
d = (json.load(sys.stdin).get("Items") or [{}])[0]
print("name       ", d.get("Name"))
print("container  ", (d.get("MediaSources") or [{}])[0].get("Container"))
print("path       ", "<redacted>")
print()
print("%-5s %-9s %-8s %-6s %-9s %s" % ("index", "type", "codec", "lang", "external", "title"))
for s in d.get("MediaStreams") or []:
    print("%-5s %-9s %-8s %-6s %-9s %s" % (
        s.get("Index"), s.get("Type"), s.get("Codec") or "-", s.get("Language") or "-",
        s.get("IsExternal"), s.get("DisplayTitle") or ""))
'
  for stream in $(printf '%s' "$item_json" | python3 -c '
import json, sys
d = (json.load(sys.stdin).get("Items") or [{}])[0]
for s in d.get("MediaStreams") or []:
    if s.get("Type") in ("Video", "Subtitle"):
        print(s["Index"])
'); do
    printf '%s' "$item_json" | python3 -c '
import json, sys
d = (json.load(sys.stdin).get("Items") or [{}])[0]
s = next(x for x in d["MediaStreams"] if x["Index"] == int(sys.argv[1]))
kind = s.get("Type")
if kind == "Video":
    print("video %s: %s %sx%s %.3ffps bitrate=%s profile=%s bitdepth=%s"
          % (s["Index"], s.get("Codec"), s.get("Width"), s.get("Height"),
             float(s.get("RealFrameRate") or 0), s.get("BitRate"),
             s.get("Profile") or "-", s.get("BitDepth") or "-"))
else:
    print("subtitle %s: %s lang=%s external=%s text=%s forced=%s default=%s"
          % (s["Index"], s.get("Codec"), s.get("Language"), s.get("IsExternal"),
             s.get("IsTextSubtitleStream"), s.get("IsForced"), s.get("IsDefault")))
' "$stream"
  done
else
  echo "could not read the item from Jellyfin"
fi

# ---------------------------------------------------------------------------
# Versions and acceleration
# ---------------------------------------------------------------------------
section "Versions"
jf "/System/Info/Public" 2>/dev/null | python3 -c '
import json, sys
try:
    d = json.load(sys.stdin)
    print("jellyfin", d.get("Version"), "on", d.get("ServerName"))
except Exception:
    print("jellyfin version unavailable")
' || true
docker exec "$JELLYFIN_CONTAINER" /usr/lib/jellyfin-ffmpeg/ffmpeg -version 2>/dev/null | head -1 || echo "ffmpeg version unavailable"
enc="$(jf "/System/Configuration/encoding" 2>/dev/null || true)"
if [ -n "$enc" ]; then
  printf '%s' "$enc" | python3 -c '
import json, sys
try:
    d = json.load(sys.stdin)
    print("hardware acceleration", d.get("HardwareAccelerationType") or "none")
    print("throttling           ", d.get("EnableThrottling"))
except Exception:
    print("hardware acceleration unknown")
'
else
  echo "hardware acceleration  unavailable (admin key required)"
fi

# ---------------------------------------------------------------------------
# Live Jellyfin session
# ---------------------------------------------------------------------------
section "Jellyfin session"
device_id="jellyshare-$SESSION_ID"
sessions="$(jf "/Sessions" 2>/dev/null || true)"
if [ -n "$sessions" ]; then
  printf '%s' "$sessions" | python3 -c '
import json, sys
device = sys.argv[1]
items = json.load(sys.stdin)
match = [s for s in items if s.get("DeviceId") == device]
if not match:
    print("no /Sessions entry for", device)
    print("(Jellyshare does not post /Sessions/Playing, so Jellyfin may not list it)")
else:
    for s in match:
        print("device      ", s.get("DeviceId"))
        print("play session", s.get("PlaySessionId"))
        print("play method ", s.get("PlayMethod") or "not reported")
        print("position    ", (s.get("PlayState") or {}).get("PositionTicks"))
        info = s.get("TranscodingInfo") or {}
        if info:
            for k in ("IsVideoDirect", "IsAudioDirect", "VideoCodec", "AudioCodec",
                      "Container", "Bitrate", "TranscodeReasons"):
                print("%-12s %s" % (k, info.get(k)))
        else:
            print("transcoding  no TranscodingInfo (likely direct play)")
' "$device_id"
else
  echo "could not read /Sessions"
fi

# ---------------------------------------------------------------------------
# ffmpeg jobs
# ---------------------------------------------------------------------------
section "ffmpeg jobs"
jobs_file="$(mktemp)"
trap 'rm -f "$jobs_file"' EXIT
if [ -n "$END_UTC" ]; then
  docker exec "$JELLYFIN_CONTAINER" sh -c '
    TZ=UTC find /config/log -maxdepth 1 -name "FFmpeg.*_$1_*.log" -newermt "$2" ! -newermt "$3" -printf "%T@ %f\n" 2>/dev/null | sort -n
  ' sh "$media_source_id" "$START_UTC" "$END_UTC" >"$jobs_file" || true
else
  docker exec "$JELLYFIN_CONTAINER" sh -c '
    TZ=UTC find /config/log -maxdepth 1 -name "FFmpeg.*_$1_*.log" -newermt "$2" -printf "%T@ %f\n" 2>/dev/null | sort -n
  ' sh "$media_source_id" "$START_UTC" >"$jobs_file" || true
fi

job_count="$(wc -l <"$jobs_file" | tr -d ' ')"
echo "count since session start: $job_count"
if [ "$job_count" -eq 0 ]; then
  echo "no FFmpeg log matches media source $media_source_id after $START_UTC"
  echo "(direct play runs no ffmpeg job, and old logs may have been rotated)"
fi

last_playlist=""
while read -r _stamp name; do
  [ -n "$name" ] || continue
  echo
  echo "--- $name"
  command_line="$(docker exec "$JELLYFIN_CONTAINER" sh -c \
    'grep -m1 -E "^/usr/lib/jellyfin-ffmpeg/ffmpeg |^ffmpeg " "/config/log/$1"' sh "$name" 2>/dev/null || true)"
  printf 'type    %s\n' "$(printf '%s' "$name" | sed -E 's/^FFmpeg\.([A-Za-z]+)-.*/\1/')"
  for flag in '-ss' '-copyts' '-start_at_zero' '-output_ts_offset' '-codec:v:0' '-codec:a:0' '-hls_time'; do
    if printf '%s' "$command_line" | grep -qE -- "(^| )$flag( |$)"; then
      if printf '%s' "$command_line" | grep -qE -- "(^| )$flag [^ -]"; then
        value="$(printf '%s' "$command_line" | grep -oE -- "(^| )$flag [^ ]+" | head -1 | sed "s/^ *$flag //")"
        printf '%-20s %s\n' "$flag" "$value"
      else
        printf '%-20s set\n' "$flag"
      fi
    fi
  done
  printf '%s\n' "$command_line" | redact | fold -s -w 160 | sed 's/^/  /'
  playlist="$(printf '%s' "$command_line" | sed -nE 's#.*-y "(/cache/transcodes/[^"]*\.m3u8)".*#\1#p')"
  [ -n "$playlist" ] && last_playlist="$playlist"
done <"$jobs_file"

# ---------------------------------------------------------------------------
# HLS playlist and first segment
# ---------------------------------------------------------------------------
section "HLS"
if [ -z "$last_playlist" ]; then
  echo "no playlist found for this playback"
else
  echo "playlist: $(basename "$last_playlist")"
  docker exec "$JELLYFIN_CONTAINER" cat "$last_playlist" >"$jobs_file.m3u8" 2>/dev/null || true
  if [ ! -s "$jobs_file.m3u8" ]; then
    echo "  (playlist no longer in the transcode cache; run this while the"
    echo "   playback is active or shortly after it stops)"
  else
    head -12 "$jobs_file.m3u8" | sed 's/^/  /'
  fi
  first_segment="$(grep -v '^#' "$jobs_file.m3u8" | head -1 || true)"
  if [ -n "$first_segment" ]; then
    segment_path="$(dirname "$last_playlist")/$first_segment"
    echo
    echo "first segment: $first_segment"
    docker exec "$JELLYFIN_CONTAINER" sh -c '
      f="$1"
      [ -f "$f" ] || { echo "  (segment already deleted)"; exit 0; }
      v=$(/usr/lib/jellyfin-ffmpeg/ffprobe -v error -select_streams v:0 \
            -show_entries packet=pts_time,dts_time -of csv=p=0 "$f" | sort -n | head -1)
      a=$(/usr/lib/jellyfin-ffmpeg/ffprobe -v error -select_streams a:0 \
            -show_entries packet=pts_time -of csv=p=0 "$f" | sort -n | head -1)
      echo "  first video pts/dts: $v"
      echo "  first audio pts:     $a"
      vp=$(printf "%s" "$v" | cut -d, -f1)
      ap=$(printf "%s" "$a" | cut -d, -f1)
      [ -n "$vp" ] && awk -v p="$vp" "BEGIN{printf \"  video offset vs playlist: %+.3fs\n\", p}"
      [ -n "$ap" ] && awk -v p="$ap" "BEGIN{printf \"  audio offset vs playlist: %+.3fs\n\", p}"
    ' sh "$segment_path" || true
    echo "  playlist positions this segment at 0.000; a difference here is the"
    echo "  offset a player has to normalize before subtitles can line up."
  fi
  echo
  echo "transcode cache: $(docker exec "$JELLYFIN_CONTAINER" du -sh /cache/transcodes 2>/dev/null | cut -f1 || true)"
fi

echo
echo "Saved to $OUT"
