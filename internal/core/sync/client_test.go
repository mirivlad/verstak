package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
