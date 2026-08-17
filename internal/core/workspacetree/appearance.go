package workspacetree

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const legacyFolderAppearancePluginID = "verstak.folder-appearance"

// FolderAppearance stores visual presentation metadata for a folder.
type FolderAppearance struct {
	Icon  string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`
}

type legacyFolderAppearance struct {
	IconID  string `json:"iconId,omitempty"`
	ColorID string `json:"colorId,omitempty"`
}

type legacyFolderAppearanceFile struct {
	Folders map[string]legacyFolderAppearance `json:"folders"`
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

// SetFolderAppearance applies non-empty fields as a patch. Core callers that
// need exact UI replacement semantics should use ReplaceFolderAppearance.
func (s *Service) SetFolderAppearance(folderID string, patch *FolderAppearance) error {
	if err := validateFolderAppearance(folderID, patch); err != nil {
		return err
	}

	existing, _ := s.GetFolderAppearance(folderID)
	if patch.Icon != "" {
		existing.Icon = patch.Icon
	}
	if patch.Color != "" {
		existing.Color = patch.Color
	}
	return writeFolderAppearance(s.vaultDir, folderID, existing)
}

// ReplaceFolderAppearance persists exactly the supplied icon and color. Empty
// values remove the corresponding customization, matching the workspace-tree
// editor's Save semantics. With both fields empty there is no metadata file.
func (s *Service) ReplaceFolderAppearance(folderID string, appearance *FolderAppearance) error {
	if err := validateFolderAppearance(folderID, appearance); err != nil {
		return err
	}
	if appearance.Icon == "" && appearance.Color == "" {
		return s.ResetFolderAppearance(folderID)
	}
	return writeFolderAppearance(s.vaultDir, folderID, appearance)
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

func validateFolderAppearance(folderID string, appearance *FolderAppearance) error {
	if _, err := uuid.Parse(folderID); err != nil {
		return fmt.Errorf("invalid folder ID")
	}
	if appearance == nil {
		return fmt.Errorf("folder appearance is nil")
	}
	if appearance.Icon != "" && !isValidIconName(appearance.Icon) {
		return fmt.Errorf("invalid icon name")
	}
	if appearance.Color != "" && !isValidColor(appearance.Color) {
		return fmt.Errorf("invalid color format, expected #RRGGBB")
	}
	return nil
}

func writeFolderAppearance(vaultDir, folderID string, appearance *FolderAppearance) error {
	path := folderAppearancePath(vaultDir, folderID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(appearance)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func folderAppearancePath(vaultDir, folderID string) string {
	return filepath.Join(vaultDir, ".verstak", "folders", folderID+".json")
}

func legacyFolderAppearancePath(vaultDir string) string {
	return filepath.Join(vaultDir, ".verstak", "plugin-data", legacyFolderAppearancePluginID, "appearance.json")
}

// migrateLegacyFolderAppearance moves still-relevant appearance values from
// the retired plugin namespace into core-owned per-folder metadata. It is
// deliberately best-effort: corrupt cosmetic legacy data must never prevent a
// vault from opening. Existing core metadata always wins and the legacy file is
// retained, making the migration idempotent and recoverable.
func (s *Service) migrateLegacyFolderAppearance() int {
	data, err := os.ReadFile(legacyFolderAppearancePath(s.vaultDir))
	if err != nil {
		return 0
	}
	var legacy legacyFolderAppearanceFile
	if err := json.Unmarshal(data, &legacy); err != nil {
		return 0
	}

	migrated := 0
	for folderID, old := range legacy.Folders {
		if _, err := uuid.Parse(folderID); err != nil {
			continue
		}
		if _, ok := s.GetFolderByID(folderID); !ok {
			continue
		}
		if _, err := os.Stat(folderAppearancePath(s.vaultDir, folderID)); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			continue
		}

		next := &FolderAppearance{}
		if isValidIconName(old.IconID) {
			next.Icon = old.IconID
		}
		if isValidColor(old.ColorID) {
			next.Color = old.ColorID
		}
		if next.Icon == "" && next.Color == "" {
			continue
		}
		if err := s.ReplaceFolderAppearance(folderID, next); err == nil {
			migrated++
		}
	}
	return migrated
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
