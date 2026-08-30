import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Project, Milestones and Git workspace UX', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    await page.evaluate(async () => {
      const workspaceId = '11111111-1111-4111-8111-111111111111';
      await window.go.api.App.UpdateWorkspaceV2Tools(workspaceId, [
        'verstak.projects', 'verstak.notes', 'verstak.files', 'verstak.todo',
        'verstak.milestones', 'verstak.git',
      ]);
      window.dispatchEvent(new CustomEvent('verstak:workspace-tools-changed', { detail: { workspaceId } }));
    });
  });

  test.afterEach(async () => consoleCollector.assertNoErrors());

  test('workspace plugin surfaces use the available width and styled selects', async ({ page }) => {
    await page.getByRole('tab', { name: 'Project', exact: true }).click();
    const projectForm = page.locator('.project-meta-form');
    await expect(projectForm).toBeVisible();
    expect((await projectForm.boundingBox()).width).toBeGreaterThan(600);
    const projectSelectStyle = await page.locator('.project-meta-select').first().evaluate((node) => ({
      appearance: getComputedStyle(node).appearance,
      backgroundImage: getComputedStyle(node).backgroundImage,
    }));
    expect(projectSelectStyle.appearance).toBe('none');
    expect(projectSelectStyle.backgroundImage).not.toBe('none');

    await expect(page.locator('.workspace-tab-list').getByRole('tab', { name: 'Milestones', exact: true })).toHaveCount(0);
    await page.locator('[data-workspace-project-section="milestones"]').click();
    const milestones = page.locator('.milestones-shell');
    await expect(milestones).toBeVisible();
    expect((await milestones.boundingBox()).width).toBeGreaterThan(600);
    await milestones.locator('[data-milestone-action="add"]').click();
    const editor = milestones.locator('.milestones-editor');
    await expect(editor).toBeVisible();
    await editor.locator('[data-milestone-input="title"]').fill('Release 1.0');
    await editor.locator('[data-milestone-input="dueAt"]').fill('2026-09-15');
    await editor.locator('[data-milestone-action="save"]').click();
    await expect(milestones.locator('[data-milestone-id]').filter({ hasText: 'Release 1.0' })).toBeVisible();
    const milestoneSelectStyle = await milestones.locator('.milestones-select').evaluate((node) => ({
      appearance: getComputedStyle(node).appearance,
      backgroundImage: getComputedStyle(node).backgroundImage,
    }));
    expect(milestoneSelectStyle.appearance).toBe('none');
    expect(milestoneSelectStyle.backgroundImage).not.toBe('none');

    await page.getByRole('tab', { name: 'Git', exact: true }).click();
    const gitRoot = page.locator('[data-git-root]');
    await gitRoot.locator('[data-git-action="add"]').click();
    const gitForm = gitRoot.locator('[data-git-form="add"]');
    await expect(gitForm).toBeVisible();
    expect((await gitForm.boundingBox()).width).toBeGreaterThan(600);
    const gitSelectStyle = await gitForm.locator('.git-select').evaluate((node) => ({
      appearance: getComputedStyle(node).appearance,
      backgroundImage: getComputedStyle(node).backgroundImage,
    }));
    expect(gitSelectStyle.appearance).toBe('none');
    expect(gitSelectStyle.backgroundImage).not.toBe('none');
  });
});
