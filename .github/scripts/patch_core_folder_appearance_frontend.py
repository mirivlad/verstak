from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


# Wails API: UI writes are exact replacements, including clearing icon/color.
path = Path("internal/api/app.go")
text = path.read_text()
text = replace_once(
    text,
    "if err := a.treeV2.SetFolderAppearance(folderID, fa); err != nil {",
    "if err := a.treeV2.ReplaceFolderAppearance(folderID, fa); err != nil {",
    "folder appearance Wails replacement semantics",
)
path.write_text(text)

# WorkspaceTree: core owns folder appearance, so the shell must call the core
# Wails API directly instead of impersonating a retired plugin namespace.
path = Path("frontend/src/lib/shell/WorkspaceTree.svelte")
text = path.read_text()
old = """  async function loadFolderAppearance(folderId) {
    try {
      // Check local cache first
      if (appearanceCache[folderId]) {
        folderIconId = appearanceCache[folderId].iconId || '';
        folderColor = appearanceCache[folderId].colorId || '';
        return;
      }
      const api = window.createPluginAPI('verstak.folder-appearance');
      if (api && api.folders && api.folders.getAppearance) {
        const a = await api.folders.getAppearance(folderId);
        folderIconId = a.iconId || '';
        folderColor = a.colorId || '';
        appearanceCache[folderId] = { iconId: folderIconId, colorId: folderColor };
        appearanceCache = appearanceCache;
      }
    } catch {}
  }
"""
new = """  async function loadFolderAppearance(folderId) {
    try {
      if (appearanceCache[folderId]) {
        folderIconId = appearanceCache[folderId].iconId || '';
        folderColor = appearanceCache[folderId].colorId || '';
        return;
      }
      const appearance = await App.GetFolderAppearance(folderId);
      if (appearance?.error) return;
      folderIconId = appearance?.icon || '';
      folderColor = appearance?.color || '';
      appearanceCache = {
        ...appearanceCache,
        [folderId]: { iconId: folderIconId, colorId: folderColor },
      };
    } catch {}
  }
"""
text = replace_once(text, old, new, "load folder appearance")

old = """  async function loadAppearanceMap() {
    try {
      const api = window.createPluginAPI('verstak.folder-appearance');
      if (!api || !api.folders || !api.folders.getAppearance) return;
      // Collect all folder UUIDs from tree
      const uuids = [];
      function walk(roots) {
        for (const r of roots) {
          if (r.kind === 'folder') { uuids.push(r.id); if (r.children) walk(r.children); }
        }
      }
      walk(tree.roots);
      // Load each appearance
      const next = {};
      for (const uuid of uuids) {
        try {
          const a = await api.folders.getAppearance(uuid);
          if (a && (a.iconId || a.colorId)) { next[uuid] = a; }
        } catch {}
      }
      appearanceCache = { ...appearanceCache, ...next };
      appearanceCache = appearanceCache; // trigger reactivity
    } catch {}
  }
"""
new = """  async function loadAppearanceMap() {
    try {
      const uuids = [];
      function walk(roots) {
        for (const node of roots || []) {
          if (node.kind === 'folder') {
            uuids.push(node.id);
            walk(node.children);
          }
        }
      }
      walk(tree.roots);
      const next = {};
      for (const folderId of uuids) {
        try {
          const appearance = await App.GetFolderAppearance(folderId);
          if (!appearance?.error && (appearance?.icon || appearance?.color)) {
            next[folderId] = {
              iconId: appearance.icon || '',
              colorId: appearance.color || '',
            };
          }
        } catch {}
      }
      appearanceCache = next;
    } catch {}
  }
"""
text = replace_once(text, old, new, "load folder appearance map")

create_start = text.index("  async function doCreateFolder() {") if "  async function doCreateFolder() {" in text else text.index("  async function doCreateFolder() { const")
create_end = text.index("  async function doCreateWorkspace() {", create_start)
new_create = """  async function doCreateFolder() {
    const n = formName.trim();
    if (!n) { formError = tr('workspaceTree.nameRequired'); return; }
    formBusy = true;
    const r = await App.CreateFolderV2(formParentId || '', n);
    if (r?.error) { formError = r.error; formBusy = false; return; }
    if (formParentId) {
      expandedIds['folder:' + formParentId] = true;
      saveExpanded();
    }
    const fid = r?.id;
    if (fid) {
      const appearanceError = await App.SetFolderAppearance(fid, {
        icon: folderIconId,
        color: folderColor,
      });
      if (appearanceError) { formError = appearanceError; formBusy = false; return; }
      appearanceCache = {
        ...appearanceCache,
        [fid]: { iconId: folderIconId, colorId: folderColor },
      };
    }
    modal = null;
    await loadTree();
  }
"""
text = text[:create_start] + new_create + text[create_end:]

edit_start = text.index("  async function doEditFolder() {")
edit_end = text.index("  async function doMove()", edit_start)
new_edit = """  async function doEditFolder() {
    const n = formName.trim();
    if (!n) { formError = tr('workspaceTree.nameRequired'); return; }
    formBusy = true;
    const origName = modal.origName || '';
    if (n !== origName) {
      const err = await App.RenameFolderV2(modal.id, n);
      if (err) { formError = err; formBusy = false; return; }
    }
    const appearanceError = await App.SetFolderAppearance(modal.id, {
      icon: folderIconId,
      color: folderColor,
    });
    if (appearanceError) { formError = appearanceError; formBusy = false; return; }
    appearanceCache = {
      ...appearanceCache,
      [modal.id]: { iconId: folderIconId, colorId: folderColor },
    };
    modal = null;
    await loadTree();
  }
"""
text = text[:edit_start] + new_edit + text[edit_end:]

if "createPluginAPI('verstak.folder-appearance')" in text:
    raise SystemExit("WorkspaceTree still impersonates folder-appearance plugin")
path.write_text(text)

# Wails browser mock: emulate the already-existing core appearance API and keep
# that state separate from plugin-scoped storage.
path = Path("frontend/src/lib/test/wails-mock.js")
text = path.read_text()
text = replace_once(
    text,
    "  var pluginData = {};\n",
    "  var pluginData = {};\n  var folderAppearances = {};\n",
    "folder appearance mock state",
)
old = """    GetWorkspaceTreeV2: function () {
      return Promise.resolve(workspaceTreeV2Snapshot());
    },
"""
new = old + """    GetFolderAppearance: function (folderId) {
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
"""
text = replace_once(text, old, new, "core folder appearance mock API")
text = replace_once(
    text,
    "      pluginData = {};\n",
    "      pluginData = {};\n      folderAppearances = {};\n",
    "reset folder appearance state",
)
path.write_text(text)

# Source contract: the built-in tree editor owns this feature and may not route
# it back through a plugin namespace. Also lock the core migration boundary.
path = Path("frontend/tests/shell-source-contract-test.mjs")
text = path.read_text()
text = replace_once(
    text,
    "const globalSearch = read('frontend/src/lib/shell/GlobalSearch.svelte');\n",
    "const globalSearch = read('frontend/src/lib/shell/GlobalSearch.svelte');\nconst workspaceTree = read('frontend/src/lib/shell/WorkspaceTree.svelte');\nconst folderAppearanceCore = read('internal/core/workspacetree/appearance.go');\n",
    "folder appearance source inputs",
)
anchor = """assertIncludes(
  app,
  '<GlobalSearch />',
  'App should expose global search in the main content header'
);
"""
addition = anchor + """assertIncludes(
  workspaceTree,
  'App.GetFolderAppearance(',
  'WorkspaceTree should read core-owned folder appearance through Wails',
);
assertIncludes(
  workspaceTree,
  'App.SetFolderAppearance(',
  'WorkspaceTree should save core-owned folder appearance through Wails',
);
assertExcludes(
  workspaceTree,
  "createPluginAPI('verstak.folder-appearance')",
  'WorkspaceTree must not impersonate the retired folder-appearance plugin',
);
assertIncludes(
  folderAppearanceCore,
  'legacyFolderAppearancePluginID = "verstak.folder-appearance"',
  'core folder appearance should preserve legacy plugin data migration',
);
for (const staleContributionHelper of ['GetFolderTreeNodeActions', 'GetWorkspaceTreeNodeActions']) {
  assertExcludes(
    folderAppearanceCore,
    staleContributionHelper,
    `core appearance should not retain abandoned contribution helper ${staleContributionHelper}`,
  );
}
"""
text = replace_once(text, anchor, addition, "folder appearance source contract")
path.write_text(text)
