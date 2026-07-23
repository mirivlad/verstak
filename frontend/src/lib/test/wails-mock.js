/**
 * Wails Mock Bridge — эмулирует window['go']['api']['App'] для тестового окружения.
 *
 * Каждый метод возвращает Promise с данными, совместимыми с Wails-контрактом.
 * Состояние мутабельно — тесты могут менять его между сценариями.
 */
import defaultEditorSource from '../../../../../verstak-official-plugins/plugins/default-editor/frontend/src/index.js?raw';
import secretsSource from '../../../../../verstak-official-plugins/plugins/secrets/frontend/src/index.js?raw';
import activitySource from '../../../../../verstak-official-plugins/plugins/activity/frontend/src/index.js?raw';
import todoSource from '../../../../../verstak-official-plugins/plugins/todo/frontend/src/index.js?raw';
import journalSource from '../../../../../verstak-official-plugins/plugins/journal/frontend/src/index.js?raw';
import importSource from '../../../../../verstak-official-plugins/plugins/import/frontend/dist/index.js?raw';
import importStyle from '../../../../../verstak-official-plugins/plugins/import/frontend/dist/style.css?raw';

(function () {
  if (window.__wailsMockReady) return;

  // ── Mutable state ──────────────────────────────────────────────────
  var pluginStates = {
    'verstak.platform-test': {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.platform-test',
        name: 'Platform Test',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Runtime test plugin for verifying the Verstak platform.',
        source: 'official',
        icon: '🧪',
        provides: ['verstak/platform-test/v1', 'verstak/diagnostics/v1'],
        requires: ['verstak/core/plugin-manager/v1', 'verstak/core/capability-registry/v1'],
        optionalRequires: ['verstak/core/vault/v1', 'verstak/core/sync/v1', 'verstak/core/files/v1', 'verstak/core/workbench/v1'],
        permissions: ['vault.read', 'events.publish', 'events.subscribe', 'ui.register', 'commands.register', 'storage.namespace', 'files.read', 'files.write', 'files.delete', 'files.openExternal', 'workbench.open'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          views: [
            { id: 'verstak.platform-test.diagnostics', title: 'Platform Diagnostics', icon: '🧪', component: 'DiagnosticsPanel' }
          ],
          commands: [
            { id: 'verstak.platform-test.run-tests', title: 'Run Platform Tests', handler: 'runAllTests' },
            { id: 'verstak.platform-test.show-version', title: 'Show Version Info', handler: 'showVersion' }
          ],
          sidebarItems: [
            { id: 'verstak.platform-test.sidebar', title: 'Platform Test', icon: '🧪', view: 'verstak.platform-test.diagnostics', position: 100 }
          ],
          statusBarItems: [
            { id: 'verstak.platform-test.status', label: '🧪 All Tests Pass', position: 'right', handler: 'openDiagnostics' }
          ],
          settingsPanels: [
            { id: 'verstak.platform-test.settings', title: 'Platform Test Settings', icon: '🧪', component: 'PlatformTestSettings' }
          ],
          openProviders: [
            {
              id: 'verstak.platform-test.markdown-diagnostic',
              title: 'Platform Test Markdown Diagnostic',
              priority: 10,
              component: 'MarkdownDiagnosticProvider',
              supports: [
                { kind: 'vault-file', extensions: ['.md', '.markdown'], contexts: ['generic-markdown', 'notes-markdown'] }
              ]
            }
          ]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/platform-test',
      error: ''
    },
    'verstak.default-editor': {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.default-editor',
        name: 'Default Editor',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Built-in text and markdown editor/viewer.',
        source: 'official',
        icon: 'edit',
        provides: ['verstak/default-editor/v1'],
        requires: ['verstak/core/files/v1', 'verstak/core/workbench/v1'],
        permissions: ['files.read', 'files.write', 'workbench.open'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          openProviders: [
            {
              id: 'verstak.default-editor.text',
              title: 'Default Text Editor',
              priority: 50,
              component: 'DefaultEditor',
              supports: [
                { kind: 'vault-file', extensions: ['.txt', '.log', '.conf', '.ini', '.toml', '.yaml', '.yml', '.json', '.csv'], mime: ['text/plain', 'application/json'], contexts: ['generic-text'] }
              ]
            },
            {
              id: 'verstak.default-editor.markdown',
              title: 'Default Markdown Editor',
              priority: 50,
              component: 'DefaultEditor',
              supports: [
                { kind: 'vault-file', extensions: ['.md', '.markdown'], contexts: ['generic-markdown'] }
              ]
            },
            {
              id: 'verstak.default-editor.notes-markdown',
              title: 'Default Notes Markdown Editor',
              priority: 50,
              component: 'DefaultEditor',
              supports: [
                { kind: 'vault-file', extensions: ['.md', '.markdown'], contexts: ['notes-markdown'] }
              ]
            }
          ]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/default-editor',
      error: ''
    },
    'verstak.files': {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.files',
        name: 'Files',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Minimal vault file navigator.',
        source: 'official',
        icon: 'folder',
        provides: ['verstak/files/v1'],
        requires: ['verstak/core/files/v1', 'verstak/core/workbench/v1'],
        permissions: ['files.read', 'files.write', 'files.delete', 'files.openExternal', 'workbench.open', 'ui.register'],
        frontend: { entry: 'frontend/dist/index.js' },
    contributes: {
      views: [{ id: 'verstak.files.view', title: 'Files', icon: 'folder', component: 'FilesView' }],
      workspaceItems: [{ id: 'verstak.files.workspace', title: 'Files', icon: 'folder', component: 'FilesView' }]
    }
      },
      rootPath: '/tmp/verstak-test/plugins/files',
      error: ''
    },
    'verstak.trash': makeTrashPluginState(),
    'verstak.notes': {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.notes',
        name: 'Notes',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Deal-scoped notes manager.',
        source: 'official',
        icon: 'edit',
        provides: ['verstak/notes/v1'],
        requires: ['verstak/core/files/v1', 'verstak/core/workbench/v1'],
        permissions: ['files.read', 'files.write', 'files.delete', 'events.subscribe', 'workbench.open', 'ui.register'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          workspaceItems: [{ id: 'verstak.notes.workspace', title: 'Notes', icon: 'edit', component: 'NotesView' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/notes',
      error: ''
    },
    'verstak.sync': {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.sync',
        name: 'Sync',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Synchronize vault data across devices.',
        source: 'official',
        icon: 'refresh-cw',
        provides: ['verstak/sync/v1', 'verstak/sync.status/v1'],
        requires: ['verstak/core/files/v1'],
        permissions: ['files.read', 'files.write', 'network.remote', 'storage.namespace', 'sync.participate', 'ui.register'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          settingsPanels: [{ id: 'verstak.sync.settings', title: 'Sync', component: 'SyncSettings' }],
          statusBarItems: [{ id: 'verstak.sync.status', label: 'Sync', position: 'right', handler: 'SyncStatusBar' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/sync',
      error: ''
    },
    'verstak.activity': {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.activity',
        name: 'Activity',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Deal-scoped activity log for public plugin events.',
        source: 'official',
        icon: 'activity',
        provides: ['activity.log', 'activity.provider', 'activity.reconstruction'],
        permissions: ['events.subscribe', 'storage.namespace', 'ui.register'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          views: [{ id: 'verstak.activity.view', title: 'Activity', icon: 'activity', component: 'ActivityView' }],
          sidebarItems: [{ id: 'verstak.activity.sidebar', title: 'Activity', icon: 'activity', view: 'verstak.activity.view', position: 20 }],
          workspaceItems: [{ id: 'verstak.activity.workspace', title: 'Activity', icon: 'activity', component: 'ActivityView' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/activity',
      error: ''
    },
    'verstak.journal': {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.journal',
        name: 'Journal',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Deal-scoped journal with user-authored entries and optional Activity links.',
        source: 'official',
        icon: 'book-open',
        provides: ['worklog', 'journal', 'report.worklog'],
        permissions: ['events.publish', 'files.read', 'storage.namespace', 'ui.register'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          views: [{ id: 'verstak.journal.view', title: 'Journal', icon: 'book-open', component: 'JournalView' }],
          sidebarItems: [{ id: 'verstak.journal.sidebar', title: 'Journal', icon: 'book-open', view: 'verstak.journal.view', position: 30 }],
          workspaceItems: [{ id: 'verstak.journal.workspace', title: 'Journal', icon: 'book-open', component: 'JournalView' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/journal',
      error: ''
    },
    'verstak.browser-inbox': {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.browser-inbox',
        name: 'Browser',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Global browser materials with explicit Deal assignment.',
        source: 'official',
        icon: 'inbox',
        provides: ['browser.inbox'],
        permissions: ['events.subscribe', 'files.read', 'storage.namespace', 'ui.register'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          views: [{ id: 'verstak.browser-inbox.view', title: 'Browser', icon: 'inbox', component: 'BrowserInboxView' }],
          sidebarItems: [{ id: 'verstak.browser-inbox.sidebar', title: 'Browser', icon: 'inbox', view: 'verstak.browser-inbox.view', position: 30 }],
          workspaceItems: [{ id: 'verstak.browser-inbox.workspace', title: 'Browser', icon: 'inbox', component: 'BrowserInboxView' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/browser-inbox',
      error: ''
    },
    'verstak.todo': makeTodoPluginState(),
    'verstak.secrets': makeSecretsPluginState(),
    'verstak.import': makeImportPluginState(),
    'verstak.search': {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.search',
        name: 'Search',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Deal-scoped vault text search provider.',
        source: 'official',
        icon: 'search',
        provides: ['verstak/search/v1', 'search.provider'],
        requires: ['verstak/core/files/v1', 'verstak/core/workbench/v1'],
        permissions: ['files.read', 'workbench.open', 'storage.namespace', 'ui.register', 'events.subscribe', 'commands.register'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          workspaceItems: [{ id: 'verstak.search.workspace', title: 'Search', icon: 'search', component: 'SearchView' }],
          commands: [{ id: 'verstak.search.searchVaultText', title: 'Search Vault Text', handler: 'verstak.search.searchVaultText' }],
          searchProviders: [{ id: 'verstak.search.vault-text', label: 'Vault Text Search', handler: 'verstak.search.searchVaultText' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/search',
      error: ''
    }
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
  Object.keys(pluginStates).forEach(function (pluginId) {
    var manifest = pluginStates[pluginId].manifest;
    manifest.localization = {
      defaultLocale: 'en',
      locales: { en: 'locales/en.json', ru: 'locales/ru.json' }
    };
  });

  function mockPluginCatalog(pluginId, locale) {
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
      searchProviders: 'label', statusBarItems: 'label', openProviders: 'title', workspaceItems: 'title'
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
  var vaultPluginState = { enabledPlugins: ['verstak.platform-test', 'verstak.default-editor', 'verstak.files', 'verstak.notes', 'verstak.sync', 'verstak.activity', 'verstak.journal', 'verstak.browser-inbox', 'verstak.search'], disabledPlugins: [], desiredPlugins: [{ id: 'verstak.platform-test', version: '0.1.0', source: 'official' }, { id: 'verstak.default-editor', version: '0.1.0', source: 'official' }, { id: 'verstak.files', version: '0.1.0', source: 'official' }, { id: 'verstak.notes', version: '0.1.0', source: 'official' }, { id: 'verstak.sync', version: '0.1.0', source: 'official' }, { id: 'verstak.activity', version: '0.1.0', source: 'official' }, { id: 'verstak.journal', version: '0.1.0', source: 'official' }, { id: 'verstak.browser-inbox', version: '0.1.0', source: 'official' }, { id: 'verstak.search', version: '0.1.0', source: 'official' }] };
  vaultPluginState.enabledPlugins.push('verstak.trash');
  vaultPluginState.desiredPlugins.push({ id: 'verstak.trash', version: '0.1.0', source: 'official' });
  vaultPluginState.enabledPlugins.push('verstak.todo');
  vaultPluginState.desiredPlugins.push({ id: 'verstak.todo', version: '0.1.0', source: 'official' });
  vaultPluginState.enabledPlugins.push('verstak.secrets');
  vaultPluginState.desiredPlugins.push({ id: 'verstak.secrets', version: '0.1.0', source: 'official' });
  vaultPluginState.enabledPlugins.push('verstak.import');
  vaultPluginState.desiredPlugins.push({ id: 'verstak.import', version: '0.1.0', source: 'official' });
  var appSettings = {
    currentVaultPath: '/tmp/verstak-test/vault',
    recentVaults: [],
    language: localStorage.getItem('verstak-test-language') || 'system'
  };
  var workbenchPreferences = {};
  var openedResources = [];
  var pluginSettings = {
    'verstak.platform-test': { savedText: 'initial value' }
  };
  var pluginNotifications = {};
  var pluginData = {};
  var secretRecords = makeDefaultSecretRecords();
  var vaultFiles = makeDefaultVaultFiles();
  var externalOpens = [];
  var trashEntries = [];
  var trashPayloads = {};
  window.__wailsMockExternalOpens = [];
  var workspaceTree = makeDefaultWorkspaceTree();
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

  function makeTrashPluginState() {
    return {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.trash',
        name: 'Trash',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Global vault trash manager for deleted files and folders.',
        source: 'official',
        icon: 'trash',
        provides: ['trash.management'],
        requires: ['verstak/core/files/v1'],
        permissions: ['files.delete', 'files.write', 'ui.register'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          views: [{ id: 'verstak.trash.view', title: 'Trash', icon: 'trash', component: 'TrashView' }],
          sidebarItems: [{ id: 'verstak.trash.sidebar', title: 'Trash', icon: 'trash', view: 'verstak.trash.view', position: 40 }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/trash',
      error: ''
    };
  }

  function makeTodoPluginState() {
    return {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.todo',
        name: 'Todos',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Global and Deal-scoped todo tracking with due and reminder metadata.',
        source: 'official',
        icon: 'list-todo',
        provides: ['todo.list', 'todo.workspace'],
        requires: ['verstak/core/notifications/v1'],
        permissions: ['files.read', 'storage.namespace', 'ui.register', 'notifications.schedule'],
        frontend: { entry: 'frontend/dist/index.js' },
        contributes: {
          views: [{ id: 'verstak.todo.view', title: 'Todos', icon: 'list-todo', component: 'TodoView' }],
          sidebarItems: [{ id: 'verstak.todo.sidebar', title: 'Todos', icon: 'list-todo', view: 'verstak.todo.view', position: 35 }],
          workspaceItems: [{ id: 'verstak.todo.workspace', title: 'Todos', icon: 'list-todo', component: 'TodoView' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/todo',
      error: ''
    };
  }

  function makeSecretsPluginState() {
    return {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.secrets',
        name: 'Secrets',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Encrypted global and Deal-scoped secret manager.',
        source: 'official',
        icon: 'key-round',
        provides: ['secret-store', 'secrets.read-ui', 'secrets.write-ui'],
        permissions: ['files.read', 'secrets.read', 'secrets.write', 'ui.register'],
        frontend: { entry: 'frontend/src/index.js' },
        contributes: {
          views: [{ id: 'verstak.secrets.view', title: 'Secrets', icon: 'key-round', component: 'SecretsView' }],
          sidebarItems: [{ id: 'verstak.secrets.sidebar', title: 'Secrets', icon: 'key-round', view: 'verstak.secrets.view', position: 45 }],
          openProviders: [{
            id: 'verstak.secrets.secret',
            title: 'Secrets',
            priority: 100,
            component: 'SecretsView',
            supports: [{ kind: 'secret', modes: ['view'] }]
          }],
          workspaceItems: [{ id: 'verstak.secrets.workspace', title: 'Secrets', icon: 'key-round', component: 'SecretsView' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/secrets',
      error: ''
    };
  }

  function makeImportPluginState() {
    return {
      status: 'loaded',
      enabled: true,
      manifest: {
        schemaVersion: 1,
        id: 'verstak.import',
        name: 'Import',
        version: '0.1.0',
        apiVersion: '0.1.0',
        description: 'Review and import current DokuWiki or Obsidian content.',
        source: 'official',
        icon: 'archive-restore',
        provides: ['verstak/import/v1'],
        requires: ['verstak/core/import/v1'],
        permissions: ['imports.readExternal', 'imports.apply', 'storage.namespace', 'ui.register'],
        frontend: { entry: 'frontend/dist/index.js', style: 'frontend/dist/style.css' },
        contributes: {
          settingsPanels: [{ id: 'verstak.import.settings', title: 'Import', component: 'ImportSettings' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/import',
      error: ''
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
    var views = [], commands = [], searchProviders = [], sidebarItems = [], statusBarItems = [], settingsPanels = [], openProviders = [], workspaceItems = [];
    for (var id in pluginStates) {
      var s = pluginStates[id];
      var c = (s.manifest && s.manifest.contributes) || {};
      if (c.views) c.views.forEach(function (v) { views.push(Object.assign({}, v, { pluginId: id })); });
      if (c.commands) c.commands.forEach(function (cmd) { commands.push(Object.assign({}, cmd, { pluginId: id })); });
      if (c.searchProviders) c.searchProviders.forEach(function (sp) { searchProviders.push(Object.assign({}, sp, { pluginId: id })); });
      if (c.sidebarItems) c.sidebarItems.forEach(function (sb) { sidebarItems.push(Object.assign({}, sb, { pluginId: id })); });
      if (c.statusBarItems) c.statusBarItems.forEach(function (st) { statusBarItems.push(Object.assign({}, st, { pluginId: id })); });
      if (c.settingsPanels) c.settingsPanels.forEach(function (sp) { settingsPanels.push(Object.assign({}, sp, { pluginId: id })); });
      if (c.openProviders) c.openProviders.forEach(function (op) { openProviders.push(Object.assign({}, op, { pluginId: id })); });
      if (c.workspaceItems) c.workspaceItems.forEach(function (wi) { workspaceItems.push(Object.assign({}, wi, { pluginId: id })); });
    }
    return { views: views, commands: commands, searchProviders: searchProviders, sidebarItems: sidebarItems, statusBarItems: statusBarItems, settingsPanels: settingsPanels, openProviders: openProviders, workspaceItems: workspaceItems };
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

  function filesPluginBundle() {
    return '(' + function () {
      var SVG = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M6 2h9l5 5v15H6V2Zm8 1.5V8h4.5L14 3.5Z"/></svg>';
      var FOLDER_SVG = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v1H3V5Zm0 6h18v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-7Z"/></svg>';
      var FILE_SVGS = {
        folder: FOLDER_SVG,
        generic: SVG,
        markdown: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M5 3h10l4 4v14H5V3Zm9 1.5V8h3.5L14 4.5ZM8 11h8v2H8v-2Zm0 4h8v2H8v-2Z"/></svg>',
        text: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M6 2h9l5 5v15H6V2Zm8 1.5V8h4.5L14 3.5ZM8 12h8v1.5H8V12Zm0 3h8v1.5H8V15Zm0 3h5v1.5H8V18Z"/></svg>',
        image: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/></svg>',
        pdf: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M20 2H8c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm-8.5 7.5c0 .83-.67 1.5-1.5 1.5H9v2H7.5V7H10c.83 0 1.5.67 1.5 1.5v1zm5 2c0 .83-.67 1.5-1.5 1.5h-2.5V7H15c.83 0 1.5.67 1.5 1.5v3zm4-3H19v1h1.5V11H19v2h-1.5V7h3v1.5zM9 9.5h1v-1H9v1zm6 2h1v-3h-1v3z"/></svg>',
        code: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M22 21H2V3h20v18zM4 5v14h16V5H4zm3 4h10v2H7V9zm0 4h6v2H7v-2z"/></svg>'
      };
      var ACTION_ICONS = {
        back: 'M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.42-1.41L7.83 13H20v-2z',
        forward: 'M4 13h12.17l-5.59 5.59L12 20l8-8-8-8-1.42 1.41L16.17 11H4v2z',
        up: 'M4 12l1.41 1.41L11 7.83V20h2V7.83l5.59 5.58L20 12 12 4l-8 8z',
        refresh: 'M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z',
        folderAdd: 'M20 6h-8.17l-2-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm-1 8h-3v3h-2v-3h-3v-2h3V9h2v3h3v2z',
        markdownAdd: 'M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm-1 8V4l5 5h-5zm-6 6h2v-3l2 3h1l2-3v3h2v-6h-2l-2.5 3.5L9 10H7v6z',
        textAdd: 'M14 2H6c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V8l-6-6zm-1 8V4l5 5h-5zM8 13h8v2H8v-2zm0 4h8v2H8v-2z',
        open: 'M14 3v2h3.59l-9.83 9.83 1.41 1.41L19 6.41V10h2V3h-7zM5 5h6V3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2v-6h-2v6H5V5z',
        rename: 'M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34a.9959.9959 0 0 0-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z',
        trash: 'M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM8 9h8v10H8V9zm7.5-5-1-1h-5l-1 1H5v2h14V4z',
        trashView: 'M4 4h16v2H4V4zm2 4h12v12H6V8zm2 2v8h8v-8H8zm2 1.5h4V13h-4v-1.5zm0 3h4V16h-4v-1.5zM9 1h6l1 2H8l1-2z',
        external: 'M14 3h7v7h-2V6.41l-9.83 9.83-1.41-1.41L17.59 5H14V3zM5 5h6v2H7v10h10v-4h2v6H5V5z',
        explorer: 'M3 5a2 2 0 0 1 2-2h5l2 3h7a2 2 0 0 1 2 2v1H3V5Zm0 6h18v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-7Z',
        duplicate: 'M5 3h10v2H5v10H3V5c0-1.1.9-2 2-2zm4 4h10c1.1 0 2 .9 2 2v10c0 1.1-.9 2-2 2H9c-1.1 0-2-.9-2-2V9c0-1.1.9-2 2-2zm1 2v10h9V9h-9zm3 3h3v2h2v3h-2v2h-3v-2h-2v-3h2v-2z',
        restore: 'M13 3a9 9 0 1 1-8.95 8H2l3-3 3 3H6.06A7 7 0 1 0 13 5V3zm-1 5h2v5h4v2h-6V8z',
        cut: 'M9.64 7.64c.23-.5.36-1.05.36-1.64 0-2.21-1.79-4-4-4S2 3.79 2 6s1.79 4 4 4c.59 0 1.14-.13 1.64-.36L10 12l-2.36 2.36C7.14 14.13 6.59 14 6 14c-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4c0-.59-.13-1.14-.36-1.64L12 14l7 7h3L9.64 7.64zM6 8c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm0 12c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm6-8.5c-.28 0-.5.22-.5.5s.22.5.5.5.5-.22.5-.5-.22-.5-.5-.5zM19 3l-6 6 2 2 7-8h-3z',
        copy: 'M16 1H4c-1.1 0-2 .9-2 2v12h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z',
        paste: 'M19 2h-4.18C14.4.84 13.3 0 12 0S9.6.84 9.18 2H5c-1.1 0-2 .9-2 2v16c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm-7 0c.55 0 1 .45 1 1s-.45 1-1 1-1-.45-1-1zm7 18H5V4h2v3h10V4h2v16z'
      };
      function actionSvg(iconKey) {
        return '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="' + (ACTION_ICONS[iconKey] || ACTION_ICONS.open) + '"/></svg>';
      }
      var ICON_EXTENSIONS = {
        md: 'markdown', markdown: 'markdown',
        txt: 'text', text: 'text', log: 'text', rtf: 'text',
        jpg: 'image', jpeg: 'image', png: 'image', gif: 'image', webp: 'image', svg: 'image',
        pdf: 'pdf',
        js: 'code', jsx: 'code', mjs: 'code', cjs: 'code', ts: 'code', tsx: 'code',
        py: 'code', go: 'code', rs: 'code', css: 'code', html: 'code', json: 'code'
      };
      function fileIconCategory(item) {
        if (item.type === 'folder') return 'folder';
        return ICON_EXTENSIONS[(item.extension || ext(item.name)).toLowerCase()] || 'generic';
      }
      function fileIconLabel(category) {
        return {
          folder: 'Folder',
          markdown: 'Markdown file',
          text: 'Text file',
          image: 'Image file',
          pdf: 'PDF file',
          code: 'Code file',
          generic: 'File'
        }[category] || 'File';
      }
      function fileIcon(item) {
        return FILE_SVGS[fileIconCategory(item)] || FILE_SVGS.generic;
      }
      function e(tag, attrs, children) {
        var node = document.createElement(tag);
        attrs = attrs || {};
        Object.keys(attrs).forEach(function (key) {
          if (key === 'className') node.className = attrs[key];
          else if (key.indexOf('on') === 0) node.addEventListener(key.slice(2).toLowerCase(), attrs[key]);
          else if (key === 'innerHTML') node.innerHTML = attrs[key];
          else if (key === 'style' && typeof attrs[key] === 'object') Object.assign(node.style, attrs[key]);
          else node.setAttribute(key, attrs[key]);
        });
        (children || []).forEach(function (child) { if (child) node.appendChild(typeof child === 'string' ? document.createTextNode(child) : child); });
        return node;
      }
      function clean(path) { return String(path || '').split('/').filter(Boolean).join('/'); }
      function parent(path) { path = clean(path); var i = path.lastIndexOf('/'); return i < 0 ? '' : path.slice(0, i); }
      function ext(name) { var i = String(name || '').lastIndexOf('.'); return i > 0 ? name.slice(i + 1).toLowerCase() : ''; }
      function base(path) { path = clean(path); var i = path.lastIndexOf('/'); return i < 0 ? path : path.slice(i + 1); }
      function isConflictError(err) {
        var message = (err && err.message) ? err.message : String(err || '');
        return /conflict|already exists|exists/i.test(message);
      }
      function formatDate(value) {
        if (!value) return '';
        var date = new Date(value);
        if (Number.isNaN(date.getTime())) return String(value);
        return new Intl.DateTimeFormat('ru-RU', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' }).format(date);
      }
      var FilesView = {
        mount: function (c, p, api) {
          if (!document.getElementById('mock-files-styles')) {
            var style = document.createElement('style');
            style.id = 'mock-files-styles';
            style.textContent = '.files-root{display:flex;flex-direction:column;height:100%;min-height:0;background:#0d0d1a;color:#e0e0e0;outline:0}.files-toolbar{display:flex;align-items:center;gap:.4rem;padding:.5rem .75rem;background:#12122a;border-bottom:1px solid #16213e;flex-wrap:wrap}.files-toolbar-btn,.files-row-btn{display:inline-flex;align-items:center;justify-content:center;gap:.3rem;border:1px solid #333;border-radius:4px;background:#1a1a2e;color:#ccc;cursor:pointer;white-space:nowrap}.files-toolbar-btn{min-height:2rem;padding:.35rem .55rem}.files-row-btn{min-height:1.75rem;padding:.25rem .45rem;font-size:.72rem}.files-toolbar-btn svg,.files-row-btn svg{width:15px;height:15px;flex-shrink:0}.files-btn-label{line-height:1}.files-breadcrumb{flex:1;min-width:150px;color:#8b8ba8}.files-breadcrumb-item{color:#4ecca3;cursor:pointer}.files-breadcrumb-current{color:#ddd}.files-filter,.files-sort,.files-create-input,.files-rename-input{font-size:.78rem;padding:.32rem .5rem;border:1px solid #333;border-radius:4px;background:#0d0d1a;color:#e0e0e0}.files-sort{appearance:none;background-color:#0d0d1a;padding-right:1rem}.files-list{flex:1;overflow:auto}.files-header,.files-item{display:grid;grid-template-columns:minmax(180px,1fr) 90px 80px 150px 210px;align-items:center;gap:.5rem;padding:.38rem .75rem;border-bottom:1px solid rgba(22,33,62,.55)}.files-header{background:#101028;color:#8b8ba8;font-size:.7rem;text-transform:uppercase}.files-item:hover{background:#17172d}.files-item.selected{background:#1a2a3a}.files-namecell{display:flex;align-items:center;gap:.5rem;min-width:0}.files-item-icon{width:1.25rem;color:#8b8ba8}.files-item-name{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.files-item-meta{font-size:.74rem;color:#777;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.files-row-actions{display:flex;justify-content:flex-end;gap:.35rem}.files-panel{display:flex;gap:.5rem;padding:.5rem .75rem;border-top:1px solid #16213e;background:#12122a}.files-create-input,.files-rename-input{flex:1}.files-ctx-menu{position:fixed;z-index:9999;min-width:170px;background:#1a1a2e;border:1px solid #333;border-radius:6px;padding:6px 0;box-shadow:0 8px 24px rgba(0,0,0,.5);font-size:.84rem;color:#e0e0e0}.files-ctx-menu-item{display:flex;align-items:center;gap:.5rem;padding:6px 16px;cursor:pointer}.files-ctx-menu-item:hover{background:#2a2a4e}.files-ctx-menu-item svg{width:14px;height:14px}.files-ctx-menu-sep{height:1px;background:#333;margin:4px 8px}.files-drag-over{outline:2px dashed #4ecca3;outline-offset:-2px}';
            style.textContent += '.files-empty{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:.75rem;color:#8b8ba8;font-size:.9rem;padding:2rem;text-align:center}.files-empty-actions{display:flex;align-items:center;justify-content:center;gap:.5rem;flex-wrap:wrap}.files-empty-btn{display:inline-flex;align-items:center;justify-content:center;gap:.35rem;min-height:2rem;padding:.35rem .6rem;border:1px solid #333;border-radius:4px;background:#1a1a2e;color:#ccc;cursor:pointer;font-size:.78rem}.files-empty-btn:hover{background:#2a2a4e;border-color:#4ecca3}.files-empty-btn svg{width:15px;height:15px;flex-shrink:0}';
            document.head.appendChild(style);
          }
          c.innerHTML = '';
          c.className = 'files-root';
          c.setAttribute('tabindex', '0');
          c.setAttribute('data-plugin-id', 'verstak.files');
          var n = p && p.workspaceNode;
          var root = clean((p && (p.workspaceRootPath || (n && (n.rootPath || n.name || n.id)))) || '');
          var workspaceName = root || 'Workspace';
          window.__filesHistoryByWorkspace = window.__filesHistoryByWorkspace || {};
          var historyKey = root || workspaceName;
          var savedHistory = window.__filesHistoryByWorkspace[historyKey] || { stack: [''], index: 0, currentPath: '' };
          var current = clean(savedHistory.currentPath || '');
          var history = savedHistory.stack && savedHistory.stack.length ? savedHistory.stack.map(clean) : [current];
          var historyIndex = Math.max(0, Math.min(savedHistory.index || 0, history.length - 1));
          var entries = [];
          var selected = {};
          var lastClicked = '';
          var filter = '';
          var sort = 'folder-name';
          var createMode = '';
          var renaming = null;
          function scoped(local) { local = clean(local); return root ? (local ? root + '/' + local : root) : local; }
          function local(full) { full = clean(full); return root && full.indexOf(root + '/') === 0 ? full.slice(root.length + 1) : full === root ? '' : full; }
          function saveHistory() { window.__filesHistoryByWorkspace[historyKey] = { stack: history.slice(), index: historyIndex, currentPath: current }; }
          var toolbar = e('div', { className: 'files-toolbar' }, []);
          var breadcrumb = e('div', { className: 'files-breadcrumb' }, []);
          function btn(title, action, fn, iconKey) {
            iconKey = iconKey || action;
            return e('button', { className: 'files-toolbar-btn', 'data-files-action': action, 'data-files-icon': iconKey, title: title, 'aria-label': title, innerHTML: actionSvg(iconKey), onClick: fn }, []);
          }
          function rowBtn(title, action, fn, iconKey) {
            iconKey = iconKey || action;
            return e('button', { className: 'files-row-btn', 'data-files-action': action, 'data-files-icon': iconKey, title: title, 'aria-label': title, innerHTML: actionSvg(iconKey), onClick: fn }, []);
          }
          function emptyBtn(title, action, mode, iconKey) {
            return e('button', {
              className: 'files-empty-btn',
              'data-files-empty-action': action,
              'data-files-icon': iconKey,
              title: title,
              'aria-label': title,
              innerHTML: actionSvg(iconKey) + '<span>' + title + '</span>',
              onClick: function () { startCreate(mode); }
            }, []);
          }
          function emptyFolderState() {
            return e('div', { className: 'files-empty' }, [
              e('div', {}, ['Empty folder']),
              e('div', { className: 'files-empty-actions' }, [
                emptyBtn('New folder', 'new-folder', 'folder', 'folderAdd'),
                emptyBtn('New markdown file', 'new-markdown', 'markdown', 'markdownAdd'),
                emptyBtn('New text file', 'new-text', 'text', 'textAdd')
              ])
            ]);
          }
          function noMatchesState() {
            return e('div', { className: 'files-empty' }, [
              e('div', {}, ['No matches']),
              e('div', { className: 'files-empty-actions' }, [
                e('button', {
                  className: 'files-empty-btn',
                  'data-files-empty-action': 'clear-filter',
                  'data-files-icon': 'refresh',
                  title: 'Clear filter',
                  'aria-label': 'Clear filter',
                  innerHTML: actionSvg('refresh') + '<span>Clear filter</span>',
                  onClick: function () {
                    filter = '';
                    filterInput.value = '';
                    render();
                    list.focus();
                  }
                }, [])
              ])
            ]);
          }
          var backButton = btn('Back', 'back', goBack, 'back');
          var forwardButton = btn('Forward', 'forward', goForward, 'forward');
          var upButton = btn('Up', 'up', function () { if (current) nav(parent(current)); }, 'up');
          var refreshButton = btn('Refresh', 'refresh', load, 'refresh');
          var newFolderButton = btn('New folder', 'new-folder', function () { startCreate('folder'); }, 'folderAdd');
          var newMarkdownButton = btn('New markdown file', 'new-markdown', function () { startCreate('markdown'); }, 'markdownAdd');
          var newTextButton = btn('New text file', 'new-text', function () { startCreate('text'); }, 'textAdd');
          var openButton = btn('Open', 'open', function () { open(firstSelected()); }, 'open');
          var renameButton = btn('Rename', 'rename', function () { startRename(firstSelected()); }, 'rename');
          var trashButton = btn('Move to trash', 'trash', function () { trashSelection(); }, 'trash');
          var cutButton = btn('Cut', 'cut', cutSelection, 'cut');
          var copyButton = btn('Copy', 'copy', copySelection, 'copy');
          var pasteButton = btn('Paste', 'paste', paste, 'paste');
          toolbar.appendChild(breadcrumb);
          [backButton, forwardButton, upButton, refreshButton, newFolderButton, newMarkdownButton, newTextButton, openButton, renameButton, trashButton, cutButton, copyButton, pasteButton].forEach(function (button) { toolbar.appendChild(button); });
          var filterInput = e('input', { className: 'files-filter', 'data-files-filter': '', placeholder: 'Filter current folder' }, []);
          filterInput.addEventListener('input', function () { filter = filterInput.value.toLowerCase(); render(); });
          toolbar.appendChild(filterInput);
          var sortSelect = e('select', { className: 'files-sort', 'data-files-sort': '' }, [
            e('option', { value: 'folder-name' }, ['Folders + name']),
            e('option', { value: 'name-asc' }, ['Name']),
            e('option', { value: 'type' }, ['Type']),
            e('option', { value: 'modified-desc' }, ['Modified']),
            e('option', { value: 'size-desc' }, ['Size'])
          ]);
          sortSelect.addEventListener('change', function () { sort = sortSelect.value; render(); });
          toolbar.appendChild(sortSelect);
          c.appendChild(toolbar);
          var list = e('div', { className: 'files-list', 'data-files-list': '' }, []);
          c.appendChild(list);
          var createPanel = e('div', { className: 'files-panel', style: 'display:none' }, []);
          var createField = e('div', { style: { display: 'flex', flex: '1', flexDirection: 'column', gap: '.25rem' } }, []);
          var createInput = e('input', { className: 'files-create-input', 'data-files-create-input': '' }, []);
          var createError = e('div', { 'data-files-create-error': '', role: 'alert', style: { display: 'none', color: '#ff8a8a', fontSize: '.72rem', lineHeight: '1.2' } }, []);
          createField.appendChild(createInput);
          createField.appendChild(createError);
          createPanel.appendChild(createField);
          createPanel.appendChild(e('button', { className: 'files-toolbar-btn', 'data-files-create-confirm': '', onClick: confirmCreate }, ['Create']));
          createPanel.appendChild(e('button', { className: 'files-toolbar-btn', onClick: cancelCreate }, ['Cancel']));
          c.appendChild(createPanel);
          var renamePanel = e('div', { className: 'files-panel', style: 'display:none' }, []);
          var renameField = e('div', { style: { display: 'flex', flex: '1', flexDirection: 'column', gap: '.25rem' } }, []);
          var renameInput = e('input', { className: 'files-rename-input', 'data-files-rename-input': '' }, []);
          var renameError = e('div', { 'data-files-rename-error': '', role: 'alert', style: { display: 'none', color: '#ff8a8a', fontSize: '.72rem', lineHeight: '1.2' } }, []);
          renameField.appendChild(renameInput);
          renameField.appendChild(renameError);
          renamePanel.appendChild(renameField);
          renamePanel.appendChild(e('button', { className: 'files-toolbar-btn', 'data-files-rename-confirm': '', onClick: confirmRename }, ['Rename']));
          renamePanel.appendChild(e('button', { className: 'files-toolbar-btn', onClick: cancelRename }, ['Cancel']));
          c.appendChild(renamePanel);
          function entryByPath(path) { return entries.find(function (item) { return item.relativePath === path; }) || null; }
          function selectedEntries() { return Object.keys(selected).map(entryByPath).filter(Boolean); }
          function firstSelected() { return selectedEntries()[0] || null; }
          function updateBreadcrumb() {
            breadcrumb.innerHTML = '';
            breadcrumb.appendChild(e('span', { className: current ? 'files-breadcrumb-item' : 'files-breadcrumb-current', onClick: function () { nav(''); } }, [workspaceName]));
            if (!current) return;
            var parts = current.split('/');
            var path = '';
            parts.forEach(function (part, index) {
              path = path ? path + '/' + part : part;
              breadcrumb.appendChild(e('span', { className: 'files-breadcrumb-sep' }, [' / ']));
              var target = path;
              var isCurrent = index === parts.length - 1;
              breadcrumb.appendChild(e('span', {
                className: isCurrent ? 'files-breadcrumb-current' : 'files-breadcrumb-item',
                onClick: function () { if (!isCurrent) nav(target); }
              }, [part]));
            });
          }
          function visible() {
            return entries.filter(function (item) { return !item.isHidden && !item.isReserved && (!filter || item.name.toLowerCase().indexOf(filter) !== -1); }).sort(function (a, b) {
              if (sort === 'folder-name') { if (a.type === 'folder' && b.type !== 'folder') return -1; if (a.type !== 'folder' && b.type === 'folder') return 1; }
              if (sort === 'modified-desc') return new Date(b.modifiedAt || 0) - new Date(a.modifiedAt || 0) || a.name.localeCompare(b.name);
              if (sort === 'size-desc') return (b.size || 0) - (a.size || 0) || a.name.localeCompare(b.name);
              if (sort === 'type') return (a.type + (a.extension || '')).localeCompare(b.type + (b.extension || '')) || a.name.localeCompare(b.name);
              return a.name.localeCompare(b.name);
            });
          }
          function render() {
            updateBreadcrumb();
            list.innerHTML = '';
            list.appendChild(e('div', { className: 'files-header' }, [e('span', {}, ['Name']), e('span', {}, ['Type']), e('span', {}, ['Size']), e('span', {}, ['Modified']), e('span', {}, ['Actions'])]));
            var shown = visible();
            if (!shown.length) {
              list.appendChild(filter ? noMatchesState() : emptyFolderState());
              return;
            }
            shown.forEach(function (item) {
              var iconCategory = fileIconCategory(item);
              var iconLabel = fileIconLabel(iconCategory);
              var row = e('div', {
                className: 'files-item' + (selected[item.relativePath] ? ' selected' : ''),
                'data-file-name': item.name,
                'data-file-type': item.type,
                'data-file-path': item.relativePath,
                draggable: 'true',
                onClick: function (ev) { select(item, ev); },
                onDblclick: function () { open(item); },
                onDragstart: function (ev) {
                  if (!selected[item.relativePath]) { selected = {}; selected[item.relativePath] = true; }
                  var paths = Object.keys(selected);
                  ev.dataTransfer.setData('application/files-paths', JSON.stringify(paths));
                  ev.dataTransfer.setData('application/x-verstak-files', JSON.stringify({
                    paths: paths,
                    workspaceRoot: root,
                    operation: 'move'
                  }));
                  ev.dataTransfer.effectAllowed = 'copyMove';
                }
              }, []);
              row.appendChild(e('span', { className: 'files-namecell' }, [e('span', { className: 'files-item-icon', 'data-file-icon': iconCategory, title: iconLabel, 'aria-label': iconLabel, innerHTML: fileIcon(item) }, []), e('span', { className: 'files-item-name' }, [item.name])]));
              row.appendChild(e('span', { className: 'files-item-meta' }, [item.type === 'folder' ? 'folder' : (item.extension || ext(item.name) || 'file')]));
              row.appendChild(e('span', { className: 'files-item-meta' }, [item.size ? String(item.size) : '']));
              row.appendChild(e('span', { className: 'files-item-meta' }, [formatDate(item.modifiedAt)]));
              row.appendChild(e('span', { className: 'files-row-actions' }, [rowBtn('Open', 'row-open', function (ev) { ev.stopPropagation(); open(item); }, 'open'), rowBtn('Rename', 'row-rename', function (ev) { ev.stopPropagation(); startRename(item); }, 'rename'), rowBtn('Move to trash', 'row-trash', function (ev) { ev.stopPropagation(); trash(item); }, 'trash')]));
              list.appendChild(row);
            });
          }
          function select(item, ev) {
            if (ev && (ev.ctrlKey || ev.metaKey)) {
              if (selected[item.relativePath]) delete selected[item.relativePath]; else selected[item.relativePath] = true;
            } else if (ev && ev.shiftKey && lastClicked) {
              var shown = visible();
              var a = shown.findIndex(function (x) { return x.relativePath === lastClicked; });
              var b = shown.findIndex(function (x) { return x.relativePath === item.relativePath; });
              if (a >= 0 && b >= 0) {
                selected = {};
                for (var i = Math.min(a, b); i <= Math.max(a, b); i++) selected[shown[i].relativePath] = true;
              }
            } else {
              selected = {}; selected[item.relativePath] = true;
            }
            lastClicked = item.relativePath;
            render();
          }
          function load() { selected = {}; api.files.list(scoped(current)).then(function (result) { entries = result || []; render(); }).catch(function (err) { list.textContent = 'Error: ' + (err.message || err); }); }
          function nav(path, push) {
            current = clean(path);
            if (push !== false) {
              if (historyIndex < history.length - 1) history = history.slice(0, historyIndex + 1);
              if (history[history.length - 1] !== current) { history.push(current); historyIndex = history.length - 1; }
            }
            saveHistory();
            load();
          }
          function goBack() { if (historyIndex <= 0) return; historyIndex -= 1; current = history[historyIndex]; saveHistory(); load(); }
          function goForward() { if (historyIndex >= history.length - 1) return; historyIndex += 1; current = history[historyIndex]; saveHistory(); load(); }
          function open(item) {
            if (!item) return;
            if (item.type === 'folder') { nav(local(item.relativePath)); return; }
            var itemExt = item.extension ? '.' + item.extension : (ext(item.name) ? '.' + ext(item.name) : '');
            var ctx = { sourcePluginId: 'verstak.files', sourceView: 'files' };
            if ((itemExt === '.md' || itemExt === '.markdown') && local(item.relativePath).split('/')[0] === 'Notes') { ctx.isInsideNotesFolder = true; ctx.notesMode = true; }
            api.workbench.openResource({ kind: 'vault-file', path: item.relativePath, mode: 'view', extension: itemExt, context: ctx });
          }
          function startCreate(mode) { createMode = mode; createInput.value = ''; setCreateError(''); createPanel.style.display = 'flex'; createInput.focus(); }
          function setCreateError(message) {
            createError.textContent = message || '';
            createError.style.display = message ? 'block' : 'none';
            createInput.setAttribute('aria-invalid', message ? 'true' : 'false');
          }
          function validateCreateName(name) {
            if (!name) return 'Name is required';
            if (/[\\/:*?"<>|\x00-\x1f]/.test(name)) return 'Invalid characters in name';
            if (name === '.' || name === '..' || name[0] === ' ' || name[name.length - 1] === ' ' || name[name.length - 1] === '.') return 'Invalid name';
            return '';
          }
          function cancelCreate() { createMode = ''; setCreateError(''); createPanel.style.display = 'none'; }
          function confirmCreate() {
            var name = createInput.value.trim();
            var mode = createMode;
            var validationError = validateCreateName(name);
            if (validationError) { setCreateError(validationError); return; }
            if (mode === 'markdown' && !/\.(md|markdown)$/i.test(name)) name += '.md';
            if (mode === 'text' && !/\.[^/.]+$/.test(name)) name += '.txt';
            var path = scoped(current ? current + '/' + name : name);
            (mode === 'folder' ? api.files.createFolder(path) : api.files.writeText(path, '', { createIfMissing: true, overwrite: false })).then(function () { cancelCreate(); load(); }).catch(function (err) { setCreateError('Error: ' + ((err && err.message) ? err.message : String(err))); });
          }
          function setRenameError(message) {
            renameError.textContent = message || '';
            renameError.style.display = message ? 'block' : 'none';
            renameInput.setAttribute('aria-invalid', message ? 'true' : 'false');
          }
          function cancelRename() { renaming = null; setRenameError(''); renamePanel.style.display = 'none'; }
          function startRename(item) { if (!item) return; renaming = item; renameInput.value = item.name; setRenameError(''); renamePanel.style.display = 'flex'; renameInput.focus(); renameInput.select(); }
          function confirmRename() {
            if (!renaming) return;
            var newName = renameInput.value.trim();
            if (!newName || newName === renaming.name) { cancelRename(); return; }
            if (/[\\/:*?"<>|\x00-\x1f]/.test(newName)) { setRenameError('Invalid characters in name'); return; }
            if (newName === '.' || newName === '..' || newName[0] === ' ' || newName[newName.length - 1] === ' ' || newName[newName.length - 1] === '.') { setRenameError('Invalid name'); return; }
            var to = parent(renaming.relativePath);
            to = to ? to + '/' + newName : newName;
            api.files.move(renaming.relativePath, to, { overwrite: false }).then(function () { cancelRename(); load(); }).catch(function (err) {
              if (isConflictError(err)) { setRenameError('A file with that name already exists'); return; }
              setRenameError('Error: ' + ((err && err.message) ? err.message : String(err)));
            });
          }
          function confirmModal(message, options) {
            options = options || {};
            return new Promise(function (resolve) {
              var overlay = e('div', {
                className: 'files-modal-overlay',
                style: {
                  position: 'fixed',
                  inset: '0',
                  background: 'rgba(0,0,0,.6)',
                  zIndex: '10000',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center'
                }
              }, []);
              var modal = e('div', {
                className: 'files-modal',
                style: {
                  width: '400px',
                  maxWidth: '90vw',
                  padding: '24px',
                  background: '#1a1a2e',
                  border: '1px solid #333',
                  borderRadius: '8px',
                  color: '#e0e0e0',
                  boxShadow: '0 12px 40px rgba(0,0,0,.5)'
                }
              }, []);
              var title = e('div', { className: 'files-modal-title', style: { marginBottom: '20px', lineHeight: '1.5' } }, [message]);
              var actions = e('div', { className: 'files-modal-actions', style: { display: 'flex', justifyContent: 'flex-end', gap: '8px' } }, []);
              function cleanup(value) {
                document.removeEventListener('keydown', onKeydown);
                if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
                resolve(value);
              }
              function modalButton(label, className, value) {
                return e('button', {
                  className: className,
                  style: {
                    padding: '.4rem 1rem',
                    border: '1px solid #333',
                    borderRadius: '6px',
                    background: className.indexOf('danger') !== -1 ? '#e74c3c' : '#2a2a4e',
                    color: className.indexOf('danger') !== -1 ? '#fff' : '#ccc',
                    cursor: 'pointer'
                  },
                  onClick: function () { cleanup(value); }
                }, [label]);
              }
              function onKeydown(ev) {
                if (ev.key === 'Escape') cleanup(false);
              }
              actions.appendChild(modalButton(options.cancelText || 'Cancel', 'files-modal-btn cancel', false));
              actions.appendChild(modalButton(options.confirmText || 'Confirm', 'files-modal-btn confirm' + (options.danger ? ' danger' : ''), true));
              modal.appendChild(title);
              modal.appendChild(actions);
              overlay.appendChild(modal);
              document.body.appendChild(overlay);
              document.addEventListener('keydown', onKeydown);
              var first = overlay.querySelector('.files-modal-btn');
              if (first) first.focus();
            });
          }
          function trash(item) {
            if (!item) return;
            confirmModal('Move "' + item.name + '" to trash?', { danger: true, confirmText: 'Move to trash' }).then(function (ok) {
              if (ok) api.files.trash(item.relativePath).then(load);
            });
          }
          function trashSelection() {
            var items = selectedEntries();
            if (items.length === 1) return trash(items[0]);
            if (!items.length) return;
            confirmModal('Move ' + items.length + ' items to trash?', { danger: true, confirmText: 'Move to trash' }).then(function (ok) {
              if (ok) Promise.all(items.map(function (item) { return api.files.trash(item.relativePath); })).then(load);
            });
          }
          function setClipboard(action, items) { if (!items.length) return; window.__filesClipboard = { action: action, workspaceRoot: root, items: items.map(function (item) { return { path: item.relativePath, name: item.name, type: item.type }; }) }; }
          function cutSelection() { setClipboard('cut', selectedEntries()); }
          function copySelection() { setClipboard('copy', selectedEntries()); }
          function uniqueName(name, occupied) { if (!occupied[name]) return name; var dot = name.lastIndexOf('.'); var b = dot > 0 ? name.slice(0, dot) : name; var x = dot > 0 ? name.slice(dot) : ''; for (var i = 2; i < 100; i++) { var c = b + ' (' + i + ')' + x; if (!occupied[c]) return c; } return b + ' (' + Date.now() + ')' + x; }
          function duplicate(item) {
            if (!item || item.type === 'folder') return;
            var dot = item.name.lastIndexOf('.');
            var stem = dot > 0 ? item.name.slice(0, dot) : item.name;
            var extension = dot > 0 ? item.name.slice(dot) : '';
            var folder = parent(item.relativePath);
            var maxAttempts = 100;
            function candidatePath(n) {
              var name = n === 1 ? stem + ' (copy)' + extension : stem + ' (copy ' + n + ')' + extension;
              return folder ? folder + '/' + name : name;
            }
            function tryDuplicate(n) {
              var target = candidatePath(n);
              return api.files.metadata(target).then(function () {
                if (n >= maxAttempts) return Promise.reject(new Error('all duplicate names are taken'));
                return tryDuplicate(n + 1);
              }, function () {
                return api.files.readText(item.relativePath).then(function (text) {
                  return api.files.writeText(target, text, { createIfMissing: true, overwrite: false });
                });
              });
            }
            tryDuplicate(1).then(load).catch(function (err) { console.error('[files] Duplicate failed:', err); });
          }
          function paste() {
            var clip = window.__filesClipboard;
            if (!clip || !clip.items || !clip.items.length) return;
            var crossWorkspace = !!(clip.workspaceRoot && clip.workspaceRoot !== root);
            var dest = crossWorkspace ? scoped(current || 'Files') : scoped(current);
            api.files.list(dest).then(function (destinationEntries) {
              var occupied = {};
              (destinationEntries || []).forEach(function (item) { occupied[item.name] = true; });
              return Promise.all(clip.items.map(function (item) {
                var name = uniqueName(item.name, occupied);
                occupied[name] = true;
                var to = dest ? dest + '/' + name : name;
                if (clip.action === 'cut') return api.files.move(item.path, to, { overwrite: false });
                return api.files.copy(item.path, to, { overwrite: false });
              }));
            }).then(function () { if (clip.action === 'cut') window.__filesClipboard = null; load(); });
          }
          var menu = e('div', { className: 'files-ctx-menu', style: { display: 'none' } }, []);
          document.body.appendChild(menu);
          function menuItem(label, action, fn, iconKey) {
            iconKey = iconKey || action;
            return e('div', {
              className: 'files-ctx-menu-item',
              'data-files-menu-action': action,
              'data-files-menu-icon': iconKey,
              onClick: function (ev) { ev.stopPropagation(); menu.style.display = 'none'; fn(); }
            }, [e('span', { innerHTML: actionSvg(iconKey) }, []), label]);
          }
          function showMenu(x, y, item) {
            menu.innerHTML = '';
            if (item) {
              if (!selected[item.relativePath]) { selected = {}; selected[item.relativePath] = true; render(); }
              menu.appendChild(menuItem('Open', 'open', function () { open(item); }, 'open'));
              menu.appendChild(menuItem('Rename', 'rename', function () { startRename(item); }, 'rename'));
              if (item.type !== 'folder') menu.appendChild(menuItem('Duplicate', 'duplicate', function () { duplicate(item); }, 'duplicate'));
              menu.appendChild(menuItem('Cut', 'cut', cutSelection, 'cut'));
              menu.appendChild(menuItem('Copy', 'copy', copySelection, 'copy'));
              menu.appendChild(menuItem('Trash', 'trash', trashSelection, 'trash'));
            } else {
              menu.appendChild(menuItem('New Folder', 'new-folder', function () { startCreate('folder'); }, 'folderAdd'));
              menu.appendChild(menuItem('New Markdown', 'new-markdown', function () { startCreate('markdown'); }, 'markdownAdd'));
              menu.appendChild(menuItem('New Text', 'new-text', function () { startCreate('text'); }, 'textAdd'));
              if (window.__filesClipboard && window.__filesClipboard.items && window.__filesClipboard.items.length) menu.appendChild(menuItem('Paste', 'paste', paste, 'paste'));
            }
            menu.style.display = 'block'; menu.style.left = x + 'px'; menu.style.top = y + 'px';
          }
          createInput.addEventListener('input', function () { setCreateError(''); });
          createInput.addEventListener('keydown', function (ev) { if (ev.key === 'Enter') confirmCreate(); if (ev.key === 'Escape') cancelCreate(); });
          renameInput.addEventListener('input', function () { setRenameError(''); });
          renameInput.addEventListener('keydown', function (ev) { if (ev.key === 'Enter') confirmRename(); if (ev.key === 'Escape') cancelRename(); });
          list.addEventListener('contextmenu', function (ev) { ev.preventDefault(); var row = ev.target.closest('.files-item'); showMenu(ev.clientX, ev.clientY, row ? entryByPath(row.getAttribute('data-file-path')) : null); });
          list.addEventListener('dragover', function (ev) { ev.preventDefault(); var row = ev.target.closest('.files-item'); if (row) row.classList.add('files-drag-over'); });
          list.addEventListener('dragleave', function (ev) { var row = ev.target.closest('.files-item'); if (row) row.classList.remove('files-drag-over'); });
          list.addEventListener('drop', function (ev) {
            ev.preventDefault();
            Array.from(list.querySelectorAll('.files-drag-over')).forEach(function (row) { row.classList.remove('files-drag-over'); });
            var raw = ev.dataTransfer.getData('application/files-paths');
            if (!raw) return;
            var paths = JSON.parse(raw);
            var row = ev.target.closest('.files-item');
            var target = row && row.getAttribute('data-file-type') === 'folder' ? row.getAttribute('data-file-path') : scoped(current);
            Promise.all(paths.map(function (path) { return api.files.move(path, target + '/' + base(path), { overwrite: false }); })).then(load);
          });
          var lastMouseHistoryAt = 0;
          var lastMouseHistoryButton = 0;
          function mouseHistoryButton(ev) {
            if (ev.button === 3 || ev.button === 8 || ev.buttons === 8 || ev.buttons === 128 || ev.which === 8) return 'back';
            if (ev.button === 4 || ev.button === 9 || ev.buttons === 16 || ev.buttons === 256 || ev.which === 9) return 'forward';
            return '';
          }
          function mouseHistory(ev) {
            var button = mouseHistoryButton(ev);
            if (!button) return;
            ev.preventDefault();
            ev.stopPropagation();
            var now = Date.now();
            if (button === lastMouseHistoryButton && now - lastMouseHistoryAt < 120) return;
            lastMouseHistoryButton = button;
            lastMouseHistoryAt = now;
            if (button === 'back') goBack();
            else goForward();
          }
          function keyHistory(ev) {
            if (ev.defaultPrevented) return;
            if (ev.target && ['INPUT', 'SELECT', 'TEXTAREA', 'BUTTON'].indexOf(ev.target.tagName) !== -1) return;
            var key = ev.key || '';
            var ctrl = ev.ctrlKey || ev.metaKey;
            var direction = '';
            if (key === 'ArrowLeft' && ev.altKey) direction = 'back';
            else if (key === 'ArrowRight' && ev.altKey) direction = 'forward';
            else if (key === '[' && ctrl) direction = 'back';
            else if (key === ']' && ctrl) direction = 'forward';
            else if (key === 'BrowserBack' || key === 'XF86Back' || ev.keyCode === 166) direction = 'back';
            else if (key === 'BrowserForward' || key === 'XF86Forward' || ev.keyCode === 167) direction = 'forward';
            if (!direction) return;
            ev.preventDefault();
            ev.stopPropagation();
            if (direction === 'back') goBack();
            else goForward();
          }
          c.addEventListener('mousedown', mouseHistory, true);
          c.addEventListener('pointerdown', mouseHistory, true);
          window.addEventListener('pointerdown', mouseHistory, true);
          document.addEventListener('pointerdown', mouseHistory, true);
          window.addEventListener('mousedown', mouseHistory, true);
          document.addEventListener('mousedown', mouseHistory, true);
          window.addEventListener('mouseup', mouseHistory, true);
          window.addEventListener('auxclick', mouseHistory, true);
          window.addEventListener('keydown', keyHistory);
          c.addEventListener('keydown', function (ev) {
            if (ev.target && ['INPUT', 'SELECT', 'TEXTAREA', 'BUTTON'].indexOf(ev.target.tagName) !== -1) return;
            var ctrl = ev.ctrlKey || ev.metaKey;
            var key = ev.key || '';

            function currentIndex(shown) {
              if (lastClicked) {
                for (var i = 0; i < shown.length; i++) {
                  if (shown[i].relativePath === lastClicked) return i;
                }
              }
              var keys = Object.keys(selected);
              if (keys.length) {
                for (var j = 0; j < shown.length; j++) {
                  if (selected[shown[j].relativePath]) return j;
                }
              }
              return 0;
            }

            function moveSelection(delta) {
              var shown = visible();
              if (!shown.length) return;
              var index = currentIndex(shown) + delta;
              if (index < 0) index = 0;
              if (index >= shown.length) index = shown.length - 1;
              selected = {};
              selected[shown[index].relativePath] = true;
              lastClicked = shown[index].relativePath;
              render();
              var rows = list.querySelectorAll('.files-item');
              if (rows[index]) rows[index].scrollIntoView({ block: 'nearest' });
            }

            if (key === 'Escape') {
              ev.preventDefault();
              selected = {};
              lastClicked = '';
              render();
              return;
            }
            if (key === 'ArrowDown' || key === 'ArrowUp') {
              ev.preventDefault();
              moveSelection(key === 'ArrowDown' ? 1 : -1);
              return;
            }
            if (ctrl && key.toLowerCase() === 'a') { ev.preventDefault(); selected = {}; visible().forEach(function (item) { selected[item.relativePath] = true; }); render(); }
            if (ctrl && key.toLowerCase() === 'x') { ev.preventDefault(); cutSelection(); }
            if (ctrl && key.toLowerCase() === 'c') { ev.preventDefault(); copySelection(); }
            if (ctrl && key.toLowerCase() === 'v') { ev.preventDefault(); paste(); }
          });
          c.__filesCleanup = function () {
            window.removeEventListener('mousedown', mouseHistory, true);
            window.removeEventListener('pointerdown', mouseHistory, true);
            document.removeEventListener('pointerdown', mouseHistory, true);
            c.removeEventListener('pointerdown', mouseHistory, true);
            document.removeEventListener('mousedown', mouseHistory, true);
            window.removeEventListener('mouseup', mouseHistory, true);
            window.removeEventListener('auxclick', mouseHistory, true);
            window.removeEventListener('keydown', keyHistory);
            if (menu.parentNode) menu.parentNode.removeChild(menu);
          };
          load();
        },
        unmount: function (c) { if (c.__filesCleanup) c.__filesCleanup(); c.innerHTML = ''; }
      };
      window.VerstakPluginRegister('verstak.files', { components: { FilesView: FilesView } });
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
      "var SyncStatusBar={mount:function(container){container.innerHTML='';var button=document.createElement('button');button.type='button';button.className='mock-sync-status';button.textContent='Synced';button.addEventListener('click',function(){window.dispatchEvent(new CustomEvent('verstak:open-settings',{detail:{pluginId:'verstak.sync',panelId:'verstak.sync.settings'}}));});container.appendChild(button);},unmount:function(container){container.innerHTML='';}};",
      "var SyncSettings={mount:function(container){container.innerHTML='<div class=\"sync-settings-root\">Sync settings</div>';},unmount:function(container){container.innerHTML='';}};",
      "window.VerstakPluginRegister('verstak.sync',{components:{SyncStatusBar:SyncStatusBar,SyncSettings:SyncSettings}});",
      "})();"
    ].join('');
  }

  function activityBundle() {
    return '(' + function () {
      var PLUGIN_ID = 'verstak.activity';
      var LOW_VALUE_EVENT_TYPES = {
        'workspace.selected': true,
        'case.selected': true,
        'file.selected': true,
        'file.opened': true,
        'note.opened': true
      };

      function injectStyles() {
        if (document.getElementById('mock-activity-style')) return;
        var style = document.createElement('style');
        style.id = 'mock-activity-style';
        style.textContent = [
          '.activity-root{height:100%;min-height:0;display:flex;flex-direction:column;background:#0d0d1a;color:#e0e0f0}',
          '.activity-toolbar{display:flex;align-items:center;gap:.5rem;padding:.55rem .75rem;border-bottom:1px solid #16213e;background:#12122a;flex-wrap:wrap}',
          '.activity-title{font-size:.86rem;font-weight:600;color:#f0f0ff}.activity-count,.activity-status{font-size:.74rem;color:#8b8ba8}.activity-spacer{flex:1}',
          '.activity-btn{min-height:1.85rem;padding:.3rem .65rem;border:1px solid #1a3a5c;border-radius:4px;background:#0f3460;color:#e0e0f0;font-size:.76rem;cursor:pointer}.activity-btn:hover{background:#1a4a7a}.activity-btn:disabled{opacity:.45;cursor:default}.activity-btn.danger{border-color:#633;color:#ffb0b0}',
          '.activity-candidates{display:grid;gap:.5rem;padding:.65rem .75rem;border-bottom:1px solid rgba(22,33,62,.75);background:#111126}.activity-candidates-title{font-size:.76rem;font-weight:600;color:#8b8ba8;text-transform:uppercase}.activity-candidate{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:.65rem;padding:.55rem .65rem;border:1px solid rgba(78,204,163,.32);border-radius:4px;background:#14142c}.activity-candidate-title{font-size:.84rem;font-weight:600;color:#f4f7fb}.activity-candidate-facts{margin-top:.22rem;display:grid;gap:.14rem;color:#aaa;font-size:.76rem;line-height:1.4}.activity-candidate-actions{display:flex;gap:.35rem;align-items:flex-start;flex-wrap:wrap}.activity-candidate-minutes{color:#4ecca3;font-size:.76rem;white-space:nowrap}',
          '.activity-list{flex:1;min-height:0;overflow:auto;background:#101020}.activity-empty{height:100%;display:flex;align-items:center;justify-content:center;padding:1.5rem;color:#8b8ba8;font-size:.84rem;line-height:1.45;text-align:center}.activity-row{display:grid;grid-template-columns:9.5rem minmax(0,1fr);gap:.75rem;padding:.72rem .85rem;border-bottom:1px solid rgba(22,33,62,.7)}.activity-time{font-size:.72rem;color:#777;white-space:nowrap}.activity-main{min-width:0}.activity-row-head{display:flex;align-items:center;gap:.45rem;min-width:0}.activity-type{font-size:.68rem;color:#4ecca3;text-transform:uppercase;letter-spacing:.04em}.activity-title-text{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:#e0e0f0;font-size:.86rem}.activity-summary{margin-top:.25rem;color:#aaa;font-size:.78rem;line-height:1.4}.activity-source{margin-top:.25rem;color:#777;font-size:.72rem}',
          '@media(max-width:760px){.activity-row,.activity-candidate{grid-template-columns:1fr;gap:.25rem}.activity-status{width:100%}}'
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
          else if (key.indexOf('data-') === 0) node.setAttribute(key, attrs[key]);
          else if (key === 'onClick') node.addEventListener('click', attrs[key]);
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
        return 'events:workspace:' + encodeURIComponent(root || '');
      }

      function rows(value) {
        return Array.isArray(value) ? value.filter(function (item) { return item && typeof item === 'object'; }) : [];
      }

      function eventTime(value) {
        var date = new Date(value || 0);
        return isNaN(date.getTime()) ? 0 : date.getTime();
      }

      function formatDate(value) {
        var date = new Date(value || 0);
        if (isNaN(date.getTime())) return '-';
        return date.toLocaleString(undefined, { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit' });
      }

      function title(activity) {
        return activity.title || activity.summary || activity.type || activity.activityId || 'Activity event';
      }

      function workSessionCandidates(events, root) {
        var ordered = rows(events).filter(function (activity) {
          return !LOW_VALUE_EVENT_TYPES[String(activity.type || '').toLowerCase()] && eventTime(activity.occurredAt || activity.receivedAt);
        }).sort(function (a, b) {
          return eventTime(a.occurredAt || a.receivedAt) - eventTime(b.occurredAt || b.receivedAt);
        });
        var candidates = [];
        var current = null;

        function addCurrent() {
          if (!current || current.events.length < 2) return;
          var first = current.events[0];
          var last = current.events[current.events.length - 1];
          var duration = Math.round((eventTime(last.occurredAt || last.receivedAt) - eventTime(first.occurredAt || first.receivedAt)) / 60000);
          if (duration < 10) return;
          candidates.push({
            candidateId: 'work-session:' + encodeURIComponent(current.workspaceRootPath) + ':' + encodeURIComponent(first.activityId || '') + ':' + encodeURIComponent(last.activityId || ''),
            workspaceRootPath: current.workspaceRootPath,
            startedAt: new Date(eventTime(first.occurredAt || first.receivedAt)).toISOString(),
            endedAt: new Date(eventTime(last.occurredAt || last.receivedAt)).toISOString(),
            estimatedMinutes: duration,
            activityCount: current.events.length,
            activityIds: current.events.map(function (activity) { return activity.activityId; }),
            activities: current.events.map(function (activity) {
              return {
                activityId: activity.activityId,
                type: activity.type || 'activity.event',
                occurredAt: new Date(eventTime(activity.occurredAt || activity.receivedAt)).toISOString(),
                sourcePluginId: activity.sourcePluginId || '',
                workspaceRootPath: activity.workspaceRootPath || root || ''
              };
            })
          });
        }

        ordered.forEach(function (activity) {
          var workspace = activity.workspaceRootPath || root || '';
          var time = eventTime(activity.occurredAt || activity.receivedAt);
          if (!workspace) return;
          if (!current) {
            current = { workspaceRootPath: workspace, events: [activity] };
            return;
          }
          var firstTime = eventTime(current.events[0].occurredAt || current.events[0].receivedAt);
          var lastTime = eventTime(current.events[current.events.length - 1].occurredAt || current.events[current.events.length - 1].receivedAt);
          if (current.workspaceRootPath !== workspace || time - lastTime > 20 * 60 * 1000 || time - firstTime > 120 * 60 * 1000) {
            addCurrent();
            current = { workspaceRootPath: workspace, events: [activity] };
            return;
          }
          current.events.push(activity);
        });
        addCurrent();
        return candidates.sort(function (a, b) { return b.endedAt.localeCompare(a.endedAt); });
      }

      function renderActivity(containerEl, props, api) {
        injectStyles();
        var rootPath = workspaceRoot(props || {});
        var key = workspaceKey(rootPath);
        var events = [];
        var dismissed = {};
        var statusText = 'Listening for workspace activity';

        containerEl.innerHTML = '';
        var root = el('div', { className: 'activity-root', 'data-plugin-id': PLUGIN_ID });
        var toolbar = el('div', { className: 'activity-toolbar' });
        var titleEl = el('span', { className: 'activity-title', textContent: rootPath ? 'Activity · ' + rootPath : 'Activity' });
        var countEl = el('span', { className: 'activity-count' });
        var statusEl = el('span', { className: 'activity-status' });
        var clearBtn = el('button', {
          className: 'activity-btn danger',
          'data-activity-action': 'clear',
          textContent: 'Clear',
          onClick: function () {
            events = [];
            persist().then(render);
          }
        });
        var candidatesEl = el('div', { className: 'activity-candidates', 'data-activity-section': 'work-session-candidates' });
        var listEl = el('div', { className: 'activity-list' });

        toolbar.appendChild(titleEl);
        toolbar.appendChild(countEl);
        toolbar.appendChild(el('span', { className: 'activity-spacer' }));
        toolbar.appendChild(statusEl);
        toolbar.appendChild(clearBtn);
        root.appendChild(toolbar);
        root.appendChild(candidatesEl);
        root.appendChild(listEl);
        containerEl.appendChild(root);

        function readSettings() {
          if (!api || !api.settings || typeof api.settings.read !== 'function') return Promise.resolve({});
          return api.settings.read().then(function (settings) { return settings || {}; }).catch(function () { return {}; });
        }

        function persist() {
          if (!api || !api.settings || typeof api.settings.write !== 'function') return Promise.resolve();
          return api.settings.write(key, events.slice()).catch(function (err) {
            statusText = 'Could not save activity: ' + (err && err.message ? err.message : String(err));
          });
        }

        function renderCandidates() {
          candidatesEl.innerHTML = '';
          var items = workSessionCandidates(events, rootPath).filter(function (item) { return !dismissed[item.candidateId]; });
          if (!items.length) return;
          candidatesEl.appendChild(el('div', { className: 'activity-candidates-title', textContent: 'Possible journal entries' }));
          items.forEach(function (item) {
            candidatesEl.appendChild(el('div', { className: 'activity-candidate', 'data-work-session-candidate': item.candidateId }, [
              el('div', {}, [
                el('div', { className: 'activity-candidate-title', textContent: 'Possible journal entry' }),
                el('div', { className: 'activity-candidate-facts' }, [
                  el('div', { textContent: 'Workspace: ' + item.workspaceRootPath }),
                  el('div', { textContent: 'Estimated duration: ' + item.estimatedMinutes + ' min' }),
                  el('div', { textContent: 'Activities: ' + item.activityCount })
                ])
              ]),
              el('div', { className: 'activity-candidate-actions' }, [
                el('div', { className: 'activity-candidate-minutes', textContent: item.estimatedMinutes + ' min' }),
                el('button', { className: 'activity-btn', type: 'button', 'data-work-session-action': 'review', textContent: 'Review', onClick: function () {
                  window.dispatchEvent(new CustomEvent('verstak:workspace-open-tool', { detail: { kind: 'journal', toolRequest: { type: 'work-session-candidate', candidate: item } } }));
                } }),
                el('button', { className: 'activity-btn', type: 'button', 'data-work-session-action': 'dismiss', textContent: 'Dismiss', onClick: function () {
                  dismissed[item.candidateId] = true;
                  render();
                } })
              ])
            ]));
          });
        }

        function renderList() {
          listEl.innerHTML = '';
          if (!events.length) {
            listEl.appendChild(el('div', {
              className: 'activity-empty',
              textContent: 'No activity events yet. File changes, browser captures, and conversions will appear here.'
            }));
            return;
          }
          events.slice().sort(function (a, b) { return eventTime(b.occurredAt || b.receivedAt) - eventTime(a.occurredAt || a.receivedAt); }).forEach(function (activity) {
            listEl.appendChild(el('div', { className: 'activity-row', 'data-activity-id': activity.activityId }, [
              el('div', { className: 'activity-time', textContent: formatDate(activity.occurredAt || activity.receivedAt) }),
              el('div', { className: 'activity-main' }, [
                el('div', { className: 'activity-row-head' }, [
                  el('span', { className: 'activity-type', textContent: activity.type || 'activity.event' }),
                  el('span', { className: 'activity-title-text', textContent: title(activity) })
                ]),
                activity.summary ? el('div', { className: 'activity-summary', textContent: activity.summary }) : null,
                activity.sourcePluginId ? el('div', { className: 'activity-source', textContent: activity.sourcePluginId }) : null
              ])
            ]));
          });
        }

        function render() {
          countEl.textContent = events.length + ' event' + (events.length === 1 ? '' : 's');
          clearBtn.disabled = events.length === 0;
          statusEl.textContent = statusText;
          renderCandidates();
          renderList();
        }

        readSettings().then(function (settings) {
          events = rows(settings[key]);
          render();
        });
        render();
      }

      var ActivityView = {
        mount: renderActivity,
        unmount: function (containerEl) { containerEl.innerHTML = ''; }
      };
      window.VerstakPluginRegister('verstak.activity', { components: { ActivityView: ActivityView } });
    }.toString() + ')();';
  }

  function journalBundle() {
    return '(' + function () {
      var PLUGIN_ID = 'verstak.journal';

      function injectStyles() {
        if (document.getElementById('mock-journal-style')) return;
        var style = document.createElement('style');
        style.id = 'mock-journal-style';
        style.textContent = [
          '.journal-root{height:100%;min-height:0;display:flex;flex-direction:column;background:#0d0d1a;color:#e0e0f0}',
          '.journal-toolbar{display:flex;align-items:center;gap:.5rem;padding:.55rem .75rem;border-bottom:1px solid #16213e;background:#12122a}.journal-title{font-size:.86rem;font-weight:600;color:#f0f0ff}.journal-count,.journal-status{font-size:.74rem;color:#8b8ba8}.journal-spacer{flex:1}',
          '.journal-btn{min-height:1.85rem;padding:.3rem .65rem;border:1px solid #1a3a5c;border-radius:4px;background:#0f3460;color:#e0e0f0;font-size:.76rem;cursor:pointer}.journal-btn.primary{background:#4ecca3;border-color:#4ecca3;color:#102018}',
          '.journal-list{flex:1;min-height:0;overflow:auto;padding:.5rem .75rem}.journal-empty{padding:1.5rem;color:#8b8ba8}.journal-row{display:grid;grid-template-columns:8rem minmax(0,1fr) auto;gap:.7rem;padding:.65rem 0;border-bottom:1px solid rgba(22,33,62,.75)}.journal-entry-title{font-weight:600}.journal-summary,.journal-meta{margin-top:.22rem;font-size:.76rem;color:#aaa}.journal-minutes{color:#4ecca3;font-size:.78rem;white-space:nowrap}',
          '.journal-modal-host[hidden]{display:none}.journal-modal-overlay{position:fixed;inset:0;z-index:10000;display:flex;align-items:center;justify-content:center;padding:1rem;background:rgba(0,0,0,.58)}.journal-modal{width:520px;max-width:96vw;display:grid;gap:.75rem;padding:1rem;border:1px solid #2c456a;border-radius:8px;background:#15152c;box-shadow:0 18px 44px rgba(0,0,0,.38)}.journal-modal-title{font-size:.95rem;font-weight:600}.journal-modal-grid{display:grid;grid-template-columns:1fr 8rem;gap:.6rem}.journal-field{display:grid;gap:.3rem;font-size:.72rem;color:#8b8ba8}.journal-field.wide{grid-column:1/-1}.journal-input{min-width:0;padding:.38rem .5rem;border:1px solid #2c456a;border-radius:4px;background:#0f1424;color:#f4f7fb;font:inherit}.journal-input.textarea{min-height:6rem;resize:vertical}.journal-candidate-context{display:grid;gap:.2rem;padding:.65rem;border:1px solid rgba(78,204,163,.34);border-radius:6px;background:#111126;font-size:.76rem;color:#b7c0d4}.journal-candidate-activities{display:grid;gap:.35rem;margin:0;padding:.65rem;border:1px solid #202b46;border-radius:6px;font-size:.74rem;color:#b7c0d4}.journal-candidate-activity{display:flex;align-items:flex-start;gap:.45rem}.journal-modal-actions{display:flex;justify-content:flex-end;gap:.5rem}'
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
          else if (key === 'checked') node.checked = !!attrs[key];
          else if (key === 'value') node.value = attrs[key];
          else if (key.indexOf('data-') === 0) node.setAttribute(key, attrs[key]);
          else if (key === 'onClick') node.addEventListener('click', attrs[key]);
          else node[key] = attrs[key];
        });
        (children || []).forEach(function (child) {
          if (child) node.appendChild(typeof child === 'string' ? document.createTextNode(child) : child);
        });
        return node;
      }

      function workspaceRoot(props) {
        return String(props.workspaceRootPath || (props.workspaceNode && (props.workspaceNode.rootPath || props.workspaceNode.name || props.workspaceNode.id)) || props.workspaceName || '').trim();
      }

      function workspaceKey(root) {
        return 'worklog:workspace:' + encodeURIComponent(root || '');
      }

      function rows(value) {
        return Array.isArray(value) ? value.filter(function (item) { return item && typeof item === 'object'; }) : [];
      }

      function candidateFromProps(props, root) {
        var request = props && props.toolRequest;
        var candidate = request && request.type === 'work-session-candidate' ? request.candidate : null;
        if (!candidate || candidate.workspaceRootPath !== root || !candidate.candidateId) return null;
        var activities = rows(candidate.activities).filter(function (activity) { return activity.activityId; });
        return {
          candidateId: String(candidate.candidateId),
          workspaceRootPath: root,
          startedAt: candidate.startedAt || '',
          endedAt: candidate.endedAt || '',
          estimatedMinutes: Number(candidate.estimatedMinutes || 0),
          activityCount: Number(candidate.activityCount || activities.length),
          activities: activities,
          activityIds: Array.isArray(candidate.activityIds) ? candidate.activityIds : activities.map(function (activity) { return activity.activityId; })
        };
      }

      function completedTodoFromProps(props, root) {
        var request = props && props.toolRequest;
        var todo = request && request.type === 'completed-todo' ? request.todo : null;
        if (!todo || todo.workspaceRootPath !== root || !todo.id || !todo.title) return null;
        return {
          id: String(todo.id),
          title: String(todo.title),
          description: String(todo.description || todo.body || ''),
          workspaceRootPath: root,
          completedAt: String(todo.completedAt || '')
        };
      }

      function candidateDate(value) {
        var date = new Date(value || '');
        return isNaN(date.getTime()) ? new Date().toISOString().slice(0, 10) : date.toISOString().slice(0, 10);
      }

      function candidateTime(value) {
        var date = new Date(value || '');
        return isNaN(date.getTime()) ? String(value || '') : date.toLocaleString(undefined, { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit' });
      }

      function JournalView() {}

      JournalView.mount = function (containerEl, props, api) {
        injectStyles();
        var rootPath = workspaceRoot(props || {});
        var key = workspaceKey(rootPath);
        var entries = [];
        var modalHost = el('div', { className: 'journal-modal-host', hidden: true });
        containerEl.innerHTML = '';
        var root = el('div', { className: 'journal-root', 'data-plugin-id': PLUGIN_ID });
        var toolbar = el('div', { className: 'journal-toolbar' });
        var countEl = el('span', { className: 'journal-count' });
        var addBtn = el('button', { className: 'journal-btn primary', 'data-journal-action': 'add', textContent: 'Add', onClick: function () { showEntryModal(null); } });
        var listEl = el('div', { className: 'journal-list' });
        toolbar.appendChild(el('span', { className: 'journal-title', textContent: rootPath ? 'Journal · ' + rootPath : 'Journal' }));
        toolbar.appendChild(countEl);
        toolbar.appendChild(el('span', { className: 'journal-spacer' }));
        toolbar.appendChild(addBtn);
        root.appendChild(toolbar);
        root.appendChild(listEl);
        root.appendChild(modalHost);
        containerEl.appendChild(root);

        function closeModal() {
          modalHost.innerHTML = '';
          modalHost.hidden = true;
        }

        function persist() {
          return api.settings.write(key, entries);
        }

        function showEntryModal(candidate, completedTodo) {
          var reviewing = !!candidate;
          var reviewingTodo = !reviewing && !!completedTodo;
          var titleInput = el('input', { className: 'journal-input', type: 'text', value: reviewingTodo ? completedTodo.title : '', 'data-journal-input': 'title' });
          var summaryInput = el('textarea', { className: 'journal-input textarea', value: reviewingTodo ? completedTodo.description : '', 'data-journal-input': 'summary' });
          var minutesInput = el('input', { className: 'journal-input', type: 'number', value: reviewing ? String(candidate.estimatedMinutes) : (reviewingTodo ? '0' : '30'), 'data-journal-input': 'minutes' });
          var dateInput = el('input', { className: 'journal-input', type: 'date', value: reviewing ? candidateDate(candidate.startedAt) : (reviewingTodo ? candidateDate(completedTodo.completedAt) : new Date().toISOString().slice(0, 10)), 'data-journal-input': 'date' });
          var billableInput = el('input', { type: 'checkbox', checked: false, 'data-journal-input': 'billable' });
          var activityInputs = reviewing ? candidate.activities.map(function (activity) {
            return { activity: activity, input: el('input', { type: 'checkbox', checked: true, 'data-journal-candidate-activity': activity.activityId }) };
          }) : [];
          var context = reviewing ? el('div', { className: 'journal-candidate-context', 'data-journal-candidate': candidate.candidateId }, [
            el('strong', { textContent: 'Possible journal entry' }),
            el('div', { textContent: 'Workspace: ' + candidate.workspaceRootPath }),
            el('div', { textContent: 'Time: ' + candidateTime(candidate.startedAt) + ' - ' + candidateTime(candidate.endedAt) }),
            el('div', { textContent: 'Estimated duration: ' + candidate.estimatedMinutes + ' min' }),
            el('div', { textContent: 'Activities: ' + candidate.activityCount })
          ]) : null;
          var todoContext = reviewingTodo ? el('div', { className: 'journal-candidate-context', 'data-journal-todo': completedTodo.id }, [
            el('strong', { textContent: 'Completed todo' }),
            el('div', { textContent: 'Workspace: ' + completedTodo.workspaceRootPath }),
            completedTodo.completedAt ? el('div', { textContent: 'Completed: ' + candidateTime(completedTodo.completedAt) }) : null
          ]) : null;
          var linked = reviewing ? el('fieldset', { className: 'journal-candidate-activities' }, [
            el('legend', { textContent: 'Linked activities' })
          ].concat(activityInputs.map(function (item) {
            return el('label', { className: 'journal-candidate-activity' }, [item.input, (item.activity.type || 'activity.event') + ' · ' + item.activity.activityId]);
          }))) : null;

          function save() {
            var title = String(titleInput.value || '').trim();
            if (!title) return;
            if (reviewingTodo && entries.some(function (entry) { return entry.sourceTodoId === completedTodo.id; })) return;
            var entry = {
              entryId: 'journal:' + Date.now(),
              workspaceRootPath: rootPath,
              date: dateInput.value,
              title: title,
              summary: String(summaryInput.value || ''),
              minutes: Number(minutesInput.value || 0),
              billable: billableInput.checked === true,
              sourceCandidateId: reviewing ? candidate.candidateId : '',
              sourceTodoId: reviewingTodo ? completedTodo.id : '',
              activityIds: reviewing ? activityInputs.filter(function (item) { return item.input.checked; }).map(function (item) { return item.activity.activityId; }) : []
            };
            entries = [entry].concat(entries);
            closeModal();
            persist().then(render);
          }

          modalHost.innerHTML = '';
          modalHost.hidden = false;
          modalHost.appendChild(el('div', { className: 'journal-modal-overlay' }, [
            el('div', { className: 'journal-modal' }, [
              el('div', { className: 'journal-modal-title', textContent: reviewing ? 'Review possible journal entry' : (reviewingTodo ? 'Create journal entry from completed todo' : 'Add journal entry') }),
              context,
              todoContext,
              el('div', { className: 'journal-modal-grid' }, [
                el('label', { className: 'journal-field' }, ['Date', dateInput]),
                el('label', { className: 'journal-field' }, ['Minutes', minutesInput]),
                el('label', { className: 'journal-field wide' }, ['Title', titleInput]),
                el('label', { className: 'journal-field wide' }, ['Body', summaryInput]),
                el('label', { className: 'journal-field wide' }, [billableInput, 'Billable'])
              ]),
              linked,
              el('div', { className: 'journal-modal-actions' }, [
                el('button', { className: 'journal-btn', textContent: 'Cancel', onClick: closeModal }),
                el('button', { className: 'journal-btn primary', 'data-journal-action': 'save-entry', textContent: 'Add entry', onClick: save })
              ])
            ])
          ]));
          titleInput.focus();
        }

        function render() {
          countEl.textContent = entries.length + ' entr' + (entries.length === 1 ? 'y' : 'ies');
          listEl.innerHTML = '';
          if (!entries.length) {
            listEl.appendChild(el('div', { className: 'journal-empty', textContent: 'No journal entries yet.' }));
            return;
          }
          entries.forEach(function (entry) {
            var activityIds = Array.isArray(entry.activityIds) ? entry.activityIds : (Array.isArray(entry.eventIds) ? entry.eventIds : []);
            listEl.appendChild(el('div', { className: 'journal-row', 'data-journal-entry': entry.entryId }, [
              el('div', { textContent: entry.date }),
              el('div', {}, [
                el('div', { className: 'journal-entry-title', textContent: entry.title }),
                entry.summary ? el('div', { className: 'journal-summary', textContent: entry.summary }) : null,
                el('div', { className: 'journal-meta', textContent: entry.workspaceRootPath + (activityIds.length ? ' · ' + activityIds.length + ' linked activities' : '') + (entry.sourceTodoId ? ' · linked todo' : '') })
              ]),
              el('div', { className: 'journal-minutes', textContent: entry.minutes + ' min' })
            ]));
          });
        }

        api.settings.read(key).then(function (stored) {
          entries = rows(stored);
          render();
          var candidate = candidateFromProps(props || {}, rootPath);
          var completedTodo = completedTodoFromProps(props || {}, rootPath);
          if (candidate) showEntryModal(candidate);
          else if (completedTodo) showEntryModal(null, completedTodo);
        }).catch(function () { render(); });
        render();
      };

      JournalView.unmount = function (containerEl) { containerEl.innerHTML = ''; };
      window.VerstakPluginRegister('verstak.journal', { components: { JournalView: JournalView } });
    }.toString() + ')();';
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
      if (pluginId === 'verstak.platform-test' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(platformTestBundle());
      }
      if (pluginId === 'verstak.default-editor') {
        return Promise.resolve(defaultEditorSource);
      }
      if (pluginId === 'verstak.files' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(filesPluginBundle());
      }
      if (pluginId === 'verstak.trash' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(trashPluginBundle());
      }
      if (pluginId === 'verstak.notes' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(simplePluginBundle('verstak.notes', 'NotesView', 'notes-root', 'Notes'));
      }
      if (pluginId === 'verstak.sync' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(syncPluginBundle());
      }
      if (pluginId === 'verstak.activity') {
        return Promise.resolve(activitySource);
      }
      if (pluginId === 'verstak.journal' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(journalSource);
      }
      if (pluginId === 'verstak.browser-inbox' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(browserInboxBundle());
      }
      if (pluginId === 'verstak.todo' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(todoSource);
      }
      if (pluginId === 'verstak.secrets') {
        return Promise.resolve(secretsSource);
      }
      if (pluginId === 'verstak.search' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(searchPluginBundle());
      }
      if (pluginId === 'verstak.import' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(importSource);
      }
      if (pluginId === 'verstak.import' && assetPath === 'frontend/dist/style.css') {
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
    PublishPluginEvent: function () { return Promise.resolve(''); },
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
      appSettings = Object.assign({}, appSettings, patch || {});
      if (patch && patch.language) localStorage.setItem('verstak-test-language', patch.language);
      return Promise.resolve('');
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
      pluginStates = {
        'verstak.platform-test': {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: 'verstak.platform-test',
            name: 'Platform Test',
            version: '0.1.0',
            apiVersion: '0.1.0',
            description: 'Runtime test plugin for verifying the Verstak platform.',
            source: 'official',
            icon: '🧪',
            provides: ['verstak/platform-test/v1', 'verstak/diagnostics/v1'],
            requires: ['verstak/core/plugin-manager/v1', 'verstak/core/capability-registry/v1'],
            optionalRequires: ['verstak/core/vault/v1', 'verstak/core/sync/v1', 'verstak/core/files/v1', 'verstak/core/workbench/v1'],
            permissions: ['vault.read', 'events.publish', 'events.subscribe', 'ui.register', 'commands.register', 'storage.namespace', 'files.read', 'files.write', 'files.delete', 'files.openExternal', 'workbench.open'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              views: [
                { id: 'verstak.platform-test.diagnostics', title: 'Platform Diagnostics', icon: '🧪', component: 'DiagnosticsPanel' }
              ],
              commands: [
                { id: 'verstak.platform-test.run-tests', title: 'Run Platform Tests', handler: 'runAllTests' },
                { id: 'verstak.platform-test.show-version', title: 'Show Version Info', handler: 'showVersion' }
              ],
              sidebarItems: [
                { id: 'verstak.platform-test.sidebar', title: 'Platform Test', icon: '🧪', view: 'verstak.platform-test.diagnostics', position: 100 }
              ],
              statusBarItems: [
                { id: 'verstak.platform-test.status', label: '🧪 All Tests Pass', position: 'right', handler: 'openDiagnostics' }
              ],
              settingsPanels: [
                { id: 'verstak.platform-test.settings', title: 'Platform Test Settings', icon: '🧪', component: 'PlatformTestSettings' }
              ],
              openProviders: [
                {
                  id: 'verstak.platform-test.markdown-diagnostic',
                  title: 'Platform Test Markdown Diagnostic',
                  priority: 10,
                  component: 'MarkdownDiagnosticProvider',
                  supports: [
                    { kind: 'vault-file', extensions: ['.md', '.markdown'], contexts: ['generic-markdown', 'notes-markdown'] }
                  ]
                }
              ]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/platform-test',
          error: ''
        },
        'verstak.default-editor': {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: 'verstak.default-editor',
            name: 'Default Editor',
            version: '0.1.0',
            apiVersion: '0.1.0',
            description: 'Built-in text and markdown editor/viewer.',
            source: 'official',
            icon: 'edit',
            provides: ['verstak/default-editor/v1'],
            requires: ['verstak/core/files/v1', 'verstak/core/workbench/v1'],
            permissions: ['files.read', 'files.write', 'workbench.open'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              openProviders: [
                {
                  id: 'verstak.default-editor.text',
                  title: 'Default Text Editor',
                  priority: 50,
                  component: 'DefaultEditor',
                  supports: [
                    { kind: 'vault-file', extensions: ['.txt', '.log', '.conf', '.ini', '.toml', '.yaml', '.yml', '.json', '.csv'], mime: ['text/plain', 'application/json'], contexts: ['generic-text'] }
                  ]
                },
                {
                  id: 'verstak.default-editor.markdown',
                  title: 'Default Markdown Editor',
                  priority: 50,
                  component: 'DefaultEditor',
                  supports: [
                    { kind: 'vault-file', extensions: ['.md', '.markdown'], contexts: ['generic-markdown'] }
                  ]
                },
                {
                  id: 'verstak.default-editor.notes-markdown',
                  title: 'Default Notes Markdown Editor',
                  priority: 50,
                  component: 'DefaultEditor',
                  supports: [
                    { kind: 'vault-file', extensions: ['.md', '.markdown'], contexts: ['notes-markdown'] }
                  ]
                }
              ]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/default-editor',
          error: ''
        },
        'verstak.files': {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: 'verstak.files',
            name: 'Files',
            version: '0.1.0',
            apiVersion: '0.1.0',
            description: 'Minimal vault file navigator.',
            source: 'official',
            icon: 'folder',
            provides: ['verstak/files/v1'],
            requires: ['verstak/core/files/v1', 'verstak/core/workbench/v1'],
            permissions: ['files.read', 'files.write', 'files.delete', 'files.openExternal', 'workbench.open', 'ui.register'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              views: [{ id: 'verstak.files.view', title: 'Files', icon: 'folder', component: 'FilesView' }],
              workspaceItems: [{ id: 'verstak.files.workspace', title: 'Files', icon: 'folder', component: 'FilesView' }]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/files',
          error: ''
        },
        'verstak.trash': makeTrashPluginState(),
        'verstak.notes': {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: 'verstak.notes',
            name: 'Notes',
            version: '0.1.0',
            apiVersion: '0.1.0',
            description: 'Deal-scoped notes manager.',
            source: 'official',
            icon: 'edit',
            provides: ['verstak/notes/v1'],
            requires: ['verstak/core/files/v1', 'verstak/core/workbench/v1'],
            permissions: ['files.read', 'files.write', 'files.delete', 'events.subscribe', 'workbench.open', 'ui.register'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              workspaceItems: [{ id: 'verstak.notes.workspace', title: 'Notes', icon: 'edit', component: 'NotesView' }]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/notes',
          error: ''
        },
        'verstak.sync': {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: 'verstak.sync',
            name: 'Sync',
            version: '0.1.0',
            apiVersion: '0.1.0',
            description: 'Synchronize vault data across devices.',
            source: 'official',
            icon: 'refresh-cw',
            provides: ['verstak/sync/v1', 'verstak/sync.status/v1'],
            requires: ['verstak/core/files/v1'],
            permissions: ['files.read', 'files.write', 'network.remote', 'storage.namespace', 'sync.participate', 'ui.register'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              settingsPanels: [{ id: 'verstak.sync.settings', title: 'Sync', component: 'SyncSettings' }],
              statusBarItems: [{ id: 'verstak.sync.status', label: 'Sync', position: 'right', handler: 'SyncStatusBar' }]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/sync',
          error: ''
        },
        'verstak.activity': {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: 'verstak.activity',
            name: 'Activity',
            version: '0.1.0',
            apiVersion: '0.1.0',
            description: 'Deal-scoped activity log for public plugin events.',
            source: 'official',
            icon: 'activity',
            provides: ['activity.log', 'activity.provider', 'activity.reconstruction'],
            permissions: ['events.subscribe', 'storage.namespace', 'ui.register'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              views: [{ id: 'verstak.activity.view', title: 'Activity', icon: 'activity', component: 'ActivityView' }],
              sidebarItems: [{ id: 'verstak.activity.sidebar', title: 'Activity', icon: 'activity', view: 'verstak.activity.view', position: 20 }],
              workspaceItems: [{ id: 'verstak.activity.workspace', title: 'Activity', icon: 'activity', component: 'ActivityView' }]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/activity',
          error: ''
        },
        'verstak.journal': {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: 'verstak.journal',
            name: 'Journal',
            version: '0.1.0',
            apiVersion: '0.1.0',
            description: 'Deal-scoped journal with user-authored entries and optional Activity links.',
            source: 'official',
            icon: 'book-open',
            provides: ['worklog', 'journal', 'report.worklog'],
            permissions: ['events.publish', 'files.read', 'storage.namespace', 'ui.register'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              views: [{ id: 'verstak.journal.view', title: 'Journal', icon: 'book-open', component: 'JournalView' }],
              sidebarItems: [{ id: 'verstak.journal.sidebar', title: 'Journal', icon: 'book-open', view: 'verstak.journal.view', position: 30 }],
              workspaceItems: [{ id: 'verstak.journal.workspace', title: 'Journal', icon: 'book-open', component: 'JournalView' }]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/journal',
          error: ''
        },
        'verstak.browser-inbox': {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: 'verstak.browser-inbox',
            name: 'Browser',
            version: '0.1.0',
            apiVersion: '0.1.0',
            description: 'Global browser materials with explicit Deal assignment.',
            source: 'official',
            icon: 'inbox',
            provides: ['browser.inbox'],
            permissions: ['events.subscribe', 'files.read', 'storage.namespace', 'ui.register'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              views: [{ id: 'verstak.browser-inbox.view', title: 'Browser', icon: 'inbox', component: 'BrowserInboxView' }],
              sidebarItems: [{ id: 'verstak.browser-inbox.sidebar', title: 'Browser', icon: 'inbox', view: 'verstak.browser-inbox.view', position: 30 }],
              workspaceItems: [{ id: 'verstak.browser-inbox.workspace', title: 'Browser', icon: 'inbox', component: 'BrowserInboxView' }]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/browser-inbox',
          error: ''
        },
        'verstak.todo': makeTodoPluginState(),
        'verstak.secrets': makeSecretsPluginState(),
        'verstak.import': makeImportPluginState(),
        'verstak.search': {
          status: 'loaded',
          enabled: true,
          manifest: {
            schemaVersion: 1,
            id: 'verstak.search',
            name: 'Search',
            version: '0.1.0',
            apiVersion: '0.1.0',
            description: 'Deal-scoped vault text search provider.',
            source: 'official',
            icon: 'search',
            provides: ['verstak/search/v1', 'search.provider'],
            requires: ['verstak/core/files/v1', 'verstak/core/workbench/v1'],
            permissions: ['files.read', 'workbench.open', 'storage.namespace', 'ui.register', 'events.subscribe', 'commands.register'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              workspaceItems: [{ id: 'verstak.search.workspace', title: 'Search', icon: 'search', component: 'SearchView' }],
              commands: [{ id: 'verstak.search.searchVaultText', title: 'Search Vault Text', handler: 'verstak.search.searchVaultText' }],
              searchProviders: [{ id: 'verstak.search.vault-text', label: 'Vault Text Search', handler: 'verstak.search.searchVaultText' }]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/search',
          error: ''
        }
      };
      vaultStatus = { status: 'open', path: '/tmp/verstak-test/vault', vaultId: 'test-vault-001' };
      vaultPluginState = { enabledPlugins: ['verstak.platform-test', 'verstak.default-editor', 'verstak.files', 'verstak.notes', 'verstak.sync', 'verstak.activity', 'verstak.journal', 'verstak.browser-inbox', 'verstak.search'], disabledPlugins: [], desiredPlugins: [{ id: 'verstak.platform-test', version: '0.1.0', source: 'official' }, { id: 'verstak.default-editor', version: '0.1.0', source: 'official' }, { id: 'verstak.files', version: '0.1.0', source: 'official' }, { id: 'verstak.notes', version: '0.1.0', source: 'official' }, { id: 'verstak.sync', version: '0.1.0', source: 'official' }, { id: 'verstak.activity', version: '0.1.0', source: 'official' }, { id: 'verstak.journal', version: '0.1.0', source: 'official' }, { id: 'verstak.browser-inbox', version: '0.1.0', source: 'official' }, { id: 'verstak.search', version: '0.1.0', source: 'official' }] };
      vaultPluginState.enabledPlugins.push('verstak.trash');
      vaultPluginState.desiredPlugins.push({ id: 'verstak.trash', version: '0.1.0', source: 'official' });
      vaultPluginState.enabledPlugins.push('verstak.todo');
      vaultPluginState.desiredPlugins.push({ id: 'verstak.todo', version: '0.1.0', source: 'official' });
      vaultPluginState.enabledPlugins.push('verstak.secrets');
      vaultPluginState.desiredPlugins.push({ id: 'verstak.secrets', version: '0.1.0', source: 'official' });
      vaultPluginState.enabledPlugins.push('verstak.import');
      vaultPluginState.desiredPlugins.push({ id: 'verstak.import', version: '0.1.0', source: 'official' });
      localStorage.removeItem('verstak-test-language');
      appSettings = { currentVaultPath: '/tmp/verstak-test/vault', recentVaults: [], language: 'system' };
      workbenchPreferences = {};
      openedResources = [];
      pluginSettings = { 'verstak.platform-test': { savedText: 'initial value' } };
      pluginNotifications = {};
      pluginData = {};
      secretRecords = makeDefaultSecretRecords();
      vaultFiles = makeDefaultVaultFiles();
      externalOpens = [];
      trashEntries = [];
      trashPayloads = {};
      window.__wailsMockExternalOpens = [];
      workspaceTree = makeDefaultWorkspaceTree();
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
