<script>
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { executePluginCommand } from '../plugin-host/VerstakPluginAPI.js';
  import { i18n } from '../i18n/index.js';

  export let workspaceRootPath = '';
  export let availableTools = [];
  export let overviewProviders = [];

  const dispatch = createEventDispatcher();
  let loading = true;
  let activeFilter = 'all';
  let providerResults = [];
  let loadedWorkspaceRoot = '';
  let loadedProviderKey = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;

  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);
  $: toolById = new Map((availableTools || []).filter(tool => tool?.id && !tool?.shell).map(tool => [tool.id, tool]));
  $: providerKey = (overviewProviders || []).map(provider => `${provider?.pluginId || ''}:${provider?.id || ''}:${provider?.handler || ''}`).join('|');
  $: aggregated = aggregateProviderResults(providerResults, toolById);
  $: summaryItems = aggregated.summary;
  $: continueItems = aggregated.resume;
  $: needsAttention = aggregated.attention;
  $: recentChanges = aggregated.recent;
  $: keyResources = aggregated.resources;
  $: lastActive = aggregated.lastActiveAt;
  $: categoryFilters = collectCategoryFilters(recentChanges);
  $: FILTERS = [{ key: 'all', label: tr('overview.filter.all') }, ...categoryFilters];
  $: if (!FILTERS.some(filter => filter.key === activeFilter)) activeFilter = 'all';
  $: filteredRecentChanges = activeFilter === 'all'
    ? recentChanges
    : recentChanges.filter(item => item.category === activeFilter);
  $: hasAttentionTools = (overviewProviders || []).length > 0;
  $: hasOverviewSideContent = Boolean(keyResources.length || needsAttention.length || (loading && hasAttentionTools));

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed && workspaceRootPath) loadOverview();
    });
  });

  onDestroy(() => unsubscribeLocale?.());

  $: if (workspaceRootPath && (workspaceRootPath !== loadedWorkspaceRoot || providerKey !== loadedProviderKey)) {
    loadOverview();
  }

  function commandResult(value) {
    if (value && value.status === 'handled') return value.result || {};
    return value && value.result ? value.result : (value || {});
  }

  async function loadOverview() {
    const workspaceAtStart = String(workspaceRootPath || '').trim();
    const providerKeyAtStart = providerKey;
    loadedWorkspaceRoot = workspaceAtStart;
    loadedProviderKey = providerKeyAtStart;
    loading = true;

    const rows = await Promise.all((overviewProviders || []).map(async provider => {
      if (!provider?.pluginId || !provider?.handler) return null;
      try {
        const response = await executePluginCommand(provider.pluginId, provider.handler, {
          workspaceRootPath: workspaceAtStart,
        });
        return {
          pluginId: provider.pluginId,
          providerId: provider.id || provider.handler,
          result: commandResult(response),
        };
      } catch (error) {
        console.warn(`[Overview] provider ${provider.pluginId}/${provider.id || provider.handler} failed:`, error);
        return null;
      }
    }));

    if (workspaceAtStart !== String(workspaceRootPath || '').trim() || providerKeyAtStart !== providerKey) return;
    providerResults = rows.filter(Boolean);
    loading = false;
  }

  function actionAvailable(action, tools) {
    if (!action?.workspaceItemId) return false;
    return tools.has(action.workspaceItemId);
  }

  function timeValue(item) {
    const value = item?.occurredAt || item?.time || '';
    if (!value) return 0;
    const parsed = new Date(value).getTime();
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function explicitOrder(item) {
    return Number.isFinite(Number(item?.order)) ? Number(item.order) : null;
  }

  function sortByTime(items) {
    return [...items].sort((a, b) => timeValue(b) - timeValue(a) || (a._sequence || 0) - (b._sequence || 0));
  }

  function sortAttention(items) {
    return [...items].sort((a, b) => {
      const ao = explicitOrder(a);
      const bo = explicitOrder(b);
      if (ao !== null || bo !== null) return (ao ?? 10000) - (bo ?? 10000);
      return timeValue(b) - timeValue(a) || (a._sequence || 0) - (b._sequence || 0);
    });
  }

  function normalizeActionItem(item, providerId, sequence, tools) {
    if (!item || !actionAvailable(item.action, tools)) return null;
    const target = tools.get(item.action.workspaceItemId);
    const occurredAt = item.occurredAt || '';
    const metaParts = [];
    if (item.meta) metaParts.push(String(item.meta));
    if (occurredAt) metaParts.push(relativeTime(occurredAt));
    return {
      ...item,
      id: `${providerId}:${item.id || sequence}`,
      meta: metaParts.join(' · '),
      absolute: occurredAt ? absoluteTime(occurredAt) : '',
      actionKind: item.action.workspaceItemId,
      actionLabel: tr('overview.openTool', { tool: target?.title || item.action.workspaceItemId }),
      toolRequest: item.action.toolRequest || null,
      _sequence: sequence,
    };
  }

  function normalizeSummaryItem(item, providerId, sequence, tools) {
    if (!item || !actionAvailable(item.action, tools)) return null;
    const target = tools.get(item.action.workspaceItemId);
    return {
      ...item,
      key: item.id || `${providerId}:${sequence}`,
      count: Number.isFinite(Number(item.count)) ? Number(item.count) : 0,
      actionKind: item.action.workspaceItemId,
      actionLabel: tr('overview.openTool', { tool: target?.title || item.action.workspaceItemId }),
      _sequence: sequence,
    };
  }

  function aggregateProviderResults(rows, tools) {
    const summary = [];
    const resume = [];
    const attention = [];
    const recent = [];
    const resources = [];
    const lastActiveCandidates = [];
    let sequence = 0;

    (rows || []).forEach(row => {
      const result = row?.result && typeof row.result === 'object' ? row.result : {};
      const providerId = `${row?.pluginId || 'provider'}:${row?.providerId || ''}`;
      (result.summary || []).forEach(item => {
        const normalized = normalizeSummaryItem(item, providerId, sequence++, tools);
        if (normalized) summary.push(normalized);
      });
      (result.resume || []).forEach(item => {
        const normalized = normalizeActionItem(item, providerId, sequence++, tools);
        if (normalized) resume.push(normalized);
      });
      (result.attention || []).forEach(item => {
        const normalized = normalizeActionItem(item, providerId, sequence++, tools);
        if (normalized) attention.push(normalized);
      });
      (result.recent || []).forEach(item => {
        const normalized = normalizeActionItem(item, providerId, sequence++, tools);
        if (normalized) {
          normalized.category = String(item.categoryId || 'other');
          normalized.categoryLabel = String(item.categoryLabel || tools.get(item.action.workspaceItemId)?.title || normalized.category);
          recent.push(normalized);
        }
      });
      (result.resources || []).forEach(item => {
        const normalized = normalizeActionItem(item, providerId, sequence++, tools);
        if (normalized) resources.push(normalized);
      });
      if (result.lastActiveAt) lastActiveCandidates.push({ occurredAt: result.lastActiveAt });
    });

    summary.sort((a, b) => (explicitOrder(a) ?? 10000) - (explicitOrder(b) ?? 10000) || String(a.label || '').localeCompare(String(b.label || '')));
    const visibleItems = [...resume, ...attention, ...recent, ...resources];
    visibleItems.forEach(item => {
      if (item.occurredAt) lastActiveCandidates.push(item);
    });
    return {
      summary,
      resume: sortByTime(resume).slice(0, 4),
      attention: sortAttention(attention).slice(0, 6),
      recent: sortByTime(recent).slice(0, 12),
      resources: resources.sort((a, b) => (explicitOrder(a) ?? 10000) - (explicitOrder(b) ?? 10000) || (a._sequence || 0) - (b._sequence || 0)),
      lastActiveAt: sortByTime(lastActiveCandidates)[0]?.occurredAt || '',
    };
  }

  function collectCategoryFilters(items) {
    const categories = [];
    const seen = new Set();
    (items || []).forEach(item => {
      const key = String(item?.category || '').trim();
      if (!key || seen.has(key)) return;
      seen.add(key);
      categories.push({ key, label: item.categoryLabel || key });
    });
    return categories;
  }

  function relativeTime(value) {
    const ms = new Date(value).getTime();
    if (!Number.isFinite(ms)) return tr('overview.time.none');
    const delta = Date.now() - ms;
    const abs = Math.abs(delta);
    if (abs < 60 * 1000) return tr('overview.time.now');
    const minutes = Math.max(1, Math.round(abs / 60000));
    if (minutes < 60) return tr(delta >= 0 ? 'overview.time.minutesAgo' : 'overview.time.inMinutes', { count: minutes });
    const hours = Math.max(1, Math.round(abs / 3600000));
    if (hours < 48) return tr(delta >= 0 ? 'overview.time.hoursAgo' : 'overview.time.inHours', { count: hours });
    const days = Math.max(1, Math.round(abs / 86400000));
    return tr(delta >= 0 ? 'overview.time.daysAgo' : 'overview.time.inDays', { count: days });
  }

  function absoluteTime(value) {
    const date = new Date(value);
    if (!Number.isFinite(date.getTime())) return '';
    try {
      return new Intl.DateTimeFormat(locale === 'ru' ? 'ru-RU' : 'en-US', {
        dateStyle: 'medium',
        timeStyle: 'short',
      }).format(date);
    } catch (_) {
      return date.toISOString();
    }
  }

  function openTool(workspaceItemId, toolRequest = null) {
    dispatch('openTool', { workspaceItemId, toolRequest });
  }
</script>

<div class="today-root overview-root" aria-label={tr('workspace.overview')} data-overview-root>
  <div class="today-header overview-header">
    <div>
      <h2>{tr('workspace.overview')}</h2>
      <p title={lastActive ? absoluteTime(lastActive) : ''}>
        {#if loading}
          {tr('overview.loadingContext')}
        {:else if lastActive}
          {tr('overview.lastActive', { time: relativeTime(lastActive) })}
        {:else}
          {tr('overview.noRecentActivity')}
        {/if}
      </p>
    </div>
    <button type="button" data-overview-action="refresh" on:click={loadOverview}>{tr('overview.refresh')}</button>
  </div>

  <div class="overview-layout" class:overview-layout-wide={!hasOverviewSideContent}>
    <main class="overview-main">
      <section class="today-resume overview-continue" data-overview-section="continue">
        <div class="today-resume-copy overview-continue-copy">
          <span>{tr('overview.continue')}</span>
          <h3>{tr('overview.continueHint')}</h3>
        </div>
        {#if loading}
          <p class="today-empty compact">{tr('overview.loadingSignals')}</p>
        {:else if continueItems.length}
          <div class="overview-continue-list">
            {#each continueItems as item}
              <button
                type="button"
                class="overview-continue-item"
                data-overview-continue-item={item.category}
                data-overview-action={item.actionKind}
                on:click={() => openTool(item.actionKind)}
              >
                <span class="overview-continue-item-copy">
                  <strong title={item.title}>{item.title}</strong>
                  <span title={item.absolute}>{item.meta}</span>
                </span>
                <span class="overview-continue-item-action">{item.actionLabel}</span>
              </button>
            {/each}
          </div>
        {:else}
          <div class="overview-continue-empty">
            <strong>{tr('overview.noResume')}</strong>
            <p>{tr('overview.noResumeHint')}</p>
          </div>
        {/if}
      </section>

      <div class="today-summary overview-summary" data-overview-section="summary" aria-label={tr('overview.summary')}>
        {#each summaryItems as item}
          <button
            type="button"
            class="today-summary-item overview-summary-item"
            data-overview-summary={item.key}
            data-overview-action={item.actionKind}
            aria-label={`${item.label}: ${item.actionLabel}`}
            on:click={() => openTool(item.actionKind)}
          >
            <strong>{loading ? '...' : item.count}</strong>
            <span>{item.label}</span>
            <small>{loading ? tr('common.loading') : item.detail}</small>
          </button>
        {/each}
      </div>

      <section class="today-panel overview-panel overview-recent" data-overview-section="recent">
        <div class="today-panel-head overview-panel-head">
          <div>
            <h3>{tr('overview.recentChanges')}</h3>
            <p>{tr('overview.recentChangesHint')}</p>
          </div>
          <div class="overview-filters" aria-label={tr('overview.recentFilter')}>
            {#each FILTERS as filter}
              <button
                type="button"
                class:is-active={activeFilter === filter.key}
                aria-pressed={activeFilter === filter.key}
                data-overview-filter={filter.key}
                on:click={() => activeFilter = filter.key}
              >
                {filter.label}
              </button>
            {/each}
          </div>
        </div>
        {#if loading}
          <p class="today-empty">{tr('overview.loadingRecent')}</p>
        {:else if filteredRecentChanges.length}
          <div class="today-list overview-list">
            {#each filteredRecentChanges as item}
              <button
                type="button"
                class="today-row overview-change-row"
                data-overview-recent-item={item.category}
                data-overview-action={item.actionKind}
                on:click={() => openTool(item.actionKind)}
              >
                <span class="overview-change-copy">
                  <strong title={item.title}>{item.title}</strong>
                  <span title={item.absolute}>{item.meta}</span>
                </span>
                <span class="overview-row-action">{item.actionLabel}</span>
              </button>
            {/each}
          </div>
        {:else}
          <p class="today-empty">{tr('overview.noChanges')}</p>
        {/if}
      </section>
    </main>

    {#if hasOverviewSideContent}
    <aside class="overview-side">
      {#if hasAttentionTools && (needsAttention.length || loading)}
        <section class="today-panel overview-panel" data-overview-section="attention">
          <div class="today-panel-head overview-panel-head">
            <div>
              <h3>{tr('overview.attention')}</h3>
              <p>{tr('overview.attentionHint')}</p>
            </div>
          </div>
          {#if loading}
            <p class="today-empty">{tr('overview.loadingPending')}</p>
          {:else if needsAttention.length}
            <div class="today-list overview-list compact">
              {#each needsAttention as item}
                <div class="today-row overview-attention-row">
                  <strong title={item.title}>{item.title}</strong>
                  <span>{item.meta}</span>
                  <button type="button" on:click={() => openTool(item.actionKind, item.toolRequest)}>{item.actionLabel}</button>
                </div>
              {/each}
            </div>
          {:else}
            <p class="today-empty compact">{tr('overview.noPending')}</p>
          {/if}
        </section>
      {/if}

      {#if keyResources.length}
        <section class="today-panel overview-panel secondary" data-overview-section="key-resources">
          <div class="today-panel-head overview-panel-head">
            <h3>{tr('overview.keyResources')}</h3>
          </div>
          <div class="today-list overview-list compact">
            {#each keyResources as item}
              <div class="today-row overview-resource-row">
                <strong title={item.title}>{item.title}</strong>
                <span title={item.meta}>{item.meta}</span>
                <button type="button" on:click={() => openTool(item.actionKind)}>{item.actionLabel}</button>
              </div>
            {/each}
          </div>
        </section>
      {/if}
    </aside>
    {/if}
  </div>
</div>

<style>
  .today-root {
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--vt-color-background);
    color: var(--vt-color-text-primary);
    overflow: auto;
  }

  .today-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem;
    border-bottom: 1px solid var(--vt-color-border);
    background: var(--vt-color-surface-muted);
  }

  .today-header h2 {
    margin: 0;
    font-size: 1.05rem;
  }

  .today-header p {
    margin: 0.25rem 0 0;
    color: var(--vt-color-text-muted);
    font-size: 0.8rem;
  }

  .today-summary {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
    gap: 0.4rem;
  }

  .today-summary-item {
    min-width: 0;
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    grid-template-rows: auto auto;
    align-items: center;
    column-gap: 0.55rem;
    row-gap: 0.12rem;
    padding: 0.5rem 0.65rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
    color: inherit;
    cursor: pointer;
    font: inherit;
    text-align: left;
  }

  .today-summary-item:hover,
  .today-summary-item:focus-visible {
    border-color: var(--vt-color-accent);
    background: var(--vt-color-surface-hover);
    outline: none;
  }

  .today-summary-item strong {
    grid-row: 1 / 3;
    color: var(--vt-color-text-primary);
    font-size: 1.05rem;
    line-height: 1;
  }

  .today-summary-item span,
  .today-summary-item small {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-muted);
    font-size: 0.72rem;
  }

  .today-summary-item span {
    color: var(--vt-color-text-secondary);
    font-weight: 600;
  }

  .overview-layout {
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(17rem, 23rem);
    gap: 0.75rem;
    padding: 0.75rem;
  }

  .overview-layout.overview-layout-wide {
    grid-template-columns: minmax(0, 1fr);
  }

  .overview-main,
  .overview-side {
    min-width: 0;
    display: grid;
    align-content: start;
    gap: 0.75rem;
  }

  .today-resume {
    display: grid;
    gap: 0.75rem;
    padding: 0.9rem 1rem;
    border: 1px solid rgba(78, 204, 163, 0.24);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-accent-muted);
  }

  .today-resume-copy {
    min-width: 0;
    display: grid;
    gap: 0.22rem;
  }

  .today-resume-copy span {
    color: var(--vt-color-accent);
    font-size: 0.72rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .today-resume-copy h3 {
    margin: 0;
    color: var(--vt-color-text-primary);
    font-size: 0.98rem;
    font-weight: 600;
  }

  .today-panel {
    min-width: 0;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
  }

  .overview-recent {
    min-height: 24rem;
  }

  .overview-panel.secondary {
    background: var(--vt-color-surface-muted);
  }

  [data-overview-section='attention'] {
    border-color: rgba(255, 200, 87, 0.45);
    background: var(--vt-color-warning-muted);
  }

  .today-panel-head {
    min-height: 2.8rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.65rem 0.75rem;
    border-bottom: 1px solid var(--vt-color-border);
  }

  .today-panel h3 {
    margin: 0;
    color: var(--vt-color-text-primary);
    font-size: 0.9rem;
  }

  .today-panel-head p {
    margin: 0.2rem 0 0;
    color: var(--vt-color-text-muted);
    font-size: 0.74rem;
  }

  .overview-filters {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.2rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-background);
  }

  .overview-filters button,
  .today-panel-head button,
  .today-header button,
  .overview-continue-item,
  .overview-list button {
    min-height: 1.85rem;
    padding: 0.3rem 0.65rem;
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface-hover);
    color: var(--vt-color-text-secondary);
    font-size: 0.76rem;
    cursor: pointer;
  }

  .overview-filters button {
    min-height: 1.55rem;
    padding: 0.16rem 0.5rem;
    border-color: transparent;
    background: transparent;
  }

  .overview-filters button.is-active,
  .overview-filters button:hover,
  .today-panel-head button:hover,
  .today-header button:hover,
  .overview-continue-item:hover,
  .overview-list button:hover {
    border-color: var(--vt-color-accent);
    color: var(--vt-color-text-primary);
  }

  .overview-filters button.is-active {
    background: var(--vt-color-accent-muted);
    color: var(--vt-color-accent);
  }

  .today-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0;
    padding: 1rem;
    color: var(--vt-color-text-muted);
    font-size: 0.82rem;
    line-height: 1.45;
    text-align: center;
  }

  .today-empty.compact {
    min-height: 5rem;
  }

  .today-list {
    display: grid;
    gap: 0.45rem;
    padding: 0.65rem;
  }

  .today-list.compact {
    gap: 0.4rem;
  }

  .today-row {
    min-width: 0;
    display: grid;
    gap: 0.2rem;
    padding: 0.55rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface-muted);
    color: inherit;
    font: inherit;
    text-align: left;
  }

  .overview-change-row {
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    padding: 0.55rem 0.75rem;
    border: 0;
    border-bottom: 1px solid var(--vt-color-border);
    border-radius: 0;
    background: transparent;
    cursor: pointer;
  }

  .overview-change-row:hover,
  .overview-change-row:focus-visible {
    background: var(--vt-color-surface-hover);
    outline: none;
  }

  .overview-change-copy,
  .overview-continue-item-copy {
    min-width: 0;
    display: grid;
    gap: 0.2rem;
  }

  .overview-row-action,
  .overview-continue-item-action {
    color: var(--vt-color-text-muted);
    font-size: 0.72rem;
    white-space: nowrap;
  }

  .overview-recent .overview-list {
    gap: 0;
    padding: 0;
  }

  .overview-recent .overview-change-row:last-child {
    border-bottom: 0;
  }

  .overview-attention-row,
  .overview-resource-row {
    grid-template-columns: minmax(0, 1fr);
  }

  .today-row strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-primary);
    font-size: 0.85rem;
  }

  .today-row span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-muted);
    font-size: 0.75rem;
  }

  .overview-continue-list {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.5rem;
  }

  .overview-continue-item {
    min-width: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.6rem;
    text-align: left;
  }

  .overview-continue-item strong,
  .overview-continue-item-copy span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .overview-continue-item strong {
    color: var(--vt-color-text-primary);
    font-size: 0.8rem;
  }

  .overview-continue-item-copy span {
    color: var(--vt-color-text-muted);
    font-size: 0.72rem;
  }

  .overview-continue-empty {
    display: grid;
    gap: 0.22rem;
    color: var(--vt-color-text-secondary);
  }

  .overview-continue-empty p {
    margin: 0;
    color: var(--vt-color-text-muted);
    font-size: 0.8rem;
  }

  @container vt-content (max-width: 980px) {
    .overview-layout {
      grid-template-columns: 1fr;
    }

    .today-panel-head {
      align-items: stretch;
      flex-direction: column;
    }

    .overview-filters {
      overflow-x: auto;
      justify-content: flex-start;
    }

    .overview-continue-list {
      grid-template-columns: 1fr;
    }

    .overview-continue-item {
      width: 100%;
    }

    .overview-change-row {
      grid-template-columns: 1fr;
      align-items: stretch;
    }
  }

  @container vt-content (max-width: 620px) {
    .today-summary {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .overview-change-row {
      grid-template-columns: 1fr;
      gap: 0.3rem;
    }
  }
</style>
