package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestScanAndRecordTracksExternalWorkspaceLifecycleByIdentity(t *testing.T) {
	root := t.TempDir()
	workspaceID := uuid.NewString()
	createSnapshotWorkspace(t, root, "Project", workspaceID)
	if err := os.Mkdir(filepath.Join(root, "Project", "Files"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Project", "Files", "note.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewService(root, "device-a")
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("baseline: %v", err)
	}

	if err := os.Rename(filepath.Join(root, "Project"), filepath.Join(root, "Renamed")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("scan rename: %v", err)
	}
	assertWorkspaceSnapshotOp(t, unpushedOps(t, service), 0, OpRename, workspaceID, "Renamed", "Project")
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("unchanged renamed scan: %v", err)
	}
	if got := len(unpushedOps(t, service)); got != 1 {
		t.Fatalf("unchanged rename produced %d operations, want 1", got)
	}

	trashPath := filepath.Join(root, ".verstak", "trash", "workspaces", "external-trash", "Renamed")
	if err := os.MkdirAll(filepath.Dir(trashPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "Renamed"), trashPath); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("scan trash: %v", err)
	}
	assertWorkspaceSnapshotOp(t, unpushedOps(t, service), 1, OpTrash, workspaceID, "Renamed", "")

	if err := os.Rename(trashPath, filepath.Join(root, "Restored")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("scan restore: %v", err)
	}
	assertWorkspaceSnapshotOp(t, unpushedOps(t, service), 2, OpRestore, workspaceID, "Restored", "")

	createSnapshotWorkspace(t, root, "Copied", workspaceID)
	warnings, err := service.ScanAndRecord()
	if err != nil {
		t.Fatalf("scan duplicate identity: %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "duplicate-workspace-id: Copied") {
		t.Fatalf("duplicate identity warnings = %v", warnings)
	}
	if got := len(unpushedOps(t, service)); got != 3 {
		t.Fatalf("duplicate identity created operations: %d, want 3", got)
	}
}

func createSnapshotWorkspace(t *testing.T, root, name, workspaceID string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, name, ".verstak"), 0o755); err != nil {
		t.Fatal(err)
	}
	marker, err := json.Marshal(map[string]string{"workspaceId": workspaceID})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, name, ".verstak", "workspace.json"), marker, 0o600); err != nil {
		t.Fatal(err)
	}
	metadataPath := filepath.Join(root, ".verstak", "workspaces", name, "metadata.json")
	if err := os.MkdirAll(filepath.Dir(metadataPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadataPath, []byte(`{"workspaceId":"`+workspaceID+`","workspaceName":"`+name+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertWorkspaceSnapshotOp(t *testing.T, ops []Op, index int, opType, workspaceID, path, previousPath string) {
	t.Helper()
	if len(ops) <= index {
		t.Fatalf("operations = %#v, want index %d", ops, index)
	}
	op := ops[index]
	if op.EntityType != EntityWorkspace || op.EntityID != workspaceID || op.OpType != opType {
		t.Fatalf("workspace operation = %+v", op)
	}
	var payload snapshotWorkspacePayload
	if err := json.Unmarshal([]byte(op.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode workspace payload: %v", err)
	}
	if payload.Path != path || payload.PreviousPath != previousPath || payload.WorkspaceID != workspaceID {
		t.Fatalf("workspace payload = %+v", payload)
	}
}

func TestScanAndRecordBaselinesThenRecordsExternalChanges(t *testing.T) {
	root := t.TempDir()
	service := NewService(root, "device-a")

	warnings, err := service.ScanAndRecord()
	if err != nil {
		t.Fatalf("initial ScanAndRecord: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("initial warnings = %v, want none", warnings)
	}
	assertUnpushedCount(t, service, 0)

	if err := os.Mkdir(filepath.Join(root, "Docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Docs", "note.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}

	warnings, err = service.ScanAndRecord()
	if err != nil {
		t.Fatalf("scan create: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("create warnings = %v", warnings)
	}
	ops := unpushedOps(t, service)
	if len(ops) != 2 {
		t.Fatalf("create ops = %#v, want folder and file", ops)
	}
	if ops[0].EntityType != EntityFolder || ops[0].EntityID != "Docs" || ops[0].OpType != OpCreate {
		t.Fatalf("folder op = %+v", ops[0])
	}
	if ops[1].EntityType != EntityFile || ops[1].EntityID != "Docs/note.txt" || ops[1].OpType != OpCreate {
		t.Fatalf("file op = %+v", ops[1])
	}
	var createPayload map[string]interface{}
	if err := json.Unmarshal([]byte(ops[1].PayloadJSON), &createPayload); err != nil {
		t.Fatalf("decode file payload: %v", err)
	}
	if createPayload["content"] != "one" || createPayload["contentHash"] == "" {
		t.Fatalf("file payload = %#v", createPayload)
	}

	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("unchanged scan: %v", err)
	}
	assertUnpushedCount(t, service, 2)

	if err := os.WriteFile(filepath.Join(root, "Docs", "note.txt"), []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("scan update: %v", err)
	}
	ops = unpushedOps(t, service)
	if len(ops) != 3 || ops[2].EntityType != EntityFile || ops[2].OpType != OpUpdate {
		t.Fatalf("update ops = %#v", ops)
	}

	if err := os.Remove(filepath.Join(root, "Docs", "note.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "Docs")); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("scan delete: %v", err)
	}
	ops = unpushedOps(t, service)
	if len(ops) != 5 {
		t.Fatalf("delete ops = %#v", ops)
	}
	if ops[3].EntityType != EntityFile || ops[3].OpType != OpDelete || ops[4].EntityType != EntityFolder || ops[4].OpType != OpDelete {
		t.Fatalf("delete ordering = %#v", ops[3:])
	}
}

func TestScanAndRecordNeverTreatsInitialFilesAsDeletes(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Existing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Existing", "before-sync.txt"), []byte("local"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewService(root, "device-a")

	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("initial ScanAndRecord: %v", err)
	}
	assertUnpushedCount(t, service, 0)
	snapshot, err := service.LoadSnapshot()
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if snapshot.Entries["Existing/before-sync.txt"].Hash == "" {
		t.Fatalf("snapshot = %#v, expected content hash", snapshot.Entries)
	}
}

func TestScanAndRecordFindsChangesMadeWhileDesktopWasClosed(t *testing.T) {
	root := t.TempDir()
	initial := NewService(root, "device-a")
	if _, err := initial.ScanAndRecord(); err != nil {
		t.Fatalf("initial baseline: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "offline.txt"), []byte("created while closed"), 0o644); err != nil {
		t.Fatal(err)
	}

	restarted := NewService(root, "device-a")
	if _, err := restarted.ScanAndRecord(); err != nil {
		t.Fatalf("scan after offline create: %v", err)
	}
	ops := unpushedOps(t, restarted)
	if len(ops) != 1 || ops[0].OpType != OpCreate || ops[0].EntityID != "offline.txt" {
		t.Fatalf("offline create operations = %#v", ops)
	}

	if err := os.WriteFile(filepath.Join(root, "offline.txt"), []byte("updated while closed"), 0o644); err != nil {
		t.Fatal(err)
	}
	restarted = NewService(root, "device-a")
	if _, err := restarted.ScanAndRecord(); err != nil {
		t.Fatalf("scan after offline update: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "offline.txt")); err != nil {
		t.Fatal(err)
	}
	restarted = NewService(root, "device-a")
	if _, err := restarted.ScanAndRecord(); err != nil {
		t.Fatalf("scan after offline delete: %v", err)
	}
	ops = unpushedOps(t, restarted)
	if len(ops) != 3 || ops[1].OpType != OpUpdate || ops[2].OpType != OpDelete {
		t.Fatalf("offline lifecycle operations = %#v", ops)
	}
}

func TestRecordBootstrapOpsPublishesExistingFilesWithoutDeletes(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "Existing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Existing", "before-sync.txt"), []byte("local"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewService(root, "device-a")
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}
	initial, err := service.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RecordBootstrapOps(initial); err != nil {
		t.Fatalf("RecordBootstrapOps: %v", err)
	}
	ops := unpushedOps(t, service)
	if len(ops) != 2 {
		t.Fatalf("bootstrap ops = %#v, want create folder and file", ops)
	}
	for _, op := range ops {
		if op.OpType != OpCreate {
			t.Fatalf("bootstrap op = %+v, initial scan must not create delete", op)
		}
	}

	empty := newSnapshot()
	if err := service.RecordBootstrapOps(empty); err != nil {
		t.Fatalf("empty bootstrap: %v", err)
	}
	if got := len(unpushedOps(t, service)); got != 2 {
		t.Fatalf("empty bootstrap added operations = %d, want 2", got)
	}
}

func TestSyncBootstrapAndWarningStateSurviveRestart(t *testing.T) {
	root := t.TempDir()
	service := NewService(root, "device-a")
	if err := service.SetBootstrapComplete(true); err != nil {
		t.Fatalf("SetBootstrapComplete: %v", err)
	}
	if err := service.SetLastWarning("file-too-large: archive.bin"); err != nil {
		t.Fatalf("SetLastWarning: %v", err)
	}

	restarted := NewService(root, "")
	bootstrapped, err := restarted.BootstrapComplete()
	if err != nil {
		t.Fatalf("BootstrapComplete: %v", err)
	}
	if !bootstrapped {
		t.Fatal("bootstrap state was lost after restart")
	}
	warning, err := restarted.LastWarning()
	if err != nil {
		t.Fatalf("LastWarning: %v", err)
	}
	if warning != "file-too-large: archive.bin" {
		t.Fatalf("warning = %q", warning)
	}
}

func TestScanAndRecordSkipsReservedTemporaryAndSymlinkPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".verstak", "sync"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".verstak", "sync", "state.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".verstak-write-local"), []byte("temporary"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "draft.tmp"), []byte("temporary"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "normal.txt"), []byte("normal"), 0o600); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink(filepath.Join(root, "normal.txt"), filepath.Join(root, "normal-link.txt")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
	}

	service := NewService(root, "device-a")
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	snapshot, err := service.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{".verstak/sync/state.json", ".verstak-write-local", "draft.tmp", "normal-link.txt"} {
		if _, ok := snapshot.Entries[path]; ok {
			t.Fatalf("reserved path %q was included in snapshot %#v", path, snapshot.Entries)
		}
	}
	if _, ok := snapshot.Entries["normal.txt"]; !ok {
		t.Fatalf("normal file missing from snapshot %#v", snapshot.Entries)
	}
}

func TestScanAndRecordKeepsUnsupportedFileUnresolved(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "too-large.bin")
	if err := os.WriteFile(path, make([]byte, maxOperationFileBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	service := NewService(root, "device-a")

	warnings, err := service.ScanAndRecord()
	if err != nil {
		t.Fatalf("scan unsupported file: %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "too-large.bin") || !strings.Contains(warnings[0], "file-too-large") {
		t.Fatalf("warnings = %v", warnings)
	}
	assertUnpushedCount(t, service, 0)
	snapshot, err := service.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := snapshot.Entries["too-large.bin"]; ok {
		t.Fatalf("unsupported file was marked synchronized: %#v", snapshot.Entries)
	}

	warnings, err = service.ScanAndRecord()
	if err != nil || len(warnings) != 1 {
		t.Fatalf("second scan warnings=%v err=%v, unresolved file must remain visible", warnings, err)
	}
}

func unpushedOps(t *testing.T, service *Service) []Op {
	t.Helper()
	ops, err := service.GetUnpushedOps()
	if err != nil {
		t.Fatalf("GetUnpushedOps: %v", err)
	}
	return ops
}

func assertUnpushedCount(t *testing.T, service *Service, want int) {
	t.Helper()
	if got := len(unpushedOps(t, service)); got != want {
		t.Fatalf("unpushed ops = %d, want %d", got, want)
	}
}
