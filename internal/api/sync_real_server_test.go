package api

import (
	"os"
	"strings"
	"testing"

	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
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

	appA, _ := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, deviceA)
	appB, _ := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, deviceB)
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
	expectSyncCounts(t, appB, 2, 2)
	expectSyncCounts(t, appA, 0, 2)
	expectText(t, appA, "Shared/two.txt", "from B")

	if _, errStr := appA.TrashVaultPath("files.plugin", "Shared/two.txt"); errStr != "" {
		t.Fatalf("appA trash: %s", errStr)
	}
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	if _, errStr := appB.GetVaultFileMetadata("files.plugin", "Shared/two.txt"); !strings.Contains(errStr, "not-found") {
		t.Fatalf("appB deleted file metadata err = %q, want not-found", errStr)
	}
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
