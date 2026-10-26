<script>
  import { onDestroy } from 'svelte';
  import Modal from './Modal.svelte';
  import IconPicker from './IconPicker.svelte';
  import { i18n } from '../i18n/index.js';
  export let current = '';
  export let finish;
  let locale = i18n.getLocale();
  const offLocale = i18n.subscribe((value) => { locale = value; });
  onDestroy(offLocale);
  $: tr = ((value) => (key) => { void value; return i18n.t(key); })(locale);
</script>

<Modal show title={tr('workspaceTree.iconPicker')} on:close={() => finish(null)} data-icon-picker-dialog>
  <IconPicker {current} defaultIcon="external-link" on:select={(event) => finish(event.detail || 'external-link')} />
  <svelte:fragment slot="actions"><button type="button" class="vt-button secondary" on:click={() => finish(null)}>{tr('common.cancel')}</button></svelte:fragment>
</Modal>
