<script>
  import PluginCard from './PluginCard.svelte';
  import { onMount } from 'svelte';
  import { GetPlugins, GetCapabilities, GetPermissions, GetContributions, ReloadPlugins, GetVaultStatus, GetVaultPluginState, EnablePlugin, DisablePlugin } from '../../../wailsjs/go/api/App';

  let plugins = [];
  let capabilities = [];
  let permissions = [];
  let contributions = {};
  let loading = true;
  let error = '';
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let vaultPluginState = { enabledPlugins: [], disabledPlugins: [], desiredPlugins: [] };
  let settingsPanel = null;
  let settingsData = {};
  let settingsPluginId = '';

  $: vaultOpen = vaultStatus.status === 'open';
  $: missingInstalled = computeMissingInstalled();

  function computeMissingInstalled() {
    if (!vaultPluginState.desiredPlugins) return [];
    const installedIDs = new Set(plugins.map(p => p.manifest?.id).filter(Boolean));
    return (vaultPluginState.desiredPlugins || []).filter(dp => !installedIDs.has(dp.id));
  }

  async function loadAll() {
    error = '';
    loading = true;
    try {
      const p = await GetPlugins();
      plugins = p || [];
    } catch (e) {
      error = 'GetPlugins: ' + String(e);
      loading = false;
      return;
    }
    // Vault status — non-critical
    GetVaultStatus().then(v => { vaultStatus = v || { status: 'unknown', path: '', vaultId: '' }; }).catch(() => {});
    // Vault plugin state
    if (vaultStatus.status === 'open') {
      GetVaultPluginState().then(s => { vaultPluginState = s || { enabledPlugins: [], disabledPlugins: [], desiredPlugins: [] }; }).catch(() => {});
    }
    // Capabilities and permissions are non-critical — load async
    GetCapabilities().then(c => { capabilities = c || []; }).catch(() => {});
    GetPermissions().then(p => { permissions = p || []; }).catch(() => {});
    GetContributions().then(c => { contributions = c || {}; }).catch(() => {});
    loading = false;
  }

  onMount(() => { loadAll(); });

  async function reload() {
    loading = true;
    error = '';
    try {
      await ReloadPlugins();
    } catch (e) {
      error = 'Reload: ' + String(e);
      loading = false;
      return;
    }
    await loadAll();
  }

  async function enablePlugin(pluginId) {
    const err = await EnablePlugin(pluginId);
    if (err) {
      error = 'Enable: ' + err;
      return;
    }
    await reload();
  }

  async function disablePlugin(pluginId) {
    const err = await DisablePlugin(pluginId);
    if (err) {
      error = 'Disable: ' + err;
      return;
    }
    await reload();
  }

  $: totalPlugins = plugins.length;
  $: totalCaps = capabilities.length;
  $: totalPerms = permissions.length;

  function openSettings(pluginId) {
    const panel = (contributions.settingsPanels || []).find(sp => sp.pluginId === pluginId);
    if (panel) {
      settingsPanel = panel;
      settingsPluginId = pluginId;
      // Load existing settings from Wails backend
      import('../../../wailsjs/go/api/App').then(mod => {
        mod.ReadPluginSettings(pluginId).then(data => {
          settingsData = data || {};
        }).catch(() => { settingsData = {}; });
      });
    }
  }

  function saveSettings() {
    try {
      import('../../../wailsjs/go/api/App').then(mod => {
        mod.WritePluginSettings(settingsPluginId, settingsData).then(err => {
          if (err) console.error('WritePluginSettings:', err);
        }).catch(e => console.error('WritePluginSettings:', e));
      });
    } catch (e) {
      console.error('saveSettings:', e);
    }
  }
</script>

<div class="plugin-manager">
  <header>
    <div class="header-left">
      <h2>Plugin Manager</h2>
      {#if vaultStatus.status !== 'unknown'}
        <span class="vault-badge" class:vault-open={vaultStatus.status === 'open'} class:vault-not-created={vaultStatus.status === 'not-created'} class:vault-closed={vaultStatus.status === 'closed'} class:vault-error={vaultStatus.status === 'error'}>
          Vault: {vaultStatus.status}{#if vaultStatus.path} ({vaultStatus.path}){/if}
        </span>
      {/if}
    </div>
    <button class="reload-btn" on:click={reload} type="button" disabled={loading}>
      {loading ? '⟳ Loading...' : '⟳ Reload'}
    </button>
  </header>

  {#if loading}
    <div class="loading">Scanning plugin directories...</div>
  {:else if error}
    <div class="error">
      <div class="error-icon">⚠</div>
      <div class="error-message">{error}</div>
      <button class="retry-btn" on:click={loadAll} type="button">⟳ Retry</button>
    </div>
  {:else}
    <div class="summary">
      <span class="badge">{totalPlugins} plugin(s) discovered</span>
      <span class="badge">{totalCaps} capabilities registered</span>
      <span class="badge">{totalPerms} permissions known</span>
    </div>

    {#if plugins.length === 0 && missingInstalled.length === 0}
      <div class="empty">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
        </div>
        <p>No plugins found</p>
        <p class="hint">Plugin directories scanned:</p>
        <ul class="hint-list">
          <li><code>~/.config/verstak/plugins/</code> — user plugins</li>
          <li><code>./plugins/</code> — bundled plugins (app directory)</li>
        </ul>
        <p class="hint">Place a plugin folder with <code>plugin.json</code> in one of these directories and click Reload.</p>
      </div>
    {:else}
      <div class="plugin-list">
        {#each plugins as p}
          <PluginCard {p} {capabilities} {permissions} {contributions} {vaultOpen} onSettings={openSettings} onEnable={enablePlugin} onDisable={disablePlugin} />
        {/each}
      </div>
    {/if}

    {#if missingInstalled.length > 0}
      <div class="missing-section">
        <h3>Missing Installed Plugins</h3>
        <p class="missing-hint">These plugins are required by this vault but their packages are not installed locally.</p>
        <div class="plugin-list">
          {#each missingInstalled as mp}
            <div class="plugin-card missing-card">
              <div class="card-header">
                <div class="plugin-id">
                  <span class="status-dot" style="background: #e94560"></span>
                  <strong>{mp.id}</strong>
                  {#if mp.version}<span class="version">v{mp.version}</span>{/if}
                </div>
                <span class="status-badge" style="color: #e94560">missing</span>
              </div>
              <p class="missing-text">
                This plugin is listed in the vault's desired plugins but the package is not installed.
                {#if mp.source && mp.source !== 'unknown'}
                  <span class="source-hint">Source: {mp.source}</span>
                {/if}
              </p>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    {#if capabilities.length > 0}
      <details class="registry-section">
        <summary>Capability Registry ({totalCaps})</summary>
        <table>
          <thead>
            <tr><th>Capability</th><th>Provider</th><th>Source</th><th>Status</th></tr>
          </thead>
          <tbody>
            {#each capabilities as cap}
              <tr>
                <td><code>{cap.name}</code></td>
                <td>{cap.pluginId}</td>
                <td><span class="source-badge" class:source-core={cap.pluginId === 'verstak-desktop'} class:source-plugin={cap.pluginId !== 'verstak-desktop'}>{cap.pluginId === 'verstak-desktop' ? 'core' : 'plugin'}</span></td>
                <td><span class="status-{cap.status}">{cap.status}</span></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </details>
    {/if}
  {/if}

  <!-- Settings Panel Modal -->
  {#if settingsPanel}
  <div class="modal-overlay" on:click|self={() => settingsPanel = null}>
  <div class="modal" role="dialog" aria-modal="true" aria-label="Plugin Settings">
    <div class="modal-header">
      <h3>{settingsPanel.item.title}</h3>
      <button class="modal-close" on:click={() => settingsPanel = null} type="button">✕</button>
    </div>
    <div class="modal-body">
      <p class="settings-hint">Plugin: <code>{settingsPluginId}</code></p>
      <p class="settings-hint">Component: <code>{settingsPanel.item.component}</code></p>

      {#if settingsPanel.item.id === 'verstak.platform-test.settings'}
        <div class="settings-form">
          <h4>Test Settings</h4>
          <div class="form-row">
            <label for="test-name">Test Name</label>
            <input id="test-name" type="text" bind:value={settingsData.testName} placeholder="Enter test name" />
          </div>
          <div class="form-row">
            <label for="test-interval">Test Interval (seconds)</label>
            <input id="test-interval" type="number" bind:value={settingsData.testInterval} min="1" max="300" />
          </div>
          <div class="form-row">
            <label><input type="checkbox" bind:checked={settingsData.autoRun} /> Auto-run on startup</label>
          </div>
          <button class="btn-save" on:click={() => saveSettings()} type="button">Save Settings</button>
        </div>
      {:else}
        <p class="placeholder">Settings component: {settingsPanel.item.component}</p>
      {/if}
    </div>
  </div>
  </div>
  {/if}
</div>

<style>
  .plugin-manager {
    max-width: 900px;
    padding-top: 0.5rem;
  }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1.25rem;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid #0f3460;
  }
  .header-left {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }
  h2 { color: #e0e0e0; font-size: 1.3rem; margin: 0; }
  .vault-badge {
    font-size: 0.75rem;
    padding: 0.2rem 0.6rem;
    border-radius: 12px;
    font-weight: 600;
    border: 1px solid;
  }
  .vault-open {
    background: rgba(78, 204, 163, 0.15);
    color: #4ecca3;
    border-color: #4ecca3;
  }
  .vault-not-created {
    background: rgba(255, 200, 87, 0.15);
    color: #ffc857;
    border-color: #ffc857;
  }
  .vault-closed {
    background: rgba(160, 160, 184, 0.15);
    color: #a0a0b8;
    border-color: #a0a0b8;
  }
  .vault-error {
    background: rgba(233, 69, 96, 0.15);
    color: #e94560;
    border-color: #e94560;
  }
  .reload-btn {
    background: #0f3460; color: #e0e0e0; border: 1px solid #533483;
    padding: 0.4rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.85rem;
  }
  .reload-btn:hover:not(:disabled) { background: #533483; }
  .reload-btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .loading, .error {
    padding: 2rem; text-align: center; color: #a0a0b8;
  }
  .error { color: #e94560; }
  .error-icon { font-size: 2rem; margin-bottom: 0.5rem; }
  .error-message {
    font-family: monospace; font-size: 0.85rem; margin-bottom: 1rem; word-break: break-word;
  }
  .retry-btn {
    background: #0f3460; color: #e0e0e0; border: 1px solid #533483;
    padding: 0.4rem 1rem; border-radius: 6px; cursor: pointer; font-size: 0.85rem;
  }
  .retry-btn:hover { background: #533483; }
  .summary {
    display: flex; gap: 0.5rem; margin-bottom: 1rem; flex-wrap: wrap;
  }
  .badge {
    background: #16213e; padding: 0.25rem 0.75rem; border-radius: 12px;
    font-size: 0.8rem; color: #a0a0b8; border: 1px solid #0f3460;
  }
  .empty {
    padding: 2rem; text-align: center; color: #a0a0b8;
    background: #16213e; border-radius: 8px; border: 1px dashed #0f3460;
  }
  .empty-icon { margin-bottom: 0.5rem; color: #0f3460; }
  .hint { font-size: 0.85rem; margin-top: 0.5rem; opacity: 0.7; }
  .hint-list { list-style: none; padding: 0; margin: 0.5rem 0; font-size: 0.8rem; opacity: 0.7; }
  .hint-list li { margin: 0.25rem 0; }
  .hint code { background: #0f3460; padding: 0.1rem 0.3rem; border-radius: 3px; }
  .plugin-list { display: flex; flex-direction: column; gap: 0.75rem; margin-bottom: 1.5rem; }

  /* Missing installed section */
  .missing-section {
    margin-bottom: 1.5rem;
  }
  .missing-section h3 {
    color: #e94560;
    font-size: 1rem;
    margin: 0 0 0.25rem;
  }
  .missing-hint {
    color: #a0a0b8;
    font-size: 0.8rem;
    margin: 0 0 0.75rem;
  }
  .missing-card {
    border-color: #e94560;
    opacity: 0.8;
  }
  .missing-text {
    color: #a0a0b8;
    font-size: 0.85rem;
    margin: 0.5rem 0 0;
  }
  .source-hint {
    display: block;
    margin-top: 0.25rem;
    font-size: 0.75rem;
    color: #666;
  }

  .registry-section {
    background: #16213e; border: 1px solid #0f3460;
    border-radius: 8px; padding: 0.75rem; margin-top: 1rem;
  }
  .registry-section summary {
    cursor: pointer; color: #a0a0b8; font-size: 0.9rem; font-weight: 600;
  }
  table { width: 100%; margin-top: 0.5rem; border-collapse: collapse; font-size: 0.85rem; }
  th {
    text-align: left; padding: 0.4rem 0.5rem; color: #a0a0b8; border-bottom: 1px solid #0f3460;
  }
  td { padding: 0.3rem 0.5rem; border-bottom: 1px solid #0f3460; }
  td code { color: #e0e0e0; }
  :global(.status-stable) { color: #4ecca3; }
  :global(.status-draft) { color: #ffc857; }
  :global(.status-deprecated) { color: #e94560; }
  .source-badge {
    font-size: 0.75rem;
    padding: 0.1rem 0.4rem;
    border-radius: 4px;
    font-weight: 600;
  }
  .source-core {
    background: #1a3a5c;
    color: #4ecca3;
    border: 1px solid #4ecca3;
  }
  .source-plugin {
    background: #0f3460;
    color: #a0a0b8;
    border: 1px solid #533483;
  }

  /* ── Modal ── */
  .modal-overlay {
    position: fixed; inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex; align-items: center; justify-content: center;
    z-index: 1000;
  }
  .modal {
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 8px;
    width: 480px;
    max-width: 90vw;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
  }
  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem;
    border-bottom: 1px solid #0f3460;
  }
  .modal-header h3 { margin: 0; color: #e0e0f0; font-size: 1.1rem; }
  .modal-close {
    background: none; border: none; color: #a0a0b8;
    font-size: 1.2rem; cursor: pointer; padding: 0.2rem 0.5rem;
  }
  .modal-close:hover { color: #e94560; }
  .modal-body { padding: 1rem; overflow-y: auto; }
  .settings-hint { color: #666; font-size: 0.8rem; margin: 0.25rem 0; }
  .settings-hint code { color: #4ecca3; }

  /* ── Settings Form ── */
  .settings-form {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .settings-form h4 {
    margin: 0 0 0.5rem 0;
    color: #e0e0f0;
    font-size: 1rem;
  }
  .form-row {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  .form-row label {
    color: #a0a0b8;
    font-size: 0.85rem;
  }
  .form-row input[type="text"],
  .form-row input[type="number"] {
    background: #0f3460;
    border: 1px solid #1a3a5c;
    color: #e0e0f0;
    padding: 0.4rem 0.6rem;
    border-radius: 4px;
    font-size: 0.9rem;
  }
  .form-row input:focus {
    outline: none;
    border-color: #4ecca3;
  }
  .btn-save {
    background: #4ecca3;
    color: #1a1a2e;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: 600;
    margin-top: 0.5rem;
  }
  .btn-save:hover {
    background: #3dbb92;
  }
</style>
