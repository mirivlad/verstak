import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Browser Inbox workflow', () => {
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

  test('workspace inbox explains empty capture flow and primary actions', async ({ page }) => {
    await page.getByRole('tab', { name: 'Browser Inbox' }).click();

    const inbox = page.locator('.browser-inbox-root');
    await expect(inbox).toBeVisible({ timeout: 10000 });
    await expect(inbox.locator('.browser-inbox-title')).toContainText('Browser Inbox');
    await expect(inbox.locator('.browser-inbox-count')).toHaveText('0 items');
    await expect(inbox.locator('.browser-inbox-empty')).toContainText('No browser captures yet');
    await expect(inbox.locator('.browser-inbox-empty')).toContainText('send a page, selection, link, or file from the browser extension');
    await expect(inbox.locator('[data-browser-inbox-action="clear"]')).toBeDisabled();
  });

  test('workspace inbox renders stored captures with conversion actions', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:workspace:Project': [{
          captureId: 'capture-e2e-1',
          capturedAt: '2026-06-30T08:00:00.000Z',
          kind: 'file',
          url: 'https://example.com/report',
          title: 'Research Report',
          domain: 'example.com',
          text: 'Selected page text',
          fileName: 'report.txt',
          fileText: 'Report file contents',
          workspaceRootPath: 'Project',
          browserName: 'Firefox',
        }],
      });
    });
    await page.getByRole('tab', { name: 'Browser Inbox' }).click();

    const inbox = page.locator('.browser-inbox-root');
    await expect(inbox.locator('.browser-inbox-count')).toHaveText('1 item');
    await expect(inbox.locator('[data-browser-capture-id="capture-e2e-1"]')).toContainText('Research Report');
    await expect(inbox.locator('.browser-inbox-detail-title')).toHaveText('Research Report');
    await expect(inbox.locator('.browser-inbox-meta')).toContainText('example.com');
    await expect(inbox.locator('.browser-inbox-text').first()).toContainText('Selected page text');
    await expect(inbox.locator('[data-browser-inbox-action="create-note"]')).toBeVisible();
    await expect(inbox.locator('[data-browser-inbox-action="create-link"]')).toBeVisible();
    await expect(inbox.locator('[data-browser-inbox-action="create-file"]')).toBeVisible();
    await expect(inbox.locator('[data-browser-inbox-action="remove"]')).toBeVisible();
  });
});
