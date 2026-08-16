from pathlib import Path

path = Path('frontend/src/lib/shell/GlobalSearch.svelte')
text = path.read_text()

old = """  import { onMount, onDestroy } from 'svelte';
"""
new = """  import { onMount, onDestroy, tick } from 'svelte';
"""
if text.count(old) != 1:
    raise SystemExit(f'svelte import anchor count: {text.count(old)}')
text = text.replace(old, new, 1)

old = """      window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
        detail: { workspaceName: workspaceRootPath, workspaceRootPath }
      }));
      window.dispatchEvent(new CustomEvent('verstak:workspace-open-tool', {
        detail: { workspaceItemId: action.workspaceItemId, toolRequest: action.toolRequest || null }
      }));
"""
new = """      window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
        detail: { workspaceName: workspaceRootPath, workspaceRootPath }
      }));
      // The target WorkspaceHost must receive the selected Deal and run its
      // workspace-change reset before the tool request is delivered. Without
      // this flush, a cross-Deal workspace-item action can select the right tab
      // but lose its toolRequest while Svelte applies the new workspace props.
      await tick();
      window.dispatchEvent(new CustomEvent('verstak:workspace-open-tool', {
        detail: { workspaceItemId: action.workspaceItemId, toolRequest: action.toolRequest || null }
      }));
"""
if text.count(old) != 1:
    raise SystemExit(f'workspace-item dispatch anchor count: {text.count(old)}')
path.write_text(text.replace(old, new, 1))

path = Path('frontend/tests/shell-source-contract-test.mjs')
text = path.read_text()
marker = """assertIncludes(
  globalSearch,
  'call.then(onBatch)',
  'GlobalSearch should publish completed provider batches without waiting for every provider/layout variant',
);
"""
addition = marker + """assertIncludes(
  globalSearch,
  'await tick();',
  'GlobalSearch should let the target Deal settle before delivering a workspace-item tool request',
);
"""
if text.count(marker) != 1:
    raise SystemExit(f'workspace navigation source assertion anchor count: {text.count(marker)}')
path.write_text(text.replace(marker, addition, 1))
