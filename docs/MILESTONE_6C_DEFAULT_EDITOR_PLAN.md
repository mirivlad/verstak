# Milestone 6c — Default Editor Plugin

## Goal

Create the official Default Editor Plugin as an openProvider for text, generic
markdown, and notes-context markdown files. Core desktop does not own or import
the editor component.

## What was built

### Plugin: `verstak.default-editor`

Location: `verstak-official-plugins/plugins/default-editor/`

**Manifest declares 3 openProviders:**

| Provider ID | Context | Extensions |
|-------------|---------|------------|
| `verstak.default-editor.text` | `generic-text` | `.txt`, `.log`, `.conf`, `.ini`, `.toml`, `.yaml`, `.yml`, `.json`, `.csv` |
| `verstak.default-editor.markdown` | `generic-markdown` | `.md`, `.markdown` |
| `verstak.default-editor.notes-markdown` | `notes-markdown` | `.md`, `.markdown` |

All providers use the same `DefaultEditor` component (unified, not 3 separate editors).

**Permissions:** `files.read`, `files.write`, `workbench.open`

**Capabilities required:** `verstak/core/files/v1`, `verstak/core/workbench/v1`

### Frontend component: `DefaultEditor`

- **Modes:** text (textarea), generic-markdown (editor + preview), notes-markdown (editor + preview + notes badge)
- **File loading:** `api.files.readText(path)` with loading/error states
- **Saving:** `api.files.writeText(path, content, { overwrite: true })` with dirty/saved/error states
- **Keyboard:** Ctrl+S / Cmd+S save, Tab indentation
- **Markdown preview:** Simple renderer (no raw HTML, no script injection)
- **Notes context:** Badge + info bar, no separate note entity, no `.verstak/notes`

## Verification

- `go test ./...` — PASS
- `go vet ./...` — PASS
- `npm run build` (frontend) — PASS
- `npm run test:e2e` — 28/28 PASS (20 existing + 8 new)
- `scripts/check.sh` (official-plugins) — PASS
- `scripts/build.sh` (official-plugins) — PASS
- SDK checks — PASS

## Manual testing

Until Files UI plugin exists, use platform-test diagnostics panel to manually
open files via workbench: click "Open Text Diagnostic", "Open Markdown Diagnostic",
or "Open Notes Diagnostic" buttons. These call `api.workbench.editResource` and
route through the default-editor provider.

## Deferred

- CodeMirror/Monaco editor
- Backlinks, internal link navigation
- Secret widgets
- Image asset pipeline
- Files UI plugin
- Notes UI plugin
- Watcher/sync
- External open
- Sidecar/security isolation
