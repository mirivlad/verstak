<script>
  import { onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';

  let sidebarItems = [];
  let activeView = '';
  let plugins = [];
  let contributions = { sidebarItems: [], views: [], commands: [], settingsPanels: [] };

  onMount(async () => {
    try {
      const [contribs, pluginList] = await Promise.all([
        App.GetContributions(),
        App.GetPlugins(),
      ]);
      contributions = contribs;
      plugins = pluginList;
      const pluginMap = new Map(pluginList.map(p => [p.manifest.id, p]));
      sidebarItems = (contribs.sidebarItems || []).filter(item => {
        const plugin = pluginMap.get(item.pluginId);
        return plugin && plugin.manifest.permissions.includes('ui.register');
      });
    } catch (e) {
      console.error('[Sidebar] load error:', e);
    }
  });

  function openView(viewId) {
    activeView = viewId;
    window.dispatchEvent(new CustomEvent('verstak:open-view', { detail: { viewId } }));
  }
</script>

<nav class="sidebar">
  {#each sidebarItems as item}
    <button
      class="sidebar-item"
      class:active={activeView === item.item.view}
      on:click={() => openView(item.item.view)}
      type="button"
    >
      {#if item.item.icon}<span class="icon">{item.item.icon}</span>{/if}
      <span class="label">{item.item.title}</span>
    </button>
  {/each}
</nav>

<style>
  .sidebar {
    width: 200px;
    background: #16213e;
    border-right: 1px solid #0f3460;
    display: flex;
    flex-direction: column;
    padding: 0.5rem 0;
    overflow-y: auto;
  }
  .sidebar-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 1rem;
    background: none;
    border: none;
    color: #a0a0b0;
    cursor: pointer;
    text-align: left;
    font-size: 0.9rem;
    width: 100%;
  }
  .sidebar-item:hover { background: #0f3460; color: #e0e0f0; }
  .sidebar-item.active { background: #0f3460; color: #4ecca3; }
  .icon { font-size: 1.1rem; }
  .label { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
</style>
