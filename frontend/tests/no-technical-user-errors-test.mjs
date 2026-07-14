import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const files = [
  'src/lib/shell/CommandPalette.svelte',
  'src/lib/plugin-manager/PluginManager.svelte',
  'src/lib/shell/VaultSelection.svelte',
  'src/lib/shell/WorkspaceTree.svelte',
  'src/lib/plugin-host/PluginBundleHost.svelte',
];
const patterns = [
  /(?:^|\n)\s*(?:error|localError|createError)\s*=\s*(?:err|String\()/,
  /(?:^|\n)\s*error\s*=\s*['"].*['"]\s*\+\s*(?:err|String\()/,
  /tr\([^\n]*\{\s*error\s*:/,
];
const violations = [];

for (const relativePath of files) {
  const source = fs.readFileSync(path.join(root, relativePath), 'utf8');
  for (const pattern of patterns) {
    if (pattern.test(source)) violations.push(relativePath);
  }
}

if (violations.length) {
  throw new Error(`Technical backend errors reach desktop UI in: ${[...new Set(violations)].join(', ')}`);
}

const bundleHost = fs.readFileSync(path.join(root, 'src/lib/plugin-host/PluginBundleHost.svelte'), 'utf8');
if (!/<p class="error-message">\{errorText \|\| tr\('bundle\.unknownError'\)\}<\/p>\s*<details class="error-details">/.test(bundleHost)) {
  throw new Error('Plugin bundle errors must keep the user-facing message outside expandable technical details');
}

console.log('desktop UI does not interpolate raw backend errors');
