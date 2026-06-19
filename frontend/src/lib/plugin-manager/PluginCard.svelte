<script>
  import Icon from '../ui/Icon.svelte';
  export let p = {};
  export let capabilities = [];
  export let permissions = [];
  export let contributions = {};
  export let vaultOpen = false;
  export let settingsPanels = [];
  export let onEnable = () => {};
  export let onDisable = () => {};

  $: m = p.manifest || {};
  $: pluginId = m.id || 'unknown';
  $: hasSettingsPanel = settingsPanels.length > 0;
  $: hasUIPermission = (m.permissions || []).includes('ui.register');
  $: hasStoragePermission = (m.permissions || []).includes('storage.namespace');
  $: hasCommandsPermission = (m.permissions || []).includes('commands.register');

  $: statusColor = ({
    loaded: '#4ecca3',
    degraded: '#ffc857',
    disabled: '#a0a0b8',
    failed: '#e94560',
    incompatible: '#e94560',
    'missing-required-capability': '#e94560',
    loading: '#ffc857',
    discovered: '#a0a0b8',
  }[p.status] || '#a0a0b8');

  $: contribCounts = {
    views: (contributions.views || []).filter(v => v.pluginId === pluginId).length,
    commands: (contributions.commands || []).filter(c => c.pluginId === pluginId).length,
    sidebar: (contributions.sidebarItems || []).filter(s => s.pluginId === pluginId).length,
    statusbar: (contributions.statusBarItems || []).filter(s => s.pluginId === pluginId).length,
    openProviders: (contributions.openProviders || []).filter(o => o.pluginId === pluginId).length,
  };

  $: contribSummary = (() => {
    const parts = [];
    if (contribCounts.views > 0) parts.push(contribCounts.views + ' view' + (contribCounts.views !== 1 ? 's' : ''));
    if (contribCounts.commands > 0) parts.push(contribCounts.commands + ' command' + (contribCounts.commands !== 1 ? 's' : ''));
    if (contribCounts.sidebar > 0) parts.push(contribCounts.sidebar + ' sidebar' + (contribCounts.sidebar !== 1 ? 's' : ''));
    if (contribCounts.statusbar > 0) parts.push(contribCounts.statusbar + ' statusbar' + (contribCounts.statusbar !== 1 ? 's' : ''));
    if (contribCounts.openProviders > 0) parts.push(contribCounts.openProviders + ' openProvider' + (contribCounts.openProviders !== 1 ? 's' : ''));
    return parts.length > 0 ? parts.join(', ') : 'none';
  })();

  $: dangerousPermissions = (m.permissions || []).filter(name => {
    let perm = permissions.find(p => p.name === name);
    return perm && perm.dangerous;
  });

  $: missingRequired = (m.requires || []).filter(req =>
    !capabilities.some(c => c.name === req)
  );

  $: availableOptional = (m.optionalRequires || []).filter(opt =>
    capabilities.some(c => c.name === opt)
  );

  $: missingOptional = (m.optionalRequires || []).filter(opt =>
    !capabilities.some(c => c.name === opt)
  );

  export let actionFeedback = {}; // { [pluginId]: 'enabling' | 'disabling' | null }

  $: isDisabled = p.status === 'disabled' || !p.enabled;
  $: canToggle = p.status !== 'failed' && p.status !== 'incompatible' && p.status !== 'missing-required-capability' && p.status !== 'discovered';
  $: isBusy = actionFeedback[pluginId] != null;
  $: busyAction = actionFeedback[pluginId] || null;
</script>

<div class="plugin-card" class:disabled={isDisabled} class:failed={p.status === 'failed'}>
  <div class="card-header">
    <div class="plugin-id">
      <span class="status-dot" style="background: {statusColor}"></span>
      <strong>{pluginId}</strong>
      <span class="version">v{m.version || '?'}</span>
    </div>
    <span class="status-badge" style="color: {statusColor}">{p.status}</span>
  </div>

  {#if p.status === 'degraded'}
    <p class="degraded-text">Plugin is usable, but some optional capabilities are unavailable.</p>
  {/if}

  {#if m.description}
    <p class="description">{m.description}</p>
  {/if}

  <div class="card-meta">
    <div class="meta-row">
      <span class="label">Name:</span>
      <span>{m.name || '-'}</span>
    </div>
    <div class="meta-row">
      <span class="label">API Version:</span>
      <span>{m.apiVersion || '-'}</span>
    </div>
    <div class="meta-row">
      <span class="label">Source:</span>
      <span>{m.source || 'unknown'}</span>
    </div>
    <div class="meta-row">
      <span class="label">Root:</span>
      <span class="path">{p.rootPath || '-'}</span>
    </div>
    <div class="meta-row">
      <span class="label">Contributions:</span>
      <span>{contribSummary}</span>
    </div>
  </div>

  <!-- Capabilities -->
  <div class="section">
    <span class="section-title">Provides</span>
    <div class="tags">
      {#each m.provides || [] as cap}
        <span class="tag provides">{cap}</span>
      {/each}
    </div>
  </div>

  {#if m.requires && m.requires.length > 0}
    <div class="section">
      <span class="section-title">Requires</span>
      <div class="tags">
        {#each m.requires as req}
          {@const found = capabilities.some(c => c.name === req)}
          <span class="tag" class:required-ok={found} class:required-missing={!found}>
            {req}
            {#if found}<span class="check">✓</span>{/if}
          </span>
        {/each}
      </div>
      {#if missingRequired.length > 0}
        <p class="warning"><Icon name="warning" size={12} /> Missing required capabilities: {missingRequired.join(', ')}</p>
      {/if}
    </div>
  {/if}

  {#if m.optionalRequires && m.optionalRequires.length > 0}
    <div class="section">
      <span class="section-title">Optional Requires</span>
      <div class="tags">
        {#each m.optionalRequires as opt}
          {@const found = capabilities.some(c => c.name === opt)}
          <span class="tag" class:optional-ok={found} class:optional-missing={!found}>
            {opt}
            {#if found}<span class="check">✓</span>{/if}
          </span>
        {/each}
      </div>
      {#if missingOptional.length > 0}
        <p class="info"><Icon name="warning" size={12} /> Optional capabilities not available — plugin running in degraded mode</p>
      {/if}
    </div>
  {/if}

  <!-- Permissions -->
  {#if m.permissions && m.permissions.length > 0}
    <div class="section">
      <span class="section-title">Permissions</span>
      <div class="tags">
        {#each m.permissions as perm}
          {@const isDangerous = dangerousPermissions.includes(perm)}
          <span class="tag" class:dangerous={isDangerous}>
            {perm}
            {#if isDangerous}<Icon name="warning" size={12} class="danger-icon" />{/if}
          </span>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Error -->
  {#if p.error}
    <div class="error-box">{p.error}</div>
  {/if}

  <!-- Actions -->
  <div class="card-actions">
    {#if hasSettingsPanel}
      <button class="btn-settings" on:click={() => window.dispatchEvent(new CustomEvent('verstak:open-settings', { detail: { pluginId: m.id, panelId: settingsPanels[0]?.id } }))} type="button" disabled={isDisabled || p.status === 'failed'}>
        <Icon name="gear" size={14} /> Settings
      </button>
    {/if}
    {#if vaultOpen && canToggle}
      {#if isDisabled}
        <button class="btn-enable" on:click={() => onEnable(m.id)} type="button" disabled={isBusy}>
          {#if busyAction === 'enabling'}⟳ Enabling...{:else}▶ Enable{/if}
        </button>
      {:else}
        <button class="btn-disable" on:click={() => onDisable(m.id)} type="button" disabled={isBusy}>
          {#if busyAction === 'disabling'}⟳ Disabling...{:else}⏸ Disable{/if}
        </button>
      {/if}
    {/if}
    {#if !vaultOpen && canToggle}
      <span class="vault-hint">Open a vault to manage plugin state</span>
    {/if}
  </div>

  <!-- Permission warnings -->
  {#if !hasUIPermission && m.contributes && ((m.contributes.views || []).length > 0 || (m.contributes.sidebarItems || []).length > 0 || (m.contributes.settingsPanels || []).length > 0)}
    <p class="warning"><Icon name="warning" size={12} /> Plugin has UI contributions but lacks ui.register permission</p>
  {/if}
</div>

<style>
  .plugin-card {
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 8px;
    padding: 1rem;
    min-width: 0;
  }

  .plugin-card.disabled {
    opacity: 0.6;
  }

  .plugin-card.failed {
    border-color: #e94560;
  }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    flex-wrap: wrap;
    margin-bottom: 0.5rem;
  }

  .plugin-id {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    min-width: 0;
    flex-wrap: wrap;
  }

  .plugin-id strong {
    overflow-wrap: anywhere;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    display: inline-block;
  }

  .version {
    color: #a0a0b8;
    font-size: 0.8rem;
  }

  .status-badge {
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-weight: 600;
  }

  .description {
    color: #a0a0b8;
    font-size: 0.85rem;
    margin-bottom: 0.75rem;
  }

  .degraded-text {
    color: #ffc857;
    font-size: 0.8rem;
    margin-bottom: 0.5rem;
    font-style: italic;
  }

  .card-meta {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.3rem;
    margin-bottom: 0.75rem;
    font-size: 0.8rem;
  }

  .meta-row {
    display: flex;
    gap: 0.5rem;
    min-width: 0;
  }

  .label {
    color: #a0a0b8;
    min-width: 80px;
  }

  .path {
    font-family: monospace;
    font-size: 0.75rem;
    color: #a0a0b8;
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .section {
    margin-bottom: 0.5rem;
  }

  .section-title {
    display: block;
    font-size: 0.75rem;
    color: #a0a0b8;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.3rem;
  }

  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.3rem;
  }

  .tag {
    background: #0f3460;
    padding: 0.15rem 0.5rem;
    border-radius: 4px;
    font-size: 0.75rem;
    font-family: monospace;
    color: #e0e0e0;
    max-width: 100%;
    overflow-wrap: anywhere;
  }

  .tag.provides {
    background: #1a3a5c;
    border: 1px solid #533483;
  }

  .tag.required-ok {
    border: 1px solid #4ecca3;
  }

  .tag.required-missing {
    border: 1px solid #e94560;
    color: #e94560;
  }

  .tag.optional-ok {
    border: 1px solid #4ecca3;
  }

  .tag.optional-missing {
    border: 1px solid #ffc857;
    color: #ffc857;
  }

  .tag.dangerous {
    border: 1px solid #e94560;
  }

  .check { color: #4ecca3; margin-left: 2px; }
  :global(.danger-icon) { color: #e94560; margin-left: 2px; vertical-align: middle; }

  .info {
    color: #ffc857;
    font-size: 0.8rem;
    margin-top: 0.3rem;
  }

  .warning {
    color: #ffc857;
    font-size: 0.8rem;
    margin-top: 0.3rem;
  }

  .error-box {
    background: rgba(233, 69, 96, 0.1);
    border: 1px solid #e94560;
    border-radius: 4px;
    padding: 0.5rem;
    margin-top: 0.5rem;
    font-size: 0.8rem;
    color: #e94560;
    font-family: monospace;
  }

  .card-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
    margin-top: 0.75rem;
    padding-top: 0.5rem;
    border-top: 1px solid #0f3460;
  }

  .btn-settings {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    background: #0f3460;
    border: 1px solid #1a3a5c;
    color: #e0e0f0;
    padding: 0.3rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
  }

  .btn-settings:hover {
    background: #1a3a5c;
  }

  .btn-enable {
    background: #4ecca3;
    color: #1a1a2e;
    border: none;
    padding: 0.3rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 600;
  }

  .btn-enable:hover {
    background: #3dbb92;
  }

  .btn-disable {
    background: #533483;
    color: #e0e0f0;
    border: none;
    padding: 0.3rem 0.75rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 600;
  }

  .btn-disable:hover {
    background: #6b44a0;
  }

  .vault-hint {
    color: #666;
    font-size: 0.75rem;
    font-style: italic;
  }

  @media (max-width: 760px) {
    .card-meta {
      grid-template-columns: 1fr;
    }

    .meta-row {
      flex-direction: column;
      gap: 0.15rem;
    }

    .label {
      min-width: 0;
    }
  }
</style>
