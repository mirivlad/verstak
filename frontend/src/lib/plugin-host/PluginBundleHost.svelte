<script>
  import { onMount, onDestroy } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';

  // Import the VerstakPluginAPI contract
  import './VerstakPluginAPI.js';

  export let pluginId = null;
  export let componentId = null;
  export let viewPluginId = null;

  let loadState = 'idle'; // idle | loading | loaded | error
  let pluginInfo = null;
  let errorText = '';
  let mountContainer = null;
  let currentPluginId = null;
  let currentComponent = null;

  $: activePluginId = pluginId || viewPluginId;
  $: activeComponent = componentId;

  // React to changes — reload on view change
  $: if (activePluginId && activeComponent) {
    loadAndMount(activePluginId, activeComponent);
  } else if (!activePluginId) {
    cleanup();
    loadState = 'idle';
  }

  onDestroy(() => {
    cleanup();
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
    if (mountContainer) {
      mountContainer.innerHTML = '';
    }
    currentPluginId = null;
    currentComponent = null;
  }

  async function loadAndMount(pId, compId) {
    // If same plugin+component and already mounted, skip
    if (currentPluginId === pId && currentComponent === compId && loadState === 'loaded') {
      return;
    }

    // Cleanup previous
    cleanup();

    loadState = 'loading';
    errorText = '';
    currentPluginId = pId;
    currentComponent = compId;

    try {
      // Get plugin frontend info
      const info = await App.GetPluginFrontendInfo(pId);
      pluginInfo = info;

      if (!info || info.status === 'no-frontend' || info.status === 'not-found') {
        loadState = 'error';
        errorText = info.status === 'no-frontend'
          ? 'Plugin has no frontend bundle'
          : 'Plugin not found';
        return;
      }

      // Check if bundle already loaded for this plugin
      const reg = window.__VERSTAK_PLUGIN_REGISTRY__ || {};
      if (!reg[pId]) {
        // Load the bundle JS content via backend API
        const [content, err] = await App.GetPluginAssetContent(pId, info.entry);
        if (err || !content) {
          loadState = 'error';
          errorText = 'Failed to load bundle: ' + (err || 'empty content');
          return;
        }

        // Execute bundle via Function constructor (safe: no access to outer scope)
        // This is equivalent to eval but more explicit
        try {
          const fn = new Function(content);
          fn();
        } catch (e) {
          loadState = 'error';
          errorText = 'Bundle execution error: ' + e.message;
          console.error('[PluginBundleHost] bundle exec error:', e);
          return;
        }

        // Verify registration happened
        if (!window.__VERSTAK_PLUGIN_REGISTRY__[pId]) {
          loadState = 'error';
          errorText = 'Bundle loaded but no VerstakPluginRegister call detected';
          return;
        }
      }

      // Find the component
      const components = window.__VERSTAK_PLUGIN_REGISTRY__[pId];
      const comp = components[compId];
      if (!comp || !comp.mount) {
        loadState = 'error';
        errorText = 'Component "' + compId + '" not found in bundle. Available: '
          + (Object.keys(components).join(', ') || 'none');
        return;
      }

      // Create API
      const api = window.VerstakPluginAPI(pId);

      // Mount component
      if (!mountContainer) {
        // Container must exist in DOM — wait for next tick
        await new Promise(r => requestAnimationFrame(r));
      }
      if (mountContainer) {
        try {
          comp.mount(mountContainer, { componentId: compId }, api);
          loadState = 'loaded';
          errorText = '';
        } catch (e) {
          loadState = 'error';
          errorText = 'Component mount error: ' + e.message;
          console.error('[PluginBundleHost] mount error:', e);
        }
      } else {
        loadState = 'error';
        errorText = 'Mount container not available';
      }
    } catch (e) {
      loadState = 'error';
      errorText = 'Unexpected error: ' + (e.message || e);
      console.error('[PluginBundleHost] error:', e);
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
      <p>Select a plugin view from the sidebar</p>
    </div>

  {:else if loadState === 'loading'}
    <div class="host-state loading">
      <div class="spinner"></div>
      <p>Loading plugin bundle...</p>
    </div>

  {:else if loadState === 'error'}
    <div class="host-state error">
      <div class="error-icon">⚠️</div>
      <p class="error-title">Plugin View Error</p>
      <div class="error-details">
        <p><strong>Plugin:</strong> {currentPluginId || 'unknown'}</p>
        <p><strong>Component:</strong> {currentComponent || 'unknown'}</p>
        <p class="error-message">{errorText || 'Unknown error'}</p>
        {#if pluginInfo}
          <p class="error-meta">Frontend entry: {pluginInfo.entry || 'none'}</p>
        {/if}
        {#if getComponentList().length > 0}
          <p class="error-meta">Available components: {getComponentList().join(', ')}</p>
        {/if}
      </div>
    </div>

  {:else if loadState === 'loaded'}
    <div
      class="plugin-mount-container"
      bind:this={mountContainer}
      data-plugin-id={currentPluginId}
      data-component={currentComponent}
    ></div>
  {/if}
</div>

<style>
  .plugin-bundle-host {
    width: 100%;
    height: 100%;
    min-height: 200px;
    display: flex;
    flex-direction: column;
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

  .error-icon {
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
    padding: 1rem;
    border-radius: 6px;
    border: 1px solid #0f3460;
  }

  .error-details p {
    margin: 0.3rem 0;
  }

  .error-details strong {
    color: #e0e0f0;
  }

  .error-message {
    color: #e94560;
    font-family: monospace;
    font-size: 0.8rem;
    margin-top: 0.5rem !important;
    padding: 0.5rem;
    background: rgba(233, 69, 96, 0.1);
    border-radius: 4px;
  }

  .error-meta {
    font-size: 0.75rem;
    color: #666;
    margin-top: 0.3rem !important;
  }

  .plugin-mount-container {
    flex: 1;
    overflow: auto;
    position: relative;
  }
</style>
