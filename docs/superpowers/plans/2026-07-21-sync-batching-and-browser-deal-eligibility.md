# Sync Batching and Browser Deal Eligibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Synchronize queues larger than the server push limit, preserve a safe paired-credential presentation, and restrict Browser Inbox assignment to Deals where its workspace tool is active.

**Architecture:** The existing desktop sync transport will push ordered adaptive batches and persist progress after every accepted batch. The guarded Deal-list API will filter UUID-keyed workspace metadata by the calling plugin ID. The Sync plugin will keep username as ordinary settings while representing the stored device token with an internal-empty password field and a masked placeholder.

**Tech Stack:** Go 1.24, net/http test servers, Svelte 4, JavaScript smoke tests, Wails plugin bridge.

## Global Constraints

- Synchronization scheduling, settings, labels, and interaction remain owned by `verstak.sync`.
- The server protocol and `max_push_operations` configuration remain unchanged.
- Password bytes are never persisted, restored, or replaced with a fake form value.
- Browser Inbox receives only read-only workspace eligibility data.
- Missing or malformed workspace metadata fails closed.
- Every production change follows a failing focused test.

---

### Task 1: Ordered Adaptive Sync Push Batches

**Files:**
- Modify: `internal/api/app.go:3780-3910`
- Test: `internal/api/app_test.go`

**Interfaces:**
- Consumes: `(*sync.Service).GetUnpushedOps()`, `(*sync.Client).Push([]sync.Op)`, `(*sync.Service).MarkPushed([]string)`, and `*sync.ServerError`.
- Produces: `func (a *App) pushPendingOps(client *sync.Client, ops []sync.Op) (*sync.PushResponse, error)`.

- [ ] **Step 1: Add a failing 103-operation batching test**

Add a test server that rejects request bodies containing more than 100
operations with:

```go
w.WriteHeader(http.StatusRequestEntityTooLarge)
_ = json.NewEncoder(w).Encode(map[string]string{
    "code": "too_many_operations",
    "error": "too many operations in one push",
})
```

Record 103 operations through `app.syncSvc.RecordOp`, run `app.syncNow()`,
and assert push request sizes are `[]int{100, 3}`, result `pushed == 103`,
and `GetUnpushedOps()` is empty.

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
go test ./internal/api -run TestSyncNowBatchesMoreThanServerPushLimit -count=1
```

Expected: FAIL because the first request contains 103 operations and
`syncNow` returns `too_many_operations`.

- [ ] **Step 3: Add failing adaptive-limit and partial-progress tests**

Add:

- `TestSyncNowReducesRejectedPushBatch`: server limit 25, expected request
  sizes begin `100, 50, 25` and all 103 operations eventually succeed in
  order.
- `TestSyncNowPreservesAcceptedBatchWhenLaterBatchFails`: accept the first
  100 and return HTTP 500 for the final 3; assert exactly 3 remain unpushed and
  no successful timestamp is written.
- `TestSyncNowDoesNotSplitPayloadError`: return HTTP 413 with code
  `payload_too_large`; assert one request only.

- [ ] **Step 4: Run the additional tests and verify RED**

Run:

```bash
go test ./internal/api -run 'TestSyncNow(Batches|Reduces|Preserves|DoesNotSplit)' -count=1
```

Expected: the new tests fail because no batching helper exists and accepted
partial work is not marked before a later request.

- [ ] **Step 5: Implement the minimal batching helper**

Add a `defaultSyncPushBatchSize = 100` constant and implement
`pushPendingOps` so it:

```go
result := &syncsvc.PushResponse{}
batchSize := min(defaultSyncPushBatchSize, len(ops))
for offset := 0; offset < len(ops); {
    end := min(offset+batchSize, len(ops))
    batchResult, err := client.Push(ops[offset:end])
    if isTooManyOperations(err) && end-offset > 1 {
        batchSize = max(1, (end-offset)/2)
        continue
    }
    if err != nil {
        return result, err
    }
    if err := a.syncSvc.MarkPushed(batchResult.Accepted); err != nil {
        return result, fmt.Errorf("mark pushed: %w", err)
    }
    result.Accepted = append(result.Accepted, batchResult.Accepted...)
    result.Conflicts = append(result.Conflicts, batchResult.Conflicts...)
    result.Count += batchResult.Count
    offset = end
    batchSize = min(batchSize, len(ops)-offset)
}
return result, nil
```

`isTooManyOperations` must use `errors.As(err, *syncsvc.ServerError)` and
match only `Code == "too_many_operations"`. Replace the one-shot
`client.Push(unpushed)` and final one-shot `MarkPushed` block in
`syncNow` with this helper.

- [ ] **Step 6: Run focused and package tests**

Run:

```bash
go test ./internal/api -run 'TestSyncNow|TestPluginSync' -count=1
go test ./internal/core/sync ./internal/api
```

Expected: PASS.

- [ ] **Step 7: Commit and push the batching fix**

```bash
git add internal/api/app.go internal/api/app_test.go
git commit -m "fix(sync): batch pending operations within server limits"
git push
```

---

### Task 2: Filter the Deal API by Active Workspace Tool

**Files:**
- Modify: `internal/api/app.go:2856-2883`
- Test: `internal/api/app_test.go:TestPluginListWorkspacesReturnsNestedDealsOnly`
- Generated after implementation: `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Consumes: tree node IDs and
  `<vault>/.verstak/workspaces/uuid-<workspace-id>.json`.
- Produces: `func workspaceHasTool(vaultPath, workspaceID, pluginID string) bool`;
  `PluginListWorkspaces(pluginID)` returns only eligible semantic Deals.

- [ ] **Step 1: Extend the existing Deal-list test and verify RED**

Write UUID metadata for:

- nested Deal with `workspaceTools: ["files.plugin"]`;
- top-level Deal with `workspaceTools: ["another.plugin"]`;
- Deal with missing metadata;
- Deal with malformed metadata.

Keep the folder marker. Assert only the nested eligible Deal is returned for
`files.plugin`.

Run:

```bash
go test ./internal/api -run TestPluginListWorkspacesReturnsNestedDealsOnly -count=1
```

Expected: FAIL because every semantic Deal is currently returned.

- [ ] **Step 2: Implement fail-closed eligibility**

Add a small metadata struct:

```go
var metadata struct {
    WorkspaceTools []string `json:"workspaceTools"`
}
```

Read only `uuid-<workspaceID>.json` beneath the current vault metadata
directory, reject read/JSON errors, and return true only for an exact
`pluginID` entry. In `PluginListWorkspaces`, append a workspace row only
when `workspaceHasTool(a.vaultPath(), node.ID, pluginID)` is true.

- [ ] **Step 3: Run API and workspace tests**

```bash
go test ./internal/api -run 'TestPluginListWorkspaces|TestWorkspace' -count=1
go test ./internal/core/workspace ./internal/core/workspacetree ./internal/api
```

Expected: PASS.

- [ ] **Step 4: Regenerate/check Wails bindings only if the DTO changes**

The DTO shape is not expected to change. Run `git status --short`; if Wails
generated files changed only in whitespace, restore those exact lines with an
`apply_patch` edit. Do not commit unrelated generated whitespace.

- [ ] **Step 5: Commit and push Deal eligibility**

```bash
git add internal/api/app.go internal/api/app_test.go
git commit -m "fix(plugins): list only Deals active for the caller"
git push
```

---

### Task 3: Safe Paired-Credential Presentation

**Files:**
- Modify: `plugins/sync/frontend/src/SyncSettings.svelte`
- Modify: `plugins/sync/locales/en.json`
- Modify: `plugins/sync/locales/ru.json`
- Test: `scripts/smoke-sync-plugin.js`

**Interfaces:**
- Consumes: `api.sync.status().tokenStored`,
  `api.settings.read/writeAll`, `api.sync.testConnection`, and
  `api.sync.configure`.
- Produces: visible persisted username, internal-empty password with masked
  placeholder, and stored-token connection verification.

- [ ] **Step 1: Add failing source and behavior assertions**

Extend `scripts/smoke-sync-plugin.js` to assert:

- `configureSync` does not contain `username = ''`;
- successful configure persists `username` through
  `api.settings.writeAll`;
- the password input uses a localized saved-password placeholder only when
  `settings.tokenStored && !password`;
- no source assignment sets `password` to bullet characters;
- testing with stored token and empty replacement password calls
  `sync.status()` and does not call `sync.testConnection`;
- entering a replacement password calls `sync.testConnection` with the
  entered value.

- [ ] **Step 2: Run the Sync plugin smoke test and verify RED**

```bash
node scripts/smoke-sync-plugin.js
```

Expected: FAIL because configure clears username and the password has no stored
placeholder behavior.

- [ ] **Step 3: Implement the minimal UI state**

After successful `configure`, keep `username`, clear only the real password,
write `{ serverUrl, vaultId, username, autoSync, syncInterval }`, then reload
status. Derive:

```svelte
$: passwordStored = !!settings?.tokenStored && !password
$: passwordPlaceholder = passwordStored
  ? tr('ui.passwordStoredPlaceholder', null, '••••••••')
  : ''
```

Bind the password input to the real empty `password` value and set
`placeholder={passwordPlaceholder}`.

For `testConnection`, when `passwordStored` is true, refresh status and
require `connected === true`; otherwise call the existing credential test.
Do not submit placeholder text.

- [ ] **Step 4: Add localized explanatory copy**

Add English and Russian keys for the saved password placeholder and a short
hint explaining that entering a new password will reconnect the device.

- [ ] **Step 5: Run plugin tests and build**

```bash
node scripts/smoke-sync-plugin.js
./scripts/check.sh
./scripts/build.sh
```

Expected: PASS with no new Svelte diagnostics.

- [ ] **Step 6: Commit and push credential presentation**

```bash
git add plugins/sync/frontend/src/SyncSettings.svelte plugins/sync/locales/en.json plugins/sync/locales/ru.json scripts/smoke-sync-plugin.js
git commit -m "fix(sync): preserve paired credential presentation"
git push
```

---

### Task 4: Browser Inbox Assignment Contract

**Files:**
- Modify: `scripts/smoke-browser-inbox-plugin.js`
- Verify unchanged behavior: `plugins/browser-inbox/frontend/src/index.js`

**Interfaces:**
- Consumes: eligible entries returned by `api.workspaces.list()`.
- Produces: a regression test proving Browser Inbox renders only host-approved
  Deal options.

- [ ] **Step 1: Change the test workspace fixture**

Represent both active and inactive semantic Deals in the host fixture and
return only active entries from `workspaces.list`. Assert active nested Deals
are present and inactive Deals are absent from every assignment select.

- [ ] **Step 2: Run Browser Inbox smoke**

```bash
node scripts/smoke-browser-inbox-plugin.js
```

Expected: PASS without production Browser Inbox changes because host-side
eligibility is the intended boundary.

- [ ] **Step 3: Run the complete plugin check**

```bash
./scripts/check.sh
```

Expected: PASS.

- [ ] **Step 4: Commit and push the Browser Inbox regression contract**

```bash
git add scripts/smoke-browser-inbox-plugin.js
git commit -m "test(browser-inbox): reject inactive Deal assignments"
git push
```

---

### Task 5: Integration Verification and Replacement Release Packages

**Files:**
- Verify: desktop and official-plugin worktrees
- Generate, do not commit: release artifacts for desktop and official plugins

**Interfaces:**
- Consumes: completed Tasks 1–4.
- Produces: clean pushed branches and locally verified replacement beta
  packages.

- [ ] **Step 1: Run language-server diagnostics**

Run gopls diagnostics for `internal/api/app.go` and Svelte diagnostics for
`plugins/sync/frontend/src/SyncSettings.svelte`. Treat worktree-root false
positives separately from compiler/test output.

- [ ] **Step 2: Run desktop verification**

```bash
./scripts/check.sh
./scripts/test.sh
(cd frontend && npm run build)
./scripts/smoke-real-sync.sh
```

Expected: all commands PASS, including a real sync-server run.

- [ ] **Step 3: Verify the user's 103-operation scenario safely**

Do not modify the user's vault directly with test helpers. Start the configured
server and run the built application/manual sync only if the existing device
token remains valid. Confirm pending operations reach zero, server operations
increase, and `lastSyncAt` becomes non-empty. If interactive runtime execution
is unavailable, report that exact limitation and rely on the 103-operation
integration test.

- [ ] **Step 4: Run official-plugin verification**

```bash
./scripts/check.sh
./scripts/build-windows.sh
```

Expected: PASS.

- [ ] **Step 5: Build replacement releases without publishing**

```bash
./scripts/release.sh 0.1.0-beta.20260721
```

Run the desktop and official-plugin release scripts in their respective
worktrees. Verify `sha256sum -c`, DEB metadata/content, AppImage extraction,
and Windows ZIP integrity. Do not invoke GitHub Actions or upload assets.

- [ ] **Step 6: Confirm repository state**

```bash
git diff --check
git status --short --branch
git rev-list --left-right --count HEAD...'@{upstream}'
```

Expected in both repositories: no worktree changes and `0 0` divergence.
