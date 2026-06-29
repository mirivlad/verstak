package secrets

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testKey(seed byte) []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = seed
	}
	return key
}

func readAllSecretStoreBytes(t *testing.T, root string) []byte {
	t.Helper()

	var data []byte
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fileData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		data = append(data, fileData...)
		return nil
	}); err != nil {
		t.Fatalf("read secret store bytes: %v", err)
	}
	return data
}

func TestStoreRoundTripsSecretWithoutPlaintextOnDisk(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, testKey(0x11))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	const id = "server.password"
	const value = "s3cr3t-value"
	if err := store.Write(id, value); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := store.Read(id)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got != value {
		t.Fatalf("Read = %q, want %q", got, value)
	}

	raw := readAllSecretStoreBytes(t, root)
	if bytes.Contains(raw, []byte(value)) {
		t.Fatal("secret value is stored as plaintext")
	}
	if bytes.Contains(raw, []byte(id)) {
		t.Fatal("secret id is stored as plaintext")
	}
}

func TestStoreRejectsWrongKey(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, testKey(0x11))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := store.Write("server.password", "s3cr3t-value"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	wrongKeyStore, err := NewStore(root, testKey(0x22))
	if err != nil {
		t.Fatalf("NewStore wrong key: %v", err)
	}
	if _, err := wrongKeyStore.Read("server.password"); err == nil {
		t.Fatal("Read with wrong key succeeded")
	}
}

func TestStoreRejectsUnsafeIDs(t *testing.T) {
	store, err := NewStore(t.TempDir(), testKey(0x11))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	for _, id := range []string{"", ".", "..", "../secret", `folder\secret`, strings.Repeat("a", 257)} {
		t.Run(id, func(t *testing.T) {
			if err := store.Write(id, "value"); err == nil {
				t.Fatalf("Write(%q): expected error", id)
			}
			if _, err := store.Read(id); err == nil {
				t.Fatalf("Read(%q): expected error", id)
			}
		})
	}
}

func TestStoreListsScopedRecordsWithoutPlaintextOnDisk(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root, testKey(0x11))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	records := []SecretRecord{
		{
			ID:        "global.server-token",
			Title:     "Server Token",
			Value:     "global-secret-value",
			Scope:     SecretScope{Kind: ScopeGlobal},
			Username:  "root",
			UpdatedAt: "caller value must be replaced",
		},
		{
			ID:    "client-a.database",
			Title: "Client A Database",
			Value: "workspace-secret-value",
			Scope: SecretScope{
				Kind:              ScopeWorkspace,
				WorkspaceRootPath: "ClientA",
			},
			Username: "app",
		},
	}
	for _, record := range records {
		if err := store.WriteRecord(record); err != nil {
			t.Fatalf("WriteRecord(%s): %v", record.ID, err)
		}
	}

	list, err := store.ListRecords()
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListRecords len = %d, want 2: %+v", len(list), list)
	}
	if list[0].Value != "" || list[1].Value != "" {
		t.Fatalf("ListRecords leaked secret values: %+v", list)
	}

	workspaceRecord, err := store.ReadRecord("client-a.database")
	if err != nil {
		t.Fatalf("ReadRecord: %v", err)
	}
	if workspaceRecord.Value != "workspace-secret-value" {
		t.Fatalf("workspace secret value = %q", workspaceRecord.Value)
	}
	if workspaceRecord.Scope.Kind != ScopeWorkspace || workspaceRecord.Scope.WorkspaceRootPath != "ClientA" {
		t.Fatalf("workspace scope = %+v", workspaceRecord.Scope)
	}
	if workspaceRecord.UpdatedAt == "" || workspaceRecord.UpdatedAt == "caller value must be replaced" {
		t.Fatalf("UpdatedAt was not set by store: %+v", workspaceRecord)
	}

	raw := readAllSecretStoreBytes(t, root)
	for _, plaintext := range []string{
		"global-secret-value",
		"workspace-secret-value",
		"global.server-token",
		"client-a.database",
		"Client A Database",
		"ClientA",
	} {
		if bytes.Contains(raw, []byte(plaintext)) {
			t.Fatalf("secret store contains plaintext %q", plaintext)
		}
	}
}

func TestVaultSessionUnlocksWithMasterPasswordOnce(t *testing.T) {
	root := t.TempDir()
	session := NewVaultSession(root)

	if session.Unlocked() {
		t.Fatal("new session is unlocked")
	}
	if _, err := session.Store(); err == nil {
		t.Fatal("Store before unlock succeeded")
	}

	store, err := session.Unlock("correct horse battery staple")
	if err != nil {
		t.Fatalf("Unlock first time: %v", err)
	}
	if !session.Unlocked() {
		t.Fatal("session did not stay unlocked")
	}
	if err := store.Write("server.password", "s3cr3t-value"); err != nil {
		t.Fatalf("Write: %v", err)
	}

	sameStore, err := session.Store()
	if err != nil {
		t.Fatalf("Store after unlock: %v", err)
	}
	if got, err := sameStore.Read("server.password"); err != nil || got != "s3cr3t-value" {
		t.Fatalf("Read after unlock = %q, %v", got, err)
	}

	nextSession := NewVaultSession(root)
	if _, err := nextSession.Unlock("wrong password"); err == nil {
		t.Fatal("Unlock with wrong password succeeded")
	}
	if _, err := nextSession.Unlock("correct horse battery staple"); err != nil {
		t.Fatalf("Unlock with correct password: %v", err)
	}

	raw := readAllSecretStoreBytes(t, root)
	for _, plaintext := range []string{"correct horse battery staple", "s3cr3t-value"} {
		if bytes.Contains(raw, []byte(plaintext)) {
			t.Fatalf("vault session storage contains plaintext %q", plaintext)
		}
	}
}
