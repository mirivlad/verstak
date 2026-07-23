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
    await expect(page.locator('.main-content-header [data-global-search-input]')).toBeVisible();

    await page.locator('.sidebar .nav-item').filter({ hasText: 'Activity' }).click();
    await expect(page.locator('.activity-root')).toBeVisible({ timeout: 10000 });
    await expect(search).toBeVisible();
    await expect(page.locator('.main-content-header [data-global-search-input]')).toBeVisible();

    await page.locator('.sidebar .nav-item').filter({ hasText: 'Browser' }).click();
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

    await expect(page.locator('.main-content-title-text')).toHaveText('Project', { timeout: 10000 });
    await expect(page.getByRole('tab', { name: 'Files' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.files-root')).toBeVisible();
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');
    await expect(page.locator('.files-item')).toContainText('Overview.md');
  });

  test('global search opens indexed browser inbox results', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:workspace:Project': [{
          captureId: 'capture-search-1',
          capturedAt: '2026-06-30T08:00:00.000Z',
          kind: 'page',
          url: 'https://example.com/research',
          title: 'Research Search Result',
          domain: 'example.com',
          text: 'Searchable captured browser text',
          workspaceRootPath: 'Project',
          browserName: 'Firefox',
        }],
      });
    });

    const search = page.locator('[data-global-search-input]');
    await search.fill('Research Search Result');
    const result = page.locator('[data-global-search-result-type="Browser"]').filter({ hasText: 'Research Search Result' });
    await expect(result).toBeVisible({ timeout: 10000 });
    await result.click();

    await expect(page.locator('.browser-inbox-root')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.browser-inbox-detail-title')).toHaveText('Research Search Result');
  });

  test('workspace Search input keeps focus while typing', async ({ page }) => {
    await page.evaluate(async () => {
      const metadata = await window.go.api.App.GetWorkspaceMetadata('Project');
      await window.go.api.App.UpdateWorkspaceMetadata('Project', {
        workspaceTools: [...metadata.workspaceTools, 'verstak.search'],
      });
    });
    await page.locator('.wt-label').filter({ hasText: 'Test' }).click();
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await page.getByRole('tab', { name: 'Search' }).click();
    const searchInput = page.locator('[data-search-input="query"]');
    await expect(searchInput).toBeVisible({ timeout: 10000 });

    await searchInput.click();
    await page.keyboard.press('p');
    await expect(searchInput).toBeFocused();
    await page.keyboard.press('r');
    await expect(searchInput).toBeFocused();
    await expect(searchInput).toHaveValue('pr');
  });

  test('mobile workspace layout gives content full width below the sidebar', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.reload();
    await waitForAppReady(page);

    const workspaceBox = await page.locator('.workspace-host').boundingBox();
    const sidebarBox = await page.locator('.sidebar').boundingBox();
    expect(workspaceBox.width).toBeGreaterThan(340);
    expect(workspaceBox.y).toBeGreaterThan(sidebarBox.y + sidebarBox.height - 1);
    await expect(page.locator('.main-content-header [data-global-search-input]')).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Overview' })).toBeVisible();

    const hasHorizontalOverflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
    expect(hasHorizontalOverflow).toBe(false);
  });

  test('plugin settings modal gives complex panels enough space', async ({ page }) => {
    await openPluginManager(page);
    await page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' }).getByRole('button', { name: 'Settings' }).click();

    const modal = page.locator('.modal[aria-label="Plugin Settings"]');
    await expect(modal).toBeVisible({ timeout: 10000 });
    const box = await modal.boundingBox();
    expect(box.width).toBeGreaterThanOrEqual(760);
    expect(box.height).toBeGreaterThanOrEqual(560);
    const bodyBox = await modal.locator('.modal-body').boundingBox();
    const surfaceBox = await modal.locator('.plugin-settings-surface').boundingBox();
    expect(surfaceBox.width / bodyBox.width).toBeGreaterThanOrEqual(0.88);
    expect(surfaceBox.width / bodyBox.width).toBeLessThanOrEqual(0.92);
    expect(Math.abs((surfaceBox.x - bodyBox.x) - (bodyBox.width - surfaceBox.width) / 2)).toBeLessThanOrEqual(2);
  });
});
