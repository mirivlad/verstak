# Verstak Design System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract and formalize Verstak's host-owned visual contract, then migrate repeated desktop and official-plugin surfaces without changing feature behavior.

**Architecture:** Import one global `design-system.css` from the desktop entry point. The stylesheet owns stable `--vt-*` foundations, composable `vt-*` primitives, and temporary `btn-*` compatibility aliases; Svelte components and plain-JS official plugins retain only component-specific layout and behavior CSS.

**Tech Stack:** Svelte 4, Vite 5, plain JavaScript plugin bundles, Node contract tests, Playwright, CSS custom properties.

## Global Constraints

- This is a visual-architecture refactor: no backend API, Wails binding, plugin manifest, persistence, command, or feature behavior changes.
- Do not add a UI dependency, npm package, theme switcher, or public SDK contract.
- Preserve all current DOM data attributes, Svelte events, API calls, translated copy, keyboard behavior, responsive breakpoints, and test selectors.
- Standard controls remain 32 pixels high; compact controls remain 26 pixels high.
- Official plugin styles retain CSS-variable fallback values for isolated rendering.
- Keep component-specific layout local and move only reusable foundations and state treatment into the host contract.

---

### Task 1: Extract and test the host CSS contract

**Files:**

- Create: `frontend/src/lib/ui/design-system.css`
- Create: `frontend/tests/design-system-contract-test.mjs`
- Modify: `frontend/src/main.js`
- Modify: `frontend/src/App.svelte`

**Interfaces:**

- Produces: globally imported `--vt-*` variables and the `vt-*` primitive class families specified in `docs/superpowers/specs/2026-07-24-design-system.md`.
- Preserves: `btn-primary`, `btn-secondary`, `btn-danger`, `btn-ghost`, and `btn-icon` as selector aliases.

- [ ] **Step 1: Write the failing stylesheet contract test**

Create `frontend/tests/design-system-contract-test.mjs` with assertions for
the import boundary, stable foundations, every primitive family, interaction
states, aliases, and reduced motion:

```js
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
```

- [ ] **Step 2: Run the contract test and confirm the missing stylesheet failure**

Run:

```bash
node frontend/tests/design-system-contract-test.mjs
```

Expected: FAIL with `ENOENT` for
`frontend/src/lib/ui/design-system.css`.

- [ ] **Step 3: Extract foundations and primitives**

Move the global reset, `:root` tokens, base button rules, current reusable
`vt-*` selectors, and scrollbar rules from `App.svelte` into
`design-system.css`. Add the missing semantic foundations:

```css
:root {
  --vt-color-control: #1b2440;
  --vt-color-control-hover: #203050;
  --vt-color-input: #0f1424;
  --vt-color-placeholder: #69758f;
  --vt-color-danger-foreground: #ffc6ce;
  --vt-color-warning-foreground: #ffe0a3;
  --vt-color-success-foreground: #9fe0c9;
  --vt-control-height: 2rem;
  --vt-control-height-compact: 1.625rem;
  --vt-transition-fast: 150ms ease;
  --vt-overlay-backdrop: rgba(4, 8, 18, 0.7);
  --vt-overlay-z-index: 10000;
}
```

Implement the missing primitives with composable modifiers. Use combined
selectors for legacy aliases:

```css
.vt-button,
.btn-secondary {
  min-height: var(--vt-control-height);
  border: 1px solid var(--vt-color-border-strong);
  border-radius: var(--vt-radius-md);
  background: var(--vt-color-control);
  color: var(--vt-color-text-primary);
}

.vt-button.primary,
.btn-primary {
  background: var(--vt-color-accent);
  border-color: var(--vt-color-accent);
  color: #101827;
}

.vt-button.compact {
  min-height: var(--vt-control-height-compact);
  padding-block: 0.25rem;
}

.vt-control,
.vt-input,
.vt-select,
.vt-textarea {
  min-height: var(--vt-control-height);
  border: 1px solid var(--vt-color-border-strong);
  border-radius: var(--vt-radius-sm);
  background: var(--vt-color-input);
  color: var(--vt-color-text-primary);
  font: inherit;
}

.vt-control:focus-visible,
.vt-input:focus-visible,
.vt-select:focus-visible,
.vt-textarea:focus-visible {
  outline: 0;
  border-color: var(--vt-color-accent);
  box-shadow: var(--vt-focus-ring);
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    transition-duration: 0s !important;
    animation-duration: 0s !important;
  }
}
```

Keep application-only `.app-loading`, `main`, `.main-content`, responsive
shell layout, and related structural selectors in `App.svelte`.

- [ ] **Step 4: Import the stylesheet before application startup**

Add this as the first local import in `frontend/src/main.js`:

```js
import './lib/ui/design-system.css';
```

- [ ] **Step 5: Run contract and build checks**

Run:

```bash
node frontend/tests/design-system-contract-test.mjs
node frontend/tests/select-styles-test.mjs
cd frontend && npm run build
```

Expected: both Node tests print their success messages; Vite exits 0 with
only the repository's known advisory warnings.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/lib/ui/design-system.css \
  frontend/tests/design-system-contract-test.mjs \
  frontend/src/main.js frontend/src/App.svelte
git commit -m "refactor(ui): extract design system contract"
```

### Task 2: Compose core controls from the shared primitives

**Files:**

- Modify: `frontend/src/lib/ui/Modal.svelte`
- Modify: `frontend/src/lib/ui/Select.svelte`
- Modify: `frontend/src/lib/shell/WorkspaceTree.svelte`
- Modify: `frontend/src/lib/shell/ViewContainer.svelte`
- Modify: `frontend/tests/design-system-contract-test.mjs`
- Test: `frontend/e2e/workspace-templates.spec.js`
- Test: `frontend/e2e/workspace-tree-overlay.spec.js`

**Interfaces:**

- Consumes: the Task 1 `vt-button`, form-control, menu, modal, alert, badge,
  and empty-state primitives.
- Preserves: existing `vt-btn*`, `vt-ctx*`, modal, Select, and workspace-tree
  selectors until their focused Playwright coverage proves unchanged.

- [ ] **Step 1: Extend the contract test with component composition assertions**

Append source assertions:

```js
const modal = readFileSync(new URL('../src/lib/ui/Modal.svelte', import.meta.url), 'utf8');
const select = readFileSync(new URL('../src/lib/ui/Select.svelte', import.meta.url), 'utf8');
const tree = readFileSync(new URL('../src/lib/shell/WorkspaceTree.svelte', import.meta.url), 'utf8');

assert.match(modal, /class="vt-modal-overlay"/);
assert.match(modal, /class="vt-modal"/);
assert.doesNotMatch(modal, /background:\s*rgba\(4,\s*8,\s*18/);
assert.match(select, /class="vt-select"/);
assert.doesNotMatch(select, /background:\s*#0f1424/);
assert.match(tree, /class="vt-button secondary vt-btn"/);
assert.match(tree, /class="vt-menu vt-ctx"/);
```

- [ ] **Step 2: Run the test and confirm it fails on uncomposed tree classes**

Run:

```bash
node frontend/tests/design-system-contract-test.mjs
```

Expected: FAIL on `vt-button secondary vt-btn` or `vt-menu vt-ctx`.

- [ ] **Step 3: Remove reusable CSS from Modal and Select**

Keep only component geometry that is not part of the global contract:

```css
/* Modal.svelte */
.vt-modal-wide { width: min(36rem, 100%); }
.vt-modal-header h2 {
  margin: 0;
  color: var(--vt-color-text-primary);
  font-size: var(--vt-font-lg);
}
```

```css
/* Select.svelte */
.vt-select-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
  width: 100%;
}
.vt-select-wrap.disabled { opacity: 0.55; cursor: not-allowed; }
.vt-select { width: 100%; padding-right: 1.7rem; appearance: none; }
.vt-select-arrow {
  position: absolute;
  right: var(--vt-space-2);
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  pointer-events: none;
  color: var(--vt-color-text-muted);
}
```

- [ ] **Step 4: Compose tree forms and menus without removing local hooks**

Add semantic classes alongside the current test and layout hooks:

```svelte
<div class="vt-menu vt-ctx" on:click|stopPropagation on:mousedown|stopPropagation on:keydown={(event) => event.key === 'Escape' && closeCtx()} role="menu" tabindex="-1">
  <button class="vt-menu-item vt-ctx-i" on:click={() => { const i = ctxMenu.id; closeCtx(); openCreateWorkspace(i); }}>{tr('workspaceTree.newDeal')}</button>
  <div class="vt-menu-separator vt-ctx-s" />
</div>

<input class="vt-input wt-rename" type="text" bind:value={formName} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doRename()} />
<button class="vt-button secondary vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button>
<button class="vt-button primary vt-btn-p" on:click={doCreateWorkspace} disabled={formBusy}>{tr('workspaceTree.create')}</button>
<button class="vt-button danger vt-btn-d" on:click={doTrash} disabled={formBusy}>{tr('workspaceTree.toTrash')}</button>
```

Use `vt-inline-alert error` on the existing tree and view-container error
elements. Remove only declarations now supplied by a shared primitive; retain
position, width, grid, overflow, and DnD rules.

- [ ] **Step 5: Run focused component and interaction checks**

Run:

```bash
node frontend/tests/design-system-contract-test.mjs
node frontend/tests/select-styles-test.mjs
cd frontend && npx playwright test --config playwright.config.js \
  e2e/workspace-templates.spec.js e2e/workspace-tree-overlay.spec.js
```

Expected: Node checks pass and all selected Playwright tests pass.

- [ ] **Step 6: Run Svelte diagnostics**

Run Svelte diagnostics for `Modal.svelte`, `Select.svelte`,
`WorkspaceTree.svelte`, and `ViewContainer.svelte`. Expected: no new errors;
baseline accessibility warnings, if any, remain unchanged.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/lib/ui/Modal.svelte \
  frontend/src/lib/ui/Select.svelte \
  frontend/src/lib/shell/WorkspaceTree.svelte \
  frontend/src/lib/shell/ViewContainer.svelte \
  frontend/tests/design-system-contract-test.mjs
git commit -m "refactor(ui): compose shell controls from primitives"
```

### Task 3: Normalize Plugin Manager and shell state surfaces

**Files:**

- Modify: `frontend/src/lib/plugin-manager/PluginManager.svelte`
- Modify: `frontend/src/lib/plugin-manager/PluginCard.svelte`
- Modify: `frontend/src/lib/plugin-host/PluginBundleHost.svelte`
- Modify: `frontend/src/lib/shell/GlobalSearch.svelte`
- Modify: `frontend/src/lib/shell/TodaySurface.svelte`
- Modify: `frontend/tests/design-system-contract-test.mjs`
- Test: `frontend/e2e/plugin-manager-layout.spec.js`
- Test: `frontend/e2e/plugin-manager-disable-enable.spec.js`
- Test: `frontend/e2e/global-search-results.spec.js`
- Test: `frontend/e2e/ux-today.spec.js`

**Interfaces:**

- Consumes: Task 1 tokens and primitives.
- Preserves: plugin enable/disable behavior, diagnostics visibility, filters,
  search routing, Today data loading, and every existing data attribute.

- [ ] **Step 1: Add failing source checks for semantic state composition**

Append:

```js
const pluginManager = readFileSync(
  new URL('../src/lib/plugin-manager/PluginManager.svelte', import.meta.url),
  'utf8',
);
const pluginCard = readFileSync(
  new URL('../src/lib/plugin-manager/PluginCard.svelte', import.meta.url),
  'utf8',
);
assert.match(pluginManager, /class="[^"]*vt-toolbar/);
assert.match(pluginManager, /class="[^"]*vt-empty-state/);
assert.match(pluginManager, /class="[^"]*vt-inline-alert/);
assert.match(pluginCard, /class="[^"]*vt-card/);
assert.doesNotMatch(pluginManager, /background:\s*#16213e/);
```

- [ ] **Step 2: Run the contract test and confirm the first missing class**

Run:

```bash
node frontend/tests/design-system-contract-test.mjs
```

Expected: FAIL on a missing Plugin Manager primitive.

- [ ] **Step 3: Compose existing shell markup**

Add shared classes while retaining local hooks:

```svelte
<header class="vt-page-header">
  <div class="header-left"></div>
</header>
<div class="summary vt-toolbar">
  <span class="badge vt-badge">{tr('pluginManager.summary.plugins', { count: totalPlugins })}</span>
</div>
<div class="empty vt-empty-state">
  <p>{tr('pluginManager.none')}</p>
</div>
<div class="error vt-inline-alert error">
  <div class="error-message">{error}</div>
</div>
<div class="plugin-card vt-card" class:disabled={isDisabled} class:failed={p.status === 'failed'}>
</div>
```

Apply the same rule to Plugin Bundle Host failures, Global Search result/empty
states, and Today cards/alerts. Do not change markup order or conditional
logic.

- [ ] **Step 4: Replace common hard-coded colors**

Within the touched style blocks, map current shared palette literals:

```css
#16213e -> var(--vt-color-surface)
#121a2c -> var(--vt-color-surface-muted)
#0f3460 -> var(--vt-color-border-strong)
#e0e0f0 -> var(--vt-color-text-primary)
#a0a0b8 -> var(--vt-color-text-muted)
#4ecca3 -> var(--vt-color-accent)
#e94560 -> var(--vt-color-danger)
```

Retain unique visualization colors only where they encode an existing
category and no semantic token matches.

- [ ] **Step 5: Run focused shell tests and build**

Run:

```bash
node frontend/tests/design-system-contract-test.mjs
cd frontend && npx playwright test --config playwright.config.js \
  e2e/plugin-manager-layout.spec.js \
  e2e/plugin-manager-disable-enable.spec.js \
  e2e/global-search-results.spec.js \
  e2e/ux-today.spec.js
npm run build
```

Expected: all focused Playwright scenarios and the Vite build pass.

- [ ] **Step 6: Run Svelte diagnostics**

Run diagnostics on the five touched Svelte components. Expected: no new
errors.

- [ ] **Step 7: Commit**

```bash
git add frontend/src/lib/plugin-manager/PluginManager.svelte \
  frontend/src/lib/plugin-manager/PluginCard.svelte \
  frontend/src/lib/plugin-host/PluginBundleHost.svelte \
  frontend/src/lib/shell/GlobalSearch.svelte \
  frontend/src/lib/shell/TodaySurface.svelte \
  frontend/tests/design-system-contract-test.mjs
git commit -m "refactor(ui): normalize shell state surfaces"
```

### Task 4: Adopt the host contract in official plugins

**Files:**

- Create: `verstak-official-plugins/scripts/smoke-design-system.js`
- Modify: `verstak-official-plugins/scripts/check.sh`
- Modify: `verstak-official-plugins/plugins/files/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/notes/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/search/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/activity/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/journal/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/secrets/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/browser-inbox/frontend/src/index.js`

**Interfaces:**

- Consumes: host `vt-*` primitives and inherited `--vt-*` variables.
- Preserves: each plugin's current local class, data attribute, mount/unmount
  contract, keyboard/mouse handling, translated text, and API calls.

- [ ] **Step 1: Write the failing plugin adoption smoke**

Create a source-level smoke that requires the expected composable classes
without depending on browser CSS computation:

```js
#!/usr/bin/env node
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '..');
const contracts = {
  files: ['vt-toolbar', 'vt-button', 'vt-list-row', 'vt-menu', 'vt-modal'],
  notes: ['vt-toolbar', 'vt-button', 'vt-list-row', 'vt-empty-state', 'vt-modal'],
  search: ['vt-toolbar', 'vt-input', 'vt-card', 'vt-badge', 'vt-inline-alert'],
  activity: ['vt-toolbar', 'vt-button', 'vt-list-row', 'vt-inline-alert'],
  journal: ['vt-toolbar', 'vt-button', 'vt-card', 'vt-modal'],
  secrets: ['vt-split-pane', 'vt-list-row', 'vt-card', 'vt-input', 'vt-modal'],
  'browser-inbox': ['vt-toolbar', 'vt-split-pane', 'vt-list-row', 'vt-badge'],
};

for (const [plugin, classes] of Object.entries(contracts)) {
  const source = fs.readFileSync(
    path.join(root, 'plugins', plugin, 'frontend', 'src', 'index.js'),
    'utf8',
  );
  for (const className of classes) {
    assert.ok(source.includes(className), `${plugin} must compose ${className}`);
  }
  assert.match(source, /var\(--vt-color-background,/);
  assert.match(source, /var\(--vt-color-text-primary,/);
}

console.log('official plugin design-system adoption: ok');
```

Add `node scripts/smoke-design-system.js` beside the other smoke commands in
`scripts/check.sh`.

- [ ] **Step 2: Run the smoke and confirm missing classes**

Run:

```bash
node scripts/smoke-design-system.js
```

Expected: FAIL with the first missing `vt-*` class.

- [ ] **Step 3: Compose plugin markup with semantic classes**

Keep local hooks and add shared classes:

```js
el('div', { className: 'files-toolbar vt-toolbar' });
el('button', { className: 'files-toolbar-btn vt-button compact' });
el('div', { className: 'files-item vt-list-row' });
el('div', { className: 'files-ctx-menu vt-menu' });
el('div', { className: 'files-modal vt-modal' });
```

Apply the equivalent classes declared by the smoke contract to Notes, Search,
Activity, Journal, Secrets, and Browser Inbox. When state classes are
assembled dynamically, include both plugin-local and semantic modifiers in
the same string.

- [ ] **Step 4: Remove duplicated primitive declarations**

From each `STYLES` string, remove only declarations now guaranteed by the
host primitive. Retain local layout and token-backed fallbacks. Replace
common literals with the matching token and fallback:

```css
background: var(--vt-color-input, #0f1424);
color: var(--vt-color-danger-foreground, #ffc6ce);
min-height: var(--vt-control-height, 2rem);
box-shadow: var(--vt-elevation-menu, 0 14px 32px rgba(0,0,0,.42));
```

Do not remove plugin-prefixed class selectors: they remain stable local and
test hooks.

- [ ] **Step 5: Run the adoption smoke and all official checks**

Run:

```bash
node scripts/smoke-design-system.js
./scripts/check.sh
```

Expected: the adoption smoke prints its success line and the full official
plugin check exits 0.

- [ ] **Step 6: Commit the official-plugin migration**

```bash
git add scripts/smoke-design-system.js scripts/check.sh \
  plugins/files/frontend/src/index.js \
  plugins/notes/frontend/src/index.js \
  plugins/search/frontend/src/index.js \
  plugins/activity/frontend/src/index.js \
  plugins/journal/frontend/src/index.js \
  plugins/secrets/frontend/src/index.js \
  plugins/browser-inbox/frontend/src/index.js
git commit -m "refactor(ui): adopt host design primitives"
```

### Task 5: Verify behavior and visual integrity

**Files:**

- Create: `docs/ui-polish-assets/2026-07-24/design-system-shell.png`
- Create: `docs/ui-polish-assets/2026-07-24/design-system-files.png`
- Create: `docs/ui-polish-assets/2026-07-24/design-system-search.png`
- Create: `docs/ui-polish-assets/2026-07-24/design-system-split-pane.png`
- Create: `docs/ui-polish-assets/2026-07-24/design-system-narrow.png`
- Create: `frontend/e2e/design-system-visual.spec.js`
- Modify: `docs/ui-polish-milestone.md`

**Interfaces:**

- Consumes: all previous tasks.
- Produces: current visual evidence and a concise record of the Stage 2
  contract location and verification.

- [ ] **Step 1: Run desktop source and binding checks**

Run from `verstak-desktop`:

```bash
node frontend/tests/design-system-contract-test.mjs
node frontend/tests/select-styles-test.mjs
node frontend/tests/shell-source-contract-test.mjs
node frontend/tests/wails-bindings-test.mjs
```

Expected: all four checks exit 0.

- [ ] **Step 2: Run desktop build and backend tests**

Run:

```bash
cd frontend && npm run build
cd .. && go test ./...
```

Expected: Vite and every Go package exit 0.

- [ ] **Step 3: Run focused cross-repository UI scenarios**

Run:

```bash
cd frontend && npx playwright test --config playwright.config.js \
  e2e/files-plugin.spec.js \
  e2e/activity.spec.js \
  e2e/browser-inbox.spec.js \
  e2e/journal-global.spec.js \
  e2e/secrets-global.spec.js \
  e2e/global-search-results.spec.js \
  e2e/plugin-manager-layout.spec.js \
  e2e/workspace-tree-overlay.spec.js
```

Expected: every selected scenario passes.

- [ ] **Step 4: Add and run deterministic screenshot coverage**

Create `frontend/e2e/design-system-visual.spec.js`:

```js
import { test, expect } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { resetMockState, waitForAppReady } from './helpers.js';

const currentDir = dirname(fileURLToPath(import.meta.url));
const outputDir = resolve(currentDir, '../../docs/ui-polish-assets/2026-07-24');

test('captures representative design-system surfaces', async ({ page }) => {
  mkdirSync(outputDir, { recursive: true });
  await page.goto('/');
  await waitForAppReady(page);
  await resetMockState(page);

  await page.screenshot({
    path: resolve(outputDir, 'design-system-shell.png'),
    fullPage: true,
  });

  await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
  await page.getByRole('tab', { name: 'Files' }).click();
  await expect(page.locator('.files-root')).toBeVisible();
  await page.screenshot({
    path: resolve(outputDir, 'design-system-files.png'),
    fullPage: true,
  });

  await page.evaluate(() => {
    window.__wailsMock.putVaultFile('Project/Files/design-system-result.txt', 'design system result');
    window.dispatchEvent(new CustomEvent('verstak:files-changed'));
  });
  await page.locator('[data-global-search-input]').fill('design-system-result');
  await expect(page.locator('[data-global-search-results]')).toBeVisible();
  await page.screenshot({
    path: resolve(outputDir, 'design-system-search.png'),
    fullPage: true,
  });

  await page.keyboard.press('Escape');
  await page.locator('.sidebar .plugin-item').filter({ hasText: 'Secrets' }).click();
  await expect(page.locator('.secrets-root')).toBeVisible();
  await page.screenshot({
    path: resolve(outputDir, 'design-system-split-pane.png'),
    fullPage: true,
  });

  await page.setViewportSize({ width: 720, height: 640 });
  await page.screenshot({
    path: resolve(outputDir, 'design-system-narrow.png'),
    fullPage: true,
  });
});
```

Run:

```bash
cd frontend && npx playwright test --config playwright.config.js \
  e2e/design-system-visual.spec.js
```

Expected: one Playwright test passes and writes five non-empty PNG files.
Inspect every image for clipping, overflow, unreadable text, broken
focus/selected states, inconsistent control height, and excessive density
changes.

- [ ] **Step 5: Run full desktop and compatibility gates**

Run:

```bash
cd frontend && npm run test:e2e
cd ../../verstak-official-plugins && ./scripts/check.sh
cd ../verstak-sdk && npm run lint && npm test && npm run build
```

Expected: full Playwright, official plugins, and SDK gates all exit 0.

- [ ] **Step 6: Update the milestone record**

Append a Stage 2 completion section to `docs/ui-polish-milestone.md` listing:

- `frontend/src/lib/ui/design-system.css` as the host contract;
- the seven migrated official plugins;
- the contract, build, full Playwright, official-plugin, SDK, and screenshot
  evidence;
- the explicit statement that no backend, manifest, SDK, persistence, or
  feature behavior changed.

- [ ] **Step 7: Run final diff checks and commit evidence**

Run:

```bash
git diff --check
git status --short
git diff --stat v0.1.0-beta.20260723.1..HEAD
```

Expected: no whitespace errors; only design-system, planned component, test,
documentation, and screenshot files appear.

Commit:

```bash
git add docs/ui-polish-milestone.md docs/ui-polish-assets/2026-07-24 \
  frontend/e2e/design-system-visual.spec.js
git commit -m "docs(ui): record design system verification"
```
