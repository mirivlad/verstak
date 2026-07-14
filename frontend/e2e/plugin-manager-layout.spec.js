import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager, setPluginStatus } from './helpers.js';

test.describe('E: Plugin Manager layout', () => {
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

  test('plugin list scrolls through the global main scroll surface and stays responsive', async ({ page }) => {
    await openPluginManager(page);
    const basePluginCount = await page.locator('.plugin-card').count();
    await page.evaluate(() => window.__wailsMock.addSyntheticPlugins(18));
    await page.locator('button.reload-btn').click();
    await expect(page.locator('.plugin-card')).toHaveCount(basePluginCount + 18, { timeout: 10000 });

    const manager = page.locator('.plugin-manager');
    const scrollSurface = page.locator('.content.scroll-surface');
    await expect(manager).toBeVisible();
    await expect(scrollSurface).toBeVisible();

    const desktopMetrics = await scrollSurface.evaluate((node) => ({
      clientHeight: node.clientHeight,
      scrollHeight: node.scrollHeight,
      overflowY: getComputedStyle(node).overflowY,
    }));
    expect(desktopMetrics.overflowY).toBe('auto');
    expect(desktopMetrics.scrollHeight).toBeGreaterThan(desktopMetrics.clientHeight);

    const scrolledTop = await scrollSurface.evaluate((node) => {
      node.scrollTop = node.scrollHeight;
      return node.scrollTop;
    });
    expect(scrolledTop).toBeGreaterThan(0);

    await page.setViewportSize({ width: 720, height: 640 });
    await expect(manager).toBeVisible();

    const narrowMetrics = await scrollSurface.evaluate((node) => ({
      clientWidth: node.clientWidth,
      scrollWidth: node.scrollWidth,
      scrollTop: node.scrollTop,
    }));
    expect(narrowMetrics.scrollWidth).toBeLessThanOrEqual(narrowMetrics.clientWidth + 1);
    expect(narrowMetrics.scrollTop).toBeGreaterThan(0);
  });

  test('platform-test buttons use the global button contract', async ({ page }) => {
    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Platform Test' }).click();

    const saveButton = page.locator('.pt-save-setting');
    await expect(saveButton).toBeVisible({ timeout: 10000 });
    await expect(saveButton).toHaveClass(/btn-primary/);

    const buttonStyle = await saveButton.evaluate((node) => {
      const style = getComputedStyle(node);
      return {
        display: style.display,
        backgroundColor: style.backgroundColor,
        borderRadius: style.borderRadius,
      };
    });
    expect(buttonStyle.display).toBe('inline-flex');
    expect(buttonStyle.backgroundColor).not.toBe('rgba(0, 0, 0, 0)');
    expect(buttonStyle.borderRadius).toBe('6px');
  });

  test('plugin manager summarizes plugin health and elevated permissions before the list', async ({ page }) => {
    await openPluginManager(page);

    const health = page.locator('[data-plugin-manager-summary="health"]');
    await expect(health).toBeVisible();
    await expect(health.locator('[data-plugin-status-summary="loaded"]')).toContainText('12');
    await expect(health.locator('[data-plugin-status-summary="failed"]')).toContainText('0');
    await expect(health.locator('[data-plugin-status-summary="disabled"]')).toContainText('0');

    const risk = page.locator('[data-plugin-manager-summary="risk"]');
    await expect(risk).toBeVisible();
    await expect(risk.locator('[data-plugin-risk-summary="elevated-permissions"]')).toContainText('6');
    await expect(risk).toContainText('elevated permissions');
  });

  test('plugin manager filters by state, declared permissions, capabilities, settings and source', async ({ page }) => {
    await setPluginStatus(page, 'verstak.todo', 'disabled', false);
    await page.evaluate(() => window.__wailsMock.addSyntheticPlugins(1, 'third-party'));
    await openPluginManager(page);

    const results = page.locator('[data-plugin-filter-results]');
    await expect(results).toContainText('Showing');
    const filterSelectStyle = await page.locator('[data-plugin-filter="status"]').evaluate((node) => {
      const style = getComputedStyle(node);
      return { appearance: style.appearance, backgroundImage: style.backgroundImage };
    });
    expect(filterSelectStyle.appearance).toBe('none');
    expect(filterSelectStyle.backgroundImage).not.toBe('none');

    await page.locator('[data-plugin-filter="status"]').selectOption('disabled');
    await expect(page.locator('.plugin-card')).toHaveCount(1);
    await expect(page.locator('.plugin-card')).toContainText('Todos');
    await page.locator('button.reload-btn').click();
    await expect(page.locator('[data-plugin-filter="status"]')).toHaveValue('disabled');
    await expect(page.locator('.plugin-card')).toHaveCount(1);

    await page.locator('[data-plugin-filter-reset]').click();
    await page.locator('[data-plugin-filter-permission="notifications.schedule"]').check();
    await expect(page.locator('.plugin-card')).toHaveCount(1);
    await expect(page.locator('.plugin-card')).toContainText('Todos');
    await page.locator('[data-plugin-filter-permission="secrets.read"]').check();
    await expect(page.locator('.plugin-card')).toHaveCount(2);
    await page.locator('[data-plugin-filter-permission="secrets.read"]').uncheck();

    await page.locator('[data-plugin-filter-reset]').click();
    await page.locator('[data-plugin-filter-capability="verstak/core/notifications/v1"]').check();
    await expect(page.locator('.plugin-card')).toHaveCount(1);
    await expect(page.locator('.plugin-card')).toContainText('Todos');
    await page.locator('[data-plugin-filter-capability="secret-store"]').check();
    await expect(page.locator('.plugin-card')).toHaveCount(2);
    await page.locator('[data-plugin-filter-capability="secret-store"]').uncheck();

    await page.locator('[data-plugin-filter-reset]').click();
    await page.locator('[data-plugin-filter="settings"]').selectOption('with');
    await expect(page.locator('.plugin-card').filter({ hasText: 'Platform Test' })).toBeVisible();
    await expect(page.locator('.plugin-card').filter({ hasText: 'Default Editor' })).toHaveCount(0);

    await page.locator('[data-plugin-filter="source"]').selectOption('third-party');
    await expect(page.locator('.plugin-card')).toHaveCount(0);
    await expect(page.locator('[data-plugin-filter-empty]')).toBeVisible();

    await page.locator('[data-plugin-filter-reset]').click();
    await expect(page.locator('.plugin-card')).toHaveCount(13);
  });

  test('workspace selection keeps exactly one active node', async ({ page }) => {
    const selected = page.locator('.wt-node.selected .wt-label');
    await expect(selected).toHaveCount(1);
    await expect(selected).toHaveText('Project');

    await page.locator('.wt-label').filter({ hasText: 'Test' }).click();

    await expect(selected).toHaveCount(1);
    await expect(selected).toHaveText('Test');
  });

  test('workspace tools render Overview first with Files as one tab', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();

    const tabs = page.locator('.workspace-tabs');
    await expect(tabs).toBeVisible({ timeout: 10000 });
    const overviewTab = tabs.locator('[role="tab"]').filter({ hasText: 'Overview' });
    const filesTab = tabs.locator('[role="tab"]').filter({ hasText: 'Files' });
    await expect(overviewTab).toBeVisible();
    await expect(overviewTab).toHaveAttribute('aria-selected', 'true');
    await expect(filesTab).toBeVisible();
    await expect(filesTab).toHaveAttribute('aria-selected', 'false');
    await expect(page.locator('.workspace-tool')).toHaveCount(0);
    await expect(page.locator('[data-overview-root]')).toBeVisible();

    await filesTab.click();
    await expect(page.locator('.files-root')).toBeVisible();
  });

  test('workspace sidebar creates renames and trashes top-level workspaces', async ({ page }) => {
    await page.locator('button[title="New Deal"]').click();
    const modal = page.locator('[data-workspace-create-modal]');
    await modal.locator('[data-workspace-name]').fill('ClientA');
    await modal.getByRole('button', { name: 'Create Deal' }).click();

    await expect(page.locator('.wt-label').filter({ hasText: 'ClientA' })).toBeVisible();

    const client = page.locator('.wt-node').filter({ hasText: 'ClientA' });
    await client.locator('button[title="Rename Deal"]').click();
    await page.locator('.wt-rename').fill('ClientB');
    await page.locator('button[title="Save rename"]').click();

    await expect(page.locator('.wt-label').filter({ hasText: 'ClientB' })).toBeVisible();
    await expect(page.locator('.wt-label').filter({ hasText: 'ClientA' })).toHaveCount(0);

    const renamed = page.locator('.wt-node').filter({ hasText: 'ClientB' });
    await renamed.locator('button[title="Move Deal to trash"]').click();

    await expect(page.locator('.wt-label').filter({ hasText: 'ClientB' })).toHaveCount(0);
  });

  test('shell icons render through bundled Lucide SVG components', async ({ page }) => {
    const logo = page.locator('.sidebar-logo');
    await expect(logo).toBeVisible();
    await expect(logo).toHaveClass(/lucide/);

    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    const workspaceIcon = page.locator('.wt-node-icon').first();
    await expect(workspaceIcon).toBeVisible();
    await expect(workspaceIcon).toHaveClass(/lucide/);
  });
});
