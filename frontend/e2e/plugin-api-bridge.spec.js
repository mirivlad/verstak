import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('D: Plugin API bridge', () => {
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

  test('platform-test reads and writes settings through scoped API after reload', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();

    const saved = page.locator('.pt-saved-setting');
    await expect(saved).toHaveText('Saved setting: initial value', { timeout: 10000 });

    const input = page.locator('.pt-setting-input');
    await input.fill('persisted through bridge');
    await page.locator('.pt-save-setting').click();
    await expect(saved).toHaveText('Saved setting: persisted through bridge', { timeout: 10000 });

    await page.locator('.sidebar .nav-item').filter({ hasText: 'Plugin Manager' }).click();
    await expect.poll(() => page.evaluate(() => Object.keys(window.__VERSTAK_COMMAND_HANDLERS__ || {}).length)).toBe(0);
    await expect.poll(() => page.evaluate(() => (window.__VERSTAK_EVENT_HANDLERS__?.['verstak.platform-test.echo'] || []).length)).toBe(0);
    await page.locator('button.reload-btn').click();
    await expect(page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' }).locator('.status-badge')).toHaveText('loaded', { timeout: 10000 });

    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();

    await expect(page.locator('.pt-saved-setting')).toHaveText('Saved setting: persisted through bridge', { timeout: 10000 });
    await expect(page.locator('.pt-badge')).toHaveAttribute('data-command-status', 'handled');
    await expect(page.locator('.pt-badge')).toContainText('capability available');
    await expect(page.locator('.pt-command-result')).toContainText('Command: handled 0.1.0 from bundled-frontend');
    await expect(page.locator('.pt-event-result')).toHaveAttribute('data-event-status', 'received');
    await expect(page.locator('.pt-event-result')).toContainText('Event: received hello-event');
    await expect(page.locator('.pt-files-result')).toHaveAttribute('data-files-status', 'ok');
    await expect(page.locator('.pt-files-result')).toContainText('Files: wrote/read/listed/moved/trashed');
    await expect(page.locator('.pt-files-error-result')).toHaveAttribute('data-files-error-status', 'expected');
    await expect(page.locator('.pt-files-error-result')).toContainText('Files error path: rejected reserved-path');

    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Notes/Overview.md',
        extension: '.md',
        context: { sourceView: 'notes', isInsideNotesFolder: true, notesMode: true },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const workbench = page.locator('.workbench-host');
    await expect(workbench).toBeVisible({ timeout: 10000 });
    await expect(workbench.locator('.workbench-title')).toHaveText('Notes/Overview.md');
  });

  test('workbench routes markdown files to default-editor provider', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Docs/readme.md',
        extension: '.md',
        context: { sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const workbench = page.locator('.workbench-host');
    await expect(workbench).toBeVisible({ timeout: 10000 });
    const title = workbench.locator('.workbench-title');
    await expect(title).toHaveText('Docs/readme.md');
  });

  test('workbench shows no-provider fallback when no provider matches', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Images/logo.png',
        extension: '.png',
        context: { sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    await expect(page.locator('[data-workbench-status="no-provider"]')).toBeVisible();
    await expect(page.locator('[data-workbench-status="no-provider"]')).toContainText('No viewer/editor available');
  });

  test('platform-test command and event handlers are cleaned up after leaving plugin view', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();

    await expect(page.locator('.pt-command-result')).toContainText('Command: handled', { timeout: 10000 });
    await expect(page.locator('.pt-event-result')).toHaveAttribute('data-event-status', 'received', { timeout: 10000 });
    await expect.poll(() => page.evaluate(() => Object.keys(window.__VERSTAK_COMMAND_HANDLERS__ || {}).length)).toBe(1);
    await expect.poll(() => page.evaluate(() => (window.__VERSTAK_EVENT_HANDLERS__?.['verstak.platform-test.echo'] || []).length)).toBe(1);

    await page.locator('.sidebar .nav-item').filter({ hasText: 'Plugin Manager' }).click();

    await expect.poll(() => page.evaluate(() => Object.keys(window.__VERSTAK_COMMAND_HANDLERS__ || {}).length)).toBe(0);
    await expect.poll(() => page.evaluate(() => (window.__VERSTAK_EVENT_HANDLERS__?.['verstak.platform-test.echo'] || []).length)).toBe(0);
  });

  test('platform-test cleanup remains empty after disable reload flow', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();
    await expect(page.locator('.pt-command-result')).toContainText('Command: handled', { timeout: 10000 });

    await page.locator('.sidebar .nav-item').filter({ hasText: 'Plugin Manager' }).click();
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await pluginCard.locator('button.btn-disable').click();
    await expect(pluginCard.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });

    await expect.poll(() => page.evaluate(() => Object.keys(window.__VERSTAK_COMMAND_HANDLERS__ || {}).length)).toBe(0);
    await expect.poll(() => page.evaluate(() => (window.__VERSTAK_EVENT_HANDLERS__?.['verstak.platform-test.echo'] || []).length)).toBe(0);
  });

  test('platform-test settings panel loads bundle content returned as raw string', async ({ page }) => {
    await page.locator('.sidebar .nav-item').filter({ hasText: 'Plugin Manager' }).click();

    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await pluginCard.locator('button.btn-settings').click();

    const modal = page.locator('.modal[aria-label="Plugin Settings"]');
    await expect(modal).toBeVisible();
    await expect(modal).toContainText('Platform Test Settings');
    await expect(modal.locator('.host-state.error')).toHaveCount(0);
  });
});
