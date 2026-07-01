<script>
  import { onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';

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
      console.error('[VaultSelection] load settings:', e);
    }
    loading = false;
  });

  async function browseNewVault() {
    const path = await App.SelectDirectory();
    if (path) {
      newVaultPath = path;
    }
  }

  async function browseOpenVault() {
    const path = await App.SelectVaultForOpen();
    if (path) {
      openVaultPath = path;
    }
  }

  async function createVault() {
    error = '';
    if (!newVaultPath.trim()) {
      error = 'Choose or enter a folder for the new vault.';
      return;
    }
    creating = true;
    try {
      const createErr = await App.CreateVault(newVaultPath.trim());
      if (createErr) {
        error = 'Could not create vault: ' + createErr;
        creating = false;
        return;
      }
      const openErr = await App.OpenVault(newVaultPath.trim());
      if (openErr) {
        error = 'Could not open vault: ' + openErr;
        creating = false;
        return;
      }
      const setErr = await App.SetCurrentVault(newVaultPath.trim());
      if (setErr) {
        console.warn('[VaultSelection] SetCurrentVault:', setErr);
      }
      window.dispatchEvent(new CustomEvent('verstak:vault-opened'));
    } catch (e) {
      error = String(e);
      creating = false;
    }
  }

  async function openExistingVault() {
    error = '';
    if (!openVaultPath.trim()) {
      error = 'Choose or enter an existing vault.';
      return;
    }
    opening = true;
    try {
      const openErr = await App.OpenVault(openVaultPath.trim());
      if (openErr) {
        error = 'Could not open vault: ' + openErr;
        opening = false;
        return;
      }
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
        error = 'Could not open vault: ' + openErr;
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
      <p class="subtitle">Choose a vault to start working</p>
    </div>

    {#if error}
      <div class="error-box">
        <Icon name="warning" size={14} class="error-icon" />
        <span class="error-text">{error}</span>
      </div>
    {/if}

    <div class="actions">
      <div class="action-card">
        <h3>Create a new vault</h3>
        <p class="hint">Create a local vault folder for workspaces and projects.</p>
        <div class="input-row">
          <input
            type="text"
            bind:value={newVaultPath}
            placeholder="Choose or enter a path..."
            disabled={creating}
          />
          <button class="btn-secondary" on:click={browseNewVault} type="button" disabled={creating}>
            Browse...
          </button>
        </div>
        <div class="button-row">
          <button class="btn-primary" on:click={createVault} type="button" disabled={creating}>
            {creating ? 'Creating...' : 'Create vault'}
          </button>
        </div>
      </div>

      <div class="action-card">
        <h3>Open an existing vault</h3>
        <p class="hint">Use a vault that is already on this computer.</p>
        <div class="input-row">
          <input
            type="text"
            bind:value={openVaultPath}
            placeholder="Choose or enter a path..."
            disabled={opening}
          />
          <button class="btn-secondary" on:click={browseOpenVault} type="button" disabled={opening}>
            Browse...
          </button>
        </div>
        <div class="button-row">
          <button class="btn-secondary open-existing-btn" on:click={openExistingVault} type="button" disabled={opening}>
            {opening ? 'Opening...' : 'Open existing'}
          </button>
        </div>
      </div>
    </div>

    {#if recentVaults.length > 0}
      <div class="recent-section">
        <h3>Recent vaults</h3>
        <ul class="recent-list">
          {#each recentVaults as path}
            <li>
              <button class="recent-item" on:click={() => openRecent(path)} type="button" disabled={opening}>
                <Icon name="vault" size={16} class="recent-icon" />
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
    background: var(--vt-color-background);
    padding: 2rem;
  }
  .vault-selection-inner {
    max-width: 560px;
    width: 100%;
  }
  .loading-text {
    color: var(--vt-color-text-muted);
    text-align: center;
    font-size: 1rem;
  }
  .logo {
    text-align: center;
    margin-bottom: 2rem;
  }
  .logo h1 {
    color: var(--vt-color-text-primary);
    font-size: 1.8rem;
    margin: 0.5rem 0 0.25rem;
  }
  .subtitle {
    color: var(--vt-color-text-muted);
    font-size: 0.95rem;
    margin: 0;
  }
  .error-box {
    background: var(--vt-color-danger-muted);
    border: 1px solid rgba(233, 69, 96, 0.5);
    border-radius: var(--vt-radius-lg);
    padding: 0.75rem 1rem;
    margin-bottom: 1.5rem;
    display: flex;
    align-items: flex-start;
    gap: 0.5rem;
    font-size: 0.85rem;
    color: #ffc6ce;
  }
  :global(.error-icon) { flex-shrink: 0; }
  .error-text { word-break: break-word; }
  .actions {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    margin-bottom: 1.5rem;
  }
  .action-card {
    background: var(--vt-color-surface);
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    padding: 1.25rem;
  }
  .action-card h3 {
    color: var(--vt-color-text-primary);
    font-size: 1rem;
    margin: 0 0 0.25rem;
  }
  .hint {
    color: var(--vt-color-text-muted);
    font-size: 0.8rem;
    margin: 0 0 0.75rem;
  }
  .input-row {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
  .input-row input {
    flex: 1;
    background: #0f1424;
    border: 1px solid var(--vt-color-border-strong);
    color: var(--vt-color-text-primary);
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    font-size: 0.9rem;
  }
  .input-row input:focus {
    outline: none;
    border-color: var(--vt-color-accent);
    box-shadow: var(--vt-focus-ring);
  }
  .input-row input::placeholder {
    color: var(--vt-color-text-muted);
  }
  .button-row {
    display: flex;
    justify-content: flex-end;
  }
  .btn-primary {
    background: var(--vt-color-accent);
    color: #101827;
    border: none;
    padding: 0.5rem 1.25rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 600;
  }
  .btn-primary:hover:not(:disabled) {
    background: #3dbb92;
  }
  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .btn-secondary {
    background: var(--vt-color-surface-hover);
    color: var(--vt-color-text-secondary);
    border: 1px solid var(--vt-color-border-strong);
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.85rem;
    white-space: nowrap;
  }
  .btn-secondary:hover:not(:disabled) {
    background: var(--vt-color-surface-hover);
    color: var(--vt-color-text-primary);
  }
  .btn-secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .recent-section {
    background: var(--vt-color-surface);
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    padding: 1rem 1.25rem;
  }
  .recent-section h3 {
    color: var(--vt-color-text-muted);
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
    color: var(--vt-color-text-primary);
    padding: 0.4rem 0;
    cursor: pointer;
    text-align: left;
    font-size: 0.85rem;
  }
  .recent-item:hover {
    color: var(--vt-color-accent);
  }
  .recent-item:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  :global(.recent-icon) { flex-shrink: 0; }
  .recent-path {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
