import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector } from './helpers.js';

const IDS = {
  folder: '11111111-1111-1111-1111-111111111111',
  child: '22222222-2222-2222-2222-222222222222',
  deal: '33333333-3333-3333-3333-333333333333',
};

function node(kind, id, name, children = []) {
  return {
    key: `${kind}:${id}`,
    kind,
    id,
    name,
    path: name,
    children,
  };
}

function basicTree() {
  return {
    roots: [
      node('folder', IDS.folder, 'Clients', [
        node('workspace', IDS.child, 'Nested Deal'),
      ]),
      node('workspace', IDS.deal, 'Loose Deal'),
    ],
    currentWorkspaceId: '',
    revision: 1,
    warnings: [],
  };
}

async function installTree(page, tree) {
  await page.evaluate((snapshot) => {
    window.__wailsMock.setWorkspaceTreeV2(snapshot);
    window.dispatchEvent(new CustomEvent('verstak:workspace-tree-changed'));
  }, tree);
  await expect(page.locator(`[data-tree-key="${tree.roots[0].key}"]`)).toBeVisible();
}

async function startTreeDrag(page, sourceKey) {
  await page.locator(`[data-tree-key="${sourceKey}"]`).evaluate((element) => {
    const transfer = new DataTransfer();
    element.dispatchEvent(new DragEvent('dragstart', {
      bubbles: true,
      cancelable: true,
      dataTransfer: transfer,
    }));
    window.__workspaceTreeDragTransfer = transfer;
  });
}

async function dispatchDragAt(page, locator, type, fractionY = 0.5) {
  const box = await locator.boundingBox();
  if (!box) throw new Error('drag target has no bounding box');
  await locator.evaluate((element, args) => {
    element.dispatchEvent(new DragEvent(args.type, {
      bubbles: true,
      cancelable: true,
      clientX: args.x,
      clientY: args.y,
      dataTransfer: window.__workspaceTreeDragTransfer,
    }));
  }, {
    type,
    x: box.x + box.width / 2,
    y: box.y + box.height * fractionY,
  });
}

async function requests(page) {
  return page.evaluate(() => window.__wailsMock.getTreePlacementRequests());
}

test.describe('Workspace tree precision drag and drop', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await page.goto('/');
    await waitForAppReady(page);
    await installTree(page, basicTree());
  });

  test.afterEach(async () => {
    consoleCollector.assertNoErrors();
  });

  test('row thirds send stable-key before, inside, and after placements', async ({ page }) => {
    const sourceKey = `workspace:${IDS.deal}`;
    const targetKey = `folder:${IDS.folder}`;
    const target = page.locator(`[data-tree-key="${targetKey}"]`);

    for (const [fraction, position] of [[0.1, 'before'], [0.5, 'inside'], [0.9, 'after']]) {
      await startTreeDrag(page, sourceKey);
      await dispatchDragAt(page, target, 'dragover', fraction);
      await expect(target).toHaveAttribute('data-drop-position', position);
      await dispatchDragAt(page, target, 'drop', fraction);
    }

    await expect.poll(() => requests(page)).toEqual([
      { sourceKey, targetKey, position: 'before' },
      { sourceKey, targetKey, position: 'inside' },
      { sourceKey, targetKey, position: 'after' },
    ]);
    await expect(page.locator('[data-drop-position]')).toHaveCount(0);
  });

  test('Deal middle area resolves to a sibling placement without index identity', async ({ page }) => {
    const sourceKey = `folder:${IDS.folder}`;
    const targetKey = `workspace:${IDS.deal}`;
    const target = page.locator(`[data-tree-key="${targetKey}"]`);

    await startTreeDrag(page, sourceKey);
    await dispatchDragAt(page, target, 'dragover', 0.49);
    await dispatchDragAt(page, target, 'drop', 0.49);

    await expect.poll(() => requests(page)).toEqual([
      { sourceKey, targetKey, position: 'before' },
    ]);
  });

  test('hover expansion uses one stable-key timer and exposes a free child-list target', async ({ page }) => {
    const sourceKey = `workspace:${IDS.deal}`;
    const targetKey = `folder:${IDS.folder}`;
    const target = page.locator(`[data-tree-key="${targetKey}"]`);
    const childArea = page.locator(`[data-tree-drop-children="${targetKey}"]`);
    await expect(childArea).toHaveCount(0);

    await startTreeDrag(page, sourceKey);
    await dispatchDragAt(page, target, 'dragover', 0.5);
    await page.waitForTimeout(760);
    await expect(childArea).toBeVisible();
    await dispatchDragAt(page, childArea, 'dragover');
    await expect(childArea).toHaveAttribute('data-drop-active', 'inside');
    await dispatchDragAt(page, childArea, 'drop');

    await expect.poll(() => requests(page)).toEqual([
      { sourceKey, targetKey, position: 'inside' },
    ]);
  });

  test('free root area sends root and backend rejection clears every drag artifact', async ({ page }) => {
    const sourceKey = `workspace:${IDS.deal}`;
    const rootArea = page.locator('[data-tree-drop-root]');
    await page.evaluate(() => window.__wailsMock.setTreePlacementError('placement rejected'));

    await startTreeDrag(page, sourceKey);
    await dispatchDragAt(page, rootArea, 'dragover');
    await expect(rootArea).toHaveAttribute('data-drop-active', 'root');
    await dispatchDragAt(page, rootArea, 'drop');

    await expect.poll(() => requests(page)).toEqual([
      { sourceKey, targetKey: '', position: 'root' },
    ]);
    await expect(page.locator('.wt-error')).toContainText('placement rejected');
    await expect(page.locator('[data-drop-active]')).toHaveCount(0);
    await expect(page.locator('[data-drop-position]')).toHaveCount(0);
  });

  test('edge drag autoscrolls and dragend stops and clears state', async ({ page }) => {
    const roots = [];
    for (let index = 0; index < 36; index += 1) {
      const id = `${String(index + 1).padStart(8, '0')}-aaaa-4aaa-8aaa-${String(index + 1).padStart(12, '0')}`;
      roots.push(node('workspace', id, `Deal ${String(index + 1).padStart(2, '0')}`));
    }
    await installTree(page, { roots, currentWorkspaceId: '', revision: 2, warnings: [] });
    const source = page.locator(`[data-tree-key="${roots[0].key}"]`);
    const list = page.locator('.wt-list');

    await startTreeDrag(page, roots[0].key);
    await dispatchDragAt(page, list, 'dragover', 0.99);
    await expect.poll(() => list.evaluate((element) => element.scrollTop)).toBeGreaterThan(0);
    await source.evaluate((element) => {
      element.dispatchEvent(new DragEvent('dragend', { bubbles: true, cancelable: true }));
    });
    const stoppedAt = await list.evaluate((element) => element.scrollTop);
    await page.waitForTimeout(150);
    expect(await list.evaluate((element) => element.scrollTop)).toBe(stoppedAt);
    await expect(page.locator('[data-drop-active]')).toHaveCount(0);
  });
});
