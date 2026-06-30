import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('UX Today workspace flow', () => {
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

  test('workspace opens with Today before plugin tools', async ({ page }) => {
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });

    const tabs = page.getByRole('tab');
    await expect(tabs.nth(0)).toHaveText('Today');
    await expect(tabs.nth(1)).toHaveText('Files');
    await expect(page.getByRole('tab', { name: 'Today' })).toHaveAttribute('aria-selected', 'true');

    const today = page.locator('.today-root');
    await expect(today).toBeVisible();
    await expect(today.locator('[data-today-section="captured"]')).toContainText('Captured');
    await expect(today.locator('[data-today-section="activity"]')).toContainText('Recent Activity');
    await expect(today.locator('[data-today-section="worklog"]')).toContainText('Worklog Suggestions');
    await expect(today.locator('[data-today-section="quick-actions"]')).toContainText('Quick Actions');
    await expect(today).toContainText('No browser captures yet');
    await expect(today).toContainText('No activity events yet');
  });

  test('Today quick action opens Browser Inbox workspace tool', async ({ page }) => {
    await page.locator('[data-today-action="browser-inbox"]').click();

    await expect(page.getByRole('tab', { name: 'Browser Inbox' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.browser-inbox-root')).toBeVisible({ timeout: 10000 });
  });
});
