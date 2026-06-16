<script>
  import PluginCard from './PluginCard.svelte';
  import { onMount } from 'svelte';

  let plugins = [];
  let capabilities = [];
  let permissions = [];
  let loading = true;
  let error = '';

  // Wails v2 + webkit2gtk-4.1 production bridge workaround:
  // `await window.go.api.App.Xxx()` deadlocks the JS event loop.
  // Use .then() instead — doesn't suspend the microtask queue.
  // Safety timer guarantees loading=false even if the bridge Promise never settles.

  function call(method, args) {
    return new Promise((resolve, reject) => {
      try {
        if (window.runtime && window.runtime.Call) {
          window.runtime.Call(method, JSON.stringify(args || []))
            .then(result => resolve(result))
            .catch(err => reject(err));
        } else {
          const parts = method.split('.');
          let obj = window.go;
          for (const p of parts) obj = obj[p];
          resolve(obj.apply(null, args || []));
        }
      } catch (e) {
        reject(e);
      }
    });
  }

  function loadPlugins() {
    error = '';
    call('api.App.GetPlugins').then(p => {
      plugins = p || [];
      loading = false;
    }).catch(e => {
      error = 'GetPlugins: ' + String(e);
      loading = false;
    });
  }

  function loadCaps() {
    call('api.App.GetCapabilities').then(c => {
      capabilities = c || [];
    }).catch(e => {
      console.warn('[PluginManager] GetCapabilities:', e);
    });
  }

  function loadPerms() {
    call('api.App.GetPermissions').then(p => {
      permissions = p || [];
    }).catch(e => {
      console.warn('[PluginManager] GetPermissions:', e);
    });
  }

  onMount(() => {
    loadPlugins();
    loadCaps();
    loadPerms();

    // Safety timeout: force loading=false if APIs never respond
    setTimeout(() => {
      if (loading) {
        loading = false;
        error = 'Plugin discovery timed out. Check backend logs.';
      }
    }, 10000);
  });

  function reload() {
    loading = true;
    error = '';
    call('api.App.ReloadPlugins').then(() => {
      loadPlugins();
      loadCaps();
      loadPerms();
    }).catch(e => {
      error = 'Reload: ' + String(e);
      loading = false;
    });
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
      <button class="retry-btn" on:click={loadPlugins} type="button">⟳ Retry</button>
    </div>
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
        <div class="empty-icon">📂</div>
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

  .error-icon {
    font-size: 2rem;
    margin-bottom: 0.5rem;
  }

  .error-message {
    font-family: monospace;
    font-size: 0.85rem;
    margin-bottom: 1rem;
    word-break: break-word;
  }

  .retry-btn {
    background: #0f3460;
    color: #e0e0e0;
    border: 1px solid #533483;
    padding: 0.4rem 1rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.85rem;
  }

  .retry-btn:hover {
    background: #533483;
  }

  .summary {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1rem;
    flex-wrap: wrap;
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

  .empty-icon {
    font-size: 2rem;
    margin-bottom: 0.5rem;
  }

  .hint {
    font-size: 0.85rem;
    margin-top: 0.5rem;
    opacity: 0.7;
  }

  .hint-list {
    list-style: none;
    padding: 0;
    margin: 0.5rem 0;
    font-size: 0.8rem;
    opacity: 0.7;
  }

  .hint-list li {
    margin: 0.25rem 0;
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
