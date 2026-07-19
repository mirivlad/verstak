package workspacetree

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// FolderAppearance stores visual presentation metadata for a folder.
type FolderAppearance struct {
	Icon  string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`
}

// GetFolderAppearance reads appearance metadata for a folder.
func (s *Service) GetFolderAppearance(folderID string) (*FolderAppearance, error) {
	if _, err := uuid.Parse(folderID); err != nil {
		return nil, fmt.Errorf("invalid folder ID")
	}
	path := folderAppearancePath(s.vaultDir, folderID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &FolderAppearance{}, nil
		}
		return nil, err
	}
	var a FolderAppearance
	if err := json.Unmarshal(data, &a); err != nil {
		return &FolderAppearance{}, nil
	}
	return &a, nil
}

// SetFolderAppearance writes appearance metadata for a folder.
func (s *Service) SetFolderAppearance(folderID string, patch *FolderAppearance) error {
	if _, err := uuid.Parse(folderID); err != nil {
		return fmt.Errorf("invalid folder ID")
	}
	if patch.Icon != "" && !isValidIconName(patch.Icon) {
		return fmt.Errorf("invalid icon name")
	}
	if patch.Color != "" && !isValidColor(patch.Color) {
		return fmt.Errorf("invalid color format, expected #RRGGBB")
	}

	// Read existing.
	existing, _ := s.GetFolderAppearance(folderID)
	if patch.Icon != "" {
		existing.Icon = patch.Icon
	}
	if patch.Color != "" {
		existing.Color = patch.Color
	}

	path := folderAppearancePath(s.vaultDir, folderID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(existing)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ResetFolderAppearance removes appearance metadata for a folder.
func (s *Service) ResetFolderAppearance(folderID string) error {
	if _, err := uuid.Parse(folderID); err != nil {
		return fmt.Errorf("invalid folder ID")
	}
	path := folderAppearancePath(s.vaultDir, folderID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func folderAppearancePath(vaultDir, folderID string) string {
	return filepath.Join(vaultDir, ".verstak", "folders", folderID+".json")
}

func isValidIconName(name string) bool {
	if len(name) > 64 || len(name) < 1 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
}

func isValidColor(color string) bool {
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	for _, r := range color[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

// ── Contribution point helpers ───────────────────────────────────────────────

// GetFolderTreeNodeActions returns contribution-compatible metadata for a folder node.
func (s *Service) GetFolderTreeNodeActions(folderID string) map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.scan.Folders[folderID]
	if !ok {
		return nil
	}
	return map[string]interface{}{
		"folderId":   folderID,
		"folderName": f.Name,
		"folderPath": f.Path,
	}
}

// GetWorkspaceTreeNodeActions returns contribution-compatible metadata for a workspace node.
func (s *Service) GetWorkspaceTreeNodeActions(workspaceID string) map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	ws, ok := s.scan.Workspaces[workspaceID]
	if !ok {
		return nil
	}
	return map[string]interface{}{
		"workspaceId":   workspaceID,
		"workspaceName": ws.Name,
		"workspacePath": ws.RootPath,
	}
}

// GetFolderAppearanceByID is a static helper for the V2 API layer.
func GetFolderAppearanceByID(vaultDir, folderID string) (*FolderAppearance, error) {
	svc := &Service{vaultDir: vaultDir}
	return svc.GetFolderAppearance(folderID)
}

// SetFolderAppearanceByID is a static helper.
func SetFolderAppearanceByID(vaultDir, folderID string, patch *FolderAppearance) error {
	svc := &Service{vaultDir: vaultDir}
	return svc.SetFolderAppearance(folderID, patch)
}


