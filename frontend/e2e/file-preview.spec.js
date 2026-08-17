import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

const PIXEL_PNG = 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9ZQmcAAAAASUVORK5CYII=';
const PIXEL_PATH = 'Project/pixel.png';

async function openImage(page) {
  await page.evaluate(async ({ dataBase64, path }) => {
    const writeErr = await window.go.api.App.WriteVaultFileBytes(
      'verstak.files',
      path,
      dataBase64,
      { createIfMissing: true, overwrite: true },
    );
    if (writeErr) throw new Error(writeErr);

    const [result, openErr] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
      kind: 'vault-file',
      path,
      extension: '.png',
      context: { sourceView: 'files' },
    });
    if (openErr) throw new Error(openErr);
    window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
  }, { dataBase64: PIXEL_PNG, path: PIXEL_PATH });
}

test.describe('Shipped File Preview plugin', () => {
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

  test('is part of the installed official catalog and previews images through its real openProvider', async ({ page }) => {
    await openPluginManager(page);
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.file-preview' });
    await expect(card).toBeVisible({ timeout: 10000 });
    await expect(card.locator('.status-badge')).toHaveText('loaded');
    await expect(card.locator('.meta-row').filter({ hasText: 'Contributions:' })).toContainText('1 openProviders');

    await page.keyboard.press('Escape');
    await openImage(page);

    const preview = page.locator('[data-plugin-id="verstak.file-preview"]');
    await expect(preview).toBeVisible({ timeout: 10000 });
    await expect(preview).toHaveAttribute('data-preview-path', PIXEL_PATH);
    const image = preview.locator('[data-preview-image="true"]');
    await expect(image).toBeVisible({ timeout: 10000 });
    await expect(image).toHaveAttribute('src', /^data:image\/png;base64,/);

    await preview.locator('[data-action="open-external"]').click();
    await expect.poll(() => page.evaluate(() => window.__wailsMockExternalOpens)).toEqual([
      { action: 'open', path: PIXEL_PATH },
    ]);
  });

  test('disabling File Preview removes the image openProvider', async ({ page }) => {
    await openPluginManager(page);
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.file-preview' });
    await expect(card).toBeVisible({ timeout: 10000 });
    await card.locator('button.btn-disable').click();
    await expect(card.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });

    await page.keyboard.press('Escape');
    await openImage(page);

    await expect(page.locator('[data-workbench-status="no-provider"]')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[data-plugin-id="verstak.file-preview"]')).toHaveCount(0);
  });
});