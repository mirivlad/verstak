# DokuWiki and Obsidian Import Plugin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a safe generic desktop import boundary and one official `verstak.import` plugin that analyzes current DokuWiki or Obsidian content, proposes an editable structure, and publishes the approved result below a unique `Импортировано/<format> — <timestamp>` run.

**Architecture:** The SDK defines one format-neutral `api.imports` contract. Desktop owns native selection, opaque source sessions, archive safety, progress/cancellation, validation, staging, transaction recovery, and workspace publication; the official plugin owns DokuWiki/Obsidian detection, conversion, link resolution, classification, the editable plan, and all user-facing UI. The plugin never receives arbitrary filesystem access or binary payloads.

**Tech Stack:** Go 1.25, Wails 2.12, Svelte 4, Vite 5, JavaScript ESM, TypeScript 5.4 SDK contracts, Vitest 1.6, Playwright 1.61, Go standard-library ZIP/TAR/GZIP readers, `golang.org/x/sys` for free-space checks.

## Global Constraints

- Support only current DokuWiki pages and current Obsidian vault content; do not import DokuWiki revisions or `.obsidian/` service data and do not report those omissions.
- Compatibility is pinned to the current stable DokuWiki `2025-05-14b "Librarian"` layout and the current Obsidian Help link/embed/properties contract reviewed on 2026-07-23; do not add format-specific legacy branches.
- Convert DokuWiki wiki markup to Markdown; preserve Obsidian Markdown/frontmatter/tasks/tags/callouts and rewrite supported wiki links, headings, block references, images, and file embeds.
- Import credentials, passwords, recovery codes, and comparable content as ordinary pages without classification or intervention; show one general post-import review warning.
- Always publish below a compatible root organizational folder named `Импортировано`; each run is a new `DokuWiki — YYYY-MM-DD HH-mm-ss` or `Obsidian — YYYY-MM-DD HH-mm-ss` folder with a deterministic numeric suffix on collision.
- Supported sources are a selected directory or `.zip`, `.tar`, `.tar.gz`, `.tgz` archive.
- Fixed limits are 250,000 entries, 20 GiB uncompressed total, 2 GiB per copied file, 16 MiB per JavaScript text read, and 1,000:1 declared archive expansion ratio.
- Reject absolute, drive-qualified, traversing, NUL/device, symlink, hardlink, special, duplicate-normalized, and case-fold-colliding paths; directory sessions never follow symlinks.
- Use the existing `default` workspace template and generic capabilities; core must not contain DokuWiki/Obsidian format logic or official plugin IDs.
- The master uses existing Verstak colors, spacing, typography, controls, modal surface, and Lucide icon contract; it adds no emoji icons or private palette.
- The personal backup archives stay outside git, and logs/tests must not print page bodies, credentials, or sensitive filenames.
- Every internally complete repository block is checked, committed, and pushed before dependent work is declared complete.
- Release version for desktop and official-plugin archives is `v0.1.0-beta.20260723`; both release notes files must exist before publication.

---

## File Map

### `verstak-sdk`

- Modify `schemas/capabilities.json`: register `verstak/core/import/v1` and `verstak/import/v1`.
- Modify `schemas/permissions.json` and `schemas/manifest.json`: register dangerous `imports.readExternal` and `imports.apply` permissions.
- Modify `src/types.ts`: add source, entry, plan, progress, and result DTOs.
- Modify `src/plugin-api.ts`: add the typed `imports` namespace.
- Modify `src/test-utils.ts`: add deterministic in-memory import sessions to `createMockPluginAPI`.
- Modify `src/plugin-api.test.ts`: lock schema, DTO, mock, cancellation, and cleanup behavior.

### `verstak-desktop`

- Create `internal/core/importservice/types.go`: JSON DTOs, limits, error codes, and interfaces shared by source and apply code.
- Create `internal/core/importservice/path.go`: canonical source/target path policy and collision keys.
- Create `internal/core/importservice/directory.go`: no-follow directory inventory and bounded reads.
- Create `internal/core/importservice/archive.go`: ZIP/TAR/TAR.GZ indexing and safe entry readers.
- Create `internal/core/importservice/service.go`: plugin-scoped opaque sessions, pagination, fingerprints, progress, cancellation, expiry, and cleanup.
- Create `internal/core/importservice/plan.go`: untrusted plan validation and final target derivation.
- Create `internal/core/importservice/apply.go`: staging, free-space check, payload streaming, publication, and result counts.
- Create `internal/core/importservice/recovery.go`: journal promotion/rollback and abandoned-stage cleanup before scanning.
- Create `internal/core/importservice/diskspace_unix.go` and `diskspace_windows.go`: platform free-space adapters.
- Create focused `*_test.go` files beside each responsibility.
- Create `internal/core/workspacetree/import.go` and `import_test.go`: prepare a template-backed Deal in staging without publishing global registry metadata.
- Modify `internal/core/capability/platform.go` and `internal/core/permissions/registry.go`: expose the new generic host contract.
- Modify `internal/api/app.go` and `internal/api/app_test.go`: bind import lifecycle and permission/capability-checked Wails methods.
- Modify `frontend/src/lib/plugin-host/VerstakPluginAPI.js`: bridge sessions/progress/cancel and dispose cleanup.
- Modify `frontend/src/lib/plugin-host/PluginBundleHost.svelte` and `CompactPluginHost.svelte`: load and remove a manifest-declared plugin stylesheet generically.
- Create `frontend/tests/plugin-api-imports-test.mjs` and modify `frontend/tests/wails-bindings-test.mjs`.
- Regenerate `frontend/wailsjs/go/api/App.js`, `App.d.ts`, and `frontend/wailsjs/go/models.ts` with Wails.
- Modify `frontend/src/lib/test/wails-mock.js` and create `frontend/e2e/import-plugin.spec.js` for the complete mocked wizard flow.

### `verstak-official-plugins`

- Create `plugins/import/plugin.json`, `locales/en.json`, and `locales/ru.json`.
- Create `plugins/import/frontend/package.json`, lockfile, `vite.config.js`, and `src/index.js`.
- Create `plugins/import/frontend/src/ImportSettings.svelte`: four-step master using host styling.
- Create `plugins/import/frontend/src/model/source.js`: paginated source loading and candidate selection.
- Create `plugins/import/frontend/src/model/graph.js`: neutral source graph and deterministic IDs.
- Create `plugins/import/frontend/src/model/plan.js`: adaptive plan generation, edits, validation, and final SDK plan serialization.
- Create `plugins/import/frontend/src/dokuwiki/detect.js`, `convert.js`, and `adapter.js`.
- Create `plugins/import/frontend/src/obsidian/detect.js`, `links.js`, and `adapter.js`.
- Create synthetic fixture trees under `plugins/import/frontend/test/fixtures/` and Vitest tests next to the adapters/model.
- Create `scripts/smoke-import-plugin.js`; modify `scripts/check.sh` so importer tests/smoke join the normal gate.
- Modify `README.md`, `README.ru.md`, and create `release-notes/v0.1.0-beta.20260723.md`.

### `verstak-docs`

- Modify `04_Plugin_System.md`: document the import capability, permissions, sessions, limits, progress, and cancellation.
- Modify `05_Official_Plugins.md`: document `verstak.import`, supported formats, conversion, isolation, and the sensitive-content warning.
- Modify `07_Full_Implementation_Roadmap.md`: replace the deferred large-import statement with the bounded session/streaming contract and mark the official importer delivered.
- Modify `README.md` and `README.ru.md`: link the import guide section.

---

### Task 1: SDK import contract

**Files:**
- Modify: `/home/mirivlad/git/verstak2/verstak-sdk/schemas/capabilities.json`
- Modify: `/home/mirivlad/git/verstak2/verstak-sdk/schemas/permissions.json`
- Modify: `/home/mirivlad/git/verstak2/verstak-sdk/schemas/manifest.json`
- Modify: `/home/mirivlad/git/verstak2/verstak-sdk/src/types.ts`
- Modify: `/home/mirivlad/git/verstak2/verstak-sdk/src/plugin-api.ts`
- Modify: `/home/mirivlad/git/verstak2/verstak-sdk/src/test-utils.ts`
- Test: `/home/mirivlad/git/verstak2/verstak-sdk/src/plugin-api.test.ts`

**Interfaces:**
- Consumes: existing `Unsubscribe`, manifest schema, and `createMockPluginAPI` patterns.
- Produces: `ImportSourceSession`, `ImportSourceEntry`, `ImportEntryPage`, `ImportPlanNode`, `ImportPlan`, `ImportProgress`, `ImportApplyResult`, and `VerstakPluginAPI.imports` used verbatim by Tasks 2-10.

- [ ] **Step 1: Add failing SDK contract tests**

Add assertions for both capabilities, both dangerous permissions, the manifest enum, and the runtime mock:

```ts
test('generic import contract is registered and mockable', async () => {
  const capabilities = (capabilitiesSchema as any).capabilities;
  const permissions = (permissionsSchema as any).permissions;
  const permissionEnum = (manifestSchema as any).properties.permissions.items.enum;
  expect(capabilities).toEqual(expect.arrayContaining([
    expect.objectContaining({ name: 'verstak/core/import/v1', status: 'draft' }),
    expect.objectContaining({ name: 'verstak/import/v1', status: 'draft' }),
  ]));
  expect(permissions).toEqual(expect.arrayContaining([
    expect.objectContaining({ name: 'imports.readExternal', dangerous: true }),
    expect.objectContaining({ name: 'imports.apply', dangerous: true }),
  ]));
  expect(permissionEnum).toEqual(expect.arrayContaining(['imports.readExternal', 'imports.apply']));

  const api = createMockPluginAPI('verstak.import', {
    importSources: [{
      session: { sourceHandle: 'source-1', kind: 'directory', displayPath: '/chosen/wiki', displayName: 'wiki', fingerprint: 'fp-1', entryCount: 1, totalBytes: 4 },
      entries: [{ id: 'entry-1', path: 'pages/start.txt', kind: 'file', size: 4, modifiedAt: '2026-07-23T00:00:00Z', mediaHint: 'text/plain' }],
      textByEntryId: { 'entry-1': 'test' },
    }],
  });
  const source = await api.imports.selectDirectory();
  expect(source?.sourceHandle).toBe('source-1');
  expect((await api.imports.listEntries('source-1')).entries).toHaveLength(1);
  expect(await api.imports.readText('source-1', 'entry-1')).toBe('test');
  await api.imports.cancel('source-1');
  await api.imports.closeSource('source-1');
});
```

- [ ] **Step 2: Run the focused SDK test and confirm red**

Run: `npm test -- src/plugin-api.test.ts`

Expected: FAIL because the import capabilities, permissions, options, and `api.imports` do not exist.

- [ ] **Step 3: Add the exact neutral DTOs and API namespace**

Add these public shapes to `src/types.ts` and import them in `src/plugin-api.ts`:

```ts
export type ImportSourceKind = 'directory' | 'archive';
export type ImportEntryKind = 'directory' | 'file';
export type ImportPlanNodeKind = 'folder' | 'workspace' | 'note' | 'file' | 'skip';

export interface ImportSourceSession {
  sourceHandle: string;
  kind: ImportSourceKind;
  displayPath: string;
  displayName: string;
  fingerprint: string;
  entryCount: number;
  totalBytes: number;
}

export interface ImportSourceEntry {
  id: string;
  path: string;
  kind: ImportEntryKind;
  size: number;
  modifiedAt: string;
  mediaHint: string;
}

export interface ImportEntryPage {
  entries: ImportSourceEntry[];
  nextCursor: string;
  fingerprint: string;
}

export interface ImportPlanNode {
  id: string;
  parentId: string;
  kind: ImportPlanNodeKind;
  name: string;
  targetSubpath?: string;
  templateId?: string;
  text?: string;
  sourceEntryId?: string;
  sourcePath?: string;
  modifiedAt?: string;
}

export interface ImportPlan {
  schemaVersion: 1;
  sourceFingerprint: string;
  runName: string;
  nodes: ImportPlanNode[];
}

export interface ImportProgress {
  sourceHandle: string;
  phase: 'indexing' | 'validating' | 'staging' | 'publishing' | 'refreshing';
  completed: number;
  total: number;
  cancellable: boolean;
  message: string;
}

export interface ImportApplyResult {
  runPath: string;
  folders: number;
  workspaces: number;
  notes: number;
  files: number;
  skipped: number;
  warnings: string[];
}
```

Add the namespace to `VerstakPluginAPI`:

```ts
imports: {
  selectDirectory(): Promise<ImportSourceSession | null>;
  selectArchive(): Promise<ImportSourceSession | null>;
  listEntries(sourceHandle: string, cursor?: string): Promise<ImportEntryPage>;
  readText(sourceHandle: string, entryId: string): Promise<string>;
  onProgress(sourceHandle: string, listener: (progress: ImportProgress) => void): Unsubscribe;
  applyPlan(sourceHandle: string, plan: ImportPlan): Promise<ImportApplyResult>;
  cancel(sourceHandle: string): Promise<void>;
  closeSource(sourceHandle: string): Promise<void>;
};
```

Register the two names in `capabilities.json`, add the two dangerous registry rows to `permissions.json`, and add both strings to `manifest.json`'s permission enum. Extend `MockPluginAPIOptions` with `importSources` using the test shape, keep one selected-source cursor, emit progress to registered listeners during `applyPlan`, and make cancel/close idempotent.

- [ ] **Step 4: Run all SDK checks and confirm green**

Run: `npm run lint && npm test && npm run build`

Expected: TypeScript emits no errors; Vitest reports all tests passed; `dist/plugin-api.d.ts` contains `imports`.

- [ ] **Step 5: Commit and push the SDK contract**

```bash
git add schemas src package-lock.json
git commit -m "feat: define generic import plugin API"
git push origin main
```

### Task 2: Safe directory and archive source sessions

**Files:**
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/types.go`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/path.go`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/directory.go`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/archive.go`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/service.go`
- Test: matching `path_test.go`, `directory_test.go`, `archive_test.go`, and `service_test.go`

**Interfaces:**
- Consumes: SDK JSON field names from Task 1.
- Produces: `New(vaultDir string, options Options) *Service`, `OpenDirectory`, `OpenArchive`, `ListEntries`, `ReadText`, `Subscribe`, `Cancel`, `Close`, `ClosePlugin`, and `CloseAll` for Tasks 4-5.

- [ ] **Step 1: Write failing path and source tests**

Use table tests that create safe trees plus ZIP/TAR/TAR.GZ fixtures in `t.TempDir()` and assert:

```go
func TestArchiveRejectsUnsafeEntries(t *testing.T) {
  for _, name := range []string{"../escape.txt", "/absolute.txt", "C:/drive.txt", "safe/../../escape.txt", "NUL"} {
    t.Run(name, func(t *testing.T) {
      archive := writeZipFixture(t, []zipFixture{{Name: name, Body: []byte("x")}})
      _, err := New(t.TempDir(), Options{}).OpenArchive("verstak.import", archive)
      if err == nil || !strings.Contains(err.Error(), "unsafe-source-path") {
        t.Fatalf("expected unsafe-source-path, got %v", err)
      }
    })
  }
}

func TestDirectoryDoesNotFollowSymlinks(t *testing.T) {
  root := t.TempDir()
  outside := filepath.Join(t.TempDir(), "secret.txt")
  if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil { t.Fatal(err) }
  if err := os.Symlink(outside, filepath.Join(root, "linked.txt")); err != nil { t.Skipf("symlink unavailable: %v", err) }
  _, err := New(t.TempDir(), Options{}).OpenDirectory("verstak.import", root)
  if err == nil || !strings.Contains(err.Error(), "unsupported-source-entry") { t.Fatalf("expected unsupported-source-entry, got %v", err) }
}

func TestSessionOwnershipAndBoundedText(t *testing.T) {
  root := t.TempDir()
  if err := os.WriteFile(filepath.Join(root, "page.md"), []byte("hello"), 0o600); err != nil { t.Fatal(err) }
  service := New(t.TempDir(), Options{})
  session, err := service.OpenDirectory("verstak.import", root)
  if err != nil { t.Fatal(err) }
  page, err := service.ListEntries("other.plugin", session.SourceHandle, "")
  if err == nil || !strings.Contains(err.Error(), "source-session-owner") { t.Fatalf("expected source-session-owner, got %v", err) }
  if len(page.Entries) != 0 { t.Fatalf("entries=%d", len(page.Entries)) }
}
```

Use local assertion helpers instead of adding `testify`; the final tests must depend only on the standard library.

- [ ] **Step 2: Run package tests and confirm red**

Run: `go test ./internal/core/importservice -run 'Test(Archive|Directory|Session)' -count=1`

Expected: FAIL because package `importservice` does not exist.

- [ ] **Step 3: Add source DTOs, policies, readers, and session ownership**

Define the service boundary exactly as:

```go
const (
  MaxEntries          = 250_000
  MaxTotalBytes int64 = 20 << 30
  MaxEntryBytes int64 = 2 << 30
  MaxTextBytes  int64 = 16 << 20
  MaxExpansionRatio   = 1000
  PageSize            = 500
)

type SourceSession struct {
  SourceHandle string `json:"sourceHandle"`
  Kind string `json:"kind"`
  DisplayPath string `json:"displayPath"`
  DisplayName string `json:"displayName"`
  Fingerprint string `json:"fingerprint"`
  EntryCount int `json:"entryCount"`
  TotalBytes int64 `json:"totalBytes"`
}

type Entry struct {
  ID string `json:"id"`
  Path string `json:"path"`
  Kind string `json:"kind"`
  Size int64 `json:"size"`
  ModifiedAt string `json:"modifiedAt"`
  MediaHint string `json:"mediaHint"`
}

type EntryPage struct {
  Entries []Entry `json:"entries"`
  NextCursor string `json:"nextCursor"`
  Fingerprint string `json:"fingerprint"`
}
```

Normalize paths with forward slashes, reject empty file paths and Windows reserved basenames after trimming trailing dots/spaces, and use `strings.ToLower(norm.NFC.String(path))` as the case-fold collision key. Entry IDs are `hex(sha256(normalizedPath))`; fingerprints are SHA-256 over source kind, outer stat metadata, and the sorted tuple `(path, kind, size, modified-nanoseconds, zip-crc-if-present)`.

`directorySource.open` must walk with `os.Lstat`, fail on every symlink/special entry, and reopen a file only after verifying that its real parent stays below the selected root and its current stat matches indexed size/mtime. `archiveSource.open` must re-open the selected archive and find the indexed safe regular entry without extracting other entries. `ReadText` reads at most `MaxTextBytes+1`, rejects NUL/binary data, strips one UTF-8 BOM, validates UTF-8, and returns `text-entry-too-large` or `binary-entry` codes.

For ZIP, enforce the 1,000:1 ratio per file from declared compressed/uncompressed sizes. For TAR.GZ, enforce it for the indexed archive total against the outer gzip byte size because TAR does not carry per-entry compressed sizes. A native dialog cancellation returns the zero `SourceSession` with an empty error; the frontend maps an empty `sourceHandle` to `null`.

- [ ] **Step 4: Run all source-session tests and Go diagnostics**

Run: `gofmt -w internal/core/importservice/*.go && go test ./internal/core/importservice -count=1 && go vet ./internal/core/importservice`

Expected: all package tests pass and `go vet` is silent. If `gopls` is unavailable, record that command-level LSP diagnostics were unavailable and use `go test`/`go vet` as the diagnostic gate.

- [ ] **Step 5: Commit and push safe source sessions**

```bash
git add internal/core/importservice
git commit -m "feat(import): add safe external source sessions"
git push origin main
```

### Task 3: Transactional generic plan application

**Files:**
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/workspacetree/import.go`
- Test: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/workspacetree/import_test.go`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/plan.go`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/apply.go`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/recovery.go`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/diskspace_unix.go`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/diskspace_windows.go`
- Test: matching `plan_test.go`, `apply_test.go`, and `recovery_test.go`

**Interfaces:**
- Consumes: Task 2 source sessions and existing `workspacetree.WriteFolderMarker`, template functions, and metadata layout.
- Produces: `Service.ApplyPlan`, `Service.Recover`, `workspacetree.PrepareImportedWorkspace`, and exact transaction journal semantics for Task 5.

- [ ] **Step 1: Write failing validation, atomicity, cancellation, and recovery tests**

Cover invalid parent graphs, duplicate/case-fold paths, path traversal in `targetSubpath`, missing source entries, wrong fingerprint, collision suffixing, successful note/file placement, cancellation before rename, injected metadata-promotion failure, and startup recovery. The happy-path assertion is:

```go
plan := Plan{SchemaVersion: 1, SourceFingerprint: session.Fingerprint, RunName: "DokuWiki — 2026-07-23 12-30-00", Nodes: []PlanNode{
  {ID: "folder", Kind: "folder", Name: "Проекты"},
  {ID: "deal", ParentID: "folder", Kind: "workspace", Name: "Сайт", TemplateID: "default"},
  {ID: "note", ParentID: "deal", Kind: "note", Name: "Старт", TargetSubpath: "Документы/Старт.md", Text: "# Старт\n"},
  {ID: "file", ParentID: "deal", Kind: "file", Name: "logo.png", TargetSubpath: "assets/logo.png", SourceEntryID: imageID},
}}
result, err := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, plan)
if err != nil { t.Fatal(err) }
if result.RunPath != "Импортировано/DokuWiki — 2026-07-23 12-30-00" { t.Fatalf("runPath=%q", result.RunPath) }
assertFileText(t, filepath.Join(vault, filepath.FromSlash(result.RunPath), "Проекты", "Сайт", "Notes", "Документы", "Старт.md"), "# Старт\n")
assertFileBytes(t, filepath.Join(vault, filepath.FromSlash(result.RunPath), "Проекты", "Сайт", "Files", "assets", "logo.png"), imageBytes)
```

- [ ] **Step 2: Run focused tests and confirm red**

Run: `go test ./internal/core/workspacetree ./internal/core/importservice -run 'Test(PrepareImported|ValidatePlan|ApplyPlan|Recover)' -count=1`

Expected: FAIL because the staged workspace and plan applier do not exist.

- [ ] **Step 3: Expose one staging-safe workspace helper**

Add this generic boundary to `workspacetree/import.go`:

```go
type PreparedImportedWorkspace struct {
  ID string
  RegistryJSON []byte
}

func PrepareImportedWorkspace(workspaceDir, name, templateID string) (PreparedImportedWorkspace, error)
```

It validates `name`, generates one UUID, writes the normal workspace marker and selected existing template into `workspaceDir`, then marshals the same metadata payload used by `writeWorkspaceMetadataV2` without writing into the vault registry. Keep `default` as the only template requested by the importer; return an error if template resolution rejects it.

- [ ] **Step 4: Implement untrusted plan validation and transaction publication**

Use these JSON DTOs in `types.go`:

```go
type PlanNode struct {
  ID string `json:"id"`
  ParentID string `json:"parentId"`
  Kind string `json:"kind"`
  Name string `json:"name"`
  TargetSubpath string `json:"targetSubpath,omitempty"`
  TemplateID string `json:"templateId,omitempty"`
  Text string `json:"text,omitempty"`
  SourceEntryID string `json:"sourceEntryId,omitempty"`
  SourcePath string `json:"sourcePath,omitempty"`
  ModifiedAt string `json:"modifiedAt,omitempty"`
}

type Plan struct {
  SchemaVersion int `json:"schemaVersion"`
  SourceFingerprint string `json:"sourceFingerprint"`
  RunName string `json:"runName"`
  Nodes []PlanNode `json:"nodes"`
}
```

Structural parents may be empty or a `folder`; `workspace` may be below a folder; `note`/`file` must directly name a workspace parent and use a normalized `targetSubpath`; `skip` writes nothing. Reject cycles, orphans, empty names, separators in structural names, reserved `.verstak`, non-`default` templates, content over 16 MiB, final duplicate paths, and any final path outside the stage root.

Call the source fingerprint verifier immediately before calculating/staging the plan. Stage at `<vault>/.verstak/import-staging/<handle>/tree`, write a journal under `<vault>/.verstak/import-staging/<handle>/transaction.json`, and prepare registry JSON under `<handle>/registry/<uuid>.json`. If `Импортировано` is absent, stage its folder marker with the run; if it exists, require it to scan as a compatible organizational folder and never merge into an ordinary directory or Deal. Publish by same-filesystem rename, promote registry files with `O_EXCL`, mark committed, then refresh the workspacetree baseline. Cancellation is accepted through staging and rejected once progress enters non-cancellable `publishing`. Recovery rolls back only paths named by an uncommitted journal and removes only abandoned session directories.

Before staging, sum planned copied bytes plus UTF-8 note bytes and require available space greater than that sum plus 64 MiB. Implement Unix with `unix.Statfs` and Windows with `windows.GetDiskFreeSpaceEx` using build tags.

- [ ] **Step 5: Run transactional tests and full backend diagnostics**

Run: `gofmt -w internal/core/workspacetree/import*.go internal/core/importservice/*.go && go test ./internal/core/workspacetree ./internal/core/importservice -count=1 && go vet ./...`

Expected: plan, staging, rollback, and recovery tests pass; `go vet` is silent.

- [ ] **Step 6: Commit and push the generic applier**

```bash
git add internal/core/workspacetree internal/core/importservice go.mod go.sum
git commit -m "feat(import): apply reviewed plans transactionally"
git push origin main
```

### Task 4: Desktop capability, permissions, lifecycle, and Wails API

**Files:**
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/capability/platform.go`
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/permissions/registry.go`
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/internal/api/app.go`
- Test: `/home/mirivlad/git/verstak2/verstak-desktop/internal/api/app_test.go`
- Regenerate: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/wailsjs/go/api/App.js`
- Regenerate: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/wailsjs/go/api/App.d.ts`
- Regenerate: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/wailsjs/go/models.ts`

**Interfaces:**
- Consumes: Task 3 `importservice.Service`.
- Produces: seven plugin-facing Wails methods and `verstak:import-progress` events for Task 5.

- [ ] **Step 1: Add failing API authorization and lifecycle tests**

Create test plugins with combinations of missing capability dependency, missing `imports.readExternal`, missing `imports.apply`, valid permissions, disabled status, and cross-plugin handles. Assert selection uses an injected dialog callback, cancel is idempotent, disabling/reloading closes sessions, closing a vault cancels apply, and `SetCurrentVault` calls recovery before `treeV2.Initialize`.

The public method signatures are:

```go
func (a *App) PluginSelectImportDirectory(pluginID string) (importservice.SourceSession, string)
func (a *App) PluginSelectImportArchive(pluginID string) (importservice.SourceSession, string)
func (a *App) PluginListImportEntries(pluginID, sourceHandle, cursor string) (importservice.EntryPage, string)
func (a *App) PluginReadImportText(pluginID, sourceHandle, entryID string) (string, string)
func (a *App) PluginApplyImportPlan(pluginID, sourceHandle string, plan importservice.Plan) (importservice.ApplyResult, string)
func (a *App) PluginCancelImport(pluginID, sourceHandle string) string
func (a *App) PluginCloseImportSource(pluginID, sourceHandle string) string
```

- [ ] **Step 2: Run focused API tests and confirm red**

Run: `go test ./internal/api -run 'TestPluginImport|TestImportLifecycle' -count=1`

Expected: FAIL because the methods and platform registry rows do not exist.

- [ ] **Step 3: Bind service initialization and cleanup to vault/plugin lifecycle**

Add `imports *importservice.Service` and injectable directory/archive dialog functions to `App`. On every vault open/create/switch, construct the service for the new vault and call `Recover()` before workspacetree initialization. On plugin disable call `ClosePlugin(pluginID)` before lifecycle mutation; on reload, vault close, and shutdown call `CloseAll()`.

Each Wails method must call `requirePluginCapabilityAccess(pluginID, "verstak/core/import/v1")` and the relevant permission: selection/list/read/close/cancel use `imports.readExternal`; apply additionally requires `imports.apply`. Native archive selection uses one `runtime.FileFilter` for `*.zip;*.tar;*.tar.gz;*.tgz`.

Progress callback emits only this DTO and no content/path:

```go
runtime.EventsEmit(a.ctx, "verstak:import-progress", map[string]any{
  "pluginId": pluginID,
  "sourceHandle": progress.SourceHandle,
  "phase": progress.Phase,
  "completed": progress.Completed,
  "total": progress.Total,
  "cancellable": progress.Cancellable,
  "message": progress.Message,
})
```

- [ ] **Step 4: Regenerate bindings and run backend checks**

Run: `$(go env GOPATH)/bin/wails generate module && gofmt -w internal/api/app.go internal/api/app_test.go internal/core/capability/platform.go internal/core/permissions/registry.go && go test ./internal/api ./internal/core/capability ./internal/core/permissions -count=1 && go vet ./...`

Expected: generated bindings export all seven methods; Go tests pass; `go vet` is silent.

- [ ] **Step 5: Commit and push the desktop backend bridge**

```bash
git add internal frontend/wailsjs go.mod go.sum
git commit -m "feat(import): expose permission-scoped desktop bridge"
git push origin main
```

### Task 5: Frontend plugin API bridge and generic stylesheet host

**Files:**
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/src/lib/plugin-host/VerstakPluginAPI.js`
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/src/lib/plugin-host/PluginBundleHost.svelte`
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/src/lib/plugin-host/CompactPluginHost.svelte`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/tests/plugin-api-imports-test.mjs`
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/tests/wails-bindings-test.mjs`

**Interfaces:**
- Consumes: Task 4 Wails methods/events and Task 1 API names.
- Produces: runtime `api.imports` and generic manifest `frontend.style` loading for the official plugin.

- [ ] **Step 1: Write a failing Node bridge smoke test**

Mirror `plugin-api-files-test.mjs` with mocked Wails calls. Assert the selected handle is tracked, only matching `{pluginId, sourceHandle}` progress reaches its listener, `cancel` calls `PluginCancelImport`, explicit close removes the handle, and `dispose()` closes every remaining handle once.

```js
const source = await api.imports.selectArchive();
const progress = [];
const unsubscribe = api.imports.onProgress(source.sourceHandle, (item) => progress.push(item.phase));
window.__emitImportProgress({ pluginId: 'other.plugin', sourceHandle: source.sourceHandle, phase: 'staging' });
window.__emitImportProgress({ pluginId: 'verstak.import', sourceHandle: source.sourceHandle, phase: 'staging' });
if (progress.join(',') !== 'staging') throw new Error(`unexpected progress: ${progress}`);
unsubscribe();
await api.imports.cancel(source.sourceHandle);
api.dispose();
```

- [ ] **Step 2: Run Node tests and confirm red**

Run: `node frontend/tests/plugin-api-imports-test.mjs && node frontend/tests/wails-bindings-test.mjs`

Expected: first command fails because `api.imports` is missing; the bindings test fails until all seven exports are listed.

- [ ] **Step 3: Implement scoped frontend session/progress cleanup**

Create one global Wails bridge for `verstak:import-progress`, but keep listeners keyed by `pluginId:sourceHandle`. In each `createPluginAPI`, track handles in a `Set`; select adds, close deletes, dispose invokes `PluginCloseImportSource` for remaining handles without awaiting and removes all progress listeners.

The bridge methods map one-to-one:

```js
imports: {
  selectDirectory: () => selectImportSource('directory'),
  selectArchive: () => selectImportSource('archive'),
  listEntries: (handle, cursor) => callBackend(pluginId, 'imports.listEntries', () => App.PluginListImportEntries(pluginId, handle, cursor || '')),
  readText: (handle, entryId) => callBackend(pluginId, 'imports.readText', () => App.PluginReadImportText(pluginId, handle, entryId)),
  onProgress: (handle, listener) => trackImportProgress(pluginId, handle, listener),
  applyPlan: (handle, plan) => callBackend(pluginId, 'imports.applyPlan', () => App.PluginApplyImportPlan(pluginId, handle, plan)),
  cancel: (handle) => callBackendErrorString(pluginId, 'imports.cancel', () => App.PluginCancelImport(pluginId, handle)),
  closeSource: (handle) => closeImportSource(pluginId, handle),
}
```

For each bundle host, fetch `info.style` through `GetPluginAssetContent`, inject one `<style data-verstak-plugin-style="pluginId">` element, and remove it in `cleanup()`. Treat an unreadable declared stylesheet as a host error; plugins without `frontend.style` remain unchanged. Define generic public variables on `.plugin-settings-surface` for surface, border, text, muted text, accent, danger, radius, and control height using the current Plugin Manager values; the importer consumes those variables instead of creating its own palette.

- [ ] **Step 4: Run frontend unit/build checks**

Run: `node frontend/tests/plugin-api-imports-test.mjs && node frontend/tests/wails-bindings-test.mjs && npm --prefix frontend run build`

Expected: both Node smokes print their pass messages; Vite build succeeds.

- [ ] **Step 5: Commit and push the frontend bridge**

```bash
git add frontend/src/lib/plugin-host frontend/tests
git commit -m "feat(import): bridge sessions and progress to plugins"
git push origin main
```

### Task 6: Official plugin scaffold and neutral source graph

**Files:**
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/plugin.json`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/locales/en.json`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/locales/ru.json`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/package.json`
- Create: lockfile via `npm install`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/vite.config.js`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/index.js`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/model/source.js`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/model/graph.js`
- Test: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/model/source.test.js`

**Interfaces:**
- Consumes: Task 1 `api.imports` and Task 5 stylesheet loading.
- Produces: `loadAllEntries(api, session, onProgress)`, `detectCandidates(entries)`, `readCandidate(api, session, candidate, onProgress)`, and deterministic graph IDs for Tasks 7-9.

- [ ] **Step 1: Write failing pagination and candidate tests**

Test pagination until empty `nextCursor`, stable ordering, a full-install DokuWiki root at `wiki/html/data`, a direct `pages/` backup, a vault below one wrapper directory, multiple candidates requiring explicit choice, and no recognized source.

```js
expect(detectCandidates(entries)).toEqual([
  { id: 'dokuwiki:wiki/html/data', format: 'dokuwiki', root: 'wiki/html/data', label: 'DokuWiki — wiki/html/data' },
  { id: 'obsidian:Notes', format: 'obsidian', root: 'Notes', label: 'Obsidian — Notes' },
]);
```

- [ ] **Step 2: Run model tests and confirm red**

Run: `npm --prefix plugins/import/frontend test -- src/model/source.test.js`

Expected: FAIL because the plugin package and model do not exist.

- [ ] **Step 3: Create manifest, build package, registration, and graph model**

The manifest is exactly scoped as:

```json
{
  "schemaVersion": 1,
  "id": "verstak.import",
  "name": "Import",
  "version": "0.1.0",
  "apiVersion": "0.1.0",
  "localization": { "defaultLocale": "en", "locales": { "en": "locales/en.json", "ru": "locales/ru.json" } },
  "source": "official",
  "provides": ["verstak/import/v1"],
  "requires": ["verstak/core/import/v1"],
  "permissions": ["imports.readExternal", "imports.apply", "storage.namespace", "ui.register"],
  "frontend": { "entry": "frontend/dist/index.js", "style": "frontend/dist/style.css" },
  "contributes": { "settingsPanels": [{ "id": "verstak.import.settings", "title": "Import", "component": "ImportSettings" }] }
}
```

Use the existing sync plugin's Vite/Svelte IIFE configuration and register `ImportSettings`. Add Vitest and `lucide-svelte` to the frontend package. Graph node IDs are SHA-like deterministic strings produced from `format + ':' + normalizedSourcePath` by a small stable FNV-1a implementation; the graph stores entry IDs, paths, roles, links, warnings, and never full binary content.

- [ ] **Step 4: Run model tests, manifest checks, and build**

Run: `npm --prefix plugins/import/frontend test && npm --prefix plugins/import/frontend run build && ./scripts/check.sh`

Expected: model tests pass, the manifest validates against the adjacent SDK, localization keys match, and `dist/import` is created by the check/build flow.

- [ ] **Step 5: Commit and push the plugin foundation**

```bash
git add plugins/import scripts/check.sh
git commit -m "feat(import): scaffold official import plugin"
git push origin main
```

### Task 7: DokuWiki current-page adapter and Markdown conversion

**Files:**
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/dokuwiki/detect.js`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/dokuwiki/convert.js`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/dokuwiki/adapter.js`
- Create: synthetic fixtures under `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/test/fixtures/dokuwiki/`
- Test: `detect.test.js`, `convert.test.js`, and `adapter.test.js` beside the source files.

**Interfaces:**
- Consumes: Task 6 entries/graph and `api.imports.readText`.
- Produces: `detectDokuWikiCandidates(entries)` and `buildDokuWikiGraph(api, session, candidate, progress)` for the planner.

- [ ] **Step 1: Add failing current-data and conversion golden tests**

Fixtures include `pages/start.txt`, namespaces, `media/logo.png`, a sibling `attic/` revision, `meta/`, `cache/`, stock wiki pages identified by exact known content fingerprints, and custom pages under a `wiki:` namespace. Golden input covers six-level headings, bold/italic/underline/monospace, lists, quotes, tables, code/file blocks, internal/external/interwiki links, media dimensions, missing targets, anchors, and unsupported plugin syntax.

```js
const result = convertDokuWikiPage({
  pageId: 'project:start',
  text: '====== Project ======\n[[project:plan#next|Plan]] {{:media:logo.png?200x100|Logo}}\n<WRAP box>Keep me</WRAP>',
  resolvePage: () => '../Plan.md#next',
  resolveMedia: () => '../../Files/media/logo.png',
});
expect(result.markdown).toContain('# Project');
expect(result.markdown).toContain('[Plan](../Plan.md#next)');
expect(result.markdown).toContain('![Logo](../../Files/media/logo.png)');
expect(result.markdown).toContain('<WRAP box>Keep me</WRAP>');
expect(result.warnings).toEqual([expect.objectContaining({ code: 'dokuwiki-unsupported-syntax' })]);
```

- [ ] **Step 2: Run adapter tests and confirm red**

Run: `npm --prefix plugins/import/frontend test -- src/dokuwiki`

Expected: FAIL because the adapter files do not exist.

- [ ] **Step 3: Implement current-page discovery and deterministic conversion**

Only `pages/**/*.txt` becomes pages. Map page ID from the relative path without `.txt`, using `:` namespace separators; map `media/**` unchanged. Ignore `attic`, `media_attic`, `meta`, `cache`, `index`, `locks`, `tmp`, `log`, `sessions`, and configuration/code paths before graph construction. Build the repository-owned stock-page fingerprint table from the official DokuWiki `2025-05-14b "Librarian"` release's unchanged bundled pages and record the release name beside the hashes; exclude a page only on exact path-and-SHA-256 match, so modified or custom `wiki:` pages remain.

Convert block constructs line-by-line before inline markup. Protect code/file/nowiki spans with sentinels; convert headings, lists, quotes, horizontal rules, and tables; then resolve links/media against the final source graph. Preserve an unsupported plugin block verbatim and attach a warning to that page. Sanitize output filenames as readable Markdown names and resolve duplicate/case-fold collisions deterministically with the page-ID namespace in the proposed target path.

- [ ] **Step 4: Run DokuWiki tests and importer build**

Run: `npm --prefix plugins/import/frontend test -- src/dokuwiki && npm --prefix plugins/import/frontend run build`

Expected: all DokuWiki golden tests pass; Vite emits `dist/index.js` and `dist/style.css`.

- [ ] **Step 5: Commit and push the DokuWiki adapter**

```bash
git add plugins/import/frontend/src/dokuwiki plugins/import/frontend/test/fixtures/dokuwiki
git commit -m "feat(import): convert current DokuWiki content"
git push origin main
```

### Task 8: Obsidian adapter, links, embeds, and block anchors

**Files:**
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/obsidian/detect.js`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/obsidian/links.js`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/obsidian/adapter.js`
- Create: synthetic fixtures under `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/test/fixtures/obsidian/`
- Test: `detect.test.js`, `links.test.js`, and `adapter.test.js` beside sources.

**Interfaces:**
- Consumes: Task 6 graph.
- Produces: `detectObsidianCandidates(entries)` and `buildObsidianGraph(api, session, candidate, progress)` for Task 9.

- [ ] **Step 1: Add failing Obsidian resolution tests**

Cover `.obsidian/` exclusion, wrapper roots, frontmatter aliases, duplicate basenames, heading links, block IDs, note embeds, images, PDFs/files, relative attachments, nested archives as ordinary files, frontmatter/tasks/tags/callouts preservation, and ambiguous link warnings.

```js
const result = rewriteObsidianMarkdown({
  sourcePath: 'Projects/A/Readme.md',
  text: '---\ntags: [work]\n---\n- [ ] Task\n> [!note] Callout\n[[Plan#Next|plan]] ![[diagram.png]] ![[Quoted note]]\nBlock ^stable-id',
  index,
  mapping,
});
expect(result.markdown).toContain('- [ ] Task');
expect(result.markdown).toContain('> [!note] Callout');
expect(result.markdown).toContain('[plan](Plan.md#next)');
expect(result.markdown).toContain('![](../Files/diagram.png)');
expect(result.markdown).toContain('[Quoted note](Quoted%20note.md)');
expect(result.markdown).toContain('<a id="block-stable-id"></a>');
expect(result.warnings).toEqual([expect.objectContaining({ code: 'obsidian-note-embed-degraded' })]);
```

- [ ] **Step 2: Run Obsidian tests and confirm red**

Run: `npm --prefix plugins/import/frontend test -- src/obsidian`

Expected: FAIL because the adapter files do not exist.

- [ ] **Step 3: Implement vault detection and mapping-aware rewriting**

Detect roots by Markdown/content density and optional `.obsidian` marker, but remove `.obsidian/**` from entries before graph creation. Index canonical relative paths, basename buckets, frontmatter aliases, headings, and `^block-id` anchors. Resolution order is exact relative path, exact vault path, unique alias/basename; multiple valid destinations remain unresolved with a warning.

Keep YAML frontmatter and ordinary Markdown bytes except recognized Obsidian link/embed tokens and block-ID suffixes. Ordinary wikilinks become relative Markdown links; image/file embeds target `Files/`; note embeds become ordinary links plus `obsidian-note-embed-degraded`; `^id` creates `<a id="block-<slug>"></a>` immediately before the owning block. Never unpack a nested archive.

- [ ] **Step 4: Run Obsidian tests and importer build**

Run: `npm --prefix plugins/import/frontend test -- src/obsidian && npm --prefix plugins/import/frontend run build`

Expected: all Obsidian tests pass and the plugin bundle builds.

- [ ] **Step 5: Commit and push the Obsidian adapter**

```bash
git add plugins/import/frontend/src/obsidian plugins/import/frontend/test/fixtures/obsidian
git commit -m "feat(import): map current Obsidian vault content"
git push origin main
```

### Task 9: Adaptive editable plan

**Files:**
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/model/plan.js`
- Test: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/model/plan.test.js`

**Interfaces:**
- Consumes: Tasks 7-8 neutral graph nodes/edges.
- Produces: `proposePlan(graph, now)`, `editPlanNode(plan, nodeId, patch)`, `validateEditablePlan(plan)`, `serializeApplyPlan(plan, fingerprint)` for Task 10.

- [ ] **Step 1: Add failing deterministic planner/edit tests**

Test these exact rules: namespaces/top-level content branches become organizational candidates; coherent work branches become Deals; ordinary Markdown becomes notes; a branch with Markdown plus code/project markers (`package.json`, `go.mod`, `.gitignore`, `src/`) becomes a file subtree; loose root material is placed in one localized `Без папки`/`Unsorted` Deal; all content belongs to a Deal; repeated input produces byte-equal plans; changing type/name reruns validation/link mapping; skip counts are retained.

```js
const plan = proposePlan(graph, new Date('2026-07-23T04:30:00Z'));
expect(plan.runName).toBe('Obsidian — 2026-07-23 12-30-00');
expect(plan.nodes.find((node) => node.sourcePath === 'Projects/App')).toMatchObject({ kind: 'workspace', templateId: 'default' });
expect(plan.nodes.find((node) => node.sourcePath === 'Projects/App/src/main.js')).toMatchObject({ kind: 'file', targetSubpath: 'src/main.js' });
expect(validateEditablePlan(plan)).toEqual([]);
```

- [ ] **Step 2: Run planner tests and confirm red**

Run: `npm --prefix plugins/import/frontend test -- src/model/plan.test.js`

Expected: FAIL because `plan.js` does not exist.

- [ ] **Step 3: Implement simple deterministic heuristics and serializer**

Sort all graph input by normalized path. Use stable graph IDs as plan IDs. Structural nodes use `name`; notes/files use owning Deal `parentId` plus `targetSubpath`. Preserve source modified time. A note carries converted `text`; a file carries `sourceEntryId`. Reasons/confidence/warnings remain plugin-only UI metadata and are omitted by `serializeApplyPlan`.

Validation returns structured issues only for cycles/orphans, invalid names/subpaths, no Deal owner, duplicate case-fold target, missing entry/text, or unsupported type. Low confidence and conversion/link warnings do not block. Serializer emits exactly Task 1's `ImportPlan` with `schemaVersion: 1`.

- [ ] **Step 4: Run all importer model tests**

Run: `npm --prefix plugins/import/frontend test`

Expected: all detector, converter, resolver, graph, and planner tests pass.

- [ ] **Step 5: Commit and push the planner**

```bash
git add plugins/import/frontend/src/model
git commit -m "feat(import): propose editable import structures"
git push origin main
```

### Task 10: Four-step Verstak-styled import master

**Files:**
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/ImportSettings.svelte`
- Modify: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/frontend/src/index.js`
- Modify: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/locales/en.json`
- Modify: `/home/mirivlad/git/verstak2/verstak-official-plugins/plugins/import/locales/ru.json`
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/scripts/smoke-import-plugin.js`
- Modify: `/home/mirivlad/git/verstak2/verstak-official-plugins/scripts/check.sh`
- Test: component behavior through the smoke script.

**Interfaces:**
- Consumes: `api.imports`, Tasks 6-9 adapters/planner, `api.i18n`, and `api.storage.data` summary storage.
- Produces: settings component `ImportSettings` and final user workflow.

- [ ] **Step 1: Write a failing DOM smoke for all four steps**

Mount the built component in JSDOM-compatible minimal DOM used by existing smoke scripts. Stub two candidate roots, analysis progress, editable plan, apply progress, cancellation, and result. Assert source buttons, candidate selector, Analyze gating, step headings, sensitive-content warning, node type/name editing, invalid-plan gating, confirmation, cancellation confirmation, completion counts, and `Open imported` dispatch.

Use stable selectors:

```text
[data-import-step="source"]
[data-import-select-directory]
[data-import-select-archive]
[data-import-analyze]
[data-import-candidate]
[data-import-step="analysis"]
[data-import-step="structure"]
[data-import-tree]
[data-import-node-type]
[data-import-node-name]
[data-import-sensitive-warning]
[data-import-step="apply"]
[data-import-cancel]
[data-import-result]
```

- [ ] **Step 2: Run smoke and confirm red**

Run: `node scripts/smoke-import-plugin.js`

Expected: FAIL because `ImportSettings` and its built bundle do not exist.

- [ ] **Step 3: Implement the state machine and existing-design UI**

Use state values `source`, `analysis`, `structure`, `apply`; close the prior handle before selecting another. Analysis loads all entries, asks for a candidate when more than one exists, builds the selected graph, and proposes a plan without vault writes. It uses a local `AbortController`; apply subscribes to host progress and calls `cancel(handle)` only while `cancellable` is true.

The structure pane renders the full proposed tree and counts; the inspector edits `name` and `kind` among folder/Deal/note/file/skip, shows reason/confidence/warnings, and filters warning/low-confidence nodes. Always render one general localized warning saying imported material may contain credentials and should be reviewed, moved to Secrets, or deleted. Do not scan or label page content.

Use only the generic `--verstak-*` variables exposed by the settings host, `clamp` spacing, the host radius/control height, application select styling, and responsive stacking below 760px. Import `FolderOpen`, `Archive`, `Search`, `AlertTriangle`, `CheckCircle2`, `X`, and `ChevronRight` from the bundled `lucide-svelte` dependency; do not use emoji.

After success, store only `{format, completedAt, counts, warningCount, runPath}` in `api.storage.data.write('last-import', summary)`. The open action dispatches the existing `verstak:nav`/workspace-tree event using `runPath`; it never stores page text or filenames.

- [ ] **Step 4: Run plugin tests, smoke, design checks, and build**

Run: `npm --prefix plugins/import/frontend test && npm --prefix plugins/import/frontend run build && node scripts/smoke-import-plugin.js && ./scripts/check.sh && ./scripts/build.sh`

Expected: all unit/smoke/design/manifest/localization checks pass and `dist/import` contains manifest, locales, `frontend/dist/index.js`, and `style.css`.

- [ ] **Step 5: Commit and push the complete official plugin**

```bash
git add plugins/import scripts README.md README.ru.md dist/import
git commit -m "feat: add DokuWiki and Obsidian importer"
git push origin main
```

### Task 11: Desktop mocked E2E and supplied-backup smoke

**Files:**
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/src/lib/test/wails-mock.js`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/frontend/e2e/import-plugin.spec.js`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/internal/core/importservice/backup_smoke_test.go`
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/scripts/install-dev-plugins.sh`

**Interfaces:**
- Consumes: packaged `dist/import`, Task 4 methods, and the two local archives one directory above the desktop repo.
- Produces: automated cross-layer gates without committing personal backup data.

- [ ] **Step 1: Add failing Playwright wizard scenarios**

Mock an Obsidian source and a DokuWiki source with synthetic filenames/content. Test settings launch from the Import card, both source buttons, multiple-root selection, analysis cancellation, plan edit, generic sensitive warning without singled-out files, successful import, repeated unique run, and disable/re-enable cleanup.

```js
test('imports a reviewed DokuWiki plan into an isolated run', async ({ page }) => {
  await openPluginManager(page);
  const card = page.locator('.plugin-card').filter({ hasText: 'verstak.import' });
  await card.locator('button.btn-settings').click();
  await page.locator('[data-import-select-archive]').click();
  await page.locator('[data-import-analyze]').click();
  await expect(page.locator('[data-import-step="structure"]')).toBeVisible();
  await expect(page.locator('[data-import-sensitive-warning]')).toBeVisible();
  await page.locator('[data-import-tree] button').first().click();
  await page.locator('[data-import-node-name]').fill('База знаний');
  await page.getByRole('button', { name: /Import|Импортировать/ }).click();
  await expect(page.locator('[data-import-result]')).toContainText('Импортировано/DokuWiki');
});
```

- [ ] **Step 2: Run focused E2E and confirm red**

Run: `npm --prefix frontend run test:e2e -- import-plugin.spec.js`

Expected: FAIL because the mock has no import plugin or source methods.

- [ ] **Step 3: Extend the mock and add privacy-safe real-backup smoke**

Bundle the built official importer exactly like existing mocked official plugins. Add mock methods for the seven Wails calls and expose only fixture entries/text. The Go backup smoke runs only when `VERSTAK_IMPORT_BACKUP_SMOKE=1`; it opens `../wiki.tar.gz` and `../Obsidian.tar.gz`, asserts format-safe inventory facts and counts without logging names/content, runs adapters through a Node smoke command that returns aggregate JSON, applies to `t.TempDir()`, asserts current DokuWiki pages and Obsidian `.obsidian` omissions, and repeats one import to confirm suffixing.

Expected safe assertions for the supplied files are: DokuWiki archive has 126 current `data/pages/**/*.txt` page entries and 27 current `data/media/**` files; Obsidian archive has 147 Markdown entries, 18 non-service asset files, and 79 `.obsidian/**` entries excluded from graph construction. If archive inventories changed, stop and report the mismatch rather than weakening the assertion.

- [ ] **Step 4: Run focused E2E, backup smoke, and desktop full tests**

Run: `npm --prefix frontend run test:e2e -- import-plugin.spec.js && VERSTAK_IMPORT_BACKUP_SMOKE=1 go test ./internal/core/importservice -run TestSuppliedBackups -count=1 && ./scripts/test.sh && ./scripts/check.sh && ./scripts/build.sh`

Expected: Playwright passes all importer scenarios; the backup smoke prints only aggregate counts; Go/full frontend checks and Wails build pass.

- [ ] **Step 5: Perform the real desktop GUI gate**

Install development plugins with `./scripts/install-dev-plugins.sh`, launch `build/bin/verstak-desktop --debug` in an available graphical session, create a disposable vault, open Plugin Manager → Import → Settings, select each supplied archive with the native dialog, inspect and edit a proposal, import both, open representative results, repeat one import, and disable/re-enable the plugin. Inspect the debug log for lifecycle/API errors; do not paste page bodies or sensitive names into the report. If no graphical session/AT-SPI automation is available, report exactly `GUI behavior was not verified` and do not substitute Playwright for this gate.

- [ ] **Step 6: Commit and push desktop integration**

```bash
git add frontend/src/lib/test/wails-mock.js frontend/e2e/import-plugin.spec.js internal/core/importservice/backup_smoke_test.go scripts/install-dev-plugins.sh
git commit -m "test(import): cover plugin workflow and backup smoke"
git push origin main
```

### Task 12: Platform and user documentation

**Files:**
- Modify: `/home/mirivlad/git/verstak2/verstak-docs/04_Plugin_System.md`
- Modify: `/home/mirivlad/git/verstak2/verstak-docs/05_Official_Plugins.md`
- Modify: `/home/mirivlad/git/verstak2/verstak-docs/07_Full_Implementation_Roadmap.md`
- Modify: `/home/mirivlad/git/verstak2/verstak-docs/README.md`
- Modify: `/home/mirivlad/git/verstak2/verstak-docs/README.ru.md`
- Modify: `/home/mirivlad/git/verstak2/verstak-desktop/docs/PLUGIN_RUNTIME.md`

**Interfaces:**
- Consumes: final names/limits/behavior from Tasks 1-11.
- Produces: current platform contract and user guidance.

- [ ] **Step 1: Update the docs with exact shipped behavior**

Document the seven `api.imports` methods, opaque session ownership, both permissions, supported archives, fixed limits, progress/cancel boundary, staged transaction/recovery, and stylesheet host behavior in `04_Plugin_System.md` and `PLUGIN_RUNTIME.md`. Add an official-plugin section with the four steps, DokuWiki conversion, Obsidian rewrite rules, `Импортировано`, repeated-import isolation, service-data omissions, and the general credentials warning. Mark only implemented roadmap items complete and replace the old statement that chunked large-file import is entirely deferred.

- [ ] **Step 2: Run documentation checks and inspect the diff**

Run: `./scripts/build.sh` in `verstak-docs`, then `git diff --check` in both docs and desktop repositories.

Expected: documentation build succeeds and both diff checks are silent.

- [ ] **Step 3: Commit and push docs repositories separately**

```bash
cd /home/mirivlad/git/verstak2/verstak-docs
git add 04_Plugin_System.md 05_Official_Plugins.md 07_Full_Implementation_Roadmap.md README.md README.ru.md
git commit -m "docs: add DokuWiki and Obsidian import guide"
git push origin main

cd /home/mirivlad/git/verstak2/verstak-desktop
git add docs/PLUGIN_RUNTIME.md
git commit -m "docs: describe generic import runtime"
git push origin main
```

### Task 13: Final verification, release notes, and GitHub publication

**Files:**
- Create: `/home/mirivlad/git/verstak2/verstak-official-plugins/release-notes/v0.1.0-beta.20260723.md`
- Create: `/home/mirivlad/git/verstak2/verstak-desktop/release-notes/v0.1.0-beta.20260723.md`

**Interfaces:**
- Consumes: clean pushed main branches from every earlier task.
- Produces: matching official-plugin and desktop prereleases with checksums and update artifacts.

- [ ] **Step 1: Run repository-wide verification from clean trees**

```bash
cd /home/mirivlad/git/verstak2/verstak-sdk && npm run lint && npm test && npm run build
cd /home/mirivlad/git/verstak2/verstak-official-plugins && ./scripts/check.sh && ./scripts/build.sh
cd /home/mirivlad/git/verstak2/verstak-desktop && ./scripts/check.sh && ./scripts/test.sh && npm --prefix frontend run test:e2e && ./scripts/build.sh
cd /home/mirivlad/git/verstak2/verstak-docs && ./scripts/build.sh
```

Expected: every command exits zero. Record exact test counts and any explicitly unavailable real-GUI gate.

- [ ] **Step 2: Self-review sensitive logging and repository cleanliness**

Run `rg -n "wiki\.tar\.gz|Obsidian\.tar\.gz|readText\(|page body|password"` over changed tracked files, inspect every match, and ensure tests/logging contain only fixture paths or aggregate checks. Run `git status --short` in all four repositories; expected output is empty before release-note creation.

- [ ] **Step 3: Add human-readable release notes and push them**

Both notes files must list the importer workflow, safety boundary, supported formats, DokuWiki conversion, Obsidian link handling, isolated `Импортировано` output, general sensitive-content warning, and known limitation that unsupported DokuWiki plugin syntax remains visible with a warning. Commit/push each repository:

```bash
git add release-notes/v0.1.0-beta.20260723.md
git commit -m "docs: add v0.1.0 beta import release notes"
git push origin main
```

- [ ] **Step 4: Confirm GitHub authentication and publish official plugins first**

Run: `gh auth status`

Expected: authenticated account `mirivlad`. If it still reports the currently observed invalid token, stop publication and request re-authentication; do not create local-only tags.

Run: `./scripts/publish-github-release.sh v0.1.0-beta.20260723` in `verstak-official-plugins`.

Expected: release URL is printed and assets include Linux/Windows plugin archives plus `SHA256SUMS`.

- [ ] **Step 5: Install the published plugins into the desktop package and publish desktop**

Update/install the just-published official-plugin artifacts through the existing desktop release workflow, then run `./scripts/publish-github-release.sh v0.1.0-beta.20260723` in `verstak-desktop`.

Expected: release URL is printed and assets include `.deb`, `.AppImage`, Windows portable `.zip`, and `SHA256SUMS`.

- [ ] **Step 6: Verify release assets and updater-visible metadata**

Use `gh release view v0.1.0-beta.20260723 --repo mirivlad/verstak-official-plugins --json url,assets,isPrerelease` and the same command for `mirivlad/verstak`. Download checksums and verify each asset locally with `sha256sum -c`. Start the previous desktop beta in a disposable profile, invoke its update check, and assert it offers `v0.1.0-beta.20260723`; install or unpack the new build and confirm the Import plugin card and settings master are present.

- [ ] **Step 7: Final clean/pushed audit**

For each repository run `git status --short`, `git rev-parse HEAD`, and `git rev-parse origin/main`; expected status is empty and both revisions match. Report changed repositories/files, exact verification labels, release URLs, asset/checksum status, GUI status, and any remaining uncertainty.
