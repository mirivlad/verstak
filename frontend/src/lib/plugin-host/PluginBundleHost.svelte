<script>
  import { onMount, onDestroy } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import { i18n } from '../i18n/index.js';

  import { acquirePluginStyle, createPluginAPI, loadPluginBundle } from './VerstakPluginAPI.js';

  export let pluginId = null;
  export let componentId = null;
  export let viewPluginId = null;
  export let componentProps = {};

  let loadState = 'idle'; // idle | loading | loaded | error
  let pluginInfo = null;
  let errorText = '';
  let mountContainer = null;
  let currentPluginId = null;
  let currentComponent = null;
  let currentAPI = null;
  let releaseCurrentStyle = null;
  let currentPropsKey = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;

  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);

  $: activePluginId = pluginId || viewPluginId;
  $: activeComponent = componentId;
  $: propsKey = JSON.stringify(componentProps || {});

  // React to changes — reload on view change
  $: if (activePluginId && activeComponent) {
    loadAndMount(activePluginId, activeComponent, propsKey);
  } else if (!activePluginId) {
    cleanup();
    loadState = 'idle';
  }

  onDestroy(() => {
    unsubscribeLocale?.();
    cleanup();
  });

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => locale = nextLocale);
  });

  function cleanup() {
    const reg = window.__VERSTAK_PLUGIN_REGISTRY__;
    if (currentPluginId && currentComponent && reg && reg[currentPluginId]) {
      const comp = reg[currentPluginId][currentComponent];
      if (comp && comp.unmount && mountContainer) {
        try {
          comp.unmount(mountContainer);
        } catch (e) {
          console.error('[PluginBundleHost] unmount error:', e);
        }
      }
    }
    if (currentAPI && typeof currentAPI.dispose === 'function') {
      try {
        currentAPI.dispose();
      } catch (e) {
        console.error('[PluginBundleHost] API dispose error:', e);
      }
    }
    if (releaseCurrentStyle) {
      releaseCurrentStyle();
      releaseCurrentStyle = null;
    }
    if (mountContainer) {
      mountContainer.innerHTML = '';
    }
    currentPluginId = null;
    currentComponent = null;
    currentAPI = null;
    currentPropsKey = '';
  }

  export function dispose() {
    cleanup();
  }

  function reportError(key, fallback, details) {
    console.warn('[PluginBundleHost] ' + key + ':', details);
    return tr(key, undefined, fallback);
  }

  function bundleErrorText(error) {
    switch (error && error.code) {
      case 'no-frontend':
        return tr('bundle.noFrontend');
      case 'not-found':
        return tr('bundle.notFound');
      case 'registration':
        return tr('bundle.registrationMissing');
      case 'execution':
        return reportError('bundle.executionError', 'Could not start the plugin interface. Please try again.', error);
      default:
        return reportError('bundle.loadFailed', 'Could not load the plugin interface. Please try again.', error);
    }
  }

  function unpackBackendResult(result) {
    if (Array.isArray(result) && result.length === 2 && (typeof result[1] === 'string' || result[1] == null)) {
      if (result[1]) throw new Error(result[1]);
      return result[0];
    }
    return result;
  }

  // The shell already owns the canonical UUID Deal tree. Expose that same
  // read-only snapshot through the public plugin API instead of forcing a
  // plugin to infer Deal structure by walking vault files. `workspaces.list`
  // remains intentionally filtered to Deals where that plugin contributes a
  // workspace tool; `tree` is the discovery counterpart.
  function attachWorkspaceTreeAPI(api) {
    if (!api?.workspaces || typeof api.workspaces.tree === 'function') return api;
    api.workspaces.tree = async function() {
      const snapshot = unpackBackendResult(await App.GetWorkspaceTreeV2());
      if (!snapshot || !Array.isArray(snapshot.roots)) {
        return { roots: [], currentWorkspaceId: '', revision: 0, warnings: [] };
      }
      return {
        roots: snapshot.roots,
        currentWorkspaceId: snapshot.currentWorkspaceId || '',
        revision: Number(snapshot.revision || 0),
        warnings: Array.isArray(snapshot.warnings) ? snapshot.warnings : []
      };
    };
    return api;
  }

  async function loadAndMount(pId, compId, nextPropsKey) {
    // If same plugin+component and already mounted, skip
    if (currentPluginId === pId && currentComponent === compId && currentPropsKey === nextPropsKey && loadState === 'loaded') {
      return;
    }

    // Cleanup previous
    cleanup();

    loadState = 'loading';
    errorText = '';
    currentPluginId = pId;
    currentComponent = compId;
    currentPropsKey = nextPropsKey;

    try {
      let loaded;
      try {
        loaded = await loadPluginBundle(pId);
      } catch (bundleError) {
        pluginInfo = bundleError && bundleError.info ? bundleError.info : null;
        loadState = 'error';
        errorText = bundleErrorText(bundleError);
        return;
      }
      const info = loaded.info;
      pluginInfo = info;

      try {
        await i18n.loadPlugin(pId, info.localization);
      } catch (catalogError) {
        console.warn(`[PluginBundleHost] localization unavailable for ${pId}:`, catalogError);
      }

      if (info.style) {
        const releaseStyle = await acquirePluginStyle(pId, info.style);
        if (currentPluginId !== pId || currentComponent !== compId) {
          releaseStyle();
          return;
        }
        releaseCurrentStyle = releaseStyle;
      }

      // Find the component
      const components = loaded.components || {};
      const comp = components[compId];
      if (!comp || !comp.mount) {
        loadState = 'error';
        errorText = tr('bundle.componentMissing', undefined, 'The requested plugin interface is unavailable.');
        return;
      }

      // Create API. The Deal-tree addition is a shell-owned public read API,
      // not a plugin-specific privileged path.
      const api = attachWorkspaceTreeAPI(createPluginAPI(pId));
      currentAPI = api;

      // Mount component
      if (!mountContainer) {
        // Container must exist in DOM — wait for next tick
        await new Promise(r => requestAnimationFrame(r));
      }
      if (mountContainer) {
        try {
          comp.mount(mountContainer, Object.assign({ componentId: compId }, componentProps || {}), api);
          loadState = 'loaded';
          errorText = '';
        } catch (e) {
          loadState = 'error';
          errorText = reportError('bundle.mountError', 'Could not open the plugin interface. Please try again.', e);
        }
      } else {
        loadState = 'error';
        errorText = tr('bundle.mountUnavailable');
      }
    } catch (e) {
      loadState = 'error';
      errorText = reportError('bundle.unexpectedError', 'Could not open the plugin interface. Please try again.', e);
    }
  }

  function getComponentList() {
    const reg = window.__VERSTAK_PLUGIN_REGISTRY__;
    if (!reg || !currentPluginId || !reg[currentPluginId]) return [];
    return Object.keys(reg[currentPluginId]);
  }
</script>

<div class="plugin-bundle-host">
  {#if loadState === 'idle'}
    <div class="host-state idle">
      <p>{tr('pluginView.select')}</p>
    </div>

  {:else if loadState === 'error'}
    <div class="host-state error">
      <Icon name="warning" size={24} class="error-icon" />
      <p class="error-title">{tr('pluginView.error')}</p>
      <p class="error-message">{errorText || tr('bundle.unknownError')}</p>
      <details class="error-details">
        <summary>{tr('common.details')}</summary>
        <p><strong>{tr('common.plugin')}:</strong> {currentPluginId || tr('common.unknown')}</p>
        <p><strong>{tr('common.component')}:</strong> {currentComponent || tr('common.unknown')}</p>
        {#if pluginInfo}
          <p class="error-meta">{tr('bundle.frontendEntry')}: {pluginInfo.entry || tr('common.none')}</p>
        {/if}
        {#if getComponentList().length > 0}
          <p class="error-meta">{tr('bundle.availableComponents')}: {getComponentList().join(', ')}</p>
        {/if}
      </details>
    </div>

  {:else}
    {#if loadState === 'loading'}
      <div class="host-state loading">
        <div class="spinner"></div>
        <p>{tr('bundle.loading')}</p>
      </div>
    {/if}
    <div
      class="plugin-mount-container"
      class:mount-hidden={loadState !== 'loaded'}
      bind:this={mountContainer}
      data-plugin-id={currentPluginId}
      data-component={currentComponent}
      style="flex:1;min-width:0;min-height:0;height:100%;display:flex;flex-direction:column;position:relative;"
    ></div>
  {/if}
</div>

<style>
  .plugin-bundle-host {
    flex: 1;
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
  }

  .host-state {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 2rem;
    text-align: center;
  }

  .host-state.idle {
    color: #555;
  }

  .host-state.loading {
    color: #a0a0b8;
  }

  .spinner {
    width: 24px;
    height: 24px;
    border: 2px solid #333;
    border-top-color: #4ecca3;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin-bottom: 1rem;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .host-state.error {
    color: #e94560;
  }

  .host-state.error :global(.error-icon) {
    font-size: 2rem;
    margin-bottom: 0.5rem;
  }

  .error-title {
    font-weight: 600;
    font-size: 1.1rem;
    margin-bottom: 1rem;
  }

  .error-details {
    font-size: 0.85rem;
    color: #a0a0b8;
    max-width: 400px;
    text-align: left;
    background: #16213e;
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    border: 1px solid #0f3460;
  }

  .error-details[open] {
    padding: 0.75rem 1rem;
  }

  .error-details summary {
    cursor: pointer;
    color: #e0e0f0;
    font-weight: 600;
  }

  .error-details p {
    margin: 0.3rem 0;
  }

  .error-details strong {
    color: #e0e0f0;
  }

  .error-message {
    color: #e94560;
    max-width: 400px;
    margin: 0 0 0.75rem;
  }

  .error-meta {
    font-size: 0.75rem;
    color: #666;
    margin-top: 0.3rem !important;
  }

  .plugin-mount-container {
    flex: 1;
    min-width: 0;
    min-height: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
  }

  .plugin-mount-container.mount-hidden {
    height: 0;
    min-height: 0;
    overflow: hidden;
    visibility: hidden;
  }
</style>
