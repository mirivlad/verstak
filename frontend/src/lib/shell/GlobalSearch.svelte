<script>
  import { onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import { i18n } from '../i18n/index.js';

  const TEXT_EXTENSIONS = new Set(['txt', 'md', 'markdown', 'log', 'json', 'csv', 'yaml', 'yml', 'toml']);
  const FILE_INDEX_LIMIT = 220;
  const RESULT_LIMIT = 8;
  const RU = 'ёйцукенгшщзхъфывапролджэячсмитьбю';
  const EN = '`qwertyuiop[]asdfghjkl;\\zxcvbnm,.';

  let query = '';
  let index = [];
  let results = [];
  let focused = false;
  let loading = true;
  let searchTimer = null;
  let buildSeq = 0;
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
      if (changed) buildIndex();
    });
    buildIndex();
    return () => {
      unsubscribeLocale();
      clearTimeout(searchTimer);
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
    searchTimer = setTimeout(() => runSearch(value), 80);
  }

  function runSearch(value) {
    const variants = queryVariants(value);
    if (!variants.length) {
      results = [];
      return;
    }
    results = index
      .map(item => ({ item, score: matchScore(item, variants) }))
      .filter(row => row.score > 0)
      .sort((a, b) => b.score - a.score || a.item.rank - b.item.rank || a.item.title.localeCompare(b.item.title))
      .slice(0, RESULT_LIMIT)
      .map(row => row.item);
  }

  function workspaceTitle(node) {
    return node?.title || node?.name || node?.id || node?.rootPath || '';
  }

  function workspaceName(node) {
    return node?.name || node?.id || node?.rootPath || '';
  }

  async function resultOrEmpty(promise, fallback) {
    try {
      const response = await promise;
      if (Array.isArray(response) && response.length === 2) return response[1] ? fallback : response[0];
      return response || fallback;
    } catch (_) {
      return fallback;
    }
  }

  async function listFilesRecursive(dir = '', depth = 0, acc = []) {
    if (acc.length >= FILE_INDEX_LIMIT || depth > 5) return acc;
    const entries = await resultOrEmpty(App.ListVaultFiles('verstak.search', dir), []);
    for (const entry of entries || []) {
      if (acc.length >= FILE_INDEX_LIMIT) break;
      const path = entry.relativePath || entry.path || entry.name || '';
      if (!path) continue;
      acc.push(entry);
      if (entry.type === 'folder') await listFilesRecursive(path, depth + 1, acc);
    }
    return acc;
  }

  async function readFileSnippet(path) {
    const ext = String(path).split('.').pop().toLowerCase();
    if (!TEXT_EXTENSIONS.has(ext)) return '';
    const text = await resultOrEmpty(App.ReadVaultTextFile('verstak.search', path), '');
    return String(text || '').slice(0, 900);
  }

  function pluginToolKind(pluginId, label) {
    if (pluginId === 'verstak.browser-inbox') return 'browser-inbox';
    if (pluginId === 'verstak.activity') return 'activity';
    if (pluginId === 'verstak.journal') return 'journal';
    return String(label || pluginId || '').toLowerCase();
  }

  async function indexPluginSettings(pluginId, label, rank, view, nodes) {
    const settings = await resultOrEmpty(App.ReadPluginSettings(pluginId), {});
    const items = [];
    Object.keys(settings || {}).forEach(key => {
      const value = settings[key];
      const rows = Array.isArray(value) ? value : [];
      rows.forEach(row => {
        if (!row || typeof row !== 'object') return;
        const title = row.title || row.summary || row.url || row.captureId || row.activityId || row.entryId || label;
        const workspaceName = row.workspaceRootPath || row.workspaceName || '';
        items.push({
          type: label,
          title,
          subtitle: row.url || row.summary || row.workspaceRootPath || key,
          keywords: JSON.stringify(row),
          rank,
          action: workspaceName ? 'workspace-tool' : (view ? 'view' : ''),
          viewId: view?.id || '',
          pluginId,
          workspaceName,
          toolKind: pluginToolKind(pluginId, label),
          nodes,
        });
      });
    });
    return items;
  }

  async function buildIndex() {
    const seq = ++buildSeq;
    loading = true;
    const next = [];

    const tree = await resultOrEmpty(App.GetWorkspaceTree(), { nodes: [] });
    const nodes = Array.isArray(tree.nodes) ? tree.nodes : [];
    nodes.forEach(node => {
      next.push({
        type: 'Workspace',
        typeLabel: tr('search.type.workspace'),
        title: workspaceTitle(node),
        subtitle: tr('search.type.workspace'),
        keywords: `${node.id || ''} ${node.rootPath || ''}`,
        rank: 10,
        action: 'workspace',
        workspaceName: workspaceName(node),
        nodes,
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
    const contributions = i18n.localizeContributionSummary(rawContributions || {});
    const viewByPluginId = new Map();
    (contributions.views || []).forEach(view => {
      if (view.pluginId && !viewByPluginId.has(view.pluginId)) viewByPluginId.set(view.pluginId, view);
    });
    (contributions.sidebarItems || []).forEach(item => {
      next.push({
        type: 'Tool',
        typeLabel: tr('search.type.tool'),
        title: item.title || item.id,
        subtitle: item.pluginId || '',
        keywords: `${item.id || ''} ${item.view || ''}`,
        rank: 20,
        action: 'view',
        viewId: item.view || item.id,
        pluginId: item.pluginId,
      });
    });

    const files = await listFilesRecursive();
    for (const entry of files) {
      const path = entry.relativePath || entry.path || entry.name || '';
      const snippet = await readFileSnippet(path);
      next.push({
        type: entry.type === 'folder' ? 'Folder' : 'File',
        typeLabel: tr(entry.type === 'folder' ? 'search.type.folder' : 'search.type.file'),
        title: path.split('/').pop() || path,
        subtitle: path,
        keywords: snippet,
        rank: entry.type === 'folder' ? 30 : 40,
        action: entry.type === 'folder' ? 'file-folder' : 'file',
        path,
        nodes,
      });
    }

    const pluginItems = await Promise.all([
      indexPluginSettings('verstak.journal', tr('search.type.journal'), 50, viewByPluginId.get('verstak.journal'), nodes),
      indexPluginSettings('verstak.browser-inbox', tr('search.type.browserInbox'), 55, viewByPluginId.get('verstak.browser-inbox'), nodes),
      indexPluginSettings('verstak.activity', tr('search.type.activity'), 60, viewByPluginId.get('verstak.activity'), nodes),
    ]);

    if (seq !== buildSeq) return;
    index = next.concat(pluginItems.flat());
    loading = false;
    runSearch(query);
  }

  function handleFocus() {
    focused = true;
    buildIndex();
  }

  async function openResult(item) {
    query = '';
    results = [];
    if (item.action === 'workspace') {
      window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
        detail: { workspaceName: item.workspaceName, nodes: item.nodes || [] }
      }));
      return;
    }
    if (item.action === 'view') {
      window.dispatchEvent(new CustomEvent('verstak:open-view', {
        detail: { viewId: item.viewId, pluginId: item.pluginId }
      }));
      return;
    }
    if (item.action === 'workspace-tool') {
      const workspaceName = item.workspaceName || '';
      if (workspaceName) {
        window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
          detail: { workspaceName, nodes: item.nodes || [] }
        }));
        window.dispatchEvent(new CustomEvent('verstak:workspace-open-tool', {
          detail: { kind: item.toolKind || item.type || '' }
        }));
      }
      return;
    }
    if (item.action === 'file-folder') {
      const parts = String(item.path || '').split('/').filter(Boolean);
      const workspaceName = parts[0] || '';
      const localPath = parts.slice(1).join('/');
      if (workspaceName) {
        window.__filesHistoryByWorkspace = window.__filesHistoryByWorkspace || {};
        window.__filesHistoryByWorkspace[workspaceName] = {
          stack: [localPath],
          index: 0,
          currentPath: localPath,
        };
        const detail = { workspaceName };
        if (Array.isArray(item.nodes) && item.nodes.length > 0) detail.nodes = item.nodes;
        window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
          detail
        }));
        window.dispatchEvent(new CustomEvent('verstak:workspace-open-tool', {
          detail: { kind: 'files' }
        }));
      }
      return;
    }
    if (item.action === 'file') {
      const response = await App.OpenWorkbenchResource('verstak.search', {
        kind: 'vault-file',
        path: item.path,
        mode: 'view',
        context: { sourceView: 'global-search' }
      });
      const [result, err] = Array.isArray(response) ? response : [response, ''];
      if (!err && result) {
        window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
      }
    }
  }
</script>

<div class="global-search" class:open={focused && (query || results.length)}>
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
            data-global-search-result-path={item.path || ''}
            on:mousedown|preventDefault={() => openResult(item)}
          >
            <span class="global-search-result-title">{item.title}</span>
            <span class="global-search-result-meta">{item.typeLabel || item.type} · {item.subtitle}</span>
          </button>
        {/each}
      {:else}
        <div class="global-search-empty vt-empty-title">{tr('search.noResults')}</div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .global-search {
    position: relative;
    padding: 0.55rem 0.75rem;
    border-bottom: 1px solid var(--vt-color-border);
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
    left: 0.75rem;
    right: 0.75rem;
    top: calc(100% - 0.25rem);
    z-index: 400;
    max-height: 20rem;
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
