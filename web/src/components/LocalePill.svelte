<script>
  import { locale, setLocale, t } from '../i18n/index.js';
</script>

<div class="locale" role="group" aria-label={$t('locale.label')}>
  <button
    type="button"
    class:locale__tab--active={$locale === 'en'}
    aria-pressed={$locale === 'en'}
    on:click={() => setLocale('en')}
  >EN</button>
  <button
    type="button"
    class:locale__tab--active={$locale === 'th'}
    aria-pressed={$locale === 'th'}
    on:click={() => setLocale('th')}
  >TH</button>
</div>

<style>
  .locale {
    display: inline-flex;
    flex: none;
    gap: 0.125rem;
    padding: 0.15rem;
    background: color-mix(in oklch, var(--color-paper-2) 70%, transparent);
    backdrop-filter: blur(10px);
    border: var(--rule-hair) solid var(--color-rule);
    border-radius: var(--radius-pill);
  }

  .locale button {
    min-width: 2.1rem;
    min-height: 1.9rem;
    padding-inline: var(--space-xs);
    background: none;
    color: var(--color-muted);
    font-family: var(--font-outlier);
    font-size: var(--text-xs);
    font-weight: 600;
    letter-spacing: 0.08em;
    border: none;
    border-radius: var(--radius-pill);
    cursor: pointer;
    transition: background-color var(--dur-micro) var(--ease-out),
                color var(--dur-micro) var(--ease-out);
  }

  .locale button.locale__tab--active {
    background: var(--color-paper-2);
    color: var(--color-ink);
  }

  @media (hover: hover) {
    .locale button:hover:not(.locale__tab--active) { color: var(--color-ink); }
  }

  .locale button:active { transform: translateY(1px); }
</style>
