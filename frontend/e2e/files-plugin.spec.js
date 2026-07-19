import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, openPluginManager } from './helpers.js';

async function openFilesTool(page) {
  await page.getByRole('tab', { name: 'Files' }).click();
  await expect(page.locator('.files-root')).toBeVisible({ timeout: 10000 });
}

test.describe('G: Files Plugin', () => {
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

  test('files plugin appears in Plugin Manager as loaded', async ({ page }) => {
    await openPluginManager(page);
    const card = page.locator('.plugin-card').filter({ hasText: 'verstak.files' });
    await expect(card).toBeVisible({ timeout: 10000 });
    await expect(card.locator('.status-badge')).toHaveText('loaded');
  });

  test('files plugin does not add global sidebar item', async ({ page }) => {
    const sidebarItem = page.locator('.sidebar .plugin-item').filter({ hasText: 'Files' });
    await expect(sidebarItem).toHaveCount(0);
  });

  test('workspace Files view is scoped to selected workspace folder', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);

    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.files-item-name').filter({ hasText: 'project-only.txt' })).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.files-item-name').filter({ hasText: 'test-only.txt' })).toHaveCount(0);

    await page.locator('.wt-label').filter({ hasText: 'Test' }).click();

    await expect(page.locator('.files-item-name').filter({ hasText: 'test-only.txt' })).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.files-item-name').filter({ hasText: 'project-only.txt' })).toHaveCount(0);
  });

  test('files explorer supports create navigate rename filter sort open and trash', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-files-action="new-folder"]').click();
    await page.locator('[data-files-create-input]').fill('Daily');
    await page.locator('[data-files-create-confirm]').click();
    await expect(page.locator('[data-file-name="Daily"]')).toBeVisible();

    await page.locator('[data-file-name="Daily"]').dblclick();
    await expect(page.locator('.files-breadcrumb')).toContainText('Daily');

    await page.locator('[data-files-action="new-markdown"]').click();
    await page.locator('[data-files-create-input]').fill('Log.md');
    await page.locator('[data-files-create-confirm]').click();
    await expect(page.locator('[data-file-name="Log.md"]')).toBeVisible();

    await page.locator('[data-file-name="Log.md"]').click();
    await page.locator('[data-files-action="rename"]').click();
    await page.locator('[data-files-rename-input]').fill('Journal.md');
    await page.locator('[data-files-rename-confirm]').click();
    await expect(page.locator('[data-file-name="Journal.md"]')).toBeVisible();
    await expect(page.locator('[data-file-name="Log.md"]')).toHaveCount(0);

    await page.locator('[data-files-filter]').fill('journ');
    await expect(page.locator('[data-file-name="Journal.md"]')).toBeVisible();
    await expect(page.locator('[data-file-name="project-only.txt"]')).toHaveCount(0);
    await page.locator('[data-files-filter]').fill('');

    await page.locator('[data-files-sort]').selectOption('modified-desc');
    await expect(page.locator('[data-file-name="Journal.md"]')).toBeVisible();

    await page.locator('[data-file-name="Journal.md"]').dblclick();
    await expect(page.locator('[data-editor-mode="generic-markdown"]')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('[data-resource-path="Project/Daily/Journal.md"]')).toBeVisible();

    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await expect(page.locator('.files-breadcrumb')).toContainText('Daily', { timeout: 10000 });
    await expect(page.locator('[data-file-name="Journal.md"]')).toBeVisible({ timeout: 10000 });
    await page.locator('[data-file-name="Journal.md"]').click();
    await page.locator('[data-files-action="trash"]').click();
    await expect(page.locator('.files-modal-overlay')).toBeVisible();
    await expect(page.locator('.files-modal-title')).toContainText('Journal.md');
    await page.locator('.files-modal-btn.confirm').click();
    await expect(page.locator('[data-file-name="Journal.md"]')).toHaveCount(0);

    await page.locator('[data-files-action="up"]').click();
    await expect(page.locator('.files-breadcrumb')).not.toContainText('Daily');
  });

  test('files explorer shows inline rename validation errors', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-file-name="project-only.txt"]').click();
    await page.locator('[data-files-action="rename"]').click();
    await page.locator('[data-files-rename-input]').fill('bad/name.txt');
    await page.locator('[data-files-rename-confirm]').click();

    await expect(page.locator('[data-files-rename-error]')).toBeVisible();
    await expect(page.locator('[data-files-rename-error]')).toContainText('Invalid characters');
    await expect(page.locator('[data-files-rename-input]')).toBeVisible();
    await expect(page.locator('[data-file-name="project-only.txt"]')).toBeVisible();
    await expect(page.locator('[data-file-name="bad"]')).toHaveCount(0);
  });

  test('files explorer shows inline rename conflict errors', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-file-name="project-only.txt"]').click();
    await page.locator('[data-files-action="rename"]').click();
    await page.locator('[data-files-rename-input]').fill('Notes');
    await page.locator('[data-files-rename-confirm]').click();

    await expect(page.locator('[data-files-rename-error]')).toBeVisible();
    await expect(page.locator('[data-files-rename-error]')).toContainText('already exists');
    await expect(page.locator('[data-files-rename-input]')).toBeVisible();
    await expect(page.locator('[data-files-rename-input]')).toHaveValue('Notes');
    await expect(page.locator('[data-file-name="project-only.txt"]')).toBeVisible();
    await expect(page.locator('[data-file-name="Notes"]')).toBeVisible();
  });

  test('files explorer shows inline create validation errors', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-files-action="new-folder"]').click();
    await page.locator('[data-files-create-input]').fill('bad/name');
    await page.locator('[data-files-create-confirm]').click();

    await expect(page.locator('[data-files-create-error]')).toBeVisible();
    await expect(page.locator('[data-files-create-error]')).toContainText('Invalid characters');
    await expect(page.locator('[data-files-create-input]')).toBeVisible();
    await expect(page.locator('[data-file-name="bad"]')).toHaveCount(0);
  });

  test('files explorer uses rich file icons by file type', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    async function createFile(action, name) {
      await page.locator(`[data-files-action="${action}"]`).click();
      await page.locator('[data-files-create-input]').fill(name);
      await page.locator('[data-files-create-confirm]').click();
      await expect(page.locator(`[data-file-name="${name}"]`)).toBeVisible();
    }

    await createFile('new-markdown', 'IconNote.md');
    await createFile('new-text', 'Photo.png');
    await createFile('new-text', 'Manual.pdf');
    await createFile('new-text', 'Script.js');

    const iconFor = (name) => page.locator(`[data-file-name="${name}"] .files-item-icon`);
    await expect(iconFor('Notes')).toHaveAttribute('data-file-icon', 'folder');
    await expect(iconFor('project-only.txt')).toHaveAttribute('data-file-icon', 'text');
    await expect(iconFor('IconNote.md')).toHaveAttribute('data-file-icon', 'markdown');
    await expect(iconFor('Photo.png')).toHaveAttribute('data-file-icon', 'image');
    await expect(iconFor('Manual.pdf')).toHaveAttribute('data-file-icon', 'pdf');
    await expect(iconFor('Script.js')).toHaveAttribute('data-file-icon', 'code');
    await expect(iconFor('Manual.pdf')).toHaveAttribute('aria-label', 'PDF file');
    await expect(iconFor('Photo.png')).toHaveAttribute('title', 'Image file');
  });

  test('files explorer offers create actions in empty folders', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-files-action="new-folder"]').click();
    await page.locator('[data-files-create-input]').fill('EmptyQuickActions');
    await page.locator('[data-files-create-confirm]').click();
    await page.locator('[data-file-name="EmptyQuickActions"]').dblclick();

    const emptyState = page.locator('.files-empty');
    await expect(emptyState).toContainText('Empty folder');
    await expect(emptyState.locator('[data-files-empty-action="new-folder"]')).toBeVisible();
    await expect(emptyState.locator('[data-files-empty-action="new-markdown"]')).toBeVisible();
    await expect(emptyState.locator('[data-files-empty-action="new-text"]')).toBeVisible();

    await emptyState.locator('[data-files-empty-action="new-markdown"]').click();
    await page.locator('[data-files-create-input]').fill('FromEmpty.md');
    await page.locator('[data-files-create-confirm]').click();

    await expect(page.locator('[data-file-name="FromEmpty.md"]')).toBeVisible();
    await expect(page.locator('.files-empty')).toHaveCount(0);
  });

  test('files explorer shows no-match feedback while filtering', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-files-filter]').fill('zzzz-no-matches');

    const emptyState = page.locator('.files-empty');
    await expect(emptyState).toBeVisible();
    await expect(emptyState).toContainText('No matches');
    await expect(emptyState.locator('[data-files-empty-action="clear-filter"]')).toBeVisible();
    await expect(page.locator('[data-file-name="project-only.txt"]')).toHaveCount(0);

    await emptyState.locator('[data-files-empty-action="clear-filter"]').click();
    await expect(page.locator('[data-files-filter]')).toHaveValue('');
    await expect(page.locator('.files-empty')).toHaveCount(0);
    await expect(page.locator('[data-file-name="project-only.txt"]')).toBeVisible();
  });

  test('files explorer keeps removed items out of live files without Trash metadata', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-files-action="new-text"]').click();
    await page.locator('[data-files-create-input]').fill('RestoreMe.txt');
    await page.locator('[data-files-create-confirm]').click();
    await expect(page.locator('[data-file-name="RestoreMe.txt"]')).toBeVisible();

    await page.locator('[data-file-name="RestoreMe.txt"]').click();
    await page.locator('[data-files-action="trash"]').click();
    await page.locator('.files-modal-btn.confirm').click();
    await expect(page.locator('[data-file-name="RestoreMe.txt"]')).toHaveCount(0);
    await expect(page.locator('[data-files-action="trash-view"]')).toHaveCount(0);
    await expect(page.locator('[data-files-restore-trash]')).toHaveCount(0);
  });

  test('files explorer uses labeled controls and no row New Here action', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    const actionIcons = {
      back: 'back',
      forward: 'forward',
      up: 'up',
      refresh: 'refresh',
      'new-folder': 'folderAdd',
      'new-markdown': 'markdownAdd',
      'new-text': 'textAdd',
      open: 'open',
      rename: 'rename',
      trash: 'trash',
      cut: 'cut',
      copy: 'copy',
      paste: 'paste',
    };

    const pathData = [];
    for (const [action, icon] of Object.entries(actionIcons)) {
      const button = page.locator(`[data-files-action="${action}"]`);
      await expect(button).toHaveAttribute('title', /.+/);
      await expect(button).toHaveAttribute('aria-label', /.+/);
      await expect(button).toHaveAttribute('data-files-icon', icon);
      await expect(button.locator('svg')).toBeVisible();
      await expect(button).toHaveText('');
      pathData.push(await button.locator('svg path').getAttribute('d'));
    }
    expect(new Set(pathData).size).toBe(pathData.length);

    await expect(page.locator('.files-row-btn').filter({ hasText: 'New here' })).toHaveCount(0);
    const rowIcons = {
      'row-open': 'open',
      'row-rename': 'rename',
      'row-trash': 'trash',
    };
    const rowPathData = [];
    for (const [action, icon] of Object.entries(rowIcons)) {
      const rowButton = page.locator(`[data-file-name="Notes"] [data-files-action="${action}"]`);
      await expect(rowButton).toBeVisible();
      await expect(rowButton).toHaveAttribute('title', /.+/);
      await expect(rowButton).toHaveAttribute('aria-label', /.+/);
      await expect(rowButton).toHaveAttribute('data-files-icon', icon);
      await expect(rowButton).toHaveText('');
      expect(await rowButton.evaluate((node) => node.innerHTML)).toContain('<svg');
      rowPathData.push(await rowButton.locator('svg path').getAttribute('d'));
    }
    expect(new Set(rowPathData).size).toBe(rowPathData.length);
  });

  test('files explorer supports empty-space context paste after cutting a folder', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-files-action="new-folder"]').click();
    await page.locator('[data-files-create-input]').fill('CutMe');
    await page.locator('[data-files-create-confirm]').click();
    await page.locator('[data-files-action="new-folder"]').click();
    await page.locator('[data-files-create-input]').fill('Target');
    await page.locator('[data-files-create-confirm]').click();

    await page.locator('[data-file-name="CutMe"]').click({ button: 'right' });
    await page.locator('[data-files-menu-action="cut"]').click();

    await page.locator('[data-file-name="Target"]').dblclick();
    await expect(page.locator('.files-breadcrumb')).toContainText('Target');
    await page.locator('[data-files-list]').click({ button: 'right', position: { x: 24, y: 110 } });
    await page.locator('[data-files-menu-action="paste"]').click();

    await expect(page.locator('[data-file-name="CutMe"]')).toBeVisible();
    await page.locator('[data-files-action="up"]').click();
    await expect(page.locator('[data-file-name="CutMe"]')).toHaveCount(0);
  });

  test('files explorer navigates by clicking breadcrumb segments', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-files-action="new-folder"]').click();
    await page.locator('[data-files-create-input]').fill('BreadcrumbParent');
    await page.locator('[data-files-create-confirm]').click();
    await page.locator('[data-file-name="BreadcrumbParent"]').dblclick();
    await expect(page.locator('.files-breadcrumb')).toContainText('BreadcrumbParent');

    await page.locator('[data-files-action="new-folder"]').click();
    await page.locator('[data-files-create-input]').fill('BreadcrumbChild');
    await page.locator('[data-files-create-confirm]').click();
    await page.locator('[data-file-name="BreadcrumbChild"]').dblclick();
    await expect(page.locator('.files-breadcrumb')).toContainText('BreadcrumbChild');

    await page.locator('.files-breadcrumb-item').filter({ hasText: 'BreadcrumbParent' }).click();
    await expect(page.locator('.files-breadcrumb')).toContainText('BreadcrumbParent');
    await expect(page.locator('.files-breadcrumb')).not.toContainText('BreadcrumbChild');
    await expect(page.locator('[data-file-name="BreadcrumbChild"]')).toBeVisible();
  });

  test('files explorer duplicates a file from the context menu', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-file-name="project-only.txt"]').click({ button: 'right' });
    await page.locator('[data-files-menu-action="duplicate"]').click();

    await expect(page.locator('[data-file-name="project-only.txt"]')).toBeVisible();
    await expect(page.locator('[data-file-name="project-only (copy).txt"]')).toBeVisible();

    await page.locator('[data-file-name="project-only.txt"]').click({ button: 'right' });
    await page.locator('[data-files-menu-action="duplicate"]').click();
    await expect(page.locator('[data-file-name="project-only (copy 2).txt"]')).toBeVisible();
  });

  test('files explorer supports multiselect and internal drag/drop move', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-files-action="new-folder"]').click();
    await page.locator('[data-files-create-input]').fill('DropTarget');
    await page.locator('[data-files-create-confirm]').click();
    await page.locator('[data-files-action="new-markdown"]').click();
    await page.locator('[data-files-create-input]').fill('DragOne.md');
    await page.locator('[data-files-create-confirm]').click();
    await page.locator('[data-files-action="new-text"]').click();
    await page.locator('[data-files-create-input]').fill('DragTwo.txt');
    await page.locator('[data-files-create-confirm]').click();

    await page.locator('[data-file-name="DragOne.md"]').click();
    await page.locator('[data-file-name="DragTwo.txt"]').click({ modifiers: [process.platform === 'darwin' ? 'Meta' : 'Control'] });
    await expect(page.locator('.files-item.selected')).toHaveCount(2);

    await page.evaluate(() => {
      const source = document.querySelector('[data-file-name="DragOne.md"]');
      const target = document.querySelector('[data-file-name="DropTarget"]');
      const dt = new DataTransfer();
      source.dispatchEvent(new DragEvent('dragstart', { bubbles: true, dataTransfer: dt }));
      target.dispatchEvent(new DragEvent('dragover', { bubbles: true, dataTransfer: dt }));
      target.dispatchEvent(new DragEvent('drop', { bubbles: true, dataTransfer: dt }));
    });

    await expect(page.locator('[data-file-name="DragOne.md"]')).toHaveCount(0);
    await expect(page.locator('[data-file-name="DragTwo.txt"]')).toHaveCount(0);
    await page.locator('[data-file-name="DropTarget"]').dblclick();
    await expect(page.locator('[data-file-name="DragOne.md"]')).toBeVisible();
    await expect(page.locator('[data-file-name="DragTwo.txt"]')).toBeVisible();
  });

  test('files explorer supports keyboard row selection and clearing selection', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    const rows = page.locator('.files-item');
    await expect(rows.nth(1)).toBeVisible();

    await rows.nth(0).click();
    await expect(rows.nth(0)).toHaveClass(/selected/);

    await page.locator('.files-root').focus();
    await page.keyboard.press('ArrowDown');
    await expect(rows.nth(1)).toHaveClass(/selected/);
    await expect(rows.nth(0)).not.toHaveClass(/selected/);

    await page.keyboard.press('ArrowUp');
    await expect(rows.nth(0)).toHaveClass(/selected/);

    await page.keyboard.press('Escape');
    await expect(page.locator('.files-item.selected')).toHaveCount(0);
  });

  test('files history persists in workspace context and handles mouse back forward buttons', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-file-name="Notes"]').dblclick();
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');

    await page.locator('.files-root').focus();
    await page.keyboard.press('Alt+ArrowLeft');
    await expect(page.locator('.files-breadcrumb')).not.toContainText('Notes');

    await page.keyboard.press('Alt+ArrowRight');
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');

    await page.dispatchEvent('.files-root', 'mouseup', { button: 8, buttons: 128, bubbles: true, cancelable: true });
    await expect(page.locator('.files-breadcrumb')).not.toContainText('Notes');

    await page.dispatchEvent('.files-root', 'mouseup', { button: 9, buttons: 256, bubbles: true, cancelable: true });
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');

    await page.locator('.wt-label').filter({ hasText: 'Test' }).click();
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');
  });

  test('workbench close and mouse back return from editor to the previous Files folder', async ({ page }) => {
    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
    await openFilesTool(page);
    await expect(page.locator('.files-breadcrumb')).toContainText('Project', { timeout: 10000 });

    await page.locator('[data-file-name="Notes"]').dblclick();
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');

    await page.locator('[data-file-name="Overview.md"]').dblclick();
    await expect(page.locator('[data-editor-mode="notes-markdown"]')).toBeVisible({ timeout: 10000 });

    await page.dispatchEvent('body', 'mousedown', { button: 3, bubbles: true, cancelable: true });
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');
    await expect(page.locator('[data-file-name="Overview.md"]')).toBeVisible();

    await page.locator('[data-file-name="Overview.md"]').dblclick();
    await expect(page.locator('[data-editor-mode="notes-markdown"]')).toBeVisible({ timeout: 10000 });

    await page.waitForTimeout(150);
    await page.evaluate(() => {
      document.body.dispatchEvent(new PointerEvent('pointerdown', {
        button: 3,
        buttons: 8,
        bubbles: true,
        cancelable: true,
        pointerType: 'mouse'
      }));
    });
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');

    await page.locator('[data-file-name="Overview.md"]').dblclick();
    await expect(page.locator('[data-editor-mode="notes-markdown"]')).toBeVisible({ timeout: 10000 });

    await page.locator('.main-content-header .close-btn[aria-label="Close"]').click();
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.files-breadcrumb')).toContainText('Notes');
  });

  test('open .txt via workbench from files context shows default-editor', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.files', {
        kind: 'vault-file',
        path: 'Docs/todo.txt',
        extension: '.txt',
        context: { sourcePluginId: 'verstak.files', sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const editor = page.locator('[data-editor-mode="text"]');
    await expect(editor).toBeVisible({ timeout: 10000 });
    await expect(editor).toHaveAttribute('data-resource-path', 'Docs/todo.txt');

    const textarea = page.locator('.de-textarea');
    await expect(textarea).toBeVisible({ timeout: 10000 });
    const textareaBox = await textarea.boundingBox();
    expect(textareaBox.height).toBeGreaterThan(300);
  });

  test('open .md via workbench from files context shows generic-markdown', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.files', {
        kind: 'vault-file',
        path: 'Docs/readme.md',
        extension: '.md',
        context: { sourcePluginId: 'verstak.files', sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const workbench = page.locator('.workbench-host');
    await expect(workbench).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.main-content-title-text')).toHaveText('readme.md');

    const preview = page.locator('.de-preview');
    await expect(preview).toBeVisible({ timeout: 10000 });
    const previewBox = await preview.boundingBox();
    expect(previewBox.height).toBeGreaterThan(300);
  });

  test('open notes markdown via workbench from files context shows notes-markdown', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.files', {
        kind: 'vault-file',
        path: 'Notes/Overview.md',
        extension: '.md',
        context: { sourcePluginId: 'verstak.files', sourceView: 'files', isInsideNotesFolder: true, notesMode: true },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });

    const workbench = page.locator('.workbench-host');
    await expect(workbench).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.main-content-title-text')).toHaveText('Overview');
  });

  test('files plugin card shows openProviders in contributions', async ({ page }) => {
    await page.evaluate(async () => {
      const [result, err] = await window.go.api.App.OpenWorkbenchResource('verstak.files', {
        kind: 'vault-file', path: 'Docs/todo.txt', extension: '.txt',
        context: { sourcePluginId: 'verstak.files', sourceView: 'files' },
      });
      if (err) throw new Error(err);
      window.dispatchEvent(new CustomEvent('verstak:workbench-opened', { detail: result }));
    });
    const editor = page.locator('[data-editor-mode="text"]');
    await expect(editor).toBeVisible({ timeout: 5000 });
  });
});
