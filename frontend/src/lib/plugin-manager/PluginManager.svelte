<script>
  import PluginCard from './PluginCard.svelte';
  import { onMount } from 'svelte';
  import { GetPlugins, GetCapabilities, GetPermissions, GetContributions, ReloadPlugins } from '../../../wailsjs/go/api/App';

  let plugins = [];
  let capabilities = [];
  let permissions = [];
  let contributions = {};
  let loading = true;
  let error = '';

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

  $: totalPlugins = plugins.length;
  $: totalCaps = capabilities.length;
  $: totalPerms = permissions.length;
</script>

<div class="plugin-manager">
  <header>
    <h2>Plugin Manager</h2>
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

    {#if plugins.length === 0}
      <div class="empty">
        <div class="empty-icon">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
        </div>        <p>No plugins found</p>
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
          <PluginCard {p} {capabilities} {permissions} {contributions} />
        {/each}
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
</div>

<style>
  .plugin-manager { max-width: 900px; }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
  }
  h2 { color: #e0e0e0; font-size: 1.3rem; }
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
</style>
