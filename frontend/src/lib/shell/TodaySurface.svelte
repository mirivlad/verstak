<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';

  export let workspaceRootPath = '';
  export let availableTools = [];

  const dispatch = createEventDispatcher();
  const FILTERS = [
    { key: 'all', label: 'All' },
    { key: 'notes', label: 'Notes' },
    { key: 'files', label: 'Files' },
    { key: 'captures', label: 'Captures' },
    { key: 'journal', label: 'Journal' },
  ];
  const LOW_VALUE_RECENT_TYPES = new Set([
    'workspace.selected',
    'case.selected',
    'file.selected',
    'file.opened',
    'note.opened',
  ]);
  const LOW_VALUE_RESUME_TYPES = new Set([
    'workspace.selected',
    'case.selected',
    'file.selected',
  ]);

  let loading = true;
  let activeFilter = 'all';
  let captures = [];
  let activityEvents = [];
  let journalEntries = [];
  let worklogSuggestions = [];
  let keyResources = [];
  let loadedWorkspaceRoot = '';
  let toolProbe = 0;

  $: hasNotes = hasTool('notes');
  $: hasBrowserInbox = hasTool('browser inbox') || hasTool('inbox');
  $: hasActivity = hasTool('activity');
  $: hasFiles = hasTool('files');
  $: recentChanges = buildRecentChanges(activityEvents, captures, journalEntries);
  $: filteredRecentChanges = activeFilter === 'all'
    ? recentChanges
    : recentChanges.filter(item => item.category === activeFilter);
  $: needsAttention = buildNeedsAttention(captures, worklogSuggestions);
  $: continueItems = buildContinueItems(activityEvents, captures);
  $: primaryContinue = continueItems[0] || null;
  $: lastActive = lastActiveDate([...recentChanges, ...continueItems], captures, journalEntries);
  $: summaryItems = [
    { key: 'notes', label: 'Notes', count: countCategory(recentChanges, 'notes'), detail: countLabel(countCategory(recentChanges, 'notes'), 'recent change') },
    { key: 'files', label: 'Files', count: countCategory(recentChanges, 'files'), detail: countLabel(countCategory(recentChanges, 'files'), 'recent change') },
    { key: 'captures', label: 'Captures', count: captures.length, detail: countLabel(captures.length, 'unprocessed capture') },
    { key: 'journal', label: 'Journal / Activity', count: countCategory(recentChanges, 'journal'), detail: countLabel(countCategory(recentChanges, 'journal'), 'entry or event') },
    { key: 'attention', label: 'Needs attention', count: needsAttention.length, detail: countLabel(needsAttention.length, 'pending item') },
  ];

  onMount(() => {
    toolProbe += 1;
  });

  $: if (workspaceRootPath && workspaceRootPath !== loadedWorkspaceRoot) {
    loadOverview();
  }

  function hasTool(name) {
    toolProbe;
    name = String(name || '').toLowerCase();
    const fromProps = (availableTools || []).some(tool => {
      const label = `${tool?.title || ''} ${tool?.id || ''} ${tool?.pluginId || ''}`.toLowerCase();
      return label.includes(name);
    });
    if (fromProps) return true;
    if (typeof document === 'undefined') return false;
    return Array.from(document.querySelectorAll('.workspace-tabs [role="tab"]')).some(tab => {
      return String(tab.textContent || '').trim().toLowerCase().includes(name);
    });
  }

  function decodeTuple(response, fallback) {
    if (Array.isArray(response) && response.length === 2) return response[1] ? fallback : (response[0] || fallback);
    return response || fallback;
  }

  async function readPluginSettings(pluginId) {
    try {
      return decodeTuple(await App.ReadPluginSettings(pluginId), {});
    } catch (_) {
      return {};
    }
  }

  function workspaceKey(prefix) {
    return prefix + encodeURIComponent(String(workspaceRootPath || '').trim());
  }

  function normalizeRows(value) {
    return Array.isArray(value) ? value.filter(item => item && typeof item === 'object') : [];
  }

  function rowsFor(settings, keys) {
    const workspace = String(workspaceRootPath || '').trim();
    return keys.flatMap(key => normalizeRows(settings?.[key])).filter(item => {
      const tagged = String(item.workspaceRootPath || item.workspaceName || item.workspaceNodeId || '').trim();
      return !tagged || !workspace || tagged === workspace;
    });
  }

  function timeValue(item) {
    return item?.capturedAt || item?.occurredAt || item?.receivedAt || item?.updatedAt || item?.modifiedAt || item?.date || item?.time || '';
  }

  function timeMs(item) {
    const value = timeValue(item);
    if (!value) return 0;
    const normalized = /^\d{4}-\d{2}-\d{2}$/.test(String(value)) ? `${value}T00:00:00` : value;
    const date = new Date(normalized);
    return Number.isNaN(date.getTime()) ? 0 : date.getTime();
  }

  function sortByTime(rows) {
    return [...rows].sort((a, b) => timeMs(b) - timeMs(a));
  }

  function fileName(path) {
    const value = String(path || '').split('/').filter(Boolean).pop() || '';
    return value || String(path || '').trim();
  }

  function titleFromPath(path) {
    return fileName(path).replace(/\.(md|markdown|txt)$/i, '').replace(/_/g, ' ') || 'Untitled';
  }

  function quoted(value) {
    const clean = String(value || '').trim();
    return clean ? `"${clean}"` : '';
  }

  function entityName(item) {
    const payload = item?.payload && typeof item.payload === 'object' ? item.payload : {};
    const path = payload.path || payload.notePath || item?.path || item?.relativePath || looksLikePath(item?.summary);
    const title = String(item?.title || payload.title || '').trim();
    if (path && (!title || /^(saved note|file opened|file changed|activity event)$/i.test(title))) return titleFromPath(path);
    return title || titleFromPath(path) || String(item?.summary || item?.activityId || 'item');
  }

  function looksLikePath(value) {
    const text = String(value || '').trim();
    if (!text) return '';
    return text.includes('/') || /\.[a-z0-9]{1,8}$/i.test(text) ? text : '';
  }

  function captureTitle(capture) {
    return capture?.title || capture?.fileName || capture?.url || capture?.captureId || 'Untitled capture';
  }

  function journalTitle(entry) {
    return entry?.title || entry?.summary || entry?.date || entry?.entryId || 'Journal entry';
  }

  function itemTimeLabel(item) {
    const value = timeValue(item);
    if (!value) return 'No timestamp';
    return relativeTime(value);
  }

  function absoluteTime(value) {
    const ms = timeMs({ occurredAt: value });
    if (!ms) return '';
    return new Date(ms).toLocaleString(undefined, {
      year: 'numeric',
      month: 'short',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  function relativeTime(value) {
    const ms = timeMs({ occurredAt: value });
    if (!ms) return 'No timestamp';
    const diff = Date.now() - ms;
    if (diff < 0) return absoluteTime(value);
    const minute = 60 * 1000;
    const hour = 60 * minute;
    const day = 24 * hour;
    if (diff < minute) return 'Just now';
    if (diff < hour) return `${Math.floor(diff / minute)} min ago`;
    if (diff < day) return `${Math.floor(diff / hour)}h ago`;
    if (diff < 2 * day) return 'Yesterday';
    if (diff < 7 * day) return `${Math.floor(diff / day)} days ago`;
    const date = new Date(ms);
    return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: date.getFullYear() === new Date().getFullYear() ? undefined : 'numeric' });
  }

  function activityCategory(item) {
    const type = String(item?.type || '').toLowerCase();
    if (type.startsWith('note.')) return 'notes';
    if (type.startsWith('file.')) return 'files';
    if (type.startsWith('browser.capture')) return 'captures';
    if (type.startsWith('journal.') || type.startsWith('worklog.') || type.startsWith('action.')) return 'journal';
    return 'activity';
  }

  function activityTitle(item) {
    const type = String(item?.type || '').toLowerCase();
    const name = entityName(item);
    if (type === 'note.saved' || type === 'note.edited') return `Edited note ${quoted(name)}`;
    if (type === 'note.opened') return `Opened note ${quoted(name)}`;
    if (type === 'note.created') return `Created note ${quoted(name)}`;
    if (type === 'file.opened') return `Opened file ${quoted(name)}`;
    if (type === 'file.changed') return `Changed file ${quoted(name)}`;
    if (type === 'file.created') return `Created file ${quoted(name)}`;
    if (type === 'file.deleted' || type === 'file.trashed') return `Removed file ${quoted(name)}`;
    if (type === 'browser.capture.page') return `Captured page ${quoted(name)}`;
    if (type === 'browser.capture.selection') return `Captured selection ${quoted(name)}`;
    if (type === 'browser.capture.link') return `Captured link ${quoted(name)}`;
    if (type === 'browser.capture.file') return `Captured file ${quoted(name)}`;
    if (type === 'browser.capture.converted') return `Converted capture ${quoted(name)}`;
    if (type === 'journal.entry.added' || type === 'worklog.entry.added') return `Added journal entry ${quoted(name)}`;
    if (type === 'action.started') return 'Work session detected';
    if (type === 'workspace.opened') return 'Workspace opened';
    return item?.title || item?.summary || 'Workspace activity';
  }

  function actionForCategory(category) {
    if (category === 'notes') return { kind: 'notes', label: 'Open Notes' };
    if (category === 'files') return { kind: 'files', label: 'Open Files' };
    if (category === 'captures') return { kind: 'browser-inbox', label: 'Review Inbox' };
    if (category === 'journal') return { kind: 'activity', label: 'View Activity' };
    return { kind: 'activity', label: 'View Activity' };
  }

  function buildRecentChanges(events, captureRows, journalRows) {
    const activityItems = sortByTime(events)
      .filter(item => !LOW_VALUE_RECENT_TYPES.has(String(item?.type || '').toLowerCase()))
      .map(item => {
        const category = activityCategory(item);
        const action = actionForCategory(category);
        return {
          id: item.activityId || `${item.type}:${timeValue(item)}`,
          category,
          title: activityTitle(item),
          meta: `${itemTimeLabel(item)}${item.sourcePluginId ? ' · ' + item.sourcePluginId.replace('verstak.', '') : ''}`,
          time: timeValue(item),
          absolute: absoluteTime(timeValue(item)),
          actionKind: action.kind,
          actionLabel: action.label,
        };
      });
    const captureItems = captureRows.map(item => ({
      id: item.captureId || `capture:${timeValue(item)}`,
      category: 'captures',
      title: `Captured ${item.kind || 'item'} ${quoted(captureTitle(item))}`,
      meta: `${itemTimeLabel(item)} · ${item.domain || item.url || 'Browser capture'}`,
      time: timeValue(item),
      absolute: absoluteTime(timeValue(item)),
      actionKind: 'browser-inbox',
      actionLabel: 'Review Inbox',
    }));
    const journalItems = journalRows.map(item => ({
      id: item.entryId || `journal:${timeValue(item)}`,
      category: 'journal',
      title: `Added journal entry ${quoted(journalTitle(item))}`,
      meta: `${itemTimeLabel(item)}${item.minutes ? ' · ' + item.minutes + ' min' : ''}`,
      time: timeValue(item),
      absolute: absoluteTime(timeValue(item)),
      actionKind: 'activity',
      actionLabel: 'View Activity',
    }));
    return sortByTime([...activityItems, ...captureItems, ...journalItems]).slice(0, 12);
  }

  function isResumeEvent(item) {
    const type = String(item?.type || '').toLowerCase();
    if (LOW_VALUE_RESUME_TYPES.has(type)) return false;
    return type === 'file.opened' ||
      type === 'note.opened' ||
      type === 'note.saved' ||
      type === 'note.edited' ||
      type === 'file.changed' ||
      type === 'file.created' ||
      type === 'browser.capture.converted';
  }

  function buildContinueItems(events, captureRows) {
    const fromEvents = sortByTime(events)
      .filter(isResumeEvent)
      .map(item => {
        const category = activityCategory(item);
        const action = actionForCategory(category);
        return {
          id: item.activityId || `${item.type}:${timeValue(item)}`,
          title: activityTitle(item),
          meta: itemTimeLabel(item),
          time: timeValue(item),
          absolute: absoluteTime(timeValue(item)),
          actionKind: action.kind,
          actionLabel: action.label,
        };
      });
    if (fromEvents.length) return fromEvents.slice(0, 4);
    return sortByTime(captureRows).slice(0, 3).map(item => ({
      id: item.captureId || `capture:${timeValue(item)}`,
      title: `Review capture ${quoted(captureTitle(item))}`,
      meta: `${itemTimeLabel(item)} · ${item.domain || item.kind || 'Browser capture'}`,
      time: timeValue(item),
      absolute: absoluteTime(timeValue(item)),
      actionKind: 'browser-inbox',
      actionLabel: 'Review Inbox',
    }));
  }

  function buildNeedsAttention(captureRows, suggestions) {
    const captureItems = sortByTime(captureRows).slice(0, 4).map(item => ({
      id: item.captureId || `capture:${timeValue(item)}`,
      title: captureTitle(item),
      meta: `${item.kind || 'capture'} · ${itemTimeLabel(item)}`,
      actionKind: 'browser-inbox',
      actionLabel: 'Review Inbox',
    }));
    const suggestionItems = sortByTime(suggestions).slice(0, 4).map(item => ({
      id: item.suggestionId || item.entryId || `suggestion:${timeValue(item)}`,
      title: item.title || item.summary || 'Pending worklog suggestion',
      meta: `${item.minutes ? item.minutes + ' min · ' : ''}${item.date || itemTimeLabel(item)}`,
      actionKind: 'activity',
      actionLabel: 'View Activity',
    }));
    return [...captureItems, ...suggestionItems].slice(0, 6);
  }

  function countCategory(items, category) {
    return items.filter(item => item.category === category).length;
  }

  function countLabel(count, singular) {
    return `${count} ${singular}${count === 1 ? '' : 's'}`;
  }

  function lastActiveDate(items, captureRows, journalRows) {
    const source = sortByTime([...items, ...captureRows, ...journalRows])[0];
    return timeValue(source);
  }

  async function listFiles(relativeDir) {
    if (!App.ListVaultFiles) return [];
    try {
      return decodeTuple(await App.ListVaultFiles('verstak.files', relativeDir), []);
    } catch (_) {
      return [];
    }
  }

  async function loadKeyResources() {
    const workspace = String(workspaceRootPath || '').trim();
    if (!workspace) return [];
    const [rootEntries, notesEntries] = await Promise.all([
      listFiles(workspace),
      listFiles(`${workspace}/Notes`),
    ]);
    const overview = [...notesEntries, ...rootEntries].find(item => /(^|\/)overview\.md$/i.test(String(item.relativePath || item.name || '')));
    if (!overview) return [];
    return [{
      id: overview.relativePath || overview.name,
      title: overview.name || fileName(overview.relativePath) || 'Overview.md',
      meta: overview.relativePath || 'Workspace overview note',
      actionKind: hasNotes ? 'notes' : 'files',
      actionLabel: hasNotes ? 'Open Notes' : 'Open Files',
    }];
  }

  async function loadOverview() {
    const workspaceAtStart = String(workspaceRootPath || '').trim();
    loadedWorkspaceRoot = workspaceAtStart;
    loading = true;
    const [browserSettings, activitySettings, journalSettings, resources] = await Promise.all([
      readPluginSettings('verstak.browser-inbox'),
      readPluginSettings('verstak.activity'),
      readPluginSettings('verstak.journal'),
      loadKeyResources(),
    ]);
    if (workspaceAtStart !== String(workspaceRootPath || '').trim()) return;

    captures = rowsFor(browserSettings, [
      workspaceKey('captures:workspace:'),
      'captures:global',
      'captures',
    ]);
    activityEvents = rowsFor(activitySettings, [
      workspaceKey('events:workspace:'),
      'events:global',
      'events',
    ]);
    journalEntries = rowsFor(journalSettings, [
      workspaceKey('worklog:workspace:'),
      'worklog',
    ]);
    worklogSuggestions = rowsFor(journalSettings, [
      workspaceKey('suggestions:workspace:'),
      'suggestions',
    ]);
    keyResources = resources;
    loading = false;
  }

  function openTool(kind) {
    dispatch('openTool', { kind });
  }
</script>

<div class="today-root overview-root" aria-label="Overview" data-overview-root>
  <div class="today-header overview-header">
    <div>
      <h2>Overview</h2>
      <p title={lastActive ? absoluteTime(lastActive) : ''}>
        {#if loading}
          Loading workspace context...
        {:else if lastActive}
          Last active {relativeTime(lastActive)}
        {:else}
          No recent workspace activity
        {/if}
      </p>
    </div>
    <button type="button" data-overview-action="refresh" on:click={loadOverview}>Refresh</button>
  </div>

  <div class="today-summary overview-summary" aria-label="Workspace overview summary">
    {#each summaryItems as item}
      <div class="today-summary-item overview-summary-item" data-overview-summary={item.key}>
        <strong>{loading ? '...' : item.count}</strong>
        <span>{item.label}</span>
        <small>{loading ? 'Loading...' : item.detail}</small>
      </div>
    {/each}
  </div>

  <div class="overview-layout">
    <main class="overview-main">
      <section class="today-resume overview-continue" data-overview-section="continue">
        <div class="today-resume-copy overview-continue-copy">
          <span>Continue working</span>
          {#if loading}
            <strong>Loading workspace signals...</strong>
            <p>Recent files, notes, captures, and journal entries will appear here.</p>
          {:else if primaryContinue}
            <strong title={primaryContinue.title}>{primaryContinue.title}</strong>
            <p title={primaryContinue.absolute}>{primaryContinue.meta}</p>
          {:else}
            <strong>No clear resume point yet</strong>
            <p>Open files or notes to create a useful return point for this workspace.</p>
          {/if}
        </div>
        {#if primaryContinue}
          <button type="button" data-overview-action="continue-primary" on:click={() => openTool(primaryContinue.actionKind)}>{primaryContinue.actionLabel}</button>
        {:else}
          <button type="button" data-overview-action="continue-primary" on:click={() => openTool('files')}>Open Files</button>
        {/if}
      </section>

      <section class="today-panel overview-panel overview-recent" data-overview-section="recent">
        <div class="today-panel-head overview-panel-head">
          <div>
            <h3>Recent changes</h3>
            <p>Latest meaningful activity in this workspace.</p>
          </div>
          <div class="overview-filters" aria-label="Recent changes filter">
            {#each FILTERS as filter}
              <button
                type="button"
                class:is-active={activeFilter === filter.key}
                aria-pressed={activeFilter === filter.key}
                data-overview-filter={filter.key}
                on:click={() => activeFilter = filter.key}
              >
                {filter.label}
              </button>
            {/each}
          </div>
        </div>
        {#if loading}
          <p class="today-empty">Loading recent changes...</p>
        {:else if filteredRecentChanges.length}
          <div class="today-list overview-list">
            {#each filteredRecentChanges as item}
              <div class="today-row overview-change-row" data-overview-recent-item={item.category}>
                <div>
                  <strong title={item.title}>{item.title}</strong>
                  <span title={item.absolute}>{item.meta}</span>
                </div>
                <button type="button" on:click={() => openTool(item.actionKind)}>{item.actionLabel}</button>
              </div>
            {/each}
          </div>
        {:else}
          <p class="today-empty">No meaningful changes for this filter yet.</p>
        {/if}
      </section>
    </main>

    <aside class="overview-side">
      {#if needsAttention.length || !loading}
        <section class="today-panel overview-panel" data-overview-section="attention">
          <div class="today-panel-head overview-panel-head">
            <div>
              <h3>Needs attention</h3>
              <p>Pending captures and worklog suggestions.</p>
            </div>
          </div>
          {#if loading}
            <p class="today-empty">Loading pending items...</p>
          {:else if needsAttention.length}
            <div class="today-list overview-list compact">
              {#each needsAttention as item}
                <div class="today-row overview-attention-row">
                  <strong title={item.title}>{item.title}</strong>
                  <span>{item.meta}</span>
                  <button type="button" on:click={() => openTool(item.actionKind)}>{item.actionLabel}</button>
                </div>
              {/each}
            </div>
          {:else}
            <p class="today-empty compact">No pending captures or worklog suggestions.</p>
          {/if}
        </section>
      {/if}

      <section class="today-panel overview-panel secondary" data-overview-section="quick-actions">
        <div class="today-panel-head overview-panel-head">
          <h3>Quick actions</h3>
        </div>
        <div class="today-actions overview-actions">
          {#if hasNotes}
            <button type="button" data-overview-action="notes" on:click={() => openTool('notes')}>Open Notes</button>
          {/if}
          <button type="button" data-overview-action="files" on:click={() => openTool('files')}>Open Files</button>
          <button type="button" data-overview-action="activity" on:click={() => openTool('activity')}>Review Activity</button>
          <button type="button" data-overview-action="browser-inbox" on:click={() => openTool('browser-inbox')}>Open Inbox</button>
        </div>
      </section>

      {#if keyResources.length}
        <section class="today-panel overview-panel secondary" data-overview-section="key-resources">
          <div class="today-panel-head overview-panel-head">
            <h3>Key resources</h3>
          </div>
          <div class="today-list overview-list compact">
            {#each keyResources as item}
              <div class="today-row overview-resource-row">
                <strong title={item.title}>{item.title}</strong>
                <span title={item.meta}>{item.meta}</span>
                <button type="button" on:click={() => openTool(item.actionKind)}>{item.actionLabel}</button>
              </div>
            {/each}
          </div>
        </section>
      {/if}
    </aside>
  </div>
</div>

<style>
  .today-root {
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--vt-color-background);
    color: var(--vt-color-text-primary);
    overflow: auto;
  }

  .today-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 1rem;
    border-bottom: 1px solid var(--vt-color-border);
    background: var(--vt-color-surface-muted);
  }

  .today-header h2 {
    margin: 0;
    font-size: 1.05rem;
  }

  .today-header p {
    margin: 0.25rem 0 0;
    color: var(--vt-color-text-muted);
    font-size: 0.8rem;
  }

  .today-summary {
    display: grid;
    grid-template-columns: repeat(5, minmax(0, 1fr));
    gap: 0.5rem;
    padding: 0.75rem 0.75rem 0;
  }

  .today-summary-item {
    min-width: 0;
    display: grid;
    gap: 0.16rem;
    padding: 0.65rem 0.75rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
  }

  .today-summary-item strong {
    color: var(--vt-color-text-primary);
    font-size: 1rem;
    line-height: 1;
  }

  .today-summary-item span,
  .today-summary-item small {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-muted);
    font-size: 0.74rem;
  }

  .today-summary-item span {
    color: var(--vt-color-text-secondary);
    font-weight: 600;
  }

  .overview-layout {
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(17rem, 23rem);
    gap: 0.75rem;
    padding: 0.75rem;
  }

  .overview-main,
  .overview-side {
    min-width: 0;
    display: grid;
    align-content: start;
    gap: 0.75rem;
  }

  .today-resume {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.9rem 1rem;
    border: 1px solid rgba(78, 204, 163, 0.24);
    border-radius: var(--vt-radius-lg);
    background: linear-gradient(135deg, rgba(78, 204, 163, 0.11), rgba(27, 36, 64, 0.6));
  }

  .today-resume-copy {
    min-width: 0;
    display: grid;
    gap: 0.22rem;
  }

  .today-resume-copy span {
    color: var(--vt-color-accent);
    font-size: 0.72rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .today-resume-copy strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-primary);
    font-size: 0.98rem;
  }

  .today-resume-copy p {
    min-width: 0;
    margin: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-secondary);
    font-size: 0.8rem;
  }

  .today-panel {
    min-width: 0;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-surface);
  }

  .overview-recent {
    min-height: 24rem;
  }

  .overview-panel.secondary {
    background: var(--vt-color-surface-muted);
  }

  .today-panel-head {
    min-height: 2.8rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.65rem 0.75rem;
    border-bottom: 1px solid var(--vt-color-border);
  }

  .today-panel h3 {
    margin: 0;
    color: var(--vt-color-text-primary);
    font-size: 0.9rem;
  }

  .today-panel-head p {
    margin: 0.2rem 0 0;
    color: var(--vt-color-text-muted);
    font-size: 0.74rem;
  }

  .overview-filters {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.2rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-background);
  }

  .overview-filters button,
  .today-panel-head button,
  .today-actions button,
  .today-header button,
  .today-resume button,
  .overview-list button {
    min-height: 1.85rem;
    padding: 0.3rem 0.65rem;
    border: 1px solid var(--vt-color-border-strong);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface-hover);
    color: var(--vt-color-text-secondary);
    font-size: 0.76rem;
    cursor: pointer;
  }

  .overview-filters button {
    min-height: 1.55rem;
    padding: 0.16rem 0.5rem;
    border-color: transparent;
    background: transparent;
  }

  .overview-filters button.is-active,
  .overview-filters button:hover,
  .today-panel-head button:hover,
  .today-actions button:hover,
  .today-header button:hover,
  .today-resume button:hover,
  .overview-list button:hover {
    border-color: var(--vt-color-accent);
    color: var(--vt-color-text-primary);
  }

  .overview-filters button.is-active {
    background: var(--vt-color-accent-muted);
    color: var(--vt-color-accent);
  }

  .today-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0;
    padding: 1rem;
    color: var(--vt-color-text-muted);
    font-size: 0.82rem;
    line-height: 1.45;
    text-align: center;
  }

  .today-empty.compact {
    min-height: 5rem;
  }

  .today-list {
    display: grid;
    gap: 0.45rem;
    padding: 0.65rem;
  }

  .today-list.compact {
    gap: 0.4rem;
  }

  .today-row {
    min-width: 0;
    display: grid;
    gap: 0.2rem;
    padding: 0.55rem;
    border: 1px solid var(--vt-color-border);
    border-radius: var(--vt-radius-md);
    background: var(--vt-color-surface-muted);
  }

  .overview-change-row {
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.75rem;
  }

  .overview-attention-row,
  .overview-resource-row {
    grid-template-columns: minmax(0, 1fr);
  }

  .today-row strong {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-primary);
    font-size: 0.85rem;
  }

  .today-row span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--vt-color-text-muted);
    font-size: 0.75rem;
  }

  .today-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    padding: 0.75rem;
  }

  @media (max-width: 980px) {
    .overview-layout,
    .today-summary {
      grid-template-columns: 1fr;
    }

    .today-panel-head {
      align-items: stretch;
      flex-direction: column;
    }

    .overview-filters {
      overflow-x: auto;
      justify-content: flex-start;
    }

    .today-resume {
      align-items: stretch;
      flex-direction: column;
    }

    .today-resume button {
      width: 100%;
    }

    .overview-change-row {
      grid-template-columns: 1fr;
      align-items: stretch;
    }
  }
</style>
