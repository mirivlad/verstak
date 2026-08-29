package workspacetree

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func newMetadataV2Service(t *testing.T) (*Service, string) {
	t.Helper()
	vault := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vault, ".verstak"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, ".verstak", "vault.json"), []byte(`{"schemaVersion":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return NewService(vault, nil), vault
}

func writeMetadataV2Fixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReadDealMetadataPrefersCanonicalUUIDRecord(t *testing.T) {
	svc, vault := newMetadataV2Service(t)
	workspaceID := testUUID("metadata-priority")
	rootPath := "Clients/Acme"
	legacyName := base64.RawURLEncoding.EncodeToString([]byte(rootPath)) + ".json"
	writeMetadataV2Fixture(t, filepath.Join(vault, ".verstak", "workspaces", legacyName), `{"workspaceId":"`+workspaceID+`","workspaceName":"Acme","workspaceTools":["legacy.tool"]}`)
	writeMetadataV2Fixture(t, filepath.Join(vault, ".verstak", "workspaces", "uuid-"+workspaceID+".json"), `{"schemaVersion":2,"workspaceId":"`+workspaceID+`","workspaceName":"Acme","workspaceTools":["canonical.tool"],"toolConfig":{},"updatedAt":"2026-08-29T00:00:00Z"}`)

	metadata, err := svc.ReadDealMetadata(workspaceID, rootPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(metadata.WorkspaceTools, []string{"canonical.tool"}) {
		t.Fatalf("workspaceTools = %#v", metadata.WorkspaceTools)
	}
}

func TestWriteDealMetadataRoundTripsUnknownToolConfig(t *testing.T) {
	svc, _ := newMetadataV2Service(t)
	workspaceID := testUUID("metadata-roundtrip")
	wantConfig := json.RawMessage(`{"presetVersion":7,"opaque":{"keep":true}}`)

	err := svc.WriteDealMetadata(DealMetadata{
		SchemaVersion:  DealMetadataSchemaVersion,
		WorkspaceID:    workspaceID,
		WorkspaceName:  "Research",
		WorkspaceTools: []string{"third.party", "verstak.notes", "third.party", ""},
		ToolConfig: map[string]json.RawMessage{
			"third.party": wantConfig,
		},
		CreatedFromTemplate: &TemplateProvenance{
			TemplateID:      "custom-research",
			TemplateName:    "Research",
			TemplateVersion: 3,
			AppliedAt:       "2026-08-29T00:00:00Z",
			RecipeDigest:    "sha256:fixture",
		},
		UpdatedAt: "2026-08-29T00:00:01Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.ReadDealMetadata(workspaceID, "ignored/address")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.WorkspaceTools, []string{"third.party", "verstak.notes"}) {
		t.Fatalf("workspaceTools = %#v", got.WorkspaceTools)
	}
	if !reflect.DeepEqual(got.ToolConfig["third.party"], wantConfig) {
		t.Fatalf("toolConfig = %s", got.ToolConfig["third.party"])
	}
	if got.CreatedFromTemplate == nil || got.CreatedFromTemplate.RecipeDigest != "sha256:fixture" {
		t.Fatalf("createdFromTemplate = %#v", got.CreatedFromTemplate)
	}
}

func TestWriteDealMetadataRejectsWorkspaceIdentityMismatch(t *testing.T) {
	svc, vault := newMetadataV2Service(t)
	workspaceID := testUUID("metadata-mismatch")
	path := filepath.Join(vault, ".verstak", "workspaces", "uuid-"+workspaceID+".json")
	writeMetadataV2Fixture(t, path, `{"schemaVersion":2,"workspaceId":"`+testUUID("other-workspace")+`","workspaceName":"Wrong","workspaceTools":[],"toolConfig":{},"updatedAt":"2026-08-29T00:00:00Z"}`)

	_, err := svc.ReadDealMetadata(workspaceID, "Wrong")
	if err == nil || !strings.Contains(err.Error(), "workspace identity") {
		t.Fatalf("error = %v", err)
	}
}

func TestMigrateLegacyDealMetadataWritesV2AndRetainsSource(t *testing.T) {
	svc, vault := newMetadataV2Service(t)
	workspaceID := testUUID("metadata-migrate")
	rootPath := "Archive/Research"
	legacyName := base64.RawURLEncoding.EncodeToString([]byte(rootPath)) + ".json"
	legacyPath := filepath.Join(vault, ".verstak", "workspaces", legacyName)
	legacyBody := `{"workspaceName":"Research","workspaceTools":["verstak.notes","verstak.notes","third.party"],"createdFromTemplate":{"templateId":"writing","templateName":"Writing","templateVersion":2,"appliedAt":"2026-08-20T00:00:00Z"},"updatedAt":"2026-08-20T00:00:01Z"}`
	writeMetadataV2Fixture(t, legacyPath, legacyBody)

	metadata, migrated, err := svc.MigrateLegacyDealMetadata(workspaceID, rootPath)
	if err != nil {
		t.Fatal(err)
	}
	if !migrated {
		t.Fatal("legacy metadata was not migrated")
	}
	if metadata.SchemaVersion != DealMetadataSchemaVersion || metadata.WorkspaceID != workspaceID {
		t.Fatalf("metadata = %#v", metadata)
	}
	if !reflect.DeepEqual(metadata.WorkspaceTools, []string{"verstak.notes", "third.party"}) {
		t.Fatalf("workspaceTools = %#v", metadata.WorkspaceTools)
	}
	if metadata.CreatedFromTemplate == nil || metadata.CreatedFromTemplate.TemplateID != "writing" {
		t.Fatalf("createdFromTemplate = %#v", metadata.CreatedFromTemplate)
	}
	retained, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(retained) != legacyBody {
		t.Fatal("legacy source was modified")
	}
	canonical, err := os.ReadFile(filepath.Join(vault, ".verstak", "workspaces", "uuid-"+workspaceID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(canonical, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.SchemaVersion != DealMetadataSchemaVersion {
		t.Fatalf("schemaVersion = %d", envelope.SchemaVersion)
	}
}

func TestCreateWorkspaceDoesNotPublishWithoutCanonicalMetadata(t *testing.T) {
	svc, vault := newMetadataV2Service(t)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	registryPath := filepath.Join(vault, ".verstak", "workspaces")
	if err := os.WriteFile(registryPath, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := svc.CreateWorkspaceWithTools("", "MustNotExist", "default", []string{"verstak.notes"}, noopRefresh)
	if err == nil {
		t.Fatal("creation unexpectedly succeeded")
	}
	if _, statErr := os.Stat(filepath.Join(vault, "MustNotExist")); !os.IsNotExist(statErr) {
		t.Fatalf("Deal was published after metadata failure: %v", statErr)
	}
}

func TestRenameWorkspaceUpdatesCanonicalMetadataName(t *testing.T) {
	svc, _ := newMetadataV2Service(t)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	workspace, err := svc.CreateWorkspaceWithTools("", "Before", "minimal", []string{"verstak.notes"}, noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RenameWorkspace(workspace.ID, "After", noopRefresh); err != nil {
		t.Fatal(err)
	}
	metadata, err := svc.ReadDealMetadata(workspace.ID, "After")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.WorkspaceName != "After" {
		t.Fatalf("workspaceName = %q", metadata.WorkspaceName)
	}
}
