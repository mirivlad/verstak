import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

async function dragDivider(page, delta) {
  const divider = page.locator('[data-sidebar-resizer]');
  const box = await divider.boundingBox();
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await expect(page.locator('body')).toHaveClass(/sidebar-resizing/);
  await page.mouse.move(box.x + box.width / 2 + delta, box.y + box.height / 2, { steps: 5 });
  await page.mouse.up();
}

test.describe('Resizable sidebar', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
    await page.setViewportSize({ width: 1000, height: 700 });
    await page.goto('/');
    await waitForAppReady(page);
  });

  test.afterEach(async () => {
    consoleCollector.assertNoErrors();
  });

  test('resizes, persists, clamps, and resets from the divider', async ({ page }) => {
    const sidebar = page.locator('.sidebar');
    const divider = page.locator('[data-sidebar-resizer]');
    await expect(divider).toHaveCSS('cursor', 'col-resize');
    await expect(sidebar).toHaveCSS('width', '220px');

    await dragDivider(page, 80);
    await expect(sidebar).toHaveCSS('width', '300px');
    await expect.poll(() => page.evaluate(async () => (await window.go.api.App.GetAppSettings()).sidebarWidth)).toBe(300);

    await page.reload();
    await waitForAppReady(page);
    await expect(sidebar).toHaveCSS('width', '300px');

    await dragDivider(page, 1000);
    await expect(sidebar).toHaveCSS('width', '420px');
    await dragDivider(page, -1000);
    await expect(sidebar).toHaveCSS('width', '180px');

    await divider.dblclick();
    await expect(sidebar).toHaveCSS('width', '220px');
    await expect.poll(() => page.evaluate(async () => (await window.go.api.App.GetAppSettings()).sidebarWidth)).toBe(220);

    await dragDivider(page, 1000);
    await page.setViewportSize({ width: 725, height: 700 });
    await expect(sidebar).toHaveCSS('width', '405px');

    const project = page.locator('.wt-node').filter({ hasText: 'Project' });
    await project.click({ button: 'right' });
    await expect(page.locator('.vt-overlay-host .vt-ctx')).toBeVisible();
  });
});
