<script>
  import { onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';

  let appSettings = {};
  let recentVaults = [];
  let error = '';
  let loading = true;
  let creating = false;
  let opening = false;
  let newVaultPath = '';
  let openVaultPath = '';

  onMount(async () => {
    try {
      appSettings = await App.GetAppSettings() || {};
      recentVaults = appSettings.recentVaults || [];
    } catch (e) {
      // App settings might fail if backend not ready — show selection anyway
      console.error('[VaultSelection] load settings:', e);
    }
    loading = false;
  });

  async function createVault() {
    error = '';
    if (!newVaultPath.trim()) {
      error = 'Please enter a path for the new vault';
      return;
    }
    creating = true;
    try {
      // Step 1: Create the vault directory + metadata
      const createErr = await App.CreateVault(newVaultPath.trim());
      if (createErr) {
        error = 'Create vault: ' + createErr;
        creating = false;
        return;
      }
      // Step 2: Open it (registers capabilities, loads plugin state)
      const openErr = await App.OpenVault(newVaultPath.trim());
      if (openErr) {
        error = 'Open vault: ' + openErr;
        creating = false;
        return;
      }
      // Step 3: Save to app settings (set current + add to recent)
      const setErr = await App.SetCurrentVault(newVaultPath.trim());
      if (setErr) {
        // Vault is open but settings save failed — still proceed
        console.warn('[VaultSelection] SetCurrentVault:', setErr);
      }
      // Success — notify app to transition to main UI
      window.dispatchEvent(new CustomEvent('verstak:vault-opened'));
    } catch (e) {
      error = String(e);
      creating = false;
    }
  }

  async function openExistingVault() {
    error = '';
    if (!openVaultPath.trim()) {
      error = 'Please enter a path to an existing vault';
      return;
    }
    opening = true;
    try {
      // Step 1: Open the vault
      const openErr = await App.OpenVault(openVaultPath.trim());
      if (openErr) {
        error = 'Open vault: ' + openErr;
        opening = false;
        return;
      }
      // Step 2: Save to app settings
      const setErr = await App.SetCurrentVault(openVaultPath.trim());
      if (setErr) {
        console.warn('[VaultSelection] SetCurrentVault:', setErr);
      }
      window.dispatchEvent(new CustomEvent('verstak:vault-opened'));
    } catch (e) {
      error = String(e);
      opening = false;
    }
  }

  async function openRecent(path) {
    error = '';
    opening = true;
    try {
      const openErr = await App.OpenVault(path);
      if (openErr) {
        error = 'Open vault: ' + openErr;
        opening = false;
        return;
      }
      const setErr = await App.SetCurrentVault(path);
      if (setErr) {
        console.warn('[VaultSelection] SetCurrentVault:', setErr);
      }
      window.dispatchEvent(new CustomEvent('verstak:vault-opened'));
    } catch (e) {
      error = String(e);
      opening = false;
    }
  }
</script>

{#if loading}
  <div class="vault-selection">
    <div class="vault-selection-inner">
      <p class="loading-text">Loading...</p>
    </div>
  </div>
{:else}
<div class="vault-selection">
  <div class="vault-selection-inner">
    <div class="logo">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#4ecca3" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
        <line x1="12" y1="11" x2="12" y2="17"/>
        <line x1="9" y1="14" x2="15" y2="14"/>
      </svg>
      <h1>Verstak</h1>
      <p class="subtitle">Choose a vault to get started</p>
    </div>

    {#if error}
      <div class="error-box">
        <span class="error-icon">⚠</span>
        <span class="error-text">{error}</span>
      </div>
    {/if}

    <div class="actions">
      <div class="action-card">
        <h3>Create New Vault</h3>
        <p class="hint">Create a new vault folder. This will be your workspace.</p>
        <div class="input-row">
          <input
            type="text"
            bind:value={newVaultPath}
            placeholder="~/Documents/MyVault"
            disabled={creating}
          />
          <button class="btn-primary" on:click={createVault} type="button" disabled={creating}>
            {creating ? 'Creating...' : 'Create & Open'}
          </button>
        </div>
      </div>

      <div class="action-card">
        <h3>Open Existing Vault</h3>
        <p class="hint">Open a vault that already exists on this computer.</p>
        <div class="input-row">
          <input
            type="text"
            bind:value={openVaultPath}
            placeholder="~/Documents/ExistingVault"
            disabled={opening}
          />
          <button class="btn-primary" on:click={openExistingVault} type="button" disabled={opening}>
            {opening ? 'Opening...' : 'Open'}
          </button>
        </div>
      </div>
    </div>

    {#if recentVaults.length > 0}
      <div class="recent-section">
        <h3>Recent Vaults</h3>
        <ul class="recent-list">
          {#each recentVaults as path}
            <li>
              <button class="recent-item" on:click={() => openRecent(path)} type="button" disabled={opening}>
                <span class="recent-icon">📁</span>
                <span class="recent-path">{path}</span>
              </button>
            </li>
          {/each}
        </ul>
      </div>
    {/if}
  </div>
</div>
{/if}

<style>
  .vault-selection {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100vh;
    background: #1a1a2e;
    padding: 2rem;
  }
  .vault-selection-inner {
    max-width: 520px;
    width: 100%;
  }
  .loading-text {
    color: #a0a0b8;
    text-align: center;
    font-size: 1rem;
  }
  .logo {
    text-align: center;
    margin-bottom: 2rem;
  }
  .logo h1 {
    color: #e0e0f0;
    font-size: 1.8rem;
    margin: 0.5rem 0 0.25rem;
  }
  .subtitle {
    color: #a0a0b8;
    font-size: 0.95rem;
    margin: 0;
  }
  .error-box {
    background: rgba(233, 69, 96, 0.1);
    border: 1px solid #e94560;
    border-radius: 8px;
    padding: 0.75rem 1rem;
    margin-bottom: 1.5rem;
    display: flex;
    align-items: flex-start;
    gap: 0.5rem;
    font-size: 0.85rem;
    color: #e94560;
  }
  .error-icon { flex-shrink: 0; }
  .error-text { word-break: break-word; }
  .actions {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    margin-bottom: 1.5rem;
  }
  .action-card {
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 8px;
    padding: 1.25rem;
  }
  .action-card h3 {
    color: #e0e0f0;
    font-size: 1rem;
    margin: 0 0 0.25rem;
  }
  .hint {
    color: #a0a0b8;
    font-size: 0.8rem;
    margin: 0 0 0.75rem;
  }
  .input-row {
    display: flex;
    gap: 0.5rem;
  }
  .input-row input {
    flex: 1;
    background: #0f3460;
    border: 1px solid #1a3a5c;
    color: #e0e0f0;
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    font-size: 0.9rem;
  }
  .input-row input:focus {
    outline: none;
    border-color: #4ecca3;
  }
  .input-row input::placeholder {
    color: #666;
  }
  .btn-primary {
    background: #4ecca3;
    color: #1a1a2e;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 600;
    white-space: nowrap;
  }
  .btn-primary:hover:not(:disabled) {
    background: #3dbb92;
  }
  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .recent-section {
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 8px;
    padding: 1rem 1.25rem;
  }
  .recent-section h3 {
    color: #a0a0b8;
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin: 0 0 0.5rem;
  }
  .recent-list {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  .recent-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    background: none;
    border: none;
    color: #e0e0f0;
    padding: 0.4rem 0;
    cursor: pointer;
    text-align: left;
    font-size: 0.85rem;
  }
  .recent-item:hover {
    color: #4ecca3;
  }
  .recent-item:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .recent-icon { flex-shrink: 0; }
  .recent-path {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
