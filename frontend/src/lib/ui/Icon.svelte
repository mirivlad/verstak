<script>
  /**
   * Icon.svelte — Renders bundled Lucide SVG icons.
   *
   * Usage:
   *   <Icon name="puzzle" size={18} />
   *   <Icon name="flask" class="my-icon" />
   *
   * Rules:
   *   - NO emoji, NO unicode pictographic symbols
   *   - NO dependency on system icon fonts
   *   - If name is not found, renders a default circle icon
   */
  import Briefcase from 'lucide-svelte/icons/briefcase';
  import ChevronDown from 'lucide-svelte/icons/chevron-down';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Circle from 'lucide-svelte/icons/circle';
  import FlaskConical from 'lucide-svelte/icons/flask-conical';
  import Folder from 'lucide-svelte/icons/folder';
  import LayoutGrid from 'lucide-svelte/icons/layout-grid';
  import PanelsTopLeft from 'lucide-svelte/icons/panels-top-left';
  import Pencil from 'lucide-svelte/icons/pencil';
  import Plug from 'lucide-svelte/icons/plug';
  import Puzzle from 'lucide-svelte/icons/puzzle';
  import Settings from 'lucide-svelte/icons/settings';
  import Shield from 'lucide-svelte/icons/shield';
  import TriangleAlert from 'lucide-svelte/icons/triangle-alert';

  export let name = 'dot';
  export let size = 16;
  export let className = '';

  const icons = {
    case: Briefcase,
    chevronDown: ChevronDown,
    chevronRight: ChevronRight,
    dot: Circle,
    edit: Pencil,
    flask: FlaskConical,
    folder: Folder,
    gear: Settings,
    logo: PanelsTopLeft,
    plugin: Plug,
    puzzle: Puzzle,
    space: LayoutGrid,
    vault: Shield,
    warning: TriangleAlert,
  };

  const aliases = {
    '🧪': 'flask',
    danger: 'warning',
    settings: 'gear',
  };

  $: iconKey = aliases[name] || name || 'dot';
  $: IconComponent = icons[iconKey] || icons.dot;
  $: iconClass = [className, $$restProps.class].filter(Boolean).join(' ');
</script>

<svelte:component
  this={IconComponent}
  size={size}
  {...$$restProps}
  class={iconClass}
  aria-hidden="true"
/>
