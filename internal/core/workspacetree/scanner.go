package workspacetree

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
)

const (
	workspaceMarkerRel = ".verstak/workspace.json"
	folderMarkerRel    = ".verstak/folder.json"
)

// scanner performs a read-only recursive scan of the vault.
type scanner struct {
	vaultDir string
	// mutable state
	warnings   []TreeDiagnostic
	workspaces map[string]ScannedWorkspace
	folders    map[string]ScannedFolder
	// parent tracking: maps folder path → folder UUID
	pathToParentID map[string]string
}

func newScanner(vaultDir string) *scanner {
	return &scanner{
		vaultDir:       vaultDir,
		workspaces:     make(map[string]ScannedWorkspace),
		folders:        make(map[string]ScannedFolder),
		pathToParentID: make(map[string]string),
	}
}

// Scan performs a full read-only scan. It does not mutate the filesystem.
func Scan(vaultDir string) (*ScanResult, error) {
	s := newScanner(vaultDir)
	if err := s.scanDir(vaultDir, ""); err != nil {
		return nil, err
	}
	return &ScanResult{
		Folders:    s.folders,
		Workspaces: s.workspaces,
		Warnings:   s.warnings,
	}, nil
}

// scanDir recursively scans a directory. relPath is slash-separated relative to vault root.
// It returns true if the directory is a workspace (to stop parent recursion).
func (s *scanner) scanDir(absDir, relPath string) error {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return err
	}

	// Sort entries for deterministic scan order.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// First pass: check for markers.
	hasWorkspaceMarker := false
	hasFolderMarker := false
	var workspaceID, folderID string

	verstakDir := filepath.Join(absDir, ".verstak")
	if info, err := os.Lstat(verstakDir); err == nil && info.IsDir() {
		// Read workspace marker.
		if id, ok := s.readWorkspaceMarker(verstakDir, relPath); ok {
			hasWorkspaceMarker = true
			workspaceID = id
		}
		// Read folder marker (only relevant if no workspace marker).
		if !hasWorkspaceMarker {
			if id, ok := s.readFolderMarker(verstakDir, relPath); ok {
				hasFolderMarker = true
				folderID = id
			}
		}
	}

	if hasWorkspaceMarker {
		name := filepath.Base(absDir)
		s.registerWorkspace(workspaceID, name, relPath)
		return nil // Stop recursion — workspace is a leaf.
	}

	if hasFolderMarker {
		name := filepath.Base(absDir)
		parentID := s.computeParentID(relPath)
		s.registerFolder(folderID, name, relPath, parentID)
	}

	// Second pass: recurse into subdirectories.
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			// Skip hidden directories and .verstak.
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			// Skip symlinks.
			continue
		}

		childAbs := filepath.Join(absDir, name)
		childRel := joinRelPath(relPath, name)

		if err := s.scanDir(childAbs, childRel); err != nil {
			// Non-fatal: record warning and continue.
			s.warnings = append(s.warnings, TreeDiagnostic{
				Level:   "warning",
				Code:    "scan-error",
				Message: "cannot scan directory: " + err.Error(),
				Path:    childRel,
			})
		}
	}

	return nil
}

// readWorkspaceMarker reads and validates the workspace marker file.
func (s *scanner) readWorkspaceMarker(verstakDir, relPath string) (string, bool) {
	markerPath := filepath.Join(verstakDir, "workspace.json")
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return "", false
	}
	marker, err := ParseWorkspaceMarker(data)
	if err != nil {
		s.warnings = append(s.warnings, TreeDiagnostic{
			Level:   "warning",
			Code:    "corrupted-marker",
			Message: "cannot parse workspace marker: " + err.Error(),
			Path:    relPath,
		})
		return "", false
	}
	if _, err := uuid.Parse(marker.WorkspaceID); err != nil {
		s.warnings = append(s.warnings, TreeDiagnostic{
			Level:   "warning",
			Code:    "corrupted-marker",
			Message: "invalid workspace UUID: " + marker.WorkspaceID,
			Path:    relPath,
		})
		return "", false
	}
	return marker.WorkspaceID, true
}

// readFolderMarker reads and validates the folder marker file.
func (s *scanner) readFolderMarker(verstakDir, relPath string) (string, bool) {
	markerPath := filepath.Join(verstakDir, "folder.json")
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return "", false
	}
	marker, err := ParseFolderMarker(data)
	if err != nil {
		s.warnings = append(s.warnings, TreeDiagnostic{
			Level:   "warning",
			Code:    "corrupted-marker",
			Message: "cannot parse folder marker: " + err.Error(),
			Path:    relPath,
		})
		return "", false
	}
	if _, err := uuid.Parse(marker.FolderID); err != nil {
		s.warnings = append(s.warnings, TreeDiagnostic{
			Level:   "warning",
			Code:    "corrupted-marker",
			Message: "invalid folder UUID: " + marker.FolderID,
			Path:    relPath,
		})
		return "", false
	}
	return marker.FolderID, true
}

// registerWorkspace records a discovered workspace, checking for duplicates.
func (s *scanner) registerWorkspace(id, name, relPath string) {
	if existing, ok := s.workspaces[id]; ok {
		s.warnings = append(s.warnings, TreeDiagnostic{
			Level:    "error",
			Code:     "duplicate-id",
			Message:  "duplicate workspace UUID " + id + ": found at " + relPath + " and " + existing.RootPath,
			EntityID: id,
			Path:     relPath,
		})
		// Keep first occurrence; second is the duplicate.
		return
	}
	s.workspaces[id] = ScannedWorkspace{
		ID:       id,
		Name:     name,
		RootPath: relPath,
	}
}

// registerFolder records a discovered folder, checking for duplicates.
func (s *scanner) registerFolder(id, name, relPath, parentID string) {
	if existing, ok := s.folders[id]; ok {
		s.warnings = append(s.warnings, TreeDiagnostic{
			Level:    "error",
			Code:     "duplicate-id",
			Message:  "duplicate folder UUID " + id + ": found at " + relPath + " and " + existing.Path,
			EntityID: id,
			Path:     relPath,
		})
		return
	}
	s.folders[id] = ScannedFolder{
		ID:       id,
		Name:     name,
		Path:     relPath,
		ParentID: parentID,
	}
	s.pathToParentID[relPath] = id
}

// computeParentID finds the parent folder UUID for a given path.
func (s *scanner) computeParentID(relPath string) string {
	parent := parentPath(relPath)
	if parent == "" {
		return ""
	}
	return s.pathToParentID[parent]
}

// parentPath returns the parent directory path, or "" for root-level.
func parentPath(relPath string) string {
	if relPath == "" {
		return ""
	}
	idx := strings.LastIndex(relPath, "/")
	if idx < 0 {
		return ""
	}
	return relPath[:idx]
}

// joinRelPath joins two slash-separated path segments.
func joinRelPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "/" + name
}
