<script>
  import PluginManager from './lib/plugin-manager/PluginManager.svelte';
  import Sidebar from './lib/shell/Sidebar.svelte';
  import CommandPalette from './lib/shell/CommandPalette.svelte';
  import StatusBar from './lib/shell/StatusBar.svelte';
  import ViewContainer from './lib/shell/ViewContainer.svelte';
  import VaultSelection from './lib/shell/VaultSelection.svelte';
  import WorkbenchHost from './lib/shell/WorkbenchHost.svelte';
  import WorkspaceHost from './lib/shell/WorkspaceHost.svelte';
  import TodaySurface from './lib/shell/TodaySurface.svelte';
  import * as App from '../wailsjs/go/api/App';
  import { debug } from './lib/log/debug.js';
  import { onMount } from 'svelte';
  import { tick } from 'svelte';

  let currentView = 'workspace';
  let vaultStatus = { status: 'unknown', path: '', vaultId: '' };
  let needsVaultSelection = false;
  let loading = true;

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
        currentView = 'today';
        emitWorkspaceActive('');
        return;
      }

      workspaceNodes = workspaces.map(workspaceAsNode);
      selectedWorkspaceName = '';
      currentView = 'today';
      emitWorkspaceActive('');
    } catch (e) {
      debug.log('[App] openDefaultWorkspaceRoute ERROR', String(e));
      workspaceNodes = [];
      selectedWorkspaceName = '';
      currentView = 'today';
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
</script>

{#if loading}
  <div class="app-loading">
    <p>Loading Verstak...</p>
  </div>
{:else if needsVaultSelection}
  <VaultSelection />
{:else}
  <main>
    <Sidebar showGlobalSearch={currentView !== 'workspace' && currentView !== 'workspace-empty'} />
    <CommandPalette />

    <section class="content-shell">
      <section class="content scroll-surface">
        {#if currentView === 'plugin-manager'}
          <PluginManager {activeSettingsPluginId} {activeSettingsPanelId} />
        {:else if currentView === 'workbench'}
          <WorkbenchHost {openedResource} />
        {:else if currentView === 'today'}
          <TodaySurface
            workspaceRootPath=""
            workspaceTitle=""
            availableTools={[]}
          />
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

  :global(html),
  :global(body),
  :global(#app) {
    width: 100%;
    height: 100%;
  }

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #1a1a2e;
    color: #e0e0f0;
    overflow: hidden;
  }

  :global(button) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.35rem;
    min-height: 2rem;
    padding: 0.4rem 0.85rem;
    border: 1px solid #1a3a5c;
    border-radius: 6px;
    background: #0f3460;
    color: #e0e0f0;
    font: inherit;
    font-size: 0.85rem;
    font-weight: 600;
    line-height: 1.2;
    cursor: pointer;
    transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease, opacity 0.15s ease;
  }

  :global(button:hover:not(:disabled)) {
    background: #1a3a5c;
    border-color: #4ecca3;
    color: #ffffff;
  }

  :global(button:focus-visible) {
    outline: 2px solid #4ecca3;
    outline-offset: 2px;
  }

  :global(button:disabled) {
    opacity: 0.55;
    cursor: not-allowed;
  }

  :global(.btn-primary) {
    background: #4ecca3;
    border-color: #4ecca3;
    color: #101827;
  }

  :global(.btn-primary:hover:not(:disabled)) {
    background: #63d9b3;
    border-color: #63d9b3;
    color: #101827;
  }

  :global(.btn-secondary) {
    background: #0f3460;
    border-color: #533483;
    color: #e0e0f0;
  }

  :global(.btn-danger) {
    background: #e94560;
    border-color: #e94560;
    color: #ffffff;
  }

  :global(.btn-danger:hover:not(:disabled)) {
    background: #ff5b73;
    border-color: #ff5b73;
  }

  :global(.btn-ghost) {
    background: transparent;
    border-color: transparent;
    color: #a0a0b8;
  }

  :global(.btn-ghost:hover:not(:disabled)) {
    background: rgba(15, 52, 96, 0.55);
    border-color: #0f3460;
    color: #e0e0f0;
  }

  :global(.btn-icon) {
    width: 2rem;
    min-width: 2rem;
    padding: 0;
  }

  :global(.scroll-surface) {
    min-width: 0;
    min-height: 0;
    overflow: auto;
    scrollbar-gutter: stable;
  }

  :global(*) {
    scrollbar-width: thin;
    scrollbar-color: #0f3460 #1a1a2e;
  }

  :global(*::-webkit-scrollbar) {
    width: 8px;
    height: 8px;
  }

  :global(*::-webkit-scrollbar-track) {
    background: #1a1a2e;
  }

  :global(*::-webkit-scrollbar-thumb) {
    background: #0f3460;
    border-radius: 4px;
  }

  :global(*::-webkit-scrollbar-thumb:hover) {
    background: #1a4a7a;
  }

  .app-loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100vh;
    background: #1a1a2e;
    color: #a0a0b8;
    font-size: 1rem;
  }

  main {
    display: flex;
    height: 100vh;
    width: 100%;
    background: #1a1a2e;
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
    padding: clamp(1rem, 2vw, 1.5rem);
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
