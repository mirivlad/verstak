# UI Polish Milestone

## Assumptions

- This milestone is visual polish only: no backend API changes, no plugin contract changes, and no feature removal.
- English remains the baseline UI language for this pass. Hardcoded Russian strings that are visible in core UI should be converted to English where touched.
- Official plugins are plain JS bundles with injected CSS. Introducing a new shared UI package would be larger than this milestone, so shared primitives will be expressed as global design tokens/classes plus consistent local CSS in each bundle.
- Desktop workbench density should stay compact. The goal is consistency, not a marketing-style redesign.

## Audit Findings

### Existing Tokens

- `frontend/src/App.svelte` already defines partial global button classes: `btn-primary`, `btn-secondary`, `btn-danger`, `btn-ghost`, `btn-icon`, plus global button focus/disabled behavior.
- Most shell and plugin surfaces hardcode the same dark palette directly: `#1a1a2e`, `#16213e`, `#0f3460`, `#4ecca3`, `#e94560`, `#8b8ba8`.
- There are no named CSS variables for semantic surface, selected row, border, muted text, warning, or focus ring.
- Plugin bundles repeat their own toolbar/button/list/empty/error styles instead of reusing shell-level primitives.

### Repeated Patterns

- Page/workspace headers: `WorkspaceHost.svelte`, `WorkbenchHost.svelte`, `ViewContainer.svelte`, plugin toolbars.
- Tab bars: workspace tabs use a custom active style separate from selected rows and sidebar active states.
- Toolbars: Files, Notes, Search, Activity, Journal, Browser Inbox, and default editor use similar but inconsistent dark toolbar rules.
- List rows: Files rows, Notes rows, Activity rows, Journal rows, Secrets list items, Browser Inbox rows, Global Search results, WorkspaceTree rows.
- Cards/panels: Today panels, worklog suggestions, Secrets detail/form, Browser Inbox detail text, plugin manager cards.
- Empty/error states: many are plain centered text or show technical details as primary text.

### Candidate Primitives

Implement as global classes/tokens first, then apply locally:

- PageHeader / WorkspaceHeader: compact title, metadata, right actions/search.
- TabBar / TabButton: consistent active tab, hover, focus.
- Toolbar / ToolbarGroup: grouped controls with separators and predictable height.
- Button variants: primary, secondary, ghost, danger, icon.
- ListRow: hover, selected, disabled, metadata line, actions zone.
- Card: flat desktop panel with 6px radius and subtle border.
- SplitPane: list/detail grid with shared border and selected item states.
- EmptyState: centered title, hint, optional action.
- InlineAlert: readable message plus optional details/debug text.
- StatusBadge: compact semantic label for type, status, warning/error.
- Menu: context menu with same surface, hover, danger item, separator.

## Implementation Plan

1. Add semantic CSS variables and global primitive classes in `frontend/src/App.svelte`.
   - Colors: background, surface, surface-muted, surface-hover, surface-selected, border, border-strong, text-primary, text-secondary, text-muted, accent, accent-muted, danger, warning, success.
   - Spacing: 4/8/12/16/24/32 via variables.
   - Radius, font sizes, focus ring, and modest elevation tokens.

2. Normalize shell surfaces.
   - Update `WorkspaceHost.svelte`, `WorkbenchHost.svelte`, `ViewContainer.svelte`, `Sidebar.svelte`, `WorkspaceTree.svelte`, `GlobalSearch.svelte`, and `TodaySurface.svelte` to use tokens and shared state rules.
   - Convert visible Russian shell search/empty strings to English.
   - Keep plugin manager architecture untouched; only global tokens should make it less visually divergent.

3. Normalize Files.
   - Group toolbar into Navigation, Create, Selection, Clipboard/More, and filter/sort areas.
   - Keep existing context menu and operations, restyle menu with shared surface/hover/danger states.
   - Make single row click open when no modifier is used; keep ctrl/meta/shift selection behavior and keyboard selection.
   - Reduce actions column visual noise by showing secondary row actions on hover/selection and using a compact More/action group.

4. Normalize Notes.
   - Make New Note the primary page action.
   - Keep filter/sort inside the toolbar group.
   - Make entire note row readable and clickable; row actions remain visible enough and use icon-button states.
   - Use shared empty/status/error styling.

5. Normalize Search.
   - Render results as compact rows/cards with type badge/icon, title/path, secondary match reason, and compact Open button.
   - Show provider errors as inline alert-style secondary information, not as raw main status text.

6. Normalize Activity and Journal.
   - Add human-readable event labels such as "Workspace selected", "File opened", "Note edited", and "Work session detected".
   - Move raw event names/source IDs into a details block or secondary metadata.
   - Style plugin errors through inline alert patterns.
   - Present worklog suggestions as cards with consistent title/summary/minutes actions.

7. Normalize Secrets and Browser Inbox.
   - Keep split-pane list/detail layout.
   - Use shared list rows, detail card/form surfaces, and empty states.
   - Render "Value hidden" as password-like hidden field state.
   - Use button variants: primary for save/create, secondary for copy/edit/remove where appropriate, danger for delete/clear/remove destructive actions.

8. Verify.
   - Run desktop frontend build and relevant Playwright checks if available.
   - Run official plugin smoke/check script.
   - Capture before/after screenshots or save available after screenshots under `docs/ui-polish-assets/`.

## Out of Scope

- Backend API or Wails method changes.
- Plugin manifest/contribution contract changes.
- New heavy UI dependencies.
- Full i18n extraction.
- Redesigning the product information architecture.
- Rebuilding official plugins into Svelte packages.
