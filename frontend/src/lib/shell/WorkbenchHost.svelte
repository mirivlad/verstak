<script>
  import { onDestroy } from 'svelte';
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import Icon from '../ui/Icon.svelte';
  import { i18n } from '../i18n/index.js';

  export let openedResource = null;
  let locale = i18n.getLocale();
  const unsubscribeLocale = i18n.subscribe((nextLocale) => locale = nextLocale);

  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);

  $: providerPluginId = openedResource?.providerPluginId || '';
  $: providerComponent = openedResource?.providerComponent || '';
  $: resourcePath = openedResource?.request?.path || '';
  $: providerId = openedResource?.providerId || '';
  $: requestMode = openedResource?.request?.mode || 'view';
  $: requestContext = openedResource?.request?.context?.notesMode || openedResource?.request?.context?.isInsideNotesFolder
    ? 'notes-markdown'
    : ((openedResource?.request?.extension === '.md' || openedResource?.request?.extension === '.markdown') ? 'generic-markdown' : 'generic-text');
  $: componentProps = openedResource || {};
  $: mountKey = [
    providerPluginId,
    providerComponent,
    resourcePath,
    requestMode,
    requestContext,
  ].join(':');

  function closeWorkbench() {
    window.dispatchEvent(new CustomEvent('verstak:close-workbench', { cancelable: true }));
  }

  onDestroy(unsubscribeLocale);
</script>

<div class="workbench-host vt-page">
  {#if openedResource?.status === 'no-provider'}
    <div class="workbench-header vt-page-header">
      <span class="workbench-title vt-page-title">{resourcePath}</span>
      <span class="workbench-provider vt-badge">{tr('workbench.noProvider')}</span>
      <button class="close-btn btn-ghost btn-icon" type="button" title={tr('common.close')} aria-label={tr('common.close')} on:click={closeWorkbench}>
        <Icon name="x" size={18} />
      </button>
    </div>
    <div class="workbench-empty no-provider vt-empty-state" data-workbench-status="no-provider">
      <p class="vt-empty-title">{tr('workbench.noViewer')}</p>
      <p class="workbench-meta">{requestMode} · {requestContext}</p>
    </div>
  {:else if openedResource}
    <div class="workbench-header vt-page-header">
      <span class="workbench-title vt-page-title">{resourcePath}</span>
      <span class="workbench-provider vt-badge accent">{providerId}</span>
      <button class="close-btn btn-ghost btn-icon" type="button" title={tr('common.close')} aria-label={tr('common.close')} on:click={closeWorkbench}>
        <Icon name="x" size={18} />
      </button>
    </div>
    <div class="workbench-content">
      {#key mountKey}
        <PluginBundleHost
          pluginId={providerPluginId}
          componentId={providerComponent}
          {componentProps}
        />
      {/key}
    </div>
  {:else}
    <div class="workbench-empty vt-empty-state">
      <p class="vt-empty-title">{tr('workbench.noResource')}</p>
    </div>
  {/if}
</div>

<style>
  .workbench-host {
    flex: 1;
    min-width: 0;
    min-height: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--vt-color-background);
  }

  .workbench-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: var(--vt-space-2) var(--vt-space-4);
    border-bottom: 1px solid var(--vt-color-border);
    flex-shrink: 0;
  }

  .workbench-title {
    color: var(--vt-color-text-primary);
    font-size: 0.95rem;
    font-weight: 600;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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

  .close-btn:hover {
    color: var(--vt-color-text-primary);
    background: var(--vt-color-surface-hover);
  }

  .workbench-provider {
    color: var(--vt-color-accent);
    font-size: 0.75rem;
    margin-left: auto;
  }

  .workbench-content {
    min-width: 0;
    min-height: 0;
    height: 100%;
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 0;
  }

  .workbench-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--vt-color-text-muted);
  }

  .workbench-empty.no-provider {
    flex-direction: column;
    gap: 0.35rem;
  }

  .workbench-meta {
    margin: 0;
    color: var(--vt-color-text-muted);
    font-size: 0.8rem;
  }
</style>
