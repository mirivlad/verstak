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
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();

    const saved = page.locator('.pt-saved-setting');
    await expect(saved).toHaveText('Saved setting: initial value', { timeout: 10000 });

    const input = page.locator('.pt-setting-input');
    await input.fill('persisted through bridge');
    await page.locator('.pt-save-setting').click();
    await expect(saved).toHaveText('Saved setting: persisted through bridge', { timeout: 10000 });

    await openPluginManager(page);
    await expect.poll(() => page.evaluate(() => Boolean(window.__VERSTAK_COMMAND_HANDLERS__?.['verstak.platform-test:verstak.platform-test.show-version']))).toBe(false);
    await expect.poll(() => page.evaluate(() => (window.__VERSTAK_EVENT_HANDLERS__?.['verstak.platform-test.echo'] || []).length)).toBe(0);
    await page.locator('[data-plugin-manager-reload]').click();
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
    await expect(page.locator('.main-content-title-text')).toHaveText('Overview');
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
      api.dispose();
      return { initial, configured, syncNow, reset, disconnected };
    });

    expect(result.initial.statusLabel).toBe('disabled');
    expect(result.configured.configured).toBe(true);
    expect(result.configured.serverUrl).toBe('https://sync.example.test');
    expect(result.configured.vaultId).toBe('existing-remote-vault');
    expect(result.configured.lastWarning).toBe('');
    expect(result.configured.syncInterval).toBe(15);
    expect(result.syncNow).toEqual({ pushed: 0, pulled: 0, serverSequence: 0 });
    expect(result.reset.configured).toBe(false);
    expect(result.reset.serverUrl).toBe('https://sync.example.test');
    expect(result.reset.tokenStored).toBe(false);
    expect(result.reset.statusLabel).toBe('disconnected');
    expect(result.disconnected.configured).toBe(false);
    expect(result.disconnected.statusLabel).toBe('disabled');
  });

  test('files API permanently deletes an existing trash entry through the bridge', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.files');
      await api.files.writeText('Project/bridge-trash.txt', 'remove me', { createIfMissing: true });
      const trash = await api.files.trash('Project/bridge-trash.txt');
      const before = await api.files.listTrash();
      await api.files.deleteTrash(trash.trashId);
      const after = await api.files.listTrash();
      let readError = '';
      try {
        await api.files.readText('Project/bridge-trash.txt');
      } catch (error) {
        readError = String(error && error.message || error);
      }
      api.dispose();
      return { before, after, readError };
    });

    expect(result.before).toHaveLength(1);
    expect(result.after).toEqual([]);
    expect(result.readError).toContain('not-found: Project/bridge-trash.txt');
  });

  test('capability invoke resolves provider and rejects undeclared consumers', async ({ page }) => {
    const result = await page.evaluate(async () => {
      window.__wailsMock.addSyntheticPlugins(3);
      const ids = [
        'verstak.synthetic-layout-01',
        'verstak.synthetic-layout-02',
        'verstak.synthetic-layout-03'
      ];
      const providerState = window.__wailsMock.getPluginState(ids[0]);
      const consumerState = window.__wailsMock.getPluginState(ids[1]);
      const rogueState = window.__wailsMock.getPluginState(ids[2]);
      const capability = 'test/notes/v1';

      providerState.manifest.provides = [capability];
      providerState.manifest.capabilityOperations = { [capability]: { list: 'test.notes.list' } };
      providerState.manifest.permissions = ['commands.register'];
      providerState.manifest.contributes.commands = [
        { id: 'test.notes.list', title: 'List notes', handler: 'test.notes.list' }
      ];
      consumerState.manifest.optionalRequires = [capability];
      rogueState.manifest.optionalRequires = [];

      const provider = window.createPluginAPI(ids[0]);
      await provider.commands.register('test.notes.list', async (args) => ({
        projectId: args.projectId,
        notes: ['README']
      }));
      const consumer = window.createPluginAPI(ids[1]);
      const handled = await consumer.capabilities.invoke(capability, 'list', { projectId: 'tos' });

      let rogueError = '';
      const rogue = window.createPluginAPI(ids[2]);
      try {
        await rogue.capabilities.invoke(capability, 'list', { projectId: 'tos' });
      } catch (error) {
        rogueError = String(error && error.message || error);
      }

      provider.dispose();
      consumer.dispose();
      rogue.dispose();
      return { handled, rogueError };
    });

    expect(result.handled.status).toBe('handled');
    expect(result.handled.pluginId).toBe('verstak.synthetic-layout-01');
    expect(result.handled.commandId).toBe('test.notes.list');
    expect(result.handled.result).toEqual({ projectId: 'tos', notes: ['README'] });
    expect(result.rogueError).toContain('capability dependency not declared');
  });

  test('backend plugin events are dispatched to subscribed frontend handlers', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.platform-test');
      let received = null;
      const unsubscribe = await api.events.subscribe('browser.capture.page', (event) => {
        received = event;
      });
      window.__VERSTAK_DISPATCH_BACKEND_EVENT__({
        name: 'browser.capture.page',
        timestamp: '2026-06-27T00:00:00.000Z',
        payload: { url: 'https://example.com/article' }
      });
      unsubscribe();
      api.dispose();
      return received;
    });

    expect(result.name).toBe('browser.capture.page');
    expect(result.payload.url).toBe('https://example.com/article');
    expect(result.timestamp).toBe('2026-06-27T00:00:00.000Z');
  });

  test('platform-test command and event handlers are cleaned up after leaving plugin view', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();

    await expect(page.locator('.pt-command-result')).toContainText('Command: handled', { timeout: 10000 });
    await expect(page.locator('.pt-event-result')).toHaveAttribute('data-event-status', 'received', { timeout: 10000 });
    await expect.poll(() => page.evaluate(() => Boolean(window.__VERSTAK_COMMAND_HANDLERS__?.['verstak.platform-test:verstak.platform-test.show-version']))).toBe(true);
    await expect.poll(() => page.evaluate(() => (window.__VERSTAK_EVENT_HANDLERS__?.['verstak.platform-test.echo'] || []).length)).toBe(1);

    await openPluginManager(page);

    await expect.poll(() => page.evaluate(() => Boolean(window.__VERSTAK_COMMAND_HANDLERS__?.['verstak.platform-test:verstak.platform-test.show-version']))).toBe(false);
    await expect.poll(() => page.evaluate(() => (window.__VERSTAK_EVENT_HANDLERS__?.['verstak.platform-test.echo'] || []).length)).toBe(0);
  });

  test('platform-test cleanup remains empty after disable reload flow', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();
    await expect(page.locator('.pt-command-result')).toContainText('Command: handled', { timeout: 10000 });

    await openPluginManager(page);
    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await pluginCard.locator('button.btn-disable').click();
    await expect(pluginCard.locator('button.btn-enable')).toBeVisible({ timeout: 10000 });

    await expect.poll(() => page.evaluate(() => Boolean(window.__VERSTAK_COMMAND_HANDLERS__?.['verstak.platform-test:verstak.platform-test.show-version']))).toBe(false);
    await expect.poll(() => page.evaluate(() => (window.__VERSTAK_EVENT_HANDLERS__?.['verstak.platform-test.echo'] || []).length)).toBe(0);
  });

  test('platform-test settings panel loads bundle content returned as raw string', async ({ page }) => {
    await openPluginManager(page);

    const pluginCard = page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' });
    await pluginCard.locator('button.btn-settings').click();

    // The card is a short path into the settings window, not a second place
    // where plugin settings live.
    const content = page.locator('[data-settings-content]');
    await expect(content).toBeVisible();
    await expect(content).toContainText('Platform Test Settings');
    await expect(content.locator('.host-state.error')).toHaveCount(0);
  });
});

test.describe('D2: workspace tree plugin API', () => {
  test('workspaces.tree exposes the complete read-only Deal hierarchy', async ({ page }) => {
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    const result = await page.evaluate(async () => {
      window.__wailsMock.setWorkspaceTreeV2({
        roots: [{
          key: 'folder:projects', kind: 'folder', id: 'folder-projects', name: 'Projects', path: 'Projects',
          children: [
            { key: 'workspace:a', kind: 'workspace', id: 'deal-a', name: 'Same', path: 'Projects/Same', children: [] },
            { key: 'workspace:b', kind: 'workspace', id: 'deal-b', name: 'Nested', path: 'Projects/Nested', children: [] },
          ],
        }],
        currentWorkspaceId: 'deal-b', revision: 9, warnings: [],
      });
      const api = window.createPluginAPI('verstak.projects');
      const tree = await api.workspaces.tree();
      api.dispose();
      return tree;
    });
    expect(result.currentWorkspaceId).toBe('deal-b');
    expect(result.revision).toBe(9);
    expect(result.roots[0].kind).toBe('folder');
    expect(result.roots[0].children.map((node) => node.id)).toEqual(['deal-a', 'deal-b']);
  });

  test('navigation.openWorkspace selects Deal and forwards opaque tool state', async ({ page }) => {
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    const result = await page.evaluate(async () => {
      window.__wailsMock.setWorkspaceTreeV2({
        roots: [{ key: 'workspace:deal-42', kind: 'workspace', id: '11111111-1111-4111-8111-111111111111', name: 'Workbench', path: 'Projects/Workbench', children: [] }],
        currentWorkspaceId: '11111111-1111-4111-8111-111111111111', revision: 10, warnings: [],
      });
      const events = [];
      const selected = (event) => events.push({ name: event.type, detail: event.detail });
      const opened = (event) => events.push({ name: event.type, detail: event.detail });
      window.addEventListener('verstak:workspace-selected', selected);
      window.addEventListener('verstak:workspace-open-tool', opened);
      const api = window.createPluginAPI('verstak.projects');
      await api.navigation.openWorkspace({
        workspaceId: '11111111-1111-4111-8111-111111111111',
        workspaceItemId: 'verstak.projects.workspace',
        toolRequest: { resourceId: 'resource-7' },
      });
      api.dispose();
      await new Promise((resolve) => setTimeout(resolve, 0));
      window.removeEventListener('verstak:workspace-selected', selected);
      window.removeEventListener('verstak:workspace-open-tool', opened);
      return events;
    });

    expect(result).toHaveLength(2);
    expect(result[0]).toEqual({
      name: 'verstak:workspace-selected',
      detail: {
        workspaceId: '11111111-1111-4111-8111-111111111111',
        workspaceName: 'Projects/Workbench',
        workspaceRootPath: 'Projects/Workbench',
        workspaceItemId: 'verstak.projects.workspace',
        toolRequest: { resourceId: 'resource-7' },
      },
    });
    expect(result[1]).toEqual({
      name: 'verstak:workspace-open-tool',
      detail: {
        workspaceId: '11111111-1111-4111-8111-111111111111',
        workspaceItemId: 'verstak.projects.workspace',
        workspaceRootPath: 'Projects/Workbench',
        toolRequest: { resourceId: 'resource-7' },
      },
    });
  });

  test('navigation.openWorkspace rejects a non-UUID Deal identity', async ({ page }) => {
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    const error = await page.evaluate(async () => {
      const api = window.createPluginAPI('verstak.projects');
      try {
        await api.navigation.openWorkspace({ workspaceId: 'Projects/Workbench' });
        return '';
      } catch (err) {
        return String(err.message || err);
      } finally {
        api.dispose();
      }
    });
    expect(error).toContain('workspaceId');
  });
});
