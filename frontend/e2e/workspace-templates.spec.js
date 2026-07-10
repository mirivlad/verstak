import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Workspace templates', () => {
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

  async function openCreateModal(page) {
    await page.locator('button[title="New workspace"]').click();
    const modal = page.locator('[data-workspace-create-modal]');
    await expect(modal).toBeVisible();
    return modal;
  }

  test('creation modal validates names, shows template tools, and persists the selected snapshot', async ({ page }) => {
    const modal = await openCreateModal(page);
    await expect(modal.locator('[data-workspace-template]')).toHaveValue('default');
    await expect(modal.locator('[data-workspace-template-tools]')).toContainText('Notes');
    await expect(modal.locator('[data-workspace-template-tools]')).toContainText('Browser Inbox');

    await modal.getByRole('button', { name: 'Create workspace' }).click();
    await expect(modal.locator('[data-workspace-create-error]')).toContainText('Name is required');

    await modal.locator('[data-workspace-name]').fill('bad/name');
    await modal.getByRole('button', { name: 'Create workspace' }).click();
    await expect(modal.locator('[data-workspace-create-error]')).toContainText('invalid-workspace-name');

    await modal.locator('[data-workspace-name]').fill('ProjectPlan');
    await modal.locator('[data-workspace-template]').selectOption('project');
    await expect(modal.locator('[data-workspace-template-description]')).toContainText('Project planning');
    await expect(modal.locator('[data-workspace-template-tools]')).toContainText('Todos');
    await modal.getByRole('button', { name: 'Create workspace' }).click();

    await expect(page.locator('.wt-label').filter({ hasText: 'ProjectPlan' })).toBeVisible();
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.GetWorkspaceMetadata('ProjectPlan');
      const metadata = Array.isArray(result) ? result[0] : result;
      return {
        templateId: metadata.createdFromTemplate?.templateId,
        tools: metadata.workspaceTools,
      };
    })).toEqual({
      templateId: 'project',
      tools: ['verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.journal', 'verstak.activity', 'verstak.browser-inbox'],
    });

    await expect(page.getByRole('tab', { name: 'Todos' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Journal' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Secrets' })).toHaveCount(0);
  });

  test('Minimal keeps global tools available while limiting workspace tabs', async ({ page }) => {
    const modal = await openCreateModal(page);
    await modal.locator('[data-workspace-name]').fill('MinimalSpace');
    await modal.locator('[data-workspace-template]').selectOption('minimal');
    await modal.getByRole('button', { name: 'Create workspace' }).click();

    await expect(page.getByRole('tab', { name: 'Overview' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Notes' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Files' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Todos' })).toHaveCount(0);
    await expect(page.getByRole('tab', { name: 'Journal' })).toHaveCount(0);
    await expect(page.getByRole('tab', { name: 'Secrets' })).toHaveCount(0);
    await expect(page.locator('.sidebar .plugin-item').filter({ hasText: 'Todos' })).toBeVisible();
    await expect(page.locator('.sidebar .plugin-item').filter({ hasText: 'Browser Inbox' })).toBeVisible();
  });

  test('Admin shows Secrets when available and missing workspace plugins degrade without breaking tabs', async ({ page }) => {
    let modal = await openCreateModal(page);
    await modal.locator('[data-workspace-name]').fill('AdminSpace');
    await modal.locator('[data-workspace-template]').selectOption('admin');
    await modal.getByRole('button', { name: 'Create workspace' }).click();
    await expect(page.getByRole('tab', { name: 'Secrets' })).toBeVisible();

    await page.evaluate(() => window.__wailsMock.setPluginStatus('verstak.todo', 'disabled', false));
    modal = await openCreateModal(page);
    await modal.locator('[data-workspace-name]').fill('ProjectWithoutTodo');
    await modal.locator('[data-workspace-template]').selectOption('project');
    await modal.getByRole('button', { name: 'Create workspace' }).click();

    await expect(page.getByRole('tab', { name: 'Notes' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Files' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Todos' })).toHaveCount(0);

    await page.locator('.wt-label').filter({ hasText: 'AdminSpace' }).click();
    await expect(page.getByRole('tab', { name: 'Secrets' })).toBeVisible();
  });
});
