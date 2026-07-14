import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Browser workflow', () => {
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

  test('global inbox explains empty capture flow and exposes assignment filters', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Browser' }).click();

    const inbox = page.locator('.browser-inbox-root');
    await expect(inbox).toBeVisible({ timeout: 10000 });
    await expect(inbox.locator('.browser-inbox-title')).toHaveText('Browser');
    await expect(inbox.locator('.browser-inbox-count')).toHaveText('0 items');
    await expect(inbox.locator('.browser-inbox-empty')).toContainText('No browser materials yet');
    await expect(inbox.locator('.browser-inbox-empty')).toContainText('Send a page, selection, or link from the extension');
    await expect(inbox.locator('[data-browser-inbox-filter="status"]')).toBeVisible();
    await expect(inbox.locator('[data-browser-inbox-filter="workspace"]')).toBeVisible();
    await expect(inbox.locator('[data-browser-inbox-action="clear"]')).toBeDisabled();
  });

  test('workspace inbox renders stored captures with conversion actions', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:global': [{
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
    await page.getByRole('tab', { name: 'Browser' }).click();

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

  test('global inbox assigns, reassigns, filters, marks, and deletes captures', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:global': [
          {
            captureId: 'inbox-unassigned',
            capturedAt: '2026-06-30T08:20:00.000Z',
            kind: 'page',
            url: 'https://example.com/unassigned',
            title: 'Unassigned research',
            domain: 'example.com',
          },
          {
            captureId: 'inbox-client-processed',
            capturedAt: '2026-06-30T08:10:00.000Z',
            kind: 'link',
            url: 'https://client.example.com/processed',
            title: 'Processed client link',
            domain: 'client.example.com',
            workspaceRootPath: 'ClientA',
            processed: true,
          },
          {
            captureId: 'inbox-project-open',
            capturedAt: '2026-06-30T08:00:00.000Z',
            kind: 'selection',
            title: 'Project quote',
            text: 'A project quote',
            workspaceRootPath: 'Project',
          },
        ],
      });
    });
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Browser' }).click();

    const inbox = page.locator('.browser-inbox-root');
    await expect(inbox.locator('.browser-inbox-count')).toHaveText('3 items');
    const statusFilter = inbox.locator('[data-browser-inbox-filter="status"]');
    const workspaceFilter = inbox.locator('[data-browser-inbox-filter="workspace"]');

    await statusFilter.selectOption('unassigned');
    await expect(inbox.locator('[data-browser-capture-id="inbox-unassigned"]')).toBeVisible();
    await expect(inbox.locator('[data-browser-capture-id="inbox-client-processed"]')).toHaveCount(0);

    await statusFilter.selectOption('all');
    await workspaceFilter.selectOption('ClientA');
    await expect(inbox.locator('[data-browser-capture-id="inbox-client-processed"]')).toBeVisible();
    await expect(inbox.locator('[data-browser-capture-id="inbox-project-open"]')).toHaveCount(0);

    await workspaceFilter.selectOption('');
    await statusFilter.selectOption('unprocessed');
    await expect(inbox.locator('[data-browser-capture-id="inbox-unassigned"]')).toBeVisible();
    await expect(inbox.locator('[data-browser-capture-id="inbox-project-open"]')).toBeVisible();
    await expect(inbox.locator('[data-browser-capture-id="inbox-client-processed"]')).toHaveCount(0);

    await statusFilter.selectOption('all');
    const assignment = inbox.locator('[data-browser-inbox-assignment="inbox-unassigned"]');
    await assignment.selectOption('ClientA');
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginSettings('verstak.browser-inbox');
      const settings = Array.isArray(result) ? result[0] : result;
      return settings['captures:global'].find((capture) => capture.captureId === 'inbox-unassigned').workspaceRootPath;
    })).toBe('ClientA');

    await inbox.locator('[data-browser-inbox-assignment="inbox-unassigned"]').selectOption('Project');
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginSettings('verstak.browser-inbox');
      const settings = Array.isArray(result) ? result[0] : result;
      return settings['captures:global'].find((capture) => capture.captureId === 'inbox-unassigned').workspaceRootPath;
    })).toBe('Project');

    await inbox.locator('[data-browser-inbox-action="clear-assignment"]').click();
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginSettings('verstak.browser-inbox');
      const settings = Array.isArray(result) ? result[0] : result;
      return settings['captures:global'].find((capture) => capture.captureId === 'inbox-unassigned').workspaceRootPath || '';
    })).toBe('');

    await inbox.locator('[data-browser-inbox-action="toggle-processed"]').click();
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginSettings('verstak.browser-inbox');
      const settings = Array.isArray(result) ? result[0] : result;
      return settings['captures:global'].find((capture) => capture.captureId === 'inbox-unassigned').processed;
    })).toBe(true);

    await inbox.locator('[data-browser-inbox-action="toggle-processed"]').click();
    await inbox.locator('[data-browser-inbox-action="remove"]').click();
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginSettings('verstak.browser-inbox');
      const settings = Array.isArray(result) ? result[0] : result;
      return settings['captures:global'].some((capture) => capture.captureId === 'inbox-unassigned');
    })).toBe(false);
  });
});
