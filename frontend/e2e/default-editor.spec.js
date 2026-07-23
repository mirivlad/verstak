import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

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

  test('soft wrap defaults on, persists, and never changes saved newlines', async ({ page }) => {
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
    const wrap = editor.locator('[data-editor-action="toggle-wrap"]');
    const textarea = editor.locator('[data-editor-textarea]');
    await expect(wrap).toHaveText('Wrap long lines');
    await expect(wrap).toHaveAttribute('aria-pressed', 'true');
    await expect(textarea).toHaveAttribute('wrap', 'soft');

    const exactText = 'one long logical line that only wraps visually and must not gain a newline\nsecond logical line';
    await textarea.fill(exactText);
    await editor.locator('[data-editor-action="save"]').click();
    await expect.poll(async () => page.evaluate(async () => {
      const [content, err] = await window.go.api.App.ReadVaultTextFile('verstak.platform-test', 'Docs/todo.txt');
      if (err) throw new Error(err);
      return content;
    })).toBe(exactText);

    await wrap.click();
    await expect(wrap).toHaveAttribute('aria-pressed', 'false');
    await expect(textarea).toHaveAttribute('wrap', 'off');
    await expect.poll(async () => page.evaluate(async () => {
      const [settings, err] = await window.go.api.App.ReadPluginSettings('verstak.default-editor');
      if (err) throw new Error(err);
      return settings.wrapLongLines;
    })).toBe(false);

    await page.locator('.main-content-header .close-btn').click();
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
    const reopened = page.locator('[data-editor-mode="text"]');
    await expect(reopened.locator('[data-editor-action="toggle-wrap"]')).toHaveAttribute('aria-pressed', 'false');
    await expect(reopened.locator('[data-editor-textarea]')).toHaveAttribute('wrap', 'off');
  });

  test('secret link opens its exact secret and closing it restores the note preview', async ({ page }) => {
    const notePath = 'Notes/Secret Link.md';
    const noteContent = '# Secret link\n\n[Target secret](verstak-secret://target.secret)';

    await page.evaluate(async ({ notePath, noteContent }) => {
      const writeError = await window.go.api.App.WriteVaultTextFile(
        'verstak.platform-test',
        notePath,
        noteContent,
        { createIfMissing: true, overwrite: true },
      );
      if (writeError) throw new Error(writeError);
      const [result, openError] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: notePath,
        extension: '.md',
        context: { sourceView: 'notes', isInsideNotesFolder: true, notesMode: true },
      });
      if (openError) throw new Error(openError);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    }, { notePath, noteContent });

    const note = page.locator('[data-editor-mode="notes-markdown"]');
    await expect(note).toBeVisible({ timeout: 10000 });
    await expect(note.locator('[data-preview]')).toContainText('Target secret');
    await note.locator('.secret-link').click();

    const secrets = page.locator('.secrets-root');
    await expect(secrets).toBeVisible({ timeout: 10000 });
    await expect(secrets.locator('.secrets-item.active .secrets-item-title')).toHaveText('Target secret');
    await expect(secrets.locator('.secrets-card h2')).toHaveText('Target secret');

    await page.locator('.main-content-header .close-btn').click();
    await expect(note).toBeVisible({ timeout: 10000 });
    await expect(note.locator('[data-preview]')).toContainText('Target secret');
    await expect(note.locator('[data-editor-textarea]')).toHaveCount(0);
    await expect(note.locator('[data-save-state]')).toHaveText('');

    const storedContent = await page.evaluate(async ({ notePath }) => {
      const [content, readError] = await window.go.api.App.ReadVaultTextFile('verstak.platform-test', notePath);
      if (readError) throw new Error(readError);
      return content;
    }, { notePath });
    expect(storedContent).toBe(noteContent);
  });

  test('unavailable secret does not fall back to the first secret', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, openError] = await window.go.api.App.OpenWorkbenchResource('verstak.default-editor', {
        kind: 'secret',
        path: 'missing.secret',
        mode: 'view',
      });
      if (openError) throw new Error(openError);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const secrets = page.locator('.secrets-root');
    await expect(secrets).toBeVisible({ timeout: 10000 });
    await expect(secrets.locator('.secrets-item.active')).toHaveCount(0);
    await expect(secrets.locator('.secrets-status.error')).toContainText(/unavailable|недоступен/i);
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
    await expect(page.locator('.main-content-title-text')).toHaveText('readme.md');
  });

  test('opening a note uses its title rather than its technical path', async ({ page }) => {
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
    await expect(page.locator('.main-content-title-text')).toHaveText('Overview');
  });

  test('default-editor plugin is listed as loaded in plugin manager', async ({ page }) => {
    await openPluginManager(page);
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.default-editor' });
    await expect(card).toBeVisible({ timeout: 10000 });
    await expect(card.locator('.status-badge')).toHaveText('loaded');
  });

  test('disable default-editor plugin removes its providers', async ({ page }) => {
    await openPluginManager(page);
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
    await openPluginManager(page);
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
