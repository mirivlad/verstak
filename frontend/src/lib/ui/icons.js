/**
 * icons.js — Centralised SVG icon set for Verstak.
 *
 * RULE: Icons MUST be SVG only. No emoji, no unicode symbols, no font-icons.
 * The Wails WebKitGTK webview does NOT render colour emoji.
 *
 * Each icon is an object: { viewBox, paths }
 *   - viewBox: string, default '0 0 24 24'
 *   - paths: array of <path> attributes or objects with { d, ...attrs }
 *
 * Usage in Svelte:
 *   import { iconPaths } from '../lib/ui/icons.js';
 *   <svg viewBox={iconPaths.puzzle.viewBox} width="16" height="16">
 *     {#each iconPaths.puzzle.paths as p}
 *       <path d={p.d} {...p.attrs} />
 *     {/each}
 *   </svg>
 */

export const iconPaths = {
  /** Plugin Manager — puzzle piece */
  puzzle: {
    viewBox: '0 0 24 24',
    paths: [
      { d: 'M4 5a3 3 0 0 1 3-3h4v2a2 2 0 0 0 4 0V2h4a2 2 0 0 1 2 2v4h-2a2 2 0 0 0 0 4h2v4a2 2 0 0 1-2 2h-4v-2a2 2 0 0 0-4 0v2H7a3 3 0 0 1-3-3v-3h2a2 2 0 0 0 0-4H4V5Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
    ],
  },

  /** Platform Test / Diagnostics — flask/test-tube */
  flask: {
    viewBox: '0 0 24 24',
    paths: [
      { d: 'M9 2v5.586l-5.293 5.293A1 1 0 0 0 4 14v4a4 4 0 0 0 4 4h8a4 4 0 0 0 4-4v-4a1 1 0 0 0-.293-.707L15 7.586V2H9Zm2 0v6a1 1 0 0 0 .293.707l5 5V14a1 1 0 0 1-1 1h-1l-2-2-2 2H8a1 1 0 0 1-1-1v-.293l3-3V2h3Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
    ],
  },

  /** Verstak logo — stack/tray */
  logo: {
    viewBox: '0 0 24 24',
    paths: [
      { d: 'M5 3h14a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Zm0 2v3h14V5H5Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
      { d: 'M5 12h14a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2Zm0 2v3h14v-3H5Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
    ],
  },

  /** Default fallback — circle */
  dot: {
    viewBox: '0 0 24 24',
    paths: [
      { d: 'M12 2C6.477 2 2 6.477 2 12s4.477 10 10 10 10-4.477 10-10S17.523 2 12 2Zm0 6a4 4 0 1 1 0 8 4 4 0 0 1 0-8Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
    ],
  },

  /** Vault — safe / shield */
  vault: {
    viewBox: '0 0 24 24',
    paths: [
      { d: 'M12 1L3 5v6c0 5.55 3.84 10.74 9 12 5.16-1.26 9-6.45 9-12V5l-9-4Zm0 2.18L19 6.4v4.6c0 4.5-3.07 8.68-7 9.82-3.93-1.14-7-5.32-7-9.82V6.4l7-3.22ZM11 8v2H9v2h2v4h2v-4h2v-2h-2V8h-2Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
    ],
  },

  /** Settings — gear */
  gear: {
    viewBox: '0 0 24 24',
    paths: [
      { d: 'M19.14 13l.57-1.43 1.79-.5-1-2.29-1.64.73-.29-.28-.73-1.64 2.29-1-1-2.29-1.79.5-.57 1.43-2.17.17-.57-1.43-1.79-.5-1 2.29 1.64.73-.29.28-.73 1.64-2.29-1-1 2.29 1.79.5.57 1.43-.57 1.43-1.79.5 1 2.29 1.64-.73.29.28.73 1.64-2.29 1 1 2.29 1.79-.5.57-1.43 2.17-.17.57 1.43 1.79.5 1-2.29-1.64-.73.29-.28.73-1.64 2.29 1 1-2.29-1.79-.5-.57-1.43ZM12 9a3 3 0 1 1 0 6 3 3 0 0 1 0-6Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
    ],
  },

  /** Warning / error — triangle */
  warning: {
    viewBox: '0 0 24 24',
    paths: [
      { d: 'M1 21h22L12 2 1 21Zm12-3h-2v-2h2v2Zm0-4h-2v-4h2v4Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
    ],
  },

  /** General plugin — extension/puzzle alternative */
  plugin: {
    viewBox: '0 0 24 24',
    paths: [
      { d: 'M11 2H5v6.172a3 3 0 0 0 0 5.656V20a1 1 0 0 0 1 1h4v-2H7v-4a1 1 0 0 0-1-1H5v-2h1a1 1 0 0 0 1-1V8h2V4h2a1 1 0 0 0 1-1V2h-1Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
      { d: 'M17 2a3 3 0 0 1 3 3v1h-4V5a1 1 0 0 0-1-1h-2v2h2v2h-2v2h4V8h2v2a3 3 0 0 1-3 3h-1v2h1a2 2 0 0 1 2 2v2h2v2a2 2 0 0 1-2 2h-4a2 2 0 0 1-2-2v-2h2v-2h-2v-2h4a1 1 0 0 0 1-1V5a1 1 0 0 0-1-1h-2v2h-2V4a2 2 0 0 1 2-2h3Z',
        attrs: { fill: 'currentColor', stroke: 'none' },
      },
    ],
  },
};

/**
 * Render an SVG icon string inline.
 * Returns an SVG string suitable for {@html } or innerHTML.
 * @param {string} name - icon key
 * @param {number|string} size - width/height
 * @param {string} className - optional CSS class
 */
export function svgIcon(name, size = 16, className = '') {
  const icon = iconPaths[name];
  if (!icon) return svgIcon('dot', size, className);
  const paths = icon.paths
    .map(p => {
      const attrs = p.attrs
        ? Object.entries(p.attrs).map(([k, v]) => `${k}="${v}"`).join(' ')
        : '';
      return `<path d="${p.d}"${attrs ? ' ' + attrs : ''}/>`;
    })
    .join('');
  const cls = className ? ` class="${className}"` : '';
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${icon.viewBox}" width="${size}" height="${size}"${cls}>${paths}</svg>`;
}
