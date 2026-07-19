<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import Modal from '../ui/Modal.svelte';
  import Select from '../ui/Select.svelte';
  import TreeNode from './TreeNode.svelte';
  import { i18n } from '../i18n/index.js';

  let tree = { roots: [], currentWorkspaceId: '', revision: 0 };
  let loading = true; let error = '';
  let expandedIds = {}; let activeWid = '';
  let focusedKey = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;
  $: tr = ((al) => (k, p, f) => { void al; return i18n.t(k, p, f); })(locale);

  // Modal state
  let modal = null; let formName = ''; let formParentId = ''; let formTemplateId = 'default';
  let formError = ''; let formBusy = false;
  let templates = [];
  let ctxMenu = null;

  // Drag state
  let dragOverRoot = false;
  let dragOverFolderId = '';

  onMount(async () => {
    unsubscribeLocale = i18n.subscribe((l) => { locale = l; });
    await loadTree(); await loadTemplates();
    window.addEventListener('verstak:workspace-tree-changed', loadTree);
  });
  onDestroy(() => {
    if (unsubscribeLocale) unsubscribeLocale();
    window.removeEventListener('verstak:workspace-tree-changed', loadTree);
  });

  async function loadTree() {
    loading = true; error = '';
    try {
      const raw = await App.GetWorkspaceTreeV2();
      if (raw?.error) { error = raw.error; return; }
      tree = raw || { roots: [], currentWorkspaceId: '', revision: 0 };
      activeWid = tree.currentWorkspaceId || '';
      await loadExpanded();
      if (activeWid) ensureExpandedToWorkspace(activeWid);
      if (!focusedKey && tree.roots?.length) focusedKey = tree.roots[0].key;
    } catch (e) { error = tr('workspaceTree.loadError'); }
    loading = false;
  }

  async function loadExpanded() {
    try {
      const settings = await App.GetAppSettings();
      if (settings?.expandedFolderIds) expandedIds = Object.fromEntries(settings.expandedFolderIds.map(id => ['folder:' + id, true]));
    } catch {}
  }
  async function saveExpanded() {
    const ids = Object.keys(expandedIds).filter(k => k.startsWith('folder:')).map(k => k.slice(7));
    try { await App.UpdateAppSettings({ expandedFolderIds: ids }); } catch {}
  }

  async function loadTemplates() {
    try {
      const tlist = await App.ListWorkspaceTemplates();
      templates = Array.isArray(tlist) ? tlist : [];
    } catch { templates = []; }
  }

  function ensureExpandedToWorkspace(wid) {
    for (const root of tree.roots || []) if (expandToChild(root, wid)) return;
  }
  function expandToChild(node, targetWid) {
    if (node.kind === 'workspace' && node.id === targetWid) return true;
    if (node.children) for (const c of node.children) {
      if (expandToChild(c, targetWid)) { expandedIds[node.key] = true; return true; }
    }
    return false;
  }

  function toggleExpand(key) {
    expandedIds[key] = !expandedIds[key];
    expandedIds = expandedIds;
    focusedKey = key;
    saveExpanded();
  }

  async function selectWorkspace(wid) {
    const err = await App.SetCurrentWorkspaceV2(wid);
    if (err) { error = err; return; }
    activeWid = wid;
    focusedKey = 'workspace:' + wid;
    const ws = await App.GetWorkspaceByID(wid);
    const rootPath = ws?.rootPath || '';
    window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
      detail: { workspaceId: wid, workspaceName: rootPath, workspaceRootPath: rootPath }
    }));
  }

  // ── Flat visible node list for keyboard nav ────────────────────────────────
  function visibleNodes() {
    const out = [];
    function walk(nodes, depth) {
      for (const n of nodes || []) {
        out.push({ key: n.key, kind: n.kind, id: n.id, name: n.name, depth });
        if (n.kind === 'folder' && expandedIds[n.key] && n.children) walk(n.children, depth + 1);
      }
    }
    walk(tree.roots || [], 0);
    return out;
  }

  function handleNav(e) {
    const dir = e.detail?.dir;
    const vis = visibleNodes();
    const idx = vis.findIndex(n => n.key === focusedKey);
    let next = -1;
    if (dir === 'next' && idx < vis.length - 1) next = idx + 1;
    else if (dir === 'prev' && idx > 0) next = idx - 1;
    else if (dir === 'child' && idx >= 0) {
      const cur = vis[idx];
      if (cur.kind === 'folder' && expandedIds[cur.key]) next = idx + 1;
    } else if (dir === 'parent' && idx >= 0) {
      for (let i = idx - 1; i >= 0; i--) {
        if (vis[i].depth < vis[idx].depth) { next = i; break; }
      }
    }
    if (next >= 0) { focusedKey = vis[next].key; }
  }

  function handleRename(e) { openRename(e.detail.kind, e.detail.id, e.detail.name); }
  function handleTrash(e) { openTrash(e.detail.kind, e.detail.id, e.detail.name); }

  // ── Create/Rename/Move/Trash modals ────────────────────────────────────────
  function openCreateFolder(pid) { modal = { type: 'create-folder', parentId: pid }; formName = ''; formParentId = pid || ''; formError = ''; formBusy = false; }
  function openCreateWorkspace(pid) { modal = { type: 'create-workspace', parentId: pid }; formName = ''; formParentId = pid || ''; formTemplateId = templates[0]?.id || 'default'; formError = ''; formBusy = false; }
  function openRename(kind, id, name) { modal = { type: 'rename', kind, id }; formName = name; formError = ''; formBusy = false; }
  function openTrash(kind, id, name) { modal = { type: 'trash', kind, id, name }; formBusy = false; }
  function closeModal() { if (!formBusy) modal = null; }

  async function doCreateFolder() { const n = formName.trim(); if (!n) { formError = tr('workspaceTree.nameRequired'); return; } formBusy = true; const r = await App.CreateFolderV2(formParentId || '', n); if (r?.error) { formError = r.error; formBusy = false; return; } if (formParentId) { expandedIds['folder:' + formParentId] = true; saveExpanded(); } modal = null; await loadTree(); }
  async function doCreateWorkspace() { const n = formName.trim(); if (!n) { formError = tr('workspaceTree.nameRequired'); return; } formBusy = true; const r = await App.CreateWorkspaceV2(formParentId || '', n, formTemplateId); if (r?.error) { formError = r.error; formBusy = false; return; } if (formParentId) { expandedIds['folder:' + formParentId] = true; saveExpanded(); } const wid = r?.id; modal = null; await loadTree(); if (wid) await selectWorkspace(wid); }
  async function doRename() { const n = formName.trim(); if (!n) { formError = tr('workspaceTree.nameRequired'); return; } formBusy = true; let err = modal.kind === 'folder' ? await App.RenameFolderV2(modal.id, n) : await App.RenameWorkspaceV2(modal.id, n); if (err) { formError = err; formBusy = false; return; } modal = null; await loadTree(); }
  async function doMove() { formBusy = true; let err = modal.kind === 'folder' ? await App.MoveFolderV2(modal.id, formParentId || '') : await App.MoveWorkspaceV2(modal.id, formParentId || ''); if (err) { formError = err; formBusy = false; return; } modal = null; await loadTree(); }
  async function doTrash() { formBusy = true; if (modal.kind === 'folder') await App.TrashFolderV2(modal.id); else { await App.TrashWorkspaceV2(modal.id); if (activeWid === modal.id) activeWid = ''; } modal = null; await loadTree(); }

  // ── Context menu ───────────────────────────────────────────────────────────
  function onCtx(e) { ctxMenu = { x: e.detail.e.clientX, y: e.detail.e.clientY, kind: e.detail.kind, id: e.detail.id, name: e.detail.name }; }
  function closeCtx() { ctxMenu = null; }

  // ── Drag-and-drop ──────────────────────────────────────────────────────────
  let dragCounter = 0;
  let draggedNodeParentId = ''; // Track whether dragged node has a parent
  function onRootDragOver(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    dragCounter++;
    // Only show root drop zone if dragged node is nested (has a parent folder).
    if (!draggedNodeParentId) return;
    dragOverRoot = true;
  }
  function onRootDragLeave(e) { dragCounter--; if (dragCounter <= 0) { dragOverRoot = false; dragCounter = 0; } }
  function resetDragState() {
    dragOverRoot = false;
    dragOverFolderId = '';
    dragCounter = 0;
    draggedNodeParentId = '';
  }
  function onNodeDragStart(e) {
    draggedNodeParentId = findNodeParentID(e.detail?.id) || '';
  }

  function onRootDrop(e) {
    e.preventDefault();
    e.stopPropagation();
    resetDragState();
    try {
      const data = JSON.parse(e.dataTransfer.getData('application/x-verstak-node'));
      if (data.kind === 'folder') App.MoveFolderV2(data.id, '').then(loadTree).catch(() => {}).finally(resetDragState);
      else App.MoveWorkspaceV2(data.id, '').then(loadTree).catch(() => {}).finally(resetDragState);
    } catch { resetDragState(); }
  }
  function onNodeDrop(e) {
    resetDragState();
    const { source, targetId } = e.detail;
    if (source.kind === 'folder') App.MoveFolderV2(source.id, targetId).then(loadTree).catch(() => {}).finally(resetDragState);
    else App.MoveWorkspaceV2(source.id, targetId).then(loadTree).catch(() => {}).finally(resetDragState);
  }

  // ── Helpers ────────────────────────────────────────────────────────────────
  function flatFolders(roots, out = []) { for (const r of roots || []) { if (r.kind === 'folder') { out.push(r); flatFolders(r.children, out); } } return out; }
  function descendantIds(node) { const ids = new Set(); function walk(n) { for (const c of n.children || []) { ids.add(c.id); walk(c); } } walk(node); return ids; }
  function moveExcludedIds() { if (!modal?.id) return new Set(); const n = findNode(tree.roots, modal.id); return n ? descendantIds(n) : new Set(); }
  $: moveExcluded = moveExcludedIds();

  function subtreeCounts(id) { let folders = 0, wss = 0; const n = findNode(tree.roots || [], id); if (n) count(n); return { folders, workspaces: wss };
    function count(nd) { for (const c of nd.children || []) { if (c.kind === 'folder') { folders++; count(c); } else wss++; } } }
  function findNode(nodes, id) { for (const n of nodes) { if (n.id === id) return n; const f = findNode(n.children || [], id); if (f) return f; } return null; }
  function findNodeParentID(id) { return parentIDFor(tree.roots, id); }
  function parentIDFor(nodes, id) { for (const n of nodes) { if (n.children) for (const c of n.children) { if (c.id === id) return n.id; } const f = parentIDFor(n.children, id); if (f) return f; } return ''; }

  // ── Template plugin display ─────────────────────────────────────────────────
  const PLUGIN_NAMES = {
    'verstak.notes': 'Заметки', 'verstak.files': 'Файлы', 'verstak.journal': 'Журнал',
    'verstak.activity': 'Активность', 'verstak.browser-inbox': 'Браузер',
    'verstak.todo': 'Задачи', 'verstak.secrets': 'Секреты',
  };
  function pluginDisplayName(pluginId) {
    const key = pluginId.replace('verstak.', '');
    return tr(`plugin.${key}`, undefined, PLUGIN_NAMES[pluginId] || pluginId);
  }
  function pluginAvailable(pluginId) {
    return true; // plugins are loaded by core; availability checked at contribution filtering time
  }

  function onKeyDown(e) { if (e.key === "Escape") { closeCtx(); closeModal(); resetDragState(); } }
</script>

<svelte:window on:keydown={onKeyDown} on:mousedown={(e) => { if (e.button === 0 && ctxMenu) closeCtx(); }} />

<div class="wt" data-workspace-tree>
  <div class="wt-header">
    <span class="wt-title">{tr('workspaceTree.title')}</span>
    <div class="wt-header-actions">
      <button class="ti-btn" on:click={() => openCreateWorkspace('')} title={tr('workspaceTree.newDeal')} aria-label={tr('workspaceTree.newDeal')}><Icon name="space" size={14} /></button>
      <button class="ti-btn" on:click={() => openCreateFolder('')} title={tr('workspaceTree.newFolder')} aria-label={tr('workspaceTree.newFolder')}><Icon name="folder" size={14} /></button>
    </div>
  </div>

  <div class="wt-list" role="tree" aria-label={tr('workspaceTree.title')}
    on:dragover={onRootDragOver} on:dragleave={onRootDragLeave} on:drop={onRootDrop}
    on:dragend={resetDragState}
  >
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
        <TreeNode {node} depth={0} {expandedIds} {activeWid} {focusedKey}
          on:toggle={(e) => toggleExpand(e.detail.key)}
          on:select={(e) => selectWorkspace(e.detail.id)}
          on:nav={handleNav}
          on:rename={handleRename}
          on:trash={handleTrash}
          on:contextmenu={onCtx}
          on:drop={onNodeDrop}
          on:dragstart={onNodeDragStart}
          on:createFolder={(e) => openCreateFolder(e.detail)}
          on:createWorkspace={(e) => openCreateWorkspace(e.detail)}
        />
      {/each}
    {/if}
    {#if dragOverRoot}
      <div class="wt-root-drop">Переместить в корень</div>
    {/if}
  </div>
</div>

<!-- Context Menu -->
{#if ctxMenu}
  <div class="vt-ctx" style="left:{ctxMenu.x}px;top:{ctxMenu.y}px" on:click|stopPropagation on:mousedown|stopPropagation>
    {#if ctxMenu.kind === 'folder'}
      <button class="vt-ctx-i" on:click={() => { closeCtx(); openCreateWorkspace(ctxMenu.id); }}>{tr('workspaceTree.newDeal')}</button>
      <button class="vt-ctx-i" on:click={() => { closeCtx(); openCreateFolder(ctxMenu.id); }}>{tr('workspaceTree.newFolder')}</button>
      <div class="vt-ctx-s" />
      <button class="vt-ctx-i" on:click={() => { closeCtx(); openRename('folder', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.renameFolder')}</button>
      <button class="vt-ctx-i vt-ctx-d" on:click={() => { closeCtx(); openTrash('folder', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.trashFolder')}</button>
    {:else}
      <button class="vt-ctx-i" on:click={() => { closeCtx(); selectWorkspace(ctxMenu.id); }}>{tr('workspaceTree.open')}</button>
      <button class="vt-ctx-i" on:click={() => { closeCtx(); openRename('workspace', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.renameDeal')}</button>
      <button class="vt-ctx-i vt-ctx-d" on:click={() => { closeCtx(); openTrash('workspace', ctxMenu.id, ctxMenu.name); }}>{tr('workspaceTree.trashDeal')}</button>
    {/if}
  </div>
{/if}

<!-- Modals -->
<Modal title={tr('workspaceTree.newFolder')} show={modal?.type === 'create-folder'} on:close={closeModal}>
  <label class="vt-field"><span>{tr('workspaceTree.location')}</span><Select options={flatFolders(tree.roots).map(f => ({ value: f.id, label: f.path }))} placeholder={tr('workspaceTree.root')} bind:value={formParentId} labelKey="label" valueKey="value" /></label>
  <label class="vt-field"><span>{tr('workspaceTree.folderName')}</span><input class="vt-input" type="text" bind:value={formName} placeholder={tr('workspaceTree.folderNamePlaceholder')} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doCreateFolder()} /></label>
  {#if formError}<p class="vt-ferr">{formError}</p>{/if}
  <svelte:fragment slot="actions"><button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button><button class="vt-btn-p" on:click={doCreateFolder} disabled={formBusy}>{tr('common.create')}</button></svelte:fragment>
</Modal>

<Modal title={tr('workspaceTree.newDeal')} show={modal?.type === 'create-workspace'} on:close={closeModal} wide>
  <label class="vt-field"><span>{tr('workspaceTree.location')}</span><Select options={flatFolders(tree.roots).map(f => ({ value: f.id, label: f.path }))} placeholder={tr('workspaceTree.root')} bind:value={formParentId} labelKey="label" valueKey="value" /></label>
  <label class="vt-field"><span>{tr('workspaceTree.name')}</span><input class="vt-input" type="text" bind:value={formName} placeholder={tr('workspaceTree.namePlaceholder')} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doCreateWorkspace()} /></label>
  <label class="vt-field"><span>{tr('workspaceTree.template')}</span><Select options={templates} bind:value={formTemplateId} labelKey="name" valueKey="id" /></label>
  {@const st = templates.find(t => t.id === formTemplateId)}
  {#if st}
    <div class="vt-template-info">
      {#if st.description}<p class="vt-template-desc">{st.description}</p>{/if}
      {#if st.workspaceTools?.length}
        <div class="vt-template-badges">
          {#each st.workspaceTools as pt}
            <span class="vt-badge vt-tool-badge" class:vt-tool-unavailable={!pluginAvailable(pt)} title={pt}>{pluginDisplayName(pt)}</span>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
  {#if formError}<p class="vt-ferr">{formError}</p>{/if}
  <svelte:fragment slot="actions"><button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button><button class="vt-btn-p" on:click={doCreateWorkspace} disabled={formBusy}>{tr('common.create')}</button></svelte:fragment>
</Modal>

<Modal title={tr('workspaceTree.rename')} show={modal?.type === 'rename'} on:close={closeModal}>
  <label class="vt-field"><span>{tr('workspaceTree.newName')}</span><input class="vt-input" type="text" bind:value={formName} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doRename()} /></label>
  {#if formError}<p class="vt-ferr">{formError}</p>{/if}
  <svelte:fragment slot="actions"><button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button><button class="vt-btn-p" on:click={doRename} disabled={formBusy}>{tr('common.save')}</button></svelte:fragment>
</Modal>

<Modal title={(modal?.kind === 'folder' ? tr('workspaceTree.trashFolder') : tr('workspaceTree.trashDeal')) + (modal?.name ? ' «' + modal.name + '»?' : '?')} show={modal?.type === 'trash'} on:close={closeModal}>
  <p class="vt-trash-desc">
    {#if modal?.kind === 'folder'}
      {@const c = subtreeCounts(modal.id)}
      {tr('workspaceTree.trashFolderDesc')}<br />
      {tr('workspaceTree.contains')}: {c.folders} {tr('workspaceTree.nestedFolders')}, {c.workspaces} {tr('workspaceTree.title')}
    {:else}
      {tr('workspaceTree.trashDealDesc')}
    {/if}
  </p>
  <svelte:fragment slot="actions"><button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button><button class="vt-btn-d" on:click={doTrash} disabled={formBusy}>{tr('workspaceTree.toTrash')}</button></svelte:fragment>
</Modal>

<style>
  .wt { display: flex; flex-direction: column; flex: 1; overflow: hidden; }
  .wt-header { display: flex; align-items: center; justify-content: space-between; padding: 0.7rem 0.6rem 0.35rem; border-bottom: 1px solid var(--vt-color-border); flex-shrink: 0; }
  .wt-title { color: var(--vt-color-text-muted); font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 600; }
  .wt-header-actions { display: flex; gap: 0.2rem; }
  .wt-list { min-height: 0; overflow-y: auto; padding: 0.2rem 0.4rem; flex: 1; position: relative; }
  .wt-status { padding: 0.5rem; font-size: 0.78rem; color: var(--vt-color-text-muted); }
  .wt-error { color: var(--vt-color-danger); }
  .wt-empty { padding: 1rem 0.5rem; text-align: center; color: var(--vt-color-text-muted); font-size: 0.8rem; }
  .wt-empty-hint { font-size: 0.72rem; opacity: 0.7; }

  .wt-root-drop { margin: 0.2rem 0.4rem; padding: 0.4rem; border: 1px dashed var(--vt-color-accent); border-radius: var(--vt-radius-sm); text-align: center; color: var(--vt-color-accent); font-size: 0.75rem; background: var(--vt-color-accent-muted); }

  .ti-btn { width: 1.6rem; height: 1.6rem; min-height: 0; padding: 0; border: 1px solid transparent; background: transparent; color: var(--vt-color-text-muted); cursor: pointer; border-radius: var(--vt-radius-sm); display: inline-flex; align-items: center; justify-content: center; }
  .ti-btn:hover { color: var(--vt-color-accent); background: var(--vt-color-accent-muted); border-color: rgba(78,204,163,0.25); }

  .vt-btn { min-height: 1.8rem; background: transparent; border: 1px solid var(--vt-color-border-strong); color: var(--vt-color-text-secondary); cursor: pointer; font-size: 0.78rem; padding: 0.3rem 0.6rem; border-radius: var(--vt-radius-sm); }
  .vt-btn:hover:not(:disabled) { color: var(--vt-color-text-primary); border-color: var(--vt-color-text-muted); }
  .vt-btn-p { min-height: 1.8rem; background: var(--vt-color-accent); color: #101827; border: none; padding: 0.3rem 0.7rem; border-radius: var(--vt-radius-sm); cursor: pointer; font-size: 0.78rem; font-weight: 600; }
  .vt-btn-p:hover:not(:disabled) { background: #3dbb92; }
  .vt-btn-d { min-height: 1.8rem; background: var(--vt-color-danger); color: #fff; border: none; padding: 0.3rem 0.7rem; border-radius: var(--vt-radius-sm); cursor: pointer; font-size: 0.78rem; font-weight: 600; }
  .vt-btn-d:hover:not(:disabled) { background: #d63851; }
  .vt-btn:disabled, .vt-btn-p:disabled, .vt-btn-d:disabled { opacity: 0.5; cursor: not-allowed; }

  .vt-field { display: grid; gap: 0.35rem; color: var(--vt-color-text-muted); font-size: 0.75rem; }
  .vt-input { width: 100%; min-height: 2rem; box-sizing: border-box; border: 1px solid var(--vt-color-border-strong); border-radius: var(--vt-radius-sm); background: #0f1424; color: var(--vt-color-text-primary); padding: 0.35rem 0.5rem; font: inherit; font-size: 0.84rem; }
  .vt-input:focus { outline: none; border-color: var(--vt-color-accent); box-shadow: var(--vt-focus-ring); }
  .vt-ferr { margin: 0; color: var(--vt-color-danger); font-size: 0.78rem; line-height: 1.4; }

  .vt-trash-desc { color: var(--vt-color-text-secondary); font-size: 0.84rem; margin: 0; line-height: 1.5; }

  .vt-ctx { position: fixed; z-index: 10001; min-width: 10rem; background: var(--vt-color-surface); border: 1px solid var(--vt-color-border-strong); border-radius: var(--vt-radius-md); box-shadow: 0 8px 24px rgba(0,0,0,0.3); padding: 0.25rem; }
  .vt-ctx-i { display: block; width: 100%; text-align: left; padding: 0.3rem 0.6rem; background: none; border: none; color: var(--vt-color-text-secondary); font-size: 0.78rem; cursor: pointer; border-radius: var(--vt-radius-sm); }
  .vt-ctx-i:hover { background: var(--vt-color-surface-hover); color: var(--vt-color-text-primary); }
  .vt-ctx-s { height: 1px; background: var(--vt-color-border); margin: 0.2rem 0.3rem; }
  .vt-ctx-d { color: var(--vt-color-danger); }
  .vt-ctx-d:hover { background: var(--vt-color-danger-muted); }
  .vt-template-info { margin: var(--vt-space-2) 0; }
  .vt-template-desc { color: var(--vt-color-text-secondary); font-size: 0.8rem; line-height: 1.4; margin-bottom: var(--vt-space-2); }
  .vt-template-badges { display: flex; flex-wrap: wrap; gap: 0.35rem; }
  .vt-tool-badge { font-size: 0.72rem; padding: 0.15rem 0.5rem; }
  .vt-tool-unavailable { opacity: 0.45; border-color: var(--vt-color-warning); color: var(--vt-color-warning); }
</style>
