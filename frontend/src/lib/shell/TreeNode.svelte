<script>
  import { createEventDispatcher } from 'svelte';
  import Icon from '../ui/Icon.svelte';

  export let node;
  export let depth = 0;
  export let expandedIds = {};
  export let activeWid = '';
  export let focusedKey = '';
  export let appearanceCache = {};
  export let dragTarget = null;
  export let newDealLabel = 'New Deal';
  export let newFolderLabel = 'New folder';

  const dispatch = createEventDispatcher();
  const INDENT = 1.15; // rem
  let fileDragOver = false;

  $: isFolder = node.kind === 'folder';
  $: isWorkspace = node.kind === 'workspace';
  $: isExpanded = isFolder && !!expandedIds[node.key];
  $: isActive = isWorkspace && node.id === activeWid;
  $: hasChildren = isFolder && node.children && node.children.length > 0;
  $: folderAppearance = isFolder ? (appearanceCache[node.id] || {}) : {};
  $: folderIconName = folderAppearance.iconId || 'folder';
  $: folderIconColor = folderAppearance.colorId || '';
  $: activeDropPosition = dragTarget?.targetKey === node.key ? dragTarget.position : '';

  function handleClick() {
    if (isFolder) {
      dispatch('toggle', { key: node.key });
    } else {
      dispatch('select', { id: node.id });
    }
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter') {
      if (isWorkspace) dispatch('select', { id: node.id });
      else dispatch('toggle', { key: node.key });
    } else if (e.key === ' ') {
      if (isFolder) { e.preventDefault(); dispatch('toggle', { key: node.key }); }
    } else if (e.key === 'ArrowRight') {
      e.preventDefault();
      if (isFolder && !isExpanded) dispatch('toggle', { key: node.key });
      else dispatch('nav', { dir: 'child' });
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault();
      if (isFolder && isExpanded) dispatch('toggle', { key: node.key });
      else dispatch('nav', { dir: 'parent' });
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      dispatch('nav', { dir: 'next' });
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      dispatch('nav', { dir: 'prev' });
    } else if (e.key === 'F2') {
      e.preventDefault();
      dispatch('rename', { kind: node.kind, id: node.id, name: node.name });
    } else if (e.key === 'Delete') {
      e.preventDefault();
      dispatch('trash', { kind: node.kind, id: node.id, name: node.name });
    }
  }

  function onCtxMenu(e) {
    e.preventDefault();
    dispatch('contextmenu', { e, kind: node.kind, id: node.id, name: node.name });
  }

  function onDragStart(e) {
    e.dataTransfer.setData('application/x-verstak-node', JSON.stringify({ key: node.key }));
    e.dataTransfer.effectAllowed = 'move';
    dispatch('dragstart', { sourceKey: node.key });
  }

  function onDragOver(e) {
    if (isWorkspace && Array.from(e.dataTransfer?.types || []).includes('application/x-verstak-files')) {
      e.preventDefault();
      e.stopPropagation();
      e.dataTransfer.dropEffect = 'move';
      fileDragOver = true;
      return;
    }
    if (!Array.from(e.dataTransfer?.types || []).includes('application/x-verstak-node')) return;
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = 'move';
    const rect = e.currentTarget.getBoundingClientRect();
    const ratio = rect.height > 0 ? (e.clientY - rect.top) / rect.height : 0.5;
    let position;
    if (isFolder) {
      position = ratio < 1 / 3 ? 'before' : ratio > 2 / 3 ? 'after' : 'inside';
    } else {
      position = ratio < 0.5 ? 'before' : 'after';
    }
    dispatch('dragtarget', { targetKey: node.key, position, clientY: e.clientY });
  }

  function onDragLeave(e) {
    fileDragOver = false;
    if (!e.currentTarget.contains(e.relatedTarget)) {
      dispatch('dragleave', { targetKey: node.key });
    }
  }

  function onDrop(e) {
    e.preventDefault();
    e.stopPropagation();
    fileDragOver = false;
    const filePayload = e.dataTransfer.getData('application/x-verstak-files');
    if (filePayload && isWorkspace) {
      try {
        dispatch('filedrop', { payload: JSON.parse(filePayload), targetId: node.id });
      } catch {}
      return;
    }
    try {
      const data = JSON.parse(e.dataTransfer.getData('application/x-verstak-node'));
      if (!data.key || !activeDropPosition) {
        dispatch('dragcancel');
        return;
      }
      dispatch('drop', {
        sourceKey: data.key,
        targetKey: node.key,
        position: activeDropPosition,
      });
    } catch {
      dispatch('dragcancel');
    }
  }

  function onChildListDragOver(e) {
    if (!Array.from(e.dataTransfer?.types || []).includes('application/x-verstak-node')) return;
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = 'move';
    dispatch('dragtarget', { targetKey: node.key, position: 'inside', clientY: e.clientY });
  }

  function onChildListDrop(e) {
    e.preventDefault();
    e.stopPropagation();
    try {
      const data = JSON.parse(e.dataTransfer.getData('application/x-verstak-node'));
      if (!data.key) throw new Error('missing stable key');
      dispatch('drop', { sourceKey: data.key, targetKey: node.key, position: 'inside' });
    } catch {
      dispatch('dragcancel');
    }
  }

  function createFolder() { dispatch('createFolder', node.id); }
  function createWorkspace() { dispatch('createWorkspace', node.id); }
</script>

<div class="tnode" style="padding-left:{depth * INDENT}rem">
  <div
    class="trow wt-node" class:selected={isActive} class:focused={focusedKey === node.key}
    class:drag-over={fileDragOver}
    class:drop-before={activeDropPosition === 'before'}
    class:drop-inside={activeDropPosition === 'inside'}
    class:drop-after={activeDropPosition === 'after'}
    data-tree-key={node.key}
    data-drop-position={activeDropPosition || undefined}
    role="treeitem"
    aria-expanded={isFolder ? isExpanded : undefined}
    aria-selected={isActive || undefined}
    aria-level={depth + 1}
    on:click={handleClick}
    on:keydown={handleKeyDown}
    on:contextmenu={onCtxMenu}
    draggable="true"
    on:dragstart={onDragStart}
    on:dragend={() => dispatch('dragend')}
    on:dragover={onDragOver}
    on:dragleave={onDragLeave}
    on:drop={onDrop}
    tabindex={focusedKey === node.key ? 0 : -1}
  >
    {#if isFolder}
      <span class="tchev" class:open={isExpanded}><Icon name="chevron-right" size={12} /></span>
    {:else}
      <span class="tchev tempty" />
    {/if}
    <span class="tico"><Icon class="wt-node-icon" name={isFolder ? folderIconName : 'layout-grid'} size={14} style={isFolder && folderIconColor ? 'color:' + folderIconColor : ''} /></span>
    <span class="tname wt-label" title={node.name}>{node.name}</span>
    <span class="tact">
      {#if isFolder}
        <button class="ti-btn" on:click|stopPropagation={createWorkspace} title={newDealLabel} aria-label={newDealLabel}><Icon name="plus" size={11} /></button>
        <button class="ti-btn" on:click|stopPropagation={createFolder} title={newFolderLabel} aria-label={newFolderLabel}><Icon name="folder-plus" size={11} /></button>
      {/if}
    </span>
  </div>

  {#if isFolder && isExpanded && hasChildren}
    {#each node.children as child (child.key)}
      <svelte:self
        node={child} depth={depth + 1} {expandedIds} {activeWid} {focusedKey} {appearanceCache} {dragTarget} {newDealLabel} {newFolderLabel}
        on:toggle on:select on:nav on:rename on:trash on:contextmenu
        on:dragstart on:dragtarget on:dragleave on:drop on:filedrop on:dragend on:dragcancel
        on:createFolder on:createWorkspace
      />
    {/each}
  {/if}
  {#if isFolder && isExpanded}
    <div
      class="tchild-drop"
      class:active={activeDropPosition === 'inside'}
      data-tree-drop-children={node.key}
      data-drop-active={activeDropPosition === 'inside' ? 'inside' : undefined}
      style="margin-left:{(depth + 1) * INDENT}rem"
      role="none"
      on:dragover={onChildListDragOver}
      on:drop={onChildListDrop}
    />
  {/if}
</div>

<style>
  .tnode { user-select: none; }
  .trow {
    display: flex; align-items: center; gap: 0.3rem;
    padding: 0.18rem 0.45rem; min-height: 1.85rem;
    border-radius: var(--vt-radius-sm); cursor: pointer;
    transition: background 0.1s;
    outline: none;
  }
  .trow:hover { background: var(--vt-color-surface-hover); }
  .trow:focus { outline: none; }
  .trow:focus-visible { outline: 1px solid var(--vt-color-accent); outline-offset: -1px; }
  .trow.selected { background: var(--vt-color-surface-selected); box-shadow: inset 2px 0 0 var(--vt-color-accent); }
  .trow.selected .tname { color: var(--vt-color-text-primary); }
  .trow.focused { outline: 1px solid var(--vt-color-accent); outline-offset: -1px; }
  .trow.drag-over { background: var(--vt-color-accent-muted); outline: 1px dashed var(--vt-color-accent); outline-offset: -2px; }
  .trow.drop-before::before, .trow.drop-after::after {
    content: ''; position: absolute; left: 0.35rem; right: 0.35rem; height: 2px;
    background: var(--vt-color-accent); border-radius: 999px; pointer-events: none; z-index: 2;
  }
  .trow.drop-before::before { top: -1px; }
  .trow.drop-after::after { bottom: -1px; }
  .trow.drop-inside { background: var(--vt-color-accent-muted); outline: 1px solid var(--vt-color-accent); outline-offset: -1px; }
  .trow { position: relative; }
  .tchild-drop { min-height: 0.55rem; border-radius: var(--vt-radius-sm); }
  .tchild-drop.active { min-height: 1rem; background: var(--vt-color-accent-muted); outline: 1px dashed var(--vt-color-accent); outline-offset: -2px; }
  .tchev { width: 0.9rem; height: 0.9rem; display: flex; align-items: center; justify-content: center; flex-shrink: 0; color: var(--vt-color-text-muted); transition: transform 0.15s; }
  .tchev.open { transform: rotate(90deg); }
  .tempty { visibility: hidden; }
  .tico { width: 1rem; height: 1rem; display: flex; align-items: center; justify-content: center; flex-shrink: 0; color: var(--vt-color-text-muted); }
  .tname { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.78rem; color: var(--vt-color-text-secondary); }
  .tact { display: flex; gap: 0.1rem; opacity: 0; transition: opacity 0.1s; }
  .trow:hover .tact, .trow:focus-within .tact { opacity: 1; }
  .ti-btn { width: 1.3rem; height: 1.3rem; min-height: 0; padding: 0; border: 1px solid transparent; background: transparent; color: var(--vt-color-text-muted); cursor: pointer; border-radius: var(--vt-radius-sm); display: inline-flex; align-items: center; justify-content: center; }
  .ti-btn:hover { color: var(--vt-color-accent); background: var(--vt-color-accent-muted); border-color: rgba(78,204,163,0.25); }
</style>
