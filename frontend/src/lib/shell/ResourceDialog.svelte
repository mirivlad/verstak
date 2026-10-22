<script>
  import { createEventDispatcher, onDestroy } from 'svelte';
  import Modal from '../ui/Modal.svelte';
  import PluginBundleHost from '../plugin-host/PluginBundleHost.svelte';
  import { i18n } from '../i18n/index.js';

  export let openedResource = null;
  const dispatch = createEventDispatcher();
  let locale = i18n.getLocale();
  const unsubscribe = i18n.subscribe(next => locale = next);
  onDestroy(unsubscribe);
  $: tr = ((activeLocale) => (key, fallback) => {
    void activeLocale;
    return i18n.t(key, undefined, fallback);
  })(locale);
</script>

<Modal show={!!openedResource} wide title={tr('workbench.resourceDialog', 'View material')} on:close={() => dispatch('close')} data-resource-dialog>
  {#if openedResource?.status === 'opened'}
    {#key [openedResource.providerId, openedResource.request?.path].join(':')}
      <PluginBundleHost autoHeight pluginId={openedResource.providerPluginId} componentId={openedResource.providerComponent} componentProps={openedResource} />
    {/key}
  {:else}
    <p>{tr('workbench.noViewer')}</p>
  {/if}
  <svelte:fragment slot="actions">
    <button class="btn btn-secondary" type="button" data-resource-dialog-close on:click={() => dispatch('close')}>{tr('common.close')}</button>
  </svelte:fragment>
</Modal>
