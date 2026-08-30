import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

// Where a tool sits among a Deal's tabs is declared in its own manifest. The
// shell used to decide by matching substrings of plugin names against a table
// it carried itself, which meant no plugin outside that table could ever place
// itself anywhere but the end.
test.describe('Workspace tool order', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
  });

  test.afterEach(async () => {
    consoleCollector.assertNoErrors();
  });

  test('tabs follow the order declared by each plugin', async ({ page }) => {
    await page.goto('/');
    await waitForAppReady(page);
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });

    const labels = await page.getByRole('tab').allInnerTexts();
    expect(labels[0]).toBe('Overview');
    // Notes (10) before Files (20) before Todos (30), as their manifests say.
    expect(labels.indexOf('Notes')).toBeLessThan(labels.indexOf('Files'));
    expect(labels.indexOf('Files')).toBeLessThan(labels.indexOf('Todos'));
  });

  test('a plugin that changes its declared order moves', async ({ page }) => {
    // Todos sorts last alphabetically, so if declaring order 1 puts it first
    // the order came from the manifest and not from a fallback.
    await page.addInitScript(() => {
      window.__VERSTAK_MOCK_TOOL_ORDER__ = { 'verstak.todo': 1 };
    });
    await page.goto('/');
    await waitForAppReady(page);
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });

    const labels = await page.getByRole('tab').allInnerTexts();
    expect(labels[0]).toBe('Overview');
    expect(labels.indexOf('Todos')).toBeLessThan(labels.indexOf('Notes'));
    expect(labels.indexOf('Todos')).toBeLessThan(labels.indexOf('Files'));
  });

  test('a tool that declares no order sorts after those that do', async ({ page }) => {
    // Files sorts first alphabetically, so dropping its order must push it
    // behind the ranked tools rather than leaving it where the alphabet puts it.
    await page.addInitScript(() => {
      window.__VERSTAK_MOCK_TOOL_ORDER__ = { 'verstak.files': 0 };
    });
    await page.goto('/');
    await waitForAppReady(page);
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });

    const labels = await page.getByRole('tab').allInnerTexts();
    expect(labels.indexOf('Notes')).toBeLessThan(labels.indexOf('Files'));
    expect(labels.indexOf('Todos')).toBeLessThan(labels.indexOf('Files'));
  });

  test('tool tabs page with directional arrows instead of scrolling', async ({ page }) => {
    await page.setViewportSize({ width: 980, height: 760 });
    await page.goto('/');
    await waitForAppReady(page);
    await page.evaluate(async () => {
      const workspaceId = '11111111-1111-4111-8111-111111111111';
      await window.go.api.App.UpdateWorkspaceV2Tools(workspaceId, [
        'verstak.projects', 'verstak.notes', 'verstak.files', 'verstak.todo',
        'verstak.milestones', 'verstak.git', 'verstak.activity',
        'verstak.browser-inbox', 'verstak.journal', 'verstak.secrets', 'verstak.search',
      ]);
      window.dispatchEvent(new CustomEvent('verstak:workspace-tools-changed', { detail: { workspaceId } }));
    });

    const next = page.locator('[data-workspace-tab-page-next]');
    const previous = page.locator('[data-workspace-tab-page-previous]');
    await expect(next).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('tab', { name: 'Overview', exact: true })).toBeVisible();
    const tabListMetrics = await page.locator('.workspace-tab-list').evaluate((node) => ({
      clientWidth: node.clientWidth,
      scrollWidth: node.scrollWidth,
      pageWidth: node.firstElementChild?.getBoundingClientRect().width,
    }));
    expect(tabListMetrics.scrollWidth).toBeLessThanOrEqual(tabListMetrics.clientWidth);

    await next.click();
    await expect(previous).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Overview', exact: true })).toHaveCount(0);
  });
});
