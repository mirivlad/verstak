<script>
  import PluginManager from './lib/plugin-manager/PluginManager.svelte';
  import Sidebar from './lib/shell/Sidebar.svelte';
  import CommandPalette from './lib/shell/CommandPalette.svelte';
  import StatusBar from './lib/shell/StatusBar.svelte';
  import ViewContainer from './lib/shell/ViewContainer.svelte';
  import VaultSelection from './lib/shell/VaultSelection.svelte';
  import WorkbenchHost from './lib/shell/WorkbenchHost.svelte';
  import WorkspaceHost from './lib/shell/WorkspaceHost.svelte';
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

  let activeView = null;
  let activeViewPluginId = '';
  let activeSettingsPluginId = '';
  let activeSettingsPanelId = '';
  let openedResource = null;

  let workspaceNodes = [];
  let selectedWorkspaceName = '';
  let activeWorkspaceToolKey = '';
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

  function onOpenSettings(e) {
    debug.log('[App] onOpenSettings:', e.detail.pluginId, e.detail.panelId);
    activeSettingsPluginId = e.detail.pluginId;
    activeSettingsPanelId = e.detail.panelId || '';
    currentView = 'plugin-manager';
    clearWorkspaceSelection();
    pushNavigation();
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
    if (currentView === 'workspace') pushNavigation();
  }

  function onWorkspaceSelected(e) {
    debug.log('[App] onWorkspaceSelected:', e.detail?.workspaceName);
    selectedWorkspaceName = e.detail?.workspaceName || '';
    workspaceNodes = e.detail?.nodes || workspaceNodes;
    if (selectedWorkspaceName) {
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

  function onNavigateBack(e) {
    if (currentView === 'workbench') return;
    if (currentView === 'workspace') {
      const upBtn = document.querySelector('[data-files-action="up"]');
      if (upBtn && !upBtn.disabled) {
        upBtn.click();
        e?.preventDefault?.();
        return;
      }
    }
    if (navigateBack()) e?.preventDefault?.();
  }

  function onNavigateForward(e) {
    if (currentView === 'workbench') return;
    if (currentView === 'workspace') {
      const fwdBtn = document.querySelector('[data-files-action="forward"]');
      if (fwdBtn && !fwdBtn.disabled) {
        fwdBtn.click();
        e?.preventDefault?.();
        return;
      }
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

  // Listen for events
  if (typeof window !== 'undefined') {
    window.addEventListener('verstak:vault-opened', onVaultOpened);
    window.addEventListener('verstak:nav', onNav);
    window.addEventListener('verstak:open-view', onOpenView);
    window.addEventListener('verstak:open-settings', onOpenSettings);
    window.addEventListener('verstak:close-settings', onCloseSettings);
    window.addEventListener('verstak:workbench-opened', onWorkbenchOpened);
    window.addEventListener('verstak:workspace-selected', onWorkspaceSelected);
    window.addEventListener('verstak:workspace-tool-selected', onWorkspaceToolSelected);
    window.addEventListener('verstak:navigate-back', onNavigateBack);
    window.addEventListener('verstak:navigate-forward', onNavigateForward);
    window.addEventListener('verstak:close-workbench', onCloseWorkbench);
    window.addEventListener('keydown', onGlobalKeydown);
    window.addEventListener('pointerdown', onGlobalMouse, true);
    window.addEventListener('mousedown', onGlobalMouse, true);
    window.addEventListener('mouseup', onGlobalMouse, true);
    window.addEventListener('auxclick', onGlobalMouse, true);
  }

  onMount(async () => {
    await checkVault();
    pushNavigation();
  });

  onDestroy(unsubscribeLocale);
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
      showGlobalSearch={currentView !== 'workspace' && currentView !== 'workspace-empty'}
      {activeView}
      {activeViewPluginId}
    />
    <CommandPalette />

    <section class="content-shell">
      <section class="content scroll-surface">
        {#if currentView === 'plugin-manager'}
          <PluginManager {activeSettingsPluginId} {activeSettingsPanelId} />
        {:else if currentView === 'workbench'}
          <WorkbenchHost {openedResource} />
        {:else if currentView === 'workspace' || currentView === 'workspace-empty'}
          <WorkspaceHost
            selectedWorkspaceName={selectedWorkspaceName}
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
  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(:root) {
    --vt-color-background: #101020;
    --vt-color-surface: #15152c;
    --vt-color-surface-muted: #111629;
    --vt-color-surface-hover: #1b2440;
    --vt-color-surface-selected: rgba(78, 204, 163, 0.14);
    --vt-color-border: #202b46;
    --vt-color-border-strong: #2c456a;
    --vt-color-text-primary: #f4f7fb;
    --vt-color-text-secondary: #b7c0d4;
    --vt-color-text-muted: #7f8aa3;
    --vt-color-accent: #4ecca3;
    --vt-color-accent-muted: rgba(78, 204, 163, 0.14);
    --vt-color-danger: #e94560;
    --vt-color-danger-muted: rgba(233, 69, 96, 0.14);
    --vt-color-warning: #ffc857;
    --vt-color-warning-muted: rgba(255, 200, 87, 0.14);
    --vt-color-success: #4ecca3;
    --vt-space-1: 0.25rem;
    --vt-space-2: 0.5rem;
    --vt-space-3: 0.75rem;
    --vt-space-4: 1rem;
    --vt-space-6: 1.5rem;
    --vt-space-8: 2rem;
    --vt-radius-sm: 4px;
    --vt-radius-md: 6px;
    --vt-radius-lg: 8px;
    --vt-font-xs: 0.72rem;
    --vt-font-sm: 0.8rem;
    --vt-font-md: 0.88rem;
    --vt-font-lg: 1rem;
    --vt-focus-ring: 0 0 0 2px rgba(78, 204, 163, 0.34);
    --vt-elevation-menu: 0 14px 32px rgba(0, 0, 0, 0.42);
  }

  :global(html),
  :global(body),
  :global(#app) {
    width: 100%;
    height: 100%;
  }

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: var(--vt-color-background);
    color: var(--vt-color-text-primary);
    overflow: hidden;
  }

  :global(button) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    min-height: 2rem;
    padding: 0.4rem 0.85rem;
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface-hover);
    color: var(--vt-color-text-primary);
    font: inherit;
    font-size: var(--vt-font-sm);
    font-weight: 600;
    line-height: 1.2;
    cursor: pointer;
    transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease, opacity 0.15s ease;
  }

  :global(button:hover:not(:disabled)) {
    background: #203050;
    border-color: var(--vt-color-accent);
    color: #ffffff;
  }

  :global(button:focus-visible) {
    outline: 0;
    box-shadow: var(--vt-focus-ring);
  }

  :global(button:disabled) {
    opacity: 0.55;
    cursor: not-allowed;
  }

  :global(.btn-primary) {
    background: var(--vt-color-accent);
    border-color: var(--vt-color-accent);
    color: #101827;
  }

  :global(.btn-primary:hover:not(:disabled)) {
    background: #63d9b3;
    border-color: #63d9b3;
    color: #101827;
  }

  :global(.btn-secondary) {
    background: var(--vt-color-surface-hover);
    border-color: var(--vt-color-border-strong);
    color: var(--vt-color-text-primary);
  }

  :global(.btn-danger) {
    background: var(--vt-color-danger);
    border-color: var(--vt-color-danger);
    color: #ffffff;
  }

  :global(.btn-danger:hover:not(:disabled)) {
    background: #ff5b73;
    border-color: #ff5b73;
  }

  :global(.btn-ghost) {
    background: transparent;
    border-color: transparent;
    color: var(--vt-color-text-secondary);
  }

  :global(.btn-ghost:hover:not(:disabled)) {
    background: var(--vt-color-surface-hover);
    border-color: var(--vt-color-border);
    color: var(--vt-color-text-primary);
  }

  :global(.btn-icon) {
    width: 2rem;
    min-width: 2rem;
    padding: 0;
  }

  :global(.vt-page) {
    min-width: 0;
    min-height: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--vt-color-background);
    color: var(--vt-color-text-primary);
  }

  :global(.vt-page-header),
  :global(.vt-toolbar) {
    min-height: 2.75rem;
    display: flex;
    align-items: center;
    gap: var(--vt-space-2);
    padding: var(--vt-space-2) var(--vt-space-3);
    border-bottom: 1px solid var(--vt-color-border);
    background: var(--vt-color-surface-muted);
    flex-shrink: 0;
  }

  :global(.vt-page-title) {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-primary);
    font-size: var(--vt-font-lg);
    font-weight: 650;
  }

  :global(.vt-page-subtitle) {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-muted);
    font-size: var(--vt-font-sm);
  }

  :global(.vt-toolbar-group) {
    display: inline-flex;
    align-items: center;
    gap: var(--vt-space-1);
    padding: 0 var(--vt-space-2) 0 0;
    border-right: 1px solid var(--vt-color-border);
  }

  :global(.vt-toolbar-group:last-child) {
    border-right: 0;
    padding-right: 0;
  }

  :global(.vt-button) {
    min-height: 2rem;
    border-radius: var(--vt-radius-md);
  }

  :global(.vt-icon-button) {
    width: 2rem;
    min-width: 2rem;
    height: 2rem;
    min-height: 2rem;
    padding: 0;
  }

  :global(.vt-tabbar) {
    display: flex;
    align-items: center;
    gap: var(--vt-space-1);
    padding: var(--vt-space-1) var(--vt-space-3) 0;
    border-bottom: 1px solid var(--vt-color-border);
    background: #12162a;
    overflow-x: auto;
    flex-shrink: 0;
  }

  :global(.vt-tab) {
    flex-shrink: 0;
    min-height: 2.1rem;
    padding: 0.38rem 0.78rem;
    border: 1px solid transparent;
    border-bottom: 0;
    border-radius: var(--vt-radius-md) var(--vt-radius-md) 0 0;
    background: transparent;
    color: var(--vt-color-text-muted);
    font-size: var(--vt-font-sm);
  }

  :global(.vt-tab:hover:not(:disabled)) {
    background: var(--vt-color-surface-hover);
    border-color: transparent;
    color: var(--vt-color-text-primary);
  }

  :global(.vt-tab.is-active),
  :global(.vt-tab.active) {
    background: var(--vt-color-background);
    border-color: var(--vt-color-border);
    color: var(--vt-color-accent);
  }

  :global(.vt-list-row) {
    border-bottom: 1px solid rgba(32, 43, 70, 0.72);
    color: var(--vt-color-text-primary);
  }

  :global(.vt-list-row:hover) {
    background: var(--vt-color-surface-hover);
  }

  :global(.vt-list-row.selected),
  :global(.vt-list-row.active),
  :global(.vt-list-row.is-selected) {
    background: var(--vt-color-surface-selected);
    box-shadow: inset 2px 0 0 var(--vt-color-accent);
  }

  :global(.vt-card) {
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
  }

  :global(.vt-split-pane) {
    display: grid;
    grid-template-columns: minmax(16rem, 22rem) minmax(0, 1fr);
    min-height: 0;
    height: 100%;
    background: var(--vt-color-background);
  }

  :global(.vt-split-list) {
    min-height: 0;
    overflow: auto;
    border-right: 1px solid var(--vt-color-border);
    background: var(--vt-color-surface-muted);
  }

  :global(.vt-split-detail) {
    min-width: 0;
    min-height: 0;
    overflow: auto;
    padding: var(--vt-space-4);
  }

  :global(.vt-empty-state) {
    min-height: 9rem;
    display: flex;
    flex: 1;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: var(--vt-space-2);
    padding: var(--vt-space-8);
    color: var(--vt-color-text-muted);
    font-size: var(--vt-font-sm);
    line-height: 1.45;
    text-align: center;
  }

  :global(.vt-empty-title) {
    color: var(--vt-color-text-secondary);
    font-size: var(--vt-font-md);
    font-weight: 650;
  }

  :global(.vt-inline-alert) {
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface-muted);
    color: var(--vt-color-text-secondary);
    padding: var(--vt-space-3);
    font-size: var(--vt-font-sm);
    line-height: 1.45;
  }

  :global(.vt-inline-alert.error) {
    border-color: rgba(233, 69, 96, 0.55);
    background: var(--vt-color-danger-muted);
    color: #ffc6ce;
  }

  :global(.vt-badge) {
    display: inline-flex;
    align-items: center;
    min-height: 1.25rem;
    padding: 0.1rem 0.4rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-sm);
    background: var(--vt-color-surface-muted);
    color: var(--vt-color-text-muted);
    font-size: var(--vt-font-xs);
    font-weight: 650;
    line-height: 1;
  }

  :global(.vt-badge.accent) {
    border-color: rgba(78, 204, 163, 0.4);
    background: var(--vt-color-accent-muted);
    color: var(--vt-color-accent);
  }

  :global(.vt-menu) {
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface);
    color: var(--vt-color-text-primary);
    box-shadow: var(--vt-elevation-menu);
  }

  :global(.vt-menu-item) {
    display: flex;
    align-items: center;
    gap: var(--vt-space-2);
    padding: 0.42rem 0.7rem;
    color: var(--vt-color-text-secondary);
    cursor: pointer;
  }

  :global(.vt-menu-item:hover) {
    background: var(--vt-color-surface-hover);
    color: var(--vt-color-text-primary);
  }

  :global(.vt-menu-item.danger) {
    color: #ff9aaa;
  }

  :global(.scroll-surface) {
    min-width: 0;
    min-height: 0;
    overflow: auto;
    scrollbar-gutter: stable;
  }

  :global(*) {
    scrollbar-width: thin;
    scrollbar-color: var(--vt-color-border-strong) var(--vt-color-background);
  }

  :global(*::-webkit-scrollbar) {
    width: 8px;
    height: 8px;
  }

  :global(*::-webkit-scrollbar-track) {
    background: var(--vt-color-background);
  }

  :global(*::-webkit-scrollbar-thumb) {
    background: var(--vt-color-border-strong);
    border-radius: var(--vt-radius-sm);
  }

  :global(*::-webkit-scrollbar-thumb:hover) {
    background: #1a4a7a;
  }

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
