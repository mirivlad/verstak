# UX/UI Refactor Plan

## Assumptions

- `verstak-desktop` is the core platform and UI shell. Notes, files, editor, activity, journal, browser inbox, and search behavior stay plugin-owned.
- The old `~/git/verstak` UI is a visual and interaction reference, not an architecture source.
- The v2 shell should improve orientation, density, keyboard/mouse ergonomics, and responsive behavior without reintroducing the v1 monolith.

## Reference Rules

Keep from v1:

- compact dark workbench rhythm;
- clear title/header zones;
- dense rows and tabs for repeated work;
- action controls that appear where the user is working;
- custom, in-app interaction surfaces instead of browser-default dialogs where the flow is important.

Avoid from v1:

- putting user business workflows into `App.svelte`;
- direct coupling between shell and notes/files/editor internals;
- global mutable UI state that plugins must know about;
- moving plugin behavior into core for visual convenience.

## Mimo Delegation Model

Use `~/bin/mimo.sh run --dir <repo> "<task>"` only for bounded junior tasks.

Good tasks:

- compare two small components and write a short report to `/tmp`;
- draft CSS for a named component within existing tokens;
- inspect one test file and suggest missing assertions;
- make a one-component mechanical change after the target behavior is already specified.

Do not delegate:

- architecture decisions;
- plugin/core boundary decisions;
- final diff review;
- verification claims;
- commits or pushes.

Every mimo result must be reviewed with `git diff` and verified independently.

## Work Plan

1. Shell orientation
   - Move persistent search and workspace context into the workspace header.
   - Keep exactly one global search entry visible at a time.
   - Preserve search availability in global plugin views.

2. Responsive shell
   - Make narrow viewports usable by stacking sidebar above workspace content.
   - Ensure tabs and workspace cards do not force horizontal page overflow.
   - Add Playwright coverage for mobile geometry.

3. Today surface
   - Make Today feel like a work-resume surface, not a static card grid.
   - Keep data loading from plugin settings/contributions; do not add business logic to core.
   - Improve empty states and quick actions based on available workspace tools.

4. Plugin manager polish
   - Improve scanning, status density, and permission readability.
   - Keep enable/disable/status behavior unchanged.
   - Verify degraded/failed/disabled plugin paths.

5. Files/workbench ergonomics
   - Use the existing files plugin comparison report as input.
   - Prefer plugin-local improvements: context menu, keyboard navigation, selection, and custom confirmation.
   - Do not add file-manager logic to the shell.

## Verification Gates

Before commit:

- `npm run build`
- focused Playwright suite for the touched flow
- full `npm run test:e2e` when App, Sidebar, WorkspaceHost, PluginManager, or shared shell layout changes
- desktop and mobile screenshots inspected manually

Before push:

- re-run the relevant verification from a clean current worktree state;
- inspect `git status` and `git diff --stat`;
- push only after tests and visual smoke have current evidence.
