<script context="module">
  import { writable } from 'svelte/store';

  const activeWorkspaceId = writable('');
</script>

<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import { i18n } from '../i18n/index.js';

  let loading = true;
  let localError = '';
  let workspaces = [];
  let currentWorkspaceId = '';
  let showCreate = false;
  let newWorkspaceName = '';
  let workspaceTemplates = [];
  let templatePluginNames = {};
  let templatePlugins = {};
  let templateCapabilities = new Set();
  let selectedTemplateId = 'default';
  let createError = '';
  let templateWarning = null;
  let templatesLoading = false;
  let creating = false;
  let renamingId = '';
  let renameValue = '';
  let busyId = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;
  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);

  function reportError(key, fallback, details) {
    console.warn('[WorkspaceTree] operation failed:', details);
    return tr(key, undefined, fallback);
  }

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed) loadWorkspaceTemplates();
    });
    loadWorkspaces();
    loadWorkspaceTemplates();
    window.addEventListener('verstak:workspace-active-changed', onActiveWorkspaceChanged);
  });

  onDestroy(() => {
    if (unsubscribeLocale) unsubscribeLocale();
    window.removeEventListener('verstak:workspace-active-changed', onActiveWorkspaceChanged);
  });

  function onActiveWorkspaceChanged(event) {
    currentWorkspaceId = event.detail?.workspaceName || '';
    activeWorkspaceId.set(currentWorkspaceId);
  }

  function resultOrError(response, fallbackValue) {
    if (Array.isArray(response) && typeof response[1] === 'string') {
      return [response[0] || fallbackValue, response[1] || ''];
    }
    return typeof response === 'string' ? [fallbackValue, response] : [response, ''];
  }

  $: selectedTemplate = workspaceTemplates.find(template => template.id === selectedTemplateId) || workspaceTemplates[0] || null;
  $: selectedTemplateTools = (selectedTemplate?.workspaceTools || []).map((pluginId) => (
    templateToolState(pluginId, templatePlugins, templateCapabilities, templatePluginNames, tr)
  ));
  $: selectedTemplateIssues = selectedTemplateTools.filter(tool => tool.status !== 'available');

  function toolLabel(pluginId, names = templatePluginNames) {
    return names[pluginId] || String(pluginId || '').replace(/^verstak\./, '');
  }

  function templateToolState(pluginId, plugins, capabilities, names, translate) {
    const plugin = plugins[pluginId];
    if (!plugin) {
      return {
        pluginId,
        name: toolLabel(pluginId, names),
        tabs: [],
        status: 'unavailable',
        reason: translate('workspaceTree.templateMissingPlugin'),
      };
    }

    const manifest = plugin.manifest || {};
    const tabs = Array.isArray(manifest.contributes?.workspaceItems)
      ? manifest.contributes.workspaceItems.map(item => item?.title || item?.id).filter(Boolean)
      : [];
    const pluginStatus = String(plugin.status || '').toLowerCase();
    const missingCapability = Array.isArray(manifest.requires)
      && manifest.requires.some(capabilityId => !capabilities.has(capabilityId));
    let status = 'available';
    let reason = translate('workspaceTree.templateAvailable');

    if (!plugin.enabled || pluginStatus === 'disabled') {
      status = 'unavailable';
      reason = translate('workspaceTree.templatePluginDisabled');
    } else if (pluginStatus === 'missing-required-capability' || missingCapability) {
      status = 'unavailable';
      reason = translate('workspaceTree.templateCapabilityUnavailable');
    } else if (pluginStatus === 'incompatible') {
      status = 'unavailable';
      reason = translate('workspaceTree.templateIncompatible');
    } else if (pluginStatus === 'failed') {
      status = 'unavailable';
      reason = translate('workspaceTree.templateLoadFailed');
    } else if (pluginStatus === 'degraded') {
      status = 'limited';
      reason = translate('workspaceTree.templateLimited');
    } else if (pluginStatus !== 'loaded') {
      status = 'unavailable';
      reason = translate('workspaceTree.templateNotReady');
    }

    return { pluginId, name: manifest.name || toolLabel(pluginId, names), tabs, status, reason };
  }

  async function loadWorkspaceTemplates() {
    templatesLoading = true;
    try {
      const [templates, plugins, capabilities] = await Promise.all([
        App.ListWorkspaceTemplates ? App.ListWorkspaceTemplates() : [],
        App.GetPlugins ? App.GetPlugins() : [],
        App.GetCapabilities ? App.GetCapabilities() : [],
      ]);
      const [list, err] = resultOrError(templates, []);
      if (err) {
        createError = reportError('workspaceTree.templatesError', 'Could not load Deal templates. Please try again.', err);
        workspaceTemplates = [];
        return;
      }
      workspaceTemplates = Array.isArray(list) ? list : [];
      await Promise.all((Array.isArray(plugins) ? plugins : []).map((plugin) => (
        i18n.loadPlugin(plugin.manifest?.id, plugin.manifest?.localization).catch(() => {})
      )));
      const localizedPlugins = (Array.isArray(plugins) ? plugins : []).map((plugin) => i18n.localizePlugin(plugin));
      templatePluginNames = localizedPlugins.reduce((names, plugin) => {
        const id = plugin?.manifest?.id;
        const name = plugin?.manifest?.name;
        if (id && name) names[id] = name;
        return names;
      }, {});
      templatePlugins = localizedPlugins.reduce((result, plugin) => {
        const id = plugin?.manifest?.id;
        if (id) result[id] = plugin;
        return result;
      }, {});
      const [capabilityList] = resultOrError(capabilities, []);
      templateCapabilities = new Set((Array.isArray(capabilityList) ? capabilityList : []).map(capability => capability?.name).filter(Boolean));
      if (!workspaceTemplates.some(template => template.id === selectedTemplateId)) {
        selectedTemplateId = workspaceTemplates[0]?.id || '';
      }
    } catch (error) {
      createError = reportError('workspaceTree.templatesError', 'Could not load Deal templates. Please try again.', error);
      workspaceTemplates = [];
    } finally {
      templatesLoading = false;
    }
  }

  function wsName(workspace) {
    return String(workspace?.name || workspace?.rootPath || '');
  }

  function asNode(workspace, order) {
    const name = wsName(workspace);
    return {
      id: name,
      type: 'space',
      title: name,
      name,
      rootPath: workspace.rootPath || name,
      status: 'active',
      order,
    };
  }

  function nodesForEvent() {
    return workspaces.map(asNode);
  }

  async function loadWorkspaces() {
    loading = true;
    localError = '';
    try {
      const [list, err] = resultOrError(await App.ListWorkspaces(), []);
      if (err) {
        localError = reportError('workspaceTree.loadError', 'Could not load Deals. Please try again.', err);
        workspaces = [];
      } else {
        workspaces = list || [];
        if (!currentWorkspaceId) {
          let currentWorkspace = null;
          try {
            currentWorkspace = await App.GetCurrentWorkspace();
          } catch {
            currentWorkspace = null;
          }
          const currentName = wsName(currentWorkspace);
          if (workspaces.some((ws) => wsName(ws) === currentName)) {
            currentWorkspaceId = currentName;
          }
        } else if (!workspaces.some((ws) => wsName(ws) === currentWorkspaceId)) {
          currentWorkspaceId = '';
        }
        activeWorkspaceId.set(currentWorkspaceId);
      }
    } catch (e) {
      localError = reportError('workspaceTree.loadError', 'Could not load Deals. Please try again.', e);
    }
    loading = false;
  }

  async function selectWorkspace(workspace) {
    const id = wsName(workspace);
    const err = await App.SetCurrentWorkspace(id);
    if (err) {
      localError = reportError('workspaceTree.selectError', 'Could not select this Deal. Please try again.', err);
      return;
    }
    currentWorkspaceId = id;
    activeWorkspaceId.set(id);
    window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
      detail: { workspaceName: id, nodes: nodesForEvent() }
    }));
  }

  async function doCreate() {
    const name = newWorkspaceName.trim();
    if (!name) {
      createError = tr('workspaceTree.nameRequired');
      return;
    }
    if (!selectedTemplate) {
      createError = tr('workspaceTree.chooseTemplate');
      return;
    }
    const creationIssues = selectedTemplateIssues.map(tool => ({
      pluginId: tool.pluginId,
      name: tool.name,
      reason: tool.reason,
    }));
    creating = true;
    createError = '';
    const [, err] = resultOrError(await App.CreateWorkspace(name, selectedTemplate.id), null);
    if (err) {
      createError = reportError('workspaceTree.createError', 'Could not create the Deal. Please try again.', err);
      creating = false;
      return;
    }
    showCreate = false;
    newWorkspaceName = '';
    creating = false;
    await loadWorkspaces();
    const created = workspaces.find((ws) => wsName(ws) === name);
    if (created) await selectWorkspace(created);
    templateWarning = creationIssues.length > 0 ? { workspaceName: name, issues: creationIssues } : null;
  }

  async function openCreateDialog() {
    showCreate = true;
    newWorkspaceName = '';
    createError = '';
    await loadWorkspaceTemplates();
  }

  function closeCreateDialog() {
    if (creating) return;
    showCreate = false;
    newWorkspaceName = '';
    createError = '';
  }

  function dismissTemplateWarning() {
    templateWarning = null;
  }

  function startRename(workspace) {
    renamingId = wsName(workspace);
    renameValue = renamingId;
    localError = '';
  }

  function cancelRename() {
    renamingId = '';
    renameValue = '';
  }

  async function commitRename(workspace) {
    const oldName = wsName(workspace);
    const newName = renameValue.trim();
    if (!newName || newName === oldName) {
      cancelRename();
      return;
    }
    busyId = oldName;
    const err = await App.RenameWorkspace(oldName, newName);
    if (err) {
      localError = reportError('workspaceTree.renameError', 'Could not rename the Deal. Please try again.', err);
      busyId = '';
      return;
    }
    renamingId = '';
    renameValue = '';
    busyId = '';
    currentWorkspaceId = newName;
    await loadWorkspaces();
    const renamed = workspaces.find((ws) => wsName(ws) === newName);
    if (renamed) await selectWorkspace(renamed);
  }

  async function trashWorkspace(workspace) {
    const name = wsName(workspace);
    busyId = name;
    const [, err] = resultOrError(await App.TrashWorkspace(name), null);
    if (err) {
      localError = reportError('workspaceTree.trashError', 'Could not move the Deal to trash. Please try again.', err);
      busyId = '';
      return;
    }
    if (currentWorkspaceId === name) currentWorkspaceId = '';
    busyId = '';
    await loadWorkspaces();
    if (currentWorkspaceId) {
      const selected = workspaces.find((ws) => wsName(ws) === currentWorkspaceId);
      if (selected) await selectWorkspace(selected);
    }
  }
</script>

<div class="wt">
  <div class="wt-header">
    <span class="wt-title">{tr('workspaceTree.title')}</span>
    <button class="wt-btn" on:click={openCreateDialog} title={tr('workspaceTree.new')} type="button">+</button>
  </div>

  {#if loading}
    <div class="wt-loading">{tr('common.loading')}</div>
  {:else if localError}
    <div class="wt-error">{localError}</div>
  {/if}

  <div class="wt-list">
    {#each workspaces as workspace (wsName(workspace))}
      {@const id = wsName(workspace)}
      <div class="wt-node vt-list-row" class:selected={id === $activeWorkspaceId}>
        <div class="wt-row">
          <span class="wt-icon"><Icon name="space" size={13} class="wt-node-icon" /></span>
          {#if renamingId === id}
            <input
              class="wt-rename"
              bind:value={renameValue}
              disabled={busyId === id}
              on:keydown={(e) => {
                if (e.key === 'Enter') commitRename(workspace);
                if (e.key === 'Escape') cancelRename();
              }}
            />
            <button class="wt-btn wt-btn-small wt-always" on:click={() => commitRename(workspace)} title={tr('workspaceTree.saveRename')} type="button" disabled={busyId === id}>{tr('common.save')}</button>
            <button class="wt-btn wt-btn-small wt-always" on:click={cancelRename} title={tr('common.cancel')} type="button" disabled={busyId === id}>{tr('common.cancel')}</button>
          {:else}
            <button class="wt-label" on:click={() => selectWorkspace(workspace)} type="button">{id}</button>
            <button class="wt-icon-btn" on:click={() => startRename(workspace)} title={tr('workspaceTree.rename')} type="button" disabled={busyId === id}>
              <Icon name="edit" size={12} />
            </button>
            <button class="wt-icon-btn danger" on:click={() => trashWorkspace(workspace)} title={tr('workspaceTree.trash')} type="button" disabled={busyId === id}>
              <Icon name="trash" size={12} />
            </button>
          {/if}
        </div>
      </div>
    {/each}
  </div>

  {#if templateWarning}
    <section class="workspace-template-warning" data-workspace-template-warning role="alert">
      <div>
        <strong>{tr('workspaceTree.templateIncompleteCreated', { name: templateWarning.workspaceName })}</strong>
        {#each templateWarning.issues as issue (issue.pluginId)}
          <div class="workspace-template-warning-item">{issue.name}: {issue.reason}</div>
        {/each}
      </div>
      <button class="wt-btn" type="button" on:click={dismissTemplateWarning} title={tr('common.close')}>{tr('common.close')}</button>
    </section>
  {/if}

  {#if showCreate}
    <div class="workspace-create-overlay" data-workspace-create-modal role="dialog" aria-modal="true" aria-label={tr('workspaceTree.create')}>
      <div class="workspace-create-modal">
        <div class="workspace-create-header">
          <div>
            <h2>{tr('workspaceTree.new')}</h2>
          </div>
          <button class="wt-btn" on:click={closeCreateDialog} type="button" disabled={creating}>{tr('common.close')}</button>
        </div>
        <label class="workspace-create-field">
          <span>{tr('pluginCard.name')}</span>
          <input data-workspace-name type="text" bind:value={newWorkspaceName} placeholder={tr('workspaceTree.namePlaceholder')} disabled={creating} on:keydown={(event) => event.key === 'Enter' && doCreate()} />
        </label>
        <label class="workspace-create-field">
          <span>{tr('workspaceTree.template')}</span>
          <select data-workspace-template bind:value={selectedTemplateId} disabled={creating || templatesLoading || !workspaceTemplates.length}>
            {#each workspaceTemplates as template (template.id)}
              <option value={template.id}>{template.name}</option>
            {/each}
          </select>
        </label>
        {#if selectedTemplate}
          <div class="workspace-template-summary">
            <p data-workspace-template-description>{selectedTemplate.description}</p>
            <div class="workspace-template-tools" data-workspace-template-tools>
              {#each selectedTemplateTools as tool (tool.pluginId)}
                <div
                  class="workspace-template-tool"
                  class:limited={tool.status === 'limited'}
                  class:unavailable={tool.status === 'unavailable'}
                  data-workspace-template-tool={tool.pluginId}
                  data-template-tool-status={tool.status}
                >
                  <span class="workspace-template-tool-name">{tool.name}</span>
                  <span class="workspace-template-tool-tabs">{tool.tabs.length ? tr('workspaceTree.templateToolTabs', { tabs: tool.tabs.join(', ') }) : tr('workspaceTree.templateToolNoTabs')}</span>
                  <span class="workspace-template-tool-reason">{tool.reason}</span>
                </div>
              {/each}
            </div>
            {#if selectedTemplateIssues.length > 0}
              <p class="workspace-template-incomplete" data-workspace-template-incomplete>{tr('workspaceTree.templateWillBeIncomplete')}</p>
            {/if}
          </div>
        {/if}
        {#if createError}
          <p class="workspace-create-error" data-workspace-create-error role="alert">{createError}</p>
        {/if}
        <div class="workspace-create-actions">
          <button class="wt-btn-primary" on:click={doCreate} type="button" disabled={creating || templatesLoading || !selectedTemplate}>{creating ? tr('workspaceTree.creating') : tr('workspaceTree.create')}</button>
          <button class="wt-btn" on:click={closeCreateDialog} type="button" disabled={creating}>{tr('common.cancel')}</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .wt { display: flex; flex-direction: column; flex: 1; overflow: hidden; position: relative; }
  .wt-header { display: flex; align-items: center; justify-content: space-between; padding: 0.7rem 0.6rem 0.35rem; border-bottom: 1px solid var(--vt-color-border); flex-shrink: 0; }
  .wt-title { color: var(--vt-color-text-muted); font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 600; }
  .wt-list { min-height: 0; overflow-y: auto; padding: 0.2rem 0.6rem; }
  .wt-btn { min-height: 1.55rem; background: transparent; border: 1px solid transparent; color: var(--vt-color-text-muted); cursor: pointer; font-size: 0.78rem; padding: 0.12rem 0.38rem; border-radius: var(--vt-radius-sm); }
  .wt-btn:hover:not(:disabled) { color: var(--vt-color-accent); background: var(--vt-color-accent-muted); border-color: rgba(78,204,163,0.25); }
  .wt-btn-small { font-size: 0.7rem; opacity: 0; }
  .wt-always { opacity: 1; }
  .wt-row:hover .wt-btn-small { opacity: 1; }
  .wt-loading, .wt-error { padding: 0.5rem; font-size: 0.75rem; color: var(--vt-color-text-muted); }
  .wt-error { color: var(--vt-color-danger); }
  .wt-row { display: flex; align-items: center; gap: 0.45rem; padding: 0.18rem 0.45rem; min-height: 1.85rem; border-radius: var(--vt-radius-sm); }
  .wt-row:hover { background: var(--vt-color-surface-hover); }
  .wt-node.selected > .wt-row { background: var(--vt-color-surface-selected); box-shadow: inset 2px 0 0 var(--vt-color-accent); }
  .wt-icon { width: 0.95rem; height: 0.95rem; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; color: var(--vt-color-text-muted); }
  :global(.wt-node-icon) { display: block; }
  .wt-label { flex: 1; min-width: 0; min-height: 0; justify-content: flex-start; background: none; border: none; color: var(--vt-color-text-secondary); font-size: 0.78rem; text-align: left; cursor: pointer; padding: 0.1rem 0; border-radius: var(--vt-radius-sm); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .wt-label:hover { color: var(--vt-color-accent); }
  .wt-icon-btn { width: 1.45rem; height: 1.45rem; min-height: 0; padding: 0; border: 1px solid transparent; background: transparent; color: var(--vt-color-text-muted); opacity: 0.75; flex-shrink: 0; cursor: pointer; border-radius: var(--vt-radius-sm); }
  .wt-row:hover .wt-icon-btn { opacity: 1; }
  .wt-icon-btn:hover:not(:disabled) { color: var(--vt-color-accent); background: var(--vt-color-accent-muted); border-color: rgba(78,204,163,0.25); }
  .wt-icon-btn.danger:hover:not(:disabled) { color: var(--vt-color-danger); background: var(--vt-color-danger-muted); border-color: rgba(233,69,96,0.35); }
  .wt-rename { flex: 1; min-width: 0; background: #0f1424; border: 1px solid var(--vt-color-border-strong); color: var(--vt-color-text-primary); padding: 0.2rem 0.35rem; border-radius: var(--vt-radius-sm); font-size: 0.78rem; }
  .wt-rename:focus { outline: none; border-color: var(--vt-color-accent); box-shadow: var(--vt-focus-ring); }
  .workspace-create-overlay { position: fixed; inset: 0; z-index: 10000; display: flex; align-items: center; justify-content: center; padding: 1rem; background: rgba(4, 8, 18, 0.7); }
  .workspace-create-modal { width: min(34rem, 100%); display: grid; gap: 0.85rem; padding: 1rem; border: 1px solid var(--vt-color-border-strong); border-radius: var(--vt-radius-lg); background: var(--vt-color-surface); box-shadow: 0 18px 44px rgba(0, 0, 0, 0.38); }
  .workspace-create-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; }
  .workspace-create-header h2 { margin: 0; font-size: 1rem; }
  .workspace-create-field { display: grid; gap: 0.35rem; color: var(--vt-color-text-muted); font-size: 0.75rem; }
  .workspace-create-field input, .workspace-create-field select { width: 100%; min-height: 2rem; box-sizing: border-box; border: 1px solid var(--vt-color-border-strong); border-radius: var(--vt-radius-sm); background: #0f1424; color: var(--vt-color-text-primary); padding: 0.35rem 0.5rem; font: inherit; font-size: 0.84rem; }
  .workspace-create-field select { appearance: none; background-color: #0f1424; background-image: linear-gradient(45deg, transparent 50%, var(--vt-color-text-muted) 50%), linear-gradient(135deg, var(--vt-color-text-muted) 50%, transparent 50%); background-position: calc(100% - 14px) 50%, calc(100% - 9px) 50%; background-size: 5px 5px, 5px 5px; background-repeat: no-repeat; padding-right: 1.7rem; }
  .workspace-create-field select option { background: #0f1424; color: var(--vt-color-text-primary); }
  .workspace-create-field input:focus, .workspace-create-field select:focus { outline: none; border-color: var(--vt-color-accent); box-shadow: var(--vt-focus-ring); }
  .workspace-template-summary { display: grid; gap: 0.55rem; padding: 0.75rem; border: 1px solid var(--vt-color-border); border-radius: var(--vt-radius-md); background: var(--vt-color-surface-muted); }
  .workspace-template-summary p { margin: 0; color: var(--vt-color-text-secondary); font-size: 0.8rem; line-height: 1.45; }
  .workspace-template-tools { display: grid; gap: 0.4rem; }
  .workspace-template-tool { display: grid; grid-template-columns: minmax(7rem, 1fr) minmax(0, 1.4fr) minmax(0, 1.2fr); gap: 0.5rem; align-items: center; padding: 0.45rem 0.55rem; border: 1px solid var(--vt-color-border-strong); border-radius: var(--vt-radius-sm); color: var(--vt-color-text-secondary); font-size: 0.72rem; }
  .workspace-template-tool-name { color: var(--vt-color-text-primary); font-weight: 600; }
  .workspace-template-tool-tabs { color: var(--vt-color-text-muted); }
  .workspace-template-tool-reason { color: var(--vt-color-accent); }
  .workspace-template-tool.limited { border-color: rgba(220, 164, 57, 0.58); background: rgba(220, 164, 57, 0.08); }
  .workspace-template-tool.limited .workspace-template-tool-reason { color: #e8bc63; }
  .workspace-template-tool.unavailable { border-color: rgba(233, 69, 96, 0.58); background: var(--vt-color-danger-muted); }
  .workspace-template-tool.unavailable .workspace-template-tool-reason { color: var(--vt-color-danger); }
  .workspace-template-incomplete { color: var(--vt-color-danger) !important; }
  .workspace-template-warning { display: flex; gap: 0.5rem; align-items: flex-start; justify-content: space-between; margin: 0.25rem 0.6rem 0.5rem; padding: 0.6rem; border: 1px solid rgba(233, 69, 96, 0.58); border-radius: var(--vt-radius-md); background: var(--vt-color-danger-muted); color: var(--vt-color-text-secondary); font-size: 0.73rem; line-height: 1.4; }
  .workspace-template-warning strong { color: var(--vt-color-danger); }
  .workspace-template-warning-item { margin-top: 0.2rem; }
  @media (max-width: 700px) { .workspace-template-tool { grid-template-columns: 1fr; gap: 0.18rem; } }
  .workspace-create-error { margin: 0; color: var(--vt-color-danger); font-size: 0.78rem; line-height: 1.4; }
  .workspace-create-actions { display: flex; gap: 0.4rem; justify-content: flex-end; }
  .wt-btn-primary { background: var(--vt-color-accent); color: #101827; border: none; padding: 0.3rem 0.6rem; border-radius: var(--vt-radius-sm); cursor: pointer; font-size: 0.75rem; font-weight: 600; }
  .wt-btn-primary:hover:not(:disabled) { background: #3dbb92; }
  .wt-btn-primary:disabled, .wt-btn:disabled, .wt-icon-btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
