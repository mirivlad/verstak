import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

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
    const sync = statusBar.locator('[data-plugin-status-handler="SyncStatusBar"]');
    await expect(sync.locator('.mock-sync-status')).toContainText('Synced');
    await sync.locator('.mock-sync-status').click();
    await expect(page.locator('[data-settings-window]')).toBeVisible();
    const statusBox = await statusBar.boundingBox();
    expect(statusBox.height).toBeLessThanOrEqual(36);
  });

  test('the gear opens the settings window, and the Plugin Manager from it', async ({ page }) => {
    await page.locator('[data-settings-menu-button]').click();

    // The gear used to open a dropdown that grew an entry per installed
    // plugin; it now opens one window with those as sections.
    await expect(page.locator('[data-settings-section="plugins"]')).toBeVisible();
    await expect(page.locator('[data-settings-section="plugin:verstak.sync:verstak.sync.settings"]')).toBeVisible();

    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();
    await expect(page.locator('[data-main-content-header] .main-content-title-text')).toHaveText('Platform Diagnostics');

    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-section="plugins"]').click();
    await page.locator('[data-settings-open-plugin-manager]').click();
    await expect(page.locator('.plugin-manager')).toBeVisible();
  });

  test('refreshes statusBarItems after disabling plugin', async ({ page }) => {
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await expect(page.locator('[data-status-item-id="verstak.platform-test.status"]')).toBeVisible();
    await openPluginManager(page);

    await pluginCard.locator('button.btn-disable').click();

    await expect(pluginCard.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[data-status-item-id="verstak.platform-test.status"]')).not.toBeVisible();
  });
});
