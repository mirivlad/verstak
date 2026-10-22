import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector } from './helpers.js';

async function openNote(page, content, path = 'Project/Notes/Admin.md') {
  await page.evaluate(async ({ content, path }) => {
    await window.go.api.App.WriteVaultTextFile('verstak.platform-test', path, content, { createIfMissing: true, overwrite: true });
    const [result, error] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
      kind: 'vault-file', path, extension: '.md', context: { notesMode: true, notesScopePath: 'Project' },
    });
    if (error) throw new Error(error);
    window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
  }, { content, path });
  return page.locator('[data-editor-mode="notes-markdown"]');
}

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await waitForAppReady(page);
});

test('forgot password requires acknowledgement and starts a new store', async ({ page }) => {
  const console = setupConsoleCollector(page);
  await page.evaluate(() => window.__wailsMock.lockSecrets());
  await page.locator('.sidebar .plugin-item').filter({ hasText: 'Secrets' }).click();
  await page.locator('[data-secret-forgot-password]').click();
  await expect(page.locator('[data-secret-archive-confirm]')).toBeDisabled();
  await page.locator('[data-secret-archive-cancel]').click();
  await expect(page.locator('[data-secret-master-password]')).toBeVisible();
  await page.locator('[data-secret-forgot-password]').click();
  await page.locator('[data-secret-archive-acknowledge]').check();
  await page.locator('[data-secret-archive-confirm]').click();
  await expect(page.locator('.secrets-status')).toContainText('secrets-archives');
  await page.locator('[data-secret-master-password]').fill('new-test-master');
  await page.locator('[data-secret-master-password-confirm]').fill('new-test-master');
  await page.locator('[data-secret-unlock]').click();
  await expect(page.locator('[data-secret-search]')).toBeVisible();
  expect(await page.evaluate(async () => (await window.go.api.App.PluginSecretsList('verstak.secrets'))[0])).toEqual([]);
  console.assertNoErrors();
});

test('secret link unlocks once, copies and edits in a dialog above the note', async ({ page, context }) => {
  const console = setupConsoleCollector(page);
  await context.grantPermissions(['clipboard-read', 'clipboard-write']);
  await page.evaluate(async () => {
    await window.go.api.App.PluginSecretsWrite('verstak.secrets', { id: 'target.secret', title: 'Target secret', username: 'target-user', value: 'target-value', scope: { kind: 'workspace', workspaceRootPath: 'Project' } });
    window.__wailsMock.lockSecrets();
    window.__secretUnlockCount = 0;
    const original = window.go.api.App.PluginSecretsUnlock;
    window.go.api.App.PluginSecretsUnlock = (...args) => { window.__secretUnlockCount++; return original(...args); };
  });
  const source = 'Сайт. Админка: [логин/пароль](verstak-secret://target.secret)';
  const editor = await openNote(page, source);
  await editor.locator('.secret-link').click();
  const dialog = page.locator('[data-resource-dialog]');
  await expect(dialog.locator('[data-secret-master-password]')).toBeVisible();
  async function expectControlsInsideDialog() {
    const outside = await dialog.evaluate(root => {
      const bounds = root.querySelector('[role="dialog"]').getBoundingClientRect();
      return [...root.querySelectorAll('input, textarea, button')].filter(node => {
        if (!node.getClientRects().length) return false;
        const rect = node.getBoundingClientRect();
        return rect.top < bounds.top - 1 || rect.bottom > bounds.bottom + 1 || rect.left < bounds.left - 1 || rect.right > bounds.right + 1;
      }).map(node => node.getAttribute('data-secret-dialog-value') ?? node.tagName);
    });
    expect(outside).toEqual([]);
  }
  await dialog.locator('[data-secret-master-password]').fill('test-master-password');
  await dialog.locator('[data-secret-master-password]').press('Enter');
  const username = dialog.locator('[data-secret-dialog-username]');
  const value = dialog.locator('[data-secret-dialog-value]');
  await expect(username).toHaveValue('target-user');
  await expect(value).toHaveValue('target-value');
  await expect(value).toHaveAttribute('readonly', '');
  await expectControlsInsideDialog();
  await dialog.locator('[data-secret-copy-username]').click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('target-user');
  await dialog.locator('[data-secret-copy-value]').click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('target-value');
  await dialog.locator('[data-secret-edit]').click();
  await expectControlsInsideDialog();
  await dialog.locator('[data-secret-username]').fill('new-admin');
  await dialog.locator('[data-secret-value]').fill('new-secret-value');
  await page.evaluate(() => {
    const original = window.go.api.App.PluginSecretsWrite;
    let fail = true;
    window.go.api.App.PluginSecretsWrite = (...args) => {
      if (fail) { fail = false; return Promise.resolve([null, 'test write failure']); }
      return original(...args);
    };
  });
  await dialog.locator('[data-secret-save]').click();
  await expect(dialog.locator('.secrets-status.error')).toBeVisible();
  await expect(dialog.locator('[data-secret-value]')).toHaveValue('new-secret-value');
  await dialog.locator('[data-secret-save]').click();
  await expect(value).toHaveValue('new-secret-value');
  await expectControlsInsideDialog();
  await dialog.locator('[data-resource-dialog-close]').click();
  await expect(editor.locator('.secret-link')).toHaveText('логин/пароль');
  await expect(editor.locator('.secret-link')).toBeFocused();
  await editor.locator('.secret-link').click();
  await expect(value).toHaveValue('new-secret-value');
  await expect(dialog.locator('[data-secret-master-password]')).toHaveCount(0);
  expect(await page.evaluate(() => window.__secretUnlockCount)).toBe(1);
  const saved = await page.evaluate(async () => (await window.go.api.App.ReadVaultTextFile('verstak.platform-test', 'Project/Notes/Admin.md'))[0]);
  expect(saved).toBe(source);
  const record = await page.evaluate(async () => (await window.go.api.App.PluginSecretsRead('verstak.secrets', 'target.secret'))[0]);
  expect(record.scope).toEqual({ kind: 'workspace', workspaceRootPath: 'Project' });
  await page.evaluate(() => {
    window.__wailsMock.lockSecrets();
    window.dispatchEvent(new CustomEvent('verstak:vault-opened'));
  });
  await expect(dialog).toHaveCount(0);
  console.assertNoErrors();
});

test('picker inserts a stable link with selected text and only lists current Deal and shared secrets', async ({ page }) => {
  const console = setupConsoleCollector(page);
  await page.evaluate(async () => {
    for (const [id, scope] of [['project.admin', 'Project'], ['other.admin', 'Other']]) {
      await window.go.api.App.PluginSecretsWrite('verstak.secrets', { id, title: id, username: 'admin', value: 'never-in-note', scope: { kind: 'workspace', workspaceRootPath: scope } });
    }
  });
  const editor = await openNote(page, 'Сайт. Админка: логин/пароль');
  await editor.locator('[data-editor-action="mode-edit"]').click();
  const content = editor.locator('.cm-content');
  await content.click();
  await content.press('Control+End');
  for (let i = 0; i < 'логин/пароль'.length; i++) await content.press('Shift+ArrowLeft');
  await editor.locator('[data-editor-action="insert-secret-link"]').click();
  const picker = editor.locator('[data-secret-picker]');
  await expect(picker.locator('[data-secret-picker-id="project.admin"]')).toBeVisible();
  await expect(picker.locator('[data-secret-picker-id="first.secret"]')).toBeVisible();
  await expect(picker.locator('[data-secret-picker-id="other.admin"]')).toHaveCount(0);
  await picker.locator('[data-secret-picker-id="project.admin"]').click();
  await expect(editor.locator('.cm-content')).toContainText('verstak-secret://project.admin');
  await editor.locator('[data-editor-action="save"]').click();
  await expect.poll(() => page.evaluate(async () => (await window.go.api.App.ReadVaultTextFile('verstak.platform-test', 'Project/Notes/Admin.md'))[0]))
    .toBe('Сайт. Админка: [логин/пароль](verstak-secret://project.admin)');
  await editor.locator('[data-editor-action="mode-preview"]').click();
  await editor.locator('.secret-link').click();
  await expect(page.locator('[data-secret-dialog-value]')).toHaveValue('never-in-note');
  console.assertNoErrors();
});

test('missing secret stays in the dialog and leaves the note open', async ({ page }) => {
  const console = setupConsoleCollector(page);
  const editor = await openNote(page, '[Old secret](verstak-secret://missing.secret)');
  await editor.locator('.secret-link').click();
  const dialog = page.locator('[data-resource-dialog]');
  await expect(dialog.locator('.secrets-status.error')).toContainText(/unavailable|could not|недоступ|не удалось/i);
  await expect(dialog.locator('[data-secret-dialog-value]')).toHaveCount(0);
  await dialog.locator('[data-resource-dialog-close]').click();
  await expect(editor.locator('.secret-link')).toHaveText('Old secret');
  console.assertNoErrors();
});
