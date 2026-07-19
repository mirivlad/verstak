<script>
  import { createEventDispatcher } from 'svelte';
  import Icon from '../ui/Icon.svelte';

  export let node;
  export let depth = 0;
  export let expandedIds = {};
  export let activeWid = '';
  export let focusedKey = '';
  export let appearanceCache = {};

  const dispatch = createEventDispatcher();
  const INDENT = 1.15; // rem

  $: isFolder = node.kind === 'folder';
  $: isWorkspace = node.kind === 'workspace';
  $: isExpanded = isFolder && !!expandedIds[node.key];
  $: isActive = isWorkspace && node.id === activeWid;
  $: hasChildren = isFolder && node.children && node.children.length > 0;
  $: folderAppearance = isFolder ? (appearanceCache[node.id] || {}) : {};
  $: folderIconName = folderAppearance.iconId || 'folder';
  $: folderIconColor = folderAppearance.colorId || '';

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
    dispatch('contextmenu', { e, kind: node.kind, id: node.id, name: node.name });
  }

  function onDragStart(e) {
    e.dataTransfer.setData('application/x-verstak-node', JSON.stringify({ kind: node.kind, id: node.id, name: node.name }));
    e.dataTransfer.effectAllowed = 'move';
    dispatch('dragstart', { kind: node.kind, id: node.id });
  }

  function onDragOver(e) {
    if (!isFolder) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    dispatch('dragover', { targetId: node.id });
  }

  function onDragLeave() {
    dispatch('dragleave', { targetId: node.id });
  }

  function onDrop(e) {
    e.preventDefault();
    e.stopPropagation();
    if (!isFolder) return;
    try {
      const data = JSON.parse(e.dataTransfer.getData('application/x-verstak-node'));
      dispatch('drop', { source: data, targetId: node.id });
    } catch {}
  }

  function createFolder() { dispatch('createFolder', node.id); }
  function createWorkspace() { dispatch('createWorkspace', node.id); }
</script>

<div class="tnode" style="padding-left:{depth * INDENT}rem" role="treeitem"
  aria-expanded={isFolder ? isExpanded : undefined}
  aria-selected={isActive || undefined}
  aria-level={depth + 1}
>
  <div
    class="trow" class:selected={isActive} class:focused={focusedKey === node.key}
    class:drag-over={false}
    on:click={handleClick}
    on:keydown={handleKeyDown}
    on:contextmenu={onCtxMenu}
    draggable="true"
    on:dragstart={onDragStart}
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
    <span class="tico"><Icon name={isFolder ? folderIconName : 'layout-grid'} size={14} style={isFolder && folderIconColor ? 'color:' + folderIconColor : ''} /></span>
    <span class="tname" title={node.name}>{node.name}</span>
    <span class="tact">
      {#if isFolder}
        <button class="ti-btn" on:click|stopPropagation={createWorkspace} title="Новое Дело" aria-label="Новое Дело"><Icon name="plus" size={11} /></button>
        <button class="ti-btn" on:click|stopPropagation={createFolder} title="Новая папка" aria-label="Новая папка"><Icon name="folder-plus" size={11} /></button>
      {/if}
    </span>
  </div>

  {#if isFolder && isExpanded && hasChildren}
    {#each node.children as child (child.key)}
      <svelte:self
        node={child} depth={depth + 1} {expandedIds} {activeWid} {focusedKey} {appearanceCache}
        on:toggle on:select on:nav on:rename on:trash on:contextmenu
        on:dragstart on:dragover on:dragleave on:drop
        on:createFolder on:createWorkspace
      />
    {/each}
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
