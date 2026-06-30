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
    await expect(page.locator('.workspace-host')).toBeVisible();
  });

  test('shows user workflow commands before diagnostics and opens workspace tools', async ({ page }) => {
    await page.keyboard.press(process.platform === 'darwin' ? 'Meta+K' : 'Control+K');

    const palette = page.locator('.command-palette');
    await expect(palette).toBeVisible();
    const items = palette.locator('.command-palette-item');
    await expect(items.nth(0)).toHaveAttribute('data-command-id', 'verstak.shell.open-today');
    await expect(items.nth(1)).toHaveAttribute('data-command-id', 'verstak.shell.open-files');
    await expect(items.nth(2)).toHaveAttribute('data-command-id', 'verstak.shell.open-activity');
    await expect(items.nth(3)).toHaveAttribute('data-command-id', 'verstak.shell.open-browser-inbox');

    await palette.locator('[data-command-palette-input]').fill('activity');
    await expect(palette.locator('[data-command-id="verstak.shell.open-activity"]')).toBeVisible();
    await expect(palette.locator('[data-command-id="verstak.platform-test.run-tests"]')).not.toBeVisible();

    await page.keyboard.press('Enter');

    await expect(palette).not.toBeVisible();
    await expect(page.getByRole('tab', { name: 'Activity' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.activity-root')).toBeVisible({ timeout: 10000 });
  });

  test('starts file creation workflows from user commands', async ({ page }) => {
    await page.keyboard.press(process.platform === 'darwin' ? 'Meta+K' : 'Control+K');

    let palette = page.locator('.command-palette');
    await palette.locator('[data-command-palette-input]').fill('markdown');
    await expect(palette.locator('[data-command-id="verstak.shell.create-markdown"]')).toBeVisible();
    await page.keyboard.press('Enter');

    await expect(palette).not.toBeVisible();
    await expect(page.getByRole('tab', { name: 'Files' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.files-root')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[data-files-create-input]')).toBeVisible();

    await page.locator('[data-files-create-input]').fill('Palette Note.md');
    await page.locator('[data-files-create-confirm]').click();
    await expect(page.locator('[data-file-name="Palette Note.md"]')).toBeVisible();

    await page.keyboard.press(process.platform === 'darwin' ? 'Meta+K' : 'Control+K');
    palette = page.locator('.command-palette');
    await palette.locator('[data-command-palette-input]').fill('text file');
    await expect(palette.locator('[data-command-id="verstak.shell.create-text"]')).toBeVisible();
    await page.keyboard.press('Enter');

    await expect(page.locator('[data-files-create-input]')).toBeVisible();
    await page.locator('[data-files-create-input]').fill('Palette Text.txt');
    await page.locator('[data-files-create-confirm]').click();
    await expect(page.locator('[data-file-name="Palette Text.txt"]')).toBeVisible();
  });

  test('runs sync workflow commands', async ({ page }) => {
    await page.evaluate(async () => {
      const err = await window.go.api.App.PluginSyncConfigure('verstak.sync', 'https://sync.example.test', 'alice', 'secret');
      if (err) throw new Error(err);
    });

    await page.keyboard.press(process.platform === 'darwin' ? 'Meta+K' : 'Control+K');

    let palette = page.locator('.command-palette');
    await palette.locator('[data-command-palette-input]').fill('sync now');
    await expect(palette.locator('[data-command-id="verstak.shell.sync-now"]')).toBeVisible();
    await page.keyboard.press('Enter');

    await expect(palette).not.toBeVisible();
    await expect(page.locator('[data-command-palette-status="success"]')).toContainText('Sync Now');
    await expect(page.locator('[data-command-palette-status="success"]')).toContainText('handled');

    await page.keyboard.press(process.platform === 'darwin' ? 'Meta+K' : 'Control+K');
    palette = page.locator('.command-palette');
    await palette.locator('[data-command-palette-input]').fill('sync settings');
    await expect(palette.locator('[data-command-id="verstak.shell.open-sync-settings"]')).toBeVisible();
    await page.keyboard.press('Enter');

    await expect(page.locator('.plugin-manager')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.modal[aria-label="Plugin Settings"]')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.modal-header h3')).toContainText('Sync');
  });
});
