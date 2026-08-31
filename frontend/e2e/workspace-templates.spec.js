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

  async function openNewDeal(page) {
    await page.locator('button[title="New Deal"]').click();
    const dialog = page.locator('[data-new-deal-dialog]');
    await expect(dialog).toBeVisible();
    return dialog;
  }

  async function openTemplateSettings(page) {
    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-section="plugin:verstak.templates:verstak.templates.settings"]').click();
    const editor = page.locator('[data-templates-form]');
    await expect(editor).toBeVisible();
    return editor;
  }

  test('Settings owns template CRUD while New Deal applies a selected persisted template', async ({ page }) => {
    const editor = await openTemplateSettings(page);
    await expect(editor.locator('[data-template-tool="verstak.notes"]')).toContainText('Notes');
    await expect(editor.locator('[data-template-tool="verstak.notes"]')).not.toContainText('verstak.notes');

    await page.locator('[data-settings-window-close]').click();
    const dialog = await openNewDeal(page);
    await dialog.locator('[data-new-deal-template="seed-minimal"]').click();
    await dialog.locator('[data-new-deal-name]').fill('Minimal picker Deal');
    await dialog.locator('[data-new-deal-create]').click();

    await expect.poll(async () => page.evaluate(async () => {
      const tree = await window.go.api.App.GetWorkspaceTreeV2();
      return tree.roots.some((node) => node.name === 'Minimal picker Deal');
    })).toBe(true);
  });

  test('Settings keeps a removed template tool visible but disabled', async ({ page }) => {
    await page.evaluate(() => window.__wailsMock.setPluginStatus('verstak.todo', 'disabled', false));
    const editor = await openTemplateSettings(page);
    await page.locator('button.templates-template', { hasText: 'Project' }).click();

    const projectTools = await page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginDataNDJSON('verstak.templates', 'templates');
      const templates = Array.isArray(result) ? result[0] : result;
      return templates.find((template) => template.id === 'seed-project')?.workspaceTools || [];
    });
    expect(projectTools).toContain('verstak.todo');

    const missingTool = editor.locator('[data-template-tool="verstak.todo"]');
    await expect(missingTool).toBeDisabled();
    await expect(missingTool).toContainText('Unavailable workspace tool');
  });

  test('Project seed passes its complete recipe snapshot through the public creation bridge', async ({ page }) => {
    const dialog = await openNewDeal(page);
    await dialog.locator('[data-new-deal-template="seed-project"]').click();
    await dialog.locator('[data-new-deal-name]').fill('ProjectPlan');
    await dialog.locator('[data-new-deal-create]').click();
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
      tools: ['verstak.projects', 'verstak.git', 'verstak.todo', 'verstak.milestones', 'verstak.notes', 'verstak.files', 'verstak.journal', 'verstak.secrets'],
      provenance: {
        templateId: 'seed-project',
        templateName: 'Project',
        templateVersion: 1,
        appliedAt: expect.any(String),
        workspaceTools: ['verstak.projects', 'verstak.git', 'verstak.todo', 'verstak.milestones', 'verstak.notes', 'verstak.files', 'verstak.journal', 'verstak.secrets'],
      },
    });
  });

  test('Edit Deal changes tools without deleting disabled plugin data', async ({ page }) => {
    const dialog = await openNewDeal(page);
    await dialog.locator('[data-new-deal-template="seed-project"]').click();
    await dialog.locator('[data-new-deal-name]').fill('EditableDeal');
    await dialog.locator('[data-new-deal-create]').click();
    await expect(dialog).toBeHidden();

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
    const form = await openTemplateSettings(page);
    await page.locator('.templates-list').getByRole('button', { name: 'New template', exact: true }).click();

    await form.locator('[data-template-field="name"]').fill('Client recipe');
    await form.locator('[data-template-field="description"]').fill('A focused client Deal.');
    await form.locator('[data-template-tool="verstak.notes"]').click();
    await form.locator('[data-template-tool="verstak.files"]').click();
    await form.locator('[data-template-field="folders"]').fill('Notes\nFiles\nBrief');
    await form.locator('[data-template-action="save"]').click();
    await expect(form.locator('.templates-message')).toContainText('Template saved.');
    await expect(page.getByRole('button', { name: 'Client recipe', exact: true })).toBeVisible();

    await form.locator('[data-template-action="duplicate"]').click();
    await expect(form.locator('.templates-message')).toContainText('Template duplicated.');
    await expect(page.getByRole('button', { name: 'Client recipe (copy)', exact: true })).toBeVisible();
    await form.locator('[data-template-field="name"]').fill('Independent client recipe');
    await form.locator('[data-template-tool="verstak.files"]').click();
    await form.locator('[data-template-action="save"]').click();
    await expect(form.locator('.templates-message')).toContainText('Template saved.');
    await page.getByRole('button', { name: 'Client recipe', exact: true }).click();
    await expect(form.locator('[data-template-tool="verstak.notes"]')).toHaveAttribute('aria-pressed', 'true');
    await expect(form.locator('[data-template-tool="verstak.files"]')).toHaveAttribute('aria-pressed', 'true');

    await page.locator('[data-settings-window-close]').click();
    const dialog = await openNewDeal(page);
    const clientTemplateId = await page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginDataNDJSON('verstak.templates', 'templates');
      const templates = Array.isArray(result) ? result[0] : result;
      return templates.find((template) => template.name === 'Client recipe')?.id || '';
    });
    await dialog.locator(`[data-new-deal-template="${clientTemplateId}"]`).click();
    await dialog.locator('[data-new-deal-name]').fill('ClientDeal');
    await dialog.locator('[data-new-deal-create]').click();
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

    const reopenedForm = await openTemplateSettings(page);
    await page.locator('.templates-list').getByRole('button', { name: 'Client recipe', exact: true }).click();
    await reopenedForm.locator('[data-template-action="delete"]').click();
    await expect(page.getByRole('button', { name: 'Client recipe', exact: true })).toHaveCount(0);
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginDataNDJSON('verstak.templates', 'templates');
      const templates = Array.isArray(result) ? result[0] : result;
      return templates.some((template) => template.name === 'Client recipe');
    })).toBe(false);
  });

  test('creation rejects a recipe whose workspace plugin is unavailable', async ({ page }) => {
    await page.evaluate(() => window.__wailsMock.setPluginStatus('verstak.todo', 'disabled', false));
    const dialog = await openNewDeal(page);
    await dialog.locator('[data-new-deal-template="seed-project"]').click();
    await dialog.locator('[data-new-deal-name]').fill('UnavailableProject');
    await dialog.locator('[data-new-deal-create]').click();

    await expect(dialog.locator('[data-new-deal-error]')).toBeVisible();
    await expect.poll(async () => page.evaluate(async () => {
      const tree = await window.go.api.App.GetWorkspaceTreeV2();
      return tree.roots.some((node) => node.name === 'UnavailableProject');
    })).toBe(false);
  });
});
