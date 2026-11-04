<script>
  import { onMount } from 'svelte';

  // Opt-in diagnostics overlay. Player only creates this when the viewer asked
  // for it (?debug=1 or localStorage.jellyshareDebug=1), so a normal viewer
  // pays nothing for it: no interval, no listeners, no DOM.
  export let getVideo = () => null;
  export let getHls = () => null;
  export let getInitPTS = () => null;
  export let mode = 'hls';
  export let startPosition = 0;
  export let sessionId = '';

  const TICK_MS = 250;

  let snapshot = {};
  let stalls = 0;
  let waits = 0;
  let hidden = false;
  let expanded = false;

  let ticker = null;
  let video = null;
  let seekTimer = null;
  let lastSeekSample = 0;

  function round(n, digits = 3) {
    return Number.isFinite(n) ? Number(n.toFixed(digits)) : null;
  }

  // The fragment whose playlist window covers the current position.
  function fragmentAt(details, time) {
    if (!details?.fragments) return null;
    const frags = details.fragments;
    for (let i = frags.length - 1; i >= 0; i--) {
      if (frags[i].start <= time + 0.001) return frags[i];
    }
    return frags[0] || null;
  }

  function levelDetails() {
    const hls = getHls();
    if (!hls || !hls.levels || !hls.levels.length) return null;
    const index = hls.currentLevel >= 0 ? hls.currentLevel : hls.loadingLevel;
    return hls.levels[index]?.details || null;
  }

  function bufferAhead(v) {
    const b = v.buffered;
    for (let i = 0; i < b.length; i++) {
      if (b.start(i) <= v.currentTime + 0.01 && v.currentTime <= b.end(i)) {
        return round(b.end(i) - v.currentTime);
      }
    }
    return null;
  }

  function isAudioId(id) {
    return typeof id === 'string' && id.toLowerCase().includes('audio');
  }

  function activeCue(v) {
    const tracks = v.textTracks;
    if (!tracks) return null;
    for (let i = 0; i < tracks.length; i++) {
      const track = tracks[i];
      if (track.mode !== 'showing') continue;
      const cues = track.activeCues;
      if (cues && cues.length) {
        const cue = cues[0];
        return {
          start: round(cue.startTime),
          end: round(cue.endTime),
          text: (cue.text || '').slice(0, 80)
        };
      }
      // Not active yet, but the next one is useful to know about.
      if (track.cues && track.cues.length) {
        for (let c = 0; c < track.cues.length; c++) {
          if (track.cues[c].startTime > v.currentTime) {
            return { next: round(track.cues[c].startTime) };
          }
        }
      }
    }
    return null;
  }

  function sample(reason) {
    const v = getVideo();
    if (!v) return;
    const hls = getHls();
    const details = levelDetails();
    const first = details?.fragments?.[0] || null;
    const current = details ? fragmentAt(details, v.currentTime) : null;
    // The offset between the media timeline and the playlist: hls.js lines the
    // first fragment up with its playlist position, so every cue authored on
    // the source clock lands this far off. Subtracting it is the correction.
    const offset = first && Number.isFinite(first.startPTS) ? round(first.startPTS - first.start) : null;
    // hls.js re-bases the media timeline on the first timestamp it parses, so
    // every fragment PTS above is already relative and reads ~0. The raw
    // timestamp it started from is the only way to see a server-side offset.
    const init = getInitPTS() || {};
    const entries = Object.entries(init)
      .map(([id, e]) => ({ ...e, id }))
      .filter((e) => e && Number.isFinite(e.seconds));
    const videoEntry = entries.find((e) => e && !isAudioId(e.id)) || entries[0] || null;
    const audioEntry = entries.find((e) => e && isAudioId(e.id)) || null;
    const offsetOf = (e) => (e && Number.isFinite(e.fragStart) ? round(e.seconds - e.fragStart) : null);
    const mediaPTS = videoEntry ? round(videoEntry.seconds) : null;
    const mediaOffset = offsetOf(videoEntry);
    const audioOffset = offsetOf(audioEntry);
    const avOffset = mediaOffset !== null && audioOffset !== null ? round(audioOffset - mediaOffset) : null;
    const quality = v.getVideoPlaybackQuality?.();

    snapshot = {
      at: new Date().toISOString(),
      reason,
      sessionId,
      mode: hls ? 'hls' : mode,
      currentTime: round(v.currentTime),
      duration: round(v.duration),
      startPosition: round(startPosition),
      readyState: v.readyState,
      paused: v.paused,
      hidden: document.visibilityState === 'hidden',
      resolution: v.videoWidth ? `${v.videoWidth}x${v.videoHeight}` : null,
      droppedFrames: quality?.droppedVideoFrames ?? null,
      totalFrames: quality?.totalVideoFrames ?? null,
      bufferAhead: bufferAhead(v),
      level: hls ? (hls.currentLevel >= 0 ? hls.currentLevel : hls.loadingLevel) : null,
      firstFragment: first ? {
        start: round(first.start),
        startPTS: round(first.startPTS),
        deltaPTS: round(first.deltaPTS)
      } : null,
      currentFragment: current ? {
        start: round(current.start),
        startPTS: round(current.startPTS),
        deltaPTS: round(current.deltaPTS)
      } : null,
      hlsOffset: offset,
      mediaPTS,
      mediaOffset,
      audioOffset,
      avOffset,
      predictedCorrection: mediaOffset !== null ? round(-mediaOffset)
        : offset === null ? null : round(-offset),
      cue: activeCue(v),
      waiting: waits,
      stalled: stalls
    };
  }

  function attach() {
    video = getVideo();
    if (!video) return;

    const onSeeked = () => {
      // Scrubbing fires a burst of seeked events; keep one sample per second.
      clearTimeout(seekTimer);
      seekTimer = setTimeout(() => {
        if (Date.now() - lastSeekSample < 1000) return;
        lastSeekSample = Date.now();
        sample('seeked');
      }, 400);
    };
    const onWaiting = () => { waits++; sample('waiting'); };
    const onStalled = () => { stalls++; sample('stalled'); };
    const onLoaded = () => sample('loadedmetadata');
    const onSeeking = () => sample('seeking');

    video.addEventListener('seeked', onSeeked);
    video.addEventListener('waiting', onWaiting);
    video.addEventListener('stalled', onStalled);
    video.addEventListener('loadedmetadata', onLoaded);
    video.addEventListener('seeking', onSeeking);

    return () => {
      clearTimeout(seekTimer);
      video?.removeEventListener('seeked', onSeeked);
      video?.removeEventListener('waiting', onWaiting);
      video?.removeEventListener('stalled', onStalled);
      video?.removeEventListener('loadedmetadata', onLoaded);
      video?.removeEventListener('seeking', onSeeking);
    };
  }

  let detach = null;

  function stop() {
    if (ticker) clearInterval(ticker);
    ticker = null;
    detach?.();
    detach = null;
  }

  onMount(() => {
    detach = attach();
    ticker = setInterval(() => sample('tick'), TICK_MS);
    sample('mount');
    return stop;
  });

  async function copyDiagnostics() {
    sample('copy');
    const text = JSON.stringify(snapshot, null, 2);
    try {
      await navigator.clipboard.writeText(text);
      copied = true;
      setTimeout(() => (copied = false), 1500);
    } catch {
      // Clipboard permission is not always granted; show it so it can be read.
      console.log(text);
      copied = false;
    }
  }

  let copied = false;

  function hide() {
    stop();
    hidden = true;
  }

  function disable() {
    try { localStorage.removeItem('jellyshareDebug'); } catch { /* private mode */ }
    hide();
  }
</script>

{#if !hidden}
  <aside class="debug" aria-label="Playback diagnostics">
    <header class="debug__head">
      <strong>sync debug</strong>
      <span class="debug__mode">{snapshot.mode}</span>
      <button class="debug__toggle" on:click={() => (expanded = !expanded)}>
        {expanded ? 'less' : 'more'}
      </button>
      <button class="debug__x" on:click={hide} title="Hide" aria-label="Hide diagnostics">×</button>
    </header>

    <dl class="debug__grid">
      <dt>time</dt>
      <dd>{snapshot.currentTime} / {snapshot.duration}</dd>

      <dt>media PTS</dt>
      <dd class:debug__warn={Math.abs(snapshot.mediaOffset || 0) > 0.25}>
        {snapshot.mediaOffset ?? '—'}
      </dd>

      <dt>audio PTS</dt>
      <dd class:debug__warn={Math.abs(snapshot.audioOffset || 0) > 0.25}>
        {snapshot.audioOffset ?? '—'}
      </dd>

      <dt>A/V start</dt>
      <dd class:debug__warn={Math.abs(snapshot.avOffset || 0) > 0.25}>
        {snapshot.avOffset ?? '—'}
      </dd>

      <dt>predicted</dt>
      <dd class:debug__warn={Math.abs(snapshot.predictedCorrection || 0) > 0.25}>
        {snapshot.predictedCorrection ?? '—'}s
      </dd>

      <dt>cue</dt>
      <dd>
        {#if snapshot.cue?.start != null}
          {snapshot.cue.start}–{snapshot.cue.end}: {snapshot.cue.text}
        {:else if snapshot.cue?.next != null}
          next at {snapshot.cue.next}
        {:else}
          none
        {/if}
      </dd>

      <dt>wait / stall</dt>
      <dd>{snapshot.waiting} / {snapshot.stalled}</dd>

      {#if expanded}
        <dt>start position</dt>
        <dd>{snapshot.startPosition}</dd>
        <dt>ready / paused</dt>
        <dd>{snapshot.readyState} / {snapshot.paused ? 'yes' : 'no'}</dd>
        <dt>buffer ahead</dt>
        <dd>{snapshot.bufferAhead ?? '—'}s</dd>
        <dt>video</dt>
        <dd>{snapshot.resolution || '—'}</dd>
        <dt>dropped / total</dt>
        <dd>{snapshot.droppedFrames ?? '—'} / {snapshot.totalFrames ?? '—'}</dd>
        <dt>level</dt>
        <dd>{snapshot.level ?? '—'}</dd>
        <dt>first frag</dt>
        <dd>
          {#if snapshot.firstFragment}
            start={snapshot.firstFragment.start} startPTS={snapshot.firstFragment.startPTS}
          {:else}
            —
          {/if}
        </dd>
        <dt>frag at time</dt>
        <dd>
          {#if snapshot.currentFragment}
            start={snapshot.currentFragment.start} startPTS={snapshot.currentFragment.startPTS}
            Δ={snapshot.currentFragment.deltaPTS}
          {:else}
            —
          {/if}
        </dd>
        <dt>frag drift (rel.)</dt>
        <dd>{snapshot.hlsOffset ?? '—'}</dd>
        <dt>visibility</dt>
        <dd>{snapshot.hidden ? 'hidden' : 'visible'}</dd>
      {/if}
    </dl>

    <footer class="debug__foot">
      <button on:click={copyDiagnostics}>{copied ? 'Copied' : 'Copy diagnostics'}</button>
      <button on:click={disable}>Turn off</button>
    </footer>
  </aside>
{/if}

<style>
  /* A bottom sheet, not a floating card: on a phone in portrait the video is a
     band across the middle and everything below it is empty, so the numbers
     stay readable without covering the picture. Collapsed by default and
     capped in height so it never eats the frame. */
  .debug {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: var(--z-dropdown);
    max-height: 46dvh;
    overflow-y: auto;
    padding: 0.4rem 0.65rem calc(0.4rem + env(safe-area-inset-bottom));
    background: color-mix(in oklch, var(--color-paper) 94%, transparent);
    border-top: var(--rule-hair) solid var(--color-rule);
    backdrop-filter: blur(12px);
    font-family: var(--font-outlier);
    font-size: 0.6875rem;
    line-height: 1.35;
    color: var(--color-ink-2);
    text-align: left;
  }

  .debug__head {
    display: flex;
    align-items: center;
    gap: var(--space-2xs);
    margin-bottom: 0.15rem;
    font-size: 0.6875rem;
  }

  .debug__head strong { color: var(--color-ink); }

  .debug__mode {
    padding-inline: 0.3rem;
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-pill);
  }

  .debug__toggle {
    margin-left: auto;
    padding: 0.1rem 0.45rem;
    background: none;
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-pill);
    color: var(--color-accent);
    font: inherit;
    cursor: pointer;
  }

  .debug__x {
    background: none;
    border: none;
    padding: 0 0.25rem;
    color: var(--color-muted);
    font-size: 1rem;
    line-height: 1;
    cursor: pointer;
  }

  .debug__grid {
    display: grid;
    grid-template-columns: minmax(5.25rem, 7.5rem) 1fr;
    gap: 0 var(--space-xs);
    margin: 0;
    font-variant-numeric: tabular-nums;
  }

  .debug__grid dt { color: var(--color-muted); }
  .debug__grid dd { margin: 0; overflow-wrap: anywhere; }

  .debug__warn { color: var(--color-accent); font-weight: 600; }

  .debug__foot {
    display: flex;
    gap: var(--space-md);
    margin-top: 0.2rem;
    padding-top: 0.2rem;
    border-top: var(--rule-hair) solid var(--color-rule);
  }

  .debug__foot button {
    background: none;
    border: none;
    padding: 0;
    color: var(--color-accent);
    font: inherit;
    cursor: pointer;
  }
</style>
