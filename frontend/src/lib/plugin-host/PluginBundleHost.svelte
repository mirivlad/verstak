<script>
  import { onMount, onDestroy } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import { i18n } from '../i18n/index.js';

  import { acquirePluginStyle, createPluginAPI } from './VerstakPluginAPI.js';

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
    if (releaseCurrentStyle) {
      releaseCurrentStyle();
      releaseCurrentStyle = null;
    }
    if (currentAPI && typeof currentAPI.dispose === 'function') {
      try {
        currentAPI.dispose();
      } catch (e) {
        console.error('[PluginBundleHost] API dispose error:', e);
      }
    }
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
    if (mountContainer) {
      mountContainer.innerHTML = '';
    }
    currentPluginId = null;
    currentComponent = null;
    currentAPI = null;
    currentPropsKey = '';
  }

  function unpackBackendResult(result) {
    if (Array.isArray(result) && result.length === 2 && (typeof result[1] === 'string' || result[1] == null)) {
      return { value: result[0], error: result[1] || '' };
    }
    return { value: result, error: '' };
  }

  function reportError(key, fallback, details) {
    console.warn('[PluginBundleHost] ' + key + ':', details);
    return tr(key, undefined, fallback);
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
      // Get plugin frontend info
      const info = await App.GetPluginFrontendInfo(pId);
      pluginInfo = info;

      if (!info || info.status === 'no-frontend' || info.status === 'not-found') {
        loadState = 'error';
        errorText = info.status === 'no-frontend'
          ? tr('bundle.noFrontend')
          : tr('bundle.notFound');
        return;
      }

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

      // Check if bundle already loaded for this plugin
      const reg = window.__VERSTAK_PLUGIN_REGISTRY__ || {};
      if (!reg[pId]) {
        // Load the bundle JS content via backend API
        const assetResult = unpackBackendResult(await App.GetPluginAssetContent(pId, info.entry));
        const content = assetResult.value;
        if (assetResult.error || !content) {
          loadState = 'error';
          errorText = reportError('bundle.loadFailed', 'Could not load the plugin interface. Please try again.', assetResult.error || tr('bundle.emptyContent'));
          return;
        }

        // Execute bundle via Function constructor (safe: no access to outer scope)
        // This is equivalent to eval but more explicit
        try {
          const fn = new Function(content);
          fn();
        } catch (e) {
          loadState = 'error';
          errorText = reportError('bundle.executionError', 'Could not start the plugin interface. Please try again.', e);
          return;
        }

        // Verify registration happened
        if (!window.__VERSTAK_PLUGIN_REGISTRY__[pId]) {
          loadState = 'error';
          errorText = tr('bundle.registrationMissing');
          return;
        }
      }

      // Find the component
      const components = window.__VERSTAK_PLUGIN_REGISTRY__[pId];
      const comp = components[compId];
      if (!comp || !comp.mount) {
        loadState = 'error';
        errorText = tr('bundle.componentMissing', undefined, 'The requested plugin interface is unavailable.');
        return;
      }

      // Create API
      const api = createPluginAPI(pId);
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
