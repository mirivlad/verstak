# Milestone 6a Files Core Service Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` or `superpowers:executing-plans` to
> implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for
> tracking.

**Goal:** Add a safe backend Files core service and plugin bridge for
vault-relative text file operations.

**Architecture:** Files Core is a backend service under `internal/core/files`
that accepts vault-relative paths, enforces reserved path policy, and performs
atomic text writes. The Wails API exposes plugin-scoped methods guarded by plugin
state and `files.*` permissions; SDK types describe the bridge shape.

**Tech Stack:** Go backend services/tests, Wails-bound API methods, TypeScript
SDK type definitions, existing Playwright/Vitest checks.

---

## Implementation Status

Milestone 6a is implemented.

Actual backend package:

- `internal/core/files/types.go`
- `internal/core/files/path_policy.go`
- `internal/core/files/service.go`
- `internal/core/files/*_test.go`

Actual plugin-scoped Wails methods:

```go
func (a *App) ListVaultFiles(pluginID string, relativeDir string) ([]files.FileEntry, string)
func (a *App) GetVaultFileMetadata(pluginID string, relativePath string) (files.FileMetadata, string)
func (a *App) ReadVaultTextFile(pluginID string, relativePath string) (string, string)
func (a *App) WriteVaultTextFile(pluginID string, relativePath string, content string, options files.WriteOptions) string
func (a *App) CreateVaultFolder(pluginID string, relativePath string) string
func (a *App) MoveVaultPath(pluginID string, fromRelativePath string, toRelativePath string, options files.MoveOptions) string
func (a *App) TrashVaultPath(pluginID string, relativePath string) (files.TrashResult, string)
```

Actual bundled frontend API:

- `api.files.list(relativeDir)`
- `api.files.metadata(relativePath)`
- `api.files.readText(relativePath)`
- `api.files.writeText(relativePath, content, options)`
- `api.files.createFolder(relativePath)`
- `api.files.move(fromRelativePath, toRelativePath, options)`
- `api.files.trash(relativePath)`

Implemented limits:

- canonical vault-relative slash paths only;
- backslashes, POSIX absolute paths, Windows drive paths, UNC paths, traversal,
  null bytes, and empty file paths are rejected;
- `.verstak/` is reserved case-insensitively and hidden from public Files API;
- metadata may report symlinks, but list-through-symlink and
  read/write/move/trash through symlink are forbidden;
- text read/write only, with `readText` limited to UTF-8 files up to 2 MB;
- trash uses `.verstak/trash/files/<trashId>/...` with restore metadata, but
  restore itself is deferred;
- binary streaming, watcher, external editor, Files UI, Notes service, sidecar,
  sandbox/security isolation deferred.

---

## Scope

Implement:

- Backend Files service.
- Safe vault-relative path handling.
- Reserved `.verstak/` policy.
- List files.
- Read/write text files.
- Create folder.
- Move path.
- Trash path.
- Atomic writes.
- Backend tests.
- SDK bridge shape draft.

Do not implement:

- Full Notes plugin.
- Notes UI.
- Sync.
- Watcher.
- Binary streaming.
- External editor integration.
- Sidecar/security isolation.

## Canonical Policy

All public Files API methods use canonical vault-relative slash paths.

Rejected inputs:

- absolute paths;
- backslashes;
- Windows drive paths and UNC/network paths;
- paths containing `..` after normalization;
- null bytes;
- empty paths where a file path is required;
- access to `.verstak/` through the public plugin Files API, including
  `.Verstak` case variants.

Delete behavior:

- `TrashVaultPath` moves files/folders into `.verstak/trash`.
- Trash metadata includes `originalPath`, `deletedAt`, `originalType`,
  `trashId`, and `basename`.
- Permanent delete is out of scope.
- Restore is out of scope.

Write behavior:

- Text writes use a temporary file in the target directory and rename into place.
- Existing files are overwritten only when the method explicitly allows overwrite.
- Parent directory must exist unless the method explicitly creates it.

Binary behavior:

- Binary files can appear in list/metadata results.
- Binary read/write streaming is out of scope.

## Public Backend Shape

Implemented plugin-scoped Wails methods:

```go
func (a *App) ListVaultFiles(pluginID string, relativeDir string) ([]files.FileEntry, string)
func (a *App) GetVaultFileMetadata(pluginID string, relativePath string) (files.FileMetadata, string)
func (a *App) ReadVaultTextFile(pluginID string, relativePath string) (string, string)
func (a *App) WriteVaultTextFile(pluginID string, relativePath string, content string, options files.WriteOptions) string
func (a *App) CreateVaultFolder(pluginID string, relativePath string) string
func (a *App) MoveVaultPath(pluginID string, fromRelativePath string, toRelativePath string, options files.MoveOptions) string
func (a *App) TrashVaultPath(pluginID string, relativePath string) (files.TrashResult, string)
```

Permission mapping:

- `ListVaultFiles`, `GetVaultFileMetadata`, `ReadVaultTextFile`: `files.read`.
- `WriteVaultTextFile`, `CreateVaultFolder`, `MoveVaultPath`: `files.write`.
- `TrashVaultPath`: `files.delete`.

## Data Types

Create `internal/core/files/types.go`:

```go
package files

type EntryKind string

const (
	KindFile      EntryKind = "file"
	KindDirectory EntryKind = "directory"
)

type Entry struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Kind         EntryKind `json:"kind"`
	Size         int64     `json:"size"`
	ModifiedAt   string    `json:"modifiedAt"`
	IsText       bool      `json:"isText"`
	IsBinary     bool      `json:"isBinary"`
	IsHidden     bool      `json:"isHidden"`
	IsReserved   bool      `json:"isReserved"`
}
```

## Task 1: Path Policy

**Files:**

- Create: `internal/core/files/path.go`
- Create: `internal/core/files/path_test.go`

- [ ] Add `NormalizeVaultRelativePath(relative string) (string, error)`.
- [ ] Reject absolute paths, null bytes, `..`, and empty file paths.
- [ ] Preserve path case, including canonical `Notes`.
- [ ] Add `IsReservedPath(relative string) bool` returning true for `.verstak`
      and `.verstak/...`.
- [ ] Add tests:
      `TestNormalizeRejectsAbsolutePath`,
      `TestNormalizeRejectsTraversal`,
      `TestNormalizeRejectsNullByte`,
      `TestNormalizePreservesCase`,
      `TestReservedPathPolicy`.
- [ ] Run:

```bash
go test ./internal/core/files
```

Expected: all `internal/core/files` tests pass.

## Task 2: Files Service

**Files:**

- Create: `internal/core/files/service.go`
- Create/modify: `internal/core/files/service_test.go`

- [ ] Define `Service` with a vault dependency that can return the current vault
      root and status.
- [ ] Implement `List(relativeDir string) ([]Entry, error)`.
- [ ] Implement `Metadata(relativePath string) (Entry, error)`.
- [ ] Implement `ReadText(relativePath string) (string, error)`.
- [ ] Implement `WriteText(relativePath, content string, overwrite bool) (Entry, error)`.
- [ ] Implement `Mkdir(relativePath string) (Entry, error)`.
- [ ] Implement `Move(fromRelativePath, toRelativePath string, overwrite bool) (Entry, error)`.
- [ ] Implement `Trash(relativePath string) (Entry, error)`.
- [ ] Use the shared path policy for every public method.
- [ ] Block `.verstak` paths in every public method.
- [ ] Add tests for closed vault, list, metadata, text read/write, mkdir, move,
      trash, overwrite false conflict, overwrite true replace, and reserved path
      rejection.
- [ ] Run:

```bash
go test ./internal/core/files
```

Expected: all `internal/core/files` tests pass.

## Task 3: Atomic Writes

**Files:**

- Modify: `internal/core/files/service.go`
- Modify: `internal/core/files/service_test.go`

- [ ] Write text content to a temp file in the target directory.
- [ ] Rename the temp file into the final path only after successful write.
- [ ] Remove temp file on write failure.
- [ ] Add test `TestWriteTextIsAtomicOnFailure` using a controlled failing path
      or permission-denied directory.
- [ ] Add test `TestWriteTextDoesNotLeaveTempFile`.
- [ ] Run:

```bash
go test ./internal/core/files
```

Expected: all `internal/core/files` tests pass.

## Task 4: Permissions And Capabilities

**Files:**

- Modify: `internal/core/permissions/registry.go`
- Modify: `main.go`
- Modify: `internal/api/app_test.go`

- [ ] Register permissions: `files.read`, `files.write`, `files.delete`.
- [ ] Register core capability `verstak/core/files/v1` when vault services are
      initialized.
- [ ] Add API guard tests proving each Files bridge method rejects plugins that
      are missing the required permission.
- [ ] Run:

```bash
go test ./internal/core/permissions ./internal/api
```

Expected: permission registry and API tests pass.

## Task 5: Wails API Bridge

**Files:**

- Modify: `internal/api/app.go`
- Modify: `internal/api/app_test.go`
- Modify after Wails generation or by hand if generation is unavailable:
  `frontend/wailsjs/go/api/App.d.ts`
- Modify after Wails generation or by hand if generation is unavailable:
  `frontend/wailsjs/go/api/App.js`

- [ ] Add `files.Service` to `api.App`.
- [ ] Add plugin-scoped methods listed in "Public Backend Shape".
- [ ] Use `requirePluginAccess(pluginID, permission)` for every method.
- [ ] Return readable errors for closed vault, missing file, reserved path,
      conflict, and missing permission.
- [ ] Add tests for successful read/write/list/mkdir/move/trash through `App`.
- [ ] Run:

```bash
go test ./internal/api
```

Expected: API tests pass.

## Task 6: Frontend Plugin API Draft

**Files:**

- Modify: `frontend/src/lib/plugin-host/VerstakPluginAPI.js`
- Modify: `frontend/src/lib/test/wails-mock.js`
- Add/modify focused frontend tests under `frontend/e2e/` only if existing test
  coverage cannot validate the shape outside Playwright.

- [ ] Add `api.files.list(relativeDir)`.
- [ ] Add `api.files.metadata(relativePath)`.
- [ ] Add `api.files.readText(relativePath)`.
- [ ] Add `api.files.writeText(relativePath, content, options)`.
- [ ] Add `api.files.mkdir(relativePath)`.
- [ ] Add `api.files.move(fromRelativePath, toRelativePath, options)`.
- [ ] Add `api.files.trash(relativePath)`.
- [ ] Keep all calls plugin-scoped; plugin code must not pass `pluginId`.
- [ ] Mock readable errors for reserved path and missing permission.
- [ ] Run:

```bash
cd frontend
npm run build
```

Expected: frontend build passes.

## Task 7: SDK Bridge Shape Draft

**Files:**

- Modify: `../verstak-sdk/src/plugin-api.ts`
- Modify: `../verstak-sdk/src/test-utils.ts`
- Modify: `../verstak-sdk/src/plugin-api.test.ts`

- [ ] Add `files` API TypeScript interfaces matching the frontend API names.
- [ ] Add mock Files API methods in `createMockPluginAPI`.
- [ ] Add contract tests for API shape, text write/read, reserved path error, and
      trash result shape.
- [ ] Run:

```bash
cd ../verstak-sdk
./scripts/check.sh
./scripts/build.sh
./scripts/test.sh
```

Expected: SDK check, build, and tests pass.

## Task 8: Documentation

**Files:**

- Modify: `docs/PLUGIN_RUNTIME.md`
- Modify: `docs/NOTES_FILES_PLUGIN_PLAN.md`

- [ ] Document Files Core API as functional for Milestone 6a.
- [ ] Keep Notes API documented as planned until Milestone 6b or later.
- [ ] Document `.verstak` reserved path policy.
- [ ] Document slash-only path policy, Windows/UNC rejection, and symlink policy.
- [ ] Document text-only write support and deferred binary streaming.

## Task 9: Final Verification

- [ ] Run desktop backend tests:

```bash
cd verstak-desktop
go test ./...
```

- [ ] Run desktop frontend build:

```bash
cd verstak-desktop/frontend
npm run build
```

- [ ] Run desktop e2e:

```bash
cd verstak-desktop/frontend
npm run test:e2e -- --reporter=list
```

- [ ] Run official plugins checks:

```bash
cd verstak-official-plugins
./scripts/check.sh
./scripts/build.sh
```

- [ ] Run SDK checks:

```bash
cd verstak-sdk
./scripts/check.sh
./scripts/build.sh
./scripts/test.sh
```

Expected: all commands exit 0. Existing Svelte unused CSS warnings are acceptable
only if they remain warnings and do not fail the build.
