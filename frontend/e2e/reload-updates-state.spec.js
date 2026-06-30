/**
 * Acceptance Test C: Reload updates UI state
 *
 * Scenario:
 * 1. Change mocked plugin state (e.g. disable a plugin in mock)
 * 2. Click Reload button
 * 3. Verify UI reflects the updated state
 */
import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, setPluginStatus, openPluginManager } from './helpers.js';

test.describe('C: Reload updates UI state', () => {
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

  test('Reload after mock state change reflects new plugin status', async ({ page }) => {
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });

    // Initial state: loaded/enabled
    await expect(pluginCard.locator('.status-badge')).toHaveText('loaded');
    await expect(pluginCard.locator('button.btn-disable')).toBeVisible();

    // Change mock state to disabled (simulating external state change)
    await setPluginStatus(page, 'verstak.platform-test', 'disabled', false);
    await page.waitForTimeout(200);

    // Click Reload button in Plugin Manager header
    const reloadBtn = page.locator('button.reload-btn');
    await expect(reloadBtn).toBeVisible();
    await reloadBtn.click();

    // Wait for reload to complete and UI to update
    await page.waitForTimeout(1000);

    // After reload: status should reflect the disabled state
    await expect(pluginCard.locator('.status-badge')).toHaveText('disabled', { timeout: 10000 });

    // Enable button should appear (since plugin is now disabled)
    await expect(pluginCard.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });

    // Sidebar item should be gone (disabled plugins are filtered from sidebar)
    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' });
    await expect(sidebarItem).not.toBeVisible();
  });

  test('Reload restores plugin after re-enabling in mock', async ({ page }) => {
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });

    // Disable in mock, reload
    await setPluginStatus(page, 'verstak.platform-test', 'disabled', false);
    await page.waitForTimeout(200);
    await page.locator('button.reload-btn').click();
    await page.waitForTimeout(1000);

    // Verify disabled
    await expect(pluginCard.locator('.status-badge')).toHaveText('disabled', { timeout: 10000 });
    await expect(pluginCard.locator('button.btn-enable')).toBeVisible();

    // Re-enable in mock
    await setPluginStatus(page, 'verstak.platform-test', 'loaded', true);
    await page.waitForTimeout(200);

    // Reload again
    await page.locator('button.reload-btn').click();
    await page.waitForTimeout(1000);

    // After reload: should be loaded again
    await expect(pluginCard.locator('.status-badge')).toHaveText('loaded', { timeout: 10000 });
    await expect(pluginCard.locator('button.btn-disable')).toBeVisible();

    // Sidebar item should return
    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' });
    await expect(sidebarItem).toBeVisible();
  });

  test('Reload button is not disabled during normal operation', async ({ page }) => {
    const reloadBtn = page.locator('button.reload-btn');
    await expect(reloadBtn).toBeVisible();
    await expect(reloadBtn).not.toBeDisabled();
  });

  test('Reload handles raw Wails count result without falling into error state', async ({ page }) => {
    await page.evaluate(() => window.__wailsMock.setReloadResponseMode('raw-count'));

    const reloadBtn = page.locator('button.reload-btn');
    await reloadBtn.click();

    await expect(page.locator('.error-state')).toHaveCount(0);
    await expect(page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' })).toBeVisible({ timeout: 10000 });
  });
});
