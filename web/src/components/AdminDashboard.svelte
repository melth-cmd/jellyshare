<script>
  import BrandMark from './BrandMark.svelte';
  import CreatePanel from './CreatePanel.svelte';
  import EditShareDialog from './EditShareDialog.svelte';
  import { copyText } from '../clipboard.js';
  import { onMount, tick, createEventDispatcher } from 'svelte';

  export let apiKey;

  const TAB_KEY = 'jfshare_admin_tab';
  let activeTab = 'shares';
  let createPanel;

  const dispatch = createEventDispatcher();

  // The index loads a page at a time, newest first. total and live come from
  // the server, so the counts are right however many rows are loaded.
  const PAGE_SIZE = 50;
  let shares = [];
  let total = 0;
  let live = 0;
  let counts = { active: 0, inactive: 0, all: 0 };
  // Active by default: the index is for links that still work.
  let statusFilter = 'active';
  let hasMore = false;
  let loadingMore = false;
  let moreError = '';
  let loading = true;
  let error = null;
  let selectedShare = null;
  let shareDetails = null;
  let detailsLoading = false;
  let dialogEl;

  // Revoking kills live sessions, so it asks — but in the row, not behind a
  // native confirm() the browser paints over the page.
  let confirmingId = null;
  let revokeError = '';
  // Copy has no visible effect, so the button says what happened, briefly.
  let copiedId = null;
  let copyTimer;

  // Selection is for mass actions only. It is cleared by anything that changes
  // which rows are on screen.
  let selectedIds = new Set();
  let selectionAnchor = null;
  let selectAllEl;
  let massConfirm = null;
  let massBusy = false;
  let massNotice = '';
  let massError = '';
  let editDialog;

  $: if (selectAllEl) {
    selectAllEl.indeterminate =
      selectedIds.size > 0 && !(shares.length > 0 && shares.every((share) => selectedIds.has(share.id)));
  }

  onMount(() => {
    const savedTab = sessionStorage.getItem(TAB_KEY);
    if (savedTab === 'create' || savedTab === 'shares') activeTab = savedTab;
    loadShares();
    return () => clearTimeout(copyTimer);
  });

  function setTab(tab) {
    activeTab = tab;
    sessionStorage.setItem(TAB_KEY, tab);
    if (tab === 'create') clearSelection();
  }

  function refreshCurrent() {
    if (activeTab === 'shares') loadShares();
    else createPanel?.reload();
  }

  // A new link may land in any filter; reload the index in the background so
  // the Shares tab and the counts are current when the admin switches over.
  function onShareCreated() {
    loadShares();
  }

  // The next page continues below the last link held, not at a row number, so
  // links created or cleaned up in between cannot shift or repeat rows.
  async function fetchPage(after) {
    const query = new URLSearchParams({ limit: String(PAGE_SIZE), status: statusFilter });
    if (after) {
      query.set('beforeCreatedAt', after.createdAt);
      query.set('beforeId', after.id);
    }
    const response = await fetch(`/api/admin/shares?${query}`, {
      headers: { 'X-Backend-Key': apiKey }
    });
    if (!response.ok) throw new Error('The backend refused the request. Check the key is still valid.');
    const data = await response.json();
    return {
      shares: data.shares || [],
      total: data.total ?? 0,
      live: data.live ?? 0,
      counts: data.counts || { active: 0, inactive: 0, all: 0 },
      hasMore: !!data.hasMore
    };
  }

  function clearSelection() {
    selectedIds = new Set();
    selectionAnchor = null;
    massConfirm = null;
  }

  async function loadShares() {
    // A page still loading from before belongs to the list being replaced.
    clearSelection();
    loadGeneration += 1;
    loadingMore = false;
    loading = true;
    error = null;
    moreError = '';
    try {
      ({ shares, total, live, counts, hasMore } = await fetchPage(null));
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  function setStatusFilter(status) {
    if (status === statusFilter) return;
    statusFilter = status;
    loadShares();
  }

  function toggleSelection(event, index, id) {
    const next = new Set(selectedIds);
    const checking = !next.has(id);

    if (event.shiftKey && selectionAnchor != null) {
      const [start, end] =
        index < selectionAnchor ? [index, selectionAnchor] : [selectionAnchor, index];
      for (let i = start; i <= end; i += 1) {
        const row = shares[i];
        if (!row) continue;
        if (checking) next.add(row.id);
        else next.delete(row.id);
      }
    } else {
      if (checking) next.add(id);
      else next.delete(id);
      selectionAnchor = index;
    }

    selectedIds = next;
    massConfirm = null;
    massNotice = '';
    massError = '';
  }

  function toggleSelectAll(event) {
    selectedIds = event.target.checked ? new Set(shares.map((share) => share.id)) : new Set();
    selectionAnchor = null;
    massConfirm = null;
    massNotice = '';
    massError = '';
  }

  async function runMassAction(action) {
    if (selectedIds.size === 0) return;
    massBusy = true;
    massError = '';
    massNotice = '';

    try {
      const response = await fetch(`/api/admin/shares/bulk/${action}`, {
        method: 'POST',
        headers: {
          'X-Backend-Key': apiKey,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ ids: [...selectedIds] })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || 'The action did not finish. Try again.');
      }

      const changed = (action === 'revoke' ? data.revoked : data.deleted) || [];
      const skipped = data.skipped || [];
      const verb = action === 'revoke' ? 'revoked' : 'deleted';
      massNotice = `${changed.length} ${verb}${skipped.length ? `, ${skipped.length} skipped` : ''}.`;
      await loadShares();
    } catch (e) {
      massError = e.message || 'The action did not finish. Try again.';
    } finally {
      massBusy = false;
      massConfirm = null;
    }
  }

  function openEdit(share) {
    editDialog?.open(share);
  }

  async function onShareEdited() {
    await loadShares();
    massNotice = 'Link updated.';
  }

  // Bumped by every reload, so a page still loading from before it is dropped
  // instead of being appended to the fresh list.
  let loadGeneration = 0;

  async function loadMore() {
    if (loadingMore) return;
    loadingMore = true;
    moreError = '';
    const generation = loadGeneration;
    try {
      // Awaited before anything is taken from it: the list may have changed
      // meanwhile - a revoke marked in place, or a reload - and must not be put
      // back as it was. The counts stay as the first page and later revokes left
      // them; only the rows and whether more follow come from here.
      const page = await fetchPage(shares[shares.length - 1]);
      if (generation === loadGeneration && page.shares.length) {
        shares = [...shares, ...page.shares];
        hasMore = page.hasMore;
        // Focus moves on to the first link just loaded: the button that had it
        // goes away with the last page, and a keyboard would land back at the top.
        await tick();
        document.querySelector(`.row__item[data-id="${page.shares[0].id}"]`)?.focus();
      } else if (generation === loadGeneration) {
        hasMore = page.hasMore;
      }
    } catch (e) {
      if (generation === loadGeneration) moreError = 'Couldn’t load more links. Try again.';
    } finally {
      if (generation === loadGeneration) loadingMore = false;
    }
  }

  async function viewDetails(share) {
    selectedShare = share;
    shareDetails = null;
    detailsLoading = true;
    dialogEl?.showModal();
    try {
      const response = await fetch(`/api/admin/shares/${share.id}`, {
        headers: { 'X-Backend-Key': apiKey }
      });
      if (!response.ok) throw new Error('Failed to load details');
      shareDetails = await response.json();
    } catch (e) {
      shareDetails = null;
    } finally {
      detailsLoading = false;
    }
  }

  async function revokeShare(share) {
    confirmingId = null;
    revokeError = '';
    try {
      const response = await fetch(`/api/admin/shares/${share.id}/revoke`, {
        method: 'POST',
        headers: { 'X-Backend-Key': apiKey }
      });
      if (!response.ok) throw new Error('revoke failed');
      const wasActive = getStatus(share) === 'active';
      if (wasActive) {
        live = Math.max(0, live - 1);
        counts = { ...counts, active: Math.max(0, counts.active - 1), inactive: counts.inactive + 1 };
      }
      if (wasActive && statusFilter === 'active') {
        // The row no longer belongs in this view. Removing it in place keeps the
        // pages already loaded below it instead of reloading the whole index.
        shares = shares.filter((s) => s.id !== share.id);
        total = Math.max(0, total - 1);
      } else {
        shares = shares.map((s) => (s.id === share.id ? { ...s, revokedAt: new Date().toISOString() } : s));
      }
      if (selectedIds.has(share.id)) {
        selectedIds = new Set([...selectedIds].filter((id) => id !== share.id));
      }
      if (selectedShare?.id === share.id) closeDetails();
    } catch (e) {
      revokeError = `Couldn’t revoke “${share.title}”. The link is still live — try again.`;
    }
  }

  function closeDetails() {
    dialogEl?.close();
  }

  function onDialogClose() {
    selectedShare = null;
    shareDetails = null;
  }

  function onDialogClick(event) {
    // Light-dismiss: a click that lands on the dialog element itself is a
    // click on its backdrop, since the panel fills it.
    if (event.target === dialogEl) closeDetails();
  }

  function logout() {
    dispatch('logout');
  }

  function getStatus(share) {
    if (share.revokedAt) return 'revoked';
    if (share.expiresAt && new Date(share.expiresAt) < new Date()) return 'expired';
    if (share.maxTotalPlays != null && share.totalPlays >= share.maxTotalPlays) return 'spent';
    return 'active';
  }

  // Absolute times render in the reader's own zone, and name it: an admin
  // deciding whether a link is still live should not have to work out whether
  // the server meant UTC. The browser's locale picks the field order.
  const absoluteTime = new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZoneName: 'short'
  });

  function formatDate(dateStr) {
    if (!dateStr) return 'Never';
    const d = new Date(dateStr);
    return Number.isNaN(d.getTime()) ? '—' : absoluteTime.format(d);
  }

  function formatTimeAgo(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now - date;
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days} d ago`;
    if (hours > 0) return `${hours} h ago`;
    if (minutes > 0) return `${minutes} min ago`;
    return 'just now';
  }

  function formatExpiry(dateStr) {
    if (!dateStr) return 'Never';
    const date = new Date(dateStr);
    const now = new Date();
    const diff = date - now;

    if (diff <= 0) return 'Expired';

    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days} d ${hours % 24} h`;
    if (hours > 0) return `${hours} h`;
    return `${Math.floor(diff / (1000 * 60))} min`;
  }

  async function copyLink(share) {
    const url = `${window.location.origin}/s/${share.publicToken}`;
    if (!(await copyText(url))) return;
    clearTimeout(copyTimer);
    copiedId = share.id;
    copyTimer = setTimeout(() => (copiedId = null), 1600);
  }
</script>

<!-- N5 · Floating pill. Content-sized, visibly detached, blurred over the canvas. -->
<nav class="pill" aria-label="Admin">
  <span class="pill__wordmark"><BrandMark /><span class="pill__name">Jellyfin Share</span></span>
  <span class="pill__count" aria-live="polite">{total} {total === 1 ? 'link' : 'links'}</span>
  <div class="pill__tabs" role="tablist" aria-label="Admin sections">
    <button
      class="pill__tab"
      class:pill__tab--active={activeTab === 'create'}
      id="tab-create"
      role="tab"
      aria-selected={activeTab === 'create'}
      aria-controls="panel-create"
      on:click={() => setTab('create')}
    >Create</button>
    <button
      class="pill__tab"
      class:pill__tab--active={activeTab === 'shares'}
      id="tab-shares"
      role="tab"
      aria-selected={activeTab === 'shares'}
      aria-controls="panel-shares"
      on:click={() => setTab('shares')}
    >Shares</button>
  </div>
  <button class="pill__icon" on:click={refreshCurrent} disabled={loading} aria-label="Reload">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
      <path d="M3 12a9 9 0 019-9 9.75 9.75 0 016.74 2.74L21 8M21 3v5h-5M21 12a9 9 0 01-9 9 9.75 9.75 0 01-6.74-2.74L3 16M3 21v-5h5"/>
    </svg>
  </button>
  <button class="pill__out" on:click={logout}>Sign out</button>
</nav>

<div class="admin bloom-ground">
  {#if activeTab === 'shares'}
  <main class="index" id="panel-shares" role="tabpanel" aria-labelledby="tab-shares">
    <p class="index__lede">
      {#if statusFilter === 'active'}
        Links that still work right now.
      {:else if statusFilter === 'inactive'}
        Revoked, expired, and used-up links.
      {:else}
        Every link you have handed out, and what it is doing right now.
      {/if}
      {#if !loading && !error && total > 0}
        <span class="index__live">
          {statusFilter === 'all' ? `${live} of ${total} still live.` : `${total} ${total === 1 ? 'link' : 'links'}.`}
        </span>
      {/if}
    </p>

    <div class="filters" role="group" aria-label="Filter links">
      <button class="filter" class:filter--active={statusFilter === 'active'} aria-pressed={statusFilter === 'active'} on:click={() => setStatusFilter('active')}>
        Active<span class="filter__n">{counts.active}</span>
      </button>
      <button class="filter" class:filter--active={statusFilter === 'inactive'} aria-pressed={statusFilter === 'inactive'} on:click={() => setStatusFilter('inactive')}>
        Inactive<span class="filter__n">{counts.inactive}</span>
      </button>
      <button class="filter" class:filter--active={statusFilter === 'all'} aria-pressed={statusFilter === 'all'} on:click={() => setStatusFilter('all')}>
        All<span class="filter__n">{counts.all}</span>
      </button>
    </div>

    <div class="mass" aria-label="Selected links">
      <label class="mass__all">
        <input
          bind:this={selectAllEl}
          type="checkbox"
          checked={shares.length > 0 && shares.every((share) => selectedIds.has(share.id))}
          on:change={toggleSelectAll}
          disabled={loading || shares.length === 0}
        />
        <span>Select all</span>
      </label>

      <span class="mass__count">{selectedIds.size} selected</span>

      <div class="mass__actions">
        {#if massConfirm === 'revoke'}
          <button class="btn-danger" on:click={() => runMassAction('revoke')} disabled={massBusy}>
            Revoke {selectedIds.size}
          </button>
          <button class="btn-ghost" on:click={() => (massConfirm = null)} disabled={massBusy}>Cancel</button>
        {:else if massConfirm === 'delete'}
          <button class="btn-danger" on:click={() => runMassAction('delete')} disabled={massBusy}>
            Delete {selectedIds.size}
          </button>
          <button class="btn-ghost" on:click={() => (massConfirm = null)} disabled={massBusy}>Cancel</button>
        {:else}
          <button class="btn-ghost" on:click={() => (massConfirm = 'revoke')} disabled={selectedIds.size === 0 || massBusy}>
            Revoke selected
          </button>
          <button class="btn-ghost" on:click={() => (massConfirm = 'delete')} disabled={selectedIds.size === 0 || massBusy}>
            Delete selected
          </button>
          <button class="btn-ghost" on:click={clearSelection} disabled={selectedIds.size === 0 || massBusy}>Clear</button>
        {/if}
      </div>

      {#if massConfirm}
        <p class="mass__warning">
          {massConfirm === 'delete'
            ? 'Only inactive links are deleted. Active links are left alone.'
            : 'Anyone watching one of these links is disconnected.'}
        </p>
      {/if}
    </div>

    {#if massNotice}<p class="banner banner--ok" role="status">{massNotice}</p>{/if}
    {#if massError}<p class="banner" role="alert">{massError}</p>{/if}

    {#if loading}
      <div class="state">
        <div class="state__meter"></div>
        <p>Reading the index</p>
      </div>
    {:else if error}
      <div class="state state--error">
        <p>{error}</p>
        <button class="btn-ghost" on:click={loadShares}>Try again</button>
      </div>
    {:else if shares.length === 0}
      <div class="state">
        {#if statusFilter === 'active'}
          <p class="state__head">No live links.</p>
          <p>Create one from the Create tab and it will appear here.</p>
        {:else if statusFilter === 'inactive'}
          <p class="state__head">Nothing inactive.</p>
          <p>Revoked, expired, and used-up links collect here.</p>
        {:else}
          <p class="state__head">No links yet.</p>
          <p>Create one from the Create tab and it will appear here.</p>
        {/if}
      </div>
    {:else}
      {#if revokeError}
        <p class="banner">{revokeError}</p>
      {/if}

      <div class="labels" aria-hidden="true">
        <span></span><span>Item</span><span>Status</span><span>Plays</span><span>Expires</span><span></span>
      </div>

      <ul class="rows">
        {#each shares as share, index (share.id)}
          {@const status = getStatus(share)}
          <li class="row" class:row--spent={status !== 'active'}>
            <span class="row__check">
              <input
                type="checkbox"
                checked={selectedIds.has(share.id)}
                on:click={(event) => toggleSelection(event, index, share.id)}
                aria-label={`Select ${share.title}`}
              />
            </span>
            <button class="row__item" data-id={share.id} on:click={() => viewDetails(share)}>
              <span class="row__title">{share.title}</span>
              <span class="row__sub">
                {share.itemType} · created {formatTimeAgo(share.createdAt)}{#if share.hasPassword}{' · password'}{/if}
              </span>
            </button>

            <span class="row__status" data-status={status}>
              <span class="dot" aria-hidden="true"></span>{status}
            </span>

            <span class="row__num">
              {share.totalPlays}{#if share.maxTotalPlays}&thinsp;/&thinsp;{share.maxTotalPlays}{/if}
              {#if share.currentConcurrentViewers > 0}
                <span class="row__watching">{share.currentConcurrentViewers} watching</span>
              {/if}
            </span>

            <span class="row__num row__expiry">
              {status === 'revoked' ? 'Revoked' : formatExpiry(share.expiresAt)}
            </span>

            <span class="row__actions">
              {#if confirmingId === share.id}
                <span class="confirm">
                  <button class="btn-danger" on:click={() => revokeShare(share)}>Revoke it</button>
                  <button class="btn-ghost" on:click={() => (confirmingId = null)}>Keep</button>
                </span>
              {:else}
                <button class="btn-ghost" on:click={() => copyLink(share)}>
                  {copiedId === share.id ? 'Copied' : 'Copy link'}
                </button>
                <button class="btn-ghost" on:click={() => openEdit(share)}>Edit</button>
                {#if status === 'active'}
                  <button class="btn-ghost btn-ghost--warn" on:click={() => (confirmingId = share.id)}>
                    Revoke
                  </button>
                {/if}
              {/if}
            </span>
          </li>
        {/each}
      </ul>

      {#if hasMore}
        <div class="more">
          <button class="btn-ghost" on:click={loadMore} disabled={loadingMore}>
            {loadingMore ? 'Loading' : 'Show older links'}
          </button>
          <span class="more__left">{shares.length} of {total} shown</span>
        </div>
      {/if}
      {#if moreError}<p class="banner" role="alert">{moreError}</p>{/if}
    {/if}
  </main>
  {:else}
    <div id="panel-create" role="tabpanel" aria-labelledby="tab-create">
      <CreatePanel
        {apiKey}
        bind:this={createPanel}
        on:created={onShareCreated}
        on:viewShares={() => setTab('shares')}
      />
    </div>
  {/if}

  <!-- Ft4 · Dense colophon -->
  <footer class="colophon">
    <p class="colophon__line">
      <a class="foot-link" href="https://github.com/melth-cmd/jellyfin-share-backend" target="_blank" rel="noopener noreferrer">Jellyfin Share</a>
      · admin · {counts.all} indexed, {counts.active} live
    </p>
    <p class="colophon__note">
      The backend key is kept in this browser’s localStorage and sent only as an
      X-Backend-Key header. Signing out clears it.
    </p>
  </footer>
</div>

<!-- Light-dismiss on backdrop click. <dialog> already handles Escape natively and
     the panel carries an explicit close button, so the keyboard path is covered. -->
<!-- svelte-ignore a11y-click-events-have-key-events a11y-no-noninteractive-element-interactions -->
<dialog bind:this={dialogEl} class="sheet" on:close={onDialogClose} on:click={onDialogClick}>
  <div class="sheet__panel">
    <header class="sheet__head">
      <h2>{selectedShare?.title || ''}</h2>
      <div class="sheet__tools">
        <button class="btn-ghost" on:click={() => { closeDetails(); openEdit(selectedShare); }}>Edit link</button>
        <button class="pill__icon" on:click={closeDetails} aria-label="Close details">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
          <path d="M18 6L6 18M6 6l12 12"/>
        </svg>
        </button>
      </div>
    </header>

    {#if detailsLoading}
      <div class="state"><div class="state__meter"></div><p>Loading</p></div>
    {:else if shareDetails}
      <dl class="spec">
        <div><dt>Token</dt><dd class="spec__mono">{shareDetails.share.publicToken}</dd></div>
        <div><dt>Item type</dt><dd>{shareDetails.share.itemType}</dd></div>
        <div>
          <dt>Total plays</dt>
          <dd class="spec__mono">
            {shareDetails.share.totalPlays}{#if shareDetails.share.maxTotalPlays}&thinsp;/&thinsp;{shareDetails.share.maxTotalPlays.Int64}{/if}
          </dd>
        </div>
        <div>
          <dt>Watching now</dt>
          <dd class="spec__mono">
            {shareDetails.share.currentConcurrentViewers}{#if shareDetails.share.maxConcurrentViewers}&thinsp;/&thinsp;{shareDetails.share.maxConcurrentViewers.Int64}{/if}
          </dd>
        </div>
        <div>
          <dt>Created</dt>
          <dd class="spec__mono">
            <time datetime={shareDetails.share.createdAt}>{formatDate(shareDetails.share.createdAt)}</time>
          </dd>
        </div>
        <div>
          <dt>Expires</dt>
          <dd class="spec__mono">
            {#if shareDetails.share.expiresAt}
              <time datetime={shareDetails.share.expiresAt}>{formatDate(shareDetails.share.expiresAt)}</time>
            {:else}
              Never
            {/if}
          </dd>
        </div>
      </dl>

      <h3 class="sheet__sub">
        Sessions
        {#if shareDetails.sessions?.length}<span class="sheet__n">{shareDetails.sessions.length}</span>{/if}
      </h3>

      {#if shareDetails.sessions && shareDetails.sessions.length > 0}
        <ul class="sessions">
          {#each shareDetails.sessions as session}
            <li class="session">
              <span class="session__state" class:session__state--live={!session.finishedAt}>
                {session.finishedAt ? 'ended' : 'live'}
              </span>
              <span class="session__when">started {formatTimeAgo(session.startedAt)}</span>
              {#if session.userAgent}
                <span class="session__ua">{session.userAgent.String}</span>
              {/if}
            </li>
          {/each}
        </ul>
      {:else}
        <p class="sheet__empty">Nobody has opened this link yet.</p>
      {/if}
    {:else}
      <p class="sheet__empty">Couldn’t load the detail for this link.</p>
    {/if}
  </div>
</dialog>

<EditShareDialog {apiKey} bind:this={editDialog} on:saved={onShareEdited} />

<style>
  /* Hallmark · genre: atmospheric · macrostructure: 13 Index-First
   * nav: N5 floating pill · footer: Ft4 dense colophon
   * design-system: design.md · designed-as-app
   */
  /* 13 · Index-First. The page IS the list: hairline rules between rows,
     the title is the button, no table chrome, no cards. */
  .admin {
    display: grid;
    grid-template-rows: 1fr auto;
    min-height: 100dvh;
    padding-inline: var(--page-gutter);
  }

  .admin > * { position: relative; z-index: var(--z-base); }

  /* ---- N5 · Floating pill ---- */
  .pill {
    position: fixed;
    inset: var(--space-md) auto auto 50%;
    transform: translateX(-50%);
    display: inline-flex;
    align-items: center;
    gap: var(--space-sm);
    max-width: calc(100% - 2 * var(--space-md));
    padding: var(--space-2xs) var(--space-2xs) var(--space-2xs) var(--space-md);
    background: color-mix(in oklch, var(--color-paper-2) 76%, transparent);
    backdrop-filter: blur(14px) saturate(130%);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-pill);
    z-index: var(--z-sticky);
  }

  .pill__wordmark {
    display: inline-flex;
    align-items: center;
    gap: var(--space-sm);
    font-family: var(--font-display);
    font-size: var(--text-md);
    letter-spacing: -0.01em;
    white-space: nowrap;
  }

  .pill__count {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    font-variant-numeric: tabular-nums;
    color: var(--color-muted);
    white-space: nowrap;
  }

  .pill__tabs {
    display: inline-flex;
    gap: 0.125rem;
    padding: 0.15rem;
    background: var(--color-paper-3);
    border-radius: var(--radius-pill);
  }

  .pill__tab {
    height: 1.95rem;
    padding-inline: var(--space-sm);
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

  .pill__tab--active {
    background: var(--color-paper-2);
    color: var(--color-ink);
  }

  @media (hover: hover) {
    .pill__tab:hover:not(.pill__tab--active) { color: var(--color-ink); }
  }

  .pill__tab:active { transform: translateY(1px); }

  .pill__icon {
    display: grid;
    place-items: center;
    width: 2.25rem;
    height: 2.25rem;
    background: none;
    color: var(--color-ink-2);
    border: none;
    border-radius: var(--radius-pill);
    cursor: pointer;
    transition: color var(--dur-micro) var(--ease-out),
                background-color var(--dur-micro) var(--ease-out);
  }

  /* 44px hit target without growing the glyph. */
  .pill__icon::before { content: ""; position: absolute; inset: -0.35rem; }
  .pill__icon { position: relative; }

  .pill__icon svg { width: 1.05rem; height: 1.05rem; }

  @media (hover: hover) {
    .pill__icon:hover:not(:disabled) { color: var(--color-accent); background: var(--color-paper-3); }
  }

  .pill__icon:active:not(:disabled) { transform: translateY(1px); }

  .pill__icon:disabled { opacity: 0.45; cursor: not-allowed; }

  .pill__out {
    height: 2.25rem;
    padding-inline: var(--space-md);
    background: var(--color-paper-3);
    color: var(--color-ink);
    font-size: var(--text-sm);
    font-weight: 600;
    white-space: nowrap;
    border: none;
    border-radius: var(--radius-pill);
    cursor: pointer;
    transition: background-color var(--dur-micro) var(--ease-out);
  }

  @media (hover: hover) {
    .pill__out:hover { background: var(--color-rule); }
  }

  .pill__out:active { transform: translateY(1px); }

  /* ---- The index ---- */
  .index {
    padding-block: calc(var(--space-3xl) + var(--space-md)) var(--space-2xl);
    max-width: 78rem;
    width: 100%;
  }

  .index__lede {
    max-width: 54ch;
    color: var(--color-muted);
    margin-bottom: var(--space-2xl);
  }

  .index__live { color: var(--color-ink-2); }

  .filters {
    display: inline-flex;
    gap: 0.125rem;
    padding: 0.15rem;
    margin-bottom: var(--space-xl);
    background: var(--color-paper-3);
    border-radius: var(--radius-pill);
  }

  .filter {
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

  .filter--active { background: var(--color-paper-2); color: var(--color-ink); }

  .filter__n {
    margin-inline-start: 0.4rem;
    font-family: var(--font-outlier);
    font-size: var(--text-xs);
    font-variant-numeric: tabular-nums;
    color: var(--color-neutral);
  }

  .filter--active .filter__n { color: var(--color-ink-2); }

  @media (hover: hover) {
    .filter:hover:not(.filter--active) { color: var(--color-ink); }
  }

  .filter:active { transform: translateY(1px); }

  .mass {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-sm);
    padding-block: var(--space-sm);
    margin-bottom: var(--space-md);
    border-bottom: var(--rule-hair) solid var(--color-rule-2);
  }

  .mass__all {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    font-size: var(--text-sm);
    color: var(--color-ink-2);
    cursor: pointer;
  }

  .mass__all input {
    width: 1rem;
    height: 1rem;
    accent-color: var(--color-accent);
    cursor: pointer;
  }

  .mass__count {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    font-variant-numeric: tabular-nums;
    color: var(--color-muted);
  }

  .mass__actions {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-xs);
    margin-inline-start: auto;
  }

  .mass__warning {
    flex-basis: 100%;
    font-size: var(--text-sm);
    color: var(--color-muted);
  }

  .labels {
    display: none;
    padding-block: var(--space-xs);
    border-bottom: var(--rule-hair) solid var(--color-rule);
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--color-neutral);
  }

  .rows { list-style: none; margin: 0; padding: 0; }

  .row {
    display: grid;
    grid-template-columns: 1.5rem minmax(0, 1fr);
    gap: var(--space-xs) var(--space-md);
    align-items: baseline;
    padding-block: var(--space-md);
    border-bottom: var(--rule-hair) solid var(--color-rule-2);
    transition: transform var(--dur-micro) var(--ease-out);
  }

  .row__check {
    grid-column: 1;
    grid-row: 1;
    display: flex;
    justify-content: center;
    padding-top: 0.3rem;
  }

  .row__check input {
    width: 1.05rem;
    height: 1.05rem;
    accent-color: var(--color-accent);
    cursor: pointer;
  }

  .row__item,
  .row__status,
  .row__num,
  .row__expiry,
  .row__actions { grid-column: 2; }

  @media (hover: hover) {
    .row:hover { transform: translateY(-1px); }
    .row:hover .row__title { color: var(--color-accent); }
  }

  .row--spent .row__title,
  .row--spent .row__num { color: var(--color-neutral); }

  .row__item {
    display: grid;
    gap: var(--space-3xs);
    justify-items: start;
    text-align: start;
    min-width: 0;
    padding: 0;
    background: none;
    border: none;
    cursor: pointer;
  }

  .row__title {
    font-weight: 600;
    line-height: 1.3;
    color: var(--color-ink);
    overflow-wrap: anywhere;
    transition: color var(--dur-micro) var(--ease-out);
  }

  .row__sub {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    color: var(--color-muted);
  }

  .row__status {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    letter-spacing: 0.06em;
    color: var(--color-muted);
    white-space: nowrap;
  }

  /* The word carries the state; the dot only reinforces it. Never colour alone. */
  .dot { width: 0.4rem; height: 0.4rem; border-radius: var(--radius-pill); background: var(--color-neutral); }
  .row__status[data-status="active"] { color: var(--color-accent); }
  .row__status[data-status="active"] .dot { background: var(--color-accent); }
  .row__status[data-status="revoked"] { color: var(--color-danger); }
  .row__status[data-status="revoked"] .dot { background: var(--color-danger); }

  .row__num {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    font-variant-numeric: tabular-nums;
    color: var(--color-ink-2);
    white-space: nowrap;
  }

  .row__watching { color: var(--color-accent); }

  .row__actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-xs);
  }

  .confirm { display: inline-flex; gap: var(--space-xs); }

  .btn-ghost, .btn-danger {
    min-height: 2.25rem;
    padding-inline: var(--space-sm);
    font-size: var(--text-sm);
    font-weight: 600;
    white-space: nowrap;
    border-radius: var(--radius-pill);
    cursor: pointer;
    transition: background-color var(--dur-micro) var(--ease-out),
                color var(--dur-micro) var(--ease-out);
  }

  .btn-ghost {
    background: none;
    color: var(--color-muted);
    border: var(--rule-hair) solid var(--color-rule);
  }

  @media (hover: hover) {
    .btn-ghost:hover { background: var(--color-paper-2); color: var(--color-ink); }
    .btn-ghost--warn:hover { color: var(--color-danger); border-color: var(--color-danger); }
  }

  .btn-danger {
    background: var(--color-danger);
    color: var(--color-danger-ink);
    border: var(--rule-hair) solid var(--color-danger);
  }

  .btn-danger:focus-visible { outline-offset: 3px; }

  .btn-ghost:active, .btn-danger:active { transform: translateY(1px); }

  .btn-ghost:disabled, .btn-danger:disabled { opacity: 0.55; cursor: not-allowed; transform: none; }

  .row__item:active .row__title { color: var(--color-ink-2); }

  .row__item:disabled { opacity: 0.55; cursor: not-allowed; }

  /* ---- States ---- */
  /* The index continues below the last row, in the row's own voice. */
  .more {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-sm);
    padding-block: var(--space-lg);
  }

  .more__left {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    font-variant-numeric: tabular-nums;
    color: var(--color-muted);
  }

  .state {
    display: grid;
    gap: var(--space-sm);
    justify-items: start;
    padding-block: var(--space-2xl);
    color: var(--color-muted);
  }

  .state__head { font-size: var(--text-md); color: var(--color-ink); }

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

  .banner {
    margin-bottom: var(--space-md);
    padding: var(--space-sm) var(--space-md);
    background: var(--color-paper-2);
    border-radius: var(--radius-card);
    color: var(--color-danger);
    font-size: var(--text-sm);
  }

  .banner--ok { color: var(--color-ink-2); }

  /* ---- Ft4 · Dense colophon ---- */
  .colophon {
    display: grid;
    gap: var(--space-2xs);
    padding-block: var(--space-lg);
    padding-bottom: max(var(--space-lg), env(safe-area-inset-bottom));
    border-top: var(--rule-hair) solid var(--color-rule-2);
    font-size: var(--text-sm);
    line-height: 1.55;
    color: var(--color-neutral);
  }

  /* Identity and counts are machine facts, so they keep the mono register. */
  .colophon__line {
    font-family: var(--font-outlier);
    font-variant-numeric: tabular-nums;
  }

  /* The note is a sentence, so it is set as one — a mono wall was what made the
     old footer read as noise rather than as a closing remark. */
  .colophon__note { max-width: 68ch; }

  /* ---- Detail sheet ---- */
  .sheet {
    position: fixed;
    inset: 0;
    margin: auto;
    width: min(46rem, calc(100% - 2 * var(--space-md)));
    height: fit-content;
    max-height: min(84dvh, 46rem);
    padding: 0;
    background: var(--color-paper-2);
    color: var(--color-ink);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-card);
    overflow: auto;
  }

  .sheet::backdrop { background: var(--scrim-strong); }

  .sheet__panel { padding: var(--space-lg); }

  .sheet__head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-md);
    margin-bottom: var(--space-lg);
  }

  .sheet__head h2 {
    font-size: var(--text-lg);
    overflow-wrap: anywhere;
    min-width: 0;
  }

  .sheet__tools {
    display: flex;
    flex: none;
    align-items: center;
    gap: var(--space-xs);
  }

  .spec {
    display: grid;
    gap: var(--space-md);
    margin: 0 0 var(--space-xl);
  }

  .spec > div {
    display: grid;
    grid-template-columns: 9.5rem minmax(0, 1fr);
    gap: var(--space-md);
    align-items: baseline;
    padding-bottom: var(--space-sm);
    border-bottom: var(--rule-hair) solid var(--color-rule-2);
  }

  dt {
    font-size: var(--text-sm);
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--color-neutral);
  }

  dd { margin: 0; overflow-wrap: anywhere; }

  .spec__mono {
    font-family: var(--font-outlier);
    font-variant-numeric: tabular-nums;
    color: var(--color-ink-2);
  }

  .sheet__sub {
    display: flex;
    align-items: baseline;
    gap: var(--space-xs);
    font-size: var(--text-md);
    margin-bottom: var(--space-md);
  }

  .sheet__n {
    font-family: var(--font-outlier);
    font-size: var(--text-sm);
    color: var(--color-muted);
  }

  .sessions { list-style: none; margin: 0; padding: 0; }

  .session {
    display: grid;
    gap: var(--space-3xs);
    padding-block: var(--space-sm);
    border-bottom: var(--rule-hair) solid var(--color-rule-2);
  }

  .session__state {
    font-size: var(--text-sm);
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--color-neutral);
  }

  .session__state--live { color: var(--color-accent); }

  .session__when { font-size: var(--text-sm); color: var(--color-ink-2); }

  .session__ua {
    font-family: var(--font-outlier);
    font-size: var(--text-xs);
    color: var(--color-neutral);
    overflow-wrap: anywhere;
  }

  .sheet__empty { color: var(--color-muted); }

  /* ---- Wider than a phone: the index gets its columns ---- */
  @media (min-width: 60rem) {
    .labels,
    .row {
      display: grid;
      grid-template-columns: 1.75rem minmax(0, 1fr) 7rem 8rem 7rem 13.5rem;
      gap: var(--space-md);
      align-items: baseline;
    }

    .row__check,
    .row__item,
    .row__status,
    .row__num,
    .row__expiry,
    .row__actions { grid-column: auto; }

    .row { padding-block: var(--space-lg); }

    .row__actions { justify-content: flex-end; }

    .row__watching { display: block; }
  }

  @media (max-width: 60rem) {
    .row__watching { margin-inline-start: var(--space-xs); }

    .row__status,
    .row__num { font-size: var(--text-sm); }

    .row__actions { margin-top: var(--space-2xs); }

    .spec > div { grid-template-columns: minmax(0, 1fr); gap: var(--space-2xs); }
  }

  @media (max-width: 48rem) {
    .pill { gap: var(--space-xs); padding-left: var(--space-sm); }
    .pill__count, .pill__name { display: none; }
    .pill__wordmark { font-size: var(--text-base); }
  }

  @media (max-width: 30rem) {
    .pill__tab { padding-inline: var(--space-xs); }
  }
</style>
