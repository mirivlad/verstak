import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const stylesheet = readFileSync(
  new URL('../src/lib/ui/design-system.css', import.meta.url),
  'utf8',
);
const main = readFileSync(new URL('../src/main.js', import.meta.url), 'utf8');
const app = readFileSync(new URL('../src/App.svelte', import.meta.url), 'utf8');

assert.match(main, /import ['"]\.\/lib\/ui\/design-system\.css['"]/);

for (const token of [
  '--vt-color-background',
  '--vt-color-surface',
  '--vt-color-control',
  '--vt-color-input',
  '--vt-color-text-primary',
  '--vt-color-text-muted',
  '--vt-color-accent',
  '--vt-color-danger',
  '--vt-color-warning',
  '--vt-color-success',
  '--vt-control-height',
  '--vt-control-height-compact',
  '--vt-focus-ring',
  '--vt-overlay-z-index',
]) {
  assert.ok(stylesheet.includes(token), `missing design token ${token}`);
}

for (const selector of [
  '.vt-page',
  '.vt-page-header',
  '.vt-toolbar',
  '.vt-toolbar-group',
  '.vt-toolbar-spacer',
  '.vt-button',
  '.vt-field',
  '.vt-input',
  '.vt-select',
  '.vt-textarea',
  '.vt-tabbar',
  '.vt-tab',
  '.vt-list-row',
  '.vt-row-actions',
  '.vt-card',
  '.vt-split-pane',
  '.vt-empty-state',
  '.vt-inline-alert',
  '.vt-badge',
  '.vt-menu',
  '.vt-menu-separator',
  '.vt-modal-overlay',
  '.vt-modal',
  '.scroll-surface',
]) {
  assert.ok(stylesheet.includes(selector), `missing primitive ${selector}`);
}

assert.match(stylesheet, /\.vt-button[^\n]*\.btn-secondary/);
assert.match(stylesheet, /\.vt-button\.primary[^\n]*\.btn-primary/);
assert.match(stylesheet, /:focus-visible/);
assert.match(stylesheet, /:disabled/);
assert.match(stylesheet, /\.vt-inline-alert\.warning/);
assert.match(stylesheet, /\.vt-badge\.danger/);
assert.match(stylesheet, /prefers-reduced-motion:\s*reduce/);
assert.doesNotMatch(app, /:global\(:root\)/);
assert.doesNotMatch(app, /:global\(\.vt-page\)/);

console.log('design system contract: ok');
