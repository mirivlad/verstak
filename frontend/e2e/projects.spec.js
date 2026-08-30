import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Project Meta', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
  });

  test.afterEach(async () => consoleCollector.assertNoErrors());

  test('global portfolio reads Project Meta from Deal UUID namespaces', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginDealConfig('verstak.projects', '11111111-1111-4111-8111-111111111111', {
        name: 'Project Alpha', description: 'Portfolio card', status: 'active', priority: 'high', tags: ['release'], startDate: '2026-08-01', dueDate: '2026-08-31',
      });
      await window.go.api.App.WritePluginDealConfig('verstak.projects', '22222222-2222-4222-8222-222222222222', {
        name: 'Project Beta', status: 'paused', priority: 'low', tags: ['client'],
      });
    });
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Project portfolio' }).click();
    const portfolio = page.locator('[data-project-meta-portfolio]');
    await expect(portfolio).toBeVisible();
    await expect(portfolio.locator('[data-project-meta-deal="11111111-1111-4111-8111-111111111111"]')).toContainText('Project Alpha');
    await expect(portfolio.locator('[data-project-meta-deal="22222222-2222-4222-8222-222222222222"]')).toContainText('Project Beta');
  });

  test('selected Deal edits and persists its own Project Meta', async ({ page }) => {
    await page.getByRole('tab', { name: 'Project Meta' }).click();
    const panel = page.locator('[role="tabpanel"][aria-label="Project Meta"]');
    await panel.locator('[data-project-meta-field="name"]').fill('Project Deal');
    await panel.locator('[data-project-meta-field="description"]').fill('Metadata is Deal-owned.');
    await panel.locator('[data-project-meta-field="status"]').selectOption('paused');
    await panel.locator('[data-project-meta-field="priority"]').selectOption('high');
    await panel.locator('[data-project-meta-field="startDate"]').fill('2026-08-01');
    await panel.locator('[data-project-meta-field="dueDate"]').fill('2026-08-31');
    await panel.locator('[data-project-meta-field="tags"]').fill('release, ux');
    await panel.locator('[data-project-meta-action="save"]').click();
    await expect(panel.locator('.project-meta-saved')).toBeVisible();
    await expect.poll(async () => page.evaluate(async () => (await window.go.api.App.ReadPluginDealConfig('verstak.projects', '11111111-1111-4111-8111-111111111111'))[0])).toMatchObject({
      name: 'Project Deal', status: 'paused', priority: 'high', tags: ['release', 'ux'], startDate: '2026-08-01', dueDate: '2026-08-31',
    });
  });
});
