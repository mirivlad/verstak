<script context="module">
  import { BRAND_ICON, CORE_ICONS, ICON_ALIASES } from './icons/core.js';

  // Icons the shell and the official plugins name are compiled in, so they are
  // on screen at first paint. Everything else — in practice the folder icon
  // picker, which offers the whole lucide catalogue on purpose — comes from
  // one lazily loaded module instead of 1703 separate chunks.
  let spritePromise = null;
  let sprite = null;
  const warned = new Set();

  function loadSprite() {
    if (!spritePromise) {
      spritePromise = import('./icons/sprite.js')
        .then((module) => {
          sprite = module.SPRITE_ICONS;
          return sprite;
        })
        .catch((error) => {
          console.error('[Icon] icon sprite could not be loaded:', error);
          sprite = {};
          return sprite;
        });
    }
    return spritePromise;
  }

  export function resolveIconName(name) {
    const requested = String(name || '');
    return ICON_ALIASES[requested] || requested;
  }

  // Exported so a test can assert the vocabulary without running a browser.
  export function iconIsCore(name) {
    return Object.prototype.hasOwnProperty.call(CORE_ICONS, resolveIconName(name));
  }

  function warnUnknown(name) {
    if (warned.has(name)) return;
    warned.add(name);
    // Deliberately noisy: a silent fallback is how the Verstak logo spent its
    // life rendering as a generic lucide circle.
    console.warn(`[Icon] unknown icon "${name}" — falling back to a placeholder`);
  }
</script>

<script>
  export let name = 'dot';
  export let size = 16;
  export let className = '';

  const PLACEHOLDER = [['circle', { cx: '12', cy: '12', r: '9', 'stroke-dasharray': '3 3' }]];

  let nodes = PLACEHOLDER;
  let viewBox = '0 0 24 24';
  let coloured = false;

  $: applyIcon(resolveIconName(name));

  function applyIcon(iconName) {
    if (iconName === 'logo') {
      nodes = BRAND_ICON.nodes;
      viewBox = BRAND_ICON.viewBox;
      coloured = true;
      return;
    }

    coloured = false;
    viewBox = '0 0 24 24';

    if (Object.prototype.hasOwnProperty.call(CORE_ICONS, iconName)) {
      nodes = CORE_ICONS[iconName];
      return;
    }

    if (sprite) {
      if (Object.prototype.hasOwnProperty.call(sprite, iconName)) {
        nodes = sprite[iconName];
      } else {
        warnUnknown(iconName);
        nodes = PLACEHOLDER;
      }
      return;
    }

    nodes = PLACEHOLDER;
    loadSprite().then((loaded) => {
      // The name may have changed while the sprite was loading.
      if (resolveIconName(name) !== iconName) return;
      if (Object.prototype.hasOwnProperty.call(loaded, iconName)) {
        nodes = loaded[iconName];
      } else {
        warnUnknown(iconName);
        nodes = PLACEHOLDER;
      }
    });
  }

  // vt-icon is the styling hook; data-icon names the resolved glyph so tests
  // and debugging can tell which icon actually rendered.
  $: iconClass = ['vt-icon', className, $$restProps.class].filter(Boolean).join(' ');
</script>

<svg
  {...$$restProps}
  class={iconClass}
  data-icon={resolveIconName(name)}
  width={size}
  height={size}
  {viewBox}
  fill={coloured ? undefined : 'none'}
  stroke={coloured ? undefined : 'currentColor'}
  stroke-width={coloured ? undefined : 2}
  stroke-linecap={coloured ? undefined : 'round'}
  stroke-linejoin={coloured ? undefined : 'round'}
  aria-hidden="true"
  focusable="false"
>
  {#each nodes || [] as node}
    {#if node[0] === 'path'}<path {...node[1]} />
    {:else if node[0] === 'circle'}<circle {...node[1]} />
    {:else if node[0] === 'rect'}<rect {...node[1]} />
    {:else if node[0] === 'line'}<line {...node[1]} />
    {:else if node[0] === 'polyline'}<polyline {...node[1]} />
    {:else if node[0] === 'polygon'}<polygon {...node[1]} />
    {:else if node[0] === 'ellipse'}<ellipse {...node[1]} />
    {/if}
  {/each}
</svg>
