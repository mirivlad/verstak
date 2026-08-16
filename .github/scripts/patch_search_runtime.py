from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


# Backend: readable vault path -> deepest owning Deal. This is intentionally
# NOT filtered by the calling plugin's workspace items.
path = Path("internal/api/app.go")
text = path.read_text()
marker = "// PluginListWorkspaces returns semantic Deal nodes where the calling plugin is active.\n"
insert = r'''func resolveOwningWorkspace(nodes []workspacetree.TreeNode, relativePath string) (workspacetree.TreeNode, bool) {
	var best workspacetree.TreeNode
	bestLength := -1
	var walk func([]workspacetree.TreeNode)
	walk = func(items []workspacetree.TreeNode) {
		for _, node := range items {
			if node.Kind == "workspace" {
				rootPath := strings.Trim(node.Path, "/")
				if rootPath != "" && (relativePath == rootPath || strings.HasPrefix(relativePath, rootPath+"/")) && len(rootPath) > bestLength {
					best = node
					bestLength = len(rootPath)
				}
			}
			if len(node.Children) > 0 {
				walk(node.Children)
			}
		}
	}
	walk(nodes)
	return best, bestLength >= 0
}

// PluginResolveWorkspacePath resolves a readable vault-relative path to the
// deepest owning Deal. Unlike PluginListWorkspaces, ownership does not require
// the calling plugin to contribute a workspace item in that Deal.
func (a *App) PluginResolveWorkspacePath(pluginID, relativePath string) (map[string]interface{}, string) {
	if err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return nil, err.Error()
	}
	cleanPath, err := corefiles.NormalizeRelativeDir(relativePath)
	if err != nil {
		return nil, err.Error()
	}
	if a.treeV2 == nil {
		return nil, "workspace tree not initialized"
	}

	snapshot := a.treeV2.GetTree()
	workspace, found := resolveOwningWorkspace(snapshot.Roots, cleanPath)
	if !found {
		return map[string]interface{}{"found": false}, ""
	}
	workspaceRootPath := strings.Trim(workspace.Path, "/")
	localPath := strings.TrimPrefix(cleanPath, workspaceRootPath)
	localPath = strings.TrimPrefix(localPath, "/")
	return map[string]interface{}{
		"found":             true,
		"workspaceId":       workspace.ID,
		"workspaceName":     workspace.Name,
		"workspaceRootPath": workspaceRootPath,
		"relativePath":      localPath,
	}, ""
}

'''
text = replace_once(text, marker, insert + marker, "PluginListWorkspaces")
path.write_text(text)

# Pure resolver test: nested roots must choose the most specific owning Deal.
path = Path("internal/api/app_test.go")
text = path.read_text()
marker = "func TestPluginListWorkspacesReturnsNestedDealsOnly(t *testing.T) {"
test = r'''func TestResolveOwningWorkspacePicksDeepestDeal(t *testing.T) {
	nodes := []workspacetree.TreeNode{
		{
			Kind: "folder", ID: "clients", Name: "Clients", Path: "Clients",
			Children: []workspacetree.TreeNode{
				{Kind: "workspace", ID: "acme", Name: "Acme", Path: "Clients/Acme"},
				{Kind: "workspace", ID: "nested", Name: "Nested", Path: "Clients/Acme/Nested"},
			},
		},
	}

	got, ok := resolveOwningWorkspace(nodes, "Clients/Acme/Nested/docs/readme.md")
	if !ok || got.ID != "nested" || got.Path != "Clients/Acme/Nested" {
		t.Fatalf("resolveOwningWorkspace nested = %+v, %v", got, ok)
	}
	got, ok = resolveOwningWorkspace(nodes, "Clients/Acme/docs/readme.md")
	if !ok || got.ID != "acme" || got.Path != "Clients/Acme" {
		t.Fatalf("resolveOwningWorkspace acme = %+v, %v", got, ok)
	}
	if got, ok = resolveOwningWorkspace(nodes, "Loose/file.txt"); ok {
		t.Fatalf("resolveOwningWorkspace loose = %+v, want not found", got)
	}
}

'''
text = replace_once(text, marker, test + marker, "PluginListWorkspaces test")
path.write_text(text)

# Checked-in Wails bridge.
path = Path("frontend/wailsjs/go/api/App.js")
text = path.read_text()
old = """export function PluginListWorkspaces(arg1) {
  return window['go']['api']['App']['PluginListWorkspaces'](arg1);
}
"""
new = old + """
export function PluginResolveWorkspacePath(arg1, arg2) {
  return window['go']['api']['App']['PluginResolveWorkspacePath'](arg1, arg2);
}
"""
path.write_text(replace_once(text, old, new, "App.js PluginListWorkspaces"))

path = Path("frontend/wailsjs/go/api/App.d.ts")
text = path.read_text()
old = "export function PluginListWorkspaces(arg1:string):Promise<Array<api.PluginWorkspaceDTO>|string>;\n"
new = old + "export function PluginResolveWorkspacePath(arg1:string,arg2:string):Promise<Record<string, any>|string>;\n"
path.write_text(replace_once(text, old, new, "App.d.ts PluginListWorkspaces"))

# Public plugin runtime implementation.
path = Path("frontend/src/lib/plugin-host/VerstakPluginAPI.js")
text = path.read_text()
old = """    workspaces: {
      list: function() {
        assertActive('workspaces.list');
        return callBackend(pluginId, 'workspaces.list', function() {
          return App.PluginListWorkspaces(pluginId);
        });
      }
    },
"""
new = """    workspaces: {
      list: function() {
        assertActive('workspaces.list');
        return callBackend(pluginId, 'workspaces.list', function() {
          return App.PluginListWorkspaces(pluginId);
        });
      },
      resolvePath: function(relativePath) {
        assertActive('workspaces.resolvePath');
        return callBackend(pluginId, 'workspaces.resolvePath', function() {
          return App.PluginResolveWorkspacePath(pluginId, String(relativePath || ''));
        });
      }
    },
"""
path.write_text(replace_once(text, old, new, "VerstakPluginAPI workspaces"))

# Test bridge: model the backend resolver from the semantic tree so the real
# Search bundle takes the same folder-action path in E2E.
path = Path("frontend/src/lib/test/search-provider-mock.js")
text = path.read_text()
marker = "const originalExecutePluginCommand = app.ExecutePluginCommand.bind(app);\n"
addition = r'''
function cleanVaultRelativePath(value) {
  const raw = String(value || '');
  if (!raw || raw.includes('\\') || raw.startsWith('/') || raw.split('/').some((part) => part === '..')) return null;
  const parts = raw.split('/').filter(Boolean);
  if (parts[0] === '.verstak') return null;
  return parts.join('/');
}

app.PluginResolveWorkspacePath = async function pluginResolveWorkspacePath(pluginId, relativePath) {
  const path = cleanVaultRelativePath(relativePath);
  if (!path) return [null, 'invalid relative path'];
  const snapshot = await app.GetWorkspaceTreeV2();
  let best = null;
  function walk(nodes) {
    (nodes || []).forEach((node) => {
      const root = String(node?.path || '').replace(/^\/+|\/+$/g, '');
      if (node?.kind === 'workspace' && root && (path === root || path.startsWith(`${root}/`))) {
        if (!best || root.length > best.root.length) best = { node, root };
      }
      walk(node?.children || []);
    });
  }
  walk(snapshot?.roots || []);
  if (!best) return [{ found: false }, ''];
  return [{
    found: true,
    workspaceId: best.node.id || '',
    workspaceName: best.node.name || '',
    workspaceRootPath: best.root,
    relativePath: path === best.root ? '' : path.slice(best.root.length + 1),
  }, ''];
};
'''
text = replace_once(text, marker, marker + addition, "search-provider mock")
path.write_text(text)

# GlobalSearch: provider-owned labels, enabled providers only, and a failed
# provider marks the aggregate result as partial.
path = Path("frontend/src/lib/shell/GlobalSearch.svelte")
text = path.read_text()
old = """    (contributions.sidebarItems || []).forEach(item => {
      next.push({
"""
new = """    const enabledPluginIds = new Set((rawPlugins || [])
      .filter(plugin => plugin?.enabled && (plugin?.status === 'loaded' || plugin?.status === 'degraded'))
      .map(plugin => plugin?.manifest?.id)
      .filter(Boolean));
    (contributions.sidebarItems || []).filter(item => enabledPluginIds.has(item?.pluginId)).forEach(item => {
      next.push({
"""
text = replace_once(text, old, new, "GlobalSearch sidebar")
old = "searchProviders = (contributions.searchProviders || []).filter(provider => provider?.pluginId && provider?.handler);"
new = "searchProviders = (contributions.searchProviders || []).filter(provider => enabledPluginIds.has(provider?.pluginId) && provider?.handler);"
text = replace_once(text, old, new, "GlobalSearch provider filter")
old = """  function providerType(provider, item) {
    const category = String(item?.categoryId || '').toLowerCase();
    if (category === 'files' || category === 'file') return { type: 'File', label: tr('search.type.file') };
    if (category === 'folders' || category === 'folder') return { type: 'Folder', label: tr('search.type.folder') };
    const label = String(item?.categoryLabel || provider?.label || item?.categoryId || tr('search.type.tool'));
    return { type: label, label };
  }
"""
new = """  function providerType(provider, item) {
    const categoryId = String(item?.categoryId || provider?.id || 'provider');
    const label = String(item?.categoryLabel || provider?.label || categoryId || tr('search.type.tool'));
    return { type: label, categoryId, label };
  }
"""
text = replace_once(text, old, new, "GlobalSearch providerType")
old = """      type: type.type,
      typeLabel: type.label,
"""
new = """      type: type.type,
      categoryId: type.categoryId,
      typeLabel: type.label,
"""
text = replace_once(text, old, new, "GlobalSearch normalized type")
text = replace_once(text, "return { rows: [], partial: false };", "return { rows: [], partial: true };", "GlobalSearch provider partial")
old = """            data-global-search-result-type={item.type}
            data-global-search-result-path={item.path || ''}
"""
new = """            data-global-search-result-type={item.type}
            data-global-search-result-category={item.categoryId || ''}
            data-global-search-result-path={item.path || ''}
"""
text = replace_once(text, old, new, "GlobalSearch DOM category")
path.write_text(text)

# Source guard: shell cannot re-learn provider category semantics or the old
# Files-history implementation detail.
path = Path("frontend/tests/shell-source-contract-test.mjs")
text = path.read_text()
old = """  'verstak.journal',
]) {
"""
new = """  'verstak.journal',
  "category === 'files'",
  "category === 'folders'",
  '__filesHistoryByWorkspace',
]) {
"""
idx = text.rfind(old)
if idx < 0:
    raise SystemExit("GlobalSearch forbidden array tail not found")
text = text[:idx] + text[idx:].replace(old, new, 1)
marker = """assertIncludes(
  globalSearch,
  'contributions.searchProviders || []',
"""
addition = """assertIncludes(
  globalSearch,
  'item?.categoryLabel || provider?.label',
  'GlobalSearch should render provider-owned category labels without interpreting provider category ids',
);
assertIncludes(
  globalSearch,
  'enabledPluginIds.has(provider?.pluginId)',
  'GlobalSearch should query only providers from enabled loaded/degraded plugins',
);

"""
text = replace_once(text, marker, addition + marker, "GlobalSearch source assertion")
path.write_text(text)

# E2E uses stable machine category ids where provider results are asserted.
path = Path("frontend/e2e/ux-followup.spec.js")
text = path.read_text()
old = "const folderResult = page.locator('[data-global-search-result-type=\"Folder\"][data-global-search-result-path=\"Project/Notes\"]');"
new = "const folderResult = page.locator('[data-global-search-result-category=\"folders\"][data-global-search-result-path=\"Project/Notes\"]');"
path.write_text(replace_once(text, old, new, "folder E2E selector"))

path = Path("frontend/e2e/global-search-results.spec.js")
text = path.read_text()
text = text.replace('[data-global-search-result-type="Browser"]', '[data-global-search-result-category="browser"]')
text = text.replace('[data-global-search-result-type="File"]', '[data-global-search-result-category="files"]')
path.write_text(text)
