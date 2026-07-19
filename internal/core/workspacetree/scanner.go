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
	unmanaged  []UnmanagedDirectory
	// duplicate tracking: UUID → list of paths
	wsDuplicatePaths map[string][]string
	fDuplicatePaths  map[string][]string
	// parent tracking: maps folder path → folder UUID
	pathToParentID map[string]string
	// set of paths that are inside a workspace (should not become folders)
	workspacePaths map[string]bool
	// previous snapshot for duplicate resolution
	prev *SemanticSnapshot
}

func newScanner(vaultDir string, prev *SemanticSnapshot) *scanner {
	return &scanner{
		vaultDir:         vaultDir,
		workspaces:       make(map[string]ScannedWorkspace),
		folders:          make(map[string]ScannedFolder),
		pathToParentID:   make(map[string]string),
		wsDuplicatePaths: make(map[string][]string),
		fDuplicatePaths:  make(map[string][]string),
		workspacePaths:   make(map[string]bool),
		prev:             prev,
	}
}

// Scan performs a full read-only scan. It does not mutate the filesystem.
func Scan(vaultDir string, prev *SemanticSnapshot) (*ScanResult, error) {
	s := newScanner(vaultDir, prev)
	if err := s.scanDir(vaultDir, ""); err != nil {
		return nil, err
	}
	s.resolveDuplicates()
	return &ScanResult{
		Folders:    s.folders,
		Workspaces: s.workspaces,
		Unmanaged:  s.unmanaged,
		Warnings:   s.warnings,
	}, nil
}

// scanDir recursively scans a directory. relPath is slash-separated relative to vault root.
func (s *scanner) scanDir(absDir, relPath string) error {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// First pass: check for markers.
	hasWorkspaceMarker := false
	hasFolderMarker := false
	var workspaceID, folderID string

	verstakDir := filepath.Join(absDir, ".verstak")
	if info, err := os.Lstat(verstakDir); err == nil && info.IsDir() {
		if id, ok := s.readWorkspaceMarker(verstakDir, relPath); ok {
			hasWorkspaceMarker = true
			workspaceID = id
		}
		if !hasWorkspaceMarker {
			if id, ok := s.readFolderMarker(verstakDir, relPath); ok {
				hasFolderMarker = true
				folderID = id
			}
		}
	}

	if hasWorkspaceMarker {
		name := filepath.Base(absDir)
		s.recordWorkspace(workspaceID, name, relPath)
		// Mark this directory and its children as workspace-internal.
		s.workspacePaths[relPath] = true
		return nil
	}

	if hasFolderMarker {
		name := filepath.Base(absDir)
		parentID := s.computeParentID(relPath)
		s.recordFolder(folderID, name, relPath, parentID)
	} else if relPath != "" {
		// Directory without marker — report as unmanaged if not inside a workspace.
		if !s.isInsideWorkspace(relPath) {
			name := filepath.Base(absDir)
			parent := parentPath(relPath)
			s.unmanaged = append(s.unmanaged, UnmanagedDirectory{
				Path:   relPath,
				Name:   name,
				Parent: parent,
			})
		}
	}

	// Second pass: recurse into subdirectories.
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		childAbs := filepath.Join(absDir, name)
		childRel := joinRelPath(relPath, name)

		if err := s.scanDir(childAbs, childRel); err != nil {
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

// recordWorkspace tracks a workspace UUID, collecting duplicates for resolution.
func (s *scanner) recordWorkspace(id, name, relPath string) {
	if existing, ok := s.workspaces[id]; ok {
		s.wsDuplicatePaths[id] = append(s.wsDuplicatePaths[id], existing.RootPath, relPath)
		return
	}
	s.workspaces[id] = ScannedWorkspace{ID: id, Name: name, RootPath: relPath}
}

// recordFolder tracks a folder UUID, collecting duplicates for resolution.
func (s *scanner) recordFolder(id, name, relPath, parentID string) {
	if existing, ok := s.folders[id]; ok {
		s.fDuplicatePaths[id] = append(s.fDuplicatePaths[id], existing.Path, relPath)
		return
	}
	s.folders[id] = ScannedFolder{ID: id, Name: name, Path: relPath, ParentID: parentID}
	s.pathToParentID[relPath] = id
}

// resolveDuplicates applies duplicate UUID resolution using previous snapshot.
func (s *scanner) resolveDuplicates() {
	// Resolve workspace duplicates.
	for id, paths := range s.wsDuplicatePaths {
		s.resolveWorkspaceDuplicate(id, paths)
	}
	// Resolve folder duplicates.
	for id, paths := range s.fDuplicatePaths {
		s.resolveFolderDuplicate(id, paths)
	}
}

func (s *scanner) resolveWorkspaceDuplicate(id string, paths []string) {
	// All paths that share this UUID: the original in s.workspaces[id] + dup paths.
	allPaths := []string{s.workspaces[id].RootPath}
	allPaths = append(allPaths, paths...)
	// Deduplicate.
	seen := map[string]bool{}
	var unique []string
	for _, p := range allPaths {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	knownPath := ""
	if s.prev != nil {
		if entry, ok := s.prev.Workspaces[id]; ok {
			knownPath = entry.Path
		}
	}

	if knownPath != "" && containsPath(unique, knownPath) {
		// Known original still exists — keep its UUID.
		// Regenerate UUID for all other paths.
		for _, p := range unique {
			if p == knownPath {
				continue
			}
			newID := uuid.NewString()
			s.workspaces[newID] = ScannedWorkspace{
				ID:       newID,
				Name:     filepath.Base(filepath.FromSlash(p)),
				RootPath: p,
			}
			// Marker will be written by reconciler.
			s.warnings = append(s.warnings, TreeDiagnostic{
				Level:   "warning",
				Code:    "duplicate-id-resolved",
				Message: "duplicate workspace UUID " + id + " at " + p + " resolved to new UUID " + newID,
				Path:    p,
			})
		}
		// Remove the original if it was only a duplicate (not the actual first)
		// Actually, the original is fine — it's at knownPath.
	} else if knownPath != "" && !containsPath(unique, knownPath) {
		// Known path is gone — this is a move. Keep single UUID.
		// Only one path should remain; if multiple, it's ambiguous.
		if len(unique) == 1 {
			// Rename/move — keep UUID, update path.
			s.workspaces[id] = ScannedWorkspace{
				ID:       id,
				Name:     filepath.Base(filepath.FromSlash(unique[0])),
				RootPath: unique[0],
			}
		} else {
			s.flagAmbiguousDuplicate("workspace", id, unique)
		}
	} else {
		// No snapshot — ambiguous.
		s.flagAmbiguousDuplicate("workspace", id, unique)
	}
}

func (s *scanner) resolveFolderDuplicate(id string, paths []string) {
	allPaths := []string{s.folders[id].Path}
	allPaths = append(allPaths, paths...)
	seen := map[string]bool{}
	var unique []string
	for _, p := range allPaths {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	knownPath := ""
	if s.prev != nil {
		if entry, ok := s.prev.Folders[id]; ok {
			knownPath = entry.Path
		}
	}

	if knownPath != "" && containsPath(unique, knownPath) {
		for _, p := range unique {
			if p == knownPath {
				continue
			}
			newID := uuid.NewString()
			s.folders[newID] = ScannedFolder{
				ID:       newID,
				Name:     filepath.Base(filepath.FromSlash(p)),
				Path:     p,
				ParentID: s.computeParentID(p),
			}
			s.pathToParentID[p] = newID
			s.warnings = append(s.warnings, TreeDiagnostic{
				Level:   "warning",
				Code:    "duplicate-id-resolved",
				Message: "duplicate folder UUID " + id + " at " + p + " resolved to new UUID " + newID,
				Path:    p,
			})
		}
	} else if knownPath != "" && !containsPath(unique, knownPath) {
		if len(unique) == 1 {
			s.folders[id] = ScannedFolder{
				ID:       id,
				Name:     filepath.Base(filepath.FromSlash(unique[0])),
				Path:     unique[0],
				ParentID: s.computeParentID(unique[0]),
			}
			s.pathToParentID[unique[0]] = id
		} else {
			s.flagAmbiguousDuplicate("folder", id, unique)
		}
	} else {
		s.flagAmbiguousDuplicate("folder", id, unique)
	}
}

func (s *scanner) flagAmbiguousDuplicate(kind, id string, paths []string) {
	// Remove the entity from the map and flag as conflict.
	switch kind {
	case "workspace":
		delete(s.workspaces, id)
	case "folder":
		delete(s.folders, id)
	}
	s.warnings = append(s.warnings, TreeDiagnostic{
		Level:    "error",
		Code:     "duplicate-id",
		Message:  "ambiguous duplicate " + kind + " UUID " + id + " at paths: " + strings.Join(paths, ", "),
		EntityID: id,
	})
}

func (s *scanner) isInsideWorkspace(relPath string) bool {
	// Check if any ancestor path is a workspace.
	for p := relPath; p != ""; p = parentPath(p) {
		if s.workspacePaths[p] {
			return true
		}
	}
	return false
}

func (s *scanner) computeParentID(relPath string) string {
	parent := parentPath(relPath)
	if parent == "" {
		return ""
	}
	return s.pathToParentID[parent]
}

func containsPath(paths []string, target string) bool {
	for _, p := range paths {
		if p == target {
			return true
		}
	}
	return false
}

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

func joinRelPath(base, name string) string {
	if base == "" {
		return name
	}
	return base + "/" + name
}
