import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('F: Default Editor Plugin', () => {
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

  test('open .txt file shows plain text editor mode', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Docs/todo.txt',
        extension: '.txt',
        context: { sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const editor = page.locator('[data-editor-mode="text"]');
    await expect(editor).toBeVisible({ timeout: 10000 });
    await expect(editor).toHaveAttribute('data-resource-path', 'Docs/todo.txt');
    await expect(editor).toHaveAttribute('data-request-mode', 'view');
    const textarea = editor.locator('[data-editor-textarea]');
    await expect(textarea).toBeVisible();
    await expect(textarea).toHaveValue('Buy groceries\nWrite tests');
  });

  test('editor supports markdown toolbar split save reopen and revert', async ({ page }) => {
    await page.evaluate(async () => {
      const err = await window.go.api.App.WriteVaultTextFile(
        'verstak.platform-test',
        'Project/Notes/editing.md',
        '# Editing\n\nplain text',
        { createIfMissing: true, overwrite: true }
      );
      if (err) throw new Error(err);
      const [result, openErr] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Project/Notes/editing.md',
        extension: '.md',
        context: { sourceView: 'files', isInsideNotesFolder: true, notesMode: true },
      });
      if (openErr) throw new Error(openErr);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const editor = page.locator('[data-editor-mode="notes-markdown"]');
    await expect(editor).toBeVisible({ timeout: 10000 });
    await expect(editor.locator('[data-notes-badge]')).toBeVisible();

    await editor.locator('[data-editor-mode-button="edit"]').click();
    const textarea = editor.locator('[data-editor-textarea]');
    await expect(textarea).toBeVisible();
    await textarea.fill('plain text');
    await textarea.selectText();
    await editor.locator('[data-md-action="bold"]').click();
    await expect(textarea).toHaveValue('**plain text**');

    await editor.locator('[data-md-action="heading"]').click();
    await expect(textarea).toHaveValue('# **plain text**');
    await expect(editor.locator('[data-save-state]')).toContainText('Modified');

    await editor.locator('[data-editor-mode-button="split"]').click();
    await expect(editor.locator('[data-editor-textarea]')).toBeVisible();
    await expect(editor.locator('[data-preview]')).toBeVisible();
    await expect(editor.locator('[data-preview]')).toContainText('plain text');

    await textarea.press(process.platform === 'darwin' ? 'Meta+S' : 'Control+S');
    await expect(editor.locator('[data-save-state]')).toContainText('Saved');

    await textarea.fill('discard me');
    page.once('dialog', (dialog) => dialog.accept());
    await editor.locator('[data-editor-action="reload"]').click();
    await expect(textarea).toHaveValue('# **plain text**');

    await page.evaluate(async () => {
      const [result, openErr] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Project/Notes/editing.md',
        extension: '.md',
        context: { sourceView: 'files', isInsideNotesFolder: true, notesMode: true },
      });
      if (openErr) throw new Error(openErr);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });
    await expect(page.locator('[data-editor-mode="notes-markdown"] [data-preview]')).toContainText('plain text', { timeout: 10000 });
  });

  test('open .md file outside Notes routes to highest-priority provider', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Docs/readme.md',
        extension: '.md',
        context: { sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const workbench = page.locator('.workbench-host');
    await expect(workbench).toBeVisible({ timeout: 10000 });
    const title = workbench.locator('.workbench-title');
    await expect(title).toHaveText('Docs/readme.md');
  });

  test('open .md with notes context routes to highest-priority provider', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Notes/Overview.md',
        extension: '.md',
        context: { sourceView: 'notes', isInsideNotesFolder: true, notesMode: true },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const workbench = page.locator('.workbench-host');
    await expect(workbench).toBeVisible({ timeout: 10000 });
    const title = workbench.locator('.workbench-title');
    await expect(title).toHaveText('Notes/Overview.md');
  });

  test('default-editor plugin is listed as loaded in plugin manager', async ({ page }) => {
    await page.locator('.sidebar .nav-item').filter({ hasText: 'Plugin Manager' }).click();
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.default-editor' });
    await expect(card).toBeVisible({ timeout: 10000 });
    await expect(card.locator('.status-badge')).toHaveText('loaded');
  });

  test('disable default-editor plugin removes its providers', async ({ page }) => {
    await page.locator('.sidebar .nav-item').filter({ hasText: 'Plugin Manager' }).click();
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.default-editor' });
    await card.locator('button.btn-disable').click();
    await expect(card.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });

    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Docs/todo.txt',
        extension: '.txt',
        context: { sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    await expect(page.locator('[data-workbench-status="no-provider"]')).toBeVisible({ timeout: 10000 });
  });

  test('default-editor plugin card shows openProviders contribution count', async ({ page }) => {
    await page.locator('.sidebar .nav-item').filter({ hasText: 'Plugin Manager' }).click();
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.default-editor' });
    await expect(card).toBeVisible({ timeout: 10000 });
    await expect(card.locator('.meta-row').filter({ hasText: 'Contributions:' })).toContainText('3 openProviders');
  });

  test('default-editor does not add sidebar item', async ({ page }) => {
    const sidebarItems = page.locator('.sidebar .plugin-item');
    const count = await sidebarItems.count();
    const hasDefaultEditor = await page.locator('.sidebar .plugin-item').filter({ hasText: /default.editor/i }).count();
    expect(hasDefaultEditor).toBe(0);
  });

  test('platform-test workbench buttons open files via default-editor', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();
    await expect(page.locator('.pt-command-result')).toContainText('Command: handled', { timeout: 10000 });

    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Docs/todo.txt',
        extension: '.txt',
        context: { sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const editor = page.locator('[data-editor-mode="text"]');
    await expect(editor).toBeVisible({ timeout: 10000 });
    await expect(editor).toHaveAttribute('data-resource-path', 'Docs/todo.txt');
  });
});
