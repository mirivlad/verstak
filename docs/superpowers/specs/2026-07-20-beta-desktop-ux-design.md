# Beta Desktop UX Design

Date: 2026-07-20

## Goal

Remove misleading and wasteful desktop layouts before beta: standardize plugin settings width, let an empty overview use available space, render real plugin status components, expose Deals safely to plugins, and make global search report accurate results and progress.

## Scope and Boundary

The desktop remains a plugin host. Synchronization scheduling, labels, settings, and user-facing behavior stay in the Sync plugin. The core only provides existing synchronization transport/state plus generic host infrastructure: component mounting, guarded platform APIs, and an observable running-state field.

## Settings Panel Contract

The plugin settings modal owns the outer layout. Every settings contribution receives a centered surface with:

- `width: 90%` of the usable modal body;
- `box-sizing: border-box`;
- consistent adaptive padding;
- vertical scrolling within the modal body;
- no host-imposed narrow content maximum.

Plugin roots must fill their host surface and must not add arbitrary root `max-width` values. Controls may retain local widths where a compact control is intentional, but sections and form rows use the full available surface.

## Workspace Overview

Summary cards use an auto-fitting grid so the cards present expand across the complete row. The overview side column is rendered only when it has actual attention or resource content. Without side content the main feed becomes a single full-width column. With side content the existing two-column hierarchy remains.

## Plugin Status Host

The status bar currently replaces any item with a handler by a static warning label. A dedicated compact plugin host will load and mount the declared handler component inside the status item.

The compact host:

- uses the same permission and bundle-loading path as other plugin UI;
- provides the normal plugin API;
- constrains the component to inline status-bar dimensions;
- displays a small inert fallback rather than a large loading/error panel;
- disposes the plugin API and component cleanup when the contribution changes.

This is generic host infrastructure and contains no synchronization-specific presentation.

## Synchronization State Support

The existing backend sync DTO gains only state required for truthful plugin presentation, including whether a sync run is currently active. The existing mutex continues to serialize runs. Every failed manual or scheduled run records a status error consistently; a success timestamp is written only after the server pull/push/pull sequence completes.

The host does not schedule synchronization and does not format Sync labels.

## Deal-list Plugin API

Browser Inbox needs semantic Deals rather than a directory listing. A guarded read-only plugin API returns a flattened list of workspace nodes from the UUID tree:

- stable workspace ID;
- display name;
- full vault-relative root path.

Folders are never included. Nested workspaces are included recursively. The API exposes no mutation methods and uses an existing read permission already held by Browser Inbox. Tree lifecycle events allow the plugin to refresh the list.

## Global Search

### Current problem

The shell builds an all-or-nothing index with silent depth and entry limits. It runs another search against the old index while rebuilding and presents `No results` even when indexing is still active. Structural/file changes can leave it stale.

### Design

Index generation has two observable stages:

1. traverse accessible folders and publish filename/path entries;
2. enrich text-file entries with content snippets.

The current query is re-run whenever a newer stage is published. While work is active, the result popup says that indexing is in progress; `No results` is reserved for a completed applicable index. Sequence tokens prevent an older build from replacing a newer one.

The traversal is breadth-first and no longer silently stops at depth five or 220 entries. A defensive cap remains to protect the UI, but reaching it marks the index as partial and displays that fact. The index refreshes on vault/tree/file change signals and on focus when stale.

Path, filename, text-content, and Russian/English keyboard-layout variants remain supported. Opening a nested folder or file preserves its workspace context.

## Error Handling

- A failed plugin status component remains compact and does not break the status bar.
- A failed search directory or file read is recorded as partial indexing, not converted to an authoritative empty result.
- Search cancellation and refresh never clear a newer result set.
- Deal-list failures leave the consuming plugin usable with an explicit empty/error state.

## Verification

Tests will cover:

- settings surfaces occupying 90% of the modal body with safe padding;
- empty and populated overview layouts;
- actual status-handler mounting, cleanup, compact failure fallback, and click behavior;
- backend sync running/error/success timestamps and serialization;
- recursive workspace flattening that excludes folder nodes;
- immediate filename results for `ddd/333/kkk/Files/test.txt`;
- indexing and partial-index labels;
- refresh after file/tree events;
- search by name, path, content, and swapped keyboard layout;
- navigation from nested search results.

Non-trivial Go and TypeScript changes receive LSP diagnostics before and after. Unit, frontend contract, Playwright, and release-build checks run before completion.

## Success Criteria

- Settings content is centered at 90% width without touching modal edges.
- An empty overview has no unused right column.
- The status bar renders plugin-owned live state rather than a constant warning.
- Browser Inbox can request every nested Deal and no ordinary folder.
- Search never says `No results` while its relevant index is still building and finds newly created nested files after refresh.
