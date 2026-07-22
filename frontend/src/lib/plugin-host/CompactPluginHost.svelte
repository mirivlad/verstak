<script>
  import { onDestroy } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import { acquirePluginStyle, createPluginAPI } from './VerstakPluginAPI.js';

  export let pluginId;
  export let handler;
  export let label = '';

  let container;
  let state = 'loading';
  let errorText = '';
  let current = null;
  let sequence = 0;

  $: if (pluginId && handler && container) mount(pluginId, handler);

  onDestroy(() => {
    sequence += 1;
    cleanup();
  });

  function unpack(result) {
    if (Array.isArray(result) && result.length === 2) return { value: result[0], error: result[1] || '' };
    return { value: result, error: '' };
  }

  function cleanup() {
    if (!current) return;
    try { current.component?.unmount?.(container); } catch (_) {}
    try { current.api?.dispose?.(); } catch (_) {}
    try { current.releaseStyle?.(); } catch (_) {}
    if (container) container.innerHTML = '';
    current = null;
  }

  async function mount(nextPluginId, nextHandler) {
    if (current?.pluginId === nextPluginId && current?.handler === nextHandler) return;
    const run = ++sequence;
    cleanup();
    state = 'loading';
    errorText = '';
    let releaseStyle = null;
    try {
      const info = await App.GetPluginFrontendInfo(nextPluginId);
      if (!info?.entry) throw new Error('plugin frontend is unavailable');
      releaseStyle = await acquirePluginStyle(nextPluginId, info.style);
      let registry = window.__VERSTAK_PLUGIN_REGISTRY__ || {};
      if (!registry[nextPluginId]) {
        const asset = unpack(await App.GetPluginAssetContent(nextPluginId, info.entry));
        if (asset.error || !asset.value) throw new Error(asset.error || 'plugin bundle is empty');
        new Function(asset.value)();
        registry = window.__VERSTAK_PLUGIN_REGISTRY__ || {};
      }
      const component = registry[nextPluginId]?.[nextHandler];
      if (!component?.mount) throw new Error(`component ${nextHandler} is unavailable`);
      if (run !== sequence) {
        releaseStyle();
        return;
      }
      const api = createPluginAPI(nextPluginId);
      component.mount(container, { componentId: nextHandler, compact: true }, api);
      current = { pluginId: nextPluginId, handler: nextHandler, component, api, releaseStyle };
      releaseStyle = null;
      state = 'loaded';
    } catch (error) {
      releaseStyle?.();
      if (run !== sequence) return;
      state = 'error';
      errorText = error?.message || String(error);
    }
  }
</script>

<span class="compact-plugin-host" data-plugin-status-handler={handler} title={state === 'error' ? `${pluginId}: ${errorText}` : pluginId}>
  <span bind:this={container} class:hidden={state !== 'loaded'}></span>
  {#if state === 'loading'}<span class="compact-state" aria-label="Loading">…</span>{/if}
  {#if state === 'error'}<span class="compact-state compact-error" aria-label={errorText}>⚠ {label}</span>{/if}
</span>

<style>
  .compact-plugin-host { min-width: 0; max-width: 100%; height: 1.2rem; display: inline-flex; align-items: center; overflow: hidden; white-space: nowrap; }
  .compact-plugin-host > span:not(.compact-state) { min-width: 0; display: inline-flex; align-items: center; }
  .compact-plugin-host :global(button) { min-height: 0 !important; height: 1.2rem !important; margin: 0 !important; padding: 0 0.25rem !important; border-width: 0 !important; background: transparent !important; color: inherit !important; font: inherit !important; line-height: 1 !important; }
  .hidden { display: none !important; }
  .compact-state { line-height: 1; color: var(--vt-color-text-muted, #9fb2ca); }
  .compact-error { max-width: 14rem; overflow: hidden; color: var(--vt-color-warning, #ffc857); text-overflow: ellipsis; }
</style>
