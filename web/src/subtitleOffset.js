// Subtitle timing is a viewer-side bandaid for sidecar WebVTT that drifts from
// the HLS timeline. The cues are ordinary VTTCue objects, so shifting every cue
// by the same constant moves the track without re-fetching it. A constant shift
// keeps the cue list in start order, which is what the browser's active-cue
// lookup relies on.

export const OFFSET_STEP = 0.2;
export const OFFSET_LIMIT = 30;

// Round to the step and keep the value inside the range. The toFixed keeps
// 0.2-based arithmetic from leaving values like 0.6000000000000001 behind.
export function clampOffset(value) {
  if (!Number.isFinite(value)) return 0;
  const rounded = Math.round(value / OFFSET_STEP) * OFFSET_STEP;
  return Math.max(-OFFSET_LIMIT, Math.min(OFFSET_LIMIT, Number(rounded.toFixed(1))));
}

export function formatOffset(value) {
  const offset = clampOffset(value);
  if (offset === 0) return '0.0s';
  return `${offset > 0 ? '+' : '−'}${Math.abs(offset).toFixed(1)}s`;
}

// `baseTimes` is a WeakMap keyed by cue, so a track that reloads with new cue
// objects starts from their own times rather than the shifted ones. Copying the
// list first avoids depending on how a browser reorders a live cue list while
// its entries are being changed.
export function shiftCues(cues, offset, baseTimes) {
  if (!cues) return;
  const list = Array.from(cues);
  for (const cue of list) {
    let base = baseTimes.get(cue);
    if (!base) {
      base = { start: cue.startTime, end: cue.endTime };
      baseTimes.set(cue, base);
    }
    cue.startTime = Math.max(0, base.start + offset);
    cue.endTime = Math.max(0, base.end + offset);
  }
}
