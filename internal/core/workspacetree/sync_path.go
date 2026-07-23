package workspacetree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
)

// ApplyPathFromSync moves or renames an existing semantic node by stable key.
// The target parent is resolved from the backend tree rather than trusted from
// the caller, and the durable UUID marker moves with the directory.
func (s *Service) ApplyPathFromSync(sourceKey, previousPath, targetPath string, refreshBaseline func() error) error {
	if !validOrderNodeKey(sourceKey) {
		return fmt.Errorf("invalid source key: %s", sourceKey)
	}
	nodes, _ := s.placementTopology()
	source, ok := nodes[sourceKey]
	if !ok {
		return fmt.Errorf("source node not found: %s", sourceKey)
	}
	previousPath = filepath.ToSlash(strings.TrimSpace(previousPath))
	if previousPath != "" && previousPath != source.Path {
		return fmt.Errorf("conflict: source %s is at %s, expected %s", sourceKey, source.Path, previousPath)
	}
	targetPath = filepath.ToSlash(strings.TrimSpace(targetPath))
	if targetPath == "" {
		return fmt.Errorf("target path is empty")
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(targetPath)))
	if cleaned != targetPath {
		return fmt.Errorf("target path is not normalized: %s", targetPath)
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if err := validateEntityName(segment); err != nil {
			return fmt.Errorf("invalid target path %s: %w", targetPath, err)
		}
	}
	if cleaned == source.Path {
		return nil
	}

	targetParentPath := parentPath(cleaned)
	if targetParentPath != "" {
		found := false
		for _, candidate := range nodes {
			if candidate.Kind == "folder" && candidate.Path == targetParentPath {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("target parent folder not found: %s", targetParentPath)
		}
	}
	if source.Kind == "folder" && isPathPrefix(source.Path, cleaned) {
		return fmt.Errorf("cannot move folder into itself or descendant")
	}

	oldAbs, err := ResolveInsideVault(s.vaultDir, source.Path)
	if err != nil {
		return err
	}
	targetAbs, err := ResolveInsideVault(s.vaultDir, cleaned)
	if err != nil {
		return err
	}
	caseOnly := strings.EqualFold(source.Path, cleaned)
	if !caseOnly {
		if _, err := os.Lstat(targetAbs); err == nil {
			return fmt.Errorf("conflict: %s already exists", cleaned)
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	s.BeginInternalMutation()
	abort := func() {
		atomic.AddInt32(&s.internalMutations, -1)
	}
	if caseOnly {
		tempAbs := targetAbs + ".case-rename." + uuid.NewString()[:8]
		if err := os.Rename(oldAbs, tempAbs); err != nil {
			abort()
			return err
		}
		if err := os.Rename(tempAbs, targetAbs); err != nil {
			_ = os.Rename(tempAbs, oldAbs)
			abort()
			return err
		}
	} else if err := os.Rename(oldAbs, targetAbs); err != nil {
		abort()
		return err
	}
	return s.EndInternalMutationAndRefreshBaseline(refreshBaseline)
}
