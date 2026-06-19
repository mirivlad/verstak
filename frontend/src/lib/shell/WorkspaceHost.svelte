<script>
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import * as App from '../../../wailsjs/go/api/App';

  export let currentNodeId = '';
  export let nodes = [];

  let contributions = {};
  let plugins = [];
  let workspaceTools = [];

  $: currentNode = nodes.find(n => n.id === currentNodeId) || null;
  $: if (currentNodeId) loadTools();

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

      workspaceTools = (contributions.workspaceItems || []).filter(tool => enabledIds.has(tool.pluginId));
    } catch (e) {
      console.error('[WorkspaceHost] loadTools error:', e);
    }
  }
</script>

<div class="workspace-host">
  {#if currentNode}
    <div class="workspace-header">
      <span class="workspace-title">{currentNode.title}</span>
      <span class="workspace-type">{currentNode.type}</span>
    </div>

    {#if workspaceTools.length > 0}
      <div class="workspace-tools">
        {#each workspaceTools as tool (tool.id + tool.pluginId)}
          <div class="workspace-tool">
            <div class="tool-header">
              <span class="tool-title">{tool.title || tool.id}</span>
              <span class="tool-plugin">{tool.pluginId}</span>
            </div>
            <div class="tool-content">
              <PluginBundleHost
                pluginId={tool.pluginId}
                componentId={tool.component}
                componentProps={{ workspaceNodeId: currentNodeId, workspaceNode: currentNode }}
              />
            </div>
          </div>
        {/each}
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
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid #16213e;
    flex-shrink: 0;
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

  .workspace-tools {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0.5rem;
  }

  .workspace-tool {
    border: 1px solid #16213e;
    border-radius: 6px;
    margin-bottom: 0.5rem;
    overflow: hidden;
  }

  .tool-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.75rem;
    background: #12122a;
    border-bottom: 1px solid #16213e;
  }

  .tool-title {
    color: #e0e0f0;
    font-size: 0.8rem;
    font-weight: 600;
  }

  .tool-plugin {
    color: #666;
    font-size: 0.65rem;
    margin-left: auto;
  }

  .tool-content {
    min-height: 300px;
    max-height: 60vh;
    overflow: auto;
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
