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

func testUUID(tag string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(tag+"@verstak.test")).String()
}

func TestScanEmptyVault(t *testing.T) {
	vault := t.TempDir()
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 0 || len(result.Workspaces) != 0 {
		t.Fatalf("empty vault: got folders=%d workspaces=%d", len(result.Folders), len(result.Workspaces))
	}
}

func TestScanSingleWorkspace(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("ws-1")
	createWS(t, vault, "Project", wsID)
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(result.Workspaces))
	}
	if ws, ok := result.Workspaces[wsID]; !ok || ws.Name != "Project" {
		t.Fatalf("workspace = %+v", ws)
	}
}

func TestScanWorkspaceRecursionStops(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("ws-stop")
	createWS(t, vault, "Project", wsID)
	mustMkdirAll(t, filepath.Join(vault, "Project", "Notes"))
	mustMkdirAll(t, filepath.Join(vault, "Project", "Files"))
	ignoredID := testUUID("ignored")
	createFolderMarker(t, filepath.Join(vault, "Project", ".verstak"), ignoredID)
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 1 || len(result.Folders) != 0 {
		t.Fatalf("workspaces=%d folders=%d", len(result.Workspaces), len(result.Folders))
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
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(result.Folders))
	}
	f2, ok := result.Folders[f2ID]
	if !ok || f2.ParentID != f1ID {
		t.Fatalf("folder f-2 parent = %q, want %s", f2.ParentID, f1ID)
	}
}

func TestScanUnmarkedDirectoriesReportedAsUnmanaged(t *testing.T) {
	vault := t.TempDir()
	mustMkdirAll(t, filepath.Join(vault, "Unmarked"))
	mustMkdirAll(t, filepath.Join(vault, "Unmarked", "Nested"))
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 0 || len(result.Workspaces) != 0 {
		t.Fatalf("unmarked dirs should not be folders/workspaces")
	}
	if len(result.Unmanaged) < 2 {
		t.Fatalf("expected at least 2 unmanaged dirs, got %d", len(result.Unmanaged))
	}
}

func TestScanHiddenDirsSkipped(t *testing.T) {
	vault := t.TempDir()
	mustMkdirAll(t, filepath.Join(vault, ".hidden"))
	id := testUUID("hidden-ws")
	createWS(t, vault, ".hidden", id)
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("hidden dirs skipped: got %d workspaces", len(result.Workspaces))
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
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("symlinks skipped: got %d workspaces", len(result.Workspaces))
	}
}

func TestScanDuplicateWorkspaceUUID(t *testing.T) {
	vault := t.TempDir()
	dupID := testUUID("dup")
	createWS(t, vault, "Project", dupID)
	createWS(t, vault, "Copy", dupID)
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("ambiguous duplicate: expected 0 workspaces, got %d", len(result.Workspaces))
	}
	hasError := false
	for _, w := range result.Warnings {
		if w.Code == "duplicate-id" && w.Level == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Fatal("expected error-level duplicate-id diagnostic")
	}
}

func TestScanCorruptedMarker(t *testing.T) {
	vault := t.TempDir()
	wsDir := filepath.Join(vault, "Project")
	mustMkdirAll(t, wsDir)
	verstakDir := filepath.Join(wsDir, ".verstak")
	mustMkdirAll(t, verstakDir)
	mustWriteFile(t, filepath.Join(verstakDir, "workspace.json"), "not json")
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 {
		t.Fatalf("corrupted marker → 0 workspaces")
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
	createWorkspaceMarker(t, verstakDir, testUUID("both-ws"))
	createFolderMarker(t, verstakDir, testUUID("both-f"))
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 1 || len(result.Folders) != 0 {
		t.Fatalf("workspace marker wins: ws=%d folders=%d", len(result.Workspaces), len(result.Folders))
	}
}

func TestScanNestedFolders(t *testing.T) {
	vault := t.TempDir()
	faID := testUUID("f-a")
	fbID := testUUID("f-b")
	fcID := testUUID("f-c")
	createFolder(t, vault, "A", faID)
	createFolder(t, vault, "A/B", fbID)
	createFolder(t, vault, "A/B/C", fcID)
	createWS(t, vault, "A/B/C/Deep", testUUID("ws-deep"))
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 3 || len(result.Workspaces) != 1 {
		t.Fatalf("folders=%d workspaces=%d", len(result.Folders), len(result.Workspaces))
	}
	fb := result.Folders[fbID]
	if fb.ParentID != faID {
		t.Fatalf("B parent = %q, want %s", fb.ParentID, faID)
	}
}

func TestScanIsReadOnly(t *testing.T) {
	vault := t.TempDir()
	createFolder(t, vault, "Clients", testUUID("ro-f"))
	createWS(t, vault, "Project", testUUID("ro-ws"))
	before := captureFiles(t, vault)
	_, err := Scan(vault, nil)
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
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Folders) != 2 || len(result.Workspaces) != 2 {
		t.Fatalf("folders=%d workspaces=%d", len(result.Folders), len(result.Workspaces))
	}
}

func TestScanInvalidUUIDInMarker(t *testing.T) {
	vault := t.TempDir()
	wsDir := filepath.Join(vault, "Bad")
	mustMkdirAll(t, wsDir)
	verstakDir := filepath.Join(wsDir, ".verstak")
	mustMkdirAll(t, verstakDir)
	mustWriteFile(t, filepath.Join(verstakDir, "workspace.json"), `{"schemaVersion":1,"workspaceId":"not-a-uuid"}`)
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Workspaces) != 0 || len(result.Warnings) == 0 {
		t.Fatal("invalid UUID should not produce workspace")
	}
}

func TestCaseInsensitiveWindowsSafePath(t *testing.T) {
	vault := t.TempDir()
	createWS(t, vault, "Project", testUUID("case-p"))
	createWS(t, vault, "project", testUUID("case-q"))
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		if len(result.Workspaces) != 1 {
			t.Fatalf("case-insensitive: expected 1, got %d", len(result.Workspaces))
		}
	} else {
		if len(result.Workspaces) != 2 {
			t.Fatalf("case-sensitive: expected 2, got %d", len(result.Workspaces))
		}
	}
}

func TestPathValidationNoBackslash(t *testing.T) {
	vault := t.TempDir()
	createFolder(t, vault, "Deep/Nested", testUUID("path-f"))
	result, err := Scan(vault, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range result.Folders {
		if strings.Contains(f.Path, "\\") {
			t.Fatalf("folder path has backslash: %q", f.Path)
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
