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
        permissions: ['vault.read', 'events.publish', 'events.subscribe', 'ui.register', 'commands.register', 'storage.namespace', 'files.read', 'files.write', 'files.delete', 'workbench.open'],
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
        permissions: ['files.read', 'files.write', 'workbench.open', 'ui.register'],
        frontend: { entry: 'frontend/dist/index.js' },
    contributes: {
      views: [{ id: 'verstak.files.view', title: 'Files', icon: 'folder', component: 'FilesView' }],
      workspaceItems: [{ id: 'verstak.files.workspace', title: 'Files', icon: 'folder', component: 'FilesView' }]
    }
      },
      rootPath: '/tmp/verstak-test/plugins/files',
      error: ''
    }
  };

  var vaultStatus = { status: 'open', path: '/tmp/verstak-test/vault', vaultId: 'test-vault-001' };
  var vaultPluginState = { enabledPlugins: ['verstak.platform-test', 'verstak.default-editor', 'verstak.files'], disabledPlugins: [], desiredPlugins: [{ id: 'verstak.platform-test', version: '0.1.0', source: 'official' }, { id: 'verstak.default-editor', version: '0.1.0', source: 'official' }, { id: 'verstak.files', version: '0.1.0', source: 'official' }] };
  var appSettings = { currentVaultPath: '/tmp/verstak-test/vault', recentVaults: [] };
  var workbenchPreferences = {};
  var openedResources = [];
  var pluginSettings = {
    'verstak.platform-test': { savedText: 'initial value' }
  };
  var vaultFiles = makeDefaultVaultFiles();
  var workspaceTree = makeDefaultWorkspaceTree();
  var reloadResponseMode = 'tuple';

  // ── Helpers ────────────────────────────────────────────────────────
  function makeDefaultWorkspaceTree() {
    return {
      status: 'initialized',
      currentNodeId: 'case-alpha',
      nodes: [
        { id: 'space-main', parentId: '', type: 'space', title: 'Main Space', path: 'Main Space', status: 'active', order: 1 },
        { id: 'case-alpha', parentId: 'space-main', type: 'case', title: 'Alpha Case', path: 'Main Space/Alpha Case', status: 'active', order: 1 },
        { id: 'case-beta', parentId: 'space-main', type: 'case', title: 'Beta Case', path: 'Main Space/Beta Case', status: 'active', order: 2 }
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

  function makeDefaultVaultFiles() {
    return {
      '': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Docs': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Docs/todo.txt': { type: 'file', content: 'Buy groceries\nWrite tests', modifiedAt: new Date().toISOString() },
      'Docs/readme.md': { type: 'file', content: '# Hello World\n\nThis is a **test** document.\n\n- item 1\n- item 2', modifiedAt: new Date().toISOString() },
      'Notes': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Notes/Overview.md': { type: 'file', content: '# Notes Overview\n\nMy notes content here.', modifiedAt: new Date().toISOString() },
      'Main Space': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Main Space/Alpha Case': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Main Space/Alpha Case/alpha-only.txt': { type: 'file', content: 'alpha file', modifiedAt: new Date().toISOString() },
      'Main Space/Beta Case': { type: 'folder', modifiedAt: new Date().toISOString() },
      'Main Space/Beta Case/beta-only.txt': { type: 'file', content: 'beta file', modifiedAt: new Date().toISOString() }
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
      { name: 'workbench.open', description: 'Request Workbench open/edit routing', dangerous: false }
    ];
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
    return (provider.supports || []).some(function (support) {
      if (support.kind && support.kind !== request.kind) return false;
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
    return [
      '(function(){',
      'var DefaultEditor={',
      'mount:function(c,p,api){',
      'if(!document.getElementById("mock-default-editor-styles")){',
      'var style=document.createElement("style");',
      'style.id="mock-default-editor-styles";',
      'style.textContent=".de-root{display:flex;flex-direction:column;height:100%;min-height:0;overflow:hidden}.de-toolbar{display:flex;align-items:center;gap:.5rem;padding:.5rem .75rem;border-bottom:1px solid #16213e;flex-shrink:0;background:#12122a}.de-toolbar-mode{font-size:.75rem;color:#4ecca3;padding:.15rem .5rem;border-radius:3px;background:#1a2a3a}.de-toolbar-context{font-size:.7rem;color:#8b8ba8}.de-editor-wrap{flex:1;display:flex;min-height:0;overflow:hidden}.de-textarea{flex:1;width:100%;height:100%;resize:none;border:0;outline:0;padding:.75rem;font-family:monospace;font-size:.85rem;line-height:1.6;background:#0d0d1a;color:#e0e0e0}.de-preview{flex:1;height:100%;padding:.75rem 1rem;overflow-y:auto;background:#0d0d1a;line-height:1.7;font-size:.9rem}.de-notes-badge{font-size:.65rem;padding:.1rem .4rem;border-radius:3px;background:#2a1a3a;color:#b388ff}";',
      'document.head.appendChild(style);',
      '}',
      'c.innerHTML="";',
      'c.className="de-root";',
      'var req=p.request||{};',
      'var path=req.path||"";',
      'var mode=req.mode||"view";',
      'var ctx=req.context||{};',
      'var isNotes=ctx.notesMode||ctx.isInsideNotesFolder;',
      'var ext=(req.extension||"").toLowerCase();',
      'var isMd=ext===".md"||ext===".markdown";',
      'var editorMode=isNotes?"notes-markdown":isMd?"generic-markdown":"text";',
      'c.setAttribute("data-editor-mode",editorMode);',
      'c.setAttribute("data-resource-path",path);',
      'c.setAttribute("data-request-mode",mode);',
      'var toolbar=document.createElement("div");',
      'toolbar.className="de-toolbar";',
      'var modeLabel=document.createElement("span");',
      'modeLabel.className="de-toolbar-mode";',
      'modeLabel.textContent=editorMode;',
      'toolbar.appendChild(modeLabel);',
      'var pathLabel=document.createElement("span");',
      'pathLabel.className="de-toolbar-context";',
      'pathLabel.textContent=path;',
      'toolbar.appendChild(pathLabel);',
      'if(isNotes){var badge=document.createElement("span");badge.className="de-notes-badge";badge.textContent="notes context";badge.setAttribute("data-notes-badge","");toolbar.appendChild(badge);}',
      'c.appendChild(toolbar);',
      'var content=document.createElement("div");',
      'content.className="de-editor-wrap";',
      'content.textContent="Loading...";',
      'c.appendChild(content);',
      'api.files.readText(path).then(function(text){',
      'content.textContent="";',
      'if(isMd){',
      'var preview=document.createElement("div");',
      'preview.className="de-preview";',
      'preview.setAttribute("data-preview","");',
      'preview.textContent=text;',
      'content.appendChild(preview);',
      '}else{',
      'var ta=document.createElement("textarea");',
      'ta.className="de-textarea";',
      'ta.value=text;',
      'ta.setAttribute("data-editor-textarea","");',
      'content.appendChild(ta);',
      '}',
      '}).catch(function(err){',
      'content.textContent="Error: "+(err.message||err);',
      '});',
      '},',
      'unmount:function(c){c.innerHTML="";}',
      '};',
      'window.VerstakPluginRegister("verstak.default-editor",{components:{DefaultEditor:DefaultEditor}});',
      '})();'
    ].join('\n');
  }

  function filesPluginBundle() {
    return [
      "(function(){",
      "var FilesView={",
      "mount:function(c,p,api){",
      "c.innerHTML='';",
      "c.className='files-root';",
      "c.setAttribute('data-plugin-id','verstak.files');",
      "var root=String((p&&(p.workspaceRootPath||(p.workspaceNode&&p.workspaceNode.path)))||'').split('/').filter(Boolean).join('/');",
      "var list=document.createElement('div');",
      "list.className='files-list';",
      "list.setAttribute('data-files-list','');",
      "c.appendChild(list);",
      "function load(){",
      "list.textContent='Loading...';",
      "api.files.list(root).then(function(entries){",
      "list.innerHTML='';",
      "if(!entries||!entries.length){list.textContent='Empty folder';return;}",
      "entries.forEach(function(e){",
      "if(e.isHidden||e.isReserved)return;",
      "var item=document.createElement('div');",
      "item.className='files-item';",
      "item.setAttribute('data-file-name',e.name);",
      "item.setAttribute('data-file-type',e.type);",
      "item.setAttribute('data-file-path',e.relativePath);",
      "var icon=document.createElement('span');",
      "icon.className='files-item-icon';",
      "icon.textContent=e.type==='folder'?'[D]':'[F]';",
      "var name=document.createElement('span');",
      "name.className='files-item-name';",
      "name.textContent=e.name;",
      "item.appendChild(icon);",
      "item.appendChild(name);",
      "if(e.type!=='folder'){",
      "item.addEventListener('dblclick',function(){",
      "var ext=e.extension?'.'+e.extension:'';",
      "var ctx={sourcePluginId:'verstak.files',sourceView:'files'};",
      "api.workbench.openResource({kind:'vault-file',path:e.relativePath,mode:'view',extension:ext,context:ctx});",
      "});",
      "}",
      "list.appendChild(item);",
      "});",
      "}).catch(function(err){list.textContent='Error: '+(err.message||err);});",
      "}",
      "load();",
      "},",
      "unmount:function(c){c.innerHTML='';}",
      "};",
      "window.VerstakPluginRegister('verstak.files',{components:{FilesView:FilesView}});",
      "})();"
    ].join('\n');
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
    GetCurrentWorkspaceNode: function () { return Promise.resolve(null); },
    GetWorkspaceTree: function () { return Promise.resolve(cloneWorkspaceTree()); },
    ArchiveWorkspaceNode: function () { return Promise.resolve(''); },
    CreateWorkspaceNode: function () { return Promise.resolve({}); },
    MoveWorkspaceNode: function () { return Promise.resolve(''); },
    RenameWorkspaceNode: function () { return Promise.resolve(''); },
    SetCurrentWorkspaceNode: function (id) {
      var found = workspaceTree.nodes.some(function (n) { return n.id === id; });
      if (!found) return Promise.resolve('workspace node not found: ' + id);
      workspaceTree.currentNodeId = id;
      return Promise.resolve('');
    },
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
            permissions: ['vault.read', 'events.publish', 'events.subscribe', 'ui.register', 'commands.register', 'storage.namespace', 'files.read', 'files.write', 'files.delete', 'workbench.open'],
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
            permissions: ['files.read', 'files.write', 'workbench.open', 'ui.register'],
            frontend: { entry: 'frontend/dist/index.js' },
            contributes: {
              views: [{ id: 'verstak.files.view', title: 'Files', icon: 'folder', component: 'FilesView' }],
              workspaceItems: [{ id: 'verstak.files.workspace', title: 'Files', icon: 'folder', component: 'FilesView' }]
            }
          },
          rootPath: '/tmp/verstak-test/plugins/files',
          error: ''
        }
      };
      vaultStatus = { status: 'open', path: '/tmp/verstak-test/vault', vaultId: 'test-vault-001' };
      vaultPluginState = { enabledPlugins: ['verstak.platform-test', 'verstak.default-editor', 'verstak.files'], disabledPlugins: [], desiredPlugins: [{ id: 'verstak.platform-test', version: '0.1.0', source: 'official' }, { id: 'verstak.default-editor', version: '0.1.0', source: 'official' }, { id: 'verstak.files', version: '0.1.0', source: 'official' }] };
      appSettings = { currentVaultPath: '/tmp/verstak-test/vault', recentVaults: [] };
      workbenchPreferences = {};
      openedResources = [];
      pluginSettings = { 'verstak.platform-test': { savedText: 'initial value' } };
      vaultFiles = makeDefaultVaultFiles();
      workspaceTree = makeDefaultWorkspaceTree();
      reloadResponseMode = 'tuple';
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
