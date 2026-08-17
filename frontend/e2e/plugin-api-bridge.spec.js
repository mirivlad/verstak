import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

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
    const initial = await page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.platform-test');
      return api.storage.settings.read();
    });
    expect(initial.savedText).toBe('initial value');

    await page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.platform-test');
      await api.storage.settings.write({ savedText: 'changed from e2e' });
    });

    await page.locator('[data-sidebar-item="verstak.platform-test.sidebar"]').click();
    await expect(page.locator('[data-plugin-id="verstak.platform-test"]')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[data-platform-settings-value]')).toHaveValue('changed from e2e');

    await page.locator('[data-platform-settings-value]').fill('changed from panel');
    await page.locator('[data-platform-settings-save]').click();
    await expect.poll(() => page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.platform-test');
      const settings = await api.storage.settings.read();
      return settings.savedText;
    })).toBe('changed from panel');
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
    await expect(page.locator('.main-content-title-text')).toHaveText('readme.md');
  });

  test('workbench shows no-provider fallback when no provider matches', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.platform-test', {
        kind: 'vault-file',
        path: 'Unknown/blob.unsupported',
        extension: '.unsupported',
        context: { sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const fallback = page.locator('[data-workbench-status="no-provider"]');
    await expect(fallback).toBeVisible();
    await expect(fallback).toContainText('No viewer/editor available');
    await expect(fallback).not.toContainText('generic-text');
  });

  test('sync plugin API routes through mocked Wails bridge', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.sync');
      const initial = await api.sync.status();
      await api.sync.testConnection('https://sync.example.test', 'alice', 'secret');
      await api.sync.configure('https://sync.example.test', 'alice', 'secret', 'existing-remote-vault');
      await api.sync.setInterval(15);
      const configured = await api.sync.status();
      const syncNow = await api.sync.now();
      await api.sync.resetKey();
      const reset = await api.sync.status();
      await api.sync.disconnect();
      const disconnected = await api.sync.status();
      return { initial, configured, syncNow, reset, disconnected };
    });

    expect(result.initial.configured).toBe(false);
    expect(result.configured.configured).toBe(true);
    expect(result.configured.intervalMinutes).toBe(15);
    expect(result.configured.remoteVaultId).toBe('existing-remote-vault');
    expect(result.syncNow.pushed).toBe(0);
    expect(result.syncNow.pulled).toBe(0);
    expect(result.reset.hasKey).toBe(false);
    expect(result.disconnected.configured).toBe(false);
  });

  test('files API permanently deletes an existing trash entry through the bridge', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.trash');
      await window.go.api.App.TrashVaultPath('verstak.files', 'Project/Docs/readme.md');
      const before = await api.files.listTrash();
      await api.files.deleteTrash(before[0].trashId);
      const after = await api.files.listTrash();
      return { before, after };
    });

    expect(result.before).toHaveLength(1);
    expect(result.after).toHaveLength(0);
  });

  test('backend plugin events are dispatched to subscribed frontend handlers', async ({ page }) => {
    const observed = await page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.platform-test');
      let payload = null;
      const off = api.events.on('platform-test.event', (value) => { payload = value; });
      window.dispatchEvent(new CustomEvent('verstak:plugin-event', {
        detail: { pluginId: 'verstak.platform-test', eventName: 'platform-test.event', payload: { ok: true } },
      }));
      await new Promise((resolve) => setTimeout(resolve, 0));
      off();
      return payload;
    });
    expect(observed).toEqual({ ok: true });
  });

  test('platform-test command and event handlers are cleaned up after leaving plugin view', async ({ page }) => {
    await page.locator('[data-sidebar-item="verstak.platform-test.sidebar"]').click();
    await expect(page.locator('[data-plugin-id="verstak.platform-test"]')).toBeVisible({ timeout: 10000 });
    await page.locator('[data-workspace-node="workspace-project"]').click();
    await expect(page.locator('[data-plugin-id="verstak.platform-test"]')).toHaveCount(0);

    const cleanup = await page.evaluate(() => window.__verstakPluginRuntimeDiagnostics?.['verstak.platform-test'] || null);
    expect(cleanup?.commands || 0).toBe(0);
    expect(cleanup?.events || 0).toBe(0);
  });

  test('platform-test cleanup remains empty after disable reload flow', async ({ page }) => {
    await page.locator('[data-sidebar-item="verstak.platform-test.sidebar"]').click();
    await expect(page.locator('[data-plugin-id="verstak.platform-test"]')).toBeVisible({ timeout: 10000 });

    await openPluginManager(page);
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await card.locator('button.btn-disable').click();
    await expect(card.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });

    const cleanup = await page.evaluate(() => window.__verstakPluginRuntimeDiagnostics?.['verstak.platform-test'] || null);
    expect(cleanup?.commands || 0).toBe(0);
    expect(cleanup?.events || 0).toBe(0);
  });

  test('platform-test settings panel loads bundle content returned as raw string', async ({ page }) => {
    await openPluginManager(page);
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await expect(card).toBeVisible({ timeout: 10000 });
    await expect(card).toContainText('1 settingsPanels');
  });
});