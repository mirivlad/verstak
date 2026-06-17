<script>
  import { onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';

  let nodes = [];
  let currentNodeId = '';
  let loading = true;
  let error = '';
  let expandedNodes = {};
  let showCreate = false;
  let newNodeTitle = '';
  let newNodeParentId = '';
  let newNodeType = 'case';
  let creating = false;

  onMount(async () => {
    await loadTree();
  });

  async function loadTree() {
    loading = true;
    error = '';
    try {
      const result = await App.GetWorkspaceTree();
      if (result.status === 'not initialized') {
        nodes = [];
        currentNodeId = '';
      } else {
        nodes = result.nodes || [];
        currentNodeId = result.currentNodeId || '';
        const root = nodes.find(n => !n.parentId);
        if (root) expandedNodes[root.id] = true;
      }
    } catch (e) {
      error = String(e);
    }
    loading = false;
  }

  function childrenOf(parentId) {
    return nodes.filter(n => n.parentId === parentId).sort((a, b) => a.order - b.order);
  }

  function roots() {
    return nodes.filter(n => !n.parentId).sort((a, b) => a.order - b.order);
  }

  function toggle(id) {
    expandedNodes[id] = !expandedNodes[id];
    expandedNodes = expandedNodes;
  }

  function hasKids(id) {
    return nodes.some(n => n.parentId === id);
  }

  function icon(type) {
    if (type === 'space') return '\u{1F310}';
    if (type === 'case') return '\u{1F4CB}';
    if (type === 'folder') return '\u{1F4C1}';
    return '\u{1F4C4}';
  }

  async function selectNode(id) {
    const err = await App.SetCurrentWorkspaceNode(id);
    if (err) { error = err; return; }
    currentNodeId = id;
  }

  function openCreate(parentId, type) {
    newNodeParentId = parentId;
    newNodeType = type;
    newNodeTitle = '';
    showCreate = true;
  }

  async function doCreate() {
    if (!newNodeTitle.trim()) return;
    creating = true;
    const res = await App.CreateWorkspaceNode(newNodeParentId, newNodeType, newNodeTitle.trim());
    if (res.error) { error = res.error; creating = false; return; }
    showCreate = false;
    creating = false;
    await loadTree();
    expandedNodes[newNodeParentId] = true;
    expandedNodes = expandedNodes;
  }

  function cancelCreate() {
    showCreate = false;
    newNodeTitle = '';
  }
</script>

<div class="wt">
  <div class="wt-header">
    <span class="wt-title">Workspace</span>
    <button class="wt-btn" on:click={() => openCreate('', 'space')} type="button">+</button>
  </div>

  {#if loading}
    <div class="wt-loading">Loading...</div>
  {:else if error}
    <div class="wt-error">{error}</div>
  {:else}
    {#each roots() as node (node.id)}
      <div class="wt-node">
        <div class="wt-row" class:selected={node.id === currentNodeId}>
          {#if hasKids(node.id)}
            <button class="wt-expand" on:click={() => toggle(node.id)} type="button">{expandedNodes[node.id] ? '\u25BE' : '\u25B8'}</button>
          {:else}
            <span class="wt-expand-spacer"></span>
          {/if}
          <span class="wt-icon">{icon(node.type)}</span>
          <button class="wt-label" on:click={() => selectNode(node.id)} type="button">{node.title}</button>
          <button class="wt-btn wt-btn-small" on:click={() => openCreate(node.id, 'case')} type="button">+</button>
        </div>
        {#if expandedNodes[node.id]}
          {#each childrenOf(node.id) as child (child.id)}
            <div class="wt-node wt-child">
              <div class="wt-row" class:selected={child.id === currentNodeId}>
                {#if hasKids(child.id)}
                  <button class="wt-expand" on:click={() => toggle(child.id)} type="button">{expandedNodes[child.id] ? '\u25BE' : '\u25B8'}</button>
                {:else}
                  <span class="wt-expand-spacer"></span>
                {/if}
                <span class="wt-icon">{icon(child.type)}</span>
                <button class="wt-label" on:click={() => selectNode(child.id)} type="button">{child.title}</button>
                <button class="wt-btn wt-btn-small" on:click={() => openCreate(child.id, 'folder')} type="button">+</button>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    {/each}
  {/if}

  {#if showCreate}
    <div class="wt-create">
      <div class="wt-create-header">
        <span>New {newNodeType}</span>
        <button class="wt-btn" on:click={cancelCreate} type="button">{'\u2715'}</button>
      </div>
      <input type="text" bind:value={newNodeTitle} placeholder="Name..." disabled={creating} />
      <div class="wt-create-actions">
        <button class="wt-btn-primary" on:click={doCreate} type="button" disabled={creating || !newNodeTitle.trim()}>{creating ? '...' : 'Create'}</button>
        <button class="wt-btn" on:click={cancelCreate} type="button" disabled={creating}>Cancel</button>
      </div>
    </div>
  {/if}
</div>

<style>
  .wt { display: flex; flex-direction: column; flex: 1; overflow: hidden; position: relative; }
  .wt-header { display: flex; align-items: center; justify-content: space-between; padding: 0.4rem 0.6rem; border-bottom: 1px solid #0f3460; }
  .wt-title { color: #a0a0b8; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 600; }
  .wt-btn { background: none; border: none; color: #666; cursor: pointer; font-size: 0.85rem; padding: 0.1rem 0.3rem; border-radius: 3px; }
  .wt-btn:hover { color: #4ecca3; background: rgba(78,204,163,0.1); }
  .wt-btn-small { font-size: 0.7rem; opacity: 0; }
  .wt-row:hover .wt-btn-small { opacity: 1; }
  .wt-loading, .wt-error { padding: 0.5rem; font-size: 0.75rem; color: #666; }
  .wt-error { color: #e94560; }
  .wt-node { }
  .wt-row { display: flex; align-items: center; gap: 0.2rem; padding: 0.2rem 0.4rem; }
  .wt-row:hover { background: rgba(15,52,96,0.4); }
  .wt-row.selected { background: rgba(78,204,163,0.1); }
  .wt-expand { width: 1rem; height: 1rem; display: flex; align-items: center; justify-content: center; font-size: 0.65rem; color: #666; background: none; border: none; cursor: pointer; padding: 0; }
  .wt-expand:hover { color: #e0e0f0; }
  .wt-expand-spacer { width: 1rem; flex-shrink: 0; }
  .wt-icon { font-size: 0.8rem; flex-shrink: 0; }
  .wt-label { flex: 1; background: none; border: none; color: #e0e0f0; font-size: 0.78rem; text-align: left; cursor: pointer; padding: 0.1rem 0.2rem; border-radius: 3px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .wt-label:hover { color: #4ecca3; }
  .wt-child .wt-row { padding-left: 1.2rem; }
  .wt-create { position: absolute; bottom: 0; left: 0; right: 0; background: #16213e; border-top: 1px solid #0f3460; padding: 0.6rem; z-index: 10; }
  .wt-create-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.4rem; color: #a0a0b8; font-size: 0.7rem; text-transform: uppercase; }
  .wt-create input { width: 100%; background: #0f3460; border: 1px solid #1a3a5c; color: #e0e0f0; padding: 0.35rem 0.5rem; border-radius: 4px; font-size: 0.8rem; margin-bottom: 0.4rem; box-sizing: border-box; }
  .wt-create input:focus { outline: none; border-color: #4ecca3; }
  .wt-create-actions { display: flex; gap: 0.4rem; justify-content: flex-end; }
  .wt-btn-primary { background: #4ecca3; color: #1a1a2e; border: none; padding: 0.3rem 0.6rem; border-radius: 4px; cursor: pointer; font-size: 0.75rem; font-weight: 600; }
  .wt-btn-primary:hover:not(:disabled) { background: #3dbb92; }
  .wt-btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
