<script>
  import { createEventDispatcher, onDestroy } from 'svelte';
  import Icon from './Icon.svelte';
  import { LUCIDE_ICONS } from './icons/catalogue.js';
  import { i18n } from '../i18n/index.js';

  export let current = '';
  export let defaultIcon = 'folder';
  let search = '';
  let locale = i18n.getLocale();
  const offLocale = i18n.subscribe((value) => { locale = value; });
  onDestroy(offLocale);
  const dispatch = createEventDispatcher();
  $: tr = ((value) => (key) => { void value; return i18n.t(key); })(locale);
  $: filtered = LUCIDE_ICONS.filter((name) => !search || name.includes(search.toLowerCase())).slice(0, 200);
</script>

<label class="vt-field">
  <span>{tr('workspaceTree.iconSearch')} ({filtered.length} / {LUCIDE_ICONS.length})</span>
  <input class="vt-input" type="search" data-modal-initial-focus bind:value={search} placeholder={tr('workspaceTree.iconSearch') + '...'} />
</label>
<div class="vt-icon-grid">
  <button type="button" class="vt-icon-item" class:vt-icon-selected={!current} on:click={() => dispatch('select', '')}><Icon name={defaultIcon} size={20} /><span>{tr('common.defaultIcon')}</span></button>
  {#each filtered as icon}
    <button type="button" class="vt-icon-item" class:vt-icon-selected={current === icon} title={icon} on:click={() => dispatch('select', icon)}><Icon name={icon} size={20} /><span>{icon}</span></button>
  {/each}
</div>

<style>
  .vt-icon-grid { display: grid; grid-template-columns: repeat(auto-fill, 64px); grid-auto-rows: 72px; gap: 6px; margin-top: 8px; max-height: 55vh; overflow-y: auto; justify-content: center; }
  .vt-icon-item { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 2px; width: 64px; height: 72px; min-width: 64px; min-height: 72px; padding: 4px 2px; border: 1px solid transparent; border-radius: var(--vt-radius-sm); background: transparent; color: var(--vt-color-text-secondary); cursor: pointer; font-size: .6rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .vt-icon-item:hover, .vt-icon-selected { border-color: var(--vt-color-accent); background: var(--vt-color-accent-muted); }
  .vt-icon-selected { color: var(--vt-color-accent); }
</style>
