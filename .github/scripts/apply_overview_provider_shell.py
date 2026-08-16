from pathlib import Path
import re


def replace_once(path, old, new):
    path = Path(path)
    text = path.read_text()
    assert old in text, f'pattern not found in {path}: {old[:100]!r}'
    assert text.count(old) == 1, f'pattern not unique in {path}: {old[:100]!r}'
    path.write_text(text.replace(old, new, 1))

# ---------------------------------------------------------------------------
# Go manifest model
# ---------------------------------------------------------------------------
p = Path('internal/core/plugin/plugin.go')
s = p.read_text()
s = s.replace(
    '\tWorklogProviders   []ContributionWorklogProvider  `json:"worklogProviders,omitempty"`\n',
    '\tWorklogProviders   []ContributionWorklogProvider  `json:"worklogProviders,omitempty"`\n\tOverviewProviders  []ContributionOverviewProvider `json:"overviewProviders,omitempty"`\n',
    1,
)
needle = '''type ContributionWorklogProvider struct {
\tID      string `json:"id"`
\tLabel   string `json:"label"`
\tHandler string `json:"handler"`
}
'''
assert needle in s
s = s.replace(needle, needle + '''
// ContributionOverviewProvider contributes normalized semantic signals to the
// Deal Overview. The provider owns its storage and business rules; the shell
// owns aggregation and presentation.
type ContributionOverviewProvider struct {
\tID      string `json:"id"`
\tLabel   string `json:"label"`
\tHandler string `json:"handler"`
}
''', 1)
validation_anchor = '''\tif m.Contributes != nil {
\t\tfor i, provider := range m.Contributes.OpenProviders {'''
assert validation_anchor in s
s = s.replace(validation_anchor, '''\tif m.Contributes != nil {
\t\tfor i, provider := range m.Contributes.OverviewProviders {
\t\t\tif provider.ID == "" {
\t\t\t\terrs.add("contributes.overviewProviders[%d].id is required", i)
\t\t\t}
\t\t\tif provider.Label == "" {
\t\t\t\terrs.add("contributes.overviewProviders[%d].label is required", i)
\t\t\t}
\t\t\tif provider.Handler == "" {
\t\t\t\terrs.add("contributes.overviewProviders[%d].handler is required", i)
\t\t\t}
\t\t}
\t\tfor i, provider := range m.Contributes.OpenProviders {''', 1)
p.write_text(s)

# ---------------------------------------------------------------------------
# Go contribution registry
# ---------------------------------------------------------------------------
p = Path('internal/core/contribution/registry.go')
s = p.read_text()
s = s.replace('\tworklogProviders  []ContributionWorklogProvider\n', '\tworklogProviders  []ContributionWorklogProvider\n\toverviewProviders []ContributionOverviewProvider\n', 1)
s = s.replace('\tPointWorklog         ContributionPointType = "worklogProviders"\n', '\tPointWorklog         ContributionPointType = "worklogProviders"\n\tPointOverview        ContributionPointType = "overviewProviders"\n', 1)
s = s.replace('''\tcase PointWorklog:
\t\tfor _, v := range r.worklogProviders {
\t\t\tresult = append(result, v)
\t\t}
''', '''\tcase PointWorklog:
\t\tfor _, v := range r.worklogProviders {
\t\t\tresult = append(result, v)
\t\t}
\tcase PointOverview:
\t\tfor _, v := range r.overviewProviders {
\t\t\tresult = append(result, v)
\t\t}
''', 1)
needle = '''type ContributionWorklogProvider struct {
\tPluginID string                             `json:"pluginId"`
\tItem     plugin.ContributionWorklogProvider `json:"item"`
}
'''
assert needle in s
s = s.replace(needle, needle + '''
type ContributionOverviewProvider struct {
\tPluginID string                              `json:"pluginId"`
\tItem     plugin.ContributionOverviewProvider `json:"item"`
}
''', 1)
s = s.replace('\tr.worklogProviders = removeWorklogProviders(r.worklogProviders, pluginID)\n', '\tr.worklogProviders = removeWorklogProviders(r.worklogProviders, pluginID)\n\tr.overviewProviders = removeOverviewProviders(r.overviewProviders, pluginID)\n', 2)
s = s.replace('''\tfor _, item := range c.WorklogProviders {
\t\tr.worklogProviders = append(r.worklogProviders, ContributionWorklogProvider{PluginID: pluginID, Item: item})
\t}
''', '''\tfor _, item := range c.WorklogProviders {
\t\tr.worklogProviders = append(r.worklogProviders, ContributionWorklogProvider{PluginID: pluginID, Item: item})
\t}
\tfor _, item := range c.OverviewProviders {
\t\tr.overviewProviders = append(r.overviewProviders, ContributionOverviewProvider{PluginID: pluginID, Item: item})
\t}
''', 1)
worklog_getter = '''func (r *Registry) WorklogProviders() []ContributionWorklogProvider {
\tr.mu.RLock()
\tdefer r.mu.RUnlock()
\tresult := make([]ContributionWorklogProvider, len(r.worklogProviders))
\tcopy(result, r.worklogProviders)
\tsort.Slice(result, func(i, j int) bool {
\t\tif result[i].PluginID != result[j].PluginID {
\t\t\treturn result[i].PluginID < result[j].PluginID
\t\t}
\t\treturn result[i].Item.ID < result[j].Item.ID
\t})
\treturn result
}
'''
assert worklog_getter in s
s = s.replace(worklog_getter, worklog_getter + '''
func (r *Registry) OverviewProviders() []ContributionOverviewProvider {
\tr.mu.RLock()
\tdefer r.mu.RUnlock()
\tresult := make([]ContributionOverviewProvider, len(r.overviewProviders))
\tcopy(result, r.overviewProviders)
\tsort.Slice(result, func(i, j int) bool {
\t\tif result[i].PluginID != result[j].PluginID {
\t\t\treturn result[i].PluginID < result[j].PluginID
\t\t}
\t\treturn result[i].Item.ID < result[j].Item.ID
\t})
\treturn result
}
''', 1)
helper = '''func removeWorklogProviders(items []ContributionWorklogProvider, pluginID string) []ContributionWorklogProvider {
\tvar result []ContributionWorklogProvider
\tfor _, item := range items {
\t\tif item.PluginID != pluginID {
\t\t\tresult = append(result, item)
\t\t}
\t}
\treturn result
}
'''
assert helper in s
s = s.replace(helper, helper + '''
func removeOverviewProviders(items []ContributionOverviewProvider, pluginID string) []ContributionOverviewProvider {
\tvar result []ContributionOverviewProvider
\tfor _, item := range items {
\t\tif item.PluginID != pluginID {
\t\t\tresult = append(result, item)
\t\t}
\t}
\treturn result
}
''', 1)
p.write_text(s)

# ---------------------------------------------------------------------------
# API flattening
# ---------------------------------------------------------------------------
p = Path('internal/api/app.go')
s = p.read_text()
needle = '''type FlatWorklogProvider struct {
\tPluginID string `json:"pluginId"`
\tID       string `json:"id"`
\tLabel    string `json:"label"`
\tHandler  string `json:"handler"`
}
'''
assert needle in s
s = s.replace(needle, needle + '''
// FlatOverviewProvider is a normalized Overview provider contribution.
type FlatOverviewProvider struct {
\tPluginID string `json:"pluginId"`
\tID       string `json:"id"`
\tLabel    string `json:"label"`
\tHandler  string `json:"handler"`
}
''', 1)
# ContributionSummary field.
s, count = re.subn(r'(\n\s*WorklogProviders\s+\[\]FlatWorklogProvider\s+`json:"worklogProviders"`)', r'\1\n\tOverviewProviders []FlatOverviewProvider `json:"overviewProviders"`', s, count=1)
assert count == 1, 'ContributionSummary WorklogProviders field not found'
s = s.replace('\tregWorklogProviders := r.WorklogProviders()\n', '\tregWorklogProviders := r.WorklogProviders()\n\tregOverviewProviders := r.OverviewProviders()\n', 1)
s = s.replace('\tworklogProviders := make([]FlatWorklogProvider, 0, len(regWorklogProviders))\n', '\tworklogProviders := make([]FlatWorklogProvider, 0, len(regWorklogProviders))\n\toverviewProviders := make([]FlatOverviewProvider, 0, len(regOverviewProviders))\n', 1)
# Append overview flattening immediately after the worklog loop.
pattern = re.compile(r'(\tfor _, p := range regWorklogProviders \{\n\t\tworklogProviders = append\(worklogProviders, FlatWorklogProvider\{.*?\n\t\}\)\n\t\}\n)', re.S)
m = pattern.search(s)
assert m, 'worklog flatten loop not found'
insert = m.group(1) + '''\tfor _, p := range regOverviewProviders {
\t\toverviewProviders = append(overviewProviders, FlatOverviewProvider{
\t\t\tPluginID: p.PluginID,
\t\t\tID:       p.Item.ID,
\t\t\tLabel:    p.Item.Label,
\t\t\tHandler:  p.Item.Handler,
\t\t})
\t}
'''
s = s[:m.start()] + insert + s[m.end():]
# Return field near WorklogProviders.
s, count = re.subn(r'(\n\s*WorklogProviders:\s+worklogProviders,)', r'\1\n\t\tOverviewProviders: overviewProviders,', s, count=1)
assert count == 1, 'ContributionSummary return WorklogProviders not found'
p.write_text(s)

# ---------------------------------------------------------------------------
# i18n contribution localization
# ---------------------------------------------------------------------------
replace_once(
    'frontend/src/lib/i18n/index.js',
    "  worklogProviders: 'label',\n",
    "  worklogProviders: 'label',\n  overviewProviders: 'label',\n",
)

# ---------------------------------------------------------------------------
# WorkspaceHost: scope providers to the current Deal and navigate by exact id.
# ---------------------------------------------------------------------------
p = Path('frontend/src/lib/shell/WorkspaceHost.svelte')
s = p.read_text()
s = s.replace('  let workspaceTools = [];\n', '  let workspaceTools = [];\n  let overviewProviders = [];\n', 1)
s = s.replace("  let requestedToolKind = '';\n", "  let requestedWorkspaceItemId = '';\n", 1)
s = s.replace('''  $: if (requestedToolKind && workspaceTools.length > 0) {
    const match = findWorkspaceTool(requestedToolKind);
    if (match) {
      requestedToolKind = '';
      activateWorkspaceTool(match, requestedToolRequest);
      requestedToolRequest = null;
    }
  }
''', '''  $: if (requestedWorkspaceItemId && workspaceTools.length > 0) {
    const match = findWorkspaceItem(requestedWorkspaceItemId);
    if (match) {
      requestedWorkspaceItemId = '';
      activateWorkspaceTool(match, requestedToolRequest);
      requestedToolRequest = null;
    }
  }
''', 1)
old = '''  function findWorkspaceTool(kind) {
    kind = String(kind || '').toLowerCase();
    return workspaceTools.find(tool => {
      const text = `${tool?.title || ''} ${tool?.id || ''} ${tool?.pluginId || ''}`.toLowerCase();
      if (kind === 'browser-inbox') return text.includes('browser') || text.includes('inbox');
      return text.includes(kind);
    });
  }

  function requestWorkspaceTool(kind, toolRequest = null) {
    const match = findWorkspaceTool(kind);
    if (match) {
      activateWorkspaceTool(match, toolRequest);
      return;
    }
    requestedToolKind = kind;
    requestedToolRequest = toolRequest;
  }

  function openWorkspaceTool(event) {
    requestWorkspaceTool(event?.detail?.kind, event?.detail?.toolRequest || null);
  }
'''
new = '''  function findWorkspaceItem(workspaceItemId) {
    const id = String(workspaceItemId || '').trim();
    if (!id) return null;
    return workspaceTools.find(tool => tool?.id === id) || null;
  }

  function requestWorkspaceItem(workspaceItemId, toolRequest = null) {
    const match = findWorkspaceItem(workspaceItemId);
    if (match) {
      activateWorkspaceTool(match, toolRequest);
      return;
    }
    requestedWorkspaceItemId = String(workspaceItemId || '').trim();
    requestedToolRequest = toolRequest;
  }

  function openWorkspaceTool(event) {
    requestWorkspaceItem(event?.detail?.workspaceItemId, event?.detail?.toolRequest || null);
  }
'''
assert old in s
s = s.replace(old, new, 1)
old = '''  function handleWorkspaceOpenTool(event) {
    requestWorkspaceTool(event?.detail?.kind, event?.detail?.toolRequest || null);
  }
'''
new = '''  function handleWorkspaceOpenTool(event) {
    const workspaceItemId = event?.detail?.workspaceItemId || event?.detail?.id || '';
    requestWorkspaceItem(workspaceItemId, event?.detail?.toolRequest || null);
  }
'''
assert old in s
s = s.replace(old, new, 1)
# Scope providers after workspace tools are known.
anchor = '''      workspaceTools = discoveredWorkspaceTools.length > 0
        ? filterWorkspaceTools(discoveredWorkspaceTools, activeWorkspace, activeWorkspaceName)
        : [];
'''
assert anchor in s
s = s.replace(anchor, anchor + '''      const workspacePluginIds = new Set(workspaceTools.map(tool => tool?.pluginId).filter(Boolean));
      overviewProviders = (localizedContributions.overviewProviders || []).filter(provider => (
        enabledIds.has(provider.pluginId) && workspacePluginIds.has(provider.pluginId)
      ));
''', 1)
s = s.replace('''          <TodaySurface
            {workspaceRootPath}
            availableTools={displayedTools}
            on:openTool={openWorkspaceTool}
          />''', '''          <TodaySurface
            {workspaceRootPath}
            availableTools={displayedTools}
            {overviewProviders}
            on:openTool={openWorkspaceTool}
          />''', 1)
p.write_text(s)

# ---------------------------------------------------------------------------
# Generic TodaySurface script. Markup/styles stay intact.
# ---------------------------------------------------------------------------
p = Path('frontend/src/lib/shell/TodaySurface.svelte')
s = p.read_text()
new_script = r'''<script>
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { executePluginCommand } from '../plugin-host/VerstakPluginAPI.js';
  import { i18n } from '../i18n/index.js';

  export let workspaceRootPath = '';
  export let availableTools = [];
  export let overviewProviders = [];

  const dispatch = createEventDispatcher();
  let loading = true;
  let activeFilter = 'all';
  let providerResults = [];
  let loadedWorkspaceRoot = '';
  let loadedProviderKey = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;

  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);
  $: toolById = new Map((availableTools || []).filter(tool => tool?.id && !tool?.shell).map(tool => [tool.id, tool]));
  $: providerKey = (overviewProviders || []).map(provider => `${provider?.pluginId || ''}:${provider?.id || ''}:${provider?.handler || ''}`).join('|');
  $: aggregated = aggregateProviderResults(providerResults, toolById);
  $: summaryItems = aggregated.summary;
  $: continueItems = aggregated.resume;
  $: needsAttention = aggregated.attention;
  $: recentChanges = aggregated.recent;
  $: keyResources = aggregated.resources;
  $: lastActive = aggregated.lastActiveAt;
  $: categoryFilters = collectCategoryFilters(recentChanges);
  $: FILTERS = [{ key: 'all', label: tr('overview.filter.all') }, ...categoryFilters];
  $: if (!FILTERS.some(filter => filter.key === activeFilter)) activeFilter = 'all';
  $: filteredRecentChanges = activeFilter === 'all'
    ? recentChanges
    : recentChanges.filter(item => item.category === activeFilter);
  $: hasAttentionTools = (overviewProviders || []).length > 0;
  $: hasOverviewSideContent = Boolean(keyResources.length || needsAttention.length || (loading && hasAttentionTools));

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => {
      const changed = locale !== nextLocale;
      locale = nextLocale;
      if (changed && workspaceRootPath) loadOverview();
    });
  });

  onDestroy(() => unsubscribeLocale?.());

  $: if (workspaceRootPath && (workspaceRootPath !== loadedWorkspaceRoot || providerKey !== loadedProviderKey)) {
    loadOverview();
  }

  function commandResult(value) {
    if (value && value.status === 'handled') return value.result || {};
    return value && value.result ? value.result : (value || {});
  }

  async function loadOverview() {
    const workspaceAtStart = String(workspaceRootPath || '').trim();
    const providerKeyAtStart = providerKey;
    loadedWorkspaceRoot = workspaceAtStart;
    loadedProviderKey = providerKeyAtStart;
    loading = true;

    const rows = await Promise.all((overviewProviders || []).map(async provider => {
      if (!provider?.pluginId || !provider?.handler) return null;
      try {
        const response = await executePluginCommand(provider.pluginId, provider.handler, {
          workspaceRootPath: workspaceAtStart,
        });
        return {
          pluginId: provider.pluginId,
          providerId: provider.id || provider.handler,
          result: commandResult(response),
        };
      } catch (error) {
        console.warn(`[Overview] provider ${provider.pluginId}/${provider.id || provider.handler} failed:`, error);
        return null;
      }
    }));

    if (workspaceAtStart !== String(workspaceRootPath || '').trim() || providerKeyAtStart !== providerKey) return;
    providerResults = rows.filter(Boolean);
    loading = false;
  }

  function actionAvailable(action, tools) {
    if (!action?.workspaceItemId) return false;
    return tools.has(action.workspaceItemId);
  }

  function timeValue(item) {
    const value = item?.occurredAt || item?.time || '';
    if (!value) return 0;
    const parsed = new Date(value).getTime();
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function explicitOrder(item) {
    return Number.isFinite(Number(item?.order)) ? Number(item.order) : null;
  }

  function sortByTime(items) {
    return [...items].sort((a, b) => timeValue(b) - timeValue(a) || (a._sequence || 0) - (b._sequence || 0));
  }

  function sortAttention(items) {
    return [...items].sort((a, b) => {
      const ao = explicitOrder(a);
      const bo = explicitOrder(b);
      if (ao !== null || bo !== null) return (ao ?? 10000) - (bo ?? 10000);
      return timeValue(b) - timeValue(a) || (a._sequence || 0) - (b._sequence || 0);
    });
  }

  function normalizeActionItem(item, providerId, sequence, tools) {
    if (!item || !actionAvailable(item.action, tools)) return null;
    const target = tools.get(item.action.workspaceItemId);
    const occurredAt = item.occurredAt || '';
    const metaParts = [];
    if (item.meta) metaParts.push(String(item.meta));
    if (occurredAt) metaParts.push(relativeTime(occurredAt));
    return {
      ...item,
      id: `${providerId}:${item.id || sequence}`,
      meta: metaParts.join(' · '),
      absolute: occurredAt ? absoluteTime(occurredAt) : '',
      actionKind: item.action.workspaceItemId,
      actionLabel: tr('overview.openTool', { tool: target?.title || item.action.workspaceItemId }),
      toolRequest: item.action.toolRequest || null,
      _sequence: sequence,
    };
  }

  function normalizeSummaryItem(item, providerId, sequence, tools) {
    if (!item || !actionAvailable(item.action, tools)) return null;
    const target = tools.get(item.action.workspaceItemId);
    return {
      ...item,
      key: item.id || `${providerId}:${sequence}`,
      count: Number.isFinite(Number(item.count)) ? Number(item.count) : 0,
      actionKind: item.action.workspaceItemId,
      actionLabel: tr('overview.openTool', { tool: target?.title || item.action.workspaceItemId }),
      _sequence: sequence,
    };
  }

  function aggregateProviderResults(rows, tools) {
    const summary = [];
    const resume = [];
    const attention = [];
    const recent = [];
    const resources = [];
    const lastActiveCandidates = [];
    let sequence = 0;

    (rows || []).forEach(row => {
      const result = row?.result && typeof row.result === 'object' ? row.result : {};
      const providerId = `${row?.pluginId || 'provider'}:${row?.providerId || ''}`;
      (result.summary || []).forEach(item => {
        const normalized = normalizeSummaryItem(item, providerId, sequence++, tools);
        if (normalized) summary.push(normalized);
      });
      (result.resume || []).forEach(item => {
        const normalized = normalizeActionItem(item, providerId, sequence++, tools);
        if (normalized) resume.push(normalized);
      });
      (result.attention || []).forEach(item => {
        const normalized = normalizeActionItem(item, providerId, sequence++, tools);
        if (normalized) attention.push(normalized);
      });
      (result.recent || []).forEach(item => {
        const normalized = normalizeActionItem(item, providerId, sequence++, tools);
        if (normalized) {
          normalized.category = String(item.categoryId || 'other');
          normalized.categoryLabel = String(item.categoryLabel || tools.get(item.action.workspaceItemId)?.title || normalized.category);
          recent.push(normalized);
        }
      });
      (result.resources || []).forEach(item => {
        const normalized = normalizeActionItem(item, providerId, sequence++, tools);
        if (normalized) resources.push(normalized);
      });
      if (result.lastActiveAt) lastActiveCandidates.push({ occurredAt: result.lastActiveAt });
    });

    summary.sort((a, b) => (explicitOrder(a) ?? 10000) - (explicitOrder(b) ?? 10000) || String(a.label || '').localeCompare(String(b.label || '')));
    const visibleItems = [...resume, ...attention, ...recent, ...resources];
    visibleItems.forEach(item => {
      if (item.occurredAt) lastActiveCandidates.push(item);
    });
    return {
      summary,
      resume: sortByTime(resume).slice(0, 4),
      attention: sortAttention(attention).slice(0, 6),
      recent: sortByTime(recent).slice(0, 12),
      resources: resources.sort((a, b) => (explicitOrder(a) ?? 10000) - (explicitOrder(b) ?? 10000) || (a._sequence || 0) - (b._sequence || 0)),
      lastActiveAt: sortByTime(lastActiveCandidates)[0]?.occurredAt || '',
    };
  }

  function collectCategoryFilters(items) {
    const categories = [];
    const seen = new Set();
    (items || []).forEach(item => {
      const key = String(item?.category || '').trim();
      if (!key || seen.has(key)) return;
      seen.add(key);
      categories.push({ key, label: item.categoryLabel || key });
    });
    return categories;
  }

  function relativeTime(value) {
    const ms = new Date(value).getTime();
    if (!Number.isFinite(ms)) return tr('overview.time.none');
    const delta = Date.now() - ms;
    const abs = Math.abs(delta);
    if (abs < 60 * 1000) return tr('overview.time.now');
    const minutes = Math.max(1, Math.round(abs / 60000));
    if (minutes < 60) return tr(delta >= 0 ? 'overview.time.minutesAgo' : 'overview.time.inMinutes', { count: minutes });
    const hours = Math.max(1, Math.round(abs / 3600000));
    if (hours < 48) return tr(delta >= 0 ? 'overview.time.hoursAgo' : 'overview.time.inHours', { count: hours });
    const days = Math.max(1, Math.round(abs / 86400000));
    return tr(delta >= 0 ? 'overview.time.daysAgo' : 'overview.time.inDays', { count: days });
  }

  function absoluteTime(value) {
    const date = new Date(value);
    if (!Number.isFinite(date.getTime())) return '';
    try {
      return new Intl.DateTimeFormat(locale === 'ru' ? 'ru-RU' : 'en-US', {
        dateStyle: 'medium',
        timeStyle: 'short',
      }).format(date);
    } catch (_) {
      return date.toISOString();
    }
  }

  function openTool(workspaceItemId, toolRequest = null) {
    dispatch('openTool', { workspaceItemId, toolRequest });
  }
</script>'''
s, count = re.subn(r'<script>.*?</script>', lambda _: new_script, s, count=1, flags=re.S)
assert count == 1, 'TodaySurface script block not found'
p.write_text(s)

# Generic action copy.
for path, needle, replacement in [
    ('frontend/src/lib/i18n/catalogs/en.js', "  'overview.openJournal': 'Open Journal',\n", "  'overview.openJournal': 'Open Journal',\n  'overview.openTool': 'Open {tool}',\n"),
    ('frontend/src/lib/i18n/catalogs/ru.js', "  'overview.openJournal': 'Открыть журнал',\n", "  'overview.openJournal': 'Открыть журнал',\n  'overview.openTool': 'Открыть: {tool}',\n"),
]:
    replace_once(path, needle, replacement)

# ---------------------------------------------------------------------------
# Wails mock: expose Overview contributions and run the real provider bundles.
# ---------------------------------------------------------------------------
p = Path('frontend/src/lib/test/wails-mock.js')
s = p.read_text()
s = s.replace("import defaultEditorSource from '../../../../../verstak-official-plugins/plugins/default-editor/frontend/src/index.js?raw';\n", "import defaultEditorSource from '../../../../../verstak-official-plugins/plugins/default-editor/frontend/src/index.js?raw';\nimport notesSource from '../../../../../verstak-official-plugins/plugins/notes/frontend/src/index.js?raw';\nimport browserInboxSource from '../../../../../verstak-official-plugins/plugins/browser-inbox/frontend/src/index.js?raw';\n", 1)
# Declaratively augment the five handwritten mock manifests without duplicating
# their full literals.
anchor = "  var russianPluginNames = {\n"
assert anchor in s
augment = r'''  var overviewProviderDefs = {
    'verstak.notes': { id: 'verstak.notes.overview', handler: 'verstak.notes.provideOverview', label: 'Notes' },
    'verstak.activity': { id: 'verstak.activity.overview', handler: 'verstak.activity.provideOverview', label: 'Activity' },
    'verstak.browser-inbox': { id: 'verstak.browser-inbox.overview', handler: 'verstak.browser-inbox.provideOverview', label: 'Browser' },
    'verstak.journal': { id: 'verstak.journal.overview', handler: 'verstak.journal.provideOverview', label: 'Journal' },
    'verstak.todo': { id: 'verstak.todo.overview', handler: 'verstak.todo.provideOverview', label: 'Todo' }
  };
  Object.keys(overviewProviderDefs).forEach(function (pluginId) {
    var state = pluginStates[pluginId];
    if (!state || !state.manifest) return;
    var manifest = state.manifest;
    var provider = overviewProviderDefs[pluginId];
    manifest.permissions = manifest.permissions || [];
    if (manifest.permissions.indexOf('commands.register') === -1) manifest.permissions.push('commands.register');
    manifest.contributes = manifest.contributes || {};
    manifest.contributes.commands = manifest.contributes.commands || [];
    if (!manifest.contributes.commands.some(function (item) { return item.id === provider.handler; })) {
      manifest.contributes.commands.push({ id: provider.handler, title: 'Provide Overview Signals', handler: provider.handler });
    }
    manifest.contributes.overviewProviders = [{ id: provider.id, label: provider.label, handler: provider.handler }];
  });

'''
s = s.replace(anchor, augment + anchor, 1)
s = s.replace('var views = [], commands = [], searchProviders = [], worklogProviders = [], sidebarItems = [], statusBarItems = [], settingsPanels = [], openProviders = [], workspaceItems = [];', 'var views = [], commands = [], searchProviders = [], worklogProviders = [], overviewProviders = [], sidebarItems = [], statusBarItems = [], settingsPanels = [], openProviders = [], workspaceItems = [];', 1)
s = s.replace("      if (c.worklogProviders) c.worklogProviders.forEach(function (wp) { worklogProviders.push(Object.assign({}, wp, { pluginId: id })); });\n", "      if (c.worklogProviders) c.worklogProviders.forEach(function (wp) { worklogProviders.push(Object.assign({}, wp, { pluginId: id })); });\n      if (c.overviewProviders) c.overviewProviders.forEach(function (op) { overviewProviders.push(Object.assign({}, op, { pluginId: id })); });\n", 1)
s = s.replace('return { views: views, commands: commands, searchProviders: searchProviders, worklogProviders: worklogProviders, sidebarItems: sidebarItems, statusBarItems: statusBarItems, settingsPanels: settingsPanels, openProviders: openProviders, workspaceItems: workspaceItems };', 'return { views: views, commands: commands, searchProviders: searchProviders, worklogProviders: worklogProviders, overviewProviders: overviewProviders, sidebarItems: sidebarItems, statusBarItems: statusBarItems, settingsPanels: settingsPanels, openProviders: openProviders, workspaceItems: workspaceItems };', 1)
s = s.replace("      if (pluginId === 'verstak.notes' && assetPath === 'frontend/dist/index.js') {\n        return Promise.resolve(simplePluginBundle('verstak.notes', 'NotesView', 'notes-root', 'Notes'));\n      }", "      if (pluginId === 'verstak.notes' && assetPath === 'frontend/dist/index.js') {\n        return Promise.resolve(notesSource);\n      }", 1)
s = s.replace("      if (pluginId === 'verstak.browser-inbox' && assetPath === 'frontend/dist/index.js') {\n        return Promise.resolve(browserInboxBundle());\n      }", "      if (pluginId === 'verstak.browser-inbox' && assetPath === 'frontend/dist/index.js') {\n        return Promise.resolve(browserInboxSource);\n      }", 1)
p.write_text(s)

# ---------------------------------------------------------------------------
# Tests / permanent architecture guards.
# ---------------------------------------------------------------------------
p = Path('frontend/tests/shell-source-contract-test.mjs')
s = p.read_text()
s = s.replace("const workspaceHost = read('frontend/src/lib/shell/WorkspaceHost.svelte');\n", "const workspaceHost = read('frontend/src/lib/shell/WorkspaceHost.svelte');\nconst overviewSurface = read('frontend/src/lib/shell/TodaySurface.svelte');\n", 1)
append_anchor = "console.log('shell source contract smoke passed');\n"
assert append_anchor in s
guards = r'''
for (const forbidden of [
  'ReadPluginSettings',
  'ReadPluginDataNDJSON',
  'ListVaultFiles',
  'verstak.browser-inbox',
  'verstak.activity',
  'verstak.journal',
  'verstak.todo',
  'captures:workspace:',
  'captures:global',
  'todos:global',
  'worklog:workspace:',
  'work-session-candidates:workspace:',
]) {
  assertExcludes(
    overviewSurface,
    forbidden,
    `Overview shell must consume provider semantics instead of plugin internals (${forbidden})`,
  );
}
assertIncludes(
  overviewSurface,
  'executePluginCommand(provider.pluginId, provider.handler',
  'Overview shell should consume declared Overview providers through the generic command runtime',
);
assertIncludes(
  workspaceHost,
  'findWorkspaceItem(workspaceItemId)',
  'WorkspaceHost should resolve Overview navigation by exact workspace item id',
);
assertExcludes(
  workspaceHost,
  "kind === 'browser-inbox'",
  'WorkspaceHost should not special-case Browser navigation',
);
assertExcludes(
  workspaceHost,
  'text.includes(kind)',
  'WorkspaceHost should not guess a workspace tool from arbitrary title/id substrings',
);

'''
s = s.replace(append_anchor, guards + append_anchor, 1)
p.write_text(s)

# Registry test covers the new contribution point as a real Go contract.
p = Path('internal/core/contribution/registry_test.go')
s = p.read_text()
s = s.replace('\t\tWorklogProviders:   []plugin.ContributionWorklogProvider{{ID: "wp1", Label: "WP1", Handler: "h"}},\n', '\t\tWorklogProviders:   []plugin.ContributionWorklogProvider{{ID: "wp1", Label: "WP1", Handler: "h"}},\n\t\tOverviewProviders:  []plugin.ContributionOverviewProvider{{ID: "ov1", Label: "OV1", Handler: "overview"}},\n', 1)
s = s.replace('\t\t{PointWorklog, 1},\n', '\t\t{PointWorklog, 1},\n\t\t{PointOverview, 1},\n', 1)
p.write_text(s)
