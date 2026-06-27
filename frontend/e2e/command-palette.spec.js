import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Command Palette', () => {
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

  test('opens with keyboard, filters commands, and executes registered frontend handler', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();
    await expect(page.locator('.pt-root')).toBeVisible({ timeout: 10000 });

    await page.keyboard.press(process.platform === 'darwin' ? 'Meta+K' : 'Control+K');

    const palette = page.locator('.command-palette');
    await expect(palette).toBeVisible();
    await expect(palette.locator('[data-command-id="verstak.platform-test.show-version"]')).toBeVisible();

    await palette.locator('[data-command-palette-input]').fill('version');
    await expect(palette.locator('[data-command-id="verstak.platform-test.show-version"]')).toBeVisible();
    await expect(palette.locator('[data-command-id="verstak.platform-test.run-tests"]')).not.toBeVisible();

    await page.keyboard.press('Enter');

    await expect(palette).not.toBeVisible();
    await expect(page.locator('[data-command-palette-status="success"]')).toContainText('Show Version Info');
    await expect(page.locator('[data-command-palette-status="success"]')).toContainText('handled');
  });

  test('Escape closes the palette without changing current view', async ({ page }) => {
    await page.keyboard.press(process.platform === 'darwin' ? 'Meta+K' : 'Control+K');
    await expect(page.locator('.command-palette')).toBeVisible();

    await page.keyboard.press('Escape');

    await expect(page.locator('.command-palette')).not.toBeVisible();
    await expect(page.locator('.plugin-manager')).toBeVisible();
  });
});
