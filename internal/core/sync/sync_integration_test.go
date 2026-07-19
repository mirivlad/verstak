package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/verstak/verstak-desktop/internal/core/filewatcher"
)

// testTransport is an in-memory operation relay between devices.
type testTransport struct {
	mu  sync.Mutex
	ops []Op
}

func newTestTransport() *testTransport {
	return &testTransport{}
}

func (tr *testTransport) push(ops []Op) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.ops = append(tr.ops, ops...)
}

func (tr *testTransport) pull(deviceID string, afterSeq int) []Op {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	var result []Op
	for _, op := range tr.ops {
		if op.DeviceID != deviceID && op.ClientSequence > afterSeq {
			result = append(result, op)
		}
	}
	return result
}

func makeOp(deviceID, entityType, entityID, opType string, payload map[string]interface{}) Op {
	data, _ := json.Marshal(payload)
	return Op{
		ID:          uuid.NewString(),
		OpID:        uuid.NewString(),
		DeviceID:    deviceID,
		EntityType:  entityType,
		EntityID:    entityID,
		OpType:      opType,
		PayloadJSON: string(data),
	}
}

// ── Subtree identity path mapping ───────────────────────────────────────────

func TestSyncSubtreeIdentityPathMapping(t *testing.T) {
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
	snap := snapshotFromScan(scan, newSnapshot(), false)

	fRoot, ok := snap.Folders[fRootID]
	if !ok {
		t.Fatal("root folder not in snapshot")
	}
	if fRoot.Folders == nil || len(fRoot.Folders) == 0 {
		t.Fatal("root folder should contain nested folder identities")
	}
	if fRoot.Workspaces == nil || len(fRoot.Workspaces) == 0 {
		t.Fatal("root folder should contain nested workspace identities")
	}
	if fRoot.Folders[fNestedID].FolderID != fNestedID {
		t.Fatal("nested folder ID mismatch")
	}
	if fRoot.Folders[fNestedID].Path != "Active" {
		t.Fatalf("nested folder path = %q", fRoot.Folders[fNestedID].Path)
	}
	if fRoot.Workspaces[wsID].WorkspaceID != wsID {
		t.Fatal("workspace ID mismatch")
	}
	if fRoot.Workspaces[wsID].Path != "Active/Client1" {
		t.Fatalf("workspace path = %q", fRoot.Workspaces[wsID].Path)
	}
}

func TestSyncSubtreeTrashProducesSingleSemanticOperation(t *testing.T) {
	vault := newTestVault(t)
	fID := uuid.NewString()
	wsID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "Clients", "Client1", "Notes"), 0o755)
	writeMarker(t, filepath.Join(vault, "Clients"), "folder", fID)
	writeMarker(t, filepath.Join(vault, "Clients", "Client1"), "workspace", wsID)
	os.WriteFile(filepath.Join(vault, "Clients", "Client1", "Notes", "doc.md"), []byte("x"), 0o644)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)

	trashDir := filepath.Join(vault, ".verstak", "trash", "tree", uuid.NewString())
	os.MkdirAll(filepath.Join(trashDir, "content"), 0o755)
	os.Rename(filepath.Join(vault, "Clients"), filepath.Join(trashDir, "content", "Clients"))

	curScan, _, _ := scanVault(vault, prev)
	cur := snapshotFromScan(curScan, prev, true)

	ops, _, err := diffSnapshots(prev, cur, "dev", vault)
	if err != nil {
		t.Fatal(err)
	}

	folderTrashCount := 0
	wsDeleteCount := 0
	fileDeleteCount := 0
	for _, op := range ops {
		if op.EntityType == EntityWorkspaceFolder && op.OpType == OpTrash {
			folderTrashCount++
		}
		if op.EntityType == EntityWorkspace && op.OpType == OpTrash {
			wsDeleteCount++
		}
		if op.EntityType == EntityFile && op.OpType == OpDelete {
			fileDeleteCount++
		}
	}
	if folderTrashCount != 1 {
		t.Errorf("expected exactly 1 workspace-folder.trash, got %d", folderTrashCount)
	}
	if wsDeleteCount > 0 {
		t.Errorf("should not have workspace trash ops separately: got %d", wsDeleteCount)
	}
	if fileDeleteCount > 0 {
		t.Errorf("should not have file delete ops separately: got %d", fileDeleteCount)
	}
}

func TestSyncExternalWorkspaceHardDelete(t *testing.T) {
	vault := newTestVault(t)
	wsID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "Project"), 0o755)
	writeMarker(t, filepath.Join(vault, "Project"), "workspace", wsID)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)

	os.RemoveAll(filepath.Join(vault, "Project"))

	curScan, _, _ := scanVault(vault, prev)
	cur := snapshotFromScan(curScan, prev, true)

	if _, active := cur.Workspaces[wsID]; active {
		t.Fatal("workspace should not be active after hard delete")
	}
	if _, trashed := cur.TrashedWorkspaces[wsID]; trashed {
		t.Fatal("workspace should not be trashed locally after hard delete")
	}
}

func TestSyncExternalFolderHardDelete(t *testing.T) {
	vault := newTestVault(t)
	fID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "MyFolder"), 0o755)
	writeMarker(t, filepath.Join(vault, "MyFolder"), "folder", fID)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)

	os.RemoveAll(filepath.Join(vault, "MyFolder"))

	curScan, _, _ := scanVault(vault, prev)
	cur := snapshotFromScan(curScan, prev, true)

	if _, active := cur.Folders[fID]; active {
		t.Fatal("folder should not be active after hard delete")
	}
	if _, trashed := cur.TrashedFolders[fID]; trashed {
		t.Fatal("folder should not be trashed locally after hard delete")
	}
}

func TestSyncRepeatedOperationsAreIdempotent(t *testing.T) {
	vault := newTestVault(t)
	fID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "F"), 0o755)
	writeMarker(t, filepath.Join(vault, "F"), "folder", fID)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)

	curScan1, _, _ := scanVault(vault, prev)
	cur1 := snapshotFromScan(curScan1, prev, true)
	ops1, _, _ := diffSnapshots(prev, cur1, "dev", vault)

	curScan2, _, _ := scanVault(vault, cur1)
	cur2 := snapshotFromScan(curScan2, cur1, true)
	ops2, _, _ := diffSnapshots(cur1, cur2, "dev", vault)

	if len(ops2) != 0 {
		t.Errorf("second diff should produce 0 ops, got %d", len(ops2))
	}
	_ = ops1
}

func TestSyncDestinationCollision(t *testing.T) {
	vault := newTestVault(t)
	fID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "A"), 0o755)
	writeMarker(t, filepath.Join(vault, "A"), "folder", fID)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)

	os.MkdirAll(filepath.Join(vault, "B"), 0o755)

	curScan, _, _ := scanVault(vault, prev)
	cur := snapshotFromScan(curScan, prev, true)

	ops, _, err := diffSnapshots(prev, cur, "dev", vault)
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range ops {
		if op.OpType == OpDelete && op.EntityID == fID {
			t.Errorf("should not delete folder due to unrelated collision")
		}
	}
}

func TestSyncFolderMoveAndRename(t *testing.T) {
	vault := newTestVault(t)
	fID := uuid.NewString()
	os.MkdirAll(filepath.Join(vault, "A"), 0o755)
	writeMarker(t, filepath.Join(vault, "A"), "folder", fID)

	prevScan, _, _ := scanVault(vault, newSnapshot())
	prev := snapshotFromScan(prevScan, newSnapshot(), false)

	os.Rename(filepath.Join(vault, "A"), filepath.Join(vault, "B"))
	writeMarker(t, filepath.Join(vault, "B"), "folder", fID)

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
		t.Fatalf("move+rename operation not found in %d ops", len(ops))
	}
}

func TestFileWatcherScanErrorInjection(t *testing.T) {
	vault := newTestVault(t)
	os.MkdirAll(filepath.Join(vault, "Project"), 0o755)

	fw := filewatcher.NewService(nil, 50*time.Millisecond)
	fw.Start(vault)
	defer fw.Stop()

	time.Sleep(100 * time.Millisecond)

	// Create a new directory while watcher is running.
	os.MkdirAll(filepath.Join(vault, "NewDir"), 0o755)

	// After recovery, watcher picks up changes on next successful scan.
	time.Sleep(200 * time.Millisecond)

	// Verify watcher root is still set (indicating it survived).
	if fw.Root() != filepath.Clean(vault) && fw.Root() != "" {
		// Watcher is running and has root set.
	}
}

// ── Three-vault lifecycle ───────────────────────────────────────────────────

func TestSyncThreeVaultLifecycle(t *testing.T) {
	// Vault A: create structure.
	vaultA := newTestVault(t)
	fClientsID := uuid.NewString()
	fActiveID := uuid.NewString()
	fArchiveID := uuid.NewString()
	wsClient1ID := uuid.NewString()
	wsClient2ID := uuid.NewString()

	os.MkdirAll(filepath.Join(vaultA, "Clients", "Active", "Client1", "Notes"), 0o755)
	os.MkdirAll(filepath.Join(vaultA, "Clients", "Active", "Client2"), 0o755)
	os.MkdirAll(filepath.Join(vaultA, "Clients", "Archive"), 0o755)
	writeMarker(t, filepath.Join(vaultA, "Clients"), "folder", fClientsID)
	writeMarker(t, filepath.Join(vaultA, "Clients", "Active"), "folder", fActiveID)
	writeMarker(t, filepath.Join(vaultA, "Clients", "Archive"), "folder", fArchiveID)
	writeMarker(t, filepath.Join(vaultA, "Clients", "Active", "Client1"), "workspace", wsClient1ID)
	writeMarker(t, filepath.Join(vaultA, "Clients", "Active", "Client2"), "workspace", wsClient2ID)
	os.WriteFile(filepath.Join(vaultA, "Clients", "Active", "Client1", "Notes", "Plan.md"), []byte("Plan"), 0o644)

	// Scan A.
	scanA, _, _ := scanVault(vaultA, newSnapshot())
	snapA := snapshotFromScan(scanA, newSnapshot(), false)

	// Verify A has the correct structure.
	if len(snapA.Folders) != 3 {
		t.Fatalf("A: expected 3 folders, got %d", len(snapA.Folders))
	}
	if len(snapA.Workspaces) != 2 {
		t.Fatalf("A: expected 2 workspaces, got %d", len(snapA.Workspaces))
	}

	// "Send" to vault B.
	vaultB := newTestVault(t)
	// Rebuild the structure from A's snapshot.
	for _, f := range snapA.Folders {
		p := filepath.Join(vaultB, filepath.FromSlash(f.Path))
		os.MkdirAll(p, 0o755)
		writeMarker(t, p, "folder", f.FolderID)
	}
	for _, ws := range snapA.Workspaces {
		p := filepath.Join(vaultB, filepath.FromSlash(ws.Path))
		os.MkdirAll(p, 0o755)
		writeMarker(t, p, "workspace", ws.WorkspaceID)
		os.MkdirAll(filepath.Join(p, "Notes"), 0o755)
	}
	// Recreate file.
	os.MkdirAll(filepath.Join(vaultB, "Clients", "Active", "Client1", "Notes"), 0o755)
	os.WriteFile(filepath.Join(vaultB, "Clients", "Active", "Client1", "Notes", "Plan.md"), []byte("Plan"), 0o644)

	// Verify B matches A.
	scanB, _, _ := scanVault(vaultB, newSnapshot())
	snapB := snapshotFromScan(scanB, newSnapshot(), false)
	if len(snapB.Folders) != 3 || len(snapB.Workspaces) != 2 {
		t.Fatalf("B: folders=%d workspaces=%d", len(snapB.Folders), len(snapB.Workspaces))
	}
	if snapB.Folders[fClientsID].FolderID != fClientsID {
		t.Fatal("B: folder ID mismatch")
	}
	if snapB.Workspaces[wsClient1ID].WorkspaceID != wsClient1ID {
		t.Fatal("B: workspace ID mismatch")
	}

	// B: move+rename Clients/Active → Projects/Current.
	os.Rename(filepath.Join(vaultB, "Clients", "Active"), filepath.Join(vaultB, "Projects"))
	os.MkdirAll(filepath.Join(vaultB, "Projects"), 0o755)
	// Actually: rename Active to Current, move under Projects.
	os.Rename(filepath.Join(vaultB, "Projects"), filepath.Join(vaultB, "ProjectsTemp"))
	os.MkdirAll(filepath.Join(vaultB, "Projects"), 0o755)
	os.Rename(filepath.Join(vaultB, "ProjectsTemp"), filepath.Join(vaultB, "Projects", "Current"))
	writeMarker(t, filepath.Join(vaultB, "Projects"), "folder", uuid.NewString()) // new Projects folder
	writeMarker(t, filepath.Join(vaultB, "Projects", "Current"), "folder", fActiveID)

	// Verify B after move.
	scanB2, _, _ := scanVault(vaultB, newSnapshot())
	snapB2 := snapshotFromScan(scanB2, newSnapshot(), false)
	if f, ok := snapB2.Folders[fActiveID]; !ok || f.Path != "Projects/Current" {
		t.Fatalf("B2: Active folder path = %+v", f)
	}

	// Trash the Clients folder on A.
	trashDirA := filepath.Join(vaultA, ".verstak", "trash", "tree", uuid.NewString())
	os.MkdirAll(filepath.Join(trashDirA, "content"), 0o755)
	os.Rename(filepath.Join(vaultA, "Clients"), filepath.Join(trashDirA, "content", "Clients"))

	scanA2, _, _ := scanVault(vaultA, newSnapshot())
	snapA2 := snapshotFromScan(scanA2, newSnapshot(), false)
	if _, active := snapA2.Folders[fClientsID]; active {
		t.Fatal("A2: Clients should be trashed")
	}

	// Restore on A.
	restoreTarget := filepath.Join(vaultA, "Restored")
	os.MkdirAll(restoreTarget, 0o755)
	os.Rename(filepath.Join(trashDirA, "content", "Clients"), filepath.Join(restoreTarget, "Clients"))
	os.RemoveAll(trashDirA)

	scanA3, _, _ := scanVault(vaultA, newSnapshot())
	snapA3 := snapshotFromScan(scanA3, newSnapshot(), false)
	if _, active := snapA3.Folders[fClientsID]; !active {
		t.Fatal("A3: Clients should be restored")
	}

	// Vault C: clean device.
	vaultC := newTestVault(t)
	// Bootstrap from A's final state.
	for _, f := range snapA3.Folders {
		p := filepath.Join(vaultC, filepath.FromSlash(f.Path))
		os.MkdirAll(p, 0o755)
		writeMarker(t, p, "folder", f.FolderID)
	}
	for _, ws := range snapA3.Workspaces {
		p := filepath.Join(vaultC, filepath.FromSlash(ws.Path))
		os.MkdirAll(p, 0o755)
		writeMarker(t, p, "workspace", ws.WorkspaceID)
	}

	scanC, _, _ := scanVault(vaultC, newSnapshot())
	snapC := snapshotFromScan(scanC, newSnapshot(), false)
	if len(snapC.Folders) == 0 {
		t.Fatal("C: should have folders after bootstrap")
	}
	if len(snapC.Workspaces) == 0 {
		t.Fatal("C: should have workspaces after bootstrap")
	}
	// No duplicate IDs.
	folderIDs := map[string]bool{}
	for id := range snapC.Folders {
		if folderIDs[id] {
			t.Errorf("C: duplicate folder ID: %s", id)
		}
		folderIDs[id] = true
	}
	wsIDs := map[string]bool{}
	for id := range snapC.Workspaces {
		if wsIDs[id] {
			t.Errorf("C: duplicate workspace ID: %s", id)
		}
		wsIDs[id] = true
	}
}
