import { test, expect } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';
import { waitForAppReady, resetMockState } from '../helpers.js';

const VISUAL_DIR = resolve(process.cwd(), 'e2e-results/visual');

async function shot(page, name) {
  await page.evaluate(() => document.activeElement?.blur?.());
  await page.waitForTimeout(80);
  await page.screenshot({
    path: resolve(VISUAL_DIR, name),
    fullPage: true,
    animations: 'disabled',
  });
}

async function seed(page) {
  return page.evaluate(async () => {
    window.__wailsMock?.setPluginStatus('verstak.platform-test', 'disabled', false);
    window.dispatchEvent(new CustomEvent('verstak:plugins-changed'));
    const tools = ['verstak.projects', 'verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.activity'];
    const creatures = await window.go.api.App.CreateWorkspaceV2WithTools('', 'Creatures2.0', 'custom', tools);
    const ai = await window.go.api.App.CreateWorkspaceV2WithTools('', 'AI-server', 'custom', tools);
    if (!creatures?.id || !ai?.id) throw new Error('visual Deals were not created');

    window.__wailsMock.setWorkspaceTreeV2({
      roots: [
        {
          key: 'folder:projects', kind: 'folder', id: 'folder-projects', name: 'Projects', path: 'Projects', children: [
            { key: `workspace:${ai.id}`, kind: 'workspace', id: ai.id, name: 'AI-server', path: 'AI-server', children: [] },
            { key: `workspace:${creatures.id}`, kind: 'workspace', id: creatures.id, name: 'Creatures2.0', path: 'Creatures2.0', children: [] },
          ],
        },
        {
          key: 'folder:archive', kind: 'folder', id: 'folder-archive', name: 'Archive', path: 'Archive', children: [
            { key: 'workspace:old-creatures', kind: 'workspace', id: 'deal-old-creatures', name: 'Creatures2.0', path: 'Archive/Creatures2.0', children: [] },
          ],
        },
      ],
      currentWorkspaceId: creatures.id, revision: 8, warnings: [],
    });
    await window.go.api.App.WritePluginSettings('verstak.projects', {
      'projects:global': [
        {
          id: 'visual-project', name: 'Projects UX v3',
          description: 'Scoped project work inside one Deal: tasks, notes and files belong to this project without hiding them from the Deal.',
          status: 'active', priority: 'high', tags: ['release', 'ux'],
          workspaceId: creatures.id, workspaceRootPath: 'Creatures2.0',
          milestones: [
            { id: 'm1', title: 'Scope model', status: 'done' },
            { id: 'm2', title: 'Portfolio', status: 'done' },
            { id: 'm3', title: 'Release verification', status: 'open', dueAt: '2026-08-25' },
          ],
          links: [{ id: 'l1', label: 'Repository', url: 'https://github.com/mirivlad/verstak' }],
          events: [
            { id: 'e1', type: 'milestone.completed', subject: 'Portfolio', at: '2026-08-24T12:00:00.000Z' },
            { id: 'e2', type: 'project.priority', from: 'normal', to: 'high', at: '2026-08-24T12:10:00.000Z' },
          ],
          createdAt: '2026-08-24T10:00:00.000Z', updatedAt: '2026-08-24T12:10:00.000Z',
        },
        {
          id: 'ai-project', name: 'Local AI steward',
          description: 'A second project in another Deal so the portfolio is visibly cross-Deal.',
          status: 'paused', priority: 'normal', tags: ['assistant'],
          workspaceId: ai.id, workspaceRootPath: 'AI-server',          milestones: [{ id: 'a1', title: 'Runtime prototype', status: 'open' }],
          links: [], events: [],
          createdAt: '2026-08-23T10:00:00.000Z', updatedAt: '2026-08-23T10:00:00.000Z',
        },
      ],
    });
    return { creaturesId: creatures.id, aiId: ai.id };
  });
}

test.describe('Visual audit: Projects scope model v3', () => {
  test('portfolio, Deal-scoped project and hierarchical Deal picker', async ({ page }) => {
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    await mkdir(VISUAL_DIR, { recursive: true });
    const ids = await seed(page);

    await expect(page.locator('.sidebar')).not.toContainText('Platform Test');
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Projects' }).click();
    const root = page.locator('[data-plugin-id="verstak.projects"]');
    await expect(root).toBeVisible({ timeout: 10000 });
    await expect(root.locator('[data-project-portfolio]')).toBeVisible();
    await expect(root.locator('[data-project-card]')).toHaveCount(2);
    await expect(root.locator('[data-project-id="visual-project"]')).toContainText('Projects/Creatures2.0');
    await expect(root.locator('[data-project-id="ai-project"]')).toContainText('Projects/AI-server');
    await shot(page, 'projects-portfolio.png');
    await root.locator('[data-project-id="visual-project"]').click();
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });
    const dealProject = page.locator('[data-plugin-id="verstak.projects"]');
    await expect(dealProject).toHaveClass(/deal/);
    await expect(dealProject.locator('[data-project-portfolio]')).toHaveCount(0);
    await expect(dealProject.locator('.projects-header-name')).toHaveText('Projects UX v3');
    await expect(dealProject.locator('[data-project-tab="tasks"]')).toBeVisible();
    await expect(dealProject.locator('[data-project-tab="files"]')).toBeVisible();
    await shot(page, 'projects-deal-project.png');

    await dealProject.locator('[data-project-action="edit"]').click();
    const modal = dealProject.locator('[data-project-modal]');
    await expect(modal).toBeVisible();
    await expect(modal.locator('[data-project-field="workspace"]')).toHaveCount(0);
    await modal.locator('[data-deal-picker-toggle]').click();
    await modal.locator('[data-deal-picker-search]').fill('Creatures2.0');
    await expect(modal.locator('[data-deal-id]')).toHaveCount(2);
    await expect(modal.locator(`[data-deal-id="${ids.creaturesId}"]`)).toContainText('Projects/Creatures2.0');
    await expect(modal.locator('[data-deal-id="deal-old-creatures"]')).toContainText('Archive/Creatures2.0');
    await expect(modal.locator(`[data-deal-id="${ids.creaturesId}"]`)).toBeVisible();
    await expect(modal.locator('[data-deal-id="deal-old-creatures"]')).toBeVisible();
    await shot(page, 'projects-deal-picker.png');
  });
});
