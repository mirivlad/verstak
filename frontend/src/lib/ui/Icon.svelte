<script>
  import Circle from 'lucide-svelte/icons/circle';

  const modules = import.meta.glob('/node_modules/lucide-svelte/dist/icons/*.svelte');

  export let name = 'dot';
  export let size = 16;
  export let className = '';

  let IconComponent = Circle;
  let lastRequested = '';

  async function resolveIcon(iconName) {
    if (!iconName || iconName === lastRequested) return;
    lastRequested = iconName;
    const key = Object.keys(modules).find(k => k.endsWith('/' + iconName + '.svelte'));
    if (key) {
      try {
        const mod = await modules[key]();
        IconComponent = mod.default;
        return;
      } catch {}
    }
    IconComponent = Circle;
  }

  $: if (name) resolveIcon(String(name));

  $: iconClass = [className, $$restProps.class].filter(Boolean).join(' ');
</script>

<svelte:component this={IconComponent} {size} {...$$restProps} class={iconClass} aria-hidden="true" />
