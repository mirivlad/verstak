// The icon vocabulary contract.
//
// Icon.svelte once resolved icons with import.meta.glob over the whole lucide
// package. That emitted 1703 chunks into dist, made every icon arrive a frame
// late, and hid a real bug: the Verstak logo silently fell back to a generic
// circle because no lucide icon is called "logo".
//
// Four things are asserted here:
//   1. no glob-based resolution comes back;
//   2. the generated core and sprite modules are current;
//   3. every icon named by the shell or an official plugin manifest is in the
//      core set, so it is present at first paint;
//   4. a production build stays within a chunk budget.
import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert';

const frontend = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repo = path.resolve(frontend, '..');

// ── 1. No glob resolution ───────────────────────────────────────────────────
const iconSource = readFileSync(path.join(frontend, 'src', 'lib', 'ui', 'Icon.svelte'), 'utf8');
assert.doesNotMatch(
  iconSource,
  /import\.meta\.glob/,
  'Icon.svelte must not resolve icons with import.meta.glob; use the generated core/sprite modules',
);
assert.match(iconSource, /icons\/core\.js/, 'Icon.svelte must import the generated core set');
assert.match(iconSource, /icons\/sprite\.js/, 'Icon.svelte must load the remaining icons lazily');
assert.match(
  iconSource,
  /warnUnknown/,
  'an unknown icon must warn rather than silently render something else',
);

// ── 2. Generated modules are current ────────────────────────────────────────
execFileSync('node', ['scripts/generate-icon-assets.mjs', '--check'], {
  cwd: repo,
  stdio: 'inherit',
});

// ── 3. Named icons are in the core set ──────────────────────────────────────
const coreSource = readFileSync(
  path.join(frontend, 'src', 'lib', 'ui', 'icons', 'core.js'),
  'utf8',
);
const coreNames = new Set([...coreSource.matchAll(/^ {2}"([a-z0-9-]+)":/gm)].map((m) => m[1]));
const aliasBlock = coreSource.match(/export const ICON_ALIASES = \{([\s\S]*?)\n\};/);
const aliases = new Map(
  [...(aliasBlock?.[1] || '').matchAll(/"([^"]+)":\s*"([^"]+)"/g)].map((m) => [m[1], m[2]]),
);

// Only the sibling source repository, never `repo/plugins`: that directory is
// a gitignored install copy, so a plugin left behind by an older
// install-dev-plugins.sh run fails this assertion on one machine and passes on
// another. The generator applies the same rule for the same reason.
const sourcePluginRoot = path.join(repo, '..', 'verstak-official-plugins', 'plugins');
assert.ok(
  existsSync(sourcePluginRoot),
  `verstak-official-plugins must be checked out beside this repository (expected ${sourcePluginRoot})`,
);
const pluginRoots = [sourcePluginRoot];
const missing = [];
for (const pluginRoot of pluginRoots) {
  if (!existsSync(pluginRoot)) continue;
  for (const entry of readdirSync(pluginRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const manifestPath = path.join(pluginRoot, entry.name, 'plugin.json');
    if (!existsSync(manifestPath)) continue;
    const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
    const named = [manifest.icon];
    for (const value of Object.values(manifest.contributes || {})) {
      if (Array.isArray(value)) for (const item of value) named.push(item?.icon);
    }
    for (const icon of named.filter(Boolean)) {
      const resolved = aliases.get(icon) || icon;
      if (!coreNames.has(resolved)) missing.push(`${entry.name}: ${icon}`);
    }
  }
}
assert.deepStrictEqual(
  missing,
  [],
  'plugin manifest icons must be in the core set — run node scripts/generate-icon-assets.mjs',
);

// ── 4. Chunk budget ─────────────────────────────────────────────────────────
const assets = path.join(frontend, 'dist', 'assets');
if (!existsSync(assets) || process.env.VERSTAK_REBUILD_DIST === '1') {
  execFileSync('npx', ['vite', 'build'], { cwd: frontend, stdio: 'inherit' });
}
const chunks = readdirSync(assets).filter((name) => name.endsWith('.js'));
const BUDGET = 25;
assert.ok(
  chunks.length <= BUDGET,
  `production build emitted ${chunks.length} JavaScript chunks, budget is ${BUDGET}. `
  + 'A per-icon dynamic import is the usual cause.',
);

console.log(
  `icon contract: ok (${coreNames.size} core icons, ${chunks.length} chunk(s) within budget ${BUDGET})`,
);
