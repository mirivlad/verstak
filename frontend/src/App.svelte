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

  // Listen for vault-opened event from VaultSelection
  if (typeof window !== 'undefined') {
    window.addEventListener('verstak:vault-opened', onVaultOpened);
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
        <PluginManager />
      {:else}
        <ViewContainer />
      {/if}
    </section>
  </main>
{/if}

<style>
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
