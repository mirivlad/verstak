package importservice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRecoverRollsBackPublishingTransaction(t *testing.T) {
	vault, _, _, service := newApplyTestService(t)
	target := filepath.Join(vault, "Импортировано", "Interrupted")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := filepath.Join(vault, ".verstak", "workspaces", "uuid-test.json")
	if err := os.WriteFile(registry, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	txnDir := filepath.Join(vault, ".verstak", "import-staging", "test-handle")
	if err := os.MkdirAll(txnDir, 0o755); err != nil {
		t.Fatal(err)
	}
	journal := transactionJournal{Version: 1, TransactionID: "txn-test", Status: transactionPublishing, PublishedRoot: "Импортировано/Interrupted", RegistryPaths: []string{".verstak/workspaces/uuid-test.json"}}
	if err := writeImportOwnershipMarker(target, journal.TransactionID); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(journal)
	if err := os.WriteFile(filepath.Join(txnDir, "transaction.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := service.Recover(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{target, registry, txnDir} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("path not removed: %s (%v)", path, err)
		}
	}
}

func TestRecoverDoesNotRemoveUnownedPublishedPath(t *testing.T) {
	vault, _, _, service := newApplyTestService(t)
	target := filepath.Join(vault, "Импортировано", "SomeoneElse")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	txnDir := filepath.Join(vault, ".verstak", "import-staging", "test-handle")
	if err := os.MkdirAll(txnDir, 0o755); err != nil {
		t.Fatal(err)
	}
	journal := transactionJournal{Version: 1, TransactionID: "txn-test", Status: transactionPublishing, PublishedRoot: "Импортировано/SomeoneElse"}
	data, _ := json.Marshal(journal)
	if err := os.WriteFile(filepath.Join(txnDir, "transaction.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := service.Recover(); err == nil {
		t.Fatal("expected ownership error")
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("unowned path removed: %v", err)
	}
}

func TestRecoverKeepsCommittedTree(t *testing.T) {
	vault, _, _, service := newApplyTestService(t)
	target := filepath.Join(vault, "Импортировано", "Complete")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	txnDir := filepath.Join(vault, ".verstak", "import-staging", "test-handle")
	if err := os.MkdirAll(txnDir, 0o755); err != nil {
		t.Fatal(err)
	}
	journal := transactionJournal{Version: 1, Status: transactionCommitted, PublishedRoot: "Импортировано/Complete"}
	data, _ := json.Marshal(journal)
	if err := os.WriteFile(filepath.Join(txnDir, "transaction.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := service.Recover(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("committed tree removed: %v", err)
	}
	if _, err := os.Stat(txnDir); !os.IsNotExist(err) {
		t.Fatalf("transaction directory remains: %v", err)
	}
}
