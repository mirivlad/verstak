package api

import (
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/externalopen"
	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
)

type externalOpenService interface {
	OpenPath(path string) error
	ShowInFolder(path string, isDir bool) error
}

// ListVaultFiles lists a vault-relative directory for a plugin with files.read.
func (a *App) ListVaultFiles(pluginID, relativeDir string) ([]corefiles.FileEntry, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return nil, err.Error()
	}
	if a.files == nil {
		return nil, "files service not initialized"
	}
	entries, err := a.files.ListVaultFiles(relativeDir)
	if err != nil {
		return nil, err.Error()
	}
	return entries, ""
}

// GetVaultFileMetadata returns metadata for a vault-relative path for a plugin with files.read.
func (a *App) GetVaultFileMetadata(pluginID, relativePath string) (corefiles.FileMetadata, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return corefiles.FileMetadata{}, err.Error()
	}
	if a.files == nil {
		return corefiles.FileMetadata{}, "files service not initialized"
	}
	meta, err := a.files.GetVaultFileMetadata(relativePath)
	if err != nil {
		return corefiles.FileMetadata{}, err.Error()
	}
	return meta, ""
}

// ReadVaultTextFile reads a UTF-8 text file for a plugin with files.read.
func (a *App) ReadVaultTextFile(pluginID, relativePath string) (string, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return "", err.Error()
	}
	if a.files == nil {
		return "", "files service not initialized"
	}
	text, err := a.files.ReadVaultTextFile(relativePath)
	if err != nil {
		return "", err.Error()
	}
	return text, ""
}

// ReadVaultFileBytes reads a bounded regular file as base64 for a plugin with files.read.
func (a *App) ReadVaultFileBytes(pluginID, relativePath string) (corefiles.FileBytes, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return corefiles.FileBytes{}, err.Error()
	}
	if a.files == nil {
		return corefiles.FileBytes{}, "files service not initialized"
	}
	data, err := a.files.ReadVaultFileBytes(relativePath)
	if err != nil {
		return corefiles.FileBytes{}, err.Error()
	}
	return data, ""
}

// WriteVaultTextFile atomically writes a UTF-8 text file for a plugin with files.write.
func (a *App) WriteVaultTextFile(pluginID, relativePath string, content string, options corefiles.WriteOptions) string {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	opType := syncsvc.OpUpdate
	if _, err := a.files.GetVaultFileMetadata(relativePath); err != nil {
		if isSyncNotFound(err) {
			opType = syncsvc.OpCreate
		} else {
			return err.Error()
		}
	}
	if err := a.files.WriteVaultTextFile(relativePath, content, options); err != nil {
		return err.Error()
	}
	if err := a.recordFileSyncPaths(relativePath); err != nil {
		return err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, relativePath, writeActivityPayload(opType, options))
	return ""
}

// WriteVaultFileBytes atomically writes a bounded base64 file for a plugin with files.write.
func (a *App) WriteVaultFileBytes(pluginID, relativePath string, dataBase64 string, options corefiles.WriteOptions) string {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	opType := syncsvc.OpUpdate
	if _, err := a.files.GetVaultFileMetadata(relativePath); err != nil {
		if isSyncNotFound(err) {
			opType = syncsvc.OpCreate
		} else {
			return err.Error()
		}
	}
	if err := a.files.WriteVaultFileBytes(relativePath, dataBase64, options); err != nil {
		return err.Error()
	}
	if err := a.recordFileSyncPaths(relativePath); err != nil {
		return err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, relativePath, writeActivityPayload(opType, options))
	return ""
}

// CreateVaultFolder creates a vault-relative folder for a plugin with files.write.
func (a *App) CreateVaultFolder(pluginID, relativePath string) string {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	if err := a.files.CreateVaultFolder(relativePath); err != nil {
		return err.Error()
	}
	if err := a.recordFileSyncPaths(relativePath); err != nil {
		return err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, relativePath, map[string]interface{}{
		"operation": syncsvc.OpCreate,
		"type":      string(corefiles.FileTypeFolder),
	})
	return ""
}

// MoveVaultPath moves a vault-relative file or folder for a plugin with files.write.
func (a *App) MoveVaultPath(pluginID, fromRelativePath string, toRelativePath string, options corefiles.MoveOptions) string {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	meta, err := a.files.GetVaultFileMetadata(fromRelativePath)
	if err != nil {
		return err.Error()
	}
	if err := a.files.MoveVaultPath(fromRelativePath, toRelativePath, options); err != nil {
		return err.Error()
	}
	// Both ends changed: one lost the entry, the other gained it.
	if err := a.recordFileSyncPaths(fromRelativePath, toRelativePath); err != nil {
		return err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, fromRelativePath, map[string]interface{}{
		"operation": syncsvc.OpMove,
		"toPath":    toRelativePath,
		"type":      string(meta.Type),
	})
	a.publishFileActivity("file.changed", pluginID, toRelativePath, map[string]interface{}{
		"operation": syncsvc.OpMove,
		"fromPath":  fromRelativePath,
		"type":      string(meta.Type),
	})
	return ""
}

// CopyVaultPath copies a vault-relative file or folder for a plugin with files.read and files.write.
func (a *App) CopyVaultPath(pluginID, fromRelativePath string, toRelativePath string, options corefiles.CopyOptions) string {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return err.Error()
	}
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	meta, err := a.files.GetVaultFileMetadata(fromRelativePath)
	if err != nil {
		return err.Error()
	}
	if err := a.files.CopyVaultPath(fromRelativePath, toRelativePath, options); err != nil {
		return err.Error()
	}
	if err := a.recordFileSyncPaths(toRelativePath); err != nil {
		return err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, toRelativePath, map[string]interface{}{
		"operation":  syncsvc.OpCreate,
		"copiedFrom": fromRelativePath,
		"type":       string(meta.Type),
	})
	return ""
}

// MoveVaultPaths moves many vault-relative paths in one call for a plugin with
// files.write.
//
// Issuing one call per file used to mean one sync scan per file, each holding a
// global lock; pasting a folder's worth of files could occupy the backend for
// the better part of a minute with nothing on screen to explain it. One call
// means one scan covering every path it touched, and gives the interface
// something to report progress against and something to cancel.
func (a *App) MoveVaultPaths(pluginID, transferID string, transfers []corefiles.PathTransfer, options corefiles.MoveOptions) (corefiles.TransferOutcome, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return corefiles.TransferOutcome{}, err.Error()
	}
	return a.runVaultTransfers(pluginID, transferID, transfers, syncsvc.OpMove, func(transfer corefiles.PathTransfer) error {
		return a.files.MoveVaultPath(transfer.From, transfer.To, options)
	})
}

// CopyVaultPaths copies many vault-relative paths in one call for a plugin with
// files.read and files.write.
func (a *App) CopyVaultPaths(pluginID, transferID string, transfers []corefiles.PathTransfer, options corefiles.CopyOptions) (corefiles.TransferOutcome, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return corefiles.TransferOutcome{}, err.Error()
	}
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return corefiles.TransferOutcome{}, err.Error()
	}
	return a.runVaultTransfers(pluginID, transferID, transfers, syncsvc.OpCreate, func(transfer corefiles.PathTransfer) error {
		return a.files.CopyVaultPath(transfer.From, transfer.To, options)
	})
}

// CancelVaultTransfer asks a running bulk transfer to stop. Items already moved
// or copied stay where they are: the operation stops, it does not undo itself.
func (a *App) CancelVaultTransfer(pluginID, transferID string) string {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if strings.TrimSpace(transferID) == "" {
		return "cancel requires a transfer id"
	}
	a.transfersMu.Lock()
	defer a.transfersMu.Unlock()
	if a.cancelledTransfers == nil {
		a.cancelledTransfers = make(map[string]bool)
	}
	a.cancelledTransfers[transferID] = true
	return ""
}

func (a *App) transferCancelled(transferID string) bool {
	if transferID == "" {
		return false
	}
	a.transfersMu.Lock()
	defer a.transfersMu.Unlock()
	return a.cancelledTransfers[transferID]
}

func (a *App) forgetTransfer(transferID string) {
	if transferID == "" {
		return
	}
	a.transfersMu.Lock()
	defer a.transfersMu.Unlock()
	delete(a.cancelledTransfers, transferID)
}

func (a *App) runVaultTransfers(pluginID, transferID string, transfers []corefiles.PathTransfer, operation string, apply func(corefiles.PathTransfer) error) (corefiles.TransferOutcome, string) {
	if a.files == nil {
		return corefiles.TransferOutcome{}, "files service not initialized"
	}
	if len(transfers) == 0 {
		return corefiles.TransferOutcome{Results: []corefiles.TransferResult{}}, ""
	}
	defer a.forgetTransfer(transferID)

	outcome := corefiles.TransferOutcome{Results: make([]corefiles.TransferResult, 0, len(transfers))}
	touched := make([]string, 0, len(transfers)*2)
	type completed struct {
		transfer corefiles.PathTransfer
		fileType corefiles.FileType
	}
	done := make([]completed, 0, len(transfers))

	for index, transfer := range transfers {
		if a.transferCancelled(transferID) {
			outcome.Cancelled = true
			for _, remaining := range transfers[index:] {
				outcome.Results = append(outcome.Results, corefiles.TransferResult{
					From: remaining.From, To: remaining.To, Skipped: true,
				})
			}
			break
		}

		result := corefiles.TransferResult{From: transfer.From, To: transfer.To}
		// Read the type before the move: afterwards the source is gone.
		meta, err := a.files.GetVaultFileMetadata(transfer.From)
		if err != nil {
			result.Error = err.Error()
		} else if err := apply(transfer); err != nil {
			result.Error = err.Error()
		}
		if result.Error == "" {
			outcome.Succeeded++
			touched = append(touched, transfer.To)
			if operation == syncsvc.OpMove {
				touched = append(touched, transfer.From)
			}
			done = append(done, completed{transfer: transfer, fileType: meta.Type})
		} else {
			outcome.Failed++
		}
		outcome.Results = append(outcome.Results, result)

		emitFrontendEvent(a.ctx, "verstak:files-transfer-progress", map[string]interface{}{
			"transferId": transferID,
			"pluginId":   pluginID,
			"completed":  index + 1,
			"total":      len(transfers),
			"path":       transfer.To,
			"succeeded":  outcome.Succeeded,
			"failed":     outcome.Failed,
		})
	}

	// One scan for the whole batch, scoped to what it actually touched. This is
	// the difference between a paste that finishes and a paste that hangs.
	if len(touched) > 0 {
		if err := a.recordFileSyncPaths(touched...); err != nil {
			return outcome, err.Error()
		}
	}
	for _, item := range done {
		if operation == syncsvc.OpMove {
			a.publishFileActivity("file.changed", pluginID, item.transfer.From, map[string]interface{}{
				"operation": syncsvc.OpMove,
				"toPath":    item.transfer.To,
				"type":      string(item.fileType),
			})
		}
		a.publishFileActivity("file.changed", pluginID, item.transfer.To, map[string]interface{}{
			"operation": operation,
			"fromPath":  item.transfer.From,
			"type":      string(item.fileType),
		})
	}
	return outcome, ""
}

// TrashVaultPath moves a vault-relative file or folder to internal trash for a plugin with files.delete.
func (a *App) TrashVaultPath(pluginID, relativePath string) (corefiles.TrashResult, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.delete"); err != nil {
		return corefiles.TrashResult{}, err.Error()
	}
	if a.files == nil {
		return corefiles.TrashResult{}, "files service not initialized"
	}
	meta, err := a.files.GetVaultFileMetadata(relativePath)
	if err != nil {
		return corefiles.TrashResult{}, err.Error()
	}
	result, err := a.files.TrashVaultPath(relativePath)
	if err != nil {
		return corefiles.TrashResult{}, err.Error()
	}
	if err := a.recordFileSyncPaths(relativePath); err != nil {
		return corefiles.TrashResult{}, err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, relativePath, map[string]interface{}{
		"operation": syncsvc.OpDelete,
		"type":      string(meta.Type),
	})
	return result, ""
}

// ListVaultTrash returns trash metadata entries for a plugin with files.delete.
func (a *App) ListVaultTrash(pluginID string) ([]corefiles.TrashEntry, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.delete"); err != nil {
		return nil, err.Error()
	}
	if a.files == nil {
		return nil, "files service not initialized"
	}
	entries, err := a.files.ListTrashEntries()
	if err != nil {
		return nil, err.Error()
	}

	// Merge tree trash workspaces and folders.
	if a.treeV2 != nil {
		treeEntries, _ := a.treeV2.ListTreeTrash()
		for _, te := range treeEntries {
			basename := filepath.Base(filepath.FromSlash(te.OriginalPath))
			entries = append(entries, corefiles.TrashEntry{
				TrashID:      te.TrashID,
				OriginalPath: te.OriginalPath,
				OriginalType: corefiles.FileType(te.EntityType),
				Basename:     basename,
				DeletedAt:    te.DeletedAt,
			})
		}
	}

	return entries, ""
}

// RestoreVaultTrash restores a file or folder from internal trash for a plugin with files.delete and files.write.
func (a *App) RestoreVaultTrash(pluginID, trashID string, options corefiles.RestoreOptions) (string, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.delete"); err != nil {
		return "", err.Error()
	}
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return "", err.Error()
	}
	if a.files == nil {
		return "", "files service not initialized"
	}
	entries, err := a.files.ListTrashEntries()
	if err != nil {
		return "", err.Error()
	}
	var entry corefiles.TrashEntry
	for _, candidate := range entries {
		if candidate.TrashID == trashID {
			entry = candidate
			break
		}
	}

	// Not found in file trash — try tree trash.
	if entry.TrashID == "" && a.treeV2 != nil {
		treeEntries, _ := a.treeV2.ListTreeTrash()
		for _, te := range treeEntries {
			if te.TrashID == trashID {
				_, err := a.treeV2.RestoreTreeTrash(trashID, "", func() error { return nil })
				if err != nil {
					return "", err.Error()
				}
				emitFrontendEvent(a.ctx, "verstak:workspace-tree-changed")
				return te.OriginalPath, ""
			}
		}
	}

	if entry.TrashID == "" {
		return "", "not-found: trash entry " + trashID
	}
	restoredPath, err := a.files.RestoreTrashEntry(trashID, options)
	if err != nil {
		return "", err.Error()
	}
	if err := a.recordFileSyncPaths(restoredPath); err != nil {
		return "", err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, restoredPath, map[string]interface{}{
		"operation": syncsvc.OpCreate,
		"type":      string(entry.OriginalType),
		"restored":  true,
		"trashId":   trashID,
	})
	return restoredPath, ""
}

// DeleteVaultTrash permanently removes an internal trash entry for a plugin with files.delete.
func (a *App) DeleteVaultTrash(pluginID, trashID string) string {
	if _, err := a.requirePluginAccess(pluginID, "files.delete"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}

	// Try tree trash first (workspaces/folders).
	if a.treeV2 != nil {
		treeEntries, _ := a.treeV2.ListTreeTrash()
		for _, te := range treeEntries {
			if te.TrashID == trashID {
				if err := a.treeV2.PurgeTreeTrash(trashID); err != nil {
					return err.Error()
				}
				emitFrontendEvent(a.ctx, "verstak:workspace-tree-changed")
				return ""
			}
		}
	}

	// Fall through to file trash.
	if err := a.files.DeleteTrashEntry(trashID); err != nil {
		return err.Error()
	}
	return ""
}

// OpenVaultPathExternal opens a vault-relative file or folder in the OS default app.
func (a *App) OpenVaultPathExternal(pluginID, relativePath string) string {
	if _, err := a.requirePluginAccess(pluginID, "files.openExternal"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	target, err := a.files.ResolveExternalOpenTarget(relativePath)
	if err != nil {
		return err.Error()
	}
	if err := a.externalOpenService().OpenPath(target.AbsolutePath); err != nil {
		return err.Error()
	}
	return ""
}

// OpenExternalURL opens an HTTP(S) URL through the platform browser opener.
// This deliberately bypasses OS file associations for InternetShortcut files.
func (a *App) OpenExternalURL(pluginID, rawURL string) string {
	if _, err := a.requirePluginAccess(pluginID, "files.openExternal"); err != nil {
		return err.Error()
	}
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "invalid HTTP(S) URL"
	}
	if err := a.externalOpenService().OpenPath(parsed.String()); err != nil {
		return err.Error()
	}
	return ""
}

// ShowVaultPathInFolder reveals a vault-relative file or folder in the OS file manager.
func (a *App) ShowVaultPathInFolder(pluginID, relativePath string) string {
	if _, err := a.requirePluginAccess(pluginID, "files.openExternal"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	target, err := a.files.ResolveExternalOpenTarget(relativePath)
	if err != nil {
		return err.Error()
	}
	isDir := target.Metadata.Type == corefiles.FileTypeFolder
	if err := a.externalOpenService().ShowInFolder(target.AbsolutePath, isDir); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) externalOpenService() externalOpenService {
	if a.externalOpen != nil {
		return a.externalOpen
	}
	return externalopen.NewService()
}

// recordFileSyncPaths records the sync consequences of a file operation this
// process just performed, given the vault-relative paths it touched.
//
// It used to take the entity type, id, operation and payload and ignore all
// four, rescanning the whole vault to rediscover what the caller already knew.
// The scanner still derives the operations from the filesystem — that is what
// keeps the snapshot and the operation log in agreement — but it is now told
// where to look.
func (a *App) recordFileSyncPaths(paths ...string) error {
	if a == nil {
		return nil
	}
	a.syncRunMu.Lock()
	defer a.syncRunMu.Unlock()
	_, err := a.recordSyncPathsLocked(paths)
	return err
}

func (a *App) publishFileActivity(eventName, pluginID, relativePath string, extra map[string]interface{}) {
	if a.eventBus == nil {
		return
	}
	path := strings.TrimSpace(filepath.ToSlash(relativePath))
	payload := map[string]interface{}{
		"path":              path,
		"title":             path,
		"workspaceRootPath": workspaceRootFromRelativePath(path),
		"pluginId":          pluginID,
	}
	for key, value := range extra {
		payload[key] = value
	}
	a.eventBus.Publish(events.Event{
		Name:      eventName,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   payload,
	})
}
