import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Git Deal plugin', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
    await page.evaluate(async () => {
      const workspaceId = '11111111-1111-4111-8111-111111111111';
      await window.go.api.App.UpdateWorkspaceV2Tools(workspaceId, [
        'verstak.projects', 'verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.journal',
        'verstak.activity', 'verstak.browser-inbox', 'verstak.git',
      ]);
      window.dispatchEvent(new CustomEvent('verstak:workspace-tools-changed', { detail: { workspaceId } }));
    });
    await page.getByRole('tab', { name: 'Git', exact: true }).click();
    await expect(page.locator('[data-git-root]')).toBeVisible();
  });

  test.afterEach(async () => consoleCollector.assertNoErrors());

  async function addRepository(page, name) {
    const root = page.locator('[data-git-root]');
    await root.locator('[data-git-action="add"]').click();
    const form = root.locator('[data-git-form="add"]');
    await form.locator('[data-git-field="name"]').fill(name);
    await form.locator('[data-git-field="remote"]').fill('https://example.com/owner/repo.git');
    await form.locator('[data-git-field="branch"]').fill('main');
    await form.locator('[data-git-action="save"]').click();
    const card = root.locator('[data-git-repository]').filter({ hasText: name });
    await expect(card).toBeVisible();
    return card;
  }

  test('descriptor can clone, report status, run network actions and open its checkout', async ({ page }) => {
    const card = await addRepository(page, 'Primary repository');
    await expect(card).toContainText('Not cloned on this device');

    await card.locator('[data-git-action="clone"]').click();
    await expect(card).toContainText('Clean');
    await expect(card).toContainText('0123456');

    await card.locator('[data-git-action="fetch"]').click();
    await card.locator('[data-git-action="pull"]').click();
    await card.locator('[data-git-action="push"]').click();
    await card.locator('[data-git-action="open-directory"]').click();

    await expect.poll(() => page.evaluate(() => window.__wailsMockExternalOpens)).toContainEqual({
      action: 'git-folder', path: 'Primary-repository',
    });

    const stored = await page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginDataNDJSON('verstak.git', 'repositories');
      return Array.isArray(result) && Array.isArray(result[0]) ? result[0] : result;
    });
    expect(stored).toHaveLength(1);
    expect(stored[0]).toMatchObject({
      workspaceId: '11111111-1111-4111-8111-111111111111',
      name: 'Primary repository',
      checkoutName: 'Primary-repository',
      remoteUrl: 'https://example.com/owner/repo.git',
      defaultBranch: 'main',
    });
  });

  test('a synced descriptor can adopt an existing local checkout through Core', async ({ page }) => {
    const card = await addRepository(page, 'Existing repository');
    await expect(card).toContainText('Not cloned on this device');
    await card.locator('[data-git-action="register-existing"]').click();
    await expect(card).toContainText('Clean');
    await expect(card.locator('[data-git-action="open-directory"]')).toBeVisible();
  });
});
