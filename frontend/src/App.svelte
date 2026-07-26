<script>
  import PluginManager from './lib/plugin-manager/PluginManager.svelte';
  import Sidebar from './lib/shell/Sidebar.svelte';
  import GlobalSearch from './lib/shell/GlobalSearch.svelte';
  import CommandPalette from './lib/shell/CommandPalette.svelte';
  import Icon from './lib/ui/Icon.svelte';
  import StatusBar from './lib/shell/StatusBar.svelte';
  import ViewContainer from './lib/shell/ViewContainer.svelte';
  import VaultSelection from './lib/shell/VaultSelection.svelte';
  import WorkbenchHost from './lib/shell/WorkbenchHost.svelte';
  import WorkspaceHost from './lib/shell/WorkspaceHost.svelte';
  import SettingsWindow from './lib/settings/SettingsWindow.svelte';
  import { offerNavigation } from './lib/shell/navigation-handlers.js';
  import * as App from '../wailsjs/go/api/App';
  import { debug } from './lib/log/debug.js';
  import { onDestroy, onMount, tick } from 'svelte';
  import { i18n } from './lib/i18n/index.js';

  let currentView = 'workspace';
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let needsVaultSelection = false;
  let loading = true;
  let locale = i18n.getLocale();
  const unsubscribeLocale = i18n.subscribe((nextLocale) => { locale = nextLocale; });
  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);

  $: defaultContentTitle = currentView === 'plugin-manager' ? tr('settings.pluginManager')
    : currentView === 'workbench' ? (openedResource?.request?.path?.split('/').filter(Boolean).pop() || '')
    : currentView === 'workspace' && selectedWorkspaceName ? selectedWorkspaceName
    : '';

  let activeView = null;
  let activeViewPluginId = '';
  let activeSettingsPluginId = '';
  let requestedSettingsSection = '';
  let activeSettingsPanelId = '';
  let openedResource = null;

  // Common header title emitted by child views
  let contentTitle = '';
  let contentTitleSub = '';
  let contentTitleIcon = '';

  let workspaceNodes = [];
  let selectedWorkspaceName = '';
  let selectedWorkspaceId = '';
  let activeWorkspaceToolKey = '';
  let activeWorkspaceToolPluginId = '';
  let navigationStack = [];
  let navigationIndex = -1;
  let applyingNavigation = false;
  let lastMouseHistoryDirection = '';
  let lastMouseHistoryAt = 0;

  function flog(msg) {
    App.WriteFrontendLog('App', msg);
  }

  function resultOrError(response, fallbackValue) {
    return typeof response === 'string' ? [fallbackValue, response] : [response, ''];
  }

  function workspaceName(workspace) {
    return String(workspace?.name || workspace?.rootPath || workspace?.id || '');
  }

  function workspaceAsNode(workspace, order) {
    const name = workspaceName(workspace);
    return {
      id: name,
      workspaceId: workspace?.id || workspace?.workspaceId || '',
      type: workspace?.type || 'space',
      title: workspace?.title || name,
      name,
      rootPath: workspace?.rootPath || name,
      status: workspace?.status || 'active',
      order,
    };
  }

  function emitWorkspaceActive(name) {
    window.dispatchEvent(new CustomEvent('verstak:workspace-active-changed', {
      detail: { workspaceName: name || '' }
    }));
  }

  function clearWorkspaceSelection() {
    selectedWorkspaceName = '';
    selectedWorkspaceId = '';
    emitWorkspaceActive('');
  }

  async function openDefaultWorkspaceRoute() {
    try {
      const [workspaces, err] = resultOrError(await App.ListWorkspaces(), []);
      if (err || !workspaces || workspaces.length === 0) {
        workspaceNodes = [];
        selectedWorkspaceName = '';
        currentView = 'workspace-empty';
        emitWorkspaceActive('');
        return;
      }

      workspaceNodes = workspaces.map(workspaceAsNode);
      let currentWorkspace = null;
      try {
        currentWorkspace = await App.GetCurrentWorkspace();
      } catch {
        currentWorkspace = null;
      }
      const currentName = workspaceName(currentWorkspace);
      const selected = workspaces.find((workspace) => workspaceName(workspace) === currentName) || workspaces[0];
      selectedWorkspaceName = workspaceName(selected);
      if (selectedWorkspaceName) {
        try { await App.SetCurrentWorkspace(selectedWorkspaceName); } catch {}
        currentView = 'workspace';
      } else {
        currentView = 'workspace-empty';
      }
      emitWorkspaceActive(selectedWorkspaceName);
    } catch (e) {
      debug.log('[App] openDefaultWorkspaceRoute ERROR', String(e));
      workspaceNodes = [];
      selectedWorkspaceName = '';
      currentView = 'workspace-empty';
      emitWorkspaceActive('');
    }
  }

  function currentSnapshot() {
    return {
      currentView,
      activeView,
      activeViewPluginId,
      activeSettingsPluginId,
      activeSettingsPanelId,
      openedResource,
      selectedWorkspaceName,
      activeWorkspaceToolKey,
    };
  }

  function sameSnapshot(a, b) {
    return JSON.stringify(a) === JSON.stringify(b);
  }

  function pushNavigation(snapshot = currentSnapshot()) {
    if (applyingNavigation) return;
    if (navigationIndex >= 0 && sameSnapshot(navigationStack[navigationIndex], snapshot)) return;
    if (navigationIndex < navigationStack.length - 1) {
      navigationStack = navigationStack.slice(0, navigationIndex + 1);
    }
    navigationStack = [...navigationStack, snapshot];
    navigationIndex = navigationStack.length - 1;
  }

  function applySnapshot(snapshot) {
    applyingNavigation = true;
    currentView = snapshot.currentView;
    activeView = snapshot.activeView;
    activeViewPluginId = snapshot.activeViewPluginId;
    activeSettingsPluginId = snapshot.activeSettingsPluginId;
    activeSettingsPanelId = snapshot.activeSettingsPanelId;
    openedResource = snapshot.openedResource;
    selectedWorkspaceName = snapshot.selectedWorkspaceName;
    activeWorkspaceToolKey = snapshot.activeWorkspaceToolKey || '';
    emitWorkspaceActive(currentView === 'workspace' ? selectedWorkspaceName : '');
    applyingNavigation = false;
  }

  function navigateBack() {
    if (navigationIndex <= 0) return false;
    navigationIndex -= 1;
    applySnapshot(navigationStack[navigationIndex]);
    return true;
  }

  function navigateForward() {
    if (navigationIndex >= navigationStack.length - 1) return false;
    navigationIndex += 1;
    applySnapshot(navigationStack[navigationIndex]);
    return true;
  }

  function mouseHistoryDirection(event) {
    if (currentView === 'workspace') return '';
    if (event.button === 3 || event.button === 8 || event.buttons === 8 || event.buttons === 128 || event.which === 8) return 'back';
    if (event.button === 4 || event.button === 9 || event.buttons === 16 || event.buttons === 256 || event.which === 9) return 'forward';
    return '';
  }

  function keyHistoryDirection(event) {
    if (currentView === 'workspace' || currentView === 'workbench') return '';
    const key = event.key || '';
    if (event.altKey && key === 'ArrowLeft') return 'back';
    if (event.altKey && key === 'ArrowRight') return 'forward';
    if (key === 'BrowserBack' || key === 'XF86Back') return 'back';
    if (key === 'BrowserForward' || key === 'XF86Forward') return 'forward';
    if (event.keyCode === 166) return 'back';
    if (event.keyCode === 167) return 'forward';
    return '';
  }

  function handleHistoryRequest(direction, event) {
    if (!direction || event?.defaultPrevented) return;
    if (event?.type === 'mousedown' || event?.type === 'mouseup' || event?.type === 'auxclick' || event?.type === 'pointerdown') {
      const now = Date.now();
      if (direction === lastMouseHistoryDirection && now - lastMouseHistoryAt < 120) return;
      lastMouseHistoryDirection = direction;
      lastMouseHistoryAt = now;
      debug.log('[App] mouse history event', {
        type: event.type,
        direction,
        button: event.button,
        buttons: event.buttons,
        which: event.which,
        pointerType: event.pointerType || '',
        currentView,
      });
    }
    const moved = direction === 'back' ? navigateBack() : navigateForward();
    if (moved && event) {
      event.preventDefault();
      event.stopPropagation();
    }
  }

  async function checkVault() {
    debug.log('[App] checkVault: START');
    flog('checkVault: START');
    loading = true;
    try {
      debug.log('[App] checkVault: calling GetAppSettings...');
      const settings = await App.GetAppSettings();
      debug.log('[App] checkVault: GetAppSettings returned', settings);
      flog('checkVault: GetAppSettings returned');
      if (settings?.debug) debug.enable({ persist: false });

      debug.log('[App] checkVault: calling GetVaultStatus...');
      vaultStatus = await App.GetVaultStatus() || { status: 'unknown', path: '', vaultId: '' };
      debug.log('[App] checkVault: GetVaultStatus returned', vaultStatus);
      flog('checkVault: vaultStatus=' + vaultStatus.status);

      if (!settings.currentVaultPath || vaultStatus.status !== 'open') {
        debug.log('[App] checkVault: vault not open, needsVaultSelection=true');
        flog('checkVault: needsVaultSelection=true');
        needsVaultSelection = true;
      } else {
        debug.log('[App] checkVault: vault open, needsVaultSelection=false');
        flog('checkVault: needsVaultSelection=false');
        needsVaultSelection = false;
        await openDefaultWorkspaceRoute();
      }
    } catch (e) {
      debug.log('[App] checkVault: ERROR', String(e));
      flog('checkVault: ERROR: ' + String(e));
      console.error('[App] startup check failed:', e);
      needsVaultSelection = true;
    }
    loading = false;
    await tick();
    debug.log('[App] checkVault: END, loading=false');
    flog('checkVault: END, loading=false');
  }

  async function onVaultOpened() {
    debug.log('[App] onVaultOpened');
    needsVaultSelection = false;
    vaultStatus = { status: 'open', path: '', vaultId: '' };
    await openDefaultWorkspaceRoute();
    pushNavigation();
  }

  function onNav(e) {
    debug.log('[App] onNav:', e.detail.viewId);
    currentView = e.detail.viewId;
    if (currentView !== 'workspace') clearWorkspaceSelection();
    pushNavigation();
  }

  function onOpenView(e) {
    debug.log('[App] onOpenView:', e.detail.viewId, 'plugin:', e.detail.pluginId);
    activeView = e.detail.viewId;
    activeViewPluginId = e.detail.pluginId || '';
    currentView = 'plugin-view';
    clearWorkspaceSelection();
    pushNavigation();
  }

  // A plugin asking to show its settings, and the gear in the status bar, land
  // in the same place. Plugin settings used to open as a modal inside the
  // Plugin Manager, which is a page about installing plugins, not configuring
  // them.
  function onOpenSettings(e) {
    debug.log('[App] onOpenSettings:', e.detail?.pluginId, e.detail?.panelId);
    const pluginId = e.detail?.pluginId || '';
    const panelId = e.detail?.panelId || '';
    activeSettingsPluginId = pluginId;
    activeSettingsPanelId = panelId;
    requestedSettingsSection = pluginId ? sectionIdForPanel(pluginId, panelId) : '';
    currentView = 'settings';
    clearWorkspaceSelection();
    pushNavigation();
  }

  // A section id has to be derivable without the contribution list, because the
  // request arrives before the settings window has loaded one.
  function sectionIdForPanel(pluginId, panelId) {
    return panelId ? `plugin:${pluginId}:${panelId}` : `plugin:${pluginId}:`;
  }

  function onCloseSettingsWindow() {
    if (currentView !== 'settings') return;
    requestedSettingsSection = '';
    if (!navigateBack()) {
      currentView = selectedWorkspaceName ? 'workspace' : 'plugin-view';
      pushNavigation();
    }
  }

  function onWorkbenchOpened(e) {
    debug.log('[App] onWorkbenchOpened:', e.detail?.request?.path, e.detail?.providerId);
    if (currentView === 'workspace') pushNavigation();
    openedResource = e.detail;
    currentView = 'workbench';
    pushNavigation();
  }

  function onWorkspaceToolSelected(e) {
    activeWorkspaceToolKey = e.detail?.toolKey || '';
    activeWorkspaceToolPluginId = e.detail?.pluginId || '';
    if (currentView === 'workspace') pushNavigation();
  }

  function onWorkspaceSelected(e) {
    const name = e.detail?.workspaceName || e.detail?.workspaceRootPath || '';
    const id = e.detail?.workspaceId || '';
    debug.log('[App] onWorkspaceSelected:', name, id);
    selectedWorkspaceName = name;
    selectedWorkspaceId = id;
    workspaceNodes = e.detail?.nodes || workspaceNodes;
    if (selectedWorkspaceName || id) {
      activeView = null;
      activeViewPluginId = '';
      activeSettingsPluginId = '';
      activeSettingsPanelId = '';
      openedResource = null;
      currentView = 'workspace';
      emitWorkspaceActive(selectedWorkspaceName);
      pushNavigation();
    }
  }

  function onCloseSettings() {
    debug.log('[App] onCloseSettings');
    activeSettingsPluginId = '';
    activeSettingsPanelId = '';
  }

  // The tool on screen gets first refusal on back and forward: inside a file
  // browser they mean the folder history, not the shell's view history. Only
  // when it declines does the shell move its own history.
  function onNavigateBack(e) {
    if (currentView === 'workbench') return;
    if (currentView === 'workspace' && offerNavigation(activeWorkspaceToolPluginId, 'back')) {
      e?.preventDefault?.();
      return;
    }
    if (navigateBack()) e?.preventDefault?.();
  }

  function onNavigateForward(e) {
    if (currentView === 'workbench') return;
    if (currentView === 'workspace' && offerNavigation(activeWorkspaceToolPluginId, 'forward')) {
      e?.preventDefault?.();
      return;
    }
    if (navigateForward()) e?.preventDefault?.();
  }

  function onCloseWorkbench(e) {
    if (currentView !== 'workbench') return;
    if (!navigateBack() && selectedWorkspaceName) {
      currentView = 'workspace';
      pushNavigation();
    }
    e?.preventDefault?.();
  }

  function onGlobalKeydown(e) {
    handleHistoryRequest(keyHistoryDirection(e), e);
  }

  function onGlobalMouse(e) {
    handleHistoryRequest(mouseHistoryDirection(e), e);
  }

  function onContentTitleChanged(e) {
    contentTitle = e.detail?.title || '';
    contentTitleSub = e.detail?.subtitle || '';
    contentTitleIcon = e.detail?.icon || '';
  }

  // Registered on mount and removed on destroy, as a pair. They used to be
  // added during component initialisation with no matching removal: harmless
  // while the shell is the only App there will ever be, but it meant every test
  // that mounted App left another live copy of these handlers behind, so one
  // keystroke ran the shell's navigation several times over.
  const windowListeners = [
    ['verstak:vault-opened', onVaultOpened, false],
    ['verstak:nav', onNav, false],
    ['verstak:open-view', onOpenView, false],
    ['verstak:open-settings', onOpenSettings, false],
    ['verstak:close-settings', onCloseSettings, false],
    ['verstak:close-settings-window', onCloseSettingsWindow, false],
    ['verstak:workbench-opened', onWorkbenchOpened, false],
    ['verstak:workspace-selected', onWorkspaceSelected, false],
    ['verstak:workspace-tool-selected', onWorkspaceToolSelected, false],
    ['verstak:navigate-back', onNavigateBack, false],
    ['verstak:navigate-forward', onNavigateForward, false],
    ['verstak:close-workbench', onCloseWorkbench, false],
    ['verstak:content-title-changed', onContentTitleChanged, false],
    ['keydown', onGlobalKeydown, false],
    ['pointerdown', onGlobalMouse, true],
    ['mousedown', onGlobalMouse, true],
    ['mouseup', onGlobalMouse, true],
    ['auxclick', onGlobalMouse, true],
  ];

  onMount(async () => {
    for (const [name, handler, capture] of windowListeners) {
      window.addEventListener(name, handler, capture);
    }
    await checkVault();
    pushNavigation();
  });

  onDestroy(() => {
    unsubscribeLocale();
    if (typeof window === 'undefined') return;
    for (const [name, handler, capture] of windowListeners) {
      window.removeEventListener(name, handler, capture);
    }
  });
</script>

{#if loading}
  <div class="app-loading">
    <p>{tr('app.loading')}</p>
  </div>
{:else if needsVaultSelection}
  <VaultSelection />
{:else}
  <main>
    <Sidebar
      {activeView}
      {activeViewPluginId}
    />
    <CommandPalette />

    <section class="content-shell">
      <header class="main-content-header" data-main-content-header>
        <div class="main-content-title">
          {#if contentTitleIcon}<Icon name={contentTitleIcon} size={16} class="main-content-title-icon" />{/if}
          <span class="main-content-title-text">{contentTitle || defaultContentTitle || tr('workspace.select')}</span>
          {#if contentTitleSub}
            <span class="main-content-title-sub">{contentTitleSub}</span>
          {/if}
        </div>
        <div class="main-content-actions">
          {#if currentView === 'workbench'}
            <button class="close-btn btn-ghost btn-icon" type="button" title={tr('common.close')} aria-label={tr('common.close')} on:click={() => window.dispatchEvent(new CustomEvent('verstak:close-workbench', { cancelable: true }))}>
              <Icon name="x" size={18} />
            </button>
          {/if}
          <GlobalSearch />
        </div>
      </header>
      <section class="content scroll-surface">
        {#if currentView === 'settings'}
          <SettingsWindow requestedSection={requestedSettingsSection} />
        {:else if currentView === 'plugin-manager'}
          <PluginManager {activeSettingsPluginId} {activeSettingsPanelId} />
        {:else if currentView === 'workbench'}
          <WorkbenchHost {openedResource} />
        {:else if currentView === 'workspace' || currentView === 'workspace-empty'}
          <WorkspaceHost
            selectedWorkspaceName={selectedWorkspaceName}
            selectedWorkspaceId={selectedWorkspaceId}
            nodes={workspaceNodes}
            bind:activeToolKey={activeWorkspaceToolKey}
          />
        {:else}
          <ViewContainer {activeView} {activeViewPluginId} />
        {/if}
      </section>
      <StatusBar />
    </section>
  </main>
{/if}

<style>
  .app-loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100vh;
    background: var(--vt-color-background);
    color: var(--vt-color-text-muted);
    font-size: 1rem;
  }

  main {
    display: flex;
    height: 100vh;
    width: 100%;
    background: var(--vt-color-background);
    overflow: hidden;
  }

  .content-shell {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .main-content-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: var(--vt-space-2) var(--vt-space-4);
    border-bottom: 1px solid var(--vt-color-border);
    background: var(--vt-color-surface-muted);
    flex-shrink: 0;
    min-height: 2.75rem;
  }

  .main-content-title {
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .main-content-title-icon { width: 1rem; height: 1rem; flex-shrink: 0; color: var(--vt-color-text-muted); }
  .main-content-title-text {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-primary);
    font-size: var(--vt-font-lg);
    font-weight: 650;
  }

  .main-content-title-sub {
    color: var(--vt-color-text-muted);
    font-size: var(--vt-font-sm);
    flex-shrink: 0;
  }

  .main-content-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-shrink: 0;
  }

  .close-btn {
    width: 2rem;
    height: 2rem;
    min-height: 0;
    padding: 0;
    border-radius: var(--vt-radius-md);
    color: var(--vt-color-text-secondary);
    flex-shrink: 0;
    cursor: pointer;
  }

  .content {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 0;
  }

  @media (max-width: 720px) {
    main {
      flex-direction: column;
    }

    .content {
      padding: 0.75rem;
    }
  }
</style>
