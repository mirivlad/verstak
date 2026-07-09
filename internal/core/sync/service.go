package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

const (
	EntityNode    = "node"
	EntityNote    = "note"
	EntityFile    = "file"
	EntityFolder  = "folder"
	EntityAction  = "action"
	EntityWorklog = "worklog"
)

const (
	OpCreate = "create"
	OpUpdate = "update"
	OpDelete = "delete"
	OpMove   = "move"
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
	ServerURL   string `json:"server_url"`
	APIKey      string `json:"api_key"`
	DeviceID    string `json:"device_id"`
	LastPullSeq int    `json:"last_pull_seq"`
	LastSyncAt  string `json:"last_sync_at"`
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

	ops, err := s.loadOps()
	if err != nil {
		return err
	}
	ops = append(ops, op)
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
	return os.WriteFile(s.opsPath(), data, 0o644)
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
	return os.WriteFile(s.statePath(), data, 0o644)
}
