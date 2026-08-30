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
    const recipe = {
      workspaceTools: ['verstak.projects', 'verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.activity'],
      initialFolders: ['Notes', 'Files'],
      initialFiles: [],
      provenance: { templateId: 'visual-project', templateName: 'Visual project', templateVersion: 1 },
    };
    const unwrap = (result) => Array.isArray(result) ? result[0] : result;
    const creatures = unwrap(await window.go.api.App.PluginCreateWorkspace('verstak.templates', '', 'Creatures2.0', recipe));
    const ai = unwrap(await window.go.api.App.PluginCreateWorkspace('verstak.templates', '', 'AI-server', recipe));
    if (!creatures?.workspaceId || !ai?.workspaceId) throw new Error('visual Deals were not created');

    await window.go.api.App.WritePluginDealConfig('verstak.projects', creatures.workspaceId, {
      name: 'Creatures UX',
      description: 'Deal properties stay on the Deal while portfolio remains global.',
      status: 'active', priority: 'high', tags: ['release', 'ux'],
      startDate: '2026-08-24', dueDate: '2026-08-30', updatedAt: '2026-08-24T12:10:00.000Z',
    });
    await window.go.api.App.WritePluginDealConfig('verstak.projects', ai.workspaceId, {
      name: 'Local AI steward',
      description: 'A second Deal makes the portfolio visibly cross-Deal.',
      status: 'paused', priority: 'normal', tags: ['assistant'],
      updatedAt: '2026-08-23T10:00:00.000Z',
    });
    return { creaturesId: creatures.workspaceId, aiId: ai.workspaceId };
  });
}

test.describe('Visual audit: Project Meta portfolio', () => {
  test('portfolio and Deal-owned Project Meta use Deal UUIDs', async ({ page }) => {
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    await mkdir(VISUAL_DIR, { recursive: true });
    const ids = await seed(page);

    await expect(page.locator('.sidebar')).not.toContainText('Platform Test');
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Project portfolio' }).click();
    const portfolio = page.locator('[data-project-meta-portfolio]');
    await expect(portfolio).toBeVisible({ timeout: 10000 });
    await expect(portfolio.locator('[data-project-meta-deal]')).toHaveCount(4);
    await expect(portfolio.locator(`[data-project-meta-deal="${ids.creaturesId}"]`)).toContainText('Creatures UX');
    await expect(portfolio.locator(`[data-project-meta-deal="${ids.aiId}"]`)).toContainText('Local AI steward');
    await shot(page, 'project-meta-portfolio.png');

    await portfolio.locator(`[data-project-meta-deal="${ids.creaturesId}"]`).click();
    const meta = page.locator('[role="tabpanel"][aria-label="Project Meta"]');
    await expect(meta).toBeVisible({ timeout: 10000 });
    await expect(meta.locator('.project-meta-title')).toHaveText('Project Meta');
    await expect(meta.locator('[data-project-meta-field="name"]')).toHaveValue('Creatures UX');
    await expect(meta.locator('[data-project-meta-field="dueDate"]')).toHaveValue('2026-08-30');
    await shot(page, 'project-meta-deal.png');
  });
});
