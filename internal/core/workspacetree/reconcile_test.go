package workspacetree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReconcileNewWorkspace(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("rec-new")
	scan := &ScanResult{Workspaces: map[string]ScannedWorkspace{
		wsID: {ID: wsID, Name: "Project", RootPath: "Project"},
	}}
	result := Reconcile(vault, nil, scan)
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Payload["action"] != "workspace.external-created" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

func TestReconcileDeletedWorkspace(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("rec-del")
	prev := &SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Workspaces: map[string]SnapshotEntry{wsID: {Path: "Project"}}}
	result := Reconcile(vault, prev, &ScanResult{Workspaces: map[string]ScannedWorkspace{}})
	if result.Events[0].Payload["action"] != "workspace.external-deleted" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

func TestReconcileRenamedWorkspace(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("rec-ren")
	prev := &SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Workspaces: map[string]SnapshotEntry{wsID: {Path: "OldName"}}}
	scan := &ScanResult{Workspaces: map[string]ScannedWorkspace{
		wsID: {ID: wsID, Name: "NewName", RootPath: "NewName"},
	}}
	result := Reconcile(vault, prev, scan)
	if result.Events[0].Payload["action"] != "workspace.external-renamed" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

func TestReconcileMovedWorkspace(t *testing.T) {
	vault := t.TempDir()
	f1ID := testUUID("mv-f1")
	wsID := testUUID("mv-ws")
	prev := &SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Folders:    map[string]SnapshotEntry{f1ID: {Path: "Clients"}},
		Workspaces: map[string]SnapshotEntry{wsID: {Path: "Clients/Client1"}},
	}
	scan := &ScanResult{
		Folders: map[string]ScannedFolder{f1ID: {ID: f1ID, Name: "Clients", Path: "Clients"}},
		Workspaces: map[string]ScannedWorkspace{
			wsID: {ID: wsID, Name: "Client1", RootPath: "Archive/Client1"},
		},
	}
	result := Reconcile(vault, prev, scan)
	found := false
	for _, evt := range result.Events {
		if a, _ := evt.Payload["action"].(string); a == "workspace.external-moved" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected workspace.external-moved event, got %+v", result.Events)
	}
}

func TestReconcileMoveAndRenameWorkspace(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("mvrn")
	prev := &SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Workspaces: map[string]SnapshotEntry{wsID: {Path: "Clients/Ivanov"}},
	}
	scan := &ScanResult{Workspaces: map[string]ScannedWorkspace{
		wsID: {ID: wsID, Name: "Ivanov 2025", RootPath: "Archive/Ivanov 2025"},
	}}
	result := Reconcile(vault, prev, scan)
	evt := result.Events[0]
	if evt.Payload["nameChanged"] != true || evt.Payload["parentChanged"] != true {
		t.Fatalf("expected nameChanged=true parentChanged=true, got %+v", evt.Payload)
	}
	if evt.Payload["previousWorkspaceRootPath"] != "Clients/Ivanov" {
		t.Fatalf("previousPath = %q", evt.Payload["previousWorkspaceRootPath"])
	}
}

func TestReconcileFolderCreated(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("rec-fc")
	scan := &ScanResult{Folders: map[string]ScannedFolder{
		fID: {ID: fID, Name: "Clients", Path: "Clients"},
	}}
	result := Reconcile(vault, nil, scan)
	if result.Events[0].Payload["action"] != "folder.external-created" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

func TestReconcileFolderDeleted(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("rec-fd")
	prev := &SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Folders: map[string]SnapshotEntry{fID: {Path: "Clients"}}}
	result := Reconcile(vault, prev, &ScanResult{Folders: map[string]ScannedFolder{}})
	if result.Events[0].Payload["action"] != "folder.external-deleted" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

func TestReconcileFolderRenamed(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("f-ren")
	prev := &SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Folders: map[string]SnapshotEntry{fID: {Path: "OldFolder"}}}
	scan := &ScanResult{Folders: map[string]ScannedFolder{
		fID: {ID: fID, Name: "NewFolder", Path: "NewFolder"},
	}}
	result := Reconcile(vault, prev, scan)
	if result.Events[0].Payload["action"] != "folder.external-renamed" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

func TestReconcileFolderMoved(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("f-mov")
	prev := &SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Folders: map[string]SnapshotEntry{fID: {Path: "A/B"}}}
	scan := &ScanResult{Folders: map[string]ScannedFolder{
		fID: {ID: fID, Name: "B", Path: "C/B", ParentID: "other"},
	}}
	result := Reconcile(vault, prev, scan)
	if result.Events[0].Payload["action"] != "folder.external-moved" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

func TestReconcileFolderMoveAndRename(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("fmvrn")
	prev := &SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Folders: map[string]SnapshotEntry{fID: {Path: "Clients/Active"}}}
	scan := &ScanResult{Folders: map[string]ScannedFolder{
		fID: {ID: fID, Name: "Active2025", Path: "Archive/Active2025", ParentID: "arch"},
	}}
	result := Reconcile(vault, prev, scan)
	evt := result.Events[0]
	if evt.Payload["nameChanged"] != true || evt.Payload["parentChanged"] != true {
		t.Fatalf("expected both flags true, got %+v", evt.Payload)
	}
}

func TestReconcileUnmanagedDirectoryBecomesFolder(t *testing.T) {
	vault := t.TempDir()
	mustMkdirAll(t, filepath.Join(vault, "Clients"))
	scan, _ := Scan(vault, nil)
	if len(scan.Unmanaged) != 1 {
		t.Fatalf("expected 1 unmanaged dir, got %d", len(scan.Unmanaged))
	}

	result := Reconcile(vault, nil, scan)
	if len(result.NewFolders) != 1 {
		t.Fatalf("expected 1 new folder created, got %d", len(result.NewFolders))
	}
	// Verify folder.json was written.
	markerPath := filepath.Join(vault, "Clients", ".verstak", "folder.json")
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("folder marker not created: %v", err)
	}
}

func TestReconcileNestedUnmanagedFolders(t *testing.T) {
	vault := t.TempDir()
	mustMkdirAll(t, filepath.Join(vault, "Clients", "Active"))
	scan, _ := Scan(vault, nil)
	result := Reconcile(vault, nil, scan)
	if len(result.NewFolders) < 2 {
		t.Fatalf("expected 2 new folders, got %d", len(result.NewFolders))
	}
}

func TestReconcileKnownOriginalCopyGetsNewUUID(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("ko-ws")
	createWS(t, vault, "Original", wsID)

	// Build initial snapshot.
	scan1, _ := Scan(vault, nil)
	snap := NewSnapshotFromScan(scan1)

	// Create a physical copy with same marker.
	copyDir := filepath.Join(vault, "Copy")
	mustMkdirAll(t, copyDir)
	verstakDir := filepath.Join(copyDir, ".verstak")
	mustMkdirAll(t, verstakDir)
	createWorkspaceMarker(t, verstakDir, wsID)

	// Scan with previous snapshot.
	scan2, _ := Scan(vault, &snap)

	// "Original" should keep wsID, "Copy" should get a new UUID.
	_, origOK := scan2.Workspaces[wsID]
	if !origOK {
		t.Fatalf("original should keep its UUID: %+v", scan2.Workspaces)
	}
	if len(scan2.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces after copy resolution, got %d", len(scan2.Workspaces))
	}
}

func TestReconcileKnownOriginalFolderCopyGetsNewUUID(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("ko-f")
	createFolder(t, vault, "Original", fID)
	scan1, _ := Scan(vault, nil)
	snap := NewSnapshotFromScan(scan1)

	copyDir := filepath.Join(vault, "Copy")
	mustMkdirAll(t, copyDir)
	verstakDir := filepath.Join(copyDir, ".verstak")
	mustMkdirAll(t, verstakDir)
	createFolderMarker(t, verstakDir, fID)

	scan2, _ := Scan(vault, &snap)
	if _, ok := scan2.Folders[fID]; !ok {
		t.Fatalf("original folder should keep its UUID")
	}
	if len(scan2.Folders) != 2 {
		t.Fatalf("expected 2 folders after copy resolution, got %d", len(scan2.Folders))
	}
}

func TestReconcileAmbiguousDuplicateRemainsConflict(t *testing.T) {
	vault := t.TempDir()
	dupID := testUUID("ambig")
	createWS(t, vault, "A", dupID)
	createWS(t, vault, "B", dupID)
	// No snapshot.
	scan, _ := Scan(vault, nil)
	if len(scan.Workspaces) != 0 {
		t.Fatalf("ambiguous duplicate should remove both, got %d", len(scan.Workspaces))
	}
	hasError := false
	for _, w := range scan.Warnings {
		if w.Code == "duplicate-id" && w.Level == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Fatal("expected duplicate-id error")
	}
}
