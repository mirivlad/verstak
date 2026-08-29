<script>
  import { onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import { executePluginCommand } from '../plugin-host/VerstakPluginAPI.js';
  import Icon from '../ui/Icon.svelte';
  import { i18n } from '../i18n/index.js';

  const RESULT_LIMIT = 8;
  const RU = 'ёйцукенгшщзхъфывапролджэячсмитьбю';
  const EN = '`qwertyuiop[]asdfghjkl;\\zxcvbnm,.';

  let query = '';
  let shellIndex = [];
  let searchProviders = [];
  let results = [];
  let focused = false;
  let loading = true;
  let searching = false;
  let contentReady = false;
  let partial = false;
  let revision = 0;
  let searchTimer = null;
  let contextSeq = 0;
  let searchSeq = 0;
  let locale = i18n.getLocale();

  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);

  $: scheduleSearch(query);

  onMount(() => {
    const unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed) refreshContext();
    });
    const refreshSignals = ['verstak:vault-opened', 'verstak:files-changed', 'verstak:workspace-tree-changed', 'verstak:workspace-created', 'verstak:workspace-renamed', 'verstak:workspace-trashed', 'verstak:workspace-restored'];
    const refresh = () => refreshContext();
    refreshSignals.forEach(name => window.addEventListener(name, refresh));
    refreshContext();
    return () => {
      unsubscribeLocale();
      clearTimeout(searchTimer);
      searchSeq += 1;
      refreshSignals.forEach(name => window.removeEventListener(name, refresh));
    };
  });

  function normalize(value) {
    return String(value == null ? '' : value).trim().toLowerCase();
  }

  function swapLayout(value, from, to) {
    return String(value || '').split('').map(ch => {
      const lower = ch.toLowerCase();
      const idx = from.indexOf(lower);
      if (idx === -1) return ch;
      const mapped = to[idx] || ch;
      return ch === lower ? mapped : mapped.toUpperCase();
    }).join('');
  }

  function queryVariants(value) {
    const base = normalize(value);
    return [...new Set([
      base,
      normalize(swapLayout(base, RU, EN)),
      normalize(swapLayout(base, EN, RU)),
    ].filter(Boolean))];
  }

  function matchScore(item, variants) {
    const haystack = normalize(`${item.title} ${item.subtitle || ''} ${item.keywords || ''}`);
    for (const variant of variants) {
      if (!variant) continue;
      if (normalize(item.title) === variant) return 100;
      if (normalize(item.title).startsWith(variant)) return 80;
      if (haystack.includes(variant)) return 50;
    }
    return 0;
  }

  function scheduleSearch(value) {
    clearTimeout(searchTimer);
    if (!normalize(value)) {
      searchSeq += 1;
      searching = false;
      partial = false;
      results = [];
      return;
    }
    searchTimer = setTimeout(() => runSearch(value), 80);
  }

  function workspaceTitle(node) {
    return node?.title || node?.name || node?.id || node?.path || node?.rootPath || '';
  }

  function workspaceName(node) {
    return node?.path || node?.rootPath || node?.name || node?.id || '';
  }

  function collectWorkspaceNodes(nodes, output = []) {
    (nodes || []).forEach(node => {
      if (node?.kind === 'workspace') output.push(node);
      if (Array.isArray(node?.children) && node.children.length) collectWorkspaceNodes(node.children, output);
    });
    return output;
  }

  async function resultOrEmpty(promise, fallback) {
    try {
      const response = await promise;
      if (Array.isArray(response) && response.length === 2 && (typeof response[1] === 'string' || response[1] == null)) {
        return response[1] ? fallback : response[0];
      }
      return response || fallback;
    } catch (_) {
      return fallback;
    }
  }

  function commandResult(value) {
    if (value && value.status === 'handled') return value.result || {};
    return value && value.result ? value.result : (value || {});
  }

  async function refreshContext() {
    const seq = ++contextSeq;
    loading = true;
    contentReady = false;
    const next = [];

    const tree = await resultOrEmpty(App.GetWorkspaceTreeV2(), { roots: [] });
    const nodes = collectWorkspaceNodes(Array.isArray(tree.roots) ? tree.roots : []);
    nodes.forEach(node => {
      const workspaceRootPath = workspaceName(node);
      const workspaceId = String(node.workspaceId || '').trim();
      if (!workspaceId) return;
      next.push({
        type: 'Workspace',
        typeLabel: tr('search.type.workspace'),
        title: workspaceTitle(node),
        subtitle: tr('search.type.workspace'),
        keywords: `${node.id || ''} ${node.path || node.rootPath || ''}`,
        rank: 10,
        action: { kind: 'workspace', workspaceId, workspaceRootPath },
      });
    });

    const [rawPlugins, rawContributions] = await Promise.all([
      resultOrEmpty(App.GetPlugins(), []),
      resultOrEmpty(App.GetContributions(), {}),
    ]);
    await Promise.all((rawPlugins || []).map((plugin) => i18n.loadPlugin(
      plugin.manifest?.id,
      plugin.manifest?.localization,
    ).catch(() => {})));
    if (seq !== contextSeq) return;

    const contributions = i18n.localizeContributionSummary(rawContributions || {});
    const enabledPluginIds = new Set((rawPlugins || [])
      .filter(plugin => plugin?.enabled && (plugin?.status === 'loaded' || plugin?.status === 'degraded'))
      .map(plugin => plugin?.manifest?.id)
      .filter(Boolean));
    (contributions.sidebarItems || []).filter(item => enabledPluginIds.has(item?.pluginId)).forEach(item => {
      next.push({
        type: 'Tool',
        typeLabel: tr('search.type.tool'),
        title: item.title || item.id,
        subtitle: item.pluginId || '',
        keywords: `${item.id || ''} ${item.view || ''}`,
        rank: 20,
        action: { kind: 'view', viewId: item.view || item.id, pluginId: item.pluginId },
      });
    });

    shellIndex = next;
    searchProviders = (contributions.searchProviders || []).filter(provider => enabledPluginIds.has(provider?.pluginId) && provider?.handler);
    loading = false;
    contentReady = true;
    revision += 1;
    if (query) runSearch(query);
  }

  function providerType(provider, item) {
    const categoryId = String(item?.categoryId || provider?.id || 'provider');
    const label = String(item?.categoryLabel || provider?.label || categoryId || tr('search.type.tool'));
    return { type: label, categoryId, label };
  }

  function resultActionKey(item) {
    const action = item?.action;
    if (!action) return '';
    if (action.kind === 'resource') return `resource:${action.resource?.kind || ''}:${action.resource?.path || ''}`;
    if (action.kind === 'workspace') return `workspace:${action.workspaceId || ''}`;
    if (action.kind === 'workspace-item') return `workspace-item:${action.workspaceId || ''}:${action.workspaceItemId || ''}:${JSON.stringify(action.toolRequest || {})}`;
    if (action.kind === 'view') return `view:${action.pluginId || ''}:${action.viewId || ''}`;
    return '';
  }

  function normalizeProviderItem(provider, item, providerRank) {
    if (!item || typeof item !== 'object' || !item.action) return null;
    const type = providerType(provider, item);
    const score = Number(item.score);
    const actionPath = item.action?.kind === 'resource' ? String(item.action.resource?.path || '') : '';
    return {
      type: type.type,
      categoryId: type.categoryId,
      typeLabel: type.label,
      title: String(item.title || item.subtitle || provider.label || provider.id || ''),
      subtitle: String(item.subtitle || actionPath || ''),
      rank: 30 + providerRank,
      score: Number.isFinite(score) ? score : 50,
      action: item.action,
      path: actionPath,
      pluginId: provider.pluginId,
      providerId: provider.id || provider.handler,
      providerResultId: item.id || resultActionKey(item) || String(item.title || ''),
    };
  }

  function queryProviderCalls(variants) {
    const calls = [];
    (searchProviders || []).forEach((provider, providerRank) => {
      variants.forEach((variant) => {
        calls.push((async () => {
          try {
            const response = await executePluginCommand(provider.pluginId, provider.handler, {
              query: variant,
              limit: RESULT_LIMIT,
            });
            const value = commandResult(response);
            const list = Array.isArray(value) ? value : (Array.isArray(value?.results) ? value.results : []);
            return {
              rows: list.map(item => normalizeProviderItem(provider, item, providerRank)).filter(Boolean),
              partial: Boolean(value?.partial),
            };
          } catch (error) {
            console.warn(`[GlobalSearch] provider ${provider.pluginId}/${provider.id || provider.handler} failed:`, error);
            return { rows: [], partial: true };
          }
        })());
      });
    });
    return calls;
  }

  function dedupeRows(rows) {
    const byKey = new Map();
    rows.forEach((row, index) => {
      const item = row.item;
      const actionKey = resultActionKey(item);
      const key = actionKey || `${item.pluginId || 'shell'}:${item.providerId || ''}:${item.providerResultId || item.title}:${item.subtitle || ''}`;
      const previous = byKey.get(key);
      if (!previous || row.score > previous.score || (row.score === previous.score && index < previous.index)) {
        byKey.set(key, { ...row, index });
      }
    });
    return Array.from(byKey.values());
  }

  function publishSearchResults(seq, shellRows, providerRows, providerPartial, pending) {
    if (seq !== searchSeq) return;
    const combined = dedupeRows(shellRows.concat(providerRows))
      .sort((a, b) => b.score - a.score || a.item.rank - b.item.rank || a.item.title.localeCompare(b.item.title));
    partial = providerPartial || pending > 0 || combined.length > RESULT_LIMIT;
    results = combined.slice(0, RESULT_LIMIT).map(row => row.item);
    searching = pending > 0;
    revision += 1;
  }

  function runSearch(value) {
    const variants = queryVariants(value);
    const seq = ++searchSeq;
    if (!variants.length) {
      searching = false;
      partial = false;
      results = [];
      return;
    }

    const shellRows = shellIndex
      .map(item => ({ item, score: matchScore(item, variants) }))
      .filter(row => row.score > 0);
    const providerRows = [];
    let providerPartial = false;
    const calls = queryProviderCalls(variants);
    let pending = calls.length;

    publishSearchResults(seq, shellRows, providerRows, providerPartial, pending);
    calls.forEach((call) => {
      call.then((batch) => {
        if (seq !== searchSeq) return;
        pending -= 1;
        providerPartial = providerPartial || batch.partial;
        providerRows.push(...batch.rows.map(item => ({ item, score: item.score })));
        publishSearchResults(seq, shellRows, providerRows, providerPartial, pending);
      });
    });
  }

  function handleFocus() {
    focused = true;
    refreshContext();
  }

  async function openResult(item) {
    query = '';
    results = [];
    const action = item?.action;
    if (!action) return;

    if (action.kind === 'workspace') {
      const workspaceId = String(action.workspaceId || '');
      const workspaceRootPath = String(action.workspaceRootPath || '');
      if (!workspaceId && !workspaceRootPath) return;
      window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
        detail: workspaceId ? { workspaceId } : { workspaceName: workspaceRootPath, workspaceRootPath }
      }));
      return;
    }

    if (action.kind === 'workspace-item') {
      const workspaceId = String(action.workspaceId || '');
      const workspaceRootPath = String(action.workspaceRootPath || '');
      if ((!workspaceId && !workspaceRootPath) || !action.workspaceItemId) return;
      window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
        detail: workspaceId
          ? { workspaceId, workspaceItemId: action.workspaceItemId, toolRequest: action.toolRequest || null }
          : { workspaceName: workspaceRootPath, workspaceRootPath, workspaceItemId: action.workspaceItemId, toolRequest: action.toolRequest || null }
      }));
      return;
    }

    if (action.kind === 'view') {
      window.dispatchEvent(new CustomEvent('verstak:open-view', {
        detail: { viewId: action.viewId || '', pluginId: action.pluginId || item.pluginId || '' }
      }));
      return;
    }

    if (action.kind === 'resource' && action.resource) {
      const request = {
        ...action.resource,
        context: { ...(action.resource.context || {}), sourceView: 'global-search' },
      };
      const response = await App.OpenWorkbenchResource(item.pluginId || '', request);
      const [result, err] = Array.isArray(response) ? response : [response, ''];
      if (!err && result) {
        window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
      }
    }
  }
</script>

<div class="global-search" class:open={focused && (query || results.length)} data-index-revision={revision} data-index-building={loading || searching} data-index-partial={partial}>
  <div class="global-search-box">
    <Icon name="search" size={14} class="global-search-icon" />
    <input
      bind:value={query}
      on:focus={handleFocus}
      on:blur={() => setTimeout(() => focused = false, 120)}
      type="search"
      placeholder={loading ? tr('search.indexing') : tr('search.placeholder')}
      aria-label={tr('search.global')}
      data-global-search-input
    />
  </div>
  {#if focused && query}
    <div class="global-search-results" data-global-search-results>
      {#if results.length}
        {#each results as item}
          <button
            type="button"
            class="global-search-result"
            data-global-search-result-type={item.type}
            data-global-search-result-category={item.categoryId || ''}
            data-global-search-result-path={item.path || ''}
            on:mousedown|preventDefault={() => openResult(item)}
          >
            <span class="global-search-result-title">{item.title}</span>
            <span class="global-search-result-meta">{item.typeLabel || item.type} · {item.subtitle}</span>
          </button>
        {/each}
      {:else if loading || searching || !contentReady}
        <div class="global-search-empty vt-empty-title">{tr('search.indexing')}</div>
      {:else if partial}
        <div class="global-search-empty vt-empty-title">{tr('search.partial')}</div>
      {:else}
        <div class="global-search-empty vt-empty-title">{tr('search.noResults')}</div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .global-search {
    position: relative;
    flex-shrink: 0;
  }

  .global-search-box {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    height: 2rem;
    padding: 0 0.55rem;
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-md);
    background: #0f1424;
    color: var(--vt-color-text-muted);
  }

  :global(.global-search-icon) {
    color: var(--vt-color-text-muted);
    flex-shrink: 0;
  }

  .global-search input {
    width: 100%;
    min-width: 0;
    border: 0;
    outline: 0;
    background: transparent;
    color: var(--vt-color-text-primary);
    font: inherit;
    font-size: 0.78rem;
  }

  .global-search input::placeholder {
    color: var(--vt-color-text-muted);
  }

  .global-search-box:focus-within {
    border-color: var(--vt-color-accent);
    box-shadow: var(--vt-focus-ring);
  }

  .global-search-results {
    position: absolute;
    left: auto;
    right: 0;
    top: calc(100% - 0.25rem);
    z-index: 400;
    width: min(36rem, calc(100vw - 2rem));
    min-height: min(15rem, calc(100vh - 5rem));
    max-height: min(28rem, calc(100vh - 5rem));
    overflow: auto;
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface);
    box-shadow: var(--vt-elevation-menu);
  }

  .global-search-result {
    display: flex;
    flex-direction: column;
    gap: 0.12rem;
    width: 100%;
    padding: 0.55rem 0.65rem;
    border: 0;
    border-bottom: 1px solid rgba(40, 70, 111, 0.55);
    background: transparent;
    color: var(--vt-color-text-primary);
    text-align: left;
    cursor: pointer;
  }

  .global-search-result:hover {
    background: var(--vt-color-surface-hover);
  }

  .global-search-result-title {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 0.8rem;
  }

  .global-search-result-meta,
  .global-search-empty {
    color: var(--vt-color-text-muted);
    font-size: 0.7rem;
  }

  .global-search-empty {
    padding: 0.7rem;
  }
</style>
