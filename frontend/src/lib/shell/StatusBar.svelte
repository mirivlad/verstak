<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import CompactPluginHost from '../plugin-host/CompactPluginHost.svelte';
  import { i18n } from '../i18n/index.js';

  let items = [];
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let build = { version: '', commit: '', buildDate: '', display: '' };
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;

  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);

  $: leftItems = items.filter((item) => item.position === 'left');
  $: centerItems = items.filter((item) => item.position === 'center');
  $: rightItems = items.filter((item) => item.position === 'right');
  $: vaultOpen = vaultStatus.status === 'open';
  $: vaultStatusLabel = tr(`vault.status.${vaultStatus.status || 'unknown'}`, undefined, vaultStatus.status || 'unknown');
  $: vaultLabel = tr('vault.label', { status: vaultStatusLabel });

  const inactiveStatuses = new Set(['disabled', 'failed', 'incompatible', 'missing-required-capability']);

  $: buildTooltip = [
    tr('build.label', { version: build.version || '' }),
    build.commit ? tr('build.commit', { commit: build.commit }) : '',
    build.buildDate ? tr('build.date', { date: build.buildDate }) : '',
  ].filter(Boolean).join('\n');

  async function loadBuildInfo() {
    try {
      build = await App.GetBuildInfo() || build;
    } catch {
      // A build that cannot report itself is not worth an error in the UI.
    }
  }

  async function loadStatusBar() {
    const [rawPlugins, rawContributions, vault] = await Promise.all([
      App.GetPlugins().catch(() => []),
      App.GetContributions().catch(() => ({})),
      App.GetVaultStatus().catch(() => ({ status: 'unknown', path: '', vaultId: '' })),
    ]);
    await Promise.all((rawPlugins || []).map((plugin) => (
      i18n.loadPlugin(plugin.manifest?.id, plugin.manifest?.localization).catch(() => {})
    )));
    const plugins = (rawPlugins || []).map((plugin) => i18n.localizePlugin(plugin));
    const contributions = i18n.localizeContributionSummary(rawContributions || {});
    const pluginById = new Map((plugins || []).map((plugin) => [plugin.manifest?.id, plugin]));
    vaultStatus = vault || { status: 'unknown', path: '', vaultId: '' };
    items = (contributions.statusBarItems || [])
      .filter((item) => {
        const plugin = pluginById.get(item.pluginId);
        if (!plugin) return false;
        return !inactiveStatuses.has(plugin.status);
      })
      .map((item) => ({
        ...item,
        position: item.position || 'left',
      }));
  }

  // The gear opens the settings window. It used to open a dropdown holding
  // the language choice, a link to the Plugin Manager and one entry per plugin
  // settings panel -- a menu that grew with every plugin installed and had
  // nowhere to put anything that was not a single choice.
  function openSettingsWindow() {
    window.dispatchEvent(new CustomEvent('verstak:open-settings', { detail: {} }));
  }

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed) loadStatusBar();
    });
    loadStatusBar();
    loadBuildInfo();
    window.addEventListener('verstak:plugins-changed', loadStatusBar);
    window.addEventListener('verstak:vault-opened', loadStatusBar);
  });

  onDestroy(() => {
    if (unsubscribeLocale) unsubscribeLocale();
    window.removeEventListener('verstak:plugins-changed', loadStatusBar);
    window.removeEventListener('verstak:vault-opened', loadStatusBar);
  });
</script>

<footer class="status-bar" aria-label={tr('statusBar.label')}>
  <div class="status-bar-group status-left">
    <span
      class="vault-status"
      class:vault-open={vaultOpen}
      class:vault-closed={!vaultOpen}
      title={vaultStatus.path || vaultStatus.vaultId || vaultLabel}
    >
      <Icon name="vault" size={13} class="status-icon" />
      {vaultLabel}
    </span>
    {#each leftItems as item}
      <span
        class:status-bar-warning={item.handler}
        class="status-bar-item"
        data-status-item-id={item.id}
        title={item.label || item.id}
      >
        {#if item.handler}<CompactPluginHost pluginId={item.pluginId} handler={item.handler} label={item.label || item.id} />{:else}{item.label || item.id}{/if}
      </span>
    {/each}
  </div>
  <div class="status-bar-group status-center">
    {#each centerItems as item}
      <span
        class:status-bar-warning={item.handler}
        class="status-bar-item"
        data-status-item-id={item.id}
        title={item.label || item.id}
      >
        {#if item.handler}<CompactPluginHost pluginId={item.pluginId} handler={item.handler} label={item.label || item.id} />{:else}{item.label || item.id}{/if}
      </span>
    {/each}
  </div>
  <div class="status-bar-group status-right">
    {#each rightItems as item}
      <span
        class:status-bar-warning={item.handler}
        class="status-bar-item"
        data-status-item-id={item.id}
        title={item.label || item.id}
      >
        {#if item.handler}<CompactPluginHost pluginId={item.pluginId} handler={item.handler} label={item.label || item.id} />{:else}{item.label || item.id}{/if}
      </span>
    {/each}
    <!-- Which build is running. Without it a fix that did not land and a
         package that was never installed look exactly the same. -->
    {#if build.display}
      <span
        class="status-bar-item build-version"
        data-build-version={build.version}
        title={buildTooltip}
      >{build.display}</span>
    {/if}

    <button
      class="settings-button"
      type="button"
      title={tr('settings.title')}
      aria-label={tr('settings.title')}
      data-settings-menu-button
      on:click={openSettingsWindow}
    >
      <Icon name="settings" size={14} class="settings-icon" />
    </button>
  </div>
</footer>

<style>
  .status-bar {
    position: relative;
    z-index: 100;
    min-height: 1.7rem;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
    align-items: center;
    gap: 0.5rem;
    padding: 0.2rem 0.65rem;
    border-top: 1px solid #16213e;
    background: #111629;
    color: #9fb2ca;
    font-size: 0.74rem;
  }

  .status-bar-group {
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 0.4rem;
    overflow: hidden;
  }

  .status-center {
    justify-content: center;
  }

  .status-right {
    justify-content: flex-end;
  }

  .status-bar-item {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    max-width: 18rem;
    overflow: hidden;
    padding: 0.12rem 0.35rem;
    border-radius: 4px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .status-bar-warning {
    color: #ffc857;
  }

  :global(.status-warning-icon) {
    flex-shrink: 0;
    color: currentColor;
  }

  .vault-status {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    max-width: 24rem;
    min-width: 0;
    overflow: hidden;
    padding: 0.12rem 0.35rem;
    border-radius: 4px;
    color: #a0a0b8;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .vault-status.vault-open {
    color: #4ecca3;
  }

  .vault-status.vault-closed {
    color: #9fb2ca;
  }

  :global(.status-icon) {
    flex-shrink: 0;
    color: currentColor;
  }

  .build-version {
    color: var(--vt-color-text-muted);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }


  .settings-button {
    min-height: 1.35rem;
    height: 1.35rem;
    padding: 0 0.35rem;
    gap: 0.15rem;
    border: 1px solid transparent;
    border-radius: 4px;
    background: transparent;
    color: #9fb2ca;
  }

  .settings-button:hover,

  :global(.settings-icon),
  :global(.settings-chevron),
  :global(.settings-menu-icon) {
    flex-shrink: 0;
    color: currentColor;
  }







</style>
