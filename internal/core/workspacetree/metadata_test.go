package workspacetree

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// metadataPath returns the UUID-keyed metadata path.
func metadataPath(vaultDir, workspaceID string) string {
	return filepath.Join(vaultDir, ".verstak", "workspaces", workspaceID+".json")
}

func writeMetadataFile(t *testing.T, vaultDir, workspaceID string, data map[string]interface{}) {
	t.Helper()
	path := metadataPath(vaultDir, workspaceID)
	mustMkdirAll(t, filepath.Dir(path))
	bytes, _ := json.Marshal(data)
	mustWriteFile(t, path, string(bytes))
}

func readMetadataFile(t *testing.T, vaultDir, workspaceID string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(metadataPath(vaultDir, workspaceID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	return m
}

func TestMetadataKeyedByWorkspaceID(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("meta-ws")
	createWS(t, vault, "Project", wsID)
	writeMetadataFile(t, vault, wsID, map[string]interface{}{
		"workspaceName": "Project",
		"features":      map[string]bool{"files": true},
	})
	m := readMetadataFile(t, vault, wsID)
	if m["workspaceName"] != "Project" {
		t.Fatalf("metadata = %+v", m)
	}
}

func TestMetadataSurvivesPhysicalRename(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("meta-ren")
	createWS(t, vault, "Project", wsID)
	writeMetadataFile(t, vault, wsID, map[string]interface{}{
		"workspaceName": "Project",
	})

	// Physically rename the directory.
	os.Rename(filepath.Join(vault, "Project"), filepath.Join(vault, "Renamed"))
	// Update the workspace marker name doesn't change — UUID stays.

	// Metadata is still at the same UUID-keyed path.
	m := readMetadataFile(t, vault, wsID)
	if m == nil {
		t.Fatal("metadata should survive rename")
	}
	if m["workspaceName"] != "Project" {
		t.Fatalf("metadata preserved: %+v", m)
	}
}

func TestMetadataSurvivesPhysicalMove(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("meta-mov")
	createFolder(t, vault, "Clients", testUUID("f-cli"))
	createWS(t, vault, "Clients/ClientX", wsID)
	writeMetadataFile(t, vault, wsID, map[string]interface{}{
		"workspaceName": "ClientX",
	})

	// Physically move to another folder.
	mustMkdirAll(t, filepath.Join(vault, "Archive"))
	newPath := filepath.Join(vault, "Archive", "ClientX")
	os.Rename(filepath.Join(vault, "Clients", "ClientX"), newPath)

	// Metadata at UUID path is unchanged.
	m := readMetadataFile(t, vault, wsID)
	if m == nil {
		t.Fatal("metadata should survive move")
	}
}

func TestMetadataNoMigrationOfOldFormat(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("nobase64")
	createWS(t, vault, "Project", wsID)

	// Old format path should NOT exist.
	oldPath := filepath.Join(vault, ".verstak", "workspaces", "UHJvamVjdA==.json")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("old base64 metadata path should not exist")
	}

	// New format path should NOT be auto-created by scanner.
	newPath := metadataPath(vault, wsID)
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatal("scanner should not auto-create metadata")
	}
}

func TestReconcileAfterPhysicalMoveUpdatesTree(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("rec-move")
	createWS(t, vault, "Clients/Client1", wsID)

	svc := NewService(vault, nil)
	svc.Initialize()

	// Verify initial tree.
	ws, ok := svc.GetWorkspaceByID(wsID)
	if !ok || ws.RootPath != "Clients/Client1" {
		t.Fatalf("initial = %+v", ws)
	}

	// Physical move.
	mustMkdirAll(t, filepath.Join(vault, "Archive"))
	os.Rename(filepath.Join(vault, "Clients", "Client1"), filepath.Join(vault, "Archive", "Client1"))

	svc.RescanWorkspaceTree()
	ws, ok = svc.GetWorkspaceByID(wsID)
	if !ok || ws.RootPath != "Archive/Client1" {
		t.Fatalf("after move = %+v", ws)
	}
}
