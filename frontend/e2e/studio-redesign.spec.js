import { test, expect } from '@playwright/test';
import { waitForAppReady } from './helpers.js';

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await waitForAppReady(page);
});

test('the visible theme control saves the choice and restores it after restart', async ({ page }) => {
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.getByRole('button', { name: 'Light theme', exact: true }).click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  expect(await page.evaluate(async () => (await window.go.api.App.GetAppSettings()).theme)).toBe('light');
  await page.reload();
  await waitForAppReady(page);
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await page.getByRole('button', { name: 'Dark theme', exact: true }).click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
});

test('system appearance follows the OS and stays in sync with the header', async ({ page }) => {
  await page.emulateMedia({ colorScheme: 'light' });
  await page.locator('[data-settings-menu-button]').click();
  await page.locator('[data-settings-theme="system"]').click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await expect(page.locator('[data-settings-theme="system"]')).toHaveAttribute('aria-checked', 'true');
  await page.emulateMedia({ colorScheme: 'dark' });
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  expect(await page.evaluate(async () => (await window.go.api.App.GetAppSettings()).theme)).toBe('system');
  await page.getByRole('button', { name: 'Light theme', exact: true }).click();
  await expect(page.locator('[data-settings-theme="light"]')).toHaveAttribute('aria-checked', 'true');
});

test('a failed theme save keeps the previous choice and supports retry', async ({ page }) => {
  await page.evaluate(() => {
    const update = window.go.api.App.UpdateAppSettings;
    let fail = true;
    window.go.api.App.UpdateAppSettings = (patch) => {
      if (patch.theme && fail) { fail = false; return Promise.resolve('test write failed'); }
      return update(patch);
    };
  });
  await page.getByRole('button', { name: 'Light theme', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Could not save the theme');
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.getByRole('button', { name: 'Light theme', exact: true }).click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
  await expect(page.getByRole('alert')).toHaveCount(0);
});

test('primary and destructive actions keep readable text in both themes, including hover', async ({ page }) => {
  for (const theme of ['dark', 'light']) {
    if (theme === 'light') await page.getByRole('button', { name: 'Light theme', exact: true }).click();
    for (const action of ['create', 'trash']) {
    if (action === 'create') await page.locator('button[title="New Deal"]').click();
    else {
      await page.locator('.wt-label').filter({ hasText: /^Project$/ }).click({ button: 'right' });
      await page.locator('[role=menu]').getByRole('button', { name: 'Move Deal to Trash', exact: true }).click();
    }
    const button = page.locator(action === 'create' ? '[data-new-deal-create]' : '.vt-btn-d');
    await expect(button).toBeEnabled();
    for (const hover of [false, true]) {
      if (hover) await button.hover(); else await page.mouse.move(1, 1);
      const contrast = await button.evaluate(node => {
        const style = getComputedStyle(node);
        const rgb = color => color.match(/[\d.]+/g).slice(0, 3).map(Number);
        const luminance = color => rgb(color).map(value => value / 255).map(value => value <= .04045 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4).reduce((sum, value, i) => sum + value * [.2126, .7152, .0722][i], 0);
        const a = luminance(style.color), b = luminance(style.backgroundColor);
        return (Math.max(a, b) + .05) / (Math.min(a, b) + .05);
      });
      expect(contrast).toBeGreaterThanOrEqual(4.5);
    }
    await page.keyboard.press('Escape');
    }
  }
});

test('the tools action opens Deal settings even with collapsed navigation', async ({ page }) => {
  await page.getByRole('button', { name: 'Close navigation', exact: true }).click();
  await expect(page.locator('.sidebar')).toBeHidden();
  await page.locator('[data-workspace-config]').click();
  await expect(page.locator('[data-workspace-edit-modal]')).toBeVisible();
  await expect(page.locator('[data-workspace-edit-name]')).toHaveValue('Project');
  await page.keyboard.press('Escape');
  await expect(page.locator('[data-workspace-config]')).toBeFocused();
  await page.locator('[data-navigation-toggle]').click();
  await expect(page.locator('.sidebar')).toBeVisible();
});

test('mobile creation keeps the dialog usable and closes navigation on success', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.locator('[data-navigation-toggle]').click();
  await page.locator('[data-create-deal]').click();
  await expect(page.locator('[data-new-deal-name]')).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(page.locator('[data-new-deal-dialog]')).toBeHidden();
  await expect(page.locator('.sidebar')).toBeVisible();
  await page.locator('[data-create-deal]').click();
  await page.locator('[data-new-deal-name]').fill('Mobile Studio');
  await page.locator('[data-new-deal-create]').click();
  await expect(page.locator('[data-new-deal-dialog]')).toBeHidden();
  await expect(page.locator('.sidebar')).toBeHidden();
  await expect(page.locator('.main-content-title-text')).toHaveText('Mobile Studio');
  await expect(page.locator('.content-shell')).not.toHaveAttribute('inert');
  await page.locator('[data-theme-toggle]').click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
});

test('a short window keeps the Deal tree and settings reachable', async ({ page }) => {
  await page.setViewportSize({ width: 800, height: 600 });
  await expect(page.locator('[data-workspace-tree]')).toBeVisible();
  const tree = await page.locator('[data-workspace-tree]').boundingBox();
  expect(tree.height).toBeGreaterThan(100);
  await page.locator('[data-settings-menu-button]').click();
  await expect(page.locator('[data-settings-theme="light"]')).toBeVisible();
});

test('the editor follows theme changes without losing the document', async ({ page }) => {
  await page.locator('[data-overview-summary="notes"]').click();
  await expect(page.locator('.notes-root')).toBeVisible();
  expect((await page.locator('.notes-btn.primary').boundingBox()).height).toBeLessThanOrEqual(36);
  expect((await page.locator('.notes-btn.primary svg').boundingBox()).width).toBe(16);
  await page.evaluate(async () => {
    const [result, error] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
      kind: 'vault-file', path: 'Docs/todo.txt', extension: '.txt', context: { sourceView: 'files' },
    });
    if (error) throw new Error(error);
    window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
  });
  const editor = page.locator('.de-root');
  await expect(editor).toBeVisible();
  await page.locator('[data-theme-toggle]').click();
  await expect(editor.locator('.cm-content')).toContainText('Buy groceries');
  expect(await editor.evaluate(node => getComputedStyle(node).backgroundColor)).toBe('rgb(255, 255, 255)');
  expect(await editor.evaluate(node => getComputedStyle(node).color)).toBe('rgb(36, 36, 45)');
});
