<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import PosterThumb from './PosterThumb.svelte';
  import { copyText } from '../clipboard.js';
  import { migrateKey } from '../storage.js';
  import qrcode from 'qrcode-generator';

  export let apiKey;

  const dispatch = createEventDispatcher();

  const EXPIRY = [
    { value: '1440', label: '24 h' },
    { value: '10080', label: '7 days' },
    { value: '43200', label: '30 days' },
    { value: 'never', label: 'Never' },
    { value: 'custom', label: 'Custom' }
  ];

  const HEIGHTS = [
    { value: '', label: 'Auto' },
    { value: '1080', label: '1080p' },
    { value: '720', label: '720p' },
    { value: '480', label: '480p' },
    { value: '360', label: '360p' }
  ];

  const BITRATES = [
    { value: '', label: 'Auto' },
    { value: '20000000', label: '20 Mbps' },
    { value: '12000000', label: '12 Mbps' },
    { value: '8000000', label: '8 Mbps' },
    { value: '4000000', label: '4 Mbps' },
    { value: '2000000', label: '2 Mbps' }
  ];

  const absoluteTime = new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short'
  });

  // Recents are a local convenience, not server state. They survive a reload
  // and are cleared by the admin, never by the backend.
  migrateKey(localStorage, 'jfshare:recent-items', 'jellyshare:recent-items');
  migrateKey(localStorage, 'jfshare:last-item', 'jellyshare:last-item');

  const RECENT_KEY = 'jellyshare:recent-items';
  const LAST_KEY = 'jellyshare:last-item';
  const RECENT_LIMIT = 8;

  let selectedItem = null;
  let recentItems = [];
  let lastItem = null;
  let searchQuery = '';
  let results = [];
  let searching = false;
  let searchError = '';
  let searchGeneration = 0;
  let searchTimer;

  // The picker is a tree. Seasons and episodes load on first expand and stay
  // cached until Refresh or a new search result set replaces them.
  let expandedSeries = new Set();
  let expandedSeasons = new Set();
  let seriesSeasons = {};
  let seasonEpisodes = {};
  let seriesEpisodes = {};
  let seriesFallback = {};
  let seasonsLoading = {};
  let seasonsError = {};
  let episodesLoading = {};
  let episodesError = {};

  let expiryChoice = '43200';
  let customDays = '30';
  let maxPlays = '';
  let maxConcurrent = '';
  let maxHeight = '';
  let maxBitrate = '';
  let password = '';
  let allowDownload = true;

  let formError = '';
  let submitting = false;
  let created = null;
  let createdPassword = '';
  let qrSvg = '';
  let copied = false;
  let copiedDetails = false;
  let copyTimer;
  let detailsTimer;
  let copyError = '';

  onMount(() => {
    loadRecents();
    runSearch('');
    return () => {
      clearTimeout(searchTimer);
      clearTimeout(copyTimer);
    };
  });

  export function reload() {
    if (created) return;
    clearLibraryCache();
    runSearch(searchQuery);
  }

  function clearLibraryCache() {
    expandedSeries = new Set();
    expandedSeasons = new Set();
    seriesSeasons = {};
    seasonEpisodes = {};
    seriesEpisodes = {};
    seriesFallback = {};
    seasonsLoading = {};
    seasonsError = {};
    episodesLoading = {};
    episodesError = {};
  }

  function isStoredItem(item) {
    return item && typeof item.id === 'string' && typeof item.name === 'string' && typeof item.type === 'string';
  }

  function readStored(key) {
    try {
      const raw = localStorage.getItem(key);
      return raw ? JSON.parse(raw) : null;
    } catch (e) {
      return null;
    }
  }

  function writeStored(key, value) {
    try {
      localStorage.setItem(key, JSON.stringify(value));
    } catch (e) {
      // Private mode or a full quota: recents are a convenience, not a requirement.
    }
  }

  function loadRecents() {
    const stored = readStored(RECENT_KEY);
    if (Array.isArray(stored)) {
      recentItems = stored.filter(isStoredItem).slice(0, RECENT_LIMIT);
    }
    const last = readStored(LAST_KEY);
    if (isStoredItem(last)) lastItem = last;
  }

  function rememberChoice(item) {
    lastItem = item;
    writeStored(LAST_KEY, item);
  }

  function rememberShare(item) {
    const next = [item, ...recentItems.filter((x) => x.id !== item.id)].slice(0, RECENT_LIMIT);
    recentItems = next;
    writeStored(RECENT_KEY, next);
  }

  function clearRecents() {
    recentItems = [];
    lastItem = null;
    try {
      localStorage.removeItem(RECENT_KEY);
      localStorage.removeItem(LAST_KEY);
    } catch (e) {
      // Nothing to clear if storage is unavailable.
    }
  }

  // The last chosen item leads the list even before it has been shared, so
  // returning to Create picks up where the admin left off.
  $: recentList = (() => {
    const seen = new Set();
    const list = [];
    for (const item of [lastItem, ...recentItems]) {
      if (!isStoredItem(item) || seen.has(item.id)) continue;
      seen.add(item.id);
      list.push(item);
    }
    return list.slice(0, RECENT_LIMIT);
  })();

  async function fetchList(url) {
    const response = await fetch(url, { headers: { 'X-Backend-Key': apiKey } });
    if (!response.ok) throw new Error('request failed');
    return response.json();
  }

  function toggleIn(set, id) {
    const next = new Set(set);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    return next;
  }

  async function toggleSeries(id) {
    expandedSeries = toggleIn(expandedSeries, id);
    if (expandedSeries.has(id)) await loadSeasons(id);
  }

  async function toggleSeason(id) {
    expandedSeasons = toggleIn(expandedSeasons, id);
    if (expandedSeasons.has(id)) await loadEpisodes(id);
  }

  async function loadSeasons(seriesID, force = false) {
    if (!force && (seriesSeasons[seriesID] || seasonsLoading[seriesID])) return;

    seasonsLoading = { ...seasonsLoading, [seriesID]: true };
    seasonsError = { ...seasonsError, [seriesID]: '' };
    try {
      const data = await fetchList(`/api/admin/jellyfin/series/${seriesID}/seasons`);
      const seasons = data.seasons || [];
      if (seasons.length === 0) {
        // Some libraries have episodes without Season objects. Show them as one
        // group rather than leaving the series looking empty.
        const fallback = await fetchList(`/api/admin/jellyfin/series/${seriesID}/episodes`);
        seriesEpisodes = { ...seriesEpisodes, [seriesID]: fallback.episodes || [] };
        seriesFallback = { ...seriesFallback, [seriesID]: true };
      } else {
        seriesSeasons = { ...seriesSeasons, [seriesID]: seasons };
        seriesFallback = { ...seriesFallback, [seriesID]: false };
      }
    } catch (e) {
      seasonsError = { ...seasonsError, [seriesID]: 'Couldn’t load seasons.' };
    } finally {
      seasonsLoading = { ...seasonsLoading, [seriesID]: false };
    }
  }

  async function loadEpisodes(seasonID, force = false) {
    if (!force && (seasonEpisodes[seasonID] || episodesLoading[seasonID])) return;

    episodesLoading = { ...episodesLoading, [seasonID]: true };
    episodesError = { ...episodesError, [seasonID]: '' };
    try {
      const data = await fetchList(`/api/admin/jellyfin/seasons/${seasonID}/episodes`);
      seasonEpisodes = { ...seasonEpisodes, [seasonID]: data.episodes || [] };
    } catch (e) {
      episodesError = { ...episodesError, [seasonID]: 'Couldn’t load episodes.' };
    } finally {
      episodesLoading = { ...episodesLoading, [seasonID]: false };
    }
  }

  function onSearchInput() {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => runSearch(searchQuery), 300);
  }

  async function runSearch(query) {
    const generation = ++searchGeneration;
    searching = true;
    searchError = '';

    try {
      const params = new URLSearchParams({ limit: '20' });
      const trimmed = query.trim();
      if (trimmed) params.set('q', trimmed);

      const response = await fetch(`/api/admin/jellyfin/items?${params}`, {
        headers: { 'X-Backend-Key': apiKey }
      });
      if (!response.ok) throw new Error('search failed');
      const data = await response.json();
      if (generation !== searchGeneration) return;
      results = data.items || [];
    } catch (e) {
      if (generation === searchGeneration) {
        results = [];
        searchError = 'Couldn’t search the Jellyfin library. Try again.';
      }
    } finally {
      if (generation === searchGeneration) searching = false;
    }
  }

  function choose(item) {
    selectedItem = item;
    rememberChoice(item);
    formError = '';
  }

  function changeItem() {
    selectedItem = null;
    formError = '';
    if (results.length === 0) runSearch(searchQuery);
  }

  function pad(value) {
    return String(value).padStart(2, '0');
  }

  function seasonLabel(season) {
    if (season.seasonNumber === 0) return 'Specials';
    return season.name || `Season ${season.seasonNumber}`;
  }

  function episodeLabel(episode) {
    const code = episode.seasonNumber
      ? `S${pad(episode.seasonNumber)}E${pad(episode.episodeNumber || 0)}`
      : '';
    return [code, episode.name].filter(Boolean).join(' · ');
  }

  function itemContext(item) {
    if (item.type === 'Episode') {
      const parts = [];
      if (item.seriesName) parts.push(item.seriesName);
      if (item.seasonNumber) parts.push(`S${pad(item.seasonNumber)}E${pad(item.episodeNumber || 0)}`);
      return parts.join(' · ');
    }
    if (item.type === 'Season') {
      return [item.seriesName, item.seasonNumber ? `Season ${item.seasonNumber}` : '']
        .filter(Boolean)
        .join(' · ');
    }
    return [item.type, item.year].filter(Boolean).join(' · ');
  }

  function formatRuntime(seconds) {
    if (!seconds) return '';
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.round((seconds % 3600) / 60);
    return hours ? `${hours} h ${minutes} min` : `${minutes} min`;
  }

  function formatDate(value) {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? '—' : absoluteTime.format(date);
  }

  function optionalPositive(raw) {
    if (raw === '' || raw == null) return null;
    const value = Number(raw);
    return Number.isInteger(value) && value > 0 ? value : NaN;
  }

  function buildPayload() {
    if (!selectedItem) {
      return { error: 'Choose a film, episode, season, or series first.' };
    }

    const payload = {
      jellyfinItemId: selectedItem.id,
      allowDownload
    };

    if (expiryChoice === 'never') {
      payload.neverExpires = true;
    } else if (expiryChoice === 'custom') {
      const days = Number(customDays);
      if (!Number.isFinite(days) || days <= 0) {
        return { error: 'Enter how many days the link should last.' };
      }
      payload.expiresInMinutes = Math.round(days * 1440);
    } else {
      payload.expiresInMinutes = Number(expiryChoice);
    }

    const plays = optionalPositive(maxPlays);
    if (Number.isNaN(plays)) return { error: 'Play limit must be a whole number of 1 or more.' };
    if (plays != null) payload.maxTotalPlays = plays;

    const concurrent = optionalPositive(maxConcurrent);
    if (Number.isNaN(concurrent)) {
      return { error: 'Watching at once must be a whole number of 1 or more.' };
    }
    if (concurrent != null) payload.maxConcurrentViewers = concurrent;

    if (maxHeight) payload.maxVideoHeight = Number(maxHeight);
    if (maxBitrate) payload.maxVideoBitrate = Number(maxBitrate);
    if (password !== '') payload.password = password;

    return { payload };
  }

  async function submit() {
    formError = '';
    const built = buildPayload();
    if (built.error) {
      formError = built.error;
      return;
    }

    submitting = true;
    try {
      const response = await fetch('/api/admin/shares', {
        method: 'POST',
        headers: {
          'X-Backend-Key': apiKey,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(built.payload)
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || 'Could not create the link.');
      }
      created = data;
      createdPassword = password;
      qrSvg = makeQr(data.publicUrl);
      rememberShare(selectedItem);
      dispatch('created');
    } catch (e) {
      formError = e.message || 'Could not create the link.';
    } finally {
      submitting = false;
    }
  }

  function makeQr(url) {
    const qr = qrcode(0, 'M');
    qr.addData(url);
    qr.make();
    return qr.createSvgTag({ cellSize: 4, margin: 2, scalable: true });
  }

  async function copyCreated() {
    const ok = await copyText(created.publicUrl);
    if (!ok) {
      copyError = 'Couldn’t copy the link. Select the address and copy it yourself.';
      return;
    }
    copyError = '';
    copied = true;
    clearTimeout(copyTimer);
    copyTimer = setTimeout(() => (copied = false), 1600);
  }

  async function copyDetails() {
    const lines = ['Jellyshare', created.publicUrl];
    if (createdPassword) lines.push(`Password: ${createdPassword}`);
    lines.push(created.expiresAt ? `Expires: ${formatDate(created.expiresAt)}` : 'Never expires');

    const ok = await copyText(lines.join('\n'));
    if (!ok) {
      copyError = 'Couldn’t copy the details. Select the address and copy it yourself.';
      return;
    }
    copyError = '';
    copiedDetails = true;
    clearTimeout(detailsTimer);
    detailsTimer = setTimeout(() => (copiedDetails = false), 1600);
  }

  function createAnother() {
    created = null;
    createdPassword = '';
    qrSvg = '';
    copied = false;
    copiedDetails = false;
    copyError = '';
    selectedItem = null;
    searchQuery = '';
    results = [];
    searchError = '';
    clearLibraryCache();
    expiryChoice = '43200';
    customDays = '30';
    maxPlays = '';
    maxConcurrent = '';
    maxHeight = '';
    maxBitrate = '';
    password = '';
    allowDownload = true;
    formError = '';
    runSearch('');
  }
</script>

<main class="create">
  <p class="create__lede">
    Make a link to a film, episode, season or series, then hand it to someone
    who has no Jellyfin account.
  </p>

  {#if created}
    <section class="success" aria-live="polite">
      <p class="success__head">Link created.</p>
      <p class="success__note">Anyone with this address can open the share page.</p>

      <div class="success__url">
        <code>{created.publicUrl}</code>
        <button class="btn-ghost" type="button" on:click={copyCreated}>
          {copied ? 'Copied' : 'Copy link'}
        </button>
        <button class="btn-ghost" type="button" on:click={copyDetails}>
          {copiedDetails ? 'Copied' : createdPassword ? 'Copy link + password' : 'Copy details'}
        </button>
      </div>
      {#if copyError}<p class="copy-error" role="alert">{copyError}</p>{/if}

      {#if qrSvg}
        <div class="success__qr">
          <div class="success__qr-code" aria-label="QR code for the share link">{@html qrSvg}</div>
          <p class="success__qr-note">Scan to open the link on a phone.</p>
        </div>
      {/if}

      <p class="success__meta">
        {#if created.expiresAt}Expires {formatDate(created.expiresAt)}{:else}Never expires{/if}
        {#if created.maxTotalPlays}· {created.maxTotalPlays} {created.maxTotalPlays === 1 ? 'play' : 'plays'}{/if}
        {#if created.maxConcurrentViewers}· {created.maxConcurrentViewers} watching at once{/if}
      </p>

      <div class="success__actions">
        <button class="btn-primary" type="button" on:click={createAnother}>Create another</button>
        <button class="btn-ghost" type="button" on:click={() => dispatch('viewShares')}>View in Shares</button>
      </div>
    </section>
  {:else}
    <form class="form" on:submit|preventDefault={submit}>
      <section class="section">
        <h2 class="section__head">Item</h2>

        {#if selectedItem}
          <div class="chosen">
            <span class="chosen__poster">
              <PosterThumb {apiKey} itemId={selectedItem.id} alt="" maxWidth={300} />
            </span>
            <div class="chosen__body">
              <p class="chosen__title">{selectedItem.name}</p>
              <p class="chosen__meta">
                {itemContext(selectedItem)}
                {#if selectedItem.runtimeSeconds}· {formatRuntime(selectedItem.runtimeSeconds)}{/if}
              </p>
              <button class="btn-ghost" type="button" on:click={changeItem}>Change item</button>
            </div>
          </div>
        {:else}
          <label class="search">
            <span class="label">Search the Jellyfin library</span>
            <input
              class="input"
              type="search"
              bind:value={searchQuery}
              on:input={onSearchInput}
              placeholder="Film or series"
              autocomplete="off"
            />
          </label>

          {#if !searchQuery.trim() && recentList.length > 0}
            <div class="recent">
              <div class="recent__head">
                <h3 class="recent__title">Recent</h3>
                <button class="recent__clear" type="button" on:click={clearRecents}>Clear</button>
              </div>
              <ul class="pick pick--recent">
                {#each recentList as item (item.id)}
                  <li>
                    <button class="pick__row" type="button" on:click={() => choose(item)}>
                      <span class="pick__poster">
                        <PosterThumb {apiKey} itemId={item.id} alt="" maxWidth={160} />
                      </span>
                      <span class="pick__text">
                        <span class="pick__title">
                          {item.name}
                          {#if lastItem?.id === item.id}<span class="recent__badge">Last</span>{/if}
                        </span>
                        <span class="pick__sub">{itemContext(item)}</span>
                      </span>
                    </button>
                  </li>
                {/each}
              </ul>
            </div>
          {/if}

          <div class="results" aria-live="polite">
            {#if searching}
              <div class="pick-state">
                <span class="meter" aria-hidden="true"></span>
                <p>Searching</p>
              </div>
            {:else if searchError}
              <div class="pick-state pick-state--error">
                <p>{searchError}</p>
                <button class="btn-ghost" type="button" on:click={() => runSearch(searchQuery)}>Try again</button>
              </div>
            {:else if results.length === 0}
              <p class="pick-state">
                No matches{#if searchQuery.trim()} for “{searchQuery.trim()}”{/if}.
                Try a film or series title.
              </p>
            {:else}
              <ul class="pick">
                {#each results as item (item.id)}
                  {#if item.type === 'Series'}
                    <li class="node">
                      <div class="node__row">
                        <button
                          class="node__toggle"
                          type="button"
                          aria-expanded={expandedSeries.has(item.id)}
                          aria-controls="series-{item.id}"
                          on:click={() => toggleSeries(item.id)}
                        >
                          <svg class="chevron" class:chevron--open={expandedSeries.has(item.id)} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                            <path d="M9 6l6 6-6 6" />
                          </svg>
                          <span class="pick__poster">
                            <PosterThumb {apiKey} itemId={item.id} alt="" maxWidth={160} />
                          </span>
                          <span class="pick__text">
                            <span class="pick__title">{item.name}</span>
                            <span class="pick__sub">{itemContext(item)}</span>
                          </span>
                        </button>
                        <button class="share-btn" type="button" on:click={() => choose(item)}>Share series</button>
                      </div>

                      {#if expandedSeries.has(item.id)}
                        <div class="branch" id="series-{item.id}">
                          {#if seasonsLoading[item.id]}
                            <div class="pick-state">
                              <span class="meter" aria-hidden="true"></span>
                              <p>Loading seasons</p>
                            </div>
                          {:else if seasonsError[item.id]}
                            <div class="pick-state pick-state--error">
                              <p>{seasonsError[item.id]}</p>
                              <button class="btn-ghost" type="button" on:click={() => loadSeasons(item.id, true)}>Try again</button>
                            </div>
                          {:else if seriesFallback[item.id]}
                            <p class="branch__note">No seasons found; showing episodes.</p>
                            {#each seriesEpisodes[item.id] || [] as episode (episode.id)}
                              <button class="pick__row pick__row--episode" type="button" on:click={() => choose(episode)}>
                                <span class="pick__poster pick__poster--small">
                                  <PosterThumb {apiKey} itemId={episode.id} alt="" maxWidth={120} />
                                </span>
                                <span class="pick__text">
                                  <span class="pick__title">{episode.name}</span>
                                  <span class="pick__sub">{episodeLabel(episode)}</span>
                                </span>
                              </button>
                            {:else}
                              <p class="branch__empty">No episodes found in Jellyfin.</p>
                            {/each}
                          {:else}
                            {#each seriesSeasons[item.id] || [] as season (season.id)}
                              <div class="node node--season">
                                <div class="node__row">
                                  <button
                                    class="node__toggle node__toggle--season"
                                    type="button"
                                    aria-expanded={expandedSeasons.has(season.id)}
                                    aria-controls="season-{season.id}"
                                    on:click={() => toggleSeason(season.id)}
                                  >
                                    <svg class="chevron" class:chevron--open={expandedSeasons.has(season.id)} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
                                      <path d="M9 6l6 6-6 6" />
                                    </svg>
                                    <span class="pick__text">
                                      <span class="pick__title">{seasonLabel(season)}</span>
                                    </span>
                                  </button>
                                  <button class="share-btn" type="button" on:click={() => choose(season)}>Share season</button>
                                </div>

                                {#if expandedSeasons.has(season.id)}
                                  <div class="branch branch--episodes" id="season-{season.id}">
                                    {#if episodesLoading[season.id]}
                                      <div class="pick-state">
                                        <span class="meter" aria-hidden="true"></span>
                                        <p>Loading episodes</p>
                                      </div>
                                    {:else if episodesError[season.id]}
                                      <div class="pick-state pick-state--error">
                                        <p>{episodesError[season.id]}</p>
                                        <button class="btn-ghost" type="button" on:click={() => loadEpisodes(season.id, true)}>Try again</button>
                                      </div>
                                    {:else}
                                      {#each seasonEpisodes[season.id] || [] as episode (episode.id)}
                                        <button class="pick__row pick__row--episode" type="button" on:click={() => choose(episode)}>
                                          <span class="pick__poster pick__poster--small">
                                            <PosterThumb {apiKey} itemId={episode.id} alt="" maxWidth={120} />
                                          </span>
                                          <span class="pick__text">
                                            <span class="pick__title">{episode.name}</span>
                                            <span class="pick__sub">{episodeLabel(episode)}</span>
                                          </span>
                                        </button>
                                      {:else}
                                        <p class="branch__empty">No episodes found in this season.</p>
                                      {/each}
                                    {/if}
                                  </div>
                                {/if}
                              </div>
                            {:else}
                              <p class="branch__empty">No seasons found in Jellyfin.</p>
                            {/each}
                          {/if}
                        </div>
                      {/if}
                    </li>
                  {:else}
                    <li>
                      <button class="pick__row" type="button" on:click={() => choose(item)}>
                        <span class="pick__poster">
                          <PosterThumb {apiKey} itemId={item.id} alt="" maxWidth={160} />
                        </span>
                        <span class="pick__text">
                          <span class="pick__title">{item.name}</span>
                          <span class="pick__sub">{itemContext(item)}</span>
                        </span>
                      </button>
                    </li>
                  {/if}
                {/each}
              </ul>
            {/if}
          </div>
        {/if}
      </section>

      <section class="section">
        <h2 class="section__head">Link terms</h2>

        <div class="field">
          <span class="label">Expires</span>
          <div class="chips">
            {#each EXPIRY as option (option.value)}
              <button
                class="chip"
                class:chip--active={expiryChoice === option.value}
                type="button"
                aria-pressed={expiryChoice === option.value}
                on:click={() => (expiryChoice = option.value)}
              >{option.label}</button>
            {/each}
          </div>
        </div>

        {#if expiryChoice === 'custom'}
          <label class="field field--narrow">
            <span class="label">Days from now</span>
            <input class="input" type="number" min="1" step="1" bind:value={customDays} />
          </label>
        {/if}

        <div class="pair">
          <label class="field">
            <span class="label">Play limit <span class="hint">optional</span></span>
            <input class="input" type="number" min="1" step="1" bind:value={maxPlays} placeholder="Unlimited" />
          </label>
          <label class="field">
            <span class="label">Watching at once <span class="hint">optional</span></span>
            <input class="input" type="number" min="1" step="1" bind:value={maxConcurrent} placeholder="Unlimited" />
          </label>
        </div>
      </section>

      <section class="section">
        <h2 class="section__head">Playback</h2>
        <div class="pair">
          <label class="field">
            <span class="label">Maximum picture</span>
            <select class="input" bind:value={maxHeight}>
              {#each HEIGHTS as option (option.value)}
                <option value={option.value}>{option.label}</option>
              {/each}
            </select>
          </label>
          <label class="field">
            <span class="label">Maximum bitrate</span>
            <select class="input" bind:value={maxBitrate}>
              {#each BITRATES as option (option.value)}
                <option value={option.value}>{option.label}</option>
              {/each}
            </select>
          </label>
        </div>
      </section>

      <section class="section">
        <h2 class="section__head">Access</h2>

        <label class="field field--narrow">
          <span class="label">Password <span class="hint">optional</span></span>
          <input class="input" type="password" bind:value={password} autocomplete="new-password" />
        </label>

        <label class="check">
          <input type="checkbox" bind:checked={allowDownload} />
          <span>Allow downloads</span>
        </label>
      </section>

      {#if formError}
        <p class="banner" role="alert">{formError}</p>
      {/if}

      <div class="submit">
        <button class="btn-primary" type="submit" disabled={submitting}>
          {submitting ? 'Creating' : 'Create link'}
        </button>
        {#if !selectedItem}
          <p class="submit__hint">Choose an item first.</p>
        {/if}
      </div>
    </form>
  {/if}
</main>

<style>
  .create {
    padding-block: calc(var(--space-3xl) + var(--space-md)) var(--space-2xl);
    max-width: 78rem;
    width: 100%;
  }

  .create__lede {
    max-width: 54ch;
    color: var(--color-muted);
    margin-bottom: var(--space-2xl);
  }

  .form {
    display: grid;
    gap: var(--space-2xl);
    max-width: 46rem;
  }

  .section {
    display: grid;
    gap: var(--space-md);
    padding-top: var(--space-lg);
    border-top: var(--rule-hair) solid var(--color-rule-2);
  }

  .section__head {
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--color-neutral);
  }

  .label {
    font-size: var(--text-sm);
    font-weight: 600;
    letter-spacing: 0.02em;
    color: var(--color-ink-2);
  }

  .hint { font-weight: 400; color: var(--color-neutral); }

  .field { display: grid; gap: var(--space-2xs); }

  .field--narrow { max-width: 18rem; }

  .pair {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(13rem, 1fr));
    gap: var(--space-md);
  }

  .input {
    width: 100%;
    min-height: var(--control-h);
    padding: var(--space-xs) var(--space-sm);
    background: var(--color-paper-2);
    color: var(--color-ink);
    font-family: var(--font-outlier);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-input);
    outline: 2px solid transparent;
    outline-offset: 1px;
    transition: background-color var(--dur-micro) var(--ease-out),
                border-color var(--dur-micro) var(--ease-out);
  }

  .input::placeholder { color: var(--color-neutral); }

  @media (hover: hover) {
    .input:hover { background: var(--color-paper-3); }
  }

  .input:focus-visible {
    outline-color: var(--color-focus);
    border-color: var(--color-ink-2);
  }

  .chips { display: flex; flex-wrap: wrap; gap: var(--space-xs); }

  .chip {
    min-height: 2.1rem;
    padding-inline: var(--space-md);
    background: none;
    color: var(--color-muted);
    font-size: var(--text-sm);
    font-weight: 600;
    white-space: nowrap;
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-pill);
    cursor: pointer;
    transition: background-color var(--dur-micro) var(--ease-out),
                color var(--dur-micro) var(--ease-out);
  }

  .chip--active {
    background: var(--color-paper-2);
    color: var(--color-ink);
    border-color: var(--color-ink-2);
  }

  @media (hover: hover) {
    .chip:hover:not(.chip--active) { background: var(--color-paper-2); color: var(--color-ink); }
  }

  .chip:active { transform: translateY(1px); }

  .search { display: grid; gap: var(--space-2xs); }

  .recent { max-width: 46rem; margin-top: var(--space-md); }

  .recent__head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: var(--space-sm);
    margin-bottom: var(--space-2xs);
  }

  .recent__title {
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--color-neutral);
  }

  .recent__clear {
    padding: 0;
    background: none;
    color: var(--color-muted);
    font-size: var(--text-sm);
    border: none;
    border-bottom: var(--rule-hair) solid var(--color-rule);
    cursor: pointer;
  }

  @media (hover: hover) {
    .recent__clear:hover { color: var(--color-ink); border-bottom-color: var(--color-ink-2); }
  }

  .recent__badge {
    margin-inline-start: var(--space-2xs);
    padding: 0.05rem 0.35rem;
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    color: var(--color-muted);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-pill);
  }

  .pick--recent { max-height: none; }

  .results { max-width: 46rem; }

  .pick { list-style: none; margin: 0; padding: 0; max-height: 24rem; overflow-y: auto; }

  .node { display: block; }

  .node__row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-xs);
    border-bottom: var(--rule-hair) solid var(--color-rule-2);
  }

  .node__toggle {
    display: flex;
    flex: 1;
    align-items: center;
    gap: var(--space-sm);
    min-width: 0;
    padding: var(--space-xs);
    background: none;
    color: var(--color-ink);
    text-align: start;
    border: none;
    cursor: pointer;
  }

  .node__toggle--season { padding-block: var(--space-2xs); }

  .chevron {
    flex: none;
    width: 0.9rem;
    height: 0.9rem;
    color: var(--color-muted);
    transition: transform var(--dur-micro) var(--ease-out);
  }

  .chevron--open { transform: rotate(90deg); }

  .share-btn {
    flex: none;
    min-height: 2rem;
    margin-inline-end: var(--space-xs);
    padding-inline: var(--space-sm);
    background: none;
    color: var(--color-muted);
    font-size: var(--text-sm);
    font-weight: 600;
    white-space: nowrap;
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-pill);
    cursor: pointer;
  }

  @media (hover: hover) {
    .node__toggle:hover .pick__title { color: var(--color-accent); }
    .share-btn:hover { background: var(--color-paper-2); color: var(--color-ink); }
  }

  .share-btn:active { transform: translateY(1px); }

  .branch {
    margin-inline-start: 1.35rem;
    padding-inline-start: var(--space-sm);
    border-inline-start: var(--rule-hair) solid var(--color-rule-2);
  }

  .branch--episodes { margin-inline-start: 0.9rem; }

  .branch__note,
  .branch__empty {
    padding: var(--space-sm) var(--space-xs);
    font-size: var(--text-sm);
    color: var(--color-muted);
  }

  .pick__poster--small { width: 2rem; }

  .pick__row--episode { padding-block: var(--space-2xs); }

  .pick__row {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    width: 100%;
    padding: var(--space-xs);
    background: none;
    color: var(--color-ink);
    text-align: start;
    border: none;
    border-bottom: var(--rule-hair) solid var(--color-rule-2);
    cursor: pointer;
    transition: background-color var(--dur-micro) var(--ease-out);
  }

  @media (hover: hover) {
    .pick__row:hover { background: var(--color-paper-2); }
  }

  .pick__poster {
    flex: none;
    display: block;
    width: 2.6rem;
    aspect-ratio: 2 / 3;
    overflow: hidden;
    border-radius: calc(var(--radius-input) / 1.5);
  }

  .pick__text { display: grid; gap: var(--space-3xs); min-width: 0; }

  .pick__title { font-weight: 600; overflow-wrap: anywhere; }

  .pick__sub {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    color: var(--color-muted);
  }

  .pick-state {
    display: grid;
    gap: var(--space-sm);
    justify-items: start;
    padding-block: var(--space-xl);
    color: var(--color-muted);
  }

  .pick-state--error { color: var(--color-danger); }

  .meter {
    width: 7.5rem;
    height: 2px;
    background: var(--color-rule);
    overflow: hidden;
    position: relative;
  }

  .meter::after {
    content: "";
    position: absolute;
    inset: 0 auto 0 0;
    width: 40%;
    background: var(--color-accent);
    animation: sweep 1.1s var(--ease-in-out) infinite;
  }

  @keyframes sweep {
    0%   { transform: translateX(-110%); }
    100% { transform: translateX(260%); }
  }

  @media (prefers-reduced-motion: reduce) {
    .meter::after { width: 100%; }
  }

  .chosen {
    display: flex;
    gap: var(--space-md);
    align-items: flex-start;
    padding: var(--space-md);
    background: var(--color-paper-2);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-card);
  }

  .chosen__poster {
    flex: none;
    display: block;
    width: 5rem;
    aspect-ratio: 2 / 3;
    overflow: hidden;
    border-radius: var(--radius-input);
  }

  .chosen__body { display: grid; gap: var(--space-xs); justify-items: start; min-width: 0; }

  .chosen__title { font-weight: 600; font-size: var(--text-md); overflow-wrap: anywhere; }

  .chosen__meta {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    color: var(--color-muted);
  }

  .check {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    font-size: var(--text-sm);
    color: var(--color-ink-2);
    cursor: pointer;
  }

  .check input { width: 1rem; height: 1rem; accent-color: var(--color-accent); }

  .banner {
    padding: var(--space-sm) var(--space-md);
    background: var(--color-paper-2);
    border-radius: var(--radius-card);
    color: var(--color-danger);
    font-size: var(--text-sm);
  }

  .submit { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-md); }

  .submit__hint { font-size: var(--text-sm); color: var(--color-muted); }

  .btn-primary, .btn-ghost {
    min-height: var(--control-h);
    padding-inline: var(--space-lg);
    font-size: var(--text-sm);
    font-weight: 600;
    white-space: nowrap;
    border-radius: var(--radius-pill);
    cursor: pointer;
    transition: background-color var(--dur-micro) var(--ease-out),
                color var(--dur-micro) var(--ease-out),
                transform var(--dur-micro) var(--ease-out);
  }

  .btn-primary {
    background: var(--color-accent);
    color: var(--color-accent-ink);
    border: none;
  }

  @media (hover: hover) {
    .btn-primary:hover:not(:disabled) { transform: translateY(-1px); }
    .btn-ghost:hover { background: var(--color-paper-2); color: var(--color-ink); }
  }

  .btn-primary:active:not(:disabled) { transform: translateY(1px); }

  .btn-primary:disabled { opacity: 0.55; cursor: not-allowed; }

  .btn-ghost {
    background: none;
    color: var(--color-muted);
    border: var(--rule-hair) solid var(--color-rule);
  }

  .success {
    display: grid;
    gap: var(--space-md);
    max-width: 46rem;
    padding: var(--space-lg);
    background: var(--color-paper-2);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-card);
  }

  .success__head { font-size: var(--text-lg); font-weight: 600; }

  .success__note { color: var(--color-muted); font-size: var(--text-sm); }

  .success__url {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-xs);
    padding: var(--space-xs);
    background: var(--color-paper-3);
    border-radius: var(--radius-input);
  }

  .success__url code {
    flex: 1;
    min-width: 0;
    padding-inline: var(--space-xs);
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    overflow-wrap: anywhere;
  }

  .copy-error {
    font-size: var(--text-sm);
    color: var(--color-danger);
  }

  .success__qr {
    display: flex;
    align-items: center;
    gap: var(--space-md);
  }

  .success__qr-code {
    flex: none;
    width: 8.5rem;
    padding: var(--space-xs);
    background: #fff;
    border-radius: var(--radius-input);
  }

  .success__qr-code :global(svg) {
    display: block;
    width: 100%;
    height: auto;
  }

  .success__qr-note {
    font-size: var(--text-sm);
    color: var(--color-muted);
  }

  .success__meta {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    color: var(--color-muted);
  }

  .success__actions { display: flex; flex-wrap: wrap; gap: var(--space-xs); }
</style>
