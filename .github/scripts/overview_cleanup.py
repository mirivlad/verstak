from pathlib import Path
import subprocess

ROOT = Path('.')


def read(path):
    return (ROOT / path).read_text()


def write(path, text):
    (ROOT / path).write_text(text)


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f'{label}: expected 1 match, got {count}: {old!r}')
    return text.replace(old, new, 1)


def replace_count(text, old, new, expected, label):
    count = text.count(old)
    if count != expected:
        raise SystemExit(f'{label}: expected {expected} matches, got {count}: {old!r}')
    return text.replace(old, new)

# Rename the shell component now that the Today -> Overview product migration is complete.
old_surface = 'frontend/src/lib/shell/TodaySurface.svelte'
new_surface = 'frontend/src/lib/shell/OverviewSurface.svelte'
if not (ROOT / old_surface).exists():
    raise SystemExit('TodaySurface.svelte is missing before cleanup')
if (ROOT / new_surface).exists():
    raise SystemExit('OverviewSurface.svelte already exists before cleanup')
subprocess.run(['git', 'mv', old_surface, new_surface], check=True)

surface = read(new_surface)

# OverviewActionTarget.toolRequest is part of the public SDK contract. Every
# Overview item type must preserve it instead of only Attention doing so.
surface = replace_count(
    surface,
    'on:click={() => openTool(item.actionKind)}',
    'on:click={() => openTool(item.actionKind, item.toolRequest)}',
    4,
    'Overview toolRequest click propagation',
)
surface = replace_once(
    surface,
    "      actionLabel: tr('overview.openTool', { tool: target?.title || item.action.workspaceItemId }),\n      _sequence: sequence,",
    "      actionLabel: tr('overview.openTool', { tool: target?.title || item.action.workspaceItemId }),\n      toolRequest: item.action.toolRequest || null,\n      _sequence: sequence,",
    'Overview summary toolRequest normalization',
)

# Remove the last internal Today-era CSS vocabulary. These are private classes;
# data-overview-* hooks stay untouched for tests and integrations.
class_map = {
    'today-root': 'overview-root',
    'today-header': 'overview-header',
    'today-resume-copy': 'overview-continue-copy',
    'today-resume': 'overview-continue',
    'today-summary-item': 'overview-summary-item',
    'today-summary': 'overview-summary',
    'today-panel-head': 'overview-panel-head',
    'today-panel': 'overview-panel',
    'today-empty': 'overview-empty',
    'today-list': 'overview-list',
    'today-row': 'overview-row',
}
for old, new in class_map.items():
    if old not in surface:
        raise SystemExit(f'Overview class cleanup: expected {old!r}')
    surface = surface.replace(old, new)

# The markup already carried overview-* aliases beside many Today-era classes.
# Collapse only the duplicate pairs created by the rename.
for token in [
    'overview-root', 'overview-header', 'overview-continue', 'overview-continue-copy',
    'overview-summary', 'overview-summary-item', 'overview-panel', 'overview-panel-head',
    'overview-list',
]:
    surface = surface.replace(f'{token} {token}', token)

if 'today-' in surface:
    raise SystemExit('OverviewSurface still contains Today-era class names')
if surface.count('openTool(item.actionKind, item.toolRequest)') != 5:
    raise SystemExit('OverviewSurface must preserve toolRequest in all five item click paths')
write(new_surface, surface)

# WorkspaceHost: remove the completed TODO, use the final component name, and
# repair indentation left by the previous exact-id navigation migration.
host_path = 'frontend/src/lib/shell/WorkspaceHost.svelte'
host = read(host_path)
host = replace_once(host, "  import TodaySurface from './TodaySurface.svelte';", "  import OverviewSurface from './OverviewSurface.svelte';", 'WorkspaceHost import')
host = replace_once(host, "  // TODO: Rename TodaySurface.svelte to OverviewSurface.svelte in a refactor-only follow-up.\n", '', 'WorkspaceHost completed TODO')
host = replace_once(host, "component: 'TodaySurface'", "component: 'OverviewSurface'", 'WorkspaceHost shell component metadata')
host = replace_once(host, '<TodaySurface', '<OverviewSurface', 'WorkspaceHost component mount')
old_block = """  function findWorkspaceItem(workspaceItemId) {
  const id = String(workspaceItemId || '').trim();
  if (!id) return null;
  return workspaceTools.find(tool => tool?.id === id) || null;
}

function requestWorkspaceItem(workspaceItemId, toolRequest = null) {
  requestedWorkspaceItemId = String(workspaceItemId || '').trim();
  requestedToolRequest = toolRequest;
  const match = findWorkspaceItem(requestedWorkspaceItemId);
  if (match) {
    requestedWorkspaceItemId = '';
    requestedToolRequest = null;
    selectTool(match, toolRequest);
  }
}

function openWorkspaceTool(event) {
    requestWorkspaceItem(event?.detail?.workspaceItemId, event?.detail?.toolRequest || null);
  }
"""
new_block = """  function findWorkspaceItem(workspaceItemId) {
    const id = String(workspaceItemId || '').trim();
    if (!id) return null;
    return workspaceTools.find(tool => tool?.id === id) || null;
  }

  function requestWorkspaceItem(workspaceItemId, toolRequest = null) {
    requestedWorkspaceItemId = String(workspaceItemId || '').trim();
    requestedToolRequest = toolRequest;
    const match = findWorkspaceItem(requestedWorkspaceItemId);
    if (match) {
      requestedWorkspaceItemId = '';
      requestedToolRequest = null;
      selectTool(match, toolRequest);
    }
  }

  function openWorkspaceTool(event) {
    requestWorkspaceItem(event?.detail?.workspaceItemId, event?.detail?.toolRequest || null);
  }
"""
host = replace_once(host, old_block, new_block, 'WorkspaceHost exact-id block formatting')
if 'TodaySurface' in host:
    raise SystemExit('WorkspaceHost still references TodaySurface')
write(host_path, host)

# Permanent architecture guard follows the final filename and contract.
test_path = 'frontend/tests/shell-source-contract-test.mjs'
test = read(test_path)
test = replace_count(test, 'frontend/src/lib/shell/TodaySurface.svelte', 'frontend/src/lib/shell/OverviewSurface.svelte', 2, 'source-contract Overview filename')
test = replace_once(
    test,
    "assertIncludes(\n  overviewSurface,\n  'executePluginCommand(provider.pluginId, provider.handler',\n  'Overview shell should consume declared Overview providers through the generic command runtime',\n);",
    "assertIncludes(\n  overviewSurface,\n  'executePluginCommand(provider.pluginId, provider.handler',\n  'Overview shell should consume declared Overview providers through the generic command runtime',\n);\nif ((overviewSurface.match(/openTool\\(item\\.actionKind, item\\.toolRequest\\)/g) || []).length !== 5) {\n  throw new Error('Overview shell should preserve provider toolRequest for summary, resume, attention, recent, and resources');\n}\nassertIncludes(\n  overviewSurface,\n  'toolRequest: item.action.toolRequest || null',\n  'Overview summary normalization should preserve provider toolRequest',\n);\nassertExcludes(overviewSurface, 'today-', 'Overview shell should not retain Today-era internal CSS names');\nassertExcludes(workspaceHost, 'TodaySurface', 'WorkspaceHost should use the final OverviewSurface component name');",
    'source-contract toolRequest guard',
)
write(test_path, test)

# Final repo-level assertions for the intended cleanup scope.
remaining = subprocess.run(
    ['git', 'grep', '-n', 'TodaySurface', '--', 'frontend/src', 'frontend/tests'],
    text=True,
    capture_output=True,
)
if remaining.returncode == 0 and remaining.stdout.strip():
    raise SystemExit('Unexpected TodaySurface references remain:\n' + remaining.stdout)

print('Overview cleanup patch applied')
