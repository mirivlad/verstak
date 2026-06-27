# Notes/Files Plugin Architecture Plan

This document locks the Notes/Files/Open architecture for the next milestones.
Files Core Service was implemented in Milestone 6a; this document still does not
start Notes UI, Notes plugin, Files UI plugin, editor plugin, watcher, sync, or
binary streaming.

## Current Readiness

The platform is ready for bundled plugin UI experiments. Files Core is available
as a safe vault-scoped text file API. Notes, Files UI, and editor/viewer UI still
need plugin-level implementations and host surfaces before real product use.

Already available:

- Plugin discovery, lifecycle, settings, capabilities, bundled commands, and
  bundled frontend events.
- Workspace lifecycle APIs for top-level physical folders under the vault root.
- Plugin-owned internal storage directories:
  `.verstak/plugin-data/<pluginId>`, `.verstak/plugin-settings/<pluginId>`, and
  `.verstak/plugin-cache/<pluginId>`.
- Contribution registry entries for `fileActions`, `noteActions`,
  `contextMenuEntries`, `searchProviders`, `activityProviders`, and
  `statusBarItems`.

Not available yet:

- Notes plugin/API as a semantic view over Markdown files.
- Files UI plugin.
- Editor/viewer plugin.
- Open/edit resource routing and provider selection.
- UI hosts for file actions, note actions, context menus, search providers,
  activity providers, or status bar items.
- Watcher/indexer for external filesystem changes.
- Real plugin isolation. Current permission checks are contract/policy checks,
  not a security boundary for bundled frontend JavaScript.

## Canonical Notes Model

Notes are ordinary human-readable Markdown files inside the vault. They must be
visible and editable outside Verstak.

Canonical rules:

- Notes are `.md` files, not opaque records.
- Canonical folder name is exactly `Notes`.
- Do not create lowercase `notes`.
- Do not store user notes in `.verstak/notes/`.
- Do not store user notes in `.verstak/plugin-data/verstak.notes/`.
- Do not use UUID-only filenames for notes.
- The note title is the source of truth.
- The filename is a normalized projection from the title.
- `RenameNote` must update both the title and the `.md` filename.

Canonical scoped paths:

- Notes live under `<workspace>/Notes/`.
- `Overview.md` is allowed as an ordinary Markdown filename, but the platform
  must not create it automatically or treat it as a dedicated UI entity.
- `<workspace>` is the top-level physical folder name under the vault root.
- Files plugin workspace views are scoped with `workspaceRootPath`, which is the
  selected top-level workspace folder name.

Visibility requirements:

- Notes UI must show notes as semantic notes.
- Files UI must show the same `.md` files as ordinary files.
- External file managers must show the same `.md` files.
- Outside Verstak, the files must remain useful as normal Markdown.

There is no canonical metadata workspace tree. Adding `note` as a workspace node
type is not part of the next milestone. The official Notes plugin can index and
manage Markdown files inside canonical `Notes/` folders under each top-level
workspace.

## Title To Filename Contract

The title is the source of truth. The filename is derived from the title when a
note is created or renamed.

Normalization rules:

- Replace spaces with `_`.
- Replace typographic dashes with `-`.
- Allow only letters, digits, `.`, `_`, and `-`.
- Append `.md` if the normalized name does not already end with `.md`.
- Reject empty normalized names.
- Preserve canonical `Notes` folder casing.

Examples:

| Title | Filename |
|---|---|
| `Overview` | `Overview.md` |
| `Meeting Notes` | `Meeting_Notes.md` |
| `Plan — Phase 1` | `Plan_-_Phase_1.md` |
| `A/B Test: Result` | `AB_Test_Result.md` |

## Collision Policy

Same-folder collisions must not be solved silently with `_2`, `_3`, or timestamp
suffixes.

Creating or renaming a note must return a conflict error if the normalized target
filename already exists in the target `Notes/` folder. The UI should show a clear
dialog or notification and ask the user to change the title.

Required conflict metadata:

- requested title;
- normalized filename;
- target vault-relative path;
- existing vault-relative path.

## Files Service Model

Files service is the raw vault file layer. It works with vault-relative paths and
does not understand note semantics.

Rules:

- All public Files API paths are canonical vault-relative slash paths.
- Backslashes are rejected instead of normalized.
- Absolute POSIX paths, Windows drive paths, and UNC/network paths are rejected.
- `..` traversal is rejected.
- Null bytes are rejected.
- `.verstak/` is reserved case-insensitively and hidden/forbidden by default.
- Access to `.verstak/` is allowed only through internal APIs, not through the
  normal plugin Files API.
- Symlink read/write/move/trash operations are forbidden in Milestone 6a.
  Metadata may report a final symlink as `type: "symlink"`.
- Writes must be atomic: write a temp file in the same directory, close it, then
  rename.
- Delete must follow the trash policy until permanent delete is explicitly
  designed.
- Trash metadata records `originalPath`, `deletedAt`, `originalType`, `trashId`,
  and `basename` for future restore work. Restore is deferred.
- Binary files are deferred for write/streaming APIs. Milestone 6a lists binary
  metadata but read/write is UTF-8 text only with a 2 MB read limit.

Minimum Files methods:

- `ListVaultFiles(relativeDir)`.
- `GetVaultFileMetadata(relativePath)`.
- `ReadVaultTextFile(relativePath)`.
- `WriteVaultTextFile(relativePath, content, options)`.
- `CreateVaultFolder(relativePath)`.
- `MoveVaultPath(fromRelativePath, toRelativePath)`.
- `TrashVaultPath(relativePath)`.

Milestone 6a status: implemented in `internal/core/files` and exposed to bundled
plugins as `api.files.list`, `api.files.metadata`, `api.files.readText`,
`api.files.writeText`, `api.files.createFolder`, `api.files.move`, and
`api.files.trash`. It is still text-only for reads/writes and has no watcher,
binary streaming, external editor integration, or Files UI plugin.

Later Files methods:

- `WatchVaultFiles(scope)` once watcher/event delivery is ready.
- `ReadVaultFileBytes` / `WriteVaultFileBytes` for binary files.
- `OpenExternal(relativePath)` with explicit permission and UX confirmation.
- `RevealInFileManager(relativePath)`.

## Official Notes Plugin Model

The official Notes plugin is a semantic layer over Markdown files managed through
the public Files plugin API. There is no core desktop Notes service or
`verstak/core/notes/v1` capability in v2.

Rules:

- A note physically is a `.md` file.
- Notes plugin and Files API must not create two sources of truth.
- Notes plugin reads/writes the same files that Files API lists.
- The note title is the semantic source of truth and is projected to the filename.
  If frontmatter or a first-heading convention is introduced later, `RenameNote`
  must keep that visible title metadata and the filename synchronized.
- Other note metadata should be derived from the file path and filesystem
  metadata, or from Markdown frontmatter if a future milestone introduces it.
- If a note is changed through Files API or an external editor, the future
  watcher/indexer must observe it.
- Until watcher/indexer exists, external changes require reload/rescan.

Minimum Notes plugin behavior:

- list Markdown files in a scoped canonical `Notes/` folder;
- create a Markdown note by writing the normalized filename atomically with
  `overwrite: false`;
- rename a note by moving the file to the normalized filename;
- open and edit notes through Workbench providers;
- never special-case `Overview.md` in the UI.

Later Notes plugin behavior:

- search notes;
- list backlinks;
- resolve note links;
- export notes.

## Notes Vs Files Relationship

Files owns safe raw vault file access. Notes owns note semantics.

The same physical note must be visible through both surfaces:

- Files sees `Project/Notes/Overview.md` as a file.
- Notes sees `Project/Notes/Overview.md` as a note with title `Overview`.

There must be no duplicate note content stored in plugin settings, plugin data,
or a separate `.verstak` note database. Indexes and caches may exist later, but
they must be rebuildable from the canonical Markdown files.

## Capabilities And Permissions

Existing permissions that remain useful:

- `vault.read` for existing vault-level read policy.
- `vault.write` for existing vault-level write policy.
- `vault.watch` for future watcher support.
- `ui.register` for sidebar/views/settings contributions.
- `commands.register` for bundled command handlers.
- `events.publish` and `events.subscribe` for frontend/backend event flows.

New permissions required before real Notes/Files plugins ship:

- `files.read`
- `files.write`
- `files.delete`
- `workbench.open`

Future Notes semantic permissions are deferred until a real Notes plugin/API
ships:

- `notes.read`
- `notes.write`
- `notes.delete`

Those permissions are still policy checks until sidecar/sandbox work provides a
real isolation boundary.

Recommended capabilities:

- `verstak/core/files/v1`
- `verstak/files/v1` provided by the official Files plugin.
- `verstak/notes/v1` provided by the official Notes plugin.

## Frontend Components And Extension Points

Contribution points already registered but not fully hosted:

- `fileActions`
- `noteActions`
- `contextMenuEntries`
- `searchProviders`
- `activityProviders`
- `statusBarItems`

UI work needed:

- Files view host with tree/list modes and selection state.
- Notes view host with note list, open/edit entry points, and preview/details
  region.
- Context menu host that merges core actions with plugin contributions.
- Command palette host for contributed commands.
- Search provider host with cancellation/debounce and result ownership.
- Status bar host for lightweight plugin state.
- Selection/event model for active file, active note, and active workspace node.

The first implementation should host only the contribution points needed by the
official Notes and Files plugins.

## Open/Edit Resource Model

Files and Notes must not embed a concrete editor or viewer. They request that the
Workbench open or edit a resource. The Workbench/provider registry selects the
plugin component.

Required model:

- Files plugin lists files and calls open/edit for a vault file.
- Notes plugin presents Markdown files under canonical `Notes/` folders and calls
  open/edit for the same vault file with notes context.
- `.md` or `.markdown` inside a canonical `Notes/` folder opens in markdown mode
  with notes context.
- `.md` or `.markdown` outside `Notes/` opens in generic markdown mode.
- Plain text opens in `generic-text` mode.
- The same editor provider may support text, generic markdown, and notes-context
  markdown.
- User preferences can select another provider for text, markdown, and
  notes-context markdown.
- Community editor plugins can replace the default editor through the same
  provider registry.
- Core desktop owns registry, routing, Workbench host slot/tab, and preferences.
- Core desktop does not own concrete Files UI, Notes UI, Markdown editor, or file
  preview UI.

Minimal contribution extension:

```json
{
  "contributes": {
    "openProviders": [
      {
        "id": "verstak.platform-test.markdown-diagnostic",
        "title": "Platform Test Markdown Diagnostic",
        "priority": 100,
        "component": "MarkdownDiagnosticProvider",
        "supports": [
          {
            "kind": "vault-file",
            "extensions": [".md", ".markdown"],
            "contexts": ["generic-markdown", "notes-markdown"]
          },
          {
            "kind": "vault-file",
            "mime": ["text/plain"],
            "extensions": [".txt", ".log", ".json", ".yaml", ".yml", ".toml", ".ini", ".conf"],
            "contexts": ["generic-text"]
          }
        ]
      }
    ]
  }
}
```

Open request shape:

```ts
type OpenResourceRequest = {
  kind: "vault-file";
  path: string;
  mode?: "view" | "edit";
  mime?: string;
  extension?: string;
  context?: {
    sourcePluginId?: string;
    sourceView?: "files" | "notes" | string;
    isInsideNotesFolder?: boolean;
    notesScopePath?: string;
    notesMode?: boolean;
  };
};
```

Provider selection rules:

1. Match resource kind.
2. Match extension and/or mime.
3. Prefer providers that explicitly support the request context.
4. Apply user preference for text, markdown, or notes-context markdown when the
   preferred provider is enabled and still supports the resource.
5. Otherwise choose highest priority.
6. Break ties deterministically by plugin id, then provider id.
7. If no provider matches, return a Workbench `no-provider` state rather than
   hardcoding a core editor.
8. If a preferred provider plugin is disabled or unavailable, fall back to the
   deterministic default and surface a non-blocking preference warning later.

Initial preferences:

- `defaultTextEditorProvider`;
- `defaultMarkdownEditorProvider`;
- `defaultNotesMarkdownEditorProvider`.

Per-extension overrides are deferred.

## Migration Risks

- Adding `note` as a workspace node type is a workspace schema migration and is
  explicitly out of scope for the next milestone.
- The canonical path rules must be locked before writing real files into user
  vaults.
- Rename behavior can break external links if link rewriting is not designed.
- Case/folder path ownership must be clear before scoped `Notes/` folders are
  created.
- Raw Files API can expose `.verstak` internals unless reserved paths are blocked.
- File writes need atomic behavior and conflict handling before sync.
- Large/binary files require streaming or byte APIs; text APIs are not enough.
- External editor changes require reload/rescan until watcher/indexer exists.
- Bundled frontend plugins are trusted/cooperative and not isolated from the
  shared JS context.

## Test Plan

Backend Go tests:

- Vault-relative path normalization and traversal rejection.
- Reserved `.verstak` path behavior.
- Files list/read/write/mkdir/move/trash with vault closed/open states.
- Atomic text writes and temp-file cleanup on failure.
- Notes `Notes/` folder casing and no lowercase `notes`.
- Title to filename normalization.
- note create/rename conflict errors without silent suffixes.
- Notes and Files read the same physical `.md` file.
- Permission checks for `files.*`, `notes.*`, `vault.read`, and `vault.write`.

Frontend/unit tests:

- SDK and plugin API shape for Files and Notes draft methods.
- Readable errors for missing permissions, closed vault, missing file, missing
  note, reserved path, and collision.
- Contribution host rendering for note/file actions and context menus.

Playwright e2e tests:

- Create a text file, reload, and verify it is visible in Files.
- Create a note, reload, and verify it is visible in both Notes and Files.
- Rename a note and verify title plus filename change together.
- Attempt same-folder collision and verify user-facing conflict handling.
- External file change requires reload/rescan until watcher exists.

## Implementation Order

1. Define canonical vault-relative path rules and reserved path policy.
2. Implement Files core service with safe list/read/write/mkdir/move/trash.
3. Define open/edit resource request, provider contribution shape, and provider
   selection rules.
4. Extend contribution registry/types with `openProviders`.
5. Add Workbench open/edit routing and a provider-hosted tab/slot.
6. Add preferences for text, markdown, and notes-context markdown provider ids.
7. Use `platform-test` diagnostic provider to verify routing; real default
   editor plugin is deferred to Milestone 6c.
8. Build official Files plugin that calls open/edit resource.
9. Build official Notes plugin as a contextual view over Markdown files in
   canonical `Notes/` folders.
10. Implement future Notes semantic helpers only as a facade over Markdown files,
    never as a second source of truth.
