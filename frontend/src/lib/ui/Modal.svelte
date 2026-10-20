<script>
  import { createEventDispatcher, onMount, tick } from 'svelte';
  import { fade } from 'svelte/transition';

  export let title = '';
  export let show = false;
  export let wide = false;
  let overlayElement;

  const dispatch = createEventDispatcher();

  // Dialogs remain visible when their owning navigation panel is collapsed.
  function portal(node) {
    document.body.appendChild(node);
    return { destroy() { node.remove(); } };
  }

  function manageFocus(dialog) {
    const opener = document.activeElement;
    const background = [];
    let branch = dialog.parentElement;
    while (branch?.parentElement) {
      for (const sibling of branch.parentElement.children) {
        if (sibling !== branch && sibling instanceof HTMLElement && !sibling.inert) {
          // Already inert regions (for example the drawer background) belong
          // to their owner; restoring them here would overwrite newer state.
          background.push([sibling, false]);
          sibling.inert = true;
        }
      }
      branch = branch.parentElement;
    }
    const focusable = () => [...dialog.querySelectorAll('button, input, select, textarea, a[href], [tabindex]')]
      .filter((node) => !node.disabled && node.tabIndex >= 0 && node.getClientRects().length && !node.closest('[inert]'));
    let disposed = false;
    tick().then(() => {
      if (!disposed) (dialog.querySelector('[data-modal-initial-focus]:not(:disabled)') || focusable()[0] || dialog).focus({ preventScroll: true });
    });
    function trapTab(event) {
      if (event.key !== 'Tab' || !dialog.contains(document.activeElement)) return;
      const targets = focusable();
      const first = targets[0] || dialog;
      const last = targets[targets.length - 1] || dialog;
      if (!targets.length || (event.shiftKey && (document.activeElement === first || document.activeElement === dialog)) || (!event.shiftKey && document.activeElement === last)) {
        event.preventDefault();
        (event.shiftKey ? last : first).focus();
      }
    }
    dialog.addEventListener('keydown', trapTab);
    return { destroy() {
      disposed = true;
      dialog.removeEventListener('keydown', trapTab);
      for (const [element, wasInert] of background) element.inert = wasInert;
      if (opener?.isConnected && !opener.closest('[inert]')) opener.focus({ preventScroll: true });
    } };
  }

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
  <div {...$$restProps} class="vt-modal-overlay" bind:this={overlayElement} use:portal>
    <div
      class="vt-modal" class:vt-modal-wide={wide}
      role="dialog" aria-modal="true" aria-label={title} tabindex="-1"
      use:manageFocus
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
