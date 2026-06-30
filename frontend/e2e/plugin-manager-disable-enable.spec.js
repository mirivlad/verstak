/**
 * Acceptance Test A: Plugin Manager Disable/Enable refresh
 *
 * Scenario:
 * 1. Open Plugin Manager
 * 2. See Platform Test as loaded/enabled
 * 3. Click Disable
 * 4. Verify Enable button appears
 * 5. Verify plugin sidebar item disappears
 * 6. Click Enable
 * 7. Verify Disable button appears
 * 8. Verify plugin sidebar item returns
 */
import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

test.describe('A: Plugin Manager Disable/Enable refresh', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    await openPluginManager(page);
  });

  test.afterEach(async () => {
    consoleCollector.assertNoErrors();
  });

  test('Platform Test plugin is initially visible and enabled', async ({ page }) => {
    // Plugin Manager should show Platform Test plugin card
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await expect(pluginCard).toBeVisible();

    // Status should show "loaded"
    const statusBadge = pluginCard.locator('.status-badge');
    await expect(statusBadge).toHaveText('loaded');

    // Disable button should be visible (not Enable)
    const disableBtn = pluginCard.locator('button.btn-disable');
    await expect(disableBtn).toBeVisible();

    // Sidebar should have Platform Test item
    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' });
    await expect(sidebarItem).toBeVisible();
  });

  test('Disable plugin: button changes to Enable, sidebar item disappears', async ({ page }) => {
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });

    // Click Disable
    const disableBtn = pluginCard.locator('button.btn-disable');
    await expect(disableBtn).toBeVisible();
    await disableBtn.click();

    // Wait for UI to update after disable
    await page.waitForTimeout(500);

    // After disable: Enable button should appear
    const enableBtn = pluginCard.locator('button.btn-enable');
    await expect(enableBtn).toBeVisible({ timeout: 10000 });

    // After disable: sidebar item for this plugin should disappear
    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' });
    await expect(sidebarItem).not.toBeVisible();

    // Status should show "disabled"
    const statusBadge = pluginCard.locator('.status-badge');
    await expect(statusBadge).toHaveText('disabled');
  });

  test('Re-enable plugin: button changes to Disable, sidebar item returns', async ({ page }) => {
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });

    // First disable
    await pluginCard.locator('button.btn-disable').click();
    await page.waitForTimeout(500);

    // Wait for Enable button
    const enableBtn = pluginCard.locator('button.btn-enable');
    await expect(enableBtn).toBeVisible({ timeout: 10000 });

    // Click Enable
    await enableBtn.click();
    await page.waitForTimeout(500);

    // After re-enable: Disable button should appear
    const disableBtn = pluginCard.locator('button.btn-disable');
    await expect(disableBtn).toBeVisible({ timeout: 10000 });

    // Sidebar item should return
    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' });
    await expect(sidebarItem).toBeVisible();

    // Status should show "loaded"
    const statusBadge = pluginCard.locator('.status-badge');
    await expect(statusBadge).toHaveText('loaded');
  });

  test('Disable → Enable full flow in sequence', async ({ page }) => {
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    const sidebar = page.locator('.sidebar');

    // Initial state: enabled
    await expect(pluginCard.locator('button.btn-disable')).toBeVisible();
    await expect(sidebar.locator('.plugin-item').filter({ hasText: 'Platform Test' })).toBeVisible();

    // Disable
    await pluginCard.locator('button.btn-disable').click();
    await page.waitForTimeout(500);
    await expect(pluginCard.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });
    await expect(sidebar.locator('.plugin-item').filter({ hasText: 'Platform Test' })).not.toBeVisible();

    // Enable
    await pluginCard.locator('button.btn-enable').click();
    await page.waitForTimeout(500);
    await expect(pluginCard.locator('button.btn-disable')).toBeVisible({ timeout: 10000 });
    await expect(sidebar.locator('.plugin-item').filter({ hasText: 'Platform Test' })).toBeVisible();
  });
});
