package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/pluginsync"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

var newSyncClient = syncsvc.NewClient

const defaultSyncPushBatchSize = 100

func (a *App) recordWorkspaceSyncOp(opType, workspaceID, path, previousPath, name string) error {
	if a.syncSvc == nil || a.treeV2 == nil {
		return nil
	}
	meta, err := a.treeV2.ReadDealMetadata(workspaceID, path)
	if err != nil && opType != syncsvc.OpTrash {
		return err
	}
	payload := syncWorkspacePayload{
		WorkspaceID:  workspaceID,
		Path:         path,
		PreviousPath: previousPath,
		Name:         name,
		Metadata:     meta,
	}
	if err := a.syncSvc.RecordOp(syncsvc.EntityWorkspace, workspaceID, opType, payload); err != nil {
		return err
	}
	_, err = a.syncSvc.RebaseSnapshot()
	return err
}

func (a *App) rebindSyncService() {
	if a == nil || a.vault == nil || a.vault.GetVaultStatus() != vault.StatusOpen {
		if a != nil {
			a.syncSvc = nil
		}
		return
	}
	a.syncSvc = syncsvc.NewService(a.vault.GetVaultPath(), "")
}

const snapshotScanDebounce = 300 * time.Millisecond

func (a *App) scheduleSnapshotScan() {
	if a == nil {
		return
	}
	a.syncTimerMu.Lock()
	if a.syncScanTimer != nil {
		a.syncScanTimer.Stop()
	}
	a.syncScanTimer = time.AfterFunc(snapshotScanDebounce, func() {
		if _, err := a.scanLocalChanges(); err != nil {
			log.Printf("[api] watcher sync snapshot scan failed: %v", err)
		}
	})
	a.syncTimerMu.Unlock()
}

func (a *App) stopScheduledSnapshotScan() {
	if a == nil {
		return
	}
	a.syncTimerMu.Lock()
	if a.syncScanTimer != nil {
		a.syncScanTimer.Stop()
		a.syncScanTimer = nil
	}
	a.syncTimerMu.Unlock()
}

func (a *App) scanLocalChanges() ([]string, error) {
	if a == nil {
		return nil, nil
	}
	a.syncRunMu.Lock()
	defer a.syncRunMu.Unlock()
	return a.scanLocalChangesLocked()
}

func (a *App) scanLocalChangesLocked() ([]string, error) {
	return a.recordSyncPathsLocked(nil)
}

// recordSyncPathsLocked records local changes, limited to the given
// vault-relative paths when the caller knows them and covering the whole vault
// when it does not.
func (a *App) recordSyncPathsLocked(paths []string) ([]string, error) {
	if a.syncSvc == nil {
		return nil, nil
	}
	warnings, err := a.syncSvc.ScanPathsAndRecord(paths)
	if err != nil {
		if a.appSettings != nil {
			_ = a.updateSyncError("snapshot scan: " + err.Error())
		}
		return nil, err
	}
	if err := a.syncSvc.SetLastWarning(strings.Join(warnings, "\n")); err != nil {
		return nil, err
	}
	return warnings, nil
}

func (a *App) requirePluginSyncAccess(pluginID string, remote bool) error {
	if _, err := a.requirePluginAccess(pluginID, "sync.participate"); err != nil {
		return err
	}
	if remote {
		if _, err := a.requirePluginAccess(pluginID, "network.remote"); err != nil {
			return err
		}
	}
	return nil
}

// SyncStatusDTO holds sync status information for the frontend.
type SyncStatusDTO struct {
	Configured   bool   `json:"configured"`
	Syncing      bool   `json:"syncing"`
	ServerURL    string `json:"serverUrl"`
	VaultID      string `json:"vaultId"`
	DeviceID     string `json:"deviceId"`
	DeviceName   string `json:"deviceName"`
	Connected    bool   `json:"connected"`
	Revoked      bool   `json:"revoked"`
	TokenStored  bool   `json:"tokenStored"`
	UnpushedOps  int    `json:"unpushedOps"`
	LastSyncAt   string `json:"lastSyncAt"`
	SyncInterval int    `json:"syncInterval"`
	LastError    string `json:"lastError"`
	LastWarning  string `json:"lastWarning"`
	StatusLabel  string `json:"statusLabel"`
}

func (a *App) syncStatus() (*SyncStatusDTO, error) {
	if a.vault == nil || a.vault.GetVaultStatus() != vault.StatusOpen {
		return &SyncStatusDTO{}, nil
	}

	vaultPath := a.vaultPath()
	if a.syncSvc == nil {
		return &SyncStatusDTO{}, nil
	}

	serverURL, apiKey, _, lastSyncAt, err := a.syncSvc.GetState()
	if err != nil {
		return &SyncStatusDTO{}, nil
	}

	cfg := a.appSettings.Get()
	deviceToken := syncsvc.LoadDeviceToken(vaultPath)
	remoteVaultID, _ := a.syncSvc.RemoteVaultID()
	lastWarning, _ := a.syncSvc.LastWarning()

	dto := &SyncStatusDTO{
		Configured:   serverURL != "" && (apiKey != "" || deviceToken != ""),
		Syncing:      a.syncRunning.Load(),
		ServerURL:    serverURL,
		VaultID:      remoteVaultID,
		LastSyncAt:   lastSyncAt,
		UnpushedOps:  0,
		TokenStored:  deviceToken != "",
		SyncInterval: cfg.Sync.SyncInterval,
		LastError:    cfg.Sync.LastError,
		LastWarning:  lastWarning,
	}

	if deviceID := a.syncSvc.GetDeviceID(); deviceID != "" {
		dto.DeviceID = deviceID
	} else if cfg.Sync.DeviceID != "" {
		dto.DeviceID = cfg.Sync.DeviceID
	}

	unpushed, _ := a.syncSvc.GetUnpushedOps()
	dto.UnpushedOps = len(unpushed)

	if deviceToken != "" {
		client := newSyncClient(serverURL, "", "", vaultPath)
		client.DeviceToken = deviceToken
		if dto.DeviceID != "" {
			client.DeviceID = dto.DeviceID
		}
		if info, err := client.GetMe(); err == nil {
			if info.DeviceID != "" {
				_ = a.syncSvc.SetDeviceID(info.DeviceID)
			}
			dto.DeviceName = info.DeviceName
			dto.DeviceID = info.DeviceID
			dto.Connected = true
			if info.RevokedAt != "" {
				dto.Revoked = true
				dto.Connected = false
			}
		}
	}

	switch {
	case dto.Revoked:
		dto.StatusLabel = "revoked"
	case dto.LastError != "":
		dto.StatusLabel = "error"
	case dto.Connected:
		dto.StatusLabel = "connected"
	case dto.Configured:
		dto.StatusLabel = "disconnected"
	case dto.ServerURL != "":
		dto.StatusLabel = "disconnected"
	default:
		dto.StatusLabel = "disabled"
	}

	if cfg.Sync.LastSyncAt != lastSyncAt || cfg.Sync.LastStatus != dto.StatusLabel {
		cfg.Sync.LastSyncAt = lastSyncAt
		cfg.Sync.LastStatus = dto.StatusLabel
		_ = a.appSettings.UpdateSync(cfg.Sync)
	}

	return dto, nil
}

// PluginSyncStatus returns sync status for plugins with sync permission.
func (a *App) PluginSyncStatus(pluginID string) (*SyncStatusDTO, string) {
	if err := a.requirePluginSyncAccess(pluginID, false); err != nil {
		return nil, err.Error()
	}
	dto, err := a.syncStatus()
	if err != nil {
		return nil, err.Error()
	}
	return dto, ""
}

func (a *App) syncConfigure(serverURL, username, password, remoteVaultID string) error {
	if err := a.requireVault(); err != nil {
		return err
	}
	meta := a.vault.GetVaultMeta()
	if meta == nil || strings.TrimSpace(meta.VaultID) == "" {
		return fmt.Errorf("vault ID is unavailable")
	}
	vaultPath := a.vaultPath()
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	targetVaultID := strings.TrimSpace(remoteVaultID)
	if targetVaultID == "" {
		targetVaultID = meta.VaultID
	}
	if a.syncSvc != nil {
		previousServerURL, _, _, _, stateErr := a.syncSvc.GetState()
		previousVaultID, _ := a.syncSvc.RemoteVaultID()
		if previousVaultID == "" {
			previousVaultID = meta.VaultID
		}
		if stateErr == nil && previousServerURL != "" && previousVaultID != targetVaultID {
			pending, err := a.syncSvc.GetUnpushedOps()
			if err != nil {
				return fmt.Errorf("read pending operations before changing remote vault: %w", err)
			}
			if len(pending) > 0 {
				return fmt.Errorf("cannot change remote vault with %d unpushed local operation(s); synchronize or resolve them first", len(pending))
			}
		}
	}
	client := newSyncClient(serverURL, "", "", vaultPath)
	deviceID, deviceToken, err := client.PairDevice(serverURL, username, password, hostname, "verstak-desktop/v2", targetVaultID)
	if err != nil {
		return fmt.Errorf("pair: %w", err)
	}
	if err := syncsvc.SaveDeviceToken(vaultPath, deviceToken); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	a.syncSvc = syncsvc.NewService(vaultPath, deviceID)
	if err := a.syncSvc.SetState(serverURL, ""); err != nil {
		return err
	}
	if err := a.syncSvc.SetRemoteVaultID(targetVaultID); err != nil {
		return err
	}
	if err := a.syncSvc.SetLastPullSeq(0); err != nil {
		return err
	}
	if err := a.syncSvc.SetBootstrapComplete(false); err != nil {
		return err
	}
	if err := a.syncSvc.SetLastWarning(""); err != nil {
		return err
	}

	cfg := a.appSettings.Get()
	cfg.Sync.Enabled = true
	cfg.Sync.ServerURL = serverURL
	cfg.Sync.DeviceID = deviceID
	cfg.Sync.DeviceName = hostname
	cfg.Sync.LastStatus = "connected"
	cfg.Sync.LastError = ""
	_ = a.appSettings.UpdateSync(cfg.Sync)

	return nil
}

// PluginSyncConfigure pairs the current vault with a sync server for a plugin.
func (a *App) PluginSyncConfigure(pluginID, serverURL, username, password, remoteVaultID string) string {
	if err := a.requirePluginSyncAccess(pluginID, true); err != nil {
		return err.Error()
	}
	if err := a.syncConfigure(serverURL, username, password, remoteVaultID); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncDisconnect() error {
	if err := a.requireVault(); err != nil {
		return err
	}
	vaultPath := a.vaultPath()
	deviceToken := syncsvc.LoadDeviceToken(vaultPath)
	cfg := a.appSettings.Get()
	serverURL := cfg.Sync.ServerURL
	if a.syncSvc != nil {
		if currentServerURL, _, _, _, err := a.syncSvc.GetState(); err == nil && currentServerURL != "" {
			serverURL = currentServerURL
		}
	}

	if deviceToken != "" && serverURL != "" {
		client := newSyncClient(serverURL, "", "", vaultPath)
		client.DeviceToken = deviceToken
		_ = client.RevokeCurrent()
	}
	_ = syncsvc.RemoveDeviceToken(vaultPath)

	cfg.Sync.Enabled = false
	cfg.Sync.ServerURL = ""
	cfg.Sync.DeviceID = ""
	cfg.Sync.DeviceName = ""
	cfg.Sync.LastStatus = "disabled"
	cfg.Sync.LastError = ""
	if err := a.appSettings.UpdateSync(cfg.Sync); err != nil {
		return err
	}
	if a.syncSvc == nil {
		return nil
	}
	if err := a.syncSvc.SetState("", ""); err != nil {
		return err
	}
	if err := a.syncSvc.SetRemoteVaultID(""); err != nil {
		return err
	}
	return a.syncSvc.SetLastWarning("")
}

// PluginSyncDisconnect disconnects sync for a plugin with sync permission.
func (a *App) PluginSyncDisconnect(pluginID string) string {
	if err := a.requirePluginSyncAccess(pluginID, false); err != nil {
		return err.Error()
	}
	if err := a.syncDisconnect(); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncTestConnection(serverURL, username, password string) error {
	vaultPath := a.vaultPath()
	if vaultPath == "" {
		vaultPath = "/tmp"
	}
	client := newSyncClient(serverURL, "", "", vaultPath)
	return client.TestAuth(serverURL, username, password)
}

// PluginSyncTestConnection tests sync server credentials for a plugin.
func (a *App) PluginSyncTestConnection(pluginID, serverURL, username, password string) string {
	if err := a.requirePluginSyncAccess(pluginID, true); err != nil {
		return err.Error()
	}
	if err := a.syncTestConnection(serverURL, username, password); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncSetInterval(minutes int) error {
	if err := a.requireVault(); err != nil {
		return err
	}
	cfg := a.appSettings.Get()
	cfg.Sync.SyncInterval = minutes
	if cfg.Sync.DeviceID == "" && a.syncSvc != nil {
		cfg.Sync.DeviceID = a.syncSvc.GetDeviceID()
	}
	return a.appSettings.UpdateSync(cfg.Sync)
}

// PluginSyncSetInterval sets the sync interval for a plugin with sync permission.
func (a *App) PluginSyncSetInterval(pluginID string, minutes int) string {
	if err := a.requirePluginSyncAccess(pluginID, false); err != nil {
		return err.Error()
	}
	if err := a.syncSetInterval(minutes); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncResetKey() error {
	if err := a.requireVault(); err != nil {
		return err
	}
	vaultPath := a.vaultPath()
	deviceToken := syncsvc.LoadDeviceToken(vaultPath)
	if err := syncsvc.RemoveDeviceToken(vaultPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	cfg := a.appSettings.Get()
	cfg.Sync.Enabled = false
	cfg.Sync.DeviceID = ""
	cfg.Sync.DeviceName = ""
	cfg.Sync.LastStatus = "disconnected"
	cfg.Sync.LastError = ""
	if a.syncSvc != nil {
		serverURL, _, _, _, err := a.syncSvc.GetState()
		if err != nil {
			return err
		}
		if serverURL == "" && deviceToken != "" {
			serverURL = cfg.Sync.ServerURL
		}
		if err := a.syncSvc.SetState(serverURL, ""); err != nil {
			return err
		}
	}
	return a.appSettings.UpdateSync(cfg.Sync)
}

// PluginSyncResetKey clears the stored sync device token for a plugin with sync permission.
func (a *App) PluginSyncResetKey(pluginID string) string {
	if err := a.requirePluginSyncAccess(pluginID, false); err != nil {
		return err.Error()
	}
	if err := a.syncResetKey(); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncNow() (map[string]interface{}, error) {
	a.syncRunMu.Lock()
	defer a.syncRunMu.Unlock()
	a.syncRunning.Store(true)
	defer a.syncRunning.Store(false)
	if err := a.requireVault(); err != nil {
		return nil, err
	}
	vaultPath := a.vaultPath()
	if a.syncSvc == nil {
		a.rebindSyncService()
	}
	if a.syncSvc == nil {
		return nil, fmt.Errorf("sync service not initialized")
	}
	if _, err := a.scanLocalChangesLocked(); err != nil {
		return nil, fmt.Errorf("snapshot scan: %w", err)
	}

	serverURL, apiKey, lastPullSeq, _, err := a.syncSvc.GetState()
	deviceToken := syncsvc.LoadDeviceToken(vaultPath)
	if err != nil || serverURL == "" || (apiKey == "" && deviceToken == "") {
		return nil, fmt.Errorf("sync not configured")
	}

	deviceID := a.syncSvc.GetDeviceID()
	cfg := a.appSettings.Get()
	if deviceID == "" && deviceToken == "" && cfg.Sync.DeviceID != "" {
		deviceID = cfg.Sync.DeviceID
	}

	client := newSyncClient(serverURL, apiKey, deviceID, vaultPath)
	client.DeviceToken = deviceToken
	if deviceID == "" && deviceToken != "" {
		info, err := client.GetMe()
		if err != nil {
			return nil, fmt.Errorf("sync identity: %w", err)
		}
		if info.DeviceID == "" {
			return nil, fmt.Errorf("sync identity: server returned an empty device ID")
		}
		if err := a.syncSvc.SetDeviceID(info.DeviceID); err != nil {
			return nil, fmt.Errorf("save sync identity: %w", err)
		}
		deviceID = info.DeviceID
		client.DeviceID = deviceID
	}

	bootstrapComplete, err := a.syncSvc.BootstrapComplete()
	if err != nil {
		return nil, fmt.Errorf("read bootstrap state: %w", err)
	}
	initialSnapshot, err := a.syncSvc.LoadSnapshot()
	if err != nil {
		return nil, fmt.Errorf("load initial snapshot: %w", err)
	}

	// Pull before publishing. This makes the first reconciliation safe: remote
	// state is applied or reported as a conflict before a pre-existing local
	// vault can enqueue its bootstrap creates.
	pulled, cursor, serverSequence, err := a.pullRemoteOps(client, lastPullSeq, !bootstrapComplete)
	if err != nil {
		return nil, err
	}
	if pulled > 0 {
		if warnings, err := a.syncSvc.RebaseSnapshot(); err != nil {
			return nil, fmt.Errorf("rebase remote snapshot: %w", err)
		} else if err := a.syncSvc.SetLastWarning(strings.Join(warnings, "\n")); err != nil {
			return nil, fmt.Errorf("save sync warning: %w", err)
		}
	}
	if !bootstrapComplete {
		if err := a.syncSvc.RecordBootstrapOps(initialSnapshot); err != nil {
			return nil, fmt.Errorf("record initial local snapshot: %w", err)
		}
		if err := a.syncSvc.SetBootstrapComplete(true); err != nil {
			return nil, fmt.Errorf("save bootstrap state: %w", err)
		}
	}

	unpushed, err := a.syncSvc.GetUnpushedOps()
	if err != nil {
		return nil, fmt.Errorf("get ops: %w", err)
	}
	for i := range unpushed {
		unpushed[i].LastSeenServerSeq = cursor
	}
	pushResult := &syncsvc.PushResponse{}
	if len(unpushed) > 0 {
		if err := a.uploadPendingBlobs(client, unpushed); err != nil {
			message := fmt.Sprintf("blob upload: %v", err)
			_ = a.updateSyncError(message)
			// A server-side file/quota limit is unresolved scanner input from the
			// user's perspective: retain it visibly and keep the operation pending
			// so a later sync retries rather than silently accepting the snapshot.
			_ = a.syncSvc.SetLastWarning(message)
			return nil, fmt.Errorf("blob upload: %w", err)
		}
		pushResult, err = a.pushPendingOps(client, unpushed)
		if err != nil {
			_ = a.updateSyncError(fmt.Sprintf("push: %v", err))
			return nil, fmt.Errorf("push: %w", err)
		}
	}

	// Pull once more so this device durably acknowledges its own accepted
	// operations and any concurrent remote operations without reapplying its own.
	pulledAfterPush, _, finalServerSequence, err := a.pullRemoteOps(client, cursor, false)
	if err != nil {
		return nil, err
	}
	if pulledAfterPush > 0 {
		if warnings, err := a.syncSvc.RebaseSnapshot(); err != nil {
			return nil, fmt.Errorf("rebase final remote snapshot: %w", err)
		} else if err := a.syncSvc.SetLastWarning(strings.Join(warnings, "\n")); err != nil {
			return nil, fmt.Errorf("save sync warning: %w", err)
		}
	}
	if finalServerSequence > serverSequence {
		serverSequence = finalServerSequence
	}
	if len(pushResult.Conflicts) > 0 {
		log.Printf("[sync] %d conflict(s) detected on push", len(pushResult.Conflicts))
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err := a.syncSvc.SetLastSyncAt(now); err != nil {
		return nil, fmt.Errorf("save sync time: %w", err)
	}
	_ = a.updateSyncSuccess(now)

	result := map[string]interface{}{
		"pushed":         len(pushResult.Accepted),
		"pulled":         pulled + pulledAfterPush,
		"serverSequence": serverSequence,
	}
	if len(pushResult.Conflicts) > 0 {
		result["conflicts"] = pushResult.Conflicts
	}
	return result, nil
}

func (a *App) pushPendingOps(client *syncsvc.Client, ops []syncsvc.Op) (*syncsvc.PushResponse, error) {
	result := &syncsvc.PushResponse{}
	if len(ops) == 0 {
		return result, nil
	}

	batchSize := min(defaultSyncPushBatchSize, len(ops))
	for offset := 0; offset < len(ops); {
		end := min(offset+batchSize, len(ops))
		batchResult, err := client.Push(ops[offset:end])
		if isTooManyOperations(err) && end-offset > 1 {
			batchSize = max(1, (end-offset)/2)
			continue
		}
		if err != nil {
			return result, err
		}
		if err := a.syncSvc.MarkPushed(batchResult.Accepted); err != nil {
			return result, fmt.Errorf("mark pushed: %w", err)
		}
		result.Accepted = append(result.Accepted, batchResult.Accepted...)
		result.Conflicts = append(result.Conflicts, batchResult.Conflicts...)
		result.Count += batchResult.Count
		offset = end
		if remaining := len(ops) - offset; remaining < batchSize {
			batchSize = remaining
		}
	}
	return result, nil
}

func isTooManyOperations(err error) bool {
	var serverErr *syncsvc.ServerError
	return errors.As(err, &serverErr) && serverErr.Code == "too_many_operations"
}

// uploadPendingBlobs ensures every operation referring to binary content has
// its immutable local cache uploaded before the operation reaches the log.
func (a *App) uploadPendingBlobs(client *syncsvc.Client, ops []syncsvc.Op) error {
	for _, op := range ops {
		if op.EntityType != syncsvc.EntityFile || (op.OpType != syncsvc.OpCreate && op.OpType != syncsvc.OpUpdate) {
			continue
		}
		payload, err := parseSyncFilePayload(op.PayloadJSON)
		if err != nil {
			return fmt.Errorf("parse operation %s: %w", op.OpID, err)
		}
		if payload.Blob == nil {
			continue
		}
		if payload.Blob.Size < 0 || payload.Blob.SHA256 == "" {
			return fmt.Errorf("invalid blob reference in operation %s", op.OpID)
		}
		cachePath := syncsvc.BlobCachePath(a.vaultPath(), payload.Blob.SHA256)
		ref, err := client.UploadBlob(cachePath)
		if err != nil {
			return fmt.Errorf("%s: %w", syncPayloadPath(op, payload), err)
		}
		if ref.SHA256 != payload.Blob.SHA256 || ref.Size != payload.Blob.Size {
			return fmt.Errorf("%s: uploaded blob does not match operation reference", syncPayloadPath(op, payload))
		}
	}
	return nil
}

func (a *App) pullRemoteOps(client *syncsvc.Client, cursor int, initialReconciliation bool) (pulled, nextCursor, serverSequence int, err error) {
	nextCursor = cursor
	for {
		pageStartCursor := nextCursor
		pullResult, pullErr := client.PullPage(nextCursor, 0)
		if pullErr != nil {
			_ = a.updateSyncError(fmt.Sprintf("pull: %v", pullErr))
			return pulled, nextCursor, serverSequence, fmt.Errorf("pull: %w", pullErr)
		}
		serverSequence = pullResult.ServerSequence
		lastSequenceInPage := nextCursor
		for _, op := range pullResult.Ops {
			if op.ServerSequence <= nextCursor {
				continue
			}
			if op.ServerSequence <= lastSequenceInPage {
				errMsg := fmt.Sprintf("pull response is not strictly ordered at sequence %d (%s)", op.ServerSequence, op.OpID)
				_ = a.updateSyncError(errMsg)
				return pulled, nextCursor, serverSequence, fmt.Errorf("%s", errMsg)
			}
			lastSequenceInPage = op.ServerSequence
			if err := a.applyRemoteOpWithClient(client, op, initialReconciliation); err != nil {
				path := syncOperationPath(op)
				errMsg := fmt.Sprintf("pull apply failed at sequence %d for %s %s (%s): %v", op.ServerSequence, op.EntityType, path, op.OpID, err)
				_ = a.updateSyncError(errMsg)
				return pulled, nextCursor, serverSequence, fmt.Errorf("%s", errMsg)
			}
			if err := a.syncSvc.RecordRemoteOp(op); err != nil {
				errMsg := fmt.Sprintf("record applied remote operation at sequence %d (%s): %v", op.ServerSequence, op.OpID, err)
				_ = a.updateSyncError(errMsg)
				return pulled, nextCursor, serverSequence, fmt.Errorf("%s", errMsg)
			}
			if err := a.syncSvc.SetLastPullSeq(op.ServerSequence); err != nil {
				errMsg := fmt.Sprintf("save pull cursor at sequence %d (%s): %v", op.ServerSequence, op.OpID, err)
				_ = a.updateSyncError(errMsg)
				return pulled, nextCursor, serverSequence, fmt.Errorf("%s", errMsg)
			}
			if warnings, err := a.syncSvc.RebaseSnapshot(); err != nil {
				errMsg := fmt.Sprintf("rebase snapshot after sequence %d (%s): %v", op.ServerSequence, op.OpID, err)
				_ = a.updateSyncError(errMsg)
				return pulled, nextCursor, serverSequence, fmt.Errorf("%s", errMsg)
			} else if err := a.syncSvc.SetLastWarning(strings.Join(warnings, "\n")); err != nil {
				return pulled, nextCursor, serverSequence, fmt.Errorf("save sync warning: %w", err)
			}
			nextCursor = op.ServerSequence
			pulled++
		}
		if pullResult.PageLastSequence != lastSequenceInPage {
			errMsg := fmt.Sprintf("pull page cursor mismatch: got %d, applied %d", pullResult.PageLastSequence, lastSequenceInPage)
			_ = a.updateSyncError(errMsg)
			return pulled, nextCursor, serverSequence, fmt.Errorf("%s", errMsg)
		}
		if !pullResult.HasMore {
			return pulled, nextCursor, serverSequence, nil
		}
		if pullResult.PageLastSequence <= pageStartCursor {
			errMsg := "pull page declared more operations without advancing cursor"
			_ = a.updateSyncError(errMsg)
			return pulled, nextCursor, serverSequence, fmt.Errorf("%s", errMsg)
		}
	}
}

func syncOperationPath(op syncsvc.Op) string {
	if op.EntityType == syncsvc.EntityWorkspaceTreeOrder {
		return ".verstak/workspace-tree/order.json"
	}
	if op.EntityType == syncsvc.EntityWorkspaceFolder {
		if payload, err := parseSyncFolderPayload(op.PayloadJSON); err == nil && payload.Path != "" {
			return payload.Path
		}
		return op.EntityID
	}
	if op.EntityType == syncsvc.EntityWorkspace {
		if payload, err := parseSyncWorkspacePayload(op.PayloadJSON); err == nil && payload.Path != "" {
			return payload.Path
		}
		return op.EntityID
	}
	payload, _ := parseSyncFilePayload(op.PayloadJSON)
	if path := syncPayloadPath(op, payload); path != "" {
		return path
	}
	return op.EntityID
}

// PluginSyncNow triggers sync for a plugin with sync permission.
func (a *App) PluginSyncNow(pluginID string) (map[string]interface{}, string) {
	if err := a.requirePluginSyncAccess(pluginID, true); err != nil {
		return nil, err.Error()
	}
	result, err := a.syncNow()
	if err != nil {
		_ = a.updateSyncError(err.Error())
		return nil, err.Error()
	}
	return result, ""
}

func (a *App) updateSyncError(errMsg string) error {
	cfg := a.appSettings.Get()
	cfg.Sync.LastError = errMsg
	cfg.Sync.LastStatus = "error"
	return a.appSettings.UpdateSync(cfg.Sync)
}

func (a *App) updateSyncSuccess(lastSyncAt string) error {
	cfg := a.appSettings.Get()
	cfg.Sync.LastError = ""
	cfg.Sync.LastStatus = "connected"
	cfg.Sync.LastSyncAt = lastSyncAt
	return a.appSettings.UpdateSync(cfg.Sync)
}

func (a *App) applyRemoteOp(op syncsvc.Op) error {
	return a.applyRemoteOpWithClient(nil, op, false)
}

func (a *App) applyRemoteOpForReconciliation(op syncsvc.Op, initialReconciliation bool) error {
	return a.applyRemoteOpWithClient(nil, op, initialReconciliation)
}

func (a *App) applyRemoteOpWithClient(client *syncsvc.Client, op syncsvc.Op, initialReconciliation bool) error {
	if a.debug {
		log.Printf("[sync] applyRemoteOp: type=%s entity=%s/%s", op.OpType, op.EntityType, op.EntityID)
	}
	if op.DeviceID != "" && op.DeviceID == a.localSyncDeviceID() {
		return nil
	}
	if op.EntityType == syncsvc.EntityWorkspaceFolder {
		payload, err := parseSyncFolderPayload(op.PayloadJSON)
		if err != nil {
			return err
		}
		if payload.FolderID != op.EntityID {
			return fmt.Errorf("workspace folder identity mismatch: entity %s payload %s", op.EntityID, payload.FolderID)
		}
		if a.treeV2 == nil {
			return fmt.Errorf("workspace tree service not initialized")
		}
		switch op.OpType {
		case syncsvc.OpMove, syncsvc.OpRename:
			return a.treeV2.ApplyPathFromSync(
				"folder:"+payload.FolderID,
				payload.PreviousPath,
				payload.Path,
				func() error {
					if a.fileWatcher != nil {
						return a.fileWatcher.RefreshBaseline()
					}
					return nil
				},
			)
		default:
			return fmt.Errorf("unsupported workspace folder sync op type: %s", op.OpType)
		}
	}
	if op.EntityType == syncsvc.EntityWorkspace {
		payload, err := parseSyncWorkspacePayload(op.PayloadJSON)
		if err != nil {
			return err
		}
		if a.treeV2 != nil && op.OpType == syncsvc.OpRename {
			if _, ok := a.treeV2.GetWorkspaceByID(payload.WorkspaceID); ok {
				if payload.WorkspaceID != op.EntityID {
					return fmt.Errorf("workspace identity mismatch: entity %s payload %s", op.EntityID, payload.WorkspaceID)
				}
				return a.treeV2.ApplyPathFromSync(
					"workspace:"+payload.WorkspaceID,
					payload.PreviousPath,
					payload.Path,
					func() error {
						if a.fileWatcher != nil {
							return a.fileWatcher.RefreshBaseline()
						}
						return nil
					},
				)
			}
		}
		return a.applyRemoteWorkspaceOp(op, payload)
	}
	if op.EntityType == syncsvc.EntityWorkspaceTreeOrder {
		if a.treeV2 == nil {
			return fmt.Errorf("workspace tree service not initialized")
		}
		if op.EntityID != syncsvc.WorkspaceTreeOrderEntityID {
			return fmt.Errorf("workspace tree order identity mismatch: %s", op.EntityID)
		}
		if op.OpType != syncsvc.OpUpdate {
			return fmt.Errorf("unsupported workspace tree order sync op type: %s", op.OpType)
		}
		state, err := workspacetree.ParseOrderState([]byte(op.PayloadJSON))
		if err != nil {
			return err
		}
		if err := a.treeV2.ApplyOrderState(state); err != nil {
			return err
		}
		if a.ctx != nil {
			emitFrontendEvent(a.ctx, "verstak:workspace-tree-changed")
		}
		return nil
	}
	if op.EntityType == pluginsync.EntityType {
		return a.applyRemotePluginRecordOp(op)
	}
	if a.files == nil {
		return fmt.Errorf("files service not initialized")
	}

	payload, err := parseSyncFilePayload(op.PayloadJSON)
	if err != nil {
		return err
	}
	switch op.EntityType {
	case syncsvc.EntityFile:
		return a.applyRemoteFileOp(client, op, payload, initialReconciliation)
	case syncsvc.EntityFolder:
		return a.applyRemoteFolderOp(op, payload, initialReconciliation)
	default:
		return fmt.Errorf("unsupported sync entity type: %s", op.EntityType)
	}
}

type syncFilePayload struct {
	Path        string                 `json:"path"`
	Content     string                 `json:"content"`
	DataBase64  *string                `json:"dataBase64"`
	Blob        *syncsvc.BlobReference `json:"blob"`
	ContentHash string                 `json:"contentHash"`
	FromPath    string                 `json:"fromPath"`
	ToPath      string                 `json:"toPath"`
}

type syncWorkspacePayload struct {
	WorkspaceID  string                     `json:"workspaceId"`
	Path         string                     `json:"path"`
	PreviousPath string                     `json:"previousPath,omitempty"`
	Name         string                     `json:"name"`
	Metadata     workspacetree.DealMetadata `json:"metadata"`
}

type syncFolderPayload struct {
	FolderID     string `json:"folderId"`
	Path         string `json:"path"`
	PreviousPath string `json:"previousPath,omitempty"`
}

func parseSyncFilePayload(payloadJSON string) (syncFilePayload, error) {
	if payloadJSON == "" {
		return syncFilePayload{}, nil
	}
	var payload syncFilePayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return syncFilePayload{}, fmt.Errorf("invalid sync payload: %w", err)
	}
	return payload, nil
}

func parseSyncWorkspacePayload(payloadJSON string) (syncWorkspacePayload, error) {
	if payloadJSON == "" {
		return syncWorkspacePayload{}, fmt.Errorf("workspace sync payload is empty")
	}
	var payload syncWorkspacePayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return syncWorkspacePayload{}, fmt.Errorf("invalid workspace sync payload: %w", err)
	}
	if payload.WorkspaceID == "" {
		return syncWorkspacePayload{}, fmt.Errorf("workspace sync payload is missing workspaceId")
	}
	if payload.Path == "" {
		payload.Path = payload.Name
	}
	return payload, nil
}

func parseSyncFolderPayload(payloadJSON string) (syncFolderPayload, error) {
	if payloadJSON == "" {
		return syncFolderPayload{}, fmt.Errorf("workspace folder sync payload is empty")
	}
	var payload syncFolderPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return syncFolderPayload{}, fmt.Errorf("invalid workspace folder sync payload: %w", err)
	}
	if payload.FolderID == "" {
		return syncFolderPayload{}, fmt.Errorf("workspace folder sync payload is missing folderId")
	}
	if payload.Path == "" {
		return syncFolderPayload{}, fmt.Errorf("workspace folder sync payload is missing path")
	}
	return payload, nil
}

func (a *App) applyRemoteWorkspaceOp(op syncsvc.Op, payload syncWorkspacePayload) error {
	if a.treeV2 == nil {
		return fmt.Errorf("workspace service not initialized")
	}
	if payload.WorkspaceID != op.EntityID {
		return fmt.Errorf("workspace identity mismatch: entity %s payload %s", op.EntityID, payload.WorkspaceID)
	}
	switch op.OpType {
	case syncsvc.OpCreate:
		_, err := a.treeV2.CreateWorkspaceFromSync(payload.Path, payload.WorkspaceID, payload.Metadata, a.refreshWorkspaceBaseline)
		return err
	case syncsvc.OpRename:
		return a.treeV2.ApplyPathFromSync("workspace:"+payload.WorkspaceID, payload.PreviousPath, payload.Path, a.refreshWorkspaceBaseline)
	case syncsvc.OpTrash:
		return a.treeV2.TrashWorkspaceFromSync(payload.WorkspaceID, payload.Path, a.refreshWorkspaceBaseline)
	case syncsvc.OpRestore:
		_, err := a.treeV2.RestoreWorkspaceFromSync(payload.WorkspaceID, payload.Path, a.refreshWorkspaceBaseline)
		return err
	default:
		return fmt.Errorf("unsupported workspace sync op type: %s", op.OpType)
	}
}

func (a *App) applyRemoteFileOp(client *syncsvc.Client, op syncsvc.Op, payload syncFilePayload, initialReconciliation bool) error {
	switch op.OpType {
	case syncsvc.OpCreate:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing file path")
		}
		matches, exists, err := a.remoteFileMatches(path, payload)
		if err != nil {
			return err
		}
		if matches {
			return nil
		}
		if exists {
			return fmt.Errorf("conflict: remote create would replace local file %s", path)
		}
		if payload.Blob != nil {
			return a.applyRemoteBlobFile(client, path, payload, corefiles.WriteOptions{CreateIfMissing: true})
		}
		if payload.DataBase64 != nil {
			return a.files.WriteVaultFileBytes(path, *payload.DataBase64, corefiles.WriteOptions{CreateIfMissing: true})
		}
		return a.files.WriteVaultTextFile(path, payload.Content, corefiles.WriteOptions{CreateIfMissing: true})
	case syncsvc.OpUpdate:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing file path")
		}
		matches, exists, err := a.remoteFileMatches(path, payload)
		if err != nil {
			return err
		}
		if matches {
			return nil
		}
		if initialReconciliation && exists {
			return fmt.Errorf("conflict: initial reconciliation would replace local file %s", path)
		}
		if pending, err := a.syncSvc.HasUnpushedPath(path); err != nil {
			return err
		} else if pending {
			return fmt.Errorf("conflict: remote update would replace unpushed local file %s", path)
		}
		if payload.Blob != nil {
			return a.applyRemoteBlobFile(client, path, payload, corefiles.WriteOptions{CreateIfMissing: !exists, Overwrite: exists})
		}
		if payload.DataBase64 != nil {
			return a.files.WriteVaultFileBytes(path, *payload.DataBase64, corefiles.WriteOptions{CreateIfMissing: !exists, Overwrite: exists})
		}
		return a.files.WriteVaultTextFile(path, payload.Content, corefiles.WriteOptions{CreateIfMissing: !exists, Overwrite: exists})
	case syncsvc.OpDelete:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing file path")
		}
		if initialReconciliation {
			exists, err := a.syncPathExists(path)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("conflict: initial reconciliation would delete local file %s", path)
			}
		}
		if pending, err := a.syncSvc.HasUnpushedPath(path); err != nil {
			return err
		} else if pending {
			return fmt.Errorf("conflict: remote delete would remove unpushed local file %s", path)
		}
		_, err := a.files.TrashVaultPath(path)
		if isSyncNotFound(err) {
			return nil
		}
		return err
	case syncsvc.OpMove:
		fromPath := payload.FromPath
		if fromPath == "" {
			fromPath = op.EntityID
		}
		if fromPath == "" || payload.ToPath == "" {
			return fmt.Errorf("missing file move path")
		}
		if initialReconciliation {
			fromExists, err := a.syncPathExists(fromPath)
			if err != nil {
				return err
			}
			toExists, err := a.syncPathExists(payload.ToPath)
			if err != nil {
				return err
			}
			if fromExists || toExists {
				return fmt.Errorf("conflict: initial reconciliation would move local file %s", fromPath)
			}
		}
		if pending, err := a.syncSvc.HasUnpushedPath(fromPath); err != nil {
			return err
		} else if pending {
			return fmt.Errorf("conflict: remote move would replace unpushed local file %s", fromPath)
		}
		err := a.files.MoveVaultPath(fromPath, payload.ToPath, corefiles.MoveOptions{})
		if isSyncNotFound(err) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unsupported file sync op type: %s", op.OpType)
	}
}

func (a *App) applyRemoteBlobFile(client *syncsvc.Client, path string, payload syncFilePayload, options corefiles.WriteOptions) error {
	if client == nil {
		return fmt.Errorf("blob operation requires an active sync client")
	}
	if payload.Blob == nil || payload.Blob.Size < 0 || payload.Blob.SHA256 == "" {
		return fmt.Errorf("invalid remote blob reference")
	}
	if payload.ContentHash != "" && payload.ContentHash != payload.Blob.SHA256 {
		return fmt.Errorf("remote blob hash does not match file content hash")
	}
	cachePath := syncsvc.BlobCachePath(a.vaultPath(), payload.Blob.SHA256)
	if err := client.DownloadBlobVerified(payload.Blob.SHA256, payload.Blob.Size, cachePath); err != nil {
		return fmt.Errorf("download blob: %w", err)
	}
	if err := a.files.WriteVaultFileFromPath(path, cachePath, options); err != nil {
		return err
	}
	return nil
}

func (a *App) remoteFileMatches(path string, payload syncFilePayload) (matches, exists bool, err error) {
	normalized, err := corefiles.NormalizeRelativeFile(path)
	if err != nil {
		return false, false, err
	}
	info, err := os.Lstat(filepath.Join(a.vaultPath(), filepath.FromSlash(normalized)))
	if err != nil {
		if os.IsNotExist(err) {
			return false, false, nil
		}
		return false, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, true, fmt.Errorf("conflict: remote file target is not a regular file: %s", normalized)
	}
	want, err := remotePayloadHash(payload)
	if err != nil {
		return false, true, err
	}
	file, err := os.Open(filepath.Join(a.vaultPath(), filepath.FromSlash(normalized)))
	if err != nil {
		return false, true, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, true, err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)) == want, true, nil
}

func remotePayloadHash(payload syncFilePayload) (string, error) {
	if payload.ContentHash != "" {
		return payload.ContentHash, nil
	}
	if payload.Blob != nil {
		return payload.Blob.SHA256, nil
	}
	data := []byte(payload.Content)
	if payload.DataBase64 != nil {
		decoded, err := base64.StdEncoding.DecodeString(*payload.DataBase64)
		if err != nil {
			return "", fmt.Errorf("invalid remote base64 payload: %w", err)
		}
		data = decoded
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:]), nil
}

func (a *App) applyRemoteFolderOp(op syncsvc.Op, payload syncFilePayload, initialReconciliation bool) error {
	switch op.OpType {
	case syncsvc.OpCreate:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing folder path")
		}
		err := a.files.CreateVaultFolder(path)
		if isSyncConflict(err) {
			return nil
		}
		return err
	case syncsvc.OpDelete:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing folder path")
		}
		if initialReconciliation {
			exists, err := a.syncPathExists(path)
			if err != nil {
				return err
			}
			if exists {
				return fmt.Errorf("conflict: initial reconciliation would delete local folder %s", path)
			}
		}
		_, err := a.files.TrashVaultPath(path)
		if isSyncNotFound(err) {
			return nil
		}
		return err
	case syncsvc.OpMove:
		fromPath := payload.FromPath
		if fromPath == "" {
			fromPath = op.EntityID
		}
		if fromPath == "" || payload.ToPath == "" {
			return fmt.Errorf("missing folder move path")
		}
		if initialReconciliation {
			fromExists, err := a.syncPathExists(fromPath)
			if err != nil {
				return err
			}
			toExists, err := a.syncPathExists(payload.ToPath)
			if err != nil {
				return err
			}
			if fromExists || toExists {
				return fmt.Errorf("conflict: initial reconciliation would move local folder %s", fromPath)
			}
		}
		err := a.files.MoveVaultPath(fromPath, payload.ToPath, corefiles.MoveOptions{})
		if isSyncNotFound(err) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unsupported folder sync op type: %s", op.OpType)
	}
}

func (a *App) syncPathExists(path string) (bool, error) {
	normalized, err := corefiles.NormalizeRelativeFile(path)
	if err != nil {
		return false, err
	}
	_, err = os.Lstat(filepath.Join(a.vaultPath(), filepath.FromSlash(normalized)))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func syncPayloadPath(op syncsvc.Op, payload syncFilePayload) string {
	if payload.Path != "" {
		return payload.Path
	}
	return op.EntityID
}

func (a *App) localSyncDeviceID() string {
	if a.syncSvc != nil {
		if deviceID := a.syncSvc.GetDeviceID(); deviceID != "" {
			return deviceID
		}
	}
	if a.vault != nil && a.vault.GetVaultStatus() == vault.StatusOpen && syncsvc.LoadDeviceToken(a.vaultPath()) != "" {
		return ""
	}
	if a.appSettings != nil {
		if deviceID := a.appSettings.Get().Sync.DeviceID; deviceID != "" {
			return deviceID
		}
	}
	return ""
}

func isSyncNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not-found")
}

func isSyncConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "conflict")
}
