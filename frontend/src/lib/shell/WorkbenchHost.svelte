<script>
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import Icon from '../ui/Icon.svelte';

  export let openedResource = null;

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
</script>

<div class="workbench-host">
  {#if openedResource?.status === 'no-provider'}
    <div class="workbench-header">
      <span class="workbench-title">{resourcePath}</span>
      <span class="workbench-provider">no-provider</span>
      <button class="close-btn" type="button" title="Close" aria-label="Close" on:click={closeWorkbench}>
        <Icon name="x" size={18} />
      </button>
    </div>
    <div class="workbench-empty no-provider" data-workbench-status="no-provider">
      <p>No viewer/editor available</p>
      <p class="workbench-meta">{requestMode} · {requestContext}</p>
    </div>
  {:else if openedResource}
    <div class="workbench-header">
      <span class="workbench-title">{resourcePath}</span>
      <span class="workbench-provider">{providerId}</span>
      <button class="close-btn" type="button" title="Close" aria-label="Close" on:click={closeWorkbench}>
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
    <div class="workbench-empty">
      <p>No resource opened</p>
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
    background: #1a1a2e;
  }

  .workbench-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem 1rem;
    border-bottom: 1px solid #16213e;
    flex-shrink: 0;
  }

  .workbench-title {
    color: #e0e0f0;
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
    border: 1px solid #1a3a5c;
    border-radius: 4px;
    background: #0f3460;
    color: #a0a0b8;
    flex-shrink: 0;
    cursor: pointer;
  }

  .close-btn:hover {
    color: #e0e0f0;
    background: #1a4a7a;
  }

  .workbench-provider {
    color: #4ecca3;
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
    color: #666;
  }

  .workbench-empty.no-provider {
    flex-direction: column;
    gap: 0.35rem;
  }

  .workbench-meta {
    margin: 0;
    color: #8b8ba8;
    font-size: 0.8rem;
  }
</style>
