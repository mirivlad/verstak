import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

async function openImportSettings(page) {
  await openPluginManager(page);
  const card = page.locator('.plugin-card').filter({ hasText: 'verstak.import' });
  await expect(card).toBeVisible();
  await card.locator('button.btn-settings').click();
  await expect(page.locator('[data-import-step="source"]')).toBeVisible();
  return card;
}

async function confirmAndImport(page) {
  await page.getByText(/I reviewed the proposed structure/).click();
  await page.getByRole('button', { name: /^Import$/ }).click();
  await expect(page.locator('[data-import-result]')).toBeVisible({ timeout: 10000 });
}

test.describe('Official import plugin', () => {
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

  test('imports a reviewed DokuWiki candidate into an isolated run', async ({ page }) => {
    await openImportSettings(page);
    await page.locator('[data-import-select-archive]').click();
    await expect(page.locator('[data-import-candidate]')).toBeVisible();
    await page.locator('[data-import-candidate]').selectOption('dokuwiki:wiki/data');
    await page.locator('[data-import-analyze]').click();
    await expect(page.locator('[data-import-step="structure"]')).toBeVisible();

    const warning = page.locator('[data-import-sensitive-warning]');
    await expect(warning).toBeVisible();
    await expect(warning).toContainText(/passwords|private data/i);
    await expect(warning).not.toContainText('private/passwords.txt');
    await page.locator('[data-import-tree] button').first().click();
    await page.locator('[data-import-node-name]').fill('Knowledge base');
    await expect(page.locator('[data-import-node-type]')).toBeVisible();

    await confirmAndImport(page);
    await expect(page.locator('[data-import-result]')).toContainText('Импортировано/DokuWiki');
  });

  test('imports an Obsidian vault twice with unique runs', async ({ page }) => {
    await openImportSettings(page);
    await page.locator('[data-import-select-directory]').click();
    await page.locator('[data-import-analyze]').click();
    await expect(page.locator('[data-import-step="structure"]')).toBeVisible();
    await confirmAndImport(page);
    await expect(page.locator('[data-import-result]')).toContainText('Импортировано/Obsidian');

    await page.getByRole('button', { name: 'Import another' }).click();
    await page.locator('[data-import-select-directory]').click();
    await page.locator('[data-import-analyze]').click();
    await expect(page.locator('[data-import-step="structure"]')).toBeVisible();
    await confirmAndImport(page);
    await expect(page.locator('[data-import-result]')).toContainText('(2)');
  });

  test('closes source sessions before disable and re-enable', async ({ page }) => {
    const card = await openImportSettings(page);
    await page.locator('[data-import-select-archive]').click();
    await page.locator('.modal[aria-label="Plugin Settings"] .modal-close').click();
    await expect.poll(() => page.evaluate(() => window.__wailsMock.getOpenImportSessionCount())).toBe(0);

    await card.locator('button.btn-disable').click();
    await expect(card.locator('button.btn-enable')).toBeVisible();
    await card.locator('button.btn-enable').click();
    await expect(card.locator('button.btn-disable')).toBeVisible();
  });

  test('confirms and cancels only while apply is cancellable', async ({ page }) => {
    await openImportSettings(page);
    await page.locator('[data-import-select-directory]').click();
    await page.locator('[data-import-analyze]').click();
    await expect(page.locator('[data-import-step="structure"]')).toBeVisible();
    await page.getByText(/I reviewed the proposed structure/).click();
    await page.getByRole('button', { name: /^Import$/ }).click();
    await expect(page.locator('[data-import-step="apply"]')).toBeVisible();
    const cancel = page.locator('[data-import-cancel]');
    await expect(cancel).toBeEnabled();
    page.once('dialog', (dialog) => dialog.accept());
    await cancel.click();
    await expect(page.locator('[data-import-step="structure"]')).toBeVisible();
    await expect(page.getByRole('alert')).toContainText('not completed');
  });
});
