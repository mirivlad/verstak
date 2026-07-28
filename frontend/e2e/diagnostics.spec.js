import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

// Somebody reporting a problem should not have to be told to find a terminal
// and pass a flag. The report is written where they can read it before sending
// it, and it says so.
test.describe('Diagnostics', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
  });

  test.afterEach(async () => {
    consoleCollector.assertNoErrors();
  });

  test('settings has a diagnostics section that names the log and writes a report', async ({ page }) => {
    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-section="diagnostics"]').click();

    await expect(page.locator('[data-settings-diagnostics-log]')).toContainText('verstak-2026-01-01-000000.log');
    await expect(page.locator('[data-settings-diagnostics-report]')).toHaveCount(0);

    await page.locator('[data-settings-collect-diagnostics]').click();
    await expect(page.locator('[data-settings-diagnostics-report]')).toContainText('verstak-diagnostics-2026-01-01-000000.txt');
    await expect(page.locator('[data-settings-diagnostics-error]')).toHaveCount(0);
  });

  test('diagnostics is findable by what it is for, not only by its name', async ({ page }) => {
    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-search]').fill('crash');
    await expect(page.locator('[data-settings-section="diagnostics"]')).toBeVisible();
    await expect(page.locator('[data-settings-section="plugins"]')).toHaveCount(0);
  });

  test('the plugin manager is still reachable beside it', async ({ page }) => {
    await openPluginManager(page);
    await expect(page.locator('.plugin-manager')).toBeVisible();
  });
});
