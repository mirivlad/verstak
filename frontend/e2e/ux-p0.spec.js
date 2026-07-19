import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

test.describe('UX P0 shell flow', () => {
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

  test('starts in the first workspace instead of Plugin Manager', async ({ page }) => {
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.plugin-manager')).toHaveCount(0);
    await expect(page.locator('.wt-node.selected .wt-label')).toHaveText('Project');
    await expect(page.locator('.main-content-title-text')).toHaveText('Project');
  });

  test('workspace selection and main content stay in sync across plugin manager round trip', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Test' }).click();

    await expect(page.locator('.main-content-title-text')).toHaveText('Test', { timeout: 10000 });
    await expect(page.locator('.wt-node.selected .wt-label')).toHaveText('Test');
    await expect(page.locator('.plugin-manager')).toHaveCount(0);

    await openPluginManager(page);
    await expect(page.locator('.plugin-manager')).toBeVisible();
    await expect(page.locator('.wt-node.selected .wt-label')).toHaveCount(0);

    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await expect(page.locator('.main-content-title-text')).toHaveText('Project', { timeout: 10000 });
    await expect(page.locator('.plugin-manager')).toHaveCount(0);
  });

  test('Deal header does not expose the internal workspace type badge', async ({ page }) => {
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.workspace-type')).toHaveCount(0);
  });

  test('status bar plugin contribution failures do not render large error panels', async ({ page }) => {
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Plugin View Error')).toHaveCount(0);
    await expect(page.locator('.status-bar [data-status-item-id]')).toHaveCount(2);
  });

  test('Plugin Manager remains reachable from the settings menu', async ({ page }) => {
    await openPluginManager(page);

    await expect(page.locator('.plugin-manager')).toBeVisible();
    await expect(page.locator('.plugin-card').filter({ hasText: 'verstak.platform-test' })).toBeVisible();
  });
});

test.describe('UX quick wins', () => {
  test('Files screen uses readable dates and understandable action controls', async ({ page }) => {
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await page.getByRole('tab', { name: 'Files' }).click();

    const files = page.locator('.files-root');
    await expect(files).toBeVisible({ timeout: 10000 });
    await expect(files.getByText(/T\d{2}:\d{2}:\d{2}/)).toHaveCount(0);

    const actions = [
      ['new-folder', 'New folder'],
      ['new-markdown', 'New markdown file'],
      ['new-text', 'New text file'],
    ];
    for (const [action, label] of actions) {
      const button = page.locator(`[data-files-action="${action}"]`).first();
      await expect(button).toBeVisible();
      await expect(button).toHaveAttribute('title', label);
      await expect(button).toHaveAttribute('aria-label', label);
      await expect(button).toHaveAttribute('data-files-icon', /.+/);
      await expect(button.locator('svg')).toBeVisible();
    }
  });

  test('Vault Selection is localized and has a clear primary action', async ({ browser }) => {
    const page = await browser.newPage();
    await page.addInitScript(() => {
      window.go = { api: { App: {
        GetAppSettings: async () => ({ currentVaultPath: '', recentVaults: ['/tmp/verstak-recent-vault'] }),
        GetVaultStatus: async () => ({ status: 'closed', path: '', vaultId: '' }),
        SelectDirectory: async () => '',
        SelectVaultForOpen: async () => '',
        CreateVault: async () => null,
        OpenVault: async () => null,
        SetCurrentVault: async () => '',
        WriteFrontendLog: async () => {},
      } } };
    });

    await page.goto('/');
    await page.waitForSelector('.vault-selection', { timeout: 10000 });

    await expect(page.getByText('Choose a vault to start working')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Create vault' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Open existing' })).toBeVisible();
    await page.close();
  });
});
