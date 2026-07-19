<script>
  import { onDestroy } from 'svelte';
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import { i18n } from '../i18n/index.js';
  import { debug } from '../log/debug.js';

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
  $: resourceTitle = resourceDisplayTitle(resourcePath, requestContext);
  $: componentProps = openedResource || {};
  $: mountKey = [
    providerPluginId,
    providerComponent,
    resourcePath,
    requestMode,
    requestContext,
  ].join(':');

  $: if (openedResource?.status === 'no-provider') {
    window.dispatchEvent(new CustomEvent('verstak:content-title-changed', {
      detail: { title: resourceTitle, subtitle: tr('workbench.noProvider') }
    }));
  } else if (openedResource) {
    window.dispatchEvent(new CustomEvent('verstak:content-title-changed', {
      detail: { title: resourceTitle }
    }));
  }

  function resourceDisplayTitle(path, context) {
    const fileName = String(path || '').split('/').filter(Boolean).pop() || String(path || '');
    return context === 'notes-markdown' ? fileName.replace(/\.(md|markdown)$/i, '') : fileName;
  }

  onDestroy(unsubscribeLocale);
</script>

<div class="workbench-host vt-page">
  {#if openedResource?.status === 'no-provider'}
    <div class="workbench-empty no-provider vt-empty-state" data-workbench-status="no-provider">
      <p class="vt-empty-title">{tr('workbench.noViewer')}</p>
      {#if debug.isEnabled()}
        <p class="workbench-meta">{requestMode} · {requestContext}</p>
      {/if}
    </div>
  {:else if openedResource}
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
