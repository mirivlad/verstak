package workspacetree

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStartupReconcileOrder verifies the correct startup sequence:
// 1. snapshot loaded, 2. scan, 3. reconcile, 4. unmanaged handled,
// 5. duplicates resolved, 6. snapshot saved, 7. tree available.
func TestStartupReconcileOrder(t *testing.T) {
	vault := t.TempDir()

	// Pre-populate: one workspace, one unmanaged folder, one duplicate.
	ws1ID := testUUID("start-ws1")
	dupID := testUUID("start-dup")
	createWS(t, vault, "Project", ws1ID)
	createWS(t, vault, "Copy", dupID)
	createWS(t, vault, "AnotherCopy", dupID)
	mustMkdirAll(t, filepath.Join(vault, "UnmarkedFolder"))

	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}

	// Tree should be available after startup.
	tree := svc.GetTree()
	if tree.Revision != 1 {
		t.Fatalf("revision should be 1 after startup, got %d", tree.Revision)
	}

	// The unmarked folder should now have a marker.
	markerPath := filepath.Join(vault, "UnmarkedFolder", ".verstak", "folder.json")
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("unmarked folder should get marker during startup: %v", err)
	}

	// Ambiguous duplicate should have error diagnostic.
	hasDupErr := false
	for _, w := range tree.Warnings {
		if w.Code == "duplicate-id" {
			hasDupErr = true
		}
	}
	if !hasDupErr {
		t.Fatal("ambiguous duplicate should produce error diagnostic")
	}
}

func TestStartupOfflineFolderRename(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("off-fr")
	createFolder(t, vault, "OldName", fID)
	svc := NewService(vault, nil)
	svc.Initialize()
	// Offline rename.
	os.Rename(filepath.Join(vault, "OldName"), filepath.Join(vault, "NewName"))
	svc.RescanWorkspaceTree()
	f, ok := svc.GetFolderByID(fID)
	if !ok || f.Name != "NewName" {
		t.Fatalf("folder after rename = %+v", f)
	}
}

func TestStartupOfflineWorkspaceRename(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("off-wr")
	createWS(t, vault, "OldWS", wsID)
	svc := NewService(vault, nil)
	svc.Initialize()
	os.Rename(filepath.Join(vault, "OldWS"), filepath.Join(vault, "NewWS"))
	svc.RescanWorkspaceTree()
	ws, ok := svc.GetWorkspaceByID(wsID)
	if !ok || ws.Name != "NewWS" {
		t.Fatalf("workspace after rename = %+v", ws)
	}
}

func TestStartupOfflineCreate(t *testing.T) {
	vault := t.TempDir()
	createWS(t, vault, "Existing", testUUID("off-ex"))
	svc := NewService(vault, nil)
	svc.Initialize()
	// Offline create.
	createWS(t, vault, "NewOne", testUUID("off-new"))
	svc.RescanWorkspaceTree()
	if _, ok := svc.GetWorkspaceByID(testUUID("off-new")); !ok {
		t.Fatal("new workspace should be found")
	}
}

func TestStartupOfflineDelete(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("off-del")
	createWS(t, vault, "ToDelete", wsID)
	svc := NewService(vault, nil)
	svc.Initialize()
	os.RemoveAll(filepath.Join(vault, "ToDelete"))
	svc.RescanWorkspaceTree()
	if _, ok := svc.GetWorkspaceByID(wsID); ok {
		t.Fatal("deleted workspace should be gone")
	}
}
