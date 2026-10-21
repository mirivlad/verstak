<script>
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { executePluginCommand } from '../plugin-host/VerstakPluginAPI.js';
  import { i18n } from '../i18n/index.js';
  import Icon from '../ui/Icon.svelte';

  export let workspaceRootPath = '';
  export let workspaceId = '';
  export let workspaceTitle = '';
  export let availableTools = [];
  export let overviewProviders = [];

  const dispatch = createEventDispatcher();
  let loading = true;
  let activeFilter = 'all';
  let providerResults = [];
  let loadedWorkspaceId = '';
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
  $: hasOverviewSideContent = Boolean(summaryItems.length || keyResources.length || needsAttention.length || (loading && hasAttentionTools));

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed && workspaceId) loadOverview();
    });
  });

  onDestroy(() => unsubscribeLocale?.());

  $: if (workspaceId && (workspaceId !== loadedWorkspaceId || providerKey !== loadedProviderKey)) {
    loadOverview();
  }

  function commandResult(value) {
    if (value && value.status === 'handled') return value.result || {};
    return value && value.result ? value.result : (value || {});
  }

  async function loadOverview() {
    const workspaceAtStart = String(workspaceId || '').trim();
    const providerKeyAtStart = providerKey;
    loadedWorkspaceId = workspaceAtStart;
    loadedProviderKey = providerKeyAtStart;
    loading = true;

    const rows = await Promise.all((overviewProviders || []).map(async provider => {
      if (!provider?.pluginId || !provider?.handler) return null;
      try {
        const response = await executePluginCommand(provider.pluginId, provider.handler, {
          scope: { kind: 'deal', workspaceId: workspaceAtStart },
          workspaceRootPath,
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

    if (workspaceAtStart !== String(workspaceId || '').trim() || providerKeyAtStart !== providerKey) return;
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
      actionLabel: item.action.toolRequest ? tr('overview.openItem') : tr('overview.openTool', { tool: target?.title || item.action.workspaceItemId }),
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
      toolRequest: item.action.toolRequest || null,
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
    const diff = Date.now() - ms;
    if (diff < 0) return absoluteTime(value);
    const minute = 60 * 1000;
    const hour = 60 * minute;
    const day = 24 * hour;
    if (diff < minute) return tr('overview.time.now');
    if (diff < hour) return tr('overview.time.minutes', { count: Math.floor(diff / minute) });
    if (diff < day) return tr('overview.time.hours', { count: Math.floor(diff / hour) });
    if (diff < 2 * day) return tr('overview.time.yesterday');
    if (diff < 7 * day) return tr('overview.time.days', { count: Math.floor(diff / day) });
    const date = new Date(ms);
    return date.toLocaleDateString(locale === 'ru' ? 'ru-RU' : 'en-US', {
      month: 'short',
      day: 'numeric',
      year: date.getFullYear() === new Date().getFullYear() ? undefined : 'numeric',
    });
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

<div class="overview-root" aria-label={tr('workspace.overview')} data-overview-root>
  <div class="overview-header">
    <div>
      <span class="overview-eyebrow">{tr('workspace.overview')}</span>
      <h1>{workspaceTitle || workspaceRootPath}</h1>
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
    <button class="vt-button ghost icon" type="button" aria-label={tr('overview.refresh')} title={tr('overview.refresh')} data-overview-action="refresh" on:click={loadOverview}><Icon name="refresh-cw" size={16} /></button>
  </div>

  <div class="overview-layout" class:overview-layout-wide={!hasOverviewSideContent}>
    <main class="overview-main">
      <section class="overview-continue" data-overview-section="continue">
        <div class="overview-continue-copy">
          <span>{tr('overview.continue')}</span>
          <h3>{tr('overview.continueHint')}</h3>
        </div>
        {#if loading}
          <p class="overview-empty compact">{tr('overview.loadingSignals')}</p>
        {:else if continueItems.length}
          <div class="overview-continue-list">
            {#each continueItems as item}
              <button
                type="button"
                class="overview-continue-item"
                data-overview-continue-item={item.category || item.actionKind || item.id}
                data-overview-action={item.actionKind}
                on:click={() => openTool(item.actionKind, item.toolRequest)}
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

      <section class="overview-panel overview-recent" data-overview-section="recent">
        <div class="overview-panel-head">
          <div>
            <h3>{tr('overview.recentChanges')}</h3>
            <p>{tr('overview.recentChangesHint')}</p>
          </div>
          {#if FILTERS.length > 1}<div class="overview-filters" aria-label={tr('overview.recentFilter')}>
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
          </div>{/if}
        </div>
        {#if loading}
          <p class="overview-empty">{tr('overview.loadingRecent')}</p>
        {:else if filteredRecentChanges.length}
          <div class="overview-list">
            {#each filteredRecentChanges as item}
              <button
                type="button"
                class="overview-row overview-change-row"
                data-overview-recent-item={item.category}
                data-overview-action={item.actionKind}
                on:click={() => openTool(item.actionKind, item.toolRequest)}
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
          <p class="overview-empty">{tr('overview.noChanges')}</p>
        {/if}
      </section>
    </main>

    {#if hasOverviewSideContent}
    <aside class="overview-side">
      {#if summaryItems.length}
      <section class="overview-summary" data-overview-section="summary" aria-label={tr('overview.summary')}>
        <h3>{tr('overview.toolsTitle')}</h3>
        {#each summaryItems as item}
          <button type="button" class="overview-summary-item" data-overview-summary={item.key} data-overview-action={item.actionKind}
            aria-label={`${item.label}: ${item.actionLabel}`} on:click={() => openTool(item.actionKind, item.toolRequest)}>
            <span>{item.label}</span><strong>{loading ? '...' : item.count}</strong>
            <small>{loading ? tr('common.loading') : item.detail}</small>
          </button>
        {/each}
      </section>
      {/if}
      {#if hasAttentionTools && (needsAttention.length || loading)}
        <section class="overview-panel" data-overview-section="attention">
          <div class="overview-panel-head">
            <div>
              <h3>{tr('overview.attention')}</h3>
              <p>{tr('overview.attentionHint')}</p>
            </div>
          </div>
          {#if loading}
            <p class="overview-empty">{tr('overview.loadingPending')}</p>
          {:else if needsAttention.length}
            <div class="overview-list compact">
              {#each needsAttention as item}
                <div class="overview-row overview-attention-row">
                  <strong title={item.title}>{item.title}</strong>
                  <span>{item.meta}</span>
                  <button type="button" on:click={() => openTool(item.actionKind, item.toolRequest)}>{item.actionLabel}</button>
                </div>
              {/each}
            </div>
          {:else}
            <p class="overview-empty compact">{tr('overview.noPending')}</p>
          {/if}
        </section>
      {/if}

      {#if keyResources.length}
        <section class="overview-panel secondary" data-overview-section="key-resources">
          <div class="overview-panel-head">
            <h3>{tr('overview.keyResources')}</h3>
          </div>
          <div class="overview-list compact">
            {#each keyResources as item}
              <div class="overview-row overview-resource-row">
                <strong title={item.title}>{item.title}</strong>
                <span title={item.meta}>{item.meta}</span>
                <button type="button" aria-label={`${item.actionLabel}: ${item.title}`} on:click={() => openTool(item.actionKind, item.toolRequest)}>{item.actionLabel}</button>
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
  .overview-root { height: 100%; min-height: 0; overflow: auto; background: var(--vt-color-background); color: var(--vt-color-text-primary); container-type: inline-size; }
  .overview-header { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: 2.1rem 1.8rem 1.8rem; }
  .overview-eyebrow { color: var(--vt-color-text-muted); font-size: 0.75rem; }
  .overview-header h1 { margin: 0.6rem 0 0.5rem; font-size: 1.85rem; line-height: 1.2; font-weight: 500; letter-spacing: -0.04em; overflow-wrap: anywhere; }
  .overview-header p { color: var(--vt-color-text-muted); font-size: 0.78rem; }
  .overview-layout { display: grid; grid-template-columns: minmax(0, 1fr) 13rem; gap: 2rem; padding: 0 1.8rem 2rem; }
  .overview-layout.overview-layout-wide { grid-template-columns: minmax(0, 1fr); }
  .overview-main, .overview-side { min-width: 0; display: grid; align-content: start; gap: 2rem; }
  .overview-side { border-left: 1px solid var(--vt-color-border); padding-left: 1.4rem; }
  .overview-continue { display: grid; gap: 0.6rem; padding: 1.1rem 1.25rem; border: 1px solid var(--vt-color-border); border-radius: var(--vt-radius-lg); background: var(--vt-color-surface-muted); }
  .overview-continue-copy { min-width: 0; display: grid; gap: 0.35rem; }
  .overview-continue-copy span { color: var(--vt-color-text-muted); font-size: 0.75rem; font-weight: 400; }
  .overview-continue-copy h3 { margin: 0; font-size: 0.95rem; font-weight: 500; }
  .overview-continue-list { display: grid; gap: 0.35rem; }
  .overview-continue-item { display: flex; align-items: center; gap: 1rem; width: 100%; padding: 0.5rem 0; border: 0; background: transparent; color: inherit; text-align: left; font-weight: 400; }
  .overview-continue-item-copy { flex: 1; min-width: 0; display: grid; gap: 0.4rem; }
  .overview-continue-item-copy strong { color: var(--vt-color-text-primary); font-size: 1rem; font-weight: 500; overflow-wrap: anywhere; }
  .overview-continue-item-copy > span { color: var(--vt-color-text-muted); font-size: 0.75rem; overflow-wrap: anywhere; }
  .overview-continue-item-action { color: var(--vt-color-text-secondary); font-size: 0.75rem; }
  .overview-continue-empty { display: grid; gap: 0.4rem; padding: 0.35rem 0; }
  .overview-continue-empty strong { font-size: 0.85rem; font-weight: 500; }
  .overview-continue-empty p { font-size: 0.78rem; color: var(--vt-color-text-muted); }
  .overview-summary { display: grid; gap: 0.2rem; }
  .overview-summary h3 { margin: 0 0 0.7rem; color: var(--vt-color-text-muted); font-size: 0.78rem; font-weight: 500; }
  .overview-summary-item { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 0.2rem 0.7rem; min-width: 0; padding: 0.6rem 0.25rem; border: 0; border-radius: var(--vt-radius-sm); background: transparent; color: var(--vt-color-text-primary); text-align: left; font-weight: 400; }
  .overview-summary-item strong { color: var(--vt-color-text-muted); font-size: 0.75rem; font-weight: 400; }
  .overview-summary-item span { font-size: 0.82rem; }
  .overview-summary-item small { grid-column: 1 / -1; color: var(--vt-color-text-muted); font-size: 0.7rem; overflow-wrap: anywhere; }
  .overview-panel { min-width: 0; display: flex; flex-direction: column; background: transparent; }
  .overview-panel-head { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.75rem; margin-bottom: 1rem; }
  .overview-panel h3 { margin: 0; font-size: 0.9rem; font-weight: 500; }
  .overview-panel-head p { margin-top: 0.3rem; color: var(--vt-color-text-muted); font-size: 0.75rem; }
  .overview-filters { display: inline-flex; flex-wrap: wrap; align-items: center; gap: 0.25rem; }
  .overview-filters button { min-height: 1.7rem; padding: 0.25rem 0.5rem; border: 1px solid transparent; background: transparent; color: var(--vt-color-text-muted); font-size: 0.75rem; font-weight: 400; }
  .overview-filters button.is-active { background: var(--vt-color-accent-muted); color: var(--vt-color-accent); }
  .overview-list { display: grid; gap: 0.3rem; }
  .overview-row { min-width: 0; padding: 0.8rem 0.4rem; border: 0; border-radius: var(--vt-radius-sm); background: transparent; }
  .overview-change-row { display: flex; align-items: center; justify-content: space-between; gap: 1rem; width: 100%; text-align: left; color: var(--vt-color-text-primary); font-weight: 400; }
  .overview-change-copy { flex: 1; min-width: 0; display: grid; gap: 0.35rem; }
  .overview-change-copy strong { font-weight: 500; font-size: 0.85rem; overflow-wrap: anywhere; }
  .overview-change-copy > span { color: var(--vt-color-text-muted); font-size: 0.72rem; overflow-wrap: anywhere; }
  .overview-row-action { flex-shrink: 0; color: var(--vt-color-text-muted); font-size: 0.72rem; }
  .overview-attention-row, .overview-resource-row { display: grid; gap: 0.3rem; padding-inline: 0; }
  .overview-attention-row > strong, .overview-resource-row > strong { font-size: 0.82rem; font-weight: 500; overflow-wrap: anywhere; }
  .overview-attention-row > span, .overview-resource-row > span { color: var(--vt-color-text-muted); font-size: 0.72rem; overflow-wrap: anywhere; }
  .overview-attention-row button, .overview-resource-row button { justify-self: start; min-height: 1.6rem; padding: 0.25rem 0; border: 0; background: transparent; color: var(--vt-color-accent); font-size: 0.75rem; font-weight: 400; }
  .overview-empty { margin: 0; padding: 0.65rem 0; color: var(--vt-color-text-muted); font-size: 0.82rem; line-height: 1.5; }
  @container (max-width: 760px) {
    .overview-layout { grid-template-columns: minmax(0, 1fr); }
    .overview-side { border-left: 0; border-top: 1px solid var(--vt-color-border); padding: 1.5rem 0 0; grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr)); gap: 1.5rem; }
  }
  @container (max-width: 480px) {
    .overview-header { padding: 1.4rem 1rem; }
    .overview-header h1 { font-size: 1.6rem; }
    .overview-layout { padding: 0 1rem 1.5rem; gap: 1.4rem; }
    .overview-continue { padding: 1rem; }
    .overview-continue-item { align-items: flex-start; flex-direction: column; gap: 0.5rem; }
    .overview-row-action { display: none; }
    .overview-side { grid-template-columns: minmax(0, 1fr); }
  }
</style>
