<script>
  import BrandMark from './BrandMark.svelte';
  import LocalePill from './LocalePill.svelte';
  import { dateTimeFormatter, localeTag, t } from '../i18n/index.js';

  export let error;

  // Errors name what broke, then what to do about it. No apology, no emoji, and
  // no status codes or server strings: those are for logs, not for whoever
  // opened the link.
  function messages(translate) {
    return {
      404: {
        title: translate('error.404.title'),
        description: translate('error.404.body'),
        action: translate('error.404.action')
      },
      410: {
        title: translate('error.410.title'),
        description: translate('error.410.body'),
        action: translate('error.410.action')
      },
      403: {
        title: translate('error.403.title'),
        description: translate('error.403.body'),
        action: translate('error.403.action')
      },
      0: {
        title: translate('error.offline.title'),
        description: translate('error.offline.body'),
        action: translate('error.offline.action')
      },
      default: {
        title: translate('error.default.title'),
        description: translate('error.default.body'),
        action: translate('error.default.action')
      }
    };
  }

  // Same day-month order and 24-hour clock as the share page, in the page's
  // language. Thai uses Thai month names with the Gregorian year.
  $: when = dateTimeFormatter($localeTag);

  // A link that ran out says which one, why and since when - the server knows,
  // so the page does not make the viewer guess between causes.
  function goneMessage(gone, translate, formatter) {
    const at = new Date(gone.endedAt);
    const date = Number.isNaN(at.getTime())
      ? ''
      : translate('error.gone.onDate', { date: formatter.format(at) });
    const name = [gone.seriesName, gone.title].filter(Boolean).join(', ');
    const link = name ? translate('error.gone.linkTo', { name }) : translate('error.gone.thisLink');
    if (gone.reason === 'revoked') {
      return {
        title: translate('error.gone.revoked.title'),
        description: translate('error.gone.revoked', { link, date }),
        action: translate('error.410.action')
      };
    }
    if (gone.reason === 'expired') {
      return {
        title: translate('error.gone.expired.title'),
        description: translate('error.gone.expired', { link, date }),
        action: translate('error.410.action')
      };
    }
    return messages(translate)[410];
  }

  $: errorInfo = error?.gone
    ? goneMessage(error.gone, $t, when)
    : messages($t)[error?.status] || messages($t).default;
</script>

<svelte:head><title>{errorInfo.title} · Jellyshare</title></svelte:head>

<div class="error bloom-ground">
  <header class="nav">
    <span class="wordmark"><BrandMark />Jellyshare</span>
    <LocalePill />
  </header>

  <main class="error__body">
    <h1 class="error__title">{errorInfo.title}</h1>
    <p class="error__lede">{errorInfo.description}</p>
    <p class="error__action">{errorInfo.action}</p>
  </main>

  <footer class="foot">
    <p>{$t('footer.sharedVia')} <a class="foot-link" href="https://github.com/melth-cmd/jellyshare" target="_blank" rel="noopener noreferrer">Jellyshare</a></p>
  </footer>
</div>

<style>
  /* Hallmark · genre: atmospheric · macrostructure: 08 Photographic (bare fold)
   * nav: N9 edge-aligned · footer: Ft2 inline single line
   * design-system: design.md · designed-as-app
   */
  /* No card, no centred column, no icon. The statement sits low-left against a
     single accent bloom; the emptiness above it is the design. */
  .error {
    display: grid;
    grid-template-rows: auto 1fr auto;
    min-height: 100dvh;
    padding-inline: var(--page-gutter);
  }

  .error > * { position: relative; z-index: var(--z-base); }

  /* N9 · Edge-aligned minimal — wordmark hard-left, nothing else: the statement below is the status. */
  .nav {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-md);
    padding-block: var(--space-lg);
  }

  .wordmark {
    display: inline-flex;
    align-items: center;
    gap: var(--space-sm);
    font-family: var(--font-display);
    font-size: var(--text-md);
    letter-spacing: -0.01em;
    color: var(--color-ink-2);
    white-space: nowrap;
  }

  .error__body {
    align-self: end;
    max-width: var(--measure);
    padding-block: var(--space-2xl) var(--space-3xl);
  }

  .error__title {
    font-size: var(--text-display);
    overflow-wrap: anywhere;
    min-width: 0;
    margin-bottom: var(--space-lg);
  }

  .error__lede {
    font-size: var(--text-md);
    line-height: 1.55;
    color: var(--color-ink-2);
    margin-bottom: var(--space-sm);
  }

  .error__action {
    color: var(--color-muted);
  }

  /* Ft2 · Inline single line */
  .foot {
    padding-block: var(--space-md) var(--space-lg);
    padding-bottom: max(var(--space-lg), env(safe-area-inset-bottom));
    border-top: var(--rule-hair) solid var(--color-rule-2);
    font-size: var(--text-sm);
    color: var(--color-neutral);
  }

  @media (min-width: 60rem) {
    .error__body { padding-block: var(--space-3xl) var(--space-4xl); }
  }
</style>
