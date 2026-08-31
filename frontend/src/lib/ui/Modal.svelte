<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  export let title = '';
  export let show = false;
  export let wide = false;
  let overlayElement;

  const dispatch = createEventDispatcher();

  function close() {
    dispatch('close');
  }

  function onKeydown(e) {
    if (show && e.key === 'Escape') close();
  }

  onMount(() => {
    function closeFromDocumentClick(event) {
      if (show && event.target === overlayElement) close();
    }
    document.addEventListener('click', closeFromDocumentClick);
    return () => document.removeEventListener('click', closeFromDocumentClick);
  });
</script>

<svelte:window on:keydown={onKeydown} />

{#if show}
  <div {...$$restProps} class="vt-modal-overlay" bind:this={overlayElement}>
    <div
      class="vt-modal" class:vt-modal-wide={wide}
      role="dialog" aria-modal="true" aria-label={title} tabindex="-1"
      transition:fade={{ duration: 120 }}
    >
      {#if title}
        <div class="vt-modal-header">
          <h2>{title}</h2>
        </div>
      {/if}
      <div class="vt-modal-body">
        <slot />
      </div>
      <div class="vt-modal-actions">
        <slot name="actions" />
      </div>
    </div>
  </div>
{/if}

<style>
  .vt-modal-wide { width: min(36rem, 100%); }
  .vt-modal-header h2 { margin: 0; font-size: 1rem; color: var(--vt-color-text-primary); }
</style>
