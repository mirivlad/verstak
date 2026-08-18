import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

// The sync status bar is the one place other than the gear where a user reaches
// settings. It asks for its own settings without naming a panel, and that
// request both failed to open the panel and left the window unable to open
// anything else: every click on another section was undone on the next tick.
test.describe('Settings opened for a plugin', () => {
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

  test('the sync status bar opens sync settings, and the rest of the list still works', async ({ page }) => {
    // Both the click target and the panel are the shipped Sync frontend now.
    // The regression this covers is exactly a plugin calling
    // api.ui.openSettings() with no panel id, so it matters that the call comes
    // from the real plugin through the real API rather than from a hand-written
    // CustomEvent in the mock.
    await page.locator('.sync-status-bar').click();

    const syncSection = page.locator('[data-settings-section="plugin:verstak.sync:verstak.sync.settings"]');
    await expect(syncSection).toBeVisible();
    await expect(syncSection).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.sync-settings-surface')).toBeVisible({ timeout: 10000 });

    // The window is not stuck on what was asked for.
    await page.locator('[data-settings-section="general"]').click();
    await expect(page.locator('[data-settings-section="general"]')).toHaveAttribute('aria-selected', 'true');
    await expect(syncSection).toHaveAttribute('aria-selected', 'false');
    await expect(page.locator('.sync-settings-surface')).toHaveCount(0);

    await page.locator('[data-settings-section="diagnostics"]').click();
    await expect(page.locator('[data-settings-collect-diagnostics]')).toBeVisible();
  });

  test('the gear opens settings with every section selectable', async ({ page }) => {
    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-section="plugins"]').click();
    await expect(page.locator('[data-settings-open-plugin-manager]')).toBeVisible();
    await page.locator('[data-settings-section="general"]').click();
    await expect(page.locator('[data-settings-section="general"]')).toHaveAttribute('aria-selected', 'true');
  });
});
