import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

const dealTree = {
  roots: [
    {
      key: 'folder:projects', kind: 'folder', id: 'folder-projects', name: 'Projects', path: 'Projects', children: [
        { key: 'workspace:deal-ai', kind: 'workspace', id: 'deal-ai', name: 'AI-server', path: 'Projects/AI-server', children: [] },
        { key: 'workspace:deal-creatures', kind: 'workspace', id: 'deal-creatures', name: 'Creatures2.0', path: 'Projects/Creatures2.0', children: [] },
      ],
    },
    {
      key: 'folder:archive', kind: 'folder', id: 'folder-archive', name: 'Archive', path: 'Archive', children: [
        { key: 'workspace:deal-archive-creatures', kind: 'workspace', id: 'deal-archive-creatures', name: 'Creatures2.0', path: 'Archive/Creatures2.0', children: [] },
      ],
    },
  ],
  currentWorkspaceId: 'deal-creatures',
  revision: 42,
  warnings: [],
};

async function seedProjects(page, projects = []) {
  await page.evaluate(async ({ tree, records }) => {
    window.__wailsMock.setWorkspaceTreeV2(tree);
    const api = window.createPluginAPI('verstak.projects');
    await api.settings.write('projects:global', records);
    api.dispose();
  }, { tree: dealTree, records: projects });
}

async function readProjects(page) {
  return page.evaluate(async () => {
    const api = window.createPluginAPI('verstak.projects');
    const value = await api.settings.read('projects:global');
    api.dispose();
    return value || [];
  });
}

async function openProjects(page) {
  await page.locator('.sidebar .plugin-item').filter({ hasText: 'Projects' }).click();
  await expect(page.locator('[data-plugin-id="verstak.projects"]')).toBeVisible({ timeout: 10000 });
}

test.describe('Projects UX v2', () => {
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

  test('creates a project through the hierarchical Deal picker and keeps the selection', async ({ page }) => {
    await seedProjects(page);
    await openProjects(page);

    await page.locator('[data-project-action="new"]').click();
    const modal = page.locator('[data-project-modal]');
    await expect(modal).toBeVisible();
    await expect(modal.locator('[data-project-field="workspace"]')).toHaveCount(0);

    await modal.locator('[data-project-field="name"]').fill('Projects UX v2');
    await modal.locator('[data-project-field="description"]').fill('Release candidate project');
    await modal.locator('[data-project-field="tags"]').fill('release, ux');

    const picker = modal.locator('[data-deal-picker]');
    await picker.locator('[data-deal-picker-toggle]').click();
    await picker.locator('[data-deal-picker-search]').fill('Projects/Creatures2.0');
    const target = picker.locator('[data-deal-id="deal-creatures"]');
    await expect(target).toContainText('Projects/Creatures2.0');
    await target.click();
    await modal.locator('[data-project-save]').click();

    const projectCard = page.locator('[data-project-id]').filter({ hasText: 'Projects UX v2' });
    await expect(projectCard).toBeVisible();
    await expect(projectCard).toContainText('Projects/Creatures2.0');

    let createdProjectId = '';
    await expect.poll(async () => {
      const records = await readProjects(page);
      const project = records.find((item) => item.name === 'Projects UX v2');
      createdProjectId = project?.id || '';
      return project ? `${project.workspaceId}|${project.workspaceRootPath}` : '';
    }).toBe('deal-creatures|Projects/Creatures2.0');

    // Global Projects is a portfolio. Selecting a card must navigate to its
    // owning Deal and open that exact project in the Deal contribution.
    await projectCard.click();
    await expect(page.locator('[data-project-action="edit"]')).toBeVisible();
    await expect(page.locator('.projects-header-name')).toHaveText('Projects UX v2');

    await page.locator('[data-project-action="edit"]').click();
    const editModal = page.locator('[data-project-modal]');
    await expect(editModal.locator('[data-deal-picker-toggle]')).toContainText('Projects/Creatures2.0');
    const nativeAppearance = await editModal.locator('[data-project-field="status"]').evaluate((node) => getComputedStyle(node).appearance);
    expect(nativeAppearance).toBe('none');
    await editModal.getByRole('button', { name: 'Cancel' }).click();

    await page.locator('[data-project-action="new"]').click();
    const secondModal = page.locator('[data-project-modal]');
    await secondModal.locator('[data-project-field="name"]').fill('Other project');
    await secondModal.locator('[data-project-save]').click();
    await expect(page.locator('.projects-header-name')).toHaveText('Other project');

    const switcher = page.locator('[data-project-switcher]');
    await expect(switcher).toBeVisible();
    await switcher.selectOption(createdProjectId);
    await expect(page.locator('.projects-header-name')).toHaveText('Projects UX v2');
    await switcher.selectOption({ label: 'Other project' });
    await switcher.selectOption(createdProjectId);
    await expect(page.locator('.projects-header-name')).toHaveText('Projects UX v2');

    const tabs = await page.locator('[data-project-tab]').allTextContents();
    expect(tabs).toEqual(['Overview', 'Milestones', 'Tasks', 'Notes', 'Files', 'Activity', 'Links']);

    await page.setViewportSize({ width: 680, height: 900 });
    await expect.poll(() => page.locator('.projects-root').evaluate((node) => node.scrollWidth <= node.clientWidth + 1)).toBe(true);
  });

  test('migrates v0.1.4 path links, preserves unresolved links and disambiguates duplicate Deal names', async ({ page }) => {
    await seedProjects(page, [
      {
        id: 'legacy-project', name: 'Creatures research', description: 'old record', status: 'active', priority: 'normal', tags: [],
        workspaceRootPath: 'Projects/Creatures2.0', milestones: [], links: [], events: [],
        createdAt: '2026-08-01T00:00:00.000Z', updatedAt: '2026-08-01T00:00:00.000Z',
      },
      {
        id: 'orphan-project', name: 'Old archived work', status: 'paused', priority: 'low', tags: [],
        workspaceRootPath: 'Deleted/OldDeal', milestones: [], links: [], events: [],
        createdAt: '2026-07-01T00:00:00.000Z', updatedAt: '2026-07-01T00:00:00.000Z',
      },
    ]);
    await openProjects(page);

    await expect.poll(async () => {
      const records = await readProjects(page);
      const legacy = records.find((item) => item.id === 'legacy-project');
      return legacy ? legacy.workspaceId : '';
    }).toBe('deal-creatures');

    let records = await readProjects(page);
    const orphan = records.find((item) => item.id === 'orphan-project');
    expect(orphan.workspaceId || '').toBe('');
    expect(orphan.workspaceRootPath).toBe('Deleted/OldDeal');

    await page.locator('[data-project-id="legacy-project"]').click();
    await page.locator('[data-project-action="edit"]').click();
    const modal = page.locator('[data-project-modal]');
    await modal.locator('[data-deal-picker-toggle]').click();
    await modal.locator('[data-deal-picker-search]').fill('Creatures2.0');

    const matches = modal.locator('[data-deal-id]');
    await expect(matches).toHaveCount(2);
    await expect(modal.locator('[data-deal-id="deal-creatures"]')).toContainText('Projects/Creatures2.0');
    await expect(modal.locator('[data-deal-id="deal-archive-creatures"]')).toContainText('Archive/Creatures2.0');
    await modal.locator('[data-deal-id="deal-archive-creatures"]').click();
    await modal.locator('[data-project-save]').click();

    await expect.poll(async () => {
      const current = (await readProjects(page)).find((item) => item.id === 'legacy-project');
      return current ? `${current.workspaceId}|${current.workspaceRootPath}` : '';
    }).toBe('deal-archive-creatures|Archive/Creatures2.0');

    records = await readProjects(page);
    const legacy = records.find((item) => item.id === 'legacy-project');
    expect(legacy.events.some((event) => event.type === 'project.linked' && String(event.to).includes('Archive/Creatures2.0'))).toBe(true);
    expect(legacy.events.some((event) => event.type === 'project.updated')).toBe(false);

    // Relinking moves the project outside the current Deal. Return to the
    // portfolio, verify its new owner there, then open the new Deal context.
    await openProjects(page);
    const movedCard = page.locator('[data-project-id="legacy-project"]');
    await expect(movedCard).toContainText('Archive/Creatures2.0');
    await movedCard.click();
    await expect(page.locator('.projects-header-name')).toHaveText('Creatures research');
    await page.locator('[data-project-action="edit"]').click();
    await expect(page.locator('[data-project-modal] [data-deal-picker-toggle]')).toContainText('Archive/Creatures2.0');
  });
});
