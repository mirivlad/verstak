<script>
  import Icon from './Icon.svelte';
  import { theme, setTheme } from './theme.js';
  import { i18n } from '../i18n/index.js';
  import { onDestroy } from 'svelte';

  let locale = i18n.getLocale();
  const unsubscribe = i18n.subscribe(value => { locale = value; });
  onDestroy(unsubscribe);
  $: label = (locale, i18n.t($theme.resolved === 'dark' ? 'theme.switchLight' : 'theme.switchDark'));
</script>

<div class="theme-control">
  <button type="button" class="vt-button ghost theme-toggle" data-theme-toggle aria-label={label} title={label}
    disabled={$theme.saving} on:click={() => setTheme($theme.resolved === 'dark' ? 'light' : 'dark')}>
    {#if $theme.resolved === 'dark'}<Icon name="sun" size={16} />{:else}<Icon name="moon" size={16} />{/if}
    <span>{label}</span>
  </button>
  {#if $theme.error}<span class="theme-error" role="alert">{i18n.t('theme.saveError')}</span>{/if}
</div>

<style>
  .theme-control { position: relative; flex-shrink: 0; }
  .theme-toggle { white-space: nowrap; font-weight: 400; }
  .theme-error { position: absolute; right: 0; top: 100%; z-index: 50; width: min(19rem, 80vw); padding: 0.7rem; background: var(--vt-color-surface); border: 1px solid var(--vt-color-danger); border-radius: var(--vt-radius-md); color: var(--vt-color-danger-foreground); font-size: var(--vt-font-sm); }
  @media (max-width: 1100px) { .theme-toggle span { display: none; } .theme-toggle { width: 2.25rem; padding: 0; } }
</style>
