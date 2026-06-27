<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';

  let items = [];
  $: leftItems = items.filter((item) => item.position === 'left');
  $: centerItems = items.filter((item) => item.position === 'center');
  $: rightItems = items.filter((item) => item.position === 'right');

  const inactiveStatuses = new Set(['disabled', 'failed', 'incompatible', 'missing-required-capability']);

  async function loadStatusBar() {
    const [plugins, contributions] = await Promise.all([
      App.GetPlugins().catch(() => []),
      App.GetContributions().catch(() => ({})),
    ]);
    const pluginById = new Map((plugins || []).map((plugin) => [plugin.manifest?.id, plugin]));
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

  onMount(() => {
    loadStatusBar();
    window.addEventListener('verstak:plugins-changed', loadStatusBar);
  });

  onDestroy(() => {
    window.removeEventListener('verstak:plugins-changed', loadStatusBar);
  });
</script>

<footer class="status-bar" aria-label="Status bar">
  <div class="status-bar-group status-left">
    {#each leftItems as item}
      <span class="status-bar-item" data-status-item-id={item.id} title={item.pluginId}>
        {item.label || item.id}
      </span>
    {/each}
  </div>
  <div class="status-bar-group status-center">
    {#each centerItems as item}
      <span class="status-bar-item" data-status-item-id={item.id} title={item.pluginId}>
        {item.label || item.id}
      </span>
    {/each}
  </div>
  <div class="status-bar-group status-right">
    {#each rightItems as item}
      <span class="status-bar-item" data-status-item-id={item.id} title={item.pluginId}>
        {item.label || item.id}
      </span>
    {/each}
  </div>
</footer>

<style>
  .status-bar {
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
    max-width: 18rem;
    overflow: hidden;
    padding: 0.12rem 0.35rem;
    border-radius: 4px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
