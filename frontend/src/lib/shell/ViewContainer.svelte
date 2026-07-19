<script>
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import { i18n } from '../i18n/index.js';

  export let activeView = null;
  export let activeViewPluginId = null;

  let views = [];
  let plugins = [];
  let renderError = null;
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;
  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);

  async function loadViews() {
    try {
      const [contribs, pluginList] = await Promise.all([
        App.GetContributions().catch(() => ({ views: [] })),
        App.GetPlugins().catch(() => []),
      ]);
      await Promise.all((pluginList || []).map((plugin) => (
        i18n.loadPlugin(plugin.manifest?.id, plugin.manifest?.localization).catch(() => {})
      )));
      views = i18n.localizeContributionSummary(contribs || {}).views || [];
      plugins = (pluginList || []).map((plugin) => i18n.localizePlugin(plugin));
    } catch (e) {
      console.error('[ViewContainer] load error:', e);
    }
  }

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed) loadViews();
    });
    loadViews();
  });

  onDestroy(() => {
    if (unsubscribeLocale) unsubscribeLocale();
  });

  $: currentView = views.find(v => v.id === activeView && v.pluginId === activeViewPluginId);
  $: currentPlugin = currentView
    ? plugins.find(p => p.manifest?.id === currentView.pluginId)
    : null;
  $: pluginStatus = currentPlugin ? currentPlugin.status : 'unknown';
  $: hasFrontend = currentPlugin?.manifest?.frontend?.entry != null;
  $: hostPluginId = currentView?.pluginId || activeViewPluginId;
  $: hostComponentId = currentView?.component || null;

  $: if (currentView) {
    window.dispatchEvent(new CustomEvent('verstak:content-title-changed', {
      detail: { title: currentView.title, icon: currentView.icon || '' }
    }));
  }

  // Reset render error when view changes
  $: if (activeView) {
    renderError = null;
  }

  function onHostError(e) {
    renderError = e.detail?.message || tr('pluginView.error');
  }
</script>

{#key `${activeViewPluginId}:${activeView}`}
  {#if renderError}
    <div class="view-container">
      <div class="error-boundary">
        <div class="error-fallback vt-inline-alert error">
          <Icon name="warning" size={24} class="error-icon" />
          <p class="error-title">{tr('pluginView.failed')}</p>
          <details class="error-details">
            <summary>{tr('common.details')}</summary>
            <p class="error-text">{renderError}</p>
          </details>
        </div>
      </div>
    </div>
  {:else if currentView}
    <div class="view-container">
      <div class="view" class:degraded={pluginStatus === 'degraded'}>
        <div class="view-content">
          {#if hasFrontend}
            <PluginBundleHost
              pluginId={hostPluginId}
              componentId={hostComponentId}
            />
          {:else}
            <div class="placeholder">
              <p class="placeholder-label">{tr('pluginView.noVisual')}</p>
              <details class="placeholder-details">
                <summary>{tr('common.details')}</summary>
                <p class="placeholder-info"><span class="placeholder-key">{tr('common.plugin')}:</span> <strong>{currentView.pluginId}</strong></p>
                <p class="placeholder-info"><span class="placeholder-key">{tr('pluginView.viewId')}:</span> <code>{currentView.id}</code></p>
                <p class="placeholder-info"><span class="placeholder-key">{tr('common.component')}:</span> <code>{currentView.component}</code></p>
              </details>
              <p class="placeholder-badge vt-badge">{tr('pluginView.bundleUnavailable')}</p>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {:else if activeView}
    <div class="view-container empty">
      <p>{tr('pluginView.unavailable')}</p>
    </div>
  {:else}
    <div class="view-container empty">
      <p>{tr('pluginView.select')}</p>
      <p class="sub">{tr('pluginView.selectHint')}</p>
    </div>
  {/if}
{/key}

<style>
  .view-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    background: var(--vt-color-background);
  }
  .view {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    padding: 1.5rem;
  }
  .view.degraded {
    border-left: 3px solid var(--vt-color-warning);
  }
  .view-content {
    flex: 1;
    min-width: 0;
  }
  .placeholder {
    color: var(--vt-color-text-muted);
    font-style: italic;
    padding: 2rem;
    text-align: center;
    border: 1px dashed var(--vt-color-border-strong);
    border-radius: var(--vt-radius-lg);
  }
  .placeholder-label {
    font-size: 1rem;
    color: var(--vt-color-text-secondary);
    font-weight: 600;
    margin-bottom: 1rem;
    font-style: normal;
  }
  .placeholder-info {
    font-size: 0.85rem;
    color: var(--vt-color-text-muted);
    margin: 0.3rem 0;
    font-style: normal;
  }
  .placeholder-key {
    color: var(--vt-color-text-muted);
  }
  .placeholder-info strong { color: var(--vt-color-accent); }
  .placeholder-info code {
    color: var(--vt-color-text-primary);
    background: var(--vt-color-surface-muted);
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
    font-size: 0.8rem;
  }
  .placeholder-badge {
    display: inline-block;
    margin-top: 1rem;
    padding: 0.25rem 0.75rem;
    background: var(--vt-color-surface-muted);
    color: var(--vt-color-text-secondary);
    border-radius: 12px;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.02em;
    font-style: normal;
  }
  .error-boundary {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
  }
  .error-fallback {
    text-align: center;
    padding: 2rem;
  }
  :global(.error-icon) {
    color: var(--vt-color-danger);
  }
  .error-title {
    color: var(--vt-color-danger);
    font-size: 1.1rem;
    font-weight: 600;
    margin: 0.5rem 0;
  }
  .error-text {
    color: var(--vt-color-text-secondary);
    font-size: 0.85rem;
    font-family: monospace;
    margin-top: 0.5rem;
  }
  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--vt-color-text-muted);
    font-size: 1rem;
    text-align: center;
  }
  .empty .sub { font-size: 0.85rem; color: var(--vt-color-text-muted); margin-top: 0.5rem; }
  .placeholder-details,
  .error-details { margin-top: 0.75rem; color: var(--vt-color-text-muted); }
  .placeholder-details summary,
  .error-details summary { cursor: pointer; color: var(--vt-color-text-secondary); }
</style>
