# Milestone 6b - Open/Edit Provider Registry + Workbench Routing Skeleton

This milestone adds the minimal infrastructure layer for open/edit routing before
Files UI, Notes UI, or editor implementation starts. It does not implement those
plugins and does not add a concrete core-owned Markdown editor.

## Existing Architecture Found

| Path | Section/title | Summary |
|---|---|---|
| `../verstak-docs/00_README.md` | Main architecture invariant | Core does not know concrete notes, file manager, or markdown editor features. It owns vault, plugin runtime, capability registry, contribution points, permissions, settings, events, storage, and UI shell. |
| `../verstak-docs/01_Product_Vision.md` | What is not in core / platform goal | Markdown editor, file manager, preview, and notes workflow are plugins. Users should be able to replace the markdown editor or install multiple editors. |
| `../verstak-docs/02_Platform_Architecture.md` | UI Shell / Capability Registry | UI Shell knows contribution points, not concrete note editor or file preview implementations. Files plugin checks editor/viewer capabilities instead of depending on `official.markdown-editor`. |
| `../verstak-docs/03_Repositories.md` | Repository split | `verstak-desktop` is Core Platform + UI Shell and does not contain mandatory notes, file manager, or markdown editor modules. Official plugins live separately. |
| `../verstak-docs/04_Plugin_System.md` | Goal / capabilities instead of plugin names / contribution points | Notes, file manager, editor, and viewer are plugin functions. The docs explicitly reject `requires: ["official.markdown-editor"]` and prefer `optionalRequires: ["editor.text.markdown"]`. |
| `../verstak-docs/05_Official_Plugins.md` | official.files / official.notes / official.markdown-editor | Files optionally depends on editor/viewer capabilities. Markdown editor provides `editor.text`, `editor.text.markdown`, and `editor.note.markdown`, and must not own note storage or depend directly on `official.notes`. |
| `../verstak-docs/06_Migration_Strategy.md` | Do not / Definition of Done | Do not make notes/files/editor mandatory core parts. The platform transition is done only when notes/files/editor/preview/activity work as plugins. |
| `docs/NOTES_FILES_PLUGIN_PLAN.md` | Canonical Notes Model | Notes are ordinary Markdown files in canonical `Notes/` folders. No lowercase `notes`, no `.verstak/notes`, no plugin-data note content, no UUID-only filenames. |
| `docs/NOTES_FILES_PLUGIN_PLAN.md` | Files Service Model | Files Core is raw vault file access and does not understand note semantics. Milestone 6a exposes safe text file methods and defers UI, watcher, binary streaming, external editor, and restore. |
| `docs/PLUGIN_RUNTIME.md` | Contribution Points / Bundled Frontend Plugin API | Current runtime hosts views/sidebar/settings/commands and exposes plugin-scoped Files API. `fileActions`, `noteActions`, `contextMenuEntries`, search/activity/status bar entries are registered but not hosted. |
| `internal/core/contribution/registry.go` | Contribution registry | Registry has the established contribution points and now extends that model with `openProviders`. |
| `internal/core/plugin/plugin.go` | Manifest contributions | Plugin manifest types now include `openProviders` alongside existing contribution points. |
| `frontend/src/lib/shell/WorkbenchHost.svelte` and `frontend/src/lib/plugin-host/PluginBundleHost.svelte` | Frontend plugin host | Workbench mounts the selected provider component by plugin id and component id. It remains generic and does not know a concrete editor. |
| `frontend/src/lib/plugin-host/VerstakPluginAPI.js` | Plugin API | Bundled plugins can call `api.workbench.openResource()` and `api.workbench.editResource()` in addition to settings, capabilities, events, commands, and Files API. |
| `../verstak-sdk/src/types.ts` and `../verstak-sdk/schemas/manifest.json` | SDK contribution contracts | SDK types/schema define `openProviders`, `OpenResourceRequest`, provider supports, and `files.*` permissions. |

## What Matches The Desired Model

- Files/Notes/Editor are already documented as plugins, not core modules.
- Official plugins are expected to use the same runtime as community plugins.
- Files plugin is already documented as capability-driven, not hard-wired to a markdown editor.
- Markdown editor is already documented as replaceable via capabilities.
- Notes are already documented as Markdown files under canonical `Notes/` folders, without `.verstak/notes`, UUID note entities, or a second storage truth.
- Desktop code has no hardcoded Markdown editor component.
- Existing plugin host can mount arbitrary plugin components, which is enough foundation for an editor provider host.

## Contradictions Found

- `docs/NOTES_FILES_PLUGIN_PLAN.md` still said Files Core API/capability/permissions were unavailable, while later sections and code show Milestone 6a implemented them.
- `../verstak-docs/05_Official_Plugins.md` calls notes "first-class Verstak entities". That is acceptable only as UI semantics; implementation must not create a separate note storage entity.
- `../verstak-docs/05_Official_Plugins.md` lists "Open externally" as a Files fallback. External open remains deferred and must not enter Milestone 6b.
- `../verstak-sdk/schemas/manifest.json` previously did not include `files.read`, `files.write`, or `files.delete` in the permissions enum, while SDK TS types, desktop permissions, and official `platform-test` manifest already used them. Milestone 6b resolves this 6a contract cleanup item.

## Missing Before 6b

- `openProviders` contribution point.
- `OpenResourceRequest` contract.
- Workbench open/edit routing API for Files/Notes plugins.
- Provider selection model using resource kind, extension/mime, notes context, user preference, provider priority, deterministic fallback, and disabled provider fallback.
- User preferences for default text editor provider, default markdown editor provider, and default notes-context markdown editor provider.
- Host slot/tab that mounts the selected provider component with an open resource request.
- Tests for provider registration, selection, preferences, disabled provider fallback, and notes-context routing.

## Added In 6b

- `contributes.openProviders` in SDK schema/types and desktop manifest structs.
- Desktop contribution registry support for registering, replacing, listing, and
  unregistering open providers.
- `OpenResourceRequest`, `OpenResourceContext`, `OpenResourceResult`, and
  opened-resource state in the Workbench routing skeleton.
- Provider selection by resource kind, extension/mime, `generic-text`,
  `generic-markdown`, or `notes-markdown` context, user preference, priority,
  deterministic `pluginId/providerId` fallback, and active plugin filtering.
- `workbench.open` policy permission for plugins that request Workbench routing.
- Draft app settings preferences for default text, markdown, and notes-context
  markdown providers.
- `api.workbench.openResource()` and `api.workbench.editResource()` exposed to
  frontend plugin bundles.
- Minimal Workbench host that mounts the selected provider component from the
  selected provider plugin.
- `no-provider` fallback state when no matching provider exists.
- `platform-test` diagnostic open provider used only to prove routing.

## Decision

Open/edit provider should be an extension of the existing contribution registry,
not a parallel system. It should add `openProviders` beside `views`, `commands`,
`fileActions`, and `noteActions`.

Capabilities remain useful for broad availability and degraded mode, but they are
too coarse for choosing between multiple providers. Provider selection needs a
declarative provider contribution with `supports`, priority, and component id.

## Correct Notes Model

- Notes are a contextual view over ordinary Markdown files under canonical
  `Notes/` folders.
- `.md` inside `Notes/` opens through the selected markdown editor with
  notes-context.
- `.md` outside `Notes/` opens through the selected markdown editor in generic
  markdown mode.
- Plain text opens through the selected text editor provider in `generic-text`
  context.
- Files and Notes call open/edit resource; neither embeds a concrete editor.
- Editor provider selection belongs to Workbench/provider registry.
- No `.verstak/notes`, no UUID note entities, no second truth separate from the
  `.md` file.

## Minimal Infrastructure Changes

1. Add `openProviders` to SDK manifest/schema/types.
2. Add `openProviders` to desktop plugin manifest structs and contribution
   registry.
3. Add `OpenResourceRequest` and provider support match types.
4. Add Workbench/provider selection service with deterministic rules.
5. Add user preferences for `defaultTextEditorProvider`,
   `defaultMarkdownEditorProvider`, and `defaultNotesMarkdownEditorProvider`.
6. Add frontend host plumbing so Workbench can mount the selected plugin
   component.
7. Fix `verstak-sdk/schemas/manifest.json` permissions enum for `files.*`.

## Proposed Milestone 6b Scope

In scope:

- Contribution/types/schema support for `openProviders`.
- Workbench `openResource`/`editResource` routing API.
- Provider selection with notes context and user preferences.
- Minimal host tab/slot for provider component mounting.
- Diagnostic provider plugin contribution sufficient to prove routing.
- Tests for routing, provider selection, disabled fallback, and notes-context
  markdown.

Out of scope:

- Full Files UI feature set.
- Full Notes UI feature set.
- Full editor implementation.
- Real default editor plugin (Milestone 6c).
- Files plugin open/edit integration.
- Notes plugin open/edit integration.
- Hardcoded core Markdown editor.
- Watcher/sync/binary streaming/external editor.
- Sidecar/security boundary.
- Large rewrite.
