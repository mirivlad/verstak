<script>
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import * as App from '../../../wailsjs/go/api/App';

  export let selectedWorkspaceName = '';
  export let nodes = [];

  let contributions = {};
  let plugins = [];
  let workspaceTools = [];
  let activeToolKey = '';

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
  $: activeTool = workspaceTools.find(tool => toolKey(tool) === activeToolKey) || workspaceTools[0] || null;
  $: if (workspaceTools.length > 0 && (!activeToolKey || !workspaceTools.some(tool => toolKey(tool) === activeToolKey))) {
    activeToolKey = toolKey(workspaceTools[0]);
  }
  $: if (selectedWorkspaceName) loadTools();

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

  async function loadTools() {
    try {
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
    }
  }
</script>

<div class="workspace-host">
  {#if selectedWorkspace}
    <div class="workspace-header">
      <div class="workspace-title-group">
        <span class="workspace-title">{workspaceTitle}</span>
        <span class="workspace-type">{workspaceType}</span>
      </div>
      <div class="workspace-search" data-workspace-search>
        <input type="search" placeholder="Search workspace" aria-label="Search workspace" />
      </div>
    </div>

    {#if workspaceTools.length > 0}
      <div class="workspace-tabs" role="tablist" aria-label="Workspace tools">
        {#each workspaceTools as tool (tool.id + tool.pluginId)}
          <button
            class:active={toolKey(tool) === toolKey(activeTool)}
            role="tab"
            aria-selected={toolKey(tool) === toolKey(activeTool)}
            type="button"
            title={tool.pluginId}
            on:click={() => activeToolKey = toolKey(tool)}
          >
            {tool.title || tool.id}
          </button>
        {/each}
      </div>
      <div class="workspace-tool-content" role="tabpanel" aria-label={activeTool?.title || activeTool?.id || 'Workspace tool'}>
        {#if activeTool}
          <PluginBundleHost
            pluginId={activeTool.pluginId}
            componentId={activeTool.component}
            componentProps={{ workspaceName: selectedWorkspaceName, workspaceNodeId: selectedWorkspaceName, workspaceNode: selectedWorkspace, workspaceRootPath }}
          />
        {/if}
      </div>
    {:else}
      <div class="workspace-empty">
        <p>No workspace tools available</p>
        <p class="workspace-hint">Install plugins that contribute workspaceItems to see tools here.</p>
      </div>
    {/if}
  {:else}
    <div class="workspace-empty">
      <p>Select a workspace node from the sidebar</p>
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
    background: #1a1a2e;
  }

  .workspace-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid #16213e;
    flex-shrink: 0;
  }

  .workspace-title-group {
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .workspace-title {
    color: #e0e0f0;
    font-size: 0.95rem;
    font-weight: 600;
  }

  .workspace-type {
    color: #4ecca3;
    font-size: 0.75rem;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    background: #1a2a3a;
  }

  .workspace-search {
    flex: 0 1 22rem;
    min-width: 12rem;
  }

  .workspace-search input {
    width: 100%;
    height: 2rem;
    padding: 0.25rem 0.55rem;
    border: 1px solid #283653;
    border-radius: 4px;
    background: #101626;
    color: #e0e0f0;
    font: inherit;
    font-size: 0.82rem;
    outline: none;
  }

  .workspace-search input:focus {
    border-color: #4ecca3;
  }

  @media (max-width: 720px) {
    .workspace-header {
      align-items: stretch;
      flex-direction: column;
    }

    .workspace-search {
      flex-basis: auto;
      min-width: 0;
    }
  }

  .workspace-tabs {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.35rem 0.75rem 0;
    background: #12122a;
    border-bottom: 1px solid #16213e;
    flex-shrink: 0;
  }

  .workspace-tabs button {
    min-height: 2rem;
    padding: 0.35rem 0.8rem;
    border: 1px solid transparent;
    border-bottom: none;
    border-radius: 6px 6px 0 0;
    background: transparent;
    color: #8b8ba8;
    cursor: pointer;
    font: inherit;
    font-size: 0.8rem;
  }

  .workspace-tabs button:hover {
    color: #e0e0f0;
    background: rgba(15, 52, 96, 0.4);
  }

  .workspace-tabs button.active {
    color: #4ecca3;
    background: #1a1a2e;
    border-color: #16213e;
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
    color: #666;
    gap: 0.5rem;
  }

  .workspace-hint {
    font-size: 0.8rem;
    color: #555;
    max-width: 300px;
    text-align: center;
  }
</style>
