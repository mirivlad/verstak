#!/usr/bin/env node
// Generates the shell's icon data from lucide-svelte.
//
// Icon.svelte used to resolve icons with import.meta.glob over every lucide
// component, which emitted 1703 chunks into dist and made each icon arrive
// asynchronously — visible as a placeholder that swaps a frame later, on every
// list render.
//
// Two outputs instead:
//
//   icons/core.js    the vocabulary the shell and the official plugins
//                    actually name, imported statically so those icons are
//                    present on first paint;
//   icons/sprite.js  every remaining lucide icon, imported lazily and only
//                    when something asks for a name outside the core set —
//                    in practice the folder icon picker, which offers the
//                    whole catalogue on purpose.
//
// Usage: node scripts/generate-icon-assets.mjs [--check]
//   --check verifies the generated files are current without writing.

import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const workspace = path.resolve(root, '..');
const iconsSourceDir = path.join(root, 'frontend', 'node_modules', 'lucide-svelte', 'dist', 'icons');
const outputDir = path.join(root, 'frontend', 'src', 'lib', 'ui', 'icons');
const brandSvgPath = path.join(root, 'packaging', 'linux', 'verstak.svg');

const checkOnly = process.argv.includes('--check');

// ── Read every lucide icon as plain node data ───────────────────────────────
function readLucideIcons() {
  if (!fs.existsSync(iconsSourceDir)) {
    throw new Error(`lucide-svelte not installed at ${iconsSourceDir}; run npm ci in frontend/`);
  }
  const icons = new Map();
  for (const file of fs.readdirSync(iconsSourceDir)) {
    if (!file.endsWith('.svelte')) continue;
    const name = file.replace(/\.svelte$/, '');
    const source = fs.readFileSync(path.join(iconsSourceDir, file), 'utf8');
    const match = source.match(/const iconNode = (\[[\s\S]*?\]);\n/);
    if (!match) continue;
    icons.set(name, JSON.parse(match[1]));
  }
  if (icons.size === 0) throw new Error('no lucide icons could be parsed');
  return icons;
}

// ── Work out which names the product actually uses ──────────────────────────
function collectShellNames() {
  const names = new Set();
  const srcDir = path.join(root, 'frontend', 'src');

  function walk(dir) {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        if (entry.name === 'icons' || entry.name === 'test') continue;
        walk(full);
        continue;
      }
      if (!/\.(svelte|js)$/.test(entry.name)) continue;
      const source = fs.readFileSync(full, 'utf8');
      // <Icon name="chevron-down" …>
      for (const m of source.matchAll(/<Icon[^>]*?\bname=["']([a-z0-9-]+)["']/g)) names.add(m[1]);
      // name={condition ? 'folder' : 'layout-grid'} and name={x || 'plugin'}.
      // Only the braces are scanned: other attributes on the same tag carry
      // class names that are not icons.
      for (const m of source.matchAll(/<Icon[^>]*?\bname=\{([^}]*)\}/g)) {
        for (const literal of m[1].matchAll(/['"]([a-z0-9-]+)['"]/g)) names.add(literal[1]);
      }
    }
  }

  walk(srcDir);
  return names;
}

function collectPluginNames() {
  const names = new Set();
  const pluginRoots = [
    path.join(workspace, 'verstak-official-plugins', 'plugins'),
    path.join(root, 'plugins'),
  ];
  for (const pluginRoot of pluginRoots) {
    if (!fs.existsSync(pluginRoot)) continue;
    for (const entry of fs.readdirSync(pluginRoot, { withFileTypes: true })) {
      if (!entry.isDirectory()) continue;
      const manifestPath = path.join(pluginRoot, entry.name, 'plugin.json');
      if (!fs.existsSync(manifestPath)) continue;
      const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
      if (manifest.icon) names.add(manifest.icon);
      for (const value of Object.values(manifest.contributes || {})) {
        if (!Array.isArray(value)) continue;
        for (const item of value) {
          if (item && typeof item === 'object' && item.icon) names.add(item.icon);
        }
      }
    }
  }
  return names;
}

// Names the shell needs regardless of where they are written, plus a small
// spine of common actions so a plugin picking an obvious verb does not fall
// through to the lazy sprite.
const ALWAYS_CORE = [
  'check', 'chevron-down', 'chevron-left', 'chevron-right', 'chevron-up',
  'circle', 'copy', 'external-link', 'eye', 'file', 'file-text', 'folder',
  'folder-open', 'folder-plus', 'info', 'list', 'pencil', 'plus', 'puzzle',
  'refresh-cw', 'save', 'scissors', 'search', 'settings', 'trash-2',
  'triangle-alert', 'x',
];

// Aliases the product uses that are not lucide names.
const ALIASES = {
  actions: 'ellipsis-vertical',
  gear: 'settings',
  warning: 'triangle-alert',
  plugin: 'puzzle',
  space: 'layout-grid',
  vault: 'vault',
  flask: 'flask-conical',
  edit: 'pencil',
  trash: 'trash-2',
  chevronDown: 'chevron-down',
};

function brandLogoNode() {
  const svg = fs.readFileSync(brandSvgPath, 'utf8');
  const nodes = [];
  for (const m of svg.matchAll(/<(rect|path|circle|line|polyline|polygon)\s+([^>]*?)\/?>/g)) {
    const attrs = {};
    for (const attr of m[2].matchAll(/([a-zA-Z-]+)="([^"]*)"/g)) attrs[attr[1]] = attr[2];
    nodes.push([m[1], attrs]);
  }
  const viewBox = svg.match(/viewBox="([^"]+)"/)?.[1] || '0 0 24 24';
  return { nodes, viewBox };
}

// ── Build ───────────────────────────────────────────────────────────────────
const lucide = readLucideIcons();
const requested = new Set([...collectShellNames(), ...collectPluginNames(), ...ALWAYS_CORE]);

const unresolved = [];
const coreNames = new Set();
for (const name of requested) {
  const resolved = ALIASES[name] || name;
  if (resolved === 'logo') continue;
  if (!lucide.has(resolved)) {
    unresolved.push(name);
    continue;
  }
  coreNames.add(resolved);
}

const brand = brandLogoNode();

function moduleHeader(what) {
  return [
    '// GENERATED FILE — do not edit.',
    '// Regenerate with: node scripts/generate-icon-assets.mjs',
    `// ${what}`,
    '',
  ].join('\n');
}

const aliasEntries = Object.entries(ALIASES)
  .filter(([, target]) => lucide.has(target))
  .map(([from, to]) => `  ${JSON.stringify(from)}: ${JSON.stringify(to)},`)
  .join('\n');

const coreBody = [...coreNames].sort()
  .map((name) => `  ${JSON.stringify(name)}: ${JSON.stringify(lucide.get(name))},`)
  .join('\n');

const coreModule = `${moduleHeader(`Icons present on first paint (${coreNames.size} of ${lucide.size}).`)}
export const ICON_ALIASES = {
${aliasEntries}
};

// The Verstak mark, taken from packaging/linux/verstak.svg so the application
// icon and the interface logo cannot drift apart. It carries its own colours,
// unlike the stroke-based lucide set.
export const BRAND_ICON = {
  viewBox: ${JSON.stringify(brand.viewBox)},
  nodes: ${JSON.stringify(brand.nodes)},
};

export const CORE_ICONS = {
${coreBody}
};
`;

const spriteBody = [...lucide.keys()].sort()
  .filter((name) => !coreNames.has(name))
  .map((name) => `  ${JSON.stringify(name)}: ${JSON.stringify(lucide.get(name))},`)
  .join('\n');

const spriteModule = `${moduleHeader('Remaining icons, loaded on demand (folder icon picker).')}
export const SPRITE_ICONS = {
${spriteBody}
};
`;

const targets = [
  [path.join(outputDir, 'core.js'), coreModule],
  [path.join(outputDir, 'sprite.js'), spriteModule],
];

if (checkOnly) {
  let stale = false;
  for (const [file, content] of targets) {
    const current = fs.existsSync(file) ? fs.readFileSync(file, 'utf8') : '';
    if (current !== content) {
      console.error(`❌ ${path.relative(root, file)} is out of date`);
      stale = true;
    }
  }
  if (unresolved.length) {
    console.error(`❌ icon names with no lucide counterpart: ${unresolved.sort().join(', ')}`);
    console.error('   add an alias in scripts/generate-icon-assets.mjs or fix the name');
    stale = true;
  }
  if (stale) {
    console.error('   run: node scripts/generate-icon-assets.mjs');
    process.exit(1);
  }
  console.log(`icon assets are current (${coreNames.size} core, ${lucide.size - coreNames.size} lazy)`);
  process.exit(0);
}

if (unresolved.length) {
  console.error(`❌ icon names with no lucide counterpart: ${unresolved.sort().join(', ')}`);
  process.exit(1);
}

fs.mkdirSync(outputDir, { recursive: true });
for (const [file, content] of targets) fs.writeFileSync(file, content);

console.log(`generated ${coreNames.size} core icons and ${lucide.size - coreNames.size} lazy icons`);
