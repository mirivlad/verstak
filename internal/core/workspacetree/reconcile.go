package workspacetree

import (
	"time"
)

// Reconciler compares a scan against the previous snapshot and produces events.
type Reconciler struct {
	vaultDir string
	prev     *SemanticSnapshot
	scan     *ScanResult
	events   []ReconEvent
	warnings []TreeDiagnostic
}

// Reconcile compares the current scan with the previous snapshot.
func Reconcile(vaultDir string, prev *SemanticSnapshot, scan *ScanResult) *ReconResult {
	r := &Reconciler{
		vaultDir: vaultDir,
		prev:     prev,
		scan:     scan,
	}
	r.run()
	return &ReconResult{
		Snapshot: NewSnapshotFromScan(scan),
		Events:   r.events,
		Warnings: append(r.scan.Warnings, r.warnings...),
	}
}

func (r *Reconciler) run() {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Detect created and moved/renamed workspaces.
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

	// Detect deleted workspaces.
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

	// Detect created folders.
	for id, f := range r.scan.Folders {
		if _, hadPrev := r.prevFolder(id); !hadPrev {
			r.events = append(r.events, ReconEvent{
				Name: "workspace-tree.changed",
				Payload: map[string]interface{}{
					"action":    "folder.external-created",
					"folderId":  id,
					"folderName": f.Name,
					"folderPath": f.Path,
					"parentId":  f.ParentID,
					"timestamp": now,
				},
			})
		}
	}

	// Detect moved/renamed and deleted folders.
	for id, prevEntry := range r.prevFolders() {
		current, exists := r.scan.Folders[id]
		if !exists {
			r.events = append(r.events, ReconEvent{
				Name: "workspace-tree.changed",
				Payload: map[string]interface{}{
					"action":     "folder.external-deleted",
					"folderId":   id,
					"folderPath": prevEntry.Path,
					"timestamp":  now,
				},
			})
		} else if current.Path != prevEntry.Path {
			r.events = append(r.events, ReconEvent{
				Name: "workspace-tree.changed",
				Payload: map[string]interface{}{
					"action":           "folder.external-moved",
					"folderId":         id,
					"folderName":       current.Name,
					"folderPath":       current.Path,
					"previousPath":     prevEntry.Path,
					"parentId":         current.ParentID,
					"timestamp":        now,
				},
			})
		}
	}
}

// emitWorkspaceMove detects if a workspace was renamed or moved
// and emits the appropriate event.
// - Same parent directory → rename.
// - Different parent directory → move.
func (r *Reconciler) emitWorkspaceMove(id string, ws ScannedWorkspace, prevEntry SnapshotEntry, now string) {
	prevParent := parentPath(prevEntry.Path)
	newParent := parentPath(ws.RootPath)

	if prevParent == newParent {
		r.events = append(r.events, ReconEvent{
			Name: "workspace-tree.changed",
			Payload: map[string]interface{}{
				"action":                    "workspace.external-renamed",
				"workspaceId":               id,
				"workspaceName":             ws.Name,
				"workspacePath":             ws.RootPath,
				"workspaceRootPath":         ws.RootPath,
				"previousWorkspaceRootPath": prevEntry.Path,
				"timestamp":                 now,
			},
		})
	} else {
		r.events = append(r.events, ReconEvent{
			Name: "workspace-tree.changed",
			Payload: map[string]interface{}{
				"action":                    "workspace.external-moved",
				"workspaceId":               id,
				"workspaceName":             ws.Name,
				"workspacePath":             ws.RootPath,
				"workspaceRootPath":         ws.RootPath,
				"previousWorkspaceRootPath": prevEntry.Path,
				"timestamp":                 now,
			},
		})
	}
}

// prevWorkspace looks up a workspace in the previous snapshot.
func (r *Reconciler) prevWorkspace(id string) (SnapshotEntry, bool) {
	if r.prev == nil {
		return SnapshotEntry{}, false
	}
	entry, ok := r.prev.Workspaces[id]
	return entry, ok
}

// prevFolder looks up a folder in the previous snapshot.
func (r *Reconciler) prevFolder(id string) (SnapshotEntry, bool) {
	if r.prev == nil {
		return SnapshotEntry{}, false
	}
	entry, ok := r.prev.Folders[id]
	return entry, ok
}

// prevWorkspaces returns all workspaces from the previous snapshot.
func (r *Reconciler) prevWorkspaces() map[string]SnapshotEntry {
	if r.prev == nil {
		return map[string]SnapshotEntry{}
	}
	return r.prev.Workspaces
}

// prevFolders returns all folders from the previous snapshot.
func (r *Reconciler) prevFolders() map[string]SnapshotEntry {
	if r.prev == nil {
		return map[string]SnapshotEntry{}
	}
	return r.prev.Folders
}
