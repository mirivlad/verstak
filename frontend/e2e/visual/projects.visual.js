import { test, expect } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';
import { waitForAppReady, resetMockState } from '../helpers.js';

const VISUAL_DIR = resolve(process.cwd(), 'e2e-results/visual');

async function shot(page, name) {
  await page.evaluate(() => document.activeElement?.blur?.());
  await page.waitForTimeout(50);
  await page.screenshot({
    path: resolve(VISUAL_DIR, name),
    fullPage: true,
    animations: 'disabled',
  });
}

test.describe('Visual audit: Projects UX v2', () => {
  test('master-detail overview and hierarchical Deal picker', async ({ page }) => {
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    await mkdir(VISUAL_DIR, { recursive: true });

    await page.evaluate(async () => {
      window.__wailsMock.setWorkspaceTreeV2({
        roots: [
          {
            key: 'folder:projects', kind: 'folder', id: 'folder-projects', name: 'Projects', path: 'Projects', children: [
              { key: 'workspace:ai', kind: 'workspace', id: 'deal-ai', name: 'AI-server', path: 'Projects/AI-server', children: [] },
              { key: 'workspace:creatures', kind: 'workspace', id: 'deal-creatures', name: 'Creatures2.0', path: 'Projects/Creatures2.0', children: [] },
            ],
          },
          {
            key: 'folder:archive', kind: 'folder', id: 'folder-archive', name: 'Archive', path: 'Archive', children: [
              { key: 'workspace:old-creatures', kind: 'workspace', id: 'deal-old-creatures', name: 'Creatures2.0', path: 'Archive/Creatures2.0', children: [] },
            ],
          },
        ],
        currentWorkspaceId: 'deal-creatures', revision: 7, warnings: [],
      });
      await window.go.api.App.WritePluginSettings('verstak.projects', {
        'projects:global': [{
          id: 'visual-project',
          name: 'Projects UX v2',
          description: 'Release candidate: native project workspace with a stable Deal link and optional tool integrations.',
          status: 'active',
          priority: 'high',
          tags: ['release', 'ux', 'desktop'],
          workspaceId: 'deal-creatures',
          workspaceRootPath: 'Projects/Creatures2.0',
          milestones: [
            { id: 'm1', title: 'Architecture', status: 'done' },
            { id: 'm2', title: 'Deal picker', status: 'done' },
            { id: 'm3', title: 'Release verification', status: 'open', dueAt: '2026-08-25' },
          ],
          links: [{ id: 'l1', label: 'Repository', url: 'https://github.com/mirivlad/verstak' }],
          events: [
            { id: 'e1', type: 'milestone.completed', milestone: 'Deal picker', at: '2026-08-24T12:00:00.000Z' },
            { id: 'e2', type: 'project.priority', from: 'normal', to: 'high', at: '2026-08-24T12:10:00.000Z' },
          ],
          createdAt: '2026-08-24T10:00:00.000Z',
          updatedAt: '2026-08-24T12:10:00.000Z',
        }],
      });
    });

    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Projects' }).click();
    const root = page.locator('[data-plugin-id="verstak.projects"]');
    await expect(root).toBeVisible({ timeout: 10000 });
    await expect(root.locator('[data-project-linked-deal]')).toContainText('Projects/Creatures2.0');
    await expect(root.locator('[data-project-field="workspace"]')).toHaveCount(0);
    await shot(page, 'projects-master-detail.png');

    await root.locator('[data-project-action="edit"]').click();
    const modal = root.locator('[data-project-modal]');
    await modal.locator('[data-deal-picker-toggle]').click();
    await modal.locator('[data-deal-picker-search]').fill('Creatures2.0');
    await expect(modal.locator('[data-deal-id]')).toHaveCount(2);
    await expect(modal.locator('[data-deal-id="deal-creatures"]')).toContainText('Projects/Creatures2.0');
    await expect(modal.locator('[data-deal-id="deal-old-creatures"]')).toContainText('Archive/Creatures2.0');
    await shot(page, 'projects-deal-picker.png');
  });
});
