<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import WorkspaceTree from './WorkspaceTree.svelte';
  import Icon from '../ui/Icon.svelte';
  import { debug } from '../log/debug.js';
  import { i18n } from '../i18n/index.js';

  export let activeView = null;
  export let activeViewPluginId = '';

  function flog(msg) {
    App.WriteFrontendLog('Sidebar', msg);
  }

  let plugins = [];
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let sidebarItems = [];
  let errorMessage = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;
  const MIN_SIDEBAR_WIDTH = 180;
  const MAX_SIDEBAR_WIDTH = 420;
  const DEFAULT_SIDEBAR_WIDTH = 220;
  const MIN_CONTENT_WIDTH = 320;
  let sidebarWidth = DEFAULT_SIDEBAR_WIDTH;
  let viewportWidth = 1200;
  let resizing = false;
  let resizeStartX = 0;
  let resizeStartWidth = DEFAULT_SIDEBAR_WIDTH;

  $: visibleSidebarWidth = Math.min(
    sidebarWidth,
    Math.max(MIN_SIDEBAR_WIDTH, Math.min(MAX_SIDEBAR_WIDTH, viewportWidth - MIN_CONTENT_WIDTH)),
  );

  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);

  $: vaultOpen = vaultStatus.status === 'open';
  $: activeSidebarKey = activeView ? `${activeViewPluginId}:${activeView}` : '';

  async function loadSidebar() {
    debug.log('[Sidebar] onMount: START');
    flog('onMount: START');
    let contribErr = false;
    try {
      debug.log('[Sidebar] onMount: loading plugins/vault/contributions...');
      flog('onMount: loading plugins/vault/contributions...');
      const [p, v, contribs] = await Promise.all([
        App.GetPlugins().catch(() => []),
        App.GetVaultStatus().catch(() => ({ status: 'unknown', path: '', vaultId: '' })),
        App.GetContributions().catch(() => { contribErr = true; return {}; }),
      ]);
      await Promise.all((p || []).map((plugin) => (
        i18n.loadPlugin(plugin.manifest?.id, plugin.manifest?.localization).catch(() => {})
      )));
      plugins = (p || []).map((plugin) => i18n.localizePlugin(plugin));
      const localizedContributions = i18n.localizeContributionSummary(contribs || {});
      vaultStatus = v;
      debug.log('[Sidebar] onMount: plugins=' + plugins.length + ' vault=' + vaultStatus.status);
      flog('onMount: plugins=' + plugins.length + ' vault=' + vaultStatus.status);
      if (contribErr) {
        errorMessage = tr('sidebar.error.contributions');
      }
      sidebarItems = (localizedContributions.sidebarItems || []).filter(item => {
        const plugin = plugins.find(p => p.manifest?.id === item.pluginId);
        if (!plugin) return false;
        return plugin.status !== 'disabled' && plugin.status !== 'failed' && plugin.status !== 'incompatible' && plugin.status !== 'missing-required-capability';
      });
      sidebarItems.sort((a, b) => (a.position || 100) - (b.position || 100));
      debug.log('[Sidebar] onMount: sidebarItems=' + sidebarItems.length);
      flog('onMount: sidebarItems=' + sidebarItems.length);
    } catch (e) {
      debug.log('[Sidebar] onMount: ERROR:', String(e));
      flog('onMount: ERROR: ' + String(e));
      console.error('[Sidebar] load error:', e);
      errorMessage = tr('sidebar.error.load');
    }
    debug.log('[Sidebar] onMount: END');
    flog('onMount: END');
  }

  onMount(() => {
    viewportWidth = window.innerWidth;
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed) loadSidebar();
    });
    loadSidebar();
    loadSidebarWidth();
    window.addEventListener('verstak:plugins-changed', loadSidebar);
    window.addEventListener('resize', handleWindowResize);
  });

  onDestroy(() => {
    stopResize();
    if (unsubscribeLocale) unsubscribeLocale();
    window.removeEventListener('verstak:plugins-changed', loadSidebar);
    window.removeEventListener('resize', handleWindowResize);
  });

  function handleSidebarItem(item) {
    debug.log('[Sidebar] handleSidebarItem:', item.id, '-> view:', item.view);
    // Use item.view (the view contribution ID) if available, fall back to item.id
    const viewId = item.view || item.id;
    window.dispatchEvent(new CustomEvent('verstak:open-view', { detail: { viewId, pluginId: item.pluginId } }));
  }

  async function loadSidebarWidth() {
    try {
      const settings = await App.GetAppSettings();
      const width = Number(settings?.sidebarWidth);
      if (Number.isInteger(width)) {
        sidebarWidth = clampWidth(width);
      }
    } catch {}
  }

  function clampWidth(width) {
    return Math.max(MIN_SIDEBAR_WIDTH, Math.min(MAX_SIDEBAR_WIDTH, Math.round(width)));
  }

  function handleWindowResize() {
    viewportWidth = window.innerWidth;
  }

  function startResize(event) {
    if (window.innerWidth <= 720) return;
    event.preventDefault();
    resizing = true;
    resizeStartX = event.clientX;
    resizeStartWidth = visibleSidebarWidth;
    event.currentTarget.setPointerCapture(event.pointerId);
    document.body.classList.add('sidebar-resizing');
  }

  function continueResize(event) {
    if (!resizing) return;
    const maxWidth = Math.max(
      MIN_SIDEBAR_WIDTH,
      Math.min(MAX_SIDEBAR_WIDTH, viewportWidth - MIN_CONTENT_WIDTH),
    );
    sidebarWidth = Math.max(
      MIN_SIDEBAR_WIDTH,
      Math.min(maxWidth, resizeStartWidth + event.clientX - resizeStartX),
    );
  }

  function stopResize(persist = false) {
    if (!resizing) return;
    resizing = false;
    document.body.classList.remove('sidebar-resizing');
    if (persist) persistSidebarWidth();
  }

  async function persistSidebarWidth() {
    const err = await App.UpdateAppSettings({ sidebarWidth: Math.round(sidebarWidth) });
    if (err) console.warn('[Sidebar] could not persist width');
  }

  function resetSidebarWidth() {
    sidebarWidth = DEFAULT_SIDEBAR_WIDTH;
    persistSidebarWidth();
  }

  function resizeWithKeyboard(event) {
    let next = sidebarWidth;
    if (event.key === 'ArrowLeft') next -= 10;
    else if (event.key === 'ArrowRight') next += 10;
    else if (event.key === 'Home') next = MIN_SIDEBAR_WIDTH;
    else if (event.key === 'End') next = MAX_SIDEBAR_WIDTH;
    else return;
    event.preventDefault();
    sidebarWidth = clampWidth(next);
    persistSidebarWidth();
  }
</script>

<aside class="sidebar" class:is-resizing={resizing} style="--sidebar-width:{visibleSidebarWidth}px">
  <div class="sidebar-header">
    <Icon name="logo" size={20} class="sidebar-logo" />
    <span class="sidebar-title">Verstak</span>
  </div>

  {#if sidebarItems.length > 0}
    <div class="sidebar-section">
      <span class="section-label">{tr('sidebar.tools')}</span>
      {#each sidebarItems as item}
        <button
          class="nav-item plugin-item vt-list-row"
          class:is-active={activeSidebarKey === `${item.pluginId || ''}:${item.view || item.id}`}
          aria-current={activeSidebarKey === `${item.pluginId || ''}:${item.view || item.id}` ? 'page' : undefined}
          on:click={() => handleSidebarItem(item)}
          type="button"
        >
          <Icon name={item.icon || 'plugin'} size={16} class="nav-icon icon-plugin" />
          <span class="nav-label">{item.title || item.id}</span>
        </button>
      {/each}
    </div>
  {/if}

  {#if vaultOpen}
    <WorkspaceTree />
  {/if}

  <div class="sidebar-footer">
    {#if errorMessage}
      <span class="sidebar-error">
        <Icon name="warning" size={10} class="sidebar-error-icon" />
        {tr('sidebar.error.ui')}
      </span>
    {/if}
  </div>
  <button
    type="button"
    class="sidebar-resizer"
    data-sidebar-resizer
    aria-label={tr('sidebar.resize')}
    title={tr('sidebar.resizeReset')}
    on:pointerdown={startResize}
    on:pointermove={continueResize}
    on:pointerup={() => stopResize(true)}
    on:pointercancel={() => stopResize(true)}
    on:dblclick={resetSidebarWidth}
    on:keydown={resizeWithKeyboard}
  />
</aside>

<style>
  .sidebar {
    position: relative;
    width: var(--sidebar-width);
    min-width: var(--sidebar-width);
    background: var(--vt-color-surface-muted);
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--vt-color-border);
    overflow: hidden;
  }

  .sidebar-resizer {
    position: absolute;
    top: 0;
    right: 0;
    bottom: 0;
    z-index: 1;
    width: 6px;
    padding: 0;
    border: 0;
    background: transparent;
    cursor: col-resize;
    touch-action: none;
  }
  .sidebar-resizer:hover,
  .sidebar-resizer:focus-visible,
  .sidebar.is-resizing .sidebar-resizer {
    background: var(--vt-color-accent);
  }
  .sidebar-resizer:focus-visible {
    outline: none;
  }
  :global(body.sidebar-resizing) {
    cursor: col-resize;
    user-select: none;
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid var(--vt-color-border);
  }

  :global(.sidebar-logo) {
    width: 1.2rem;
    height: 1.2rem;
    color: var(--vt-color-accent);
    flex-shrink: 0;
  }

  .sidebar-title {
    color: var(--vt-color-text-primary);
    font-size: 1rem;
    font-weight: 600;
  }

  .sidebar-section {
    display: flex;
    flex-direction: column;
    padding: 0.45rem 0.6rem 0.55rem;
    gap: 0.15rem;
    border-bottom: 1px solid var(--vt-color-border);
  }

  :global(workspace-tree) {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .section-label {
    color: var(--vt-color-text-muted);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 0.25rem 0.45rem 0.35rem;
    font-weight: 600;
  }

  .nav-item {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 0.45rem;
    min-height: 1.85rem;
    padding: 0.15rem 0.45rem;
    background: none;
    border: none;
    color: var(--vt-color-text-secondary);
    font-size: 0.78rem;
    cursor: pointer;
    border-radius: var(--vt-radius-sm);
    text-align: left;
    width: 100%;
    transition: background 0.15s, color 0.15s;
  }

  .nav-item:hover {
    background: var(--vt-color-surface-hover);
    color: var(--vt-color-text-primary);
  }

  .nav-item.is-active {
    background: var(--vt-color-surface-selected);
    color: var(--vt-color-accent);
  }

  .nav-item.is-active :global(.nav-icon.icon-plugin) {
    color: currentColor;
    opacity: 1;
  }

  :global(.nav-icon) {
    width: 0.9rem;
    height: 0.9rem;
    flex-shrink: 0;
    color: currentColor;
  }
  :global(.nav-icon.icon-plugin) {
    color: var(--vt-color-text-muted);
    opacity: 0.9;
  }

  .nav-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .sidebar-footer {
    margin-top: auto;
    padding: 0.75rem 1.25rem;
    border-top: 1px solid var(--vt-color-border);
  }

  .sidebar-error {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.7rem;
    color: var(--vt-color-danger);
    margin-bottom: 0.25rem;
  }
  :global(.sidebar-error-icon) {
    color: var(--vt-color-danger);
  }

  @media (max-width: 720px) {
    .sidebar {
      width: 100%;
      min-width: 0;
      max-height: 14rem;
      border-right: 0;
      border-bottom: 1px solid var(--vt-color-border);
    }

    .sidebar-resizer {
      display: none;
    }

    .sidebar-header {
      padding: 0.75rem 1rem;
    }

    .sidebar-section {
      max-height: 5.5rem;
      overflow: auto;
    }

    .sidebar-footer {
      display: none;
    }
  }
</style>
