/**
 * Wails Mock Bridge — эмулирует window['go']['api']['App'] для тестового окружения.
 *
 * Каждый метод возвращает Promise с данными, совместимыми с Wails-контрактом.
 * Состояние мутабельно — тесты могут менять его между сценариями.
 */
import defaultEditorSource from '../../../../../verstak-official-plugins/plugins/default-editor/frontend/src/index.js?raw';
import filesSource from '../../../../../verstak-official-plugins/plugins/files/frontend/src/index.js?raw';
import filePreviewSource from '../../../../../verstak-official-plugins/plugins/file-preview/frontend/src/index.js?raw';
import trashSource from '../../../../../verstak-official-plugins/plugins/trash/frontend/src/index.js?raw';
import searchSource from '../../../../../verstak-official-plugins/plugins/search/frontend/src/index.js?raw';
import platformTestSource from '../../../../../verstak-official-plugins/plugins/platform-test/frontend/src/index.js?raw';
import filesManifest from '../../../../../verstak-official-plugins/plugins/files/plugin.json';
import platformTestManifest from '../../../../../verstak-official-plugins/plugins/platform-test/plugin.json';
import defaultEditorManifest from '../../../../../verstak-official-plugins/plugins/default-editor/plugin.json';
import filePreviewManifest from '../../../../../verstak-official-plugins/plugins/file-preview/plugin.json';
import trashManifest from '../../../../../verstak-official-plugins/plugins/trash/plugin.json';
import notesManifest from '../../../../../verstak-official-plugins/plugins/notes/plugin.json';
import projectsManifest from '../../../../../verstak-official-plugins/plugins/projects/plugin.json';
import syncManifest from '../../../../../verstak-official-plugins/plugins/sync/plugin.json';
import activityManifest from '../../../../../verstak-official-plugins/plugins/activity/plugin.json';
import journalManifest from '../../../../../verstak-official-plugins/plugins/journal/plugin.json';
import browserInboxManifest from '../../../../../verstak-official-plugins/plugins/browser-inbox/plugin.json';
import todoManifest from '../../../../../verstak-official-plugins/plugins/todo/plugin.json';
import secretsManifest from '../../../../../verstak-official-plugins/plugins/secrets/plugin.json';
import importManifest from '../../../../../verstak-official-plugins/plugins/import/plugin.json';
import searchManifest from '../../../../../verstak-official-plugins/plugins/search/plugin.json';
import templatesManifest from '../../../../../verstak-official-plugins/plugins/templates/plugin.json';
import milestonesManifest from '../../../../../verstak-official-plugins/plugins/milestones/plugin.json';
import gitManifest from '../../../../../verstak-official-plugins/plugins/git/plugin.json';
import notesSource from '../../../../../verstak-official-plugins/plugins/notes/frontend/src/index.js?raw';
import projectsSource from '../../../../../verstak-official-plugins/plugins/projects/frontend/src/index.js?raw';
import browserInboxSource from '../../../../../verstak-official-plugins/plugins/browser-inbox/frontend/src/index.js?raw';
import secretsSource from '../../../../../verstak-official-plugins/plugins/secrets/frontend/src/index.js?raw';
import activitySource from '../../../../../verstak-official-plugins/plugins/activity/frontend/src/index.js?raw';
import todoSource from '../../../../../verstak-official-plugins/plugins/todo/frontend/src/index.js?raw';
import journalSource from '../../../../../verstak-official-plugins/plugins/journal/frontend/src/index.js?raw';
import templatesSource from '../../../../../verstak-official-plugins/plugins/templates/frontend/src/index.js?raw';
import gitSource from '../../../../../verstak-official-plugins/plugins/git/frontend/src/index.js?raw';
import notesEnCatalog from '../../../../../verstak-official-plugins/plugins/notes/locales/en.json';
import notesRuCatalog from '../../../../../verstak-official-plugins/plugins/notes/locales/ru.json';
import projectsEnCatalog from '../../../../../verstak-official-plugins/plugins/projects/locales/en.json';
import projectsRuCatalog from '../../../../../verstak-official-plugins/plugins/projects/locales/ru.json';
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
import gitEnCatalog from '../../../../../verstak-official-plugins/plugins/git/locales/en.json';
import gitRuCatalog from '../../../../../verstak-official-plugins/plugins/git/locales/ru.json';
import importSource from '../../../../../verstak-official-plugins/plugins/import/frontend/dist/index.js?raw';
import importStyle from '../../../../../verstak-official-plugins/plugins/import/frontend/dist/style.css?raw';
// Sync ships built output rather than raw source, so its manifest entry points
// at frontend/dist. Running the E2E therefore needs the official plugins built
// first, exactly as CI already does before this suite.
import syncSource from '../../../../../verstak-official-plugins/plugins/sync/frontend/dist/index.js?raw';
import syncStyle from '../../../../../verstak-official-plugins/plugins/sync/frontend/dist/style.css?raw';

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
    [projectsManifest, 'projects'],
    [syncManifest, 'sync'],
    [activityManifest, 'activity'],
    [journalManifest, 'journal'],
    [browserInboxManifest, 'browser-inbox'],
    [todoManifest, 'todo'],
    [secretsManifest, 'secrets'],
    [importManifest, 'import'],
    [searchManifest, 'search'],
    [templatesManifest, 'templates'],
    [milestonesManifest, 'milestones'],
    [gitManifest, 'git']
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
    'verstak.projects': { en: projectsEnCatalog, ru: projectsRuCatalog },
    'verstak.activity': { en: activityEnCatalog, ru: activityRuCatalog },
    'verstak.browser-inbox': { en: browserEnCatalog, ru: browserRuCatalog },
    'verstak.file-preview': { en: filePreviewEnCatalog, ru: filePreviewRuCatalog },
    'verstak.journal': { en: journalEnCatalog, ru: journalRuCatalog },
    'verstak.todo': { en: todoEnCatalog, ru: todoRuCatalog },
    'verstak.git': { en: gitEnCatalog, ru: gitRuCatalog }
  };

  var russianPluginNames = {
    'verstak.platform-test': 'Тест платформы',
    'verstak.default-editor': 'Стандартный редактор',
    'verstak.files': 'Файлы',
    'verstak.notes': 'Заметки',
    'verstak.projects': 'Проекты',
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
  var pluginDealConfig = {};
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
  var workspaceSequence = 3;
  var gitCheckouts = {};

  // ── Helpers ────────────────────────────────────────────────────────
  function makeDefaultWorkspaceTree() {
    return {
      status: 'initialized',
      currentNodeId: 'Project',
      nodes: [
        { id: 'Project', workspaceId: '11111111-1111-4111-8111-111111111111', parentId: '', type: 'space', title: 'Project', name: 'Project', rootPath: 'Project', status: 'active', order: 1 },
        { id: 'Test', workspaceId: '22222222-2222-4222-8222-222222222222', parentId: '', type: 'space', title: 'Test', name: 'Test', rootPath: 'Test', status: 'active', order: 2 }
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
      .map(function (n) { return { id: n.workspaceId || n.id, name: n.name || n.id, rootPath: n.rootPath || n.name || n.id }; });
  }

  function makeWorkspaceNode(name, order) {
    var node = makeWorkspaceNodeV2(name, order);
    return Object.assign(node, { parentId: '', type: 'space', title: name, status: 'active' });
  }

  function makeWorkspaceNodeV2(name, order) {
    var wsid = '00000000-0000-4000-8000-' + String(workspaceSequence++).padStart(12, '0');
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

  function findWorkspaceNodeV2(id) {
    var wanted = String(id || '');
    var found = null;
    function walk(nodes) {
      (nodes || []).some(function (node) {
        if (node.kind === 'workspace' && String(node.id || '') === wanted) {
          found = node;
          return true;
        }
        return walk(node.children || []);
      });
      return !!found;
    }
    walk(workspaceTreeV2Snapshot().roots);
    return found;
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

  function metadataForRecipe(name, recipe) {
    var now = new Date().toISOString();
    var tools = Array.isArray(recipe && recipe.workspaceTools) ? recipe.workspaceTools.slice() : [];
    var metadata = {
      workspaceName: name,
      features: {},
      folders: {},
      workspaceTools: tools,
      updatedAt: now,
    };
    tools.forEach(function (toolID) {
      var key = toolID.replace('verstak.', '');
      metadata.features[key] = true;
      if (key === 'notes') metadata.folders.notes = 'Notes';
      if (key === 'files') metadata.folders.files = 'Files';
      if (key === 'secrets') metadata.folders.secrets = 'Secrets';
    });
    if (recipe && recipe.provenance) {
      metadata.createdFromTemplate = {
        templateId: recipe.provenance.templateId,
        templateName: recipe.provenance.templateName || '',
        templateVersion: recipe.provenance.templateVersion || 0,
        appliedAt: now,
        workspaceTools: tools.slice(),
      };
    }
    return metadata;
  }

  function createWorkspaceFromRecipe(name, recipe) {
    var norm = normalizeVaultPath(name, false);
    if (norm.error || norm.path !== String(name || '').trim() || norm.path.indexOf('/') !== -1) {
      return { error: norm.error || 'invalid-workspace-name' };
    }
    if (vaultFiles[norm.path]) return { error: 'conflict: ' + norm.path };
    var tools = Array.isArray(recipe && recipe.workspaceTools) ? recipe.workspaceTools.slice() : [];
    var eligible = allPlugins().filter(function (plugin) {
      return (plugin.manifest && plugin.manifest.contributes && plugin.manifest.contributes.workspaceItems || []).length > 0;
    }).map(function (plugin) { return plugin.manifest.id; });
    var unavailable = tools.find(function (toolID) { return eligible.indexOf(toolID) === -1; });
    if (unavailable) return { error: 'workspace tool is not available: ' + unavailable };
    vaultFiles[norm.path] = { type: 'folder', modifiedAt: new Date().toISOString() };
    (recipe.initialFolders || []).forEach(function (folder) {
      vaultFiles[norm.path + '/' + folder] = { type: 'folder', modifiedAt: new Date().toISOString() };
    });
    (recipe.initialFiles || []).forEach(function (file) {
      if (file && file.path) vaultFiles[norm.path + '/' + file.path] = { type: 'file', content: String(file.content || ''), modifiedAt: new Date().toISOString() };
    });
    var node = makeWorkspaceNode(norm.path, workspaceTree.nodes.length + 1);
    workspaceMetadata[norm.path] = metadataForRecipe(norm.path, recipe || {});
    workspaceTree.nodes.push(node);
    return { id: node.workspaceId, name: node.name, rootPath: node.rootPath };
  }

  function makeDefaultWorkspaceMetadata() {
    var fixtureRecipe = { workspaceTools: ['verstak.projects', 'verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.journal', 'verstak.activity', 'verstak.browser-inbox'] };
    return {
      Project: metadataForRecipe('Project', fixtureRecipe),
      Test: metadataForRecipe('Test', fixtureRecipe),
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

  function gitCheckoutKey(request) {
    request = request || {};
    return String(request.workspaceId || '') + ':' + String(request.repositoryId || '');
  }

  function gitNotClonedStatus() {
    return { state: 'not-cloned', branch: '', clean: true, changedCount: 0, untrackedCount: 0, changedFiles: [], ahead: 0, behind: 0, recentCommits: [] };
  }

  function gitClonedStatus(request) {
    return {
      state: 'cloned', branch: String(request.branch || 'main'), clean: true, changedCount: 0, untrackedCount: 0,
      changedFiles: [], ahead: 0, behind: 0,
      recentCommits: [{ id: '0123456789012345678901234567890123456789', shortId: '0123456', subject: 'Initial mock commit', author: 'Verstak', committed: '2026-08-30T00:00:00Z' }]
    };
  }

  function requireGitPermission(pluginId, remote, extra) {
    var err = requirePluginPermission(pluginId, 'process.spawn');
    if (err) return err;
    if (remote) { err = requirePluginPermission(pluginId, 'network.remote'); if (err) return err; }
    if (extra) { err = requirePluginPermission(pluginId, extra); if (err) return err; }
    return '';
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
      if (!s || s.status !== 'loaded' || !s.enabled) continue;
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
    ReadPluginDealConfig: function (pluginId, workspaceId) {
      var err = requirePluginPermission(pluginId, 'storage.namespace');
      if (err) return Promise.resolve([{}, err]);
      return Promise.resolve([cloneJson((pluginDealConfig[pluginId] || {})[workspaceId] || {}), '']);
    },
    WritePluginDealConfig: function (pluginId, workspaceId, config) {
      var err = requirePluginPermission(pluginId, 'storage.namespace');
      if (err) return Promise.resolve(err);
      if (!findWorkspaceNodeV2(workspaceId)) return Promise.resolve('workspace not found: ' + workspaceId);
      pluginDealConfig[pluginId] = pluginDealConfig[pluginId] || {};
      pluginDealConfig[pluginId][workspaceId] = cloneJson(config || {});
      return Promise.resolve('');
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
    PluginGitClone: function (pluginId, request) {
      var err = requireGitPermission(pluginId, true);
      if (err) return Promise.resolve([{}, err]);
      var key = gitCheckoutKey(request);
      gitCheckouts[key] = gitClonedStatus(request);
      return Promise.resolve([{ checkoutPath: 'Project/Repositories/' + String(request.checkoutName || 'repository') }, '']);
    },
    PluginGitRegisterExisting: function (pluginId, request) {
      var err = requireGitPermission(pluginId, false, 'imports.readExternal');
      if (err) return Promise.resolve([{}, err]);
      var key = gitCheckoutKey(request);
      gitCheckouts[key] = gitClonedStatus(request);
      return Promise.resolve([{ checkoutPath: 'Project/Repositories/' + String(request.checkoutName || 'repository') }, '']);
    },
    PluginGitStatus: function (pluginId, request) {
      var err = requireGitPermission(pluginId, false);
      if (err) return Promise.resolve([{}, err]);
      return Promise.resolve([cloneJson(gitCheckouts[gitCheckoutKey(request)] || gitNotClonedStatus()), '']);
    },
    PluginGitFetch: function (pluginId, request) {
      var err = requireGitPermission(pluginId, true);
      return Promise.resolve(err || '');
    },
    PluginGitPull: function (pluginId, request) {
      var err = requireGitPermission(pluginId, true);
      if (err) return Promise.resolve(err);
      var status = gitCheckouts[gitCheckoutKey(request)]; if (status) status.behind = 0;
      return Promise.resolve('');
    },
    PluginGitPush: function (pluginId, request) {
      var err = requireGitPermission(pluginId, true);
      if (err) return Promise.resolve(err);
      var status = gitCheckouts[gitCheckoutKey(request)]; if (status) status.ahead = 0;
      return Promise.resolve('');
    },
    PluginGitOpenDirectory: function (pluginId, request) {
      var err = requireGitPermission(pluginId, false, 'files.openExternal');
      if (err) return Promise.resolve(err);
      externalOpens.push({ action: 'git-folder', path: String(request.checkoutName || '') });
      window.__wailsMockExternalOpens = externalOpens.slice();
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
        return Promise.resolve(platformTestSource);
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
        return Promise.resolve(trashSource);
      }
      if (pluginId === notesManifest.id && assetPath === notesManifest.frontend.entry) {
        return Promise.resolve(notesSource);
      }
      if (pluginId === projectsManifest.id && assetPath === projectsManifest.frontend.entry) {
        return Promise.resolve(projectsSource);
      }
      if (pluginId === syncManifest.id && assetPath === syncManifest.frontend.entry) {
        return Promise.resolve(syncSource);
      }
      if (pluginId === syncManifest.id && assetPath === syncManifest.frontend.style) {
        return Promise.resolve(syncStyle);
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
        return Promise.resolve(searchSource);
      }
      if (pluginId === importManifest.id && assetPath === importManifest.frontend.entry) {
        return Promise.resolve(importSource);
      }
      if (pluginId === importManifest.id && assetPath === importManifest.frontend.style) {
        return Promise.resolve(importStyle);
      }
      if (pluginId === templatesManifest.id && assetPath === templatesManifest.frontend.entry) {
        return Promise.resolve(templatesSource);
      }
      if (pluginId === gitManifest.id && assetPath === gitManifest.frontend.entry) {
        return Promise.resolve(gitSource);
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
      return Promise.resolve(found ? { workspaceId: found.workspaceId || found.id, name: found.name || found.id, rootPath: found.rootPath || found.name || found.id } : null);
    },
    GetCurrentWorkspaceNode: function () {
      var found = workspaceTree.nodes.find(function (n) { return n.id === workspaceTree.currentNodeId; });
      return Promise.resolve(found ? Object.assign({}, found) : null);
    },
    GetWorkspaceTree: function () { return Promise.resolve(cloneWorkspaceTree()); },
    ArchiveWorkspaceNode: function (id) { return this.TrashWorkspace(id).then(function (response) { return typeof response === 'string' ? response : ''; }); },
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
    PluginCreateWorkspace: function (pluginId, parentFolderID, name, recipe) {
      var err = requirePluginPermission(pluginId, 'workspaces.create');
      if (err) return Promise.resolve([null, err]);
      if (!recipe || typeof recipe !== 'object' || Array.isArray(recipe) || !recipe.provenance || !recipe.provenance.templateId) {
        return Promise.resolve([null, 'recipe provenance template ID is required']);
      }
      var created = createWorkspaceFromRecipe(name, recipe);
      if (created.error) return Promise.resolve([null, created.error]);
      window.dispatchEvent(new CustomEvent('verstak:workspace-tree-changed'));
      return Promise.resolve([{ workspaceId: created.id, name: created.name }, '']);
    },
    GetWorkspaceByID: function (id) {
      var v2 = findWorkspaceNodeV2(id);
      if (v2) {
        return Promise.resolve({ id: v2.id, name: v2.name, rootPath: v2.path || v2.rootPath || v2.name });
      }
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
      var v2 = findWorkspaceNodeV2(id);
      if (v2) {
        if (workspaceTreeV2Override) workspaceTreeV2Override.currentWorkspaceId = v2.id;
        workspaceTree.currentNodeId = v2.path || v2.name || v2.id;
        return Promise.resolve('');
      }
      var found = workspaceTree.nodes.find(function (node) { return (node.workspaceId || node.id) === id; });
      if (!found) return Promise.resolve('workspace not found: ' + id);
      workspaceTree.currentNodeId = found.id;
      return Promise.resolve('');
    },
    UpdateWorkspaceV2Tools: function (workspaceID, workspaceTools) {
      var found = workspaceTree.nodes.find(function (node) { return (node.workspaceId || node.id) === workspaceID; });
      if (!found) return Promise.resolve('workspace not found: ' + workspaceID);
      var eligible = allPlugins().filter(function (plugin) {
        return (plugin.manifest && plugin.manifest.contributes && plugin.manifest.contributes.workspaceItems || []).length > 0;
      }).map(function (plugin) { return plugin.manifest.id; });
      var tools = Array.isArray(workspaceTools) ? workspaceTools.filter(function (toolID, index, values) {
        return toolID && values.indexOf(toolID) === index;
      }) : [];
      var invalid = tools.find(function (toolID) { return eligible.indexOf(toolID) === -1; });
      if (invalid) return Promise.resolve('workspace tool is not available: ' + invalid);
      var rootPath = found.rootPath || found.name || found.id;
      var metadata = Object.assign({}, workspaceMetadata[rootPath] || genericWorkspaceMetadata(rootPath));
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
      Object.keys(metadata.folders).forEach(function (key) {
        var folder = metadata.folders[key];
        if (!vaultFiles[rootPath + '/' + folder]) vaultFiles[rootPath + '/' + folder] = { type: 'folder', modifiedAt: new Date().toISOString() };
      });
      metadata.updatedAt = new Date().toISOString();
      workspaceMetadata[rootPath] = metadata;
      return Promise.resolve('');
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
      pluginDealConfig = {};
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
      workspaceSequence = 3;
      gitCheckouts = {};
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
