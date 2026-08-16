from pathlib import Path


def replace_once(path, old, new):
    p = Path(path)
    text = p.read_text()
    assert old in text, f'pattern not found in {path}'
    assert text.count(old) == 1, f'pattern not unique in {path}'
    p.write_text(text.replace(old, new, 1))

# Preserve the already-established shell time semantics/keys.
p = Path('frontend/src/lib/shell/TodaySurface.svelte')
s = p.read_text()
start = s.index('  function relativeTime(value) {')
end = s.index('\n\n  function absoluteTime(value)', start)
old = s[start:end]
new = '''  function relativeTime(value) {
    const ms = new Date(value).getTime();
    if (!Number.isFinite(ms)) return tr('overview.time.none');
    const diff = Date.now() - ms;
    if (diff < 0) return absoluteTime(value);
    const minute = 60 * 1000;
    const hour = 60 * minute;
    const day = 24 * hour;
    if (diff < minute) return tr('overview.time.now');
    if (diff < hour) return tr('overview.time.minutes', { count: Math.floor(diff / minute) });
    if (diff < day) return tr('overview.time.hours', { count: Math.floor(diff / hour) });
    if (diff < 2 * day) return tr('overview.time.yesterday');
    if (diff < 7 * day) return tr('overview.time.days', { count: Math.floor(diff / day) });
    const date = new Date(ms);
    return date.toLocaleDateString(locale === 'ru' ? 'ru-RU' : 'en-US', {
      month: 'short',
      day: 'numeric',
      year: date.getFullYear() === new Date().getFullYear() ? undefined : 'numeric',
    });
  }'''
p.write_text(s[:start] + new + s[end:])

replace_once('frontend/src/lib/i18n/catalogs/ru.js', "  'overview.openTool': 'Открыть: {tool}',", "  'overview.openTool': 'Открыть «{tool}»',")
replace_once('frontend/src/lib/i18n/catalogs/en.js', "  'overview.openTool': 'Open {tool}',", "  'overview.openTool': 'Open “{tool}”',")

# E2E must use the real provider plugin locale catalogs, not English fallbacks.
p = Path('frontend/src/lib/test/wails-mock.js')
s = p.read_text()
import_anchor = "import journalSource from '../../../../../verstak-official-plugins/plugins/journal/frontend/src/index.js?raw';\n"
assert import_anchor in s
imports = import_anchor + """import notesEnCatalog from '../../../../../verstak-official-plugins/plugins/notes/locales/en.json';
import notesRuCatalog from '../../../../../verstak-official-plugins/plugins/notes/locales/ru.json';
import activityEnCatalog from '../../../../../verstak-official-plugins/plugins/activity/locales/en.json';
import activityRuCatalog from '../../../../../verstak-official-plugins/plugins/activity/locales/ru.json';
import browserEnCatalog from '../../../../../verstak-official-plugins/plugins/browser-inbox/locales/en.json';
import browserRuCatalog from '../../../../../verstak-official-plugins/plugins/browser-inbox/locales/ru.json';
import journalEnCatalog from '../../../../../verstak-official-plugins/plugins/journal/locales/en.json';
import journalRuCatalog from '../../../../../verstak-official-plugins/plugins/journal/locales/ru.json';
import todoEnCatalog from '../../../../../verstak-official-plugins/plugins/todo/locales/en.json';
import todoRuCatalog from '../../../../../verstak-official-plugins/plugins/todo/locales/ru.json';
"""
s = s.replace(import_anchor, imports, 1)

anchor = "  var russianPluginNames = {\n"
assert anchor in s
catalogs = """  var realOverviewPluginCatalogs = {
    'verstak.notes': { en: notesEnCatalog, ru: notesRuCatalog },
    'verstak.activity': { en: activityEnCatalog, ru: activityRuCatalog },
    'verstak.browser-inbox': { en: browserEnCatalog, ru: browserRuCatalog },
    'verstak.journal': { en: journalEnCatalog, ru: journalRuCatalog },
    'verstak.todo': { en: todoEnCatalog, ru: todoRuCatalog }
  };

"""
s = s.replace(anchor, catalogs + anchor, 1)

func_anchor = "  function mockPluginCatalog(pluginId, locale) {\n"
assert func_anchor in s
s = s.replace(func_anchor, func_anchor + "    var realCatalog = realOverviewPluginCatalogs[pluginId] && realOverviewPluginCatalogs[pluginId][locale];\n    if (realCatalog) return Object.assign({}, realCatalog);\n", 1)
s = s.replace("searchProviders: 'label', worklogProviders: 'label', statusBarItems: 'label'", "searchProviders: 'label', worklogProviders: 'label', overviewProviders: 'label', statusBarItems: 'label'", 1)
p.write_text(s)

# The old plugin-specific copy is intentionally replaced by a generic, natural label.
p = Path('frontend/e2e/visual/overview.visual.js')
s = p.read_text()
s = s.replace("toContainText('Открыть заметки')", "toContainText('Открыть «Заметки»')")
s = s.replace("not.toContainText('Open Notes')", "not.toContainText('Open Notes')")
# Guard the provider localization boundary in the populated visual state.
needle = "    await expect(overview.locator('[data-overview-section=\"attention\"]')).toContainText('План выпуска');\n"
assert needle in s
s = s.replace(needle, needle + "    await expect(overview).not.toContainText('overview.time.');\n    await expect(overview).not.toContainText('Possible journal entry');\n    await expect(overview.locator('[data-overview-section=\"attention\"]')).toContainText('Возможная запись журнала');\n", 1)
p.write_text(s)
