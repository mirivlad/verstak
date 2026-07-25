import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector } from './helpers.js';

const IDS = {
  folder: '11111111-1111-1111-1111-111111111111',
  child: '22222222-2222-2222-2222-222222222222',
  middle: '44444444-4444-4444-4444-444444444444',
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

// Three roots, so that "before"/"after" placements have somewhere to land:
// with only two siblings almost every relative drop resolves to the position
// the node already holds, which the shell now refuses to send.
function basicTree() {
  return {
    roots: [
      node('folder', IDS.folder, 'Clients', [
        node('workspace', IDS.child, 'Nested Deal'),
      ]),
      node('workspace', IDS.middle, 'Middle Deal'),
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

  test('a drop that does not change the placement never reaches the backend', async ({ page }) => {
    const nestedKey = `workspace:${IDS.child}`;
    const folderKey = `folder:${IDS.folder}`;
    const looseKey = `workspace:${IDS.deal}`;
    const middleKey = `workspace:${IDS.middle}`;

    // Dropping a Deal into the folder that already holds it.
    await page.locator(`[data-tree-key="${folderKey}"]`).click();
    await expect(page.locator(`[data-tree-key="${nestedKey}"]`)).toBeVisible();
    await startTreeDrag(page, nestedKey);
    await dispatchDragAt(page, page.locator(`[data-tree-key="${folderKey}"]`), 'dragover', 0.5);
    await dispatchDragAt(page, page.locator(`[data-tree-key="${folderKey}"]`), 'drop', 0.5);

    // Dropping a root Deal onto the free root area it already sits in.
    const rootArea = page.locator('[data-tree-drop-root]');
    await startTreeDrag(page, looseKey);
    await dispatchDragAt(page, rootArea, 'dragover');
    await dispatchDragAt(page, rootArea, 'drop');

    // Dropping a Deal directly after the sibling it already follows.
    await startTreeDrag(page, looseKey);
    await dispatchDragAt(page, page.locator(`[data-tree-key="${middleKey}"]`), 'dragover', 0.9);
    await dispatchDragAt(page, page.locator(`[data-tree-key="${middleKey}"]`), 'drop', 0.9);

    await expect(page.locator(`[data-tree-key="${folderKey}"]`)).toBeVisible();
    await expect(page.locator('[data-workspace-tree-notice]')).toHaveCount(0);
    expect(await requests(page)).toEqual([]);
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

  test('free root area sends root and backend rejection keeps the tree usable', async ({ page }) => {
    // A nested Deal: moving it to the root actually changes its placement.
    const sourceKey = `workspace:${IDS.child}`;
    const rootArea = page.locator('[data-tree-drop-root]');
    await page.locator(`[data-tree-key="folder:${IDS.folder}"]`).click();
    await expect(page.locator(`[data-tree-key="${sourceKey}"]`)).toBeVisible();
    await page.evaluate(() => window.__wailsMock.setTreePlacementError('placement rejected'));

    await startTreeDrag(page, sourceKey);
    await dispatchDragAt(page, rootArea, 'dragover');
    await expect(rootArea).toHaveAttribute('data-drop-active', 'root');
    await dispatchDragAt(page, rootArea, 'drop');

    await expect.poll(() => requests(page)).toEqual([
      { sourceKey, targetKey: '', position: 'root' },
    ]);

    // The rejection is a dismissible notice, never a replacement for the tree,
    // and it must not leak the backend string to the user.
    const notice = page.locator('[data-workspace-tree-notice]');
    await expect(notice).toBeVisible();
    await expect(notice).not.toContainText('placement rejected');
    await expect(page.locator(`[data-tree-key="folder:${IDS.folder}"]`)).toBeVisible();
    await expect(page.locator('[data-workspace-tree-load-error]')).toHaveCount(0);
    await expect(page.locator('[data-drop-active]')).toHaveCount(0);
    await expect(page.locator('[data-drop-position]')).toHaveCount(0);

    await notice.getByRole('button').click();
    await expect(notice).toHaveCount(0);
  });

  test('hover expansion is temporary and does not survive the drag', async ({ page }) => {
    const folderKey = `folder:${IDS.folder}`;
    const looseKey = `workspace:${IDS.deal}`;
    const folder = page.locator(`[data-tree-key="${folderKey}"]`);

    // Collapsed to begin with.
    await expect(folder).toHaveAttribute('aria-expanded', 'false');

    // Hover over the folder long enough for it to open, then drop elsewhere.
    await startTreeDrag(page, looseKey);
    await dispatchDragAt(page, folder, 'dragover', 0.5);
    await expect(folder).toHaveAttribute('aria-expanded', 'true');

    const middle = page.locator(`[data-tree-key="workspace:${IDS.middle}"]`);
    await dispatchDragAt(page, middle, 'dragover', 0.05);
    await dispatchDragAt(page, middle, 'drop', 0.05);

    // The look must not become a decision.
    await expect(folder).toHaveAttribute('aria-expanded', 'false');
  });

  test('a folder the item is dropped into stays open', async ({ page }) => {
    const folderKey = `folder:${IDS.folder}`;
    const looseKey = `workspace:${IDS.deal}`;
    const folder = page.locator(`[data-tree-key="${folderKey}"]`);

    await startTreeDrag(page, looseKey);
    await dispatchDragAt(page, folder, 'dragover', 0.5);
    await expect(folder).toHaveAttribute('aria-expanded', 'true');
    await dispatchDragAt(page, folder, 'drop', 0.5);

    await expect(folder).toHaveAttribute('aria-expanded', 'true');
    await expect.poll(() => requests(page)).toEqual([
      { sourceKey: looseKey, targetKey: folderKey, position: 'inside' },
    ]);
  });

  test('a folder stays open when the drop lands next to one of its children', async ({ page }) => {
    // The common gesture: hover a collapsed folder until it opens, then release
    // over the children that just appeared. The node lands in that folder, so
    // the folder must stay open — dropping "inside" the row is not the only
    // way to put something in a folder.
    const folderKey = `folder:${IDS.folder}`;
    const looseKey = `workspace:${IDS.deal}`;
    const folder = page.locator(`[data-tree-key="${folderKey}"]`);

    await startTreeDrag(page, looseKey);
    await dispatchDragAt(page, folder, 'dragover', 0.5);
    await expect(folder).toHaveAttribute('aria-expanded', 'true');

    // Release over the child Deal, not over the folder row.
    const child = page.locator(`[data-tree-key="workspace:${IDS.child}"]`);
    await dispatchDragAt(page, child, 'dragover', 0.8);
    await dispatchDragAt(page, child, 'drop', 0.8);

    await expect.poll(() => requests(page)).toEqual([
      { sourceKey: looseKey, targetKey: `workspace:${IDS.child}`, position: 'after' },
    ]);
    await expect(folder).toHaveAttribute('aria-expanded', 'true');
  });

  test('the folder drop zone covers the row except a thin reorder strip', async ({ page }) => {
    const folder = page.locator(`[data-tree-key="folder:${IDS.folder}"]`);
    const box = await folder.boundingBox();
    const rowHeight = box.height;

    // A few pixels in from either edge must already mean "inside", otherwise
    // holding a drag steady inside a folder is a test of hand stability.
    for (const fraction of [0.3, 0.5, 0.7]) {
      await startTreeDrag(page, `workspace:${IDS.deal}`);
      await dispatchDragAt(page, folder, 'dragover', fraction);
      await expect(folder).toHaveAttribute('data-drop-position', 'inside');
      await page.keyboard.press('Escape');
    }
    expect(rowHeight).toBeGreaterThan(20);
  });

  test('a collapsed folder is not reopened by the next tree reload', async ({ page }) => {
    const folderKey = `folder:${IDS.folder}`;
    const folder = page.locator(`[data-tree-key="${folderKey}"]`);

    // Open it, then close it again — the state a user leaves behind constantly.
    await folder.click();
    await expect(folder).toHaveAttribute('aria-expanded', 'true');
    await folder.click();
    await expect(folder).toHaveAttribute('aria-expanded', 'false');

    // What gets persisted is the whole contract here: a folder that is closed
    // must not appear in the list of open folders. It used to, because the map
    // kept the key with a false value and only the key was checked.
    await expect.poll(async () => page.evaluate(
      async () => (await window.go.api.App.GetAppSettings()).expandedFolderIds || [],
    )).not.toContain(IDS.folder);

    // Any reload re-reads that list. A drop is only the most visible trigger;
    // an external file change or a restart does the same.
    const looseKey = `workspace:${IDS.deal}`;
    const middle = page.locator(`[data-tree-key="workspace:${IDS.middle}"]`);
    await startTreeDrag(page, looseKey);
    await dispatchDragAt(page, middle, 'dragover', 0.05);
    await dispatchDragAt(page, middle, 'drop', 0.05);

    await expect.poll(() => requests(page)).toHaveLength(1);
    await expect(folder).toHaveAttribute('aria-expanded', 'false');
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
