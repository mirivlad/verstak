# Synchronized Workspace-Tree Order Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan.

**Goal:** Add backend-authoritative stable-ID workspace-tree placement with vault-scoped sibling order that synchronizes deterministically between devices, plus complete precision drag-and-drop behavior in the sidebar.

**Architecture:** The workspace-tree core owns placement, filesystem movement, validation, canonical order persistence, and ordered tree construction. One versioned document at `.verstak/workspace-tree/order.json` is read by an exact-path sync adapter and transmitted as the dedicated complete-state entity `workspace-tree-order/tree`; ordinary `.verstak` exclusion remains unchanged. The Svelte tree derives drop intent from row geometry but sends only `{sourceKey,targetKey,position}` to the backend.

**Tech Stack:** Go 1.25, Svelte 4, Wails v2 bindings, Vitest/Vite checks already present in the repository, Playwright E2E.

## Global Constraints

- Use stable keys only: `folder:<uuid>` and `workspace:<uuid>`.
- Preserve the broad `.verstak` file-sync exclusion.
- Keep sidebar width, scroll, expansion, focus, current selection, and drag state device-local.
- Make the smallest changes in the listed files; do not refactor unrelated tree or sync code.
- Write the focused failing test before each production change.
- Run Go and TypeScript/Svelte diagnostics before and after non-trivial edits.
- Keep the existing `MoveFolderV2` and `MoveWorkspaceV2` APIs for compatibility; new DnD uses the placement API.

---

## Task 1: Versioned vault order document and deterministic tree projection

**Files:**

- Create: `internal/core/workspacetree/order.go`
- Create: `internal/core/workspacetree/order_test.go`
- Modify: `internal/core/workspacetree/index.go`
- Modify: `internal/core/workspacetree/service.go`
- Modify: `internal/core/workspacetree/index_test.go`

- [ ] **Step 1: Write failing order-document tests**

Cover:

- missing file returns an empty version-1 state;
- exact path is `.verstak/workspace-tree/order.json`;
- atomic write/read round trip preserves sibling arrays;
- stable JSON output sorts parent keys;
- malformed version, parent UUID, stable key, and duplicate node key are rejected;
- syntactically valid missing IDs remain readable.

Define the minimal public data types:

```go
const OrderVersion = 1

type OrderState struct {
    Version  int                 `json:"version"`
    Children map[string][]string `json:"children"`
}

func OrderMetadataPath(vaultDir string) string
func ReadOrderState(vaultDir string) (OrderState, error)
func WriteOrderState(vaultDir string, state OrderState) error
func ParseOrderState(data []byte) (OrderState, error)
func MarshalOrderState(state OrderState) ([]byte, error)
```

- [ ] **Step 2: Run the focused tests and confirm RED**

Run:

```bash
go test ./internal/core/workspacetree -run 'Test(OrderState|OrderMetadata)' -count=1
```

Expected: compile or assertion failure because the order API is not implemented.

- [ ] **Step 3: Implement the smallest parser and atomic writer**

Requirements:

- validate UUIDs with `google/uuid`;
- validate `folder:<uuid>` / `workspace:<uuid>`;
- allow `root` only as the special parent key;
- create `.verstak/workspace-tree` with private permissions;
- write a same-directory temporary file, `Sync`, close, chmod `0600`, rename;
- never mutate generic application settings.

- [ ] **Step 4: Write failing ordered-builder tests**

Add cases for:

- mixed folder/Deal root order;
- nested order;
- stale keys ignored in rendering;
- keys stored under the wrong actual parent ignored there;
- new IDs appended folders-first and case-insensitively by name;
- stable-key tie-breaker for equal folded names.

The builder entry point becomes:

```go
func BuildTree(scan *ScanResult, currentWorkspaceID string, revision uint64, order OrderState) *TreeSnapshot
```

- [ ] **Step 5: Implement deterministic projection and service loading**

Read the order once per `fullReconcile`. If invalid, append a diagnostic and use
an empty order. Apply stored keys only to actual children of that parent, then
append unlisted children in fallback order.

- [ ] **Step 6: Verify Task 1**

Run:

```bash
gofmt -w internal/core/workspacetree/order.go internal/core/workspacetree/order_test.go internal/core/workspacetree/index.go internal/core/workspacetree/index_test.go internal/core/workspacetree/service.go
go test ./internal/core/workspacetree -count=1
git diff --check
```

---

## Task 2: Backend-authoritative placement

**Files:**

- Create: `internal/core/workspacetree/placement.go`
- Create: `internal/core/workspacetree/placement_test.go`
- Modify: `internal/core/workspacetree/lifecycle.go`
- Modify: `internal/core/workspacetree/service.go`

- [ ] **Step 1: Write failing placement tests**

Define:

```go
type PlacementRequest struct {
    SourceKey string `json:"sourceKey"`
    TargetKey string `json:"targetKey"`
    Position  string `json:"position"`
}

func (s *Service) PlaceNode(
    request PlacementRequest,
    refreshBaseline func() error,
) (OrderState, error)
```

Test:

- `before` and `after` within one parent;
- `inside` a folder;
- `root` from a nested parent;
- mixed-kind order;
- restart persistence;
- order-only placement leaves filesystem paths unchanged;
- parent-changing placement updates the filesystem;
- missing/malformed source or target;
- unsupported position;
- self target;
- `inside` a Deal;
- folder into descendant;
- destination path conflict;
- write failure reports an error rather than success.

- [ ] **Step 2: Run focused tests and confirm RED**

```bash
go test ./internal/core/workspacetree -run 'TestPlaceNode' -count=1
```

- [ ] **Step 3: Implement stable-key resolution and validation**

Resolve source, target, current parent, and destination parent only from the
current backend scan/tree. Do not accept a parent or array index from the caller.
Validate the whole request and destination path before mutation.

- [ ] **Step 4: Reuse guarded lifecycle movement**

Extract only the smallest private move helpers needed so placement and legacy
move APIs share:

- folder self/descendant protection;
- target-parent resolution;
- path collision checks;
- watcher suppression and baseline refresh.

Avoid changing legacy API behavior.

- [ ] **Step 5: Canonicalize and persist placement**

After any required physical move:

1. reconcile the scan;
2. derive actual sibling lists;
3. remove the source key from every list;
4. insert it exactly once at the requested location;
5. preserve syntactically valid unresolved keys outside affected current lists;
6. atomically write the document;
7. reconcile again so the returned tree reflects stored order.

- [ ] **Step 6: Verify Task 2**

```bash
gofmt -w internal/core/workspacetree/placement.go internal/core/workspacetree/placement_test.go internal/core/workspacetree/lifecycle.go internal/core/workspacetree/service.go
go test ./internal/core/workspacetree -count=1
git diff --check
```

---

## Task 3: Narrow sync adapter for order metadata

**Files:**

- Modify: `internal/core/sync/service.go`
- Modify: `internal/core/sync/snapshot.go`
- Modify: `internal/core/sync/snapshot_test.go`
- Modify: `internal/core/sync/sync_tree_test.go`
- Modify: `internal/core/sync/sync_integration_test.go`

- [ ] **Step 1: Write failing exact-path scan tests**

Add snapshot state:

```go
type Snapshot struct {
    // existing fields...
    TreeOrder            json.RawMessage `json:"treeOrder,omitempty"`
    TreeOrderInitialized bool            `json:"treeOrderInitialized,omitempty"`
}
```

Test that:

- valid `.verstak/workspace-tree/order.json` is captured;
- unrelated `.verstak` content is still excluded;
- an absent order file is represented deterministically;
- invalid order JSON produces a warning/unresolved condition without creating a
  normal file operation.

- [ ] **Step 2: Run focused sync tests and confirm RED**

```bash
go test ./internal/core/sync -run 'Test.*TreeOrder' -count=1
```

- [ ] **Step 3: Add the dedicated entity**

Add:

```go
const EntityWorkspaceTreeOrder = "workspace-tree-order"
const WorkspaceTreeOrderEntityID = "tree"
```

Read only the exact metadata path after the ordinary filesystem walk. Do not
change `excludedFromSync`.

- [ ] **Step 4: Write failing diff/bootstrap tests**

Test:

- no operation for the initial baseline;
- bootstrap emits one complete-state `update`;
- subsequent change emits one `update`;
- no normal `file` entity for the metadata path;
- semantic folder/Deal move operations precede the order operation;
- no operation for byte-only formatting changes after canonical parsing, if
  canonical payloads are equal.

- [ ] **Step 5: Implement snapshot diff and ordering**

Emit the complete validated order document as
`workspace-tree-order/tree/update`. Append it after semantic structural
operations and before ordinary file operations. Preserve previous valid snapshot
state when the exact file is temporarily invalid, and surface a warning.

- [ ] **Step 6: Verify Task 3**

```bash
gofmt -w internal/core/sync/service.go internal/core/sync/snapshot.go internal/core/sync/snapshot_test.go internal/core/sync/sync_tree_test.go internal/core/sync/sync_integration_test.go
go test ./internal/core/sync -count=1
git diff --check
```

---

## Task 4: Wails placement API and remote application

**Files:**

- Modify: `internal/api/app.go`
- Modify: `internal/api/app_test.go`
- Modify: `internal/api/sync_tree_test.go` if present, otherwise add focused cases
  to the nearest existing sync API test file
- Modify: `frontend/wailsjs/go/api/App.js`
- Modify: `frontend/wailsjs/go/api/App.d.ts`
- Modify: `frontend/wailsjs/go/models.ts`
- Modify: `frontend/tests/wails-bindings-test.mjs`
- Modify: `frontend/src/lib/test/wails-mock.js`

- [ ] **Step 1: Write failing API tests**

Add:

```go
func (a *App) PlaceWorkspaceTreeNodeV2(
    request workspacetree.PlacementRequest,
) string
```

Test:

- request is delegated and errors are returned;
- successful placement refreshes the watcher baseline;
- a local scan records structural movement before the order update;
- `syncOperationPath` identifies the tree-order entity without parsing it as a
  file;
- a valid remote complete-state payload is validated, written, and reconciled;
- a malformed remote payload changes neither filesystem nor current order;
- applying device A's operation to device B yields identical stable-key trees;
- rebase prevents echo.

- [ ] **Step 2: Run focused API tests and confirm RED**

```bash
go test ./internal/api -run 'Test(PlaceWorkspaceTreeNodeV2|ApplyRemote.*TreeOrder|SyncOperationPath.*TreeOrder)' -count=1
```

- [ ] **Step 3: Implement local and remote API paths**

Local placement:

1. call `treeV2.PlaceNode`;
2. perform the focused sync scan so structure and order enter one ordered diff;
3. return a backend error string on any failure.

Remote application:

1. branch on `EntityWorkspaceTreeOrder` before the Files service guard;
2. parse and validate with the workspace-tree order parser;
3. atomically write through the workspace-tree service;
4. reconcile and emit the existing tree-changed signal;
5. rely on the existing pull rebase to accept the new snapshot.

- [ ] **Step 4: Update generated-surface fixtures**

Regenerate Wails bindings if the repository command is available; otherwise
make the same minimal generated-form edits used by existing bindings. Add the
mock method and a binding contract assertion.

- [ ] **Step 5: Verify Task 4**

```bash
gofmt -w internal/api/app.go internal/api/app_test.go
go test ./internal/api ./internal/core/sync ./internal/core/workspacetree -count=1
node frontend/tests/wails-bindings-test.mjs
git diff --check
```

---

## Task 5: Stable-key precision DnD controller

**Files:**

- Modify: `frontend/src/lib/shell/WorkspaceTree.svelte`
- Modify: `frontend/src/lib/shell/TreeNode.svelte`
- Create: `frontend/e2e/workspace-tree-dnd.spec.js`
- Modify: `frontend/src/lib/test/wails-mock.js`

- [ ] **Step 1: Write failing Playwright coverage**

Use the existing application mock harness. Test:

- top/middle/bottom row thirds map to before/inside/after;
- a Deal middle third maps deterministically to before or after;
- payload and API request use stable keys, never indexes;
- expanded nonempty and empty folder free-list areas send `inside`;
- free root area sends `root`;
- before/after and inside indicators are visually distinct;
- one folder hover timer expands after approximately 700 ms;
- moving to another key cancels the old timer;
- top/bottom edge drag starts autoscroll and leaving stops it;
- drop success and backend rejection both clear every indicator;
- dragend and component teardown clear timers and scroll frames;
- Files payload still drops on Deal rows without calling placement.

- [ ] **Step 2: Run the new E2E file and confirm RED**

```bash
cd frontend
npx playwright test --config playwright.config.js e2e/workspace-tree-dnd.spec.js
```

- [ ] **Step 3: Implement row-third intent in `TreeNode.svelte`**

Dispatch a target descriptor:

```js
{
  sourceKey,
  targetKey: node.key,
  position: 'before' | 'after' | 'inside'
}
```

Calculate thirds from `getBoundingClientRect()`. Render a top/bottom insertion
line or inside highlight from parent-owned active target state. Do not use the
render-loop index as identity.

- [ ] **Step 4: Implement one parent-owned drag controller**

`WorkspaceTree.svelte` owns:

- `dragSourceKey`;
- one active target descriptor;
- one hover timer and hover stable key;
- one animation frame and latest pointer Y;
- one cleanup function.

The hover timer is approximately 700 ms. The edge bands scroll the `.wt-list`
with a bounded speed. All drop, rejection, malformed-payload, dragend, leave,
and destroy paths call cleanup.

- [ ] **Step 5: Add free-list areas**

Render a root free-list target and a child-list target for every expanded folder,
including empty folders. Keep them large enough to acquire during a drag but
visually unobtrusive when idle.

- [ ] **Step 6: Call only the placement API for tree nodes**

On a valid tree-node drop:

```js
const err = await App.PlaceWorkspaceTreeNodeV2({
  sourceKey,
  targetKey,
  position,
});
```

Surface backend rejection in the existing tree error area, reload on success,
and always clean up. Keep the existing Files payload branch isolated.

- [ ] **Step 7: Verify Task 5**

```bash
cd frontend
npm run build
node tests/wails-bindings-test.mjs
npx playwright test --config playwright.config.js e2e/workspace-tree-dnd.spec.js
npx playwright test --config playwright.config.js e2e/workspace-tree-overlay.spec.js e2e/files-plugin.spec.js
```

---

## Task 6: Cross-device convergence and Stage 1 regression gate

**Files:**

- Modify only tests found insufficient during the gate.

- [ ] **Step 1: Run focused Go race-free package tests**

```bash
go test ./internal/core/workspacetree ./internal/core/sync ./internal/api -count=1
```

- [ ] **Step 2: Run a two-device convergence test**

Use two temporary vault roots and distinct device IDs with the existing sync
test client/server harness:

1. create the same semantic folders and Deals;
2. place mixed siblings on device A;
3. push/pull;
4. assert device B's recursive stable-key tree equals A's;
5. add a new ID on B and verify deterministic append;
6. remove an ID and verify the stale metadata does not perturb visible order;
7. place on B, sync back, and assert convergence again.

- [ ] **Step 3: Run diagnostics**

Run Go LSP diagnostics for modified packages and TypeScript/Svelte diagnostics
for both tree components. Resolve new errors; record pre-existing warnings
separately.

- [ ] **Step 4: Run the Stage 1 repository gate**

```bash
go test ./...
cd frontend
npm run build
node tests/wails-bindings-test.mjs
npx playwright test --config playwright.config.js
```

Also run the official-plugin checks from the sibling repository as required by
the parent Stage 1 plan.

- [ ] **Step 5: Inspect the final diff**

```bash
git status --short
git diff --check
git diff --stat
```

Confirm:

- no generic settings field was added for order;
- `excludedFromSync` still excludes all `.verstak` paths;
- only the exact order metadata path has a dedicated sync adapter;
- no array index participates in DnD identity;
- all transient state remains local.

- [ ] **Step 6: Commit the feature**

```bash
git add internal/core/workspacetree internal/core/sync internal/api frontend/src/lib/shell frontend/src/lib/test/wails-mock.js frontend/wailsjs frontend/tests/wails-bindings-test.mjs frontend/e2e/workspace-tree-dnd.spec.js docs/superpowers/specs/2026-07-23-synchronized-tree-order-design.md docs/superpowers/plans/2026-07-23-synchronized-tree-order.md
git commit -m "feat(tree): synchronize stable sibling order"
```

After this commit, return to the parent Stage 1 plan for changelog/versioning,
release publication, clean-device verification, and only then begin the approved
Stage 2 design-system work.
