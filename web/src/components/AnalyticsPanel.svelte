<script>
  import { onMount } from 'svelte';

  export let apiKey;

  let data = null;
  let loading = true;
  let error = '';
  let tab = 'overview';

  onMount(load);

  export function reload() {
    load();
  }

  async function load() {
    loading = true;
    error = '';
    try {
      const response = await fetch('/api/admin/analytics', {
        headers: { 'X-Backend-Key': apiKey }
      });
      if (!response.ok) throw new Error('The backend refused the request. Check the key is still valid.');
      data = await response.json();
    } catch (e) {
      error = e.message || 'Couldn’t load analytics.';
      data = null;
    } finally {
      loading = false;
    }
  }

  function formatDuration(seconds) {
    if (!seconds) return '—';
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.round((seconds % 3600) / 60);
    return hours ? `${hours} h ${minutes} min` : `${minutes} min`;
  }

  function buildDays(rows) {
    const counts = new Map();
    for (const row of rows || []) {
      counts.set(String(row.date).slice(0, 10), row.views);
    }

    const days = [];
    const now = new Date();
    for (let i = 29; i >= 0; i -= 1) {
      const date = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() - i));
      const key = date.toISOString().slice(0, 10);
      days.push({ key, label: key.slice(5), views: counts.get(key) || 0 });
    }
    return days;
  }

  $: days = buildDays(data?.viewsByDay);
  $: maxViews = Math.max(1, ...days.map((day) => day.views));
</script>

<main class="analytics">
  <p class="analytics__lede">How your links are being used.</p>

  <div class="subtabs" role="tablist" aria-label="Analytics views">
    <button
      class="subtab"
      class:subtab--active={tab === 'overview'}
      role="tab"
      aria-selected={tab === 'overview'}
      on:click={() => (tab = 'overview')}
    >Overview</button>
    <button
      class="subtab"
      class:subtab--active={tab === 'links'}
      role="tab"
      aria-selected={tab === 'links'}
      on:click={() => (tab = 'links')}
    >Links</button>
  </div>

  {#if loading}
    <div class="state">
      <div class="state__meter"></div>
      <p>Reading the numbers</p>
    </div>
  {:else if error}
    <div class="state state--error">
      <p>{error}</p>
      <button class="btn-ghost" on:click={load}>Try again</button>
    </div>
  {:else if data}
    {#if tab === 'overview'}
      <div class="cards">
        <div class="card">
          <span class="card__label">Links</span>
          <span class="card__value">{data.totalShares}</span>
          <span class="card__note">{data.activeShares} active</span>
        </div>
        <div class="card">
          <span class="card__label">Plays</span>
          <span class="card__value">{data.totalPlays}</span>
          <span class="card__note">charged to links</span>
        </div>
        <div class="card">
          <span class="card__label">Views</span>
          <span class="card__value">{data.totalViews}</span>
          <span class="card__note">playback starts</span>
        </div>
        <div class="card">
          <span class="card__label">Unique viewers</span>
          <span class="card__value">{data.uniqueViewers}</span>
          <span class="card__note">by hashed address</span>
        </div>
        <div class="card">
          <span class="card__label">Average watch</span>
          <span class="card__value card__value--small">{formatDuration(data.avgWatchTimeSeconds)}</span>
          <span class="card__note">per session</span>
        </div>
      </div>

      <section class="panel">
        <h2 class="panel__head">Views, last 30 days</h2>
        {#if data.totalViews === 0}
          <p class="empty">No plays have been recorded yet.</p>
        {:else}
          <div class="chart" role="img" aria-label="Views per day over the last 30 days">
            {#each days as day (day.key)}
              <span class="chart__col" title="{day.key}: {day.views} {day.views === 1 ? 'view' : 'views'}">
                <span class="chart__bar" style="height: {Math.max(2, (day.views / maxViews) * 100)}%"></span>
              </span>
            {/each}
          </div>
          <div class="chart__axis">
            <span>{days[0].key}</span>
            <span>{days[days.length - 1].key}</span>
          </div>
        {/if}
      </section>
    {:else}
      <section class="panel">
        <h2 class="panel__head">Links</h2>
        {#if data.shares.length === 0}
          <p class="empty">No links yet.</p>
        {:else}
          <div class="table">
            <div class="table__head" aria-hidden="true">
              <span>Link</span><span>Views</span><span>Unique</span><span>Avg watch</span><span>Plays</span>
            </div>
            {#each data.shares as share (share.id)}
              <div class="table__row">
                <span class="table__title">{share.title}</span>
                <span class="table__num">{share.views}</span>
                <span class="table__num">{share.uniqueViewers}</span>
                <span class="table__num">{formatDuration(share.avgWatchTimeSeconds)}</span>
                <span class="table__num">{share.totalPlays}</span>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    {/if}
  {/if}
</main>

<style>
  .analytics {
    padding-block: calc(var(--space-3xl) + var(--space-md)) var(--space-2xl);
    max-width: 78rem;
    width: 100%;
  }

  .analytics__lede {
    max-width: 54ch;
    color: var(--color-muted);
    margin-bottom: var(--space-xl);
  }

  .subtabs {
    display: inline-flex;
    gap: 0.125rem;
    padding: 0.15rem;
    margin-bottom: var(--space-xl);
    background: var(--color-paper-3);
    border-radius: var(--radius-pill);
  }

  .subtab {
    min-height: 2.1rem;
    padding-inline: var(--space-md);
    background: none;
    color: var(--color-muted);
    font-size: var(--text-sm);
    font-weight: 600;
    white-space: nowrap;
    border: none;
    border-radius: var(--radius-pill);
    cursor: pointer;
    transition: background-color var(--dur-micro) var(--ease-out),
                color var(--dur-micro) var(--ease-out);
  }

  .subtab--active { background: var(--color-paper-2); color: var(--color-ink); }

  @media (hover: hover) {
    .subtab:hover:not(.subtab--active) { color: var(--color-ink); }
  }

  .subtab:active { transform: translateY(1px); }

  .cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
    gap: var(--space-md);
    margin-bottom: var(--space-2xl);
  }

  .card {
    display: grid;
    gap: var(--space-3xs);
    padding: var(--space-md);
    background: var(--color-paper-2);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-card);
  }

  .card__label {
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--color-neutral);
  }

  .card__value {
    font-family: var(--font-outlier);
    font-size: var(--text-2xl);
    font-variant-numeric: tabular-nums;
    color: var(--color-ink);
  }

  .card__value--small { font-size: var(--text-lg); }

  .card__note { font-size: var(--text-sm); color: var(--color-muted); }

  .panel {
    padding-top: var(--space-lg);
    border-top: var(--rule-hair) solid var(--color-rule-2);
  }

  .panel__head {
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--color-neutral);
    margin-bottom: var(--space-md);
  }

  .chart {
    display: flex;
    align-items: flex-end;
    gap: 2px;
    height: 9rem;
    padding-top: var(--space-sm);
  }

  .chart__col {
    display: flex;
    flex: 1;
    align-items: flex-end;
    height: 100%;
    min-width: 0;
  }

  .chart__bar {
    display: block;
    width: 100%;
    min-height: 2px;
    background: var(--color-accent);
    border-radius: 1px 1px 0 0;
    opacity: 0.85;
  }

  .chart__axis {
    display: flex;
    justify-content: space-between;
    margin-top: var(--space-xs);
    font-family: var(--font-outlier);
    font-size: var(--text-xs);
    color: var(--color-neutral);
  }

  .table { overflow-x: auto; }

  .table__head,
  .table__row {
    display: grid;
    grid-template-columns: minmax(12rem, 1fr) 4rem 4rem 6rem 4rem;
    gap: var(--space-sm);
    align-items: baseline;
    min-width: 34rem;
    padding-block: var(--space-sm);
    border-bottom: var(--rule-hair) solid var(--color-rule-2);
  }

  .table__head {
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--color-neutral);
  }

  .table__title { overflow-wrap: anywhere; }

  .table__num {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    font-variant-numeric: tabular-nums;
    color: var(--color-ink-2);
    text-align: end;
  }

  .empty { color: var(--color-muted); }

  .state {
    display: grid;
    gap: var(--space-sm);
    justify-items: start;
    padding-block: var(--space-2xl);
    color: var(--color-muted);
  }

  .state--error { color: var(--color-danger); }

  .state__meter {
    width: 7.5rem;
    height: 2px;
    background: var(--color-rule);
    overflow: hidden;
  }

  .state__meter::after {
    content: "";
    display: block;
    width: 40%;
    height: 100%;
    background: var(--color-accent);
    animation: sweep 1.1s var(--ease-in-out) infinite;
  }

  @keyframes sweep {
    0%   { transform: translateX(-110%); }
    100% { transform: translateX(260%); }
  }

  @media (prefers-reduced-motion: reduce) {
    .state__meter::after { width: 100%; }
  }

  .btn-ghost {
    min-height: 2.25rem;
    padding-inline: var(--space-sm);
    background: none;
    color: var(--color-muted);
    font-size: var(--text-sm);
    font-weight: 600;
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-pill);
    cursor: pointer;
  }

  @media (hover: hover) {
    .btn-ghost:hover { background: var(--color-paper-2); color: var(--color-ink); }
  }
</style>
