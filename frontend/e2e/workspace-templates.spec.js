import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Deal templates plugin', () => {
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

  async function openTemplates(page) {
    await page.locator('button[title="New Deal"]').click();
    const form = page.locator('[data-templates-form]');
    await expect(form).toBeVisible();
    return form;
  }

  test('Project seed passes its complete recipe snapshot through the public creation bridge', async ({ page }) => {
    const form = await openTemplates(page);
    await page.getByRole('button', { name: 'Project', exact: true }).click();

    await expect(form.locator('[data-template-field="tools"]')).toHaveValue(/verstak\.git/);
    await expect(form.locator('[data-template-field="tools"]')).toHaveValue(/verstak\.milestones/);
    await expect(form.locator('[data-template-field="tools"]')).toHaveValue(/verstak\.secrets/);
    await form.locator('[data-template-field="deal-name"]').fill('ProjectPlan');
    await form.locator('[data-template-action="create-deal"]').click();

    await expect(form.locator('.templates-message')).toContainText('Deal created.');
    await expect.poll(async () => page.evaluate(async () => {
      const tree = await window.go.api.App.GetWorkspaceTreeV2();
      return tree.roots.some((node) => node.name === 'ProjectPlan');
    })).toBe(true);
    await expect.poll(async () => page.evaluate(async () => {
      const metadata = await window.go.api.App.GetWorkspaceMetadata('ProjectPlan');
      return {
        tools: metadata.workspaceTools,
        provenance: metadata.createdFromTemplate,
      };
    })).toEqual({
      tools: ['verstak.projects', 'verstak.git', 'verstak.todo', 'verstak.milestones', 'verstak.notes', 'verstak.files', 'verstak.activity', 'verstak.journal', 'verstak.secrets'],
      provenance: {
        templateId: 'seed-project',
        templateName: 'Project',
        templateVersion: 1,
        appliedAt: expect.any(String),
        workspaceTools: ['verstak.projects', 'verstak.git', 'verstak.todo', 'verstak.milestones', 'verstak.notes', 'verstak.files', 'verstak.activity', 'verstak.journal', 'verstak.secrets'],
      },
    });
  });

  test('Edit Deal changes tools without deleting disabled plugin data', async ({ page }) => {
    const form = await openTemplates(page);
    await page.getByRole('button', { name: 'Project', exact: true }).click();
    await form.locator('[data-template-field="deal-name"]').fill('EditableDeal');
    await form.locator('[data-template-action="create-deal"]').click();
    await expect(form.locator('.templates-message')).toContainText('Deal created.');

    const workspaceId = await page.evaluate(async () => {
      const tree = await window.go.api.App.GetWorkspaceTreeV2();
      return tree.roots.find((node) => node.name === 'EditableDeal')?.id || '';
    });
    expect(workspaceId).not.toBe('');
    await page.evaluate(async (id) => {
      await window.go.api.App.WritePluginDealConfig('verstak.projects', id, { description: 'survives tool disable' });
    }, workspaceId);

    const deal = page.locator('.wt-node').filter({ hasText: 'EditableDeal' });
    await deal.click();
    await expect(page.getByRole('tab', { name: 'Project' })).toBeVisible();
    await deal.click({ button: 'right' });
    await page.getByRole('button', { name: 'Edit Deal' }).click();

    const edit = page.locator('[data-workspace-edit-modal]');
    await expect(edit).toBeVisible();
    await expect(edit.locator('[data-workspace-edit-name]')).toHaveValue('EditableDeal');
    const projects = edit.locator('[data-workspace-edit-tool="verstak.projects"]');
    await expect(projects).toHaveAttribute('aria-pressed', 'true');
    await edit.locator('[data-workspace-edit-name]').fill('EditableDealRenamed');
    await projects.click();
    await edit.getByRole('button', { name: 'Save' }).click();

    await expect(edit).toBeHidden();
    await expect(page.getByRole('tab', { name: 'Project' })).toHaveCount(0);
    await expect(page.locator('.wt-node').filter({ hasText: 'EditableDealRenamed' })).toBeVisible();
    await expect.poll(async () => page.evaluate(async (id) => {
      const metadata = await window.go.api.App.GetWorkspaceMetadataByUUID(id);
      const config = await window.go.api.App.ReadPluginDealConfig('verstak.projects', id);
      return {
        hasProject: metadata.workspaceTools.includes('verstak.projects'),
        description: Array.isArray(config) ? config[0]?.description : config?.description,
      };
    }, workspaceId)).toEqual({ hasProject: false, description: 'survives tool disable' });

    const renamedDeal = page.locator('.wt-node').filter({ hasText: 'EditableDealRenamed' });
    await renamedDeal.click({ button: 'right' });
    await page.getByRole('button', { name: 'Edit Deal' }).click();
    await expect(projects).toHaveAttribute('aria-pressed', 'false');
    await projects.click();
    await edit.getByRole('button', { name: 'Save' }).click();
    await expect(page.getByRole('tab', { name: 'Project' })).toBeVisible();
    await expect.poll(async () => page.evaluate(async (id) => {
      const config = await window.go.api.App.ReadPluginDealConfig('verstak.projects', id);
      return Array.isArray(config) ? config[0]?.description : config?.description;
    }, workspaceId)).toBe('survives tool disable');
  });

  test('templates are persisted plugin-owned recipes with CRUD and creation', async ({ page }) => {
    const form = await openTemplates(page);
    await page.getByRole('button', { name: 'New template', exact: true }).click();

    await form.locator('[data-template-field="name"]').fill('Client recipe');
    await form.locator('[data-template-field="description"]').fill('A focused client Deal.');
    await form.locator('[data-template-field="tools"]').fill('verstak.notes\nverstak.files');
    await form.locator('[data-template-field="folders"]').fill('Notes\nFiles\nBrief');
    await form.locator('[data-template-action="save"]').click();
    await expect(form.locator('.templates-message')).toContainText('Template saved.');
    await expect(page.getByRole('button', { name: 'Client recipe', exact: true })).toBeVisible();

    await form.locator('[data-template-action="duplicate"]').click();
    await expect(form.locator('.templates-message')).toContainText('Template duplicated.');
    await expect(page.getByRole('button', { name: 'Client recipe (copy)', exact: true })).toBeVisible();
    await form.locator('[data-template-field="name"]').fill('Independent client recipe');
    await form.locator('[data-template-field="tools"]').fill('verstak.notes');
    await form.locator('[data-template-action="save"]').click();
    await expect(form.locator('.templates-message')).toContainText('Template saved.');
    await page.getByRole('button', { name: 'Client recipe', exact: true }).click();
    await expect(form.locator('[data-template-field="tools"]')).toHaveValue('verstak.notes\nverstak.files');

    await form.locator('[data-template-field="deal-name"]').fill('ClientDeal');
    await form.locator('[data-template-action="create-deal"]').click();
    await expect(form.locator('.templates-message')).toContainText('Deal created.');
    await expect.poll(async () => page.evaluate(async () => {
      const metadata = await window.go.api.App.GetWorkspaceMetadata('ClientDeal');
      return {
        tools: metadata.workspaceTools,
        templateName: metadata.createdFromTemplate?.templateName,
        briefExists: Boolean((await window.go.api.App.ListVaultFiles('verstak.files', 'ClientDeal'))[0].find((entry) => entry.relativePath === 'ClientDeal/Brief')),
      };
    })).toEqual({
      tools: ['verstak.notes', 'verstak.files'],
      templateName: 'Client recipe',
      briefExists: true,
    });

    await form.locator('[data-template-action="delete"]').click();
    await expect(page.getByRole('button', { name: 'Client recipe', exact: true })).toHaveCount(0);
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginDataNDJSON('verstak.templates', 'templates');
      const templates = Array.isArray(result) ? result[0] : result;
      return templates.some((template) => template.name === 'Client recipe');
    })).toBe(false);
  });

  test('creation rejects a recipe whose workspace plugin is unavailable', async ({ page }) => {
    await page.evaluate(() => window.__wailsMock.setPluginStatus('verstak.todo', 'disabled', false));
    const form = await openTemplates(page);
    await page.getByRole('button', { name: 'Project', exact: true }).click();
    await form.locator('[data-template-field="deal-name"]').fill('UnavailableProject');
    await form.locator('[data-template-action="create-deal"]').click();

    await expect(form.locator('.templates-missing')).toContainText('verstak.todo');
    await expect.poll(async () => page.evaluate(async () => {
      const tree = await window.go.api.App.GetWorkspaceTreeV2();
      return tree.roots.some((node) => node.name === 'UnavailableProject');
    })).toBe(false);
  });
});
