# Milestone 6d — Minimal Files Plugin

## Goal

Create a minimal Files plugin that shows vault files/folders and opens files
through Workbench openResource. No editor embedded.

## What was built

### Plugin: `verstak.files`

Location: `verstak-official-plugins/plugins/files/`

**Contributions:**
- views: `verstak.files.view` → `FilesView` component
- No sidebarItems — Files is not a global sidebar item

**Permissions:** `files.read`, `files.write`, `workbench.open`, `ui.register`

### Files View

- Root listing on mount
- Folder navigation (double-click)
- File open via `api.workbench.openResource()`
- Breadcrumb navigation
- Create folder/file buttons
- Refresh button
- Loading/error/empty states
- `.verstak` filtered out

### Provider priority

- default-editor: priority 50
- platform-test diagnostic: priority 10
- default-editor wins for normal file opens

### Bundle fix

Fixed missing opening quote in STYLES string (`.files-empty` → `'.files-empty'`).
Added automated bundle execution check to `scripts/check.sh`.

## 6d-hotfix

- Removed sidebarItems from Files plugin (Files is not a global sidebar item)
- Added `[frontend bundle execution]` check to `check.sh` — verifies all plugin
  bundles parse via `new Function()` and register via `VerstakPluginRegister`
- Updated E2E tests: Files no longer expected in global sidebar
- Documented: sidebarItems are global shell navigation, not workspace template tabs

## Verification

- `go test ./...` — PASS
- `go vet ./...` — PASS
- `npm run build` — PASS
- `npm run test:e2e` — 34/34 PASS
- Official plugins — 3 plugins built, bundle execution check passes
- SDK — 11/11 tests pass

## Deferred

- Notes plugin, rename/move/trash UI, drag-and-drop, context menu,
  watcher/inotify, sync, external open, binary streaming, sidecar/security,
  workspace template host (Milestone 6d2)

