import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

test.describe('Activity visibility', () => {
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

  test('Activity is a background provider without sidebar, Deal tab, or Overview surface', async ({ page }) => {
    await expect(page.getByRole('tab', { name: 'Activity', exact: true })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Activity', exact: true })).toHaveCount(0);
    await expect(page.locator('[data-overview-provider="verstak.activity"]')).toHaveCount(0);

    await openPluginManager(page);
    await expect(page.locator('.plugin-manager').getByText('Activity', { exact: true })).toBeVisible();
  });
});
