package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	EntityNode              = "node"
	EntityNote              = "note"
	EntityFile              = "file"
	EntityFolder            = "folder"
	EntityWorkspaceFolder   = "workspace-folder"
	EntityWorkspace         = "workspace"
	EntityAction            = "action"
	EntityWorklog           = "worklog"
)

const (
	OpCreate     = "create"
	OpUpdate     = "update"
	OpDelete     = "delete"
	OpMove       = "move"
	OpRename     = "rename"
	OpTrash      = "trash"
	OpRestore    = "restore"
	OpHardDelete = "hard-delete"
)

// Op represents a sync operation.
type Op struct {
	ID                string  `json:"id"`
	OpID              string  `json:"op_id"`
	ServerSequence    int     `json:"server_sequence,omitempty"`
	DeviceID          string  `json:"device_id,omitempty"`
	EntityType        string  `json:"entity_type"`
	EntityID          string  `json:"entity_id"`
	OpType            string  `json:"op_type"`
	PayloadJSON       string  `json:"payload_json"`
	CreatedAt         string  `json:"created_at"`
	PushedAt          *string `json:"pushed_at,omitempty"`
	AppliedAt         *string `json:"applied_at,omitempty"`
	ClientSequence    int     `json:"client_sequence,omitempty"`
	LastSeenServerSeq int     `json:"last_seen_server_seq,omitempty"`
}

// syncState persists connection state to JSON file.
type syncState struct {
	ServerURL         string `json:"server_url"`
	APIKey            string `json:"api_key"`
	DeviceID          string `json:"device_id"`
	LastPullSeq       int    `json:"last_pull_seq"`
	LastSyncAt        string `json:"last_sync_at"`
	BootstrapComplete bool   `json:"bootstrap_complete"`
	LastWarning       string `json:"last_warning"`
	RemoteVaultID     string `json:"remote_vault_id"`
}

// Service records and manages sync operations using JSON file storage.
type Service struct {
	vaultRoot string
	deviceID  string
}

// NewService creates a sync service.
func NewService(vaultRoot, deviceID string) *Service {
	service := &Service{vaultRoot: vaultRoot, deviceID: deviceID}
	if deviceID == "" {
		if state, err := service.loadState(); err == nil {
			service.deviceID = state.DeviceID
		}
	}
	return service
}

func (s *Service) syncDir() string {
	return filepath.Join(s.vaultRoot, ".verstak", "sync")
}

func (s *Service) opsPath() string {
	return filepath.Join(s.syncDir(), "ops.json")
}

func (s *Service) statePath() string {
	return filepath.Join(s.syncDir(), "state.json")
}

func (s *Service) snapshotPath() string {
	return filepath.Join(s.syncDir(), "snapshot.json")
}

func (s *Service) scanJournalPath() string {
	return filepath.Join(s.syncDir(), "scan-journal.json")
}

func (s *Service) ensureDir() error {
	return os.MkdirAll(s.syncDir(), 0o755)
}

// RecordOp writes a sync operation to the local ops file.
func (s *Service) RecordOp(entityType, entityID, opType string, payload interface{}) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	var payloadStr string
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		payloadStr = string(b)
	}

	op := Op{
		ID:          id,
		OpID:        id,
		DeviceID:    s.deviceID,
		EntityType:  entityType,
		EntityID:    entityID,
		OpType:      opType,
		PayloadJSON: payloadStr,
		CreatedAt:   now,
	}

	return s.recordOps([]Op{op})
}

// recordOps is idempotent by op ID so a scanner recovery journal can safely
// resume after a crash between recording operations and replacing its snapshot.
func (s *Service) recordOps(newOps []Op) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	ops, err := s.loadOps()
	if err != nil {
		return err
	}
	existing := make(map[string]bool, len(ops))
	for _, op := range ops {
		existing[op.OpID] = true
	}
	for _, op := range newOps {
		if op.OpID == "" || existing[op.OpID] {
			continue
		}
		if op.ID == "" {
			op.ID = op.OpID
		}
		ops = append(ops, op)
		existing[op.OpID] = true
	}
	return s.saveOps(ops)
}

// RecordRemoteOp writes a remote op to the local ops file.
func (s *Service) RecordRemoteOp(op Op) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)

	ops, err := s.loadOps()
	if err != nil {
		return err
	}
	remoteID := op.OpID + "-remote"
	for _, existing := range ops {
		if existing.ID == remoteID {
			return nil
		}
	}
	op.ID = remoteID
	op.PushedAt = &now
	op.AppliedAt = &now
	ops = append(ops, op)
	return s.saveOps(ops)
}

// GetUnpushedOps returns ops that have not been pushed yet.
func (s *Service) GetUnpushedOps() ([]Op, error) {
	ops, err := s.loadOps()
	if err != nil {
		return nil, err
	}
	var unpushed []Op
	for _, op := range ops {
		if op.PushedAt == nil {
			unpushed = append(unpushed, op)
		}
	}
	return unpushed, nil
}

// HasUnpushedPath reports whether a local operation still owns a path (or one
// of its descendants). Pull uses it to turn an incoming overwrite/delete into
// a visible conflict instead of silently replacing a local external edit.
func (s *Service) HasUnpushedPath(path string) (bool, error) {
	ops, err := s.GetUnpushedOps()
	if err != nil {
		return false, err
	}
	for _, op := range ops {
		if syncPathsOverlap(path, op.EntityID) {
			return true, nil
		}
		var payload struct {
			Path     string `json:"path"`
			FromPath string `json:"fromPath"`
			ToPath   string `json:"toPath"`
		}
		if op.PayloadJSON == "" || json.Unmarshal([]byte(op.PayloadJSON), &payload) != nil {
			continue
		}
		if syncPathsOverlap(path, payload.Path) || syncPathsOverlap(path, payload.FromPath) || syncPathsOverlap(path, payload.ToPath) {
			return true, nil
		}
	}
	return false, nil
}

func syncPathsOverlap(left, right string) bool {
	left = strings.Trim(left, "/")
	right = strings.Trim(right, "/")
	if left == "" || right == "" {
		return false
	}
	return left == right || strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}

// MarkPushed marks ops as pushed to server.
func (s *Service) MarkPushed(opIDs []string) error {
	ops, err := s.loadOps()
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	idSet := make(map[string]bool, len(opIDs))
	for _, id := range opIDs {
		idSet[id] = true
	}
	for i := range ops {
		if idSet[ops[i].OpID] {
			ops[i].PushedAt = &now
		}
	}
	return s.saveOps(ops)
}

// MarkApplied marks remote ops as applied locally.
func (s *Service) MarkApplied(opIDs []string) error {
	ops, err := s.loadOps()
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	idSet := make(map[string]bool, len(opIDs))
	for _, id := range opIDs {
		idSet[id] = true
	}
	for i := range ops {
		if idSet[ops[i].OpID] {
			ops[i].AppliedAt = &now
		}
	}
	return s.saveOps(ops)
}

// GetState returns the current sync state.
func (s *Service) GetState() (serverURL, apiKey string, lastPullSeq int, lastSyncAt string, err error) {
	st, err := s.loadState()
	if err != nil {
		return "", "", 0, "", err
	}
	return st.ServerURL, st.APIKey, st.LastPullSeq, st.LastSyncAt, nil
}

// SetState saves sync connection state.
func (s *Service) SetState(serverURL, apiKey string) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	st, err := s.loadState()
	if err != nil {
		st = &syncState{}
	}
	st.ServerURL = serverURL
	st.APIKey = apiKey
	if s.deviceID != "" {
		st.DeviceID = s.deviceID
	}
	return s.saveState(st)
}

// SetLastPullSeq updates the last pulled server sequence.
func (s *Service) SetLastPullSeq(seq int) error {
	st, err := s.loadState()
	if err != nil {
		return err
	}
	st.LastPullSeq = seq
	return s.saveState(st)
}

// SetLastSyncAt updates the last sync timestamp.
func (s *Service) SetLastSyncAt(t string) error {
	st, err := s.loadState()
	if err != nil {
		return err
	}
	st.LastSyncAt = t
	return s.saveState(st)
}

// BootstrapComplete reports whether the initial pull/reconcile/bootstrap cycle
// finished successfully for this vault connection.
func (s *Service) BootstrapComplete() (bool, error) {
	st, err := s.loadState()
	if err != nil {
		return false, err
	}
	return st.BootstrapComplete, nil
}

// SetBootstrapComplete marks the initial reconciliation as complete only after
// all remote operations were applied and the local initial snapshot was queued.
func (s *Service) SetBootstrapComplete(done bool) error {
	st, err := s.loadState()
	if err != nil {
		return err
	}
	st.BootstrapComplete = done
	return s.saveState(st)
}

// LastWarning returns the persistent scanner warning shown by sync status.
func (s *Service) LastWarning() (string, error) {
	st, err := s.loadState()
	if err != nil {
		return "", err
	}
	return st.LastWarning, nil
}

// SetLastWarning persists an unresolved scanner condition. An empty string
// clears the warning once a later complete scan no longer reports it.
func (s *Service) SetLastWarning(message string) error {
	st, err := s.loadState()
	if err != nil {
		return err
	}
	st.LastWarning = message
	return s.saveState(st)
}

// RemoteVaultID returns the optional target vault chosen while pairing a new
// local vault for restore. Empty means this vault's own durable ID was used.
func (s *Service) RemoteVaultID() (string, error) {
	st, err := s.loadState()
	if err != nil {
		return "", err
	}
	return st.RemoteVaultID, nil
}

func (s *Service) SetRemoteVaultID(vaultID string) error {
	st, err := s.loadState()
	if err != nil {
		return err
	}
	st.RemoteVaultID = vaultID
	return s.saveState(st)
}

// GetDeviceID returns the device ID used by this service.
func (s *Service) GetDeviceID() string {
	return s.deviceID
}

// SetDeviceID persists the device ID for this vault's sync state.
func (s *Service) SetDeviceID(deviceID string) error {
	if err := s.ensureDir(); err != nil {
		return err
	}
	st, err := s.loadState()
	if err != nil {
		st = &syncState{}
	}
	st.DeviceID = deviceID
	if err := s.saveState(st); err != nil {
		return err
	}
	s.deviceID = deviceID
	return nil
}

// --- file helpers ---

func (s *Service) loadOps() ([]Op, error) {
	data, err := os.ReadFile(s.opsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read ops: %w", err)
	}
	var ops []Op
	if err := json.Unmarshal(data, &ops); err != nil {
		return nil, fmt.Errorf("parse ops: %w", err)
	}
	return ops, nil
}

func (s *Service) saveOps(ops []Op) error {
	data, err := json.MarshalIndent(ops, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ops: %w", err)
	}
	return atomicWriteFile(s.opsPath(), data, 0o600)
}

func (s *Service) loadState() (*syncState, error) {
	data, err := os.ReadFile(s.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &syncState{}, nil
		}
		return nil, fmt.Errorf("read state: %w", err)
	}
	var st syncState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	return &st, nil
}

func (s *Service) saveState(st *syncState) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return atomicWriteFile(s.statePath(), data, 0o600)
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".verstak-sync-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
