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
    await expect(statusBar.locator('.vault-status')).toContainText('Vault: open');
    await expect(statusBar.locator('[data-status-item-id="verstak.platform-test.status"]')).toContainText('All Tests Pass');
  });

  test('opens settings menu with plugin manager and plugin settings', async ({ page }) => {
    await page.locator('[data-settings-menu-button]').click();

    await expect(page.locator('[data-settings-action="plugin-manager"]')).toBeVisible();
    await expect(page.locator('[data-settings-panel-id="verstak.sync.settings"]')).toBeVisible();

    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();
    await expect(page.locator('.view-container .view-header h2')).toHaveText('Platform Diagnostics');

    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-action="plugin-manager"]').click();
    await expect(page.locator('.plugin-manager')).toBeVisible();
  });

  test('refreshes statusBarItems after disabling plugin', async ({ page }) => {
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await expect(page.locator('[data-status-item-id="verstak.platform-test.status"]')).toBeVisible();

    await pluginCard.locator('button.btn-disable').click();

    await expect(pluginCard.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[data-status-item-id="verstak.platform-test.status"]')).not.toBeVisible();
  });
});
