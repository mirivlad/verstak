package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/workspace"
)

func TestSyncNowAgainstRealServerTwoVaults(t *testing.T) {
	serverURL := os.Getenv("VERSTAK_SYNC_SMOKE_SERVER_URL")
	deviceA := os.Getenv("VERSTAK_SYNC_SMOKE_DEVICE_A")
	deviceB := os.Getenv("VERSTAK_SYNC_SMOKE_DEVICE_B")
	apiKeyA := os.Getenv("VERSTAK_SYNC_SMOKE_KEY_A")
	apiKeyB := os.Getenv("VERSTAK_SYNC_SMOKE_KEY_B")
	if serverURL == "" || deviceA == "" || deviceB == "" || apiKeyA == "" || apiKeyB == "" {
		t.Skip("set VERSTAK_SYNC_SMOKE_* env vars to run the real sync-server smoke test")
	}

	appA, rootA := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, deviceA)
	appB, rootB := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, deviceB)
	appA.workspace = workspace.NewManager(rootA)
	appB.workspace = workspace.NewManager(rootB)
	if err := appA.workspace.Load(); err != nil {
		t.Fatalf("load workspace A: %v", err)
	}
	if err := appB.workspace.Load(); err != nil {
		t.Fatalf("load workspace B: %v", err)
	}
	if err := appA.syncSvc.SetState(serverURL, apiKeyA); err != nil {
		t.Fatalf("appA SetState: %v", err)
	}
	if err := appB.syncSvc.SetState(serverURL, apiKeyB); err != nil {
		t.Fatalf("appB SetState: %v", err)
	}

	if errStr := appA.CreateVaultFolder("files.plugin", "Shared"); errStr != "" {
		t.Fatalf("appA CreateVaultFolder: %s", errStr)
	}
	if errStr := appA.WriteVaultTextFile("files.plugin", "Shared/one.txt", "from A", corefiles.WriteOptions{CreateIfMissing: true}); errStr != "" {
		t.Fatalf("appA WriteVaultTextFile: %s", errStr)
	}
	expectSyncCounts(t, appA, 2, 2)
	expectSyncCounts(t, appB, 0, 2)
	expectText(t, appB, "Shared/one.txt", "from A")

	if errStr := appB.WriteVaultTextFile("files.plugin", "Shared/one.txt", "from B", corefiles.WriteOptions{Overwrite: true}); errStr != "" {
		t.Fatalf("appB update: %s", errStr)
	}
	if errStr := appB.MoveVaultPath("files.plugin", "Shared/one.txt", "Shared/two.txt", corefiles.MoveOptions{}); errStr != "" {
		t.Fatalf("appB move: %s", errStr)
	}
	expectSyncCounts(t, appB, 3, 3)
	expectSyncCounts(t, appA, 0, 3)
	expectText(t, appA, "Shared/two.txt", "from B")

	if _, errStr := appA.TrashVaultPath("files.plugin", "Shared/two.txt"); errStr != "" {
		t.Fatalf("appA trash: %s", errStr)
	}
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	if _, errStr := appB.GetVaultFileMetadata("files.plugin", "Shared/two.txt"); !strings.Contains(errStr, "not-found") {
		t.Fatalf("appB deleted file metadata err = %q, want not-found", errStr)
	}

	if errStr := appB.CreateVaultFolder("files.plugin", "Shared/Folder"); errStr != "" {
		t.Fatalf("appB create folder: %s", errStr)
	}
	expectSyncCounts(t, appB, 1, 1)
	expectSyncCounts(t, appA, 0, 1)
	if meta, errStr := appA.GetVaultFileMetadata("files.plugin", "Shared/Folder"); errStr != "" || meta.Type != corefiles.FileTypeFolder {
		t.Fatalf("appA folder metadata = %+v err=%q, want folder", meta, errStr)
	}

	if errStr := appA.MoveVaultPath("files.plugin", "Shared/Folder", "Shared/Archive", corefiles.MoveOptions{}); errStr != "" {
		t.Fatalf("appA move folder: %s", errStr)
	}
	expectSyncCounts(t, appA, 2, 2)
	expectSyncCounts(t, appB, 0, 2)
	if _, errStr := appB.GetVaultFileMetadata("files.plugin", "Shared/Folder"); !strings.Contains(errStr, "not-found") {
		t.Fatalf("appB moved folder old metadata err = %q, want not-found", errStr)
	}
	if meta, errStr := appB.GetVaultFileMetadata("files.plugin", "Shared/Archive"); errStr != "" || meta.Type != corefiles.FileTypeFolder {
		t.Fatalf("appB moved folder metadata = %+v err=%q, want folder", meta, errStr)
	}

	if _, errStr := appB.TrashVaultPath("files.plugin", "Shared/Archive"); errStr != "" {
		t.Fatalf("appB trash folder: %s", errStr)
	}
	expectSyncCounts(t, appB, 1, 1)
	expectSyncCounts(t, appA, 0, 1)
	if _, errStr := appA.GetVaultFileMetadata("files.plugin", "Shared/Archive"); !strings.Contains(errStr, "not-found") {
		t.Fatalf("appA deleted folder metadata err = %q, want not-found", errStr)
	}

	if err := os.WriteFile(filepath.Join(rootA, "Shared", "external.txt"), []byte("external while running"), 0o644); err != nil {
		t.Fatalf("external create: %v", err)
	}
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	expectText(t, appB, "Shared/external.txt", "external while running")
	assertNoUnpushedOps(t, appB)

	if err := os.WriteFile(filepath.Join(rootB, "Shared", "external.txt"), []byte("external while closed"), 0o644); err != nil {
		t.Fatalf("offline external update: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootB, "Shared", "offline-created.txt"), []byte("created while closed"), 0o644); err != nil {
		t.Fatalf("offline external create: %v", err)
	}
	if err := os.Remove(filepath.Join(rootB, "Shared", "external.txt")); err != nil {
		t.Fatalf("offline external delete: %v", err)
	}
	expectSyncCounts(t, appB, 2, 2)
	expectSyncCounts(t, appA, 0, 2)
	if _, errStr := appA.GetVaultFileMetadata("files.plugin", "Shared/external.txt"); !strings.Contains(errStr, "not-found") {
		t.Fatalf("offline deleted file remained on appA: %q", errStr)
	}
	expectText(t, appA, "Shared/offline-created.txt", "created while closed")
	assertNoUnpushedOps(t, appA)

	deal, errStr := appA.CreateWorkspace("Synced Deal", "minimal")
	if errStr != "" {
		t.Fatalf("appA CreateWorkspace: %s", errStr)
	}
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	assertWorkspaceIdentity(t, appB, "Synced Deal", deal.ID)

	if errStr := appA.RenameWorkspace("Synced Deal", "Renamed Deal"); errStr != "" {
		t.Fatalf("appA RenameWorkspace: %s", errStr)
	}
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	assertWorkspaceIdentity(t, appB, "Renamed Deal", deal.ID)

	trash, errStr := appA.TrashWorkspace("Renamed Deal")
	if errStr != "" {
		t.Fatalf("appA TrashWorkspace: %s", errStr)
	}
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	if _, err := appB.workspace.GetWorkspaceIdentity("Renamed Deal"); err == nil {
		t.Fatal("trashed workspace is still active on appB")
	}

	if _, errStr := appA.RestoreWorkspaceTrash(trash.TrashID, "Restored Deal"); errStr != "" {
		t.Fatalf("appA RestoreWorkspaceTrash: %s", errStr)
	}
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	assertWorkspaceIdentity(t, appB, "Restored Deal", deal.ID)
}

func expectSyncCounts(t *testing.T, app *App, pushed, pulled int) {
	t.Helper()
	result, err := app.syncNow()
	if err != nil {
		t.Fatalf("syncNow: %v", err)
	}
	if result["pushed"] != pushed || result["pulled"] != pulled {
		t.Fatalf("sync result = %#v, want pushed=%d pulled=%d", result, pushed, pulled)
	}
}

func expectText(t *testing.T, app *App, path, want string) {
	t.Helper()
	text, errStr := app.ReadVaultTextFile("files.plugin", path)
	if errStr != "" {
		t.Fatalf("ReadVaultTextFile(%s): %s", path, errStr)
	}
	if text != want {
		t.Fatalf("ReadVaultTextFile(%s) = %q, want %q", path, text, want)
	}
}

func assertNoUnpushedOps(t *testing.T, app *App) {
	t.Helper()
	ops, err := app.syncSvc.GetUnpushedOps()
	if err != nil {
		t.Fatalf("GetUnpushedOps: %v", err)
	}
	if len(ops) != 0 {
		t.Fatalf("remote operations were echoed as local operations: %#v", ops)
	}
}

func assertWorkspaceIdentity(t *testing.T, app *App, name, wantID string) {
	t.Helper()
	identity, err := app.workspace.GetWorkspaceIdentity(name)
	if err != nil {
		t.Fatalf("GetWorkspaceIdentity(%s): %v", name, err)
	}
	if identity.WorkspaceID != wantID {
		t.Fatalf("workspace %s ID = %s, want %s", name, identity.WorkspaceID, wantID)
	}
}
