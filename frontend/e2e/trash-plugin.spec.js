import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

async function openTrash(page) {
  await page.locator('.sidebar .plugin-item').filter({ hasText: 'Trash' }).click();
  await expect(page.locator('.trash-root')).toBeVisible({ timeout: 10000 });
}

async function createTrashEntry(page, path, content, deletedAt) {
  return page.evaluate(async ({ path, content, deletedAt }) => {
    const api = window.createPluginAPI('verstak.files');
    await api.files.writeText(path, content, { createIfMissing: true });
    const entry = await api.files.trash(path);
    window.__wailsMock.setTrashDeletedAt(entry.trashId, deletedAt);
    api.dispose();
    return entry;
  }, { path, content, deletedAt });
}

test.describe('L: Global Trash Plugin', () => {
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

  test('moves a Files item into global Trash without exposing Trash metadata in Files', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await page.getByRole('tab', { name: 'Files' }).click();
    await expect(page.locator('.files-root')).toBeVisible({ timeout: 10000 });

    await page.locator('[data-files-action="new-text"]').click();
    await page.locator('[data-files-create-input]').fill('GlobalTrash.txt');
    await page.locator('[data-files-create-confirm]').click();
    await expect(page.locator('[data-file-name="GlobalTrash.txt"]')).toBeVisible();
    await page.locator('[data-file-name="GlobalTrash.txt"]').click();
    await page.locator('[data-files-action="trash"]').click();
    await page.locator('.files-modal-btn.confirm').click();
    await expect(page.locator('[data-file-name="GlobalTrash.txt"]')).toHaveCount(0);
    await expect(page.locator('[data-files-action="trash-view"]')).toHaveCount(0);

    await openTrash(page);
    const row = page.locator('[data-trash-row]').filter({ hasText: 'GlobalTrash.txt' });
    await expect(row).toBeVisible();
    await expect(row).toContainText('Project');
    await expect(row).toContainText('Project/GlobalTrash.txt');
  });

  test('filters and sorts global Trash and restores without overwriting a conflict', async ({ page }) => {
    const older = await createTrashEntry(page, 'Project/Older.txt', 'older original', '2026-06-20T08:00:00.000Z');
    const newest = await createTrashEntry(page, 'Test/Newest.txt', 'newest original', '2026-06-29T08:00:00.000Z');
    const restore = await createTrashEntry(page, 'Project/Restore.txt', 'restore original', '2026-06-25T08:00:00.000Z');
    const conflict = await page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.files');
      await api.files.writeText('Project/Conflict.txt', 'trashed content', { createIfMissing: true });
      const entry = await api.files.trash('Project/Conflict.txt');
      await api.files.writeText('Project/Conflict.txt', 'existing content', { createIfMissing: true });
      window.__wailsMock.setTrashDeletedAt(entry.trashId, '2026-06-24T08:00:00.000Z');
      api.dispose();
      return entry;
    });

    await openTrash(page);
    await page.locator('[data-trash-filter-workspace]').selectOption('Project');
    await expect(page.locator(`[data-trash-row="${older.trashId}"]`)).toBeVisible();
    await expect(page.locator(`[data-trash-row="${newest.trashId}"]`)).toHaveCount(0);

    await page.locator('[data-trash-filter-workspace]').selectOption('');
    await page.locator('[data-trash-filter-search]').fill('Newest');
    await expect(page.locator(`[data-trash-row="${newest.trashId}"]`)).toBeVisible();
    await expect(page.locator(`[data-trash-row="${older.trashId}"]`)).toHaveCount(0);

    await page.locator('[data-trash-filter-search]').fill('');
    await page.locator('[data-trash-sort]').selectOption('date-asc');
    await expect(page.locator('[data-trash-row]').first()).toHaveAttribute('data-trash-row', older.trashId);

    await page.locator(`[data-trash-restore="${restore.trashId}"]`).click();
    await expect(page.locator(`[data-trash-row="${restore.trashId}"]`)).toHaveCount(0);
    await expect.poll(() => page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.files');
      const content = await api.files.readText('Project/Restore.txt');
      api.dispose();
      return content;
    })).toBe('restore original');

    await page.locator(`[data-trash-restore="${conflict.trashId}"]`).click();
    await expect(page.locator('[data-trash-status]')).toContainText('Restore blocked');
    await expect(page.locator(`[data-trash-row="${conflict.trashId}"]`)).toBeVisible();
    await expect.poll(() => page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.files');
      const content = await api.files.readText('Project/Conflict.txt');
      api.dispose();
      return content;
    })).toBe('existing content');
  });

  test('requires confirmation before permanently deleting a Trash item', async ({ page }) => {
    const entry = await createTrashEntry(page, 'Test/Permanent.txt', 'remove forever', '2026-06-26T08:00:00.000Z');
    await openTrash(page);

    await page.locator(`[data-trash-delete="${entry.trashId}"]`).click();
    await expect(page.locator(`[data-trash-confirm="${entry.trashId}"]`)).toBeVisible();
    await expect(page.locator(`[data-trash-row="${entry.trashId}"]`)).toBeVisible();

    await page.locator(`[data-trash-confirm-delete="${entry.trashId}"]`).click();
    await expect(page.locator(`[data-trash-row="${entry.trashId}"]`)).toHaveCount(0);
    await expect.poll(() => page.evaluate(async (trashId) => {
      const api = window.createPluginAPI('verstak.files');
      const entries = await api.files.listTrash();
      api.dispose();
      return entries.some((item) => item.trashId === trashId);
    }, entry.trashId)).toBe(false);
  });
});
