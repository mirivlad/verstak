<script>
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import GlobalSearch from './GlobalSearch.svelte';
  import TodaySurface from './TodaySurface.svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import { onDestroy, onMount } from 'svelte';

  export let selectedWorkspaceName = '';
  export let nodes = [];
  export let activeToolKey = '';

  let contributions = {};
  let plugins = [];
  let workspaceTools = [];
  let toolsLoaded = false;
  let requestedToolKind = '';
  const todayTool = { id: '__today', title: 'Today', pluginId: 'verstak.shell', component: 'TodaySurface', shell: true };

  const toolOrder = new Map([
    ['notes', 10],
    ['files', 20],
    ['activity', 40],
    ['browser', 50],
    ['inbox', 50],
    ['search', 90],
  ]);

  $: selectedWorkspace = nodes.find(n => n.id === selectedWorkspaceName || n.name === selectedWorkspaceName || n.rootPath === selectedWorkspaceName) || null;
  $: workspaceRootPath = selectedWorkspace?.rootPath || selectedWorkspace?.name || selectedWorkspace?.id || '';
  $: workspaceTitle = selectedWorkspace?.title || selectedWorkspace?.name || selectedWorkspace?.id || selectedWorkspaceName;
  $: workspaceType = selectedWorkspace?.type || 'workspace';
  $: displayedTools = selectedWorkspace ? [todayTool, ...workspaceTools] : [];
  $: activeTool = displayedTools.find(tool => toolKey(tool) === activeToolKey) || displayedTools[0] || null;
  $: if (displayedTools.length > 0 && (!activeToolKey || (toolsLoaded && !displayedTools.some(tool => toolKey(tool) === activeToolKey)))) {
    activeToolKey = toolKey(todayTool);
  }
  $: if (requestedToolKind && workspaceTools.length > 0) {
    const match = findWorkspaceTool(requestedToolKind);
    if (match) {
      requestedToolKind = '';
      selectTool(match);
    }
  }
  $: if (selectedWorkspaceName) loadTools();

  onMount(() => {
    window.addEventListener('verstak:workspace-open-tool', handleWorkspaceOpenTool);
  });

  onDestroy(() => {
    window.removeEventListener('verstak:workspace-open-tool', handleWorkspaceOpenTool);
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

  function selectTool(tool) {
    activeToolKey = toolKey(tool);
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

  function requestWorkspaceTool(kind) {
    requestedToolKind = String(kind || '').toLowerCase();
    const match = findWorkspaceTool(requestedToolKind);
    if (match) selectTool(match);
  }

  function openWorkspaceTool(event) {
    requestWorkspaceTool(event?.detail?.kind);
  }

  function handleWorkspaceOpenTool(event) {
    requestWorkspaceTool(event?.detail?.kind);
  }

  async function loadTools() {
    try {
      toolsLoaded = false;
      const [c, p] = await Promise.all([
        App.GetContributions().catch(() => ({})),
        App.GetPlugins().catch(() => []),
      ]);
      contributions = c || {};
      plugins = p || [];

      const enabledIds = new Set(
        plugins.filter(pl => pl.enabled && (pl.status === 'loaded' || pl.status === 'degraded')).map(pl => pl.manifest?.id)
      );

      workspaceTools = sortWorkspaceTools((contributions.workspaceItems || []).filter(tool => enabledIds.has(tool.pluginId)));
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
        <span class="workspace-type vt-badge accent">{workspaceType}</span>
      </div>
      <div class="workspace-search" aria-label="Workspace search">
        <GlobalSearch />
      </div>
    </div>

    {#if displayedTools.length > 0}
      <div class="workspace-tabs vt-tabbar" role="tablist" aria-label="Workspace tools">
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
      <div class="workspace-tool-content" role="tabpanel" aria-label={activeTool?.title || activeTool?.id || 'Workspace tool'}>
        {#if activeTool}
          {#if activeTool.shell}
            <TodaySurface
              {workspaceRootPath}
              {workspaceTitle}
              availableTools={workspaceTools}
              on:openTool={openWorkspaceTool}
            />
          {:else}
            <PluginBundleHost
              pluginId={activeTool.pluginId}
              componentId={activeTool.component}
              componentProps={{ workspaceName: selectedWorkspaceName, workspaceNodeId: selectedWorkspaceName, workspaceNode: selectedWorkspace, workspaceRootPath }}
            />
          {/if}
        {/if}
      </div>
    {:else}
      <div class="workspace-empty vt-empty-state">
        <p class="vt-empty-title">No workspace tools available</p>
        <p class="workspace-hint">Enable plugins with workspace tools or open Plugin Manager from settings.</p>
      </div>
    {/if}
  {:else}
    <div class="workspace-empty vt-empty-state">
      <p class="vt-empty-title">Select a workspace</p>
      <p class="workspace-hint">Use the + button in Workspaces to add your first project.</p>
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

  .workspace-type {
    color: var(--vt-color-accent);
    font-size: 0.75rem;
    padding: 0.1rem 0.4rem;
    border-radius: var(--vt-radius-sm);
    background: var(--vt-color-accent-muted);
  }

  .workspace-search {
    width: min(27rem, 46vw);
    min-width: 16rem;
    flex-shrink: 1;
  }

  .workspace-search :global(.global-search) {
    padding: 0;
    border-bottom: 0;
  }

  .workspace-search :global(.global-search-box) {
    background: #0f1424;
    border-color: var(--vt-color-border-strong);
  }

  .workspace-search :global(.global-search-results) {
    left: 0;
    right: 0;
    top: calc(100% + 0.35rem);
  }

  @media (max-width: 720px) {
    .workspace-header {
      align-items: stretch;
      flex-direction: column;
    }

    .workspace-search {
      width: 100%;
      min-width: 0;
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
