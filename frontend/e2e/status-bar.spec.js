import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Status Bar host', () => {
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

  test('renders enabled plugin statusBarItems', async ({ page }) => {
    const statusBar = page.locator('.status-bar');
    await expect(statusBar).toBeVisible();
    await expect(statusBar.locator('[data-status-item-id="verstak.platform-test.status"]')).toContainText('All Tests Pass');
  });

  test('refreshes statusBarItems after disabling plugin', async ({ page }) => {
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await expect(page.locator('[data-status-item-id="verstak.platform-test.status"]')).toBeVisible();

    await pluginCard.locator('button.btn-disable').click();

    await expect(pluginCard.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[data-status-item-id="verstak.platform-test.status"]')).not.toBeVisible();
  });
});
