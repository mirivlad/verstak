import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test('global Secrets lists, filters, and creates Deal-scoped secrets', async ({ page }) => {
  const consoleCollector = setupConsoleCollector(page);
  await resetMockState(page);
  await page.goto('/');
  await waitForAppReady(page);

  await page.evaluate(async () => {
    const [record, error] = await window.go.api.App.PluginSecretsWrite('verstak.secrets', {
      id: 'project.global-secret',
      title: 'Project API',
      username: 'project-user',
      value: 'project-value',
      scope: { kind: 'workspace', workspaceRootPath: 'Project' },
    });
    if (error || !record?.id) throw new Error(error || 'could not create a project secret');
  });

  await page.locator('.sidebar .plugin-item').filter({ hasText: 'Secrets' }).click();
  const secrets = page.locator('.secrets-root');
  await expect(secrets).toBeVisible({ timeout: 10000 });
  await expect(secrets).toContainText('First secret');
  await expect(secrets).toContainText('Project API');

  await secrets.locator('[data-secret-scope-filter]').selectOption('workspace:Project');
  await expect(secrets).toContainText('Project API');
  await expect(secrets).not.toContainText('First secret');

  await secrets.locator('[data-secret-search]').fill('no matching secret');
  await expect(secrets).toContainText('No secrets');
  await secrets.locator('[data-secret-search]').fill('');

  await secrets.locator('[data-secret-scope-filter]').selectOption('all');
  await secrets.getByRole('button', { name: 'New' }).click();
  await secrets.locator('[data-secret-title]').fill('Project deployment token');
  await secrets.locator('[data-secret-value]').fill('deployment-token-value');
  await secrets.locator('[data-secret-scope]').selectOption('workspace');
  await secrets.locator('[data-secret-workspace]').selectOption('Project');
  await secrets.locator('[data-secret-save]').click();

  await expect.poll(async () => page.evaluate(async () => {
    const [records, error] = await window.go.api.App.PluginSecretsList('verstak.secrets');
    if (error) throw new Error(error);
    const record = records.find((item) => item.title === 'Project deployment token');
    return record && [record.scope?.kind, record.scope?.workspaceRootPath].join('|');
  })).toBe('workspace|Project');
  consoleCollector.assertNoErrors();
});
