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
  let workSessionCandidates = [];
  let keyResources = [];
  let totalNotes = 0;
  let loadedWorkspaceRoot = '';
  let toolProbe = 0;

  $: hasNotes = hasTool('notes');
  $: recentChanges = buildRecentChanges(activityEvents, captures, journalEntries);
  $: filteredRecentChanges = activeFilter === 'all'
    ? recentChanges
    : recentChanges.filter(item => item.category === activeFilter);
  $: noteRecentChanges = countCategory(recentChanges, 'notes');
  $: fileRecentChanges = countCategory(recentChanges, 'files');
  $: unprocessedCaptures = captures.filter(item => item?.processed !== true);
  $: linkedCandidateIds = new Set(journalEntries.map(item => String(item?.sourceCandidateId || '')).filter(Boolean));
  $: pendingWorkSessionCandidates = workSessionCandidates.filter(item => !linkedCandidateIds.has(String(item?.candidateId || '')));
  $: needsAttention = buildNeedsAttention(unprocessedCaptures, pendingWorkSessionCandidates);
  $: continueItems = buildContinueItems(activityEvents, unprocessedCaptures, journalEntries);
  $: attentionActionKind = needsAttention[0]?.actionKind || 'browser-inbox';
  $: lastActive = lastActiveDate([...recentChanges, ...continueItems], captures, journalEntries);
  $: summaryItems = [
    { key: 'notes', label: 'Notes', count: totalNotes, detail: totalAndRecentLabel(totalNotes, noteRecentChanges), actionKind: 'notes', actionLabel: 'Open Notes' },
    { key: 'files', label: 'Files', count: fileRecentChanges, detail: countLabel(fileRecentChanges, 'recent change'), actionKind: 'files', actionLabel: 'Open Files' },
    { key: 'captures', label: 'Captures', count: unprocessedCaptures.length, detail: captureReviewLabel(unprocessedCaptures.length), actionKind: 'browser-inbox', actionLabel: 'Review Inbox' },
    { key: 'activity', label: 'Activity', count: activityEvents.length, detail: countLabel(activityEvents.length, 'recorded event'), actionKind: 'activity', actionLabel: 'View Activity' },
    { key: 'journal', label: 'Journal', count: journalEntries.length, detail: journalEntryLabel(journalEntries.length), actionKind: 'journal', actionLabel: 'Open Journal' },
    { key: 'attention', label: 'Needs attention', count: needsAttention.length, detail: countLabel(needsAttention.length, 'pending item'), actionKind: attentionActionKind, actionLabel: 'Review pending items' },
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
    return item?.capturedAt || item?.endedAt || item?.startedAt || item?.occurredAt || item?.receivedAt || item?.updatedAt || item?.modifiedAt || item?.date || item?.time || '';
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
    if (type.startsWith('journal.') || type.startsWith('worklog.')) return 'journal';
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
    if (category === 'journal') return { kind: 'journal', label: 'Open Journal' };
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
      actionKind: 'journal',
      actionLabel: 'Open Journal',
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

  function continueItemFromActivity(item) {
    const category = activityCategory(item);
    const action = actionForCategory(category);
    return {
      id: item.activityId || `${item.type}:${timeValue(item)}`,
      category,
      title: activityTitle(item),
      meta: itemTimeLabel(item),
      time: timeValue(item),
      absolute: absoluteTime(timeValue(item)),
      actionKind: action.kind,
      actionLabel: action.label,
    };
  }

  function buildContinueItems(events, captureRows, journalRows) {
    const captureCandidates = sortByTime(captureRows).map(item => ({
      id: item.captureId || `capture:${timeValue(item)}`,
      category: 'captures',
      title: `Review capture ${quoted(captureTitle(item))}`,
      meta: `${itemTimeLabel(item)} · ${item.domain || item.kind || 'Browser capture'}`,
      time: timeValue(item),
      absolute: absoluteTime(timeValue(item)),
      actionKind: 'browser-inbox',
      actionLabel: 'Review Inbox',
    }));
    const noteCandidates = sortByTime(events)
      .filter(item => isResumeEvent(item) && activityCategory(item) === 'notes')
      .map(continueItemFromActivity);
    const fileCandidates = sortByTime(events)
      .filter(item => ['file.changed', 'file.created'].includes(String(item?.type || '').toLowerCase()))
      .map(continueItemFromActivity);
    const journalCandidates = sortByTime(journalRows).map(item => ({
      id: item.entryId || `journal:${timeValue(item)}`,
      category: 'journal',
      title: `Continue journal entry ${quoted(journalTitle(item))}`,
      meta: itemTimeLabel(item),
      time: timeValue(item),
      absolute: absoluteTime(timeValue(item)),
      actionKind: 'journal',
      actionLabel: 'Open Journal',
    }));
    return [...captureCandidates, ...noteCandidates, ...fileCandidates, ...journalCandidates].slice(0, 4);
  }

  function buildNeedsAttention(captureRows, candidates) {
    const captureItems = sortByTime(captureRows).slice(0, 4).map(item => ({
      id: item.captureId || `capture:${timeValue(item)}`,
      title: captureTitle(item),
      meta: `${item.kind || 'capture'} · ${itemTimeLabel(item)}`,
      actionKind: 'browser-inbox',
      actionLabel: 'Review Inbox',
    }));
    const candidateItems = sortByTime(candidates).slice(0, 4).map(item => ({
      id: item.candidateId || `work-session:${timeValue(item)}`,
      title: 'Possible journal entry',
      meta: `Workspace: ${item.workspaceRootPath || workspaceRootPath || 'Unknown'} · ${item.estimatedMinutes || 0} min · ${item.activityCount || (item.activityIds || []).length || 0} activities`,
      actionKind: 'journal',
      actionLabel: 'Review candidate',
      toolRequest: { type: 'work-session-candidate', candidate: item },
    }));
    return [...captureItems, ...candidateItems].slice(0, 6);
  }

  function countCategory(items, category) {
    return items.filter(item => item.category === category).length;
  }

  function countLabel(count, singular) {
    return `${count} ${singular}${count === 1 ? '' : 's'}`;
  }

  function totalAndRecentLabel(total, recent) {
    return `${total} total · ${countLabel(recent, 'recent change')}`;
  }

  function captureReviewLabel(count) {
    return `${count} capture${count === 1 ? '' : 's'} to review`;
  }

  function journalEntryLabel(count) {
    return `${count} journal entr${count === 1 ? 'y' : 'ies'}`;
  }

  function lastActiveDate(items, captureRows, journalRows) {
    const source = sortByTime([...items, ...captureRows, ...journalRows])[0];
    return timeValue(source);
  }

  async function listFiles(pluginId, relativeDir) {
    if (!App.ListVaultFiles) return [];
    try {
      return decodeTuple(await App.ListVaultFiles(pluginId, relativeDir), []);
    } catch (_) {
      return [];
    }
  }

  async function loadWorkspaceResources() {
    const workspace = String(workspaceRootPath || '').trim();
    if (!workspace) return { keyResources: [], totalNotes: 0 };
    const [rootEntries, notesEntries] = await Promise.all([
      listFiles('verstak.files', workspace),
      listFiles('verstak.notes', `${workspace}/Notes`),
    ]);
    const noteFiles = notesEntries.filter(item => item?.type === 'file');
    const overview = [...notesEntries, ...rootEntries].find(item => /(^|\/)overview\.md$/i.test(String(item.relativePath || item.name || '')));
    return {
      totalNotes: noteFiles.length,
      keyResources: overview ? [{
        id: overview.relativePath || overview.name,
        title: overview.name || fileName(overview.relativePath) || 'Overview.md',
        meta: overview.relativePath || 'Workspace overview note',
        actionKind: hasNotes ? 'notes' : 'files',
        actionLabel: hasNotes ? 'Open Notes' : 'Open Files',
      }] : [],
    };
  }

  async function loadOverview() {
    const workspaceAtStart = String(workspaceRootPath || '').trim();
    loadedWorkspaceRoot = workspaceAtStart;
    loading = true;
    const [browserSettings, activitySettings, journalSettings, resources] = await Promise.all([
      readPluginSettings('verstak.browser-inbox'),
      readPluginSettings('verstak.activity'),
      readPluginSettings('verstak.journal'),
      loadWorkspaceResources(),
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
    workSessionCandidates = rowsFor(activitySettings, [
      workspaceKey('work-session-candidates:workspace:'),
    ]);
    keyResources = resources.keyResources;
    totalNotes = resources.totalNotes;
    loading = false;
  }

  function openTool(kind, toolRequest = null) {
    dispatch('openTool', { kind, toolRequest });
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
      <button
        type="button"
        class="today-summary-item overview-summary-item"
        class:summary-attention={item.key === 'attention'}
        data-overview-summary={item.key}
        data-overview-action={item.actionKind}
        aria-label={`${item.label}: ${item.actionLabel}`}
        on:click={() => openTool(item.actionKind)}
      >
        <strong>{loading ? '...' : item.count}</strong>
        <span>{item.label}</span>
        <small>{loading ? 'Loading...' : item.detail}</small>
        <em>{item.actionLabel}</em>
      </button>
    {/each}
  </div>

  <div class="overview-layout">
    <main class="overview-main">
      <section class="today-resume overview-continue" data-overview-section="continue">
        <div class="today-resume-copy overview-continue-copy">
          <span>Continue working</span>
          <h3>Pick up the next useful item in this workspace.</h3>
        </div>
        {#if loading}
          <p class="today-empty compact">Loading workspace signals...</p>
        {:else if continueItems.length}
          <div class="overview-continue-list">
            {#each continueItems as item}
              <button
                type="button"
                class="overview-continue-item"
                data-overview-continue-item={item.category}
                data-overview-action={item.actionKind}
                on:click={() => openTool(item.actionKind)}
              >
                <span class="overview-continue-item-copy">
                  <strong title={item.title}>{item.title}</strong>
                  <span title={item.absolute}>{item.meta}</span>
                </span>
                <span class="overview-continue-item-action">{item.actionLabel}</span>
              </button>
            {/each}
          </div>
        {:else}
          <div class="overview-continue-empty">
            <strong>No clear resume point yet</strong>
            <p>Recent notes, files, captures, and journal entries will appear here.</p>
          </div>
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
              <button
                type="button"
                class="today-row overview-change-row"
                data-overview-recent-item={item.category}
                data-overview-action={item.actionKind}
                on:click={() => openTool(item.actionKind)}
              >
                <span class="overview-change-copy">
                  <strong title={item.title}>{item.title}</strong>
                  <span title={item.absolute}>{item.meta}</span>
                </span>
                <span class="overview-row-action">{item.actionLabel}</span>
              </button>
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
              <p>Pending captures and possible journal entries.</p>
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
                  <button type="button" on:click={() => openTool(item.actionKind, item.toolRequest)}>{item.actionLabel}</button>
                </div>
              {/each}
            </div>
          {:else}
            <p class="today-empty compact">No pending captures or possible journal entries.</p>
          {/if}
        </section>
      {/if}

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
    grid-template-columns: repeat(6, minmax(0, 1fr));
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
    color: inherit;
    cursor: pointer;
    font: inherit;
    text-align: left;
  }

  .today-summary-item:hover,
  .today-summary-item:focus-visible {
    border-color: var(--vt-color-accent);
    background: var(--vt-color-surface-hover);
    outline: none;
  }

  .today-summary-item strong {
    color: var(--vt-color-text-primary);
    font-size: 1rem;
    line-height: 1;
  }

  .today-summary-item span,
  .today-summary-item small,
  .today-summary-item em {
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

  .today-summary-item em {
    color: var(--vt-color-accent);
    font-size: 0.7rem;
    font-style: normal;
  }

  .today-summary-item.summary-attention {
    border-color: rgba(255, 200, 87, 0.5);
    background: var(--vt-color-warning-muted);
  }

  .today-summary-item.summary-attention em {
    color: var(--vt-color-warning);
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
    display: grid;
    gap: 0.75rem;
    padding: 0.9rem 1rem;
    border: 1px solid rgba(78, 204, 163, 0.24);
    border-radius: var(--vt-radius-lg);
    background: var(--vt-color-accent-muted);
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

  .today-resume-copy h3 {
    margin: 0;
    color: var(--vt-color-text-primary);
    font-size: 0.98rem;
    font-weight: 600;
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

  [data-overview-section='attention'] {
    border-color: rgba(255, 200, 87, 0.45);
    background: var(--vt-color-warning-muted);
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
  .today-header button,
  .overview-continue-item,
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
  .today-header button:hover,
  .overview-continue-item:hover,
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
    color: inherit;
    font: inherit;
    text-align: left;
  }

  .overview-change-row {
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    padding: 0.55rem 0.75rem;
    border: 0;
    border-bottom: 1px solid var(--vt-color-border);
    border-radius: 0;
    background: transparent;
    cursor: pointer;
  }

  .overview-change-row:hover,
  .overview-change-row:focus-visible {
    background: var(--vt-color-surface-hover);
    outline: none;
  }

  .overview-change-copy,
  .overview-continue-item-copy {
    min-width: 0;
    display: grid;
    gap: 0.2rem;
  }

  .overview-row-action,
  .overview-continue-item-action {
    color: var(--vt-color-text-muted);
    font-size: 0.72rem;
    white-space: nowrap;
  }

  .overview-recent .overview-list {
    gap: 0;
    padding: 0;
  }

  .overview-recent .overview-change-row:last-child {
    border-bottom: 0;
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

  .overview-continue-list {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.5rem;
  }

  .overview-continue-item {
    min-width: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: center;
    gap: 0.6rem;
    text-align: left;
  }

  .overview-continue-item strong,
  .overview-continue-item-copy span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .overview-continue-item strong {
    color: var(--vt-color-text-primary);
    font-size: 0.8rem;
  }

  .overview-continue-item-copy span {
    color: var(--vt-color-text-muted);
    font-size: 0.72rem;
  }

  .overview-continue-empty {
    display: grid;
    gap: 0.22rem;
    color: var(--vt-color-text-secondary);
  }

  .overview-continue-empty p {
    margin: 0;
    color: var(--vt-color-text-muted);
    font-size: 0.8rem;
  }

  @media (max-width: 1120px) {
    .today-summary {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (max-width: 980px) {
    .overview-layout {
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

    .overview-continue-list {
      grid-template-columns: 1fr;
    }

    .overview-continue-item {
      width: 100%;
    }

    .overview-change-row {
      grid-template-columns: 1fr;
      align-items: stretch;
    }
  }

  @media (max-width: 620px) {
    .today-summary {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    .overview-change-row {
      grid-template-columns: 1fr;
      gap: 0.3rem;
    }
  }
</style>
