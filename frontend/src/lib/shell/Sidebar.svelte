<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import WorkspaceTree from './WorkspaceTree.svelte';
  import GlobalSearch from './GlobalSearch.svelte';
  import Icon from '../ui/Icon.svelte';
  import { debug } from '../log/debug.js';

  function flog(msg) {
    App.WriteFrontendLog('Sidebar', msg);
  }

  let plugins = [];
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let sidebarItems = [];
  let errorMessage = '';

  $: vaultOpen = vaultStatus.status === 'open';

  async function loadSidebar() {
    debug.log('[Sidebar] onMount: START');
    flog('onMount: START');
    let contribErr = false;
    try {
      debug.log('[Sidebar] onMount: loading plugins/vault/contributions...');
      flog('onMount: loading plugins/vault/contributions...');
      const [p, v, contribs] = await Promise.all([
        App.GetPlugins().catch(() => []),
        App.GetVaultStatus().catch(() => ({ status: 'unknown', path: '', vaultId: '' })),
        App.GetContributions().catch(() => { contribErr = true; return {}; }),
      ]);
      plugins = p || [];
      vaultStatus = v;
      debug.log('[Sidebar] onMount: plugins=' + plugins.length + ' vault=' + vaultStatus.status);
      flog('onMount: plugins=' + plugins.length + ' vault=' + vaultStatus.status);
      if (contribErr) {
        errorMessage = 'Failed to load plugin contributions';
      }
      sidebarItems = (contribs.sidebarItems || []).filter(item => {
        const plugin = plugins.find(p => p.manifest?.id === item.pluginId);
        if (!plugin) return false;
        return plugin.status !== 'disabled' && plugin.status !== 'failed' && plugin.status !== 'incompatible' && plugin.status !== 'missing-required-capability';
      });
      sidebarItems.sort((a, b) => (a.position || 100) - (b.position || 100));
      debug.log('[Sidebar] onMount: sidebarItems=' + sidebarItems.length);
      flog('onMount: sidebarItems=' + sidebarItems.length);
    } catch (e) {
      debug.log('[Sidebar] onMount: ERROR:', String(e));
      flog('onMount: ERROR: ' + String(e));
      console.error('[Sidebar] load error:', e);
      errorMessage = 'Failed to load sidebar';
    }
    debug.log('[Sidebar] onMount: END');
    flog('onMount: END');
  }

  onMount(() => {
    loadSidebar();
    window.addEventListener('verstak:plugins-changed', loadSidebar);
  });

  onDestroy(() => {
    window.removeEventListener('verstak:plugins-changed', loadSidebar);
  });

  function handleSidebarItem(item) {
    debug.log('[Sidebar] handleSidebarItem:', item.id, '-> view:', item.view);
    // Use item.view (the view contribution ID) if available, fall back to item.id
    const viewId = item.view || item.id;
    window.dispatchEvent(new CustomEvent('verstak:open-view', { detail: { viewId, pluginId: item.pluginId } }));
  }
</script>

<aside class="sidebar">
  <div class="sidebar-header">
    <Icon name="logo" size={20} class="sidebar-logo" />
    <span class="sidebar-title">Verstak</span>
  </div>

  {#if vaultOpen}
    <GlobalSearch />
  {/if}

  {#if sidebarItems.length > 0}
    <div class="sidebar-section">
      <span class="section-label">Tools</span>
      {#each sidebarItems as item}
        <button
          class="nav-item plugin-item"
          on:click={() => handleSidebarItem(item)}
          type="button"
        >
          <Icon name={item.icon || 'plugin'} size={16} class="nav-icon icon-plugin" />
          <span class="nav-label">{item.title || item.id}</span>
        </button>
      {/each}
    </div>
  {/if}

  {#if vaultOpen}
    <WorkspaceTree />
  {/if}

  <div class="sidebar-footer">
    {#if errorMessage}
      <span class="sidebar-error">
        <Icon name="warning" size={10} class="sidebar-error-icon" />
        Plugin UI error
      </span>
    {/if}
  </div>
</aside>

<style>
  .sidebar {
    width: 220px;
    min-width: 220px;
    background: #16213e;
    display: flex;
    flex-direction: column;
    border-right: 1px solid #0f3460;
    overflow: hidden;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid #0f3460;
  }

  :global(.sidebar-logo) {
    width: 1.2rem;
    height: 1.2rem;
    color: #4ecca3;
    flex-shrink: 0;
  }

  .sidebar-title {
    color: #e0e0f0;
    font-size: 1rem;
    font-weight: 600;
  }

  .sidebar-section {
    display: flex;
    flex-direction: column;
    padding: 0.45rem 0.6rem 0.55rem;
    gap: 0.15rem;
    border-bottom: 1px solid #0f3460;
  }

  :global(workspace-tree) {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .section-label {
    color: #a0a0b8;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 0.25rem 0.45rem 0.35rem;
    font-weight: 600;
  }

  .nav-item {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 0.45rem;
    min-height: 1.7rem;
    padding: 0.15rem 0.45rem;
    background: none;
    border: none;
    color: #a0a0b8;
    font-size: 0.78rem;
    cursor: pointer;
    border-radius: 3px;
    text-align: left;
    width: 100%;
    transition: background 0.15s, color 0.15s;
  }

  .nav-item:hover {
    background: rgba(15,52,96,0.4);
    color: #e0e0f0;
  }

  :global(.nav-icon) {
    width: 0.9rem;
    height: 0.9rem;
    flex-shrink: 0;
    color: currentColor;
  }
  :global(.nav-icon.icon-plugin) {
    color: #a78bfa;
  }

  .nav-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sidebar-footer {
    margin-top: auto;
    padding: 0.75rem 1.25rem;
    border-top: 1px solid #0f3460;
  }

  .sidebar-error {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.7rem;
    color: #e94560;
    margin-bottom: 0.25rem;
  }
  :global(.sidebar-error-icon) {
    color: #e94560;
  }
</style>
