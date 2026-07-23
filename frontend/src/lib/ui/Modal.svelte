<script>
  import { createEventDispatcher } from 'svelte';
  import { fade } from 'svelte/transition';

  export let title = '';
  export let show = false;
  export let wide = false;

  const dispatch = createEventDispatcher();

  function close() {
    dispatch('close');
  }

  function closeFromOverlay(event) {
    if (event.target === event.currentTarget) close();
  }

  function onKeydown(e) {
    if (e.key === 'Escape') close();
  }
</script>

{#if show}
  <div {...$$restProps} class="vt-modal-overlay" on:click={closeFromOverlay} on:keydown={onKeydown} role="dialog" aria-modal="true" aria-label={title} tabindex="-1">
    <div
      class="vt-modal" class:vt-modal-wide={wide}
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
