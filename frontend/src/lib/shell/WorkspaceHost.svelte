<script>
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import TodaySurface from './TodaySurface.svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import { onDestroy, onMount } from 'svelte';
  import { i18n } from '../i18n/index.js';

  export let selectedWorkspaceName = '';
  export let nodes = [];
  export let activeToolKey = '';

  let contributions = {};
  let plugins = [];
  let discoveredWorkspaceTools = [];
  let workspaceTools = [];
  let workspaceMetadata = null;
  let metadataWorkspaceRoot = '';
  let toolsLoaded = false;
  let requestedToolKind = '';
  let requestedToolRequest = null;
  let activeToolRequest = null;
  let requestedWorkspaceRoot = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;
  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);
  // TODO: Rename TodaySurface.svelte to OverviewSurface.svelte in a refactor-only follow-up.
  $: overviewTool = { id: '__overview', title: tr('workspace.overview'), pluginId: 'verstak.shell', component: 'TodaySurface', shell: true };

  const toolOrder = new Map([
    ['notes', 10],
    ['files', 20],
    ['todo', 30],
    ['activity', 40],
    ['browser', 50],
    ['inbox', 50],
    ['secrets', 60],
    ['search', 90],
  ]);

  $: selectedWorkspace = nodes.find(n => n.id === selectedWorkspaceName || n.name === selectedWorkspaceName || n.rootPath === selectedWorkspaceName) || null;
  $: workspaceRootPath = selectedWorkspace?.rootPath || selectedWorkspace?.name || selectedWorkspace?.id || '';
  $: workspaceId = selectedWorkspace?.workspaceId || '';
  $: workspaceTitle = selectedWorkspace?.title || selectedWorkspace?.name || selectedWorkspace?.id || selectedWorkspaceName;
  $: if (workspaceRootPath !== metadataWorkspaceRoot) {
    metadataWorkspaceRoot = workspaceRootPath;
    workspaceMetadata = null;
    if (workspaceRootPath) loadWorkspaceMetadata(workspaceRootPath);
  }
  $: workspaceTools = sortWorkspaceTools(filterWorkspaceTools(discoveredWorkspaceTools, workspaceMetadata));
  $: if (workspaceRootPath !== requestedWorkspaceRoot) {
    requestedWorkspaceRoot = workspaceRootPath;
    requestedToolRequest = null;
    activeToolRequest = null;
  }
  $: displayedTools = selectedWorkspace ? [overviewTool, ...workspaceTools] : [];
  $: activeTool = displayedTools.find(tool => toolKey(tool) === activeToolKey) || displayedTools[0] || null;
  $: if (displayedTools.length > 0 && (!activeToolKey || (toolsLoaded && !displayedTools.some(tool => toolKey(tool) === activeToolKey)))) {
    activeToolKey = toolKey(overviewTool);
  }
  $: if (requestedToolKind && workspaceTools.length > 0) {
    const match = findWorkspaceTool(requestedToolKind);
    if (match) {
      const toolRequest = requestedToolRequest;
      requestedToolKind = '';
      requestedToolRequest = null;
      selectTool(match, toolRequest);
    }
  }
  $: if (selectedWorkspaceName) loadTools();

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed && selectedWorkspaceName) loadTools();
    });
    window.addEventListener('verstak:workspace-open-tool', handleWorkspaceOpenTool);
    window.addEventListener('verstak:plugins-changed', handlePluginsChanged);
  });

  onDestroy(() => {
    if (unsubscribeLocale) unsubscribeLocale();
    window.removeEventListener('verstak:workspace-open-tool', handleWorkspaceOpenTool);
    window.removeEventListener('verstak:plugins-changed', handlePluginsChanged);
  });

  function toolKey(tool) {
    return `${tool?.pluginId || ''}:${tool?.id || ''}`;
  }

  function toolRank(tool) {
    const text = `${tool?.title || ''} ${tool?.id || ''} ${tool?.pluginId || ''}`.toLowerCase();
    for (const [needle, rank] of toolOrder.entries()) {
      if (text.includes(needle)) return rank;
    }
    return 1000;
  }

  function sortWorkspaceTools(tools) {
    return [...tools].sort((a, b) => {
      const rankDiff = toolRank(a) - toolRank(b);
      if (rankDiff !== 0) return rankDiff;
      return String(a.title || a.id).localeCompare(String(b.title || b.id));
    });
  }

  function filterWorkspaceTools(tools, metadata) {
    if (!Array.isArray(metadata?.workspaceTools)) return tools;
    const allowedPluginIds = new Set(metadata.workspaceTools);
    return tools.filter(tool => allowedPluginIds.has(tool.pluginId));
  }

  function resultOrError(response, fallbackValue) {
    if (Array.isArray(response) && typeof response[1] === 'string') {
      return [response[0] || fallbackValue, response[1] || ''];
    }
    return typeof response === 'string' ? [fallbackValue, response] : [response || fallbackValue, ''];
  }

  async function loadWorkspaceMetadata(rootPath) {
    try {
      const [metadata, err] = resultOrError(await App.GetWorkspaceMetadata(rootPath), null);
      if (rootPath !== workspaceRootPath) return;
      workspaceMetadata = err ? null : metadata;
    } catch (_) {
      if (rootPath === workspaceRootPath) workspaceMetadata = null;
    }
  }

  function selectTool(tool, toolRequest = null) {
    activeToolKey = toolKey(tool);
    activeToolRequest = toolRequest;
    window.dispatchEvent(new CustomEvent('verstak:workspace-tool-selected', {
      detail: {
        toolKey: activeToolKey,
        toolId: tool?.id || '',
        pluginId: tool?.pluginId || '',
      },
    }));
  }

  function findWorkspaceTool(kind) {
    kind = String(kind || '').toLowerCase();
    return workspaceTools.find(tool => {
      const text = `${tool?.title || ''} ${tool?.id || ''} ${tool?.pluginId || ''}`.toLowerCase();
      if (kind === 'browser-inbox') return text.includes('browser') || text.includes('inbox');
      return text.includes(kind);
    });
  }

  function requestWorkspaceTool(kind, toolRequest = null) {
    requestedToolKind = String(kind || '').toLowerCase();
    requestedToolRequest = toolRequest;
    const match = findWorkspaceTool(requestedToolKind);
    if (match) {
      requestedToolKind = '';
      requestedToolRequest = null;
      selectTool(match, toolRequest);
    }
  }

  function openWorkspaceTool(event) {
    requestWorkspaceTool(event?.detail?.kind, event?.detail?.toolRequest || null);
  }

  function handleWorkspaceOpenTool(event) {
    requestWorkspaceTool(event?.detail?.kind, event?.detail?.toolRequest || null);
  }

  function handlePluginsChanged() {
    if (selectedWorkspaceName) loadTools();
  }

  async function loadTools() {
    try {
      toolsLoaded = false;
      const [c, p] = await Promise.all([
        App.GetContributions().catch(() => ({})),
        App.GetPlugins().catch(() => []),
      ]);
      await Promise.all((p || []).map((plugin) => (
        i18n.loadPlugin(plugin.manifest?.id, plugin.manifest?.localization).catch(() => {})
      )));
      contributions = i18n.localizeContributionSummary(c || {});
      plugins = (p || []).map((plugin) => i18n.localizePlugin(plugin));

      const enabledIds = new Set(
        plugins.filter(pl => pl.enabled && (pl.status === 'loaded' || pl.status === 'degraded')).map(pl => pl.manifest?.id)
      );

      discoveredWorkspaceTools = (contributions.workspaceItems || []).filter(tool => enabledIds.has(tool.pluginId));
    } catch (e) {
      console.error('[WorkspaceHost] loadTools error:', e);
    } finally {
      toolsLoaded = true;
    }
  }
</script>

<div class="workspace-host vt-page">
  {#if selectedWorkspace}
    <div class="workspace-header vt-page-header">
      <div class="workspace-title-group">
        <span class="workspace-title vt-page-title">{workspaceTitle}</span>
      </div>
    </div>

    {#if displayedTools.length > 0}
      <div class="workspace-tabs vt-tabbar" role="tablist" aria-label={tr('workspace.tools')}>
        {#each displayedTools as tool (tool.id + tool.pluginId)}
          <button
            class="vt-tab"
            class:is-active={toolKey(tool) === toolKey(activeTool)}
            role="tab"
            aria-selected={toolKey(tool) === toolKey(activeTool)}
            type="button"
            title={tool.pluginId}
            on:click={() => selectTool(tool)}
          >
            {tool.title || tool.id}
          </button>
        {/each}
      </div>
      <div class="workspace-tool-content" role="tabpanel" aria-label={activeTool?.title || activeTool?.id || tr('workspace.tool')}>
        {#if activeTool}
          {#if activeTool.shell}
            <TodaySurface
              {workspaceRootPath}
              availableTools={displayedTools}
              on:openTool={openWorkspaceTool}
            />
          {:else}
            <PluginBundleHost
              pluginId={activeTool.pluginId}
              componentId={activeTool.component}
              componentProps={{ workspaceName: selectedWorkspaceName, workspaceNodeId: selectedWorkspaceName, workspaceNode: selectedWorkspace, workspaceRootPath, workspaceId, toolRequest: activeToolRequest }}
            />
          {/if}
        {/if}
      </div>
    {:else}
      <div class="workspace-empty vt-empty-state">
        <p class="vt-empty-title">{tr('workspace.emptyTools')}</p>
        <p class="workspace-hint">{tr('workspace.emptyToolsHint')}</p>
      </div>
    {/if}
  {:else}
    <div class="workspace-empty vt-empty-state">
      <p class="vt-empty-title">{tr('workspace.select')}</p>
      <p class="workspace-hint">{tr('workspace.selectHint')}</p>
    </div>
  {/if}
</div>

<style>
  .workspace-host {
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--vt-color-background);
  }

  .workspace-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: var(--vt-space-2) var(--vt-space-4);
    border-bottom: 1px solid var(--vt-color-border);
    flex-shrink: 0;
  }

  .workspace-title-group {
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .workspace-title {
    color: var(--vt-color-text-primary);
    font-size: 0.95rem;
    font-weight: 600;
  }

  @media (max-width: 720px) {
    .workspace-header {
      align-items: stretch;
      flex-direction: column;
    }
  }

  .workspace-tabs {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding: var(--vt-space-1) var(--vt-space-3) 0;
    background: #12162a;
    border-bottom: 1px solid var(--vt-color-border);
    flex-shrink: 0;
    overflow-x: auto;
    scrollbar-gutter: auto;
  }

  .workspace-tabs button {
    flex-shrink: 0;
    min-height: 2rem;
    padding: 0.35rem 0.8rem;
    border: 1px solid transparent;
    border-bottom: none;
    border-radius: var(--vt-radius-md) var(--vt-radius-md) 0 0;
    background: transparent;
    color: var(--vt-color-text-muted);
    cursor: pointer;
    font: inherit;
    font-size: 0.8rem;
  }

  .workspace-tabs button:hover {
    color: var(--vt-color-text-primary);
    background: var(--vt-color-surface-hover);
  }

  .workspace-tabs button.is-active {
    color: var(--vt-color-accent);
    background: var(--vt-color-background);
    border-color: var(--vt-color-border);
  }

  .workspace-tool-content {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .workspace-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: var(--vt-color-text-muted);
    gap: 0.5rem;
    padding: 2rem;
    text-align: center;
  }

  .workspace-hint {
    font-size: 0.8rem;
    color: var(--vt-color-text-muted);
    max-width: 300px;
    text-align: center;
  }
</style>
