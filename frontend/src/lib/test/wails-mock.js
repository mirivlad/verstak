/**
 * Wails Mock Bridge — эмулирует window['go']['api']['App'] для тестового окружения.
 *
 * Каждый метод возвращает Promise с данными, совместимыми с Wails-контрактом.
 * Состояние мутабельно — тесты могут менять его между сценариями.
 */
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
          statusBarItems: [{ id: 'verstak.sync.status', label: 'Sync', position: 'right' }]
        }
      },
      rootPath: '/tmp/verstak-test/plugins/sync',
      error: ''
    }
  };

  var vaultStatus = { status: 'open', path: '/tmp/verstak-test/vault', vaultId: 'test-vault-001' };
  var vaultPluginState = { enabledPlugins: ['verstak.platform-test', 'verstak.default-editor', 'verstak.files', 'verstak.sync'], disabledPlugins: [], desiredPlugins: [{ id: 'verstak.platform-test', version: '0.1.0', source: 'official' }, { id: 'verstak.default-editor', version: '0.1.0', source: 'official' }, { id: 'verstak.files', version: '0.1.0', source: 'official' }, { id: 'verstak.sync', version: '0.1.0', source: 'official' }] };
  var appSettings = { currentVaultPath: '/tmp/verstak-test/vault', recentVaults: [] };
  var workbenchPreferences = {};
  var openedResources = [];
  var pluginSettings = {
    'verstak.platform-test': { savedText: 'initial value' }
  };
  var vaultFiles = makeDefaultVaultFiles();
  var externalOpens = [];
  window.__wailsMockExternalOpens = [];
  var workspaceTree = makeDefaultWorkspaceTree();
  var reloadResponseMode = 'tuple';
  var syncState = makeDefaultSyncState();

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

  function makeDefaultSyncState() {
    return {
      configured: false,
      serverUrl: '',
      deviceId: 'mock-device',
      deviceName: '',
      connected: false,
      revoked: false,
      tokenStored: false,
      unpushedOps: 0,
      lastSyncAt: '',
      syncInterval: 0,
      lastError: '',
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
    if (parts[0] && parts[0].toLowerCase() === '.verstak') return { error: 'reserved-path: .verstak is internal' };
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
      { name: 'sync.participate', description: 'Participate in vault sync', dangerous: true }
    ];
  }

  function syncStatusDTO() {
    return {
      configured: syncState.configured,
      serverUrl: syncState.serverUrl,
      deviceId: syncState.deviceId,
      deviceName: syncState.deviceName,
      connected: syncState.connected,
      revoked: syncState.revoked,
      tokenStored: syncState.tokenStored,
      unpushedOps: syncState.unpushedOps,
      lastSyncAt: syncState.lastSyncAt,
      syncInterval: syncState.syncInterval,
      lastError: syncState.lastError,
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
    var views = [], commands = [], sidebarItems = [], statusBarItems = [], settingsPanels = [], openProviders = [], workspaceItems = [];
    for (var id in pluginStates) {
      var s = pluginStates[id];
      var c = (s.manifest && s.manifest.contributes) || {};
      if (c.views) c.views.forEach(function (v) { views.push(Object.assign({}, v, { pluginId: id })); });
      if (c.commands) c.commands.forEach(function (cmd) { commands.push(Object.assign({}, cmd, { pluginId: id })); });
      if (c.sidebarItems) c.sidebarItems.forEach(function (sb) { sidebarItems.push(Object.assign({}, sb, { pluginId: id })); });
      if (c.statusBarItems) c.statusBarItems.forEach(function (st) { statusBarItems.push(Object.assign({}, st, { pluginId: id })); });
      if (c.settingsPanels) c.settingsPanels.forEach(function (sp) { settingsPanels.push(Object.assign({}, sp, { pluginId: id })); });
      if (c.openProviders) c.openProviders.forEach(function (op) { openProviders.push(Object.assign({}, op, { pluginId: id })); });
      if (c.workspaceItems) c.workspaceItems.forEach(function (wi) { workspaceItems.push(Object.assign({}, wi, { pluginId: id })); });
    }
    return { views: views, commands: commands, sidebarItems: sidebarItems, statusBarItems: statusBarItems, settingsPanels: settingsPanels, openProviders: openProviders, workspaceItems: workspaceItems };
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
      var FilesView = {
        mount: function (c, p, api) {
          if (!document.getElementById('mock-files-styles')) {
            var style = document.createElement('style');
            style.id = 'mock-files-styles';
            style.textContent = '.files-root{display:flex;flex-direction:column;height:100%;min-height:0;background:#0d0d1a;color:#e0e0e0;outline:0}.files-toolbar{display:flex;align-items:center;gap:.4rem;padding:.5rem .75rem;background:#12122a;border-bottom:1px solid #16213e;flex-wrap:wrap}.files-toolbar-btn,.files-row-btn{display:inline-flex;align-items:center;justify-content:center;border:1px solid #333;border-radius:4px;background:#1a1a2e;color:#ccc;cursor:pointer}.files-toolbar-btn{width:2rem;height:2rem}.files-row-btn{width:1.75rem;height:1.75rem}.files-toolbar-btn svg,.files-row-btn svg{width:16px;height:16px}.files-breadcrumb{flex:1;min-width:150px;color:#8b8ba8}.files-breadcrumb-item{color:#4ecca3;cursor:pointer}.files-breadcrumb-current{color:#ddd}.files-filter,.files-sort,.files-create-input,.files-rename-input{font-size:.78rem;padding:.32rem .5rem;border:1px solid #333;border-radius:4px;background:#0d0d1a;color:#e0e0e0}.files-sort{appearance:none;background-color:#0d0d1a;padding-right:1rem}.files-list{flex:1;overflow:auto}.files-header,.files-item{display:grid;grid-template-columns:minmax(160px,1fr) 90px 90px 150px 160px;align-items:center;gap:.5rem;padding:.38rem .75rem;border-bottom:1px solid rgba(22,33,62,.55)}.files-header{background:#101028;color:#8b8ba8;font-size:.7rem;text-transform:uppercase}.files-item:hover{background:#17172d}.files-item.selected{background:#1a2a3a}.files-namecell{display:flex;align-items:center;gap:.5rem;min-width:0}.files-item-icon{width:1.25rem;color:#8b8ba8}.files-item-name{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.files-item-meta{font-size:.74rem;color:#777;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.files-row-actions{display:flex;justify-content:flex-end;gap:.35rem}.files-panel{display:flex;gap:.5rem;padding:.5rem .75rem;border-top:1px solid #16213e;background:#12122a}.files-create-input,.files-rename-input{flex:1}.files-ctx-menu{position:fixed;z-index:9999;min-width:170px;background:#1a1a2e;border:1px solid #333;border-radius:6px;padding:6px 0;box-shadow:0 8px 24px rgba(0,0,0,.5);font-size:.84rem;color:#e0e0e0}.files-ctx-menu-item{display:flex;align-items:center;gap:.5rem;padding:6px 16px;cursor:pointer}.files-ctx-menu-item:hover{background:#2a2a4e}.files-ctx-menu-item svg{width:14px;height:14px}.files-ctx-menu-sep{height:1px;background:#333;margin:4px 8px}.files-drag-over{outline:2px dashed #4ecca3;outline-offset:-2px}';
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
          function btn(title, action, fn) { return e('button', { className: 'files-toolbar-btn', 'data-files-action': action, title: title, 'aria-label': title, innerHTML: SVG, onClick: fn }, []); }
          function rowBtn(title, action, fn) { return e('button', { className: 'files-row-btn', 'data-files-action': action, title: title, 'aria-label': title, innerHTML: SVG, onClick: fn }, []); }
          toolbar.appendChild(breadcrumb);
          toolbar.appendChild(btn('Back', 'back', goBack));
          toolbar.appendChild(btn('Forward', 'forward', goForward));
          toolbar.appendChild(btn('Up', 'up', function () { if (current) nav(parent(current)); }));
          toolbar.appendChild(btn('Refresh', 'refresh', load));
          toolbar.appendChild(btn('New folder', 'new-folder', function () { startCreate('folder'); }));
          toolbar.appendChild(btn('New markdown file', 'new-markdown', function () { startCreate('markdown'); }));
          toolbar.appendChild(btn('New text file', 'new-text', function () { startCreate('text'); }));
          toolbar.appendChild(btn('Open', 'open', function () { open(firstSelected()); }));
          toolbar.appendChild(btn('Rename', 'rename', function () { startRename(firstSelected()); }));
          toolbar.appendChild(btn('Move to trash', 'trash', function () { trashSelection(); }));
          toolbar.appendChild(btn('Cut', 'cut', cutSelection));
          toolbar.appendChild(btn('Copy', 'copy', copySelection));
          toolbar.appendChild(btn('Paste', 'paste', paste));
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
          var createInput = e('input', { className: 'files-create-input', 'data-files-create-input': '' }, []);
          createPanel.appendChild(createInput);
          createPanel.appendChild(e('button', { className: 'files-toolbar-btn', 'data-files-create-confirm': '', onClick: confirmCreate }, ['Create']));
          createPanel.appendChild(e('button', { className: 'files-toolbar-btn', onClick: function () { createPanel.style.display = 'none'; } }, ['Cancel']));
          c.appendChild(createPanel);
          var renamePanel = e('div', { className: 'files-panel', style: 'display:none' }, []);
          var renameInput = e('input', { className: 'files-rename-input', 'data-files-rename-input': '' }, []);
          renamePanel.appendChild(renameInput);
          renamePanel.appendChild(e('button', { className: 'files-toolbar-btn', 'data-files-rename-confirm': '', onClick: confirmRename }, ['Rename']));
          renamePanel.appendChild(e('button', { className: 'files-toolbar-btn', onClick: function () { renamePanel.style.display = 'none'; } }, ['Cancel']));
          c.appendChild(renamePanel);
          function entryByPath(path) { return entries.find(function (item) { return item.relativePath === path; }) || null; }
          function selectedEntries() { return Object.keys(selected).map(entryByPath).filter(Boolean); }
          function firstSelected() { return selectedEntries()[0] || null; }
          function updateBreadcrumb() {
            breadcrumb.innerHTML = '';
            breadcrumb.appendChild(e('span', { className: 'files-breadcrumb-item', onClick: function () { nav(''); } }, [workspaceName]));
            if (current) breadcrumb.appendChild(e('span', { className: 'files-breadcrumb-current' }, [' / ' + current]));
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
            shown.forEach(function (item) {
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
                  ev.dataTransfer.setData('application/files-paths', JSON.stringify(Object.keys(selected)));
                  ev.dataTransfer.effectAllowed = 'move';
                }
              }, []);
              row.appendChild(e('span', { className: 'files-namecell' }, [e('span', { className: 'files-item-icon', innerHTML: item.type === 'folder' ? FOLDER_SVG : SVG }, []), e('span', { className: 'files-item-name' }, [item.name])]));
              row.appendChild(e('span', { className: 'files-item-meta' }, [item.type === 'folder' ? 'folder' : (item.extension || ext(item.name) || 'file')]));
              row.appendChild(e('span', { className: 'files-item-meta' }, [item.size ? String(item.size) : '']));
              row.appendChild(e('span', { className: 'files-item-meta' }, [item.modifiedAt || '']));
              row.appendChild(e('span', { className: 'files-row-actions' }, [rowBtn('Open', 'row-open', function (ev) { ev.stopPropagation(); open(item); }), rowBtn('Rename', 'row-rename', function (ev) { ev.stopPropagation(); startRename(item); }), rowBtn('Move to trash', 'row-trash', function (ev) { ev.stopPropagation(); trash(item); })]));
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
          function startCreate(mode) { createMode = mode; createInput.value = ''; createPanel.style.display = 'flex'; createInput.focus(); }
          function confirmCreate() {
            var name = createInput.value.trim();
            var mode = createMode;
            if (!name) return;
            if (mode === 'markdown' && !/\.(md|markdown)$/i.test(name)) name += '.md';
            if (mode === 'text' && !/\.[^/.]+$/.test(name)) name += '.txt';
            var path = scoped(current ? current + '/' + name : name);
            (mode === 'folder' ? api.files.createFolder(path) : api.files.writeText(path, '', { createIfMissing: true, overwrite: false })).then(function () { createPanel.style.display = 'none'; load(); });
          }
          function startRename(item) { if (!item) return; renaming = item; renameInput.value = item.name; renamePanel.style.display = 'flex'; renameInput.focus(); renameInput.select(); }
          function confirmRename() {
            if (!renaming) return;
            var to = parent(renaming.relativePath);
            to = to ? to + '/' + renameInput.value.trim() : renameInput.value.trim();
            api.files.move(renaming.relativePath, to, { overwrite: false }).then(function () { renamePanel.style.display = 'none'; renaming = null; load(); });
          }
          function trash(item) { if (!item || !window.confirm('Move "' + item.name + '" to trash?')) return; api.files.trash(item.relativePath).then(load); }
          function trashSelection() { var items = selectedEntries(); if (items.length === 1) return trash(items[0]); if (!items.length || !window.confirm('Move ' + items.length + ' items to trash?')) return; Promise.all(items.map(function (item) { return api.files.trash(item.relativePath); })).then(load); }
          function setClipboard(action, items) { if (!items.length) return; window.__filesClipboard = { action: action, workspaceRoot: root, items: items.map(function (item) { return { path: item.relativePath, name: item.name, type: item.type }; }) }; }
          function cutSelection() { setClipboard('cut', selectedEntries()); }
          function copySelection() { setClipboard('copy', selectedEntries().filter(function (item) { return item.type !== 'folder'; })); }
          function uniqueName(name, occupied) { if (!occupied[name]) return name; var dot = name.lastIndexOf('.'); var b = dot > 0 ? name.slice(0, dot) : name; var x = dot > 0 ? name.slice(dot) : ''; for (var i = 2; i < 100; i++) { var c = b + ' (' + i + ')' + x; if (!occupied[c]) return c; } return b + ' (' + Date.now() + ')' + x; }
          function paste() {
            var clip = window.__filesClipboard;
            if (!clip || !clip.items || !clip.items.length) return;
            var dest = scoped(current);
            var occupied = {};
            entries.forEach(function (item) { occupied[item.name] = true; });
            Promise.all(clip.items.map(function (item) {
              var name = uniqueName(item.name, occupied);
              occupied[name] = true;
              var to = dest ? dest + '/' + name : name;
              if (clip.action === 'cut') return api.files.move(item.path, to, { overwrite: false });
              return api.files.readText(item.path).then(function (text) { return api.files.writeText(to, text, { createIfMissing: true, overwrite: false }); });
            })).then(function () { if (clip.action === 'cut') window.__filesClipboard = null; load(); });
          }
          var menu = e('div', { className: 'files-ctx-menu', style: { display: 'none' } }, []);
          document.body.appendChild(menu);
          function menuItem(label, action, fn) { return e('div', { className: 'files-ctx-menu-item', 'data-files-menu-action': action, onClick: function (ev) { ev.stopPropagation(); menu.style.display = 'none'; fn(); } }, [e('span', { innerHTML: SVG }, []), label]); }
          function showMenu(x, y, item) {
            menu.innerHTML = '';
            if (item) {
              if (!selected[item.relativePath]) { selected = {}; selected[item.relativePath] = true; render(); }
              menu.appendChild(menuItem('Open', 'open', function () { open(item); }));
              menu.appendChild(menuItem('Rename', 'rename', function () { startRename(item); }));
              menu.appendChild(menuItem('Cut', 'cut', cutSelection));
              menu.appendChild(menuItem('Copy', 'copy', copySelection));
              menu.appendChild(menuItem('Trash', 'trash', trashSelection));
            } else {
              menu.appendChild(menuItem('New Folder', 'new-folder', function () { startCreate('folder'); }));
              menu.appendChild(menuItem('New Markdown', 'new-markdown', function () { startCreate('markdown'); }));
              menu.appendChild(menuItem('New Text', 'new-text', function () { startCreate('text'); }));
              if (window.__filesClipboard && window.__filesClipboard.items && window.__filesClipboard.items.length) menu.appendChild(menuItem('Paste', 'paste', paste));
            }
            menu.style.display = 'block'; menu.style.left = x + 'px'; menu.style.top = y + 'px';
          }
          createInput.addEventListener('keydown', function (ev) { if (ev.key === 'Enter') confirmCreate(); });
          renameInput.addEventListener('keydown', function (ev) { if (ev.key === 'Enter') confirmRename(); });
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
            var ctrl = ev.ctrlKey || ev.metaKey;
            if (ctrl && ev.key.toLowerCase() === 'a') { ev.preventDefault(); selected = {}; visible().forEach(function (item) { selected[item.relativePath] = true; }); render(); }
            if (ctrl && ev.key.toLowerCase() === 'x') { ev.preventDefault(); cutSelection(); }
            if (ctrl && ev.key.toLowerCase() === 'c') { ev.preventDefault(); copySelection(); }
            if (ctrl && ev.key.toLowerCase() === 'v') { ev.preventDefault(); paste(); }
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
    ReadPluginSettings: function (pluginId) {
      return Promise.resolve([Object.assign({}, pluginSettings[pluginId] || {}), '']);
    },
    WritePluginSettings: function (pluginId, settings) {
      pluginSettings[pluginId] = Object.assign({}, settings || {});
      return Promise.resolve('');
    },
    ReadPluginSetting: function () { return Promise.resolve(null); },
    WritePluginSetting: function () { return Promise.resolve(null); },
    ReadPluginDataJSON: function () { return Promise.resolve({}); },
    WritePluginDataJSON: function () { return Promise.resolve(null); },
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
    PluginSyncConfigure: function (pluginId, serverUrl) {
      var err = requirePluginSyncPermission(pluginId, true);
      if (err) return Promise.resolve(err);
      syncState.configured = true;
      syncState.serverUrl = serverUrl || '';
      syncState.deviceId = 'mock-device';
      syncState.deviceName = 'mock-device';
      syncState.connected = true;
      syncState.revoked = false;
      syncState.tokenStored = true;
      syncState.lastError = '';
      syncState.statusLabel = 'connected';
      pluginSettings[pluginId] = Object.assign({}, pluginSettings[pluginId] || {}, {
        serverUrl: syncState.serverUrl,
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
      if (pluginId === 'verstak.default-editor' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(defaultEditorBundle());
      }
      if (pluginId === 'verstak.files' && assetPath === 'frontend/dist/index.js') {
        return Promise.resolve(filesPluginBundle());
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
      return Promise.resolve([entries, '']);
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
      return Promise.resolve([node.content || '', '']);
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
    TrashVaultPath: function (pluginId, relativePath) {
      var err = requirePluginPermission(pluginId, 'files.delete');
      if (err) return Promise.resolve([{}, err]);
      var norm = normalizeVaultPath(relativePath, false);
      if (norm.error) return Promise.resolve([{}, norm.error]);
      if (!vaultFiles[norm.path]) return Promise.resolve([{}, 'not-found: ' + norm.path]);
      var trashId = 'mock-' + Date.now() + '-' + Math.random().toString(16).slice(2);
      var trashPath = '.verstak/trash/files/' + trashId + '/' + baseName(norm.path);
      var moving = Object.keys(vaultFiles).filter(function (path) { return path === norm.path || path.indexOf(norm.path + '/') === 0; });
      moving.forEach(function (path) { delete vaultFiles[path]; });
      return Promise.resolve([{ originalPath: norm.path, trashPath: trashPath, trashId: trashId, deletedAt: new Date().toISOString() }, '']);
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
    CreateWorkspace: function (name) {
      var norm = normalizeVaultPath(name, false);
      if (norm.error || norm.path !== String(name || '').trim() || norm.path.indexOf('/') !== -1) {
        return Promise.resolve(norm.error || 'invalid-workspace-name');
      }
      if (vaultFiles[norm.path]) return Promise.resolve('conflict: ' + norm.path);
      vaultFiles[norm.path] = { type: 'folder', modifiedAt: new Date().toISOString() };
      vaultFiles[norm.path + '/Notes'] = { type: 'folder', modifiedAt: new Date().toISOString() };
      vaultFiles[norm.path + '/Notes/Overview.md'] = { type: 'file', content: '# Overview\n', modifiedAt: new Date().toISOString() };
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
      if (workspaceTree.currentNodeId === norm.path) workspaceTree.currentNodeId = workspaceTree.nodes[0] ? workspaceTree.nodes[0].id : '';
      return Promise.resolve({ originalPath: norm.path, trashPath: '.verstak/trash/workspaces/mock/' + norm.path, trashId: 'mock', deletedAt: new Date().toISOString() });
    },
    GetWorkspaceMetadata: function (name) {
      var norm = normalizeVaultPath(name, false);
      if (norm.error) return Promise.resolve(norm.error);
      if (!vaultFiles[norm.path]) return Promise.resolve('not-found: ' + norm.path);
      return Promise.resolve({
        workspaceName: norm.path,
        features: { files: true },
        folders: { notes: 'Notes', files: 'Files' }
      });
    },
    UpdateWorkspaceMetadata: function (name, patch) {
      return Promise.resolve(Object.assign({ workspaceName: name, features: { files: true }, folders: { notes: 'Notes', files: 'Files' } }, patch || {}));
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
    SelectDirectory: function () { return Promise.resolve(''); },
    SelectVaultForOpen: function () { return Promise.resolve(''); },
    CreateVault: function () { return Promise.resolve(null); },
    OpenVault: function () { return Promise.resolve(null); },
    CloseVault: function () { return Promise.resolve(null); },
    SetCurrentVault: function () { return Promise.resolve(''); },
    UpdateAppSettings: function () { return Promise.resolve(''); },
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
              statusBarItems: [{ id: 'verstak.sync.status', label: 'Sync', position: 'right' }]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/sync',
          error: ''
        }
      };
      vaultStatus = { status: 'open', path: '/tmp/verstak-test/vault', vaultId: 'test-vault-001' };
      vaultPluginState = { enabledPlugins: ['verstak.platform-test', 'verstak.default-editor', 'verstak.files', 'verstak.sync'], disabledPlugins: [], desiredPlugins: [{ id: 'verstak.platform-test', version: '0.1.0', source: 'official' }, { id: 'verstak.default-editor', version: '0.1.0', source: 'official' }, { id: 'verstak.files', version: '0.1.0', source: 'official' }, { id: 'verstak.sync', version: '0.1.0', source: 'official' }] };
      appSettings = { currentVaultPath: '/tmp/verstak-test/vault', recentVaults: [] };
      workbenchPreferences = {};
      openedResources = [];
      pluginSettings = { 'verstak.platform-test': { savedText: 'initial value' } };
      vaultFiles = makeDefaultVaultFiles();
      externalOpens = [];
      window.__wailsMockExternalOpens = [];
      workspaceTree = makeDefaultWorkspaceTree();
      reloadResponseMode = 'tuple';
      syncState = makeDefaultSyncState();
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
    addSyntheticPlugins: function (count) {
      var total = Number(count || 0);
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
            source: 'test',
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
          vaultPluginState.desiredPlugins.push({ id: id, version: '0.0.' + i, source: 'test' });
        }
      }
    },
    setVaultStatus: function (status) { vaultStatus = status; },
    setVaultPluginState: function (state) { vaultPluginState = state; },
    setReloadResponseMode: function (mode) { reloadResponseMode = mode || 'tuple'; }
  };

  window.__wailsMockReady = true;
  console.log('[wails-mock] bridge installed');
})();
