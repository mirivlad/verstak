package workspacetree

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ── Trash types ──────────────────────────────────────────────────────────────

// TrashEntry represents a trashed folder or workspace.
type TrashEntry struct {
	TrashID      string `json:"trashId"`
	EntityType   string `json:"entityType"` // "folder" | "workspace"
	EntityID     string `json:"entityId"`
	OriginalPath string `json:"originalPath"`
	DeletedAt    string `json:"deletedAt"`
}

type trashMetadata struct {
	EntityType   string   `json:"entityType"`
	EntityID     string   `json:"entityId"`
	OriginalPath string   `json:"originalPath"`
	DeletedAt    string   `json:"deletedAt"`
	WorkspaceIDs []string `json:"workspaceIds,omitempty"`
}

const treeTrashRelPath = ".verstak/trash/tree"

// TrashFolder moves a folder and all its contents to the internal trash.
func (s *Service) TrashFolder(folderID string, refreshBaseline func() error) (TrashEntry, error) {
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	f, ok := s.GetFolderByID(folderID)
	if !ok {
		return TrashEntry{}, fmt.Errorf("folder not found: %s", folderID)
	}

	return s.trashEntity(vaultDir, "folder", f.ID, f.Path, refreshBaseline)
}

// TrashWorkspace moves a workspace to the internal trash.
func (s *Service) TrashWorkspace(workspaceID string, refreshBaseline func() error) (TrashEntry, error) {
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	ws, ok := s.GetWorkspaceByID(workspaceID)
	if !ok {
		return TrashEntry{}, fmt.Errorf("workspace not found: %s", workspaceID)
	}

	// If this was the current workspace, clear it.
	s.mu.Lock()
	if s.currentWS == workspaceID {
		s.currentWS = ""
		if s.tree != nil {
			s.tree.CurrentWorkspaceID = ""
		}
	}
	s.mu.Unlock()

	return s.trashEntity(vaultDir, "workspace", ws.ID, ws.RootPath, refreshBaseline)
}

func (s *Service) trashEntity(vaultDir, entityType, entityID, relPath string, refreshBaseline func() error) (TrashEntry, error) {
	absPath := filepath.Join(vaultDir, filepath.FromSlash(relPath))
	if _, err := os.Lstat(absPath); err != nil {
		return TrashEntry{}, fmt.Errorf("entity path not found: %s", relPath)
	}

	trashID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	trashDir := filepath.Join(vaultDir, treeTrashRelPath, trashID)
	contentDir := filepath.Join(trashDir, "content")
	entityName := filepath.Base(filepath.FromSlash(relPath))
	trashContentTarget := filepath.Join(contentDir, entityName)

	s.BeginInternalMutation()

	if err := os.MkdirAll(contentDir, 0o755); err != nil {
		return TrashEntry{}, err
	}

	// Move entity into trash.
	if err := os.Rename(absPath, trashContentTarget); err != nil {
		return TrashEntry{}, err
	}

	// Collect workspace IDs if trashing a folder (for subtree information).
	var wsIDs []string
	if entityType == "folder" {
		s.mu.Lock()
		if s.scan != nil {
			for _, ws := range s.scan.Workspaces {
				if isPathPrefix(relPath, ws.RootPath) {
					wsIDs = append(wsIDs, ws.ID)
				}
			}
		}
		s.mu.Unlock()
	}

	meta := trashMetadata{
		EntityType:   entityType,
		EntityID:     entityID,
		OriginalPath: relPath,
		DeletedAt:    now,
		WorkspaceIDs: wsIDs,
	}
	metaData, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(trashDir, "metadata.json"), metaData, 0o644); err != nil {
		return TrashEntry{}, err
	}

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return TrashEntry{}, err
	}

	return TrashEntry{
		TrashID:      trashID,
		EntityType:   entityType,
		EntityID:     entityID,
		OriginalPath: relPath,
		DeletedAt:    now,
	}, nil
}

// ListTreeTrash returns all trashed folders and workspaces.
func (s *Service) ListTreeTrash() ([]TrashEntry, error) {
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	trashRoot := filepath.Join(vaultDir, treeTrashRelPath)
	entries, err := os.ReadDir(trashRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []TrashEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(trashRoot, entry.Name(), "metadata.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta trashMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		result = append(result, TrashEntry{
			TrashID:      entry.Name(),
			EntityType:   meta.EntityType,
			EntityID:     meta.EntityID,
			OriginalPath: meta.OriginalPath,
			DeletedAt:    meta.DeletedAt,
		})
	}
	return result, nil
}

// RestoreTreeTrash restores a trashed entity.
func (s *Service) RestoreTreeTrash(trashID, targetParentFolderID string, refreshBaseline func() error) (interface{}, error) {
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	trashDir := filepath.Join(vaultDir, treeTrashRelPath, trashID)
	metaPath := filepath.Join(trashDir, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("trash entry not found: %s", trashID)
	}
	var meta trashMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("invalid trash metadata: %w", err)
	}

	entityName := filepath.Base(filepath.FromSlash(meta.OriginalPath))
	contentDir := filepath.Join(trashDir, "content", entityName)

	if _, err := os.Lstat(contentDir); err != nil {
		return nil, fmt.Errorf("trash content missing: %s", trashID)
	}

	// Determine restore path.
	targetParentPath := ""
	if targetParentFolderID != "" {
		tf, ok := s.GetFolderByID(targetParentFolderID)
		if !ok {
			return nil, fmt.Errorf("target parent folder not found: %s", targetParentFolderID)
		}
		targetParentPath = tf.Path
	}

	targetRel := joinRelPath(targetParentPath, entityName)
	targetAbs := filepath.Join(vaultDir, filepath.FromSlash(targetRel))

	if _, err := os.Lstat(targetAbs); err == nil {
		return nil, fmt.Errorf("conflict: %s already exists", targetRel)
	}

	s.BeginInternalMutation()

	if err := os.Rename(contentDir, targetAbs); err != nil {
		return nil, err
	}

	// Remove trash directory.
	_ = os.RemoveAll(trashDir)

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return nil, err
	}

	if meta.EntityType == "workspace" {
		ws, _ := s.GetWorkspaceByID(meta.EntityID)
		return ws, nil
	}
	f, _ := s.GetFolderByID(meta.EntityID)
	return f, nil
}

// PurgeTreeTrash permanently removes a trash entry.
func (s *Service) PurgeTreeTrash(trashID string) error {
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	if trashID == "" || strings.Contains(trashID, "/") || strings.Contains(trashID, "\\") {
		return fmt.Errorf("invalid trash ID")
	}
	if _, err := uuid.Parse(trashID); err != nil {
		return fmt.Errorf("invalid trash ID: %w", err)
	}

	trashDir := filepath.Join(vaultDir, treeTrashRelPath, trashID)
	metaPath := filepath.Join(trashDir, "metadata.json")
	if _, err := os.Stat(metaPath); err != nil {
		return fmt.Errorf("trash entry not found: %s", trashID)
	}

	// Validate trashDir is within the vault's trash tree.
	expected := filepath.Join(vaultDir, treeTrashRelPath, trashID)
	if filepath.Clean(trashDir) != filepath.Clean(expected) {
		return fmt.Errorf("invalid trash path")
	}

	return os.RemoveAll(trashDir)
}
