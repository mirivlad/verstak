<script>
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import { onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';

  export let activeView = null;
  export let activeViewPluginId = null;

  let views = [];
  let plugins = [];
  let renderError = null;

  onMount(async () => {
    try {
      const [contribs, pluginList] = await Promise.all([
        App.GetContributions().catch(() => ({ views: [] })),
        App.GetPlugins().catch(() => []),
      ]);
      views = contribs.views || [];
      plugins = pluginList;
    } catch (e) {
      console.error('[ViewContainer] load error:', e);
    }
  });

  $: currentView = views.find(v => v.id === activeView && v.pluginId === activeViewPluginId);
  $: currentPlugin = currentView
    ? plugins.find(p => p.manifest?.id === currentView.pluginId)
    : null;
  $: pluginStatus = currentPlugin ? currentPlugin.status : 'unknown';
  $: hasFrontend = currentPlugin?.manifest?.frontend?.entry != null;
  $: hostPluginId = currentView?.pluginId || activeViewPluginId;
  $: hostComponentId = currentView?.component || null;

  // Reset render error when view changes
  $: if (activeView) {
    renderError = null;
  }

  function onHostError(e) {
    renderError = e.detail?.message || 'Plugin view error';
  }
</script>

{#key `${activeViewPluginId}:${activeView}`}
  {#if renderError}
    <div class="view-container">
      <div class="error-boundary">
        <div class="error-fallback">
          <span class="error-icon">⚠</span>
          <p class="error-title">Plugin UI failed</p>
          <p class="error-text">{renderError}</p>
        </div>
      </div>
    </div>
  {:else if currentView}
    <div class="view-container">
      <div class="view" class:degraded={pluginStatus === 'degraded'}>
        <div class="view-header">
          <span class="view-icon">{currentView.icon || '📦'}</span>
          <h2>{currentView.title}</h2>
          {#if hasFrontend}
            <span class="frontend-badge">frontend bundle</span>
          {:else}
            <span class="no-frontend-badge">no frontend bundle</span>
          {/if}
        </div>
        <div class="view-content">
          {#if hasFrontend}
            <PluginBundleHost
              pluginId={hostPluginId}
              componentId={hostComponentId}
            />
          {:else}
            <div class="placeholder">
              <p class="placeholder-label">Plugin View Host</p>
              <p class="placeholder-info"><span class="placeholder-key">Plugin:</span> <strong>{currentView.pluginId}</strong></p>
              <p class="placeholder-info"><span class="placeholder-key">View ID:</span> <code>{currentView.id}</code></p>
              <p class="placeholder-info"><span class="placeholder-key">Component:</span> <code>{currentView.component}</code></p>
              <p class="placeholder-badge">frontend bundle not available</p>
            </div>
          {/if}
        </div>
      </div>
    </div>
  {:else if activeView}
    <div class="view-container empty">
      <p>View "{activeView}" not found in contributions</p>
    </div>
  {:else}
    <div class="view-container empty">
      <p>Select a plugin view from the sidebar</p>
      <p class="sub">Plugin views will appear here</p>
    </div>
  {/if}
{/key}

<style>
  .view-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: #1a1a2e;
  }
  .view {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 1.5rem;
  }
  .view.degraded {
    border-left: 3px solid #ffc857;
  }
  .view-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid #16213e;
  }
  .view-header h2 {
    margin: 0;
    font-size: 1.2rem;
    color: #e0e0f0;
    flex: 1;
  }
  .view-icon { font-size: 1.3rem; }
  .frontend-badge {
    font-size: 0.7rem;
    padding: 0.15rem 0.5rem;
    background: rgba(78, 204, 163, 0.15);
    color: #4ecca3;
    border-radius: 8px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .no-frontend-badge {
    font-size: 0.7rem;
    padding: 0.15rem 0.5rem;
    background: rgba(233, 69, 96, 0.1);
    color: #e94560;
    border-radius: 8px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }
  .view-content {
    flex: 1;
    overflow: auto;
  }
  .placeholder {
    color: #666;
    font-style: italic;
    padding: 2rem;
    text-align: center;
    border: 1px dashed #333;
    border-radius: 8px;
  }
  .placeholder-label {
    font-size: 1rem;
    color: #a0a0b8;
    font-weight: 600;
    margin-bottom: 1rem;
    font-style: normal;
  }
  .placeholder-info {
    font-size: 0.85rem;
    color: #666;
    margin: 0.3rem 0;
    font-style: normal;
  }
  .placeholder-key {
    color: #a0a0b8;
  }
  .placeholder-info strong { color: #4ecca3; }
  .placeholder-info code {
    color: #e0e0f0;
    background: #16213e;
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
    font-size: 0.8rem;
  }
  .placeholder-badge {
    display: inline-block;
    margin-top: 1rem;
    padding: 0.25rem 0.75rem;
    background: #533483;
    color: #e0e0f0;
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
  .error-icon {
    font-size: 2rem;
    color: #e94560;
  }
  .error-title {
    color: #e94560;
    font-size: 1.1rem;
    font-weight: 600;
    margin: 0.5rem 0;
  }
  .error-text {
    color: #a0a0b8;
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
    color: #555;
    font-size: 1rem;
  }
  .empty .sub { font-size: 0.85rem; color: #444; margin-top: 0.5rem; }
</style>
