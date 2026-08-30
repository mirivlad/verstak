# Task 3 report — Templates

## Result

- Templates no longer contribute a sidebar item. Their CRUD editor is the `verstak.templates.settings` Settings panel.
- The editor uses the enabled `workspaceItems` contribution catalog, persists only plugin IDs, renders localized title/description metadata, and retains removed tools as disabled entries.
- New Deal now selects a persisted template in the Workspace Tree dialog and invokes the Templates command with only `templateId`, parent folder, and Deal name.
- No WorkspaceHost, GlobalSearch, migration, Git Sync, secret-boundary, packaging, release, journal, git, milestone, project, search, or activity code was changed by this task.

## Changed files

### `verstak-official-plugins`

- `plugins/templates/plugin.json`
- `plugins/templates/frontend/src/index.js`
- `plugins/templates/locales/en.json`
- `plugins/templates/locales/ru.json`
- `scripts/smoke-templates-plugin.js`

### `verstak-desktop`

- `frontend/src/lib/shell/WorkspaceTree.svelte`
- `frontend/src/lib/settings/SettingsWindow.svelte`
- `frontend/e2e/workspace-templates.spec.js`

## RED/GREEN evidence

### RED

```bash
cd /home/mirivlad/git/verstak2/verstak-official-plugins
node scripts/smoke-templates-plugin.js
```

Failed as expected with `Templates settings panel was not declared` before the manifest/editor implementation.

The new E2E scenario was also added before implementation: Settings-owned CRUD, catalog labels without raw plugin IDs, and New Deal selection by persisted template ID.

### GREEN

```bash
cd /home/mirivlad/git/verstak2/verstak-official-plugins
node scripts/smoke-templates-plugin.js
```

Passed: `templates plugin smoke passed`.

```bash
cd /home/mirivlad/git/verstak2/verstak-desktop/frontend
npx playwright test e2e/settings-window.spec.js --reporter=line
```

Passed: 6 tests.

```bash
cd /home/mirivlad/git/verstak2/verstak-desktop/frontend
npx playwright test e2e/workspace-templates.spec.js --grep 'Settings owns template CRUD' --reporter=line
```

Passed: 1 test.

```bash
cd /home/mirivlad/git/verstak2/verstak-desktop/frontend
npx playwright test e2e/workspace-templates.spec.js --grep 'removed template tool' --reporter=line
```

Passed: 1 test.

```bash
cd /home/mirivlad/git/verstak2/verstak-desktop/frontend
npx playwright test e2e/workspace-templates.spec.js --grep 'templates are persisted' --reporter=line
```

Passed after retaining unsaved text fields during checkbox rerenders.

`git diff --check` passed in both repositories before their scoped commits.

## Commits

- `verstak-official-plugins`: `6741902 feat(templates): move recipes into settings`
- `verstak-desktop`: `2d8a24c feat(shell): pick templates when creating deals`

Neither commit was pushed.

## Concerns

- The brief names `e2e/settings.spec.js` and `e2e/workspace-tree.spec.js`; those paths do not exist in this checkout. The directly related existing suites are `e2e/settings-window.spec.js` and `e2e/workspace-templates.spec.js`, which were used instead.
- Playwright emits existing Svelte a11y/unused-selector warnings from `App.svelte`, `SettingsWindow.svelte`, `WorkspaceTree.svelte`, and `Modal.svelte`. No warning was introduced or changed by this task.
- Concurrent unrelated edits were present in both repositories while this task ran. They were not staged or included in either Task 3 implementation commit.
