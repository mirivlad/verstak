import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Desktop localization', () => {
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

  test('switches shell language live and persists the choice', async ({ page }) => {
    await page.locator('[data-settings-menu-button]').click();
    await expect(page.locator('[data-settings-language="system"]')).toHaveAttribute('aria-checked', 'true');
    await page.locator('[data-settings-language="ru"]').click();

    await expect(page.locator('[data-settings-menu-button]')).toHaveAttribute('title', 'Настройки');
    await expect(page.locator('.vault-status')).toContainText('Хранилище: открыто');
    await expect(page.locator('.sidebar .plugin-item').filter({ hasText: 'Тест платформы' })).toBeVisible();
    await expect(page.locator('[data-status-item-id="verstak.platform-test.status"]')).toContainText('Все тесты пройдены');

    await page.locator('[data-settings-menu-button]').click();
    await expect(page.locator('[data-settings-action="plugin-manager"]')).toContainText('Менеджер плагинов');
    await expect(page.locator('[data-settings-language="ru"]')).toHaveAttribute('aria-checked', 'true');
    await page.locator('[data-settings-action="plugin-manager"]').click();
    const pluginManager = page.locator('.plugin-manager');
    await expect(pluginManager).toContainText('Зарегистрировано возможностей:');
    await expect(pluginManager.locator('.registry-section')).toContainText('Реестр возможностей');
    await expect(pluginManager.locator('.registry-section')).toContainText('Возможность');

    await page.reload();
    await waitForAppReady(page);
    await expect(page.locator('[data-settings-menu-button]')).toHaveAttribute('title', 'Настройки');
    await page.locator('[data-settings-menu-button]').click();
    await expect(page.locator('[data-settings-language="ru"]')).toHaveAttribute('aria-checked', 'true');
  });

  test('switches back to English without reloading', async ({ page }) => {
    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-language="ru"]').click();
    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-language="en"]').click();

    await expect(page.locator('[data-settings-menu-button]')).toHaveAttribute('title', 'Settings');
    await expect(page.locator('.vault-status')).toContainText('Vault: open');
  });
});
