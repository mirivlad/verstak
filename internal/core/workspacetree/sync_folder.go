package workspacetree

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/google/uuid"
)

// CreateFolderFromSync applies a remote organizational-folder creation with
// its stable identity. It must never replace a local folder of the same name
// with a different UUID, nor adopt an unmarked directory without consent.
func (s *Service) CreateFolderFromSync(path, folderID string, refreshBaseline func() error) (ScannedFolder, error) {
	cleaned, err := validateSyncWorkspacePath(path)
	if err != nil {
		return ScannedFolder{}, err
	}
	if _, err := uuid.Parse(folderID); err != nil {
		return ScannedFolder{}, fmt.Errorf("invalid folder identity: %w", err)
	}
	if existing, ok := s.GetFolderByID(folderID); ok {
		if existing.Path == cleaned {
			return existing, nil
		}
		return ScannedFolder{}, fmt.Errorf("conflict: folder %s already exists at %s", folderID, existing.Path)
	}
	parent := parentPath(cleaned)
	if parent != "" {
		s.mu.Lock()
		parentFound := false
		if s.scan != nil {
			for _, folder := range s.scan.Folders {
				if folder.Path == parent {
					parentFound = true
					break
				}
			}
		}
		s.mu.Unlock()
		if !parentFound {
			return ScannedFolder{}, fmt.Errorf("remote folder parent does not exist: %s", parent)
		}
	}
	root := s.metadataVaultDir()
	targetAbs, err := ResolveInsideVault(root, cleaned)
	if err != nil {
		return ScannedFolder{}, err
	}
	if _, err := os.Lstat(targetAbs); err == nil {
		return ScannedFolder{}, fmt.Errorf("conflict: %s already exists", cleaned)
	} else if !os.IsNotExist(err) {
		return ScannedFolder{}, err
	}
	stagingAbs := targetAbs + ".sync-staging." + uuid.NewString()[:8]
	if err := os.MkdirAll(stagingAbs, 0o755); err != nil {
		return ScannedFolder{}, err
	}
	defer os.RemoveAll(stagingAbs)
	if err := WriteFolderMarker(stagingAbs, folderID); err != nil {
		return ScannedFolder{}, err
	}
	s.BeginInternalMutation()
	if err := os.Rename(stagingAbs, targetAbs); err != nil {
		// EndInternalMutationAndRefreshBaseline must not run after a failed
		// rename; it would scan an incomplete operation.
		atomic.AddInt32(&s.internalMutations, -1)
		return ScannedFolder{}, err
	}
	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedFolder{}, err
	}
	created, ok := s.GetFolderByID(folderID)
	if !ok || created.Path != cleaned {
		return ScannedFolder{}, fmt.Errorf("remote folder %s missing after sync create", folderID)
	}
	return created, nil
}
