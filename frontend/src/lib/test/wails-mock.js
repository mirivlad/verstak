/**
 * Wails Mock Bridge — эмулирует window['go']['api']['App'] для тестового окружения.
 *
 * Каждый метод возвращает Promise с данными, совместимыми с Wails-контрактом.
 * Состояние мутабельно — тесты могут менять его между сценариями.
 */
import defaultEditorSource from '../../../../../verstak-official-plugins/plugins/default-editor/frontend/src/index.js?raw';
import filesSource from '../../../../../verstak-official-plugins/plugins/files/frontend/src/index.js?raw';
import filePreviewSource from '../../../../../verstak-official-plugins/plugins/file-preview/frontend/src/index.js?raw';
import filesManifest from '../../../../../verstak-official-plugins/plugins/files/plugin.json';
import platformTestManifest from '../../../../../verstak-official-plugins/plugins/platform-test/plugin.json';
import defaultEditorManifest from '../../../../../verstak-official-plugins/plugins/default-editor/plugin.json';
import filePreviewManifest from '../../../../../verstak-official-plugins/plugins/file-preview/plugin.json';
import trashManifest from '../../../../../verstak-official-plugins/plugins/trash/plugin.json';
import notesManifest from '../../../../../verstak-official-plugins/plugins/notes/plugin.json';
import syncManifest from '../../../../../verstak-official-plugins/plugins/sync/plugin.json';
import activityManifest from '../../../../../verstak-official-plugins/plugins/activity/plugin.json';
import journalManifest from '../../../../../verstak-official-plugins/plugins/journal/plugin.json';
import browserInboxManifest from '../../../../../verstak-official-plugins/plugins/browser-inbox/plugin.json';
import todoManifest from '../../../../../verstak-official-plugins/plugins/todo/plugin.json';
import secretsManifest from '../../../../../verstak-official-plugins/plugins/secrets/plugin.json';
import importManifest from '../../../../../verstak-official-plugins/plugins/import/plugin.json';
import searchManifest from '../../../../../verstak-official-plugins/plugins/search/plugin.json';
import notesSource from '../../../../../verstak-official-plugins/plugins/notes/frontend/src/index.js?raw';
import browserInboxSource from '../../../../../verstak-official-plugins/plugins/browser-inbox/frontend/src/index.js?raw';
import secretsSource from '../../../../../verstak-official-plugins/plugins/secrets/frontend/src/index.js?raw';
import activitySource from '../../../../../verstak-official-plugins/plugins/activity/frontend/src/index.js?raw';
import todoSource from '../../../../../verstak-official-plugins/plugins/todo/frontend/src/index.js?raw';
import journalSource from '../../../../../verstak-official-plugins/plugins/journal/frontend/src/index.js?raw';
import notesEnCatalog from '../../../../../verstak-official-plugins/plugins/notes/locales/en.json';
import notesRuCatalog from '../../../../../verstak-official-plugins/plugins/notes/locales/ru.json';
import activityEnCatalog from '../../../../../verstak-official-plugins/plugins/activity/locales/en.json';
import activityRuCatalog from '../../../../../verstak-official-plugins/plugins/activity/locales/ru.json';
import browserEnCatalog from '../../../../../verstak-official-plugins/plugins/browser-inbox/locales/en.json';
import browserRuCatalog from '../../../../../verstak-official-plugins/plugins/browser-inbox/locales/ru.json';
import filePreviewEnCatalog from '../../../../../verstak-official-plugins/plugins/file-preview/locales/en.json';
import filePreviewRuCatalog from '../../../../../verstak-official-plugins/plugins/file-preview/locales/ru.json';
import journalEnCatalog from '../../../../../verstak-official-plugins/plugins/journal/locales/en.json';
import journalRuCatalog from '../../../../../verstak-official-plugins/plugins/journal/locales/ru.json';
import todoEnCatalog from '../../../../../verstak-official-plugins/plugins/todo/locales/en.json';
import todoRuCatalog from '../../../../../verstak-official-plugins/plugins/todo/locales/ru.json';
import importSource from '../../../../../verstak-official-plugins/plugins/import/frontend/dist/index.js?raw';
import importStyle from '../../../../../verstak-official-plugins/plugins/import/frontend/dist/style.css?raw';

(function () {
  if (window.__wailsMockReady) return;

  // ── Mutable state ──────────────────────────────────────────────────
  function makePluginState(manifest, slug) {
    return {
      status: 'loaded',
      enabled: true,
      manifest: JSON.parse(JSON.stringify(manifest)),
      rootPath: '/tmp/verstak-test/plugins/' + slug,
      error: ''
    };
  }

  var officialPluginFixtures = [
    [platformTestManifest, 'platform-test'],
    [defaultEditorManifest, 'default-editor'],
    [filePreviewManifest, 'file-preview'],
    [filesManifest, 'files'],
    [trashManifest, 'trash'],
    [notesManifest, 'notes'],
    [syncManifest, 'sync'],
    [activityManifest, 'activity'],
    [journalManifest, 'journal'],
    [browserInboxManifest, 'browser-inbox'],
    [todoManifest, 'todo'],
    [secretsManifest, 'secrets'],
    [importManifest, 'import'],
    [searchManifest, 'search']
  ];

  function makeDefaultPluginStates() {
    var states = {};
    officialPluginFixtures.forEach(function (fixture) {
      states[fixture[0].id] = makePluginState(fixture[0], fixture[1]);
    });
    return states;
  }

  function makeDefaultVaultPluginState() {
    return {
      enabledPlugins: officialPluginFixtures.map(function (fixture) { return fixture[0].id; }),
      disabledPlugins: [],
      desiredPlugins: officialPluginFixtures.map(function (fixture) {
        return { id: fixture[0].id, version: fixture[0].version, source: 'official' };
      })
    };
  }

  var pluginStates = makeDefaultPluginStates();

  var realPluginCatalogs = {
    'verstak.notes': { en: notesEnCatalog, ru: notesRuCatalog },
    'verstak.activity': { en: activityEnCatalog, ru: activityRuCatalog },
    'verstak.browser-inbox': { en: browserEnCatalog, ru: browserRuCatalog },
    'verstak.file-preview': { en: filePreviewEnCatalog, ru: filePreviewRuCatalog },
    'verstak.journal': { en: journalEnCatalog, ru: journalRuCatalog },
    'verstak.todo': { en: todoEnCatalog, ru: todoRuCatalog }
  };

  var russianPluginNames = {
    'verstak.platform-test': 'Тест платформы',
    'verstak.default-editor': 'Стандартный редактор',
    'verstak.files': 'Файлы',
    'verstak.notes': 'Заметки',
    'verstak.sync': 'Синхронизация',
    'verstak.activity': 'Активность',
    'verstak.journal': 'Журнал',
    'verstak.browser-inbox': 'Браузер',
    'verstak.search': 'Поиск',
    'verstak.trash': 'Корзина',
    'verstak.todo': 'Задачи',
    'verstak.secrets': 'Секреты',
    'verstak.import': 'Импорт'
  };
  var russianContributionLabels = {
    'verstak.platform-test.diagnostics': 'Диагностика платформы',
    'verstak.platform-test.run-tests': 'Запустить тесты платформы',
    'verstak.platform-test.show-version': 'Показать сведения о версии',
    'verstak.platform-test.sidebar': 'Тест платформы',
    'verstak.platform-test.status': '[OK] Все тесты пройдены',
    'verstak.platform-test.settings': 'Настройки теста платформы',
    'verstak.platform-test.markdown-diagnostic': 'Диагностика Markdown платформы',
    'verstak.search.searchVaultText': 'Искать текст в хранилище',
    'verstak.search.vault-text': 'Поиск по тексту хранилища'
  };
  function mockPluginCatalog(pluginId, locale) {
    var realCatalog = realPluginCatalogs[pluginId] && realPluginCatalogs[pluginId][locale];
    if (realCatalog) return Object.assign({}, realCatalog);
    var state = pluginStates[pluginId];
    if (!state || !state.manifest) return {};
    var manifest = state.manifest;
    var translatedName = locale === 'ru' ? (russianPluginNames[pluginId] || manifest.name) : manifest.name;
    var catalog = {
      'manifest.name': translatedName,
      'manifest.description': manifest.description || ''
    };
    var contributionFields = {
      views: 'title', commands: 'title', settingsPanels: 'title', sidebarItems: 'title',
      fileActions: 'label', noteActions: 'label', contextMenuEntries: 'label',
      searchProviders: 'label', worklogProviders: 'label', overviewProviders: 'label', statusBarItems: 'label', openProviders: 'title', workspaceItems: 'title'
    };
    Object.keys(contributionFields).forEach(function (point) {
      var field = contributionFields[point];
      ((manifest.contributes || {})[point] || []).forEach(function (item) {
        catalog['contributions.' + point + '.' + item.id + '.' + field] = locale === 'ru'
          ? (russianContributionLabels[item.id] || translatedName)
          : item[field];
      });
    });
    return catalog;
  }

  var vaultStatus = { status: 'open', path: '/tmp/verstak-test/vault', vaultId: 'test-vault-001' };
  var diagnosticsReports = [];
  var vaultPluginState = makeDefaultVaultPluginState();
  var appSettings = {
    currentVaultPath: '/tmp/verstak-test/vault',
    recentVaults: [],
    language: localStorage.getItem('verstak-test-language') || 'system',
    sidebarWidth: Number(localStorage.getItem('verstak-test-sidebar-width') || 220),
    expandedFolderIds: [],
    settingsSection: ''
  };
  var workbenchPreferences = {};
  var openedResources = [];
  var pluginSettings = {
    'verstak.platform-test': { savedText: 'initial value' }
  };
  var pluginNotifications = {};
  var pluginData = {};
  var folderAppearances = {};
  var secretRecords = makeDefaultSecretRecords();
  var vaultFiles = makeDefaultVaultFiles();
  var externalOpens = [];
  var trashEntries = [];
  var trashPayloads = {};
  window.__wailsMockExternalOpens = [];
  var workspaceTree = makeDefaultWorkspaceTree();
  var workspaceTreeV2Override = null;
  var treePlacementRequests = [];
  var treePlacementError = '';
  var workspaceMetadata = makeDefaultWorkspaceMetadata();
  var reloadResponseMode = 'tuple';
  var listVaultFilesResponseMode = 'tuple';
  var syncState = makeDefaultSyncState();
  var readTextDelay = 0;
  var importSessions = {};
  var importSequence = 0;
  var importRunCounts = { dokuwiki: 0, obsidian: 0 };

  // ── Helpers ────────────────────────────────────────────────────────
  function makeDefaultWorkspaceTree() {
    return {
      status: 'initialized',
      currentNodeId: 'Project',
      nodes: [
        { id: 'Project', parentId: '', type: 'space', title: 'Project', name: 'Project', rootPath: 'Project', status: 'active', order: 1 },
        { id: 'Test', parentId: '', type: 'space', title: 'Test', name: 'Test', rootPath: 'Test', status: 'active', order: 2 }
      ]
    };
  }

  function cloneWorkspaceTree() {
    return {
      status: workspaceTree.status,
      currentNodeId: workspaceTree.currentNodeId,
      nodes: workspaceTree.nodes.map(function (n) { return Object.assign({}, n); })
    };
  }

  function listWorkspacesFromTree() {
    return workspaceTree.nodes
      .filter(function (n) { return !n.parentId; })
      .map(function (n) { return { name: n.name || n.id, rootPath: n.rootPath || n.name || n.id }; });
  }

  function makeWorkspaceNode(name, order) {
    return { id: name, parentId: '', type: 'space', title: name, name: name, rootPath: name, status: 'active', order: order };
  }

  function makeWorkspaceNodeV2(name, order) {
    var wsid = 'ws-' + Math.random().toString(36).slice(2, 10);
    return { id: name, workspaceId: wsid, name: name, rootPath: name, order: order };
  }

  function workspaceTreeV2Snapshot() {
    if (workspaceTreeV2Override) return cloneJson(workspaceTreeV2Override);
    var roots = workspaceTree.nodes.map(function (n, i) {
      return {
        key: 'workspace:' + (n.workspaceId || n.id),
        kind: 'workspace',
        id: n.workspaceId || n.id,
        name: n.name,
        path: n.rootPath || n.name,
        children: []
      };
    });
    var current = roots.find(function (node) { return node.path === workspaceTree.currentNodeId || node.name === workspaceTree.currentNodeId; });
    return { roots: roots, currentWorkspaceId: current ? current.id : '', revision: 1, warnings: [] };
  }

  function applyAppSettingsPatch(patch) {
  appSettings = Object.assign({}, appSettings, patch || {});
  if (patch && patch.language) localStorage.setItem('verstak-test-language', patch.language);
  if (patch && patch.sidebarWidth) localStorage.setItem('verstak-test-sidebar-width', String(patch.sidebarWidth));
}

function cloneJson(value) {
    return JSON.parse(JSON.stringify(value));
  }

  function makeDefaultSecretRecords() {
    return [
      {
        id: 'first.secret',
        title: 'First secret',
        username: 'first-user',
        value: 'first-value',
        scope: { kind: 'global' },
        updatedAt: '2026-07-14T00:00:00Z'
      },
      {
        id: 'target.secret',
        title: 'Target secret',
        username: 'target-user',
        value: 'target-value',
        scope: { kind: 'global' },
        updatedAt: '2026-07-14T00:00:00Z'
      }
    ];
  }

  function builtInWorkspaceTemplates() {
    return [
      {
        id: 'default',
        name: 'General',
        description: 'Everyday Deal with notes, files, journal, activity, and browser captures.',
        version: 2,
        workspaceTools: ['verstak.notes', 'verstak.files', 'verstak.journal', 'verstak.activity', 'verstak.browser-inbox'],
        folders: ['Notes', 'Files'],
        features: { files: true, notes: true, activity: true, journal: true, 'browser-inbox': true },
      },
      {
        id: 'project',
        name: 'Project',
        description: 'Project planning with todos, journal, activity, and browser captures.',
        version: 1,
        workspaceTools: ['verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.journal', 'verstak.activity', 'verstak.browser-inbox'],
        folders: ['Notes', 'Files'],
        features: { files: true, notes: true, todo: true, journal: true, activity: true, 'browser-inbox': true },
      },
      {
        id: 'writing',
        name: 'Writing',
        description: 'Focused notes, files, and journal Deal for documentation and writing.',
        version: 1,
        workspaceTools: ['verstak.notes', 'verstak.files', 'verstak.journal'],
        folders: ['Notes', 'Files'],
        features: { files: true, notes: true, journal: true },
      },
      {
        id: 'admin',
        name: 'Admin',
        description: 'Infrastructure Deal with secrets, todos, and journal.',
        version: 1,
        workspaceTools: ['verstak.notes', 'verstak.files', 'verstak.secrets', 'verstak.todo', 'verstak.journal'],
        folders: ['Notes', 'Files', 'Secrets'],
        features: { files: true, notes: true, secrets: true, todo: true, journal: true },
      },
      {
        id: 'minimal',
        name: 'Minimal',
        description: 'Only notes and files for a lightweight Deal.',
        version: 1,
        workspaceTools: ['verstak.notes', 'verstak.files'],
        folders: ['Notes', 'Files'],
        features: { files: true, notes: true },
      },
    ];
  }

  function workspaceTemplateByID(templateID) {
    var id = String(templateID || 'default');
    return builtInWorkspaceTemplates().find(function (template) { return template.id === id; }) || null;
  }

  function metadataForTemplate(name, template) {
    var now = new Date().toISOString();
    var folders = { notes: 'Notes', files: 'Files' };
    if (template.features.secrets) folders.secrets = 'Secrets';
    return {
      workspaceName: name,
      createdFromTemplate: {
        templateId: template.id,
        templateName: template.name,
        templateVersion: template.version,
        appliedAt: now,
        workspaceTools: template.workspaceTools.slice(),
      },
      features: Object.assign({}, template.features),
      folders: folders,
      workspaceTools: template.workspaceTools.slice(),
      updatedAt: now,
    };
  }

  function makeDefaultWorkspaceMetadata() {
    var projectTemplate = workspaceTemplateByID('project');
    return {
      Project: metadataForTemplate('Project', projectTemplate),
      Test: metadataForTemplate('Test', projectTemplate),
    };
  }

  function genericWorkspaceMetadata(name) {
    return {
      workspaceName: name,
      features: { files: true },
      folders: { notes: 'Notes', files: 'Files' },
    };
  }

  function makeDefaultVaultFiles() {
    return {
      '': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Docs': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Docs/todo.txt': { type: 'file', content: 'Buy groceries\nWrite tests', modifiedAt: new Date().toISOString() },
      'Docs/readme.md': { type: 'file', content: '# Hello World\n\nThis is a **test** document.\n\n- item 1\n- item 2', modifiedAt: new Date().toISOString() },
      'Notes': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Notes/Overview.md': { type: 'file', content: '# Notes Overview\n\nMy notes content here.', modifiedAt: new Date().toISOString() },
      'Project': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Project/Notes': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Project/Notes/Overview.md': { type: 'file', content: '# Project Overview\n', modifiedAt: new Date().toISOString() },
      'Project/project-only.txt': { type: 'file', content: 'project file', modifiedAt: new Date().toISOString() },
      'Test': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Test/test-only.txt': { type: 'file', content: 'test file', modifiedAt: new Date().toISOString() }
    };
  }

  function importEntry(id, path, mediaHint) {
    return {
      id: id,
      path: path,
      kind: 'file',
      size: 32,
      modifiedAt: '2026-07-20T10:00:00Z',
      mediaHint: mediaHint || 'application/octet-stream'
    };
  }

  function makeImportSource(kind) {
    importSequence += 1;
    var handle = 'mock-import-' + kind + '-' + importSequence;
    var isArchive = kind === 'archive';
    var entries = isArchive ? [
      importEntry('doku-start', 'wiki/data/pages/project/start.txt', 'text/plain'),
      importEntry('doku-plan', 'wiki/data/pages/project/plan.txt', 'text/plain'),
      importEntry('doku-private', 'wiki/data/pages/private/passwords.txt', 'text/plain'),
      importEntry('doku-logo', 'wiki/data/media/media/logo.png', 'image/png'),
      importEntry('legacy-home', 'legacy/pages/home.txt', 'text/plain')
    ] : [
      importEntry('obsidian-settings', 'Vault/.obsidian/app.json', 'application/json'),
      importEntry('obsidian-readme', 'Vault/Projects/Readme.md', 'text/markdown'),
      importEntry('obsidian-plan', 'Vault/Projects/Plan.md', 'text/markdown'),
      importEntry('obsidian-diagram', 'Vault/Projects/diagram.png', 'image/png'),
      importEntry('obsidian-backup', 'Vault/Projects/backup.zip', 'application/zip')
    ];
    var texts = isArchive ? {
      'doku-start': '====== Start ======\n[[project:plan|Plan]] {{:media:logo.png|Logo}}',
      'doku-plan': '====== Plan ======\n  * Review',
      'doku-private': 'Ordinary synthetic page',
      'legacy-home': '====== Legacy ======'
    } : {
      'obsidian-readme': '# Readme\n[[Plan]] ![[diagram.png]]',
      'obsidian-plan': '# Plan\n- [ ] Review'
    };
    var fingerprint = 'mock-fingerprint-' + kind;
    importSessions[handle] = { handle: handle, kind: kind, entries: entries, texts: texts, fingerprint: fingerprint, cancelled: false, closed: false };
    return {
      sourceHandle: handle,
      kind: isArchive ? 'archive' : 'directory',
      displayPath: isArchive ? 'backup.tar.gz' : 'Vault',
      displayName: isArchive ? 'backup.tar.gz' : 'Vault',
      fingerprint: fingerprint,
      entryCount: entries.length,
      totalBytes: entries.reduce(function (total, entry) { return total + entry.size; }, 0)
    };
  }

  function importSession(pluginId, sourceHandle, permission) {
    var err = requirePluginPermission(pluginId, permission);
    if (err) return { error: err };
    var session = importSessions[sourceHandle];
    if (!session || session.closed) return { error: 'import-source-not-found' };
    return { session: session };
  }

  function makeDefaultSyncState() {
    return {
      configured: false,
      serverUrl: '',
      vaultId: '',
      deviceId: 'mock-device',
      deviceName: '',
      connected: false,
      revoked: false,
      tokenStored: false,
      unpushedOps: 0,
      lastSyncAt: '',
      syncInterval: 0,
      lastError: '',
      lastWarning: '',
      statusLabel: 'disabled',
      serverSequence: 0
    };
  }

  function normalizeVaultPath(relativePath, allowRoot) {
    var p = String(relativePath || '');
    if (p.indexOf('\x00') !== -1) return { error: 'invalid-path: null-byte' };
    if (p.indexOf('\\') !== -1) return { error: 'invalid-path: backslash not allowed' };
    if (p.indexOf('./') === 0) p = p.slice(2);
    if (!allowRoot && !p) return { error: 'invalid-path: empty path' };
    if (p.charAt(0) === '/' || /^[A-Za-z]:/.test(p)) return { error: 'invalid-path: absolute path rejected' };
    var parts = p.split('/').filter(Boolean);
    if (parts.indexOf('..') !== -1) return { error: 'invalid-path: path-traversal' };
    if (parts.some(function(part) { return part.toLowerCase() === '.verstak'; })) return { error: 'reserved-path: .verstak is internal' };
    return { path: parts.join('/') };
  }

  function parentPath(path) {
    var idx = path.lastIndexOf('/');
    return idx === -1 ? '' : path.slice(0, idx);
  }

  var cancelledTransfers = {};

  function emptyTransferOutcome() {
    return { results: [], succeeded: 0, failed: 0, cancelled: false };
  }

  // Mirrors the host: one failing item does not abandon the batch, cancellation
  // stops it without undoing what already landed, and progress is emitted after
  // every item.
  function runMockTransfers(pluginId, transferId, transfers, operation, apply) {
    var outcome = emptyTransferOutcome();
    var list = Array.isArray(transfers) ? transfers : [];
    var chain = Promise.resolve();
    list.forEach(function (transfer, index) {
      chain = chain.then(function () {
        if (outcome.cancelled) return null;
        if (transferId && cancelledTransfers[transferId]) {
          outcome.cancelled = true;
          list.slice(index).forEach(function (remaining) {
            outcome.results.push({ from: remaining.from, to: remaining.to, skipped: true });
          });
          return null;
        }
        return apply(transfer).then(function (error) {
          if (error) {
            outcome.failed += 1;
            outcome.results.push({ from: transfer.from, to: transfer.to, error: error });
          } else {
            outcome.succeeded += 1;
            outcome.results.push({ from: transfer.from, to: transfer.to });
          }
          window.__VERSTAK_DISPATCH_TRANSFER_PROGRESS__?.({
            transferId: transferId,
            pluginId: pluginId,
            completed: index + 1,
            total: list.length,
            path: transfer.to,
            succeeded: outcome.succeeded,
            failed: outcome.failed
          });
        });
      });
    });
    return chain.then(function () {
      delete cancelledTransfers[transferId];
      return [outcome, ''];
    });
  }

  function baseName(path) {
    var idx = path.lastIndexOf('/');
    return idx === -1 ? path : path.slice(idx + 1);
  }

  function fileEntry(path, node) {
    var name = path ? baseName(path) : '';
    var ext = '';
    var dot = name.lastIndexOf('.');
    if (dot > 0) ext = name.slice(dot + 1);
    return {
      name: name,
      relativePath: path,
      type: node.type,
      size: node.type === 'file' ? (node.content || '').length : 0,
      modifiedAt: node.modifiedAt || new Date().toISOString(),
      extension: ext,
      isHidden: name.charAt(0) === '.',
      isReserved: false,
      canRead: node.type === 'file' || node.type === 'folder',
      canWrite: node.type === 'file' || node.type === 'folder'
    };
  }

  function requirePluginPermission(pluginId, permission) {
    var s = pluginStates[pluginId];
    if (!s || !s.enabled || (s.status !== 'loaded' && s.status !== 'degraded')) {
      return 'plugin not enabled and loaded';
    }
    if (!s.manifest.permissions || s.manifest.permissions.indexOf(permission) === -1) {
      return 'plugin lacks required permission ' + permission;
    }
    if (vaultStatus.status !== 'open') return 'vault-not-open';
    return '';
  }

  function makePlugin(id) {
    var s = pluginStates[id];
    if (!s) return null;
    return {
      manifest: s.manifest,
      status: s.status,
      enabled: s.enabled,
      rootPath: s.rootPath,
      error: s.error
    };
  }

  function allPlugins() {
    return Object.keys(pluginStates).map(makePlugin).filter(Boolean);
  }

  function allCapabilities() {
    var caps = [];
    caps.push({ name: 'verstak/core/plugin-manager/v1', description: 'Plugin management', pluginId: 'verstak-desktop', status: 'stable' });
    caps.push({ name: 'verstak/core/capability-registry/v1', description: 'Capability registry', pluginId: 'verstak-desktop', status: 'stable' });
    caps.push({ name: 'verstak/core/files/v1', description: 'Files API', pluginId: 'verstak-desktop', status: 'stable' });
    caps.push({ name: 'verstak/core/workbench/v1', description: 'Workbench routing', pluginId: 'verstak-desktop', status: 'stable' });
    caps.push({ name: 'verstak/core/sync/v1', description: 'Sync API', pluginId: 'verstak-desktop', status: 'stable' });
    caps.push({ name: 'verstak/core/notifications/v1', description: 'Native notifications', pluginId: 'verstak-desktop', status: 'stable' });
    caps.push({ name: 'verstak/core/import/v1', description: 'Safe external import sessions', pluginId: 'verstak-desktop', status: 'stable' });
    for (var id in pluginStates) {
      var s = pluginStates[id];
      if (s.status === 'loaded' && s.enabled && s.manifest && s.manifest.provides) {
        s.manifest.provides.forEach(function (p) {
          caps.push({ name: p, description: '', pluginId: id, status: 'stable' });
        });
      }
    }
    return caps;
  }

  function allPermissions() {
    return [
      { name: 'vault.read', description: 'Read vault data', dangerous: false },
      { name: 'events.publish', description: 'Publish events', dangerous: false },
      { name: 'events.subscribe', description: 'Subscribe to events', dangerous: false },
      { name: 'ui.register', description: 'Register UI contributions', dangerous: false },
      { name: 'commands.register', description: 'Register commands', dangerous: false },
      { name: 'storage.namespace', description: 'Access plugin storage', dangerous: false },
      { name: 'files.read', description: 'Read vault files', dangerous: false },
      { name: 'files.write', description: 'Write vault files', dangerous: true },
      { name: 'files.delete', description: 'Trash vault files', dangerous: true },
      { name: 'files.openExternal', description: 'Open vault files and folders externally', dangerous: true },
      { name: 'workbench.open', description: 'Request Workbench open/edit routing', dangerous: false },
      { name: 'network.remote', description: 'Connect to remote network services', dangerous: true },
      { name: 'sync.participate', description: 'Participate in vault sync', dangerous: true },
      { name: 'imports.readExternal', description: 'Read a selected external source', dangerous: true },
      { name: 'imports.apply', description: 'Apply a reviewed import plan', dangerous: true }
    ];
  }

  function syncStatusDTO() {
    return {
      configured: syncState.configured,
      serverUrl: syncState.serverUrl,
      vaultId: syncState.vaultId,
      deviceId: syncState.deviceId,
      deviceName: syncState.deviceName,
      connected: syncState.connected,
      revoked: syncState.revoked,
      tokenStored: syncState.tokenStored,
      unpushedOps: syncState.unpushedOps,
      lastSyncAt: syncState.lastSyncAt,
      syncInterval: syncState.syncInterval,
      lastError: syncState.lastError,
      lastWarning: syncState.lastWarning,
      statusLabel: syncState.statusLabel
    };
  }

  function requirePluginSyncPermission(pluginId, remote) {
    var err = requirePluginPermission(pluginId, 'sync.participate');
    if (err) return err;
    if (remote) {
      err = requirePluginPermission(pluginId, 'network.remote');
      if (err) return err;
    }
    return '';
  }

  function allContributions() {
    var views = [], commands = [], searchProviders = [], worklogProviders = [], overviewProviders = [], sidebarItems = [], statusBarItems = [], settingsPanels = [], openProviders = [], workspaceItems = [];
    for (var id in pluginStates) {
      var s = pluginStates[id];
      var c = (s.manifest && s.manifest.contributes) || {};
      if (c.views) c.views.forEach(function (v) { views.push(Object.assign({}, v, { pluginId: id })); });
      if (c.commands) c.commands.forEach(function (cmd) { commands.push(Object.assign({}, cmd, { pluginId: id })); });
      if (c.searchProviders) c.searchProviders.forEach(function (sp) { searchProviders.push(Object.assign({}, sp, { pluginId: id })); });
      if (c.worklogProviders) c.worklogProviders.forEach(function (wp) { worklogProviders.push(Object.assign({}, wp, { pluginId: id })); });
      if (c.overviewProviders) c.overviewProviders.forEach(function (op) { overviewProviders.push(Object.assign({}, op, { pluginId: id })); });
      if (c.sidebarItems) c.sidebarItems.forEach(function (sb) { sidebarItems.push(Object.assign({}, sb, { pluginId: id })); });
      if (c.statusBarItems) c.statusBarItems.forEach(function (st) { statusBarItems.push(Object.assign({}, st, { pluginId: id })); });
      if (c.settingsPanels) c.settingsPanels.forEach(function (sp) { settingsPanels.push(Object.assign({}, sp, { pluginId: id })); });
      if (c.openProviders) c.openProviders.forEach(function (op) { openProviders.push(Object.assign({}, op, { pluginId: id })); });
      if (c.workspaceItems) c.workspaceItems.forEach(function (wi) {
        var item = Object.assign({}, wi, { pluginId: id });
        // Test seam: lets a spec change a plugin's declared tab order without
        // rewriting the manifest fixtures.
        var overrides = window.__VERSTAK_MOCK_TOOL_ORDER__;
        if (overrides && Object.prototype.hasOwnProperty.call(overrides, id)) {
          item.order = overrides[id];
        }
        workspaceItems.push(item);
      });
    }
    return { views: views, commands: commands, searchProviders: searchProviders, worklogProviders: worklogProviders, overviewProviders: overviewProviders, sidebarItems: sidebarItems, statusBarItems: statusBarItems, settingsPanels: settingsPanels, openProviders: openProviders, workspaceItems: workspaceItems };
  }

  function requestExtension(request) {
    if (request && request.extension) {
      var explicit = String(request.extension).toLowerCase();
      return explicit.charAt(0) === '.' ? explicit : '.' + explicit;
    }
    var p = String((request && request.path) || '').toLowerCase();
    var slash = p.lastIndexOf('/');
    var name = slash === -1 ? p : p.slice(slash + 1);
    var dot = name.lastIndexOf('.');
    return dot > 0 ? name.slice(dot) : '';
  }

  function requestContextName(request) {
    var ctx = (request && request.context) || {};
    if (ctx.notesMode || ctx.isInsideNotesFolder || ctx.sourceView === 'notes') return 'notes-markdown';
    var ext = requestExtension(request);
    if (ext === '.md' || ext === '.markdown') return 'generic-markdown';
    return 'generic-text';
  }

  function providerSupports(provider, request) {
    var ext = requestExtension(request);
    var contextName = requestContextName(request);
    var mode = String((request && request.mode) || 'view').toLowerCase();
    return (provider.supports || []).some(function (support) {
      if (support.kind && support.kind !== request.kind) return false;
      if (support.modes && support.modes.length && support.modes.map(function (m) { return String(m).toLowerCase(); }).indexOf(mode) === -1) return false;
      if (support.extensions && support.extensions.length && support.extensions.map(function (e) { return String(e).toLowerCase(); }).indexOf(ext) === -1) return false;
      if (support.contexts && support.contexts.length && support.contexts.indexOf(contextName) === -1) return false;
      return true;
    });
  }

  function selectOpenProvider(request) {
    var providers = allContributions().openProviders.filter(function (provider) {
      var s = pluginStates[provider.pluginId];
      return s && s.enabled && (s.status === 'loaded' || s.status === 'degraded') && providerSupports(provider, request);
    });
    providers.sort(function (a, b) {
      var byPriority = (b.priority || 0) - (a.priority || 0);
      if (byPriority) return byPriority;
      return String(a.id).localeCompare(String(b.id));
    });
    return providers[0] || null;
  }

  function openWorkbenchResource(pluginId, request, forcedMode) {
    var s = pluginStates[pluginId];
    if (!s || !s.enabled || (s.status !== 'loaded' && s.status !== 'degraded')) {
      return Promise.resolve([{}, 'plugin not enabled and loaded']);
    }
    if (!s.manifest.permissions || s.manifest.permissions.indexOf('workbench.open') === -1) {
      return Promise.resolve([{}, 'plugin lacks required permission workbench.open']);
    }
    var normalized = Object.assign({}, request || {});
    normalized.kind = normalized.kind || 'vault-file';
    normalized.mode = forcedMode || normalized.mode || 'view';
    normalized.extension = requestExtension(normalized);
    normalized.context = Object.assign({}, normalized.context || {}, { sourcePluginId: pluginId });
    var provider = selectOpenProvider(normalized);
    if (!provider) {
      return Promise.resolve([{
        status: 'no-provider',
        request: normalized,
        message: 'no open provider for resource'
      }, '']);
    }
    var result = {
      status: 'opened',
      providerId: provider.id,
      providerPluginId: provider.pluginId,
      providerComponent: provider.component,
      request: normalized
    };
    openedResources.push(Object.assign({ id: provider.id + ':' + openedResources.length, openedAt: new Date().toISOString() }, result));
    return Promise.resolve([result, '']);
  }

  function defaultEditorBundle() {
    return '(' + function () {
      function e(tag, attrs, children) {
        var node = document.createElement(tag);
        attrs = attrs || {};
        Object.keys(attrs).forEach(function (key) {
          if (key === 'className') node.className = attrs[key];
          else if (key.indexOf('on') === 0) node.addEventListener(key.slice(2).toLowerCase(), attrs[key]);
          else node.setAttribute(key, attrs[key]);
        });
        (children || []).forEach(function (child) { node.appendChild(typeof child === 'string' ? document.createTextNode(child) : child); });
        return node;
      }
      function esc(s) { return String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }
      function renderMarkdown(text) {
        return String(text || '').split(/\n/).map(function (line) {
          if (/^#\s+/.test(line)) return '<h1>' + esc(line.replace(/^#\s+/, '')) + '</h1>';
          if (/^-\s+\[[ x]\]\s+/i.test(line)) return '<ul><li><input type="checkbox" disabled> ' + esc(line.replace(/^-\s+\[[ x]\]\s+/i, '')) + '</li></ul>';
          if (/^-\s+/.test(line)) return '<ul><li>' + esc(line.replace(/^-\s+/, '')) + '</li></ul>';
          return line ? '<p>' + esc(line).replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>') + '</p>' : '';
        }).join('');
      }
      function insertAround(ta, before, after, fallback) {
        var start = ta.selectionStart;
        var end = ta.selectionEnd;
        var text = ta.value.slice(start, end) || fallback || '';
        ta.value = ta.value.slice(0, start) + before + text + after + ta.value.slice(end);
        ta.selectionStart = start + before.length;
        ta.selectionEnd = start + before.length + text.length;
        ta.dispatchEvent(new Event('input', { bubbles: true }));
      }
      var DefaultEditor = {
        mount: function (c, p, api) {
          if (!document.getElementById('mock-default-editor-styles')) {
            var style = document.createElement('style');
            style.id = 'mock-default-editor-styles';
            style.textContent = '.de-root{display:flex;flex-direction:column;height:100%;min-height:0;overflow:hidden}.de-toolbar,.de-md-toolbar{display:flex;align-items:center;gap:.5rem;padding:.5rem .75rem;border-bottom:1px solid #16213e;background:#12122a;flex-wrap:wrap}.de-toolbar-mode{font-size:.75rem;color:#4ecca3;padding:.15rem .5rem;border-radius:3px;background:#1a2a3a}.de-toolbar-context{font-size:.75rem;color:#8b8ba8}.de-toolbar-spacer{flex:1}.de-toolbar-btn,.de-md-btn{font-size:.75rem;padding:.25rem .6rem;border:1px solid #333;border-radius:4px;background:#1a1a2e;color:#ccc}.de-toolbar-btn.active{border-color:#4ecca3;color:#4ecca3}.de-status.dirty{color:#f39c12}.de-status.saved{color:#4ecca3}.de-editor-wrap{flex:1;display:flex;min-height:0;overflow:hidden}.de-pane{flex:1;display:flex;min-width:0}.de-pane+.de-pane{border-left:1px solid #16213e}.de-lines{padding:.75rem .4rem;background:#0a0a15;color:#555;font-family:monospace;line-height:1.6;white-space:pre}.de-textarea{flex:1;height:100%;resize:none;border:0;outline:0;padding:.75rem;font-family:monospace;font-size:.85rem;line-height:1.6;background:#0d0d1a;color:#e0e0e0}.de-preview{flex:1;padding:1rem;overflow:auto;background:#0d0d1a;color:#ddd}.de-notes-badge{font-size:.65rem;padding:.1rem .4rem;border-radius:3px;background:#2a1a3a;color:#b388ff}';
            document.head.appendChild(style);
          }
          c.innerHTML = '';
          c.className = 'de-root';
          var req = p.request || {};
          var path = req.path || '';
          var ctx = req.context || {};
          var isNotes = ctx.notesMode || ctx.isInsideNotesFolder;
          var ext = (req.extension || '').toLowerCase();
          var isMd = ext === '.md' || ext === '.markdown';
          var editorMode = isNotes ? 'notes-markdown' : isMd ? 'generic-markdown' : 'text';
          var viewMode = isMd && req.mode !== 'edit' ? 'preview' : 'edit';
          var current = '';
          var saved = '';
          var dirty = false;
          var ta = null;
          var preview = null;
          var status = e('span', { className: 'de-status', 'data-save-state': '' }, []);
          c.setAttribute('data-editor-mode', editorMode);
          c.setAttribute('data-resource-path', path);
          c.setAttribute('data-request-mode', req.mode || 'view');
          var toolbar = e('div', { className: 'de-toolbar' }, [e('span', { className: 'de-toolbar-mode' }, [editorMode]), e('span', { className: 'de-toolbar-context' }, [path])]);
          if (isNotes) toolbar.appendChild(e('span', { className: 'de-notes-badge', 'data-notes-badge': '' }, ['notes context']));
          toolbar.appendChild(e('span', { className: 'de-toolbar-spacer' }, []));
          ['edit', 'preview', 'split'].forEach(function (mode) {
            if (!isMd) return;
            toolbar.appendChild(e('button', { className: 'de-toolbar-btn', 'data-editor-mode-button': mode, onClick: function () { viewMode = mode; rebuild(); } }, [mode[0].toUpperCase() + mode.slice(1)]));
          });
          toolbar.appendChild(e('button', { className: 'de-toolbar-btn', 'data-editor-action': 'reload', onClick: reload }, ['Reload']));
          toolbar.appendChild(e('button', { className: 'de-toolbar-btn', onClick: save }, ['Save']));
          toolbar.appendChild(status);
          c.appendChild(toolbar);
          if (isMd) {
            var md = e('div', { className: 'de-md-toolbar' }, []);
            [['heading', 'H'], ['bold', 'B'], ['italic', 'I'], ['link', 'Link'], ['code', 'Code'], ['code-block', '```'], ['bullet', 'List'], ['numbered', '1.'], ['quote', 'Quote'], ['task', 'Task']].forEach(function (item) {
              md.appendChild(e('button', { className: 'de-md-btn', 'data-md-action': item[0], onClick: function () { mdAction(item[0]); } }, [item[1]]));
            });
            c.appendChild(md);
          }
          var wrap = e('div', { className: 'de-editor-wrap' }, []);
          c.appendChild(wrap);
          function setStatus(text, cls) { status.textContent = text; status.className = 'de-status ' + (cls || ''); }
          function update() { dirty = current !== saved; setStatus(dirty ? 'Modified' : 'Saved', dirty ? 'dirty' : 'saved'); if (preview) preview.innerHTML = renderMarkdown(current); }
          function makeEditor() {
            var pane = e('div', { className: 'de-pane' }, []);
            var lines = e('div', { className: 'de-lines' }, []);
            ta = e('textarea', { className: 'de-textarea', 'data-editor-textarea': '', spellcheck: 'false' }, []);
            ta.value = current;
            function renumber() { lines.textContent = Array.from({ length: ta.value.split('\n').length }, function (_, i) { return i + 1; }).join('\n'); }
            ta.addEventListener('input', function () { current = ta.value; renumber(); update(); });
            ta.addEventListener('keydown', function (ev) { if ((ev.ctrlKey || ev.metaKey) && ev.key.toLowerCase() === 's') { ev.preventDefault(); save(); } if (ev.key === 'Tab') { ev.preventDefault(); insertAround(ta, '  ', '', ''); } });
            renumber();
            pane.appendChild(lines);
            pane.appendChild(ta);
            return pane;
          }
          function makePreview() { preview = e('div', { className: 'de-preview', 'data-preview': '' }, []); preview.innerHTML = renderMarkdown(current); return e('div', { className: 'de-pane' }, [preview]); }
          function rebuild() {
            wrap.innerHTML = '';
            ta = null;
            preview = null;
            if (!isMd || viewMode === 'edit' || viewMode === 'split') wrap.appendChild(makeEditor());
            if (isMd && (viewMode === 'preview' || viewMode === 'split')) wrap.appendChild(makePreview());
            Array.from(toolbar.querySelectorAll('[data-editor-mode-button]')).forEach(function (btn) { btn.className = 'de-toolbar-btn' + (btn.getAttribute('data-editor-mode-button') === viewMode ? ' active' : ''); });
            update();
          }
          function save() {
            return api.files.writeText(path, current, { createIfMissing: false, overwrite: true }).then(function () { saved = current; dirty = false; setStatus('Saved', 'saved'); });
          }
          function reload() {
            if (dirty && !window.confirm('Discard unsaved changes and reload from disk?')) return;
            api.files.readText(path).then(function (text) { current = text || ''; saved = current; dirty = false; rebuild(); });
          }
          function mdAction(action) {
            if (!ta) { viewMode = 'edit'; rebuild(); }
            if (action === 'heading') insertAround(ta, '# ', '', '');
            else if (action === 'bold') insertAround(ta, '**', '**', 'bold text');
            else if (action === 'italic') insertAround(ta, '*', '*', 'italic text');
            else if (action === 'link') insertAround(ta, '[', '](https://)', 'link text');
            else if (action === 'code') insertAround(ta, '`', '`', 'code');
            else if (action === 'code-block') insertAround(ta, '```\n', '\n```', 'code');
            else if (action === 'bullet') insertAround(ta, '- ', '', 'item');
            else if (action === 'numbered') insertAround(ta, '1. ', '', 'item');
            else if (action === 'quote') insertAround(ta, '> ', '', 'quote');
            else if (action === 'task') insertAround(ta, '- [ ] ', '', 'task');
          }
          reload();
        },
        unmount: function (c) { c.innerHTML = ''; }
      };
      window.VerstakPluginRegister('verstak.default-editor', { components: { DefaultEditor: DefaultEditor } });
    }.toString() + ')();';
  }

  function trashPluginBundle() {
    return '(' + function () {
      var PLUGIN_ID = 'verstak.trash';

      function el(tag, attrs, children) {
        var node = document.createElement(tag);
        attrs = attrs || {};
        Object.keys(attrs).forEach(function (key) {
          var value = attrs[key];
          if (value == null) return;
          if (key === 'className') node.className = value;
          else if (key === 'textContent') node.textContent = value;
          else if (key.indexOf('on') === 0) node.addEventListener(key.slice(2).toLowerCase(), value);
          else if (key === 'value') node.value = value;
          else if (key === 'disabled') node.disabled = !!value;
          else node.setAttribute(key, value);
        });
        (children || []).forEach(function (child) {
          if (child == null) return;
          node.appendChild(typeof child === 'string' ? document.createTextNode(child) : child);
        });
        return node;
      }

      function text(value) { return String(value == null ? '' : value); }
      function cleanPath(value) { return text(value).split('/').filter(Boolean).join('/'); }
      function nameFor(entry) {
        var path = cleanPath(entry && entry.originalPath);
        return text(entry && entry.basename).trim() || path.split('/').pop() || 'Untitled item';
      }
      function workspaceFor(entry) { return cleanPath(entry && entry.originalPath).split('/')[0] || 'Vault root'; }
      function typeFor(entry) { return entry && entry.originalType === 'folder' ? 'Folder' : 'File'; }
      function dateFor(entry) { return text(entry && entry.deletedAt); }
      function errorText(error) { return error && error.message ? error.message : text(error); }
      function isConflict(error) { return /conflict:/i.test(errorText(error)); }

      var TrashView = {
        mount: function (containerEl, props, api) {
          var state = {
            entries: [], workspace: '', query: '', sort: 'date-desc', loading: true,
            busyId: '', confirmingId: '', status: '', statusError: false, disposed: false
          };

          function workspaces() {
            var values = {};
            state.entries.forEach(function (entry) { values[workspaceFor(entry)] = true; });
            return Object.keys(values).sort();
          }

          function visible() {
            var query = state.query.toLowerCase();
            return state.entries.filter(function (entry) {
              if (state.workspace && workspaceFor(entry) !== state.workspace) return false;
              return !query || (nameFor(entry) + ' ' + text(entry.originalPath) + ' ' + workspaceFor(entry)).toLowerCase().indexOf(query) !== -1;
            }).sort(function (left, right) {
              if (state.sort === 'date-asc') return dateFor(left).localeCompare(dateFor(right));
              if (state.sort === 'name-asc') return nameFor(left).localeCompare(nameFor(right));
              return dateFor(right).localeCompare(dateFor(left));
            });
          }

          function render() {
            var rows = visible();
            containerEl.innerHTML = '';
            containerEl.className = 'trash-root';
            containerEl.setAttribute('data-plugin-id', PLUGIN_ID);

            var workspaceSelect = el('select', {
              'data-trash-filter-workspace': '', value: state.workspace,
              onChange: function (event) { state.workspace = event.target.value; render(); }
            }, [el('option', { value: '' }, ['All workspaces'])]);
            workspaces().forEach(function (workspace) {
              workspaceSelect.appendChild(el('option', { value: workspace }, [workspace]));
            });
            var search = el('input', {
              type: 'search', value: state.query, placeholder: 'Filter name or path',
              'data-trash-filter-search': '',
              onInput: function (event) { state.query = event.target.value; render(); }
            }, []);
            var sort = el('select', {
              'data-trash-sort': '', value: state.sort,
              onChange: function (event) { state.sort = event.target.value; render(); }
            }, [
              el('option', { value: 'date-desc' }, ['Deleted: newest']),
              el('option', { value: 'date-asc' }, ['Deleted: oldest']),
              el('option', { value: 'name-asc' }, ['Name'])
            ]);
            containerEl.appendChild(el('div', { className: 'trash-toolbar' }, [
              el('strong', {}, ['Trash']), search, workspaceSelect, sort,
              el('button', { type: 'button', onClick: load }, ['Refresh'])
            ]));
            containerEl.appendChild(el('div', {
              className: 'trash-status' + (state.statusError ? ' error' : ''),
              'data-trash-status': ''
            }, [state.loading ? 'Loading deleted items...' : (state.status || rows.length + ' deleted items')]));

            var list = el('div', { className: 'trash-list', 'data-trash-list': '' }, []);
            if (state.loading) {
              list.appendChild(el('div', { className: 'trash-empty' }, ['Loading deleted items...']));
            } else if (!rows.length) {
              list.appendChild(el('div', { className: 'trash-empty' }, [state.entries.length ? 'No deleted items match the current filters.' : 'Trash is empty.']));
            } else {
              rows.forEach(function (entry) {
                list.appendChild(el('div', {
                  className: 'trash-row', 'data-trash-row': entry.trashId, 'data-trash-workspace': workspaceFor(entry)
                }, [
                  el('span', { className: 'trash-name' }, [nameFor(entry)]),
                  el('span', { className: 'trash-workspace' }, [workspaceFor(entry)]),
                  el('span', { className: 'trash-path' }, [entry.originalPath || '']),
                  el('span', { className: 'trash-meta' }, [dateFor(entry)]),
                  el('span', { className: 'trash-meta' }, [typeFor(entry)]),
                  el('button', {
                    type: 'button', disabled: state.busyId === entry.trashId,
                    'data-trash-restore': entry.trashId,
                    onClick: function () { restore(entry); }
                  }, [state.busyId === entry.trashId ? 'Restoring...' : 'Restore']),
                  el('button', {
                    type: 'button', disabled: state.busyId === entry.trashId,
                    'data-trash-delete': entry.trashId,
                    onClick: function () { state.confirmingId = entry.trashId; render(); }
                  }, ['Delete permanently'])
                ]));
              });
            }
            containerEl.appendChild(list);

            var entry = state.entries.find(function (item) { return item.trashId === state.confirmingId; });
            if (entry) {
              containerEl.appendChild(el('div', { className: 'trash-confirm', 'data-trash-confirm': entry.trashId }, [
                el('p', {}, ['Delete permanently?']),
                el('span', {}, [entry.originalPath || nameFor(entry)]),
                el('button', {
                  type: 'button', 'data-trash-confirm-cancel': entry.trashId,
                  onClick: function () { state.confirmingId = ''; render(); }
                }, ['Cancel']),
                el('button', {
                  type: 'button', disabled: state.busyId === entry.trashId,
                  'data-trash-confirm-delete': entry.trashId,
                  onClick: function () { deletePermanently(entry); }
                }, ['Delete permanently'])
              ]));
            }
          }

          function restore(entry) {
            state.busyId = entry.trashId;
            state.status = '';
            state.statusError = false;
            render();
            api.files.restoreTrash(entry.trashId, { overwrite: false }).then(function () {
              if (state.disposed) return;
              state.entries = state.entries.filter(function (item) { return item.trashId !== entry.trashId; });
              state.busyId = '';
              state.status = 'Restored ' + nameFor(entry) + '.';
              render();
            }).catch(function (error) {
              if (state.disposed) return;
              state.busyId = '';
              state.statusError = true;
              state.status = isConflict(error)
                ? 'Restore blocked: an item already exists at the original path. Nothing was overwritten.'
                : 'Restore failed: ' + errorText(error);
              render();
            });
          }

          function deletePermanently(entry) {
            state.busyId = entry.trashId;
            state.status = '';
            state.statusError = false;
            render();
            api.files.deleteTrash(entry.trashId).then(function () {
              if (state.disposed) return;
              state.entries = state.entries.filter(function (item) { return item.trashId !== entry.trashId; });
              state.busyId = '';
              state.confirmingId = '';
              state.status = 'Permanently deleted ' + nameFor(entry) + '.';
              render();
            }).catch(function (error) {
              if (state.disposed) return;
              state.busyId = '';
              state.statusError = true;
              state.status = 'Permanent delete failed: ' + errorText(error);
              render();
            });
          }

          function load() {
            state.loading = true;
            state.status = '';
            state.statusError = false;
            render();
            api.files.listTrash().then(function (entries) {
              if (state.disposed) return;
              state.entries = Array.isArray(entries) ? entries : [];
              state.loading = false;
              render();
            }).catch(function (error) {
              if (state.disposed) return;
              state.entries = [];
              state.loading = false;
              state.statusError = true;
              state.status = 'Could not load Trash: ' + errorText(error);
              render();
            });
          }

          containerEl.__trashCleanup = function () { state.disposed = true; containerEl.innerHTML = ''; };
          load();
        },
        unmount: function (containerEl) { if (containerEl.__trashCleanup) containerEl.__trashCleanup(); }
      };
      window.VerstakPluginRegister(PLUGIN_ID, { components: { TrashView: TrashView } });
    }.toString() + ')();';
  }

  function simplePluginBundle(pluginId, componentName, rootClass, title) {
    var markup = '<div class="' + rootClass + '"><h2>' + title + '</h2></div>';
    return '(function(){var Component={mount:function(containerEl){containerEl.innerHTML=' + JSON.stringify(markup) + ';},unmount:function(containerEl){containerEl.innerHTML="";}};window.VerstakPluginRegister(' + JSON.stringify(pluginId) + ',{components:{' + componentName + ':Component}});})();';
  }

  function syncPluginBundle() {
    return [
      "(function(){",
      "var SyncStatusBar={mount:function(container){container.innerHTML='';var button=document.createElement('button');button.type='button';button.className='mock-sync-status';button.textContent='Synced';button.addEventListener('click',function(){window.dispatchEvent(new CustomEvent('verstak:open-settings',{detail:{pluginId:'verstak.sync',panelId:''}}));});container.appendChild(button);},unmount:function(container){container.innerHTML='';}};",
      "var SyncSettings={mount:function(container){container.innerHTML='<div class=\"sync-settings-root\">Sync settings</div>';},unmount:function(container){container.innerHTML='';}};",
      "window.VerstakPluginRegister('verstak.sync',{components:{SyncStatusBar:SyncStatusBar,SyncSettings:SyncSettings}});",
      "})();"
    ].join('');
  }

  function browserInboxBundle() {
    return '(' + function () {
      var PLUGIN_ID = 'verstak.browser-inbox';
      var GLOBAL_KEY = 'captures:global';
      var LEGACY_KEY = 'captures';
      var WORKSPACE_PREFIX = 'captures:workspace:';

      function injectStyles() {
        if (document.getElementById('mock-browser-inbox-style')) return;
        var style = document.createElement('style');
        style.id = 'mock-browser-inbox-style';
        style.textContent = [
          '.browser-inbox-root{height:100%;min-height:0;display:flex;flex-direction:column;background:#0d0d1a;color:#e0e0f0}',
          '.browser-inbox-toolbar{display:flex;align-items:center;gap:.5rem;padding:.55rem .75rem;border-bottom:1px solid #16213e;background:#12122a;flex-wrap:wrap}',
          '.browser-inbox-title{font-size:.86rem;font-weight:600;color:#f0f0ff}.browser-inbox-count,.browser-inbox-status{font-size:.74rem;color:#8b8ba8}.browser-inbox-spacer{flex:1}',
          '.browser-inbox-filters{display:flex;align-items:center;gap:.35rem;flex:1;flex-wrap:wrap}.browser-inbox-select,.browser-inbox-input{min-height:1.85rem;max-width:12rem;border:1px solid #1a3a5c;border-radius:4px;background:#101020;color:#e0e0f0;padding:.25rem .4rem;font-size:.76rem}.browser-inbox-input{width:12rem}',
          '.browser-inbox-btn{min-height:1.85rem;padding:.3rem .65rem;border:1px solid #1a3a5c;border-radius:4px;background:#0f3460;color:#e0e0f0;font-size:.76rem;cursor:pointer}.browser-inbox-btn:hover{background:#1a4a7a}.browser-inbox-btn:disabled{opacity:.45;cursor:default}.browser-inbox-btn.danger{border-color:#633;color:#ffb0b0}',
          '.browser-inbox-body{flex:1;min-height:0;display:grid;grid-template-columns:minmax(260px,360px) minmax(0,1fr)}.browser-inbox-list{min-height:0;overflow:auto;border-right:1px solid #16213e;background:#101020}.browser-inbox-detail{min-width:0;min-height:0;overflow:auto;padding:1rem;display:flex;flex-direction:column;gap:.75rem}',
          '.browser-inbox-empty,.browser-inbox-detail-empty{height:100%;display:flex;align-items:center;justify-content:center;padding:1.5rem;color:#8b8ba8;font-size:.84rem;line-height:1.45;text-align:center}.browser-inbox-detail-empty{height:auto;margin:auto}',
          '.browser-inbox-row{display:grid;gap:.25rem;padding:.65rem .75rem;border-bottom:1px solid rgba(22,33,62,.75);cursor:pointer}.browser-inbox-row:hover{background:#17172d}.browser-inbox-row.selected{background:#1a2a3a}.browser-inbox-row-head{display:flex;align-items:center;gap:.45rem;min-width:0}.browser-inbox-kind{color:#4ecca3;font-size:.68rem;text-transform:uppercase}.browser-inbox-row-title{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:.86rem}.browser-inbox-row-url,.browser-inbox-row-text{min-width:0;overflow:hidden;text-overflow:ellipsis;color:#8b8ba8;font-size:.74rem}',
          '.browser-inbox-detail-title{font-size:1rem;font-weight:600;color:#f4f7fb;overflow-wrap:anywhere}.browser-inbox-meta{display:grid;grid-template-columns:7rem minmax(0,1fr);gap:.35rem .75rem;font-size:.78rem}.browser-inbox-meta-label{color:#777}.browser-inbox-meta-value{color:#ccc;overflow-wrap:anywhere}.browser-inbox-text{padding:.75rem;border:1px solid #24304f;border-radius:6px;background:#101020;color:#ddd;font-size:.84rem;line-height:1.5;white-space:pre-wrap}.browser-inbox-detail-actions{display:flex;gap:.5rem;flex-wrap:wrap}',
          '@media(max-width:760px){.browser-inbox-body{grid-template-columns:1fr}.browser-inbox-list{border-right:0;border-bottom:1px solid #16213e;max-height:45vh}.browser-inbox-meta{grid-template-columns:1fr}}'
        ].join('');
        document.head.appendChild(style);
      }

      function el(tag, attrs, children) {
        var node = document.createElement(tag);
        attrs = attrs || {};
        Object.keys(attrs).forEach(function (key) {
          if (key === 'textContent') node.textContent = attrs[key];
          else if (key === 'className') node.className = attrs[key];
          else if (key === 'disabled') node.disabled = !!attrs[key];
          else if (key === 'value') node.value = attrs[key];
          else if (key.indexOf('data-') === 0) node.setAttribute(key, attrs[key]);
          else if (key === 'onClick') node.addEventListener('click', attrs[key]);
          else if (key === 'onChange') node.addEventListener('change', attrs[key]);
          else if (key === 'onInput') node.addEventListener('input', attrs[key]);
          else node[key] = attrs[key];
        });
        (children || []).forEach(function (child) {
          if (child) node.appendChild(child);
        });
        return node;
      }

      function workspaceRoot(props) {
        return String(
          props.workspaceRootPath ||
          (props.workspaceNode && (props.workspaceNode.rootPath || props.workspaceNode.name || props.workspaceNode.id)) ||
          props.workspaceName ||
          ''
        ).trim();
      }

      function workspaceKey(root) {
        return 'captures:workspace:' + encodeURIComponent(root || '');
      }

      function cleanWorkspace(value) {
        return String(value == null ? '' : value).trim().replace(/^\/+|\/+$/g, '');
      }

      function workspaceFromKey(key) {
        key = String(key || '');
        if (key.indexOf(WORKSPACE_PREFIX) !== 0) return '';
        try {
          return cleanWorkspace(decodeURIComponent(key.slice(WORKSPACE_PREFIX.length)));
        } catch (_) {
          return '';
        }
      }

      function rows(value) {
        return Array.isArray(value) ? value.filter(function (item) { return item && typeof item === 'object'; }) : [];
      }

      function normalizeRows(value, storageKey) {
        return rows(value).filter(function (item) { return item.captureId; }).map(function (item) {
          var workspaceRootPath = cleanWorkspace(item.workspaceRootPath || item.workspaceName) || workspaceFromKey(storageKey);
          return Object.assign({}, item, {
            workspaceRootPath: workspaceRootPath,
            workspaceName: cleanWorkspace(item.workspaceName || workspaceRootPath),
            processed: item.processed === true
          });
        });
      }

      function sortCaptures(items) {
        var seen = {};
        return items.filter(function (item) {
          if (!item || !item.captureId || seen[item.captureId]) return false;
          seen[item.captureId] = true;
          return true;
        }).sort(function (a, b) {
          return String(b.capturedAt || b.receivedAt || '').localeCompare(String(a.capturedAt || a.receivedAt || ''));
        });
      }

      function title(capture) {
        return capture.title || capture.fileName || capture.url || capture.captureId || 'Untitled material';
      }

      function renderBrowserInbox(containerEl, props, api) {
        injectStyles();
        var rootPath = workspaceRoot(props || {});
        var captures = [];
        var selectedId = '';
        var statusText = 'Ready for browser captures';
        var workspaceOptions = [];
        var statusFilter = 'all';
        var workspaceFilter = '';
        var searchQuery = '';

        containerEl.innerHTML = '';
        var root = el('div', { className: 'browser-inbox-root', 'data-plugin-id': PLUGIN_ID });
        var toolbar = el('div', { className: 'browser-inbox-toolbar' });
        var titleEl = el('span', { className: 'browser-inbox-title', textContent: rootPath ? 'Browser · ' + rootPath : 'Browser' });
        var countEl = el('span', { className: 'browser-inbox-count' });
        var statusEl = el('span', { className: 'browser-inbox-status' });
        var filtersEl = el('div', { className: 'browser-inbox-filters' });
        var statusFilterEl = el('select', {
          className: 'browser-inbox-select',
          'data-browser-inbox-filter': 'status',
          onChange: function (event) {
            statusFilter = String(event.target.value || 'all');
            selectedId = '';
            render();
          }
        }, [
          el('option', { value: 'all', textContent: 'All captures' }),
          el('option', { value: 'unassigned', textContent: 'Unassigned' }),
          el('option', { value: 'unprocessed', textContent: 'Unprocessed' }),
          el('option', { value: 'processed', textContent: 'Processed' })
        ]);
        var workspaceFilterEl = el('select', {
          className: 'browser-inbox-select',
          'data-browser-inbox-filter': 'workspace',
          onChange: function (event) {
            workspaceFilter = cleanWorkspace(event.target.value);
            selectedId = '';
            render();
          }
        });
        var searchInput = el('input', {
          className: 'browser-inbox-input',
          type: 'search',
          placeholder: 'Search captures',
          'data-browser-inbox-filter': 'search',
          onInput: function (event) {
            searchQuery = String(event.target.value || '').trim().toLowerCase();
            selectedId = '';
            renderList();
            renderDetail();
            renderCount();
          }
        });
        var clearBtn = el('button', {
          className: 'browser-inbox-btn danger',
          'data-browser-inbox-action': 'clear',
          textContent: 'Clear',
          onClick: function () {
            clearScope().then(render);
          }
        });
        var body = el('div', { className: 'browser-inbox-body' });
        var listEl = el('div', { className: 'browser-inbox-list' });
        var detailEl = el('div', { className: 'browser-inbox-detail' });

        toolbar.appendChild(titleEl);
        toolbar.appendChild(countEl);
        filtersEl.appendChild(statusFilterEl);
        if (!rootPath) filtersEl.appendChild(workspaceFilterEl);
        filtersEl.appendChild(searchInput);
        toolbar.appendChild(filtersEl);
        toolbar.appendChild(el('span', { className: 'browser-inbox-spacer' }));
        toolbar.appendChild(statusEl);
        toolbar.appendChild(clearBtn);
        body.appendChild(listEl);
        body.appendChild(detailEl);
        root.appendChild(toolbar);
        root.appendChild(body);
        containerEl.appendChild(root);

        function option(value, label) {
          return el('option', { value: value, textContent: label });
        }

        function workspaceRoots() {
          var roots = workspaceOptions.slice();
          captures.forEach(function (capture) {
            var workspace = cleanWorkspace(capture.workspaceRootPath);
            if (workspace && roots.indexOf(workspace) === -1) roots.push(workspace);
          });
          if (rootPath && roots.indexOf(rootPath) === -1) roots.push(rootPath);
          return roots.sort(function (a, b) { return a.localeCompare(b); });
        }

        function renderWorkspaceFilterOptions() {
          if (rootPath) return;
          workspaceFilterEl.innerHTML = '';
          workspaceFilterEl.appendChild(option('', 'All Deals'));
          workspaceRoots().forEach(function (workspace) {
            workspaceFilterEl.appendChild(option(workspace, workspace));
          });
          workspaceFilterEl.value = workspaceFilter;
        }

        function visibleCaptures() {
          return captures.filter(function (capture) {
            var workspace = cleanWorkspace(capture.workspaceRootPath);
            if (rootPath && workspace !== rootPath) return false;
            if (!rootPath && workspaceFilter && workspace !== workspaceFilter) return false;
            if (statusFilter === 'unassigned' && workspace) return false;
            if (statusFilter === 'unprocessed' && capture.processed === true) return false;
            if (statusFilter === 'processed' && capture.processed !== true) return false;
            if (!searchQuery) return true;
            return [title(capture), capture.url, capture.domain, capture.text, workspace].join('\n').toLowerCase().indexOf(searchQuery) !== -1;
          });
        }

        function readSettings() {
          if (!api || !api.settings || typeof api.settings.read !== 'function') return Promise.resolve({});
          return api.settings.read().then(function (settings) { return settings || {}; }).catch(function () { return {}; });
        }

        function capturesFromSettings(settings) {
          var keys = [GLOBAL_KEY, LEGACY_KEY];
          Object.keys(settings || {}).forEach(function (name) {
            if (name.indexOf(WORKSPACE_PREFIX) === 0) keys.push(name);
          });
          var all = [];
          keys.forEach(function (name) {
            all = all.concat(normalizeRows((settings || {})[name], name));
          });
          return sortCaptures(all);
        }

        function loadWorkspaceOptions() {
          if (!api || !api.files || typeof api.files.list !== 'function') return Promise.resolve();
          return api.files.list('').then(function (entries) {
            workspaceOptions = (Array.isArray(entries) ? entries : []).filter(function (entry) {
              return String(entry && entry.type || '').toLowerCase() === 'folder';
            }).map(function (entry) {
              return cleanWorkspace(entry.relativePath || entry.name);
            }).filter(function (workspace) {
              return workspace && workspace.indexOf('/') === -1;
            });
          }).catch(function () {
            workspaceOptions = [];
          });
        }

        function persist() {
          if (!api || !api.settings || typeof api.settings.write !== 'function') return Promise.resolve();
          return api.settings.write(GLOBAL_KEY, sortCaptures(captures)).catch(function (err) {
            statusText = 'Could not save inbox: ' + (err && err.message ? err.message : String(err));
          });
        }

        function selectedCapture() {
          var visible = visibleCaptures();
          for (var i = 0; i < visible.length; i += 1) {
            if (visible[i].captureId === selectedId) return visible[i];
          }
          return visible[0] || null;
        }

        function clearScope() {
          var ids = (rootPath ? captures.filter(function (capture) {
            return cleanWorkspace(capture.workspaceRootPath) === rootPath;
          }) : captures).map(function (capture) { return capture.captureId; });
          captures = captures.filter(function (capture) { return ids.indexOf(capture.captureId) === -1; });
          selectedId = '';
          statusText = rootPath ? 'Deal materials cleared' : 'Inbox cleared';
          return persist();
        }

        function removeCapture(captureId) {
          captures = captures.filter(function (item) { return item.captureId !== captureId; });
          selectedId = '';
          statusText = 'Capture deleted';
          return persist().then(render);
        }

        function assignWorkspace(captureId, workspace) {
          workspace = cleanWorkspace(workspace);
          captures = captures.map(function (capture) {
            if (capture.captureId !== captureId) return capture;
            return Object.assign({}, capture, { workspaceRootPath: workspace, workspaceName: workspace });
          });
          if (workspace && workspaceOptions.indexOf(workspace) === -1) workspaceOptions.push(workspace);
          statusText = workspace ? 'Capture assigned to ' + workspace : 'Capture is unassigned';
          return persist().then(render);
        }

        function setProcessed(captureId, processed) {
          captures = captures.map(function (capture) {
            return capture.captureId === captureId ? Object.assign({}, capture, { processed: processed === true }) : capture;
          });
          statusText = processed ? 'Capture marked processed' : 'Capture marked unprocessed';
          return persist().then(render);
        }

        function conversionAction(kind, capture) {
          statusText = 'Ready to create ' + kind + ': ' + title(capture);
          render();
        }

        function renderList() {
          listEl.innerHTML = '';
          var visible = visibleCaptures();
          if (visible.length === 0) {
            listEl.appendChild(el('div', {
              className: 'browser-inbox-empty',
              textContent: captures.length === 0
                ? 'No browser materials yet. Send a page, selection, or link from the extension.'
                : 'No captures match the current filters.'
            }));
            return;
          }
          visible.forEach(function (capture) {
            var workspace = cleanWorkspace(capture.workspaceRootPath);
            var row = el('div', {
              className: 'browser-inbox-row' + (capture.captureId === selectedId ? ' selected' : ''),
              'data-browser-capture-id': capture.captureId,
              onClick: function () {
                selectedId = capture.captureId;
                render();
              }
            }, [
              el('div', { className: 'browser-inbox-row-head' }, [
                el('span', { className: 'browser-inbox-kind', textContent: capture.kind || 'capture' }),
                el('span', { className: 'browser-inbox-row-title', textContent: title(capture) })
              ]),
              el('div', { className: 'browser-inbox-row-url', textContent: capture.url || capture.domain || capture.captureId || '' })
            ]);
            row.appendChild(el('div', { className: 'browser-inbox-row-text', textContent: (workspace || 'Unassigned') + ' · ' + (capture.processed ? 'Processed' : 'Unprocessed') }));
            if (capture.text) row.appendChild(el('div', { className: 'browser-inbox-row-text', textContent: capture.text }));
            listEl.appendChild(row);
          });
        }

        function renderDetail() {
          detailEl.innerHTML = '';
          var capture = selectedCapture();
          if (!capture) {
            detailEl.appendChild(el('div', { className: 'browser-inbox-detail-empty', textContent: 'Select a capture to inspect it.' }));
            return;
          }
          selectedId = capture.captureId;
          detailEl.appendChild(el('div', { className: 'browser-inbox-detail-title', textContent: title(capture) }));
          detailEl.appendChild(el('div', { className: 'browser-inbox-meta' }, [
            el('div', { className: 'browser-inbox-meta-label', textContent: 'Kind' }),
            el('div', { className: 'browser-inbox-meta-value', textContent: capture.kind || '-' }),
            el('div', { className: 'browser-inbox-meta-label', textContent: 'URL' }),
            el('div', { className: 'browser-inbox-meta-value', textContent: capture.url || '-' }),
            el('div', { className: 'browser-inbox-meta-label', textContent: 'Domain' }),
            el('div', { className: 'browser-inbox-meta-value', textContent: capture.domain || '-' }),
            el('div', { className: 'browser-inbox-meta-label', textContent: 'Browser' }),
            el('div', { className: 'browser-inbox-meta-value', textContent: capture.browserName || capture.source || '-' }),
            el('div', { className: 'browser-inbox-meta-label', textContent: 'Deal' }),
            el('div', { className: 'browser-inbox-meta-value', textContent: capture.workspaceRootPath || 'Unassigned' }),
            el('div', { className: 'browser-inbox-meta-label', textContent: 'Status' }),
            el('div', { className: 'browser-inbox-meta-value', textContent: capture.processed ? 'Processed' : 'Unprocessed' })
          ]));
          var assignment = el('select', {
            className: 'browser-inbox-select',
            'data-browser-inbox-assignment': capture.captureId,
            onChange: function (event) { assignWorkspace(capture.captureId, event.target.value); }
          });
          assignment.appendChild(option('', 'Unassigned'));
          workspaceRoots().forEach(function (workspace) {
            assignment.appendChild(option(workspace, workspace));
          });
          assignment.value = capture.workspaceRootPath || '';
          var assignmentRow = el('div', { className: 'browser-inbox-detail-actions' }, [assignment]);
          if (capture.workspaceRootPath) {
            assignmentRow.appendChild(el('button', {
              className: 'browser-inbox-btn',
              'data-browser-inbox-action': 'clear-assignment',
              textContent: 'Clear assignment',
              onClick: function () { assignWorkspace(capture.captureId, ''); }
            }));
          }
          detailEl.appendChild(assignmentRow);
          if (capture.text) detailEl.appendChild(el('div', { className: 'browser-inbox-text', textContent: capture.text }));
          if (capture.fileText) detailEl.appendChild(el('div', { className: 'browser-inbox-text', textContent: capture.fileText }));
          var actionButtons = [
            el('button', {
              className: 'browser-inbox-btn',
              'data-browser-inbox-action': 'toggle-processed',
              textContent: capture.processed ? 'Mark Unprocessed' : 'Mark Processed',
              onClick: function () { setProcessed(capture.captureId, !capture.processed); }
            })
          ];
          if (capture.workspaceRootPath) {
            actionButtons.push(el('button', { className: 'browser-inbox-btn', 'data-browser-inbox-action': 'create-note', textContent: 'Create Note', onClick: function () { conversionAction('note', capture); } }));
            if (capture.url) actionButtons.push(el('button', { className: 'browser-inbox-btn', 'data-browser-inbox-action': 'create-link', textContent: 'Create Link', onClick: function () { conversionAction('link', capture); } }));
            if (capture.kind === 'file') actionButtons.push(el('button', { className: 'browser-inbox-btn', 'data-browser-inbox-action': 'create-file', textContent: 'Create File', onClick: function () { conversionAction('file', capture); } }));
          }
          actionButtons.push(el('button', { className: 'browser-inbox-btn danger', 'data-browser-inbox-action': 'remove', textContent: 'Delete', onClick: function () { removeCapture(capture.captureId); } }));
          detailEl.appendChild(el('div', { className: 'browser-inbox-detail-actions' }, actionButtons));
        }

        function renderCount() {
          var visible = visibleCaptures();
          countEl.textContent = visible.length === captures.length
            ? captures.length + ' item' + (captures.length === 1 ? '' : 's')
            : visible.length + ' of ' + captures.length + ' items';
          clearBtn.disabled = rootPath
            ? !captures.some(function (capture) { return cleanWorkspace(capture.workspaceRootPath) === rootPath; })
            : captures.length === 0;
        }

        function render() {
          statusFilterEl.value = statusFilter;
          searchInput.value = searchQuery;
          renderWorkspaceFilterOptions();
          renderCount();
          statusEl.textContent = statusText;
          renderList();
          renderDetail();
        }

        Promise.all([readSettings(), loadWorkspaceOptions()]).then(function (result) {
          captures = capturesFromSettings(result[0]);
          selectedId = captures[0] ? captures[0].captureId : '';
          render();
        });
        render();
      }

      var BrowserInboxView = {
        mount: renderBrowserInbox,
        unmount: function (containerEl) { containerEl.innerHTML = ''; }
      };
      window.VerstakPluginRegister('verstak.browser-inbox', { components: { BrowserInboxView: BrowserInboxView } });
    }.toString() + ')();';
  }

  function todoBundle() {
    return '(' + function () {
      var PLUGIN_ID = 'verstak.todo';
      var GLOBAL_KEY = 'todos:global';

      function injectStyles() {
        if (document.getElementById('mock-todo-style')) return;
        var style = document.createElement('style');
        style.id = 'mock-todo-style';
        style.textContent = [
          '.todo-root{height:100%;min-height:0;display:flex;flex-direction:column;background:#0d0d1a;color:#e0e0f0}',
          '.todo-toolbar{display:flex;align-items:center;gap:.5rem;padding:.55rem .75rem;border-bottom:1px solid #16213e;background:#12122a;flex-wrap:wrap}.todo-title{font-size:.86rem;font-weight:600;color:#f0f0ff}.todo-count,.todo-status{font-size:.74rem;color:#8b8ba8}.todo-spacer{flex:1}.todo-filters{display:flex;align-items:center;gap:.35rem;flex-wrap:wrap}',
          '.todo-input,.todo-select{min-height:1.85rem;box-sizing:border-box;border:1px solid #1a3a5c;border-radius:4px;background:#101020;color:#e0e0f0;padding:.28rem .42rem;font-size:.76rem}.todo-input.search{width:12rem}.todo-input.textarea{min-height:6rem;resize:vertical}',
          '.todo-btn{min-height:1.85rem;padding:.3rem .65rem;border:1px solid #1a3a5c;border-radius:4px;background:#0f3460;color:#e0e0f0;font-size:.76rem;cursor:pointer}.todo-btn.primary{background:#4ecca3;border-color:#4ecca3;color:#102018}.todo-btn.danger{border-color:#633;color:#ffb0b0}',
          '.todo-list{flex:1;min-height:0;overflow:auto;padding:.5rem .75rem}.todo-empty{padding:1.5rem;color:#8b8ba8}.todo-row{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:.7rem;padding:.7rem 0;border-bottom:1px solid rgba(22,33,62,.75)}.todo-row.done .todo-row-title{text-decoration:line-through;color:#8b8ba8}.todo-row-title{font-weight:600}.todo-row-description,.todo-row-meta{margin-top:.25rem;font-size:.76rem;color:#aaa}.todo-row-actions{display:flex;justify-content:flex-end;gap:.35rem;flex-wrap:wrap}.todo-badge{display:inline-flex;margin-right:.3rem;padding:.08rem .28rem;border:1px solid #24304f;border-radius:4px}.todo-badge.overdue,.todo-badge.reminder-due{border-color:#633;color:#ffb0b0}.todo-badge.due-soon{border-color:#7a6633;color:#f2d17b}',
          '.todo-modal-host[hidden]{display:none}.todo-modal-overlay{position:fixed;inset:0;z-index:10000;display:flex;align-items:center;justify-content:center;padding:1rem;background:rgba(0,0,0,.58)}.todo-modal{width:540px;max-width:96vw;display:grid;gap:.75rem;padding:1rem;border:1px solid #2c456a;border-radius:8px;background:#15152c;box-shadow:0 18px 44px rgba(0,0,0,.38)}.todo-modal-title{font-size:.95rem;font-weight:600}.todo-form-grid{display:grid;grid-template-columns:1fr 1fr;gap:.6rem}.todo-field{display:grid;gap:.3rem;font-size:.72rem;color:#8b8ba8}.todo-field.wide{grid-column:1/-1}.todo-modal-actions{display:flex;justify-content:flex-end;gap:.5rem}'
        ].join('');
        document.head.appendChild(style);
      }

      function el(tag, attrs, children) {
        var node = document.createElement(tag);
        attrs = attrs || {};
        Object.keys(attrs).forEach(function (key) {
          if (key === 'textContent') node.textContent = attrs[key];
          else if (key === 'className') node.className = attrs[key];
          else if (key === 'value') node.value = attrs[key];
          else if (key === 'checked') node.checked = !!attrs[key];
          else if (key === 'disabled') node.disabled = !!attrs[key];
          else if (key.indexOf('data-') === 0) node.setAttribute(key, attrs[key]);
          else if (key === 'onClick') node.addEventListener('click', attrs[key]);
          else if (key === 'onChange') node.addEventListener('change', attrs[key]);
          else if (key === 'onInput') node.addEventListener('input', attrs[key]);
          else node[key] = attrs[key];
        });
        (children || []).forEach(function (child) {
          if (child) node.appendChild(typeof child === 'string' ? document.createTextNode(child) : child);
        });
        return node;
      }

      function text(value) { return String(value == null ? '' : value); }
      function rootFromProps(props) {
        return text((props && (props.workspaceRootPath || props.workspaceName || props.workspaceNodeId)) || (props && props.workspaceNode && (props.workspaceNode.rootPath || props.workspaceNode.name || props.workspaceNode.id))).trim();
      }
      function rows(value) { return Array.isArray(value) ? value.filter(function (item) { return item && typeof item === 'object'; }) : []; }
      function now() { return new Date().toISOString(); }
      function dateMs(value) {
        value = text(value).trim();
        if (!value) return 0;
        var normalized = /^\d{4}-\d{2}-\d{2}$/.test(value) ? value + 'T00:00:00' : value;
        var date = new Date(normalized);
        return isNaN(date.getTime()) ? 0 : date.getTime();
      }
      function todoId(root, title) { return 'todo:' + (root || 'global') + ':' + Date.now() + ':' + text(title).trim().replace(/\s+/g, '-'); }
      function normalizeTodo(value) {
        value = value || {};
        var status = ['open', 'done', 'cancelled'].indexOf(text(value.status).toLowerCase()) === -1 ? 'open' : text(value.status).toLowerCase();
        var createdAt = text(value.createdAt).trim() || now();
        return {
          id: text(value.id).trim() || todoId(text(value.workspaceRootPath).trim(), value.title),
          title: text(value.title).trim(),
          description: text(value.description || value.body),
          workspaceRootPath: text(value.workspaceRootPath || value.workspaceName).trim(),
          workspaceName: text(value.workspaceName || value.workspaceRootPath).trim(),
          status: status,
          priority: ['low', 'normal', 'high'].indexOf(text(value.priority).toLowerCase()) === -1 ? 'normal' : text(value.priority).toLowerCase(),
          dueAt: /^\d{4}-\d{2}-\d{2}$/.test(text(value.dueAt)) ? text(value.dueAt) : '',
          reminderAt: text(value.reminderAt),
          createdAt: createdAt,
          updatedAt: text(value.updatedAt).trim() || createdAt,
          completedAt: status === 'done' ? (text(value.completedAt).trim() || createdAt) : '',
          sourceUrl: text(value.sourceUrl),
          linkedJournalEntryId: text(value.linkedJournalEntryId)
        };
      }
      function sortTodos(list, sortMode) {
        return list.slice().sort(function (a, b) {
          if (sortMode === 'updated') return text(b.updatedAt).localeCompare(text(a.updatedAt));
          var aValue = dateMs(sortMode === 'reminder' ? a.reminderAt : a.dueAt) || Number.MAX_SAFE_INTEGER;
          var bValue = dateMs(sortMode === 'reminder' ? b.reminderAt : b.dueAt) || Number.MAX_SAFE_INTEGER;
          return aValue - bValue || text(b.updatedAt).localeCompare(text(a.updatedAt));
        });
      }
      function dueState(todo) {
        var dueAt = dateMs(todo.dueAt);
        if (!dueAt || todo.status !== 'open') return '';
        var nowMs = Date.now();
        if (dueAt < nowMs) return 'overdue';
        if (dueAt <= nowMs + 3 * 24 * 60 * 60 * 1000) return 'due-soon';
        return '';
      }
      function reminderIsDue(todo) { return todo.status === 'open' && dateMs(todo.reminderAt) > 0 && dateMs(todo.reminderAt) <= Date.now(); }

      function TodoView() {}

      TodoView.mount = function (containerEl, props, api) {
        injectStyles();
        var workspaceRoot = rootFromProps(props || {});
        var isWorkspace = !!workspaceRoot;
        var todos = [];
        var statusFilter = 'all';
        var workspaceFilter = '';
        var sortMode = 'due';
        var searchQuery = '';
        var modalHost = el('div', { className: 'todo-modal-host', hidden: true });
        var titleEl = el('span', { className: 'todo-title', textContent: isWorkspace ? 'Todos · ' + workspaceRoot : 'Todos' });
        var countEl = el('span', { className: 'todo-count' });
        var statusEl = el('span', { className: 'todo-status' });
        var statusFilterEl = el('select', { className: 'todo-select', 'data-todo-filter': 'status', onChange: function (event) { statusFilter = event.target.value; render(); } }, [
          el('option', { value: 'all', textContent: 'All statuses' }),
          el('option', { value: 'open', textContent: 'Open' }),
          el('option', { value: 'done', textContent: 'Done' }),
          el('option', { value: 'cancelled', textContent: 'Cancelled' })
        ]);
        var workspaceFilterEl = el('select', { className: 'todo-select', 'data-todo-filter': 'workspace', onChange: function (event) { workspaceFilter = event.target.value; render(); } });
        var sortEl = el('select', { className: 'todo-select', 'data-todo-filter': 'sort', onChange: function (event) { sortMode = event.target.value; render(); } }, [
          el('option', { value: 'due', textContent: 'Sort by due date' }),
          el('option', { value: 'reminder', textContent: 'Sort by reminder' }),
          el('option', { value: 'updated', textContent: 'Sort by updated' })
        ]);
        var searchEl = el('input', { className: 'todo-input search', type: 'search', placeholder: 'Search todos', 'data-todo-filter': 'search', onInput: function (event) { searchQuery = text(event.target.value).trim().toLowerCase(); render(); } });
        var addBtn = el('button', { className: 'todo-btn primary', 'data-todo-action': 'add', textContent: 'Add Todo', onClick: function () { showTodoModal(null); } });
        var listEl = el('div', { className: 'todo-list' });
        var root = el('div', { className: 'todo-root', 'data-plugin-id': PLUGIN_ID });
        var toolbar = el('div', { className: 'todo-toolbar' });
        var filters = el('div', { className: 'todo-filters' });
        toolbar.appendChild(titleEl);
        toolbar.appendChild(countEl);
        filters.appendChild(statusFilterEl);
        if (!isWorkspace) filters.appendChild(workspaceFilterEl);
        filters.appendChild(sortEl);
        filters.appendChild(searchEl);
        toolbar.appendChild(filters);
        toolbar.appendChild(el('span', { className: 'todo-spacer' }));
        toolbar.appendChild(statusEl);
        toolbar.appendChild(addBtn);
        root.appendChild(toolbar);
        root.appendChild(listEl);
        root.appendChild(modalHost);
        containerEl.innerHTML = '';
        containerEl.appendChild(root);

        function workspaceRoots() {
          var found = {};
          todos.forEach(function (todo) { if (todo.workspaceRootPath) found[todo.workspaceRootPath] = true; });
          if (workspaceRoot) found[workspaceRoot] = true;
          return Object.keys(found).sort();
        }

        function renderWorkspaceOptions() {
          if (isWorkspace) return;
          workspaceFilterEl.innerHTML = '';
          workspaceFilterEl.appendChild(el('option', { value: '', textContent: 'All workspaces' }));
          workspaceFilterEl.appendChild(el('option', { value: '__unassigned__', textContent: 'Unassigned' }));
          workspaceRoots().forEach(function (rootName) {
            workspaceFilterEl.appendChild(el('option', { value: rootName, textContent: rootName }));
          });
          workspaceFilterEl.value = workspaceFilter;
        }

        function visibleTodos() {
          return sortTodos(todos.filter(function (todo) {
            if (isWorkspace && todo.workspaceRootPath !== workspaceRoot) return false;
            if (!isWorkspace && workspaceFilter === '__unassigned__' && todo.workspaceRootPath) return false;
            if (!isWorkspace && workspaceFilter && workspaceFilter !== '__unassigned__' && todo.workspaceRootPath !== workspaceFilter) return false;
            if (statusFilter !== 'all' && todo.status !== statusFilter) return false;
            if (!searchQuery) return true;
            return [todo.title, todo.description, todo.workspaceRootPath].join('\n').toLowerCase().indexOf(searchQuery) !== -1;
          }), sortMode);
        }

        function persist() { return api.settings.write(GLOBAL_KEY, todos); }
        function closeTodoModal() { modalHost.innerHTML = ''; modalHost.hidden = true; }

        function showTodoModal(existingTodo) {
          var editing = !!existingTodo;
          var titleInput = el('input', { className: 'todo-input', type: 'text', value: editing ? existingTodo.title : '', 'data-todo-input': 'title' });
          var descriptionInput = el('textarea', { className: 'todo-input textarea', value: editing ? existingTodo.description : '', 'data-todo-input': 'description' });
          var priorityInput = el('select', { className: 'todo-select', 'data-todo-input': 'priority' }, [
            el('option', { value: 'low', textContent: 'Low' }),
            el('option', { value: 'normal', textContent: 'Normal' }),
            el('option', { value: 'high', textContent: 'High' })
          ]);
          priorityInput.value = editing ? existingTodo.priority : 'normal';
          var dueInput = el('input', { className: 'todo-input', type: 'date', value: editing ? existingTodo.dueAt : '', 'data-todo-input': 'dueAt' });
          var reminderInput = el('input', { className: 'todo-input', type: 'datetime-local', value: editing ? existingTodo.reminderAt : '', 'data-todo-input': 'reminderAt' });
          var workspaceInput = null;
          if (!isWorkspace) {
            workspaceInput = el('select', { className: 'todo-select', 'data-todo-input': 'workspaceRootPath' });
            workspaceInput.appendChild(el('option', { value: '', textContent: 'Unassigned' }));
            workspaceRoots().forEach(function (rootName) { workspaceInput.appendChild(el('option', { value: rootName, textContent: rootName })); });
            workspaceInput.value = editing ? existingTodo.workspaceRootPath : '';
          }

          function saveTodo() {
            var title = text(titleInput.value).trim();
            if (!title) return;
            var rootName = isWorkspace ? workspaceRoot : text(workspaceInput && workspaceInput.value).trim();
            var timestamp = now();
            var todo = normalizeTodo({
              id: editing ? existingTodo.id : todoId(rootName, title),
              title: title,
              description: descriptionInput.value,
              workspaceRootPath: rootName,
              workspaceName: rootName,
              status: editing ? existingTodo.status : 'open',
              priority: priorityInput.value,
              dueAt: dueInput.value,
              reminderAt: reminderInput.value,
              createdAt: editing ? existingTodo.createdAt : timestamp,
              updatedAt: timestamp,
              completedAt: editing ? existingTodo.completedAt : '',
              sourceUrl: editing ? existingTodo.sourceUrl : '',
              linkedJournalEntryId: editing ? existingTodo.linkedJournalEntryId : ''
            });
            todos = editing ? todos.map(function (item) { return item.id === existingTodo.id ? todo : item; }) : [todo].concat(todos);
            closeTodoModal();
            statusEl.textContent = editing ? 'Todo updated' : 'Todo added';
            persist().then(render);
          }

          var fields = [
            el('label', { className: 'todo-field wide' }, ['Title', titleInput]),
            el('label', { className: 'todo-field wide' }, ['Description', descriptionInput]),
            el('label', { className: 'todo-field' }, ['Priority', priorityInput]),
            el('label', { className: 'todo-field' }, ['Due date', dueInput]),
            el('label', { className: 'todo-field' }, ['Reminder', reminderInput])
          ];
          if (workspaceInput) fields.push(el('label', { className: 'todo-field' }, ['Workspace', workspaceInput]));
          else fields.push(el('div', { className: 'todo-field', textContent: 'Workspace: ' + workspaceRoot }));
          modalHost.innerHTML = '';
          modalHost.hidden = false;
          modalHost.appendChild(el('div', { className: 'todo-modal-overlay' }, [
            el('div', { className: 'todo-modal' }, [
              el('div', { className: 'todo-modal-title', textContent: editing ? 'Edit Todo' : 'Add Todo' }),
              el('div', { className: 'todo-form-grid' }, fields),
              el('div', { className: 'todo-modal-actions' }, [
                el('button', { className: 'todo-btn', textContent: 'Cancel', onClick: closeTodoModal }),
                el('button', { className: 'todo-btn primary', 'data-todo-action': 'save', textContent: editing ? 'Save changes' : 'Add Todo', onClick: saveTodo })
              ])
            ])
          ]));
          titleInput.focus();
        }

        function setTodoStatus(todo, status) {
          var timestamp = now();
          todos = todos.map(function (item) {
            return item.id === todo.id ? Object.assign({}, item, { status: status, completedAt: status === 'done' ? timestamp : '', updatedAt: timestamp }) : item;
          });
          statusEl.textContent = status === 'done' ? 'Todo marked done' : (status === 'cancelled' ? 'Todo cancelled' : 'Todo reopened');
          persist().then(render);
        }

        function deleteTodo(todo) {
          todos = todos.filter(function (item) { return item.id !== todo.id; });
          statusEl.textContent = 'Todo deleted';
          persist().then(render);
        }

        function openWorkspace(todo) {
          if (!todo.workspaceRootPath) return;
          window.dispatchEvent(new CustomEvent('verstak:workspace-selected', { detail: { workspaceName: todo.workspaceRootPath } }));
          window.dispatchEvent(new CustomEvent('verstak:workspace-open-tool', { detail: { kind: 'todo' } }));
        }

        function createJournalEntry(todo) {
          if (!isWorkspace || todo.status !== 'done') return;
          window.dispatchEvent(new CustomEvent('verstak:workspace-open-tool', {
            detail: {
              kind: 'journal',
              toolRequest: {
                type: 'completed-todo',
                todo: {
                  id: todo.id,
                  title: todo.title,
                  description: todo.description,
                  workspaceRootPath: workspaceRoot,
                  completedAt: todo.completedAt
                }
              }
            }
          }));
        }

        function todoMeta(todo) {
          var due = dueState(todo);
          var reminderDue = reminderIsDue(todo);
          var badges = [
            el('span', { className: 'todo-badge', textContent: todo.priority + ' priority' }),
            el('span', { className: 'todo-badge', textContent: todo.status })
          ];
          if (!isWorkspace) badges.unshift(el('span', { className: 'todo-badge', textContent: todo.workspaceRootPath || 'Unassigned' }));
          if (todo.dueAt) badges.push(el('span', { className: 'todo-badge ' + due, textContent: (due === 'overdue' ? 'Overdue · ' : (due === 'due-soon' ? 'Due soon · ' : '')) + 'Due ' + todo.dueAt }));
          if (todo.reminderAt) badges.push(el('span', { className: 'todo-badge ' + (reminderDue ? 'reminder-due' : ''), textContent: (reminderDue ? 'Reminder due ' : 'Reminder ') + todo.reminderAt }));
          return el('div', { className: 'todo-row-meta' }, badges);
        }

        function renderList() {
          var visible = visibleTodos();
          listEl.innerHTML = '';
          if (!visible.length) {
            listEl.appendChild(el('div', { className: 'todo-empty', textContent: todos.length ? 'No todos match the current filters.' : 'No todos yet.' }));
            return;
          }
          visible.forEach(function (todo) {
            var actions = [];
            if (!isWorkspace && todo.workspaceRootPath) actions.push(el('button', { className: 'todo-btn', 'data-todo-action': 'open-workspace', textContent: 'Open workspace', onClick: function () { openWorkspace(todo); } }));
            if (todo.status === 'open') {
              actions.push(el('button', { className: 'todo-btn', 'data-todo-action': 'mark-done', textContent: 'Done', onClick: function () { setTodoStatus(todo, 'done'); } }));
              actions.push(el('button', { className: 'todo-btn', 'data-todo-action': 'cancel', textContent: 'Cancel', onClick: function () { setTodoStatus(todo, 'cancelled'); } }));
            } else {
              actions.push(el('button', { className: 'todo-btn', 'data-todo-action': 'reopen', textContent: 'Reopen', onClick: function () { setTodoStatus(todo, 'open'); } }));
            }
            if (isWorkspace && todo.status === 'done') actions.push(el('button', { className: 'todo-btn', 'data-todo-action': 'create-journal-entry', textContent: 'Create Journal Entry', onClick: function () { createJournalEntry(todo); } }));
            actions.push(el('button', { className: 'todo-btn', 'data-todo-action': 'edit', textContent: 'Edit', onClick: function () { showTodoModal(todo); } }));
            actions.push(el('button', { className: 'todo-btn danger', 'data-todo-action': 'delete', textContent: 'Delete', onClick: function () { deleteTodo(todo); } }));
            listEl.appendChild(el('div', { className: 'todo-row' + (todo.status === 'done' ? ' done' : ''), 'data-todo-id': todo.id }, [
              el('div', {}, [
                el('div', { className: 'todo-row-title', textContent: todo.title || 'Untitled todo' }),
                todo.description ? el('div', { className: 'todo-row-description', textContent: todo.description }) : null,
                todoMeta(todo)
              ]),
              el('div', { className: 'todo-row-actions' }, actions)
            ]));
          });
        }

        function render() {
          var visible = visibleTodos();
          countEl.textContent = visible.length === todos.length ? todos.length + ' todo' + (todos.length === 1 ? '' : 's') : visible.length + ' of ' + todos.length + ' todos';
          statusFilterEl.value = statusFilter;
          sortEl.value = sortMode;
          searchEl.value = searchQuery;
          renderWorkspaceOptions();
          renderList();
        }

        api.settings.read().then(function (settings) {
          todos = rows((settings || {})[GLOBAL_KEY]).map(normalizeTodo);
          render();
        }).catch(render);
        render();
      };

      TodoView.unmount = function (containerEl) { containerEl.innerHTML = ''; };
      window.VerstakPluginRegister('verstak.todo', { components: { TodoView: TodoView } });
    }.toString() + ')();';
  }

  function searchPluginBundle() {
    return '(' + function () {
      function el(tag, attrs, children) {
        var node = document.createElement(tag);
        attrs = attrs || {};
        Object.keys(attrs).forEach(function (key) {
          if (attrs[key] == null) return;
          if (key === 'className') node.className = attrs[key];
          else if (key.indexOf('on') === 0) node.addEventListener(key.slice(2).toLowerCase(), attrs[key]);
          else if (key === 'textContent') node.textContent = attrs[key];
          else node.setAttribute(key, attrs[key]);
        });
        (children || []).forEach(function (child) {
          if (child == null) return;
          node.appendChild(typeof child === 'string' ? document.createTextNode(child) : child);
        });
        return node;
      }
      function clean(path) { return String(path || '').split('/').filter(Boolean).join('/'); }
      function SearchView(containerEl, props, api) {
        if (!document.getElementById('mock-search-style')) {
          var style = document.createElement('style');
          style.id = 'mock-search-style';
          style.textContent = '.search-root{height:100%;min-height:0;display:flex;flex-direction:column;background:#0d0d1a;color:#e0e0e0}.search-toolbar{display:flex;gap:.5rem;padding:.55rem .75rem;border-bottom:1px solid #16213e;background:#12122a}.search-input{flex:1;min-width:180px;font-size:.86rem;padding:.42rem .55rem;border:1px solid #333;border-radius:4px;background:#0d0d1a;color:#e0e0e0;outline:none}.search-input:focus{border-color:#4ecca3}.search-btn{font-size:.8rem;padding:.42rem .7rem;border:1px solid #333;border-radius:4px;background:#1a1a2e;color:#ddd}.search-scope,.search-status{font-size:.78rem;color:#8b8ba8}.search-status{padding:.45rem .75rem;border-bottom:1px solid rgba(22,33,62,.55)}.search-results{flex:1;min-height:0;overflow:auto}.search-empty{padding:2rem;color:#666;text-align:center}.search-result{padding:.7rem .85rem;border-bottom:1px solid rgba(22,33,62,.55)}.search-path{color:#4ecca3}.search-snippet{margin-top:.25rem;color:#cfcfe0;font-size:.8rem}';
          document.head.appendChild(style);
        }
        containerEl.innerHTML = '';
        containerEl.className = 'search-root';
        containerEl.setAttribute('data-plugin-id', 'verstak.search');
        var rootPath = clean(props && (props.workspaceRootPath || props.workspaceName));
        var query = '';
        var timer = null;
        var results = [];
        var input = el('input', { className: 'search-input', type: 'search', placeholder: 'Search files, folders, text', 'data-search-input': 'query' });
        var button = el('button', { className: 'search-btn', 'data-search-action': 'run', textContent: 'Search' });
        var status = el('div', { className: 'search-status', textContent: 'Enter at least 2 characters.' });
        var list = el('div', { className: 'search-results' });
        containerEl.appendChild(el('div', { className: 'search-toolbar' }, [
          input,
          button,
          el('span', { className: 'search-scope', title: rootPath || 'Vault' }, [rootPath || 'Vault'])
        ]));
        containerEl.appendChild(status);
        containerEl.appendChild(list);
        function render() {
          list.innerHTML = '';
          if (!results.length) {
            list.appendChild(el('div', { className: 'search-empty' }, [query.length < 2 ? 'Enter at least 2 characters.' : 'No results']));
            return;
          }
          results.forEach(function (item) {
            list.appendChild(el('div', { className: 'search-result' }, [
              el('div', { className: 'search-path', textContent: item.relativePath }),
              el('div', { className: 'search-snippet', textContent: item.name })
            ]));
          });
        }
        async function run() {
          query = input.value.trim();
          if (query.length < 2) {
            results = [];
            status.textContent = 'Enter at least 2 characters.';
            render();
            return;
          }
          var entries = await api.files.list(rootPath);
          var needle = query.toLowerCase();
          results = (Array.isArray(entries) ? entries : []).filter(function (item) {
            return String(item.name || item.relativePath || '').toLowerCase().indexOf(needle) !== -1;
          });
          status.textContent = results.length + ' result' + (results.length === 1 ? '' : 's');
          render();
        }
        function schedule() {
          if (timer) clearTimeout(timer);
          timer = setTimeout(run, 100);
        }
        input.addEventListener('input', schedule);
        button.addEventListener('click', run);
        render();
        containerEl.__searchMockCleanup = function () { if (timer) clearTimeout(timer); };
      }
      window.VerstakPluginRegister('verstak.search', {
        components: {
          SearchView: {
            mount: SearchView,
            unmount: function (containerEl) {
              if (containerEl.__searchMockCleanup) containerEl.__searchMockCleanup();
              containerEl.innerHTML = '';
            }
          }
        }
      });
    }.toString() + ')();';
  }

  function platformTestBundle() {
    return [
      "(function(){",
      "var DiagnosticsPanel={",
      "mount:function(containerEl,props,api){",
      "containerEl.innerHTML='';",
      "containerEl.__ptCleanup=[];",
      "function track(fn){if(typeof fn==='function')containerEl.__ptCleanup.push(fn);}",
      "var root=document.createElement('div');",
      "root.className='pt-root';",
      "var title=document.createElement('h2');",
      "title.className='pt-plugin-name';",
      "title.textContent='Platform Diagnostics';",
      "var pluginId=document.createElement('p');",
      "pluginId.className='pt-plugin-id';",
      "pluginId.textContent=api.pluginId;",
      "var status=document.createElement('div');",
      "status.className='pt-badge pt-badge-success';",
      "status.textContent='Frontend Bundle Loaded';",
      "var saved=document.createElement('div');",
      "saved.className='pt-card pt-saved-setting';",
      "saved.textContent='Saved setting: loading...';",
      "var cap=document.createElement('div');",
      "cap.className='pt-capability-result';",
      "cap.textContent='Capabilities: loading...';",
      "api.capabilities.list().then(function(caps){cap.textContent='Capabilities: '+caps.length+' available';});",
      "api.settings.read('savedText').then(function(value){saved.textContent='Saved setting: '+(value||'');});",
      "var input=document.createElement('input');",
      "input.className='pt-setting-input';",
      "input.setAttribute('aria-label','Saved setting');",
      "input.value='changed value';",
      "var button=document.createElement('button');",
      "button.className='btn btn-primary pt-save-setting';",
      "button.textContent='Save Setting';",
      "button.addEventListener('click',function(){api.settings.write('savedText',input.value).then(function(){saved.textContent='Saved setting: '+input.value;});});",
      "api.capabilities.has('verstak/platform-test/v1').then(function(ok){status.textContent='Frontend Bundle Loaded | capability '+(ok?'available':'missing');});",
      "var command=document.createElement('div');",
      "command.className='pt-command-result';",
      "command.textContent='Command: registering...';",
      "api.commands.register('verstak.platform-test.show-version',function(){return {version:'0.1.0',source:'bundled-frontend'};}).then(function(unregister){track(unregister);return api.commands.execute('verstak.platform-test.show-version',{});}).then(function(result){status.setAttribute('data-command-status',result.status||'');command.textContent='Command: '+result.status+' '+result.result.version+' from '+result.result.source;});",
      "var eventResult=document.createElement('div');",
      "eventResult.className='pt-event-result';",
      "eventResult.textContent='Event: subscribing...';",
      "api.events.subscribe('verstak.platform-test.echo',function(event){eventResult.textContent='Event: received '+event.payload.message;eventResult.setAttribute('data-event-status','received');}).then(function(unsubscribe){track(unsubscribe);return api.events.publish('verstak.platform-test.echo',{message:'hello-event'});});",
      "var filesResult=document.createElement('div');",
      "filesResult.className='pt-files-result';",
      "filesResult.textContent='Files: running...';",
      "var filesError=document.createElement('div');",
      "filesError.className='pt-files-error-result';",
      "filesError.textContent='Files error path: checking...';",
      "var workbenchResult=document.createElement('div');",
      "workbenchResult.className='pt-workbench-result';",
      "workbenchResult.textContent='Workbench: ready';",
      "function makeWorkbenchButton(cls,label,request){var b=document.createElement('button');b.className='btn btn-primary '+cls;b.textContent=label;b.addEventListener('click',function(){workbenchResult.textContent='Workbench: opening...';api.workbench.editResource(request).then(function(result){workbenchResult.textContent='Workbench: opened '+result.request.path+' with '+(result.providerId||'no-provider');workbenchResult.setAttribute('data-workbench-status',result.status==='opened'?'ok':result.status);}).catch(function(err){workbenchResult.textContent='Workbench error: '+(err&&err.message?err.message:String(err));workbenchResult.setAttribute('data-workbench-status','error');});});return b;}",
      "var textWorkbenchButton=makeWorkbenchButton('pt-open-workbench-text','Open Text Diagnostic',{kind:'vault-file',path:'Docs/todo.txt',extension:'.txt',mime:'text/plain',context:{sourceView:'files'}});",
      "var markdownWorkbenchButton=makeWorkbenchButton('pt-open-workbench-markdown','Open Markdown Diagnostic',{kind:'vault-file',path:'Docs/readme.md',extension:'.md',context:{sourceView:'files'}});",
      "var notesWorkbenchButton=makeWorkbenchButton('pt-open-workbench-notes','Open Notes Diagnostic',{kind:'vault-file',path:'Notes/Overview.md',extension:'.md',context:{sourceView:'notes',isInsideNotesFolder:true,notesMode:true}});",
      "api.files.createFolder('PlatformTest').catch(function(e){if(String(e).indexOf('conflict')===-1)throw e;}).then(function(){return api.files.writeText('PlatformTest/files-api.txt','hello files',{createIfMissing:true,overwrite:true});}).then(function(){return api.files.readText('PlatformTest/files-api.txt');}).then(function(text){if(text!=='hello files')throw new Error('read mismatch');return api.files.list('PlatformTest');}).then(function(entries){if(!entries.some(function(e){return e.relativePath==='PlatformTest/files-api.txt';}))throw new Error('list missing file');return api.files.move('PlatformTest/files-api.txt','PlatformTest/files-api-moved.txt',{overwrite:true});}).then(function(){return api.files.trash('PlatformTest/files-api-moved.txt');}).then(function(){filesResult.textContent='Files: wrote/read/listed/moved/trashed';filesResult.setAttribute('data-files-status','ok');}).catch(function(err){filesResult.textContent='Files error: '+(err&&err.message?err.message:String(err));filesResult.setAttribute('data-files-status','error');});",
      "api.files.readText('.verstak/vault.json').then(function(){filesError.textContent='Files error path: unexpectedly allowed';filesError.setAttribute('data-files-error-status','error');}).catch(function(err){var message=err&&err.message?err.message:String(err);if(message.indexOf('reserved-path')===-1&&message.indexOf('.verstak')===-1){filesError.textContent='Files error path: wrong error '+message;filesError.setAttribute('data-files-error-status','error');return;}filesError.textContent='Files error path: rejected reserved-path';filesError.setAttribute('data-files-error-status','expected');});",
      "root.appendChild(title);",
      "root.appendChild(pluginId);",
      "root.appendChild(status);",
      "root.appendChild(saved);",
      "root.appendChild(input);",
      "root.appendChild(button);",
      "root.appendChild(cap);",
      "root.appendChild(command);",
      "root.appendChild(eventResult);",
      "root.appendChild(filesResult);",
      "root.appendChild(filesError);",
      "root.appendChild(textWorkbenchButton);",
      "root.appendChild(markdownWorkbenchButton);",
      "root.appendChild(notesWorkbenchButton);",
      "root.appendChild(workbenchResult);",
      "containerEl.appendChild(root);",
      "},",
      "unmount:function(containerEl){while(containerEl.__ptCleanup&&containerEl.__ptCleanup.length){containerEl.__ptCleanup.pop()();}containerEl.innerHTML='';}",
      "};",
      "var MarkdownDiagnosticProvider={",
      "mount:function(containerEl,props,api){",
      "containerEl.innerHTML='';",
      "var root=document.createElement('div');",
      "root.className='pt-root pt-workbench-result';",
      "root.setAttribute('data-workbench-status','ok');",
      "var req=(props&&props.request)||{};",
      "var ctx=(req.context&&req.context.notesMode)||false?'notes-markdown':((req.extension==='.md'||req.extension==='.markdown')?'generic-markdown':'generic-text');",
      "root.setAttribute('data-resource-path',req.path||'');",
      "root.setAttribute('data-resource-mode',req.mode||'');",
      "root.setAttribute('data-resource-context',ctx);",
      "root.textContent='Workbench: opened '+(req.path||'')+' with '+((props&&props.providerId)||'')+' mode='+(req.mode||'')+' context='+ctx;",
      "containerEl.appendChild(root);",
      "},",
      "unmount:function(containerEl){containerEl.innerHTML='';}",
      "};",
      "var PlatformTestSettings={",
      "mount:function(containerEl,props,api){",
      "containerEl.innerHTML='<div class=\"pt-root\"><h2>Platform Test Settings</h2><p>'+api.pluginId+'</p></div>';",
      "},",
      "unmount:function(containerEl){containerEl.innerHTML='';}",
      "};",
      "window.VerstakPluginRegister('verstak.platform-test',{components:{DiagnosticsPanel:DiagnosticsPanel,PlatformTestSettings:PlatformTestSettings,MarkdownDiagnosticProvider:MarkdownDiagnosticProvider}});",
      "})();"
    ].join('');
  }

  // ── Mock API ───────────────────────────────────────────────────────
  var mock = {
    GetPlugins: function () { return Promise.resolve(allPlugins()); },
    GetCapabilities: function () { return Promise.resolve(allCapabilities()); },
    GetPermissions: function () { return Promise.resolve(allPermissions()); },
    GetContributions: function () { return Promise.resolve(allContributions()); },
    GetVaultStatus: function () { return Promise.resolve(vaultStatus); },
    GetVaultPluginState: function () { return Promise.resolve(vaultPluginState); },
    GetDiagnosticsInfo: function () {
      return Promise.resolve({
        logPath: '/home/tester/.local/share/verstak/logs/verstak-2026-01-01-000000.log',
        logDir: '/home/tester/.local/share/verstak/logs',
        verbose: false,
      });
    },
    CollectDiagnostics: function () {
      diagnosticsReports.push('/home/tester/.local/share/verstak/logs/verstak-diagnostics-2026-01-01-000000.txt');
      return Promise.resolve([diagnosticsReports[diagnosticsReports.length - 1], '']);
    },
    GetBuildInfo: function () {
      return Promise.resolve({
        version: 'test',
        commit: 'testing',
        buildDate: '2026-01-01T00:00:00Z',
        display: 'test (testing)',
      });
    },
    GetAppSettings: function () { return Promise.resolve(appSettings); },
    GetPluginFrontendInfo: function (pluginId) {
      var s = pluginStates[pluginId];
      if (s && s.manifest && s.manifest.frontend) {
        return Promise.resolve({ entry: s.manifest.frontend.entry });
      }
      return Promise.resolve({});
    },
    GetPluginLocalization: function (pluginId, locale) {
      return Promise.resolve([mockPluginCatalog(pluginId, locale), '']);
    },
    PluginSelectImportDirectory: function (pluginId) {
      var err = requirePluginPermission(pluginId, 'imports.readExternal');
      return Promise.resolve(err ? [{}, err] : [makeImportSource('directory'), '']);
    },
    PluginSelectImportArchive: function (pluginId) {
      var err = requirePluginPermission(pluginId, 'imports.readExternal');
      return Promise.resolve(err ? [{}, err] : [makeImportSource('archive'), '']);
    },
    PluginListImportEntries: function (pluginId, sourceHandle, cursor) {
      var found = importSession(pluginId, sourceHandle, 'imports.readExternal');
      if (found.error) return Promise.resolve([{}, found.error]);
      if (cursor) return Promise.resolve([{ entries: [], nextCursor: '', fingerprint: found.session.fingerprint }, '']);
      return Promise.resolve([{ entries: found.session.entries.map(cloneJson), nextCursor: '', fingerprint: found.session.fingerprint }, '']);
    },
    PluginReadImportText: function (pluginId, sourceHandle, entryId) {
      var found = importSession(pluginId, sourceHandle, 'imports.readExternal');
      if (found.error) return Promise.resolve(['', found.error]);
      if (!Object.prototype.hasOwnProperty.call(found.session.texts, entryId)) return Promise.resolve(['', 'import-entry-not-text']);
      return Promise.resolve([found.session.texts[entryId], '']);
    },
    PluginApplyImportPlan: function (pluginId, sourceHandle, plan) {
      var found = importSession(pluginId, sourceHandle, 'imports.apply');
      if (found.error) return Promise.resolve([{}, found.error]);
      var session = found.session;
      session.cancelled = false;
      var format = String((plan && plan.runName) || '').indexOf('DokuWiki') === 0 ? 'dokuwiki' : 'obsidian';
      importRunCounts[format] += 1;
      var runName = plan && plan.runName ? plan.runName : (format === 'dokuwiki' ? 'DokuWiki' : 'Obsidian');
      var suffix = importRunCounts[format] > 1 ? ' (' + importRunCounts[format] + ')' : '';
      var nodes = Array.isArray(plan && plan.nodes) ? plan.nodes : [];
      var count = function (kind) { return nodes.filter(function (node) { return node.kind === kind; }).length; };
      window.__VERSTAK_DISPATCH_IMPORT_PROGRESS__?.({ pluginId: pluginId, sourceHandle: sourceHandle, phase: 'staging', completed: 1, total: 2, cancellable: true, message: '' });
      return new Promise(function (resolve) {
        setTimeout(function () {
          if (session.cancelled) {
            resolve([{}, 'import-cancelled']);
            return;
          }
          window.__VERSTAK_DISPATCH_IMPORT_PROGRESS__?.({ pluginId: pluginId, sourceHandle: sourceHandle, phase: 'publishing', completed: 2, total: 2, cancellable: false, message: '' });
          resolve([{
            runPath: 'Импортировано/' + runName + suffix,
            folders: count('folder'),
            workspaces: count('workspace'),
            notes: count('note'),
            files: count('file'),
            skipped: count('skip'),
            warnings: []
          }, '']);
        }, 250);
      });
    },
    PluginCancelImport: function (pluginId, sourceHandle) {
      var found = importSession(pluginId, sourceHandle, 'imports.apply');
      if (found.error) return Promise.resolve(found.error);
      found.session.cancelled = true;
      return Promise.resolve('');
    },
    PluginCloseImportSource: function (pluginId, sourceHandle) {
      var found = importSession(pluginId, sourceHandle, 'imports.readExternal');
      if (found.error === 'import-source-not-found') return Promise.resolve('');
      if (found.error) return Promise.resolve(found.error);
      found.session.closed = true;
      return Promise.resolve('');
    },
    PluginSecretsStatus: function () {
      return Promise.resolve([{ initialized: true, unlocked: true }, '']);
    },
    PluginSecretsUnlock: function () {
      return Promise.resolve('');
    },
    PluginSecretsList: function () {
      return Promise.resolve([secretRecords.map(function (record) {
        var listed = cloneJson(record);
        delete listed.value;
        return listed;
      }), '']);
    },
    PluginSecretsRead: function (_pluginId, secretID) {
      var record = secretRecords.find(function (item) { return item.id === secretID; });
      if (!record) return Promise.resolve([{}, 'not-found: secret ' + secretID]);
      return Promise.resolve([cloneJson(record), '']);
    },
    PluginSecretsWrite: function (_pluginId, nextRecord) {
      var record = Object.assign({}, nextRecord || {});
      if (!record.id) return Promise.resolve([{}, 'secret id is required']);
      record.scope = record.scope || { kind: 'global' };
      record.updatedAt = new Date().toISOString();
      var index = secretRecords.findIndex(function (item) { return item.id === record.id; });
      if (index === -1) secretRecords.push(record);
      else secretRecords[index] = record;
      return Promise.resolve([cloneJson(record), '']);
    },
    PluginSecretsDelete: function (_pluginId, secretID) {
      secretRecords = secretRecords.filter(function (record) { return record.id !== secretID; });
      return Promise.resolve('');
    },
    PluginSecretsCopyLink: function (_pluginId, secretID) {
      var record = secretRecords.find(function (item) { return item.id === secretID; });
      if (!record) return Promise.resolve(['', 'not-found: secret ' + secretID]);
      return Promise.resolve(['[' + (record.title || record.id) + '](verstak-secret://' + encodeURIComponent(record.id) + ')', '']);
    },
    ReadPluginSettings: function (pluginId) {
      return Promise.resolve([Object.assign({}, pluginSettings[pluginId] || {}), '']);
    },
    WritePluginSettings: function (pluginId, settings) {
      pluginSettings[pluginId] = Object.assign({}, settings || {});
      return Promise.resolve('');
    },
    ReadPluginSetting: function (pluginId, key) {
      return Promise.resolve([pluginSettings[pluginId] && pluginSettings[pluginId][key], '']);
    },
    WritePluginSetting: function (pluginId, key, value) {
      pluginSettings[pluginId] = pluginSettings[pluginId] || {};
      pluginSettings[pluginId][key] = value;
      return Promise.resolve('');
    },
    ReplacePluginNotifications: function (pluginId, items) {
      pluginNotifications[pluginId] = Array.isArray(items) ? items.slice() : [];
      return Promise.resolve('');
    },
    ClearPluginNotifications: function (pluginId) {
      delete pluginNotifications[pluginId];
      return Promise.resolve('');
    },
    ReadPluginDataJSON: function (pluginId, name) {
      var data = (pluginData[pluginId] && pluginData[pluginId][name]) || {};
      return Promise.resolve([Object.assign({}, data), '']);
    },
    ReadPluginDataNDJSON: function (pluginId, name) {
      var data = (pluginData[pluginId] && pluginData[pluginId][name]) || [];
      if (!Array.isArray(data) && pluginId === 'verstak.activity' && name === 'activity-events') data = [];
      if (!data.length && pluginId === 'verstak.activity' && name === 'activity-events') {
        data = Object.keys(pluginSettings[pluginId] || {}).filter(function (key) {
          return key === 'events' || key === 'events:global' || key.indexOf('events:workspace:') === 0;
        }).flatMap(function (key) {
          return Array.isArray(pluginSettings[pluginId][key]) ? pluginSettings[pluginId][key] : [];
        });
      }
      return Promise.resolve([Array.isArray(data) ? data.slice() : [], '']);
    },
    WritePluginDataJSON: function (pluginId, name, data) {
      pluginData[pluginId] = pluginData[pluginId] || {};
      pluginData[pluginId][name] = Object.assign({}, data || {});
      return Promise.resolve('');
    },
    WritePluginDataNDJSON: function (pluginId, name, records) {
      pluginData[pluginId] = pluginData[pluginId] || {};
      pluginData[pluginId][name] = Array.isArray(records) ? records.slice() : [];
      return Promise.resolve('');
    },
    OpenWorkbenchResource: function (pluginId, request) {
      return openWorkbenchResource(pluginId, request || {}, '');
    },
    EditWorkbenchResource: function (pluginId, request) {
      return openWorkbenchResource(pluginId, request || {}, 'edit');
    },
    GetWorkbenchOpenedResources: function () {
      return Promise.resolve(openedResources.map(function (resource) {
        return Object.assign({}, resource, { request: Object.assign({}, resource.request || {}) });
      }));
    },
    GetWorkbenchPreferences: function () {
      return Promise.resolve(Object.assign({}, workbenchPreferences));
    },
    UpdateWorkbenchPreferences: function (preferences) {
      workbenchPreferences = Object.assign({}, workbenchPreferences, preferences || {});
      return Promise.resolve('');
    },
    PluginSyncStatus: function (pluginId) {
      var err = requirePluginSyncPermission(pluginId, false);
      if (err) return Promise.resolve([{}, err]);
      return Promise.resolve([syncStatusDTO(), '']);
    },
    PluginSyncConfigure: function (pluginId, serverUrl, username, password, vaultId) {
      var err = requirePluginSyncPermission(pluginId, true);
      if (err) return Promise.resolve(err);
      syncState.configured = true;
      syncState.serverUrl = serverUrl || '';
      syncState.vaultId = vaultId || 'test-vault-001';
      syncState.deviceId = 'mock-device';
      syncState.deviceName = 'mock-device';
      syncState.connected = true;
      syncState.revoked = false;
      syncState.tokenStored = true;
      syncState.lastError = '';
      syncState.statusLabel = 'connected';
      pluginSettings[pluginId] = Object.assign({}, pluginSettings[pluginId] || {}, {
        serverUrl: syncState.serverUrl,
        vaultId: syncState.vaultId,
        syncStatus: syncState.statusLabel
      });
      return Promise.resolve('');
    },
    PluginSyncDisconnect: function (pluginId) {
      var err = requirePluginSyncPermission(pluginId, false);
      if (err) return Promise.resolve(err);
      syncState = makeDefaultSyncState();
      pluginSettings[pluginId] = Object.assign({}, pluginSettings[pluginId] || {}, {
        serverUrl: '',
        syncStatus: syncState.statusLabel
      });
      return Promise.resolve('');
    },
    PluginSyncTestConnection: function (pluginId, serverUrl) {
      var err = requirePluginSyncPermission(pluginId, true);
      if (err) return Promise.resolve(err);
      if (!serverUrl) return Promise.resolve('server URL is required');
      return Promise.resolve('');
    },
    PluginSyncSetInterval: function (pluginId, minutes) {
      var err = requirePluginSyncPermission(pluginId, false);
      if (err) return Promise.resolve(err);
      syncState.syncInterval = Number(minutes) || 0;
      return Promise.resolve('');
    },
    PluginSyncResetKey: function (pluginId) {
      var err = requirePluginSyncPermission(pluginId, false);
      if (err) return Promise.resolve(err);
      syncState.configured = false;
      syncState.deviceId = '';
      syncState.deviceName = '';
      syncState.connected = false;
      syncState.revoked = false;
      syncState.tokenStored = false;
      syncState.lastError = '';
      syncState.statusLabel = 'disconnected';
      pluginSettings[pluginId] = Object.assign({}, pluginSettings[pluginId] || {}, {
        syncStatus: syncState.statusLabel
      });
      return Promise.resolve('');
    },
    PluginSyncNow: function (pluginId) {
      var err = requirePluginSyncPermission(pluginId, true);
      if (err) return Promise.resolve([{}, err]);
      if (!syncState.configured) return Promise.resolve([{}, 'sync not configured']);
      syncState.lastSyncAt = new Date().toISOString();
      syncState.lastError = '';
      syncState.statusLabel = 'connected';
      return Promise.resolve([{ pushed: 0, pulled: 0, serverSequence: syncState.serverSequence }, '']);
    },
    GetPluginAssetContent: function (pluginId, assetPath) {
      if (pluginId === platformTestManifest.id && assetPath === platformTestManifest.frontend.entry) {
        return Promise.resolve(platformTestBundle());
      }
      if (pluginId === defaultEditorManifest.id && assetPath === defaultEditorManifest.frontend.entry) {
        return Promise.resolve(defaultEditorSource);
      }
      if (pluginId === filePreviewManifest.id && assetPath === filePreviewManifest.frontend.entry) {
        return Promise.resolve(filePreviewSource);
      }
      if (pluginId === filesManifest.id && assetPath === filesManifest.frontend.entry) {
        return Promise.resolve(filesSource);
      }
      if (pluginId === trashManifest.id && assetPath === trashManifest.frontend.entry) {
        return Promise.resolve(trashPluginBundle());
      }
      if (pluginId === notesManifest.id && assetPath === notesManifest.frontend.entry) {
        return Promise.resolve(notesSource);
      }
      if (pluginId === syncManifest.id && assetPath === syncManifest.frontend.entry) {
        return Promise.resolve(syncPluginBundle());
      }
      if (pluginId === activityManifest.id && assetPath === activityManifest.frontend.entry) {
        return Promise.resolve(activitySource);
      }
      if (pluginId === journalManifest.id && assetPath === journalManifest.frontend.entry) {
        return Promise.resolve(journalSource);
      }
      if (pluginId === browserInboxManifest.id && assetPath === browserInboxManifest.frontend.entry) {
        return Promise.resolve(browserInboxSource);
      }
      if (pluginId === todoManifest.id && assetPath === todoManifest.frontend.entry) {
        return Promise.resolve(todoSource);
      }
      if (pluginId === secretsManifest.id && assetPath === secretsManifest.frontend.entry) {
        return Promise.resolve(secretsSource);
      }
      if (pluginId === searchManifest.id && assetPath === searchManifest.frontend.entry) {
        return Promise.resolve(searchPluginBundle());
      }
      if (pluginId === importManifest.id && assetPath === importManifest.frontend.entry) {
        return Promise.resolve(importSource);
      }
      if (pluginId === importManifest.id && assetPath === importManifest.frontend.style) {
        return Promise.resolve(importStyle);
      }
      return Promise.resolve('');
    },
    GetPluginCapability: function (pluginId, capId) {
      var caps = allCapabilities();
      var found = caps.find(function (cap) { return cap.name === capId; });
      return Promise.resolve([found ? Object.assign({ available: true }, found) : { available: false, name: capId }, '']);
    },
    ListPluginCapabilities: function () { return Promise.resolve([allCapabilities(), '']); },
    ExecutePluginCommand: function (pluginId, commandId, args) {
      var s = pluginStates[pluginId];
      var commands = ((s && s.manifest && s.manifest.contributes && s.manifest.contributes.commands) || []);
      var found = commands.find(function (cmd) { return cmd.id === commandId; });
      if (!found) return Promise.resolve([{}, 'command not declared']);
      return Promise.resolve([{ status: 'declared', pluginId: pluginId, commandId: commandId, handler: found.handler, args: args || {} }, '']);
    },
    PublishPluginEvent: function (pluginId, eventName, payload) {
      if (pluginId === 'verstak.browser-inbox' && eventName === 'browser-inbox.storage.mutate') {
        payload = payload || {};
        var settings = Object.assign({}, pluginSettings[pluginId] || {});
        var globalKey = 'captures:global';
        var legacyKey = 'captures';
        var workspacePrefix = 'captures:workspace:';
        var captures = Array.isArray(settings[globalKey]) ? settings[globalKey].map(cloneJson) : [];
        var action = String(payload.action || '');
        if (action === 'migrate') {
          var seen = {};
          var migrated = [];
          Object.keys(settings).filter(function (key) {
            return key === globalKey || key === legacyKey || key.indexOf(workspacePrefix) === 0;
          }).forEach(function (key) {
            var scopedWorkspace = '';
            if (key.indexOf(workspacePrefix) === 0) {
              try { scopedWorkspace = decodeURIComponent(key.slice(workspacePrefix.length)); }
              catch (_) { scopedWorkspace = key.slice(workspacePrefix.length); }
            }
            (Array.isArray(settings[key]) ? settings[key] : []).forEach(function (row) {
              if (!row || typeof row !== 'object') return;
              var item = cloneJson(row);
              if (scopedWorkspace && !item.workspaceRootPath) item.workspaceRootPath = scopedWorkspace;
              if (scopedWorkspace && !item.workspaceName) item.workspaceName = scopedWorkspace;
              var id = String(item.captureId || '');
              if (id && seen[id]) return;
              if (id) seen[id] = true;
              migrated.push(item);
            });
          });
          captures = migrated;
          delete settings[legacyKey];
          Object.keys(settings).forEach(function (key) {
            if (key.indexOf(workspacePrefix) === 0) delete settings[key];
          });
        } else if (action === 'assign') {
          var workspaceRoot = String(payload.workspaceRootPath || '').replace(/^\/+|\/+$/g, '');
          captures = captures.map(function (capture) {
            return capture.captureId === payload.captureId
              ? Object.assign({}, capture, { workspaceRootPath: workspaceRoot, workspaceName: workspaceRoot })
              : capture;
          });
        } else if (action === 'processed') {
          captures = captures.map(function (capture) {
            return capture.captureId === payload.captureId
              ? Object.assign({}, capture, { processed: payload.processed === true })
              : capture;
          });
        } else if (action === 'archive') {
          var archiveIds = {};
          (Array.isArray(payload.captureIds) ? payload.captureIds : []).forEach(function (id) { archiveIds[id] = true; });
          captures = captures.map(function (capture) {
            return archiveIds[capture.captureId] ? Object.assign({}, capture, { globalState: 'archived' }) : capture;
          });
        } else if (action === 'restore') {
          captures = captures.map(function (capture) {
            return capture.captureId === payload.captureId ? Object.assign({}, capture, { globalState: 'inbox' }) : capture;
          });
        } else if (action === 'delete' && payload.permanent === true) {
          captures = captures.filter(function (capture) { return capture.captureId !== payload.captureId; });
        }
        settings[globalKey] = captures;
        pluginSettings[pluginId] = settings;
      }
      return Promise.resolve('');
    },
    SubscribePluginEvent: function (pluginId, eventName) {
      var s = pluginStates[pluginId];
      if (!s || !s.enabled || s.status !== 'loaded') return Promise.resolve('plugin not enabled and loaded');
      if (!eventName) return Promise.resolve('event name is empty');
      if (!s.manifest.permissions || s.manifest.permissions.indexOf('events.subscribe') === -1) {
        return Promise.resolve('plugin lacks required permission events.subscribe');
      }
      return Promise.resolve('');
    },
    ListVaultFiles: function (pluginId, relativeDir) {
      var err = requirePluginPermission(pluginId, 'files.read');
      if (err) return Promise.resolve([[], err]);
      var norm = normalizeVaultPath(relativeDir, true);
      if (norm.error) return Promise.resolve([[], norm.error]);
      var dir = norm.path;
      if (!vaultFiles[dir] || vaultFiles[dir].type !== 'folder') return Promise.resolve([[], 'not-found: ' + dir]);
      var prefix = dir ? dir + '/' : '';
      var entries = [];
      Object.keys(vaultFiles).forEach(function (path) {
        if (path === dir || path.indexOf(prefix) !== 0) return;
        var rest = path.slice(prefix.length);
        if (!rest || rest.indexOf('/') !== -1) return;
        entries.push(fileEntry(path, vaultFiles[path]));
      });
      return Promise.resolve(listVaultFilesResponseMode === 'plain' ? entries : [entries, '']);
    },
    GetVaultFileMetadata: function (pluginId, relativePath) {
      var err = requirePluginPermission(pluginId, 'files.read');
      if (err) return Promise.resolve([{}, err]);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve([{}, norm.error]);
      var node = vaultFiles[norm.path];
      if (!node) return Promise.resolve([{}, 'not-found: ' + norm.path]);
      return Promise.resolve([fileEntry(norm.path, node), '']);
    },
    ReadVaultTextFile: function (pluginId, relativePath) {
      var err = requirePluginPermission(pluginId, 'files.read');
      if (err) return Promise.resolve(['', err]);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve(['', norm.error]);
      var node = vaultFiles[norm.path];
      if (!node) return Promise.resolve(['', 'not-found: ' + norm.path]);
      if (node.type !== 'file') return Promise.resolve(['', 'not-regular-file: ' + norm.path]);
      return new Promise(function(resolve) {
        setTimeout(function() { resolve([node.content || '', '']); }, readTextDelay);
      });
    },
    ReadVaultFileBytes: function (pluginId, relativePath) {
      var err = requirePluginPermission(pluginId, 'files.read');
      if (err) return Promise.resolve([{}, err]);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve([{}, norm.error]);
      var node = vaultFiles[norm.path];
      if (!node) return Promise.resolve([{}, 'not-found: ' + norm.path]);
      if (node.type !== 'file') return Promise.resolve([{}, 'not-regular-file: ' + norm.path]);
      var content = node.content || '';
      var dataBase64 = typeof btoa === 'function' ? btoa(content) : '';
      return Promise.resolve([{
        relativePath: norm.path,
        size: content.length,
        mimeHint: norm.path.toLowerCase().endsWith('.png') ? 'image/png' : '',
        dataBase64: dataBase64
      }, '']);
    },
    WriteVaultTextFile: function (pluginId, relativePath, content, options) {
      var err = requirePluginPermission(pluginId, 'files.write');
      if (err) return Promise.resolve(err);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve(norm.error);
      options = options || {};
      var existing = vaultFiles[norm.path];
      if (existing && existing.type !== 'file') return Promise.resolve('not-regular-file: ' + norm.path);
      if (existing && !options.overwrite) return Promise.resolve('conflict: ' + norm.path);
      if (!existing && !options.createIfMissing) return Promise.resolve('not-found: ' + norm.path);
      var parent = parentPath(norm.path);
      if (!vaultFiles[parent] || vaultFiles[parent].type !== 'folder') return Promise.resolve('parent-not-found: ' + parent);
      vaultFiles[norm.path] = { type: 'file', content: String(content == null ? '' : content), modifiedAt: new Date().toISOString() };
      return Promise.resolve('');
    },
    WriteVaultFileBytes: function (pluginId, relativePath, dataBase64, options) {
      var err = requirePluginPermission(pluginId, 'files.write');
      if (err) return Promise.resolve(err);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve(norm.error);
      options = options || {};
      var existing = vaultFiles[norm.path];
      if (existing && existing.type !== 'file') return Promise.resolve('not-regular-file: ' + norm.path);
      if (existing && !options.overwrite) return Promise.resolve('conflict: ' + norm.path);
      if (!existing && !options.createIfMissing) return Promise.resolve('not-found: ' + norm.path);
      var parent = parentPath(norm.path);
      if (!vaultFiles[parent] || vaultFiles[parent].type !== 'folder') return Promise.resolve('parent-not-found: ' + parent);
      var content = typeof atob === 'function' ? atob(String(dataBase64 || '')) : '';
      vaultFiles[norm.path] = { type: 'file', content: content, modifiedAt: new Date().toISOString() };
      return Promise.resolve('');
    },
    CreateVaultFolder: function (pluginId, relativePath) {
      var err = requirePluginPermission(pluginId, 'files.write');
      if (err) return Promise.resolve(err);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve(norm.error);
      if (vaultFiles[norm.path]) return Promise.resolve('conflict: ' + norm.path);
      var parent = parentPath(norm.path);
      if (!vaultFiles[parent] || vaultFiles[parent].type !== 'folder') return Promise.resolve('parent-not-found: ' + parent);
      vaultFiles[norm.path] = { type: 'folder', modifiedAt: new Date().toISOString() };
      return Promise.resolve('');
    },
    MoveVaultPath: function (pluginId, fromRelativePath, toRelativePath, options) {
      var err = requirePluginPermission(pluginId, 'files.write');
      if (err) return Promise.resolve(err);
      var from = normalizeVaultPath(fromRelativePath, false);
      var to = normalizeVaultPath(toRelativePath, false);
      if (from.error) return Promise.resolve(from.error);
      if (to.error) return Promise.resolve(to.error);
      options = options || {};
      if (!vaultFiles[from.path]) return Promise.resolve('not-found: ' + from.path);
      if (vaultFiles[from.path].type === 'folder' && (to.path === from.path || to.path.indexOf(from.path + '/') === 0)) {
        return Promise.resolve('move-into-self: ' + from.path + ' -> ' + to.path);
      }
      if (vaultFiles[to.path] && !options.overwrite) return Promise.resolve('conflict: ' + to.path);
      var parent = parentPath(to.path);
      if (!vaultFiles[parent] || vaultFiles[parent].type !== 'folder') return Promise.resolve('parent-not-found: ' + parent);
      var moving = Object.keys(vaultFiles).filter(function (path) { return path === from.path || path.indexOf(from.path + '/') === 0; });
      moving.forEach(function (path) {
        var suffix = path.slice(from.path.length);
        vaultFiles[to.path + suffix] = vaultFiles[path];
        delete vaultFiles[path];
      });
      return Promise.resolve('');
    },
    CopyVaultPath: function (pluginId, fromRelativePath, toRelativePath, options) {
      var readErr = requirePluginPermission(pluginId, 'files.read');
      if (readErr) return Promise.resolve(readErr);
      var writeErr = requirePluginPermission(pluginId, 'files.write');
      if (writeErr) return Promise.resolve(writeErr);
      var from = normalizeVaultPath(fromRelativePath, false);
      var to = normalizeVaultPath(toRelativePath, false);
      if (from.error) return Promise.resolve(from.error);
      if (to.error) return Promise.resolve(to.error);
      options = options || {};
      if (!vaultFiles[from.path]) return Promise.resolve('not-found: ' + from.path);
      if (vaultFiles[from.path].type === 'folder' && (to.path === from.path || to.path.indexOf(from.path + '/') === 0)) {
        return Promise.resolve('copy-into-self: ' + from.path + ' -> ' + to.path);
      }
      if (vaultFiles[to.path] && !options.overwrite) return Promise.resolve('conflict: ' + to.path);
      var parent = parentPath(to.path);
      if (!vaultFiles[parent] || vaultFiles[parent].type !== 'folder') return Promise.resolve('parent-not-found: ' + parent);
      var copying = Object.keys(vaultFiles).filter(function (path) { return path === from.path || path.indexOf(from.path + '/') === 0; });
      copying.forEach(function (path) {
        var suffix = path.slice(from.path.length);
        vaultFiles[to.path + suffix] = Object.assign({}, vaultFiles[path], { modifiedAt: new Date().toISOString() });
      });
      return Promise.resolve('');
    },
    MoveVaultPaths: function (pluginId, transferId, transfers, options) {
      var err = requirePluginPermission(pluginId, 'files.write');
      if (err) return Promise.resolve([emptyTransferOutcome(), err]);
      return runMockTransfers(pluginId, transferId, transfers, 'move', function (transfer) {
        return mock.MoveVaultPath(pluginId, transfer.from, transfer.to, options);
      });
    },
    CopyVaultPaths: function (pluginId, transferId, transfers, options) {
      var readErr = requirePluginPermission(pluginId, 'files.read');
      if (readErr) return Promise.resolve([emptyTransferOutcome(), readErr]);
      var writeErr = requirePluginPermission(pluginId, 'files.write');
      if (writeErr) return Promise.resolve([emptyTransferOutcome(), writeErr]);
      return runMockTransfers(pluginId, transferId, transfers, 'create', function (transfer) {
        return mock.CopyVaultPath(pluginId, transfer.from, transfer.to, options);
      });
    },
    CancelVaultTransfer: function (pluginId, transferId) {
      var err = requirePluginPermission(pluginId, 'files.write');
      if (err) return Promise.resolve(err);
      if (!transferId) return Promise.resolve('cancel requires a transfer id');
      cancelledTransfers[transferId] = true;
      return Promise.resolve('');
    },
    TrashVaultPath: function (pluginId, relativePath) {
      var err = requirePluginPermission(pluginId, 'files.delete');
      if (err) return Promise.resolve([{}, err]);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve([{}, norm.error]);
      if (!vaultFiles[norm.path]) return Promise.resolve([{}, 'not-found: ' + norm.path]);
      var trashId = 'mock-' + Date.now() + '-' + Math.random().toString(16).slice(2);
      var trashPath = '.verstak/trash/files/' + trashId + '/' + baseName(norm.path);
      var originalNode = vaultFiles[norm.path];
      var originalType = originalNode.type || 'file';
      var originalSize = originalType === 'file' ? String(originalNode.content || '').length : 0;
      var moving = Object.keys(vaultFiles).filter(function (path) { return path === norm.path || path.indexOf(norm.path + '/') === 0; });
      trashPayloads[trashId] = moving.map(function (path) {
        return { suffix: path.slice(norm.path.length), entry: Object.assign({}, vaultFiles[path]) };
      });
      moving.forEach(function (path) { delete vaultFiles[path]; });
      var entry = { originalPath: norm.path, trashPath: trashPath, trashId: trashId, deletedAt: new Date().toISOString(), originalType: originalType, basename: baseName(norm.path), size: originalSize };
      trashEntries.unshift(entry);
      return Promise.resolve([entry, '']);
    },
    ListVaultTrash: function (pluginId) {
      var err = requirePluginPermission(pluginId, 'files.delete');
      if (err) return Promise.resolve([[], err]);
      return Promise.resolve([trashEntries.slice(), '']);
    },
    RestoreVaultTrash: function (pluginId, trashId, options) {
      var deleteErr = requirePluginPermission(pluginId, 'files.delete');
      if (deleteErr) return Promise.resolve(['', deleteErr]);
      var writeErr = requirePluginPermission(pluginId, 'files.write');
      if (writeErr) return Promise.resolve(['', writeErr]);
      options = options || {};
      var entry = trashEntries.find(function (item) { return item.trashId === trashId; });
      if (!entry) return Promise.resolve(['', 'not-found: trash entry ' + trashId]);
      var target = normalizeVaultPath(options.targetPath || entry.originalPath, false);
      if (target.error) return Promise.resolve(['', target.error]);
      if (vaultFiles[target.path] && !options.overwrite) return Promise.resolve(['', 'conflict: ' + target.path]);
      var parent = parentPath(target.path);
      if (!vaultFiles[parent] || vaultFiles[parent].type !== 'folder') return Promise.resolve(['', 'parent-not-found: ' + parent]);
      if (options.overwrite) {
        Object.keys(vaultFiles).filter(function (path) { return path === target.path || path.indexOf(target.path + '/') === 0; }).forEach(function (path) { delete vaultFiles[path]; });
      }
      (trashPayloads[trashId] || []).forEach(function (item) {
        vaultFiles[target.path + item.suffix] = Object.assign({}, item.entry, { modifiedAt: new Date().toISOString() });
      });
      delete trashPayloads[trashId];
      trashEntries = trashEntries.filter(function (item) { return item.trashId !== trashId; });
      return Promise.resolve([target.path, '']);
    },
    DeleteVaultTrash: function (pluginId, trashId) {
      var err = requirePluginPermission(pluginId, 'files.delete');
      if (err) return Promise.resolve(err);
      var entry = trashEntries.find(function (item) { return item.trashId === trashId; });
      if (!entry) return Promise.resolve('not-found: trash entry ' + trashId);
      delete trashPayloads[trashId];
      trashEntries = trashEntries.filter(function (item) { return item.trashId !== trashId; });
      return Promise.resolve('');
    },
    OpenVaultPathExternal: function (pluginId, relativePath) {
      var err = requirePluginPermission(pluginId, 'files.openExternal');
      if (err) return Promise.resolve(err);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve(norm.error);
      if (!vaultFiles[norm.path]) return Promise.resolve('not-found: ' + norm.path);
      externalOpens.push({ action: 'open', path: norm.path });
      window.__wailsMockExternalOpens = externalOpens.slice();
      return Promise.resolve('');
    },
    OpenExternalURL: function (pluginId, rawURL) {
      var err = requirePluginPermission(pluginId, 'files.openExternal');
      if (err) return Promise.resolve(err);
      externalOpens.push({ action: 'url', path: String(rawURL || '') });
      window.__wailsMockExternalOpens = externalOpens.slice();
      return Promise.resolve('');
    },
    ShowVaultPathInFolder: function (pluginId, relativePath) {
      var err = requirePluginPermission(pluginId, 'files.openExternal');
      if (err) return Promise.resolve(err);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve(norm.error);
      if (!vaultFiles[norm.path]) return Promise.resolve('not-found: ' + norm.path);
      externalOpens.push({ action: 'show', path: norm.path });
      window.__wailsMockExternalOpens = externalOpens.slice();
      return Promise.resolve('');
    },
    ListWorkspaces: function () {
      return Promise.resolve(listWorkspacesFromTree());
    },
    ListWorkspaceTemplates: function () {
      return Promise.resolve(builtInWorkspaceTemplates().map(function (template) {
        return {
          id: template.id,
          name: template.name,
          description: template.description,
          version: template.version,
          workspaceTools: template.workspaceTools.slice()
        };
      }));
    },
    CreateWorkspace: function (name, templateID) {
      var norm = normalizeVaultPath(name, false);
      if (norm.error || norm.path !== String(name || '').trim() || norm.path.indexOf('/') !== -1) {
        return Promise.resolve(norm.error || 'invalid-workspace-name');
      }
      if (vaultFiles[norm.path]) return Promise.resolve('conflict: ' + norm.path);
      var template = workspaceTemplateByID(templateID);
      if (!template) return Promise.resolve('template-not-found: ' + String(templateID || ''));
      vaultFiles[norm.path] = { type: 'folder', modifiedAt: new Date().toISOString() };
      template.folders.forEach(function (folder) {
        vaultFiles[norm.path + '/' + folder] = { type: 'folder', modifiedAt: new Date().toISOString() };
      });
      workspaceMetadata[norm.path] = metadataForTemplate(norm.path, template);
      workspaceTree.nodes.push(makeWorkspaceNode(norm.path, workspaceTree.nodes.length + 1));
      return Promise.resolve({ name: norm.path, rootPath: norm.path });
    },
    RenameWorkspace: function (oldName, newName) {
      var oldNorm = normalizeVaultPath(oldName, false);
      var newNorm = normalizeVaultPath(newName, false);
      if (oldNorm.error) return Promise.resolve(oldNorm.error);
      if (newNorm.error || newNorm.path.indexOf('/') !== -1) return Promise.resolve(newNorm.error || 'invalid-workspace-name');
      if (!vaultFiles[oldNorm.path]) return Promise.resolve('not-found: ' + oldNorm.path);
      if (vaultFiles[newNorm.path]) return Promise.resolve('conflict: ' + newNorm.path);
      Object.keys(vaultFiles).filter(function (path) {
        return path === oldNorm.path || path.indexOf(oldNorm.path + '/') === 0;
      }).forEach(function (path) {
        var suffix = path.slice(oldNorm.path.length);
        vaultFiles[newNorm.path + suffix] = vaultFiles[path];
        delete vaultFiles[path];
      });
      workspaceTree.nodes = workspaceTree.nodes.map(function (n) {
        if (n.id !== oldNorm.path) return n;
        return makeWorkspaceNode(newNorm.path, n.order);
      });
      if (workspaceMetadata[oldNorm.path]) {
        workspaceMetadata[newNorm.path] = Object.assign({}, workspaceMetadata[oldNorm.path], { workspaceName: newNorm.path });
        delete workspaceMetadata[oldNorm.path];
      }
      if (workspaceTree.currentNodeId === oldNorm.path) workspaceTree.currentNodeId = newNorm.path;
      return Promise.resolve('');
    },
    TrashWorkspace: function (name) {
      var norm = normalizeVaultPath(name, false);
      if (norm.error) return Promise.resolve(norm.error);
      if (!vaultFiles[norm.path]) return Promise.resolve('not-found: ' + norm.path);
      Object.keys(vaultFiles).filter(function (path) {
        return path === norm.path || path.indexOf(norm.path + '/') === 0;
      }).forEach(function (path) { delete vaultFiles[path]; });
      workspaceTree.nodes = workspaceTree.nodes.filter(function (n) { return n.id !== norm.path; });
      delete workspaceMetadata[norm.path];
      if (workspaceTree.currentNodeId === norm.path) workspaceTree.currentNodeId = workspaceTree.nodes[0] ? workspaceTree.nodes[0].id : '';
      return Promise.resolve({ originalPath: norm.path, trashPath: '.verstak/trash/workspaces/mock/' + norm.path, trashId: 'mock', deletedAt: new Date().toISOString() });
    },
    GetWorkspaceMetadata: function (name) {
      var norm = normalizeVaultPath(name, false);
      if (norm.error) return Promise.resolve(norm.error);
      if (!vaultFiles[norm.path]) return Promise.resolve('not-found: ' + norm.path);
      return Promise.resolve(cloneJson(workspaceMetadata[norm.path] || genericWorkspaceMetadata(norm.path)));
    },
    UpdateWorkspaceMetadata: function (name, patch) {
      var norm = normalizeVaultPath(name, false);
      if (norm.error) return Promise.resolve(norm.error);
      if (!vaultFiles[norm.path]) return Promise.resolve('not-found: ' + norm.path);
      var next = Object.assign({}, workspaceMetadata[norm.path] || genericWorkspaceMetadata(norm.path), patch || {}, { workspaceName: norm.path, updatedAt: new Date().toISOString() });
      workspaceMetadata[norm.path] = next;
      return Promise.resolve(cloneJson(next));
    },
    GetCurrentWorkspace: function () {
      var found = workspaceTree.nodes.find(function (n) { return n.id === workspaceTree.currentNodeId; });
      return Promise.resolve(found ? { name: found.name || found.id, rootPath: found.rootPath || found.name || found.id } : null);
    },
    GetCurrentWorkspaceNode: function () {
      var found = workspaceTree.nodes.find(function (n) { return n.id === workspaceTree.currentNodeId; });
      return Promise.resolve(found ? Object.assign({}, found) : null);
    },
    GetWorkspaceTree: function () { return Promise.resolve(cloneWorkspaceTree()); },
    ArchiveWorkspaceNode: function (id) { return this.TrashWorkspace(id).then(function (response) { return typeof response === 'string' ? response : ''; }); },
    CreateWorkspaceNode: function (parentId, nodeType, title) {
      return this.CreateWorkspace(title, 'default').then(function (response) {
        if (typeof response === 'string') return { error: response };
        var ws = response;
        return makeWorkspaceNode(ws.name, workspaceTree.nodes.length);
      });
    },
    MoveWorkspaceNode: function () { return Promise.resolve(''); },
    RenameWorkspaceNode: function (id, title) { return this.RenameWorkspace(id, title); },
    SetCurrentWorkspace: function (id) {
      var found = workspaceTree.nodes.some(function (n) { return n.id === id; });
      if (!found) return Promise.resolve('workspace not found: ' + id);
      workspaceTree.currentNodeId = id;
      return Promise.resolve('');
    },
    SetCurrentWorkspaceNode: function (id) { return this.SetCurrentWorkspace(id); },
    // ── V2 Tree API ──────────────────────────────────────────────────────────
    GetWorkspaceTreeV2: function () {
      return Promise.resolve(workspaceTreeV2Snapshot());
    },
    GetFolderAppearance: function (folderId) {
      var appearance = folderAppearances[folderId] || {};
      return Promise.resolve({ icon: appearance.icon || '', color: appearance.color || '' });
    },
    SetFolderAppearance: function (folderId, patch) {
      var appearance = patch || {};
      var icon = String(appearance.icon || '');
      var color = String(appearance.color || '');
      if (!icon && !color) delete folderAppearances[folderId];
      else folderAppearances[folderId] = { icon: icon, color: color };
      return Promise.resolve('');
    },
    ResetFolderAppearance: function (folderId) {
      delete folderAppearances[folderId];
      return Promise.resolve('');
    },
    PluginListWorkspaces: function (pluginId) {
      var err = requirePluginPermission(pluginId, 'files.read');
      if (err) return Promise.resolve([[], err]);
      var out = [];
      function collect(nodes) {
        (nodes || []).forEach(function (node) {
          var metadata = workspaceMetadata[node.path] || {};
          var workspaceTools = Array.isArray(metadata.workspaceTools) ? metadata.workspaceTools : [];
          if (node.kind === 'workspace' && workspaceTools.indexOf(pluginId) !== -1) {
            out.push({ id: node.id, name: node.name, rootPath: node.path });
          }
          collect(node.children);
        });
      }
      collect(workspaceTreeV2Snapshot().roots);
      return Promise.resolve([out, '']);
    },
    GetWorkspaceByID: function (id) {
      for (var i = 0; i < workspaceTree.nodes.length; i++) {
        var n = workspaceTree.nodes[i];
        if (n.workspaceId === id || n.id === id) {
          return Promise.resolve({ id: n.workspaceId || n.id, name: n.name, rootPath: n.rootPath || n.name });
        }
      }
      return Promise.resolve(null);
    },
    GetFolderByID: function (id) {
      return Promise.resolve(null);
    },
    SetCurrentWorkspaceV2: function (id) {
      var found = workspaceTree.nodes.find(function (node) { return (node.workspaceId || node.id) === id; });
      if (!found) return Promise.resolve('workspace not found: ' + id);
      workspaceTree.currentNodeId = found.id;
      return Promise.resolve('');
    },
    CreateWorkspaceV2: function (parentFolderID, name, templateID) {
      var norm = normalizeVaultPath(name, false);
      if (norm.error || norm.path !== String(name || '').trim() || norm.path.indexOf('/') !== -1) {
        return Promise.resolve({ error: norm.error || 'invalid-workspace-name' });
      }
      if (vaultFiles[norm.path]) return Promise.resolve({ error: 'conflict: ' + norm.path });
      var template = workspaceTemplateByID(templateID || 'default');
      if (!template) return Promise.resolve({ error: 'template-not-found: ' + String(templateID || '') });
      vaultFiles[norm.path] = { type: 'folder', modifiedAt: new Date().toISOString() };
      template.folders.forEach(function (folder) {
        vaultFiles[norm.path + '/' + folder] = { type: 'folder', modifiedAt: new Date().toISOString() };
      });
      workspaceMetadata[norm.path] = metadataForTemplate(norm.path, template);
      var node = makeWorkspaceNodeV2(norm.path, workspaceTree.nodes.length + 1);
      workspaceTree.nodes.push(node);
      return Promise.resolve({ id: node.workspaceId || node.id, name: norm.path, rootPath: norm.path });
    },
    CreateWorkspaceV2WithTools: function (parentFolderID, name, templateID, workspaceTools) {
      var norm = normalizeVaultPath(name, false);
      if (norm.error || norm.path !== String(name || '').trim() || norm.path.indexOf('/') !== -1) {
        return Promise.resolve({ error: norm.error || 'invalid-workspace-name' });
      }
      if (vaultFiles[norm.path]) return Promise.resolve({ error: 'conflict: ' + norm.path });
      var eligible = allPlugins().filter(function (plugin) {
        return (plugin.manifest && plugin.manifest.contributes && plugin.manifest.contributes.workspaceItems || []).length > 0;
      }).map(function (plugin) { return plugin.manifest.id; });
      var tools = Array.isArray(workspaceTools) ? workspaceTools.slice() : [];
      var invalid = tools.find(function (toolID) { return eligible.indexOf(toolID) === -1; });
      if (invalid) return Promise.resolve({ error: 'workspace tool is not available: ' + invalid });
      var template = workspaceTemplateByID(templateID || 'default');
      if (!template && templateID !== 'custom') return Promise.resolve({ error: 'template-not-found: ' + String(templateID || '') });
      template = template || { id: 'custom', name: 'Custom', version: 1, folders: ['Notes', 'Files'], features: {}, workspaceTools: [] };
      vaultFiles[norm.path] = { type: 'folder', modifiedAt: new Date().toISOString() };
      template.folders.forEach(function (folder) {
        vaultFiles[norm.path + '/' + folder] = { type: 'folder', modifiedAt: new Date().toISOString() };
      });
      if (tools.indexOf('verstak.secrets') !== -1) {
        vaultFiles[norm.path + '/Secrets'] = { type: 'folder', modifiedAt: new Date().toISOString() };
      }
      var metadata = metadataForTemplate(norm.path, template);
      metadata.workspaceTools = tools.slice();
      metadata.features = {};
      metadata.folders = {};
      tools.forEach(function (toolID) {
        var key = toolID.replace('verstak.', '');
        metadata.features[key] = true;
        if (key === 'notes') metadata.folders.notes = 'Notes';
        if (key === 'files') metadata.folders.files = 'Files';
        if (key === 'secrets') metadata.folders.secrets = 'Secrets';
      });
      workspaceMetadata[norm.path] = metadata;
      var node = makeWorkspaceNodeV2(norm.path, workspaceTree.nodes.length + 1);
      workspaceTree.nodes.push(node);
      return Promise.resolve({ id: node.workspaceId || node.id, name: norm.path, rootPath: norm.path });
    },
    CreateFolderV2: function (parentFolderID, name) {
      return Promise.resolve({ id: 'folder-' + Math.random().toString(36).slice(2, 10), name: name, path: name });
    },
    RenameWorkspaceV2: function (workspaceID, newName) {
      var found = workspaceTree.nodes.find(function (node) { return (node.workspaceId || node.id) === workspaceID; });
      return found ? this.RenameWorkspace(found.id, newName) : Promise.resolve('workspace not found: ' + workspaceID);
    },
    RenameFolderV2: function (folderID, newName) { return Promise.resolve(''); },
    MoveWorkspaceV2: function (workspaceID, targetParentFolderID) { return Promise.resolve(''); },
    MoveFolderV2: function (folderID, targetParentFolderID) { return Promise.resolve(''); },
    PlaceWorkspaceTreeNodeV2: function (request) {
      treePlacementRequests.push(cloneJson(request || {}));
      // The real backend emits this from the placement itself, which makes the
      // sidebar reload the tree a second time, concurrently with the reload
      // the caller does. Without it here the mock hid a race.
      if (!treePlacementError) {
        setTimeout(function () {
          window.dispatchEvent(new CustomEvent('verstak:workspace-tree-changed'));
        }, 0);
      }
      return Promise.resolve(treePlacementError);
    },
    TrashWorkspaceV2: function (workspaceID) {
      var found = workspaceTree.nodes.find(function (node) { return (node.workspaceId || node.id) === workspaceID; });
      return found ? this.TrashWorkspace(found.id) : Promise.resolve('workspace not found: ' + workspaceID);
    },
    RescanWorkspaceTree: function () { return Promise.resolve(''); },
    GetWorkspaceTreeDiagnostics: function () { return Promise.resolve([]); },
    // ── End V2 Tree API ──────────────────────────────────────────────────────
    SelectDirectory: function () { return Promise.resolve(''); },
    SelectVaultForOpen: function () { return Promise.resolve(''); },
    CreateVault: function () { return Promise.resolve(null); },
    OpenVault: function () { return Promise.resolve(null); },
    CloseVault: function () { return Promise.resolve(null); },
    SetCurrentVault: function () { return Promise.resolve(''); },
    UpdateAppSettings: function (patch) {
      // The real call crosses the Wails bridge and writes a file, so a read
      // that starts before it completes still sees the old value. Applying the
      // patch synchronously here hid a race where a concurrent tree reload
      // restored settings from before the write.
      return new Promise(function (resolve) {
        setTimeout(function () {
          applyAppSettingsPatch(patch);
          resolve('');
        }, 0);
      });
    },
    RecordDesiredPlugin: function () { return Promise.resolve(''); },
    WriteFrontendLog: function () { return Promise.resolve(); },

    EnablePlugin: function (pluginId) {
      if (pluginStates[pluginId]) {
        pluginStates[pluginId].status = 'loaded';
        pluginStates[pluginId].enabled = true;
        if (vaultPluginState.disabledPlugins.indexOf(pluginId) !== -1) {
          vaultPluginState.disabledPlugins = vaultPluginState.disabledPlugins.filter(function (id) { return id !== pluginId; });
        }
        if (vaultPluginState.enabledPlugins.indexOf(pluginId) === -1) {
          vaultPluginState.enabledPlugins.push(pluginId);
        }
      }
      return Promise.resolve(null);
    },

    DisablePlugin: function (pluginId) {
      if (pluginStates[pluginId]) {
        pluginStates[pluginId].status = 'disabled';
        pluginStates[pluginId].enabled = false;
        if (vaultPluginState.enabledPlugins.indexOf(pluginId) !== -1) {
          vaultPluginState.enabledPlugins = vaultPluginState.enabledPlugins.filter(function (id) { return id !== pluginId; });
        }
        if (vaultPluginState.disabledPlugins.indexOf(pluginId) === -1) {
          vaultPluginState.disabledPlugins.push(pluginId);
        }
      }
      return Promise.resolve(null);
    },

    ReloadPlugins: function () {
      if (reloadResponseMode === 'raw-count') {
        return Promise.resolve(Object.keys(pluginStates).length);
      }
      return Promise.resolve([Object.keys(pluginStates).length, 'Reloaded ' + Object.keys(pluginStates).length + ' plugin(s).']);
    }
  };

  // ── Install bridge ─────────────────────────────────────────────────
  if (!window['go']) window['go'] = {};
  if (!window['go']['api']) window['go']['api'] = {};
  window['go']['api']['App'] = mock;

  // ── Test helpers (exposed for Playwright) ──────────────────────────
  window.__wailsMock = {
    reset: function () {
      pluginStates = makeDefaultPluginStates();
      vaultStatus = { status: 'open', path: '/tmp/verstak-test/vault', vaultId: 'test-vault-001' };
      vaultPluginState = makeDefaultVaultPluginState();
      appSettings = { currentVaultPath: '/tmp/verstak-test/vault', recentVaults: [], language: 'system', sidebarWidth: 220, expandedFolderIds: [], settingsSection: '' };
      workbenchPreferences = {};
      openedResources = [];
      pluginSettings = { 'verstak.platform-test': { savedText: 'initial value' } };
      pluginNotifications = {};
      pluginData = {};
      folderAppearances = {};
      secretRecords = makeDefaultSecretRecords();
      vaultFiles = makeDefaultVaultFiles();
      externalOpens = [];
      trashEntries = [];
      trashPayloads = {};
      window.__wailsMockExternalOpens = [];
      workspaceTree = makeDefaultWorkspaceTree();
      workspaceTreeV2Override = null;
      treePlacementRequests = [];
      treePlacementError = '';
      workspaceMetadata = makeDefaultWorkspaceMetadata();
      reloadResponseMode = 'tuple';
      listVaultFilesResponseMode = 'tuple';
      syncState = makeDefaultSyncState();
      readTextDelay = 0;
      importSessions = {};
      importSequence = 0;
      importRunCounts = { dokuwiki: 0, obsidian: 0 };
    },
    setPluginStatus: function (pluginId, status, enabled) {
      if (pluginStates[pluginId]) {
        pluginStates[pluginId].status = status;
        pluginStates[pluginId].enabled = enabled;
      }
    },
    getPluginState: function (pluginId) {
      return pluginStates[pluginId] ? Object.assign({}, pluginStates[pluginId]) : null;
    },
    getOpenImportSessionCount: function () {
      return Object.keys(importSessions).filter(function (handle) { return !importSessions[handle].closed; }).length;
    },
    addSyntheticPlugins: function (count, source) {
      var total = Number(count || 0);
      var pluginSource = source === 'official' || source === 'local' || source === 'third-party' ? source : 'third-party';
      for (var i = 1; i <= total; i++) {
        var id = 'verstak.synthetic-layout-' + String(i).padStart(2, '0');
        pluginStates[id] = {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: id,
            name: 'Synthetic Layout Plugin ' + i,
            version: '0.0.' + i,
            apiVersion: '0.1.0',
            description: 'Synthetic plugin used by frontend layout tests.',
            source: pluginSource,
            provides: ['verstak/synthetic-layout-' + i + '/v1'],
            requires: [],
            optionalRequires: [],
            permissions: [],
            contributes: {
              views: [],
              commands: [],
              sidebarItems: [],
              statusBarItems: [],
              settingsPanels: []
            }
          },
          rootPath: '/tmp/verstak-test/plugins/synthetic-layout-' + i + '/with/a/long/path/for/responsive-checks',
          error: ''
        };
        if (vaultPluginState.enabledPlugins.indexOf(id) === -1) {
          vaultPluginState.enabledPlugins.push(id);
        }
        if (!vaultPluginState.desiredPlugins.some(function (p) { return p.id === id; })) {
          vaultPluginState.desiredPlugins.push({ id: id, version: '0.0.' + i, source: pluginSource });
        }
      }
    },
    setVaultStatus: function (status) { vaultStatus = status; },
    setVaultPluginState: function (state) { vaultPluginState = state; },
    setTrashDeletedAt: function (trashId, deletedAt) {
      trashEntries.forEach(function (entry) {
        if (entry.trashId === trashId) entry.deletedAt = deletedAt;
      });
    },
    setReloadResponseMode: function (mode) { reloadResponseMode = mode || 'tuple'; },
    setListVaultFilesResponseMode: function (mode) { listVaultFilesResponseMode = mode || 'tuple'; },
    setReadTextDelay: function (delay) { readTextDelay = Math.max(0, Number(delay || 0)); },
    setWorkspaceTreeV2: function (snapshot) {
      workspaceTreeV2Override = cloneJson(snapshot);
      treePlacementRequests = [];
      treePlacementError = '';
    },
    getTreePlacementRequests: function () { return cloneJson(treePlacementRequests); },
    setTreePlacementError: function (message) { treePlacementError = String(message || ''); },
    putVaultFile: function (relativePath, content) {
      var path = String(relativePath || '').replace(/^\/+|\/+$/g, '');
      var parts = path.split('/');
      for (var i = 1; i < parts.length; i++) {
        var dir = parts.slice(0, i).join('/');
        if (!vaultFiles[dir]) vaultFiles[dir] = { type: 'folder', modifiedAt: new Date().toISOString() };
      }
      vaultFiles[path] = { type: 'file', content: String(content || ''), modifiedAt: new Date().toISOString() };
    }
  };

  window.__wailsMockReady = true;
  console.log('[wails-mock] bridge installed');
})();
