# Verstak Desktop — Development Plugin Flow

## Overview

Official plugins live in the **verstak-official-plugins** monorepo and are developed
separately from the desktop core. During development, plugins are installed into
the desktop's local `plugins/` directory as **packaged bundles** (not source code).

```
git/
  verstak-desktop/                ← desktop core
  verstak-official-plugins/       ← official plugin source + dist output
```

## Plugin Source → Package Flow

1. **Source code** lives in `verstak-official-plugins/plugins/<plugin-id>/`
2. **Running `build.sh`** in official-plugins:
   - Builds frontend (if frontend/package.json exists)
   - Builds backend Go binary (if backend/main.go exists)
   - Packages the result into `dist/<plugin-id>/`
3. **Package structure** (`dist/platform-test/`):
   ```
   plugin.json
   frontend/dist/index.js
   backend/platform-test          (compiled binary)
   ```

## Installing Dev Plugins in Desktop

From the `verstak-desktop/` directory:

```bash
./scripts/install-dev-plugins.sh
```

This script:
- Locates `../verstak-official-plugins/`
- Builds packages there if `dist/` is stale
- Creates `./plugins/<plugin-id>/` from the dist package
- Does **not** affect other plugins in `./plugins/`

The `plugins/` directory is in `.gitignore` — it is never committed.

## Smoke Test

After installing, verify that the desktop runtime discovers the plugin:

```bash
./scripts/smoke-platform.sh
```

This validates:
- Plugin directory exists
- `plugin.json` manifest is valid
- All required manifest fields are present
- `DiscoverPlugins()` finds `verstak.platform-test`
- Capabilities are registerable

## Desktop Runtime Scanning Paths

The desktop resolves plugin directories in one shared backend resolver. Priority:

| Path | Purpose |
|------|---------|
| `VERSTAK_PLUGIN_DIR` | Dev/test override. Multiple paths can be separated with the OS path separator |
| `./plugins/` | Bundled/dev plugins relative to the current working directory |
| `<binary-dir>/plugins/` | Packaged plugins shipped next to the desktop executable |
| `~/.config/verstak/plugins/` | User-installed plugins |

The resolver normalizes paths and removes duplicates before scanning. Missing
directories are ignored by discovery.

Discovery scans all resolved directories in order. If two plugin packages declare
the same `id`, the first package wins and later duplicates are skipped. The
warning includes both package paths, so during development check the log if an
updated plugin appears to be ignored.

## Bundled Plugin API During Development

Frontend bundles are mounted with a plugin-scoped API created by
`createPluginAPI(pluginId)`. The current API supports:

- `settings.read/write/writeAll`
- `capabilities.list/get/has`
- `commands.register/execute` for handlers declared in `contributes.commands`
- `events.publish/subscribe` using the bundled frontend event bus
- `files.list/metadata/readText/writeText/createFolder/move/trash/openExternal/showInFolder`
  for canonical vault-relative slash paths guarded by `files.read`,
  `files.write`, `files.delete`, and `files.openExternal`. Backslashes,
  Windows absolute paths, UNC paths, traversal, `.verstak` variants, and
  symlink read/write/move/trash/external-open operations are rejected. Text
  read/write is UTF-8 only and limited to 2 MB for reads.
- `workbench.openResource/editResource` for routing vault resources to
  contributed `openProviders`. Plugins must declare `workbench.open`; this is a
  policy/contract check. Files and Notes plugins call this API and do not import
  a concrete editor plugin.

Editor/viewer plugins contribute providers with `contributes.openProviders`.
Workbench selects by resource kind, extension/mime, context (`generic-text`,
`generic-markdown`, `notes-markdown`), user preference, priority, then
deterministic `pluginId/providerId` tie-break. If nothing matches, Workbench
shows `no-provider` fallback instead of a core editor.

The official `verstak.default-editor` plugin provides three openProviders:
`verstak.default-editor.text` (generic-text), `verstak.default-editor.markdown`
(generic-markdown), and `verstak.default-editor.notes-markdown` (notes-context).
It uses a single unified `DefaultEditor` component with textarea-based editing,
simple markdown preview, dirty state tracking, and Ctrl+S save.

This is a cooperative contract, not a sandbox. Bundled plugins run in the same JS
context as the desktop frontend; real isolation is deferred to the sidecar/sandbox
milestone.

## Important Rules

- **Never commit** `plugins/` to `verstak-desktop` — it's in `.gitignore`
- **Never copy source code** from `verstak-official-plugins/plugins/` directly —
  always use the dist package from `verstak-official-plugins/dist/`
- **Run `install-dev-plugins.sh`** after any change in the plugin source
- **Run `smoke-platform.sh`** after installing to verify discovery
