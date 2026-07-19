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

// ── Create Folder ────────────────────────────────────────────────────────────

// CreateFolder creates a new organizational folder under parentFolderID (empty = root).
func (s *Service) CreateFolder(parentFolderID, name string, refreshBaseline func() error) (ScannedFolder, error) {
	name = strings.TrimSpace(name)
	if err := validateEntityName(name); err != nil {
		return ScannedFolder{}, err
	}

	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	// Resolve parent path.
	parentPath := ""
	if parentFolderID != "" {
		f, ok := s.GetFolderByID(parentFolderID)
		if !ok {
			return ScannedFolder{}, fmt.Errorf("parent folder not found: %s", parentFolderID)
		}
		parentPath = f.Path
	}

	childRel := joinRelPath(parentPath, name)
	childAbs := filepath.Join(vaultDir, filepath.FromSlash(childRel))

	// Check collision.
	if _, err := os.Lstat(childAbs); err == nil {
		return ScannedFolder{}, fmt.Errorf("conflict: %s already exists", childRel)
	}

	// Create staging directory in parent.
	stagingName := "." + name + ".staging." + uuid.NewString()[:8]
	stagingRel := joinRelPath(parentPath, stagingName)
	stagingAbs := filepath.Join(vaultDir, filepath.FromSlash(stagingRel))

	s.BeginInternalMutation()
	defer func() {
		// Cleanup staging on failure.
		if _, err := os.Lstat(stagingAbs); err == nil {
			_ = os.RemoveAll(stagingAbs)
		}
	}()

	if err := os.MkdirAll(stagingAbs, 0o755); err != nil {
		return ScannedFolder{}, err
	}

	newID := uuid.NewString()
	if err := WriteFolderMarker(stagingAbs, newID); err != nil {
		return ScannedFolder{}, err
	}

	// Atomic rename.
	if err := os.Rename(stagingAbs, childAbs); err != nil {
		return ScannedFolder{}, err
	}
	// Staging is gone now — disable cleanup.
	_ = os.RemoveAll(stagingAbs) // no-op if rename succeeded

	// Refresh.
	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedFolder{}, err
	}

	f, _ := s.GetFolderByID(newID)
	return f, nil
}

// ── Create Workspace ─────────────────────────────────────────────────────────

// CreateWorkspace creates a new workspace under parentFolderID (empty = root).
func (s *Service) CreateWorkspace(parentFolderID, name, templateID string, refreshBaseline func() error) (ScannedWorkspace, error) {
	name = strings.TrimSpace(name)
	if err := validateEntityName(name); err != nil {
		return ScannedWorkspace{}, err
	}

	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	parentPath := ""
	if parentFolderID != "" {
		f, ok := s.GetFolderByID(parentFolderID)
		if !ok {
			return ScannedWorkspace{}, fmt.Errorf("parent folder not found: %s", parentFolderID)
		}
		parentPath = f.Path
	}

	childRel := joinRelPath(parentPath, name)
	childAbs := filepath.Join(vaultDir, filepath.FromSlash(childRel))

	if _, err := os.Lstat(childAbs); err == nil {
		return ScannedWorkspace{}, fmt.Errorf("conflict: %s already exists", childRel)
	}

	stagingName := "." + name + ".staging." + uuid.NewString()[:8]
	stagingRel := joinRelPath(parentPath, stagingName)
	stagingAbs := filepath.Join(vaultDir, filepath.FromSlash(stagingRel))

	s.BeginInternalMutation()
	defer func() {
		_ = os.RemoveAll(stagingAbs)
	}()

	if err := os.MkdirAll(stagingAbs, 0o755); err != nil {
		return ScannedWorkspace{}, err
	}

	wsID := uuid.NewString()
	if err := WriteWorkspaceMarker(stagingAbs, wsID); err != nil {
		return ScannedWorkspace{}, err
	}

	// Apply template.
	if templateID != "" {
		if err := applyWorkspaceTemplate(stagingAbs, templateID); err != nil {
			return ScannedWorkspace{}, err
		}
	}

	// Write metadata with template features for UUID-keyed lookup.
	if err := writeWorkspaceMetadataV2(stagingAbs, wsID, name, templateID); err != nil {
		// Non-fatal: workspace is created even if metadata write fails.
		// The tree reconciliation will pick up the workspace marker.
	}

	// Atomic rename.
	if err := os.Rename(stagingAbs, childAbs); err != nil {
		return ScannedWorkspace{}, err
	}
	_ = os.RemoveAll(stagingAbs)

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedWorkspace{}, err
	}

	// Select the new workspace.
	s.SetCurrentWorkspace(wsID)

	ws, _ := s.GetWorkspaceByID(wsID)
	return ws, nil
}

// ── Rename ───────────────────────────────────────────────────────────────────

// RenameFolder renames a folder, preserving its UUID.
func (s *Service) RenameFolder(folderID, newName string, refreshBaseline func() error) (ScannedFolder, error) {
	newName = strings.TrimSpace(newName)
	if err := validateEntityName(newName); err != nil {
		return ScannedFolder{}, err
	}

	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	f, ok := s.GetFolderByID(folderID)
	if !ok {
		return ScannedFolder{}, fmt.Errorf("folder not found: %s", folderID)
	}

	oldAbs := filepath.Join(vaultDir, filepath.FromSlash(f.Path))
	parent := parentPath(f.Path)
	newRel := joinRelPath(parent, newName)
	newAbs := filepath.Join(vaultDir, filepath.FromSlash(newRel))

	if newRel == f.Path {
		return f, nil // no-op
	}
	if _, err := os.Lstat(newAbs); err == nil {
		return ScannedFolder{}, fmt.Errorf("conflict: %s already exists", newRel)
	}

	s.BeginInternalMutation()
	if strings.EqualFold(f.Name, newName) && f.Name != newName {
		// Case-only rename on case-insensitive FS: use intermediate name.
		tmp := newRel + ".case-rename." + uuid.NewString()[:8]
		tmpAbs := filepath.Join(vaultDir, filepath.FromSlash(tmp))
		if err := os.Rename(oldAbs, tmpAbs); err != nil {
			return ScannedFolder{}, err
		}
		if err := os.Rename(tmpAbs, newAbs); err != nil {
			_ = os.Rename(tmpAbs, oldAbs)
			return ScannedFolder{}, err
		}
	} else {
		if err := os.Rename(oldAbs, newAbs); err != nil {
			return ScannedFolder{}, err
		}
	}

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedFolder{}, err
	}

	updated, _ := s.GetFolderByID(folderID)
	return updated, nil
}

// RenameWorkspace renames a workspace, preserving its UUID.
func (s *Service) RenameWorkspace(workspaceID, newName string, refreshBaseline func() error) (ScannedWorkspace, error) {
	newName = strings.TrimSpace(newName)
	if err := validateEntityName(newName); err != nil {
		return ScannedWorkspace{}, err
	}

	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	ws, ok := s.GetWorkspaceByID(workspaceID)
	if !ok {
		return ScannedWorkspace{}, fmt.Errorf("workspace not found: %s", workspaceID)
	}

	oldAbs := filepath.Join(vaultDir, filepath.FromSlash(ws.RootPath))
	parent := parentPath(ws.RootPath)
	newRel := joinRelPath(parent, newName)
	newAbs := filepath.Join(vaultDir, filepath.FromSlash(newRel))

	if newRel == ws.RootPath {
		return ws, nil
	}
	if _, err := os.Lstat(newAbs); err == nil {
		return ScannedWorkspace{}, fmt.Errorf("conflict: %s already exists", newRel)
	}

	s.BeginInternalMutation()
	if strings.EqualFold(ws.Name, newName) && ws.Name != newName {
		tmp := newRel + ".case-rename." + uuid.NewString()[:8]
		tmpAbs := filepath.Join(vaultDir, filepath.FromSlash(tmp))
		if err := os.Rename(oldAbs, tmpAbs); err != nil {
			return ScannedWorkspace{}, err
		}
		if err := os.Rename(tmpAbs, newAbs); err != nil {
			_ = os.Rename(tmpAbs, oldAbs)
			return ScannedWorkspace{}, err
		}
	} else {
		if err := os.Rename(oldAbs, newAbs); err != nil {
			return ScannedWorkspace{}, err
		}
	}

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedWorkspace{}, err
	}

	updated, _ := s.GetWorkspaceByID(workspaceID)
	return updated, nil
}

// ── Move ─────────────────────────────────────────────────────────────────────

// MoveFolder moves a folder to a new parent.
func (s *Service) MoveFolder(folderID, targetParentFolderID string, refreshBaseline func() error) (ScannedFolder, error) {
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	f, ok := s.GetFolderByID(folderID)
	if !ok {
		return ScannedFolder{}, fmt.Errorf("folder not found: %s", folderID)
	}

	// Resolve target parent path.
	targetParentPath := ""
	if targetParentFolderID != "" {
		tf, ok := s.GetFolderByID(targetParentFolderID)
		if !ok {
			return ScannedFolder{}, fmt.Errorf("target parent folder not found: %s", targetParentFolderID)
		}
		targetParentPath = tf.Path
	}

	// Reject moving into itself or descendant.
	if targetParentPath == f.Path || isPathPrefix(f.Path, targetParentPath) {
		return ScannedFolder{}, fmt.Errorf("cannot move folder into itself or descendant")
	}

	newRel := joinRelPath(targetParentPath, f.Name)
	newAbs := filepath.Join(vaultDir, filepath.FromSlash(newRel))

	if newRel == f.Path {
		return f, nil
	}
	if _, err := os.Lstat(newAbs); err == nil {
		return ScannedFolder{}, fmt.Errorf("conflict: %s already exists", newRel)
	}

	s.BeginInternalMutation()
	if err := os.Rename(
		filepath.Join(vaultDir, filepath.FromSlash(f.Path)),
		newAbs,
	); err != nil {
		return ScannedFolder{}, err
	}

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedFolder{}, err
	}

	updated, _ := s.GetFolderByID(folderID)
	return updated, nil
}

// MoveWorkspace moves a workspace to a new parent folder.
func (s *Service) MoveWorkspace(workspaceID, targetParentFolderID string, refreshBaseline func() error) (ScannedWorkspace, error) {
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	ws, ok := s.GetWorkspaceByID(workspaceID)
	if !ok {
		return ScannedWorkspace{}, fmt.Errorf("workspace not found: %s", workspaceID)
	}

	targetParentPath := ""
	if targetParentFolderID != "" {
		tf, ok := s.GetFolderByID(targetParentFolderID)
		if !ok {
			return ScannedWorkspace{}, fmt.Errorf("target parent folder not found: %s", targetParentFolderID)
		}
		targetParentPath = tf.Path
	}

	newRel := joinRelPath(targetParentPath, ws.Name)
	newAbs := filepath.Join(vaultDir, filepath.FromSlash(newRel))
	fmt.Fprintf(os.Stderr, "[MOVE-DIAG] MoveWorkspace wsID=%s wsName=%s oldPath=%q newPath=%q targetParent=%q\n",
		workspaceID, ws.Name, ws.RootPath, newRel, targetParentPath)

	if newRel == ws.RootPath {
		return ws, nil
	}
	if _, err := os.Lstat(newAbs); err == nil {
		return ScannedWorkspace{}, fmt.Errorf("conflict: %s already exists", newRel)
	}

	s.BeginInternalMutation()
	if err := os.Rename(
		filepath.Join(vaultDir, filepath.FromSlash(ws.RootPath)),
		newAbs,
	); err != nil {
		return ScannedWorkspace{}, err
	}

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedWorkspace{}, err
	}

	updated, _ := s.GetWorkspaceByID(workspaceID)
	return updated, nil
}

// ── Validation ───────────────────────────────────────────────────────────────

func validateEntityName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid name: %s", name)
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("name cannot start with dot")
	}
	for _, r := range name {
		if r < 32 {
			return fmt.Errorf("name contains control character")
		}
	}
	return nil
}

// applyWorkspaceTemplate creates default folders for a new workspace.
func applyWorkspaceTemplate(workspaceDir, templateID string) error {
	// Default template: Notes + Files.
	folders := []string{"Notes", "Files"}
	for _, folder := range folders {
		if err := os.MkdirAll(filepath.Join(workspaceDir, folder), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// writeWorkspaceMetadataV2 writes workspace metadata to the vault's workspace registry
// keyed by the workspace UUID. This enables metadata lookup after move/rename.
func writeWorkspaceMetadataV2(workspaceDir, workspaceID, workspaceName, templateID string) error {
	if workspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	// Determine vault root: go up from workspaceDir until we find .verstak
	vaultDir := workspaceDir
	for {
		if _, err := os.Stat(filepath.Join(vaultDir, ".verstak")); err == nil {
			if _, err := os.Stat(filepath.Join(vaultDir, ".verstak", "vault.json")); err == nil {
				break
			}
		}
		parent := filepath.Dir(vaultDir)
		if parent == vaultDir {
			return fmt.Errorf("vault root not found")
		}
		vaultDir = parent
	}

	wsDir := filepath.Join(vaultDir, ".verstak", "workspaces")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	meta := map[string]interface{}{
		"workspaceId":   workspaceID,
		"workspaceName": workspaceName,
		"features":      defaultFeaturesV2(),
		"folders":       defaultFoldersV2(),
		"updatedAt":     now,
	}
	if templateID != "" {
		meta["createdFromTemplate"] = map[string]interface{}{
			"templateId": templateID,
			"appliedAt":  now,
		}
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(wsDir, "uuid-"+workspaceID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func defaultFeaturesV2() map[string]bool {
	return map[string]bool{
		"files":         true,
		"notes":         true,
		"journal":       true,
		"activity":      true,
		"browser-inbox": true,
	}
}

func defaultFoldersV2() map[string]string {
	return map[string]string{
		"notes": "Notes",
		"files": "Files",
	}
}

// ── Set current workspace ────────────────────────────────────────────────────

// SetCurrentWorkspaceID sets the current workspace by UUID.
func (s *Service) SetCurrentWorkspaceID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scan != nil {
		if _, ok := s.scan.Workspaces[id]; !ok && id != "" {
			return fmt.Errorf("workspace not found: %s", id)
		}
	}
	s.currentWS = id
	if s.tree != nil {
		s.tree.CurrentWorkspaceID = id
	}
	return nil
}

// GetCurrentWorkspaceID returns the current workspace UUID.
func (s *Service) GetCurrentWorkspaceID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentWS
}
