package workspacetree

import (
	"encoding/json"
	"testing"
)

func TestCreateWorkspaceFromSyncPreservesNestedPathIdentityAndMetadata(t *testing.T) {
	svc, _ := newMetadataV2Service(t)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	workspaceID := testUUID("sync-create")
	metadata := DealMetadata{
		SchemaVersion:  DealMetadataSchemaVersion,
		WorkspaceID:    workspaceID,
		WorkspaceName:  "Acme",
		WorkspaceTools: []string{"verstak.notes", "third.party"},
		ToolConfig:     map[string]json.RawMessage{"third.party": json.RawMessage(`{"keep":true}`)},
		UpdatedAt:      "2026-08-29T00:00:00Z",
	}

	created, err := svc.CreateWorkspaceFromSync("Clients/Acme", workspaceID, metadata, nil)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != workspaceID || created.RootPath != "Clients/Acme" {
		t.Fatalf("created = %#v", created)
	}
	if _, err := svc.CreateWorkspaceFromSync("Clients/Acme", workspaceID, metadata, nil); err != nil {
		t.Fatalf("idempotent create: %v", err)
	}
	stored, err := svc.ReadDealMetadata(workspaceID, created.RootPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(stored.ToolConfig["third.party"]) != `{"keep":true}` {
		t.Fatalf("tool config = %s", stored.ToolConfig["third.party"])
	}
}

func TestTrashAndRestoreWorkspaceFromSyncAreIdempotent(t *testing.T) {
	svc, _ := newMetadataV2Service(t)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	workspaceID := testUUID("sync-trash-restore")
	metadata := newDealMetadata(workspaceID, "Acme", "minimal", []string{"verstak.notes"})
	if _, err := svc.CreateWorkspaceFromSync("Clients/Acme", workspaceID, metadata, nil); err != nil {
		t.Fatal(err)
	}
	if err := svc.TrashWorkspaceFromSync(workspaceID, "Clients/Acme", nil); err != nil {
		t.Fatal(err)
	}
	if err := svc.TrashWorkspaceFromSync(workspaceID, "Clients/Acme", nil); err != nil {
		t.Fatalf("idempotent trash: %v", err)
	}
	restored, err := svc.RestoreWorkspaceFromSync(workspaceID, "Archive/Acme", nil)
	if err != nil {
		t.Fatal(err)
	}
	if restored.RootPath != "Archive/Acme" {
		t.Fatalf("restored = %#v", restored)
	}
	if _, err := svc.RestoreWorkspaceFromSync(workspaceID, "Archive/Acme", nil); err != nil {
		t.Fatalf("idempotent restore: %v", err)
	}
}
