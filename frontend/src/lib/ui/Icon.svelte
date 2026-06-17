<script>
  /**
   * Icon.svelte — Renders inline SVG icons.
   *
   * Usage:
   *   <Icon name="puzzle" size={18} />
   *   <Icon name="flask" class="my-icon" />
   *
   * Rules:
   *   - NO emoji, NO unicode pictographic symbols
   *   - ALL icons must be SVG registered in ../ui/icons.js
   *   - If name is not found, renders a default dot icon
   */
  import { iconPaths } from './icons.js';

  export let name = 'dot';
  export let size = 16;
  export let className = '';

  $: icon = iconPaths[name] || iconPaths.dot;
</script>

<svg
  xmlns="http://www.w3.org/2000/svg"
  viewBox={icon.viewBox}
  width={size}
  height={size}
  class={className}
>
  {#each icon.paths as p}
    <path d={p.d} {...p.attrs} />
  {/each}
</svg>
