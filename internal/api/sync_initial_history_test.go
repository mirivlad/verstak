package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

func testInitialFileOp(seq int, opID, opType, entityID, payload string) syncsvc.Op {
	return syncsvc.Op{
		OpID: opID, ID: opID, ServerSequence: seq,
		DeviceID:   "other-device",
		EntityType: syncsvc.EntityFile, EntityID: entityID,
		OpType: opType, PayloadJSON: payload,
	}
}

func TestObsoleteInitialFileOpsOnlyCompleteDeletedLifetimes(t *testing.T) {
	ops := []syncsvc.Op{
		testInitialFileOp(1, "create", syncsvc.OpCreate, "PlatformTest/files-api.txt", `{"path":"PlatformTest/files-api.txt","content":"old"}`),
		testInitialFileOp(2, "move", syncsvc.OpMove, "PlatformTest/files-api.txt", `{"fromPath":"PlatformTest/files-api.txt","toPath":"PlatformTest/files-api-moved.txt"}`),
		testInitialFileOp(3, "delete", syncsvc.OpDelete, "PlatformTest/files-api-moved.txt", `{"path":"PlatformTest/files-api-moved.txt"}`),
		testInitialFileOp(4, "recreate", syncsvc.OpCreate, "PlatformTest/files-api.txt", `{"path":"PlatformTest/files-api.txt","content":"new"}`),
		testInitialFileOp(5, "live-update", syncsvc.OpUpdate, "PlatformTest/files-api.txt", `{"path":"PlatformTest/files-api.txt","content":"final"}`),
		testInitialFileOp(6, "untraced-move", syncsvc.OpMove, "unknown", `{"fromPath":"unknown","toPath":"elsewhere"}`),
		testInitialFileOp(7, "untraced-delete", syncsvc.OpDelete, "elsewhere", `{"path":"elsewhere"}`),
	}
	skipped := obsoleteInitialFileOps(ops)
	for _, id := range []string{"create", "move", "delete"} {
		if !skipped[id] {
			t.Errorf("expected historical %s to be skipped", id)
		}
	}
	for _, id := range []string{"recreate", "live-update", "untraced-move", "untraced-delete"} {
		if skipped[id] {
			t.Errorf("still-live or untraceable %s must never be skipped", id)
		}
	}
}

func TestInitialSyncDoesNotReplayDeletedRemoteMoveAcrossLocalFile(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.read", "files.write", "files.delete"})
	if err := os.MkdirAll(filepath.Join(root, "PlatformTest"), 0o755); err != nil {
		t.Fatal(err)
	}
	// This file is independently present on a not-yet-paired client. It is
	// neither the moved file nor the deleted file in the remote history.
	localPath := filepath.Join(root, "PlatformTest", "files-api.txt")
	if err := os.WriteFile(localPath, []byte("my local version"), 0o600); err != nil {
		t.Fatal(err)
	}
	app.syncSvc = syncsvc.NewService(root, "local-device")
	if _, err := app.syncSvc.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}
	// Existing locally queued work must survive the remote initial pull.
	if err := app.syncSvc.RecordOp(syncsvc.EntityFile, "PlatformTest/files-api.txt", syncsvc.OpUpdate, map[string]string{"path": "PlatformTest/files-api.txt", "content": "my local version"}); err != nil {
		t.Fatal(err)
	}
	app.appSettings = appsettings.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err := app.appSettings.Load(); err != nil {
		t.Fatal(err)
	}
	remote := []syncsvc.Op{
		{OpID: "folder", ServerSequence: 1, DeviceID: "remote", EntityType: syncsvc.EntityFolder, EntityID: "PlatformTest", OpType: syncsvc.OpCreate, PayloadJSON: `{"path":"PlatformTest"}`},
		testInitialFileOp(2, "created", syncsvc.OpCreate, "PlatformTest/files-api.txt", `{"path":"PlatformTest/files-api.txt","content":"remote old content"}`),
		testInitialFileOp(3, "moved", syncsvc.OpMove, "PlatformTest/files-api.txt", `{"fromPath":"PlatformTest/files-api.txt","toPath":"PlatformTest/files-api-moved.txt"}`),
		testInitialFileOp(4, "deleted", syncsvc.OpDelete, "PlatformTest/files-api-moved.txt", `{"path":"PlatformTest/files-api-moved.txt"}`),
		testInitialFileOp(5, "remote-only", syncsvc.OpCreate, "PlatformTest/remote.txt", `{"path":"PlatformTest/remote.txt","content":"remote content"}`),
	}
	var pushed []syncsvc.PushOp
	pullCalls := 0
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer device-token" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/sync/pull":
			pullCalls++
			var request syncsvc.PullRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			remaining := []syncsvc.Op{}
			for _, op := range remote {
				if op.ServerSequence > request.SinceSequence {
					remaining = append(remaining, op)
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"server_sequence": 5,
				"page_last_sequence": func() int {
					if len(remaining) > 0 {
						return remaining[len(remaining)-1].ServerSequence
					}
					return request.SinceSequence
				}(),
				"has_more": false,
				"ops":      remaining,
			})
		case "/api/v1/sync/push":
			var request struct {
				Ops []syncsvc.PushOp `json:"ops"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			pushed = append(pushed, request.Ops...)
			accepted := make([]string, 0, len(request.Ops))
			for _, op := range request.Ops {
				accepted = append(accepted, op.OpID)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"accepted": accepted, "count": len(accepted), "conflicts": []interface{}{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	if err := app.syncSvc.SetState(server.URL, ""); err != nil {
		t.Fatal(err)
	}
	if err := syncsvc.SaveDeviceToken(root, "device-token"); err != nil {
		t.Fatal(err)
	}
	result, err := app.syncNow()
	if err != nil {
		t.Fatalf("initial sync failed on historical move: %v", err)
	}
	if pullCalls < 3 {
		t.Fatalf("pulls = %d, want preflight, apply, and final pull", pullCalls)
	}
	data, err := os.ReadFile(localPath)
	if err != nil || string(data) != "my local version" {
		t.Fatalf("local data changed: %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(root, "PlatformTest", "files-api-moved.txt")); !os.IsNotExist(err) {
		t.Fatalf("historically deleted file exists: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(root, "PlatformTest", "remote.txt")); err != nil || string(data) != "remote content" {
		t.Fatalf("remote file: %q %v", data, err)
	}
	assertLastPullSequence(t, app.syncSvc, 5)
	if result["pulled"] != 5 {
		t.Fatalf("expected full cursor replay, got %#v", result)
	}
	publishedLocal := false
	for _, op := range pushed {
		if op.EntityID == "PlatformTest/files-api.txt" && (op.OpType == syncsvc.OpUpdate || op.OpType == syncsvc.OpCreate) {
			publishedLocal = true
		}
		if strings.Contains(op.EntityID, "moved.txt") {
			t.Fatalf("published transient remote file: %#v", op)
		}
	}
	if !publishedLocal {
		t.Fatalf("local queued change not published (got %d ops): %s", len(pushed), fmt.Sprint(result))
	}
	pending, err := app.syncSvc.GetUnpushedOps()
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending after successful push: %d, %v", len(pending), err)
	}
}

func TestInitialHistoryPreflightFailureDoesNotPublishLocalQueue(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.read", "files.write", "files.delete"})
	app.syncSvc = syncsvc.NewService(root, "local-device")
	if _, err := app.syncSvc.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}
	if err := app.syncSvc.RecordOp(syncsvc.EntityFile, "draft.md", syncsvc.OpCreate, map[string]string{"path": "draft.md", "content": "draft"}); err != nil {
		t.Fatal(err)
	}
	app.appSettings = appsettings.NewManager(filepath.Join(t.TempDir(), "settings.json"))
	if err := app.appSettings.Load(); err != nil {
		t.Fatal(err)
	}
	pushed := false
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sync/pull":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"server_sequence": 3, "page_last_sequence": 3, "has_more": true, "ops": []interface{}{}})
		case "/api/v1/sync/push":
			pushed = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()
	if err := app.syncSvc.SetState(server.URL, ""); err != nil {
		t.Fatal(err)
	}
	if err := syncsvc.SaveDeviceToken(root, "device-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.syncNow(); err == nil || !strings.Contains(err.Error(), "cursor mismatch") {
		t.Fatalf("expected read-only preflight failure, got %v", err)
	}
	if pushed {
		t.Fatal("local queue pushed after failed preflight")
	}
	pending, err := app.syncSvc.GetUnpushedOps()
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending queue changed: %d %v", len(pending), err)
	}
	assertLastPullSequence(t, app.syncSvc, 0)
}

func TestRePairSameServerAndVaultPreservesCursorAndPendingQueue(t *testing.T) {
	app, root := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, "old-device")
	meta := app.vault.GetVaultMeta()
	if meta == nil {
		t.Fatal("missing vault metadata")
	}
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/client/pair" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"device_id":    "new-device",
			"device_token": "new-device-token",
		})
	}))
	defer server.Close()
	if err := app.syncSvc.SetState(server.URL, ""); err != nil {
		t.Fatal(err)
	}
	if err := app.syncSvc.SetRemoteVaultID(meta.VaultID); err != nil {
		t.Fatal(err)
	}
	if err := app.syncSvc.SetLastPullSeq(65); err != nil {
		t.Fatal(err)
	}
	if err := app.syncSvc.RecordOp(syncsvc.EntityFile, "draft.md", syncsvc.OpUpdate, map[string]string{
		"path": "draft.md", "content": "locally edited",
	}); err != nil {
		t.Fatal(err)
	}
	if err := app.syncConfigure(server.URL+"/", "alice", "secret", meta.VaultID); err != nil {
		t.Fatal(err)
	}
	assertLastPullSequence(t, app.syncSvc, 65)
	complete, err := app.syncSvc.BootstrapComplete()
	if err != nil || !complete {
		t.Fatalf("re-pair cleared bootstrap state: %v, %v", complete, err)
	}
	pending, err := app.syncSvc.GetUnpushedOps()
	if err != nil || len(pending) != 1 {
		t.Fatalf("re-pair lost queued operations: %d, %v", len(pending), err)
	}
	if app.syncSvc.GetDeviceID() != "new-device" {
		t.Errorf("new device identity not recorded: %q", app.syncSvc.GetDeviceID())
	}
	if got := syncsvc.LoadDeviceToken(root); got != "new-device-token" {
		t.Errorf("new token not stored")
	}
}

func TestApplyRemoteWorkspaceFolderCreatePreservesUUID(t *testing.T) {
	app, root := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, "local-device")
	app.treeV2 = workspacetree.NewService(root, nil)
	if err := app.treeV2.Initialize(); err != nil {
		t.Fatal(err)
	}
	folderID := "da4cd52f-c82d-415f-a548-8cd8b16220ad"
	op := syncsvc.Op{
		OpID:        "server-folder-create",
		DeviceID:    "server-device",
		EntityType:  syncsvc.EntityWorkspaceFolder,
		EntityID:    folderID,
		OpType:      syncsvc.OpCreate,
		PayloadJSON: `{"folderId":"da4cd52f-c82d-415f-a548-8cd8b16220ad","path":"Работа"}`,
	}
	if err := app.applyRemoteOp(op); err != nil {
		t.Fatal(err)
	}
	if err := app.applyRemoteOp(op); err != nil {
		t.Fatalf("repeated folder create: %v", err)
	}
	if folder, ok := app.treeV2.GetFolderByID(folderID); !ok || folder.Path != "Работа" {
		t.Fatalf("folder identity was not retained: %+v %v", folder, ok)
	}
}
