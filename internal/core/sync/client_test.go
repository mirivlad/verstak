package sync

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func withoutSyncRetrySleep(t *testing.T) {
	t.Helper()
	original := syncRetrySleep
	syncRetrySleep = func(time.Duration) {}
	t.Cleanup(func() {
		syncRetrySleep = original
	})
}

func TestPushRetriesTransientServerErrors(t *testing.T) {
	withoutSyncRetrySleep(t)

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sync/push" {
			http.NotFound(w, r)
			return
		}
		attempts++
		if attempts < 3 {
			http.Error(w, "try again", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"accepted":  []string{"op-1"},
			"count":     1,
			"conflicts": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "device-1", t.TempDir())
	client.DeviceToken = "token"
	result, err := client.Push([]Op{{
		OpID:       "op-1",
		EntityType: EntityFile,
		EntityID:   "Docs/one.txt",
		OpType:     OpCreate,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if len(result.Accepted) != 1 || result.Accepted[0] != "op-1" {
		t.Fatalf("push result = %#v", result)
	}
}

func TestPushDoesNotRetryClientErrors(t *testing.T) {
	withoutSyncRetrySleep(t)

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "device-1", t.TempDir())
	client.DeviceToken = "token"
	_, err := client.Push([]Op{{
		OpID:       "op-1",
		EntityType: EntityFile,
		EntityID:   "Docs/one.txt",
		OpType:     OpCreate,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}})
	if err == nil {
		t.Fatal("Push should fail on unauthorized response")
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestPairDeviceSendsVaultID(t *testing.T) {
	var pairedVaultID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/client/pair" {
			http.NotFound(w, r)
			return
		}
		var request struct {
			VaultID string `json:"vault_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		pairedVaultID = request.VaultID
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"device_id":    "device-1",
			"device_token": "token-1",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "", t.TempDir())
	_, _, err := client.PairDevice(server.URL, "alice", "secret", "Desktop", "verstak-desktop/v2", "vault-123")
	if err != nil {
		t.Fatalf("PairDevice: %v", err)
	}
	if pairedVaultID != "vault-123" {
		t.Fatalf("paired vault ID = %q, want vault-123", pairedVaultID)
	}
}

func TestPullPageReadsPaginationMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sync/pull" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"server_sequence": 9, "page_last_sequence": 4, "has_more": true,
			"ops": []map[string]interface{}{{"op_id": "op-4", "server_sequence": 4, "device_id": "other", "entity_type": "file", "entity_id": "a.bin", "op_type": "update", "payload_json": `{}`, "created_at": "2026-01-01T00:00:00Z"}},
		})
	}))
	defer server.Close()
	client := NewClient(server.URL, "", "device", t.TempDir())
	client.DeviceToken = "token"
	response, err := client.PullPage(2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if response.PageLastSequence != 4 || !response.HasMore || response.ServerSequence != 9 {
		t.Fatalf("pagination response = %+v", response)
	}
}

func TestUploadBlobStreamsMultipartAndChecksReturnedSize(t *testing.T) {
	data := []byte("streamed binary payload")
	path := filepath.Join(t.TempDir(), "blob.bin")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1024); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()
		received, err := io.ReadAll(file)
		if err != nil || string(received) != string(data) {
			http.Error(w, "bad upload", http.StatusBadRequest)
			return
		}
		hash := sha256.Sum256(data)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"sha256": fmt.Sprintf("%x", hash[:]), "size": len(data)})
	}))
	defer server.Close()
	client := NewClient(server.URL, "", "device", t.TempDir())
	client.DeviceToken = "token"
	ref, err := client.UploadBlob(path)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Size != int64(len(data)) {
		t.Fatalf("uploaded reference = %+v", ref)
	}
}

func TestDownloadBlobVerifiesHashAndLeavesNoCorruptDestination(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "received.bin")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "7")
		_, _ = w.Write([]byte("corrupt"))
	}))
	defer server.Close()
	client := NewClient(server.URL, "", "device", t.TempDir())
	client.DeviceToken = "token"
	err := client.DownloadBlobVerified("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 7, dest)
	if err == nil {
		t.Fatal("corrupt blob was accepted")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("corrupt destination remains: %v", err)
	}
}

func TestClientPreservesStableServerErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = w.Write([]byte(`{"error":"internal wording must not reach UI","code":"quota_exceeded"}`))
	}))
	defer server.Close()
	client := NewClient(server.URL, "token", "device", t.TempDir())
	err := client.post("/api/v1/sync/push", map[string]string{}, nil)
	serverErr, ok := err.(*ServerError)
	if !ok || serverErr.Code != "quota_exceeded" || serverErr.Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("error = %#v, want quota server error", err)
	}
}
