import { test, expect } from '@playwright/test';
import { waitForAppReady, resetMockState, openPluginManager } from './helpers.js';

test.describe('UX follow-up fixes', () => {
  test.beforeEach(async ({ page }) => {
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
  });

  test('global search stays available after opening tool sidebar views', async ({ page }) => {
    const search = page.locator('[data-global-search-input]');
    await expect(search).toBeVisible();

    await page.locator('.sidebar .nav-item').filter({ hasText: 'Activity' }).click();
    await expect(page.locator('.activity-root')).toBeVisible({ timeout: 10000 });
    await expect(search).toBeVisible();

    await page.locator('.sidebar .nav-item').filter({ hasText: 'Browser Inbox' }).click();
    await expect(page.locator('.browser-inbox-root')).toBeVisible({ timeout: 10000 });
    await expect(search).toBeVisible();
  });

  test('global search types ahead across workspaces and file contents with keyboard layout fallback', async ({ page }) => {
    const search = page.locator('[data-global-search-input]');

    await search.fill('Зкщоусе');
    await expect(page.locator('[data-global-search-results]')).toContainText('Project', { timeout: 10000 });

    await search.fill('project file');
    await expect(page.locator('[data-global-search-results]')).toContainText('project-only.txt', { timeout: 10000 });
  });

  test('global search folder results open the workspace Files context', async ({ page }) => {
    const search = page.locator('[data-global-search-input]');
    await search.fill('Project/Notes');

    const folderResult = page.locator('[data-global-search-result-type="Folder"][data-global-search-result-path="Project/Notes"]');
    await expect(folderResult).toBeVisible({ timeout: 10000 });
    await folderResult.click();

    await expect(page.locator('.workspace-title')).toHaveText('Project', { timeout: 10000 });
    await expect(page.getByRole('tab', { name: 'Files' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.files-root')).toBeVisible();
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');
    await expect(page.locator('.files-item')).toContainText('Overview.md');
  });

  test('plugin settings modal gives complex panels enough space', async ({ page }) => {
    await openPluginManager(page);
    await page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' }).getByRole('button', { name: 'Settings' }).click();

    const modal = page.locator('.modal[aria-label="Plugin Settings"]');
    await expect(modal).toBeVisible({ timeout: 10000 });
    const box = await modal.boundingBox();
    expect(box.width).toBeGreaterThanOrEqual(760);
    expect(box.height).toBeGreaterThanOrEqual(560);
  });
});
