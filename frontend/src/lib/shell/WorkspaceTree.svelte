<script>
  import { onDestroy, onMount } from 'svelte';
  import * as App from '../../../wailsjs/go/api/App';
  import Icon from '../ui/Icon.svelte';
  import Modal from '../ui/Modal.svelte';
  import Select from '../ui/Select.svelte';
  import TreeNode from './TreeNode.svelte';
  import { i18n } from '../i18n/index.js';

  let tree = { roots: [], currentWorkspaceId: '', revision: 0 };
  let loading = true; let error = '';
  let expandedIds = {}; let activeWid = '';
  let focusedKey = '';
  let locale = i18n.getLocale();
  let unsubscribeLocale = null;
  $: tr = ((al) => (k, p, f) => { void al; return i18n.t(k, p, f); })(locale);

  // Modal state
  let modal = null; let formName = ''; let formParentId = ''; let formTemplateId = 'default'; let folderIconId = ''; let folderColor = ''; let folderEditorView = 'form'; let iconSearch = ''; $: filteredIcons = LUCIDE_ICONS.filter(i => !iconSearch || i.toLowerCase().includes(iconSearch.toLowerCase())).slice(0, 200);
  let formError = ''; let formBusy = false;
  let templates = [];
  let ctxMenu = null;

  // Drag state
  let dragOverRoot = false;
  let dragOverFolderId = '';

  onMount(async () => {
    unsubscribeLocale = i18n.subscribe((l) => { locale = l; });
    await loadTree(); await loadTemplates();
    window.addEventListener('verstak:workspace-tree-changed', loadTree);
    // Also listen via Wails runtime events (Go EventsEmit).
    if (window.runtime?.EventsOn) {
      window.runtime.EventsOn('verstak:workspace-tree-changed', loadTree);
    }
  });
  onDestroy(() => {
    if (unsubscribeLocale) unsubscribeLocale();
    window.removeEventListener('verstak:workspace-tree-changed', loadTree);
  });

  async function loadTree() {
    loading = true; error = '';
    try {
      const raw = await App.GetWorkspaceTreeV2();
      if (raw?.error) { error = raw.error; return; }
      tree = raw || { roots: [], currentWorkspaceId: '', revision: 0 };
      activeWid = tree.currentWorkspaceId || '';
      await loadExpanded();
      if (activeWid) ensureExpandedToWorkspace(activeWid);
      if (!focusedKey && tree.roots?.length) focusedKey = tree.roots[0].key;
    } catch (e) { error = tr('workspaceTree.loadError'); }
    loading = false;
  }

  async function loadExpanded() {
    try {
      const settings = await App.GetAppSettings();
      if (settings?.expandedFolderIds) expandedIds = Object.fromEntries(settings.expandedFolderIds.map(id => ['folder:' + id, true]));
    } catch {}
  }
  async function saveExpanded() {
    const ids = Object.keys(expandedIds).filter(k => k.startsWith('folder:')).map(k => k.slice(7));
    try { await App.UpdateAppSettings({ expandedFolderIds: ids }); } catch {}
  }

  async function loadTemplates() {
    try {
      const tlist = await App.ListWorkspaceTemplates();
      templates = Array.isArray(tlist) ? tlist : [];
    } catch { templates = []; }
  }

  function ensureExpandedToWorkspace(wid) {
    for (const root of tree.roots || []) if (expandToChild(root, wid)) return;
  }
  function expandToChild(node, targetWid) {
    if (node.kind === 'workspace' && node.id === targetWid) return true;
    if (node.children) for (const c of node.children) {
      if (expandToChild(c, targetWid)) { expandedIds[node.key] = true; return true; }
    }
    return false;
  }

  function toggleExpand(key) {
    expandedIds[key] = !expandedIds[key];
    expandedIds = expandedIds;
    focusedKey = key;
    saveExpanded();
  }

  async function selectWorkspace(wid) {
    const err = await App.SetCurrentWorkspaceV2(wid);
    if (err) { error = err; return; }
    activeWid = wid;
    focusedKey = 'workspace:' + wid;
    const ws = await App.GetWorkspaceByID(wid);
    const rootPath = ws?.rootPath || '';
    window.dispatchEvent(new CustomEvent('verstak:workspace-selected', {
      detail: { workspaceId: wid, workspaceName: rootPath, workspaceRootPath: rootPath }
    }));
  }

  // ── Flat visible node list for keyboard nav ────────────────────────────────
  function visibleNodes() {
    const out = [];
    function walk(nodes, depth) {
      for (const n of nodes || []) {
        out.push({ key: n.key, kind: n.kind, id: n.id, name: n.name, depth });
        if (n.kind === 'folder' && expandedIds[n.key] && n.children) walk(n.children, depth + 1);
      }
    }
    walk(tree.roots || [], 0);
    return out;
  }

  function handleNav(e) {
    const dir = e.detail?.dir;
    const vis = visibleNodes();
    const idx = vis.findIndex(n => n.key === focusedKey);
    let next = -1;
    if (dir === 'next' && idx < vis.length - 1) next = idx + 1;
    else if (dir === 'prev' && idx > 0) next = idx - 1;
    else if (dir === 'child' && idx >= 0) {
      const cur = vis[idx];
      if (cur.kind === 'folder' && expandedIds[cur.key]) next = idx + 1;
    } else if (dir === 'parent' && idx >= 0) {
      for (let i = idx - 1; i >= 0; i--) {
        if (vis[i].depth < vis[idx].depth) { next = i; break; }
      }
    }
    if (next >= 0) { focusedKey = vis[next].key; }
  }

  function handleRename(e) { openRename(e.detail.kind, e.detail.id, e.detail.name); }
  function handleTrash(e) { openTrash(e.detail.kind, e.detail.id, e.detail.name); }

  // ── Create/Rename/Move/Trash modals ────────────────────────────────────────
  function openCreateFolder(pid) { modal = { type: 'create-folder', parentId: pid }; formName = ''; formParentId = pid || ''; formError = ''; formBusy = false; folderIconId = ''; folderColor = ''; folderEditorView = 'form'; }
  function openCreateWorkspace(pid) { modal = { type: 'create-workspace', parentId: pid }; formName = ''; formParentId = pid || ''; formTemplateId = templates[0]?.id || 'default'; formError = ''; formBusy = false; }
  function openRename(kind, id, name) { modal = { type: 'rename', kind, id }; formName = name; formError = ''; formBusy = false; }
  function openTrash(kind, id, name) { modal = { type: 'trash', kind, id, name }; formBusy = false; }
  function openEditFolder(id, name) {
    modal = { type: 'edit-folder', id };
    formName = name;
    folderIconId = ''; folderColor = '';
    folderEditorView = 'form';
    formError = ''; formBusy = false;
    loadFolderAppearance(id);
  }
  async function loadFolderAppearance(folderId) {
    try {
      const reg = window.__VERSTAK_PLUGIN_REGISTRY__;
      const comp = reg && reg['verstak.folder-appearance'];
      if (!comp) return;
      const api = window.createPluginAPI('verstak.folder-appearance');
      if (api && api.folders && api.folders.getAppearance) {
        const a = await api.folders.getAppearance(folderId);
        folderIconId = a.iconId || '';
        folderColor = a.colorId || '';
      }
    } catch {}
  }
  function closeModal() { if (!formBusy) modal = null; }

  async function doCreateFolder() { const n = formName.trim(); if (!n) { formError = tr('workspaceTree.nameRequired'); return; } formBusy = true; const r = await App.CreateFolderV2(formParentId || '', n); if (r?.error) { formError = r.error; formBusy = false; return; } if (formParentId) { expandedIds['folder:' + formParentId] = true; saveExpanded(); }
  const fid = r?.id;
  if (fid && (folderIconId || folderColor)) {
    try {
      const api = window.createPluginAPI('verstak.folder-appearance');
      if (api && api.folders && api.folders.setAppearance) {
        await api.folders.setAppearance(fid, { iconId: folderIconId, colorId: folderColor });
      }
    } catch {}
  }
  modal = null; await loadTree(); }
  async function doCreateWorkspace() { const n = formName.trim(); if (!n) { formError = tr('workspaceTree.nameRequired'); return; } formBusy = true; const r = await App.CreateWorkspaceV2(formParentId || '', n, formTemplateId); if (r?.error) { formError = r.error; formBusy = false; return; } if (formParentId) { expandedIds['folder:' + formParentId] = true; saveExpanded(); } const wid = r?.id; modal = null; await loadTree(); if (wid) await selectWorkspace(wid); }
  async function doRename() { const n = formName.trim(); if (!n) { formError = tr('workspaceTree.nameRequired'); return; } formBusy = true; let err = modal.kind === 'folder' ? await App.RenameFolderV2(modal.id, n) : await App.RenameWorkspaceV2(modal.id, n); if (err) { formError = err; formBusy = false; return; } modal = null; await loadTree(); }
  function openIconPicker() { folderEditorView = 'icon-picker'; iconSearch = ''; }
  function selectFolderIcon(id) { folderIconId = id; folderEditorView = 'form'; }
  function resetFolderColor() { folderColor = ''; }

  async function doEditFolder() {
    const n = formName.trim();
    if (!n) { formError = tr('workspaceTree.nameRequired'); return; }
    formBusy = true;
    const err = await App.RenameFolderV2(modal.id, n);
    if (err) { formError = err; formBusy = false; return; }
    // Save appearance if plugin available
    try {
      const api = window.createPluginAPI('verstak.folder-appearance');
      if (api && api.folders && api.folders.setAppearance) {
        await api.folders.setAppearance(modal.id, { iconId: folderIconId, colorId: folderColor });
      }
    } catch {}
    modal = null;
    await loadTree();
  }
  async function doMove() { formBusy = true; let err = modal.kind === 'folder' ? await App.MoveFolderV2(modal.id, formParentId || '') : await App.MoveWorkspaceV2(modal.id, formParentId || ''); if (err) { formError = err; formBusy = false; return; } modal = null; await loadTree(); }
  async function doTrash() { formBusy = true; if (modal.kind === 'folder') await App.TrashFolderV2(modal.id); else { await App.TrashWorkspaceV2(modal.id); if (activeWid === modal.id) activeWid = ''; } modal = null; await loadTree(); }

  // ── Context menu ───────────────────────────────────────────────────────────
  function onCtx(e) { ctxMenu = { x: e.detail.e.clientX, y: e.detail.e.clientY, kind: e.detail.kind, id: e.detail.id, name: e.detail.name }; }
  function closeCtx() { ctxMenu = null; }

  // ── Drag-and-drop ──────────────────────────────────────────────────────────
  let dragCounter = 0;
  let draggedNodeParentId = ''; // Track whether dragged node has a parent
  function onRootDragOver(e) {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    dragCounter++;
    // Only show root drop zone if dragged node is nested (has a parent folder).
    if (!draggedNodeParentId) return;
    dragOverRoot = true;
  }
  function onRootDragLeave(e) { dragCounter--; if (dragCounter <= 0) { dragOverRoot = false; dragCounter = 0; } }
  function resetDragState() {
    dragOverRoot = false;
    dragOverFolderId = '';
    dragCounter = 0;
    draggedNodeParentId = '';
  }
  function onNodeDragStart(e) {
    draggedNodeParentId = findNodeParentID(e.detail?.id) || '';
  }

  function onRootDrop(e) {
    e.preventDefault();
    e.stopPropagation();
    resetDragState();
    try {
      const data = JSON.parse(e.dataTransfer.getData('application/x-verstak-node'));
      if (data.kind === 'folder') App.MoveFolderV2(data.id, '').then(loadTree).catch(() => {}).finally(resetDragState);
      else App.MoveWorkspaceV2(data.id, '').then(loadTree).catch(() => {}).finally(resetDragState);
    } catch { resetDragState(); }
  }
  function onNodeDrop(e) {
    resetDragState();
    const { source, targetId } = e.detail;
    if (source.kind === 'folder') App.MoveFolderV2(source.id, targetId).then(loadTree).catch(() => {}).finally(resetDragState);
    else App.MoveWorkspaceV2(source.id, targetId).then(loadTree).catch(() => {}).finally(resetDragState);
  }

  // ── Helpers ────────────────────────────────────────────────────────────────
  function flatFolders(roots, out = []) { for (const r of roots || []) { if (r.kind === 'folder') { out.push(r); flatFolders(r.children, out); } } return out; }
  function descendantIds(node) { const ids = new Set(); function walk(n) { for (const c of n.children || []) { ids.add(c.id); walk(c); } } walk(node); return ids; }
  function moveExcludedIds() { if (!modal?.id) return new Set(); const n = findNode(tree.roots, modal.id); return n ? descendantIds(n) : new Set(); }
  $: moveExcluded = moveExcludedIds();

  function subtreeCounts(id) { let folders = 0, wss = 0; const n = findNode(tree.roots || [], id); if (n) count(n); return { folders, workspaces: wss };
    function count(nd) { for (const c of nd.children || []) { if (c.kind === 'folder') { folders++; count(c); } else wss++; } } }
  function findNode(nodes, id) { for (const n of nodes) { if (n.id === id) return n; const f = findNode(n.children || [], id); if (f) return f; } return null; }
  function findNodeParentID(id) { return parentIDFor(tree.roots, id); }
  function parentIDFor(nodes, id) { for (const n of nodes) { if (n.children) for (const c of n.children) { if (c.id === id) return n.id; } const f = parentIDFor(n.children, id); if (f) return f; } return ''; }

  // ── Template plugin display ─────────────────────────────────────────────────
  const PLUGIN_NAMES = {
    'verstak.notes': 'Заметки', 'verstak.files': 'Файлы', 'verstak.journal': 'Журнал',
    'verstak.activity': 'Активность', 'verstak.browser-inbox': 'Браузер',
    'verstak.todo': 'Задачи', 'verstak.secrets': 'Секреты',
  };
  function pluginDisplayName(pluginId) {
    const key = pluginId.replace('verstak.', '');
    return tr(`plugin.${key}`, undefined, PLUGIN_NAMES[pluginId] || pluginId);
  }
  function pluginAvailable(pluginId) {
    return true; // plugins are loaded by core; availability checked at contribution filtering time
  }

  function onKeyDown(e) { if (e.key === "Escape") { closeCtx(); closeModal(); resetDragState(); } }

  const LUCIDE_ICONS = ["a-arrow-down", "a-arrow-up", "a-large-small", "accessibility", "activity", "air-vent", "airplay", "alarm-clock", "alarm-clock-check", "alarm-clock-minus", "alarm-clock-off", "alarm-clock-plus", "alarm-smoke", "album", "align-center-horizontal", "align-center-vertical", "align-end-horizontal", "align-end-vertical", "align-horizontal-distribute-center", "align-horizontal-distribute-end", "align-horizontal-distribute-start", "align-horizontal-justify-center", "align-horizontal-justify-end", "align-horizontal-justify-start", "align-horizontal-space-around", "align-horizontal-space-between", "align-start-horizontal", "align-start-vertical", "align-vertical-distribute-center", "align-vertical-distribute-end", "align-vertical-distribute-start", "align-vertical-justify-center", "align-vertical-justify-end", "align-vertical-justify-start", "align-vertical-space-around", "align-vertical-space-between", "ambulance", "ampersand", "ampersands", "amphora", "anchor", "angry", "annoyed", "antenna", "anvil", "aperture", "app-window", "app-window-mac", "apple", "archive", "archive-restore", "archive-x", "armchair", "arrow-big-down", "arrow-big-down-dash", "arrow-big-left", "arrow-big-left-dash", "arrow-big-right", "arrow-big-right-dash", "arrow-big-up", "arrow-big-up-dash", "arrow-down", "arrow-down-0-1", "arrow-down-1-0", "arrow-down-a-z", "arrow-down-from-line", "arrow-down-left", "arrow-down-narrow-wide", "arrow-down-right", "arrow-down-to-dot", "arrow-down-to-line", "arrow-down-up", "arrow-down-wide-narrow", "arrow-down-z-a", "arrow-left", "arrow-left-from-line", "arrow-left-right", "arrow-left-to-line", "arrow-right", "arrow-right-from-line", "arrow-right-left", "arrow-right-to-line", "arrow-up", "arrow-up-0-1", "arrow-up-1-0", "arrow-up-a-z", "arrow-up-down", "arrow-up-from-dot", "arrow-up-from-line", "arrow-up-left", "arrow-up-narrow-wide", "arrow-up-right", "arrow-up-to-line", "arrow-up-wide-narrow", "arrow-up-z-a", "arrows-up-from-line", "asterisk", "at-sign", "atom", "audio-lines", "audio-waveform", "award", "axe", "axis-3d", "baby", "backpack", "badge", "badge-alert", "badge-cent", "badge-check", "badge-dollar-sign", "badge-euro", "badge-indian-rupee", "badge-info", "badge-japanese-yen", "badge-minus", "badge-percent", "badge-plus", "badge-pound-sterling", "badge-question-mark", "badge-russian-ruble", "badge-swiss-franc", "badge-turkish-lira", "badge-x", "baggage-claim", "balloon", "ban", "banana", "bandage", "banknote", "banknote-arrow-down", "banknote-arrow-up", "banknote-x", "barcode", "barrel", "baseline", "bath", "battery", "battery-charging", "battery-full", "battery-low", "battery-medium", "battery-plus", "battery-warning", "beaker", "bean", "bean-off", "bed", "bed-double", "bed-single", "beef", "beer", "beer-off", "bell", "bell-dot", "bell-electric", "bell-minus", "bell-off", "bell-plus", "bell-ring", "between-horizontal-end", "between-horizontal-start", "between-vertical-end", "between-vertical-start", "biceps-flexed", "bike", "binary", "binoculars", "biohazard", "bird", "birdhouse", "bitcoin", "blend", "blinds", "blocks", "bluetooth", "bluetooth-connected", "bluetooth-off", "bluetooth-searching", "bold", "bolt", "bomb", "bone", "book", "book-a", "book-alert", "book-audio", "book-check", "book-copy", "book-dashed", "book-down", "book-headphones", "book-heart", "book-image", "book-key", "book-lock", "book-marked", "book-minus", "book-open", "book-open-check", "book-open-text", "book-plus", "book-search", "book-text", "book-type", "book-up", "book-up-2", "book-user", "book-x", "bookmark", "bookmark-check", "bookmark-minus", "bookmark-plus", "bookmark-x", "boom-box", "bot", "bot-message-square", "bot-off", "bottle-wine", "bow-arrow", "box", "boxes", "braces", "brackets", "brain", "brain-circuit", "brain-cog", "brick-wall", "brick-wall-fire", "brick-wall-shield", "briefcase", "briefcase-business", "briefcase-conveyor-belt", "briefcase-medical", "bring-to-front", "brush", "brush-cleaning", "bubbles", "bug", "bug-off", "bug-play", "building", "building-2", "bus", "bus-front", "cable", "cable-car", "cake", "cake-slice", "calculator", "calendar", "calendar-1", "calendar-arrow-down", "calendar-arrow-up", "calendar-check", "calendar-check-2", "calendar-clock", "calendar-cog", "calendar-days", "calendar-fold", "calendar-heart", "calendar-minus", "calendar-minus-2", "calendar-off", "calendar-plus", "calendar-plus-2", "calendar-range", "calendar-search", "calendar-sync", "calendar-x", "calendar-x-2", "calendars", "camera", "camera-off", "candy", "candy-cane", "candy-off", "cannabis", "cannabis-off", "captions", "captions-off", "car", "car-front", "car-taxi-front", "caravan", "card-sim", "carrot", "case-lower", "case-sensitive", "case-upper", "cassette-tape", "cast", "castle", "cat", "cctv", "chart-area", "chart-bar", "chart-bar-big", "chart-bar-decreasing", "chart-bar-increasing", "chart-bar-stacked", "chart-candlestick", "chart-column", "chart-column-big", "chart-column-decreasing", "chart-column-increasing", "chart-column-stacked", "chart-gantt", "chart-line", "chart-network", "chart-no-axes-column", "chart-no-axes-column-decreasing", "chart-no-axes-column-increasing", "chart-no-axes-combined", "chart-no-axes-gantt", "chart-pie", "chart-scatter", "chart-spline", "check", "check-check", "check-line", "chef-hat", "cherry", "chess-bishop", "chess-king", "chess-knight", "chess-pawn", "chess-queen", "chess-rook", "chevron-down", "chevron-first", "chevron-last", "chevron-left", "chevron-right", "chevron-up", "chevrons-down", "chevrons-down-up", "chevrons-left", "chevrons-left-right", "chevrons-left-right-ellipsis", "chevrons-right", "chevrons-right-left", "chevrons-up", "chevrons-up-down", "chromium", "church", "cigarette", "cigarette-off", "circle", "circle-alert", "circle-arrow-down", "circle-arrow-left", "circle-arrow-out-down-left", "circle-arrow-out-down-right", "circle-arrow-out-up-left", "circle-arrow-out-up-right", "circle-arrow-right", "circle-arrow-up", "circle-check", "circle-check-big", "circle-chevron-down", "circle-chevron-left", "circle-chevron-right", "circle-chevron-up", "circle-dashed", "circle-divide", "circle-dollar-sign", "circle-dot", "circle-dot-dashed", "circle-ellipsis", "circle-equal", "circle-fading-arrow-up", "circle-fading-plus", "circle-gauge", "circle-minus", "circle-off", "circle-parking", "circle-parking-off", "circle-pause", "circle-percent", "circle-pile", "circle-play", "circle-plus", "circle-pound-sterling", "circle-power", "circle-question-mark", "circle-slash", "circle-slash-2", "circle-small", "circle-star", "circle-stop", "circle-user", "circle-user-round", "circle-x", "circuit-board", "citrus", "clapperboard", "clipboard", "clipboard-check", "clipboard-clock", "clipboard-copy", "clipboard-list", "clipboard-minus", "clipboard-paste", "clipboard-pen", "clipboard-pen-line", "clipboard-plus", "clipboard-type", "clipboard-x", "clock", "clock-1", "clock-10", "clock-11", "clock-12", "clock-2", "clock-3", "clock-4", "clock-5", "clock-6", "clock-7", "clock-8", "clock-9", "clock-alert", "clock-arrow-down", "clock-arrow-up", "clock-check", "clock-fading", "clock-plus", "closed-caption", "cloud", "cloud-alert", "cloud-backup", "cloud-check", "cloud-cog", "cloud-download", "cloud-drizzle", "cloud-fog", "cloud-hail", "cloud-lightning", "cloud-moon", "cloud-moon-rain", "cloud-off", "cloud-rain", "cloud-rain-wind", "cloud-snow", "cloud-sun", "cloud-sun-rain", "cloud-sync", "cloud-upload", "cloudy", "clover", "club", "code", "code-xml", "codepen", "codesandbox", "coffee", "cog", "coins", "columns-2", "columns-3", "columns-3-cog", "columns-4", "combine", "command", "compass", "component", "computer", "concierge-bell", "cone", "construction", "contact", "contact-round", "container", "contrast", "cookie", "cooking-pot", "copy", "copy-check", "copy-minus", "copy-plus", "copy-slash", "copy-x", "copyleft", "copyright", "corner-down-left", "corner-down-right", "corner-left-down", "corner-left-up", "corner-right-down", "corner-right-up", "corner-up-left", "corner-up-right", "cpu", "creative-commons", "credit-card", "croissant", "crop", "cross", "crosshair", "crown", "cuboid", "cup-soda", "currency", "cylinder", "dam", "database", "database-backup", "database-search", "database-zap", "decimals-arrow-left", "decimals-arrow-right", "delete", "dessert", "diameter", "diamond", "diamond-minus", "diamond-percent", "diamond-plus", "dice-1", "dice-2", "dice-3", "dice-4", "dice-5", "dice-6", "dices", "diff", "disc", "disc-2", "disc-3", "disc-album", "divide", "dna", "dna-off", "dock", "dog", "dollar-sign", "donut", "door-closed", "door-closed-locked", "door-open", "dot", "download", "drafting-compass", "drama", "dribbble", "drill", "drone", "droplet", "droplet-off", "droplets", "drum", "drumstick", "dumbbell", "ear", "ear-off", "earth", "earth-lock", "eclipse", "egg", "egg-fried", "egg-off", "ellipse", "ellipsis", "ellipsis-vertical", "equal", "equal-approximately", "equal-not", "eraser", "ethernet-port", "euro", "ev-charger", "expand", "external-link", "eye", "eye-closed", "eye-off", "facebook", "factory", "fan", "fast-forward", "feather", "fence", "ferris-wheel", "figma", "file", "file-archive", "file-axis-3d", "file-badge", "file-box", "file-braces", "file-braces-corner", "file-chart-column", "file-chart-column-increasing", "file-chart-line", "file-chart-pie", "file-check", "file-check-corner", "file-clock", "file-code", "file-code-corner", "file-cog", "file-diff", "file-digit", "file-down", "file-exclamation-point", "file-headphone", "file-heart", "file-image", "file-input", "file-key", "file-lock", "file-minus", "file-minus-corner", "file-music", "file-output", "file-pen", "file-pen-line", "file-play", "file-plus", "file-plus-corner", "file-question-mark", "file-scan", "file-search", "file-search-corner", "file-signal", "file-sliders", "file-spreadsheet", "file-stack", "file-symlink", "file-terminal", "file-text", "file-type", "file-type-corner", "file-up", "file-user", "file-video-camera", "file-volume", "file-x", "file-x-corner", "files", "film", "fingerprint-pattern", "fire-extinguisher", "fish", "fish-off", "fish-symbol", "fishing-hook", "fishing-rod", "flag", "flag-off", "flag-triangle-left", "flag-triangle-right", "flame", "flame-kindling", "flashlight", "flashlight-off", "flask-conical", "flask-conical-off", "flask-round", "flip-horizontal-2", "flip-vertical-2", "flower", "flower-2", "focus", "fold-horizontal", "fold-vertical", "folder", "folder-archive", "folder-check", "folder-clock", "folder-closed", "folder-code", "folder-cog", "folder-dot", "folder-down", "folder-git", "folder-git-2", "folder-heart", "folder-input", "folder-kanban", "folder-key", "folder-lock", "folder-minus", "folder-open", "folder-open-dot", "folder-output", "folder-pen", "folder-plus", "folder-root", "folder-search", "folder-search-2", "folder-symlink", "folder-sync", "folder-tree", "folder-up", "folder-x", "folders", "footprints", "forklift", "form", "forward", "frame", "framer", "frown", "fuel", "fullscreen", "funnel", "funnel-plus", "funnel-x", "gallery-horizontal", "gallery-horizontal-end", "gallery-thumbnails", "gallery-vertical", "gallery-vertical-end", "gamepad", "gamepad-2", "gamepad-directional", "gauge", "gavel", "gem", "georgian-lari", "ghost", "gift", "git-branch", "git-branch-minus", "git-branch-plus", "git-commit-horizontal", "git-commit-vertical", "git-compare", "git-compare-arrows", "git-fork", "git-graph", "git-merge", "git-merge-conflict", "git-pull-request", "git-pull-request-arrow", "git-pull-request-closed", "git-pull-request-create", "git-pull-request-create-arrow", "git-pull-request-draft", "github", "gitlab", "glass-water", "glasses", "globe", "globe-lock", "globe-off", "globe-x", "goal", "gpu", "graduation-cap", "grape", "grid-2x2", "grid-2x2-check", "grid-2x2-plus", "grid-2x2-x", "grid-3x2", "grid-3x3", "grip", "grip-horizontal", "grip-vertical", "group", "guitar", "ham", "hamburger", "hammer", "hand", "hand-coins", "hand-fist", "hand-grab", "hand-heart", "hand-helping", "hand-metal", "hand-platter", "handbag", "handshake", "hard-drive", "hard-drive-download", "hard-drive-upload", "hard-hat", "hash", "hat-glasses", "haze", "hd", "hdmi-port", "heading", "heading-1", "heading-2", "heading-3", "heading-4", "heading-5", "heading-6", "headphone-off", "headphones", "headset", "heart", "heart-crack", "heart-handshake", "heart-minus", "heart-off", "heart-plus", "heart-pulse", "heater", "helicopter", "hexagon", "highlighter", "history", "hop", "hop-off", "hospital", "hotel", "hourglass", "house", "house-heart", "house-plug", "house-plus", "house-wifi", "ice-cream-bowl", "ice-cream-cone", "id-card", "id-card-lanyard", "image", "image-down", "image-minus", "image-off", "image-play", "image-plus", "image-up", "image-upscale", "images", "import", "inbox", "indian-rupee", "infinity", "info", "inspection-panel", "instagram", "italic", "iteration-ccw", "iteration-cw", "japanese-yen", "joystick", "kanban", "kayak", "key", "key-round", "key-square", "keyboard", "keyboard-music", "keyboard-off", "lamp", "lamp-ceiling", "lamp-desk", "lamp-floor", "lamp-wall-down", "lamp-wall-up", "land-plot", "landmark", "languages", "laptop", "laptop-minimal", "laptop-minimal-check", "lasso", "lasso-select", "laugh", "layers", "layers-2", "layers-plus", "layout-dashboard", "layout-grid", "layout-list", "layout-panel-left", "layout-panel-top", "layout-template", "leaf", "leafy-green", "lectern", "lens-concave", "lens-convex", "library", "library-big", "life-buoy", "ligature", "lightbulb", "lightbulb-off", "line-dot-right-horizontal", "line-squiggle", "link", "link-2", "link-2-off", "linkedin", "list", "list-check", "list-checks", "list-chevrons-down-up", "list-chevrons-up-down", "list-collapse", "list-end", "list-filter", "list-filter-plus", "list-indent-decrease", "list-indent-increase", "list-minus", "list-music", "list-ordered", "list-plus", "list-restart", "list-start", "list-todo", "list-tree", "list-video", "list-x", "loader", "loader-circle", "loader-pinwheel", "locate", "locate-fixed", "locate-off", "lock", "lock-keyhole", "lock-keyhole-open", "lock-open", "log-in", "log-out", "logs", "lollipop", "luggage", "magnet", "mail", "mail-check", "mail-minus", "mail-open", "mail-plus", "mail-question-mark", "mail-search", "mail-warning", "mail-x", "mailbox", "mails", "map", "map-minus", "map-pin", "map-pin-check", "map-pin-check-inside", "map-pin-house", "map-pin-minus", "map-pin-minus-inside", "map-pin-off", "map-pin-pen", "map-pin-plus", "map-pin-plus-inside", "map-pin-x", "map-pin-x-inside", "map-pinned", "map-plus", "mars", "mars-stroke", "martini", "maximize", "maximize-2", "medal", "megaphone", "megaphone-off", "meh", "memory-stick", "menu", "merge", "message-circle", "message-circle-check", "message-circle-code", "message-circle-dashed", "message-circle-heart", "message-circle-more", "message-circle-off", "message-circle-plus", "message-circle-question-mark", "message-circle-reply", "message-circle-warning", "message-circle-x", "message-square", "message-square-check", "message-square-code", "message-square-dashed", "message-square-diff", "message-square-dot", "message-square-heart", "message-square-lock", "message-square-more", "message-square-off", "message-square-plus", "message-square-quote", "message-square-reply", "message-square-share", "message-square-text", "message-square-warning", "message-square-x", "messages-square", "metronome", "mic", "mic-off", "mic-vocal", "microchip", "microscope", "microwave", "milestone", "milk", "milk-off", "minimize", "minimize-2", "minus", "mirror-rectangular", "mirror-round", "monitor", "monitor-check", "monitor-cloud", "monitor-cog", "monitor-dot", "monitor-down", "monitor-off", "monitor-pause", "monitor-play", "monitor-smartphone", "monitor-speaker", "monitor-stop", "monitor-up", "monitor-x", "moon", "moon-star", "motorbike", "mountain", "mountain-snow", "mouse", "mouse-left", "mouse-off", "mouse-pointer", "mouse-pointer-2", "mouse-pointer-2-off", "mouse-pointer-ban", "mouse-pointer-click", "mouse-right", "move", "move-3d", "move-diagonal", "move-diagonal-2", "move-down", "move-down-left", "move-down-right", "move-horizontal", "move-left", "move-right", "move-up", "move-up-left", "move-up-right", "move-vertical", "music", "music-2", "music-3", "music-4", "navigation", "navigation-2", "navigation-2-off", "navigation-off", "network", "newspaper", "nfc", "non-binary", "notebook", "notebook-pen", "notebook-tabs", "notebook-text", "notepad-text", "notepad-text-dashed", "nut", "nut-off", "octagon", "octagon-alert", "octagon-minus", "octagon-pause", "octagon-x", "omega", "option", "orbit", "origami", "package", "package-2", "package-check", "package-minus", "package-open", "package-plus", "package-search", "package-x", "paint-bucket", "paint-roller", "paintbrush", "paintbrush-vertical", "palette", "panda", "panel-bottom", "panel-bottom-close", "panel-bottom-dashed", "panel-bottom-open", "panel-left", "panel-left-close", "panel-left-dashed", "panel-left-open", "panel-left-right-dashed", "panel-right", "panel-right-close", "panel-right-dashed", "panel-right-open", "panel-top", "panel-top-bottom-dashed", "panel-top-close", "panel-top-dashed", "panel-top-open", "panels-left-bottom", "panels-right-bottom", "panels-top-left", "paperclip", "parentheses", "parking-meter", "party-popper", "pause", "paw-print", "pc-case", "pen", "pen-line", "pen-off", "pen-tool", "pencil", "pencil-line", "pencil-off", "pencil-ruler", "pentagon", "percent", "person-standing", "philippine-peso", "phone", "phone-call", "phone-forwarded", "phone-incoming", "phone-missed", "phone-off", "phone-outgoing", "pi", "piano", "pickaxe", "picture-in-picture", "picture-in-picture-2", "piggy-bank", "pilcrow", "pilcrow-left", "pilcrow-right", "pill", "pill-bottle", "pin", "pin-off", "pipette", "pizza", "plane", "plane-landing", "plane-takeoff", "play", "plug", "plug-2", "plug-zap", "plus", "pocket", "pocket-knife", "podcast", "pointer", "pointer-off", "popcorn", "popsicle", "pound-sterling", "power", "power-off", "presentation", "printer", "printer-check", "printer-x", "projector", "proportions", "puzzle", "pyramid", "qr-code", "quote", "rabbit", "radar", "radiation", "radical", "radio", "radio-receiver", "radio-tower", "radius", "rail-symbol", "rainbow", "rat", "ratio", "receipt", "receipt-cent", "receipt-euro", "receipt-indian-rupee", "receipt-japanese-yen", "receipt-pound-sterling", "receipt-russian-ruble", "receipt-swiss-franc", "receipt-text", "receipt-turkish-lira", "rectangle-circle", "rectangle-ellipsis", "rectangle-goggles", "rectangle-horizontal", "rectangle-vertical", "recycle", "redo", "redo-2", "redo-dot", "refresh-ccw", "refresh-ccw-dot", "refresh-cw", "refresh-cw-off", "refrigerator", "regex", "remove-formatting", "repeat", "repeat-1", "repeat-2", "replace", "replace-all", "reply", "reply-all", "rewind", "ribbon", "rocket", "rocking-chair", "roller-coaster", "rose", "rotate-3d", "rotate-ccw", "rotate-ccw-key", "rotate-ccw-square", "rotate-cw", "rotate-cw-square", "route", "route-off", "router", "rows-2", "rows-3", "rows-4", "rss", "ruler", "ruler-dimension-line", "russian-ruble", "sailboat", "salad", "sandwich", "satellite", "satellite-dish", "saudi-riyal", "save", "save-all", "save-off", "scale", "scale-3d", "scaling", "scan", "scan-barcode", "scan-eye", "scan-face", "scan-heart", "scan-line", "scan-qr-code", "scan-search", "scan-text", "school", "scissors", "scissors-line-dashed", "scooter", "screen-share", "screen-share-off", "scroll", "scroll-text", "search", "search-alert", "search-check", "search-code", "search-slash", "search-x", "section", "send", "send-horizontal", "send-to-back", "separator-horizontal", "separator-vertical", "server", "server-cog", "server-crash", "server-off", "settings", "settings-2", "shapes", "share", "share-2", "sheet", "shell", "shelving-unit", "shield", "shield-alert", "shield-ban", "shield-check", "shield-ellipsis", "shield-half", "shield-minus", "shield-off", "shield-plus", "shield-question-mark", "shield-user", "shield-x", "ship", "ship-wheel", "shirt", "shopping-bag", "shopping-basket", "shopping-cart", "shovel", "shower-head", "shredder", "shrimp", "shrink", "shrub", "shuffle", "sigma", "signal", "signal-high", "signal-low", "signal-medium", "signal-zero", "signature", "signpost", "signpost-big", "siren", "skip-back", "skip-forward", "skull", "slack", "slash", "slice", "sliders-horizontal", "sliders-vertical", "smartphone", "smartphone-charging", "smartphone-nfc", "smile", "smile-plus", "snail", "snowflake", "soap-dispenser-droplet", "sofa", "solar-panel", "soup", "space", "spade", "sparkle", "sparkles", "speaker", "speech", "spell-check", "spell-check-2", "spline", "spline-pointer", "split", "spool", "spotlight", "spray-can", "sprout", "square", "square-activity", "square-arrow-down", "square-arrow-down-left", "square-arrow-down-right", "square-arrow-left", "square-arrow-out-down-left", "square-arrow-out-down-right", "square-arrow-out-up-left", "square-arrow-out-up-right", "square-arrow-right", "square-arrow-right-enter", "square-arrow-right-exit", "square-arrow-up", "square-arrow-up-left", "square-arrow-up-right", "square-asterisk", "square-bottom-dashed-scissors", "square-centerline-dashed-horizontal", "square-centerline-dashed-vertical", "square-chart-gantt", "square-check", "square-check-big", "square-chevron-down", "square-chevron-left", "square-chevron-right", "square-chevron-up", "square-code", "square-dashed", "square-dashed-bottom", "square-dashed-bottom-code", "square-dashed-kanban", "square-dashed-mouse-pointer", "square-dashed-top-solid", "square-divide", "square-dot", "square-equal", "square-function", "square-kanban", "square-library", "square-m", "square-menu", "square-minus", "square-mouse-pointer", "square-parking", "square-parking-off", "square-pause", "square-pen", "square-percent", "square-pi", "square-pilcrow", "square-play", "square-plus", "square-power", "square-radical", "square-round-corner", "square-scissors", "square-sigma", "square-slash", "square-split-horizontal", "square-split-vertical", "square-square", "square-stack", "square-star", "square-stop", "square-terminal", "square-user", "square-user-round", "square-x", "squares-exclude", "squares-intersect", "squares-subtract", "squares-unite", "squircle", "squircle-dashed", "squirrel", "stamp", "star", "star-half", "star-off", "step-back", "step-forward", "stethoscope", "sticker", "sticky-note", "stone", "store", "stretch-horizontal", "stretch-vertical", "strikethrough", "subscript", "sun", "sun-dim", "sun-medium", "sun-moon", "sun-snow", "sunrise", "sunset", "superscript", "swatch-book", "swiss-franc", "switch-camera", "sword", "swords", "syringe", "table", "table-2", "table-cells-merge", "table-cells-split", "table-columns-split", "table-of-contents", "table-properties", "table-rows-split", "tablet", "tablet-smartphone", "tablets", "tag", "tags", "tally-1", "tally-2", "tally-3", "tally-4", "tally-5", "tangent", "target", "telescope", "tent", "tent-tree", "terminal", "test-tube", "test-tube-diagonal", "test-tubes", "text-align-center", "text-align-end", "text-align-justify", "text-align-start", "text-cursor", "text-cursor-input", "text-initial", "text-quote", "text-search", "text-select", "text-wrap", "theater", "thermometer", "thermometer-snowflake", "thermometer-sun", "thumbs-down", "thumbs-up", "ticket", "ticket-check", "ticket-minus", "ticket-percent", "ticket-plus", "ticket-slash", "ticket-x", "tickets", "tickets-plane", "timer", "timer-off", "timer-reset", "toggle-left", "toggle-right", "toilet", "tool-case", "toolbox", "tornado", "torus", "touchpad", "touchpad-off", "towel-rack", "tower-control", "toy-brick", "tractor", "traffic-cone", "train-front", "train-front-tunnel", "train-track", "tram-front", "transgender", "trash", "trash-2", "tree-deciduous", "tree-palm", "tree-pine", "trees", "trello", "trending-down", "trending-up", "trending-up-down", "triangle", "triangle-alert", "triangle-dashed", "triangle-right", "trophy", "truck", "truck-electric", "turkish-lira", "turntable", "turtle", "tv", "tv-minimal", "tv-minimal-play", "twitch", "twitter", "type", "type-outline", "umbrella", "umbrella-off", "underline", "undo", "undo-2", "undo-dot", "unfold-horizontal", "unfold-vertical", "ungroup", "university", "unlink", "unlink-2", "unplug", "upload", "usb", "user", "user-check", "user-cog", "user-key", "user-lock", "user-minus", "user-pen", "user-plus", "user-round", "user-round-check", "user-round-cog", "user-round-key", "user-round-minus", "user-round-pen", "user-round-plus", "user-round-search", "user-round-x", "user-search", "user-star", "user-x", "users", "users-round", "utensils", "utensils-crossed", "utility-pole", "van", "variable", "vault", "vector-square", "vegan", "venetian-mask", "venus", "venus-and-mars", "vibrate", "vibrate-off", "video", "video-off", "videotape", "view", "voicemail", "volleyball", "volume", "volume-1", "volume-2", "volume-off", "volume-x", "vote", "wallet", "wallet-cards", "wallet-minimal", "wallpaper", "wand", "wand-sparkles", "warehouse", "washing-machine", "watch", "waves", "waves-arrow-down", "waves-arrow-up", "waves-ladder", "waypoints", "webcam", "webhook", "webhook-off", "weight", "weight-tilde", "wheat", "wheat-off", "whole-word", "wifi", "wifi-cog", "wifi-high", "wifi-low", "wifi-off", "wifi-pen", "wifi-sync", "wifi-zero", "wind", "wind-arrow-down", "wine", "wine-off", "workflow", "worm", "wrench", "x", "x-line-top", "youtube", "zap", "zap-off", "zodiac-aquarius", "zodiac-aries", "zodiac-cancer", "zodiac-capricorn", "zodiac-gemini", "zodiac-leo", "zodiac-libra", "zodiac-ophiuchus", "zodiac-pisces", "zodiac-sagittarius", "zodiac-scorpio", "zodiac-taurus", "zodiac-virgo", "zoom-in", "zoom-out"];

</script>

<svelte:window on:keydown={onKeyDown} on:mousedown={(e) => { if (e.button === 0 && ctxMenu) closeCtx(); }} />

<div class="wt" data-workspace-tree>
  <div class="wt-header">
    <span class="wt-title">{tr('workspaceTree.title')}</span>
    <div class="wt-header-actions">
      <button class="ti-btn" on:click={() => openCreateWorkspace('')} title={tr('workspaceTree.newDeal')} aria-label={tr('workspaceTree.newDeal')}><Icon name="space" size={14} /></button>
      <button class="ti-btn" on:click={() => openCreateFolder('')} title={tr('workspaceTree.newFolder')} aria-label={tr('workspaceTree.newFolder')}><Icon name="folder" size={14} /></button>
    </div>
  </div>

  <div class="wt-list" role="tree" aria-label={tr('workspaceTree.title')}
    on:dragover={onRootDragOver} on:dragleave={onRootDragLeave} on:drop={onRootDrop}
    on:dragend={resetDragState}
  >
    {#if loading}
      <div class="wt-status">{tr('common.loading')}</div>
    {:else if error}
      <div class="wt-status wt-error">{error}</div>
    {:else if !tree.roots || tree.roots.length === 0}
      <div class="wt-empty">
        <p>{tr('workspaceTree.emptyTitle')}</p>
        <p class="wt-empty-hint">{tr('workspaceTree.emptyHint')}</p>
      </div>
    {:else}
      {#each tree.roots as node (node.key)}
        <TreeNode {node} depth={0} {expandedIds} {activeWid} {focusedKey}
          on:toggle={(e) => toggleExpand(e.detail.key)}
          on:select={(e) => selectWorkspace(e.detail.id)}
          on:nav={handleNav}
          on:rename={handleRename}
          on:trash={handleTrash}
          on:contextmenu={onCtx}
          on:drop={onNodeDrop}
          on:dragstart={onNodeDragStart}
          on:createFolder={(e) => openCreateFolder(e.detail)}
          on:createWorkspace={(e) => openCreateWorkspace(e.detail)}
        />
      {/each}
    {/if}
    {#if dragOverRoot}
      <div class="wt-root-drop">Переместить в корень</div>
    {/if}
  </div>
</div>

<!-- Context Menu -->
{#if ctxMenu}
  <div class="vt-ctx" style="left:{ctxMenu.x}px;top:{ctxMenu.y}px" on:click|stopPropagation on:mousedown|stopPropagation>
    {#if ctxMenu.kind === 'folder'}
      <button class="vt-ctx-i" on:click={() => { const i = ctxMenu.id; closeCtx(); openCreateWorkspace(i); }}>{tr('workspaceTree.newDeal')}</button>
      <button class="vt-ctx-i" on:click={() => { const i = ctxMenu.id; closeCtx(); openCreateFolder(i); }}>{tr('workspaceTree.newFolder')}</button>
      <div class="vt-ctx-s" />
      <button class="vt-ctx-i" on:click={() => { const {id: i, name: n} = ctxMenu; closeCtx(); openEditFolder(i, n); }}>{tr('workspaceTree.editFolder')}</button>
      <button class="vt-ctx-i vt-ctx-d" on:click={() => { const {id: i, name: n} = ctxMenu; closeCtx(); openTrash('folder', i, n); }}>{tr('workspaceTree.trashFolder')}</button>
    {:else}
      <button class="vt-ctx-i" on:click={() => { const i = ctxMenu.id; closeCtx(); selectWorkspace(i); }}>{tr('workspaceTree.open')}</button>
      <button class="vt-ctx-i" on:click={() => { const {id: i, name: n} = ctxMenu; closeCtx(); openRename('workspace', i, n); }}>{tr('workspaceTree.renameDeal')}</button>
      <button class="vt-ctx-i vt-ctx-d" on:click={() => { const {id: i, name: n} = ctxMenu; closeCtx(); openTrash('workspace', i, n); }}>{tr('workspaceTree.trashDeal')}</button>
    {/if}
  </div>
{/if}

<!-- Modals -->
<Modal title={folderEditorView === 'icon-picker' ? tr('workspaceTree.iconPicker') : tr('workspaceTree.newFolder')} show={modal?.type === 'create-folder'} on:close={folderEditorView === 'icon-picker' ? () => { folderEditorView = 'form'; } : closeModal}>
  {#if folderEditorView === 'icon-picker'}
    <label class="vt-field"><span>{tr('workspaceTree.iconSearch')} ({filteredIcons.length} / {LUCIDE_ICONS.length})</span><input class="vt-input" type="text" bind:value={iconSearch} placeholder={tr('workspaceTree.iconSearch') + '...'} autofocus /></label>
    <div class="vt-icon-grid">
      <button type="button" class="vt-icon-item" class:vt-icon-selected={!folderIconId} on:click={() => selectFolderIcon('')}><Icon name="folder" size={20} /><span>{tr('workspaceTree.defaultIcon')}</span></button>
      {#each filteredIcons as icon}
        <button type="button" class="vt-icon-item" class:vt-icon-selected={folderIconId === icon} on:click={() => selectFolderIcon(icon)}><Icon name={icon} size={20} /><span>{icon}</span></button>
      {/each}
    </div>
  {:else}
    <label class="vt-field"><span>{tr('workspaceTree.location')}</span><Select options={flatFolders(tree.roots).map(f => ({ value: f.id, label: f.path }))} placeholder={tr('workspaceTree.root')} bind:value={formParentId} labelKey="label" valueKey="value" /></label>
    <label class="vt-field"><span>{tr('workspaceTree.folderName')}</span><input class="vt-input" type="text" bind:value={formName} placeholder={tr('workspaceTree.folderNamePlaceholder')} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doCreateFolder()} /></label>
    <label class="vt-field"><span>{tr('workspaceTree.appearance')}</span>
      <div class="vt-appearance-row">
        <button type="button" class="vt-appearance-btn" on:click={openIconPicker} disabled={formBusy}>
          <Icon name={folderIconId || 'folder'} size={18} style="color:{folderColor || ''}" />
          <span>{folderIconId || tr('workspaceTree.defaultIcon')}</span>
        </button>
        <div class="vt-color-row">
          <input type="color" bind:value={folderColor} disabled={formBusy} class="vt-color-native" />
          <input class="vt-input vt-color-hex" type="text" bind:value={folderColor} placeholder="#RRGGBB" disabled={formBusy} />
          <button type="button" class="vt-btn" on:click={resetFolderColor} disabled={formBusy}>{tr('workspaceTree.resetColor')}</button>
        </div>
      </div>
    </label>
    {#if formError}<p class="vt-ferr">{formError}</p>{/if}
  {/if}
  <svelte:fragment slot="actions">
    {#if folderEditorView === 'icon-picker'}
      <button type="button" class="vt-btn" on:click={() => { folderEditorView = 'form'; }}>{tr('common.back')}</button>
    {:else}
      <button type="button" class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button>
      <button class="vt-btn-p" on:click={doCreateFolder} disabled={formBusy}>{tr('common.create')}</button>
    {/if}
  </svelte:fragment>
</Modal>


<Modal title={folderEditorView === 'icon-picker' ? tr('workspaceTree.iconPicker') : tr('workspaceTree.editFolder')} show={modal?.type === 'edit-folder'} on:close={folderEditorView === 'icon-picker' ? () => { folderEditorView = 'form'; } : closeModal}>
  {#if folderEditorView === 'icon-picker'}
    <label class="vt-field"><span>{tr('workspaceTree.iconSearch')} ({filteredIcons.length} / {LUCIDE_ICONS.length})</span><input class="vt-input" type="text" bind:value={iconSearch} placeholder={tr('workspaceTree.iconSearch') + '...'} autofocus /></label>
    <div class="vt-icon-grid">
      <button type="button" class="vt-icon-item" class:vt-icon-selected={!folderIconId} on:click={() => selectFolderIcon('')}><Icon name="folder" size={20} /><span>{tr('workspaceTree.defaultIcon')}</span></button>
      {#each filteredIcons as icon}
        <button type="button" class="vt-icon-item" class:vt-icon-selected={folderIconId === icon} on:click={() => selectFolderIcon(icon)}><Icon name={icon} size={20} /><span>{icon}</span></button>
      {/each}
    </div>
  {:else}
    <label class="vt-field"><span>{tr('workspaceTree.folderName')}</span><input class="vt-input" type="text" bind:value={formName} placeholder={tr('workspaceTree.folderNamePlaceholder')} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doEditFolder()} /></label>
    <label class="vt-field"><span>{tr('workspaceTree.appearance')}</span>
      <div class="vt-appearance-row">
        <button type="button" class="vt-appearance-btn" on:click={openIconPicker} disabled={formBusy}>
          <Icon name={folderIconId || 'folder'} size={18} style="color:{folderColor || ''}" />
          <span>{folderIconId || tr('workspaceTree.defaultIcon')}</span>
        </button>
        <div class="vt-color-row">
          <input type="color" bind:value={folderColor} disabled={formBusy} class="vt-color-native" />
          <input class="vt-input vt-color-hex" type="text" bind:value={folderColor} placeholder="#RRGGBB" disabled={formBusy} />
          <button type="button" class="vt-btn" on:click={resetFolderColor} disabled={formBusy}>{tr('workspaceTree.resetColor')}</button>
        </div>
      </div>
    </label>
    {#if formError}<p class="vt-ferr">{formError}</p>{/if}
  {/if}
  <svelte:fragment slot="actions">
    {#if folderEditorView === 'icon-picker'}
      <button type="button" class="vt-btn" on:click={() => { folderEditorView = 'form'; }}>{tr('common.back')}</button>
    {:else}
      <button type="button" class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button>
      <button class="vt-btn-p" on:click={doEditFolder} disabled={formBusy}>{tr('common.save')}</button>
    {/if}
  </svelte:fragment>
</Modal>

<Modal title={tr('workspaceTree.newDeal')} show={modal?.type === 'create-workspace'} on:close={closeModal} wide>
  <label class="vt-field"><span>{tr('workspaceTree.location')}</span><Select options={flatFolders(tree.roots).map(f => ({ value: f.id, label: f.path }))} placeholder={tr('workspaceTree.root')} bind:value={formParentId} labelKey="label" valueKey="value" /></label>
  <label class="vt-field"><span>{tr('workspaceTree.name')}</span><input class="vt-input" type="text" bind:value={formName} placeholder={tr('workspaceTree.namePlaceholder')} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doCreateWorkspace()} /></label>
  <label class="vt-field"><span>{tr('workspaceTree.template')}</span><Select options={templates} bind:value={formTemplateId} labelKey="name" valueKey="id" /></label>
  {@const st = templates.find(t => t.id === formTemplateId)}
  {#if st}
    <div class="vt-template-info">
      {#if st.description}<p class="vt-template-desc">{st.description}</p>{/if}
      {#if st.workspaceTools?.length}
        <div class="vt-template-badges">
          {#each st.workspaceTools as pt}
            <span class="vt-badge vt-tool-badge" class:vt-tool-unavailable={!pluginAvailable(pt)} title={pt}>{pluginDisplayName(pt)}</span>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
  {#if formError}<p class="vt-ferr">{formError}</p>{/if}
  <svelte:fragment slot="actions"><button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button><button class="vt-btn-p" on:click={doCreateWorkspace} disabled={formBusy}>{tr('common.create')}</button></svelte:fragment>
</Modal>

<Modal title={tr('workspaceTree.rename')} show={modal?.type === 'rename'} on:close={closeModal}>
  <label class="vt-field"><span>{tr('workspaceTree.newName')}</span><input class="vt-input" type="text" bind:value={formName} disabled={formBusy} on:keydown={(e) => e.key === 'Enter' && doRename()} /></label>
  {#if formError}<p class="vt-ferr">{formError}</p>{/if}
  <svelte:fragment slot="actions"><button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button><button class="vt-btn-p" on:click={doRename} disabled={formBusy}>{tr('common.save')}</button></svelte:fragment>
</Modal>

<Modal title={(modal?.kind === 'folder' ? tr('workspaceTree.trashFolder') : tr('workspaceTree.trashDeal')) + (modal?.name ? ' «' + modal.name + '»?' : '?')} show={modal?.type === 'trash'} on:close={closeModal}>
  <p class="vt-trash-desc">
    {#if modal?.kind === 'folder'}
      {@const c = subtreeCounts(modal.id)}
      {tr('workspaceTree.trashFolderDesc')}<br />
      {tr('workspaceTree.contains')}: {c.folders} {tr('workspaceTree.nestedFolders')}, {c.workspaces} {tr('workspaceTree.title')}
    {:else}
      {tr('workspaceTree.trashDealDesc')}
    {/if}
  </p>
  <svelte:fragment slot="actions"><button class="vt-btn" on:click={closeModal} disabled={formBusy}>{tr('common.cancel')}</button><button class="vt-btn-d" on:click={doTrash} disabled={formBusy}>{tr('workspaceTree.toTrash')}</button></svelte:fragment>
</Modal>

<style>
  .wt { display: flex; flex-direction: column; flex: 1; overflow: hidden; }
  .wt-header { display: flex; align-items: center; justify-content: space-between; padding: 0.7rem 0.6rem 0.35rem; border-bottom: 1px solid var(--vt-color-border); flex-shrink: 0; }
  .wt-title { color: var(--vt-color-text-muted); font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em; font-weight: 600; }
  .wt-header-actions { display: flex; gap: 0.2rem; }
  .wt-list { min-height: 0; overflow-y: auto; padding: 0.2rem 0.4rem; flex: 1; position: relative; }
  .wt-status { padding: 0.5rem; font-size: 0.78rem; color: var(--vt-color-text-muted); }
  .wt-error { color: var(--vt-color-danger); }
  .wt-empty { padding: 1rem 0.5rem; text-align: center; color: var(--vt-color-text-muted); font-size: 0.8rem; }
  .wt-empty-hint { font-size: 0.72rem; opacity: 0.7; }

  .wt-root-drop { margin: 0.2rem 0.4rem; padding: 0.4rem; border: 1px dashed var(--vt-color-accent); border-radius: var(--vt-radius-sm); text-align: center; color: var(--vt-color-accent); font-size: 0.75rem; background: var(--vt-color-accent-muted); }

  .ti-btn { width: 1.6rem; height: 1.6rem; min-height: 0; padding: 0; border: 1px solid transparent; background: transparent; color: var(--vt-color-text-muted); cursor: pointer; border-radius: var(--vt-radius-sm); display: inline-flex; align-items: center; justify-content: center; }
  .ti-btn:hover { color: var(--vt-color-accent); background: var(--vt-color-accent-muted); border-color: rgba(78,204,163,0.25); }

  .vt-btn { min-height: 1.8rem; background: transparent; border: 1px solid var(--vt-color-border-strong); color: var(--vt-color-text-secondary); cursor: pointer; font-size: 0.78rem; padding: 0.3rem 0.6rem; border-radius: var(--vt-radius-sm); }
  .vt-btn:hover:not(:disabled) { color: var(--vt-color-text-primary); border-color: var(--vt-color-text-muted); }
  .vt-btn-p { min-height: 1.8rem; background: var(--vt-color-accent); color: #101827; border: none; padding: 0.3rem 0.7rem; border-radius: var(--vt-radius-sm); cursor: pointer; font-size: 0.78rem; font-weight: 600; }
  .vt-btn-p:hover:not(:disabled) { background: #3dbb92; }
  .vt-btn-d { min-height: 1.8rem; background: var(--vt-color-danger); color: #fff; border: none; padding: 0.3rem 0.7rem; border-radius: var(--vt-radius-sm); cursor: pointer; font-size: 0.78rem; font-weight: 600; }
  .vt-btn-d:hover:not(:disabled) { background: #d63851; }
  .vt-btn:disabled, .vt-btn-p:disabled, .vt-btn-d:disabled { opacity: 0.5; cursor: not-allowed; }

  .vt-field { display: grid; gap: 0.35rem; color: var(--vt-color-text-muted); font-size: 0.75rem; }
  .vt-input { width: 100%; min-height: 2rem; box-sizing: border-box; border: 1px solid var(--vt-color-border-strong); border-radius: var(--vt-radius-sm); background: #0f1424; color: var(--vt-color-text-primary); padding: 0.35rem 0.5rem; font: inherit; font-size: 0.84rem; }
  .vt-input:focus { outline: none; border-color: var(--vt-color-accent); box-shadow: var(--vt-focus-ring); }
  .vt-ferr { margin: 0; color: var(--vt-color-danger); font-size: 0.78rem; line-height: 1.4; }

  .vt-trash-desc { color: var(--vt-color-text-secondary); font-size: 0.84rem; margin: 0; line-height: 1.5; }

  .vt-ctx { position: fixed; z-index: 10001; min-width: 10rem; background: var(--vt-color-surface); border: 1px solid var(--vt-color-border-strong); border-radius: var(--vt-radius-md); box-shadow: 0 8px 24px rgba(0,0,0,0.3); padding: 0.25rem; }
  .vt-ctx-i { display: block; width: 100%; text-align: left; padding: 0.3rem 0.6rem; background: none; border: none; color: var(--vt-color-text-secondary); font-size: 0.78rem; cursor: pointer; border-radius: var(--vt-radius-sm); }
  .vt-ctx-i:hover { background: var(--vt-color-surface-hover); color: var(--vt-color-text-primary); }
  .vt-ctx-s { height: 1px; background: var(--vt-color-border); margin: 0.2rem 0.3rem; }
  .vt-ctx-d { color: var(--vt-color-danger); }
  .vt-ctx-d:hover { background: var(--vt-color-danger-muted); }
  .vt-template-info { margin: var(--vt-space-2) 0; }
  .vt-template-desc { color: var(--vt-color-text-secondary); font-size: 0.8rem; line-height: 1.4; margin-bottom: var(--vt-space-2); }
  .vt-template-badges { display: flex; flex-wrap: wrap; gap: 0.35rem; }
  .vt-tool-badge { font-size: 0.72rem; padding: 0.15rem 0.5rem; }
  .vt-tool-unavailable { opacity: 0.45; border-color: var(--vt-color-warning); color: var(--vt-color-warning); }

  .vt-appearance-row { display: flex; gap: 8px; }
  .vt-appearance-btn { display: inline-flex; align-items: center; gap: 6px; min-height: 2rem; padding: 4px 10px; border: 1px solid var(--vt-color-border); border-radius: var(--vt-radius-sm); background: var(--vt-color-surface); color: var(--vt-color-text-secondary); cursor: pointer; font-size: .78rem; }
  .vt-appearance-btn:hover { border-color: var(--vt-color-accent); }
  .vt-color-swatch { width: 16px; height: 16px; border-radius: 50%; border: 1px solid var(--vt-color-border); }


  .vt-appearance-row { display: flex; flex-direction: column; gap: 8px; }
  .vt-appearance-btn { display: inline-flex; align-items: center; gap: 6px; min-height: 2rem; padding: 4px 10px; border: 1px solid var(--vt-color-border); border-radius: var(--vt-radius-sm); background: var(--vt-color-surface); color: var(--vt-color-text-secondary); cursor: pointer; font-size: .78rem; }
  .vt-appearance-btn:hover { border-color: var(--vt-color-accent); }
  .vt-color-row { display: flex; align-items: center; gap: 6px; }
  .vt-color-native { width: 2rem; height: 2rem; cursor: pointer; border: 1px solid var(--vt-color-border); border-radius: var(--vt-radius-sm); background: none; padding: 2px; }
  .vt-color-hex { width: 7rem; }
  .vt-icon-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(4.5rem, 1fr)); gap: 4px; margin-top: 8px; max-height: 40vh; overflow-y: auto; }
  .vt-icon-item { display: flex; flex-direction: column; align-items: center; gap: 2px; padding: 6px 4px; border: 1px solid transparent; border-radius: var(--vt-radius-sm); background: transparent; color: var(--vt-color-text-secondary); cursor: pointer; font-size: .65rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .vt-icon-item:hover { border-color: var(--vt-color-accent); background: var(--vt-color-accent-muted); }
  .vt-icon-selected { border-color: var(--vt-color-accent); background: var(--vt-color-accent-muted); color: var(--vt-color-accent); }
</style>
