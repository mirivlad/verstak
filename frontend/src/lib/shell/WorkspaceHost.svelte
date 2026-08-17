<script>
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import OverviewSurface from './OverviewSurface.svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import { onDestroy, onMount } from 'svelte';
  import { i18n } from '../i18n/index.js';
  import Icon from '../ui/Icon.svelte';

  export let selectedWorkspaceName = '';
  export let nodes = [];
  export let selectedWorkspaceId = '';
  export let activeToolKey = '';

  let contributions = {};
  let plugins = [];
  let discoveredWorkspaceTools = [];
  let workspaceTools = [];
  let overviewProviders = [];
  let workspaceMetadata = null;
  let metadataWorkspaceRoot = '';
  let toolsLoaded = false;
  let requestedWorkspaceItemId = '';
  let requestedToolRequest = null;
  let requestedTargetWorkspaceRoot = '';
  let activeToolRequest = null;
  let requestedWorkspaceRoot = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;
  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);
  $: overviewTool = { id: '__overview', title: tr('workspace.overview'), pluginId: 'verstak.shell', component: 'OverviewSurface', shell: true };

  // Where a tool sits among a Deal's tabs comes from its own manifest. The
  // shell used to hold a table matching substrings of plugin names to ranks,
  // which quietly meant every third-party tool landed at the end and the order
  // of the official ones could only be changed by editing the core.
  const UNRANKED = Number.MAX_SAFE_INTEGER;

  // Resolve workspace: try flat nodes first, then UUID-based lookup.
  $: selectedWorkspace = nodes.find(n => n.id === selectedWorkspaceName || n.name === selectedWorkspaceName || n.rootPath === selectedWorkspaceName) || null;
  $: workspaceRootPath = selectedWorkspace?.rootPath || selectedWorkspace?.name || selectedWorkspace?.id || selectedWorkspaceName || '';
  $: workspaceId = selectedWorkspace?.workspaceId || selectedWorkspaceId || '';
  $: workspaceTitle = selectedWorkspace?.title || selectedWorkspace?.name || selectedWorkspace?.id || selectedWorkspaceName;

  // If flat nodes lookup failed but we have a UUID, resolve via backend.
  $: if (!selectedWorkspace && selectedWorkspaceId && workspaceRootPath) {
    resolveWorkspaceByUUID(selectedWorkspaceId);
  }
  $: if (workspaceRootPath !== metadataWorkspaceRoot) {
    metadataWorkspaceRoot = workspaceRootPath;
    workspaceMetadata = null;
    if (workspaceRootPath) loadWorkspaceMetadata(workspaceRootPath);
  }
  $: workspaceTools = sortWorkspaceTools(filterWorkspaceTools(discoveredWorkspaceTools, workspaceMetadata));
  $: workspacePluginIds = new Set(workspaceTools.map(tool => tool?.pluginId).filter(Boolean));
  $: overviewProviders = (contributions.overviewProviders || []).filter(provider => workspacePluginIds.has(provider?.pluginId));
  $: if (workspaceRootPath !== requestedWorkspaceRoot) {
    requestedWorkspaceRoot = workspaceRootPath;
    activeToolRequest = null;
    if (requestedTargetWorkspaceRoot && requestedTargetWorkspaceRoot !== workspaceRootPath) {
      requestedWorkspaceItemId = '';
      requestedToolRequest = null;
      requestedTargetWorkspaceRoot = '';
    } else if (!requestedTargetWorkspaceRoot) {
      requestedToolRequest = null;
    }
  }
  $: displayedTools = selectedWorkspace ? [overviewTool, ...workspaceTools] : [];
  $: activeTool = displayedTools.find(tool => toolKey(tool) === activeToolKey) || displayedTools[0] || null;
  $: if (displayedTools.length > 0 && (!activeToolKey || (toolsLoaded && !displayedTools.some(tool => toolKey(tool) === activeToolKey)))) {
    activeToolKey = toolKey(overviewTool);
  }
  $: if (selectedWorkspace) {
    window.dispatchEvent(new CustomEvent('verstak:content-title-changed', {
      detail: { title: workspaceTitle }
    }));
  }
  $: if (requestedWorkspaceItemId && workspaceTools.length > 0 && (!requestedTargetWorkspaceRoot || requestedTargetWorkspaceRoot === workspaceRootPath)) {
    const match = findWorkspaceItem(requestedWorkspaceItemId);
    if (match) {
      const toolRequest = requestedToolRequest;
      requestedWorkspaceItemId = '';
      requestedToolRequest = null;
      requestedTargetWorkspaceRoot = '';
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

  // An empty state that only describes the emptiness leaves the user to work
  // out where the action lives. These do the step for them.
  function requestCreateDeal() {
    window.dispatchEvent(new CustomEvent('verstak:create-workspace-requested'));
  }

  function openPluginManager() {
    window.dispatchEvent(new CustomEvent('verstak:nav', { detail: { viewId: 'plugin-manager' } }));
  }

  function toolKey(tool) {
    return `${tool?.pluginId || ''}:${tool?.id || ''}`;
  }

  function toolRank(tool) {
    const order = Number(tool?.order);
    return Number.isFinite(order) && order !== 0 ? order : UNRANKED;
  }

  function sortWorkspaceTools(tools) {
    return [...tools].sort((a, b) => {
      const rankDiff = toolRank(a) - toolRank(b);
      if (rankDiff !== 0) return rankDiff;
      return String(a.title || a.id).localeCompare(String(b.title || b.id));
    });
  }

  function filterWorkspaceTools(tools, metadata) {
    if (!metadata) return tools;
    let allowed = metadata.workspaceTools;
    if (!Array.isArray(allowed)) {
      // Metadata created before workspace templates did not restrict tools.
      // Preserve that behavior for manual and migrated workspaces.
      if (!metadata.createdFromTemplate) return tools;
      // Template metadata can derive the intended tool set from its features.
      const f = metadata.features || {};
      allowed = ['verstak.notes', 'verstak.files'];
      if (f.journal) allowed.push('verstak.journal');
      if (f.activity) allowed.push('verstak.activity');
      if (f['browser-inbox']) allowed.push('verstak.browser-inbox');
      if (f.todo) allowed.push('verstak.todo');
      if (f.secrets) allowed.push('verstak.secrets');
    }
    const allowedSet = new Set(allowed);
    return tools.filter(tool => allowedSet.has(tool.pluginId));
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

  let resolvingUUID = '';
  async function resolveWorkspaceByUUID(uuid) {
    if (!uuid || uuid === resolvingUUID) return;
    resolvingUUID = uuid;
    try {
      const ws = await App.GetWorkspaceByID(uuid);
      if (ws && ws.rootPath && uuid === selectedWorkspaceId) {
        // Create a synthetic node for the flat nodes array
        const synth = { id: ws.rootPath, workspaceId: uuid, name: ws.name, rootPath: ws.rootPath, title: ws.name || ws.rootPath };
        nodes = [...nodes.filter(n => n.workspaceId !== uuid), synth];
        if (ws.rootPath !== workspaceRootPath) {
          // Force re-evaluation of reactive bindings
          selectedWorkspaceName = ws.rootPath;
        }
      }
    } catch {} finally {
      resolvingUUID = '';
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

  function findWorkspaceItem(workspaceItemId) {
    const id = String(workspaceItemId || '').trim();
    if (!id) return null;
    return workspaceTools.find(tool => tool?.id === id) || null;
  }

  function requestWorkspaceItem(workspaceItemId, toolRequest = null, targetWorkspaceRoot = '') {
    requestedWorkspaceItemId = String(workspaceItemId || '').trim();
    requestedToolRequest = toolRequest;
    requestedTargetWorkspaceRoot = String(targetWorkspaceRoot || '').trim();
    if (requestedTargetWorkspaceRoot && requestedTargetWorkspaceRoot !== workspaceRootPath) return;
    const match = findWorkspaceItem(requestedWorkspaceItemId);
    if (match) {
      requestedWorkspaceItemId = '';
      requestedToolRequest = null;
      requestedTargetWorkspaceRoot = '';
      selectTool(match, toolRequest);
    }
  }

  function openWorkspaceTool(event) {
    requestWorkspaceItem(event?.detail?.workspaceItemId, event?.detail?.toolRequest || null, event?.detail?.workspaceRootPath || '');
  }

  function handleWorkspaceOpenTool(event) {
    requestWorkspaceItem(event?.detail?.workspaceItemId, event?.detail?.toolRequest || null, event?.detail?.workspaceRootPath || '');
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
    {#if displayedTools.length > 0}
      <div class="workspace-tabs vt-tabbar" role="tablist" aria-label={tr('workspace.tools')}>
        {#each displayedTools as tool (tool.id + tool.pluginId)}
          <button
            class="vt-tab"
            class:is-active={toolKey(tool) === toolKey(activeTool)}
            role="tab"
            aria-selected={toolKey(tool) === toolKey(activeTool)}
            type="button"
            title={tool.title || tool.id}
            on:click={() => selectTool(tool)}
          >
            {tool.title || tool.id}
          </button>
        {/each}
      </div>
      <div class="workspace-tool-content" role="tabpanel" aria-label={activeTool?.title || activeTool?.id || tr('workspace.tool')}>
        {#if activeTool}
          {#if activeTool.shell}
            <OverviewSurface
              {workspaceRootPath}
              availableTools={displayedTools}
              {overviewProviders}
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
      <div class="workspace-empty vt-empty-state" data-workspace-no-tools>
        <Icon name="puzzle" size={28} />
        <p class="vt-empty-title">{tr('workspace.emptyTools')}</p>
        <p class="workspace-hint">{tr('workspace.emptyToolsHint')}</p>
        <button class="vt-button" type="button" data-workspace-open-plugins on:click={openPluginManager}>
          {tr('settings.openPluginManager', undefined, 'Open Plugin Manager')}
        </button>
      </div>
    {/if}
  {:else}
    <div class="workspace-empty vt-empty-state" data-workspace-empty>
      <Icon name="layout-grid" size={28} />
      <p class="vt-empty-title">{tr('workspace.select')}</p>
      <p class="workspace-hint">{tr('workspace.selectHint')}</p>
      <button class="vt-button" type="button" data-workspace-create on:click={requestCreateDeal}>
        {tr('workspace.createFirst', undefined, 'Create a Deal')}
      </button>
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
