package workspacetree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveInsideVault resolves a relative vault path to an absolute path,
// rejecting traversal, symlinks, and other unsafe constructs.
func ResolveInsideVault(vaultRoot string, relativePath string) (string, error) {
	vaultRoot = filepath.Clean(vaultRoot)
	if !filepath.IsAbs(vaultRoot) {
		return "", fmt.Errorf("vault root must be absolute: %s", vaultRoot)
	}

	relativePath = filepath.ToSlash(strings.TrimSpace(relativePath))
	if relativePath == "" {
		return vaultRoot, nil
	}
	if filepath.IsAbs(relativePath) || strings.HasPrefix(relativePath, "/") || strings.HasPrefix(relativePath, "\\") {
		return "", fmt.Errorf("relative path must not be absolute: %s", relativePath)
	}
	// On any OS, reject Windows drive paths.
	if len(relativePath) >= 2 && relativePath[1] == ':' {
		return "", fmt.Errorf("relative path must not be absolute: %s", relativePath)
	}

	// Reject traversal.
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relativePath)))
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path traversal rejected: %s", relativePath)
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == ".." {
			return "", fmt.Errorf("path traversal rejected: %s", relativePath)
		}
		if strings.Contains(segment, "\x00") {
			return "", fmt.Errorf("null byte in path: %s", relativePath)
		}
	}

	target := filepath.Join(vaultRoot, filepath.FromSlash(cleaned))
	target = filepath.Clean(target)

	// Verify target is inside vault.
	rel, err := filepath.Rel(vaultRoot, target)
	if err != nil {
		return "", fmt.Errorf("cannot resolve relative path: %w", err)
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path escapes vault: %s", relativePath)
	}

	// Check target parent is not a symlink.
	parent := filepath.Dir(target)
	if info, err := os.Lstat(parent); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("parent is symlink: %s", parent)
	}

	return target, nil
}

// ResolveEntityPath resolves a folder or workspace ID to its absolute path
// using the current tree index. Returns error if the entity is not found.
func (s *Service) ResolveEntityPath(entityID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.scan == nil || s.vaultDir == "" {
		return "", fmt.Errorf("service not initialized")
	}

	// Check folders.
	if f, ok := s.scan.Folders[entityID]; ok {
		return ResolveInsideVault(s.vaultDir, f.Path)
	}

	// Check workspaces.
	if ws, ok := s.scan.Workspaces[entityID]; ok {
		return ResolveInsideVault(s.vaultDir, ws.RootPath)
	}

	return "", fmt.Errorf("entity not found: %s", entityID)
}
