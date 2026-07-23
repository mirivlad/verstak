# Real-World Usage Regressions — Stage 1 Plan

> **Execution rule:** complete and verify Stage 1, publish and audit the release, then create a fresh `refactor/design-system` branch from the published release for Stage 2. Do not mix design-system work into this branch.

**Goal:** Repair the seven reported desktop/plugin regressions with shared, testable contracts and publish the next beta release with complete artifacts.

**Architecture:** Reuse the existing semantic Deal API for plugin-scoped Deal discovery. Extend the existing workspace lifecycle with an explicit selected-tool set. Put binary-safe copy and protected-path policy in core Files. Escape sidebar clipping through one body-level overlay host. Persist sidebar UI state in app settings and tree order by stable UUIDs. Keep plugin-specific UI in official plugins and generic safety/lifecycle behavior in desktop core.

**Repositories and branches**

- `verstak-desktop`: `fix/real-world-usage-regressions`
- `verstak-official-plugins`: `fix/real-world-usage-regressions`
- `verstak-docs`: `fix/real-world-usage-regressions`
- `verstak-sdk`: compatibility verification only unless a public type must change

## Success criteria

- Todo lists only nested semantic Deals whose metadata enables `verstak.todo`, and reflects disable/re-enable without restart.
- Create Deal exposes independently selectable eligible user tools for every template; template changes only reset the initial selection; cancel writes nothing; created metadata has the exact chosen set.
- Default Editor wraps long lines by default, persists the RU/EN toggle, and writes the original text/newlines unchanged.
- Files copies and moves files or folders across Deals through core APIs; Deal drops target the existing `Files` root; protected roots and `.verstak` are rejected; conflicts and missing Files support are visible and non-destructive.
- Context menus are portaled outside the clipped sidebar, clamped to the viewport, and close correctly.
- Sidebar width is bounded, persisted, responsive, and resettable without breaking menus or DnD.
- Tree DnD shows inside/before/after/root/invalid targets, expands on stable-ID hover, autoscrolls, persists reorder by stable IDs, and rejects self/descendant moves.
- All repository checks and focused E2E pass; real Wails GUI evidence is captured when the environment permits it.
- A new beta tag, release notes, `.deb`, AppImage, Windows ZIP, official-plugin archives, and `SHA256SUMS` are published and read back from GitHub.

## Task 1: semantic Deal discovery for Todo

**Files**

- Modify `verstak-official-plugins/plugins/todo/frontend/src/index.js`
- Modify `verstak-official-plugins/scripts/smoke-todo-plugin.js`
- Modify `verstak-desktop/internal/api/app_test.go`
- Modify `verstak-desktop/frontend/src/lib/test/wails-mock.js`
- Modify/add focused Todo Playwright coverage under `verstak-desktop/frontend/e2e/`

- [ ] Add a failing plugin smoke test whose Files tree contains an ordinary folder, while `api.workspaces.list()` returns one nested Todo-enabled Deal.
- [ ] Run the focused smoke and confirm it fails because Todo calls `api.files.list('')`.
- [ ] Switch Todo to `api.workspaces.list()` and use semantic `rootPath`.
- [ ] Extend backend/mock tests for ordinary folders, non-Todo Deals, nested Todo Deals, and disable/re-enable metadata changes.
- [ ] Run the focused plugin smoke, API Go test, bridge test, and Todo E2E.

## Task 2: exact tool selection in Create Deal

**Files**

- Modify `verstak-desktop/internal/core/workspacetree/lifecycle.go`
- Modify `verstak-desktop/internal/core/workspacetree/lifecycle_test.go`
- Modify `verstak-desktop/internal/api/app.go` and `app_test.go`
- Modify `verstak-desktop/frontend/src/lib/shell/WorkspaceTree.svelte`
- Modify RU/EN catalogs, Wails bindings, mock, and `frontend/e2e/workspace-templates.spec.js`

- [ ] Add failing lifecycle/API tests for a valid explicit tool set, exact metadata, required canonical folders, invalid/unavailable IDs, and no partial creation.
- [ ] Add failing E2E for template defaults, selected/unselected styling, independent toggles, Custom, exact created set, and cancel.
- [ ] Add one selectable `custom` template with an empty initial tool set.
- [ ] Add a backward-compatible explicit-tools creation method; validate eligible installed workspace contributions at the API boundary; keep legacy creation unchanged.
- [ ] Render eligible tool controls separately from availability badges and submit the exact set.
- [ ] Regenerate Wails bindings and run focused Go/frontend/E2E checks.

## Task 3: persistent soft wrap

**Files**

- Modify `verstak-official-plugins/plugins/default-editor/plugin.json`
- Modify its RU/EN locales and `frontend/src/index.js`
- Modify `verstak-official-plugins/scripts/smoke-default-editor-plugin.js`
- Modify `verstak-desktop/frontend/e2e/default-editor.spec.js`

- [ ] Add a failing smoke test for default-on wrap, persisted off state, translated label, and byte-equivalent saved text.
- [ ] Add `storage.namespace`, load default `true`, persist the boolean, and toggle only textarea presentation.
- [ ] Add focused desktop E2E and run both checks.

## Task 4: safe cross-Deal Files operations

**Files**

- Modify `verstak-desktop/internal/core/files/{types.go,service.go,service_test.go}`
- Modify `verstak-desktop/internal/api/{app.go,app_test.go}`
- Modify plugin bridge/bindings/mock/tests in `verstak-desktop/frontend/`
- Modify `verstak-official-plugins/plugins/files/frontend/src/index.js`
- Modify Files smoke and Playwright specs

- [ ] Add failing core tests for binary file and recursive folder copy, collision refusal, symlink/protected semantic root/`.verstak` rejection, missing destination parent, and no partial destination.
- [ ] Add a binary-safe `CopyVaultPath` beside existing move; strengthen move source protection; publish source and destination refresh events.
- [ ] Expose `api.files.copy` through the shared plugin bridge and tests.
- [ ] Replace the cross-workspace clipboard prohibition with copy/move operations; use existing destination folders only and deterministic conflict messages.
- [ ] Support file payload drops onto eligible Deal rows, mapping them to `<deal>/Files`; reject Deals without Files without creating anything.
- [ ] Run core/API/bridge/plugin/E2E checks.

## Task 5: overlay and sidebar behavior

**Files**

- Add `verstak-desktop/frontend/src/lib/ui/OverlayHost.svelte`
- Modify `WorkspaceTree.svelte`, `Sidebar.svelte`, app settings Go files/tests, RU/EN catalogs, mock, and focused E2E

- [ ] Add failing E2E for a context menu beside the sidebar edge, viewport bounds, outside/Escape/resize close, and divider overlap.
- [ ] Portal the menu to `document.body`, use a shared overlay layer, measure and clamp after render.
- [ ] Add failing Go/E2E tests for default/min/max/persisted sidebar width and responsive window clamping.
- [ ] Persist width in app settings; add a pointer divider and double-click reset; hide it in narrow responsive layout.
- [ ] Run LSP diagnostics before/after and focused Go/Svelte/E2E checks.

## Task 6: unified stable-ID tree DnD

**Files**

- Modify/add order storage and tests under `verstak-desktop/internal/core/workspacetree/`
- Modify `internal/api/app.go` and tests
- Modify `frontend/src/lib/shell/{WorkspaceTree.svelte,TreeNode.svelte}`
- Modify mock, catalogs, and workspace-tree Playwright specs

- [ ] Add failing Go tests for root/inside/before/after reorder persistence, reconciliation of new/missing IDs, and self/descendant rejection.
- [ ] Store only stable UUID keys in a vault-local tree-order file and apply it after semantic tree construction.
- [ ] Add one placement API that composes physical parent moves with order updates atomically enough to avoid stale in-memory state.
- [ ] Add failing E2E for indicators, 700 ms hover expand/cancel/neighbor identity, free child area, autoscroll, root/reorder/cancel, and invalid descendants.
- [ ] Implement pointer thirds, one keyed hover timer, scroll-edge animation, and complete cleanup on leave/drop/cancel.
- [ ] Run LSP references/definitions and diagnostics before/after, then focused Go/Svelte/E2E checks.

## Task 7: documentation, compatibility, and release

**Files**

- Update relevant product/plugin docs and release notes in `verstak-docs`
- Update desktop/plugin changelogs, version files, and release notes according to existing scripts

- [ ] Run SDK build/tests and confirm no accidental public-contract break.
- [ ] Run official-plugin build/tests, desktop Go/frontend tests, Playwright, and diff-check.
- [ ] Run the real Wails GUI against a disposable test vault and store screenshots/logs in `/tmp/verstak-release-validation/`; if unavailable, report exactly `GUI behavior was not verified`.
- [ ] Choose the next version from repository policy, update versions/notes, and build `.deb`, AppImage, Windows ZIP, official-plugin packages, and `SHA256SUMS`.
- [ ] Commit each repository intentionally, push branches, integrate to release branches/main as policy requires, tag, and publish the GitHub Release.
- [ ] Read back commits, tag, release metadata, every asset name/size/digest, and checksums; report the Stage 1 audit.
- [ ] Only after the release audit passes, branch Stage 2 as `refactor/design-system` from the released commits and write its separate implementation plan.
