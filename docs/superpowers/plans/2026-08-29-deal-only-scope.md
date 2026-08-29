# Deal-only Resource Scope Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Deal UUID the only runtime resource scope and deliver one-shot migration, Project Meta, Milestones, Templates CRUD, and multi-repository Git without syncing checkouts or exposing credentials.

**Architecture:** `workspacetree.Service` becomes the UUID-first Deal authority and owns versioned metadata. Official plugins own Deal-scoped records through versioned SDK capabilities; a dormant migration framework is activated only after all target schemas exist. Templates send generic recipe snapshots to Core, while Git keeps descriptors synced and checkouts/credentials device-local.

**Tech Stack:** Go 1.24/Wails v2, Svelte 4 plain JavaScript, TypeScript SDK/Vitest, JSON Schema, Node.js smoke tests, Playwright, system Git CLI.

## Global Constraints

- `DealScope { workspaceId: UUID }` is the only runtime resource scope.
- Paths are mutable addresses, never resource identity.
- No production runtime reads legacy nested Projects after successful migration.
- Preserve Notes/Files bytes and provider-owned record identities; Project metadata is best-effort.
- No migration wizard, permanent compatibility layer, release, or tag.
- Core must not know official plugin IDs or business configuration.
- Use test-first RED/GREEN cycles for production behavior changes.
- End every completed logical slice with focused tests, repository checks, commit, push, and remote SHA verification.

---

## File map

### `verstak-desktop`

- `internal/core/workspacetree/metadata.go`: canonical metadata v2 parsing, validation, UUID-first lookup, and atomic persistence.
- `internal/core/workspacetree/recipe.go`: generic recipe snapshot validation and transactional creation inputs.
- `internal/core/dealmigration/runner.go`: dormant one-shot plan/backup/apply/verify ledger.
- `internal/core/dealmigration/transforms.go`: provider/project migration transformations with no runtime adapters.
- `internal/core/gitservice/service.go`: bounded Git operations and sanitized status/result types.
- `internal/core/secrets/credentials.go`: purpose-bound local credential-slot resolution.
- `internal/api/app.go`: Wails bridge wiring only; business records remain in plugins.
- `internal/core/sync/snapshot.go`: managed-checkout exclusion and snapshot rebase.
- `frontend/src/lib/plugin-host/VerstakPluginAPI.js`: Deal scope, recipe creation, Git broker, and credential-slot bridges.
- `frontend/src/lib/test/wails-mock.js`: exact browser mock of new public contracts.

### `verstak-sdk`

- `src/plugin-api.ts`, `src/types.ts`: DealScope, recipe, Git broker, and typed navigation contracts.
- `schemas/deal-scope.json`, `schemas/capability-operations.json`, `schemas/manifest.json`: enforceable v2 public shapes.
- `src/test-utils.ts`: tree-aware, scope-aware plugin test API.

### `verstak-official-plugins`

- `plugins/projects/`: rewritten as one Project Meta record per Deal plus global portfolio.
- `plugins/milestones/`: independent Deal milestone provider.
- `plugins/templates/`: persisted CRUD, seed defaults, preview, and create-from-recipe.
- `plugins/git/`: repository descriptors and Git UI.
- `plugins/notes/`, `plugins/files/`, `plugins/todo/`, `plugins/activity/`: Deal-only operations and storage.
- `scripts/smoke-*-plugin.js`: focused contract and migration coverage.

---

### Task 1: Canonical versioned Deal metadata

**Files:**
- Create: `verstak-desktop/internal/core/workspacetree/metadata.go`
- Create: `verstak-desktop/internal/core/workspacetree/metadata_v2_test.go`
- Modify: `verstak-desktop/internal/core/workspacetree/lifecycle.go`
- Modify: `verstak-desktop/internal/core/workspacetree/service.go`

**Interfaces:**
- Produces: `DealMetadata`, `TemplateProvenance`, `ReadDealMetadata(workspaceID, rootPath)`, `WriteDealMetadata(metadata)`, `MigrateLegacyDealMetadata(workspaceID, rootPath)`.
- Invariant: UUID metadata wins; legacy path metadata is input only and never shadows v2.

- [ ] **Step 1: Add failing UUID-first and round-trip tests**

```go
func TestReadDealMetadataPrefersCanonicalUUIDRecord(t *testing.T) {
    svc, ws := newMetadataFixture(t)
    writeLegacyMetadata(t, svc.vaultDir, ws.RootPath, `{"workspaceTools":["legacy.tool"]}`)
    writeCanonicalMetadata(t, svc.vaultDir, ws.ID, `{"schemaVersion":2,"workspaceId":"`+ws.ID+`","workspaceName":"Deal","workspaceTools":["canonical.tool"],"toolConfig":{},"updatedAt":"2026-08-29T00:00:00Z"}`)
    got, err := svc.ReadDealMetadata(ws.ID, ws.RootPath)
    require.NoError(t, err)
    require.Equal(t, []string{"canonical.tool"}, got.WorkspaceTools)
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/core/workspacetree -run 'Test(ReadDealMetadata|WriteDealMetadata|MigrateLegacyDealMetadata)' -count=1`

Expected: compile failure because the metadata v2 API does not exist.

- [ ] **Step 3: Implement the metadata API**

```go
const DealMetadataSchemaVersion = 2

type DealMetadata struct {
    SchemaVersion       int                        `json:"schemaVersion"`
    WorkspaceID         string                     `json:"workspaceId"`
    WorkspaceName       string                     `json:"workspaceName"`
    WorkspaceTools      []string                   `json:"workspaceTools"`
    ToolConfig          map[string]json.RawMessage `json:"toolConfig"`
    CreatedFromTemplate *TemplateProvenance        `json:"createdFromTemplate,omitempty"`
    UpdatedAt           time.Time                  `json:"updatedAt"`
}
```

Validate UUID/file identity, preserve unknown tool/config values, normalize duplicate tools, and use temp-file plus rename for writes.

- [ ] **Step 4: Route V2 creation/update through metadata v2**

Remove `writeWorkspaceMetadataV2`'s map-shaped payload and make metadata failure abort Deal publication.

- [ ] **Step 5: Verify GREEN and repository gate**

Run:

```bash
go test ./internal/core/workspacetree -count=1
bash scripts/check.sh
```

Expected: all tests/checks pass.

- [ ] **Step 6: Commit and push**

```bash
git add internal/core/workspacetree
git commit -m "feat: version canonical Deal metadata"
git push origin main
```

---

### Task 2: Consolidate Deal authority on `workspacetree.Service`

**Files:**
- Modify: `verstak-desktop/internal/api/app.go`
- Modify: `verstak-desktop/internal/api/app_test.go`
- Modify: `verstak-desktop/internal/core/vault/vault.go`
- Modify: `verstak-desktop/internal/core/workspacetree/service.go`
- Modify: `verstak-desktop/frontend/src/App.svelte`
- Modify: `verstak-desktop/frontend/src/lib/shell/WorkspaceTree.svelte`
- Modify: `verstak-desktop/frontend/src/lib/shell/WorkspaceHost.svelte`

**Interfaces:**
- Consumes: Task 1 metadata API.
- Produces: persistent UUID current selection and UUID-based startup/navigation; legacy Wails methods delegate to the canonical service without rescanning.

- [ ] **Step 1: Add failing tests for nested startup and current UUID persistence**

```go
func TestOpenDefaultWorkspaceUsesNestedDealFromCanonicalTree(t *testing.T) {
    app := newAppWithNestedDeal(t, "folder/deal")
    route := app.openDefaultWorkspaceRoute()
    require.Equal(t, app.treeV2.GetTree().CurrentWorkspaceID, route.WorkspaceID)
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/api -run 'Test(OpenDefaultWorkspace|SetCurrentWorkspaceV2|Navigation)' -count=1`

Expected: nested Deal/current UUID assertions fail against the legacy flat manager path.

- [ ] **Step 3: Implement canonical selection and adapters**

Persist `<vault>/.verstak/workspace-ui.json` as:

```json
{"schemaVersion":2,"currentWorkspaceId":"UUID","updatedAt":"RFC3339"}
```

Make startup, navigation, workspace listing, metadata lookup, and remote sync application resolve through `treeV2`. Keep old exported method names only as thin callers until frontend bindings are removed.

- [ ] **Step 4: Verify GREEN and full Desktop tests**

Run:

```bash
go test ./internal/api ./internal/core/vault ./internal/core/workspacetree -count=1
bash scripts/test.sh
bash scripts/check.sh
```

- [ ] **Step 5: Commit and push**

```bash
git add internal frontend/src
git commit -m "refactor: make Deal tree the workspace authority"
git push origin main
```

---

### Task 3: Dormant one-shot backup and migration runner

**Files:**
- Create: `verstak-desktop/internal/core/dealmigration/runner.go`
- Create: `verstak-desktop/internal/core/dealmigration/backup.go`
- Create: `verstak-desktop/internal/core/dealmigration/runner_test.go`
- Create: `verstak-desktop/internal/core/dealmigration/testdata/v016-vault/`
- Modify: `verstak-desktop/internal/api/app.go`

**Interfaces:**
- Produces: `Runner.Preflight`, `Runner.Prepare`, `Runner.Apply`, `Runner.Verify`, `Ledger` states `prepared|applied|verified`.
- Constraint: production startup does not invoke `Apply` until Task 12 registers the complete target schema bundle.

- [ ] **Step 1: Add failing tests for immutable backup, rollback, and idempotence**

```go
func TestRunnerRestoresExactBytesWhenApplyFails(t *testing.T) {
    vault := copyFixture(t, "testdata/v016-vault")
    before := hashTree(t, vault)
    runner := NewRunner(vault, WithInjectedFailure("after-provider-write"))
    require.Error(t, runner.Run(context.Background()))
    require.Equal(t, before, hashTree(t, vault))
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/core/dealmigration -count=1`

Expected: package/API missing.

- [ ] **Step 3: Implement backup and ledger**

Use SHA-256 manifests, exclusive backup-directory creation, atomic ledger writes, explicit source/target digests, and rollback from exact bytes. Reject symlinks and paths outside the vault metadata roots.

- [ ] **Step 4: Keep production activation disabled**

Expose diagnostics/preflight through internal wiring only. Do not call the runner from `Initialize` or vault open.

- [ ] **Step 5: Verify GREEN**

Run:

```bash
go test ./internal/core/dealmigration -count=1
bash scripts/check.sh
```

- [ ] **Step 6: Commit and push**

```bash
git add internal/core/dealmigration internal/api/app.go
git commit -m "feat: add one-shot Deal migration runner"
git push origin main
```

---

### Task 4: SDK and Core Deal-only public contracts

**Files:**
- Create: `verstak-sdk/schemas/deal-scope.json`
- Modify: `verstak-sdk/src/plugin-api.ts`
- Modify: `verstak-sdk/src/types.ts`
- Modify: `verstak-sdk/src/test-utils.ts`
- Modify: `verstak-sdk/src/plugin-api.test.ts`
- Modify: `verstak-sdk/schemas/capability-operations.json`
- Modify: `verstak-sdk/schemas/manifest.json`
- Modify: `verstak-desktop/frontend/src/lib/plugin-host/VerstakPluginAPI.js`
- Modify: `verstak-desktop/internal/api/capability_operations.go`
- Modify: `verstak-desktop/internal/api/capability_operations_test.go`
- Modify: `verstak-desktop/frontend/src/lib/test/wails-mock.js`

**Interfaces:**
- Produces: `DealScope`, `DealRef`, UUID-required navigation, typed v2 provider envelopes, host API version validation.

- [ ] **Step 1: Add failing SDK tests**

```ts
expectTypeOf<DealScope>().toEqualTypeOf<{ kind: 'deal'; workspaceId: string }>();
await expect(api.navigation.openWorkspace({ workspaceId: '' })).rejects.toThrow('workspaceId');
```

- [ ] **Step 2: Verify RED**

Run: `npm test -- --run src/plugin-api.test.ts`

- [ ] **Step 3: Implement source/schema/test-utils contracts**

Define provider requests as:

```ts
export interface DealOperationRequest {
  scope: DealScope;
}
```

Add official provider capabilities `verstak/notes/v2`, `verstak/files/v2`,
`verstak/todo/v2`, and `verstak/activity/v2` without disabling current v1
providers in this slice. Validate operation names and the v2 envelope before
dispatch. Task 5 moves every official consumer/provider and then removes the
project-scoped v1 declarations, so the additive bridge exists only between
two bounded commits and never becomes the final compatibility architecture.

- [ ] **Step 4: Build generated SDK output and verify**

Run:

```bash
npm run build
npm run lint
npm test
bash scripts/check.sh
```

- [ ] **Step 5: Commit/push SDK, then Desktop bridge**

```bash
git add src schemas dist package.json
git commit -m "feat: define Deal-only plugin contracts"
git push origin main
```

In `verstak-desktop`:

```bash
git add internal frontend
git commit -m "feat: enforce Deal-only plugin contracts"
git push origin main
```

---

### Task 5: Flatten Notes, Files, Todo, and Activity

**Files:**
- Modify: `verstak-official-plugins/plugins/notes/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/notes/plugin.json`
- Modify: `verstak-official-plugins/plugins/files/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/files/plugin.json`
- Modify: `verstak-official-plugins/plugins/todo/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/todo/plugin.json`
- Modify: `verstak-official-plugins/plugins/activity/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/activity/plugin.json`
- Modify: `verstak-official-plugins/scripts/smoke-notes-plugin.js`
- Modify: `verstak-official-plugins/scripts/smoke-files-plugin.js`
- Modify: `verstak-official-plugins/scripts/smoke-todo-plugin.js`
- Modify: `verstak-official-plugins/scripts/smoke-activity-plugin.js`
- Create: `verstak-desktop/internal/core/dealmigration/provider_transforms.go`
- Create: `verstak-desktop/internal/core/dealmigration/provider_transforms_test.go`

**Interfaces:**
- Consumes: Task 4 `scope: DealScope` contracts.
- Produces: provider storage and operations with `workspaceId` only; migration removes Todo `projectId` and Notes/Files side tables without touching file bytes.

- [ ] **Step 1: Add failing provider and migration tests**

```js
if ('projectId' in migratedTodo) throw new Error('legacy projectId survived');
if (migratedTodo.workspaceId !== 'deal-uuid') throw new Error('Deal UUID missing');
```

```go
require.Equal(t, beforeFileSHA, fileSHA(t, notePath))
require.NoFileExists(t, notesProjectScopeStore)
```

- [ ] **Step 2: Verify RED**

Run focused smoke scripts plus `go test ./internal/core/dealmigration -run Provider -count=1`; expect project-scope assertions to fail.

- [ ] **Step 3: Implement Deal-only storage/operations**

Remove `PROJECT_SCOPE_KEY`, bindings, filters, open-context fields, and `projectId` persistence. Resolve filesystem paths from `workspaceId` through Core immediately before file access.

- [ ] **Step 4: Verify GREEN**

Run:

```bash
bash scripts/check.sh
```

Then in Desktop:

```bash
go test ./internal/core/dealmigration ./internal/api -count=1
VERSTAK_CHECK_E2E=0 bash scripts/verstak-check.sh
```

- [ ] **Step 5: Commit/push provider repo, then Desktop transforms**

Commit messages:

```text
refactor: return provider resources to Deal scope
feat: migrate provider records to Deal scope
```

---

### Task 6: Rewrite Projects as Project Meta and portfolio

**Files:**
- Modify: `verstak-official-plugins/plugins/projects/frontend/src/index.js`
- Modify: `verstak-official-plugins/plugins/projects/plugin.json`
- Modify: `verstak-official-plugins/plugins/projects/locales/en.json`
- Modify: `verstak-official-plugins/plugins/projects/locales/ru.json`
- Rewrite: `verstak-official-plugins/scripts/smoke-projects-plugin.js`
- Modify: `verstak-desktop/frontend/e2e/projects.spec.js`
- Modify: `verstak-desktop/frontend/e2e/visual/projects.visual.js`
- Modify: `verstak-desktop/frontend/src/lib/test/wails-mock.js`
- Create: `verstak-desktop/internal/core/dealmigration/project_meta_transform.go`

**Interfaces:**
- Produces: one Project Meta record per `workspaceId`; global portfolio navigation by Deal UUID; no nested tabs/resources.

- [ ] **Step 1: Add failing smoke/E2E tests**

```js
if (settings['project-meta:records'].filter(x => x.workspaceId === 'deal-a').length !== 1) throw new Error('Project Meta cardinality');
if (navigation.workspaceId !== 'deal-a' || navigation.toolRequest?.projectId) throw new Error('portfolio navigation is not Deal-only');
```

- [ ] **Step 2: Verify RED**

Run: `node scripts/smoke-projects-plugin.js` and focused Playwright Projects spec.

- [ ] **Step 3: Replace nested Projects implementation**

Keep plugin ID `verstak.projects`, rename display copy to Project Meta/Projects as appropriate, persist records keyed by `workspaceId`, and make the global view derive cards from Deals where `workspaceTools` includes `verstak.projects`.

- [ ] **Step 4: Add best-effort transform**

Exactly one unambiguous Project becomes Project Meta. Multiple Projects produce no chosen metadata and are counted in the migration report/backup.

- [ ] **Step 5: Verify, commit, and push**

Run official checks and focused Desktop E2E, then commit:

```text
refactor: make Projects a Deal metadata plugin
test: cover Deal-only Projects portfolio
```

---

### Task 7: Independent Milestones plugin

**Files:**
- Create: `verstak-official-plugins/plugins/milestones/plugin.json`
- Create: `verstak-official-plugins/plugins/milestones/frontend/src/index.js`
- Create: `verstak-official-plugins/plugins/milestones/locales/en.json`
- Create: `verstak-official-plugins/plugins/milestones/locales/ru.json`
- Create: `verstak-official-plugins/scripts/smoke-milestones-plugin.js`
- Modify: `verstak-official-plugins/scripts/check.sh`
- Create: `verstak-desktop/internal/core/dealmigration/milestone_transform.go`
- Create: `verstak-desktop/internal/core/dealmigration/milestone_transform_test.go`

**Interfaces:**
- Produces: `verstak/milestones/v1` list/create/update/delete operations scoped by Deal UUID and independent Deal workspace UI.

- [ ] **Step 1: Add failing smoke and migration tests**

```js
await mountMilestones({ workspaceId: 'deal-a' }, api);
await commands.create({ scope: { kind: 'deal', workspaceId: 'deal-a' }, title: 'Beta' });
if (records[0].workspaceId !== 'deal-a') throw new Error('milestone not Deal-scoped');
```

- [ ] **Step 2: Verify RED, implement CRUD/UI, then GREEN**

Run `node scripts/smoke-milestones-plugin.js`; implement stable IDs, status, due date, timestamps, and migration provenance; rerun until green.

- [ ] **Step 3: Verify non-Project Deal behavior**

Mount with workspace tools containing Milestones but not Projects and assert full CRUD.

- [ ] **Step 4: Run checks, commit, push**

Commit official plugin and Desktop migration transforms separately:

```text
feat: add Deal milestones plugin
feat: migrate legacy project milestones
```

---

### Task 8: Templates CRUD and generic recipe creation

**Files:**
- Create: `verstak-desktop/internal/core/workspacetree/recipe.go`
- Create: `verstak-desktop/internal/core/workspacetree/recipe_test.go`
- Modify: `verstak-desktop/internal/core/workspacetree/lifecycle.go`
- Modify: `verstak-desktop/internal/api/app.go`
- Modify: `verstak-desktop/frontend/src/lib/shell/WorkspaceTree.svelte`
- Delete runtime registries from: `verstak-desktop/internal/core/workspace/manager.go`, `verstak-desktop/internal/core/workspacetree/lifecycle.go`
- Create: `verstak-official-plugins/plugins/templates/plugin.json`
- Create: `verstak-official-plugins/plugins/templates/frontend/src/index.js`
- Create: `verstak-official-plugins/plugins/templates/defaults.json`
- Create: `verstak-official-plugins/plugins/templates/locales/en.json`
- Create: `verstak-official-plugins/plugins/templates/locales/ru.json`
- Create: `verstak-official-plugins/scripts/smoke-templates-plugin.js`
- Modify: `verstak-sdk/src/plugin-api.ts`
- Modify: `verstak-sdk/schemas/manifest.json`

**Interfaces:**
- Produces: generic `DealRecipeSnapshot`, template preset contribution schema, persisted template CRUD, and transactional `createDeal(recipe)`.

- [ ] **Step 1: Add RED tests for generic recipes and CRUD**

```go
recipe := DealRecipeSnapshot{WorkspaceTools: []string{"third.party"}, InitialFolders: []string{"Brief"}}
ws, err := svc.CreateWorkspaceFromRecipe(parentID, "Deal", recipe, nil)
require.NoError(t, err)
require.DirExists(t, filepath.Join(vault, ws.RootPath, "Brief"))
```

```js
const copy = await duplicateTemplate('project');
if (copy.id === 'project' || copy.workspaceTools.join() !== seeds.project.workspaceTools.join()) throw new Error('duplicate failed');
```

- [ ] **Step 2: Implement Core recipe validation without plugin IDs**

Validate relative folders/files, size/count limits, duplicate paths, namespaced presets, installed tool IDs, and deterministic SHA-256 provenance digest.

- [ ] **Step 3: Implement Templates persistence/UI/seeds**

Seed General, Project, Writing, Admin, and Minimal into plugin settings. Restore is explicit and idempotent. The Project seed lists `verstak.projects`, `verstak.git`, `verstak.todo`, `verstak.milestones`, `verstak.notes`, `verstak.files`, `verstak.activity`, `verstak.journal`, and `verstak.secrets`.

- [ ] **Step 4: Remove hardcoded registries/mocks and verify**

Run SDK tests, official plugin checks, Desktop Go tests, and `frontend/e2e/workspace-templates.spec.js`.

- [ ] **Step 5: Commit/push in dependency order**

```text
SDK: feat: define Deal recipe contracts
Plugins: feat: add persisted Deal templates
Desktop: refactor: create Deals from recipe snapshots
```

---

### Task 9: Backend Git and Secrets boundary

**Files:**
- Create: `verstak-desktop/internal/core/gitservice/service.go`
- Create: `verstak-desktop/internal/core/gitservice/service_test.go`
- Create: `verstak-desktop/internal/core/secrets/credentials.go`
- Create: `verstak-desktop/internal/core/secrets/credentials_test.go`
- Modify: `verstak-desktop/internal/api/app.go`
- Modify: `verstak-desktop/frontend/src/lib/plugin-host/VerstakPluginAPI.js`
- Modify: `verstak-sdk/src/plugin-api.ts`
- Modify: `verstak-sdk/src/types.ts`
- Modify: `verstak-sdk/schemas/permissions.json`

**Interfaces:**
- Produces: narrow Git operations, sanitized results, logical credential slots mapped to local secrets, no frontend plaintext.

- [ ] **Step 1: Add failing security tests**

```go
func TestCloneRedactsCredentialsFromResultAndLogs(t *testing.T) {
    result, logs := runCloneWithRemote(t, "https://token@example.test/repo.git")
    require.NotContains(t, string(result.JSON), "token")
    require.NotContains(t, logs, "token")
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./internal/core/gitservice ./internal/core/secrets -count=1`.

- [ ] **Step 3: Implement bounded process execution**

Use `exec.CommandContext` with explicit argv, validated Deal/repository roots, environment allowlist, output caps, timeouts, cancellation, and URL redaction. Implement status through porcelain v2, branch/upstream counts, and bounded recent log output.

- [ ] **Step 4: Implement credential-slot resolution**

Store `{workspaceId, slotId, secretId}` only in device-local encrypted Secrets data. Materialize one-use askpass/SSH environment in a mode-0700 temporary directory, remove it after the process, and never return secret values through Wails.

- [ ] **Step 5: Verify, commit, push SDK then Desktop**

Run SDK full tests and Desktop package tests/checks. Commit messages:

```text
feat: define secure Git broker contracts
feat: add credential-bound Git service
```

---

### Task 10: Multi-repository Git plugin

**Files:**
- Create: `verstak-official-plugins/plugins/git/plugin.json`
- Create: `verstak-official-plugins/plugins/git/frontend/src/index.js`
- Create: `verstak-official-plugins/plugins/git/locales/en.json`
- Create: `verstak-official-plugins/plugins/git/locales/ru.json`
- Create: `verstak-official-plugins/scripts/smoke-git-plugin.js`
- Modify: `verstak-official-plugins/scripts/check.sh`
- Modify: `verstak-desktop/frontend/src/lib/test/wails-mock.js`
- Create: `verstak-desktop/frontend/e2e/git.spec.js`

**Interfaces:**
- Consumes: Task 9 Git broker and credential slots.
- Produces: synced sanitized descriptors and device-local clone state for zero-to-many repositories per Deal.

- [ ] **Step 1: Add failing smoke tests**

```js
await addDescriptor({ workspaceId: 'deal-a', id: 'repo-1', remoteUrl: 'git@example.test:a.git' });
await addDescriptor({ workspaceId: 'deal-a', id: 'repo-2', remoteUrl: 'https://example.test/b.git' });
if (listForDeal('deal-a').length !== 2) throw new Error('multiple repositories not retained');
if (JSON.stringify(settings).includes('plaintext-token')) throw new Error('credential leaked');
```

- [ ] **Step 2: Implement descriptor/device state split**

Sync only stable ID, workspace ID, display name, sanitized remote, preferred branch, relative checkout name, and logical slot. Store absolute checkout resolution, cloned flag, and local slot mapping device-locally.

- [ ] **Step 3: Implement bounded UI**

Provide add existing, clone, status/refresh, changed/untracked list, branch, ahead/behind, recent commits, fetch, fast-forward pull, push, open directory, and not-cloned state. Exclude commit/merge/history editing.

- [ ] **Step 4: Verify, commit, push**

Run official checks and focused Playwright. Commit: `feat: add multi-repository Deal Git plugin`.

---

### Task 11: Sync exclusion and snapshot transition

**Files:**
- Modify: `verstak-desktop/internal/core/sync/snapshot.go`
- Modify: `verstak-desktop/internal/core/sync/snapshot_test.go`
- Modify: `verstak-desktop/internal/core/sync/sync_integration_test.go`
- Modify: `verstak-desktop/internal/api/app.go`
- Modify: `verstak-desktop/internal/api/app_test.go`
- Create: `verstak-desktop/internal/core/dealmigration/sync_checkout_transform.go`
- Create: `verstak-desktop/internal/core/dealmigration/sync_checkout_transform_test.go`

**Interfaces:**
- Produces: exclusion of managed `<Deal>/Repositories/**` on scan and pull; snapshot rebase without delete operations.

- [ ] **Step 1: Add RED tests**

```go
func TestManagedRepositoryTreeIsNeitherPushedNorApplied(t *testing.T) {
    writeFile(t, vault, "Deal/Repositories/repo/.git/config", "secret")
    writeFile(t, vault, "Deal/Repositories/repo/main.go", "package main")
    ops := scanOps(t, vault)
    require.Empty(t, filterPrefix(ops, "Deal/Repositories/"))
    require.NoError(t, applyRemoteFile(t, "Deal/Repositories/repo/.git/config"))
    require.NoFileExists(t, filepath.Join(vault, "Deal/Repositories/repo/.git/config"))
}
```

- [ ] **Step 2: Verify RED, implement scan/apply guard and rebase, then GREEN**

The transition removes excluded entries from the local snapshot without generating deletes. Pull consumes sequence numbers but never materializes excluded paths.

- [ ] **Step 3: Verify server protocol remains unchanged**

Run Desktop sync unit/integration tests and existing Sync Server `go test ./...`; no server commit is made unless these tests expose a wire requirement.

- [ ] **Step 4: Commit and push**

Commit: `fix: keep Git checkouts device-local in Sync`.

---

### Task 12: Activate migration and close integrated gates

**Files:**
- Modify: `verstak-desktop/internal/core/dealmigration/transforms.go`
- Modify: `verstak-desktop/internal/api/app.go`
- Modify: `verstak-desktop/internal/api/app_test.go`
- Modify: `verstak-desktop/frontend/e2e/projects.spec.js`
- Modify: `verstak-desktop/frontend/e2e/workspace-templates.spec.js`
- Modify: `verstak-desktop/frontend/e2e/git.spec.js`
- Modify: `verstak-desktop/docs/PLUGIN_RUNTIME.md`
- Modify: `verstak-desktop/docs/WORKSPACE_TEMPLATES.md`
- Create: `verstak-desktop/release-notes/deal-only-scope.md`
- Modify: `verstak-official-plugins/AGENTS.md`
- Modify: `verstak-sdk/README.md`
- Modify: `verstak-sync-server/README.md` only to correct sync ownership documentation.

**Interfaces:**
- Consumes: all prior target schemas/transforms.
- Produces: one production migration bundle version, verified new runtime, and current documentation.

- [ ] **Step 1: Add end-to-end fixture migration test**

Assert exact provider IDs/counts, SHA-256 of every note/file, removal of all runtime `projectId`, inert backup presence, Project Meta best-effort behavior, Milestones transfer, and second-run no-op.

- [ ] **Step 2: Enable the complete migration bundle**

Register all transforms under `deal-only-v1` and invoke once during vault open before plugins read provider stores. Refuse normal open on apply/verify failure and show the backup/report path.

- [ ] **Step 3: Run focused and full automated gates**

```bash
go test ./internal/core/dealmigration ./internal/core/workspacetree ./internal/core/sync ./internal/api -count=1
npm --prefix frontend run test:e2e
bash scripts/verstak-check.sh
```

- [ ] **Step 4: Build and run real GUI smoke**

```bash
bash scripts/build.sh
bash scripts/gui-probe.sh
```

Inspect the captured Project Meta, Milestones, Templates, and Git views in the real Wails/WebKitGTK shell. Record absolute artifact and screenshot paths.

- [ ] **Step 5: Update docs and release notes without publishing**

Document Deal-only contracts, migration backup/report, Templates ownership, Git device-local behavior, Secrets slots, and removed nested Projects semantics.

- [ ] **Step 6: Commit/push each repository and verify remotes**

Use one final integration/docs commit per changed repository, push dependency order SDK -> official plugins -> desktop -> sync-server docs, then prove `HEAD == origin/main` and every worktree is clean.
