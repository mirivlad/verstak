<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import Modal from '../ui/Modal.svelte';
  import Select from '../ui/Select.svelte';
  import { i18n } from '../i18n/index.js';

  // ── State ──────────────────────────────────────────────────────────────────
  let tree = { roots: [], currentWorkspaceId: '', revision: 0 };
  let loading = true;
  let error = '';
  let expandedIds = {};
  let activeWid = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;
  $: tr = ((al) => (k, p, f) => { void al; return i18n.t(k, p, f); })(locale);

  // ── Modal state ────────────────────────────────────────────────────────────
  let modal = null; // { type, folderId?, workspaceId?, parentId?, name?, templateId? }
  let formName = '';
  let formParentId = '';
  let formTemplateId = 'default';
  let formError = '';
  let formBusy = false;
  let templates = [];
  let templateNames = {};

  // ── Context menu ───────────────────────────────────────────────────────────
  let ctxMenu = null; // { x, y, kind, id, name, parentId }

  // ── Keyboard ───────────────────────────────────────────────────────────────
  let focusedKey = '';

  // ── Load ───────────────────────────────────────────────────────────────────
  onMount(async () => {
    unsubscribeLocale = i18n.subscribe((l) => { locale = l; });
    await loadTree();
    await loadTemplates();
    window.addEventListener('verstak:workspace-tree-changed', onTreeChanged);
  });
  onDestroy(() => {
    if (unsubscribeLocale) unsubscribeLocale();
    window.removeEventListener('verstak:workspace-tree-changed', onTreeChanged);
  });

  async function loadTree() {
    loading = true;
    try {
      const raw = await App.GetWorkspaceTreeV2();
      if (raw?.error) { error = raw.error; return; }
      tree = raw || { roots: [], currentWorkspaceId: '', revision: 0 };
      activeWid = tree.currentWorkspaceId || '';
      if (activeWid) ensureExpandedToWorkspace(activeWid);
    } catch (e) { error = tr('workspaceTree.loadError'); }
    loading = false;
  }

  async function loadTemplates() {
    try {
      const [tlist] = (await App.ListWorkspaceTemplates()) || [[]];
      templates = Array.isArray(tlist) ? tlist : [];
      const plugins = (await App.GetPlugins()) || [];
      templateNames = {};
      for (const p of plugins) {
        if (p?.manifest?.id && p?.manifest?.name) templateNames[p.manifest.id] = p.manifest.name;
      }
    } catch { templates = []; }
  }

  function onTreeChanged() { loadTree(); }

  function ensureExpandedToWorkspace(wid) {
    for (const root of tree.roots || []) {
      if (expandToChild(root, wid)) break;
    }
  }
  function expandToChild(node, targetWid) {
    if (node.kind === 'workspace' && node.id === targetWid) return true;
    if (node.children) {
      for (const child of node.children) {
        if (expandToChild(child, targetWid)) {
          expandedIds[node.key] = true;
          expandedIds = expandedIds;
          return true;
        }
      }
    }
    return false;
  }

  // ── Actions ────────────────────────────────────────────────────────────────
  function toggleExpand(key) {
    expandedIds[key] = !expandedIds[key];
    expandedIds = expandedIds;
  }

  async function selectWorkspace(wid) {
    const err = await App.SetCurrentWorkspaceV2(wid);
    if (err) { error = err; return; }
    activeWid = wid;
    window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
      detail: { workspaceId: wid }
    }));
  }

  function openCreateFolder(parentId) {
    modal = { type: 'create-folder', parentId };
    formName = '';
    formParentId = parentId || '';
    formError = '';
    formBusy = false;
  }
  function openCreateWorkspace(parentId) {
    modal = { type: 'create-workspace', parentId };
    formName = '';
    formParentId = parentId || '';
    formTemplateId = templates[0]?.id || 'default';
    formError = '';
    formBusy = false;
  }
  function openRename(kind, id, name) {
    modal = { type: 'rename', kind, id };
    formName = name;
    formError = '';
    formBusy = false;
  }
  function openMove(kind, id, name) {
    modal = { type: 'move', kind, id, name };
    formParentId = '';
    formError = '';
    formBusy = false;
  }
  function openTrash(kind, id, name) {
    modal = { type: 'trash', kind, id, name };
    formBusy = false;
  }
  function closeModal() { if (!formBusy) modal = null; }

  async function doCreateFolder() {
    const name = formName.trim();
    if (!name) { formError = tr('workspaceTree.nameRequired'); return; }
    formBusy = true;
    const result = await App.CreateFolderV2(formParentId || '', name);
    if (result?.error) { formError = result.error; formBusy = false; return; }
    if (formParentId) expandedIds['folder:' + formParentId] = true;
    modal = null;
    await loadTree();
  }

  async function doCreateWorkspace() {
    const name = formName.trim();
    if (!name) { formError = tr('workspaceTree.nameRequired'); return; }
    formBusy = true;
    const result = await App.CreateWorkspaceV2(formParentId || '', name, formTemplateId);
    if (result?.error) { formError = result.error; formBusy = false; return; }
    if (formParentId) expandedIds['folder:' + formParentId] = true;
    const wid = result?.id;
    modal = null;
    await loadTree();
    if (wid) await selectWorkspace(wid);
  }

  async function doRename() {
    const name = formName.trim();
    if (!name) { formError = tr('workspaceTree.nameRequired'); return; }
    formBusy = true;
    let err = '';
    if (modal.kind === 'folder') err = await App.RenameFolderV2(modal.id, name);
    else err = await App.RenameWorkspaceV2(modal.id, name);
    if (err) { formError = err; formBusy = false; return; }
    modal = null;
    await loadTree();
  }

  async function doMove() {
    formBusy = true;
    let err = '';
    if (modal.kind === 'folder') err = await App.MoveFolderV2(modal.id, formParentId || '');
    else err = await App.MoveWorkspaceV2(modal.id, formParentId || '');
    if (err) { formError = err; formBusy = false; return; }
    modal = null;
    await loadTree();
  }

  async function doTrash() {
    formBusy = true;
    if (modal.kind === 'folder') await App.TrashFolderV2(modal.id);
    else {
      await App.TrashWorkspaceV2(modal.id);
      if (activeWid === modal.id) activeWid = '';
    }
    modal = null;
    await loadTree();
  }

  // ── Context menu ───────────────────────────────────────────────────────────
  function onContextMenu(e, kind, id, name, parentId) {
    e.preventDefault();
    ctxMenu = { x: e.clientX, y: e.clientY, kind, id, name, parentId };
  }
  function closeCtxMenu() { ctxMenu = null; }

  function flatFolders(roots, out = []) {
    for (const r of roots || []) {
      if (r.kind === 'folder') { out.push(r); flatFolders(r.children, out); }
    }
    return out;
  }

  function subtreeCounts(id) {
    let folders = 0, workspaces = 0;
    const node = findNode(tree.roots || [], id);
    if (!node) return { folders: 0, workspaces: 0 };
    countChildren(node);
    return { folders, workspaces };
    function countChildren(n) {
      for (const c of n.children || []) {
        if (c.kind === 'folder') { folders++; countChildren(c); }
        else workspaces++;
      }
    }
  }
  function findNode(nodes, id) {
    for (const n of nodes) {
      if (n.id === id) return n;
      const found = findNode(n.children || [], id);
      if (found) return found;
    }
    return null;
  }

  function onKeyDown(e) {
    if (e.key === 'Escape') { closeCtxMenu(); closeModal(); }
  }

  function folderPathForId(id) {
    const f = flatFolders(tree.roots || []).find(x => x.id === id);
    return f ? f.path : id;
  }
</script>

<svelte:window on:keydown={onKeyDown} on:click={closeCtxMenu} />

<div class="wt" data-workspace-tree>
  <div class="wt-header">
    <span class="wt-title">{tr('workspaceTree.title')}</span>
    <div class="wt-header-actions">
      <button class="vt-icon-btn" on:click={() => openCreateWorkspace('')} title={tr('workspaceTree.newDeal')} aria-label={tr('workspaceTree.newDeal')} type="button">
        <Icon name="space" size={14} />
      </button>
      <button class="vt-icon-btn" on:click={() => openCreateFolder('')} title={tr('workspaceTree.newFolder')} aria-label={tr('workspaceTree.newFolder')} type="button">
        <Icon name="folder" size={14} />
      </button>
    </div>
  </div>

  <div class="wt-list" role="tree" aria-label={tr('workspaceTree.title')}>
    {#if loading}
      <div class="wt-status">{tr('common.loading')}</div>
    {:else if error}
      <div class="wt-status wt-error">{error}</div>
    {:else if !tree.roots || tree.roots.length === 0}
      <div class="wt-empty">
        <p>{tr('workspaceTree.emptyTitle')}</p>
        <p class="wt-empty-hint">{tr('workspaceTree.emptyHint')}</p>
      </div>
    {:else}
      {#each tree.roots as node (node.key)}
        {@const isExpanded = expandedIds[node.key] || false}
        {@const isActive = node.kind === 'workspace' && node.id === activeWid}
        {@const isFolder = node.kind === 'folder'}
        {@const hasChildren = isFolder && node.children && node.children.length > 0}
        <div class="wt-node" style="padding-left:0">
          <div class="wt-row" class:selected={isActive} class:focus={focusedKey === node.key}
            on:click={() => isFolder ? toggleExpand(node.key) : selectWorkspace(node.id)}
            on:contextmenu={(e) => onContextMenu(e, node.kind, node.id, node.name, node.parentFolderId)}
            role="treeitem" aria-expanded={isFolder ? isExpanded : undefined} aria-selected={isActive || undefined}
            tabindex="0"
          >
            {#if isFolder}
              <span class="wt-chevron" class:open={isExpanded}><Icon name="chevron-right" size={12} /></span>
            {:else}
              <span class="wt-chevron wt-chevron-empty" />
            {/if}
            <span class="wt-node-icon"><Icon name={isFolder ? 'folder' : 'space'} size={14} /></span>
            <span class="wt-node-name" title={node.name}>{node.name}</span>
            <span class="wt-node-actions">
              {#if isFolder}
                <button class="vt-icon-btn vt-icon-btn-sm" on:click|stopPropagation={() => openCreateWorkspace(node.id)} title={tr('workspaceTree.newDeal')}><Icon name="plus" size={11} /></button>
                <button class="vt-icon-btn vt-icon-btn-sm" on:click|stopPropagation={() => openCreateFolder(node.id)} title={tr('workspaceTree.newFolder')}><Icon name="folder-plus" size={11} /></button>
              {/if}
            </span>
          </div>
          {#if isFolder && isExpanded && hasChildren}
            {#each node.children as child (child.key)}
              {@const childExpanded = expandedIds[child.key] || false}
              <div class="wt-node" style="padding-left:1.2rem">
                <div class="wt-row" class:selected={child.kind === 'workspace' && child.id === activeWid}
                  on:click={() => child.kind === 'folder' ? toggleExpand(child.key) : selectWorkspace(child.id)}
                  on:contextmenu={(e) => onContextMenu(e, child.kind, child.id, child.name, node.id)}
                  role="treeitem" aria-expanded={child.kind === 'folder' ? childExpanded : undefined}
                  tabindex="0"
                >
                  {#if child.kind === 'folder'}
                    <span class="wt-chevron" class:open={childExpanded}><Icon name="chevron-right" size={12} /></span>
                  {:else}
                    <span class="wt-chevron wt-chevron-empty" />
                  {/if}
                  <span class="wt-node-icon"><Icon name={child.kind === 'folder' ? 'folder' : 'space'} size={14} /></span>
                  <span class="wt-node-name" title={child.name}>{child.name}</span>
                  <span class="wt-node-actions">
                    {#if child.kind === 'folder'}
                      <button class="vt-icon-btn vt-icon-btn-sm" on:click|stopPropagation={() => openCreateWorkspace(child.id)} title={tr('workspaceTree.newDeal')}><Icon name="plus" size={11} /></button>
                      <button class="vt-icon-btn vt-icon-btn-sm" on:click|stopPropagation={() => openCreateFolder(child.id)} title={tr('workspaceTree.newFolder')}><Icon name="folder-plus" size={11} /></button>
                    {/if}
                  </span>
                </div>
              </div>
            {/each}
          {/if}
        </div>
      {/each}
    {/if}
  </div>
</div>

<!-- Context Menu -->
{#if ctxMenu}
  <div class="vt-ctx-menu" style="left:{ctxMenu.x}px;top:{ctxMenu.y}px" on:click|stopPropagation>
    {#if ctxMenu.kind === 'folder'}
      <button class="vt-ctx-item" on:click={() => { closeCtxMenu(); openCreateWorkspace(ctxMenu.id); }}>{tr('workspaceTree.newDeal')}</button>
      <button class="vt-ctx-item" on:click={() => { closeCtxMenu(); openCreateFolder(ctxMenu.id); }}>{tr('workspaceTree.newFolder')}</button>
      <div class="vt-ctx-sep" />
      <button class="vt-ctx-item" on:click={() => { closeCtxMenu(); openRename('folder', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.rename')}</button>
      <button class="vt-ctx-item" on:click={() => { closeCtxMenu(); openMove('folder', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.move')}</button>
      <div class="vt-ctx-sep" />
      <button class="vt-ctx-item vt-ctx-danger" on:click={() => { closeCtxMenu(); openTrash('folder', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.trash')}</button>
    {:else}
      <button class="vt-ctx-item" on:click={() => { closeCtxMenu(); selectWorkspace(ctxMenu.id); }}>{tr('workspaceTree.open')}</button>
      <button class="vt-ctx-item" on:click={() => { closeCtxMenu(); openRename('workspace', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.rename')}</button>
      <button class="vt-ctx-item" on:click={() => { closeCtxMenu(); openMove('workspace', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.move')}</button>
      <div class="vt-ctx-sep" />
      <button class="vt-ctx-item vt-ctx-danger" on:click={() => { closeCtxMenu(); openTrash('workspace', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.trash')}</button>
    {/if}
  </div>
{/if}

<!-- Modals -->
<Modal title={tr('workspaceTree.newFolder')} show={modal?.type === 'create-folder'} on:close={closeModal}>
  <label class="vt-field">
    <span>{tr('workspaceTree.location')}</span>
    <Select options={flatFolders(tree.roots || []).map(f => ({ value: f.id, label: f.path }))} placeholder={tr('workspaceTree.root')} bind:value={formParentId} labelKey="label" valueKey="value" />
  </label>
  <label class="vt-field">
    <span>{tr('workspaceTree.name')}</span>
    <input class="vt-input" type="text" bind:value={formName} placeholder={tr('workspaceTree.namePlaceholder')} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doCreateFolder()} />
  </label>
  {#if formError}<p class="vt-form-error">{formError}</p>{/if}
  <svelte:fragment slot="actions">
    <button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button>
    <button class="vt-btn-primary" on:click={doCreateFolder} disabled={formBusy}>{tr('common.create')}</button>
  </svelte:fragment>
</Modal>

<Modal title={tr('workspaceTree.newDeal')} show={modal?.type === 'create-workspace'} on:close={closeModal} wide>
  <label class="vt-field">
    <span>{tr('workspaceTree.location')}</span>
    <Select options={flatFolders(tree.roots || []).map(f => ({ value: f.id, label: f.path }))} placeholder={tr('workspaceTree.root')} bind:value={formParentId} labelKey="label" valueKey="value" />
  </label>
  <label class="vt-field">
    <span>{tr('workspaceTree.name')}</span>
    <input class="vt-input" type="text" bind:value={formName} placeholder={tr('workspaceTree.namePlaceholder')} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doCreateWorkspace()} />
  </label>
  <label class="vt-field">
    <span>{tr('workspaceTree.template')}</span>
    <Select options={templates} bind:value={formTemplateId} labelKey="name" valueKey="id" />
  </label>
  {#if formError}<p class="vt-form-error">{formError}</p>{/if}
  <svelte:fragment slot="actions">
    <button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button>
    <button class="vt-btn-primary" on:click={doCreateWorkspace} disabled={formBusy}>{tr('common.create')}</button>
  </svelte:fragment>
</Modal>

<Modal title={tr('workspaceTree.rename')} show={modal?.type === 'rename'} on:close={closeModal}>
  <label class="vt-field">
    <span>{tr('workspaceTree.newName')}</span>
    <input class="vt-input" type="text" bind:value={formName} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doRename()} />
  </label>
  {#if formError}<p class="vt-form-error">{formError}</p>{/if}
  <svelte:fragment slot="actions">
    <button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button>
    <button class="vt-btn-primary" on:click={doRename} disabled={formBusy}>{tr('common.save')}</button>
  </svelte:fragment>
</Modal>

<Modal title={tr('workspaceTree.move') + (modal?.name ? ' «' + modal.name + '»' : '')} show={modal?.type === 'move'} on:close={closeModal}>
  <label class="vt-field">
    <span>{tr('workspaceTree.newLocation')}</span>
    <Select options={flatFolders(tree.roots || []).filter(f => f.id !== modal?.id).map(f => ({ value: f.id, label: f.path }))} placeholder={tr('workspaceTree.root')} bind:value={formParentId} labelKey="label" valueKey="value" />
  </label>
  {#if formError}<p class="vt-form-error">{formError}</p>{/if}
  <svelte:fragment slot="actions">
    <button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button>
    <button class="vt-btn-primary" on:click={doMove} disabled={formBusy}>{tr('workspaceTree.move')}</button>
  </svelte:fragment>
</Modal>

<Modal title={(modal?.kind === 'folder' ? tr('workspaceTree.trashFolder') : tr('workspaceTree.trashDeal')) + (modal?.name ? ' «' + modal.name + '»?' : '?')} show={modal?.type === 'trash'} on:close={closeModal}>
  <p style="color:var(--vt-color-text-secondary);font-size:0.84rem;margin:0;">
    {#if modal?.kind === 'folder'}
      {@const c = subtreeCounts(modal.id)}
      {tr('workspaceTree.trashFolderDesc')}
      <br />{tr('workspaceTree.contains')}: {c.folders} {tr('workspaceTree.nestedFolders')}, {c.workspaces} {tr('workspaceTree.title')}
    {:else}
      {tr('workspaceTree.trashDealDesc')}
    {/if}
  </p>
  <svelte:fragment slot="actions">
    <button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button>
    <button class="vt-btn-danger" on:click={doTrash} disabled={formBusy}>{tr('workspaceTree.toTrash')}</button>
  </svelte:fragment>
</Modal>

<style>
  .wt { display: flex; flex-direction: column; flex: 1; overflow: hidden; }
  .wt-header { display: flex; align-items: center; justify-content: space-between; padding: 0.7rem 0.6rem 0.35rem; border-bottom: 1px solid var(--vt-color-border); flex-shrink: 0; }
  .wt-title { color: var(--vt-color-text-muted); font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 600; }
  .wt-header-actions { display: flex; gap: 0.2rem; }
  .wt-list { min-height: 0; overflow-y: auto; padding: 0.2rem 0.4rem; flex: 1; }
  .wt-status { padding: 0.5rem; font-size: 0.78rem; color: var(--vt-color-text-muted); }
  .wt-error { color: var(--vt-color-danger); }
  .wt-empty { padding: 1rem 0.5rem; text-align: center; color: var(--vt-color-text-muted); font-size: 0.8rem; }
  .wt-empty-hint { font-size: 0.72rem; opacity: 0.7; }

  /* Icon buttons */
  .vt-icon-btn { width: 1.6rem; height: 1.6rem; min-height: 0; padding: 0; border: 1px solid transparent; background: transparent; color: var(--vt-color-text-muted); cursor: pointer; border-radius: var(--vt-radius-sm); display: inline-flex; align-items: center; justify-content: center; }
  .vt-icon-btn:hover { color: var(--vt-color-accent); background: var(--vt-color-accent-muted); border-color: rgba(78,204,163,0.25); }

  /* Buttons */
  .vt-btn { min-height: 1.8rem; background: transparent; border: 1px solid var(--vt-color-border-strong); color: var(--vt-color-text-secondary); cursor: pointer; font-size: 0.78rem; padding: 0.3rem 0.6rem; border-radius: var(--vt-radius-sm); }
  .vt-btn:hover:not(:disabled) { color: var(--vt-color-text-primary); border-color: var(--vt-color-text-muted); }
  .vt-btn-primary { min-height: 1.8rem; background: var(--vt-color-accent); color: #101827; border: none; padding: 0.3rem 0.7rem; border-radius: var(--vt-radius-sm); cursor: pointer; font-size: 0.78rem; font-weight: 600; }
  .vt-btn-primary:hover:not(:disabled) { background: #3dbb92; }
  .vt-btn-danger { min-height: 1.8rem; background: var(--vt-color-danger); color: #fff; border: none; padding: 0.3rem 0.7rem; border-radius: var(--vt-radius-sm); cursor: pointer; font-size: 0.78rem; font-weight: 600; }
  .vt-btn-danger:hover:not(:disabled) { background: #d63851; }
  .vt-btn:disabled, .vt-btn-primary:disabled, .vt-btn-danger:disabled { opacity: 0.5; cursor: not-allowed; }

  /* Form */
  .vt-field { display: grid; gap: 0.35rem; color: var(--vt-color-text-muted); font-size: 0.75rem; }
  .vt-input { width: 100%; min-height: 2rem; box-sizing: border-box; border: 1px solid var(--vt-color-border-strong); border-radius: var(--vt-radius-sm); background: #0f1424; color: var(--vt-color-text-primary); padding: 0.35rem 0.5rem; font: inherit; font-size: 0.84rem; }
  .vt-input:focus { outline: none; border-color: var(--vt-color-accent); box-shadow: var(--vt-focus-ring); }
  .vt-form-error { margin: 0; color: var(--vt-color-danger); font-size: 0.78rem; line-height: 1.4; }

  /* Context menu */
  .vt-ctx-menu { position: fixed; z-index: 10001; min-width: 10rem; background: var(--vt-color-surface); border: 1px solid var(--vt-color-border-strong); border-radius: var(--vt-radius-md); box-shadow: 0 8px 24px rgba(0,0,0,0.3); padding: 0.25rem; }
  .vt-ctx-item { display: block; width: 100%; text-align: left; padding: 0.3rem 0.6rem; background: none; border: none; color: var(--vt-color-text-secondary); font-size: 0.78rem; cursor: pointer; border-radius: var(--vt-radius-sm); }
  .vt-ctx-item:hover { background: var(--vt-color-surface-hover); color: var(--vt-color-text-primary); }
  .vt-ctx-sep { height: 1px; background: var(--vt-color-border); margin: 0.2rem 0.3rem; }
  .vt-ctx-danger { color: var(--vt-color-danger); }
  .vt-ctx-danger:hover { background: var(--vt-color-danger-muted); }

  /* Tree nodes */
  .wt-node { user-select: none; }
  .wt-row { display: flex; align-items: center; gap: 0.3rem; padding: 0.18rem 0.45rem; min-height: 1.85rem; border-radius: var(--vt-radius-sm); cursor: pointer; transition: background 0.1s; }
  .wt-row:hover { background: var(--vt-color-surface-hover); }
  .wt-row.selected { background: var(--vt-color-surface-selected); box-shadow: inset 2px 0 0 var(--vt-color-accent); color: var(--vt-color-text-primary); }
  .wt-row.focus { outline: 1px solid var(--vt-color-accent); outline-offset: -1px; }
  .wt-chevron { width: 0.9rem; height: 0.9rem; display: flex; align-items: center; justify-content: center; flex-shrink: 0; color: var(--vt-color-text-muted); transition: transform 0.15s; }
  .wt-chevron.open { transform: rotate(90deg); }
  .wt-chevron-empty { visibility: hidden; }
  .wt-node-icon { width: 1rem; height: 1rem; display: flex; align-items: center; justify-content: center; flex-shrink: 0; color: var(--vt-color-text-muted); }
  .wt-node-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.78rem; color: var(--vt-color-text-secondary); }
  .wt-row.selected .wt-node-name { color: var(--vt-color-text-primary); }
  .wt-node-actions { display: flex; gap: 0.1rem; opacity: 0; transition: opacity 0.1s; }
  .wt-row:hover .wt-node-actions, .wt-row:focus-within .wt-node-actions { opacity: 1; }
  .vt-icon-btn-sm { width: 1.3rem; height: 1.3rem; }
</style>
