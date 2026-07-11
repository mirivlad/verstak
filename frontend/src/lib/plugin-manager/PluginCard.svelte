<script>
  import Icon from '../ui/Icon.svelte';
  import { onDestroy } from 'svelte';
  import { i18n } from '../i18n/index.js';
  export let p = {};
  export let capabilities = [];
  export let permissions = [];
  export let contributions = {};
  export let vaultOpen = false;
  export let settingsPanels = [];
  export let onEnable = () => {};
  export let onDisable = () => {};
  let locale = i18n.getLocale();
  const unsubscribeLocale = i18n.subscribe((nextLocale) => { locale = nextLocale; });
  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);
  onDestroy(unsubscribeLocale);

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
    searchProviders: (contributions.searchProviders || []).filter(s => s.pluginId === pluginId).length,
    sidebar: (contributions.sidebarItems || []).filter(s => s.pluginId === pluginId).length,
    statusbar: (contributions.statusBarItems || []).filter(s => s.pluginId === pluginId).length,
    openProviders: (contributions.openProviders || []).filter(o => o.pluginId === pluginId).length,
    workspaceItems: (contributions.workspaceItems || []).filter(w => w.pluginId === pluginId).length,
  };

  $: contribSummary = (() => {
    const parts = [];
    if (contribCounts.views > 0) parts.push(tr('pluginCard.count.views', { count: contribCounts.views }));
    if (contribCounts.commands > 0) parts.push(tr('pluginCard.count.commands', { count: contribCounts.commands }));
    if (contribCounts.searchProviders > 0) parts.push(tr('pluginCard.count.searchProviders', { count: contribCounts.searchProviders }));
    if (contribCounts.sidebar > 0) parts.push(tr('pluginCard.count.sidebar', { count: contribCounts.sidebar }));
    if (contribCounts.statusbar > 0) parts.push(tr('pluginCard.count.statusbar', { count: contribCounts.statusbar }));
    if (contribCounts.openProviders > 0) parts.push(tr('pluginCard.count.openProviders', { count: contribCounts.openProviders }));
    if (contribCounts.workspaceItems > 0) parts.push(tr('pluginCard.count.workspace', { count: contribCounts.workspaceItems }));
    return parts.length > 0 ? parts.join(', ') : tr('common.none');
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
    <span class="status-badge" style="color: {statusColor}">{tr(`status.${p.status}`, undefined, p.status)}</span>
  </div>

  {#if p.status === 'degraded'}
    <p class="degraded-text">{tr('pluginCard.degraded')}</p>
  {/if}

  {#if m.description}
    <p class="description">{m.description}</p>
  {/if}

  <div class="card-meta">
    <div class="meta-row">
      <span class="label">{tr('pluginCard.name')}:</span>
      <span>{m.name || '-'}</span>
    </div>
    <div class="meta-row">
      <span class="label">{tr('common.source')}:</span>
      <span>{m.source || tr('common.unknown')}</span>
    </div>
    <div class="meta-row">
      <span class="label">{tr('pluginCard.contributions')}:</span>
      <span>{contribSummary}</span>
    </div>
  </div>

  <details class="plugin-details">
    <summary>{tr('pluginCard.technicalDetails')}</summary>
    <div class="card-meta technical-meta">
      <div class="meta-row">
        <span class="label">{tr('pluginCard.apiVersion')}:</span>
        <span>{m.apiVersion || '-'}</span>
      </div>
      <div class="meta-row">
        <span class="label">{tr('pluginCard.root')}:</span>
        <span class="path">{p.rootPath || '-'}</span>
      </div>
    </div>

    <div class="section">
      <span class="section-title">{tr('pluginCard.provides')}</span>
      <div class="tags">
        {#each m.provides || [] as cap}
          <span class="tag provides">{cap}</span>
        {/each}
      </div>
    </div>

    {#if m.requires && m.requires.length > 0}
    <div class="section">
      <span class="section-title">{tr('pluginCard.requires')}</span>
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
        <p class="warning"><Icon name="warning" size={12} /> {tr('pluginCard.missingRequired')}: {missingRequired.join(', ')}</p>
      {/if}
    </div>
    {/if}

    {#if m.optionalRequires && m.optionalRequires.length > 0}
    <div class="section">
      <span class="section-title">{tr('pluginCard.optionalRequires')}</span>
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
        <p class="info"><Icon name="warning" size={12} /> {tr('pluginCard.optionalUnavailable')}</p>
      {/if}
    </div>
    {/if}

    {#if m.permissions && m.permissions.length > 0}
    <div class="section">
      <span class="section-title">{tr('pluginCard.permissions')}</span>
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
  </details>

  <!-- Error -->
  {#if p.error}
    <div class="error-box">{p.error}</div>
  {/if}

  <!-- Actions -->
  <div class="card-actions">
    {#if hasSettingsPanel}
      <button class="btn-settings" on:click={() => window.dispatchEvent(new CustomEvent('verstak:open-settings', { detail: { pluginId: m.id, panelId: settingsPanels[0]?.id } }))} type="button" disabled={isDisabled || p.status === 'failed'}>
        <Icon name="gear" size={14} /> {tr('settings.title')}
      </button>
    {/if}
    {#if vaultOpen && canToggle}
      {#if isDisabled}
        <button class="btn-enable" on:click={() => onEnable(m.id)} type="button" disabled={isBusy}>
          {#if busyAction === 'enabling'}{tr('pluginCard.enabling')}{:else}{tr('pluginCard.enable')}{/if}
        </button>
      {:else}
        <button class="btn-disable" on:click={() => onDisable(m.id)} type="button" disabled={isBusy}>
          {#if busyAction === 'disabling'}{tr('pluginCard.disabling')}{:else}{tr('pluginCard.disable')}{/if}
        </button>
      {/if}
    {/if}
    {#if !vaultOpen && canToggle}
      <span class="vault-hint">{tr('pluginCard.openVault')}</span>
    {/if}
  </div>

  <!-- Permission warnings -->
  {#if !hasUIPermission && m.contributes && ((m.contributes.views || []).length > 0 || (m.contributes.sidebarItems || []).length > 0 || (m.contributes.settingsPanels || []).length > 0)}
    <p class="warning"><Icon name="warning" size={12} /> {tr('pluginCard.missingUiPermission')}</p>
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

  .technical-meta {
    margin-top: 0.55rem;
  }

  .plugin-details {
    margin: 0.6rem 0;
    padding: 0.45rem 0;
    border-top: 1px solid rgba(15, 52, 96, 0.75);
    border-bottom: 1px solid rgba(15, 52, 96, 0.75);
  }

  .plugin-details summary {
    cursor: pointer;
    color: #8b8ba8;
    font-size: 0.78rem;
    font-weight: 600;
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
