import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

const FOLDER_ID = '11111111-1111-4111-8111-111111111111';

function folderTree() {
  return {
    roots: [{
      key: `folder:${FOLDER_ID}`,
      kind: 'folder',
      id: FOLDER_ID,
      name: 'Archive',
      path: 'Archive',
      children: [],
    }],
    currentWorkspaceId: '',
    revision: 1,
    warnings: [],
  };
}

async function openFolderEditor(page) {
  const row = page.locator('.wt-node').filter({ hasText: 'Archive' }).first();
  await expect(row).toBeVisible();
  await row.click({ button: 'right' });
  await expect(page.locator('.vt-ctx')).toBeVisible();
  await page.locator('.vt-ctx .vt-menu-item').filter({ hasText: 'Edit folder' }).click();
  await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeVisible();
}

test.describe('Core folder appearance', () => {
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

  test('workspace tree reads and writes appearance through the core Wails API', async ({ page }) => {
    await page.evaluate(async ({ folderId, snapshot }) => {
      window.__wailsMock.setWorkspaceTreeV2(snapshot);
      const error = await window.go.api.App.SetFolderAppearance(folderId, {
        icon: 'star',
        color: '#123456',
      });
      if (error) throw new Error(error);
      window.dispatchEvent(new CustomEvent('verstak:workspace-tree-changed'));
    }, { folderId: FOLDER_ID, snapshot: folderTree() });

    await openFolderEditor(page);
    await expect(page.locator('.vt-appearance-btn')).toContainText('star');
    await expect(page.locator('.vt-color-native')).toHaveValue('#123456');

    await page.locator('.vt-appearance-btn').click();
    const archiveIcon = page.locator('.vt-icon-item').filter({ hasText: /^archive$/i });
    await expect(archiveIcon).toBeVisible();
    await archiveIcon.click();
    await page.locator('.vt-color-native').fill('#654321');
    await page.getByRole('button', { name: 'Save', exact: true }).click();

    await expect.poll(() => page.evaluate((folderId) => (
      window.go.api.App.GetFolderAppearance(folderId)
    ), FOLDER_ID)).toEqual({ icon: 'archive', color: '#654321' });

    await openFolderEditor(page);
    await expect(page.locator('.vt-appearance-btn')).toContainText('archive');
    await expect(page.locator('.vt-color-native')).toHaveValue('#654321');
  });
});
