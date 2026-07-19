import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

test.describe('Global search layout', () => {
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

  test('GlobalSearchRenderedExactlyOnce', async ({ page }) => {
    const searchInputs = page.locator('[data-global-search-input]');
    await expect(searchInputs).toHaveCount(1);
  });

  test('GlobalSearchSharesRowWithWorkspaceTitle', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();

    const header = page.locator('[data-main-content-header]');
    await expect(header).toBeVisible({ timeout: 10000 });

    // Title and search are siblings inside the same header
    await expect(header.locator('.main-content-title-text')).toBeVisible();
    await expect(header.locator('[data-global-search-input]')).toBeVisible();

    // Search is in the actions area (right side of header)
    const actions = header.locator('.main-content-actions');
    await expect(actions.locator('[data-global-search-input]')).toBeVisible();
  });

  test('GlobalSearchSharesRowWithToolTitle', async ({ page }) => {
    // Navigate to a sidebar view (e.g., Platform Test)
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();
    await expect(page.locator('.pt-command-result')).toContainText('Command: handled', { timeout: 10000 });

    const header = page.locator('[data-main-content-header]');
    await expect(header).toBeVisible();

    // Title and search are in the same header row
    await expect(header.locator('.main-content-title-text')).toBeVisible();
    await expect(header.locator('.main-content-actions [data-global-search-input]')).toBeVisible();
  });

  test('GlobalSearchHasNoSeparateTopRow', async ({ page }) => {
    // The old .content-header div no longer exists
    await expect(page.locator('.content-header')).toHaveCount(0);

    // Search is inside .main-content-header, not in a standalone top row
    const header = page.locator('[data-main-content-header]');
    await expect(header.locator('[data-global-search-input]')).toBeVisible();

    // Verify no separate search-only container exists above the content title
    // by checking that the first <header> child in content-shell contains both title and search
    const contentShell = page.locator('.content-shell');
    const firstHeader = contentShell.locator('> header').first();
    await expect(firstHeader.locator('.main-content-title-text')).toBeVisible();
    await expect(firstHeader.locator('[data-global-search-input]')).toBeVisible();
  });

  test('SwitchingWorkspaceAndToolKeepsSingleHeaderSearch', async ({ page }) => {
    // Start with workspace
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await expect(page.locator('[data-main-content-header] .main-content-title-text')).toHaveText('Project', { timeout: 10000 });
    await expect(page.locator('[data-global-search-input]')).toHaveCount(1);

    // Switch to workspace tool view
    await page.getByRole('tab', { name: 'Files' }).click();
    await expect(page.locator('.files-root')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[data-main-content-header] .main-content-title-text')).toHaveText('Project');
    await expect(page.locator('[data-global-search-input]')).toHaveCount(1);

    // Open Plugin Manager
    await openPluginManager(page);
    await expect(page.locator('.plugin-manager')).toBeVisible();
    await expect(page.locator('[data-main-content-header]')).toBeVisible();
    await expect(page.locator('[data-global-search-input]')).toHaveCount(1);

    // Switch back to workspace
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await expect(page.locator('[data-main-content-header] .main-content-title-text')).toHaveText('Project', { timeout: 10000 });
    await expect(page.locator('[data-global-search-input]')).toHaveCount(1);
  });

  test('GlobalSearchIsInsideMainContentHeader', async ({ page }) => {
    // Verify DOM nesting: main-content-header > main-content-actions > GlobalSearch
    const header = page.locator('[data-main-content-header]');
    await expect(header).toBeVisible();

    const actions = header.locator('.main-content-actions');
    await expect(actions).toBeVisible();

    const searchInput = actions.locator('[data-global-search-input]');
    await expect(searchInput).toBeVisible();
  });

  test('SearchStaysRightAlignedWithLongTitle', async ({ page }) => {
    // Select a workspace with a name
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await expect(page.locator('[data-main-content-header] .main-content-title-text')).toHaveText('Project', { timeout: 10000 });

    const header = page.locator('[data-main-content-header]');
    const headerBox = await header.boundingBox();

    const titleBox = await header.locator('.main-content-title-text').boundingBox();
    const searchBox = await header.locator('[data-global-search-input]').boundingBox();

    // Title starts from the left
    expect(titleBox.x).toBeLessThan(headerBox.x + headerBox.width / 2);
    // Search is to the right of the title
    expect(searchBox.x).toBeGreaterThan(titleBox.x + titleBox.width - 1);
  });
});
