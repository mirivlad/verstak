package dealmigration

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestDealMetadataTransformMaterializesPathKeyedMetadata(t *testing.T) {
	vault := t.TempDir()
	workspaceID := "0181c5b6-7a13-7c45-9e0e-07e0d3119da3"
	rootPath := "Clients/Acme"
	writeMigrationJSON(t, vault, rootPath+"/.verstak/workspace.json", map[string]any{
		"schemaVersion": 1,
		"workspaceId":   workspaceID,
	})
	legacyPath := filepath.Join(vault, ".verstak", "workspaces", base64.RawURLEncoding.EncodeToString([]byte(rootPath))+".json")
	legacyBody := []byte(`{"workspaceName":"Acme","workspaceTools":["verstak.notes"],"updatedAt":"2026-08-20T00:00:00Z"}`)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, legacyBody, 0o600); err != nil {
		t.Fatal(err)
	}

	runner := NewDealOnlyRunner(vault)
	if needed, err := runner.NeedsMigration(context.Background()); err != nil || !needed {
		t.Fatalf("NeedsMigration = %v, %v", needed, err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	metadata := readMigrationJSON(t, vault, ".verstak/workspaces/uuid-"+workspaceID+".json")
	if metadata["workspaceId"] != workspaceID || metadata["workspaceName"] != "Acme" {
		t.Fatalf("canonical metadata = %#v", metadata)
	}
	retained, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(retained) != string(legacyBody) {
		t.Fatalf("legacy metadata source changed: %s", retained)
	}
}

func TestDealMetadataTransformCreatesMinimalMetadataForMarkedDeal(t *testing.T) {
	vault := t.TempDir()
	workspaceID := "0181c5b6-7a13-7c45-9e0e-07e0d3119da3"
	writeMigrationJSON(t, vault, "Clients/Acme/.verstak/workspace.json", map[string]any{
		"schemaVersion": 1,
		"workspaceId":   workspaceID,
	})

	if err := NewDealOnlyRunner(vault).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	metadata := readMigrationJSON(t, vault, ".verstak/workspaces/uuid-"+workspaceID+".json")
	if metadata["workspaceId"] != workspaceID || metadata["workspaceName"] != "Acme" {
		t.Fatalf("minimal metadata = %#v", metadata)
	}
}
