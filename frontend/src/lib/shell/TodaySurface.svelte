<script>
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import { i18n } from '../i18n/index.js';

  export let workspaceRootPath = '';
  export let availableTools = [];

  const dispatch = createEventDispatcher();
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
  let todos = [];
  let workSessionCandidates = [];
  let keyResources = [];
  let totalNotes = 0;
  let loadedWorkspaceRoot = '';
  let loadedToolKey = '';
  let toolProbe = 0;
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;

  $: tr = ((activeLocale) => (key, params, fallback) => {
    void activeLocale;
    return i18n.t(key, params, fallback);
  })(locale);
  function countText(key, count, params = {}) {
    return tr(`${key}.${count === 1 ? 'one' : 'many'}`, { count, ...params });
  }

  $: hasNotes = hasTool('notes', availableTools);
  $: hasFiles = hasTool('files', availableTools);
  $: hasBrowserInbox = hasTool('browser-inbox', availableTools);
  $: hasActivity = hasTool('activity', availableTools);
  $: hasJournal = hasTool('journal', availableTools);
  $: hasTodos = hasTool('todo', availableTools);
  $: overviewToolKey = [hasNotes, hasFiles, hasBrowserInbox, hasActivity, hasJournal, hasTodos].join('|');
  $: FILTERS = ['all']
    .concat(hasNotes ? ['notes'] : [])
    .concat(hasFiles ? ['files'] : [])
    .concat(hasBrowserInbox ? ['captures'] : [])
    .concat(hasJournal ? ['journal'] : [])
    .map((key) => ({ key, label: tr(`overview.filter.${key}`) }));
  $: if (!FILTERS.some(filter => filter.key === activeFilter)) activeFilter = 'all';
  $: recentChanges = filterAvailableItems(buildRecentChanges(activityEvents, captures, journalEntries), overviewToolKey);
  $: filteredRecentChanges = activeFilter === 'all'
    ? recentChanges
    : recentChanges.filter(item => item.category === activeFilter);
  $: noteRecentChanges = countCategory(recentChanges, 'notes');
  $: fileRecentChanges = countCategory(recentChanges, 'files');
  $: unprocessedCaptures = captures.filter(item => item?.processed !== true);
  $: linkedCandidateIds = new Set(journalEntries.map(item => String(item?.sourceCandidateId || '')).filter(Boolean));
  $: pendingWorkSessionCandidates = workSessionCandidates.filter(item => !linkedCandidateIds.has(String(item?.candidateId || '')));
  $: urgentTodos = hasTodos ? todos.filter(item => todoAttentionState(item)) : [];
  $: needsAttention = filterAvailableItems(buildNeedsAttention(unprocessedCaptures, pendingWorkSessionCandidates, urgentTodos), overviewToolKey);
  $: continueItems = filterAvailableItems(buildContinueItems(activityEvents, unprocessedCaptures, journalEntries, urgentTodos), overviewToolKey);
  $: hasAttentionTools = hasBrowserInbox || hasTodos || (hasActivity && hasJournal);
  $: attentionActionKind = needsAttention[0]?.actionKind || fallbackAttentionAction();
  $: lastActive = lastActiveDate([...recentChanges, ...continueItems], captures, journalEntries, todos);
  $: summaryItems = [
    hasNotes ? { key: 'notes', label: tr('overview.notes'), count: totalNotes, detail: countText('overview.count.totalRecent', noteRecentChanges, { total: totalNotes, recent: noteRecentChanges }), actionKind: 'notes', actionLabel: tr('overview.openNotes') } : null,
    hasFiles ? { key: 'files', label: tr('overview.files'), count: fileRecentChanges, detail: countText('overview.count.recentChanges', fileRecentChanges), actionKind: 'files', actionLabel: tr('overview.openFiles') } : null,
    hasBrowserInbox ? { key: 'captures', label: tr('overview.captures'), count: unprocessedCaptures.length, detail: countText('overview.count.captures', unprocessedCaptures.length), actionKind: 'browser-inbox', actionLabel: tr('overview.reviewInbox') } : null,
    hasActivity ? { key: 'activity', label: tr('overview.activity'), count: activityEvents.length, detail: countText('overview.count.events', activityEvents.length), actionKind: 'activity', actionLabel: tr('overview.viewActivity') } : null,
    hasJournal ? { key: 'journal', label: tr('overview.journal'), count: journalEntries.length, detail: countText('overview.count.journal', journalEntries.length), actionKind: 'journal', actionLabel: tr('overview.openJournal') } : null,
    hasAttentionTools ? { key: 'attention', label: tr('overview.attention'), count: needsAttention.length, detail: countText('overview.count.pending', needsAttention.length), actionKind: attentionActionKind, actionLabel: tr('overview.reviewPending') } : null,
  ].filter(Boolean);

  onMount(() => {
    unsubscribeLocale = i18n.subscribe((nextLocale) => locale = nextLocale);
    toolProbe += 1;
  });

  onDestroy(() => unsubscribeLocale?.());

  $: if (workspaceRootPath && (workspaceRootPath !== loadedWorkspaceRoot || overviewToolKey !== loadedToolKey)) {
    loadOverview();
  }

  function hasTool(name, tools = availableTools) {
    toolProbe;
    name = String(name || '').toLowerCase();
    const fromProps = (tools || []).some(tool => {
      const label = `${tool?.title || ''} ${tool?.id || ''} ${tool?.pluginId || ''}`.toLowerCase();
      return label.includes(name);
    });
    return fromProps;
  }

  function actionIsAvailable(kind) {
    if (kind === 'notes') return hasNotes;
    if (kind === 'files') return hasFiles;
    if (kind === 'browser-inbox') return hasBrowserInbox;
    if (kind === 'activity') return hasActivity;
    if (kind === 'journal') return hasJournal;
    if (kind === 'todo') return hasTodos;
    return false;
  }

  function filterAvailableItems(items, _toolKey) {
    void _toolKey;
    return (items || []).filter(item => actionIsAvailable(item?.actionKind));
  }

  function fallbackAttentionAction() {
    if (hasBrowserInbox) return 'browser-inbox';
    if (hasTodos) return 'todo';
    if (hasActivity && hasJournal) return 'journal';
    return '';
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

  function browserCaptureRowsForWorkspace(settings) {
    const workspace = String(workspaceRootPath || '').trim();
    if (!workspace) return [];
    const seen = new Set();
    return [
      'captures:global',
      workspaceKey('captures:workspace:'),
      'captures',
    ].flatMap(key => normalizeRows(settings?.[key])).filter(item => {
      const tagged = String(item.workspaceRootPath || item.workspaceName || item.workspaceNodeId || '').trim();
      if (tagged !== workspace) return false;
      if (String(item.globalState || 'inbox') === 'archived') return false;
      const captureId = String(item.captureId || '');
      if (!captureId || seen.has(captureId)) return !captureId;
      seen.add(captureId);
      return true;
    });
  }

  function todoRowsForWorkspace(settings) {
    const workspace = String(workspaceRootPath || '').trim();
    if (!workspace) return [];
    const seen = new Set();
    return normalizeRows(settings?.['todos:global']).filter(item => {
      const tagged = String(item.workspaceRootPath || item.workspaceName || item.workspaceNodeId || '').trim();
      if (tagged !== workspace) return false;
      const todoId = String(item.id || '');
      if (!todoId || seen.has(todoId)) return !todoId;
      seen.add(todoId);
      return true;
    });
  }

  function todoDateMs(value) {
    const raw = String(value || '').trim();
    if (!raw) return 0;
    const normalized = /^\d{4}-\d{2}-\d{2}$/.test(raw) ? `${raw}T00:00:00` : raw;
    const date = new Date(normalized);
    return Number.isNaN(date.getTime()) ? 0 : date.getTime();
  }

  function todoTitle(item) {
    return String(item?.title || item?.name || 'Untitled todo').trim() || 'Untitled todo';
  }

  function todoAttentionState(item) {
    if (String(item?.status || 'open').toLowerCase() !== 'open') return '';
    const now = Date.now();
    const dueAt = todoDateMs(item?.dueAt);
    const reminderAt = todoDateMs(item?.reminderAt);
    if (reminderAt && reminderAt <= now) return 'Reminder due';
    if (dueAt && dueAt < now) return 'Overdue';
    if (dueAt && dueAt <= now + 3 * 24 * 60 * 60 * 1000) return 'Due soon';
    return '';
  }

  function todoTimeValue(item) {
    return item?.dueAt || item?.reminderAt || item?.updatedAt || item?.createdAt || '';
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
    if (!ms) return tr('overview.time.none');
    const diff = Date.now() - ms;
    if (diff < 0) return absoluteTime(value);
    const minute = 60 * 1000;
    const hour = 60 * minute;
    const day = 24 * hour;
    if (diff < minute) return tr('overview.time.now');
    if (diff < hour) return tr('overview.time.minutes', { count: Math.floor(diff / minute) });
    if (diff < day) return tr('overview.time.hours', { count: Math.floor(diff / hour) });
    if (diff < 2 * day) return tr('overview.time.yesterday');
    if (diff < 7 * day) return tr('overview.time.days', { count: Math.floor(diff / day) });
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
    if (type === 'workspace.opened') return tr('overview.event.workspaceOpened');
    return item?.title || item?.summary || tr('overview.event.activity');
  }

  function actionForCategory(category) {
    if (category === 'notes') return { kind: 'notes', label: tr('overview.openNotes') };
    if (category === 'files') return { kind: 'files', label: tr('overview.openFiles') };
    if (category === 'captures') return { kind: 'browser-inbox', label: tr('overview.reviewInbox') };
    if (category === 'journal') return { kind: 'journal', label: tr('overview.openJournal') };
    return { kind: 'activity', label: tr('overview.viewActivity') };
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
      actionLabel: tr('overview.reviewInbox'),
    }));
    const journalItems = journalRows.map(item => ({
      id: item.entryId || `journal:${timeValue(item)}`,
      category: 'journal',
      title: `Added journal entry ${quoted(journalTitle(item))}`,
      meta: `${itemTimeLabel(item)}${item.minutes ? ' · ' + item.minutes + ' min' : ''}`,
      time: timeValue(item),
      absolute: absoluteTime(timeValue(item)),
      actionKind: 'journal',
      actionLabel: tr('overview.openJournal'),
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

  function buildContinueItems(events, captureRows, journalRows, todoRows) {
    const todoCandidates = [...todoRows].sort((a, b) => todoDateMs(todoTimeValue(a)) - todoDateMs(todoTimeValue(b))).map(item => ({
      id: item.id || `todo:${todoTitle(item)}`,
      category: 'todo',
      title: `Todo ${quoted(todoTitle(item))}`,
      meta: `${todoAttentionState(item)}${item.dueAt ? ` · Due ${item.dueAt}` : ''}`,
      time: todoTimeValue(item),
      absolute: absoluteTime(todoTimeValue(item)),
      actionKind: 'todo',
      actionLabel: tr('overview.openTodos'),
    }));
    const captureCandidates = sortByTime(captureRows).map(item => ({
      id: item.captureId || `capture:${timeValue(item)}`,
      category: 'captures',
      title: `Review capture ${quoted(captureTitle(item))}`,
      meta: `${itemTimeLabel(item)} · ${item.domain || item.kind || 'Browser capture'}`,
      time: timeValue(item),
      absolute: absoluteTime(timeValue(item)),
      actionKind: 'browser-inbox',
      actionLabel: tr('overview.reviewInbox'),
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
      actionLabel: tr('overview.openJournal'),
    }));
    return [...todoCandidates, ...captureCandidates, ...noteCandidates, ...fileCandidates, ...journalCandidates].slice(0, 4);
  }

  function buildNeedsAttention(captureRows, candidates, todoRows) {
    const captureItems = sortByTime(captureRows).slice(0, 4).map(item => ({
      id: item.captureId || `capture:${timeValue(item)}`,
      title: captureTitle(item),
      meta: `${item.kind || 'capture'} · ${itemTimeLabel(item)}`,
      actionKind: 'browser-inbox',
      actionLabel: tr('overview.reviewInbox'),
    }));
    const candidateItems = sortByTime(candidates).slice(0, 4).map(item => ({
      id: item.candidateId || `work-session:${timeValue(item)}`,
      title: 'Possible journal entry',
      meta: `Workspace: ${item.workspaceRootPath || workspaceRootPath || 'Unknown'} · ${item.estimatedMinutes || 0} min · ${item.activityCount || (item.activityIds || []).length || 0} activities`,
      actionKind: 'journal',
      actionLabel: tr('overview.reviewCandidate'),
      toolRequest: { type: 'work-session-candidate', candidate: item },
    }));
    const todoItems = [...todoRows].sort((a, b) => todoDateMs(todoTimeValue(a)) - todoDateMs(todoTimeValue(b))).slice(0, 4).map(item => ({
      id: item.id || `todo:${todoTitle(item)}`,
      title: todoTitle(item),
      meta: `${todoAttentionState(item)}${item.dueAt ? ` · Due ${item.dueAt}` : ''}`,
      actionKind: 'todo',
      actionLabel: tr('overview.openTodos'),
    }));
    return [...todoItems, ...captureItems, ...candidateItems].slice(0, 6);
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

  function lastActiveDate(items, captureRows, journalRows, todoRows) {
    const source = sortByTime([...items, ...captureRows, ...journalRows, ...todoRows])[0];
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

  async function loadWorkspaceResources(toolState) {
    const workspace = String(workspaceRootPath || '').trim();
    if (!workspace) return { keyResources: [], totalNotes: 0 };
    const [rootEntries, notesEntries] = await Promise.all([
      toolState.files ? listFiles('verstak.files', workspace) : Promise.resolve([]),
      toolState.notes ? listFiles('verstak.notes', `${workspace}/Notes`) : Promise.resolve([]),
    ]);
    const noteFiles = notesEntries.filter(item => item?.type === 'file');
    const overview = [...notesEntries, ...rootEntries].find(item => /(^|\/)overview\.md$/i.test(String(item.relativePath || item.name || '')));
    return {
      totalNotes: noteFiles.length,
      keyResources: overview ? [{
        id: overview.relativePath || overview.name,
        title: overview.name || fileName(overview.relativePath) || 'Overview.md',
        meta: overview.relativePath || tr('overview.overviewNote'),
        actionKind: toolState.notes ? 'notes' : 'files',
        actionLabel: toolState.notes ? tr('overview.openNotes') : tr('overview.openFiles'),
      }] : [],
    };
  }

  async function loadOverview() {
    const workspaceAtStart = String(workspaceRootPath || '').trim();
    const toolKeyAtStart = overviewToolKey;
    const toolState = {
      notes: hasNotes,
      files: hasFiles,
      browserInbox: hasBrowserInbox,
      activity: hasActivity,
      journal: hasJournal,
      todos: hasTodos,
    };
    loadedWorkspaceRoot = workspaceAtStart;
    loadedToolKey = toolKeyAtStart;
    loading = true;
    const [browserSettings, activitySettings, activityRecords, journalSettings, todoSettings, resources] = await Promise.all([
      toolState.browserInbox ? readPluginSettings('verstak.browser-inbox') : Promise.resolve({}),
      toolState.activity ? readPluginSettings('verstak.activity') : Promise.resolve({}),
      toolState.activity && App.ReadPluginDataNDJSON
        ? App.ReadPluginDataNDJSON('verstak.activity', 'activity-events')
          .then(value => decodeTuple(value, []))
          .catch(() => [])
        : Promise.resolve([]),
      toolState.journal ? readPluginSettings('verstak.journal') : Promise.resolve({}),
      toolState.todos ? readPluginSettings('verstak.todo') : Promise.resolve({}),
      loadWorkspaceResources(toolState),
    ]);
    if (workspaceAtStart !== String(workspaceRootPath || '').trim() || toolKeyAtStart !== overviewToolKey) return;

    captures = toolState.browserInbox ? browserCaptureRowsForWorkspace(browserSettings) : [];
    activityEvents = toolState.activity ? normalizeRows(activityRecords).filter(item => {
      const tagged = String(item.workspaceRootPath || item.workspaceName || item.workspaceNodeId || '').trim();
      return !tagged || tagged === workspaceAtStart;
    }) : [];
    journalEntries = toolState.journal ? rowsFor(journalSettings, [
      workspaceKey('worklog:workspace:'),
      'worklog',
    ]) : [];
    todos = toolState.todos ? todoRowsForWorkspace(todoSettings) : [];
    workSessionCandidates = toolState.activity && toolState.journal ? rowsFor(activitySettings, [
      workspaceKey('work-session-candidates:workspace:'),
    ]) : [];
    keyResources = resources.keyResources;
    totalNotes = resources.totalNotes;
    loading = false;
  }

  function openTool(kind, toolRequest = null) {
    dispatch('openTool', { kind, toolRequest });
  }
</script>

<div class="today-root overview-root" aria-label={tr('workspace.overview')} data-overview-root>
  <div class="today-header overview-header">
    <div>
      <h2>{tr('workspace.overview')}</h2>
      <p title={lastActive ? absoluteTime(lastActive) : ''}>
        {#if loading}
          {tr('overview.loadingContext')}
        {:else if lastActive}
          {tr('overview.lastActive', { time: relativeTime(lastActive) })}
        {:else}
          {tr('overview.noRecentActivity')}
        {/if}
      </p>
    </div>
    <button type="button" data-overview-action="refresh" on:click={loadOverview}>{tr('overview.refresh')}</button>
  </div>

  <div class="today-summary overview-summary" aria-label={tr('overview.summary')}>
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
        <small>{loading ? tr('common.loading') : item.detail}</small>
        <em>{item.actionLabel}</em>
      </button>
    {/each}
  </div>

  <div class="overview-layout">
    <main class="overview-main">
      <section class="today-resume overview-continue" data-overview-section="continue">
        <div class="today-resume-copy overview-continue-copy">
          <span>{tr('overview.continue')}</span>
          <h3>{tr('overview.continueHint')}</h3>
        </div>
        {#if loading}
          <p class="today-empty compact">{tr('overview.loadingSignals')}</p>
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
            <strong>{tr('overview.noResume')}</strong>
            <p>{tr('overview.noResumeHint')}</p>
          </div>
        {/if}
      </section>

      <section class="today-panel overview-panel overview-recent" data-overview-section="recent">
        <div class="today-panel-head overview-panel-head">
          <div>
            <h3>{tr('overview.recentChanges')}</h3>
            <p>{tr('overview.recentChangesHint')}</p>
          </div>
          <div class="overview-filters" aria-label={tr('overview.recentFilter')}>
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
          <p class="today-empty">{tr('overview.loadingRecent')}</p>
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
          <p class="today-empty">{tr('overview.noChanges')}</p>
        {/if}
      </section>
    </main>

    <aside class="overview-side">
      {#if hasAttentionTools && (needsAttention.length || !loading)}
        <section class="today-panel overview-panel" data-overview-section="attention">
          <div class="today-panel-head overview-panel-head">
            <div>
              <h3>{tr('overview.attention')}</h3>
              <p>{tr('overview.attentionHint')}</p>
            </div>
          </div>
          {#if loading}
            <p class="today-empty">{tr('overview.loadingPending')}</p>
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
            <p class="today-empty compact">{tr('overview.noPending')}</p>
          {/if}
        </section>
      {/if}

      {#if keyResources.length}
        <section class="today-panel overview-panel secondary" data-overview-section="key-resources">
          <div class="today-panel-head overview-panel-head">
            <h3>{tr('overview.keyResources')}</h3>
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
