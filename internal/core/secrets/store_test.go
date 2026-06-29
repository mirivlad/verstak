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
