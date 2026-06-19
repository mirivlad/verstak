<script>
  import PluginManager from './lib/plugin-manager/PluginManager.svelte';
  import Sidebar from './lib/shell/Sidebar.svelte';
  import ViewContainer from './lib/shell/ViewContainer.svelte';
  import VaultSelection from './lib/shell/VaultSelection.svelte';
  import WorkbenchHost from './lib/shell/WorkbenchHost.svelte';
  import WorkspaceHost from './lib/shell/WorkspaceHost.svelte';
  import * as App from '../wailsjs/go/api/App';
  import { debug } from './lib/log/debug.js';
  import { onMount } from 'svelte';
  import { tick } from 'svelte';

  let currentView = 'plugin-manager';
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let needsVaultSelection = false;
  let loading = true;

  let activeView = null;
  let activeViewPluginId = '';
  let activeSettingsPluginId = '';
  let activeSettingsPanelId = '';
  let openedResource = null;

  let workspaceNodes = [];
  let currentWorkspaceNodeId = '';

  function flog(msg) {
    App.WriteFrontendLog('App', msg);
  }

  async function checkVault() {
    debug.log('[App] checkVault: START');
    flog('checkVault: START');
    loading = true;
    try {
      debug.log('[App] checkVault: calling GetAppSettings...');
      const settings = await App.GetAppSettings();
      debug.log('[App] checkVault: GetAppSettings returned', settings);
      flog('checkVault: GetAppSettings returned');

      debug.log('[App] checkVault: calling GetVaultStatus...');
      vaultStatus = await App.GetVaultStatus() || { status: 'unknown', path: '', vaultId: '' };
      debug.log('[App] checkVault: GetVaultStatus returned', vaultStatus);
      flog('checkVault: vaultStatus=' + vaultStatus.status);

      if (!settings.currentVaultPath || vaultStatus.status !== 'open') {
        debug.log('[App] checkVault: vault not open, needsVaultSelection=true');
        flog('checkVault: needsVaultSelection=true');
        needsVaultSelection = true;
      } else {
        debug.log('[App] checkVault: vault open, needsVaultSelection=false');
        flog('checkVault: needsVaultSelection=false');
        needsVaultSelection = false;
      }
    } catch (e) {
      debug.log('[App] checkVault: ERROR', String(e));
      flog('checkVault: ERROR: ' + String(e));
      console.error('[App] startup check failed:', e);
      needsVaultSelection = true;
    }
    loading = false;
    await tick();
    debug.log('[App] checkVault: END, loading=false');
    flog('checkVault: END, loading=false');
  }

  function onVaultOpened() {
    debug.log('[App] onVaultOpened');
    needsVaultSelection = false;
    vaultStatus = { status: 'open', path: '', vaultId: '' };
  }

  function onNav(e) {
    debug.log('[App] onNav:', e.detail.viewId);
    currentView = e.detail.viewId;
  }

  function onOpenView(e) {
    debug.log('[App] onOpenView:', e.detail.viewId, 'plugin:', e.detail.pluginId);
    activeView = e.detail.viewId;
    activeViewPluginId = e.detail.pluginId || '';
    currentView = 'plugin-view';
  }

  function onOpenSettings(e) {
    debug.log('[App] onOpenSettings:', e.detail.pluginId, e.detail.panelId);
    activeSettingsPluginId = e.detail.pluginId;
    activeSettingsPanelId = e.detail.panelId || '';
    currentView = 'plugin-manager';
  }

  function onWorkbenchOpened(e) {
    debug.log('[App] onWorkbenchOpened:', e.detail?.request?.path, e.detail?.providerId);
    openedResource = e.detail;
    currentView = 'workbench';
  }

  function onWorkspaceNodeSelected(e) {
    debug.log('[App] onWorkspaceNodeSelected:', e.detail?.nodeId);
    currentWorkspaceNodeId = e.detail?.nodeId || '';
    workspaceNodes = e.detail?.nodes || workspaceNodes;
    if (currentWorkspaceNodeId) {
      currentView = 'workspace';
    }
  }

  function onCloseSettings() {
    debug.log('[App] onCloseSettings');
    activeSettingsPluginId = '';
    activeSettingsPanelId = '';
  }

  // Listen for events
  if (typeof window !== 'undefined') {
    window.addEventListener('verstak:vault-opened', onVaultOpened);
    window.addEventListener('verstak:nav', onNav);
    window.addEventListener('verstak:open-view', onOpenView);
    window.addEventListener('verstak:open-settings', onOpenSettings);
    window.addEventListener('verstak:close-settings', onCloseSettings);
    window.addEventListener('verstak:workbench-opened', onWorkbenchOpened);
    window.addEventListener('verstak:workspace-node-selected', onWorkspaceNodeSelected);
  }

  onMount(() => { checkVault(); });
</script>

{#if loading}
  <div class="app-loading">
    <p>Loading Verstak...</p>
  </div>
{:else if needsVaultSelection}
  <VaultSelection />
{:else}
  <main>
    <Sidebar />

    <section class="content scroll-surface">
      {#if currentView === 'plugin-manager'}
        <PluginManager {activeSettingsPluginId} {activeSettingsPanelId} />
      {:else if currentView === 'workbench'}
        <WorkbenchHost {openedResource} />
      {:else if currentView === 'workspace'}
        <WorkspaceHost currentNodeId={currentWorkspaceNodeId} nodes={workspaceNodes} />
      {:else}
        <ViewContainer {activeView} {activeViewPluginId} />
      {/if}
    </section>
  </main>
{/if}

<style>
  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(html),
  :global(body),
  :global(#app) {
    width: 100%;
    height: 100%;
  }

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #1a1a2e;
    color: #e0e0f0;
    overflow: hidden;
  }

  :global(button) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    min-height: 2rem;
    padding: 0.4rem 0.85rem;
    border: 1px solid #1a3a5c;
    border-radius: 6px;
    background: #0f3460;
    color: #e0e0f0;
    font: inherit;
    font-size: 0.85rem;
    font-weight: 600;
    line-height: 1.2;
    cursor: pointer;
    transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease, opacity 0.15s ease;
  }

  :global(button:hover:not(:disabled)) {
    background: #1a3a5c;
    border-color: #4ecca3;
    color: #ffffff;
  }

  :global(button:focus-visible) {
    outline: 2px solid #4ecca3;
    outline-offset: 2px;
  }

  :global(button:disabled) {
    opacity: 0.55;
    cursor: not-allowed;
  }

  :global(.btn-primary) {
    background: #4ecca3;
    border-color: #4ecca3;
    color: #101827;
  }

  :global(.btn-primary:hover:not(:disabled)) {
    background: #63d9b3;
    border-color: #63d9b3;
    color: #101827;
  }

  :global(.btn-secondary) {
    background: #0f3460;
    border-color: #533483;
    color: #e0e0f0;
  }

  :global(.btn-danger) {
    background: #e94560;
    border-color: #e94560;
    color: #ffffff;
  }

  :global(.btn-danger:hover:not(:disabled)) {
    background: #ff5b73;
    border-color: #ff5b73;
  }

  :global(.btn-ghost) {
    background: transparent;
    border-color: transparent;
    color: #a0a0b8;
  }

  :global(.btn-ghost:hover:not(:disabled)) {
    background: rgba(15, 52, 96, 0.55);
    border-color: #0f3460;
    color: #e0e0f0;
  }

  :global(.btn-icon) {
    width: 2rem;
    min-width: 2rem;
    padding: 0;
  }

  :global(.scroll-surface) {
    min-width: 0;
    min-height: 0;
    overflow: auto;
    scrollbar-gutter: stable;
  }

  :global(*) {
    scrollbar-width: thin;
    scrollbar-color: #0f3460 #1a1a2e;
  }

  :global(*::-webkit-scrollbar) {
    width: 8px;
    height: 8px;
  }

  :global(*::-webkit-scrollbar-track) {
    background: #1a1a2e;
  }

  :global(*::-webkit-scrollbar-thumb) {
    background: #0f3460;
    border-radius: 4px;
  }

  :global(*::-webkit-scrollbar-thumb:hover) {
    background: #1a4a7a;
  }

  .app-loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100vh;
    background: #1a1a2e;
    color: #a0a0b8;
    font-size: 1rem;
  }

  main {
    display: flex;
    height: 100vh;
    width: 100%;
    background: #1a1a2e;
    overflow: hidden;
  }

  .content {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: clamp(1rem, 2vw, 1.5rem);
  }
</style>
