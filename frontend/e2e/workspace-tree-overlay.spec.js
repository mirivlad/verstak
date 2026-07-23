import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Workspace tree overlay', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
    await page.setViewportSize({ width: 800, height: 640 });
    await page.goto('/');
    await waitForAppReady(page);
  });

  test.afterEach(async () => {
    consoleCollector.assertNoErrors();
  });

  test('context menu beside the sidebar divider stays in the viewport and closes normally', async ({ page }) => {
    const deal = page.locator('.wt-node').filter({ hasText: 'Project' });
    const dealBox = await deal.boundingBox();
    await deal.click({
      button: 'right',
      position: { x: dealBox.width - 2, y: dealBox.height / 2 },
    });

    const menu = page.locator('.vt-overlay-host .vt-ctx');
    await expect(menu).toBeVisible();
    expect(await menu.evaluate((node) => node.parentElement?.classList.contains('vt-overlay-host'))).toBe(true);
    expect(await menu.evaluate((node) => node.closest('.sidebar'))).toBeNull();

    await page.setViewportSize({ width: 360, height: 240 });
    await expect.poll(async () => {
      const box = await menu.boundingBox();
      return box && {
        left: box.x >= 8,
        top: box.y >= 8,
        right: box.x + box.width <= 352,
        bottom: box.y + box.height <= 232,
      };
    }).toEqual({ left: true, top: true, right: true, bottom: true });

    await page.keyboard.press('Escape');
    await expect(menu).toHaveCount(0);

    await page.setViewportSize({ width: 800, height: 640 });
    await deal.click({ button: 'right' });
    await expect(menu).toBeVisible();
    await page.locator('.main-content-header').click();
    await expect(menu).toHaveCount(0);
  });
});
