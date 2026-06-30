<script context="module">
  import { writable } from 'svelte/store';

  const activeWorkspaceId = writable('');
</script>

<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';

  let loading = true;
  let localError = '';
  let workspaces = [];
  let currentWorkspaceId = '';
  let showCreate = false;
  let newWorkspaceName = '';
  let creating = false;
  let renamingId = '';
  let renameValue = '';
  let busyId = '';

  onMount(() => {
    loadWorkspaces();
    window.addEventListener('verstak:workspace-active-changed', onActiveWorkspaceChanged);
  });

  onDestroy(() => {
    window.removeEventListener('verstak:workspace-active-changed', onActiveWorkspaceChanged);
  });

  function onActiveWorkspaceChanged(event) {
    currentWorkspaceId = event.detail?.workspaceName || '';
    activeWorkspaceId.set(currentWorkspaceId);
  }

  function resultOrError(response, fallbackValue) {
    return typeof response === 'string' ? [fallbackValue, response] : [response, ''];
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
        localError = err;
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
      localError = String(e);
    }
    loading = false;
  }

  async function selectWorkspace(workspace) {
    const id = wsName(workspace);
    const err = await App.SetCurrentWorkspace(id);
    if (err) {
      localError = err;
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
    if (!name) return;
    creating = true;
    localError = '';
    const [, err] = resultOrError(await App.CreateWorkspace(name, 'default'), null);
    if (err) {
      localError = err;
      creating = false;
      return;
    }
    showCreate = false;
    newWorkspaceName = '';
    creating = false;
    await loadWorkspaces();
    const created = workspaces.find((ws) => wsName(ws) === name);
    if (created) await selectWorkspace(created);
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
      localError = err;
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
      localError = err;
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
    <span class="wt-title">Workspaces</span>
    <button class="wt-btn" on:click={() => { showCreate = true; newWorkspaceName = ''; }} title="New workspace" type="button">+</button>
  </div>

  {#if loading}
    <div class="wt-loading">Loading...</div>
  {:else if localError}
    <div class="wt-error">{localError}</div>
  {/if}

  <div class="wt-list">
    {#each workspaces as workspace (wsName(workspace))}
      {@const id = wsName(workspace)}
      <div class="wt-node" class:selected={id === $activeWorkspaceId}>
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
            <button class="wt-btn wt-btn-small wt-always" on:click={() => commitRename(workspace)} title="Save rename" type="button" disabled={busyId === id}>OK</button>
            <button class="wt-btn wt-btn-small wt-always" on:click={cancelRename} title="Cancel rename" type="button" disabled={busyId === id}>x</button>
          {:else}
            <button class="wt-label" on:click={() => selectWorkspace(workspace)} type="button">{id}</button>
            <button class="wt-icon-btn" on:click={() => startRename(workspace)} title="Rename workspace" type="button" disabled={busyId === id}>
              <Icon name="edit" size={12} />
            </button>
            <button class="wt-icon-btn danger" on:click={() => trashWorkspace(workspace)} title="Trash workspace" type="button" disabled={busyId === id}>
              <Icon name="trash" size={12} />
            </button>
          {/if}
        </div>
      </div>
    {/each}
  </div>

  {#if showCreate}
    <div class="wt-create">
      <div class="wt-create-header">
        <span>New workspace</span>
        <button class="wt-btn" on:click={() => { showCreate = false; newWorkspaceName = ''; }} type="button">x</button>
      </div>
      <input type="text" bind:value={newWorkspaceName} placeholder="Name..." disabled={creating} />
      <div class="wt-create-actions">
        <button class="wt-btn-primary" on:click={doCreate} type="button" disabled={creating || !newWorkspaceName.trim()}>{creating ? '...' : 'Create'}</button>
        <button class="wt-btn" on:click={() => { showCreate = false; newWorkspaceName = ''; }} type="button" disabled={creating}>Cancel</button>
      </div>
    </div>
  {/if}
</div>

<style>
  .wt { display: flex; flex-direction: column; flex: 1; overflow: hidden; position: relative; }
  .wt-header { display: flex; align-items: center; justify-content: space-between; padding: 0.7rem 0.6rem 0.35rem; border-bottom: 1px solid #0f3460; flex-shrink: 0; }
  .wt-title { color: #a0a0b8; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 600; }
  .wt-list { min-height: 0; overflow-y: auto; padding: 0.2rem 0.6rem; }
  .wt-btn { min-height: 0; background: none; border: none; color: #666; cursor: pointer; font-size: 0.85rem; padding: 0.1rem 0.3rem; border-radius: 3px; }
  .wt-btn:hover:not(:disabled) { color: #4ecca3; background: rgba(78,204,163,0.1); }
  .wt-btn-small { font-size: 0.68rem; opacity: 0; }
  .wt-always { opacity: 1; }
  .wt-row:hover .wt-btn-small { opacity: 1; }
  .wt-loading, .wt-error { padding: 0.5rem; font-size: 0.75rem; color: #666; }
  .wt-error { color: #e94560; }
  .wt-row { display: flex; align-items: center; gap: 0.45rem; padding: 0.15rem 0.45rem; min-height: 1.7rem; border-radius: 3px; }
  .wt-row:hover { background: rgba(15,52,96,0.4); }
  .wt-node.selected > .wt-row { background: rgba(78,204,163,0.1); }
  .wt-icon { width: 0.9rem; height: 0.9rem; display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; color: #a0a0b8; }
  :global(.wt-node-icon) { display: block; }
  .wt-label { flex: 1; min-width: 0; min-height: 0; justify-content: flex-start; background: none; border: none; color: #a0a0b8; font-size: 0.78rem; text-align: left; cursor: pointer; padding: 0.1rem 0; border-radius: 3px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .wt-label:hover { color: #4ecca3; }
  .wt-icon-btn { width: 1.25rem; height: 1.25rem; min-height: 0; padding: 0; border: none; background: transparent; color: #666; opacity: 0; flex-shrink: 0; cursor: pointer; border-radius: 3px; }
  .wt-row:hover .wt-icon-btn { opacity: 1; }
  .wt-icon-btn:hover:not(:disabled) { color: #4ecca3; background: rgba(78,204,163,0.1); }
  .wt-icon-btn.danger:hover:not(:disabled) { color: #e94560; background: rgba(233,69,96,0.12); }
  .wt-rename { flex: 1; min-width: 0; background: #0f3460; border: 1px solid #1a3a5c; color: #e0e0f0; padding: 0.2rem 0.35rem; border-radius: 4px; font-size: 0.78rem; }
  .wt-rename:focus { outline: none; border-color: #4ecca3; }
  .wt-create { position: absolute; bottom: 0; left: 0; right: 0; background: #16213e; border-top: 1px solid #0f3460; padding: 0.6rem; z-index: 10; }
  .wt-create-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.4rem; color: #a0a0b8; font-size: 0.7rem; text-transform: uppercase; }
  .wt-create input { width: 100%; background: #0f3460; border: 1px solid #1a3a5c; color: #e0e0f0; padding: 0.35rem 0.5rem; border-radius: 4px; font-size: 0.8rem; margin-bottom: 0.4rem; box-sizing: border-box; }
  .wt-create input:focus { outline: none; border-color: #4ecca3; }
  .wt-create-actions { display: flex; gap: 0.4rem; justify-content: flex-end; }
  .wt-btn-primary { background: #4ecca3; color: #1a1a2e; border: none; padding: 0.3rem 0.6rem; border-radius: 4px; cursor: pointer; font-size: 0.75rem; font-weight: 600; }
  .wt-btn-primary:hover:not(:disabled) { background: #3dbb92; }
  .wt-btn-primary:disabled, .wt-btn:disabled, .wt-icon-btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
