# Beta Desktop UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Standardize settings and overview layouts, mount compact plugin status handlers, expose semantic Deals to plugins, and make global search progressive and truthful.

**Architecture:** Keep the desktop as a generic host. Add focused host components/APIs, add only observable sync running state to the existing backend transport, and update shell layouts/search without embedding Sync or Browser Inbox presentation in core.

**Tech Stack:** Go, Wails v2, Svelte, JavaScript, Node contract tests, Playwright Chromium, shell release scripts.

## Global Constraints

- Sync scheduling, labels, settings, and user-facing behavior remain in the Sync plugin.
- Settings contributions use a centered surface at exactly 90% of usable modal width.
- Deal listing is read-only and returns only workspace nodes, never folders.
- Search never renders an authoritative empty state while the applicable index is still building.
- Run LSP diagnostics before and after non-trivial Go/TypeScript changes.
- Build local release packages without publishing them.

---

### Task 1: Settings surface and overview layout

**Files:**
- Modify: `frontend/src/lib/plugin-manager/PluginManager.svelte`
- Modify: `frontend/src/lib/shell/TodaySurface.svelte`
- Test: `frontend/e2e/ux-followup.spec.js`
- Test: `frontend/e2e/ux-today.spec.js`

**Interfaces:**
- Produces: `.plugin-settings-surface` with 90% width.
- Produces: an overview layout class/state that omits the empty side column.

- [ ] **Step 1: Add failing Playwright layout assertions**

Measure modal body and settings surface bounding boxes and assert the surface is centered and `surface.width / body.width` is within `0.88..0.92`. Seed an overview with no side content and assert its main region reaches the right content edge and no empty aside is rendered.

- [ ] **Step 2: Run focused Playwright tests and confirm failure**

Run: `cd frontend && npx playwright test e2e/ux-followup.spec.js e2e/ux-today.spec.js --config playwright.config.js`

Expected: FAIL for narrow settings content and persistent overview side column.

- [ ] **Step 3: Add the host settings surface**

Wrap `PluginBundleHost` with a centered 90% surface, keep modal-body scrolling, and use adaptive padding with `box-sizing:border-box`. Do not add a narrow maximum width.

- [ ] **Step 4: Make overview grids content-aware**

Use an auto-fitting summary-card grid. Compute whether side content exists; render the `<aside>` only when true and apply a single-column layout otherwise.

- [ ] **Step 5: Run focused tests, commit, and push**

Run: `cd frontend && npx playwright test e2e/ux-followup.spec.js e2e/ux-today.spec.js --config playwright.config.js`

Expected: PASS.

```bash
git add frontend/src/lib/plugin-manager/PluginManager.svelte frontend/src/lib/shell/TodaySurface.svelte frontend/e2e/ux-followup.spec.js frontend/e2e/ux-today.spec.js
git commit -m "fix(ui): use available settings and overview space"
git push origin fix/beta-readiness-2026-07-20
```

### Task 2: Compact plugin status component host

**Files:**
- Create: `frontend/src/lib/plugin-host/CompactPluginHost.svelte`
- Modify: `frontend/src/lib/shell/StatusBar.svelte`
- Modify: `frontend/tests/shell-source-contract-test.mjs`
- Modify: `frontend/e2e/status-bar.spec.js`
- Modify: `frontend/e2e/ux-p0.spec.js`

**Interfaces:**
- Consumes: `pluginId`, `handler`, existing bundle loader/registry, and `createPluginAPI(pluginId)`.
- Produces: compact handler mount with cleanup and `[data-plugin-status-handler]`.

- [ ] **Step 1: Replace the old source-contract expectation with a failing handler-host expectation**

Require `StatusBar.svelte` to render `CompactPluginHost` for `item.handler` and forbid the literal `compact status only` fallback.

- [ ] **Step 2: Run the contract test and confirm failure**

Run: `node frontend/tests/shell-source-contract-test.mjs`

Expected: FAIL because handler items are still static warning labels.

- [ ] **Step 3: Implement the compact host**

Reuse the established asset loading and plugin registry resolution from `PluginBundleHost.svelte`, but render only a small loading/failure marker. Mount the declared component with `{ api }`; on changes/unmount call component cleanup and `api.dispose()`.

- [ ] **Step 4: Render handlers in all status-bar positions**

For handler items mount `CompactPluginHost`; for label-only items keep the existing span. Preserve item ordering and settings controls.

- [ ] **Step 5: Add Playwright coverage**

Assert the Sync handler is mounted, a handler failure stays within status-bar height, and clicking the mounted component can dispatch settings navigation.

- [ ] **Step 6: Run tests, commit, and push**

Run: `node frontend/tests/shell-source-contract-test.mjs`

Run: `cd frontend && npx playwright test e2e/status-bar.spec.js e2e/ux-p0.spec.js --config playwright.config.js`

Expected: PASS.

```bash
git add frontend/src/lib/plugin-host/CompactPluginHost.svelte frontend/src/lib/shell/StatusBar.svelte frontend/tests/shell-source-contract-test.mjs frontend/e2e/status-bar.spec.js frontend/e2e/ux-p0.spec.js
git commit -m "fix(plugins): mount compact status handlers"
git push origin fix/beta-readiness-2026-07-20
```

### Task 3: Observable sync running state and consistent failures

**Files:**
- Modify: `internal/api/app.go`
- Modify: `internal/api/app_test.go`
- Modify generated bindings only through the repository generation/build flow if the DTO schema changes.

**Interfaces:**
- Produces: `SyncStatusDTO.Syncing bool` serialized as `syncing`.
- Consumes: existing `syncRunMu`, `syncNow()`, `updateSyncError`, and `updateSyncSuccess`.

- [ ] **Step 1: Run Go LSP diagnostics and record baseline**

Run the configured Go language-server diagnostics for `internal/api/app.go` and `internal/api/app_test.go`.

Expected: no task-related diagnostics.

- [ ] **Step 2: Add failing sync-state tests**

Block a fake sync request, call `syncStatus()` during the run, and assert `Syncing == true`; after completion assert false. Add an early failure case and assert `LastError` is retained while `LastSyncAt` is unchanged.

- [ ] **Step 3: Run focused tests and confirm failure**

Run: `go test ./internal/api -run 'TestSync.*(Running|FailureStatus)' -count=1`

Expected: FAIL because the DTO has no running field and early failures are not consistently recorded.

- [ ] **Step 4: Implement atomic running state and one error boundary**

Set an `atomic.Bool` immediately after acquiring the run lock, clear it with `defer`, include it in `syncStatus()`, and ensure `PluginSyncNow`/scheduled callers persist any returned failure exactly once without advancing success time.

- [ ] **Step 5: Run focused tests and post-change LSP diagnostics**

Run: `go test ./internal/api -run 'TestSync|TestPluginSync' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit and push**

```bash
git add internal/api/app.go internal/api/app_test.go frontend/wailsjs/go/api/App.js frontend/wailsjs/go/api/App.d.ts
git commit -m "fix(sync): expose truthful running and failure state"
git push origin fix/beta-readiness-2026-07-20
```

Omit generated files from `git add` if the repository generator produces no DTO binding changes.

### Task 4: Read-only Deal-list plugin API

**Files:**
- Modify: `internal/api/app.go`
- Modify: `internal/api/app_test.go`
- Modify: `frontend/src/lib/plugin-host/VerstakPluginAPI.js`
- Modify: `frontend/src/lib/test/wails-mock.js`
- Modify: `frontend/tests/plugin-api-files-test.mjs`

**Interfaces:**
- Produces: `PluginListWorkspaces(pluginID string) ([]PluginWorkspaceDTO, string)`.
- Produces: `api.workspaces.list(): Promise<Array<{id:string,name:string,rootPath:string}>>`.
- Consumes: UUID tree snapshot and existing `files.read` permission.

- [ ] **Step 1: Run Go/TypeScript LSP diagnostics and add failing tests**

Create a tree containing folders, top-level workspaces, and a nested workspace. Assert the backend result recursively includes both workspaces and excludes folders. In the JS contract test call `api.workspaces.list()` and verify plugin ID forwarding/error unpacking.

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `go test ./internal/api -run TestPluginListWorkspaces -count=1`

Run: `node frontend/tests/plugin-api-files-test.mjs`

Expected: FAIL because the API does not exist.

- [ ] **Step 3: Implement the guarded backend DTO and traversal**

Add a small recursive helper over `workspacetree.TreeNode`; append only `Kind == "workspace"` with `ID`, `Name`, and `Path`. Reject callers without `files.read`.

- [ ] **Step 4: Add the frontend namespace and mock**

Expose only `list()`. Use `callBackend` and return an empty array only for a valid empty result, not permission errors. Update the Wails mock with nested workspace fixtures.

- [ ] **Step 5: Run focused/full tests and LSP diagnostics**

Run: `go test ./internal/api -run 'TestPluginListWorkspaces|TestPluginPermission' -count=1`

Run: `node frontend/tests/plugin-api-files-test.mjs`

Expected: PASS.

- [ ] **Step 6: Commit and push**

```bash
git add internal/api/app.go internal/api/app_test.go frontend/src/lib/plugin-host/VerstakPluginAPI.js frontend/src/lib/test/wails-mock.js frontend/tests/plugin-api-files-test.mjs
git commit -m "feat(plugins): expose read-only Deal listing"
git push origin fix/beta-readiness-2026-07-20
```

### Task 5: Progressive truthful global search

**Files:**
- Modify: `frontend/src/lib/shell/GlobalSearch.svelte`
- Create or modify: `frontend/e2e/global-search-results.spec.js`
- Modify: `frontend/e2e/ux-followup.spec.js`
- Modify: `frontend/src/lib/test/wails-mock.js`

**Interfaces:**
- Produces: index state `{entries, contentReady, building, partial, revision}`.
- Consumes: vault file APIs, tree/file refresh signals, and existing result-opening routes.

- [ ] **Step 1: Add failing progressive-index tests**

Seed `ddd/333/kkk/Files/test.txt`, delay text reads, focus search, and type `test`. Assert the filename result appears before content indexing completes and the popup does not say `No results`. Add a file after initial indexing, dispatch the refresh signal, and assert it becomes searchable.

- [ ] **Step 2: Run focused tests and confirm failure**

Run: `cd frontend && npx playwright test e2e/global-search-results.spec.js e2e/ux-followup.spec.js --config playwright.config.js`

Expected: FAIL because the current index is all-or-nothing and stale.

- [ ] **Step 3: Split traversal from content enrichment**

Use breadth-first directory traversal with a defensive cap. Publish path entries immediately, then read supported text content and publish a newer revision. Mark traversal/read failures and cap exhaustion as `partial`.

- [ ] **Step 4: Make UI states truthful and sequence-safe**

Show indexing text while `building`; reserve empty text for a completed relevant revision. Re-run the current query after every published revision. Discard results from obsolete build sequence numbers.

- [ ] **Step 5: Refresh on lifecycle signals**

Rebuild for vault open, workspace-tree/file changes, and focus when the cached revision is stale. Remove listeners on component destroy.

- [ ] **Step 6: Run search and navigation tests**

Run: `cd frontend && npx playwright test e2e/global-search-results.spec.js e2e/ux-followup.spec.js e2e/global-search-layout.spec.js --config playwright.config.js`

Expected: PASS for nested filename/path/content, keyboard-layout fallback, partial label, and result navigation.

- [ ] **Step 7: Commit and push**

```bash
git add frontend/src/lib/shell/GlobalSearch.svelte frontend/e2e/global-search-results.spec.js frontend/e2e/ux-followup.spec.js frontend/src/lib/test/wails-mock.js
git commit -m "fix(search): publish progressive current results"
git push origin fix/beta-readiness-2026-07-20
```

### Task 6: Desktop integration and release packages

**Files:**
- Generated only: `build/`, `release/`

**Interfaces:**
- Consumes: built official plugin packages from the sibling official-plugins worktree.
- Produces: local Debian, AppImage, and Windows portable release artifacts plus checksums; publishes nothing.

- [ ] **Step 1: Run repository verification**

Run: `./scripts/check.sh && ./scripts/test.sh`

Expected: PASS.

Run: `cd frontend && npm run build && npm run test:e2e`

Expected: PASS.

- [ ] **Step 2: Run platform and real-sync smoke tests**

Run: `./scripts/smoke-platform.sh`

Expected: PASS.

Run the documented `./scripts/smoke-real-sync.sh` with its temporary local server harness.

Expected: PASS without using the user's live vault or credentials.

- [ ] **Step 3: Build local release packages**

Run: `./scripts/release.sh 0.1.0-beta.20260720`

Expected non-empty artifacts:

- `release/verstak_0.1.0-beta.20260720_amd64.deb`;
- `release/verstak-linux-x86_64-0.1.0-beta.20260720.AppImage`;
- `release/verstak-windows-amd64-0.1.0-beta.20260720.zip`;
- `release/SHA256SUMS`.

- [ ] **Step 4: Verify artifacts**

Run: `cd release && sha256sum -c SHA256SUMS`

Run: `dpkg-deb --info verstak_0.1.0-beta.20260720_amd64.deb >/dev/null`

Run: `unzip -t verstak-windows-amd64-0.1.0-beta.20260720.zip`

Run the AppImage extraction/version smoke supported by `scripts/package-appimage.sh` documentation.

Expected: all checks pass. Do not run `scripts/publish-github-release.sh`.
