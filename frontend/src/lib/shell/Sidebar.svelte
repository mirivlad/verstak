<script>
  import { onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import WorkspaceTree from './WorkspaceTree.svelte';

  let plugins = [];
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let sidebarItems = [];
  let errorMessage = '';

  let navItems = [
    { id: 'plugin-manager', label: 'Plugin Manager', icon: '🧩' },
  ];

  $: vaultOpen = vaultStatus.status === 'open';

  onMount(async () => {
    let contribErr = false;
    try {
      const [p, v, contribs] = await Promise.all([
        App.GetPlugins().catch(() => []),
        App.GetVaultStatus().catch(() => ({ status: 'unknown', path: '', vaultId: '' })),
        App.GetContributions().catch(() => { contribErr = true; return {}; }),
      ]);
      plugins = p || [];
      vaultStatus = v;
      if (contribErr) {
        errorMessage = 'Failed to load plugin contributions';
      }
      sidebarItems = (contribs.sidebarItems || []).filter(item => {
        const plugin = plugins.find(p => p.manifest?.id === item.pluginId);
        if (!plugin) return false;
        return plugin.status !== 'disabled' && plugin.status !== 'failed' && plugin.status !== 'incompatible' && plugin.status !== 'missing-required-capability';
      });
      sidebarItems.sort((a, b) => (a.position || 100) - (b.position || 100));
    } catch (e) {
      console.error('[Sidebar] load error:', e);
      errorMessage = 'Failed to load sidebar';
    }
  });

  function handleNav(id) {
    window.dispatchEvent(new CustomEvent('verstak:nav', { detail: { viewId: id } }));
  }

  function handleSidebarItem(item) {
    window.dispatchEvent(new CustomEvent('verstak:open-view', { detail: { viewId: item.id, pluginId: item.pluginId } }));
  }
</script>

<aside class="sidebar">
  <div class="sidebar-header">
    <span class="sidebar-logo">📦</span>
    <span class="sidebar-title">Verstak</span>
  </div>

  <nav class="sidebar-nav">
    {#each navItems as item}
      <button
        class="nav-item"
        on:click={() => handleNav(item.id)}
        type="button"
      >
        <span class="nav-icon">{item.icon}</span>
        <span class="nav-label">{item.label}</span>
      </button>
    {/each}
  </nav>

  {#if sidebarItems.length > 0}
    <div class="sidebar-section">
      <span class="section-label">Plugins</span>
      {#each sidebarItems as item}
        <button
          class="nav-item plugin-item"
          on:click={() => handleSidebarItem(item)}
          type="button"
        >
          <span class="nav-icon">{item.icon || '📌'}</span>
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
      <span class="sidebar-error">⚠️ Plugin UI error</span>
    {/if}
    {#if vaultStatus.status !== 'unknown'}
      <span class="vault-indicator" class:vault-open={vaultStatus.status === 'open'} class:vault-closed={vaultStatus.status !== 'open'}>
        ● Vault: {vaultStatus.status}
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

  .sidebar-logo {
    font-size: 1.2rem;
  }

  .sidebar-title {
    color: #e0e0f0;
    font-size: 1rem;
    font-weight: 600;
  }

  .sidebar-nav {
    display: flex;
    flex-direction: column;
    padding: 0.5rem 0.75rem;
    gap: 0.15rem;
  }

  .sidebar-section {
    display: flex;
    flex-direction: column;
    padding: 0.5rem 0.75rem;
    gap: 0.15rem;
    border-top: 1px solid #0f3460;
    margin-top: 0.25rem;
  }

  :global(workspace-tree) {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .section-label {
    color: #666;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 0.25rem 0.5rem;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.45rem 0.75rem;
    background: none;
    border: none;
    color: #a0a0b8;
    font-size: 0.85rem;
    cursor: pointer;
    border-radius: 6px;
    text-align: left;
    width: 100%;
    transition: background 0.15s, color 0.15s;
  }

  .nav-item:hover {
    background: #0f3460;
    color: #e0e0f0;
  }

  .nav-icon {
    font-size: 1rem;
    flex-shrink: 0;
    width: 1.2rem;
    text-align: center;
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

  .vault-indicator {
    font-size: 0.7rem;
    color: #666;
  }

  .vault-indicator.vault-open {
    color: #4ecca3;
  }

  .vault-indicator.vault-closed {
    color: #a0a0b8;
  }

  .sidebar-error {
    display: block;
    font-size: 0.7rem;
    color: #e94560;
    margin-bottom: 0.25rem;
  }
</style>
