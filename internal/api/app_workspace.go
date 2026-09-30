package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/dealmigration"
	"github.com/verstak/verstak-desktop/internal/core/events"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/core/workspace"
	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

const workspaceCreatedEventName = "workspace.created"
const workspaceRenamedEventName = "workspace.renamed"
const workspaceTrashedEventName = "workspace.trashed"
const workspaceRestoredEventName = "workspace.restored"
const workspacePurgedEventName = "workspace.purged"
const workspaceSelectedEventName = "workspace.selected"

func workspaceRootFromRelativePath(relativePath string) string {
	path := strings.Trim(strings.TrimSpace(filepath.ToSlash(relativePath)), "/")
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return path
}

func (a *App) publishWorkspaceLifecycleEvent(eventName string, payload map[string]interface{}) {
	if a.eventBus == nil {
		return
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	workspaceRoot := strings.TrimSpace(fmt.Sprint(payload["workspaceRootPath"]))
	if workspaceRoot == "" || workspaceRoot == "<nil>" {
		workspaceRoot = strings.TrimSpace(fmt.Sprint(payload["workspaceName"]))
	}
	if workspaceRoot != "" && workspaceRoot != "<nil>" {
		payload["workspaceRootPath"] = workspaceRoot
		if _, ok := payload["workspaceName"]; !ok {
			payload["workspaceName"] = workspaceRoot
		}
		if _, ok := payload["title"]; !ok {
			payload["title"] = workspaceRoot
		}
	}
	a.eventBus.Publish(events.Event{
		Name:      eventName,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   payload,
	})
}

// preflightDealMigration remains a read-only diagnostic for tests and support.
func (a *App) preflightDealMigration(ctx context.Context) (dealmigration.Ledger, error) {
	if a == nil || a.vault == nil || a.vault.GetVaultStatus() != vault.StatusOpen {
		return dealmigration.Ledger{}, fmt.Errorf("vault is not open")
	}
	runner := dealmigration.NewDealOnlyRunner(a.vault.GetVaultPath())
	if err := runner.Preflight(ctx); err != nil {
		return dealmigration.Ledger{}, err
	}
	ledger, err := runner.ReadLedger()
	if errors.Is(err, os.ErrNotExist) {
		return dealmigration.Ledger{}, nil
	}
	return ledger, err
}

func (a *App) runDealMigration(ctx context.Context) error {
	if a == nil || a.vault == nil || a.vault.GetVaultStatus() != vault.StatusOpen {
		return fmt.Errorf("vault is not open")
	}
	runner := dealmigration.NewDealOnlyRunner(a.vault.GetVaultPath())
	needed, err := runner.NeedsMigration(ctx)
	if err != nil || !needed {
		return err
	}
	return runner.Run(ctx)
}

// ListWorkspaces returns all Deals from the canonical UUID tree.
func (a *App) ListWorkspaces() ([]workspace.Workspace, string) {
	if a.treeV2 == nil {
		return nil, "workspace not initialized"
	}
	items := a.treeV2.ListWorkspaces()
	workspaces := make([]workspace.Workspace, 0, len(items))
	for _, item := range items {
		workspaces = append(workspaces, workspace.Workspace{ID: item.ID, Name: item.Name, RootPath: item.RootPath})
	}
	return workspaces, ""
}

// ListWorkspaceIdentities returns durable workspace identities for relation-aware plugins.
func (a *App) ListWorkspaceIdentities() ([]workspace.WorkspaceIdentity, string) {
	if a.treeV2 == nil {
		return nil, "workspace not initialized"
	}
	items := a.treeV2.ListWorkspaces()
	identities := make([]workspace.WorkspaceIdentity, 0, len(items))
	for _, item := range items {
		identities = append(identities, workspace.WorkspaceIdentity{WorkspaceID: item.ID, RootPath: item.RootPath, State: "active"})
	}
	return identities, ""
}

// RepairWorkspaceIdentity resolves a duplicated workspace marker without moving relations.
func (a *App) RepairWorkspaceIdentity(keepName, regenerateName string) string {
	return "legacy identity repair is retired; use Deal tree diagnostics"
}

func (a *App) refreshWorkspaceBaseline() error {
	if a.fileWatcher != nil {
		return a.fileWatcher.RefreshBaseline()
	}
	return nil
}

// RenameWorkspace is a compatibility adapter accepting a Deal UUID or path.
func (a *App) RenameWorkspace(oldName, newName string) string {
	if a.treeV2 == nil {
		return "workspace not initialized"
	}
	current, ok := a.treeV2.ResolveWorkspace(oldName)
	if !ok {
		return "workspace not found"
	}
	updated, err := a.treeV2.RenameWorkspace(current.ID, newName, a.refreshWorkspaceBaseline)
	if err != nil {
		return err.Error()
	}
	if err := a.recordWorkspaceSyncOp(syncsvc.OpRename, updated.ID, updated.RootPath, current.RootPath, updated.Name); err != nil {
		return err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceRenamedEventName, map[string]interface{}{
		"operation":                 "rename",
		"workspaceId":               updated.ID,
		"workspaceRootPath":         updated.RootPath,
		"workspaceName":             updated.Name,
		"previousWorkspaceRootPath": current.RootPath,
		"previousWorkspaceName":     current.Name,
	})
	return ""
}

// TrashWorkspace is a compatibility adapter accepting a Deal UUID or path.
func (a *App) TrashWorkspace(reference string) (workspace.TrashResult, string) {
	if a.treeV2 == nil {
		return workspace.TrashResult{}, "workspace not initialized"
	}
	current, ok := a.treeV2.ResolveWorkspace(reference)
	if !ok {
		return workspace.TrashResult{}, "workspace not found"
	}
	entry, err := a.treeV2.TrashWorkspace(current.ID, a.refreshWorkspaceBaseline)
	if err != nil {
		return workspace.TrashResult{}, err.Error()
	}
	result := workspace.TrashResult{
		WorkspaceID:  entry.EntityID,
		OriginalPath: entry.OriginalPath,
		TrashPath:    filepath.ToSlash(filepath.Join(".verstak", "trash", "tree", entry.TrashID)),
		TrashID:      entry.TrashID,
		DeletedAt:    entry.DeletedAt,
	}
	if err := a.recordWorkspaceSyncOp(syncsvc.OpTrash, result.WorkspaceID, current.RootPath, "", current.Name); err != nil {
		return workspace.TrashResult{}, err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceTrashedEventName, map[string]interface{}{
		"operation":         "trash",
		"workspaceId":       result.WorkspaceID,
		"workspaceRootPath": current.RootPath,
		"workspaceName":     current.Name,
		"trashId":           result.TrashID,
		"trashPath":         result.TrashPath,
		"deletedAt":         result.DeletedAt,
	})
	return result, ""
}

// RestoreWorkspaceTrash restores a trashed workspace and publishes its durable identity.
func (a *App) RestoreWorkspaceTrash(trashID, targetName string) (workspace.Workspace, string) {
	if a.treeV2 == nil {
		return workspace.Workspace{}, "workspace not initialized"
	}
	value, err := a.treeV2.RestoreTreeTrash(trashID, "", a.refreshWorkspaceBaseline)
	if err != nil {
		return workspace.Workspace{}, err.Error()
	}
	restored, ok := value.(workspacetree.ScannedWorkspace)
	if !ok {
		return workspace.Workspace{}, "trash entry is not a workspace"
	}
	if targetName = strings.TrimSpace(targetName); targetName != "" && targetName != restored.Name {
		restored, err = a.treeV2.RenameWorkspace(restored.ID, targetName, a.refreshWorkspaceBaseline)
		if err != nil {
			return workspace.Workspace{}, err.Error()
		}
	}
	result := workspace.Workspace{ID: restored.ID, Name: restored.Name, RootPath: restored.RootPath}
	if err := a.recordWorkspaceSyncOp(syncsvc.OpRestore, restored.ID, restored.RootPath, "", restored.Name); err != nil {
		return workspace.Workspace{}, err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceRestoredEventName, map[string]interface{}{
		"operation":         "restore",
		"workspaceId":       restored.ID,
		"workspaceRootPath": restored.RootPath,
		"workspaceName":     restored.Name,
		"trashId":           trashID,
	})
	return result, ""
}

// PurgeWorkspaceTrash permanently removes a trashed workspace and publishes its former identity.
func (a *App) PurgeWorkspaceTrash(trashID string) string {
	if a.treeV2 == nil {
		return "workspace not initialized"
	}
	entries, err := a.treeV2.ListTreeTrash()
	if err != nil {
		return err.Error()
	}
	var entry workspacetree.TrashEntry
	found := false
	for _, candidate := range entries {
		if candidate.TrashID == trashID && candidate.EntityType == "workspace" {
			entry = candidate
			found = true
			break
		}
	}
	if !found {
		return "workspace trash entry not found"
	}
	if err := a.treeV2.PurgeTreeTrash(trashID); err != nil {
		return err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspacePurgedEventName, map[string]interface{}{
		"operation":         "purge",
		"workspaceId":       entry.EntityID,
		"workspaceRootPath": entry.OriginalPath,
		"workspaceName":     filepath.Base(filepath.FromSlash(entry.OriginalPath)),
		"trashId":           trashID,
	})
	return ""
}

func legacyWorkspaceMetadata(metadata workspacetree.DealMetadata) workspace.Metadata {
	legacy := workspace.Metadata{
		WorkspaceID:    metadata.WorkspaceID,
		WorkspaceName:  metadata.WorkspaceName,
		WorkspaceTools: append([]string(nil), metadata.WorkspaceTools...),
		UpdatedAt:      metadata.UpdatedAt,
	}
	if provenance := metadata.CreatedFromTemplate; provenance != nil {
		legacy.CreatedFromTemplate = &workspace.TemplateSnapshot{
			TemplateID:      provenance.TemplateID,
			TemplateName:    provenance.TemplateName,
			TemplateVersion: provenance.TemplateVersion,
			AppliedAt:       provenance.AppliedAt,
			WorkspaceTools:  append([]string(nil), metadata.WorkspaceTools...),
		}
	}
	return legacy
}

// GetWorkspaceMetadata returns canonical metadata for a Deal reference.
func (a *App) GetWorkspaceMetadata(name string) (workspace.Metadata, string) {
	if a.treeV2 == nil {
		return workspace.Metadata{}, "workspace not initialized"
	}
	item, ok := a.treeV2.ResolveWorkspace(name)
	if !ok {
		return workspace.Metadata{}, "workspace not found"
	}
	meta, err := a.treeV2.ReadDealMetadata(item.ID, item.RootPath)
	if err != nil {
		return workspace.Metadata{}, err.Error()
	}
	return legacyWorkspaceMetadata(meta), ""
}

// GetWorkspaceMetadataByUUID returns metadata for a workspace by its durable UUID.
func (a *App) GetWorkspaceMetadataByUUID(workspaceID string) (workspace.Metadata, string) {
	if a.treeV2 == nil {
		return workspace.Metadata{}, "workspace not initialized"
	}
	item, ok := a.treeV2.GetWorkspaceByID(workspaceID)
	if !ok {
		return workspace.Metadata{}, "workspace not found"
	}
	meta, err := a.treeV2.ReadDealMetadata(item.ID, item.RootPath)
	if err != nil {
		return workspace.Metadata{}, err.Error()
	}
	return legacyWorkspaceMetadata(meta), ""
}

// ReadPluginDealConfig returns only the calling plugin's namespaced Deal
// metadata. It intentionally does not expose another plugin's ToolConfig.
func (a *App) ReadPluginDealConfig(pluginID, workspaceID string) (map[string]interface{}, string) {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		return map[string]interface{}{}, err.Error()
	}
	if a.treeV2 == nil {
		return map[string]interface{}{}, "workspace not initialized"
	}
	item, ok := a.treeV2.GetWorkspaceByID(strings.TrimSpace(workspaceID))
	if !ok {
		return map[string]interface{}{}, "workspace not found"
	}
	metadata, err := a.treeV2.ReadDealMetadata(item.ID, item.RootPath)
	if err != nil {
		return map[string]interface{}{}, err.Error()
	}
	raw := metadata.ToolConfig[pluginID]
	if len(raw) == 0 {
		return map[string]interface{}{}, ""
	}
	var config map[string]interface{}
	if err := json.Unmarshal(raw, &config); err != nil || config == nil {
		if err == nil {
			err = errors.New("config must be an object")
		}
		return map[string]interface{}{}, fmt.Sprintf("decode Deal config: %v", err)
	}
	return config, ""
}

// WritePluginDealConfig atomically replaces the calling plugin's own config
// namespace in canonical Deal metadata while retaining every other namespace.
func (a *App) WritePluginDealConfig(pluginID, workspaceID string, config map[string]interface{}) string {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		return err.Error()
	}
	if a.treeV2 == nil {
		return "workspace not initialized"
	}
	item, ok := a.treeV2.GetWorkspaceByID(strings.TrimSpace(workspaceID))
	if !ok {
		return "workspace not found"
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return fmt.Sprintf("encode Deal config: %v", err)
	}
	metadata, err := a.treeV2.ReadDealMetadata(item.ID, item.RootPath)
	if err != nil {
		return err.Error()
	}
	if metadata.ToolConfig == nil {
		metadata.ToolConfig = make(map[string]json.RawMessage)
	}
	metadata.ToolConfig[pluginID] = append(json.RawMessage(nil), encoded...)
	metadata.UpdatedAt = ""
	if err := a.treeV2.WriteDealMetadata(metadata); err != nil {
		return err.Error()
	}
	return ""
}

// UpdateWorkspaceMetadata merges metadata for an existing workspace.
func (a *App) UpdateWorkspaceMetadata(name string, patch workspace.MetadataPatch) (workspace.Metadata, string) {
	if a.treeV2 == nil {
		return workspace.Metadata{}, "workspace not initialized"
	}
	if len(patch.Features) > 0 || len(patch.Folders) > 0 {
		return workspace.Metadata{}, "legacy feature/folder metadata is retired"
	}
	item, ok := a.treeV2.ResolveWorkspace(name)
	if !ok {
		return workspace.Metadata{}, "workspace not found"
	}
	meta, err := a.treeV2.ReadDealMetadata(item.ID, item.RootPath)
	if err != nil {
		return workspace.Metadata{}, err.Error()
	}
	if patch.WorkspaceTools != nil {
		meta.WorkspaceTools = append([]string(nil), patch.WorkspaceTools...)
		meta.UpdatedAt = ""
		if err := a.treeV2.WriteDealMetadata(meta); err != nil {
			return workspace.Metadata{}, err.Error()
		}
	}
	return legacyWorkspaceMetadata(meta), ""
}

// GetCurrentWorkspace returns the currently selected Deal.
func (a *App) GetCurrentWorkspace() map[string]interface{} {
	if a.treeV2 == nil {
		return map[string]interface{}{"status": "not initialized"}
	}
	workspaceID := a.treeV2.GetCurrentWorkspaceID()
	if workspaceID == "" {
		return map[string]interface{}{"error": "no current workspace"}
	}
	item, ok := a.treeV2.GetWorkspaceByID(workspaceID)
	if !ok {
		return map[string]interface{}{"error": "current workspace not found"}
	}
	return map[string]interface{}{
		"id":          item.ID,
		"workspaceId": item.ID,
		"name":        item.Name,
		"rootPath":    item.RootPath,
	}
}

// SetCurrentWorkspace is a compatibility adapter accepting a Deal UUID or path.
func (a *App) SetCurrentWorkspace(reference string) string {
	if a.treeV2 == nil {
		return "workspace not initialized"
	}
	item, ok := a.treeV2.ResolveWorkspace(reference)
	if !ok {
		return "workspace not found"
	}
	if err := a.treeV2.SetCurrentWorkspaceID(item.ID); err != nil {
		return err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceSelectedEventName, map[string]interface{}{
		"operation":         "select",
		"workspaceId":       item.ID,
		"workspaceRootPath": item.RootPath,
		"workspaceName":     item.Name,
	})
	return ""
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer ListWorkspaces.
func (a *App) GetWorkspaceTree() map[string]interface{} {
	if a.treeV2 == nil {
		return map[string]interface{}{"status": "not initialized"}
	}
	tree := a.treeV2.GetTree()
	nodes := make([]workspace.WorkspaceNode, 0)
	var appendNodes func([]workspacetree.TreeNode, string)
	appendNodes = func(items []workspacetree.TreeNode, parentID string) {
		for _, item := range items {
			nodeType := workspace.TypeFolder
			if item.Kind == "workspace" {
				nodeType = workspace.TypeSpace
			}
			nodes = append(nodes, workspace.WorkspaceNode{
				ID:       item.ID,
				ParentID: parentID,
				Type:     nodeType,
				Title:    item.Name,
				Name:     item.Name,
				RootPath: item.Path,
				Status:   workspace.StatusActive,
			})
			appendNodes(item.Children, item.ID)
		}
	}
	appendNodes(tree.Roots, "")
	return map[string]interface{}{
		"schemaVersion": 2,
		"nodes":         nodes,
		"currentNodeId": tree.CurrentWorkspaceID,
	}
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer RenameWorkspace.
func (a *App) RenameWorkspaceNode(id, title string) string {
	if a.treeV2 == nil {
		return "workspace not initialized"
	}
	if _, ok := a.treeV2.GetWorkspaceByID(id); ok {
		_, err := a.treeV2.RenameWorkspace(id, title, a.refreshWorkspaceBaseline)
		if err != nil {
			return err.Error()
		}
		return ""
	}
	if _, ok := a.treeV2.GetFolderByID(id); ok {
		_, err := a.treeV2.RenameFolder(id, title, a.refreshWorkspaceBaseline)
		if err != nil {
			return err.Error()
		}
		return ""
	}
	return "workspace tree node not found"
}

// Deprecated: compatibility wrapper retained only to reject the retired node
// API. Use PlaceWorkspaceTreeNodeV2 for UUID-based Deal tree placement.
func (a *App) MoveWorkspaceNode(id, newParentID string) string {
	return "legacy workspace node moves are unsupported; use the Deal tree"
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer TrashWorkspace.
func (a *App) ArchiveWorkspaceNode(id string) string {
	if a.treeV2 == nil {
		return "workspace not initialized"
	}
	if _, ok := a.treeV2.GetWorkspaceByID(id); ok {
		_, err := a.treeV2.TrashWorkspace(id, a.refreshWorkspaceBaseline)
		if err != nil {
			return err.Error()
		}
		return ""
	}
	if _, ok := a.treeV2.GetFolderByID(id); ok {
		_, err := a.treeV2.TrashFolder(id, a.refreshWorkspaceBaseline)
		if err != nil {
			return err.Error()
		}
		return ""
	}
	return "workspace tree node not found"
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer GetCurrentWorkspace.
func (a *App) GetCurrentWorkspaceNode() map[string]interface{} {
	return a.GetCurrentWorkspace()
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer SetCurrentWorkspace.
func (a *App) SetCurrentWorkspaceNode(id string) string {
	return a.SetCurrentWorkspace(id)
}
