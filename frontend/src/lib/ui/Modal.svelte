<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  export let title = '';
  export let show = false;
  export let wide = false;

  const dispatch = createEventDispatcher();

  function close() {
    dispatch('close');
  }

  function onKeydown(e) {
    if (e.key === 'Escape') close();
  }
</script>

{#if show}
  <div class="vt-modal-overlay" on:click={close} on:keydown={onKeydown} role="dialog" aria-modal="true" aria-label={title}>
    <div
      class="vt-modal" class:vt-modal-wide={wide}
      on:click|stopPropagation
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
  .vt-modal-overlay {
    position: fixed; inset: 0; z-index: 10000;
    display: flex; align-items: center; justify-content: center;
    padding: 1rem;
    background: rgba(4, 8, 18, 0.7);
  }
  .vt-modal {
    width: min(28rem, 100%);
    display: grid; gap: 0.85rem;
    padding: 1rem;
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
    box-shadow: 0 18px 44px rgba(0, 0, 0, 0.38);
  }
  .vt-modal-wide { width: min(36rem, 100%); }
  .vt-modal-header { display: flex; align-items: flex-start; justify-content: space-between; }
  .vt-modal-header h2 { margin: 0; font-size: 1rem; color: var(--vt-color-text-primary); }
  .vt-modal-body { display: grid; gap: 0.75rem; }
  .vt-modal-actions { display: flex; gap: 0.4rem; justify-content: flex-end; }
</style>
