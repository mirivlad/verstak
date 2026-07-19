package workspacetree

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Reconciler compares a scan against the previous snapshot and produces events.
// It also handles unmanaged directories by creating folder markers.
type Reconciler struct {
	vaultDir string
	prev     *SemanticSnapshot
	scan     *ScanResult
	events   []ReconEvent
	warnings []TreeDiagnostic
	// Created markers during this reconciliation.
	newFolders    []ScannedFolder
	newWorkspaces []ScannedWorkspace
	// Paths of workspaces and folders after reconciliation.
	wsPaths map[string]bool
	fPaths  map[string]bool
}

// Reconcile compares the current scan with the previous snapshot.
// It handles unmanaged directories by creating folder.json markers.
func Reconcile(vaultDir string, prev *SemanticSnapshot, scan *ScanResult) *ReconResult {
	r := &Reconciler{
		vaultDir: vaultDir,
		prev:     prev,
		scan:     scan,
		wsPaths:  make(map[string]bool),
		fPaths:   make(map[string]bool),
	}
	r.run()
	return &ReconResult{
		Snapshot:      NewSnapshotFromScan(scan),
		Events:        r.events,
		Warnings:      append(scan.Warnings, r.warnings...),
		NewFolders:    r.newFolders,
		NewWorkspaces: r.newWorkspaces,
	}
}

func (r *Reconciler) run() {
	// Build path sets for existing entities.
	for _, ws := range r.scan.Workspaces {
		r.wsPaths[ws.RootPath] = true
	}
	for _, f := range r.scan.Folders {
		r.fPaths[f.Path] = true
	}

	// Handle unmanaged directories — create folder markers.
	r.handleUnmanaged()

	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Workspaces: created, moved, renamed, deleted.
	for id, ws := range r.scan.Workspaces {
		prevEntry, hadPrev := r.prevWorkspace(id)
		if !hadPrev {
			r.events = append(r.events, ReconEvent{
				Name: "workspace-tree.changed",
				Payload: map[string]interface{}{
					"action":          "workspace.external-created",
					"workspaceId":     id,
					"workspaceName":   ws.Name,
					"workspacePath":   ws.RootPath,
					"workspaceRootPath": ws.RootPath,
					"timestamp":       now,
				},
			})
		} else if prevEntry.Path != ws.RootPath {
			r.emitWorkspaceMove(id, ws, prevEntry, now)
		}
	}

	for id, prevEntry := range r.prevWorkspaces() {
		if _, exists := r.scan.Workspaces[id]; !exists {
			r.events = append(r.events, ReconEvent{
				Name: "workspace-tree.changed",
				Payload: map[string]interface{}{
					"action":          "workspace.external-deleted",
					"workspaceId":     id,
					"workspacePath":   prevEntry.Path,
					"workspaceRootPath": prevEntry.Path,
					"timestamp":       now,
				},
			})
		}
	}

	// Folders: created, moved, renamed, deleted.
	for id, f := range r.scan.Folders {
		prevEntry, hadPrev := r.prevFolder(id)
		if !hadPrev {
			r.events = append(r.events, ReconEvent{
				Name: "workspace-tree.changed",
				Payload: map[string]interface{}{
					"action":     "folder.external-created",
					"folderId":   id,
					"folderName": f.Name,
					"folderPath": f.Path,
					"parentId":   f.ParentID,
					"timestamp":  now,
				},
			})
		} else if prevEntry.Path != f.Path {
			r.emitFolderMove(id, f, prevEntry, now)
		}
	}

	for id, prevEntry := range r.prevFolders() {
		if _, exists := r.scan.Folders[id]; !exists {
			r.events = append(r.events, ReconEvent{
				Name: "workspace-tree.changed",
				Payload: map[string]interface{}{
					"action":     "folder.external-deleted",
					"folderId":   id,
					"folderPath": prevEntry.Path,
					"timestamp":  now,
				},
			})
		}
	}
}

// handleUnmanaged creates folder.json markers for directories without markers.
// Directories that were previously workspaces (per snapshot) are NOT converted
// to folders — they contain Notes/Files and should get a diagnostic instead.
func (r *Reconciler) handleUnmanaged() {
	// Build a set of paths that were workspaces in the previous snapshot.
	prevWorkspacePaths := make(map[string]bool)
	if r.prev != nil {
		for _, entry := range r.prev.Workspaces {
			prevWorkspacePaths[entry.Path] = true
		}
	}

	// Pre-compute which paths already have .verstak directories before
	// we start creating markers. This prevents newly-created .verstak
	// from affecting the ancestor check for sibling directories.
	existingVerstak := make(map[string]bool)
	for _, u := range r.scan.Unmanaged {
		d := filepath.Join(r.vaultDir, filepath.FromSlash(u.Path), ".verstak")
		if info, err := os.Lstat(d); err == nil && info.IsDir() {
			existingVerstak[u.Path] = true
		}
	}

	// Sort unmanaged directories by depth (shallow first) so parent folders
	// get markers before their children are processed.
	sortUnmanagedByDepth(r.scan.Unmanaged)

	for _, u := range r.scan.Unmanaged {
		// Check if .verstak directory exists — indicates intermediate state.
		verstakDir := filepath.Join(r.vaultDir, filepath.FromSlash(u.Path), ".verstak")
		hasVerstakDir := false
		if info, err := os.Lstat(verstakDir); err == nil && info.IsDir() {
			hasVerstakDir = true
		}

		// If this path was previously a workspace, emit diagnostic and skip.
		// Never auto-convert a former workspace to a folder.
		if prevWorkspacePaths[u.Path] {
			r.warnings = append(r.warnings, TreeDiagnostic{
				Level:   "error",
				Code:    "workspace-marker-missing",
				Message: "directory was previously a workspace but marker is now missing: " + u.Path,
				Path:    u.Path,
			})
			continue
		}

		// If .verstak exists but no valid markers → intermediate state.
		// Do NOT create folder.json. Wait for next reconciliation cycle.
		// This handles both copy-in-progress and marker-deletion scenarios.
		if hasVerstakDir {
			continue
		}

		// Skip if already inside a workspace.
		if r.wsPaths[u.Path] {
			continue
		}
		// Skip if path is a child of a known workspace.
		insideWS := false
		for wsPath := range r.wsPaths {
			if isPathPrefix(wsPath, u.Path) {
				insideWS = true
				break
			}
		}
		if insideWS {
			continue
		}
		// Skip if any ancestor had a .verstak directory BEFORE this reconciliation.
		// This prevents Notes/Files inside a workspace whose marker was deleted
		// from being converted to organizational folders.
		if hasVerstakAncestorInSet(u.Path, existingVerstak) {
			continue
		}

		absDir := filepath.Join(r.vaultDir, filepath.FromSlash(u.Path))

		// Check .verstak is not corrupted.
		verstakDir = filepath.Join(absDir, ".verstak")
		if info, err := os.Lstat(verstakDir); err == nil {
			if !info.IsDir() {
				r.warnings = append(r.warnings, TreeDiagnostic{
					Level:   "warning",
					Code:    "corrupted-verstak",
					Message: ".verstak is not a directory at " + u.Path,
					Path:    u.Path,
				})
				continue
			}
			// Check if there's a corrupted marker inside.
			if _, err := os.Lstat(filepath.Join(verstakDir, "workspace.json")); err == nil {
				r.warnings = append(r.warnings, TreeDiagnostic{
					Level:   "warning",
					Code:    "corrupted-verstak",
					Message: ".verstak contains workspace marker but directory is unmanaged: " + u.Path,
					Path:    u.Path,
				})
				continue
			}
		}

		// Create folder marker.
		newID := uuid.NewString()
		if err := WriteFolderMarker(absDir, newID); err != nil {
			r.warnings = append(r.warnings, TreeDiagnostic{
				Level:   "error",
				Code:    "marker-write-failed",
				Message: "failed to write folder marker: " + err.Error(),
				Path:    u.Path,
			})
			continue
		}

		parentID := ""
		if u.Parent != "" {
			for _, f := range r.scan.Folders {
				if f.Path == u.Parent {
					parentID = f.ID
					break
				}
			}
		}

		sf := ScannedFolder{ID: newID, Name: u.Name, Path: u.Path, ParentID: parentID}
		r.scan.Folders[newID] = sf
		r.fPaths[u.Path] = true
		r.newFolders = append(r.newFolders, sf)

		now := time.Now().UTC().Format(time.RFC3339Nano)
		r.events = append(r.events, ReconEvent{
			Name: "workspace-tree.changed",
			Payload: map[string]interface{}{
				"action":     "folder.external-created",
				"folderId":   newID,
				"folderName": u.Name,
				"folderPath": u.Path,
				"parentId":   parentID,
				"timestamp":  now,
			},
		})
	}
}

// emitWorkspaceMove emits a move/rename event with MoveInfo.
func (r *Reconciler) emitWorkspaceMove(id string, ws ScannedWorkspace, prevEntry SnapshotEntry, now string) {
	mi := DetectMoveInfo(prevEntry.Path, ws.RootPath)
	action := "workspace.external-moved"
	if mi.NameChanged && !mi.ParentChanged {
		action = "workspace.external-renamed"
	}
	r.events = append(r.events, ReconEvent{
		Name: "workspace-tree.changed",
		Payload: map[string]interface{}{
			"action":                    action,
			"workspaceId":               id,
			"workspaceName":             ws.Name,
			"workspacePath":             ws.RootPath,
			"workspaceRootPath":         ws.RootPath,
			"previousWorkspaceRootPath": prevEntry.Path,
			"nameChanged":               mi.NameChanged,
			"parentChanged":             mi.ParentChanged,
			"timestamp":                 now,
		},
	})
}

// emitFolderMove emits a move/rename event for a folder.
func (r *Reconciler) emitFolderMove(id string, f ScannedFolder, prevEntry SnapshotEntry, now string) {
	mi := DetectMoveInfo(prevEntry.Path, f.Path)
	action := "folder.external-moved"
	if mi.NameChanged && !mi.ParentChanged {
		action = "folder.external-renamed"
	}
	r.events = append(r.events, ReconEvent{
		Name: "workspace-tree.changed",
		Payload: map[string]interface{}{
			"action":       action,
			"folderId":     id,
			"folderName":   f.Name,
			"folderPath":   f.Path,
			"previousPath": prevEntry.Path,
			"parentId":     f.ParentID,
			"nameChanged":  mi.NameChanged,
			"parentChanged": mi.ParentChanged,
			"timestamp":    now,
		},
	})
}

func (r *Reconciler) prevWorkspace(id string) (SnapshotEntry, bool) {
	if r.prev == nil {
		return SnapshotEntry{}, false
	}
	entry, ok := r.prev.Workspaces[id]
	return entry, ok
}

func (r *Reconciler) prevFolder(id string) (SnapshotEntry, bool) {
	if r.prev == nil {
		return SnapshotEntry{}, false
	}
	entry, ok := r.prev.Folders[id]
	return entry, ok
}

func (r *Reconciler) prevWorkspaces() map[string]SnapshotEntry {
	if r.prev == nil {
		return map[string]SnapshotEntry{}
	}
	return r.prev.Workspaces
}

func (r *Reconciler) prevFolders() map[string]SnapshotEntry {
	if r.prev == nil {
		return map[string]SnapshotEntry{}
	}
	return r.prev.Folders
}

// isPathPrefix returns true if prefix is a path prefix of target.
func isPathPrefix(prefix, target string) bool {
	if prefix == target {
		return true
	}
	return len(target) > len(prefix) && target[:len(prefix)] == prefix && target[len(prefix)] == '/'
}

// sortUnmanagedByDepth sorts unmanaged directories by path depth (shallow first).
func sortUnmanagedByDepth(dirs []UnmanagedDirectory) {
	sort.SliceStable(dirs, func(i, j int) bool {
		return strings.Count(dirs[i].Path, "/") < strings.Count(dirs[j].Path, "/")
	})
}

// hasVerstakAncestorInSet checks if any ancestor of relPath has a .verstak in the given set.
func hasVerstakAncestorInSet(relPath string, existingVerstak map[string]bool) bool {
	p := parentPath(relPath)
	for p != "" {
		if existingVerstak[p] {
			return true
		}
		p = parentPath(p)
	}
	return false
}
