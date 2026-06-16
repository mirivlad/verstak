<script>
  import PluginCard from './PluginCard.svelte';
  import { onMount } from 'svelte';

  let plugins = [];
  let capabilities = [];
  let permissions = [];
  let loading = true;
  let error = '';

  async function loadData() {
    loading = true;
    error = '';
    try {
      plugins = await window.go.api.App.GetPlugins();
      capabilities = await window.go.api.App.GetCapabilities();
      permissions = await window.go.api.App.GetPermissions();
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadData();
  });

  async function reload() {
    await window.go.api.App.ReloadPlugins();
    await loadData();
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
    <div class="error">Error: {error}</div>
  {:else}
    <!-- Plugin Count -->
    <div class="summary">
      <span class="badge">{totalPlugins} plugin(s) discovered</span>
      <span class="badge">{totalCaps} capabilities registered</span>
      <span class="badge">{totalPerms} permissions known</span>
    </div>

    <!-- Plugin List -->
    {#if plugins.length === 0}
      <div class="empty">
        <p>No plugins discovered.</p>
        <p class="hint">Place plugins in <code>~/.config/verstak/plugins/</code> or <code>./plugins/</code></p>
      </div>
    {:else}
      <div class="plugin-list">
        {#each plugins as p}
          <PluginCard {p} {capabilities} {permissions} />
        {/each}
      </div>
    {/if}

    <!-- Capabilities Registry -->
    {#if capabilities.length > 0}
      <details class="registry-section">
        <summary>Capability Registry ({totalCaps})</summary>
        <table>
          <thead>
            <tr>
              <th>Capability</th>
              <th>Provider</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            {#each capabilities as cap}
              <tr>
                <td><code>{cap.name}</code></td>
                <td>{cap.pluginId}</td>
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
  .plugin-manager {
    max-width: 900px;
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 1rem;
  }

  h2 {
    color: #e0e0e0;
    font-size: 1.3rem;
  }

  .reload-btn {
    background: #0f3460;
    color: #e0e0e0;
    border: 1px solid #533483;
    padding: 0.4rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.85rem;
  }

  .reload-btn:hover:not(:disabled) {
    background: #533483;
  }

  .reload-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .loading, .error {
    padding: 2rem;
    text-align: center;
    color: #a0a0b8;
  }

  .error {
    color: #e94560;
  }

  .summary {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .badge {
    background: #16213e;
    padding: 0.25rem 0.75rem;
    border-radius: 12px;
    font-size: 0.8rem;
    color: #a0a0b8;
    border: 1px solid #0f3460;
  }

  .empty {
    padding: 2rem;
    text-align: center;
    color: #a0a0b8;
    background: #16213e;
    border-radius: 8px;
    border: 1px dashed #0f3460;
  }

  .hint {
    font-size: 0.85rem;
    margin-top: 0.5rem;
    opacity: 0.7;
  }

  .hint code {
    background: #0f3460;
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
  }

  .plugin-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 1.5rem;
  }

  .registry-section {
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 8px;
    padding: 0.75rem;
    margin-top: 1rem;
  }

  .registry-section summary {
    cursor: pointer;
    color: #a0a0b8;
    font-size: 0.9rem;
    font-weight: 600;
  }

  table {
    width: 100%;
    margin-top: 0.5rem;
    border-collapse: collapse;
    font-size: 0.85rem;
  }

  th {
    text-align: left;
    padding: 0.4rem 0.5rem;
    color: #a0a0b8;
    border-bottom: 1px solid #0f3460;
  }

  td {
    padding: 0.3rem 0.5rem;
    border-bottom: 1px solid #0f3460;
  }

  td code {
    color: #e0e0e0;
  }

  :global(.status-stable) { color: #4ecca3; }
  :global(.status-draft) { color: #ffc857; }
  :global(.status-deprecated) { color: #e94560; }
</style>
