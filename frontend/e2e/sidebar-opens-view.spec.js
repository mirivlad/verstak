/**
 * Acceptance Test B: Sidebar opens plugin view by item.view, not item.id
 *
 * Data:
 * - sidebar item id = verstak.platform-test.sidebar
 * - sidebar item view = verstak.platform-test.diagnostics
 *
 * Scenario:
 * 1. Click sidebar item "Platform Test"
 * 2. Verify diagnostics view is opened (verstak.platform-test.diagnostics)
 * 3. Verify NOT opened empty container by sidebar id
 */
import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('B: Sidebar opens plugin view by item.view', () => {
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

  test('Sidebar item exists with correct label', async ({ page }) => {
    await expect(page.locator('.sidebar .nav-item').filter({ hasText: 'Plugin Manager' })).not.toBeVisible();
    await expect(page.locator('.sidebar .plugin-item').filter({ hasText: 'Activity' })).toBeVisible();
    await expect(page.locator('.sidebar .plugin-item').filter({ hasText: 'Browser' })).toBeVisible();

    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' });
    await expect(sidebarItem).toBeVisible();
  });

  test('Global Activity and Browser sidebar items open plugin views', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Activity' }).click();
    await expect(page.locator('[data-main-content-header] .main-content-title-text')).toHaveText('Activity', { timeout: 10000 });

    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Browser' }).click();
    await expect(page.locator('[data-main-content-header] .main-content-title-text')).toHaveText('Browser', { timeout: 10000 });
  });

  test('selected global tool remains visibly active through navigation and sidebar reloads', async ({ page }) => {
    const activity = page.locator('.sidebar .plugin-item').filter({ hasText: 'Activity' });
    const browserInbox = page.locator('.sidebar .plugin-item').filter({ hasText: 'Browser' });

    await activity.click();
    await expect(activity).toHaveAttribute('aria-current', 'page');
    await expect(activity).toHaveClass(/is-active/);

    await page.evaluate(() => {
      window.dispatchEvent(new CustomEvent('verstak:plugins-changed'));
    });
    await expect(activity).toHaveAttribute('aria-current', 'page');

    await page.evaluate(() => {
      window.dispatchEvent(new CustomEvent('verstak:open-view', {
        detail: { viewId: 'verstak.browser-inbox.view', pluginId: 'verstak.browser-inbox' },
      }));
    });
    await expect(page.locator('[data-main-content-header] .main-content-title-text')).toHaveText('Browser', { timeout: 10000 });
    await expect(browserInbox).toHaveAttribute('aria-current', 'page');
    await expect(browserInbox).toHaveClass(/is-active/);
    await expect(activity).not.toHaveAttribute('aria-current', 'page');

    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await expect(browserInbox).not.toHaveAttribute('aria-current', 'page');
  });

  test('Click sidebar item opens diagnostics view by view ID, not sidebar ID', async ({ page }) => {
    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' });
    await expect(sidebarItem).toBeVisible();

    // Click the sidebar item
    await sidebarItem.click();
    await page.waitForTimeout(500);

    // View container should be visible
    const viewContainer = page.locator('.view-container');
    await expect(viewContainer).toBeVisible();

    // The view header should show "Platform Diagnostics" (from view contribution title)
    // This proves the view was opened by item.view = "verstak.platform-test.diagnostics"
    // NOT by item.id = "verstak.platform-test.sidebar"
    const viewHeader = page.locator('[data-main-content-header] .main-content-title-text');
    await expect(viewHeader).toHaveText('Platform Diagnostics', { timeout: 10000 });

    // The view should NOT show "View ... not found" error
    // (which would happen if it tried to open by sidebar item id)
    await expect(viewContainer).not.toHaveText(/not found/);

    // The view should NOT show an empty container message
    const emptyView = viewContainer.locator('.empty');
    await expect(emptyView).not.toBeVisible();
  });

  test('View header shows correct title from view contribution', async ({ page }) => {
    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' });
    await sidebarItem.click();
    await page.waitForTimeout(500);

    // Verify the view title comes from the view contribution (item.view)
    // NOT from the sidebar item (item.id)
    const viewHeader = page.locator('[data-main-content-header] .main-content-title-text');
    await expect(viewHeader).toHaveText('Platform Diagnostics', { timeout: 10000 });

    // Should NOT show sidebar item id as view title
    await expect(viewHeader).not.toHaveText('verstak.platform-test.sidebar');
  });
});
