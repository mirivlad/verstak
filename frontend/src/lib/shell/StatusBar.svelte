<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import CompactPluginHost from '../plugin-host/CompactPluginHost.svelte';
  import { i18n } from '../i18n/index.js';

  let items = [];
  let settingsPanels = [];
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let settingsOpen = false;
  let locale = i18n.getLocale();
  let selectedLanguage = i18n.getLanguagePreference();
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
    settingsPanels = (contributions.settingsPanels || [])
      .filter((panel) => {
        const plugin = pluginById.get(panel.pluginId);
        if (!plugin) return false;
        return !inactiveStatuses.has(plugin.status);
      })
      .sort((a, b) => String(a.title || a.id).localeCompare(String(b.title || b.id)));
  }

  function openPluginManager() {
    settingsOpen = false;
    window.dispatchEvent(new CustomEvent('verstak:close-settings'));
    window.dispatchEvent(new CustomEvent('verstak:nav', { detail: { viewId: 'plugin-manager' } }));
  }

  function openSettingsPanel(panel) {
    settingsOpen = false;
    window.dispatchEvent(new CustomEvent('verstak:open-settings', {
      detail: { pluginId: panel.pluginId, panelId: panel.id }
    }));
  }

  async function selectLanguage(language) {
    const err = await App.UpdateAppSettings({ language });
    if (err) {
      throw new Error(err);
    }
    await i18n.setLanguagePreference(language);
    selectedLanguage = language;
    settingsOpen = false;
  }

  function toggleSettings(event) {
    event.stopPropagation();
    settingsOpen = !settingsOpen;
  }

  function closeSettings() {
    settingsOpen = false;
  }

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      selectedLanguage = i18n.getLanguagePreference();
      if (changed) loadStatusBar();
    });
    loadStatusBar();
    window.addEventListener('verstak:plugins-changed', loadStatusBar);
    window.addEventListener('verstak:vault-opened', loadStatusBar);
    window.addEventListener('click', closeSettings);
  });

  onDestroy(() => {
    if (unsubscribeLocale) unsubscribeLocale();
    window.removeEventListener('verstak:plugins-changed', loadStatusBar);
    window.removeEventListener('verstak:vault-opened', loadStatusBar);
    window.removeEventListener('click', closeSettings);
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
        title={item.pluginId}
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
        title={item.pluginId}
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
        title={item.pluginId}
      >
        {#if item.handler}<CompactPluginHost pluginId={item.pluginId} handler={item.handler} label={item.label || item.id} />{:else}{item.label || item.id}{/if}
      </span>
    {/each}
    <div class="settings-menu-wrap">
      <button
        class="settings-button"
        class:active={settingsOpen}
        type="button"
        title={tr('settings.title')}
        aria-haspopup="menu"
        aria-expanded={settingsOpen}
        data-settings-menu-button
        on:click={toggleSettings}
      >
        <Icon name="settings" size={14} class="settings-icon" />
        <Icon name="chevronDown" size={12} class="settings-chevron" />
      </button>
      {#if settingsOpen}
        <div class="settings-menu" role="menu">
          <div class="settings-menu-heading">{tr('settings.language')}</div>
          {#each ['system', 'en', 'ru'] as language}
            <button
              class="settings-menu-item language-item"
              class:active-language={selectedLanguage === language}
              type="button"
              role="menuitemradio"
              aria-checked={selectedLanguage === language}
              data-settings-language={language}
              on:click={() => selectLanguage(language)}
            >
              <span class="language-check" aria-hidden="true">{selectedLanguage === language ? '✓' : ''}</span>
              <span>{tr(`settings.language.${language}`)}</span>
            </button>
          {/each}
          <div class="settings-menu-separator"></div>
          <button
            class="settings-menu-item"
            type="button"
            role="menuitem"
            data-settings-action="plugin-manager"
            on:click={openPluginManager}
          >
            <Icon name="puzzle" size={14} class="settings-menu-icon" />
            <span>{tr('settings.pluginManager')}</span>
          </button>
          {#if settingsPanels.length > 0}
            <div class="settings-menu-separator"></div>
            {#each settingsPanels as panel}
              <button
                class="settings-menu-item"
                type="button"
                role="menuitem"
                data-settings-panel-id={panel.id}
                title={panel.pluginId}
                on:click={() => openSettingsPanel(panel)}
              >
                <Icon name={panel.icon || 'settings'} size={14} class="settings-menu-icon" />
                <span>{panel.title || panel.id}</span>
              </button>
            {/each}
          {/if}
        </div>
      {/if}
    </div>
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

  .settings-menu-wrap {
    position: relative;
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
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
  .settings-button.active {
    border-color: #1a3a5c;
    background: #16213e;
    color: #e0e0f0;
  }

  :global(.settings-icon),
  :global(.settings-chevron),
  :global(.settings-menu-icon) {
    flex-shrink: 0;
    color: currentColor;
  }

  .settings-menu {
    position: fixed;
    right: 0.65rem;
    bottom: 2rem;
    z-index: 10000;
    min-width: 13rem;
    padding: 0.3rem;
    border: 1px solid #1a3a5c;
    border-radius: 6px;
    background: #12122a;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.35);
  }

  .settings-menu-item {
    width: 100%;
    min-height: 1.8rem;
    justify-content: flex-start;
    gap: 0.45rem;
    padding: 0.3rem 0.45rem;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: #cfd8e3;
    font-size: 0.78rem;
    font-weight: 500;
    text-align: left;
  }

  .settings-menu-item:hover {
    background: #0f3460;
    color: #ffffff;
  }

  .settings-menu-heading {
    padding: 0.25rem 0.45rem 0.2rem;
    color: #7f8aa3;
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }

  .language-check {
    width: 0.9rem;
    color: #4ecca3;
    text-align: center;
  }

  .active-language {
    background: rgba(78, 204, 163, 0.1);
  }

  .settings-menu-separator {
    height: 1px;
    margin: 0.25rem 0.2rem;
    background: #1a3a5c;
  }
</style>
