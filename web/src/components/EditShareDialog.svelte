<script>
  import { createEventDispatcher } from 'svelte';

  export let apiKey;

  const dispatch = createEventDispatcher();

  let dialogEl;
  let share = null;
  let extendDays = '';
  let maxPlays = '';
  let maxConcurrent = '';
  let password = '';
  let removePassword = false;
  let allowDownload = true;
  let originalAllowDownload = true;
  let maxHeight = '';
  let originalMaxHeight = '';
  let maxBitrate = '';
  let originalMaxBitrate = '';
  let error = '';
  let saving = false;

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

  export function open(item) {
    share = item;
    extendDays = '';
    maxPlays = '';
    maxConcurrent = '';
    password = '';
    removePassword = false;
    originalAllowDownload = item.allowDownloadStored ?? item.allowDownload;
    allowDownload = originalAllowDownload;
    originalMaxHeight = item.maxVideoHeight != null ? String(item.maxVideoHeight) : '';
    maxHeight = originalMaxHeight;
    originalMaxBitrate = item.maxVideoBitrate != null ? String(item.maxVideoBitrate) : '';
    maxBitrate = originalMaxBitrate;
    error = '';
    saving = false;
    dialogEl?.showModal();
  }

  function close() {
    dialogEl?.close();
  }

  function onDialogClose() {
    share = null;
  }

  function onDialogClick(event) {
    if (event.target === dialogEl) close();
  }

  function currentLimit(value) {
    return value == null ? 'Unlimited' : String(value);
  }

  async function save() {
    error = '';

    const body = {};

    if (extendDays !== '') {
      const days = Number(extendDays);
      if (!Number.isInteger(days) || days <= 0) {
        error = 'Days to add must be a whole number of 1 or more.';
        return;
      }
      body.extendMinutes = days * 1440;
    }

    if (maxPlays !== '') {
      const plays = Number(maxPlays);
      if (!Number.isInteger(plays) || plays <= 0) {
        error = 'Play limit must be a whole number of 1 or more.';
        return;
      }
      body.maxTotalPlays = plays;
    }

    if (maxConcurrent !== '') {
      const concurrent = Number(maxConcurrent);
      if (!Number.isInteger(concurrent) || concurrent <= 0) {
        error = 'Watching at once must be a whole number of 1 or more.';
        return;
      }
      body.maxConcurrentViewers = concurrent;
    }

    if (removePassword) {
      body.removePassword = true;
    } else if (password !== '') {
      body.password = password;
    }

    if (allowDownload !== originalAllowDownload) {
      body.allowDownload = allowDownload;
    }
    if (maxHeight !== originalMaxHeight) {
      body.maxVideoHeight = maxHeight === '' ? 0 : Number(maxHeight);
    }
    if (maxBitrate !== originalMaxBitrate) {
      body.maxVideoBitrate = maxBitrate === '' ? 0 : Number(maxBitrate);
    }

    if (Object.keys(body).length === 0) {
      error = 'Change something first.';
      return;
    }

    saving = true;
    try {
      const response = await fetch(`/api/admin/shares/${share.id}`, {
        method: 'PATCH',
        headers: {
          'X-Backend-Key': apiKey,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(body)
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || 'Could not update the link.');
      dispatch('saved');
      close();
    } catch (e) {
      error = e.message || 'Could not update the link.';
    } finally {
      saving = false;
    }
  }
</script>

<!-- svelte-ignore a11y-click-events-have-key-events a11y-no-noninteractive-element-interactions -->
<dialog bind:this={dialogEl} class="edit" on:close={onDialogClose} on:click={onDialogClick}>
  <form class="panel" on:submit|preventDefault={save}>
    <header class="head">
      <div>
        <h2>Edit link</h2>
        <p class="head__title">{share?.title || ''}</p>
      </div>
      <button class="icon" type="button" on:click={close} aria-label="Close the editor">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" aria-hidden="true">
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
      </button>
    </header>

    <div class="field">
      <label for="edit-extend">Add days to expiry</label>
      <input
        id="edit-extend"
        class="input"
        type="number"
        min="1"
        step="1"
        bind:value={extendDays}
        placeholder={share?.expiresAt ? 'Leave blank to keep it' : 'This link never expires'}
        disabled={!share?.expiresAt}
      />
      <p class="help">
        {#if share?.expiresAt}
          Extends from the current expiry.
        {:else}
          A link that never expires has no deadline to extend.
        {/if}
      </p>
    </div>

    <div class="pair">
      <div class="field">
        <label for="edit-plays">Play limit</label>
        <input
          id="edit-plays"
          class="input"
          type="number"
          min="1"
          step="1"
          bind:value={maxPlays}
          placeholder={currentLimit(share?.maxTotalPlays)}
        />
        <p class="help">Leave blank to keep the current limit.</p>
      </div>

      <div class="field">
        <label for="edit-concurrent">Watching at once</label>
        <input
          id="edit-concurrent"
          class="input"
          type="number"
          min="1"
          step="1"
          bind:value={maxConcurrent}
          placeholder={currentLimit(share?.maxConcurrentViewers)}
        />
        <p class="help">Leave blank to keep the current limit.</p>
      </div>
    </div>

    <div class="pair">
      <div class="field">
        <label for="edit-height">Maximum picture</label>
        <select id="edit-height" class="input" bind:value={maxHeight}>
          {#each HEIGHTS as option (option.value)}
            <option value={option.value}>{option.label}</option>
          {/each}
        </select>
        <p class="help">Auto removes the cap.</p>
      </div>

      <div class="field">
        <label for="edit-bitrate">Maximum bitrate</label>
        <select id="edit-bitrate" class="input" bind:value={maxBitrate}>
          {#each BITRATES as option (option.value)}
            <option value={option.value}>{option.label}</option>
          {/each}
        </select>
        <p class="help">Auto removes the cap.</p>
      </div>
    </div>

    <label class="check">
      <input type="checkbox" bind:checked={allowDownload} />
      <span>Allow downloads for this link</span>
    </label>
    <p class="help">The server-wide setting can still switch downloads off.</p>

    <div class="field">
      <label for="edit-password">New password</label>
      <input
        id="edit-password"
        class="input"
        type="password"
        bind:value={password}
        autocomplete="new-password"
        disabled={removePassword}
        placeholder={share?.hasPassword ? 'Leave blank to keep the password' : 'Optional'}
      />
    </div>

    {#if share?.hasPassword}
      <label class="check">
        <input type="checkbox" bind:checked={removePassword} />
        <span>Remove the password</span>
      </label>
    {/if}

    {#if error}
      <p class="error" role="alert">{error}</p>
    {/if}

    <div class="actions">
      <button class="btn-primary" type="submit" disabled={saving}>
        {saving ? 'Saving' : 'Save changes'}
      </button>
      <button class="btn-ghost" type="button" on:click={close}>Cancel</button>
    </div>
  </form>
</dialog>

<style>
  .edit {
    position: fixed;
    inset: 0;
    margin: auto;
    width: min(34rem, calc(100% - 2 * var(--space-md)));
    height: fit-content;
    max-height: min(88dvh, 40rem);
    padding: 0;
    background: var(--color-paper-2);
    color: var(--color-ink);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-card);
    overflow: auto;
  }

  .edit::backdrop { background: var(--scrim-strong); }

  .panel {
    display: grid;
    gap: var(--space-lg);
    padding: var(--space-lg);
  }

  .head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-md);
  }

  .head h2 { font-size: var(--text-lg); }

  .head__title {
    margin-top: var(--space-3xs);
    font-size: var(--text-sm);
    color: var(--color-muted);
    overflow-wrap: anywhere;
  }

  .icon {
    position: relative;
    display: grid;
    place-items: center;
    width: 2.25rem;
    height: 2.25rem;
    background: none;
    color: var(--color-ink-2);
    border: none;
    border-radius: var(--radius-pill);
    cursor: pointer;
  }

  .icon::before { content: ""; position: absolute; inset: -0.35rem; }
  .icon svg { width: 1.05rem; height: 1.05rem; }

  @media (hover: hover) {
    .icon:hover { background: var(--color-paper-3); color: var(--color-accent); }
  }

  .field { display: grid; gap: var(--space-2xs); }

  .pair {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: var(--space-md);
  }

  label {
    font-size: var(--text-sm);
    font-weight: 600;
    letter-spacing: 0.02em;
    color: var(--color-ink-2);
  }

  .input {
    width: 100%;
    height: var(--control-h);
    padding-inline: var(--space-sm);
    background: var(--color-paper-3);
    color: var(--color-ink);
    font-family: var(--font-outlier);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-input);
    outline: 2px solid transparent;
    outline-offset: 1px;
  }

  .input::placeholder { color: var(--color-neutral); }

  .input:focus-visible {
    outline-color: var(--color-focus);
    border-color: var(--color-ink-2);
  }

  .input:disabled { opacity: 0.55; cursor: not-allowed; }

  .help { font-size: var(--text-sm); color: var(--color-muted); }

  .check {
    display: inline-flex;
    align-items: center;
    gap: var(--space-xs);
    font-size: var(--text-sm);
    color: var(--color-ink-2);
    cursor: pointer;
  }

  .check input { width: 1rem; height: 1rem; accent-color: var(--color-accent); }

  .error {
    padding: var(--space-sm) var(--space-md);
    background: var(--color-paper-3);
    border-radius: var(--radius-card);
    color: var(--color-danger);
    font-size: var(--text-sm);
  }

  .actions { display: flex; flex-wrap: wrap; gap: var(--space-xs); }

  .btn-primary, .btn-ghost {
    min-height: var(--control-h);
    padding-inline: var(--space-lg);
    font-size: var(--text-sm);
    font-weight: 600;
    white-space: nowrap;
    border-radius: var(--radius-pill);
    cursor: pointer;
    transition: background-color var(--dur-micro) var(--ease-out),
                transform var(--dur-micro) var(--ease-out);
  }

  .btn-primary {
    background: var(--color-accent);
    color: var(--color-accent-ink);
    border: none;
  }

  .btn-primary:disabled { opacity: 0.55; cursor: not-allowed; }

  @media (hover: hover) {
    .btn-primary:hover:not(:disabled) { transform: translateY(-1px); }
    .btn-ghost:hover { background: var(--color-paper-3); color: var(--color-ink); }
  }

  .btn-primary:active:not(:disabled) { transform: translateY(1px); }

  .btn-ghost {
    background: none;
    color: var(--color-muted);
    border: var(--rule-hair) solid var(--color-rule);
  }
</style>
