from pathlib import Path


def replace_once(path, old, new):
    p = Path(path)
    text = p.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f'{path}: expected one match, got {count}: {old!r}')
    p.write_text(text.replace(old, new, 1))


def replace_all(path, old, new, expected):
    p = Path(path)
    text = p.read_text()
    count = text.count(old)
    if count != expected:
        raise SystemExit(f'{path}: expected {expected} matches, got {count}: {old!r}')
    p.write_text(text.replace(old, new))


def replace_present(path, old, new):
    p = Path(path)
    text = p.read_text()
    count = text.count(old)
    if count < 1:
        raise SystemExit(f'{path}: expected at least one match: {old!r}')
    text = text.replace(old, new)
    if old in text:
        raise SystemExit(f'{path}: replacement left legacy text behind: {old!r}')
    p.write_text(text)

# GlobalSearch must navigate by exact contribution ids, never semantic/string kinds.
p = 'frontend/src/lib/shell/GlobalSearch.svelte'
replace_once(p, "  function pluginToolKind(pluginId, label) {\n    if (pluginId === 'verstak.browser-inbox') return 'browser-inbox';\n    if (pluginId === 'verstak.activity') return 'activity';\n    if (pluginId === 'verstak.journal') return 'journal';\n    return String(label || pluginId || '').toLowerCase();\n  }\n\n", '')
replace_once(p, '  async function indexPluginSettings(pluginId, label, rank, view, nodes) {', '  async function indexPluginSettings(pluginId, label, rank, view, nodes, workspaceItemId) {')
replace_once(p, '          toolKind: pluginToolKind(pluginId, label),', "          workspaceItemId: workspaceItemId || '',")
replace_once(p, "    const contributions = i18n.localizeContributionSummary(rawContributions || {});\n    const viewByPluginId = new Map();", "    const contributions = i18n.localizeContributionSummary(rawContributions || {});\n    const workspaceItemByPluginId = new Map();\n    (contributions.workspaceItems || []).forEach(item => {\n      if (item.pluginId && item.id && !workspaceItemByPluginId.has(item.pluginId)) workspaceItemByPluginId.set(item.pluginId, item.id);\n    });\n    const viewByPluginId = new Map();")
replace_once(p, "        action: entry.type === 'folder' ? 'file-folder' : 'file',\n        path,\n        nodes,", "        action: entry.type === 'folder' ? 'file-folder' : 'file',\n        path,\n        nodes,\n        workspaceItemId: workspaceItemByPluginId.get('verstak.files') || '',")
replace_once(p, "      indexPluginSettings('verstak.journal', tr('search.type.journal'), 50, viewByPluginId.get('verstak.journal'), nodes),", "      indexPluginSettings('verstak.journal', tr('search.type.journal'), 50, viewByPluginId.get('verstak.journal'), nodes, workspaceItemByPluginId.get('verstak.journal') || ''),")
replace_once(p, "      indexPluginSettings('verstak.browser-inbox', tr('search.type.browserInbox'), 55, viewByPluginId.get('verstak.browser-inbox'), nodes),", "      indexPluginSettings('verstak.browser-inbox', tr('search.type.browserInbox'), 55, viewByPluginId.get('verstak.browser-inbox'), nodes, workspaceItemByPluginId.get('verstak.browser-inbox') || ''),")
replace_once(p, "      indexPluginSettings('verstak.activity', tr('search.type.activity'), 60, viewByPluginId.get('verstak.activity'), nodes),", "      indexPluginSettings('verstak.activity', tr('search.type.activity'), 60, viewByPluginId.get('verstak.activity'), nodes, workspaceItemByPluginId.get('verstak.activity') || ''),")
replace_once(p, "          detail: { kind: item.toolKind || item.type || '' }", "          detail: { workspaceItemId: item.workspaceItemId || '' }")
replace_once(p, "          detail: { kind: 'files' }", "          detail: { workspaceItemId: item.workspaceItemId || '' }")

# Resume provider items no longer need a category. Keep the DOM/test hook stable anyway.
p = 'frontend/src/lib/shell/TodaySurface.svelte'
replace_once(
    p,
    'data-overview-continue-item={item.category}',
    'data-overview-continue-item={item.category || item.actionKind || item.id}',
)

# The Wails mock must emulate the backend side of Browser's mutation event.
p = 'frontend/src/lib/test/wails-mock.js'
replace_once(p, "    PublishPluginEvent: function () { return Promise.resolve(''); },", """    PublishPluginEvent: function (pluginId, eventName, payload) {
      if (pluginId === 'verstak.browser-inbox' && eventName === 'browser-inbox.storage.mutate') {
        payload = payload || {};
        var settings = Object.assign({}, pluginSettings[pluginId] || {});
        var globalKey = 'captures:global';
        var legacyKey = 'captures';
        var workspacePrefix = 'captures:workspace:';
        var captures = Array.isArray(settings[globalKey]) ? settings[globalKey].map(cloneJson) : [];
        var action = String(payload.action || '');
        if (action === 'migrate') {
          var seen = {};
          var migrated = [];
          Object.keys(settings).filter(function (key) {
            return key === globalKey || key === legacyKey || key.indexOf(workspacePrefix) === 0;
          }).forEach(function (key) {
            var scopedWorkspace = '';
            if (key.indexOf(workspacePrefix) === 0) {
              try { scopedWorkspace = decodeURIComponent(key.slice(workspacePrefix.length)); }
              catch (_) { scopedWorkspace = key.slice(workspacePrefix.length); }
            }
            (Array.isArray(settings[key]) ? settings[key] : []).forEach(function (row) {
              if (!row || typeof row !== 'object') return;
              var item = cloneJson(row);
              if (scopedWorkspace && !item.workspaceRootPath) item.workspaceRootPath = scopedWorkspace;
              if (scopedWorkspace && !item.workspaceName) item.workspaceName = scopedWorkspace;
              var id = String(item.captureId || '');
              if (id && seen[id]) return;
              if (id) seen[id] = true;
              migrated.push(item);
            });
          });
          captures = migrated;
          delete settings[legacyKey];
          Object.keys(settings).forEach(function (key) {
            if (key.indexOf(workspacePrefix) === 0) delete settings[key];
          });
        } else if (action === 'assign') {
          var workspaceRoot = String(payload.workspaceRootPath || '').replace(/^\\/+|\\/+$/g, '');
          captures = captures.map(function (capture) {
            return capture.captureId === payload.captureId
              ? Object.assign({}, capture, { workspaceRootPath: workspaceRoot, workspaceName: workspaceRoot })
              : capture;
          });
        } else if (action === 'processed') {
          captures = captures.map(function (capture) {
            return capture.captureId === payload.captureId
              ? Object.assign({}, capture, { processed: payload.processed === true })
              : capture;
          });
        } else if (action === 'archive') {
          var archiveIds = {};
          (Array.isArray(payload.captureIds) ? payload.captureIds : []).forEach(function (id) { archiveIds[id] = true; });
          captures = captures.map(function (capture) {
            return archiveIds[capture.captureId] ? Object.assign({}, capture, { globalState: 'archived' }) : capture;
          });
        } else if (action === 'restore') {
          captures = captures.map(function (capture) {
            return capture.captureId === payload.captureId ? Object.assign({}, capture, { globalState: 'inbox' }) : capture;
          });
        } else if (action === 'delete' && payload.permanent === true) {
          captures = captures.filter(function (capture) { return capture.captureId !== payload.captureId; });
        }
        settings[globalKey] = captures;
        pluginSettings[pluginId] = settings;
      }
      return Promise.resolve('');
    },""")

# Browser UI is now the real plugin bundle, not the retired handwritten mock.
p = 'frontend/e2e/browser-inbox.spec.js'
replace_all(p, "inbox.locator('.browser-inbox-count')", "inbox.locator('.browser-inbox-toolbar > .browser-inbox-count')", 3)
replace_once(p, "await inbox.locator('[data-browser-inbox-action=\"remove\"]').click();", "await inbox.locator('[data-browser-inbox-action=\"delete-permanently\"]').click();")
replace_once(p, "await expect(inbox.locator('[data-browser-inbox-action=\"remove\"]')).toBeVisible();", "await expect(inbox.locator('[data-browser-inbox-action=\"delete-permanently\"]')).toBeVisible();")
replace_once(p, "await expect(inbox.locator('[data-browser-capture-id=\"capture-e2e-1\"]')).toContainText('Research Report');", "await expect(inbox.locator('[data-browser-capture-id=\"capture-e2e-1\"]')).toContainText('report.txt');")
replace_once(p, "await expect(inbox.locator('.browser-inbox-detail-title')).toHaveText('Research Report');", "await expect(inbox.locator('.browser-inbox-detail-title')).toHaveText('report.txt');")

# Background provider commands legitimately stay registered. Test only platform-test cleanup.
p = 'frontend/e2e/plugin-api-bridge.spec.js'
replace_all(p, "await expect.poll(() => page.evaluate(() => Object.keys(window.__VERSTAK_COMMAND_HANDLERS__ || {}).length)).toBe(0);", "await expect.poll(() => page.evaluate(() => Boolean(window.__VERSTAK_COMMAND_HANDLERS__?.['verstak.platform-test:verstak.platform-test.show-version']))).toBe(false);", 3)
replace_once(p, "await expect.poll(() => page.evaluate(() => Object.keys(window.__VERSTAK_COMMAND_HANDLERS__ || {}).length)).toBe(1);", "await expect.poll(() => page.evaluate(() => Boolean(window.__VERSTAK_COMMAND_HANDLERS__?.['verstak.platform-test:verstak.platform-test.show-version']))).toBe(true);")

# Generic Overview actions use the current workspace item's localized title.
replace_once('frontend/e2e/todo.spec.js', "getByRole('button', { name: 'Open Todos' })", "getByRole('button', { name: 'Open “Todos”' })")

# Provider-owned Overview copy: assert facts/entities, not the shell's retired phrasing.
p = 'frontend/e2e/ux-today.spec.js'
replace_all(p, "toContainText('1 total')", "toContainText('1 note')", 3)
replace_once(p, "    await expect(overview.locator('[data-overview-summary=\"notes\"]')).toContainText('0 recent changes');\n", '')
replace_once(p, "    await expect(overview.locator('[data-overview-summary=\"notes\"]')).toContainText('1 recent change');\n", '')
replace_once(p, "    await expect(attention).toContainText('Deal: Project · 10 min · 2 activities');", "    await expect(attention).toContainText('10 min');\n    await expect(attention).toContainText('2 activities');")
replace_once(p, "    await attention.locator('.overview-attention-row', { hasText: 'Possible journal entry' }).getByRole('button', { name: 'Review candidate' }).click();", "    await attention.locator('.overview-attention-row', { hasText: 'Possible journal entry' }).getByRole('button').click();")
for old, new in [
  ('Edited note "Overview"', 'Overview'),
  ('Changed file "draft.md"', 'draft.md'),
  ('Continue journal entry "Write project summary"', 'Write project summary'),
  ('Captured page "Research Report"', 'Research Report'),
  ('Added journal entry "Write project summary"', 'Write project summary'),
  ('Captured selection "Quote to process"', 'Quote to process'),
]:
    replace_present(p, old, new)
replace_once(p, "    await expect(recent).toContainText('Изменена заметка «Локализация»');", "    await expect(recent).toContainText('Изменена заметка — Локализация');")

# Make the exact-id rule permanent for GlobalSearch too.
p = 'frontend/tests/shell-source-contract-test.mjs'
replace_once(p, "const overviewSurface = read('frontend/src/lib/shell/TodaySurface.svelte');", "const overviewSurface = read('frontend/src/lib/shell/TodaySurface.svelte');\nconst globalSearch = read('frontend/src/lib/shell/GlobalSearch.svelte');")
replace_once(p, "assertExcludes(\n  workspaceHost,\n  'text.includes(kind)',\n  'WorkspaceHost should not guess a workspace tool from arbitrary title/id substrings',\n);", "assertExcludes(\n  workspaceHost,\n  'text.includes(kind)',\n  'WorkspaceHost should not guess a workspace tool from arbitrary title/id substrings',\n);\nassertIncludes(\n  globalSearch,\n  'workspaceItemId:',\n  'GlobalSearch should navigate workspace tools by exact workspace item id',\n);\nassertExcludes(\n  globalSearch,\n  'toolKind:',\n  'GlobalSearch should not encode semantic workspace tool kinds',\n);\nassertExcludes(\n  globalSearch,\n  'detail: { kind:',\n  'GlobalSearch should not dispatch legacy kind-based workspace navigation',\n);")
