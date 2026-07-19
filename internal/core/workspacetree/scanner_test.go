package workspacetree

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// testUUID generates a deterministic UUID v5 from a short test tag.
func testUUID(tag string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(tag+"@verstak.test")).String()
}

// ── Scanner tests ────────────────────────────────────────────────────────────

func TestScanEmptyVault(t *testing.T) {
	vault := t.TempDir()
	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 0 || len(result.Workspaces) != 0 {
		t.Fatalf("empty vault should have no entities, got folders=%d workspaces=%d",
			len(result.Folders), len(result.Workspaces))
	}
}

func TestScanSingleWorkspace(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("ws-1")
	createWS(t, vault, "Project", wsID)

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(result.Workspaces))
	}
	ws, ok := result.Workspaces[wsID]
	if !ok {
		t.Fatalf("workspace %s not found: %+v", wsID, result.Workspaces)
	}
	if ws.Name != "Project" || ws.RootPath != "Project" {
		t.Fatalf("workspace = %+v, want Name=Project RootPath=Project", ws)
	}
}

func TestScanWorkspaceRecursionStops(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("ws-stop")
	createWS(t, vault, "Project", wsID)
	mustMkdirAll(t, filepath.Join(vault, "Project", "Notes"))
	mustMkdirAll(t, filepath.Join(vault, "Project", "Files"))

	// Folder marker inside workspace — ignored.
	ignoredID := testUUID("ignored")
	createFolderMarker(t, filepath.Join(vault, "Project", ".verstak"), ignoredID)

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(result.Workspaces))
	}
	if len(result.Folders) != 0 {
		t.Fatalf("folders inside workspace should be ignored, got %d", len(result.Folders))
	}
}

func TestScanFolders(t *testing.T) {
	vault := t.TempDir()
	f1ID := testUUID("f-1")
	f2ID := testUUID("f-2")
	wsID := testUUID("ws-1")
	createFolder(t, vault, "Clients", f1ID)
	createFolder(t, vault, "Clients/Active", f2ID)
	createWS(t, vault, "Clients/Active/Client1", wsID)

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 2 {
		t.Fatalf("expected 2 folders, got %d: %+v", len(result.Folders), result.Folders)
	}
	f1, ok := result.Folders[f1ID]
	if !ok || f1.Name != "Clients" || f1.ParentID != "" {
		t.Fatalf("folder f-1 = %+v", f1)
	}
	f2, ok := result.Folders[f2ID]
	if !ok || f2.Name != "Active" || f2.ParentID != f1ID {
		t.Fatalf("folder f-2 = %+v (want parent=%s)", f2, f1ID)
	}
	if len(result.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(result.Workspaces))
	}
}

func TestScanUnmarkedDirectoriesNotListed(t *testing.T) {
	vault := t.TempDir()
	mustMkdirAll(t, filepath.Join(vault, "Unmarked"))
	mustMkdirAll(t, filepath.Join(vault, "Unmarked", "Nested"))

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 0 {
		t.Fatalf("unmarked dirs should not be folders, got %d", len(result.Folders))
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("unmarked dirs should not be workspaces, got %d", len(result.Workspaces))
	}
}

func TestScanHiddenDirsSkipped(t *testing.T) {
	vault := t.TempDir()
	mustMkdirAll(t, filepath.Join(vault, ".hidden"))
	// Workspace inside hidden dir should not be found because hidden dirs are skipped.
	id := testUUID("hidden-ws")
	createWS(t, vault, ".hidden", id)

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("hidden dirs should be skipped, got %d workspaces", len(result.Workspaces))
	}
}

func TestScanSymlinkSkipped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test requires unix")
	}
	vault := t.TempDir()
	target := filepath.Join(t.TempDir(), "outside")
	mustMkdirAll(t, target)
	linkedID := testUUID("linked")
	createWS(t, t.TempDir(), "outside", linkedID)
	if err := os.Symlink(target, filepath.Join(vault, "Linked")); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("symlinks should be skipped, got %d workspaces", len(result.Workspaces))
	}
}

func TestScanDuplicateWorkspaceUUID(t *testing.T) {
	vault := t.TempDir()
	dupID := testUUID("dup")
	createWS(t, vault, "Project", dupID)
	createWS(t, vault, "Copy", dupID) // same UUID

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 1 {
		t.Fatalf("duplicate UUID: first wins, expected 1 workspace, got %d", len(result.Workspaces))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for duplicate UUID")
	}
}

func TestScanCorruptedMarker(t *testing.T) {
	vault := t.TempDir()
	wsDir := filepath.Join(vault, "Project")
	mustMkdirAll(t, wsDir)
	verstakDir := filepath.Join(wsDir, ".verstak")
	mustMkdirAll(t, verstakDir)
	mustWriteFile(t, filepath.Join(verstakDir, "workspace.json"), "not json")

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("corrupted marker should not produce workspace, got %d", len(result.Workspaces))
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for corrupted marker")
	}
}

func TestScanBothMarkersInSameDir(t *testing.T) {
	vault := t.TempDir()
	wsDir := filepath.Join(vault, "Both")
	mustMkdirAll(t, wsDir)
	verstakDir := filepath.Join(wsDir, ".verstak")
	mustMkdirAll(t, verstakDir)
	wsID := testUUID("both-ws")
	fID := testUUID("both-f")
	createWorkspaceMarker(t, verstakDir, wsID)
	createFolderMarker(t, verstakDir, fID)

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 1 {
		t.Fatalf("workspace marker should win, got %d workspaces", len(result.Workspaces))
	}
	if len(result.Folders) != 0 {
		t.Fatalf("folder marker should be ignored when workspace marker present, got %d",
			len(result.Folders))
	}
}

func TestScanNestedFolders(t *testing.T) {
	vault := t.TempDir()
	faID := testUUID("f-a")
	fbID := testUUID("f-b")
	fcID := testUUID("f-c")
	wsID := testUUID("ws-deep")
	createFolder(t, vault, "A", faID)
	createFolder(t, vault, "A/B", fbID)
	createFolder(t, vault, "A/B/C", fcID)
	createWS(t, vault, "A/B/C/Deep", wsID)

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 3 {
		t.Fatalf("expected 3 folders, got %d", len(result.Folders))
	}
	if len(result.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(result.Workspaces))
	}
	fb, ok := result.Folders[fbID]
	if !ok || fb.ParentID != faID {
		t.Fatalf("folder B parent = %q, want %s", fb.ParentID, faID)
	}
	fc, ok := result.Folders[fcID]
	if !ok || fc.ParentID != fbID {
		t.Fatalf("folder C parent = %q, want %s", fc.ParentID, fbID)
	}
}

func TestScanIsReadOnly(t *testing.T) {
	vault := t.TempDir()
	createFolder(t, vault, "Clients", testUUID("ro-f"))
	createWS(t, vault, "Project", testUUID("ro-ws"))

	before := captureFiles(t, vault)
	_, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	after := captureFiles(t, vault)

	if !filesEqual(before, after) {
		t.Fatal("scanner modified the filesystem")
	}
}

func TestScanSameNamesDifferentParents(t *testing.T) {
	vault := t.TempDir()
	createFolder(t, vault, "Clients/Active", testUUID("active-1"))
	createFolder(t, vault, "Archive/Active", testUUID("active-2"))
	createWS(t, vault, "Clients/Active/Same", testUUID("same-1"))
	createWS(t, vault, "Archive/Active/Same", testUUID("same-2"))

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(result.Folders))
	}
	if len(result.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(result.Workspaces))
	}
}

func TestScanInvalidUUIDInMarker(t *testing.T) {
	vault := t.TempDir()
	wsDir := filepath.Join(vault, "Bad")
	mustMkdirAll(t, wsDir)
	verstakDir := filepath.Join(wsDir, ".verstak")
	mustMkdirAll(t, verstakDir)
	mustWriteFile(t, filepath.Join(verstakDir, "workspace.json"),
		`{"schemaVersion":1,"workspaceId":"not-a-uuid"}`)

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("invalid UUID should not produce workspace")
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for invalid UUID")
	}
}

// ── Snapshot tests ───────────────────────────────────────────────────────────

func TestSnapshotWriteReadRoundtrip(t *testing.T) {
	vault := t.TempDir()
	f1 := testUUID("snap-f")
	ws1 := testUUID("snap-ws")
	original := SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Folders:       map[string]SnapshotEntry{f1: {Path: "Clients"}},
		Workspaces:    map[string]SnapshotEntry{ws1: {Path: "Project"}},
	}
	if err := WriteSnapshot(vault, &original); err != nil {
		t.Fatal(err)
	}

	read, err := ReadSnapshot(vault)
	if err != nil {
		t.Fatal(err)
	}
	if read == nil {
		t.Fatal("snapshot not found")
	}
	if read.Folders[f1].Path != "Clients" {
		t.Fatalf("folder path = %q", read.Folders[f1].Path)
	}
	if read.Workspaces[ws1].Path != "Project" {
		t.Fatalf("workspace path = %q", read.Workspaces[ws1].Path)
	}
}

func TestSnapshotMissingFileReturnsNil(t *testing.T) {
	vault := t.TempDir()
	snap, err := ReadSnapshot(vault)
	if err != nil {
		t.Fatal(err)
	}
	if snap != nil {
		t.Fatal("expected nil snapshot for missing file")
	}
}

func TestSnapshotAtomicWrite(t *testing.T) {
	vault := t.TempDir()
	snap := SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Folders:       map[string]SnapshotEntry{testUUID("atom-f"): {Path: "A"}},
		Workspaces:    map[string]SnapshotEntry{testUUID("atom-ws"): {Path: "B"}},
	}
	if err := WriteSnapshot(vault, &snap); err != nil {
		t.Fatal(err)
	}
	tmpPath := filepath.Join(vault, ".verstak", "cache", "workspace-tree.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatal("tmp file should not exist after atomic write")
	}
}

func TestRemoveSnapshot(t *testing.T) {
	vault := t.TempDir()
	snap := SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Workspaces:    map[string]SnapshotEntry{testUUID("rm-ws"): {Path: "A"}},
	}
	if err := WriteSnapshot(vault, &snap); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSnapshot(vault); err != nil {
		t.Fatal(err)
	}
	read, _ := ReadSnapshot(vault)
	if read != nil {
		t.Fatal("snapshot should be gone")
	}
}

func TestSnapshotCorruptedJSONReturnsNil(t *testing.T) {
	vault := t.TempDir()
	path := filepath.Join(vault, ".verstak", "cache", "workspace-tree.json")
	mustMkdirAll(t, filepath.Dir(path))
	mustWriteFile(t, path, "not json")
	snap, err := ReadSnapshot(vault)
	if err != nil {
		t.Fatal(err)
	}
	if snap != nil {
		t.Fatal("corrupted snapshot should return nil")
	}
}

// ── Reconciliation tests ─────────────────────────────────────────────────────

func TestReconcileNewWorkspace(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("rec-new")
	scan := &ScanResult{
		Workspaces: map[string]ScannedWorkspace{
			wsID: {ID: wsID, Name: "Project", RootPath: "Project"},
		},
	}
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
	prev := &SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Workspaces:    map[string]SnapshotEntry{wsID: {Path: "Project"}},
	}
	scan := &ScanResult{Workspaces: map[string]ScannedWorkspace{}}
	result := Reconcile(vault, prev, scan)
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Payload["action"] != "workspace.external-deleted" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

func TestReconcileRenamedWorkspace(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("rec-ren")
	prev := &SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Workspaces:    map[string]SnapshotEntry{wsID: {Path: "OldName"}},
	}
	scan := &ScanResult{
		Workspaces: map[string]ScannedWorkspace{
			wsID: {ID: wsID, Name: "NewName", RootPath: "NewName"},
		},
	}
	result := Reconcile(vault, prev, scan)
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	action := result.Events[0].Payload["action"].(string)
	if action != "workspace.external-renamed" {
		t.Fatalf("unexpected action: %q", action)
	}
}

func TestReconcileMovedWorkspace(t *testing.T) {
	vault := t.TempDir()
	f1ID := testUUID("mv-f1")
	f2ID := testUUID("mv-f2")
	wsID := testUUID("mv-ws")
	prev := &SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Folders:       map[string]SnapshotEntry{f1ID: {Path: "Clients"}, f2ID: {Path: "Clients/Active"}},
		Workspaces:    map[string]SnapshotEntry{wsID: {Path: "Clients/Active/Client1"}},
	}
	scan := &ScanResult{
		Folders: map[string]ScannedFolder{
			f1ID: {ID: f1ID, Name: "Clients", Path: "Clients"},
			f2ID: {ID: f2ID, Name: "Active", Path: "Clients/Active"},
		},
		Workspaces: map[string]ScannedWorkspace{
			wsID: {ID: wsID, Name: "Client1", RootPath: "Archive/Client1"},
		},
	}
	result := Reconcile(vault, prev, scan)
	found := false
	for _, evt := range result.Events {
		a, _ := evt.Payload["action"].(string)
		if a == "workspace.external-moved" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected workspace.external-moved event, got %+v", result.Events)
	}
}

func TestReconcileWorkspaceReplaced(t *testing.T) {
	vault := t.TempDir()
	ws1ID := testUUID("rep-ws1")
	ws2ID := testUUID("rep-ws2")
	prev := &SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Workspaces:    map[string]SnapshotEntry{ws1ID: {Path: "Project"}},
	}
	scan := &ScanResult{
		Workspaces: map[string]ScannedWorkspace{
			ws1ID: {ID: ws1ID, Name: "Renamed", RootPath: "Renamed"},
			ws2ID: {ID: ws2ID, Name: "Project", RootPath: "Project"},
		},
	}
	result := Reconcile(vault, prev, scan)
	if len(result.Events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(result.Events))
	}
}

func TestReconcileFolderCreated(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("rec-fc")
	scan := &ScanResult{
		Folders: map[string]ScannedFolder{
			fID: {ID: fID, Name: "Clients", Path: "Clients"},
		},
	}
	result := Reconcile(vault, nil, scan)
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Payload["action"] != "folder.external-created" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

func TestReconcileFolderDeleted(t *testing.T) {
	vault := t.TempDir()
	fID := testUUID("rec-fd")
	prev := &SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Folders:       map[string]SnapshotEntry{fID: {Path: "Clients"}},
	}
	result := Reconcile(vault, prev, &ScanResult{Folders: map[string]ScannedFolder{}})
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Payload["action"] != "folder.external-deleted" {
		t.Fatalf("action = %q", result.Events[0].Payload["action"])
	}
}

// ── Tree builder tests ───────────────────────────────────────────────────────

func TestBuildTreeFlatWorkspaces(t *testing.T) {
	ws1 := testUUID("tree-ws1")
	ws2 := testUUID("tree-ws2")
	scan := &ScanResult{
		Workspaces: map[string]ScannedWorkspace{
			ws1: {ID: ws1, Name: "Project", RootPath: "Project"},
			ws2: {ID: ws2, Name: "Test", RootPath: "Test"},
		},
	}
	tree := BuildTree(scan, "", 1)
	if len(tree.Roots) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(tree.Roots))
	}
	if tree.Roots[0].Name != "Project" || tree.Roots[1].Name != "Test" {
		t.Fatalf("order: %s, %s", tree.Roots[0].Name, tree.Roots[1].Name)
	}
}

func TestBuildTreeFoldersAndWorkspaces(t *testing.T) {
	f1 := testUUID("tbf-1")
	f2 := testUUID("tbf-2")
	ws1 := testUUID("tbf-ws1")
	ws2 := testUUID("tbf-ws2")
	scan := &ScanResult{
		Folders: map[string]ScannedFolder{
			f1: {ID: f1, Name: "Clients", Path: "Clients"},
			f2: {ID: f2, Name: "Archive", Path: "Archive"},
		},
		Workspaces: map[string]ScannedWorkspace{
			ws1: {ID: ws1, Name: "Project", RootPath: "Project"},
			ws2: {ID: ws2, Name: "Client1", RootPath: "Clients/Client1"},
		},
	}
	tree := BuildTree(scan, ws1, 1)
	if len(tree.Roots) < 2 {
		t.Fatalf("expected roots for folders, got %d", len(tree.Roots))
	}
	if tree.CurrentWorkspaceID != ws1 {
		t.Fatalf("current = %q, want %s", tree.CurrentWorkspaceID, ws1)
	}
	if tree.Roots[0].Kind != "folder" {
		t.Fatalf("first root should be folder, got %s", tree.Roots[0].Kind)
	}
	if tree.Roots[0].Name != "Archive" {
		t.Fatalf("expected Archive first, got %s", tree.Roots[0].Name)
	}
}

func TestBuildTreeFolderChildWorkspace(t *testing.T) {
	fID := testUUID("tbfc-f")
	wsID := testUUID("tbfc-ws")
	scan := &ScanResult{
		Folders: map[string]ScannedFolder{
			fID: {ID: fID, Name: "Clients", Path: "Clients"},
		},
		Workspaces: map[string]ScannedWorkspace{
			wsID: {ID: wsID, Name: "Client1", RootPath: "Clients/Client1"},
		},
	}
	tree := BuildTree(scan, "", 1)
	if len(tree.Roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree.Roots))
	}
	folder := tree.Roots[0]
	if folder.Kind != "folder" || len(folder.Children) != 1 {
		t.Fatalf("folder should have 1 child, got %d", len(folder.Children))
	}
	if folder.Children[0].Kind != "workspace" {
		t.Fatalf("child should be workspace, got %s", folder.Children[0].Kind)
	}
}

func TestBuildTreeSortOrder(t *testing.T) {
	fID := testUUID("sort-f")
	wsA := testUUID("sort-a")
	wsB := testUUID("sort-b")
	scan := &ScanResult{
		Folders: map[string]ScannedFolder{
			fID: {ID: fID, Name: "Folder", Path: "Folder"},
		},
		Workspaces: map[string]ScannedWorkspace{
			wsA: {ID: wsA, Name: "a-Workspace", RootPath: "a-Workspace"},
			wsB: {ID: wsB, Name: "B-Workspace", RootPath: "B-Workspace"},
		},
	}
	tree := BuildTree(scan, "", 1)
	if tree.Roots[0].Kind != "folder" {
		t.Fatalf("folder should be first, got %s", tree.Roots[0].Kind)
	}
	if tree.Roots[1].Name != "a-Workspace" {
		t.Fatalf("expected a-Workspace before B-Workspace, got %s", tree.Roots[1].Name)
	}
}

func TestBuildTreeInvalidCurrentWorkspace(t *testing.T) {
	wsID := testUUID("invld-ws")
	scan := &ScanResult{
		Workspaces: map[string]ScannedWorkspace{
			wsID: {ID: wsID, Name: "Project", RootPath: "Project"},
		},
	}
	tree := BuildTree(scan, testUUID("nonexistent"), 1)
	if tree.CurrentWorkspaceID != "" {
		t.Fatalf("invalid current should be cleared, got %q", tree.CurrentWorkspaceID)
	}
}

// ── Service tests ────────────────────────────────────────────────────────────

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
	if tree.Revision != 1 {
		t.Fatalf("revision = %d, want 1", tree.Revision)
	}
}

func TestServiceGetByID(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("svc-byid")
	createWS(t, vault, "Project", wsID)

	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}

	ws, ok := svc.GetWorkspaceByID(wsID)
	if !ok || ws.Name != "Project" {
		t.Fatalf("GetWorkspaceByID = %+v, %v", ws, ok)
	}

	_, ok = svc.GetWorkspaceByID(testUUID("nonexistent"))
	if ok {
		t.Fatal("nonexistent workspace should not be found")
	}
}

func TestServiceInternalMutationBaseline(t *testing.T) {
	vault := t.TempDir()
	ws1ID := testUUID("mut-ws1")
	ws2ID := testUUID("mut-ws2")
	createWS(t, vault, "Project", ws1ID)

	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}

	svc.BeginInternalMutation()
	createWS(t, vault, "NewOne", ws2ID)
	svc.EndInternalMutationAndRefreshBaseline()

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
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}

	tree := svc.GetTree()
	if len(tree.Roots) != 2 {
		t.Fatalf("initial: expected 2 roots, got %d", len(tree.Roots))
	}

	// Simulate offline: rename, create, delete.
	if err := os.Rename(filepath.Join(vault, "Project"), filepath.Join(vault, "ArchiveProject")); err != nil {
		t.Fatal(err)
	}
	createWS(t, vault, "NewOne", ws3ID)
	if err := os.RemoveAll(filepath.Join(vault, "Clients", "Client1")); err != nil {
		t.Fatal(err)
	}

	if err := svc.RescanWorkspaceTree(); err != nil {
		t.Fatal(err)
	}

	tree = svc.GetTree()
	if tree.Revision != 2 {
		t.Fatalf("revision = %d, want 2", tree.Revision)
	}

	ws2, ok := svc.GetWorkspaceByID(ws2ID)
	if !ok || ws2.RootPath != "ArchiveProject" {
		t.Fatalf("ws2 = %+v, want RootPath=ArchiveProject", ws2)
	}
	if _, ok := svc.GetWorkspaceByID(ws1ID); ok {
		t.Fatal("ws1 should be gone after deletion")
	}
	if _, ok := svc.GetWorkspaceByID(ws3ID); !ok {
		t.Fatal("ws3 should exist")
	}
}

func TestWatcherDebounceDoesNotFireDuringInternalMutation(t *testing.T) {
	vault := t.TempDir()
	createWS(t, vault, "Project", testUUID("debounce-ws"))

	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}

	svc.BeginInternalMutation()
	svc.OnFileChanged() // Should be suppressed.
	svc.EndInternalMutationAndRefreshBaseline()

	tree := svc.GetTree()
	if len(tree.Roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree.Roots))
	}
}

func TestCaseInsensitiveWindowsSafePath(t *testing.T) {
	vault := t.TempDir()
	createWS(t, vault, "Project", testUUID("case-p"))
	createWS(t, vault, "project", testUUID("case-q"))

	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		if len(result.Workspaces) != 1 {
			t.Fatalf("case-insensitive: expected 1 workspace, got %d", len(result.Workspaces))
		}
	} else {
		if len(result.Workspaces) != 2 {
			t.Fatalf("case-sensitive: expected 2 workspaces, got %d", len(result.Workspaces))
		}
	}
}

func TestPathValidationNoBackslash(t *testing.T) {
	vault := t.TempDir()
	createFolder(t, vault, "Deep/Nested", testUUID("path-f"))
	result, err := Scan(vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range result.Folders {
		if strings.Contains(f.Path, "\\") {
			t.Fatalf("folder path contains backslash: %q", f.Path)
		}
	}
	for _, ws := range result.Workspaces {
		if strings.Contains(ws.RootPath, "\\") {
			t.Fatalf("workspace path contains backslash: %q", ws.RootPath)
		}
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func createWS(t *testing.T, vaultDir, name, id string) {
	t.Helper()
	dir := filepath.Join(vaultDir, filepath.FromSlash(name))
	mustMkdirAll(t, dir)
	verstakDir := filepath.Join(dir, ".verstak")
	mustMkdirAll(t, verstakDir)
	createWorkspaceMarker(t, verstakDir, id)
}

func createFolder(t *testing.T, vaultDir, relPath, id string) {
	t.Helper()
	dir := filepath.Join(vaultDir, filepath.FromSlash(relPath))
	mustMkdirAll(t, dir)
	verstakDir := filepath.Join(dir, ".verstak")
	mustMkdirAll(t, verstakDir)
	createFolderMarker(t, verstakDir, id)
}

func createWorkspaceMarker(t *testing.T, verstakDir, id string) {
	t.Helper()
	marker := WorkspaceMarker{SchemaVersion: 1, WorkspaceID: id}
	data, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(verstakDir, "workspace.json"), string(data))
}

func createFolderMarker(t *testing.T, verstakDir, id string) {
	t.Helper()
	marker := FolderMarker{SchemaVersion: 1, FolderID: id}
	data, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(verstakDir, "folder.json"), string(data))
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func captureFiles(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			out[rel+"/"] = ""
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out[rel] = string(data)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func filesEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
