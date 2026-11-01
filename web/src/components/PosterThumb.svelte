<script>
  import { onDestroy } from 'svelte';

  export let apiKey;
  export let itemId;
  export let alt = '';
  export let maxWidth = 240;

  let url = null;
  let currentId = null;

  $: if (itemId !== currentId) {
    currentId = itemId;
    loadPoster(itemId);
  }

  async function loadPoster(id) {
    if (url) {
      URL.revokeObjectURL(url);
      url = null;
    }
    if (!id || !apiKey) return;

    try {
      const response = await fetch(
        `/api/admin/jellyfin/items/${encodeURIComponent(id)}/poster?maxWidth=${maxWidth}`,
        { headers: { 'X-Backend-Key': apiKey } }
      );
      if (!response.ok) return;
      const blob = await response.blob();
      if (currentId !== id) return;
      url = URL.createObjectURL(blob);
    } catch (e) {
      // A missing poster is not an error worth showing; the placeholder stands in.
    }
  }

  onDestroy(() => {
    if (url) URL.revokeObjectURL(url);
  });
</script>

{#if url}
  <img src={url} {alt} loading="lazy" />
{:else}
  <span class="placeholder" aria-hidden="true"></span>
{/if}

<style>
  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .placeholder {
    display: block;
    width: 100%;
    height: 100%;
    background: var(--color-paper-3);
  }
</style>
