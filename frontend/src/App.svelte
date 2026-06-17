<script>
  import PluginManager from './lib/plugin-manager/PluginManager.svelte';
  import Sidebar from './lib/shell/Sidebar.svelte';
  import ViewContainer from './lib/shell/ViewContainer.svelte';
  import VaultSelection from './lib/shell/VaultSelection.svelte';
  import * as App from '../wailsjs/go/api/App';

  let currentView = 'plugin-manager';
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let needsVaultSelection = false;
  let loading = true;

  let activeView = null;
  let activeViewPluginId = '';
  let activeSettingsPluginId = '';
  let activeSettingsPanelId = '';

  async function checkVault() {
    loading = true;
    try {
      const settings = await App.GetAppSettings();
      vaultStatus = await App.GetVaultStatus() || { status: 'unknown', path: '', vaultId: '' };

      if (!settings.currentVaultPath || vaultStatus.status !== 'open') {
        needsVaultSelection = true;
      } else {
        needsVaultSelection = false;
      }
    } catch (e) {
      console.error('[App] startup check failed:', e);
      needsVaultSelection = true;
    }
    loading = false;
  }

  function onVaultOpened() {
    needsVaultSelection = false;
    vaultStatus = { status: 'open', path: '', vaultId: '' };
  }

  function onNav(e) {
    currentView = e.detail.viewId;
  }

  function onOpenView(e) {
    activeView = e.detail.viewId;
    activeViewPluginId = e.detail.pluginId || '';
    currentView = 'plugin-view';
  }

  function onOpenSettings(e) {
    activeSettingsPluginId = e.detail.pluginId;
    activeSettingsPanelId = e.detail.panelId || '';
    currentView = 'plugin-manager';
  }

  function onCloseSettings() {
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
  }

  checkVault();
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

    <section class="content">
      {#if currentView === 'plugin-manager'}
        <PluginManager {activeSettingsPluginId} {activeSettingsPanelId} />
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

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #1a1a2e;
    color: #e0e0f0;
    overflow: hidden;
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
    background: #1a1a2e;
  }

  .content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 1.5rem;
  }
</style>
