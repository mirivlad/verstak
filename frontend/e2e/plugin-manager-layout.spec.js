import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

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
    await page.evaluate(() => window.__wailsMock.addSyntheticPlugins(18));
    await page.locator('button.reload-btn').click();
    await expect(page.locator('.plugin-card')).toHaveCount(19, { timeout: 10000 });

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

  test('workspace selection keeps exactly one active node', async ({ page }) => {
    const selected = page.locator('.wt-node.selected .wt-label');
    await expect(selected).toHaveCount(1);
    await expect(selected).toHaveText('Alpha Case');

    await page.locator('.wt-label').filter({ hasText: 'Beta Case' }).click();

    await expect(selected).toHaveCount(1);
    await expect(selected).toHaveText('Beta Case');
  });
});
