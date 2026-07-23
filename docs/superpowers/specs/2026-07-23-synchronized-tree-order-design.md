# Synchronized Workspace-Tree Order Design

**Date:** 2026-07-23

## Status

Approved for implementation. This document incorporates the correction that
user-defined folder and Deal order is vault organization and must synchronize
between devices.

## Problem

The workspace tree currently derives every child list from filesystem discovery:
folders first, then Deals, alphabetically. Drag-and-drop can change a node's
parent, but it cannot express a stable mixed sibling order. The current drop
handling also lacks precise before/inside/after placement, robust hover expansion,
free child-list targets, edge autoscroll, and complete cleanup.

The desired order must survive restart and must converge across devices that
synchronize the same vault. It must not be stored in generic application
settings, which are device-local.

## Goals

- Give every folder and Deal a stable drag identity:
  - `folder:<uuid>`
  - `workspace:<uuid>`
- Support one backend placement request:

  ```text
  { sourceKey, targetKey, position }
  ```

  where `position` is `before`, `after`, `inside`, or `root`.
- Keep the backend authoritative for tree topology, validation, filesystem
  movement, order persistence, and reconciliation.
- Persist sibling order as internal vault metadata and synchronize it as an
  explicit sync entity.
- Render the same tree order on every synchronized device.
- Preserve safe deterministic behavior when metadata mentions missing IDs or
  discovery finds IDs not yet present in metadata.
- Provide row-third targets, free child-list targets, clear indicators, one
  stable-key hover timer of about 700 ms, edge autoscroll, and complete drag
  cleanup.

## Non-goals

- Synchronizing sidebar width, scroll position, temporary expansion, keyboard
  focus, current selection, or active drag state.
- Changing the physical name of an item for an order-only move.
- Replacing the existing UUID marker or workspace-tree scan model.
- Relaxing the general `.verstak` sync exclusion.
- Adding a generic settings synchronization framework.

## Existing Constraints

The workspace-tree service already owns semantic UUID discovery, parent
resolution, filesystem moves, descendant checks, reconciliation, and the public
tree snapshot. Its builder currently applies a deterministic folders-first
alphabetical fallback.

The sync scanner intentionally excludes any path containing `.verstak`. It then
adds narrowly selected internal identity data through dedicated semantic
entities such as `workspace-folder` and `workspace`. The server accepts generic
entity types and applies a global sequence, so a new desktop entity does not
require weakening filesystem exclusions or changing the wire protocol.

## Storage Decision

Store the canonical order document at:

```text
.verstak/workspace-tree/order.json
```

The file is core-owned internal vault metadata. It is not included by the
ordinary file scanner. Instead, the sync snapshot scanner explicitly reads this
one exact path and represents it as the dedicated entity:

```text
entity_type: workspace-tree-order
entity_id: tree
op_type: update
```

This is the least invasive sync-eligible placement:

- it keeps all existing `.verstak` exclusion rules intact;
- it does not expose internal metadata as user files;
- it follows the existing dedicated-semantic-entity pattern;
- it gives bootstrap and ordinary local scanning a durable view of the order;
- it avoids coupling vault organization to device-local application settings.

No other path below `.verstak/workspace-tree` is allowlisted.

## Order Document

Version 1 uses one complete canonical document:

```json
{
  "version": 1,
  "children": {
    "root": [
      "folder:11111111-1111-1111-1111-111111111111",
      "workspace:22222222-2222-2222-2222-222222222222"
    ],
    "11111111-1111-1111-1111-111111111111": [
      "workspace:33333333-3333-3333-3333-333333333333"
    ]
  }
}
```

`root` is the root sibling list. Every other map key is a parent folder UUID.
Values contain stable node keys. A current node appears in at most one list.
Serialization sorts parent-map keys while retaining sibling array order. Writes
use a same-directory temporary file and atomic rename.

The synchronized payload is the same versioned document, not an internal file
path. This keeps remote application independent of platform path syntax.

## Validation and Reconciliation

The metadata parser rejects:

- unsupported versions;
- malformed JSON;
- parent keys other than `root` or UUIDs;
- malformed stable node keys;
- duplicate current node keys across lists.

An invalid local document does not mutate the filesystem. The service reports a
tree diagnostic and uses deterministic fallback order. An invalid remote payload
is rejected as an apply error and remains visible through existing sync error
handling.

Valid metadata may contain syntactically valid IDs that are not currently
discovered. This is necessary because structural and order operations can arrive
at different moments. Reconciliation behaves as follows for each actual parent:

1. Start with discovered children in deterministic folders-first,
   case-insensitive name order, with stable key as the tie-breaker.
2. Emit actual children named in that parent's stored list, in stored order.
3. Ignore stored keys that are missing or whose actual parent differs.
4. Append actual children not named in the stored list using the deterministic
   fallback order.

Stale keys remain in the stored document until a successful local placement
canonicalizes the affected lists. Keeping them temporarily prevents an order
operation from being lost if it is applied before its corresponding structural
operation.

Newly discovered IDs therefore appear deterministically on all devices even
before another placement rewrites the document.

## Placement Semantics

The backend accepts:

```go
type PlacementRequest struct {
    SourceKey string `json:"sourceKey"`
    TargetKey string `json:"targetKey"`
    Position  string `json:"position"`
}
```

Semantics:

- `before`: place the source immediately before the target in the target's
  current parent.
- `after`: place the source immediately after the target in the target's current
  parent.
- `inside`: place the source as the final child of the target folder.
- `root`: place the source as the final root child. `targetKey` must be empty.

Only `inside` accepts a folder target. `before` and `after` accept either node
kind. The source and target are resolved from the current backend tree by stable
key, never by array index or caller-provided parent.

Before mutation the backend validates:

- source and target key syntax and existence;
- supported position;
- source and target are different;
- target requirements for the position;
- folder-only `inside`;
- no folder move into itself or a descendant;
- destination path availability when the parent changes.

The backend derives the destination parent. If that parent differs, it performs
the existing guarded filesystem move. It then canonicalizes current sibling
lists, removes the source from every list, inserts it once at the requested
position, atomically writes order metadata, and rebuilds the public tree.

Validation happens before filesystem mutation. If the filesystem rename fails,
the order document is untouched. If the metadata write fails after a successful
rename, the filesystem remains safely moved and the service reconciles using the
previous order plus deterministic fallback; the API reports the write error.

Order-only placement never renames filesystem entries.

## Sync Behavior

The sync snapshot gains an optional initialized order document read only from
`.verstak/workspace-tree/order.json`.

- First local snapshot: establish a baseline without operations, as today.
- Bootstrap: publish an `update` for a valid pre-existing order document after
  the initial pull.
- Local placement: the order file is written by the tree service; the normal
  local scan detects the exact metadata change and queues one complete-state
  order operation after semantic structure operations.
- Remote order operation: validate and atomically write the document through the
  workspace-tree service, then reconcile the tree.
- Remote rebase: accept the resulting metadata snapshot without echoing it.

Tree-order operations are sorted after folder/workspace structural operations
within one scan. If cross-device timing still causes order to arrive first,
retained stable IDs and later deterministic reconciliation preserve the intent.

The server's global sequence makes complete-state operations deterministic:
devices apply order documents in server order and the last accepted state wins.
Existing conflict reporting still identifies concurrent writes to
`workspace-tree-order/tree`.

## Frontend Interaction

The frontend sends only stable keys and a position.

Each row is divided geometrically into thirds:

- top third: `before`;
- middle third: `inside` for folders;
- bottom third: `after`.

For a Deal, the middle third is assigned to the nearer of `before` or `after`,
because Deals cannot contain children.

Indicators:

- before/after: a high-contrast insertion line spanning the sibling row;
- inside: a highlighted row and folder affordance;
- root/free child list: a highlighted empty-list insertion area.

Expanded folders render a free child-list drop area after their children.
Empty expanded folders render the same area. Dropping there means `inside` and
does not depend on a child row. The tree root has an equivalent free root-list
area using `root`.

One hover-expansion timer exists for the whole tree. It is keyed by the target's
stable folder key and fires after approximately 700 ms only while the same
`inside` target remains active. Changing target, leaving the tree, dropping,
ending, cancelling, or unmounting clears it.

While dragging near the top or bottom edge of the scrollable tree, one animation
loop scrolls proportionally toward that edge. It stops outside the edge bands
and during every cleanup path.

Cleanup clears the active source key, target descriptor, hover timer,
autoscroll frame, transient indicators, native payload assumptions, and any
root/child drag counters. It runs for success, backend error, malformed payload,
`drop`, `dragend`, escape/cancel-equivalent loss, and component destruction.

The existing cross-Deal Files payload remains separate and continues to target
Deal rows.

## Device-local State Boundary

These remain in application settings or memory and never enter the order
document or its sync payload:

- sidebar width;
- tree scroll position;
- temporary expanded folder IDs;
- current selection and active Deal;
- focused node;
- hover and drag state.

## Testing Strategy

Backend tests cover:

- root, before, after, and inside placement for folders and Deals;
- mixed-kind sibling order;
- restart persistence;
- stable fallback for new and missing IDs;
- malformed metadata;
- missing targets, self targets, non-folder `inside`, and descendant rejection;
- path conflicts and failed writes without silent success;
- order-only moves that do not rename files;
- exact-path sync scanning while other `.verstak` paths remain excluded;
- bootstrap/update operation generation and operation ordering;
- remote payload application, rebase, and second-device convergence.

Frontend component/E2E tests cover:

- row-third target selection;
- Deal middle-third behavior;
- free root and child-list areas;
- indicators;
- one stable-key 700 ms hover timer;
- edge autoscroll start/stop;
- stable-key payloads;
- successful and failed placement cleanup;
- preservation of the Files drag payload.

Release verification will include focused Go tests, frontend checks, desktop
build/tests, Playwright DnD coverage, and a two-vault sync smoke test before
Stage 1 publication.
