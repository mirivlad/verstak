package api

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
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
	appA.treeV2 = workspacetree.NewService(rootA, nil)
	appB.treeV2 = workspacetree.NewService(rootB, nil)
	if err := appA.treeV2.Initialize(); err != nil {
		t.Fatalf("initialize Deal tree A: %v", err)
	}
	if err := appB.treeV2.Initialize(); err != nil {
		t.Fatalf("initialize Deal tree B: %v", err)
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

	// This exceeds the former 8 MiB inline/base64 ceiling. The operation must
	// contain only a blob reference; the actual bytes travel through Blob API.
	binary := make([]byte, corefiles.MaxBinaryReadBytes+1)
	for i := range binary {
		binary[i] = byte(i % 251)
	}
	if err := os.WriteFile(filepath.Join(rootA, "Shared", "large.bin"), binary, 0o644); err != nil {
		t.Fatalf("write large binary: %v", err)
	}
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	received, err := os.ReadFile(filepath.Join(rootB, "Shared", "large.bin"))
	if err != nil {
		t.Fatalf("read synced large binary: %v", err)
	}
	if !bytes.Equal(received, binary) {
		t.Fatal("large binary content differs after blob sync")
	}
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
	if _, ok := appB.treeV2.ResolveWorkspace("Renamed Deal"); ok {
		t.Fatal("trashed workspace is still active on appB")
	}

	if _, errStr := appA.RestoreWorkspaceTrash(trash.TrashID, "Restored Deal"); errStr != "" {
		t.Fatalf("appA RestoreWorkspaceTrash: %s", errStr)
	}
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	assertWorkspaceIdentity(t, appB, "Restored Deal", deal.ID)

	// A plugin's own data, through the same server. The record is the unit that
	// travels, so a todo written on each device leaves both devices with both.
	enableTodoRecordSync(appA)
	enableTodoRecordSync(appB)
	writeSyncedTodos(t, appA, map[string]string{"shared": "prepare the invoice"})
	expectSyncCounts(t, appA, 1, 1)
	expectSyncCounts(t, appB, 0, 1)
	expectSyncedTodos(t, appB, map[string]string{"shared": "prepare the invoice"})
	assertNoUnpushedOps(t, appB)

	writeSyncedTodos(t, appA, map[string]string{
		"shared":      "prepare the invoice",
		"from-laptop": "written on A",
	})
	writeSyncedTodos(t, appB, map[string]string{
		"shared":       "prepare the invoice",
		"from-desktop": "written on B",
	})
	expectSyncCounts(t, appA, 1, 1)
	// B pushes its own and pulls two: A's new todo and its own coming back.
	expectSyncCounts(t, appB, 1, 2)
	expectSyncCounts(t, appA, 0, 1)
	both := map[string]string{
		"shared":       "prepare the invoice",
		"from-laptop":  "written on A",
		"from-desktop": "written on B",
	}
	expectSyncedTodos(t, appA, both)
	expectSyncedTodos(t, appB, both)

	writeSyncedTodos(t, appB, map[string]string{
		"shared":       "prepare the invoice",
		"from-desktop": "written on B",
	})
	expectSyncCounts(t, appB, 1, 1)
	expectSyncCounts(t, appA, 0, 1)
	expectSyncedTodos(t, appA, map[string]string{
		"shared":       "prepare the invoice",
		"from-desktop": "written on B",
	})
	assertNoUnpushedOps(t, appA)
}

// enableTodoRecordSync gives a test app a plugin that shares one list, which is
// all the record contract is: a document, and the field that identifies a row.
func enableTodoRecordSync(app *App) {
	app.storage = storage.New(app.vault)
	app.plugins = append(app.plugins, plugin.Plugin{
		Manifest: plugin.Manifest{
			ID:          "verstak.todo",
			Name:        "Todos",
			Version:     "1.0.0",
			Permissions: []string{"storage.namespace"},
			Sync: &plugin.SyncConfig{
				Records: []plugin.SyncRecordSet{
					{ID: "todos", Storage: "settings", Key: "todos:global", Identity: "id"},
				},
			},
		},
		Status:  plugin.StatusLoaded,
		Enabled: true,
	})
}

func writeSyncedTodos(t *testing.T, app *App, titlesByID map[string]string) {
	t.Helper()
	ids := make([]string, 0, len(titlesByID))
	for id := range titlesByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	value := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		value = append(value, map[string]interface{}{"id": id, "title": titlesByID[id]})
	}
	if errStr := app.WritePluginSetting("verstak.todo", "todos:global", value); errStr != "" {
		t.Fatalf("WritePluginSetting: %s", errStr)
	}
}

func expectSyncedTodos(t *testing.T, app *App, want map[string]string) {
	t.Helper()
	settings, errStr := app.ReadPluginSettings("verstak.todo")
	if errStr != "" {
		t.Fatalf("ReadPluginSettings: %s", errStr)
	}
	got := map[string]string{}
	for _, record := range recordsFromSettingsValue(settings["todos:global"]) {
		id, _ := record["id"].(string)
		title, _ := record["title"].(string)
		got[id] = title
	}
	if len(got) != len(want) {
		t.Fatalf("todos = %#v, want %#v", got, want)
	}
	for id, title := range want {
		if got[id] != title {
			t.Fatalf("todos = %#v, want %#v", got, want)
		}
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
	identity, ok := app.treeV2.ResolveWorkspace(name)
	if !ok {
		t.Fatalf("Deal %s not found", name)
	}
	if identity.ID != wantID {
		t.Fatalf("Deal %s ID = %s, want %s", name, identity.ID, wantID)
	}
}
