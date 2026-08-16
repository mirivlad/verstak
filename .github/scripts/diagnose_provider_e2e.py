from pathlib import Path
import runpy

runpy.run_path('.github/scripts/fix_provider_e2e_contracts.py', run_name='__main__')


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

# Real Browser has a second informational span with the same class inside filters.
replace_all(
    'frontend/e2e/browser-inbox.spec.js',
    "inbox.locator('.browser-inbox-toolbar .browser-inbox-count')",
    "inbox.locator('.browser-inbox-toolbar > .browser-inbox-count')",
    3,
)

# The real Browser lifecycle is archive/restore/permanent-delete, not the retired remove action.
replace_once(
    'frontend/e2e/browser-inbox.spec.js',
    "await inbox.locator('[data-browser-inbox-action=\"remove\"]').click();",
    "await inbox.locator('[data-browser-inbox-action=\"delete-permanently\"]').click();",
)

# Temporary diagnostics only: establish whether providers have data before the Journal round trip,
# and whether their background handlers/results survive after returning to Overview.
p = 'frontend/e2e/ux-today.spec.js'
helper = """
    const dumpOverviewProviders = async (label) => {
      const state = await page.evaluate(async () => {
        const handlers = window.__VERSTAK_COMMAND_HANDLERS__ || {};
        const keys = Object.keys(handlers).filter((key) => key.includes('provideOverview')).sort();
        const results = {};
        for (const key of keys) {
          try {
            const value = await handlers[key]({ workspaceRootPath: 'Project' }, {});
            results[key] = {
              resume: Array.isArray(value?.resume) ? value.resume.map((item) => item.title) : [],
              recent: Array.isArray(value?.recent) ? value.recent.map((item) => item.title) : [],
              summary: Array.isArray(value?.summary) ? value.summary.map((item) => [item.id, item.count]) : [],
              attention: Array.isArray(value?.attention) ? value.attention.map((item) => item.title) : [],
            };
          } catch (error) {
            results[key] = { error: String(error?.message || error) };
          }
        }
        return { keys, results };
      });
      const count = await overview.locator('[data-overview-section="continue"] [data-overview-continue-item]').count();
      console.log('OVERVIEW_PROVIDER_DIAG', label, 'domResume=' + count, JSON.stringify(state));
    };
"""
replace_once(
    p,
    "    const overview = page.locator('[data-overview-root]');\n    await expect(overview.locator('[data-overview-summary=\"notes\"]')).toContainText('1 note');",
    "    const overview = page.locator('[data-overview-root]');\n" + helper + "    await expect(overview.locator('[data-overview-summary=\"notes\"]')).toContainText('1 note');",
)
replace_once(
    p,
    "    const attention = overview.locator('[data-overview-section=\"attention\"]');",
    "    await dumpOverviewProviders('before-journal');\n    const attention = overview.locator('[data-overview-section=\"attention\"]');",
)
replace_once(
    p,
    "    await page.getByRole('tab', { name: 'Overview' }).click();\n\n    const resume = overview.locator('[data-overview-section=\"continue\"]');",
    "    await page.getByRole('tab', { name: 'Overview' }).click();\n    await page.waitForTimeout(250);\n    await dumpOverviewProviders('after-journal');\n\n    const resume = overview.locator('[data-overview-section=\"continue\"]');",
)
