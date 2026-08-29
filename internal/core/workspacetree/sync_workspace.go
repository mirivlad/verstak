package workspacetree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
)

// CreateWorkspaceFromSync materializes a remote Deal with its durable UUID.
// Replaying the same identity/path pair is a no-op.
func (s *Service) CreateWorkspaceFromSync(rootPath, workspaceID string, metadata DealMetadata, refreshBaseline func() error) (ScannedWorkspace, error) {
	cleaned, err := validateSyncWorkspacePath(rootPath)
	if err != nil {
		return ScannedWorkspace{}, err
	}
	if _, err := uuid.Parse(workspaceID); err != nil {
		return ScannedWorkspace{}, fmt.Errorf("invalid workspace identity: %w", err)
	}
	if existing, ok := s.GetWorkspaceByID(workspaceID); ok {
		if existing.RootPath == cleaned {
			return existing, nil
		}
		return ScannedWorkspace{}, fmt.Errorf("conflict: workspace %s already exists at %s", workspaceID, existing.RootPath)
	}
	for _, existing := range s.ListWorkspaces() {
		if existing.RootPath == cleaned {
			return ScannedWorkspace{}, fmt.Errorf("conflict: %s already belongs to workspace %s", cleaned, existing.ID)
		}
	}

	parentPath := parentPath(cleaned)
	if err := s.ensureSyncParentFolders(parentPath); err != nil {
		return ScannedWorkspace{}, err
	}
	vaultDir := s.metadataVaultDir()
	targetAbs, err := ResolveInsideVault(vaultDir, cleaned)
	if err != nil {
		return ScannedWorkspace{}, err
	}
	if _, err := os.Lstat(targetAbs); err == nil {
		return ScannedWorkspace{}, fmt.Errorf("conflict: %s already exists", cleaned)
	} else if !os.IsNotExist(err) {
		return ScannedWorkspace{}, err
	}

	stagingAbs := targetAbs + ".sync-staging." + uuid.NewString()[:8]
	if err := os.MkdirAll(stagingAbs, 0o755); err != nil {
		return ScannedWorkspace{}, err
	}
	defer os.RemoveAll(stagingAbs)
	if err := WriteWorkspaceMarker(stagingAbs, workspaceID); err != nil {
		return ScannedWorkspace{}, err
	}
	if err := applyWorkspaceToolFolders(stagingAbs, metadata.WorkspaceTools); err != nil {
		return ScannedWorkspace{}, err
	}
	metadata.WorkspaceID = workspaceID
	metadata.WorkspaceName = filepath.Base(filepath.FromSlash(cleaned))
	metadata.UpdatedAt = ""
	if err := s.WriteDealMetadata(metadata); err != nil {
		return ScannedWorkspace{}, err
	}
	metadataPath := canonicalDealMetadataPath(vaultDir, workspaceID)
	published := false
	defer func() {
		if !published {
			_ = os.Remove(metadataPath)
		}
	}()

	s.BeginInternalMutation()
	if err := os.Rename(stagingAbs, targetAbs); err != nil {
		atomic.AddInt32(&s.internalMutations, -1)
		return ScannedWorkspace{}, err
	}
	published = true
	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedWorkspace{}, err
	}
	created, ok := s.GetWorkspaceByID(workspaceID)
	if !ok {
		return ScannedWorkspace{}, fmt.Errorf("workspace %s missing after sync create", workspaceID)
	}
	return created, nil
}

// TrashWorkspaceFromSync idempotently moves a remote Deal into local tree trash.
func (s *Service) TrashWorkspaceFromSync(workspaceID, expectedPath string, refreshBaseline func() error) error {
	if active, ok := s.GetWorkspaceByID(workspaceID); ok {
		if expectedPath != "" && active.RootPath != filepath.ToSlash(expectedPath) {
			return fmt.Errorf("conflict: workspace %s is at %s, expected %s", workspaceID, active.RootPath, expectedPath)
		}
		_, err := s.TrashWorkspace(workspaceID, refreshBaseline)
		return err
	}
	entries, err := s.ListTreeTrash()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.EntityType == "workspace" && entry.EntityID == workspaceID {
			return nil
		}
	}
	return fmt.Errorf("workspace not found: %s", workspaceID)
}

// RestoreWorkspaceFromSync idempotently restores a Deal by durable UUID and
// then places it at the remote canonical path.
func (s *Service) RestoreWorkspaceFromSync(workspaceID, targetPath string, refreshBaseline func() error) (ScannedWorkspace, error) {
	cleaned, err := validateSyncWorkspacePath(targetPath)
	if err != nil {
		return ScannedWorkspace{}, err
	}
	if active, ok := s.GetWorkspaceByID(workspaceID); ok {
		if active.RootPath == cleaned {
			return active, nil
		}
		if err := s.ensureSyncParentFolders(parentPath(cleaned)); err != nil {
			return ScannedWorkspace{}, err
		}
		if err := s.ApplyPathFromSync("workspace:"+workspaceID, active.RootPath, cleaned, refreshBaseline); err != nil {
			return ScannedWorkspace{}, err
		}
		active, _ = s.GetWorkspaceByID(workspaceID)
		return active, nil
	}

	entries, err := s.ListTreeTrash()
	if err != nil {
		return ScannedWorkspace{}, err
	}
	trashID := ""
	for _, entry := range entries {
		if entry.EntityType == "workspace" && entry.EntityID == workspaceID {
			trashID = entry.TrashID
			break
		}
	}
	if trashID == "" {
		return ScannedWorkspace{}, fmt.Errorf("workspace not found: %s", workspaceID)
	}
	value, err := s.RestoreTreeTrash(trashID, "", refreshBaseline)
	if err != nil {
		return ScannedWorkspace{}, err
	}
	restored, ok := value.(ScannedWorkspace)
	if !ok {
		return ScannedWorkspace{}, fmt.Errorf("trash entry is not a workspace")
	}
	if restored.RootPath == cleaned {
		return restored, nil
	}
	if err := s.ensureSyncParentFolders(parentPath(cleaned)); err != nil {
		return ScannedWorkspace{}, err
	}
	if err := s.ApplyPathFromSync("workspace:"+workspaceID, restored.RootPath, cleaned, refreshBaseline); err != nil {
		return ScannedWorkspace{}, err
	}
	restored, _ = s.GetWorkspaceByID(workspaceID)
	return restored, nil
}

func (s *Service) ensureSyncParentFolders(path string) error {
	if path == "" {
		return nil
	}
	abs, err := ResolveInsideVault(s.metadataVaultDir(), path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	return s.fullReconcile()
}

func validateSyncWorkspacePath(path string) (string, error) {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return "", fmt.Errorf("workspace path is empty")
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if cleaned != path {
		return "", fmt.Errorf("workspace path is not normalized: %s", path)
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if err := validateEntityName(segment); err != nil {
			return "", fmt.Errorf("invalid workspace path %s: %w", path, err)
		}
	}
	return cleaned, nil
}
