package workspacetree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServiceInitializeAndGetTree(t *testing.T) {
	vault := t.TempDir()
	createWS(t, vault, "Project", testUUID("svc-ws"))
	createFolder(t, vault, "Clients", testUUID("svc-f"))
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	tree := svc.GetTree()
	if len(tree.Roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree.Roots))
	}
}

func TestServiceGetByID(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("svc-byid")
	createWS(t, vault, "Project", wsID)
	svc := NewService(vault, nil)
	svc.Initialize()
	ws, ok := svc.GetWorkspaceByID(wsID)
	if !ok || ws.Name != "Project" {
		t.Fatalf("GetWorkspaceByID = %+v, %v", ws, ok)
	}
}

func TestServiceInternalMutationBaseline(t *testing.T) {
	vault := t.TempDir()
	ws1ID := testUUID("mut-ws1")
	ws2ID := testUUID("mut-ws2")
	createWS(t, vault, "Project", ws1ID)
	svc := NewService(vault, nil)
	svc.Initialize()
	svc.BeginInternalMutation()
	createWS(t, vault, "NewOne", ws2ID)
	svc.EndInternalMutationAndRefreshBaseline(nil)
	tree := svc.GetTree()
	found := false
	for _, r := range tree.Roots {
		if r.ID == ws2ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("%s should appear after internal mutation refresh", ws2ID)
	}
}

func TestStartupOfflineChanges(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("off-f")
	ws1ID := testUUID("off-ws1")
	ws2ID := testUUID("off-ws2")
	ws3ID := testUUID("off-ws3")
	createFolder(t, vault, "Clients", fID)
	createWS(t, vault, "Clients/Client1", ws1ID)
	createWS(t, vault, "Project", ws2ID)
	svc := NewService(vault, nil)
	svc.Initialize()

	// Offline: rename, create, delete.
	os.Rename(filepath.Join(vault, "Project"), filepath.Join(vault, "ArchiveProject"))
	createWS(t, vault, "NewOne", ws3ID)
	os.RemoveAll(filepath.Join(vault, "Clients", "Client1"))

	svc.RescanWorkspaceTree()
	tree := svc.GetTree()
	if tree.Revision != 2 {
		t.Fatalf("revision = %d", tree.Revision)
	}
	if ws, ok := svc.GetWorkspaceByID(ws2ID); !ok || ws.RootPath != "ArchiveProject" {
		t.Fatalf("ws2 = %+v", ws)
	}
}

func TestWatcherDebounceDoesNotFireDuringInternalMutation(t *testing.T) {
	vault := t.TempDir()
	createWS(t, vault, "Project", testUUID("deb-ws"))
	svc := NewService(vault, nil)
	svc.Initialize()
	svc.BeginInternalMutation()
	svc.OnFileChanged()
	svc.EndInternalMutationAndRefreshBaseline(nil)
	if len(svc.GetTree().Roots) != 1 {
		t.Fatalf("expected 1 root")
	}
}
