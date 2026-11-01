<script>
  export let apiKey;

  // Filled in Step 4. The shell calls this when Refresh is pressed on this tab.
  export function reload() {}
</script>

<main class="create">
  <p class="create__lede">
    Make a link to a film, episode, season or series, then hand it to someone
    who has no Jellyfin account.
  </p>

  <div class="state">
    <p class="state__head">Share creation lands in the next step.</p>
    <p>The library picker and link terms are being built here.</p>
  </div>
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

  .state {
    display: grid;
    gap: var(--space-sm);
    justify-items: start;
    padding-block: var(--space-2xl);
    color: var(--color-muted);
  }

  .state__head { font-size: var(--text-md); color: var(--color-ink); }
</style>
