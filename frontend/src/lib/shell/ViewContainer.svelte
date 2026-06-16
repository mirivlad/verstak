<script>
  import { onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';

  let views = [];
  let activeView = '';
  let pluginStates = {};
  let plugins = [];

  onMount(async () => {
    try {
      const [contribs, pluginList] = await Promise.all([
        App.GetContributions(),
        App.GetPlugins(),
      ]);
      views = contribs.views || [];
      plugins = pluginList;
      for (const p of pluginList) {
        pluginStates[p.manifest.id] = p.status;
      }
    } catch (e) {
      console.error('[ViewContainer] load error:', e);
    }

    window.addEventListener('verstak:open-view', (e) => {
      activeView = e.detail.viewId;
    });
  });

  function getViewStatus(view) {
    const status = pluginStates[view.pluginId];
    if (status === 'failed' || status === 'incompatible') return 'error';
    if (status === 'degraded') return 'degraded';
    return 'ok';
  }
</script>

<div class="view-container">
  {#if activeView}
    {#each views.filter(v => v.item.id === activeView) as view}
      <div class="view" class:degraded={getViewStatus(view) === 'degraded'}>
        <div class="view-header">
          <span class="view-icon">{view.item.icon || '📦'}</span>
          <h2>{view.item.title}</h2>
          {#if getViewStatus(view) === 'degraded'}
            <span class="badge degraded">degraded</span>
          {/if}
        </div>
        <div class="view-content">
          <div class="plugin-view-host" data-view-id={view.item.id} data-component={view.item.component}>
            <p class="placeholder">
              Plugin view: <strong>{view.item.component}</strong>
              <br />
              <span class="sub">from {view.pluginId}</span>
            </p>
          </div>
        </div>
      </div>
    {:else}
      <div class="empty">View "{activeView}" not found in contributions</div>
    {/each}
  {:else}
    <div class="empty">
      <p>Select an item from the sidebar</p>
      <p class="sub">Plugin views will appear here</p>
    </div>
  {/if}
</div>

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
  .view-content {
    flex: 1;
    overflow: auto;
  }
  .plugin-view-host {
    width: 100%;
    min-height: 200px;
  }
  .placeholder {
    color: #666;
    font-style: italic;
    padding: 2rem;
    text-align: center;
    border: 1px dashed #333;
    border-radius: 8px;
  }
  .placeholder strong { color: #4ecca3; }
  .placeholder .sub { font-size: 0.85rem; color: #555; }
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
  .badge {
    padding: 0.15rem 0.5rem;
    border-radius: 3px;
    font-size: 0.75rem;
    font-weight: 600;
  }
  .badge.degraded { background: #ffc857; color: #1a1a2e; }
</style>
