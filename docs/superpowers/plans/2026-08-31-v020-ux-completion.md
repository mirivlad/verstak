# Verstak v0.2.0 UX Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `executing-plans` task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the approved Deal UX cleanup and publish Verstak v0.2.0 from `main`.

**Architecture:** Keep Deal UUID as the sole resource scope. Changes are presentation and plugin-contribution changes: the shell owns tab paging and Deal-scoped global search, while official plugins own their views, configuration, and operation feedback. Project Meta and Milestones remain independent plugins; the shell composes their Deal presentation only when both are enabled.

**Tech Stack:** Go/Wails v2, Svelte 4, plain JavaScript official-plugin frontends, Node source-contract and smoke tests, Playwright E2E, Linux packaging scripts, GitHub Actions.

## Global Constraints

- Work directly on `main`, with bounded commits and a push after every green logical slice.
- Do not alter the accepted Deal-only scope, legacy migration rules, Git checkout Sync exclusion, or credential boundaries.
- Use test-first RED/GREEN cycles for behavior changes.
- Activity remains an enabled background provider for Journal, but gets no end-user navigation surface.
- Project identity remains the Deal UUID; Milestones must still work without Project Meta.
- `v0.2.0` is released only after fresh local gates and green GitHub Actions for the exact candidate commit.

---

## File map

### Desktop shell and packaging

- `frontend/src/lib/shell/WorkspaceHost.svelte`: paged tool strip, active-page behavior, and Project/Milestones composition.
- `frontend/src/lib/shell/GlobalSearch.svelte` and `frontend/src/App.svelte`: Deal-only search toggle and active Deal input.
- `frontend/src/lib/shell/WorkspaceTree.svelte`: create-Deal template picker entry point.
- `frontend/src/lib/settings/SettingsWindow.svelte`: templates settings panel host and accessibility repairs.
- `frontend/src/lib/test/wails-mock.js` and `frontend/tests/*.mjs`: real-manifest E2E fixture and source-contract coverage.
- `packaging/linux/verstak.desktop`, `scripts/package-deb.sh`, `scripts/package-appimage.sh`, and icon tests: XFCE-compatible icon identity and raster icon installation.

### Official plugins

- `plugins/activity/plugin.json`: hide end-user view/sidebar/workspace/overview/search contributions while retaining provider commands used by Journal.
- `plugins/templates/plugin.json`, `plugins/templates/frontend/src/index.js`: Settings panel and human-readable tool picker.
- `plugins/search/plugin.json`, `plugins/search/frontend/src/index.js`: provider-only search; Deal scope filtering.
- `plugins/projects/plugin.json`, `plugins/projects/frontend/src/index.js`: no redundant workspace title; composition surface.
- `plugins/milestones/plugin.json`, `plugins/milestones/frontend/src/index.js`: no sidebar entry, standalone fallback, and embedded mount mode.
- `plugins/journal/frontend/src/index.js`: grouped candidate review and responsive modal.
- `plugins/git/frontend/src/index.js`: visible per-repository busy state and result/error feedback.
- `scripts/smoke-*.js`: regression coverage for manifest surfaces and plugin behavior.

---

### Task 1: Tab paging and reduced Deal navigation

**Files:**

- Modify: `verstak-desktop/frontend/src/lib/shell/WorkspaceHost.svelte`
- Modify: `verstak-desktop/frontend/tests/shell-source-contract-test.mjs`
- Test: `verstak-desktop/frontend/e2e/workspace-tools.spec.js` (create if no existing workspace tab E2E owns this behavior)

**Interfaces:**

- Produces `tabPage` state with previous/next affordance instead of a horizontally scrollable list.
- `Overview` and the active tool are on the visible page; page changes preserve the selected tool.

- [ ] **Step 1: Write failing source/E2E assertions**

```js
assertExcludes(workspaceHost, 'overflow-x: auto', 'Deal tabs must not use a horizontal scrollbar');
assertIncludes(workspaceHost, 'data-workspace-tab-page-next', 'Deal tabs expose a next-page arrow');
assertIncludes(workspaceHost, 'data-workspace-tab-page-previous', 'Deal tabs expose a previous-page arrow');
```

- [ ] **Step 2: Verify RED**

Run: `node frontend/tests/shell-source-contract-test.mjs`

Expected: failure because the strip still uses `overflow-x: auto` and has no paging control.

- [ ] **Step 3: Implement minimal paged-strip layout**

```svelte
{#if hasPreviousTabPage}
  <button data-workspace-tab-page-previous aria-label={tr('workspace.previousTools')} on:click={showPreviousTabPage}>…</button>
{/if}
{#each visibleTools as tool (toolKey(tool))}
  <button data-workspace-tool={tool.id}>…</button>
{/each}
{#if hasNextTabPage}
  <button data-workspace-tab-page-next aria-label={tr('workspace.moreTools')} on:click={showNextTabPage}>…</button>
{/if}
```

Measure available strip width with `ResizeObserver`; recompute pages on resize, place the active tool on its page, and animate only the page transition.

- [ ] **Step 4: Verify GREEN and visual behavior**

Run:

```bash
node frontend/tests/shell-source-contract-test.mjs
npx playwright test e2e/workspace-tools.spec.js
```

- [ ] **Step 5: Commit and push**

```bash
git add frontend/src/lib/shell/WorkspaceHost.svelte frontend/tests
git commit -m "feat: page Deal tool tabs"
git push origin main
```

### Task 2: Hide technical tools, compose Planning, and scope global search

**Files:**

- Modify: `verstak-official-plugins/plugins/activity/plugin.json`
- Modify: `verstak-official-plugins/plugins/milestones/plugin.json`
- Modify: `verstak-official-plugins/plugins/projects/plugin.json`
- Modify: `verstak-official-plugins/plugins/search/plugin.json`
- Modify: `verstak-desktop/frontend/src/lib/shell/WorkspaceHost.svelte`
- Modify: `verstak-desktop/frontend/src/lib/shell/GlobalSearch.svelte`
- Modify: `verstak-desktop/frontend/src/App.svelte`
- Test: `verstak-official-plugins/scripts/smoke-activity-plugin.js`, `scripts/smoke-milestones-plugin.js`, `scripts/smoke-search-plugin.js`, desktop E2E scope test.

**Interfaces:**

- Activity keeps `activityProviders` and `worklogProviders`, but exposes no sidebar, workspace, overview, search, or palette entry.
- Search providers receive `{ query, limit, scope: { kind: 'deal', workspaceId } }` only when the checkbox is selected.
- Milestones renders inside Project only when both plugins are enabled; its standalone workspace item is available otherwise.

- [ ] **Step 1: Add failing manifest and E2E tests**

```js
assert.equal(activity.contributes.workspaceItems, undefined);
assert.equal(activity.contributes.sidebarItems, undefined);
expect(page.getByLabel('Искать в этом Деле')).toBeVisible();
await expect(searchProvider).toHaveBeenCalledWith(expect.objectContaining({
  scope: { kind: 'deal', workspaceId: dealId },
}));
```

- [ ] **Step 2: Verify RED**

Run: `node scripts/smoke-activity-plugin.js && node scripts/smoke-search-plugin.js`

Expected: failure because Activity still declares UI surfaces and search has no Deal scope.

- [ ] **Step 3: Implement the contribution and shell changes**

Remove only presentation contributions from Activity and Search. Add an unchecked, localized Deal-search checkbox beside the global field only while a Deal is open. Introduce an optional generic `embeddedWorkspaceItems` slot in the shell rather than a Project-to-Milestones storage dependency.

- [ ] **Step 4: Verify GREEN**

Run:

```bash
node scripts/smoke-activity-plugin.js
node scripts/smoke-milestones-plugin.js
node scripts/smoke-search-plugin.js
npx playwright test e2e/global-search.spec.js e2e/workspace-tools.spec.js
```

- [ ] **Step 5: Commit and push each repository**

```bash
git -C ../verstak-official-plugins add plugins scripts
git -C ../verstak-official-plugins commit -m "feat: simplify Deal tool surfaces"
git -C ../verstak-official-plugins push origin main
git add frontend/src frontend/tests
git commit -m "feat: scope global search to the active Deal"
git push origin main
```

### Task 3: Templates as Settings and usable Deal creation

**Files:**

- Modify: `verstak-official-plugins/plugins/templates/plugin.json`
- Modify: `verstak-official-plugins/plugins/templates/frontend/src/index.js`
- Modify: `verstak-desktop/frontend/src/lib/shell/WorkspaceTree.svelte`
- Modify: `verstak-desktop/frontend/src/lib/settings/SettingsWindow.svelte`
- Test: `verstak-official-plugins/scripts/smoke-templates-plugin.js`, desktop E2E template-picker scenario.

**Interfaces:**

- `verstak.templates.settings` mounts in Settings; no Templates sidebar contribution remains.
- The editor consumes `api.contributions.list('workspaceItems')` and renders selected tool IDs through titles/icons/descriptions.
- New Deal flow receives a selected persisted template ID, not manually typed plugin IDs.

- [ ] **Step 1: Write failing tests**

```js
assert.ok(manifest.contributes.settingsPanels.some((panel) => panel.id === 'verstak.templates.settings'));
assert.equal(manifest.contributes.sidebarItems, undefined);
assertIncludes(source, "data-template-tool", 'Template editor exposes a selectable tool catalog');
assertExcludes(source, 'Workspace plugin IDs (one per line)', 'Template editor must not ask for raw plugin IDs');
```

- [ ] **Step 2: Verify RED**

Run: `node scripts/smoke-templates-plugin.js`

Expected: failure because Templates is a sidebar view and uses a raw textarea for IDs.

- [ ] **Step 3: Implement Settings panel, checkbox catalog, and New Deal picker**

Use the installed enabled contribution catalog. Save only plugin IDs in the recipe, display only localized names and status, and keep a disabled missing-tool line if an existing template refers to a removed plugin. Move create-from-template selection to the New Deal dialog while retaining full CRUD in Settings.

- [ ] **Step 4: Verify GREEN**

Run:

```bash
node scripts/smoke-templates-plugin.js
npx playwright test e2e/settings.spec.js e2e/workspace-tree.spec.js
```

- [ ] **Step 5: Commit and push**

Commit official-plugin and desktop changes as separate repository commits, pushing each immediately after its focused tests are green.

### Task 4: Journal review and Git operation feedback

**Files:**

- Modify: `verstak-official-plugins/plugins/journal/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/git/frontend/src/index.js`
- Modify: `verstak-official-plugins/scripts/smoke-journal-plugin.js`
- Modify: `verstak-official-plugins/scripts/smoke-git-plugin.js`

**Interfaces:**

- Journal groups candidate activities by meaningful action/resource while retaining expandable event-level selection.
- Every Git operation sets `{ repoId, kind, startedAt }` until it settles and renders a localized operation label in the affected card.

- [ ] **Step 1: Add failing smoke assertions**

```js
assertIncludes(journal, 'journal-candidate-group', 'Journal renders grouped candidate activity');
assertIncludes(journal, 'max-width: min(900px, 96vw)', 'Journal review modal is desktop-width responsive');
assertIncludes(git, 'operationLabel', 'Git cards announce the active operation');
assertIncludes(git, "kind: 'push'", 'Git preserves the operation name while push is pending');
```

- [ ] **Step 2: Verify RED**

Run: `node scripts/smoke-journal-plugin.js && node scripts/smoke-git-plugin.js`

Expected: failure because candidate rows are flat and Git busy state has no visible operation label.

- [ ] **Step 3: Implement grouped review and operation status**

Keep underlying activity IDs and existing Git backend operations unchanged. Add an expandable group summary, select-all/none per group, a wider bounded modal with scrolling body, per-card spinner/text, conflict-safe action disabling, and a persistent success/error line after settlement.

- [ ] **Step 4: Verify GREEN and GUI path**

Run:

```bash
node scripts/smoke-journal-plugin.js
node scripts/smoke-git-plugin.js
bash scripts/gui-probe.sh
```

- [ ] **Step 5: Commit and push**

```bash
git add plugins/journal plugins/git scripts/smoke-journal-plugin.js scripts/smoke-git-plugin.js
git commit -m "feat: clarify journal review and Git operations"
git push origin main
```

### Task 5: Remove redundant headings and repair Linux taskbar identity

**Files:**

- Modify: `verstak-official-plugins/plugins/projects/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/milestones/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/git/frontend/src/index.js`
- Modify: `verstak-desktop/packaging/linux/verstak.desktop`
- Modify: `verstak-desktop/scripts/package-deb.sh`
- Modify: `verstak-desktop/scripts/package-appimage.sh`
- Modify: `verstak-desktop/scripts/test-brand-icons.sh`
- Test: plugin smokes and packaged icon contract test.

**Interfaces:**

- Workspace content does not repeat the selected top-level tool title; Project portfolio keeps its global heading.
- Linux distribution artifacts contain `hicolor` PNG sizes and matching `.desktop` `StartupWMClass`; no desktop environment relies exclusively on a scalable SVG.

- [ ] **Step 1: Write failing source/package assertions**

```js
assertExcludes(projectWorkspaceSource, "textContent: translate(api, 'ui.title', 'Project')", 'Workspace Project view must not repeat its tab title');
assertIncludes(packageDeb, '48x48/apps/verstak.png', 'Deb package contains standard raster app icon');
assertIncludes(desktopEntry, 'StartupWMClass=', 'Desktop entry declares the Wails window class');
```

- [ ] **Step 2: Verify RED**

Run: `node scripts/smoke-projects-plugin.js && bash scripts/test-brand-icons.sh`

Expected: failure because workspace headings and raster package icons are still absent.

- [ ] **Step 3: Implement smallest platform-correct changes**

Delete only workspace headings, retaining portfolio and empty-state context. Generate and install standard raster sizes from the canonical brand PNG, set the actual observed Linux Wails window class, and validate both `.deb` and AppImage trees. Confirm the class with an XFCE/X11 probe before committing.

- [ ] **Step 4: Verify GREEN**

Run:

```bash
node scripts/smoke-projects-plugin.js
node scripts/smoke-milestones-plugin.js
node scripts/smoke-git-plugin.js
bash scripts/test-brand-icons.sh
bash scripts/test-package-formats.sh
```

- [ ] **Step 5: Commit and push each repository**

Push the official-plugin heading change and desktop packaging change as separate commits after their relevant tests pass.

### Task 6: Accessibility, regression gates, documentation, and v0.2.0 release

**Files:**

- Modify: `verstak-desktop/frontend/src/lib/settings/SettingsWindow.svelte` and `frontend/src/lib/ui/Modal.svelte` only for currently emitted accessibility diagnostics.
- Modify: `verstak-desktop/docs/WORKSPACE_TEMPLATES.md`, `docs/GUI_TESTING.md`, and release notes.
- Create: `verstak-desktop/release-notes/v0.2.0.md`

- [ ] **Step 1: Write focused regression checks before accessibility fixes**

Add test assertions for the settings tablist focus target, valid tabpanel element, associated labels, and modal dialog semantics; execute them once against the current source.

- [ ] **Step 2: Implement and run full local gates**

Run:

```bash
cd ../verstak-sdk && bash scripts/check.sh
cd ../verstak-official-plugins && bash scripts/check.sh
cd ../verstak-desktop && bash scripts/test.sh && bash scripts/check.sh
cd frontend && npx playwright test
cd .. && bash scripts/verstak-check.sh
```

- [ ] **Step 3: Commit/push docs and release preparation**

Write canonical documentation only after behavior is proven. Commit and push the final Desktop documentation/release-note slice.

- [ ] **Step 4: Release exact candidate**

Dispatch the Desktop release workflow from `main` with version `v0.2.0`, `official_plugins_ref=main`, and `publish=true`. Wait for the full build and publish jobs, then verify the GitHub Release assets/checksums and the official-plugin release state if that repository has a corresponding release workflow.

