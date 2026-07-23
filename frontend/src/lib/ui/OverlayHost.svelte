<script>
  import { onMount, tick } from 'svelte';

  export let x = 0;
  export let y = 0;
  export let margin = 8;

  let host;
  let left = x;
  let top = y;
  let frame = 0;
  let resizeObserver;

  function portal(node) {
    document.body.appendChild(node);
    return {
      destroy() {
        node.remove();
      },
    };
  }

  async function place() {
    await tick();
    if (!host) return;
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => {
      if (!host) return;
      const rect = host.getBoundingClientRect();
      left = Math.max(margin, Math.min(x, window.innerWidth - rect.width - margin));
      top = Math.max(margin, Math.min(y, window.innerHeight - rect.height - margin));
    });
  }

  $: {
    x;
    y;
    place();
  }

  onMount(() => {
    window.addEventListener('resize', place);
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(place);
      resizeObserver.observe(host);
    }
    place();
    return () => {
      cancelAnimationFrame(frame);
      resizeObserver?.disconnect();
      window.removeEventListener('resize', place);
    };
  });
</script>

<div
  bind:this={host}
  use:portal
  class="vt-overlay-host"
  style="left:{left}px;top:{top}px"
>
  <slot />
</div>

<style>
  .vt-overlay-host {
    position: fixed;
    z-index: 10000;
  }
</style>
