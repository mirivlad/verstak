package api

import (
	"fmt"
	"path/filepath"
	"strings"

	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

// GetWorkspaceTreeV2 returns the UUID-based tree for the sidebar.
func (a *App) GetWorkspaceTreeV2() map[string]interface{} {
	if a.treeV2 == nil {
		return map[string]interface{}{"status": "not initialized"}
	}
	tree := a.treeV2.GetTree()
	return map[string]interface{}{
		"roots":              tree.Roots,
		"currentWorkspaceId": tree.CurrentWorkspaceID,
		"revision":           tree.Revision,
		"warnings":           tree.Warnings,
	}
}

// PluginWorkspaceDTO is the read-only Deal identity exposed to plugins.
type PluginWorkspaceDTO struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RootPath string `json:"rootPath"`
}

func resolveOwningWorkspace(nodes []workspacetree.TreeNode, relativePath string) (workspacetree.TreeNode, bool) {
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
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
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

// PluginListWorkspaces returns semantic Deal nodes where the calling plugin is active.
func (a *App) PluginListWorkspaces(pluginID string) ([]PluginWorkspaceDTO, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return nil, err.Error()
	}
	if a.treeV2 == nil {
		return nil, "workspace tree not initialized"
	}
	rows := make([]PluginWorkspaceDTO, 0)
	var collect func([]workspacetree.TreeNode)
	collect = func(nodes []workspacetree.TreeNode) {
		for _, node := range nodes {
			if node.Kind == "workspace" && a.workspaceHasTool(node.ID, pluginID) {
				rows = append(rows, PluginWorkspaceDTO{ID: node.ID, Name: node.Name, RootPath: node.Path})
			}
			collect(node.Children)
		}
	}
	collect(a.treeV2.GetTree().Roots)
	return rows, ""
}

// PluginCreateWorkspace creates a Deal from a complete plugin-owned recipe.
// The recipe's template identity is retained only as provenance in canonical
// Deal metadata; Core neither resolves nor interprets template definitions.
func (a *App) PluginCreateWorkspace(pluginID, parentFolderID, name string, recipe workspacetree.DealRecipeSnapshot) (map[string]interface{}, string) {
	if _, err := a.requirePluginAccess(pluginID, "workspaces.create"); err != nil {
		return nil, err.Error()
	}
	if a.treeV2 == nil {
		return nil, "workspace tree not initialized"
	}
	if err := a.validateWorkspaceTools(recipe.WorkspaceTools); err != nil {
		return nil, err.Error()
	}
	ws, err := a.createDealFromRecipe(parentFolderID, name, recipe)
	if err != nil {
		return nil, err.Error()
	}
	return map[string]interface{}{"workspaceId": ws.ID, "name": ws.Name}, ""
}

func (a *App) createDealFromRecipe(parentFolderID, name string, recipe workspacetree.DealRecipeSnapshot) (workspacetree.ScannedWorkspace, error) {
	if a.treeV2 == nil {
		return workspacetree.ScannedWorkspace{}, fmt.Errorf("workspace tree not initialized")
	}
	ws, err := a.treeV2.CreateWorkspaceFromRecipe(parentFolderID, name, recipe, a.refreshWorkspaceBaseline)
	if err != nil {
		return workspacetree.ScannedWorkspace{}, err
	}
	if err := a.recordWorkspaceSyncOp(syncsvc.OpCreate, ws.ID, ws.RootPath, "", ws.Name); err != nil {
		return workspacetree.ScannedWorkspace{}, err
	}
	a.publishWorkspaceLifecycleEvent(workspaceCreatedEventName, map[string]interface{}{
		"operation": "create", "workspaceId": ws.ID, "workspaceRootPath": ws.RootPath, "workspaceName": ws.Name,
		"templateId": recipe.Provenance.TemplateID,
	})
	return ws, nil
}

func (a *App) validateWorkspaceTools(workspaceTools []string) error {
	eligible := make(map[string]bool)
	for _, loaded := range a.plugins {
		if loaded.Manifest.Contributes != nil && len(loaded.Manifest.Contributes.WorkspaceItems) > 0 {
			eligible[loaded.Manifest.ID] = true
		}
	}
	for _, toolID := range workspaceTools {
		if !eligible[toolID] {
			return fmt.Errorf("workspace tool is not available: %s", toolID)
		}
	}
	return nil
}

// currentWorkspaceTools drops a retired service plugin from a Deal's persisted
// tool list. Existing Deals can carry those IDs from an older release, but a
// service plugin without a workspaceItems contribution is no longer a Deal
// tool. Unknown IDs remain errors so corrupt or mistyped tool selections are
// never silently discarded.
func (a *App) currentWorkspaceTools(workspaceTools []string) ([]string, error) {
	eligible := make(map[string]bool)
	installed := make(map[string]bool)
	for _, loaded := range a.plugins {
		installed[loaded.Manifest.ID] = true
		if loaded.Manifest.Contributes != nil && len(loaded.Manifest.Contributes.WorkspaceItems) > 0 {
			eligible[loaded.Manifest.ID] = true
		}
	}
	current := make([]string, 0, len(workspaceTools))
	for _, toolID := range workspaceTools {
		if eligible[toolID] {
			current = append(current, toolID)
			continue
		}
		if !installed[toolID] {
			return nil, fmt.Errorf("workspace tool is not available: %s", toolID)
		}
	}
	return current, nil
}

// workspaceHasTool answers the same question the workspace host answers when it
// decides which tabs a Deal gets, and has to answer it the same way. A Deal
// made by hand or carried in by an import has no tool list, and the host reads
// that as "nothing is restricted" -- it shows every tool. Reading it here as
// "no tools at all" made a Deal show the Journal tab and be invisible to the
// Journal, which is most of a migrated vault.
func (a *App) workspaceHasTool(workspaceID, pluginID string) bool {
	if a.treeV2 == nil || workspaceID == "" || pluginID == "" {
		return false
	}
	workspace, ok := a.treeV2.GetWorkspaceByID(workspaceID)
	if !ok {
		return false
	}
	metadata, err := a.treeV2.ReadDealMetadata(workspaceID, workspace.RootPath)
	if err != nil {
		return true
	}
	for _, toolID := range metadata.WorkspaceTools {
		if toolID == pluginID {
			return true
		}
	}
	return false
}

// GetWorkspaceByID returns a single workspace by its durable UUID.
func (a *App) GetWorkspaceByID(id string) map[string]interface{} {
	if a.treeV2 == nil {
		return map[string]interface{}{"error": "not initialized"}
	}
	ws, ok := a.treeV2.GetWorkspaceByID(id)
	if !ok {
		return map[string]interface{}{"error": "not found"}
	}
	return map[string]interface{}{
		"id":       ws.ID,
		"name":     ws.Name,
		"rootPath": ws.RootPath,
	}
}

// GetFolderByID returns a single folder by its durable UUID.
func (a *App) GetFolderByID(id string) map[string]interface{} {
	if a.treeV2 == nil {
		return map[string]interface{}{"error": "not initialized"}
	}
	f, ok := a.treeV2.GetFolderByID(id)
	if !ok {
		return map[string]interface{}{"error": "not found"}
	}
	return map[string]interface{}{
		"id":       f.ID,
		"name":     f.Name,
		"path":     f.Path,
		"parentId": f.ParentID,
	}
}

// RescanWorkspaceTree triggers an immediate full tree rebuild.
func (a *App) RescanWorkspaceTree() string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	if err := a.treeV2.RescanWorkspaceTree(); err != nil {
		return err.Error()
	}
	return ""
}

// GetWorkspaceTreeDiagnostics returns current tree warnings.
func (a *App) GetWorkspaceTreeDiagnostics() []workspacetree.TreeDiagnostic {
	if a.treeV2 == nil {
		return nil
	}
	return a.treeV2.GetWorkspaceTreeDiagnostics()
}

func (a *App) CreateFolderV2(parentFolderID, name string) map[string]interface{} {
	if a.treeV2 == nil {
		return map[string]interface{}{"error": "not initialized"}
	}
	f, err := a.treeV2.CreateFolder(parentFolderID, name, func() error {
		if a.fileWatcher != nil {
			return a.fileWatcher.RefreshBaseline()
		}
		return nil
	})
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"id":       f.ID,
		"name":     f.Name,
		"path":     f.Path,
		"parentId": f.ParentID,
	}
}

// UpdateWorkspaceV2Tools replaces the exact tool set for an existing Deal.
func (a *App) UpdateWorkspaceV2Tools(workspaceID string, workspaceTools []string) string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	currentTools, err := a.currentWorkspaceTools(workspaceTools)
	if err != nil {
		return err.Error()
	}
	if err := a.treeV2.UpdateWorkspaceTools(workspaceID, currentTools, func() error {
		if a.fileWatcher != nil {
			return a.fileWatcher.RefreshBaseline()
		}
		return nil
	}); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) RenameFolderV2(folderID, newName string) string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	_, err := a.treeV2.RenameFolder(folderID, newName, func() error {
		if a.fileWatcher != nil {
			return a.fileWatcher.RefreshBaseline()
		}
		return nil
	})
	if err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) RenameWorkspaceV2(workspaceID, newName string) string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	previous, ok := a.treeV2.GetWorkspaceByID(workspaceID)
	if !ok {
		return "workspace not found"
	}
	updated, err := a.treeV2.RenameWorkspace(workspaceID, newName, func() error {
		if a.fileWatcher != nil {
			return a.fileWatcher.RefreshBaseline()
		}
		return nil
	})
	if err != nil {
		return err.Error()
	}
	if err := a.recordWorkspaceSyncOp(syncsvc.OpRename, updated.ID, updated.RootPath, previous.RootPath, updated.Name); err != nil {
		return err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceRenamedEventName, map[string]interface{}{
		"operation": "rename", "workspaceId": updated.ID, "workspaceRootPath": updated.RootPath, "workspaceName": updated.Name,
		"previousWorkspaceRootPath": previous.RootPath, "previousWorkspaceName": previous.Name,
	})
	return ""
}

func (a *App) MoveFolderV2(folderID, targetParentFolderID string) string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	_, err := a.treeV2.MoveFolder(folderID, targetParentFolderID, func() error {
		if a.fileWatcher != nil {
			return a.fileWatcher.RefreshBaseline()
		}
		return nil
	})
	if err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) MoveWorkspaceV2(workspaceID, targetParentFolderID string) string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	previous, ok := a.treeV2.GetWorkspaceByID(workspaceID)
	if !ok {
		return "workspace not found"
	}
	updated, err := a.treeV2.MoveWorkspace(workspaceID, targetParentFolderID, func() error {
		if a.fileWatcher != nil {
			return a.fileWatcher.RefreshBaseline()
		}
		return nil
	})
	if err != nil {
		return err.Error()
	}
	if err := a.recordWorkspaceSyncOp(syncsvc.OpRename, updated.ID, updated.RootPath, previous.RootPath, updated.Name); err != nil {
		return err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceRenamedEventName, map[string]interface{}{
		"operation": "move", "workspaceId": updated.ID, "workspaceRootPath": updated.RootPath, "workspaceName": updated.Name,
		"previousWorkspaceRootPath": previous.RootPath, "previousWorkspaceName": previous.Name,
	})
	return ""
}

// PlaceWorkspaceTreeNodeV2 applies one backend-authoritative stable-key
// placement and records both structural and order metadata changes for sync.
func (a *App) PlaceWorkspaceTreeNodeV2(request workspacetree.PlacementRequest) string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	_, err := a.treeV2.PlaceNode(request, func() error {
		if a.fileWatcher != nil {
			return a.fileWatcher.RefreshBaseline()
		}
		return nil
	})
	if err != nil {
		return err.Error()
	}
	if _, err := a.scanLocalChanges(); err != nil {
		return err.Error()
	}
	if a.ctx != nil {
		emitFrontendEvent(a.ctx, "verstak:workspace-tree-changed")
	}
	return ""
}

func (a *App) TrashFolderV2(folderID string) map[string]interface{} {
	if a.treeV2 == nil {
		return map[string]interface{}{"error": "not initialized"}
	}
	entry, err := a.treeV2.TrashFolder(folderID, func() error {
		if a.fileWatcher != nil {
			return a.fileWatcher.RefreshBaseline()
		}
		return nil
	})
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"trashId":      entry.TrashID,
		"entityType":   entry.EntityType,
		"entityId":     entry.EntityID,
		"originalPath": entry.OriginalPath,
		"deletedAt":    entry.DeletedAt,
	}
}

func (a *App) TrashWorkspaceV2(workspaceID string) map[string]interface{} {
	if a.treeV2 == nil {
		return map[string]interface{}{"error": "not initialized"}
	}
	previous, ok := a.treeV2.GetWorkspaceByID(workspaceID)
	if !ok {
		return map[string]interface{}{"error": "workspace not found"}
	}
	entry, err := a.treeV2.TrashWorkspace(workspaceID, func() error {
		if a.fileWatcher != nil {
			return a.fileWatcher.RefreshBaseline()
		}
		return nil
	})
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if err := a.recordWorkspaceSyncOp(syncsvc.OpTrash, entry.EntityID, previous.RootPath, "", previous.Name); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	trashPath := filepath.ToSlash(filepath.Join(".verstak", "trash", "tree", entry.TrashID))
	a.publishWorkspaceLifecycleEvent(workspaceTrashedEventName, map[string]interface{}{
		"operation": "trash", "workspaceId": entry.EntityID, "workspaceRootPath": previous.RootPath, "workspaceName": previous.Name,
		"trashId": entry.TrashID, "trashPath": trashPath, "deletedAt": entry.DeletedAt,
	})
	return map[string]interface{}{
		"trashId":      entry.TrashID,
		"entityType":   entry.EntityType,
		"entityId":     entry.EntityID,
		"originalPath": entry.OriginalPath,
		"deletedAt":    entry.DeletedAt,
		"trashPath":    trashPath,
	}
}

func (a *App) GetFolderAppearance(folderID string) map[string]interface{} {
	if a.treeV2 == nil {
		return map[string]interface{}{"error": "not initialized"}
	}
	appearance, err := a.treeV2.GetFolderAppearance(folderID)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"icon":  appearance.Icon,
		"color": appearance.Color,
	}
}

func (a *App) SetFolderAppearance(folderID string, patch map[string]interface{}) string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	fa := &workspacetree.FolderAppearance{}
	if v, ok := patch["icon"].(string); ok {
		fa.Icon = v
	}
	if v, ok := patch["color"].(string); ok {
		fa.Color = v
	}
	if err := a.treeV2.ReplaceFolderAppearance(folderID, fa); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) ResetFolderAppearance(folderID string) string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	if err := a.treeV2.ResetFolderAppearance(folderID); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) SetCurrentWorkspaceV2(workspaceID string) string {
	if a.treeV2 == nil {
		return "not initialized"
	}
	if err := a.treeV2.SetCurrentWorkspaceID(workspaceID); err != nil {
		return err.Error()
	}
	if workspace, ok := a.treeV2.GetWorkspaceByID(workspaceID); ok {
		a.publishWorkspaceLifecycleEvent(workspaceSelectedEventName, map[string]interface{}{
			"operation":         "select",
			"workspaceId":       workspace.ID,
			"workspaceRootPath": workspace.RootPath,
			"workspaceName":     workspace.Name,
		})
	}
	return ""
}
