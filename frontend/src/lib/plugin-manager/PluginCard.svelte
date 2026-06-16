<script>
  export let p = {};
  export let capabilities = [];
  export let permissions = [];

  $: m = p.manifest || {};

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
</script>

<div class="plugin-card" class:disabled={!p.enabled} class:failed={p.status === 'failed'}>
  <div class="card-header">
    <div class="plugin-id">
      <span class="status-dot" style="background: {statusColor}"></span>
      <strong>{m.id || 'unknown'}</strong>
      <span class="version">v{m.version || '?'}</span>
    </div>
    <span class="status-badge" style="color: {statusColor}">{p.status}</span>
  </div>

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
        <p class="warning">⚠ Missing required capabilities: {missingRequired.join(', ')}</p>
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
            {#if found}<span class="check">✓</span>{:else}<span class="x">✗</span>{/if}
          </span>
        {/each}
      </div>
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
            {#if isDangerous}<span class="danger-icon">⚠</span>{/if}
          </span>
        {/each}
      </div>
    </div>
  {/if}

  <!-- Error -->
  {#if p.error}
    <div class="error-box">{p.error}</div>
  {/if}
</div>

<style>
  .plugin-card {
    background: #16213e;
    border: 1px solid #0f3460;
    border-radius: 8px;
    padding: 1rem;
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
    margin-bottom: 0.5rem;
  }

  .plugin-id {
    display: flex;
    align-items: center;
    gap: 0.5rem;
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

  .card-meta {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.3rem;
    margin-bottom: 0.75rem;
    font-size: 0.8rem;
  }

  .meta-row {
    display: flex;
    gap: 0.5rem;
  }

  .label {
    color: #a0a0b8;
    min-width: 80px;
  }

  .path {
    font-family: monospace;
    font-size: 0.75rem;
    color: #a0a0b8;
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
  .x { color: #e94560; margin-left: 2px; }
  .danger-icon { color: #e94560; margin-left: 2px; }

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
</style>
