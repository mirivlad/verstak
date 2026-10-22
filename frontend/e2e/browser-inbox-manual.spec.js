import { test, expect } from '@playwright/test';
import { waitForAppReady, resetMockState, setupConsoleCollector } from './helpers.js';

const storedCaptures = page => page.evaluate(async () => {
  const [settings, error] = await window.go.api.App.ReadPluginSettings('verstak.browser-inbox');
  if (error) throw new Error(error);
  return settings['captures:global'] || [];
});

test.describe('Manual browser materials', () => {
  let consoleCollector;
  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Browser' }).click();
  });
  test.afterEach(() => consoleCollector.assertNoErrors());

  test('adds a Deal material without losing a capture received while the form is open', async ({ page }) => {
    await page.locator('[data-browser-inbox-filter="status"]').selectOption('processed');
    await page.locator('[data-browser-inbox-filter="search"]').fill('unrelated');
    await page.getByRole('button', { name: 'Add material', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: 'Add material', exact: true });
    await expect(dialog.getByLabel('Link', { exact: true })).toBeFocused();
    await dialog.getByLabel('Link', { exact: true }).fill('https://example.test/guide');
    await dialog.getByLabel('Title', { exact: true }).fill('Saved guide');
    const quote = '  useful quote\nsecond line\n';
    await dialog.getByLabel('Text', { exact: true }).fill(quote);
    await dialog.getByLabel('Deal', { exact: true }).selectOption({ label: 'Project' });
    await page.evaluate(async () => {
      const [settings] = await window.go.api.App.ReadPluginSettings('verstak.browser-inbox');
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', { ...settings, 'captures:global': [{ captureId: 'arrived-during-form', kind: 'link', title: 'Incoming material', url: 'https://incoming.test/', capturedAt: '2026-10-01T00:00:00Z' }] });
    });
    await dialog.getByRole('button', { name: 'Add', exact: true }).click();
    await expect(dialog).toBeHidden();
    await expect(page.locator('.browser-inbox-detail-title')).toHaveText('Saved guide');
    await expect(page.locator('.browser-inbox-row')).toHaveCount(2);
    const records = await storedCaptures(page);
    expect(records).toHaveLength(2);
    expect(records.find(record => record.source === 'manual')).toMatchObject({ title: 'Saved guide', url: 'https://example.test/guide', text: quote, kind: 'selection', workspaceRootPath: 'Project', workspaceState: 'active' });
    expect(records.find(record => record.source === 'manual').workspaceId).toBeTruthy();
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Journal' }).click();
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Browser' }).click();
    await expect(page.locator('.browser-inbox-row')).toHaveCount(2);
    expect(await storedCaptures(page)).toEqual(records);
  });

  test('validates input, contains focus, cancels, and supports text without a URL', async ({ page }) => {
    const add = page.getByRole('button', { name: 'Add material', exact: true });
    await add.click();
    const dialog = page.getByRole('dialog', { name: 'Add material', exact: true });
    await page.keyboard.press('Shift+Tab');
    await expect(dialog.getByRole('button', { name: 'Close', exact: true })).toBeFocused();
    await page.keyboard.press('Shift+Tab');
    await expect(dialog.getByRole('button', { name: 'Add', exact: true })).toBeFocused();
    await dialog.getByRole('button', { name: 'Add', exact: true }).click();
    await expect(dialog.getByRole('alert')).toContainText('Enter a link or some text');
    await dialog.getByLabel('Link', { exact: true }).fill('javascript:alert(1)');
    await dialog.getByRole('button', { name: 'Add', exact: true }).click();
    await expect(dialog.getByRole('alert')).toContainText('http:// or https://');
    await page.keyboard.press('Escape');
    await expect(dialog).toBeHidden();
    await expect(add).toBeFocused();
    expect(await storedCaptures(page)).toHaveLength(0);
    await add.click();
    await expect(dialog.getByLabel('Link', { exact: true })).toHaveValue('');
    await dialog.getByLabel('Text', { exact: true }).fill('First line\nSecond line');
    await dialog.getByRole('button', { name: 'Add', exact: true }).click();
    await expect(dialog).toBeHidden();
    expect(await storedCaptures(page)).toEqual([expect.objectContaining({ title: 'First line', url: '', text: 'First line\nSecond line', workspaceRootPath: '', workspaceState: 'unassigned', source: 'manual' })]);
  });

  test('keeps input after an interrupted acknowledgement and retry updates one record', async ({ page }) => {
    await page.evaluate(() => {
      const upsert = window.go.api.App.UpsertPluginSettingRecord;
      let interrupted = false;
      window.go.api.App.UpsertPluginSettingRecord = async (...args) => {
        const result = await upsert(...args);
        if (!interrupted) { interrupted = true; throw new Error('acknowledgement interrupted'); }
        return result;
      };
    });
    await page.getByRole('button', { name: 'Add material', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: 'Add material', exact: true });
    await dialog.getByLabel('Link', { exact: true }).fill('https://retry.test/');
    await dialog.getByLabel('Title', { exact: true }).fill('Original title');
    await dialog.getByRole('button', { name: 'Add', exact: true }).click();
    await expect(dialog.getByRole('alert')).toContainText('Your input is still here');
    await expect(dialog.getByLabel('Title', { exact: true })).toHaveValue('Original title');
    expect(await storedCaptures(page)).toHaveLength(1);
    await dialog.getByLabel('Title', { exact: true }).fill('Updated title');
    await dialog.getByRole('button', { name: 'Add', exact: true }).click();
    await expect(dialog).toBeHidden();
    expect(await storedCaptures(page)).toEqual([expect.objectContaining({ title: 'Updated title', url: 'https://retry.test/' })]);
  });

  test('adds text directly to the current Deal and converts it to a note', async ({ page }) => {
    await page.getByRole('treeitem', { name: 'Project', exact: true }).click();
    await page.getByRole('tab', { name: 'Browser', exact: true }).click();
    await page.getByRole('button', { name: 'Add material', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: 'Add material', exact: true });
    await expect(dialog).toContainText('Deal: Project');
    await expect(dialog.getByRole('combobox')).toHaveCount(0);
    await dialog.getByLabel('Title', { exact: true }).fill('Manual note');
    await dialog.getByLabel('Text', { exact: true }).fill('Text captured without an extension.');
    await dialog.getByRole('button', { name: 'Add', exact: true }).click();
    await expect(dialog).toBeHidden();
    await expect(page.locator('.browser-inbox-kind')).toHaveText('Text');
    const [capture] = await storedCaptures(page);
    expect(capture).toMatchObject({ title: 'Manual note', url: '', workspaceRootPath: 'Project', source: 'manual' });
    await page.locator('[data-browser-inbox-action="create-note"]').click();
    await expect.poll(() => page.evaluate(async () => (await window.go.api.App.ReadVaultTextFile('verstak.platform-test', 'Project/Notes/Manual_note.md'))[0]))
      .toContain('Text captured without an extension.');
    await expect.poll(async () => (await storedCaptures(page)).find(record => record.captureId === capture.captureId)?.globalState).toBe('archived');
  });
});
