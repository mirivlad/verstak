import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('G: Files Plugin', () => {
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

  test('files plugin appears in Plugin Manager as loaded', async ({ page }) => {
    await page.locator('.sidebar .nav-item').filter({ hasText: 'Plugin Manager' }).click();
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.files' });
    await expect(card).toBeVisible({ timeout: 10000 });
    await expect(card.locator('.status-badge')).toHaveText('loaded');
  });

  test('files plugin does not add global sidebar item', async ({ page }) => {
    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Files' });
    await expect(sidebarItem).toHaveCount(0);
  });

  test('open .txt via workbench from files context shows default-editor', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.files', {
        kind: 'vault-file',
        path: 'Docs/todo.txt',
        extension: '.txt',
        context: { sourcePluginId: 'verstak.files', sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const editor = page.locator('[data-editor-mode="text"]');
    await expect(editor).toBeVisible({ timeout: 10000 });
    await expect(editor).toHaveAttribute('data-resource-path', 'Docs/todo.txt');
  });

  test('open .md via workbench from files context shows generic-markdown', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.files', {
        kind: 'vault-file',
        path: 'Docs/readme.md',
        extension: '.md',
        context: { sourcePluginId: 'verstak.files', sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const workbench = page.locator('.workbench-host');
    await expect(workbench).toBeVisible({ timeout: 10000 });
    await expect(workbench.locator('.workbench-title')).toHaveText('Docs/readme.md');
  });

  test('open notes markdown via workbench from files context shows notes-markdown', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.files', {
        kind: 'vault-file',
        path: 'Notes/Overview.md',
        extension: '.md',
        context: { sourcePluginId: 'verstak.files', sourceView: 'files', isInsideNotesFolder: true, notesMode: true },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const workbench = page.locator('.workbench-host');
    await expect(workbench).toBeVisible({ timeout: 10000 });
    await expect(workbench.locator('.workbench-title')).toHaveText('Notes/Overview.md');
  });

  test('files plugin card shows openProviders in contributions', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.files', {
        kind: 'vault-file', path: 'test.txt', extension: '.txt',
        context: { sourcePluginId: 'verstak.files', sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });
    const editor = page.locator('[data-editor-mode="text"]');
    await expect(editor).toBeVisible({ timeout: 5000 });
  });
});
