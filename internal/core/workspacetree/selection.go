package workspacetree

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const workspaceUIStateSchemaVersion = 2

type workspaceUIState struct {
	SchemaVersion      int    `json:"schemaVersion"`
	CurrentWorkspaceID string `json:"currentWorkspaceId,omitempty"`
	UpdatedAt          string `json:"updatedAt"`
}

type legacyWorkspaceUIState struct {
	SelectedWorkspace string `json:"selectedWorkspace"`
}

func (s *Service) workspaceUIStatePath() string {
	return filepath.Join(s.metadataVaultDir(), ".verstak", "workspace-ui.json")
}

func (s *Service) restoreCurrentWorkspace() error {
	data, err := os.ReadFile(s.workspaceUIStatePath())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read workspace UI state: %w", err)
	}

	var envelope struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil
	}

	workspaceID := ""
	migrateLegacy := false
	if envelope.SchemaVersion == workspaceUIStateSchemaVersion {
		var state workspaceUIState
		if err := json.Unmarshal(data, &state); err != nil {
			return nil
		}
		workspaceID = strings.TrimSpace(state.CurrentWorkspaceID)
	} else if envelope.SchemaVersion == 0 {
		var legacy legacyWorkspaceUIState
		if err := json.Unmarshal(data, &legacy); err != nil {
			return nil
		}
		workspaceID = s.resolveLegacyWorkspaceSelection(legacy.SelectedWorkspace)
		migrateLegacy = workspaceID != ""
	} else {
		return nil
	}

	if workspaceID != "" {
		if _, err := uuid.Parse(workspaceID); err != nil {
			return nil
		}
		if _, ok := s.GetWorkspaceByID(workspaceID); !ok {
			return nil
		}
	}

	s.mu.Lock()
	s.currentWS = workspaceID
	if s.tree != nil {
		s.tree.CurrentWorkspaceID = workspaceID
	}
	s.mu.Unlock()

	if migrateLegacy {
		return s.writeWorkspaceUIState(workspaceID)
	}
	return nil
}

func (s *Service) resolveLegacyWorkspaceSelection(selection string) string {
	selection = strings.Trim(strings.TrimSpace(filepath.ToSlash(selection)), "/")
	if selection == "" {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scan == nil {
		return ""
	}
	if _, ok := s.scan.Workspaces[selection]; ok {
		return selection
	}
	nameMatch := ""
	for id, workspace := range s.scan.Workspaces {
		if workspace.RootPath == selection {
			return id
		}
		if workspace.Name == selection {
			if nameMatch != "" {
				nameMatch = ""
				break
			}
			nameMatch = id
		}
	}
	return nameMatch
}

func (s *Service) writeWorkspaceUIState(workspaceID string) error {
	state := workspaceUIState{
		SchemaVersion:      workspaceUIStateSchemaVersion,
		CurrentWorkspaceID: workspaceID,
		UpdatedAt:          time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal workspace UI state: %w", err)
	}
	path := s.workspaceUIStatePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create workspace UI state directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".workspace-ui-*.tmp")
	if err != nil {
		return fmt.Errorf("create workspace UI state temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set workspace UI state permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write workspace UI state: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync workspace UI state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close workspace UI state: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("publish workspace UI state: %w", err)
	}
	return nil
}
