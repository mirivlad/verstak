package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func newTestVault(t *testing.T) string {
	t.Helper()
	v := filepath.Join(t.TempDir(), "vault")
	os.MkdirAll(v, 0o755)
	os.MkdirAll(filepath.Join(v, ".verstak", "sync"), 0o755)
	return v
}

func writeMarker(t *testing.T, dir, markerType, id string) {
	t.Helper()
	os.MkdirAll(filepath.Join(dir, ".verstak"), 0o755)
	m := map[string]interface{}{"schemaVersion": 1}
	if markerType == "workspace" {
		m["workspaceId"] = id
		os.WriteFile(filepath.Join(dir, ".verstak", "workspace.json"), mustJSON(m), 0o644)
	} else {
		m["folderId"] = id
		os.WriteFile(filepath.Join(dir, ".verstak", "folder.json"), mustJSON(m), 0o644)
	}
}

func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func TestSyncNestedFolderCreate(t *testing.T) {
	vault := newTestVault(t)

	f1ID := uuid.NewString()
	f2ID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "Clients"), 0o755)
	writeMarker(t, filepath.Join(vault, "Clients"), "folder", f1ID)
	os.MkdirAll(filepath.Join(vault, "Clients", "Active"), 0o755)
	writeMarker(t, filepath.Join(vault, "Clients", "Active"), "folder", f2ID)

	scan, warnings, err := scanVault(vault, newSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}
	if len(scan.Folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(scan.Folders))
	}
	if _, ok := scan.Folders[f1ID]; !ok {
		t.Fatal("f1 not found")
	}
}

func TestSyncNestedWorkspaceCreate(t *testing.T) {
	vault := newTestVault(t)

	fID := uuid.NewString()
	wsID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "Clients", "Client1"), 0o755)
	writeMarker(t, filepath.Join(vault, "Clients"), "folder", fID)
	writeMarker(t, filepath.Join(vault, "Clients", "Client1"), "workspace", wsID)

	scan, _, _ := scanVault(vault, newSnapshot())
	if len(scan.Folders) != 1 {
		t.Fatalf("expected 1 folder, got %d", len(scan.Folders))
	}
	if len(scan.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(scan.Workspaces))
	}
}

func TestSyncGenericFolderAndWorkspaceFolderAreDistinct(t *testing.T) {
	vault := newTestVault(t)

	fID := uuid.NewString()
	wsID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "Clients"), 0o755)
	writeMarker(t, filepath.Join(vault, "Clients"), "folder", fID)
	os.MkdirAll(filepath.Join(vault, "Clients", "Client1", "Notes"), 0o755)
	writeMarker(t, filepath.Join(vault, "Clients", "Client1"), "workspace", wsID)
	os.WriteFile(filepath.Join(vault, "Clients", "Client1", "Notes", "doc.md"), []byte("# Doc"), 0o644)

	scan, _, _ := scanVault(vault, newSnapshot())
	// Workspace folder is in Folders map.
	if scan.Folders[fID].FolderID != fID {
		t.Fatal("folder should be in Folders")
	}
	// Generic folder (Notes) is in Entries.
	foundNotes := false
	for path, e := range scan.Entries {
		if strings.Contains(path, "Notes") && e.Type == EntityFolder {
			foundNotes = true
		}
	}
	if !foundNotes {
		t.Fatal("generic Notes folder should be in Entries")
	}
}

func TestSyncFolderRename(t *testing.T) {
	vault := newTestVault(t)

	fID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "OldName"), 0o755)
	writeMarker(t, filepath.Join(vault, "OldName"), "folder", fID)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)
	s := NewService(vault, "dev")
	s.saveSnapshot(prev)

	// Rename.
	os.Rename(filepath.Join(vault, "OldName"), filepath.Join(vault, "NewName"))
	writeMarker(t, filepath.Join(vault, "NewName"), "folder", fID)

	curScan, _, _ := scanVault(vault, prev)
	cur := snapshotFromScan(curScan, prev, true)

	ops, _, err := diffSnapshots(prev, cur, "dev", vault)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, op := range ops {
		if op.EntityType == EntityWorkspaceFolder && op.EntityID == fID {
			found = true
		}
	}
	if !found {
		t.Fatalf("rename operation not found in %d ops: %+v", len(ops), ops)
	}
}

func TestSyncFolderMove(t *testing.T) {
	vault := newTestVault(t)

	f1ID := uuid.NewString()
	f2ID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "A"), 0o755)
	writeMarker(t, filepath.Join(vault, "A"), "folder", f1ID)
	os.MkdirAll(filepath.Join(vault, "B"), 0o755)
	writeMarker(t, filepath.Join(vault, "B"), "folder", f2ID)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)

	// Move A into B.
	os.Rename(filepath.Join(vault, "A"), filepath.Join(vault, "B", "A"))
	writeMarker(t, filepath.Join(vault, "B", "A"), "folder", f1ID)

	curScan, _, _ := scanVault(vault, prev)
	cur := snapshotFromScan(curScan, prev, true)

	ops, _, err := diffSnapshots(prev, cur, "dev", vault)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, op := range ops {
		if op.EntityType == EntityWorkspaceFolder && op.EntityID == f1ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("move operation not found in %d ops", len(ops))
	}
}

func TestSyncOldSnapshotVersionRejected(t *testing.T) {
	vault := newTestVault(t)
	path := filepath.Join(vault, ".verstak", "sync", "snapshot.json")
	os.WriteFile(path, []byte(`{"version":1,"entries":{}}`), 0o600)

	s := NewService(vault, "dev")
	_, exists, err := s.loadSnapshot()
	if err == nil || exists {
		t.Fatalf("old version should be rejected: err=%v exists=%v", err, exists)
	}
}

func TestSyncSubtreeTrash(t *testing.T) {
	vault := newTestVault(t)
	fID := uuid.NewString()
	wsID := uuid.NewString()

	os.MkdirAll(filepath.Join(vault, "Clients", "Client1"), 0o755)
	writeMarker(t, filepath.Join(vault, "Clients"), "folder", fID)
	writeMarker(t, filepath.Join(vault, "Clients", "Client1"), "workspace", wsID)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)

	// Trash folder subtree.
	trashDir := filepath.Join(vault, ".verstak", "trash", "tree", uuid.NewString())
	os.MkdirAll(filepath.Join(trashDir, "content"), 0o755)
	os.Rename(filepath.Join(vault, "Clients"), filepath.Join(trashDir, "content", "Clients"))

	curScan, _, _ := scanVault(vault, prev)
	cur := snapshotFromScan(curScan, prev, true)

	if _, active := cur.Folders[fID]; active {
		t.Fatal("folder should not be active after trash")
	}
}

func TestSyncWorkspaceMove(t *testing.T) {
	vault := newTestVault(t)
	fID := uuid.NewString()
	wsID := uuid.NewString()

	os.MkdirAll(filepath.Join(vault, "Clients", "Client1"), 0o755)
	writeMarker(t, filepath.Join(vault, "Clients"), "folder", fID)
	writeMarker(t, filepath.Join(vault, "Clients", "Client1"), "workspace", wsID)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)

	os.Rename(filepath.Join(vault, "Clients", "Client1"), filepath.Join(vault, "Client1"))
	writeMarker(t, filepath.Join(vault, "Client1"), "workspace", wsID)

	curScan, _, _ := scanVault(vault, prev)
	cur := snapshotFromScan(curScan, prev, true)

	ops, _, err := diffSnapshots(prev, cur, "dev", vault)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, op := range ops {
		if op.EntityType == EntityWorkspace && op.EntityID == wsID {
			found = true
		}
	}
	if !found {
		t.Fatalf("workspace move operation not found in %d ops", len(ops))
	}
}

func TestSyncDuplicateFolderID(t *testing.T) {
	vault := newTestVault(t)
	fID := uuid.NewString()

	os.MkdirAll(filepath.Join(vault, "A"), 0o755)
	writeMarker(t, filepath.Join(vault, "A"), "folder", fID)
	os.MkdirAll(filepath.Join(vault, "B"), 0o755)
	writeMarker(t, filepath.Join(vault, "B"), "folder", fID)

	scan, warnings, _ := scanVault(vault, newSnapshot())
	if len(scan.Folders) != 1 {
		t.Fatalf("duplicate folder ID: expected 1, got %d", len(scan.Folders))
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "duplicate-folder-id") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected duplicate-folder-id warning")
	}
}

func TestSyncDuplicateWorkspaceID(t *testing.T) {
	vault := newTestVault(t)
	wsID := uuid.NewString()

	os.MkdirAll(filepath.Join(vault, "A"), 0o755)
	writeMarker(t, filepath.Join(vault, "A"), "workspace", wsID)
	os.MkdirAll(filepath.Join(vault, "B"), 0o755)
	writeMarker(t, filepath.Join(vault, "B"), "workspace", wsID)

	scan, warnings, _ := scanVault(vault, newSnapshot())
	if len(scan.Workspaces) != 1 {
		t.Fatalf("duplicate WS ID: expected 1, got %d", len(scan.Workspaces))
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "duplicate-workspace-id") {
			found = true
		}
	}
	if !found {
		t.Fatal("expected duplicate-workspace-id warning")
	}
}

func TestSyncOperationOrdering(t *testing.T) {
	vault := newTestVault(t)

	fRootID := uuid.NewString()
	fNestedID := uuid.NewString()
	wsID := uuid.NewString()

	os.MkdirAll(filepath.Join(vault, "Clients", "Active", "Client1", "Notes"), 0o755)
	writeMarker(t, filepath.Join(vault, "Clients"), "folder", fRootID)
	writeMarker(t, filepath.Join(vault, "Clients", "Active"), "folder", fNestedID)
	writeMarker(t, filepath.Join(vault, "Clients", "Active", "Client1"), "workspace", wsID)
	os.WriteFile(filepath.Join(vault, "Clients", "Active", "Client1", "Notes", "doc.md"), []byte("x"), 0o644)

	scan, _, _ := scanVault(vault, newSnapshot())
	emptyPrev := newSnapshot()
	emptyPrev.WorkspacesInitialized = true
	cur := snapshotFromScan(scan, emptyPrev, true)

	ops, _, _ := diffSnapshots(emptyPrev, cur, "dev", vault)

	// Check ordering: workspace-folders before workspaces before files.
	var lastKind string
	for _, op := range ops {
		switch {
		case op.EntityType == EntityWorkspaceFolder:
			if lastKind == "workspace" || lastKind == "file" {
				t.Errorf("folder after %s", lastKind)
			}
			lastKind = "folder"
		case op.EntityType == EntityWorkspace:
			if lastKind == "file" {
				t.Errorf("workspace after file")
			}
			lastKind = "workspace"
		default:
			lastKind = "file"
		}
	}
	if len(ops) == 0 {
		t.Fatal("expected some operations")
	}
}
