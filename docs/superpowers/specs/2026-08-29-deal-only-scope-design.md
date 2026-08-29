# Deal-only Resource Scope Architecture and Migration

**Status:** Accepted design for the post-v0.1.6 foundation refactor

**Date:** 2026-08-29

## 1. Goal

Verstak has one universal runtime container: a Deal. The stable Deal UUID is
the only resource scope used by Core, plugins, storage, navigation, and sync.

```text
DealScope {
  workspaceId: UUID
}
```

Projects are not nested containers. A Deal can opt into Project Meta, which
adds project-oriented metadata and makes the Deal visible in the global
portfolio. Milestones, Git, Tasks, Notes, Files, Activity, Journal, and
Secrets remain independent Deal plugins.

This design intentionally replaces the v0.1.5/v0.1.6
`Deal -> Project -> project-scoped resources` model. It does not preserve that
model as a supported compatibility mode.

## 2. Non-negotiable invariants

1. `workspaceId` is the authoritative identity and runtime scope for a Deal.
2. A filesystem path is a mutable address. It never establishes ownership.
3. Resource records, requests, events, open contexts, and navigation requests
   do not use `projectId` as a scope axis.
4. Provider plugins own their resources. Project Meta does not copy or proxy
   Notes, Files, Todo, Activity, or Milestones data.
5. Core does not import official plugins or know their IDs, names, folders, or
   private configuration.
6. Templates are creation recipes. After creation, a Deal is governed by its
   own metadata and enabled tools, not by a live template.
7. Legacy Projects data may be backed up and consumed by one migration. A
   successful new runtime never reads it.
8. Git credentials are never stored in Git plugin plaintext data or exposed to
   frontend code when backend-side resolution is possible.
9. Repository checkouts and Git state are device-local and excluded from
   ordinary Verstak Sync.
10. No release or tag is created as part of this refactor.

## 3. Canonical Deal registry and metadata

`internal/core/workspacetree.Service` becomes the sole Deal registry. It owns
tree discovery, stable identity, current Deal selection, creation, update,
move, trash, restore, import publication, and sync application.

The legacy `internal/core/workspace.Manager` may exist only as a short-lived
foundation adapter while callers move. It must not remain as a second scanner,
template registry, metadata authority, or current-selection store.

### 3.1 Canonical metadata location

Deal metadata is stored by UUID:

```text
<vault>/.verstak/workspaces/uuid-<workspaceId>.json
```

Path-keyed metadata remains migration input only. The canonical reader checks
UUID metadata first. A legacy path-keyed file may be imported once when no
canonical record exists; it never shadows a canonical record.

### 3.2 Metadata schema v2

```json
{
  "schemaVersion": 2,
  "workspaceId": "00000000-0000-4000-8000-000000000000",
  "workspaceName": "Deal name",
  "workspaceTools": ["verstak.notes", "verstak.files"],
  "toolConfig": {
    "verstak.example": { "presetVersion": 1 }
  },
  "createdFromTemplate": {
    "templateId": "project",
    "templateName": "Project",
    "templateVersion": 1,
    "appliedAt": "2026-08-29T00:00:00Z",
    "recipeDigest": "sha256:..."
  },
  "updatedAt": "2026-08-29T00:00:00Z"
}
```

Rules:

- `workspaceId` must match the Deal marker and filename.
- `workspaceTools` is the definitive enabled-tool snapshot.
- `toolConfig` is namespaced by plugin ID and contains only validated,
  JSON-compatible template-time configuration.
- `createdFromTemplate` is immutable provenance. It is not consulted when
  deciding which tools run.
- Unknown tool IDs and unknown `toolConfig` namespaces are preserved so a
  temporarily unavailable plugin does not lose data.
- Writes are atomic and metadata publication is mandatory for new Deals.

### 3.3 Creation transaction

Core accepts a complete, validated recipe snapshot rather than a template ID.
It stages the Deal marker, initial folders/files, metadata, and tree-order
change, then publishes them as one recoverable operation. Failure before
publication leaves no visible Deal. Startup recovery either completes or
rolls back a journaled publication.

Core validates generic limits, paths, JSON sizes, plugin installation, and
configuration ownership. It does not interpret business-plugin fields.

## 4. Public Deal-only contracts

The SDK exposes an explicit scope type:

```ts
export type WorkspaceId = string;

export interface DealScope {
  kind: 'deal';
  workspaceId: WorkspaceId;
}

export interface DealRef {
  workspaceId: WorkspaceId;
  workspaceRootPath?: string;
}
```

`workspaceRootPath` may be returned as an address/display hint. Requests that
own, mutate, or enumerate Deal data require `DealScope`.

Notes, Files, Todo, and other provider capability versions introduced by this
refactor use request/result schemas that contain `scope: DealScope` and no
`projectId`. Existing v1 operations may be readable during the migration
commit sequence, but official consumers move to the new contracts before v1
project-scoped behavior is removed. The completed runtime does not route or
store Project scope.

Navigation requires a `workspaceId`. Tool-local request state may select a tab
or item inside that tool, but cannot override resource scope. Search,
Overview, open-resource contexts, and lifecycle events carry stable Deal
identity consistently.

Manifest validation checks supported host API versions, declared capability
operations, and generic operation envelopes. SDK source, schemas, generated
declarations, Desktop structs, official manifests, mocks, and contract tests
must agree mechanically.

## 5. One-shot legacy migration

The migration protects useful provider-owned data and discards the mistaken
nested runtime model.

### 5.1 Inputs

The runner audits:

- legacy path-keyed and unversioned Deal metadata;
- `projects:global` records;
- Notes `notes:projectScopes` bindings;
- Files `files:projectScopes` bindings;
- Todo records containing `projectId`;
- project-owned metadata, milestones, links, and history;
- sync snapshots and plugin-record state that mention the old sets.

### 5.2 Backup

Before mutation, the runner writes an immutable backup directory:

```text
<vault>/.verstak/migrations/deal-only-v1/backups/<timestamp>/
```

The backup contains exact source bytes, a manifest of paths and SHA-256
digests, the app/build version, and the planned mutations. Backup publication
is atomic. Existing backup directories are never overwritten.

### 5.3 Migration behavior

1. Materialize canonical metadata v2 for every valid Deal UUID.
2. Convert historical template-derived feature/tool state into definitive
   `workspaceTools`; preserve unknown IDs and provenance.
3. Remove `projectId` from Todo records without changing task identity or
   other fields.
4. Delete Notes and Files project-scope side tables after backing them up.
   Physical note/file paths and bytes are not moved or rewritten.
5. Preserve Activity/provider data unchanged except for removal of an explicit
   Project scope field if one exists.
6. If exactly one legacy Project maps unambiguously to a Deal, copy useful
   status, priority, description, dates, and tags into that Deal's Project Meta
   record.
7. If multiple legacy Projects map to one Deal, do not create Deals and do not
   choose one record as authoritative. Keep Project Meta empty/default for
   that Deal and preserve all source records in the backup.
8. Copy legacy milestones into the new Milestones store when workspace
   identity is unambiguous. For a multi-project Deal, milestones may all map to
   the same Deal; retain their IDs where valid and record the source project
   name/ID as inert provenance to avoid title-based guessing.
9. Keep links/history only in the inert backup unless the target plugin has a
   direct, lossless field for them.
10. Remove active sync declarations for old Projects and project-scope record
    sets after the new provider records are committed.

The runner never infers membership from a filename, folder name, title, or
display name when an ID is available.

### 5.4 Ledger and failure behavior

The migration has a small ledger with `prepared`, `applied`, and `verified`
states plus source/target digests. It is resumable only for the current
one-shot operation; it is not a permanent compatibility subsystem.

- A failed preflight changes nothing.
- A failed apply restores backed-up bytes and leaves the ledger retryable.
- Verification proves provider record counts/IDs and file bytes before the
  ledger becomes `verified`.
- After `verified`, normal runtime ignores all legacy Projects data and old
  project-scope sets.
- Re-running a verified migration is a no-op.

Foundation commits may add and test the runner before every target store is
available, but production activation is gated by one complete target-schema
bundle version. No intermediate pushed build starts a partial migration. A
later integration commit enables the one-shot runner only after Project Meta,
Milestones, and Deal-only providers can accept every enabled transform.

## 6. Plugin ownership

### 6.1 Project Meta and portfolio

The existing `verstak.projects` plugin ID is retained to avoid an unnecessary
enabled-tool identity migration, but its implementation and display role
become Project Meta. It stores at most one record per `workspaceId`:

```text
workspaceId, status, priority, description, startDate?, dueDate?, tags,
createdAt, updatedAt
```

The Deal workspace item edits this record. The global portfolio enumerates
Deals with Project Meta enabled, joins their current Deal names/paths by UUID,
and opens the Deal through public navigation. It does not contain provider
tabs or nested resource views.

### 6.2 Milestones

`verstak.milestones` is independent of Project Meta. Each milestone has a
stable ID, `workspaceId`, title, status, optional due date, timestamps, and
optional inert migration provenance. It works in any Deal and exposes its own
Deal UI and capability operations.

### 6.3 Provider cleanup

- Notes and Files use current Deal UUID for list/create/open operations.
- Todo persists `workspaceId` and no `projectId`.
- Activity accepts/publishes Deal identity only.
- Provider UI may use current paths for filesystem access after resolving the
  UUID through Core, but persisted ownership remains UUID-based.
- Disabling Project Meta or Milestones never deletes provider data.

## 7. Templates

`verstak.templates` is a first-party plugin with persisted CRUD:

```text
id, name, description, version, order, workspaceTools,
initialFolders, initialFiles, pluginPresets, createdAt, updatedAt, isDefault
```

The plugin supports list, create, edit, duplicate, delete, preview, create Deal,
and reseed/restore defaults. Default IDs are stable. Reseeding is idempotent:
it restores missing defaults and updates defaults only through an explicit
user action; it does not overwrite user templates silently.

Plugins may contribute a generic template-preset descriptor and validator.
Templates discovers these contributions through the public registry and
stores namespaced JSON. It never imports private Git, Milestones, or Project
Meta code.

General, Project, Writing, Admin, and Minimal are seed records. The Project
seed enables Project Meta, Git, Todo, Milestones, Notes, Files, Activity,
Journal, and Secrets when available. Missing optional plugins are shown before
creation and never replaced with hidden Core defaults.

## 8. Git, Secrets, and device-local checkouts

`verstak.git` owns repository descriptors keyed by `workspaceId` and stable
repository ID. A Deal can own zero or many descriptors.

Synced descriptor fields are limited to display name, sanitized remote URL,
preferred branch, relative checkout name, and a logical credential-slot ID.
Raw credentials, local secret IDs, absolute paths, HEAD, dirty state, and
process state are excluded.

Each device maps the logical credential slot to a local Secrets record. Core
resolves credentials backend-side for a single bounded Git operation. The
frontend receives only success/error/status data and never plaintext.

Git operations use argument arrays with validated repository roots, bounded
environment/output, cancellation, timeouts, and no shell interpolation. V1
supports add existing repository, clone, status, current branch, changed and
untracked counts/list, ahead/behind when available, recent commits, fetch,
fast-forward pull, push, open directory, and terminal only when a safe platform
launcher exists. Commit editing, merge UI, and advanced history are excluded.

Checkout layout defaults to:

```text
<Deal>/Repositories/<repository-name>/
```

The entire managed `Repositories` subtree is device-local for Sync purposes,
not merely `.git`. This avoids competing Git/Sync authorities over working
trees.

### 8.1 Sync transition

Before activating the exclusion, migration inventories existing snapshot and
remote operations under managed repository roots. It rebases the local
snapshot without emitting delete operations. Pull application rejects or
ignores excluded checkout paths while still advancing the sync cursor. Old
clients may continue uploading such paths, so the new client enforces the
boundary on both scan and apply.

Generic `plugin-record` operations are sufficient for sanitized descriptors;
no Sync Server protocol change is required. A server change is justified only
if later work adds encrypted secret synchronization or operation pruning.

## 9. Error handling and observability

- Schema/version mismatches fail closed with actionable diagnostics.
- Migration, Deal creation, and Git operations write structured logs without
  secrets.
- A migration report lists backed-up inputs, transformed record counts,
  preserved file hashes, skipped best-effort metadata, and verification state.
- Plugin absence is a degraded state, never data deletion.
- Git clone/fetch/pull/push errors include sanitized command context and exit
  status; remote URLs are scrubbed before logs or sync.

## 10. Verification and release boundary

Required automated evidence includes:

- canonical v2 metadata and UUID-first read/write behavior;
- path-keyed migration cannot shadow canonical metadata;
- migration backup byte/digest integrity and rollback;
- provider IDs/counts and Notes/Files bytes preserved while `projectId` and
  scope tables disappear;
- new runtime never reads legacy Projects records after verification;
- Project Meta portfolio opens the correct Deal UUID;
- Milestones works without Project Meta;
- Templates CRUD, duplicate/delete, seed restore, preview, and Deal creation;
- Core creation consumes a recipe snapshot without plugin-specific knowledge;
- multiple Git repositories per Deal;
- no credential plaintext in frontend, Git settings, logs, or sync payloads;
- repository checkout trees are absent from scan/push and ignored on pull;
- new-device descriptors render as not cloned and can be cloned;
- cross-repository SDK/schema/manifest/mock contracts;
- Linux Playwright E2E and the existing real Wails/WebKitGTK GUI smoke.

Each bounded slice runs focused tests, then its repository's normal checks.
Cross-repository contract changes are pushed in dependency order and followed
by the full `scripts/verstak-check.sh` gate against the combined pushed state.

No release/tag is published until migration, E2E, GUI, packaging, and owner
validation are separately requested and complete.

## 11. Bounded implementation slices

1. Architecture/migration design and implementation plan.
2. Canonical versioned Deal metadata and registry consolidation.
3. One-shot backup/migration framework and fixtures.
4. SDK/Core Deal-only contracts and version enforcement.
5. Provider migration and removal of project scope.
6. Project Meta and portfolio.
7. Milestones.
8. Templates persistence, CRUD, seed data, and recipe application.
9. Hardened backend Git/Secrets boundary and Git plugin.
10. Sync checkout exclusion and transition handling.
11. Integrated migrations, E2E/GUI, documentation, and release notes.

Every completed slice is committed and pushed. A slice may contain coordinated
commits in multiple repositories, but the tree must be green at its declared
gate before the next slice begins.
